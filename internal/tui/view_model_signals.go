package tui

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	agentmodel "AgentMeter/internal/model"
)

func (s *state) modelSignalViewportLines() []string {
	lines := modelSignalLines(s.signals, s.width)
	height := s.contentHeight()
	if s.scroll >= len(lines) {
		s.scroll = len(lines) - 1
	}
	if s.scroll < 0 {
		s.scroll = 0
	}
	end := s.scroll + height
	if end > len(lines) {
		end = len(lines)
	}
	return lines[s.scroll:end]
}

func modelSignalLines(signals agentmodel.ModelSignals, width int) []string {
	lines := []string{bold("Model Signals")}
	if modelSignalsEmpty(signals) {
		return append(lines, "No model signals found. Press i to update the index.")
	}

	summary := signals.HealthSummary
	lines = append(lines,
		fmt.Sprintf("Health: %s  Current: %s  Baseline: %s",
			modelSignalSeverityTag(summary.Severity),
			formatSignalWindow(summary.CurrentWindow),
			formatSignalWindow(summary.BaselineWindow),
		),
		fmt.Sprintf("Cohorts: %s  Critical: %s  Warning: %s  Low confidence: %s",
			formatInt(int64(summary.CohortCount)),
			formatInt(int64(summary.CriticalCohorts)),
			formatInt(int64(summary.WarningCohorts)),
			formatInt(int64(summary.LowConfidenceCohorts)),
		),
		fmt.Sprintf("Sessions: %s  Model calls: %s  Tool calls: %s  Failed tools: %s",
			formatInt(int64(signals.TotalSessions)),
			formatInt(int64(signals.TotalModelCalls)),
			formatInt(int64(signals.TotalToolCalls)),
			formatInt(int64(signals.FailedToolCalls)),
		),
		fmt.Sprintf("Throughput: %s tok/s  Output: %s tok/s  Tool failure: %s  Tool dependency: %s  Calls/session: %s",
			formatSignalRate(signals.ModelThroughputTokensPerSecond, 1),
			formatSignalRate(signals.ModelThroughputOutputTokensPerSecond, 1),
			formatSignalPercent(signals.ToolFailureRate),
			formatSignalPercent(signals.ToolDependencyRate),
			formatSignalRate(signals.AvgModelCallsPerSession, 2),
		),
		fmt.Sprintf("Output/input: %sx  Reasoning share: %s  Reasoning overhead: %s  Cache miss: %s",
			formatSignalRate(signals.OutputExpansionRate, 2),
			formatSignalPercent(signals.ReasoningTokenShare),
			formatSignalPercent(signals.ReasoningOverheadRate),
			formatSignalPercent(signals.CacheMissRate),
		),
		fmt.Sprintf("Visible output: %s  Billable output: %s",
			formatInt(signals.VisibleOutputTokens),
			formatInt(signals.BillableOutputTokens),
		),
	)
	if len(summary.TopReasons) > 0 {
		lines = append(lines, fit("Top reasons: "+strings.Join(limitStrings(summary.TopReasons, 4), "; "), width))
	}

	lines = appendModelSignalBreakdownLines(lines, signals.ModelBreakdown, width)
	lines = appendModelSignalCohortLines(lines, signals.Cohorts, width)
	lines = appendModelSignalDailyLines(lines, signals.DailyMetrics, width)
	lines = appendModelSignalProjectLines(lines, signals, width)
	lines = appendModelSignalMatrixLines(lines, signals.Matrix, width)
	lines = appendModelSignalAnomalyLines(lines, signals.AnomalySessions, width)
	return lines
}

func modelSignalsEmpty(signals agentmodel.ModelSignals) bool {
	return signals.TotalSessions == 0 &&
		signals.TotalModelCalls == 0 &&
		len(signals.ModelBreakdown) == 0 &&
		len(signals.Cohorts) == 0 &&
		len(signals.DailyMetrics) == 0 &&
		len(signals.ProjectMetrics) == 0 &&
		len(signals.ProjectHotspots) == 0 &&
		len(signals.AnomalySessions) == 0
}

