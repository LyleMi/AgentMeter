package query

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"AgentMeter/internal/db"
	"AgentMeter/internal/model"
)

func TestToolCallsIncludeRawEventsAndSessionContext(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(filepath.Join(t.TempDir(), "agentmeter.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	now := time.Date(2026, 6, 27, 1, 2, 3, 0, time.UTC)
	sourceID := insertRow(t, conn, `INSERT INTO sources
		(kind, name, root_path, sessions_path, platform, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"codex", "Codex", "/workspace", "/workspace/.codex/sessions", "test", db.FormatTime(now), db.FormatTime(now))
	sourceFileID := insertRow(t, conn, `INSERT INTO source_files
		(source_id, path, size_bytes, modified_at, content_hash, last_scanned_at, scan_status, error)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		sourceID, "/workspace/.codex/sessions/run.jsonl", 128, db.FormatTime(now), "hash", db.FormatTime(now), "indexed", "")
	sessionID := insertRow(t, conn, `INSERT INTO sessions
		(source_id, source_file_id, session_key, codex_session_id, project_path, model, model_provider, originator, thread_source, agent_nickname, agent_role,
		 started_at, ended_at, wall_duration_ms, active_duration_ms, model_duration_ms, tool_duration_ms, idle_duration_ms, event_count, parse_status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sourceID, sourceFileID, "session-key", "codex-session", "/workspace/project", "gpt-5", "openai", "cli", "local", "", "",
		db.FormatTime(now), db.FormatTime(now.Add(2*time.Second)), 2000, 1500, 500, 1000, 500, 2, "ok")
	startEventID := insertRow(t, conn, `INSERT INTO events
		(session_id, source_file_id, source_line, timestamp, kind, raw_type, summary, raw_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		sessionID, sourceFileID, 10, db.FormatTime(now), "tool", "function_call", "Tool call: shell_command", `{"type":"function_call"}`)
	endEventID := insertRow(t, conn, `INSERT INTO events
		(session_id, source_file_id, source_line, timestamp, kind, raw_type, summary, raw_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		sessionID, sourceFileID, 11, db.FormatTime(now.Add(time.Second)), "tool", "function_call_output", "Tool output", `{"type":"function_call_output"}`)
	insertRow(t, conn, `INSERT INTO tool_calls
		(session_id, started_at, ended_at, duration_ms, tool_name, status, input_summary, output_summary, error, raw_event_id, call_id, raw_start_event_id, raw_end_event_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sessionID, db.FormatTime(now), db.FormatTime(now.Add(time.Second)), 1000, "shell_command", "completed", "go test ./...", "ok", "",
		startEventID, "call-1", startEventID, endEventID)

	service := New(conn)
	calls, err := service.ToolCalls(ctx, model.ToolCallFilters{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 {
		t.Fatalf("tool calls = %d", len(calls))
	}
	assertToolCallDetail(t, calls[0], sessionID, startEventID, endEventID)

	detail, err := service.SessionDetail(ctx, sessionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.ToolCalls) != 1 {
		t.Fatalf("session detail tool calls = %d", len(detail.ToolCalls))
	}
	assertToolCallDetail(t, detail.ToolCalls[0], sessionID, startEventID, endEventID)
}

func TestToolCallsFiltersByAgentTimeAndSortsByDuration(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(filepath.Join(t.TempDir(), "agentmeter.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	now := time.Date(2026, 6, 27, 1, 2, 3, 0, time.UTC)
	insertToolCallFixture(t, conn, "codex", "Codex", now, time.Second, "shell_command")
	middleID := insertToolCallFixture(t, conn, "codex", "Codex", now.Add(time.Hour), 2*time.Second, "read_file")
	insertToolCallFixture(t, conn, "claude", "Claude Code", now.Add(2*time.Hour), 5*time.Second, "Bash")

	service := New(conn)
	calls, err := service.ToolCalls(ctx, model.ToolCallFilters{Agent: "codex", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 {
		t.Fatalf("codex calls = %d", len(calls))
	}
	for _, call := range calls {
		if call.AgentKind != "codex" {
			t.Fatalf("agent filter returned %+v", call)
		}
	}

	calls, err = service.ToolCalls(ctx, model.ToolCallFilters{
		StartedFrom: db.FormatTime(now.Add(30 * time.Minute)),
		StartedTo:   db.FormatTime(now.Add(90 * time.Minute)),
		Limit:       10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].ID != middleID {
		t.Fatalf("time filtered calls = %+v", calls)
	}

	calls, err = service.ToolCalls(ctx, model.ToolCallFilters{Sort: "duration_desc", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 3 || calls[0].DurationMS != 5000 || calls[1].DurationMS != 2000 || calls[2].DurationMS != 1000 {
		t.Fatalf("duration desc calls = %+v", calls)
	}

	calls, err = service.ToolCalls(ctx, model.ToolCallFilters{Sort: "duration_asc", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 3 || calls[0].DurationMS != 1000 || calls[1].DurationMS != 2000 || calls[2].DurationMS != 5000 {
		t.Fatalf("duration asc calls = %+v", calls)
	}
}

func TestToolsFiltersByAgent(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(filepath.Join(t.TempDir(), "agentmeter.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	now := time.Date(2026, 6, 27, 1, 2, 3, 0, time.UTC)
	insertToolCallFixture(t, conn, "codex", "Codex", now, time.Second, "shell_command")
	insertToolCallFixture(t, conn, "codex", "Codex", now.Add(time.Hour), 2*time.Second, "read_file")
	insertToolCallFixture(t, conn, "claude", "Claude Code", now.Add(2*time.Hour), 5*time.Second, "Bash")

	service := New(conn)
	stats, err := service.Tools(ctx, model.ToolFilters{Agent: "codex"})
	if err != nil {
		t.Fatal(err)
	}
	if len(stats) != 2 {
		t.Fatalf("codex tools = %+v", stats)
	}
	for _, stat := range stats {
		if stat.ToolName == "Bash" {
			t.Fatalf("agent filtered tools included claude tool: %+v", stats)
		}
		if stat.Calls != 1 {
			t.Fatalf("tool calls for %s = %d, want 1", stat.ToolName, stat.Calls)
		}
	}
}

func assertToolCallDetail(t *testing.T, call model.ToolCall, sessionID, startEventID, endEventID int64) {
	t.Helper()
	if call.SessionID != sessionID || call.SessionKey != "session-key" || call.ProjectPath != "/workspace/project" {
		t.Fatalf("session context = %+v", call)
	}
	if call.CallID != "call-1" || call.RawStartEventID != startEventID || call.RawEndEventID != endEventID {
		t.Fatalf("raw event ids = %+v", call)
	}
	if call.RawStartEventLine != 10 || call.RawEndEventLine != 11 {
		t.Fatalf("raw event lines = %+v", call)
	}
	if call.RawStartEventJSON == "" || call.RawEndEventJSON == "" {
		t.Fatalf("raw event json missing = %+v", call)
	}
}

func insertToolCallFixture(t *testing.T, conn *sql.DB, kind, name string, started time.Time, duration time.Duration, toolName string) int64 {
	t.Helper()
	key := kind + "-" + started.Format("20060102150405")
	sourceID := insertRow(t, conn, `INSERT INTO sources
		(kind, name, root_path, sessions_path, platform, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		kind, name, "/workspace", "/workspace/"+key+"/sessions", "test", db.FormatTime(started), db.FormatTime(started))
	sourceFileID := insertRow(t, conn, `INSERT INTO source_files
		(source_id, path, size_bytes, modified_at, content_hash, last_scanned_at, scan_status, error)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		sourceID, "/workspace/"+key+"/run.jsonl", 128, db.FormatTime(started), "hash-"+key, db.FormatTime(started), "indexed", "")
	sessionID := insertRow(t, conn, `INSERT INTO sessions
		(source_id, source_file_id, session_key, codex_session_id, project_path, model, model_provider, originator, thread_source, agent_nickname, agent_role,
		 started_at, ended_at, wall_duration_ms, active_duration_ms, model_duration_ms, tool_duration_ms, idle_duration_ms, event_count, parse_status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sourceID, sourceFileID, "session-"+key, "codex-"+key, "/workspace/project", "gpt-5", "openai", "cli", "local", "", "",
		db.FormatTime(started), db.FormatTime(started.Add(duration)), duration.Milliseconds(), duration.Milliseconds(), 0, duration.Milliseconds(), 0, 1, "ok")
	return insertRow(t, conn, `INSERT INTO tool_calls
		(session_id, started_at, ended_at, duration_ms, tool_name, status, input_summary, output_summary, error, raw_event_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sessionID, db.FormatTime(started), db.FormatTime(started.Add(duration)), duration.Milliseconds(), toolName, "completed", "input", "output", "", 0)
}

func insertRow(t *testing.T, conn *sql.DB, query string, args ...any) int64 {
	t.Helper()
	result, err := conn.Exec(query, args...)
	if err != nil {
		t.Fatal(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	return id
}
