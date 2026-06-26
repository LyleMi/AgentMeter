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
	"time"

	"AgentMeter/internal/codex"
	"AgentMeter/internal/db"
	"AgentMeter/internal/model"
	"AgentMeter/internal/platform"
	"AgentMeter/internal/pricing"
)

type Indexer struct {
	conn   *sql.DB
	dbPath string
}

type existingFile struct {
	ID          int64
	SizeBytes   int64
	ModifiedAt  time.Time
	ContentHash string
	ScanStatus  string
}

func New(conn *sql.DB, dbPath string) *Indexer {
	return &Indexer{conn: conn, dbPath: dbPath}
}

func (i *Indexer) Index(ctx context.Context, sessionsPath string, rebuild bool) (model.IndexResult, error) {
	start := time.Now()
	result := model.IndexResult{
		SourcePath: sessionsPath,
		Database:   i.dbPath,
		Rebuild:    rebuild,
	}
	if sessionsPath == "" {
		return result, errors.New("source path is empty")
	}
	stat, err := os.Stat(sessionsPath)
	if err != nil {
		return result, err
	}
	if !stat.IsDir() {
		return result, fmt.Errorf("%s is not a directory", sessionsPath)
	}

	rootPath := filepath.Dir(sessionsPath)
	source, err := db.EnsureSource(ctx, i.conn, rootPath, sessionsPath, platform.PlatformName())
	if err != nil {
		return result, err
	}
	if rebuild {
		if _, err := i.conn.ExecContext(ctx, `DELETE FROM source_files WHERE source_id = ?`, source.ID); err != nil {
			return result, err
		}
	}

	files, err := findJSONLFiles(sessionsPath)
	if err != nil {
		return result, err
	}
	result.FilesSeen = len(files)
	for _, path := range files {
		indexed, skipped, failed, sessions, warnings := i.indexFile(ctx, source, path, rebuild)
		result.Indexed += indexed
		result.Skipped += skipped
		result.Failed += failed
		result.Sessions += sessions
		result.Warnings = append(result.Warnings, warnings...)
	}
	result.DurationMS = time.Since(start).Milliseconds()
	return result, nil
}

func (i *Indexer) indexFile(ctx context.Context, source model.Source, path string, force bool) (indexed, skipped, failed, sessions int, warnings []string) {
	stat, err := os.Stat(path)
	if err != nil {
		return 0, 0, 1, 0, []string{fmt.Sprintf("%s: %v", path, err)}
	}
	modified := stat.ModTime().UTC()
	existing, err := i.getExistingFile(ctx, path)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, 0, 1, 0, []string{fmt.Sprintf("%s: %v", path, err)}
	}
	hash := ""
	if !force && err == nil && existing.SizeBytes == stat.Size() && existing.ModifiedAt.Equal(modified) && existing.ScanStatus == "indexed" {
		return 0, 1, 0, 0, nil
	}
	hash, err = codex.HashFile(path)
	if err != nil {
		return 0, 0, 1, 0, []string{fmt.Sprintf("%s: %v", path, err)}
	}
	if !force && err == nil && existing.ContentHash == hash && existing.ScanStatus == "indexed" {
		return 0, 1, 0, 0, nil
	}

	sourceFileID, err := i.upsertSourceFile(ctx, source.ID, path, stat.Size(), modified, hash, "scanning", "")
	if err != nil {
		return 0, 0, 1, 0, []string{fmt.Sprintf("%s: %v", path, err)}
	}
	parsed, err := codex.ParseFile(path, source.ID, sourceFileID)
	if err != nil {
		_ = i.markSourceFile(ctx, sourceFileID, "error", err.Error())
		return 0, 0, 1, 0, []string{fmt.Sprintf("%s: %v", path, err)}
	}
	if err := i.replaceParsedSession(ctx, sourceFileID, parsed); err != nil {
		_ = i.markSourceFile(ctx, sourceFileID, "error", err.Error())
		return 0, 0, 1, 0, []string{fmt.Sprintf("%s: %v", path, err)}
	}
	status := "indexed"
	message := ""
	if parsed.Session.ParseStatus == "warning" {
		status = "warning"
		message = joinWarnings(parsed.Warnings)
	}
	if err := i.markSourceFile(ctx, sourceFileID, status, message); err != nil {
		return 0, 0, 1, 0, []string{fmt.Sprintf("%s: %v", path, err)}
	}
	return 1, 0, 0, 1, parsed.Warnings
}

