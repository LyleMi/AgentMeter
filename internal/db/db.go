package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"AgentMeter/internal/model"
	"AgentMeter/internal/pricing"

	_ "modernc.org/sqlite"
)

func Open(path string) (*sql.DB, error) {
	conn, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}
	conn.SetMaxOpenConns(4)
	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, err
	}
	if err := Migrate(context.Background(), conn); err != nil {
		conn.Close()
		return nil, err
	}
	if err := pricing.Seed(context.Background(), conn); err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}

func Migrate(ctx context.Context, conn *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS sources (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			kind TEXT NOT NULL,
			name TEXT NOT NULL,
			root_path TEXT NOT NULL,
			sessions_path TEXT NOT NULL,
			platform TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			UNIQUE(kind, sessions_path)
		)`,
		`CREATE TABLE IF NOT EXISTS source_files (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_id INTEGER NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
			path TEXT NOT NULL UNIQUE,
			size_bytes INTEGER NOT NULL,
			modified_at TEXT NOT NULL,
			content_hash TEXT NOT NULL,
			last_scanned_at TEXT NOT NULL,
			scan_status TEXT NOT NULL,
			error TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_id INTEGER NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
			source_file_id INTEGER NOT NULL REFERENCES source_files(id) ON DELETE CASCADE,
			session_key TEXT NOT NULL,
			codex_session_id TEXT NOT NULL,
			project_path TEXT NOT NULL,
			model TEXT NOT NULL,
			model_provider TEXT NOT NULL,
			originator TEXT NOT NULL,
			thread_source TEXT NOT NULL,
			agent_nickname TEXT NOT NULL,
			agent_role TEXT NOT NULL,
			started_at TEXT NOT NULL,
			ended_at TEXT NOT NULL,
			wall_duration_ms INTEGER NOT NULL,
			active_duration_ms INTEGER NOT NULL,
			model_duration_ms INTEGER NOT NULL,
			tool_duration_ms INTEGER NOT NULL,
			idle_duration_ms INTEGER NOT NULL,
			event_count INTEGER NOT NULL,
			parse_status TEXT NOT NULL,
			UNIQUE(source_file_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_started_at ON sessions(started_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_model ON sessions(model)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_source_started ON sessions(source_id, started_at DESC)`,
		`CREATE TABLE IF NOT EXISTS events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id INTEGER NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
			source_file_id INTEGER NOT NULL REFERENCES source_files(id) ON DELETE CASCADE,
			source_line INTEGER NOT NULL,
			timestamp TEXT NOT NULL,
			kind TEXT NOT NULL,
			raw_type TEXT NOT NULL,
			summary TEXT NOT NULL,
			raw_json TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_events_session_time ON events(session_id, timestamp)`,
		`CREATE TABLE IF NOT EXISTS token_usage (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			owner_kind TEXT NOT NULL,
			owner_id INTEGER NOT NULL,
			model TEXT NOT NULL,
			input_tokens INTEGER NOT NULL,
			cached_input_tokens INTEGER NOT NULL,
			output_tokens INTEGER NOT NULL,
			reasoning_output_tokens INTEGER NOT NULL,
			total_tokens INTEGER NOT NULL,
			source TEXT NOT NULL,
			UNIQUE(owner_kind, owner_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_token_usage_owner ON token_usage(owner_kind, owner_id)`,
		`DELETE FROM token_usage
			WHERE owner_kind = 'session'
			AND NOT EXISTS (SELECT 1 FROM sessions s WHERE s.id = token_usage.owner_id)`,
		`CREATE TABLE IF NOT EXISTS model_calls (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id INTEGER NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
			started_at TEXT NOT NULL,
			ended_at TEXT NOT NULL,
			duration_ms INTEGER NOT NULL,
			model TEXT NOT NULL,
			provider TEXT NOT NULL,
			status TEXT NOT NULL,
			input_tokens INTEGER NOT NULL,
			cached_input_tokens INTEGER NOT NULL,
			output_tokens INTEGER NOT NULL,
			reasoning_output_tokens INTEGER NOT NULL,
			total_tokens INTEGER NOT NULL,
			cost_usd REAL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_model_calls_session ON model_calls(session_id)`,
		`CREATE TABLE IF NOT EXISTS tool_calls (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id INTEGER NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
			started_at TEXT NOT NULL,
			ended_at TEXT NOT NULL,
			duration_ms INTEGER NOT NULL,
			tool_name TEXT NOT NULL,
			status TEXT NOT NULL,
			input_summary TEXT NOT NULL,
			output_summary TEXT NOT NULL,
			error TEXT NOT NULL,
			raw_event_id INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tool_calls_session ON tool_calls(session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tool_calls_name ON tool_calls(tool_name)`,
		`CREATE INDEX IF NOT EXISTS idx_tool_calls_started_at ON tool_calls(started_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_tool_calls_duration ON tool_calls(duration_ms DESC)`,
		`CREATE TABLE IF NOT EXISTS audit_runs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_file_id INTEGER NOT NULL REFERENCES source_files(id) ON DELETE CASCADE,
			session_id INTEGER NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
			source TEXT NOT NULL DEFAULT 'session_jsonl',
			status TEXT NOT NULL,
			finding_count INTEGER NOT NULL,
			audited_at TEXT NOT NULL,
			UNIQUE(source_file_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_runs_session ON audit_runs(session_id)`,
		`CREATE TABLE IF NOT EXISTS audit_findings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id INTEGER NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
			tool_call_id INTEGER NOT NULL DEFAULT 0,
			source_file_id INTEGER NOT NULL REFERENCES source_files(id) ON DELETE CASCADE,
			raw_event_id INTEGER NOT NULL DEFAULT 0,
			source_line INTEGER NOT NULL DEFAULT 0,
			timestamp TEXT NOT NULL,
			source TEXT NOT NULL DEFAULT 'session_jsonl',
			event_type TEXT NOT NULL DEFAULT 'finding',
			category TEXT NOT NULL,
			severity TEXT NOT NULL,
			rule_id TEXT NOT NULL,
			title TEXT NOT NULL,
			description TEXT NOT NULL,
			evidence TEXT NOT NULL,
			command TEXT NOT NULL,
			shell_family TEXT NOT NULL,
			platform TEXT NOT NULL,
			decision TEXT NOT NULL DEFAULT 'observed',
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_findings_session ON audit_findings(session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_findings_category_severity ON audit_findings(category, severity)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_findings_timestamp ON audit_findings(timestamp DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_findings_shell ON audit_findings(shell_family)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_findings_rule ON audit_findings(rule_id)`,
		`CREATE TABLE IF NOT EXISTS pricing_models (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			model TEXT NOT NULL,
			normalized_model TEXT NOT NULL UNIQUE,
			input_per_1m REAL NOT NULL,
			cached_input_per_1m REAL NOT NULL,
			output_per_1m REAL NOT NULL,
			source TEXT NOT NULL,
			effective_from TEXT NOT NULL,
			is_custom INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS app_config (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
	}
	for _, stmt := range statements {
		if _, err := conn.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	if err := ensureColumn(ctx, conn, "sessions", "session_key", "session_key TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := ensureColumn(ctx, conn, "tool_calls", "call_id", "call_id TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := ensureColumn(ctx, conn, "tool_calls", "raw_start_event_id", "raw_start_event_id INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := ensureColumn(ctx, conn, "tool_calls", "raw_end_event_id", "raw_end_event_id INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := ensureColumn(ctx, conn, "pricing_models", "is_custom", "is_custom INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, `UPDATE sessions SET session_key = codex_session_id WHERE session_key = ''`); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, `UPDATE tool_calls SET raw_start_event_id = raw_event_id WHERE raw_start_event_id = 0 AND raw_event_id != 0`); err != nil {
		return err
	}
	return nil
}

func EnsureSource(ctx context.Context, conn *sql.DB, kind, name, rootPath, sessionsPath, platform string) (model.Source, error) {
	now := time.Now().UTC()
	var src model.Source
	row := conn.QueryRowContext(ctx, `SELECT id, kind, name, root_path, sessions_path, platform, created_at, updated_at FROM sources WHERE kind = ? AND sessions_path = ?`, kind, sessionsPath)
	if err := scanSource(row, &src); err == nil {
		src.RootPath = rootPath
		src.Platform = platform
		src.Name = name
		src.UpdatedAt = now
		_, err := conn.ExecContext(ctx, `UPDATE sources SET name = ?, root_path = ?, platform = ?, updated_at = ? WHERE id = ?`, name, rootPath, platform, formatTime(now), src.ID)
		return src, err
	} else if !errors.Is(err, sql.ErrNoRows) {
		return src, err
	}

	res, err := conn.ExecContext(ctx, `INSERT INTO sources (kind, name, root_path, sessions_path, platform, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		kind, name, rootPath, sessionsPath, platform, formatTime(now), formatTime(now))
	if err != nil {
		return src, err
	}
	id, _ := res.LastInsertId()
	src = model.Source{
		ID:           id,
		Kind:         kind,
		Name:         name,
		RootPath:     rootPath,
		SessionsPath: sessionsPath,
		Platform:     platform,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	return src, nil
}

func ensureColumn(ctx context.Context, conn *sql.DB, table, column, definition string) error {
	rows, err := conn.QueryContext(ctx, `PRAGMA table_info(`+table+`)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, columnType string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = conn.ExecContext(ctx, `ALTER TABLE `+table+` ADD COLUMN `+definition)
	return err
}

func GetConfig(ctx context.Context, conn *sql.DB, key string) (string, bool, error) {
	var value string
	err := conn.QueryRowContext(ctx, `SELECT value FROM app_config WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return value, true, nil
}

func SetConfig(ctx context.Context, conn *sql.DB, key, value string) error {
	_, err := conn.ExecContext(ctx, `INSERT INTO app_config (key, value, updated_at) VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		key, value, formatTime(time.Now().UTC()))
	return err
}

func FormatTime(t time.Time) string {
	return formatTime(t)
}

func ParseTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return t
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func scanSource(row interface{ Scan(dest ...any) error }, src *model.Source) error {
	var created, updated string
	if err := row.Scan(&src.ID, &src.Kind, &src.Name, &src.RootPath, &src.SessionsPath, &src.Platform, &created, &updated); err != nil {
		return err
	}
	src.CreatedAt = ParseTime(created)
	src.UpdatedAt = ParseTime(updated)
	return nil
}

func Close(conn *sql.DB) error {
	if conn == nil {
		return nil
	}
	if err := conn.Close(); err != nil {
		return fmt.Errorf("close database: %w", err)
	}
	return nil
}
