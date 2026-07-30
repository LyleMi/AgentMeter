package attention

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/LyleMi/AgentMeter/internal/model"
)

const (
	maxItems         = 20
	maxAuditItems    = 5
	maxCohortItems   = 5
	maxAnomalyItems  = 3
	defaultWindowDay = 7
)

type Input struct {
	Window         model.AttentionWindow
	Current        model.Overview
	Baseline       model.Overview
	Signals        model.ModelSignals
	AuditFindings  []model.AuditFinding
	Privacy        []model.PrivacyConfigStatus
	PrivacyWarning string
	Settings       model.Settings
}

func ResolveWindow(now time.Time, filters model.AnalyticsFilters) (model.AttentionWindow, model.AnalyticsFilters, model.AnalyticsFilters) {
	now = now.UTC()
	to := parseBoundary(filters.StartedTo, now, true)
	from := parseBoundary(filters.StartedFrom, to.AddDate(0, 0, -defaultWindowDay), false)
	if !from.Before(to) {
		from = to.AddDate(0, 0, -defaultWindowDay)
	}
	duration := to.Sub(from)
	baselineFrom := from.Add(-duration)

	current := filters
	current.StartedFrom = from.Format(time.RFC3339Nano)
	current.StartedTo = to.Format(time.RFC3339Nano)
	baseline := filters
	baseline.StartedFrom = baselineFrom.Format(time.RFC3339Nano)
	baseline.StartedTo = from.Format(time.RFC3339Nano)
	return model.AttentionWindow{
		Current: model.AttentionPeriod{
			From: current.StartedFrom,
			To:   current.StartedTo,
		},
		Baseline: model.AttentionPeriod{
			From: baseline.StartedFrom,
			To:   baseline.StartedTo,
		},
	}, current, baseline
}

func parseBoundary(value string, fallback time.Time, end bool) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed.UTC()
	}
	if parsed, err := time.ParseInLocation("2006-01-02", value, time.UTC); err == nil {
		if end {
			return parsed.AddDate(0, 0, 1)
		}
		return parsed
	}
	return fallback
}

func Build(input Input) model.AttentionResponse {
	response := model.AttentionResponse{
		Window: input.Window,
		Snapshot: model.AttentionSnapshot{
			Sessions:   metric(float64(input.Current.TotalSessions), float64(input.Baseline.TotalSessions)),
			Tokens:     metric(float64(input.Current.TotalTokens), float64(input.Baseline.TotalTokens)),
			CostUSD:    metric(pointerValue(input.Current.EstimatedCostUSD), pointerValue(input.Baseline.EstimatedCostUSD)),
			ActiveTime: metric(float64(input.Current.TotalActiveDurationMS), float64(input.Baseline.TotalActiveDurationMS)),
			ToolCalls:  metric(float64(input.Current.TotalToolCalls), float64(input.Baseline.TotalToolCalls)),
		},
		Items: []model.AttentionItem{},
	}

	items := statusItems(input)
	items = append(items, unpricedItem(input.Current)...)
	items = append(items, auditItems(input.AuditFindings)...)
	items = append(items, modelItems(input.Signals)...)
	items = append(items, anomalyItems(input.Signals.AnomalySessions)...)
	items = append(items, privacyItems(input.Privacy, input.PrivacyWarning)...)
	sortItems(items)
	for _, item := range items {
		switch item.Severity {
		case "critical":
			response.Counts.Critical++
		case "warning":
			response.Counts.Warning++
		}
	}
	response.Counts.Total = len(items)
	items = applyItemLimits(items)
	if len(items) > maxItems {
		items = items[:maxItems]
	}
	response.Items = items
	return response
}

func metric(current, baseline float64) model.AttentionMetric {
	result := model.AttentionMetric{Current: current, Baseline: baseline}
	if baseline != 0 {
		value := (current - baseline) / baseline
		result.ChangePct = &value
	}
	return result
}