func appendModelSignalBreakdownLines(lines []string, rows []agentmodel.ModelSignalsBreakdown, width int) []string {
	lines = append(lines, "", bold("Model Breakdown"))
	if len(rows) == 0 {
		return append(lines, "No model breakdown rows.")
	}
	lines = append(lines, fit(fmt.Sprintf("  %-24s %8s %8s %11s %9s %9s %9s %10s",
		"Model", "Sessions", "Calls", "Tokens", "Cache", "Reason", "ToolFail", "Tok/s"), width))
	for _, item := range limitModelBreakdown(rows, 8) {
		lines = append(lines, fit(fmt.Sprintf("  %-24s %8s %8s %11s %9s %9s %9s %10s",
			truncate(empty(item.Model, "unknown"), 24),
			formatInt(int64(item.SessionCount)),
			formatInt(int64(item.ModelCalls)),
			formatInt(item.TotalTokens),
			formatSignalPercent(item.CacheMissRate),
			formatSignalPercent(item.ReasoningTokenShare),
			formatSignalPercent(item.ToolFailureRate),
			formatSignalRate(item.ModelThroughputTokensPerSecond, 1),
		), width))
	}
	return lines
}

func appendModelSignalCohortLines(lines []string, rows []agentmodel.ModelSignalsCohort, width int) []string {
	lines = append(lines, "", bold("Top Drift Cohorts"))
	if len(rows) == 0 {
		return append(lines, "No cohort drift rows.")
	}
	lines = append(lines, fit(fmt.Sprintf("  %-14s %-18s %-20s %-9s %-10s %8s %9s %10s %8s %s",
		"Source", "Project", "Model", "Severity", "Confidence", "Samples", "Latency", "Tok/s", "Failure", "Reason"), width))
	for _, row := range topSignalCohorts(rows, 8) {
		metric := cohortCurrentMetric(row)
		lines = append(lines, fit(fmt.Sprintf("  %-14s %-18s %-20s %-9s %-10s %8s %9s %10s %8s %s",
			truncate(modelSignalSourceName(row.SourceLabel, row.AgentName, row.AgentKind, row.SourceKey), 14),
			truncate(shortPath(row.ProjectPath, 18), 18),
			truncate(modelProviderLabel(row.ModelProvider, row.Model), 20),
			truncate(modelSignalSeverityLabel(row.Drift.Severity), 9),
			truncate(modelSignalConfidence(row.Drift.Confidence), 10),
			formatInt(int64(metric.SessionCount)),
			formatLatencyPer1K(metric),
			formatSignalRate(metric.ModelThroughputTokensPerSecond, 1),
			formatSignalPercent(metric.ToolFailureRate),
			truncate(modelSignalDriftReason(row.Drift), 32),
		), width))
	}
	return lines
}

func appendModelSignalDailyLines(lines []string, rows []agentmodel.ModelSignalsDailyMetric, width int) []string {
	lines = append(lines, "", bold("Daily Efficiency"))
	if len(rows) == 0 {
		return append(lines, "No daily efficiency rows.")
	}
	lines = append(lines, fit(fmt.Sprintf("  %-10s %8s %10s %10s %10s %10s %8s %-9s %s",
		"Date", "Sessions", "Cost", "Cost/S", "P90", "P10 tok/s", "Risk", "Severity", "Reason"), width))
	for _, row := range recentDailyMetrics(rows, 8) {
		lines = append(lines, fit(fmt.Sprintf("  %-10s %8s %10s %10s %10s %10s %8s %-9s %s",
			truncate(row.Date, 10),
			formatInt(int64(row.SessionCount)),
			formatCost(row.EstimatedCostUSD),
			formatOptionalSignalCost(row.CostPerSession),
			formatLatencyPer1K(row.ModelSignalsMetricSet),
			formatSignalRate(p10Throughput(row.ModelSignalsMetricSet), 1),
			formatSignalRate(row.DegradationRiskScore, 2),
			truncate(modelSignalSeverityLabel(row.Drift.Severity), 9),
			truncate(modelSignalDailyReason(row), 34),
		), width))
	}
	return lines
}

func appendModelSignalProjectLines(lines []string, signals agentmodel.ModelSignals, width int) []string {
	if len(signals.ProjectMetrics) > 0 {
		return appendModelSignalProjectMetricLines(lines, signals.ProjectMetrics, width)
	}
	return appendModelSignalProjectHotspotLines(lines, signals.ProjectHotspots, width)
}

