package query

import (
	"context"
	"math"
	"path/filepath"
	"testing"
	"time"

	"github.com/LyleMi/AgentMeter/internal/db"
	"github.com/LyleMi/AgentMeter/internal/model"
	"github.com/LyleMi/AgentMeter/internal/pricing"
)

func TestDeepSeekTimedCostsAcrossSharedViews(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	ctx := context.Background()
	start := time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC) // Session starts off-peak.
	source := insertSource(t, conn, "dsh", "DeepSeek Harness", "/home/.dsh", "/home/.dsh/sessions", start)
	session := insertOverviewUsageSession(t, conn, source, start, "mixed-tariff", "deepseek-flash", "deepseek-flash", "actual", 2_000_000, 400_000, 2_000_000, 4_000_000)
	for _, at := range []time.Time{start.Add(time.Hour), start.Add(4 * time.Hour)} {
		insertRow(t, conn, `INSERT INTO model_calls (session_id, started_at, ended_at, duration_ms, model, provider, status, input_tokens, cached_input_tokens, output_tokens, reasoning_output_tokens, context_compression_tokens, total_tokens, cost_usd) VALUES (?, ?, ?, 1000, 'deepseek-flash', 'deepseek', 'completed', 1000000, 200000, 1000000, 0, 0, 2000000, 999)`, session, db.FormatTime(at), db.FormatTime(at.Add(time.Second)))
	}
	service := New(conn)
	assertCost := func(label string, got *float64, want float64) {
		t.Helper()
		if got == nil || math.Abs(*got-want) > 1e-9 {
			t.Fatalf("%s cost=%v want=%v", label, got, want)
		}
	}
	check := func(want float64) {
		t.Helper()
		detail, err := service.SessionDetail(ctx, session)
		if err != nil {
			t.Fatal(err)
		}
		assertCost("session", detail.Session.EstimatedCostUSD, want)
		total := 0.0
		for _, call := range detail.ModelCalls {
			if call.CostUSD == nil {
				t.Fatal("unpriced call")
			}
			total += *call.CostUSD
		}
		assertCost("calls", &total, want)
		analytics, err := service.TokenAnalytics(ctx)
		if err != nil {
			t.Fatal(err)
		}
		assertCost("analytics", analytics.EstimatedCostUSD, want)
		assertCost("model", analytics.ModelUsage[0].EstimatedCostUSD, want)
		assertCost("agent", analytics.AgentUsage[0].EstimatedCostUSD, want)
		overview, err := service.Overview(ctx)
		if err != nil {
			t.Fatal(err)
		}
		assertCost("overview", overview.EstimatedCostUSD, want)
		for _, group := range []string{"agent", "model", "agent,model", "project", "day"} {
			breakdown, err := service.UsageBreakdown(ctx, group, model.AnalyticsFilters{})
			if err != nil {
				t.Fatal(err)
			}
			assertCost(group, breakdown.Buckets[0].EstimatedCostUSD, want)
		}
		metrics, err := service.modelSignalSessionMetrics(ctx, model.AnalyticsFilters{})
		if err != nil {
			t.Fatal(err)
		}
		assertCost("signals", metrics[0].EstimatedCostUSD, want)
	}
	check(2.1618) // One peak request + one off-peak request, not a single session tariff.
	metrics, err := service.modelSignalSessionMetrics(ctx, model.AnalyticsFilters{})
	if err != nil {
		t.Fatal(err)
	}
	assertCost("cache savings", metrics[0].CacheSavingsUSD, .0882)
	// Custom rates must take effect everywhere without reindexing or being halved.
	if _, err := pricing.UpsertCustom(ctx, conn, model.PricingModelInput{Model: "deepseek-flash", InputPer1M: 10, CachedInputPer1M: 2, OutputPer1M: 20}); err != nil {
		t.Fatal(err)
	}
	check(56.8)
}

func TestDeepSeekSessionTimeFallbackWithPartialCalls(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	ctx := context.Background()
	start := time.Date(2026, 9, 21, 4, 0, 0, 0, time.UTC)
	source := insertSource(t, conn, "dsh", "DeepSeek Harness", "/home/.dsh", "/home/.dsh/sessions", start)
	session := insertOverviewUsageSession(t, conn, source, start, "partial", "deepseek-flash", "deepseek-flash", "actual", 2_000_000, 0, 0, 2_000_000)
	insertRow(t, conn, `INSERT INTO model_calls (session_id, started_at, ended_at, duration_ms, model, provider, status, input_tokens, cached_input_tokens, output_tokens, reasoning_output_tokens, total_tokens) VALUES (?, ?, ?, 1000, 'deepseek-flash', 'deepseek', 'completed', 1000000, 0, 0, 0, 1000000)`, session, db.FormatTime(start), db.FormatTime(start.Add(time.Second)))
	detail, err := New(conn).SessionDetail(ctx, session)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Session.EstimatedCostUSD == nil || math.Abs(*detail.Session.EstimatedCostUSD-.3) > 1e-9 {
		t.Fatalf("partial history must not drop tokens: %+v", detail.Session)
	}
}
