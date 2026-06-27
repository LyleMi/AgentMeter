package viewmodel

import (
	"testing"

	"AgentMeter/internal/model"
)

func TestDerivedOverviewMetrics(t *testing.T) {
	cost := 2.5
	metrics := DerivedOverviewMetrics(model.Overview{
		TotalSessions:          2,
		TotalInputTokens:       1000,
		TotalCachedInputTokens: 250,
		TotalOutputTokens:      500,
		TotalTokens:            1500,
		EstimatedCostUSD:       &cost,
		TotalWallDurationMS:    120_000,
		TotalActiveDurationMS:  60_000,
		TotalToolCalls:         3,
	})
	if len(metrics) != 8 {
		t.Fatalf("len(metrics) = %d", len(metrics))
	}
	assertMetric(t, metrics, "avgTokensPerSession", "750")
	assertMetric(t, metrics, "avgWallPerSession", "1m 0s")
	assertMetric(t, metrics, "activeShare", "50%")
	assertMetric(t, metrics, "toolsPerSession", "1.5")
	assertMetric(t, metrics, "cacheHitRate", "25%")
	assertMetric(t, metrics, "outputInputRatio", "0.5x")
	assertMetric(t, metrics, "costPerThousandTokens", "$1.6667")
	assertMetric(t, metrics, "tokensPerActiveHour", "90,000")
}

func TestDerivedOverviewMetricsUnpriced(t *testing.T) {
	cost := 1.25
	metrics := DerivedOverviewMetrics(model.Overview{
		TotalSessions:    1,
		TotalTokens:      1000,
		EstimatedCostUSD: &cost,
		UnpricedSessions: 1,
	})
	assertMetric(t, metrics, "costPerThousandTokens", "unpriced")
}

func TestSummarizeTools(t *testing.T) {
	summary := SummarizeTools([]model.ToolStat{
		{ToolName: "Bash", Calls: 10, SuccessCalls: 9, FailedCalls: 1, TotalDurationMS: 20_000},
		{ToolName: "Read", Calls: 5, SuccessCalls: 5, FailedCalls: 0, TotalDurationMS: 10_000},
	})
	if summary.TotalCalls != 15 || summary.ToolsUsed != 2 || summary.FailedPendingCalls != 1 {
		t.Fatalf("summary counts = %+v", summary)
	}
	if summary.AverageDurationLabel != "2s" {
		t.Fatalf("average duration label = %q", summary.AverageDurationLabel)
	}
	if summary.DurationSignal.Tone != ToneSuccess {
		t.Fatalf("duration signal = %+v", summary.DurationSignal)
	}
}

func TestToolStatuses(t *testing.T) {
	if got := ToolSuccessStatus(model.ToolStat{Calls: 100, SuccessCalls: 99}); got.Tone != ToneSuccess || got.Label != "99% ok" {
		t.Fatalf("success status = %+v", got)
	}
	if got := ToolSuccessStatus(model.ToolStat{Calls: 100, SuccessCalls: 95}); got.Tone != ToneWarning || got.Label != "95% ok" {
		t.Fatalf("warning success status = %+v", got)
	}
	if got := ToolFailureStatus(model.ToolStat{Calls: 20, FailedCalls: 2}); got.Tone != ToneError || got.Label != "2 affected" {
		t.Fatalf("failure status = %+v", got)
	}
	if got := ToolDurationSignal(60_000, 1); got.Tone != ToneWarning || got.Label != "Long average" {
		t.Fatalf("duration signal = %+v", got)
	}
}

func TestTopToolsByCalls(t *testing.T) {
	top := TopToolsByCalls([]model.ToolStat{
		{ToolName: "Zed", Calls: 2},
		{ToolName: "Bash", Calls: 4},
		{ToolName: "Apply", Calls: 4},
	}, 2)
	if len(top) != 2 || top[0].ToolName != "Apply" || top[1].ToolName != "Bash" {
		t.Fatalf("top = %+v", top)
	}
}

func assertMetric(t *testing.T, metrics []Metric, key, want string) {
	t.Helper()
	for _, metric := range metrics {
		if metric.Key == key {
			if metric.Value != want {
				t.Fatalf("metric %s = %q, want %q", key, metric.Value, want)
			}
			return
		}
	}
	t.Fatalf("metric %s not found in %+v", key, metrics)
}
