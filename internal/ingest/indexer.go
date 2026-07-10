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
	"github.com/LyleMi/AgentMeter/internal/db"
	"github.com/LyleMi/AgentMeter/internal/model"
	"github.com/LyleMi/AgentMeter/internal/platform"
	"github.com/LyleMi/AgentMeter/internal/pricing"
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

type sourceIndex struct {
	source model.Source
	files  []string
	run    indexRun
}

type sourceFileRecord struct {
	SourceID   int64
	Path       string
	SizeBytes  int64
	ModifiedAt time.Time
	Hash       string
	Status     string
	Message    string
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
	spec := sourceSpec(sessionsPath, label)
	result := model.IndexResult{
		SourcePath:  spec.SessionsPath,
		SourcePaths: []string{spec.SessionsPath},
		Database:    i.dbPath,
		Rebuild:     rebuild,
	}
	if err := validateSourcePath(spec.SessionsPath); err != nil {
		return result, err
	}
	index, err := i.prepareSourceIndex(ctx, spec, rebuild)
	if err != nil {
		return result, err
	}
	result.FilesSeen = len(index.files)
	for _, path := range index.files {
		fileResult := index.run.indexFile(ctx, index.source, path, rebuild)
		result.Indexed += fileResult.Indexed
		result.Skipped += fileResult.Skipped
		result.Failed += fileResult.Failed
		result.Sessions += fileResult.Sessions
		result.Warnings = append(result.Warnings, fileResult.Warnings...)
	}
	result.DurationMS = time.Since(start).Milliseconds()
	return result, nil
}

func sourceSpec(sessionsPath, label string) agent.SourceSpec {
	spec := agent.ResolveSource(sessionsPath)
	if cleanedLabel := strings.TrimSpace(label); cleanedLabel != "" {
		spec.Name = cleanedLabel
	}
	return spec
}

func validateSourcePath(path string) error {
	if path == "" {
		return errors.New("source path is empty")
	}
	stat, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !stat.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}
	return nil
}

func (i *Indexer) prepareSourceIndex(ctx context.Context, spec agent.SourceSpec, rebuild bool) (sourceIndex, error) {
	source, err := db.EnsureSource(ctx, i.conn, db.SourceInput{
		Kind:         spec.Kind,
		Name:         spec.Name,
		RootPath:     spec.RootPath,
		SessionsPath: spec.SessionsPath,
		Platform:     platform.PlatformName(),
	})
	if err != nil {
		return sourceIndex{}, err
	}
	if rebuild {
		if err := i.deleteSourceFilesForRebuild(ctx, source.ID); err != nil {
			return sourceIndex{}, err
		}
	}
	files, err := findJSONLFilesForSource(spec)
	if err != nil {
		return sourceIndex{}, err
	}
	calculator, err := pricing.LoadCalculator(ctx, i.conn)
	if err != nil {
		calculator = pricing.Calculator{}
	}
	existingFiles, err := i.loadExistingFiles(ctx, source.ID)
	if err != nil {
		return sourceIndex{}, err
	}
	return sourceIndex{
		source: source,
		files:  files,
		run: indexRun{
			indexer:       i,
			calculator:    calculator,
			existingFiles: existingFiles,
		},
	}, nil
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

func (i *Indexer) upsertSourceFile(ctx context.Context, record sourceFileRecord) (int64, error) {
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
		record.SourceID, record.Path, record.SizeBytes, db.FormatTime(record.ModifiedAt), record.Hash, db.FormatTime(now), record.Status, record.Message)
	if err != nil {
		return 0, err
	}
	var id int64
	err = i.conn.QueryRowContext(ctx, `SELECT id FROM source_files WHERE path = ?`, record.Path).Scan(&id)
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

type parsedSessionWriter struct {
	indexer      *Indexer
	ctx          context.Context
	tx           *sql.Tx
	source       model.Source
	sourceFileID int64
	parsed       model.ParsedSession
	calculator   pricing.Calculator
	session      model.Session
	events       indexedSessionEvents
	toolCalls    []model.ToolCall
}

func (i *Indexer) replaceParsedSession(ctx context.Context, source model.Source, sourceFileID int64, parsed model.ParsedSession, calculator pricing.Calculator) error {
	tx, err := i.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	writer := parsedSessionWriter{
		indexer:      i,
		ctx:          ctx,
		tx:           tx,
		source:       source,
		sourceFileID: sourceFileID,
		parsed:       parsed,
		calculator:   calculator,
	}
	if err := writer.replaceSessionData(); err != nil {
		return err
	}
	if err := writer.replaceCalls(); err != nil {
		return err
	}
	if err := writer.replaceAudit(); err != nil {
		return err
	}
	return tx.Commit()
}

func (w *parsedSessionWriter) replaceSessionData() error {
	if err := clearParsedSessionRows(w.ctx, w.tx, w.sourceFileID); err != nil {
		return err
	}
	session, err := insertParsedSession(w.ctx, w.tx, w.sourceFileID, w.parsed.Session)
	if err != nil {
		return err
	}
	w.session = session
	events, err := insertSessionEvents(w.ctx, w.tx, session.ID, w.sourceFileID, w.parsed.Events)
	if err != nil {
		return err
	}
	w.events = events
	return insertSessionUsage(w.ctx, w.tx, session.ID, w.parsed.Session.Model, w.parsed.Usage)
}

func (w *parsedSessionWriter) replaceCalls() error {
	modelCallCosts := calculateModelCallCosts(w.parsed.ModelCall, w.calculator)
	if err := insertModelCalls(w.ctx, w.tx, w.session.ID, w.parsed.ModelCall, modelCallCosts); err != nil {
		return err
	}
	toolCalls, err := insertToolCalls(w.ctx, w.tx, w.session, w.parsed.ToolCall, w.events)
	if err != nil {
		return err
	}
	w.toolCalls = toolCalls
	return nil
}

func (w *parsedSessionWriter) replaceAudit() error {
	return w.indexer.replaceAuditFindings(w.ctx, w.tx, auditReplacement{
		Source:       w.source,
		SourceFileID: w.sourceFileID,
		Session:      w.session,
		ToolCalls:    w.toolCalls,
		Events:       w.events.Events,
	})
}

func calculateModelCallCosts(calls []model.ModelCall, calculator pricing.Calculator) []*float64 {
	costs := make([]*float64, len(calls))
	for index, call := range calls {
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
		costs[index] = cost
	}
	return costs
}

func clearParsedSessionRows(ctx context.Context, tx *sql.Tx, sourceFileID int64) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM token_usage
		WHERE owner_kind = 'session'
		AND owner_id IN (SELECT id FROM sessions WHERE source_file_id = ?)`, sourceFileID); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE source_file_id = ?`, sourceFileID)
	return err
}