func appendModelSignalProjectMetricLines(lines []string, rows []agentmodel.ModelSignalsProjectMetric, width int) []string {
	lines = append(lines, "", bold("Project Hotspots"))
	lines = append(lines, fit(fmt.Sprintf("  %-24s %8s %-20s %10s %-9s %10s %10s %8s %s",
		"Project", "Sessions", "Dominant model", "Cost", "Health", "P90", "P10 tok/s", "Risk", "Reason"), width))
	for _, row := range limitProjectMetrics(rows, 8) {
		metric := projectMetricCurrent(row)
		lines = append(lines, fit(fmt.Sprintf("  %-24s %8s %-20s %10s %-9s %10s %10s %8s %s",
			truncate(shortPath(row.ProjectPath, 24), 24),
			formatInt(int64(row.SessionCount)),
			truncate(projectModelMix(row), 20),
			formatCost(metric.EstimatedCostUSD),
			truncate(modelSignalSeverityLabel(row.Drift.Severity), 9),
			formatLatencyPer1K(metric),
			formatSignalRate(p10Throughput(metric), 1),
			formatSignalRate(metric.DegradationRiskScore, 2),
			truncate(modelSignalDriftReason(row.Drift), 30),
		), width))
	}
	return lines
}

func appendModelSignalProjectHotspotLines(lines []string, rows []agentmodel.ModelSignalsProjectHotspot, width int) []string {
	lines = append(lines, "", bold("Project Hotspots"))
	if len(rows) == 0 {
		return append(lines, "No project hotspot rows.")
	}
	lines = append(lines, fit(fmt.Sprintf("  %-28s %8s %7s %7s %11s %-9s %-10s %s",
		"Project", "Sessions", "Models", "Sources", "Tokens", "Severity", "Confidence", "Reason"), width))
	for _, row := range limitProjectHotspots(rows, 8) {
		lines = append(lines, fit(fmt.Sprintf("  %-28s %8s %7s %7s %11s %-9s %-10s %s",
			truncate(shortPath(row.ProjectPath, 28), 28),
			formatInt(int64(row.SessionCount)),
			formatInt(int64(row.ModelCount)),
			formatInt(int64(row.SourceCount)),
			formatInt(row.TotalTokens),
			truncate(modelSignalSeverityLabel(row.Drift.Severity), 9),
			truncate(modelSignalConfidence(row.Drift.Confidence), 10),
			truncate(modelSignalDriftReason(row.Drift), 32),
		), width))
	}
	return lines
}

func appendModelSignalMatrixLines(lines []string, rows []agentmodel.ModelSignalsMatrixRow, width int) []string {
	lines = append(lines, "", bold("Source Model Matrix"))
	if len(rows) == 0 {
		return append(lines, "No source/model matrix rows.")
	}
	lines = append(lines, fit(fmt.Sprintf("  %-18s %s", "Source", "Model health cells"), width))
	for _, row := range limitMatrixRows(rows, 6) {
		cellLabels := make([]string, 0, len(row.Cells))
		for _, cell := range limitMatrixCells(row.Cells, 4) {
			cellLabels = append(cellLabels, fmt.Sprintf("%s:%s/%s",
				empty(cell.Model, "unknown"),
				modelSignalSeverityLabel(cell.Severity),
				modelSignalConfidence(cell.Confidence),
			))
		}
		if len(row.Cells) > len(cellLabels) {
			cellLabels = append(cellLabels, fmt.Sprintf("+%d", len(row.Cells)-len(cellLabels)))
		}
		lines = append(lines, fit(fmt.Sprintf("  %-18s %s",
			truncate(modelSignalSourceName(row.SourceLabel, row.AgentName, row.AgentKind, row.SourceKey), 18),
			strings.Join(cellLabels, "  "),
		), width))
	}
	return lines
}

func appendModelSignalAnomalyLines(lines []string, rows []agentmodel.ModelSignalsAnomalySession, width int) []string {
	lines = append(lines, "", bold("Anomaly Sessions"))
	if len(rows) == 0 {
		return append(lines, "No anomaly sessions.")
	}
	lines = append(lines, fit(fmt.Sprintf("  %-14s %-14s %-20s %10s %6s %8s %8s %9s %s",
		"Session", "Source", "Model", "Tokens", "Failed", "Reasoning", "Cache", "Tok/s", "Signal"), width))
	for _, row := range limitAnomalies(rows, 8) {
		lines = append(lines, fit(fmt.Sprintf("  %-14s %-14s %-20s %10s %6s %8s %8s %9s %s",
			truncate(anomalySessionLabel(row), 14),
			truncate(modelSignalSourceName(row.SourceLabel, row.AgentName, row.AgentKind, row.SourceKey), 14),
			truncate(empty(row.Model, "unknown"), 20),
			formatInt(row.TotalTokens),
			formatInt(int64(row.FailedToolCalls)),
			formatSignalPercent(row.ReasoningTokenShare),
			formatSignalPercent(row.CacheMissRate),
			formatSignalRate(row.ModelThroughputTokensPerSecond, 1),
			truncate(strings.Join(row.ReasonLabels, ", "), 34),
		), width))
	}
	return lines
}

