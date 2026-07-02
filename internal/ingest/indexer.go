package ingest

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/LyleMi/AgentMeter/internal/agent"
	"github.com/LyleMi/AgentMeter/internal/audit"
	"github.com/LyleMi/AgentMeter/internal/db"
	"github.com/LyleMi/AgentMeter/internal/model"
	"github.com/LyleMi/AgentMeter/internal/platform"
	"github.com/LyleMi/AgentMeter/internal/pricing"
	"github.com/LyleMi/AgentMeter/internal/sessionjsonl"
	"github.com/LyleMi/AgentMeter/internal/sourcepath"
)

const sourceFileParserVersion = 5

type Indexer struct {
	conn   *sql.DB
	dbPath string
}

type existingFile struct {
	ID            int64
	SizeBytes     int64
	ModifiedAt    time.Time
	ContentHash   string
	ParserVersion int
	ScanStatus    string
	Message       string
	HasAuditRun   bool
}

type indexRun struct {
	indexer       *Indexer
	calculator    pricing.Calculator
	existingFiles map[string]existingFile
}

func New(conn *sql.DB, dbPath string) *Indexer {
	return &Indexer{conn: conn, dbPath: dbPath}
}

func (i *Indexer) IndexPaths(ctx context.Context, sourcePaths []string, rebuild bool) (model.IndexResult, error) {
	return i.IndexEntries(ctx, sourcepath.SourceEntriesFromPaths(sourcePaths, true), rebuild)
}

func (i *Indexer) IndexEntries(ctx context.Context, sourceEntries []model.SourceEntry, rebuild bool) (model.IndexResult, error) {
	start := time.Now()
	entries := sourcepath.EnabledSourceEntries(sourceEntries)
	paths := sourcepath.SourceEntryPaths(entries)
	result := model.IndexResult{
		SourcePath:  strings.Join(paths, "\n"),
		SourcePaths: paths,
		Database:    i.dbPath,
		Rebuild:     rebuild,
	}
	if len(paths) == 0 {
		return result, errors.New("source path is empty")
	}
	var indexedAny bool
	for _, entry := range entries {
		stat, err := os.Stat(entry.Path)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: %v", entry.Path, err))
			result.Failed++
			continue
		}
		if !stat.IsDir() {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s is not a directory", entry.Path))
			result.Failed++
			continue
		}
		next, err := i.IndexSource(ctx, entry.Path, entry.Label, rebuild)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: %v", entry.Path, err))
			result.Failed++
			continue
		}
		indexedAny = true
		result.FilesSeen += next.FilesSeen
		result.Indexed += next.Indexed
		result.Skipped += next.Skipped
		result.Failed += next.Failed
		result.Sessions += next.Sessions
		result.Warnings = append(result.Warnings, next.Warnings...)
	}
	result.DurationMS = time.Since(start).Milliseconds()
	if !indexedAny && result.Failed > 0 {
		return result, errors.New("no configured source path could be indexed")
	}
	return result, nil
}

func (i *Indexer) Index(ctx context.Context, sessionsPath string, rebuild bool) (model.IndexResult, error) {
	return i.IndexSource(ctx, sessionsPath, "", rebuild)
}

func (i *Indexer) IndexSource(ctx context.Context, sessionsPath, label string, rebuild bool) (model.IndexResult, error) {
	start := time.Now()
	spec := agent.ResolveSource(sessionsPath)
	if cleanedLabel := strings.TrimSpace(label); cleanedLabel != "" {
		spec.Name = cleanedLabel
	}
	result := model.IndexResult{
		SourcePath:  spec.SessionsPath,
		SourcePaths: []string{spec.SessionsPath},
		Database:    i.dbPath,
		Rebuild:     rebuild,
	}
	if spec.SessionsPath == "" {
		return result, errors.New("source path is empty")
	}
	stat, err := os.Stat(spec.SessionsPath)
	if err != nil {
		return result, err
	}
	if !stat.IsDir() {
		return result, fmt.Errorf("%s is not a directory", spec.SessionsPath)
	}

	source, err := db.EnsureSource(ctx, i.conn, spec.Kind, spec.Name, spec.RootPath, spec.SessionsPath, platform.PlatformName())
	if err != nil {
		return result, err
	}
	if rebuild {
		if err := i.deleteSourceFilesForRebuild(ctx, source.ID); err != nil {
			return result, err
		}
	}

	files, err := findJSONLFilesForSource(spec)
	if err != nil {
		return result, err
	}
	calculator, err := pricing.LoadCalculator(ctx, i.conn)
	if err != nil {
		calculator = pricing.Calculator{}
	}
	existingFiles, err := i.loadExistingFiles(ctx, source.ID)
	if err != nil {
		return result, err
	}
	run := indexRun{
		indexer:       i,
		calculator:    calculator,
		existingFiles: existingFiles,
	}
	result.FilesSeen = len(files)
	for _, path := range files {
		indexed, skipped, failed, sessions, warnings := run.indexFile(ctx, source, path, rebuild)
		result.Indexed += indexed
		result.Skipped += skipped
		result.Failed += failed
		result.Sessions += sessions
		result.Warnings = append(result.Warnings, warnings...)
	}
	result.DurationMS = time.Since(start).Milliseconds()
	return result, nil
}