func (i *Indexer) getExistingFile(ctx context.Context, path string) (existingFile, error) {
	var item existingFile
	var modified string
	err := i.conn.QueryRowContext(ctx, `SELECT id, size_bytes, modified_at, content_hash, scan_status FROM source_files WHERE path = ?`, path).
		Scan(&item.ID, &item.SizeBytes, &modified, &item.ContentHash, &item.ScanStatus)
	if err != nil {
		return item, err
	}
	item.ModifiedAt = db.ParseTime(modified)
	return item, nil
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

func (i *Indexer) markSourceFile(ctx context.Context, sourceFileID int64, status, message string) error {
	_, err := i.conn.ExecContext(ctx, `UPDATE source_files SET scan_status = ?, error = ?, last_scanned_at = ? WHERE id = ?`,
		status, message, db.FormatTime(time.Now().UTC()), sourceFileID)
	return err
}

func (i *Indexer) replaceParsedSession(ctx context.Context, sourceFileID int64, parsed model.ParsedSession) error {
	modelCallCosts := make([]*float64, len(parsed.ModelCall))
	for index, call := range parsed.ModelCall {
		usage := model.Usage{
			Model:                 call.Model,
			InputTokens:           call.InputTokens,
			CachedInputTokens:     call.CachedInputTokens,
			OutputTokens:          call.OutputTokens,
			ReasoningOutputTokens: call.ReasoningOutputTokens,
			TotalTokens:           call.TotalTokens,
			Source:                "actual",
		}
		cost, _ := pricing.Compute(i.conn, usage)
		modelCallCosts[index] = cost
	}

	tx, err := i.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE source_file_id = ?`, sourceFileID); err != nil {
		return err
	}

	res, err := tx.ExecContext(ctx, `INSERT INTO sessions
		(source_id, source_file_id, codex_session_id, project_path, model, model_provider, originator, thread_source, agent_nickname, agent_role,
		 started_at, ended_at, wall_duration_ms, active_duration_ms, model_duration_ms, tool_duration_ms, idle_duration_ms, event_count, parse_status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		parsed.Session.SourceID,
		sourceFileID,
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

	rawEventIDs := map[int]int64{}
	for _, event := range parsed.Events {
		res, err := tx.ExecContext(ctx, `INSERT INTO events
			(session_id, source_file_id, source_line, timestamp, kind, raw_type, summary, raw_json)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			sessionID, sourceFileID, event.SourceLine, db.FormatTime(event.Timestamp), event.Kind, event.RawType, event.Summary, event.RawJSON)
		if err != nil {
			return err
		}
		eventID, _ := res.LastInsertId()
		rawEventIDs[event.SourceLine] = eventID
	}

	if parsed.Usage.Source == "" {
		parsed.Usage.Source = "unknown"
	}
	if parsed.Usage.Model == "" {
		parsed.Usage.Model = parsed.Session.Model
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO token_usage
		(owner_kind, owner_id, model, input_tokens, cached_input_tokens, output_tokens, reasoning_output_tokens, total_tokens, source)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"session", sessionID, parsed.Usage.Model, parsed.Usage.InputTokens, parsed.Usage.CachedInputTokens,
		parsed.Usage.OutputTokens, parsed.Usage.ReasoningOutputTokens, parsed.Usage.TotalTokens, parsed.Usage.Source)
	if err != nil {
		return err
	}

	for index, call := range parsed.ModelCall {
		_, err := tx.ExecContext(ctx, `INSERT INTO model_calls
			(session_id, started_at, ended_at, duration_ms, model, provider, status, input_tokens, cached_input_tokens, output_tokens, reasoning_output_tokens, total_tokens, cost_usd)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			sessionID, db.FormatTime(call.StartedAt), db.FormatTime(call.EndedAt), call.DurationMS, call.Model, call.Provider, call.Status,
			call.InputTokens, call.CachedInputTokens, call.OutputTokens, call.ReasoningOutputTokens, call.TotalTokens, nullableFloat(modelCallCosts[index]))
		if err != nil {
			return err
		}
	}

	for _, call := range parsed.ToolCall {
		rawEventID := rawEventIDs[call.RawEventLine]
		_, err := tx.ExecContext(ctx, `INSERT INTO tool_calls
			(session_id, started_at, ended_at, duration_ms, tool_name, status, input_summary, output_summary, error, raw_event_id)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			sessionID, db.FormatTime(call.StartedAt), db.FormatTime(call.EndedAt), call.DurationMS,
			call.ToolName, call.Status, call.InputSummary, call.OutputSummary, call.Error, rawEventID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func findJSONLFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if filepath.Ext(entry.Name()) == ".jsonl" {
			files = append(files, path)
		}
		return nil
	})
	sort.Strings(files)
	return files, err
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