func insertParsedSession(ctx context.Context, tx *sql.Tx, sourceFileID int64, session model.Session) (model.Session, error) {
	res, err := tx.ExecContext(ctx, `INSERT INTO sessions
		(source_id, source_file_id, session_key, codex_session_id, project_path, model, model_provider, originator, thread_source, agent_nickname, agent_role,
		 started_at, ended_at, wall_duration_ms, active_duration_ms, model_duration_ms, tool_duration_ms, idle_duration_ms, event_count, parse_status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		session.SourceID,
		sourceFileID,
		session.SessionKey,
		session.CodexSessionID,
		session.ProjectPath,
		session.Model,
		session.ModelProvider,
		session.Originator,
		session.ThreadSource,
		session.AgentNickname,
		session.AgentRole,
		db.FormatTime(session.StartedAt),
		db.FormatTime(session.EndedAt),
		session.WallDurationMS,
		session.ActiveDurationMS,
		session.ModelDurationMS,
		session.ToolDurationMS,
		session.IdleDurationMS,
		session.EventCount,
		session.ParseStatus)
	if err != nil {
		return model.Session{}, err
	}
	sessionID, err := res.LastInsertId()
	if err != nil {
		return model.Session{}, err
	}
	session.ID = sessionID
	session.SourceFileID = sourceFileID
	return session, nil
}

type indexedSessionEvents struct {
	RawEventIDs  map[int]int64
	EventsByLine map[int]model.Event
	Events       []model.Event
}

func insertSessionEvents(ctx context.Context, tx *sql.Tx, sessionID, sourceFileID int64, events []model.Event) (indexedSessionEvents, error) {
	insertEvent, err := tx.PrepareContext(ctx, `INSERT INTO events
		(session_id, source_file_id, source_line, timestamp, kind, raw_type, summary, raw_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return indexedSessionEvents{}, err
	}
	defer insertEvent.Close()

	indexed := indexedSessionEvents{
		RawEventIDs:  map[int]int64{},
		EventsByLine: map[int]model.Event{},
		Events:       make([]model.Event, 0, len(events)),
	}
	for _, event := range events {
		res, err := insertEvent.ExecContext(ctx,
			sessionID, sourceFileID, event.SourceLine, db.FormatTime(event.Timestamp), event.Kind, event.RawType, event.Summary, event.RawJSON)
		if err != nil {
			return indexedSessionEvents{}, err
		}
		eventID, _ := res.LastInsertId()
		indexed.RawEventIDs[event.SourceLine] = eventID
		event.ID = eventID
		event.SessionID = sessionID
		event.SourceFileID = sourceFileID
		indexed.EventsByLine[event.SourceLine] = event
		indexed.Events = append(indexed.Events, event)
	}
	return indexed, nil
}

func insertSessionUsage(ctx context.Context, tx *sql.Tx, sessionID int64, sessionModel string, usage model.Usage) error {
	if usage.Source == "" {
		usage.Source = "unknown"
	}
	if usage.Model == "" {
		usage.Model = sessionModel
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO token_usage
		(owner_kind, owner_id, model, input_tokens, cached_input_tokens, output_tokens, reasoning_output_tokens, context_compression_tokens, total_tokens, source)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"session", sessionID, usage.Model, usage.InputTokens, usage.CachedInputTokens,
		usage.OutputTokens, usage.ReasoningOutputTokens, usage.ContextCompressionTokens, usage.TotalTokens, usage.Source)
	return err
}

func insertModelCalls(ctx context.Context, tx *sql.Tx, sessionID int64, calls []model.ModelCall, costs []*float64) error {
	insertModelCall, err := tx.PrepareContext(ctx, `INSERT INTO model_calls
		(session_id, started_at, ended_at, duration_ms, model, provider, status, input_tokens, cached_input_tokens, output_tokens, reasoning_output_tokens, context_compression_tokens, total_tokens, cost_usd)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer insertModelCall.Close()

	for index, call := range calls {
		_, err := insertModelCall.ExecContext(ctx,
			sessionID, db.FormatTime(call.StartedAt), db.FormatTime(call.EndedAt), call.DurationMS, call.Model, call.Provider, call.Status,
			call.InputTokens, call.CachedInputTokens, call.OutputTokens, call.ReasoningOutputTokens, call.ContextCompressionTokens, call.TotalTokens, nullableFloat(costs[index]))
		if err != nil {
			return err
		}
	}
	return nil
}

func insertToolCalls(ctx context.Context, tx *sql.Tx, session model.Session, calls []model.ToolCall, events indexedSessionEvents) ([]model.ToolCall, error) {
	insertToolCall, err := tx.PrepareContext(ctx, `INSERT INTO tool_calls
		(session_id, started_at, ended_at, duration_ms, tool_name, status, input_summary, output_summary, error, raw_event_id, call_id, raw_start_event_id, raw_end_event_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return nil, err
	}
	defer insertToolCall.Close()

	indexedToolCalls := make([]model.ToolCall, 0, len(calls))
	for _, call := range calls {
		rawStartEventLine := firstNonZero(call.RawStartEventLine, call.RawEventLine)
		rawStartEventID := events.RawEventIDs[rawStartEventLine]
		rawEndEventID := events.RawEventIDs[call.RawEndEventLine]
		res, err := insertToolCall.ExecContext(ctx,
			session.ID, db.FormatTime(call.StartedAt), db.FormatTime(call.EndedAt), call.DurationMS,
			call.ToolName, call.Status, call.InputSummary, call.OutputSummary, call.Error, rawStartEventID, call.CallID, rawStartEventID, rawEndEventID)
		if err != nil {
			return nil, err
		}
		toolCallID, _ := res.LastInsertId()
		call.ID = toolCallID
		call.SessionID = session.ID
		call.RawEventID = rawStartEventID
		call.RawStartEventID = rawStartEventID
		call.RawEndEventID = rawEndEventID
		call.ProjectPath = session.ProjectPath
		if startEvent, ok := events.EventsByLine[rawStartEventLine]; ok {
			call.RawStartEventLine = startEvent.SourceLine
			call.RawStartEventType = startEvent.RawType
			call.RawStartEventSummary = startEvent.Summary
			call.RawStartEventJSON = startEvent.RawJSON
		}
		if endEvent, ok := events.EventsByLine[call.RawEndEventLine]; ok {
			call.RawEndEventType = endEvent.RawType
			call.RawEndEventSummary = endEvent.Summary
			call.RawEndEventJSON = endEvent.RawJSON
		}
		indexedToolCalls = append(indexedToolCalls, call)
	}
	return indexedToolCalls, nil
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
