package attention

import (
	"fmt"
	"testing"
	"time"

	"github.com/LyleMi/AgentMeter/internal/model"
)

func TestResolveWindowDefaultsToSevenDaysAndEqualBaseline(t *testing.T) {
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	window, current, baseline := ResolveWindow(now, model.AnalyticsFilters{})
	if got, want := current.StartedFrom, "2026-07-23T12:00:00Z"; got != want {
		t.Fatalf("current from = %q, want %q", got, want)
	}
	if got, want := baseline.StartedFrom, "2026-07-16T12:00:00Z"; got != want {
		t.Fatalf("baseline from = %q, want %q", got, want)
	}
	if window.Baseline.To != window.Current.From {
		t.Fatalf("baseline should end where current starts: %#v", window)
	}
}

func TestResolveWindowUsesCustomEqualLengthBaseline(t *testing.T) {
	window, _, _ := ResolveWindow(time.Now(), model.AnalyticsFilters{
		StartedFrom: "2026-07-10",
		StartedTo:   "2026-07-13",
	})
	if got, want := window.Current.To, "2026-07-14T00:00:00Z"; got != want {
		t.Fatalf("current to = %q, want %q", got, want)
	}
	if got, want := window.Baseline.From, "2026-07-06T00:00:00Z"; got != want {
		t.Fatalf("baseline from = %q, want %q", got, want)
	}
}

func TestBuildSortsAndCapsKinds(t *testing.T) {
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	findings := make([]model.AuditFinding, 7)
	for index := range findings {
		findings[index] = model.AuditFinding{
			ID: int64(index + 1), Severity: "critical", Title: fmt.Sprintf("finding %d", index),
			Description: "critical audit evidence", Timestamp: now.Add(-time.Duration(index) * time.Minute),
		}
	}
	cohorts := make([]model.ModelSignalsCohort, 7)
	for index := range cohorts {
		cohorts[index] = model.ModelSignalsCohort{
			CohortKey: fmt.Sprintf("cohort-%d", index), Model: "gpt",
			Drift: model.ModelSignalsDrift{Severity: "warning", Confidence: "high", Reasons: []string{"drift"}},
		}
	}
	anomalies := make([]model.ModelSignalsAnomalySession, 5)
	for index := range anomalies {
		anomalies[index] = model.ModelSignalsAnomalySession{
			SessionID: int64(index + 1), SessionKey: fmt.Sprintf("session-%d", index),
			StartedAt: now.Add(-time.Duration(index) * time.Hour), Score: .8,
		}
	}
	response := Build(Input{
		Current:       model.Overview{UnpricedSessions: 2},
		Signals:       model.ModelSignals{Cohorts: cohorts, AnomalySessions: anomalies},
		AuditFindings: findings,
		Settings:      model.Settings{SourceEntries: []model.SourceEntry{{Path: "/tmp/source", Enabled: true}}},
	})
	counts := map[string]int{}
	for _, item := range response.Items {
		counts[item.Kind]++
	}
	if counts["audit_finding"] != 5 || counts["model_cohort"] != 5 || counts["session_anomaly"] != 3 {
		t.Fatalf("unexpected kind caps: %#v", counts)
	}
	if response.Items[0].Severity != "critical" {
		t.Fatalf("first item severity = %q", response.Items[0].Severity)
	}
	if got, want := response.Counts.Total, 21; got != want {
		t.Fatalf("counts total = %d, want all %d qualifying items", got, want)
	}
}

func TestBuildEmptyDataKeepsSnapshotAndReturnsSourceStatus(t *testing.T) {
	response := Build(Input{})
	if len(response.Items) != 1 || response.Items[0].Key != "source:none" {
		t.Fatalf("items = %#v", response.Items)
	}
	if response.Snapshot.Sessions.Current != 0 || response.Counts.Critical != 1 {
		t.Fatalf("response = %#v", response)
	}
}

func TestBuildMapsUnpricedLowConfidenceAndPrivacy(t *testing.T) {
	response := Build(Input{
		Current: model.Overview{UnpricedSessions: 4},
		Signals: model.ModelSignals{Cohorts: []model.ModelSignalsCohort{{
			CohortKey: "low-confidence", Model: "test",
			Drift: model.ModelSignalsDrift{Severity: "critical", Confidence: "low"},
		}}},
		Privacy: []model.PrivacyConfigStatus{{
			Target: "codex", Name: "Codex", Summary: model.PrivacyConfigSummary{Attention: 2},
		}},
		Settings: model.Settings{SourceEntries: []model.SourceEntry{{Path: "/tmp/source", Enabled: true}}},
	})
	byKey := map[string]model.AttentionItem{}
	for _, item := range response.Items {
		byKey[item.Key] = item
	}
	if byKey["pricing:unpriced"].Metric.Value != 4 {
		t.Fatalf("unpriced item = %#v", byKey["pricing:unpriced"])
	}
	if byKey["model-cohort:low-confidence"].Confidence >= .5 {
		t.Fatalf("low confidence cohort = %#v", byKey["model-cohort:low-confidence"])
	}
	if byKey["privacy:codex:attention"].Destination.PrivacyTarget != "codex" {
		t.Fatalf("privacy item = %#v", byKey["privacy:codex:attention"])
	}
}

func TestBuildCountsAllQualifyingItemsBeforeDisplayLimit(t *testing.T) {
	warnings := make([]string, 25)
	for index := range warnings {
		warnings[index] = fmt.Sprintf("warning %02d", index)
	}
	response := Build(Input{
		Settings: model.Settings{
			SourceEntries:   []model.SourceEntry{{Path: "/tmp/source", Enabled: true}},
			LastIndexResult: &model.IndexResult{Warnings: warnings},
		},
	})
	if got, want := len(response.Items), 20; got != want {
		t.Fatalf("displayed items = %d, want %d", got, want)
	}
	if got, want := response.Counts.Total, 25; got != want {
		t.Fatalf("matching item count = %d, want %d", got, want)
	}
	if got, want := response.Counts.Warning, 25; got != want {
		t.Fatalf("warning count = %d, want %d", got, want)
	}
}