func (r *indexRun) indexFile(ctx context.Context, source model.Source, path string, force bool) (indexed, skipped, failed, sessions int, warnings []string) {
	i := r.indexer
	stat, err := os.Stat(path)
	if err != nil {
		return 0, 0, 1, 0, []string{fmt.Sprintf("%s: %v", path, err)}
	}
	modified := stat.ModTime().UTC()
	existing, hasExisting := r.existingFiles[path]
	knownHash := ""
	if hasExisting && existing.SizeBytes == stat.Size() && existing.ModifiedAt.Equal(modified) {
		knownHash = existing.ContentHash
	}
	hasCurrentParser := !hasExisting || existing.ParserVersion >= sourceFileParserVersion
	if !force && hasCurrentParser && hasExisting && existing.SizeBytes == stat.Size() && existing.HasAuditRun && isCompleteScanStatus(existing.ScanStatus) {
		if existing.ModifiedAt.Equal(modified) {
			return 0, 1, 0, 0, nil
		}
		hash, err := sessionjsonl.HashFile(path)
		if err != nil {
			return 0, 0, 1, 0, []string{fmt.Sprintf("%s: %v", path, err)}
		}
		if existing.ContentHash != "" && existing.ContentHash == hash {
			if _, err := i.upsertSourceFile(ctx, source.ID, path, stat.Size(), modified, hash, existing.ScanStatus, existing.Message); err != nil {
				return 0, 0, 1, 0, []string{fmt.Sprintf("%s: %v", path, err)}
			}
			return 0, 1, 0, 0, nil
		}
		knownHash = hash
	}

	sourceFileID, err := i.upsertSourceFile(ctx, source.ID, path, stat.Size(), modified, knownHash, "scanning", "")
	if err != nil {
		return 0, 0, 1, 0, []string{fmt.Sprintf("%s: %v", path, err)}
	}
	parsed, hash, err := r.parseFile(path, source.ID, sourceFileID, knownHash)
	if err != nil {
		if hash == "" {
			if fallbackHash, hashErr := sessionjsonl.HashFile(path); hashErr == nil {
				hash = fallbackHash
			}
		}
		_ = i.finishSourceFile(ctx, sourceFileID, hash, "error", err.Error())
		return 0, 0, 1, 0, []string{fmt.Sprintf("%s: %v", path, err)}
	}
	if err := i.replaceParsedSession(ctx, source, sourceFileID, parsed, r.calculator); err != nil {
		_ = i.finishSourceFile(ctx, sourceFileID, hash, "error", err.Error())
		return 0, 0, 1, 0, []string{fmt.Sprintf("%s: %v", path, err)}
	}
	status := "indexed"
	message := ""
	if parsed.Session.ParseStatus == "warning" {
		status = "warning"
		message = joinWarnings(parsed.Warnings)
	}
	if err := i.finishSourceFile(ctx, sourceFileID, hash, status, message); err != nil {
		return 0, 0, 1, 0, []string{fmt.Sprintf("%s: %v", path, err)}
	}
	return 1, 0, 0, 1, parsed.Warnings
}

func (r *indexRun) parseFile(path string, sourceID, sourceFileID int64, knownHash string) (model.ParsedSession, string, error) {
	if knownHash != "" {
		parsed, err := sessionjsonl.ParseFile(path, sourceID, sourceFileID)
		return parsed, knownHash, err
	}
	return sessionjsonl.ParseFileWithHash(path, sourceID, sourceFileID)
}

func isCompleteScanStatus(status string) bool {
	return status == "indexed" || status == "warning"
}