func modelSignalSourceName(sourceLabel, agentName, agentKind, sourceKey string) string {
	return sourceDisplayName(sourceLabel, agentName, agentKind, sourceKey)
}

func modelProviderLabel(provider, model string) string {
	model = empty(model, "unknown")
	if strings.TrimSpace(provider) == "" {
		return model
	}
	return provider + "/" + model
}

func projectModelMix(row agentmodel.ModelSignalsProjectMetric) string {
	model := empty(row.DominantModel, "unknown")
	share := formatSignalPercent(row.DominantModelShare)
	if strings.TrimSpace(row.DominantModelProvider) != "" {
		model = row.DominantModelProvider + "/" + model
	}
	if row.DominantModelShare <= 0 {
		return model
	}
	return model + " " + share
}

func cohortCurrentMetric(row agentmodel.ModelSignalsCohort) agentmodel.ModelSignalsMetricSet {
	if metricSetHasData(row.Current) {
		return row.Current
	}
	return row.ModelSignalsMetricSet
}

func projectMetricCurrent(row agentmodel.ModelSignalsProjectMetric) agentmodel.ModelSignalsMetricSet {
	if metricSetHasData(row.Current) {
		return row.Current
	}
	return row.ModelSignalsMetricSet
}

func metricSetHasData(metric agentmodel.ModelSignalsMetricSet) bool {
	return metric.SessionCount > 0 || metric.ModelCalls > 0 || metric.TotalTokens > 0 || metric.ModelDurationMS > 0
}

func formatLatencyPer1K(metric agentmodel.ModelSignalsMetricSet) string {
	value := metric.P90ModelLatencyMsPer1kOutputTokens
	if value <= 0 {
		value = metric.ModelLatencyMsPer1kOutputTokens
	}
	return formatSignalRate(value, 0) + "ms"
}

func p10Throughput(metric agentmodel.ModelSignalsMetricSet) float64 {
	if metric.P10ModelThroughputTokensPerSecond > 0 {
		return metric.P10ModelThroughputTokensPerSecond
	}
	return metric.ModelThroughputTokensPerSecond
}

func formatSignalWindow(window agentmodel.ModelSignalsWindow) string {
	from := shortDate(window.From)
	to := shortDate(window.To)
	dateRange := from
	if dateRange == "" || (to != "" && to != from) {
		if dateRange != "" && to != "" {
			dateRange += "-"
		}
		dateRange += to
	}
	if dateRange == "" {
		dateRange = "unknown"
	}
	if window.SessionCount <= 0 && window.ModelCalls <= 0 {
		return dateRange
	}
	return fmt.Sprintf("%s, %s sessions/%s calls", dateRange, formatInt(int64(window.SessionCount)), formatInt(int64(window.ModelCalls)))
}

func shortDate(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 10 {
		return value[:10]
	}
	return value
}

func modelSignalSeverityRank(value string) int {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "critical", "high":
		return 3
	case "warning", "medium":
		return 2
	case "watch", "low", "unknown":
		return 1
	default:
		return 0
	}
}

func modelSignalSeverityLabel(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return "unknown"
	}
	switch normalized {
	case "critical", "warning", "watch", "healthy", "unknown", "high", "medium", "low", "ok":
		return normalized
	default:
		return normalized
	}
}

func modelSignalSeverityTag(value string) string {
	label := modelSignalSeverityLabel(value)
	switch modelSignalSeverityRank(value) {
	case 3:
		return danger(label)
	case 2:
		return warning(label)
	case 1:
		return accent(label)
	default:
		return success(label)
	}
}

func modelSignalConfidence(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "unknown"
	}
	return value
}

func modelSignalDriftReason(drift agentmodel.ModelSignalsDrift) string {
	for _, reason := range drift.Reasons {
		if strings.TrimSpace(reason) != "" {
			return strings.TrimSpace(reason)
		}
	}
	if strings.TrimSpace(drift.SampleNote) != "" {
		return strings.TrimSpace(drift.SampleNote)
	}
	return "No drift reason"
}