func pointerValue(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func statusItems(input Input) []model.AttentionItem {
	enabled := 0
	for _, entry := range input.Settings.SourceEntries {
		if entry.Enabled {
			enabled++
		}
	}
	if enabled == 0 {
		return []model.AttentionItem{{
			Key: "source:none", Kind: "source_status", Severity: "critical", Confidence: 1,
			Subject: "No enabled data source", Reason: "Configure at least one local agent source before running analysis.",
			Destination: model.AttentionItemDestination{Kind: "settings", SettingsPanel: "sources"},
		}}
	}
	result := []model.AttentionItem{}
	if input.Settings.LastIndexResult == nil && input.Current.TotalSessions == 0 {
		result = append(result, model.AttentionItem{
			Key: "index:not-run", Kind: "index_status", Severity: "warning", Confidence: 1,
			Subject: "Data has not been indexed", Reason: "Run an index update to make local sessions available.",
			Destination: model.AttentionItemDestination{Kind: "settings", SettingsPanel: "data"},
		})
	}
	if last := input.Settings.LastIndexResult; last != nil {
		occurred := input.Settings.LastIndexStartedAt
		if last.Failed > 0 {
			result = append(result, model.AttentionItem{
				Key: "index:failed", Kind: "index_status", Severity: "critical", Confidence: 1,
				Subject: "Indexing failures", Reason: fmt.Sprintf("%d files failed during the latest index update.", last.Failed),
				Metric:     &model.AttentionItemMetric{Label: "failed files", Value: float64(last.Failed)},
				OccurredAt: occurred, Destination: model.AttentionItemDestination{Kind: "settings", SettingsPanel: "data"},
			})
		}
		for index, warning := range last.Warnings {
			result = append(result, model.AttentionItem{
				Key: fmt.Sprintf("index:warning:%03d", index), Kind: "index_status", Severity: "warning", Confidence: 1,
				Subject: "Index warning", Reason: warning, OccurredAt: occurred,
				Destination: model.AttentionItemDestination{Kind: "settings", SettingsPanel: "data"},
			})
		}
	}
	return result
}

func unpricedItem(overview model.Overview) []model.AttentionItem {
	if overview.UnpricedSessions <= 0 {
		return nil
	}
	return []model.AttentionItem{{
		Key: "pricing:unpriced", Kind: "unpriced_sessions", Severity: "warning", Confidence: 1,
		Subject: "Unpriced sessions", Reason: fmt.Sprintf("%d sessions use models without pricing.", overview.UnpricedSessions),
		Metric:      &model.AttentionItemMetric{Label: "sessions", Value: float64(overview.UnpricedSessions)},
		Destination: model.AttentionItemDestination{Kind: "settings", SettingsPanel: "pricing"},
	}}
}

func auditItems(findings []model.AuditFinding) []model.AttentionItem {
	result := make([]model.AttentionItem, 0, len(findings))
	for _, finding := range findings {
		if finding.Severity != "critical" && finding.Severity != "high" {
			continue
		}
		severity := "critical"
		confidence := .95
		if finding.Severity == "high" {
			confidence = .85
		}
		occurred := finding.Timestamp
		result = append(result, model.AttentionItem{
			Key: fmt.Sprintf("audit:%d", finding.ID), Kind: "audit_finding", Severity: severity, Confidence: confidence,
			Subject: finding.Title, Reason: finding.Description, OccurredAt: &occurred,
			Destination: model.AttentionItemDestination{Kind: "audit_finding", AuditFindingID: finding.ID},
		})
	}
	return result
}

func modelItems(signals model.ModelSignals) []model.AttentionItem {
	result := []model.AttentionItem{}
	if severity := normalizedSeverity(signals.HealthSummary.Severity); severity != "" {
		result = append(result, model.AttentionItem{
			Key: "model-health:overall", Kind: "model_health", Severity: severity, Confidence: .8,
			Subject: "Model health changed", Reason: firstReason(signals.HealthSummary.TopReasons, "Selected-period model health differs from its baseline."),
			Destination: model.AttentionItemDestination{Kind: "model_analysis"},
		})
	}
	for _, cohort := range signals.Cohorts {
		severity := normalizedSeverity(cohort.Drift.Severity)
		if severity == "" {
			continue
		}
		confidence := .9
		if cohort.Drift.Confidence == "low" {
			confidence = .4
		}
		result = append(result, model.AttentionItem{
			Key: "model-cohort:" + cohort.CohortKey, Kind: "model_cohort", Severity: severity, Confidence: confidence,
			Subject: strings.TrimSpace(cohort.Model), Reason: firstReason(cohort.Drift.Reasons, "This model cohort drifted from the preceding period."),
			Destination: model.AttentionItemDestination{
				Kind: "model_analysis", Agent: cohort.SourceKey, Model: cohort.Model, Project: cohort.ProjectPath,
			},
		})
	}
	return result
}

func anomalyItems(anomalies []model.ModelSignalsAnomalySession) []model.AttentionItem {
	result := []model.AttentionItem{}
	for _, anomaly := range anomalies {
		severity := ""
		switch {
		case anomaly.Score >= .75:
			severity = "critical"
		case anomaly.Score >= .45:
			severity = "warning"
		default:
			continue
		}
		occurred := anomaly.StartedAt
		result = append(result, model.AttentionItem{
			Key: fmt.Sprintf("session-anomaly:%d", anomaly.SessionID), Kind: "session_anomaly", Severity: severity,
			Confidence: anomaly.Score, Subject: firstNonEmpty(anomaly.SessionKey, fmt.Sprintf("Session %d", anomaly.SessionID)),
			Reason:     firstReason(anomaly.ReasonLabels, "Session behavior differs from the selected-period population."),
			Metric:     &model.AttentionItemMetric{Label: "anomaly score", Value: anomaly.Score},
			OccurredAt: &occurred, Destination: model.AttentionItemDestination{Kind: "session", SessionID: anomaly.SessionID},
		})
	}
	return result
}

func applyItemLimits(items []model.AttentionItem) []model.AttentionItem {
	limits := map[string]int{
		"audit_finding":   maxAuditItems,
		"model_cohort":    maxCohortItems,
		"session_anomaly": maxAnomalyItems,
	}
	counts := map[string]int{}
	result := make([]model.AttentionItem, 0, len(items))
	for _, item := range items {
		if limit, limited := limits[item.Kind]; limited {
			if counts[item.Kind] >= limit {
				continue
			}
			counts[item.Kind]++
		}
		result = append(result, item)
	}
	return result
}

func privacyItems(statuses []model.PrivacyConfigStatus, readWarning string) []model.AttentionItem {
	result := []model.AttentionItem{}
	if strings.TrimSpace(readWarning) != "" {
		result = append(result, model.AttentionItem{
			Key: "privacy:read-warning", Kind: "privacy_status", Severity: "warning", Confidence: 1,
			Subject: "Privacy settings could not be read", Reason: readWarning,
			Destination: model.AttentionItemDestination{Kind: "privacy"},
		})
	}
	for _, status := range statuses {
		if status.Summary.Attention > 0 {
			result = append(result, model.AttentionItem{
				Key: "privacy:" + status.Target + ":attention", Kind: "privacy_status", Severity: "warning", Confidence: 1,
				Subject: firstNonEmpty(status.Name, status.Target), Reason: fmt.Sprintf("%d privacy settings need attention.", status.Summary.Attention),
				Metric:      &model.AttentionItemMetric{Label: "settings", Value: float64(status.Summary.Attention)},
				Destination: model.AttentionItemDestination{Kind: "privacy", PrivacyTarget: status.Target},
			})
		}
		for index, warning := range status.Warnings {
			result = append(result, model.AttentionItem{
				Key: fmt.Sprintf("privacy:%s:warning:%03d", status.Target, index), Kind: "privacy_status", Severity: "warning", Confidence: 1,
				Subject: firstNonEmpty(status.Name, status.Target), Reason: warning,
				Destination: model.AttentionItemDestination{Kind: "privacy", PrivacyTarget: status.Target},
			})
		}
	}
	return result
}

func normalizedSeverity(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "critical":
		return "critical"
	case "warning":
		return "warning"
	default:
		return ""
	}
}

func firstReason(values []string, fallback string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return fallback
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func sortItems(items []model.AttentionItem) {
	sort.SliceStable(items, func(i, j int) bool {
		left, right := items[i], items[j]
		if severityRank(left.Severity) != severityRank(right.Severity) {
			return severityRank(left.Severity) > severityRank(right.Severity)
		}
		if left.Confidence != right.Confidence {
			return left.Confidence > right.Confidence
		}
		if compareTime(left.OccurredAt, right.OccurredAt) != 0 {
			return compareTime(left.OccurredAt, right.OccurredAt) > 0
		}
		return left.Key < right.Key
	})
}

func severityRank(value string) int {
	if value == "critical" {
		return 2
	}
	if value == "warning" {
		return 1
	}
	return 0
}

func compareTime(left, right *time.Time) int {
	if left == nil && right == nil {
		return 0
	}
	if left == nil {
		return -1
	}
	if right == nil {
		return 1
	}
	if left.Equal(*right) {
		return 0
	}
	if left.After(*right) {
		return 1
	}
	return -1
}
