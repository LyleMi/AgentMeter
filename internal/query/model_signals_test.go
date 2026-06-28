package query

import (
	"context"
	"database/sql"
	"encoding/json"
	"math"
	"path/filepath"
	"testing"
	"time"

	"AgentMeter/internal/db"
	"AgentMeter/internal/model"
)

func TestModelSignalsAggregatesFiltersAndRanksAnomalies(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(filepath.Join(t.TempDir(), "agentmeter.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	now := time.Date(2026, 6, 25, 1, 2, 3, 0, time.UTC)
	codexSourceID := insertTimeSource(t, conn, "codex", "Codex", now)
	claudeSourceID := insertTimeSource(t, conn, "claude", "Claude Code", now)

	steady := insertModelSignalSession(t, conn, codexSourceID, now, "steady", "/workspace/api", "gpt-5", "gpt-5", 1_000, 200, 500, 100, 1_500, 9_000)
	spiky := insertModelSignalSession(t, conn, codexSourceID, now.Add(24*time.Hour), "spiky", "/workspace/api/.", "gpt-5", "gpt-5", 1_000, 0, 4_000, 3_000, 5_000, 120_000)
	insertModelSignalSession(t, conn, codexSourceID, now.Add(24*time.Hour), "web", "/workspace/web", "gpt-5", "gpt-5", 100, 0, 100, 0, 200, 1_000)
	insertModelSignalSession(t, conn, claudeSourceID, now.Add(time.Hour), "claude", "/workspace/api", "gpt-5", "gpt-5", 10_000, 0, 10_000, 0, 20_000, 10_000)

	insertModelSignalCall(t, conn, steady, now, 400, "gpt-5", "completed")
	insertModelSignalCall(t, conn, steady, now.Add(time.Second), 600, "gpt-5", "completed")
	insertModelSignalCall(t, conn, spiky, now.Add(24*time.Hour), 100_000, "gpt-5", "completed")

	insertModelSignalToolCall(t, conn, steady, now, "shell_command", "completed")
	insertModelSignalToolCall(t, conn, steady, now.Add(time.Second), "shell_command", "failed")
	insertModelSignalToolCall(t, conn, spiky, now.Add(24*time.Hour), "shell_command", "failed")
	insertModelSignalToolCall(t, conn, spiky, now.Add(24*time.Hour+time.Second), "read_file", "error")
	insertModelSignalToolCall(t, conn, spiky, now.Add(24*time.Hour+2*time.Second), "web.run", "pending")

	signals, err := New(conn).ModelSignalsWithFilters(ctx, model.AnalyticsFilters{
		Agent:       sourceInstanceKey(codexSourceID),
		Project:     "/workspace/api",
		StartedFrom: "2026-06-25",
		StartedTo:   "2026-06-26",
	})
	if err != nil {
		t.Fatal(err)
	}

	if signals.TotalSessions != 2 || signals.TotalModelCalls != 3 || signals.TotalToolCalls != 5 || signals.FailedToolCalls != 4 {
		t.Fatalf("model signals totals = %+v", signals)
	}
	assertFloat(t, signals.ToolFailureRate, 0.8)
	assertFloat(t, signals.ToolDependencyRate, 1)
	assertFloat(t, signals.AvgModelCallsPerSession, 1.5)
	assertFloat(t, signals.OutputExpansionRate, 2.25)
	assertFloat(t, signals.ReasoningTokenShare, float64(3_100)/4_500)
	assertFloat(t, signals.CacheMissRate, 0.9)
	assertFloat(t, signals.ModelThroughputTokensPerSecond, float64(6_500)/101)
	assertFloat(t, signals.ModelThroughputOutputTokensPerSecond, float64(4_500)/101)

	if len(signals.ModelBreakdown) != 1 {
		t.Fatalf("model breakdown = %+v", signals.ModelBreakdown)
	}
	breakdown := signals.ModelBreakdown[0]
	if breakdown.Model != "gpt-5" || breakdown.SessionCount != 2 || breakdown.TotalTokens != 6_500 {
		t.Fatalf("breakdown = %+v", breakdown)
	}
	assertFloat(t, breakdown.ToolFailureRate, 0.8)
	assertFloat(t, breakdown.ToolDependencyRate, 1)

	if len(signals.Trend) != 2 {
		t.Fatalf("trend = %+v", signals.Trend)
	}
	firstDay := signals.Trend[0]
	if firstDay.Date != "2026-06-25" || firstDay.SessionCount != 1 || firstDay.ModelCalls != 2 || !firstDay.LowSample {
		t.Fatalf("first trend point = %+v", firstDay)
	}
	assertFloat(t, firstDay.ToolFailureRate, 0.5)
	assertFloat(t, firstDay.ModelThroughputTokensPerSecond, 1_500)
	assertFloat(t, firstDay.ModelThroughputOutputTokensPerSecond, 500)
	secondDay := signals.Trend[1]
	if secondDay.Date != "2026-06-26" || secondDay.SessionCount != 1 || secondDay.FailedToolCalls != 3 {
		t.Fatalf("second trend point = %+v", secondDay)
	}
	assertFloat(t, secondDay.RollingToolFailureRate, 0.8)
	assertFloat(t, secondDay.RollingModelThroughputTokensPerSecond, float64(6_500)/101)

	if len(signals.AnomalySessions) == 0 || signals.AnomalySessions[0].SessionID != spiky {
		t.Fatalf("anomalies = %+v", signals.AnomalySessions)
	}
	top := signals.AnomalySessions[0]
	for _, label := range []string{"high reasoning share", "high output/input ratio", "slow model throughput", "failed tool calls", "high cache miss"} {
		if !containsModelSignalLabel(top.ReasonLabels, label) {
			t.Fatalf("top anomaly labels = %+v, missing %q", top.ReasonLabels, label)
		}
	}

	modelScoped, err := New(conn).ModelSignalsWithFilters(ctx, model.AnalyticsFilters{Model: "gpt-5"})
	if err != nil {
		t.Fatal(err)
	}
	if modelScoped.TotalSessions != 4 || len(modelScoped.ModelBreakdown) != 1 || modelScoped.ModelBreakdown[0].Model != "gpt-5" {
		t.Fatalf("model-scoped signals = %+v", modelScoped)
	}
}

func TestModelSignalsEmptyAndZeroDenominatorResponses(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(filepath.Join(t.TempDir(), "agentmeter.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	service := New(conn)
	empty, err := service.ModelSignals(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if empty.TotalSessions != 0 || empty.Trend == nil || empty.ModelBreakdown == nil || empty.AnomalySessions == nil || empty.Cohorts == nil || empty.Matrix == nil || empty.ProjectHotspots == nil || empty.HealthSummary.TopReasons == nil {
		t.Fatalf("empty model signals = %+v", empty)
	}
	if empty.HealthSummary.Severity != "unknown" {
		t.Fatalf("empty health summary severity = %q", empty.HealthSummary.Severity)
	}
	if _, err := json.Marshal(empty); err != nil {
		t.Fatalf("empty model signals should marshal without NaN/Inf: %v", err)
	}

	now := time.Date(2026, 6, 27, 1, 2, 3, 0, time.UTC)
	sourceID := insertTimeSource(t, conn, "codex", "Codex", now)
	insertModelSignalSession(t, conn, sourceID, now, "zero", "/workspace/project", "gpt-5", "gpt-5", 100, 200, 0, 10, 0, 0)

	signals, err := service.ModelSignals(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if signals.OutputExpansionRate != 0 || signals.ReasoningTokenShare != 0 || signals.CacheMissRate != 0 || signals.ModelThroughputTokensPerSecond != 0 {
		t.Fatalf("zero-denominator rates should be clamped: %+v", signals)
	}
	if len(signals.Trend) != 1 || signals.Trend[0].CacheMissRate != 0 || signals.Trend[0].ModelThroughputTokensPerSecond != 0 {
		t.Fatalf("zero-denominator trend = %+v", signals.Trend)
	}
	if _, err := json.Marshal(signals); err != nil {
		t.Fatalf("model signals should marshal without NaN/Inf: %v", err)
	}
}

func TestModelSignalsEmitsDriftCohortsMatrixAndHotspots(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(filepath.Join(t.TempDir(), "agentmeter.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	anchor := time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)
	sourceID := insertTimeSource(t, conn, "codex", "Codex", anchor)

	baselineStarts := []time.Time{anchor.Add(-7 * 24 * time.Hour), anchor.Add(-6 * 24 * time.Hour)}
	for index, started := range baselineStarts {
		sessionID := insertModelSignalSession(t, conn, sourceID, started, "baseline-drift-"+string(rune('a'+index)), "/workspace/api", "gpt-5", "gpt-5", 1_000, 500, 1_000, 100, 2_000, 1_000)
		insertModelSignalCall(t, conn, sessionID, started, 1_000, "gpt-5", "completed")
		insertModelSignalToolCall(t, conn, sessionID, started, "shell_command", "completed")
	}

	currentStarts := []time.Time{anchor.Add(-2 * time.Hour), anchor}
	for index, started := range currentStarts {
		sessionID := insertModelSignalSession(t, conn, sourceID, started, "current-drift-"+string(rune('a'+index)), "/workspace/api/.", "gpt-5", "gpt-5", 1_000, 0, 1_000, 500, 2_000, 3_000)
		insertModelSignalCall(t, conn, sessionID, started, 3_000, "gpt-5", "completed")
		insertModelSignalToolCall(t, conn, sessionID, started, "shell_command", "failed")
	}

	signals, err := New(conn).ModelSignals(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if signals.HealthSummary.Severity != "critical" || signals.HealthSummary.CohortCount != 1 || signals.HealthSummary.CriticalCohorts != 1 {
		t.Fatalf("health summary = %+v", signals.HealthSummary)
	}
	if signals.HealthSummary.CurrentWindow.SessionCount != 2 || signals.HealthSummary.BaselineWindow.SessionCount != 2 {
		t.Fatalf("health windows = %+v / %+v", signals.HealthSummary.CurrentWindow, signals.HealthSummary.BaselineWindow)
	}
	if !containsModelSignalLabel(signals.HealthSummary.TopReasons, "model latency increased") {
		t.Fatalf("top reasons = %+v", signals.HealthSummary.TopReasons)
	}

	if len(signals.Cohorts) != 1 {
		t.Fatalf("cohorts = %+v", signals.Cohorts)
	}
	cohort := signals.Cohorts[0]
	if cohort.ModelProvider != "openai" || cohort.Model != "gpt-5" || cohort.ProjectPath != "/workspace/api" || cohort.CohortKey == "" {
		t.Fatalf("cohort identity = %+v", cohort)
	}
	if cohort.SessionCount != 4 || cohort.Current.SessionCount != 2 || cohort.Baseline.SessionCount != 2 {
		t.Fatalf("cohort windows = %+v", cohort)
	}
	if cohort.Drift.Severity != "critical" || cohort.Drift.Confidence != "high" {
		t.Fatalf("cohort drift = %+v", cohort.Drift)
	}
	if !containsModelSignalDriftMetric(cohort.Drift.Metrics, "modelLatencyMsPer1kOutputTokens") || !containsModelSignalDriftMetric(cohort.Drift.Metrics, "modelThroughputOutputTokensPerSecond") {
		t.Fatalf("cohort drift metrics = %+v", cohort.Drift.Metrics)
	}

	if len(signals.Matrix) != 1 || len(signals.Matrix[0].Cells) != 1 {
		t.Fatalf("matrix = %+v", signals.Matrix)
	}
	cell := signals.Matrix[0].Cells[0]
	if cell.ModelProvider != "openai" || cell.Model != "gpt-5" || cell.CohortCount != 1 || cell.Severity != "critical" || cell.KeyReason == "" {
		t.Fatalf("matrix cell = %+v", cell)
	}

	if len(signals.ProjectHotspots) != 1 {
		t.Fatalf("project hotspots = %+v", signals.ProjectHotspots)
	}
	hotspot := signals.ProjectHotspots[0]
	if hotspot.ProjectPath != "/workspace/api" || hotspot.ModelCount != 1 || hotspot.SourceCount != 1 || hotspot.Drift.Severity != "critical" {
		t.Fatalf("project hotspot = %+v", hotspot)
	}
	if _, err := json.Marshal(signals); err != nil {
		t.Fatalf("model signals with drift should marshal without NaN/Inf: %v", err)
	}
}

func TestModelSignalsDriftLowConfidenceWhenWindowMissing(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(filepath.Join(t.TempDir(), "agentmeter.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	anchor := time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)
	sourceID := insertTimeSource(t, conn, "codex", "Codex", anchor)
	sessionID := insertModelSignalSession(t, conn, sourceID, anchor, "current-only", "/workspace/api", "gpt-5", "gpt-5", 1_000, 0, 1_000, 100, 2_000, 1_000)
	insertModelSignalCall(t, conn, sessionID, anchor, 1_000, "gpt-5", "completed")

	signals, err := New(conn).ModelSignals(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if signals.HealthSummary.Severity != "unknown" || signals.HealthSummary.LowConfidenceCohorts != 1 {
		t.Fatalf("health summary = %+v", signals.HealthSummary)
	}
	if len(signals.Cohorts) != 1 {
		t.Fatalf("cohorts = %+v", signals.Cohorts)
	}
	drift := signals.Cohorts[0].Drift
	if drift.Severity != "unknown" || drift.Confidence != "low" || drift.SampleNote != "missing baseline window" {
		t.Fatalf("low-confidence drift = %+v", drift)
	}
	if drift.Reasons == nil || drift.Metrics == nil {
		t.Fatalf("low-confidence drift slices should be non-nil: %+v", drift)
	}
	if len(signals.Matrix) != 1 || signals.Matrix[0].Cells[0].Confidence != "low" || signals.Matrix[0].Cells[0].Severity != "unknown" {
		t.Fatalf("matrix low confidence = %+v", signals.Matrix)
	}
	if len(signals.ProjectHotspots) != 1 || signals.ProjectHotspots[0].Drift.Confidence != "low" {
		t.Fatalf("hotspots low confidence = %+v", signals.ProjectHotspots)
	}
}

func insertModelSignalSession(t *testing.T, conn *sql.DB, sourceID int64, started time.Time, key, projectPath, sessionModel, usageModel string, inputTokens, cachedInputTokens, outputTokens, reasoningTokens, totalTokens, modelDurationMS int64) int64 {
	t.Helper()
	sourceFileID := insertRow(t, conn, `INSERT INTO source_files
		(source_id, path, size_bytes, modified_at, content_hash, last_scanned_at, scan_status, error)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		sourceID, "/workspace/model-signals/"+key+".jsonl", 128, db.FormatTime(started), "hash-model-signals-"+key, db.FormatTime(started), "indexed", "")
	sessionID := insertRow(t, conn, `INSERT INTO sessions
		(source_id, source_file_id, session_key, codex_session_id, project_path, model, model_provider, originator, thread_source, agent_nickname, agent_role,
		 started_at, ended_at, wall_duration_ms, active_duration_ms, model_duration_ms, tool_duration_ms, idle_duration_ms, event_count, parse_status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sourceID, sourceFileID, key, "codex-"+key, projectPath, sessionModel, "openai", "cli", "local", "", "",
		db.FormatTime(started), db.FormatTime(started.Add(time.Duration(modelDurationMS)*time.Millisecond)),
		modelDurationMS, modelDurationMS, modelDurationMS, 0, 0, 1, "ok")
	insertRow(t, conn, `INSERT INTO token_usage
		(owner_kind, owner_id, model, input_tokens, cached_input_tokens, output_tokens, reasoning_output_tokens, total_tokens, source)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"session", sessionID, usageModel, inputTokens, cachedInputTokens, outputTokens, reasoningTokens, totalTokens, "actual")
	return sessionID
}

func insertModelSignalCall(t *testing.T, conn *sql.DB, sessionID int64, started time.Time, durationMS int64, modelName, status string) int64 {
	t.Helper()
	return insertRow(t, conn, `INSERT INTO model_calls
		(session_id, started_at, ended_at, duration_ms, model, provider, status, input_tokens, cached_input_tokens, output_tokens, reasoning_output_tokens, total_tokens, cost_usd)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sessionID, db.FormatTime(started), db.FormatTime(started.Add(time.Duration(durationMS)*time.Millisecond)), durationMS, modelName, "openai", status, 0, 0, 0, 0, 0, nil)
}

func insertModelSignalToolCall(t *testing.T, conn *sql.DB, sessionID int64, started time.Time, toolName, status string) int64 {
	t.Helper()
	return insertRow(t, conn, `INSERT INTO tool_calls
		(session_id, started_at, ended_at, duration_ms, tool_name, status, input_summary, output_summary, error, raw_event_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sessionID, db.FormatTime(started), db.FormatTime(started.Add(time.Second)), 1_000, toolName, status, "input", "output", "", 0)
}

func assertFloat(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.000001 {
		t.Fatalf("float = %.12f, want %.12f", got, want)
	}
}

func containsModelSignalLabel(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func containsModelSignalDriftMetric(values []model.ModelSignalsDriftMetric, key string) bool {
	for _, candidate := range values {
		if candidate.Key == key {
			return true
		}
	}
	return false
}