func modelSignalDailyReason(row agentmodel.ModelSignalsDailyMetric) string {
	if strings.TrimSpace(row.KeyReason) != "" {
		return row.KeyReason
	}
	if row.LowSample {
		return "low sample"
	}
	return modelSignalDriftReason(row.Drift)
}

func anomalySessionLabel(row agentmodel.ModelSignalsAnomalySession) string {
	for _, value := range []string{row.SessionKey, row.CodexSessionID} {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	if row.SessionID > 0 {
		return "#" + strconv.FormatInt(row.SessionID, 10)
	}
	return "unknown"
}

func formatSignalPercent(value float64) string {
	if !finiteSignal(value) || value < 0 {
		value = 0
	}
	percent := value * 100
	if percent > 0 && percent < 10 {
		return trimSignalFloat(percent, 1) + "%"
	}
	return formatInt(int64(math.Round(percent))) + "%"
}

func formatSignalRate(value float64, digits int) string {
	if !finiteSignal(value) || value < 0 {
		value = 0
	}
	if digits <= 0 {
		return formatInt(int64(math.Round(value)))
	}
	return trimSignalFloat(value, digits)
}

func formatOptionalSignalCost(value *float64) string {
	if value == nil {
		return "-"
	}
	return formatCost(value)
}

func trimSignalFloat(value float64, digits int) string {
	raw := strconv.FormatFloat(value, 'f', digits, 64)
	if strings.Contains(raw, ".") {
		raw = strings.TrimRight(strings.TrimRight(raw, "0"), ".")
	}
	if raw == "" || raw == "-0" {
		return "0"
	}
	return raw
}

func finiteSignal(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func topSignalCohorts(rows []agentmodel.ModelSignalsCohort, limit int) []agentmodel.ModelSignalsCohort {
	result := append([]agentmodel.ModelSignalsCohort(nil), rows...)
	sort.SliceStable(result, func(i, j int) bool {
		left := result[i]
		right := result[j]
		leftMetric := cohortCurrentMetric(left)
		rightMetric := cohortCurrentMetric(right)
		if modelSignalSeverityRank(left.Drift.Severity) != modelSignalSeverityRank(right.Drift.Severity) {
			return modelSignalSeverityRank(left.Drift.Severity) > modelSignalSeverityRank(right.Drift.Severity)
		}
		if leftMetric.DegradationRiskScore != rightMetric.DegradationRiskScore {
			return leftMetric.DegradationRiskScore > rightMetric.DegradationRiskScore
		}
		return leftMetric.TotalTokens > rightMetric.TotalTokens
	})
	return limitSlice(result, limit)
}

func recentDailyMetrics(rows []agentmodel.ModelSignalsDailyMetric, limit int) []agentmodel.ModelSignalsDailyMetric {
	result := append([]agentmodel.ModelSignalsDailyMetric(nil), rows...)
	sort.SliceStable(result, func(i, j int) bool { return result[i].Date > result[j].Date })
	return limitSlice(result, limit)
}

func limitModelBreakdown(rows []agentmodel.ModelSignalsBreakdown, limit int) []agentmodel.ModelSignalsBreakdown {
	return limitSlice(rows, limit)
}

func limitProjectMetrics(rows []agentmodel.ModelSignalsProjectMetric, limit int) []agentmodel.ModelSignalsProjectMetric {
	return limitSlice(rows, limit)
}

func limitProjectHotspots(rows []agentmodel.ModelSignalsProjectHotspot, limit int) []agentmodel.ModelSignalsProjectHotspot {
	return limitSlice(rows, limit)
}

func limitMatrixRows(rows []agentmodel.ModelSignalsMatrixRow, limit int) []agentmodel.ModelSignalsMatrixRow {
	return limitSlice(rows, limit)
}

func limitMatrixCells(rows []agentmodel.ModelSignalsMatrixCell, limit int) []agentmodel.ModelSignalsMatrixCell {
	return limitSlice(rows, limit)
}

func limitAnomalies(rows []agentmodel.ModelSignalsAnomalySession, limit int) []agentmodel.ModelSignalsAnomalySession {
	return limitSlice(rows, limit)
}

func limitStrings(values []string, limit int) []string {
	return limitSlice(values, limit)
}
