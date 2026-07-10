package ingest

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/LyleMi/AgentMeter/internal/audit"
	"github.com/LyleMi/AgentMeter/internal/db"
	"github.com/LyleMi/AgentMeter/internal/model"
	"github.com/LyleMi/AgentMeter/internal/platform"
)

type auditReplacement struct {
	Source       model.Source
	SourceFileID int64
	Session      model.Session
	ToolCalls    []model.ToolCall
	Events       []model.Event
}

type auditEventIndex struct {
	rawEventsByToolCall map[int64]int64
	eventIDsByLine      map[int]int64
}

type auditIndexRun struct {
	input  auditReplacement
	events auditEventIndex
	now    string
}

type indexedAuditFinding struct {
	finding     audit.Finding
	timestamp   time.Time
	rawEventID  int64
	description string
	platform    string
}

func (i *Indexer) replaceAuditFindings(ctx context.Context, tx *sql.Tx, input auditReplacement) error {
	findings := audit.AuditSession(input.Session, input.ToolCalls, input.Events)
	run := newAuditIndexRun(input)
	if err := run.insertFindings(ctx, tx, findings); err != nil {
		return err
	}
	return run.upsertAuditRun(ctx, tx, len(findings))
}

func newAuditIndexRun(input auditReplacement) auditIndexRun {
	return auditIndexRun{
		input:  input,
		events: newAuditEventIndex(input.ToolCalls, input.Events),
		now:    db.FormatTime(time.Now().UTC()),
	}
}

func newAuditEventIndex(toolCalls []model.ToolCall, events []model.Event) auditEventIndex {
	index := auditEventIndex{
		rawEventsByToolCall: make(map[int64]int64, len(toolCalls)),
		eventIDsByLine:      make(map[int]int64, len(events)),
	}
	for _, call := range toolCalls {
		index.rawEventsByToolCall[call.ID] = call.RawStartEventID
	}
	for _, event := range events {
		index.eventIDsByLine[event.SourceLine] = event.ID
	}
	return index
}

func (r auditIndexRun) insertFindings(ctx context.Context, tx *sql.Tx, findings []audit.Finding) error {
	insertFinding, err := tx.PrepareContext(ctx, `INSERT INTO audit_findings
		(session_id, tool_call_id, source_file_id, raw_event_id, source_line, timestamp, source, event_type, category, severity, rule_id, title, description, evidence, command, shell_family, platform, decision, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer insertFinding.Close()

	for _, finding := range findings {
		if err := r.insertFinding(ctx, insertFinding, r.indexFinding(finding)); err != nil {
			return err
		}
	}
	return nil
}

func (r auditIndexRun) indexFinding(finding audit.Finding) indexedAuditFinding {
	timestamp := finding.StartedAt
	if timestamp.IsZero() {
		timestamp = r.input.Session.StartedAt
	}
	return indexedAuditFinding{
		finding:     finding,
		timestamp:   timestamp,
		rawEventID:  r.events.rawEventID(finding),
		description: auditDescription(finding),
		platform:    inferAuditPlatform(r.input.Source, r.input.Session, finding),
	}
}

func (i auditEventIndex) rawEventID(finding audit.Finding) int64 {
	if finding.EventID != 0 {
		return finding.EventID
	}
	if finding.ToolCallID != 0 {
		if eventID := i.rawEventsByToolCall[finding.ToolCallID]; eventID != 0 {
			return eventID
		}
	}
	if finding.SourceLine != 0 {
		return i.eventIDsByLine[finding.SourceLine]
	}
	return 0
}

func (r auditIndexRun) insertFinding(ctx context.Context, statement *sql.Stmt, indexed indexedAuditFinding) error {
	finding := indexed.finding
	_, err := statement.ExecContext(ctx,
		r.input.Session.ID,
		finding.ToolCallID,
		r.input.SourceFileID,
		indexed.rawEventID,
		finding.SourceLine,
		db.FormatTime(indexed.timestamp),
		"session_jsonl",
		firstNonEmptyString(finding.Source, "finding"),
		firstNonEmptyString(finding.Category, "command"),
		firstNonEmptyString(finding.Severity, "low"),
		finding.RuleID,
		finding.Title,
		indexed.description,
		finding.Evidence,
		finding.Command,
		string(finding.ShellFamily),
		indexed.platform,
		"observed",
		r.now)
	return err
}

func (r auditIndexRun) upsertAuditRun(ctx context.Context, tx *sql.Tx, findingCount int) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO audit_runs
		(source_file_id, session_id, source, status, finding_count, audited_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(source_file_id) DO UPDATE SET
			session_id = excluded.session_id,
			source = excluded.source,
			status = excluded.status,
			finding_count = excluded.finding_count,
			audited_at = excluded.audited_at`,
		r.input.SourceFileID, r.input.Session.ID, "session_jsonl", "completed", findingCount, r.now)
	return err
}

func auditDescription(finding audit.Finding) string {
	var parts []string
	if finding.ToolName != "" {
		parts = append(parts, "tool: "+finding.ToolName)
	}
	if finding.Field != "" {
		parts = append(parts, "field: "+finding.Field)
	}
	if finding.Source != "" {
		parts = append(parts, "source: "+finding.Source)
	}
	return strings.Join(parts, "; ")
}

func inferAuditPlatform(source model.Source, session model.Session, finding audit.Finding) string {
	switch finding.ShellFamily {
	case audit.ShellPowerShell, audit.ShellCmd:
		return "windows"
	case audit.ShellPosix:
		return "posix"
	}
	path := firstNonEmptyString(finding.ProjectPath, session.ProjectPath, source.RootPath, source.SessionsPath)
	if looksLikeWindowsPath(path) {
		return "windows"
	}
	if strings.HasPrefix(path, "/") {
		return "posix"
	}
	return firstNonEmptyString(source.Platform, platform.PlatformName())
}

func looksLikeWindowsPath(path string) bool {
	if len(path) >= 3 && ((path[0] >= 'A' && path[0] <= 'Z') || (path[0] >= 'a' && path[0] <= 'z')) && path[1] == ':' && (path[2] == '\\' || path[2] == '/') {
		return true
	}
	return strings.HasPrefix(path, `\\`) || strings.Contains(path, `\`)
}