func (i *Indexer) loadExistingFiles(ctx context.Context, sourceID int64) (map[string]existingFile, error) {
	rows, err := i.conn.QueryContext(ctx, `SELECT sf.id, sf.path, sf.size_bytes, sf.modified_at, sf.content_hash, sf.parser_version, sf.scan_status, sf.error, ar.id
		FROM source_files sf
		LEFT JOIN audit_runs ar ON ar.source_file_id = sf.id
		WHERE sf.source_id = ?`, sourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[string]existingFile{}
	for rows.Next() {
		var item existingFile
		var path string
		var modified string
		var auditRunID sql.NullInt64
		if err := rows.Scan(&item.ID, &path, &item.SizeBytes, &modified, &item.ContentHash, &item.ParserVersion, &item.ScanStatus, &item.Message, &auditRunID); err != nil {
			return nil, err
		}
		item.ModifiedAt = db.ParseTime(modified)
		item.HasAuditRun = auditRunID.Valid
		result[path] = item
	}
	return result, rows.Err()
}

func (i *Indexer) upsertSourceFile(ctx context.Context, sourceID int64, path string, size int64, modified time.Time, hash, status, message string) (int64, error) {
	now := time.Now().UTC()
	_, err := i.conn.ExecContext(ctx, `INSERT INTO source_files
		(source_id, path, size_bytes, modified_at, content_hash, last_scanned_at, scan_status, error)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			source_id = excluded.source_id,
			size_bytes = excluded.size_bytes,
			modified_at = excluded.modified_at,
			content_hash = excluded.content_hash,
			last_scanned_at = excluded.last_scanned_at,
			scan_status = excluded.scan_status,
			error = excluded.error`,
		sourceID, path, size, db.FormatTime(modified), hash, db.FormatTime(now), status, message)
	if err != nil {
		return 0, err
	}
	var id int64
	err = i.conn.QueryRowContext(ctx, `SELECT id FROM source_files WHERE path = ?`, path).Scan(&id)
	return id, err
}

func (i *Indexer) finishSourceFile(ctx context.Context, sourceFileID int64, hash, status, message string) error {
	_, err := i.conn.ExecContext(ctx, `UPDATE source_files SET content_hash = ?, parser_version = ?, scan_status = ?, error = ?, last_scanned_at = ? WHERE id = ?`,
		hash, sourceFileParserVersion, status, message, db.FormatTime(time.Now().UTC()), sourceFileID)
	return err
}

func (i *Indexer) deleteSourceFilesForRebuild(ctx context.Context, sourceID int64) error {
	tx, err := i.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM token_usage
		WHERE owner_kind = 'session'
		AND owner_id IN (
			SELECT s.id
			FROM sessions s
			JOIN source_files sf ON sf.id = s.source_file_id
			WHERE sf.source_id = ?
		)`, sourceID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM source_files WHERE source_id = ?`, sourceID); err != nil {
		return err
	}
	return tx.Commit()
}

