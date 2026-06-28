package query

import (
	"context"
	"database/sql"
	"math"
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

func TestAuditFindingsFiltersByAgentAndReturnsDetail(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(filepath.Join(t.TempDir(), "agentmeter.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	now := time.Date(2026, 6, 27, 1, 2, 3, 0, time.UTC)
	codexSessionID, codexFileID := insertAuditSession(t, conn, "codex", "Codex", now, "codex-session")
	claudeSessionID, claudeFileID := insertAuditSession(t, conn, "claude", "Claude Code", now.Add(time.Minute), "claude-session")

	codexFindingID := insertAuditFinding(t, conn, codexSessionID, codexFileID, now, "command", "high", "shell-risk", "rm -rf /tmp/example")
	insertAuditFinding(t, conn, codexSessionID, codexFileID, now.Add(time.Second), "privacy", "medium", "privacy-risk", "")
	insertAuditFinding(t, conn, claudeSessionID, claudeFileID, now.Add(2*time.Second), "command", "critical", "claude-risk", "curl https://example.com")

	service := New(conn)
	summary, err := service.AuditSummaryWithFilters(ctx, model.AuditFindingFilters{Agent: "codex"})
	if err != nil {
		t.Fatal(err)
	}
	if summary.TotalFindings != 2 || summary.HighFindings != 1 || summary.MediumFindings != 1 || summary.CriticalFindings != 0 {
		t.Fatalf("codex summary severities = %+v", summary)
	}
	if summary.CommandFindings != 1 || summary.PrivacyFindings != 1 || summary.SessionsWithFindings != 1 {
		t.Fatalf("codex summary categories/sessions = %+v", summary)
	}
	if len(summary.RecentFindings) != 2 {
		t.Fatalf("codex recent findings = %+v", summary.RecentFindings)
	}
	for _, finding := range summary.RecentFindings {
		if finding.AgentKind != "codex" {
			t.Fatalf("summary recent finding ignored agent filter: %+v", finding)
		}
	}

	findings, err := service.AuditFindings(ctx, model.AuditFindingFilters{Agent: "codex", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 2 {
		t.Fatalf("codex findings = %+v", findings)
	}
	for _, finding := range findings {
		if finding.AgentKind != "codex" {
			t.Fatalf("agent filtered findings included other agent: %+v", findings)
		}
	}

	detail, err := service.AuditFinding(ctx, codexFindingID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.ID != codexFindingID || detail.SessionID != codexSessionID || detail.SessionKey != "codex-session" {
		t.Fatalf("audit detail session association = %+v", detail)
	}
	if detail.AgentKind != "codex" || detail.AgentName != "Codex" || detail.RawSourcePath == "" {
		t.Fatalf("audit detail source context = %+v", detail)
	}
}

func TestTokenAnalyticsAggregatesUsageCostsAndSessions(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(filepath.Join(t.TempDir(), "agentmeter.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	now := time.Date(2026, 6, 27, 1, 2, 3, 0, time.UTC)
	codexSourceID := insertTimeSource(t, conn, "codex", "Codex", now)
	claudeSourceID := insertTimeSource(t, conn, "claude", "Claude Code", now)

	sessionA := insertTokenAnalyticsSession(t, conn, codexSourceID, now, "codex-a", "gpt-5", "gpt-5", "actual", 1_000_000, 200_000, 500_000, 50_000, 1_550_000)
	sessionB := insertTokenAnalyticsSession(t, conn, codexSourceID, now.Add(time.Hour), "codex-b", "unknown-model", "unknown-model", "actual", 100, 20, 30, 0, 130)
	sessionC := insertTokenAnalyticsSession(t, conn, claudeSourceID, now.Add(2*time.Hour), "claude-c", "gpt-5-mini", "gpt-5-mini", "actual", 500_000, 100_000, 250_000, 25_000, 775_000)

	analytics, err := New(conn).TokenAnalytics(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if analytics.TotalSessions != 3 || analytics.TotalInputTokens != 1_500_100 || analytics.TotalCachedInputTokens != 300_020 ||
		analytics.TotalOutputTokens != 750_030 || analytics.TotalReasoningTokens != 75_000 || analytics.TotalTokens != 2_325_130 {
		t.Fatalf("token analytics totals = %+v", analytics)
	}
	wantCacheRate := float64(300_020) / float64(1_500_100)
	if math.Abs(analytics.CacheUtilizationRate-wantCacheRate) > 0.000001 {
		t.Fatalf("cache utilization = %f, want %f", analytics.CacheUtilizationRate, wantCacheRate)
	}
	if analytics.UnpricedCount != 1 {
		t.Fatalf("unpriced count = %d", analytics.UnpricedCount)
	}
	if analytics.EstimatedCostUSD == nil || math.Abs(*analytics.EstimatedCostUSD-6.6275) > 0.000001 {
		t.Fatalf("estimated cost = %v", analytics.EstimatedCostUSD)
	}

	gptUsage := findModelUsage(t, analytics.ModelUsage, "gpt-5")
	if gptUsage.SessionCount != 1 || gptUsage.TotalTokens != 1_550_000 || gptUsage.CachedInputTokens != 200_000 ||
		gptUsage.ReasoningOutputTokens != 50_000 || gptUsage.Unpriced || gptUsage.EstimatedCostUSD == nil {
		t.Fatalf("gpt model usage = %+v", gptUsage)
	}
	unknownUsage := findModelUsage(t, analytics.ModelUsage, "unknown-model")
	if !unknownUsage.Unpriced || unknownUsage.EstimatedCostUSD != nil {
		t.Fatalf("unknown model usage pricing = %+v", unknownUsage)
	}

	codexUsage := findAgentUsage(t, analytics.AgentUsage, "codex", "Codex")
	if codexUsage.SessionCount != 2 || codexUsage.TotalTokens != 1_550_130 || codexUsage.CachedInputTokens != 200_020 ||
		codexUsage.ReasoningOutputTokens != 50_000 || !codexUsage.Unpriced {
		t.Fatalf("codex agent usage = %+v", codexUsage)
	}
	claudeUsage := findAgentUsage(t, analytics.AgentUsage, "claude", "Claude Code")
	if claudeUsage.SessionCount != 1 || claudeUsage.TotalTokens != 775_000 || claudeUsage.Unpriced {
		t.Fatalf("claude agent usage = %+v", claudeUsage)
	}

	if len(analytics.RecentSessions) != 3 || analytics.RecentSessions[0].ID != sessionC || analytics.RecentSessions[1].ID != sessionB || analytics.RecentSessions[2].ID != sessionA {
		t.Fatalf("recent sessions = %+v", analytics.RecentSessions)
	}
	if len(analytics.HighTokenSessions) != 3 || analytics.HighTokenSessions[0].ID != sessionA || analytics.HighTokenSessions[1].ID != sessionC || analytics.HighTokenSessions[2].ID != sessionB {
		t.Fatalf("high token sessions = %+v", analytics.HighTokenSessions)
	}
}

func TestTokenAnalyticsWithFiltersScopesTotalsAndSlices(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(filepath.Join(t.TempDir(), "agentmeter.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	now := time.Date(2026, 6, 27, 1, 2, 3, 0, time.UTC)
	codexSourceID := insertTimeSource(t, conn, "codex", "Codex", now)
	claudeSourceID := insertTimeSource(t, conn, "claude", "Claude Code", now)

	sessionA := insertTokenAnalyticsSession(t, conn, codexSourceID, now, "codex-a", "gpt-5", "gpt-5", "actual", 1_000, 200, 500, 50, 1_550)
	insertTokenAnalyticsSession(t, conn, codexSourceID, now.Add(time.Hour), "codex-b", "unknown-model", "unknown-model", "actual", 100, 20, 30, 0, 130)
	insertTokenAnalyticsSession(t, conn, claudeSourceID, now.Add(2*time.Hour), "claude-c", "gpt-5-mini", "gpt-5-mini", "actual", 500, 100, 250, 25, 775)

	analytics, err := New(conn).TokenAnalyticsWithFilters(ctx, model.AnalyticsFilters{
		Agent:       sourceInstanceKey(codexSourceID),
		Model:       "gpt-5",
		StartedFrom: db.FormatTime(now.Add(-time.Minute)),
		StartedTo:   db.FormatTime(now.Add(30 * time.Minute)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if analytics.TotalSessions != 1 || analytics.TotalInputTokens != 1_000 || analytics.TotalCachedInputTokens != 200 ||
		analytics.TotalOutputTokens != 500 || analytics.TotalReasoningTokens != 50 || analytics.TotalTokens != 1_550 {
		t.Fatalf("filtered token analytics totals = %+v", analytics)
	}
	if math.Abs(analytics.CacheUtilizationRate-0.2) > 0.000001 {
		t.Fatalf("filtered cache utilization = %f", analytics.CacheUtilizationRate)
	}
	if analytics.UnpricedCount != 0 || analytics.EstimatedCostUSD == nil {
		t.Fatalf("filtered pricing = cost %v unpriced %d", analytics.EstimatedCostUSD, analytics.UnpricedCount)
	}
	if len(analytics.ModelUsage) != 1 || analytics.ModelUsage[0].Model != "gpt-5" {
		t.Fatalf("filtered model usage = %+v", analytics.ModelUsage)
	}
	if len(analytics.AgentUsage) != 1 || analytics.AgentUsage[0].SourceID != codexSourceID {
		t.Fatalf("filtered agent usage = %+v", analytics.AgentUsage)
	}
	if len(analytics.RecentSessions) != 1 || analytics.RecentSessions[0].ID != sessionA {
		t.Fatalf("filtered recent sessions = %+v", analytics.RecentSessions)
	}
	if len(analytics.HighTokenSessions) != 1 || analytics.HighTokenSessions[0].ID != sessionA {
		t.Fatalf("filtered high token sessions = %+v", analytics.HighTokenSessions)
	}

	empty, err := New(conn).TokenAnalyticsWithFilters(ctx, model.AnalyticsFilters{Agent: "source:not-a-number"})
	if err != nil {
		t.Fatal(err)
	}
	if empty.TotalSessions != 0 || empty.TotalTokens != 0 || len(empty.RecentSessions) != 0 {
		t.Fatalf("invalid source filter should be empty, got %+v", empty)
	}
}

func TestSourceInstanceAggregationAndFilters(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(filepath.Join(t.TempDir(), "agentmeter.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	now := time.Date(2026, 6, 27, 1, 2, 3, 0, time.UTC)
	codexStable := insertSource(t, conn, "codex", "Codex", "/home/me/.codex", "/home/me/.codex/sessions", now)
	codexNightly := insertSource(t, conn, "codex", "Codex nightly", "/home/me/.ycodex", "/home/me/.ycodex/sessions", now)
	claude := insertSource(t, conn, "claude", "Claude Code", "/home/me/.claude", "/home/me/.claude/projects", now)

	stableSession := insertTokenAnalyticsSession(t, conn, codexStable, now, "stable", "gpt-5", "gpt-5", "actual", 100, 0, 20, 0, 120)
	nightlySession := insertTokenAnalyticsSession(t, conn, codexNightly, now.Add(time.Minute), "nightly", "gpt-5", "gpt-5", "actual", 300, 0, 40, 0, 340)
	insertTokenAnalyticsSession(t, conn, claude, now.Add(2*time.Minute), "claude", "gpt-5", "gpt-5", "actual", 500, 0, 60, 0, 560)
	insertOverviewToolCall(t, conn, stableSession, now, 100, "shell_command", "completed", "go test ./...")
	insertOverviewToolCall(t, conn, nightlySession, now.Add(time.Minute), 200, "read_file", "completed", "file")

	service := New(conn)
	analytics, err := service.TokenAnalytics(ctx)
	if err != nil {
		t.Fatal(err)
	}
	stableUsage := findAgentUsageBySource(t, analytics.AgentUsage, codexStable)
	if stableUsage.SourceKey != "source:1" && stableUsage.SourceKey != sourceInstanceKey(codexStable) {
		t.Fatalf("stable source key = %+v", stableUsage)
	}
	if stableUsage.SourceLabel != "Codex" || stableUsage.SourceRootPath != "/home/me/.codex" || stableUsage.SessionCount != 1 {
		t.Fatalf("stable usage = %+v", stableUsage)
	}
	nightlyUsage := findAgentUsageBySource(t, analytics.AgentUsage, codexNightly)
	if nightlyUsage.SourceLabel != "Codex nightly" || nightlyUsage.TotalTokens != 340 || nightlyUsage.SessionCount != 1 {
		t.Fatalf("nightly usage = %+v", nightlyUsage)
	}

	sessions, err := service.Sessions(ctx, model.SessionFilters{Agent: sourceInstanceKey(codexNightly), Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 || sessions[0].SourceID != codexNightly || sessions[0].SourceLabel != "Codex nightly" {
		t.Fatalf("source filtered sessions = %+v", sessions)
	}
	sessions, err = service.Sessions(ctx, model.SessionFilters{Agent: "codex", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 2 {
		t.Fatalf("family filtered sessions = %+v", sessions)
	}
	tools, err := service.Tools(ctx, model.ToolFilters{Agent: sourceInstanceKey(codexNightly)})
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) != 1 || tools[0].ToolName != "read_file" {
		t.Fatalf("source filtered tools = %+v", tools)
	}
}

func TestSourceScopedTokenAnalyticsSumMatchesUnfilteredTotals(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(filepath.Join(t.TempDir(), "agentmeter.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	now := time.Date(2026, 6, 27, 1, 2, 3, 0, time.UTC)
	sourceIDs := []int64{
		insertSource(t, conn, "codex", "Codex", "/home/me/.codex", "/home/me/.codex/sessions", now),
		insertSource(t, conn, "codex", "Codex nightly", "/home/me/.ycodex", "/home/me/.ycodex/sessions", now),
		insertSource(t, conn, "claude", "Claude Code", "/home/me/.claude", "/home/me/.claude/projects", now),
	}
	insertTokenAnalyticsSession(t, conn, sourceIDs[0], now, "stable-a", "gpt-5", "gpt-5", "actual", 100, 10, 20, 5, 125)
	insertTokenAnalyticsSession(t, conn, sourceIDs[0], now.Add(time.Minute), "stable-b", "gpt-5-mini", "gpt-5-mini", "actual", 200, 20, 30, 6, 256)
	insertTokenAnalyticsSession(t, conn, sourceIDs[1], now.Add(2*time.Minute), "nightly", "gpt-5", "gpt-5", "actual", 300, 30, 40, 7, 377)
	insertTokenAnalyticsSession(t, conn, sourceIDs[2], now.Add(3*time.Minute), "claude", "claude-sonnet", "claude-sonnet", "actual", 400, 40, 50, 8, 498)

	service := New(conn)
	unfiltered, err := service.TokenAnalytics(ctx)
	if err != nil {
		t.Fatal(err)
	}

	var scopedSessions int
	var scopedInput int64
	var scopedCached int64
	var scopedOutput int64
	var scopedReasoning int64
	var scopedTotal int64
	for _, sourceID := range sourceIDs {
		scoped, err := service.TokenAnalyticsWithFilters(ctx, model.AnalyticsFilters{Agent: sourceInstanceKey(sourceID)})
		if err != nil {
			t.Fatal(err)
		}
		scopedSessions += scoped.TotalSessions
		scopedInput += scoped.TotalInputTokens
		scopedCached += scoped.TotalCachedInputTokens
		scopedOutput += scoped.TotalOutputTokens
		scopedReasoning += scoped.TotalReasoningTokens
		scopedTotal += scoped.TotalTokens
	}

	if scopedSessions != unfiltered.TotalSessions || scopedInput != unfiltered.TotalInputTokens ||
		scopedCached != unfiltered.TotalCachedInputTokens || scopedOutput != unfiltered.TotalOutputTokens ||
		scopedReasoning != unfiltered.TotalReasoningTokens || scopedTotal != unfiltered.TotalTokens {
		t.Fatalf("source scoped totals = sessions %d input %d cached %d output %d reasoning %d total %d; unfiltered = %+v",
			scopedSessions, scopedInput, scopedCached, scopedOutput, scopedReasoning, scopedTotal, unfiltered)
	}
}

func TestTokenAnalyticsIgnoresOrphanTokenUsage(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(filepath.Join(t.TempDir(), "agentmeter.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	now := time.Date(2026, 6, 27, 1, 2, 3, 0, time.UTC)
	codexSourceID := insertSource(t, conn, "codex", "Codex", "/home/me/.codex", "/home/me/.codex/sessions", now)
	claudeSourceID := insertSource(t, conn, "claude", "Claude Code", "/home/me/.claude", "/home/me/.claude/projects", now)
	insertTokenAnalyticsSession(t, conn, codexSourceID, now, "codex", "gpt-5", "gpt-5", "actual", 1_000_000, 100_000, 200_000, 0, 1_200_000)
	insertTokenAnalyticsSession(t, conn, claudeSourceID, now.Add(time.Minute), "claude", "gpt-5-mini", "gpt-5-mini", "actual", 500_000, 50_000, 100_000, 0, 600_000)
	insertRow(t, conn, `INSERT INTO token_usage
		(owner_kind, owner_id, model, input_tokens, cached_input_tokens, output_tokens, reasoning_output_tokens, total_tokens, source)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"session", int64(9_999_999), "gpt-5", 100_000_000, 0, 100_000_000, 0, 200_000_000, "actual")

	service := New(conn)
	unfiltered, err := service.TokenAnalytics(ctx)
	if err != nil {
		t.Fatal(err)
	}
	codexScoped, err := service.TokenAnalyticsWithFilters(ctx, model.AnalyticsFilters{Agent: sourceInstanceKey(codexSourceID)})
	if err != nil {
		t.Fatal(err)
	}
	claudeScoped, err := service.TokenAnalyticsWithFilters(ctx, model.AnalyticsFilters{Agent: sourceInstanceKey(claudeSourceID)})
	if err != nil {
		t.Fatal(err)
	}

	scopedTotal := codexScoped.TotalTokens + claudeScoped.TotalTokens
	if unfiltered.TotalSessions != 2 || unfiltered.TotalTokens != scopedTotal {
		t.Fatalf("unfiltered analytics included orphan usage: unfiltered=%+v scopedTotal=%d", unfiltered, scopedTotal)
	}
	scopedCost := valueCost(codexScoped.EstimatedCostUSD) + valueCost(claudeScoped.EstimatedCostUSD)
	if math.Abs(valueCost(unfiltered.EstimatedCostUSD)-scopedCost) > 0.000001 {
		t.Fatalf("unfiltered cost should match source-scoped cost sum: unfiltered=%v scoped=%.6f", unfiltered.EstimatedCostUSD, scopedCost)
	}
	gptUsage := findModelUsage(t, unfiltered.ModelUsage, "gpt-5")
	if gptUsage.TotalTokens != codexScoped.TotalTokens {
		t.Fatalf("model usage included orphan usage: %+v", gptUsage)
	}
}

func TestOverviewWithFiltersScopesTotalsAndLists(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(filepath.Join(t.TempDir(), "agentmeter.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	now := time.Date(2026, 6, 27, 1, 2, 3, 0, time.UTC)
	codexSourceID := insertTimeSource(t, conn, "codex", "Codex", now)
	claudeSourceID := insertTimeSource(t, conn, "claude", "Claude Code", now)

	sessionA := insertOverviewTimeSession(t, conn, codexSourceID, now, "session-a", "gpt-5", 10_000, 7_000, 4_000, 2_000, 3_000, 1_000)
	sessionB := insertOverviewTimeSession(t, conn, codexSourceID, now.Add(time.Minute), "session-b", "gpt-5", 20_000, 15_000, 10_000, 4_000, 5_000, 2_000)
	sessionC := insertOverviewTimeSession(t, conn, claudeSourceID, now.Add(2*time.Minute), "session-c", "claude-sonnet", 15_000, 10_000, 8_000, 1_000, 5_000, 300)
	insertOverviewToolCall(t, conn, sessionA, now, 1_200, "shell_command", "completed", "curl https://example.com")
	insertOverviewToolCall(t, conn, sessionB, now.Add(time.Minute), 2_500, "web.run", "completed", "search latest docs")
	insertOverviewToolCall(t, conn, sessionC, now.Add(2*time.Minute), 900, "Bash", "failed", "npm ci")

	overview, err := New(conn).OverviewWithFilters(ctx, model.AnalyticsFilters{Agent: sourceInstanceKey(codexSourceID)})
	if err != nil {
		t.Fatal(err)
	}
	if overview.TotalSessions != 2 || overview.TotalTokens != 3_000 || overview.TotalWallDurationMS != 30_000 || overview.TotalToolCalls != 2 {
		t.Fatalf("source filtered overview totals = %+v", overview)
	}
	if overview.SuspectedNetworkToolDurationMS != 3_700 || overview.SuspectedNetworkToolCalls != 2 {
		t.Fatalf("source filtered network totals = duration %d calls %d", overview.SuspectedNetworkToolDurationMS, overview.SuspectedNetworkToolCalls)
	}
	if len(overview.RecentSessions) != 2 || overview.RecentSessions[0].ID != sessionB || overview.RecentSessions[1].ID != sessionA {
		t.Fatalf("source filtered recent sessions = %+v", overview.RecentSessions)
	}
	if len(overview.SlowSessions) != 2 || overview.SlowSessions[0].ID != sessionB || overview.SlowSessions[1].ID != sessionA {
		t.Fatalf("source filtered slow sessions = %+v", overview.SlowSessions)
	}
	if len(overview.DailyUsage) != 1 || overview.DailyUsage[0].SessionCount != 2 || overview.DailyUsage[0].TotalTokens != 3_000 {
		t.Fatalf("source filtered daily usage = %+v", overview.DailyUsage)
	}
	if len(overview.AgentUsage) != 1 || overview.AgentUsage[0].SourceID != codexSourceID {
		t.Fatalf("source filtered agent usage = %+v", overview.AgentUsage)
	}

	overview, err = New(conn).OverviewWithFilters(ctx, model.AnalyticsFilters{StartedFrom: db.FormatTime(now.Add(90 * time.Second))})
	if err != nil {
		t.Fatal(err)
	}
	if overview.TotalSessions != 1 || overview.TotalTokens != 300 || len(overview.RecentSessions) != 1 || overview.RecentSessions[0].ID != sessionC {
		t.Fatalf("time filtered overview = %+v", overview)
	}
}

func TestOverviewDateOnlyToIncludesWholeDay(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(filepath.Join(t.TempDir(), "agentmeter.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	firstDay := time.Date(2026, 6, 27, 1, 2, 3, 0, time.UTC)
	sourceID := insertTimeSource(t, conn, "codex", "Codex", firstDay)
	insertOverviewTimeSession(t, conn, sourceID, firstDay, "early", "gpt-5", 1_000, 1_000, 1_000, 0, 0, 100)
	insertOverviewTimeSession(t, conn, sourceID, firstDay.Add(22*time.Hour), "late", "gpt-5", 1_000, 1_000, 1_000, 0, 0, 200)
	insertOverviewTimeSession(t, conn, sourceID, firstDay.Add(24*time.Hour), "next-day", "gpt-5", 1_000, 1_000, 1_000, 0, 0, 300)

	overview, err := New(conn).OverviewWithFilters(ctx, model.AnalyticsFilters{StartedTo: "2026-06-27"})
	if err != nil {
		t.Fatal(err)
	}
	if overview.TotalSessions != 2 || overview.TotalTokens != 300 {
		t.Fatalf("date-only to should include all of 2026-06-27, got %+v", overview)
	}
}

func TestOverviewDailyUsageIncludesCachedInputAndCacheRate(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(filepath.Join(t.TempDir(), "agentmeter.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	now := time.Date(2026, 6, 27, 1, 2, 3, 0, time.UTC)
	sourceID := insertTimeSource(t, conn, "codex", "Codex", now)
	insertOverviewUsageSession(t, conn, sourceID, now, "day-one-a", "gpt-5", "gpt-5", "actual", 1_000, 250, 300, 1_300)
	insertOverviewUsageSession(t, conn, sourceID, now.Add(time.Hour), "day-one-b", "gpt-5", "gpt-5", "actual", 500, 50, 100, 600)
	insertOverviewUsageSession(t, conn, sourceID, now.Add(24*time.Hour), "day-two", "gpt-5", "gpt-5", "actual", 0, 0, 200, 200)

	overview, err := New(conn).Overview(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(overview.DailyUsage) != 2 {
		t.Fatalf("daily usage = %+v", overview.DailyUsage)
	}
	firstDay := overview.DailyUsage[0]
	if firstDay.Date != "2026-06-27" || firstDay.SessionCount != 2 || firstDay.TotalTokens != 1_900 ||
		firstDay.InputTokens != 1_500 || firstDay.CachedInputTokens != 300 || firstDay.OutputTokens != 400 {
		t.Fatalf("first daily usage bucket = %+v", firstDay)
	}
	if math.Abs(firstDay.CacheUtilizationRate-0.2) > 0.000001 {
		t.Fatalf("first daily cache utilization = %f", firstDay.CacheUtilizationRate)
	}
	secondDay := overview.DailyUsage[1]
	if secondDay.Date != "2026-06-28" || secondDay.InputTokens != 0 || secondDay.CachedInputTokens != 0 || secondDay.CacheUtilizationRate != 0 {
		t.Fatalf("second daily usage bucket = %+v", secondDay)
	}
}

func TestCacheHitTrendScopesProjectFillsGapsAndWeightsByInput(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(filepath.Join(t.TempDir(), "agentmeter.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	now := time.Date(2026, 6, 25, 1, 2, 3, 0, time.UTC)
	sourceID := insertTimeSource(t, conn, "codex", "Codex", now)
	apiFirst := insertTokenAnalyticsSession(t, conn, sourceID, now, "api-first", "gpt-5", "gpt-5", "actual", 10_000, 1_000, 2_000, 0, 12_000)
	apiTiny := insertTokenAnalyticsSession(t, conn, sourceID, now.Add(48*time.Hour), "api-tiny", "gpt-5", "gpt-5", "actual", 100, 100, 20, 0, 120)
	webSession := insertTokenAnalyticsSession(t, conn, sourceID, now.Add(48*time.Hour), "web", "gpt-5", "gpt-5", "actual", 50_000, 10_000, 3_000, 0, 53_000)
	setSessionProjectPath(t, conn, apiFirst, "/workspace/api")
	setSessionProjectPath(t, conn, apiTiny, "/workspace/api/.")
	setSessionProjectPath(t, conn, webSession, "/workspace/web")

	service := New(conn)
	analytics, err := service.TokenAnalyticsWithFilters(ctx, model.AnalyticsFilters{Project: "/workspace/api"})
	if err != nil {
		t.Fatal(err)
	}
	if analytics.TotalSessions != 2 || analytics.TotalInputTokens != 10_100 || analytics.TotalCachedInputTokens != 1_100 {
		t.Fatalf("project-scoped token analytics = %+v", analytics)
	}
	if len(analytics.CacheHitTrend) != 3 {
		t.Fatalf("cache hit trend = %+v", analytics.CacheHitTrend)
	}
	firstDay := analytics.CacheHitTrend[0]
	if firstDay.Date != "2026-06-25" || !firstDay.HasUsage || firstDay.LowInputVolume || math.Abs(firstDay.CacheUtilizationRate-0.1) > 0.000001 {
		t.Fatalf("first trend point = %+v", firstDay)
	}
	gapDay := analytics.CacheHitTrend[1]
	if gapDay.Date != "2026-06-26" || gapDay.HasUsage || gapDay.InputTokens != 0 || math.Abs(gapDay.RollingCacheUtilizationRate-0.1) > 0.000001 {
		t.Fatalf("gap trend point = %+v", gapDay)
	}
	tinyDay := analytics.CacheHitTrend[2]
	if tinyDay.Date != "2026-06-27" || !tinyDay.HasUsage || !tinyDay.LowInputVolume || math.Abs(tinyDay.CacheUtilizationRate-1) > 0.000001 {
		t.Fatalf("tiny trend point = %+v", tinyDay)
	}
	weighted := float64(1_100) / float64(10_100)
	if math.Abs(tinyDay.RollingCacheUtilizationRate-weighted) > 0.000001 {
		t.Fatalf("weighted rolling cache utilization = %f, want %f", tinyDay.RollingCacheUtilizationRate, weighted)
	}

	overview, err := service.OverviewWithFilters(ctx, model.AnalyticsFilters{Project: "/workspace/api"})
	if err != nil {
		t.Fatal(err)
	}
	if overview.TotalSessions != 2 || len(overview.CacheHitTrend) != 3 {
		t.Fatalf("project-scoped overview trend = %+v", overview)
	}
}

func TestOverviewPricingIgnoresZeroTokenUnknownUsage(t *testing.T) {
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
	insertOverviewUsageSession(t, conn, sourceID, now, "actual-gpt", "gpt-5.5", "gpt-5.5", "actual", 1_000_000, 200_000, 500_000, 1_500_000)
	insertOverviewUsageSession(t, conn, sourceID, now.Add(time.Minute), "empty-gpt", "gpt-5.5", "gpt-5.5", "unknown", 0, 0, 0, 0)
	insertOverviewUsageSession(t, conn, sourceID, now.Add(2*time.Minute), "empty-unknown", "unknown", "unknown", "unknown", 0, 0, 0, 0)

	overview, err := New(conn).Overview(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if overview.UnpricedSessions != 0 {
		t.Fatalf("unpriced sessions = %d", overview.UnpricedSessions)
	}

	var foundGPT bool
	for _, usage := range overview.ModelUsage {
		if usage.Model == "unknown" {
			t.Fatalf("zero-token unknown model should not appear in model usage: %+v", overview.ModelUsage)
		}
		if usage.Model == "gpt-5.5" {
			foundGPT = true
			if usage.Unpriced || usage.EstimatedCostUSD == nil {
				t.Fatalf("gpt usage should be priced: %+v", usage)
			}
		}
	}
	if !foundGPT {
		t.Fatalf("gpt model usage missing: %+v", overview.ModelUsage)
	}
}

func TestCostAggregationCharacterizesPricingStates(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(filepath.Join(t.TempDir(), "agentmeter.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	now := time.Date(2026, 6, 27, 1, 2, 3, 0, time.UTC)
	sourceID := insertTimeSource(t, conn, "codex", "Codex", now)
	insertTokenAnalyticsSession(t, conn, sourceID, now, "priced", "gpt-5", "gpt-5", "actual", 1_000_000, 200_000, 500_000, 0, 1_500_000)
	insertTokenAnalyticsSession(t, conn, sourceID, now.Add(time.Minute), "unpriced", "unknown-model", "unknown-model", "actual", 100_000, 0, 50_000, 0, 150_000)
	insertTokenAnalyticsSession(t, conn, sourceID, now.Add(24*time.Hour), "zero-unknown", "empty-unknown", "empty-unknown", "unknown", 0, 0, 0, 0, 0)

	const wantCost = 6.025
	service := New(conn)
	calculator := service.pricingCalculator(ctx)

	totalCost, unpricedCount, err := service.totalCost(ctx)
	if err != nil {
		t.Fatal(err)
	}
	assertCostUSD(t, totalCost, wantCost)
	if unpricedCount != 1 {
		t.Fatalf("total unpriced count = %d", unpricedCount)
	}

	dailyCosts, err := service.dailyCosts(ctx, calculator)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(dailyCosts["2026-06-27"]-wantCost) > 0.000001 {
		t.Fatalf("daily cost = %+v, want %.6f", dailyCosts, wantCost)
	}
	if _, ok := dailyCosts["2026-06-28"]; ok {
		t.Fatalf("zero-token unknown usage should not create a daily cost bucket: %+v", dailyCosts)
	}

	modelCosts, unpricedModels, err := service.modelCostsWithFilters(ctx, calculator, model.AnalyticsFilters{})
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(modelCosts["gpt-5"]-wantCost) > 0.000001 {
		t.Fatalf("model costs = %+v, want gpt-5 %.6f", modelCosts, wantCost)
	}
	if !unpricedModels["unknown-model"] {
		t.Fatalf("billable unknown model should be marked unpriced: %+v", unpricedModels)
	}
	if unpricedModels["empty-unknown"] || modelCosts["empty-unknown"] != 0 {
		t.Fatalf("zero-token unknown model should be ignored: costs=%+v unpriced=%+v", modelCosts, unpricedModels)
	}

	agentCosts, unpricedAgents, err := service.agentCosts(ctx, calculator)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(agentCosts[sourceID]-wantCost) > 0.000001 {
		t.Fatalf("agent costs = %+v, want source %d %.6f", agentCosts, sourceID, wantCost)
	}
	if !unpricedAgents[sourceID] {
		t.Fatalf("agent should retain unpriced marker: %+v", unpricedAgents)
	}
}

func TestEstimatedCostsUseRowPricingAcrossBreakdowns(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(filepath.Join(t.TempDir(), "agentmeter.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	now := time.Date(2026, 6, 27, 1, 2, 3, 0, time.UTC)
	sourceID := insertTimeSource(t, conn, "codex", "Codex", now)
	insertTokenAnalyticsSession(t, conn, sourceID, now, "priced-fallback", "gpt-5", "", "actual", 1_000_000, 200_000, 500_000, 0, 1_500_000)
	insertTokenAnalyticsSession(t, conn, sourceID, now.Add(time.Minute), "unpriced-fallback", "unknown-model", "", "unknown", 100_000, 0, 50_000, 0, 150_000)

	const wantCost = 6.025
	service := New(conn)
	analytics, err := service.TokenAnalytics(ctx)
	if err != nil {
		t.Fatal(err)
	}
	assertCostUSD(t, analytics.EstimatedCostUSD, wantCost)
	if analytics.UnpricedCount != 1 {
		t.Fatalf("token analytics unpriced count = %d", analytics.UnpricedCount)
	}
	assertCostUSD(t, findModelUsage(t, analytics.ModelUsage, "gpt-5").EstimatedCostUSD, wantCost)
	unknownModel := findModelUsage(t, analytics.ModelUsage, "unknown-model")
	if !unknownModel.Unpriced || unknownModel.EstimatedCostUSD != nil {
		t.Fatalf("unknown model usage = %+v", unknownModel)
	}
	agentUsage := findAgentUsageBySource(t, analytics.AgentUsage, sourceID)
	assertCostUSD(t, agentUsage.EstimatedCostUSD, wantCost)
	if !agentUsage.Unpriced {
		t.Fatalf("agent usage should retain unpriced marker: %+v", agentUsage)
	}

	overview, err := service.Overview(ctx)
	if err != nil {
		t.Fatal(err)
	}
	assertCostUSD(t, overview.EstimatedCostUSD, wantCost)
	if overview.UnpricedSessions != 1 {
		t.Fatalf("overview unpriced sessions = %d", overview.UnpricedSessions)
	}
	if len(overview.DailyUsage) != 1 {
		t.Fatalf("daily usage = %+v", overview.DailyUsage)
	}
	assertCostUSD(t, overview.DailyUsage[0].EstimatedCostUSD, wantCost)
	assertCostUSD(t, findModelUsage(t, overview.ModelUsage, "gpt-5").EstimatedCostUSD, wantCost)
	assertCostUSD(t, findAgentUsageBySource(t, overview.AgentUsage, sourceID).EstimatedCostUSD, wantCost)

	agentBreakdown, err := service.UsageBreakdown(ctx, "agent", model.AnalyticsFilters{})
	if err != nil {
		t.Fatal(err)
	}
	agentBucket := findUsageBreakdownBucket(t, agentBreakdown.Buckets, sourceID, "", "")
	assertCostUSD(t, agentBucket.EstimatedCostUSD, wantCost)
	if !agentBucket.Unpriced {
		t.Fatalf("agent breakdown should retain unpriced marker: %+v", agentBucket)
	}

	modelBreakdown, err := service.UsageBreakdown(ctx, "model", model.AnalyticsFilters{})
	if err != nil {
		t.Fatal(err)
	}
	assertCostUSD(t, findUsageBreakdownBucket(t, modelBreakdown.Buckets, 0, "gpt-5", "").EstimatedCostUSD, wantCost)
	unknownBucket := findUsageBreakdownBucket(t, modelBreakdown.Buckets, 0, "unknown-model", "")
	if !unknownBucket.Unpriced || unknownBucket.EstimatedCostUSD != nil {
		t.Fatalf("unknown model breakdown = %+v", unknownBucket)
	}

	agentModelBreakdown, err := service.UsageBreakdown(ctx, "agent,model", model.AnalyticsFilters{})
	if err != nil {
		t.Fatal(err)
	}
	assertCostUSD(t, findUsageBreakdownBucket(t, agentModelBreakdown.Buckets, sourceID, "gpt-5", "").EstimatedCostUSD, wantCost)

	dayBreakdown, err := service.UsageBreakdown(ctx, "day", model.AnalyticsFilters{})
	if err != nil {
		t.Fatal(err)
	}
	dayBucket := findUsageBreakdownBucket(t, dayBreakdown.Buckets, 0, "", "2026-06-27")
	assertCostUSD(t, dayBucket.EstimatedCostUSD, wantCost)
	if !dayBucket.Unpriced {
		t.Fatalf("day breakdown should retain unpriced marker: %+v", dayBucket)
	}
}

func TestUsageBreakdownGroupsByAgentModelAndDay(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(filepath.Join(t.TempDir(), "agentmeter.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	now := time.Date(2026, 6, 27, 1, 2, 3, 0, time.UTC)
	codexSourceID := insertTimeSource(t, conn, "codex", "Codex", now)
	claudeSourceID := insertTimeSource(t, conn, "claude", "Claude Code", now)
	insertTokenAnalyticsSession(t, conn, codexSourceID, now, "codex-a", "gpt-5", "gpt-5", "actual", 1_000, 200, 500, 50, 1_550)
	insertTokenAnalyticsSession(t, conn, codexSourceID, now.Add(time.Hour), "codex-b", "unknown-model", "unknown-model", "actual", 100, 20, 30, 0, 130)
	insertTokenAnalyticsSession(t, conn, claudeSourceID, now.Add(24*time.Hour), "claude-c", "gpt-5-mini", "gpt-5-mini", "actual", 500, 100, 250, 25, 775)

	service := New(conn)
	agentBreakdown, err := service.UsageBreakdown(ctx, "agent", model.AnalyticsFilters{})
	if err != nil {
		t.Fatal(err)
	}
	if agentBreakdown.GroupBy != "agent" || len(agentBreakdown.Buckets) != 2 {
		t.Fatalf("agent breakdown shape = %+v", agentBreakdown)
	}
	codexAgent := findUsageBreakdownBucket(t, agentBreakdown.Buckets, codexSourceID, "", "")
	if codexAgent.SessionCount != 2 || codexAgent.TotalTokens != 1_680 || codexAgent.InputTokens != 1_100 ||
		codexAgent.CachedInputTokens != 220 || !codexAgent.Unpriced || math.Abs(codexAgent.CacheUtilizationRate-0.2) > 0.000001 {
		t.Fatalf("codex agent bucket = %+v", codexAgent)
	}

	modelBreakdown, err := service.UsageBreakdown(ctx, "model", model.AnalyticsFilters{})
	if err != nil {
		t.Fatal(err)
	}
	gptBucket := findUsageBreakdownBucket(t, modelBreakdown.Buckets, 0, "gpt-5", "")
	if gptBucket.SessionCount != 1 || gptBucket.TotalTokens != 1_550 || gptBucket.Unpriced || gptBucket.EstimatedCostUSD == nil {
		t.Fatalf("gpt model bucket = %+v", gptBucket)
	}
	unknownBucket := findUsageBreakdownBucket(t, modelBreakdown.Buckets, 0, "unknown-model", "")
	if !unknownBucket.Unpriced || unknownBucket.EstimatedCostUSD != nil {
		t.Fatalf("unknown model bucket = %+v", unknownBucket)
	}

	agentModelBreakdown, err := service.UsageBreakdown(ctx, "agent,model", model.AnalyticsFilters{Agent: sourceInstanceKey(codexSourceID)})
	if err != nil {
		t.Fatal(err)
	}
	if len(agentModelBreakdown.Buckets) != 2 {
		t.Fatalf("agent model filtered buckets = %+v", agentModelBreakdown.Buckets)
	}
	agentModelBucket := findUsageBreakdownBucket(t, agentModelBreakdown.Buckets, codexSourceID, "gpt-5", "")
	if agentModelBucket.SessionCount != 1 || agentModelBucket.TotalTokens != 1_550 {
		t.Fatalf("agent model bucket = %+v", agentModelBucket)
	}

	dayBreakdown, err := service.UsageBreakdown(ctx, "day", model.AnalyticsFilters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(dayBreakdown.Buckets) != 2 {
		t.Fatalf("day buckets = %+v", dayBreakdown.Buckets)
	}
	firstDay := findUsageBreakdownBucket(t, dayBreakdown.Buckets, 0, "", "2026-06-27")
	if firstDay.SessionCount != 2 || firstDay.TotalTokens != 1_680 || !firstDay.Unpriced {
		t.Fatalf("first day bucket = %+v", firstDay)
	}
}

func TestUsageBreakdownGroupsByProject(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(filepath.Join(t.TempDir(), "agentmeter.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	now := time.Date(2026, 6, 27, 1, 2, 3, 0, time.UTC)
	codexSourceID := insertTimeSource(t, conn, "codex", "Codex", now)
	claudeSourceID := insertTimeSource(t, conn, "claude", "Claude Code", now)

	apiPriced := insertTokenAnalyticsSession(t, conn, codexSourceID, now, "api-priced", "gpt-5", "gpt-5", "actual", 1_000_000, 200_000, 500_000, 0, 1_500_000)
	apiUnpriced := insertTokenAnalyticsSession(t, conn, claudeSourceID, now.Add(time.Minute), "api-unpriced", "unknown-model", "unknown-model", "actual", 100_000, 0, 50_000, 0, 150_000)
	cliSession := insertTokenAnalyticsSession(t, conn, codexSourceID, now.Add(2*time.Minute), "cli", "gpt-5-mini", "gpt-5-mini", "actual", 500_000, 100_000, 250_000, 0, 750_000)
	webSession := insertTokenAnalyticsSession(t, conn, claudeSourceID, now.Add(3*time.Minute), "web", "gpt-5-mini", "gpt-5-mini", "actual", 500_000, 100_000, 250_000, 0, 750_000)
	setSessionProjectPath(t, conn, apiPriced, "/workspace/api")
	setSessionProjectPath(t, conn, apiUnpriced, "/workspace/api/.")
	setSessionProjectPath(t, conn, cliSession, "/workspace/cli")
	setSessionProjectPath(t, conn, webSession, "/workspace/web")

	breakdown, err := New(conn).UsageBreakdown(ctx, "project", model.AnalyticsFilters{})
	if err != nil {
		t.Fatal(err)
	}
	if breakdown.GroupBy != "project" || len(breakdown.Buckets) != 3 {
		t.Fatalf("project breakdown shape = %+v", breakdown)
	}
	if breakdown.Buckets[0].ProjectPath != "/workspace/api" || breakdown.Buckets[1].ProjectPath != "/workspace/cli" || breakdown.Buckets[2].ProjectPath != "/workspace/web" {
		t.Fatalf("project breakdown sort = %+v", breakdown.Buckets)
	}

	apiBucket := findProjectUsageBreakdownBucket(t, breakdown.Buckets, "/workspace/api")
	if apiBucket.SessionCount != 2 || apiBucket.TotalTokens != 1_650_000 || apiBucket.InputTokens != 1_100_000 ||
		apiBucket.CachedInputTokens != 200_000 || !apiBucket.Unpriced {
		t.Fatalf("api project bucket = %+v", apiBucket)
	}
	if math.Abs(apiBucket.CacheUtilizationRate-(float64(200_000)/float64(1_100_000))) > 0.000001 {
		t.Fatalf("api project cache utilization = %f", apiBucket.CacheUtilizationRate)
	}
	assertCostUSD(t, apiBucket.EstimatedCostUSD, 6.025)

	cliBucket := findProjectUsageBreakdownBucket(t, breakdown.Buckets, "/workspace/cli")
	if cliBucket.SessionCount != 1 || cliBucket.TotalTokens != 750_000 || cliBucket.Unpriced {
		t.Fatalf("cli project bucket = %+v", cliBucket)
	}
	assertCostUSD(t, cliBucket.EstimatedCostUSD, 0.6025)

	filtered, err := New(conn).UsageBreakdown(ctx, "project", model.AnalyticsFilters{
		Agent:   sourceInstanceKey(codexSourceID),
		Model:   "gpt-5",
		Project: "/workspace/api",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered.Buckets) != 1 {
		t.Fatalf("filtered project buckets = %+v", filtered.Buckets)
	}
	filteredAPI := findProjectUsageBreakdownBucket(t, filtered.Buckets, "/workspace/api")
	if filteredAPI.SessionCount != 1 || filteredAPI.TotalTokens != 1_500_000 || filteredAPI.Unpriced {
		t.Fatalf("filtered api project bucket = %+v", filteredAPI)
	}
}

func TestOverviewReturnsEmptyArraysWithoutSessions(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(filepath.Join(t.TempDir(), "agentmeter.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	overview, err := New(conn).Overview(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if overview.TotalSessions != 0 || overview.TotalWallDurationMS != 0 || overview.TotalToolCalls != 0 {
		t.Fatalf("empty overview totals = %+v", overview)
	}
	if overview.DailyUsage == nil || overview.ModelUsage == nil || overview.AgentUsage == nil || overview.RecentSessions == nil {
		t.Fatalf("legacy overview slices should be empty arrays, got %+v", overview)
	}
	if overview.ToolTimeLeaders == nil || overview.AgentTimeUsage == nil || overview.ModelTimeUsage == nil || overview.SlowSessions == nil {
		t.Fatalf("time-analysis overview slices should be empty arrays, got %+v", overview)
	}
}

func TestOverviewTimeAnalysisAggregates(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(filepath.Join(t.TempDir(), "agentmeter.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	now := time.Date(2026, 6, 27, 1, 2, 3, 0, time.UTC)
	codexSourceID := insertTimeSource(t, conn, "codex", "Codex", now)
	claudeSourceID := insertTimeSource(t, conn, "claude", "Claude Code", now)

	sessionA := insertOverviewTimeSession(t, conn, codexSourceID, now, "session-a", "gpt-5", 10_000, 7_000, 4_000, 2_000, 3_000, 1_000)
	sessionB := insertOverviewTimeSession(t, conn, codexSourceID, now.Add(time.Minute), "session-b", "gpt-5", 20_000, 15_000, 10_000, 4_000, 5_000, 2_000)
	sessionC := insertOverviewTimeSession(t, conn, claudeSourceID, now.Add(2*time.Minute), "session-c", "claude-sonnet", 15_000, 10_000, 8_000, 1_000, 5_000, 300)

	insertOverviewToolCall(t, conn, sessionA, now, 1_200, "shell_command", "completed", "curl https://example.com")
	insertOverviewToolCall(t, conn, sessionA, now.Add(time.Second), 300, "shell_command", "failed", "go test ./...")
	insertOverviewToolCall(t, conn, sessionB, now.Add(time.Minute), 2_500, "web.run", "completed", "search latest docs")
	insertOverviewToolCall(t, conn, sessionB, now.Add(time.Minute+time.Second), 700, "read_file", "success", "internal/query/service.go")
	insertOverviewToolCall(t, conn, sessionC, now.Add(2*time.Minute), 900, "Bash", "failed", "npm ci")

	overview, err := New(conn).Overview(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if overview.TotalSessions != 3 || overview.TotalWallDurationMS != 45_000 || overview.TotalActiveDurationMS != 32_000 {
		t.Fatalf("overview duration totals = %+v", overview)
	}
	if overview.TotalModelDurationMS != 22_000 || overview.TotalToolDurationMS != 7_000 || overview.TotalIdleDurationMS != 13_000 {
		t.Fatalf("overview time attribution totals = %+v", overview)
	}
	if overview.TotalTokens != 3_300 || overview.TotalToolCalls != 5 {
		t.Fatalf("overview legacy totals changed = %+v", overview)
	}
	if overview.SuspectedNetworkToolDurationMS != 4_600 || overview.SuspectedNetworkToolCalls != 3 {
		t.Fatalf("suspected network totals = duration %d calls %d", overview.SuspectedNetworkToolDurationMS, overview.SuspectedNetworkToolCalls)
	}
	if overview.SuspectedNetworkToolDurationMS < 0 || overview.SuspectedNetworkToolDurationMS > overview.TotalToolDurationMS {
		t.Fatalf("suspected network duration is not a subset of total tool duration: %+v", overview)
	}

	if len(overview.ToolTimeLeaders) < 4 {
		t.Fatalf("tool time leaders = %+v", overview.ToolTimeLeaders)
	}
	wantToolOrder := []string{"web.run", "shell_command", "Bash", "read_file"}
	for index, want := range wantToolOrder {
		if overview.ToolTimeLeaders[index].ToolName != want {
			t.Fatalf("tool time leader %d = %s, want %s: %+v", index, overview.ToolTimeLeaders[index].ToolName, want, overview.ToolTimeLeaders)
		}
	}
	shellUsage := findToolTimeUsage(t, overview.ToolTimeLeaders, "shell_command")
	if shellUsage.Calls != 2 || shellUsage.SuccessCalls != 1 || shellUsage.FailedCalls != 1 || shellUsage.TotalDurationMS != 1_500 || shellUsage.AvgDurationMS != 750 || shellUsage.MaxDurationMS != 1_200 || !shellUsage.SuspectedNetwork {
		t.Fatalf("shell tool time usage = %+v", shellUsage)
	}

	if len(overview.SlowSessions) != 3 || overview.SlowSessions[0].ID != sessionB || overview.SlowSessions[1].ID != sessionC || overview.SlowSessions[2].ID != sessionA {
		t.Fatalf("slow sessions = %+v", overview.SlowSessions)
	}

	codexTime := findAgentTimeUsage(t, overview.AgentTimeUsage, "codex", "Codex")
	if codexTime.SessionCount != 2 || codexTime.ToolCalls != 4 || codexTime.WallDurationMS != 30_000 || codexTime.ActiveDurationMS != 22_000 ||
		codexTime.ModelDurationMS != 14_000 || codexTime.ToolDurationMS != 6_000 || codexTime.IdleDurationMS != 8_000 || codexTime.SuspectedNetworkToolDurationMS != 3_700 {
		t.Fatalf("codex agent time usage = %+v", codexTime)
	}
	claudeTime := findAgentTimeUsage(t, overview.AgentTimeUsage, "claude", "Claude Code")
	if claudeTime.SessionCount != 1 || claudeTime.ToolCalls != 1 || claudeTime.SuspectedNetworkToolDurationMS != 900 {
		t.Fatalf("claude agent time usage = %+v", claudeTime)
	}

	gptTime := findModelTimeUsage(t, overview.ModelTimeUsage, "gpt-5")
	if gptTime.SessionCount != 2 || gptTime.TotalTokens != 3_000 || gptTime.WallDurationMS != 30_000 || gptTime.ActiveDurationMS != 22_000 ||
		gptTime.ModelDurationMS != 14_000 || gptTime.ToolDurationMS != 6_000 || gptTime.IdleDurationMS != 8_000 {
		t.Fatalf("gpt model time usage = %+v", gptTime)
	}
	claudeModelTime := findModelTimeUsage(t, overview.ModelTimeUsage, "claude-sonnet")
	if claudeModelTime.SessionCount != 1 || claudeModelTime.TotalTokens != 300 || claudeModelTime.WallDurationMS != 15_000 {
		t.Fatalf("claude model time usage = %+v", claudeModelTime)
	}
}

func TestSuspectedNetworkToolHeuristic(t *testing.T) {
	tests := []struct {
		name         string
		toolName     string
		inputSummary string
		want         bool
	}{
		{name: "http url", toolName: "shell_command", inputSummary: "curl https://example.com", want: true},
		{name: "git pull", toolName: "Bash", inputSummary: "git pull --ff-only", want: true},
		{name: "package install", toolName: "shell_command", inputSummary: "go mod download", want: true},
		{name: "web tool", toolName: "web.run", inputSummary: "search query", want: true},
		{name: "go test", toolName: "shell_command", inputSummary: "go test ./...", want: false},
		{name: "git status", toolName: "Bash", inputSummary: "git status --short", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSuspectedNetworkTool(tt.toolName, tt.inputSummary); got != tt.want {
				t.Fatalf("isSuspectedNetworkTool(%q, %q) = %v, want %v", tt.toolName, tt.inputSummary, got, tt.want)
			}
		})
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

func insertTimeSource(t *testing.T, conn *sql.DB, kind, name string, now time.Time) int64 {
	t.Helper()
	return insertRow(t, conn, `INSERT INTO sources
		(kind, name, root_path, sessions_path, platform, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		kind, name, "/workspace", "/workspace/"+kind+"/sessions", "test", db.FormatTime(now), db.FormatTime(now))
}

func insertOverviewTimeSession(t *testing.T, conn *sql.DB, sourceID int64, started time.Time, key, sessionModel string, wallDurationMS, activeDurationMS, modelDurationMS, toolDurationMS, idleDurationMS, totalTokens int64) int64 {
	t.Helper()
	sourceFileID := insertRow(t, conn, `INSERT INTO source_files
		(source_id, path, size_bytes, modified_at, content_hash, last_scanned_at, scan_status, error)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		sourceID, "/workspace/.codex/sessions/"+key+".jsonl", 128, db.FormatTime(started), "hash-"+key, db.FormatTime(started), "indexed", "")
	sessionID := insertRow(t, conn, `INSERT INTO sessions
		(source_id, source_file_id, session_key, codex_session_id, project_path, model, model_provider, originator, thread_source, agent_nickname, agent_role,
		 started_at, ended_at, wall_duration_ms, active_duration_ms, model_duration_ms, tool_duration_ms, idle_duration_ms, event_count, parse_status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sourceID, sourceFileID, key, "codex-"+key, "/workspace/project", sessionModel, "openai", "cli", "local", "", "",
		db.FormatTime(started), db.FormatTime(started.Add(time.Duration(wallDurationMS)*time.Millisecond)),
		wallDurationMS, activeDurationMS, modelDurationMS, toolDurationMS, idleDurationMS, 1, "ok")
	insertRow(t, conn, `INSERT INTO token_usage
		(owner_kind, owner_id, model, input_tokens, cached_input_tokens, output_tokens, reasoning_output_tokens, total_tokens, source)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"session", sessionID, sessionModel, totalTokens, 0, 0, 0, totalTokens, "actual")
	return sessionID
}

func insertOverviewToolCall(t *testing.T, conn *sql.DB, sessionID int64, started time.Time, durationMS int64, toolName, status, inputSummary string) int64 {
	t.Helper()
	return insertRow(t, conn, `INSERT INTO tool_calls
		(session_id, started_at, ended_at, duration_ms, tool_name, status, input_summary, output_summary, error, raw_event_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sessionID, db.FormatTime(started), db.FormatTime(started.Add(time.Duration(durationMS)*time.Millisecond)), durationMS, toolName, status, inputSummary, "output", "", 0)
}

func findToolTimeUsage(t *testing.T, items []model.ToolTimeUsage, toolName string) model.ToolTimeUsage {
	t.Helper()
	for _, item := range items {
		if item.ToolName == toolName {
			return item
		}
	}
	t.Fatalf("tool time usage for %s missing: %+v", toolName, items)
	return model.ToolTimeUsage{}
}

func findAgentTimeUsage(t *testing.T, items []model.AgentTimeUsage, agentKind, agentName string) model.AgentTimeUsage {
	t.Helper()
	for _, item := range items {
		if item.AgentKind == agentKind && item.AgentName == agentName {
			return item
		}
	}
	t.Fatalf("agent time usage for %s/%s missing: %+v", agentKind, agentName, items)
	return model.AgentTimeUsage{}
}

func findModelTimeUsage(t *testing.T, items []model.ModelTimeUsage, modelName string) model.ModelTimeUsage {
	t.Helper()
	for _, item := range items {
		if item.Model == modelName {
			return item
		}
	}
	t.Fatalf("model time usage for %s missing: %+v", modelName, items)
	return model.ModelTimeUsage{}
}

func findUsageBreakdownBucket(t *testing.T, items []model.UsageBreakdownBucket, sourceID int64, modelName, date string) model.UsageBreakdownBucket {
	t.Helper()
	for _, item := range items {
		if sourceID > 0 && item.SourceID != sourceID {
			continue
		}
		if modelName != "" && item.Model != modelName {
			continue
		}
		if date != "" && item.Date != date {
			continue
		}
		return item
	}
	t.Fatalf("usage breakdown bucket source=%d model=%q date=%q missing: %+v", sourceID, modelName, date, items)
	return model.UsageBreakdownBucket{}
}

func findProjectUsageBreakdownBucket(t *testing.T, items []model.UsageBreakdownBucket, projectPath string) model.UsageBreakdownBucket {
	t.Helper()
	for _, item := range items {
		if item.ProjectPath == projectPath {
			return item
		}
	}
	t.Fatalf("usage breakdown bucket project=%q missing: %+v", projectPath, items)
	return model.UsageBreakdownBucket{}
}

func assertCostUSD(t *testing.T, got *float64, want float64) {
	t.Helper()
	if got == nil || math.Abs(*got-want) > 0.000001 {
		t.Fatalf("estimated cost = %v, want %.6f", got, want)
	}
}

func valueCost(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func findModelUsage(t *testing.T, items []model.ModelUsage, modelName string) model.ModelUsage {
	t.Helper()
	for _, item := range items {
		if item.Model == modelName {
			return item
		}
	}
	t.Fatalf("model usage for %s missing: %+v", modelName, items)
	return model.ModelUsage{}
}

func findAgentUsage(t *testing.T, items []model.AgentUsage, agentKind, agentName string) model.AgentUsage {
	t.Helper()
	for _, item := range items {
		if item.AgentKind == agentKind && item.AgentName == agentName {
			return item
		}
	}
	t.Fatalf("agent usage for %s/%s missing: %+v", agentKind, agentName, items)
	return model.AgentUsage{}
}

func findAgentUsageBySource(t *testing.T, items []model.AgentUsage, sourceID int64) model.AgentUsage {
	t.Helper()
	for _, item := range items {
		if item.SourceID == sourceID {
			return item
		}
	}
	t.Fatalf("agent usage for source %d missing: %+v", sourceID, items)
	return model.AgentUsage{}
}

func insertSource(t *testing.T, conn *sql.DB, kind, name, rootPath, sessionsPath string, now time.Time) int64 {
	t.Helper()
	return insertRow(t, conn, `INSERT INTO sources
		(kind, name, root_path, sessions_path, platform, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		kind, name, rootPath, sessionsPath, "test", db.FormatTime(now), db.FormatTime(now))
}

func insertAuditSession(t *testing.T, conn *sql.DB, kind, name string, started time.Time, key string) (int64, int64) {
	t.Helper()
	sourceID := insertTimeSource(t, conn, kind, name, started)
	sourceFileID := insertRow(t, conn, `INSERT INTO source_files
		(source_id, path, size_bytes, modified_at, content_hash, last_scanned_at, scan_status, error)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		sourceID, "/workspace/audit/"+key+".jsonl", 128, db.FormatTime(started), "hash-"+key, db.FormatTime(started), "indexed", "")
	sessionID := insertRow(t, conn, `INSERT INTO sessions
		(source_id, source_file_id, session_key, codex_session_id, project_path, model, model_provider, originator, thread_source, agent_nickname, agent_role,
		 started_at, ended_at, wall_duration_ms, active_duration_ms, model_duration_ms, tool_duration_ms, idle_duration_ms, event_count, parse_status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sourceID, sourceFileID, key, "codex-"+key, "/workspace/project", "gpt-5", "openai", "cli", "local", "", "",
		db.FormatTime(started), db.FormatTime(started.Add(time.Second)), 1000, 1000, 1000, 0, 0, 1, "ok")
	return sessionID, sourceFileID
}

func insertAuditFinding(t *testing.T, conn *sql.DB, sessionID, sourceFileID int64, timestamp time.Time, category, severity, ruleID, command string) int64 {
	t.Helper()
	return insertRow(t, conn, `INSERT INTO audit_findings
		(session_id, tool_call_id, source_file_id, raw_event_id, source_line, timestamp, source, event_type, category, severity, rule_id,
		 title, description, evidence, command, shell_family, platform, decision, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sessionID, 0, sourceFileID, 0, 10, db.FormatTime(timestamp), "session_jsonl", "finding", category, severity, ruleID,
		"Finding "+ruleID, "description", "evidence", command, "sh", "test", "observed", db.FormatTime(timestamp))
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

func insertTokenAnalyticsSession(t *testing.T, conn *sql.DB, sourceID int64, started time.Time, key, sessionModel, usageModel, usageSource string, inputTokens, cachedInputTokens, outputTokens, reasoningTokens, totalTokens int64) int64 {
	t.Helper()
	sourceFileID := insertRow(t, conn, `INSERT INTO source_files
		(source_id, path, size_bytes, modified_at, content_hash, last_scanned_at, scan_status, error)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		sourceID, "/workspace/tokens/"+key+".jsonl", 128, db.FormatTime(started), "hash-"+key, db.FormatTime(started), "indexed", "")
	sessionID := insertRow(t, conn, `INSERT INTO sessions
		(source_id, source_file_id, session_key, codex_session_id, project_path, model, model_provider, originator, thread_source, agent_nickname, agent_role,
		 started_at, ended_at, wall_duration_ms, active_duration_ms, model_duration_ms, tool_duration_ms, idle_duration_ms, event_count, parse_status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sourceID, sourceFileID, key, "codex-"+key, "/workspace/project", sessionModel, "openai", "cli", "local", "", "",
		db.FormatTime(started), db.FormatTime(started.Add(time.Second)), 1000, 1000, 1000, 0, 0, 1, "ok")
	insertRow(t, conn, `INSERT INTO token_usage
		(owner_kind, owner_id, model, input_tokens, cached_input_tokens, output_tokens, reasoning_output_tokens, total_tokens, source)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"session", sessionID, usageModel, inputTokens, cachedInputTokens, outputTokens, reasoningTokens, totalTokens, usageSource)
	return sessionID
}

func setSessionProjectPath(t *testing.T, conn *sql.DB, sessionID int64, projectPath string) {
	t.Helper()
	if _, err := conn.Exec(`UPDATE sessions SET project_path = ? WHERE id = ?`, projectPath, sessionID); err != nil {
		t.Fatal(err)
	}
}

func insertOverviewUsageSession(t *testing.T, conn *sql.DB, sourceID int64, started time.Time, key, sessionModel, usageModel, usageSource string, inputTokens, cachedInputTokens, outputTokens, totalTokens int64) int64 {
	t.Helper()
	sourceFileID := insertRow(t, conn, `INSERT INTO source_files
		(source_id, path, size_bytes, modified_at, content_hash, last_scanned_at, scan_status, error)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		sourceID, "/workspace/.codex/sessions/"+key+".jsonl", 128, db.FormatTime(started), "hash-"+key, db.FormatTime(started), "indexed", "")
	sessionID := insertRow(t, conn, `INSERT INTO sessions
		(source_id, source_file_id, session_key, codex_session_id, project_path, model, model_provider, originator, thread_source, agent_nickname, agent_role,
		 started_at, ended_at, wall_duration_ms, active_duration_ms, model_duration_ms, tool_duration_ms, idle_duration_ms, event_count, parse_status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sourceID, sourceFileID, key, "codex-"+key, "/workspace/project", sessionModel, "openai", "cli", "local", "", "",
		db.FormatTime(started), db.FormatTime(started.Add(time.Second)), 1000, 1000, 1000, 0, 0, 1, "ok")
	insertRow(t, conn, `INSERT INTO token_usage
		(owner_kind, owner_id, model, input_tokens, cached_input_tokens, output_tokens, reasoning_output_tokens, total_tokens, source)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"session", sessionID, usageModel, inputTokens, cachedInputTokens, outputTokens, 0, totalTokens, usageSource)
	return sessionID
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