func (i *Indexer) replaceParsedSession(ctx context.Context, source model.Source, sourceFileID int64, parsed model.ParsedSession, calculator pricing.Calculator) error {
	modelCallCosts := make([]*float64, len(parsed.ModelCall))
	for index, call := range parsed.ModelCall {
		usage := model.Usage{
			Model:                    call.Model,
			InputTokens:              call.InputTokens,
			CachedInputTokens:        call.CachedInputTokens,
			OutputTokens:             call.OutputTokens,
			ReasoningOutputTokens:    call.ReasoningOutputTokens,
			ContextCompressionTokens: call.ContextCompressionTokens,
			TotalTokens:              call.TotalTokens,
			Source:                   "actual",
		}
		cost, _ := calculator.Compute(usage)
		modelCallCosts[index] = cost
	}

	tx, err := i.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM token_usage
		WHERE owner_kind = 'session'
		AND owner_id IN (SELECT id FROM sessions WHERE source_file_id = ?)`, sourceFileID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE source_file_id = ?`, sourceFileID); err != nil {
		return err
	}

	res, err := tx.ExecContext(ctx, `INSERT INTO sessions
		(source_id, source_file_id, session_key, codex_session_id, project_path, model, model_provider, originator, thread_source, agent_nickname, agent_role,
		 started_at, ended_at, wall_duration_ms, active_duration_ms, model_duration_ms, tool_duration_ms, idle_duration_ms, event_count, parse_status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		parsed.Session.SourceID,
		sourceFileID,
		parsed.Session.SessionKey,
		parsed.Session.CodexSessionID,
		parsed.Session.ProjectPath,
		parsed.Session.Model,
		parsed.Session.ModelProvider,
		parsed.Session.Originator,
		parsed.Session.ThreadSource,
		parsed.Session.AgentNickname,
		parsed.Session.AgentRole,
		db.FormatTime(parsed.Session.StartedAt),
		db.FormatTime(parsed.Session.EndedAt),
		parsed.Session.WallDurationMS,
		parsed.Session.ActiveDurationMS,
		parsed.Session.ModelDurationMS,
		parsed.Session.ToolDurationMS,
		parsed.Session.IdleDurationMS,
		parsed.Session.EventCount,
		parsed.Session.ParseStatus)
	if err != nil {
		return err
	}
	sessionID, err := res.LastInsertId()
	if err != nil {
		return err
	}
	session := parsed.Session
	session.ID = sessionID
	session.SourceFileID = sourceFileID

	insertEvent, err := tx.PrepareContext(ctx, `INSERT INTO events
		(session_id, source_file_id, source_line, timestamp, kind, raw_type, summary, raw_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer insertEvent.Close()

	rawEventIDs := map[int]int64{}
	eventsByLine := map[int]model.Event{}
	indexedEvents := make([]model.Event, 0, len(parsed.Events))
	for _, event := range parsed.Events {
		res, err := insertEvent.ExecContext(ctx,
			sessionID, sourceFileID, event.SourceLine, db.FormatTime(event.Timestamp), event.Kind, event.RawType, event.Summary, event.RawJSON)
		if err != nil {
			return err
		}
		eventID, _ := res.LastInsertId()
		rawEventIDs[event.SourceLine] = eventID
		event.ID = eventID
		event.SessionID = sessionID
		event.SourceFileID = sourceFileID
		eventsByLine[event.SourceLine] = event
		indexedEvents = append(indexedEvents, event)
	}

	if parsed.Usage.Source == "" {
		parsed.Usage.Source = "unknown"
	}
	if parsed.Usage.Model == "" {
		parsed.Usage.Model = parsed.Session.Model
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO token_usage
		(owner_kind, owner_id, model, input_tokens, cached_input_tokens, output_tokens, reasoning_output_tokens, context_compression_tokens, total_tokens, source)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"session", sessionID, parsed.Usage.Model, parsed.Usage.InputTokens, parsed.Usage.CachedInputTokens,
		parsed.Usage.OutputTokens, parsed.Usage.ReasoningOutputTokens, parsed.Usage.ContextCompressionTokens, parsed.Usage.TotalTokens, parsed.Usage.Source)
	if err != nil {
		return err
	}

	insertModelCall, err := tx.PrepareContext(ctx, `INSERT INTO model_calls
		(session_id, started_at, ended_at, duration_ms, model, provider, status, input_tokens, cached_input_tokens, output_tokens, reasoning_output_tokens, context_compression_tokens, total_tokens, cost_usd)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer insertModelCall.Close()

	for index, call := range parsed.ModelCall {
		_, err := insertModelCall.ExecContext(ctx,
			sessionID, db.FormatTime(call.StartedAt), db.FormatTime(call.EndedAt), call.DurationMS, call.Model, call.Provider, call.Status,
			call.InputTokens, call.CachedInputTokens, call.OutputTokens, call.ReasoningOutputTokens, call.ContextCompressionTokens, call.TotalTokens, nullableFloat(modelCallCosts[index]))
		if err != nil {
			return err
		}
	}

	insertToolCall, err := tx.PrepareContext(ctx, `INSERT INTO tool_calls
		(session_id, started_at, ended_at, duration_ms, tool_name, status, input_summary, output_summary, error, raw_event_id, call_id, raw_start_event_id, raw_end_event_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer insertToolCall.Close()

	indexedToolCalls := make([]model.ToolCall, 0, len(parsed.ToolCall))
	for _, call := range parsed.ToolCall {
		rawStartEventLine := firstNonZero(call.RawStartEventLine, call.RawEventLine)
		rawStartEventID := rawEventIDs[rawStartEventLine]
		rawEndEventID := rawEventIDs[call.RawEndEventLine]
		res, err := insertToolCall.ExecContext(ctx,
			sessionID, db.FormatTime(call.StartedAt), db.FormatTime(call.EndedAt), call.DurationMS,
			call.ToolName, call.Status, call.InputSummary, call.OutputSummary, call.Error, rawStartEventID, call.CallID, rawStartEventID, rawEndEventID)
		if err != nil {
			return err
		}
		toolCallID, _ := res.LastInsertId()
		call.ID = toolCallID
		call.SessionID = sessionID
		call.RawEventID = rawStartEventID
		call.RawStartEventID = rawStartEventID
		call.RawEndEventID = rawEndEventID
		call.ProjectPath = session.ProjectPath
		if startEvent, ok := eventsByLine[rawStartEventLine]; ok {
			call.RawStartEventLine = startEvent.SourceLine
			call.RawStartEventType = startEvent.RawType
			call.RawStartEventSummary = startEvent.Summary
			call.RawStartEventJSON = startEvent.RawJSON
		}
		if endEvent, ok := eventsByLine[call.RawEndEventLine]; ok {
			call.RawEndEventType = endEvent.RawType
			call.RawEndEventSummary = endEvent.Summary
			call.RawEndEventJSON = endEvent.RawJSON
		}
		indexedToolCalls = append(indexedToolCalls, call)
	}

	if err := i.replaceAuditFindings(ctx, tx, source, sourceFileID, session, indexedToolCalls, indexedEvents); err != nil {
		return err
	}

	return tx.Commit()
}

func (i *Indexer) replaceAuditFindings(ctx context.Context, tx *sql.Tx, source model.Source, sourceFileID int64, session model.Session, toolCalls []model.ToolCall, events []model.Event) error {
	findings := audit.AuditSession(session, toolCalls, events)
	toolRawEvents := map[int64]int64{}
	for _, call := range toolCalls {
		toolRawEvents[call.ID] = call.RawStartEventID
	}
	eventIDsByLine := map[int]int64{}
	for _, event := range events {
		eventIDsByLine[event.SourceLine] = event.ID
	}
	now := db.FormatTime(time.Now().UTC())
	insertFinding, err := tx.PrepareContext(ctx, `INSERT INTO audit_findings
		(session_id, tool_call_id, source_file_id, raw_event_id, source_line, timestamp, source, event_type, category, severity, rule_id, title, description, evidence, command, shell_family, platform, decision, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer insertFinding.Close()

	for _, finding := range findings {
		timestamp := finding.StartedAt
		if timestamp.IsZero() {
			timestamp = session.StartedAt
		}
		rawEventID := finding.EventID
		if rawEventID == 0 && finding.ToolCallID != 0 {
			rawEventID = toolRawEvents[finding.ToolCallID]
		}
		if rawEventID == 0 && finding.SourceLine != 0 {
			rawEventID = eventIDsByLine[finding.SourceLine]
		}
		description := auditDescription(finding)
		_, err := insertFinding.ExecContext(ctx,
			session.ID,
			finding.ToolCallID,
			sourceFileID,
			rawEventID,
			finding.SourceLine,
			db.FormatTime(timestamp),
			"session_jsonl",
			firstNonEmptyString(finding.Source, "finding"),
			firstNonEmptyString(finding.Category, "command"),
			firstNonEmptyString(finding.Severity, "low"),
			finding.RuleID,
			finding.Title,
			description,
			finding.Evidence,
			finding.Command,
			string(finding.ShellFamily),
			inferAuditPlatform(source, session, finding),
			"observed",
			now)
		if err != nil {
			return err
		}
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO audit_runs
		(source_file_id, session_id, source, status, finding_count, audited_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(source_file_id) DO UPDATE SET
			session_id = excluded.session_id,
			source = excluded.source,
			status = excluded.status,
			finding_count = excluded.finding_count,
			audited_at = excluded.audited_at`,
		sourceFileID, session.ID, "session_jsonl", "completed", len(findings), now)
	if err != nil {
		return err
	}
	return nil
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

func findJSONLFiles(root string) ([]string, error) {
	return findJSONLFilesForSource(agent.ResolveSource(root))
}

func findJSONLFilesForSource(spec agent.SourceSpec) ([]string, error) {
	return findJSONLFilesFromSources(agent.UsageSources(spec))
}

func findJSONLFilesFromSources(sources []agent.UsageSource) ([]string, error) {
	var files []string
	seen := map[string]struct{}{}
	for _, source := range sources {
		err := filepath.WalkDir(source.Dir, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			if filepath.Ext(entry.Name()) != ".jsonl" {
				return nil
			}
			key := usageFileKey(source, path)
			if _, ok := seen[key]; ok {
				return nil
			}
			seen[key] = struct{}{}
			files = append(files, path)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(files)
	return files, nil
}

func usageFileKey(source agent.UsageSource, path string) string {
	relative, err := filepath.Rel(source.Dir, path)
	if err != nil {
		relative = path
	}
	return sourcepath.DedupeKey(source.DedupeScope, relative)
}

func isDir(path string) bool {
	stat, err := os.Stat(path)
	return err == nil && stat.IsDir()
}

func joinWarnings(warnings []string) string {
	if len(warnings) == 0 {
		return ""
	}
	if len(warnings) > 5 {
		warnings = warnings[:5]
	}
	result := ""
	for index, warning := range warnings {
		if index > 0 {
			result += "; "
		}
		result += warning
	}
	return result
}

func nullableFloat(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}

func firstNonZero(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
