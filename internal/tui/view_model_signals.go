package tui

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	agentmodel "github.com/LyleMi/AgentMeter/internal/model"
)

func (s *state) modelSignalViewportLines() []string {
	lines := modelSignalLines(s.signals, s.width, s.modelSignalsTab)
	return s.viewportLines(lines)
}

func modelSignalLines(signals agentmodel.ModelSignals, width int, tab modelSignalsTab) []string {
	lines := []string{
		bold("Model Signals"),
		modelSignalTabLine(tab, width),
	}
	if modelSignalsEmpty(signals) {
		return append(lines, "", "No model signals found. Press i to update the index.")
	}

	lines = appendModelSignalHeaderLines(lines, signals, width)
	switch tab {
	case modelSignalsTabOverview:
		lines = appendModelSignalOverviewLines(lines, signals, width)
	case modelSignalsTabDaily:
		lines = appendModelSignalDailyLines(lines, signals.DailyMetrics, width)
	case modelSignalsTabCohorts:
		lines = appendModelSignalCohortLines(lines, signals.Cohorts, width)
	case modelSignalsTabMatrix:
		lines = appendModelSignalMatrixLines(lines, signals.Matrix, width)
	case modelSignalsTabProjects:
		lines = appendModelSignalProjectLines(lines, signals, width)
	case modelSignalsTabAnomalies:
		lines = appendModelSignalAnomalyLines(lines, signals.AnomalySessions, width)
	default:
		lines = appendModelSignalMetricExplorerLines(lines, signals, width)
	}
	return lines
}

func modelSignalTabLine(active modelSignalsTab, width int) string {
	labels := make([]string, 0, len(modelSignalsTabs))
	for _, tab := range modelSignalsTabs {
		label := tab.title()
		if tab == active {
			label = inverse(" " + label + " ")
		}
		labels = append(labels, label)
	}
	return fit("Tabs: "+strings.Join(labels, "  "), width)
}

func appendModelSignalHeaderLines(lines []string, signals agentmodel.ModelSignals, width int) []string {
	summary := signals.HealthSummary
	lines = append(lines, "",
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
	return lines
}

func appendModelSignalOverviewLines(lines []string, signals agentmodel.ModelSignals, width int) []string {
	lines = append(lines, "", bold("Health Overview"))
	lines = appendModelSignalBreakdownLines(lines, signals.ModelBreakdown, width)
	lines = appendModelSignalCohortLines(lines, signals.Cohorts, width)
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

func appendModelSignalMetricExplorerLines(lines []string, signals agentmodel.ModelSignals, width int) []string {
	lines = append(lines, "", bold("Metric Explorer"))
	lines = append(lines, dim("Terminal view of the Web chart metrics. Switch tabs for source tables."))
	lines = append(lines, "", bold("Performance"))
	lines = append(lines,
		fit(fmt.Sprintf("  %-24s %-14s %s", "P90 latency", modelSignalBestLatency(signals.DailyMetrics, true), "slower is worse"),
			width),
		fit(fmt.Sprintf("  %-24s %-14s %s", "P50 latency", modelSignalBestLatency(signals.DailyMetrics, false), "typical ms/1k output"),
			width),
		fit(fmt.Sprintf("  %-24s %-14s %s", "P10 throughput", modelSignalWorstThroughput(signals.DailyMetrics, true), "lower tail tok/s"),
			width),
		fit(fmt.Sprintf("  %-24s %-14s %s", "Output throughput", formatSignalRate(signals.ModelThroughputOutputTokensPerSecond, 1)+" tok/s", "visible output speed"),
			width),
	)
	lines = append(lines, "", bold("Cost"))
	lines = append(lines,
		fit(fmt.Sprintf("  %-24s %-14s %s", "Estimated cost", modelSignalTotalCost(signals), "priced indexed usage"),
			width),
		fit(fmt.Sprintf("  %-24s %-14s %s", "Cost/session", modelSignalLatestCostPerSession(signals.DailyMetrics), "latest daily cost burn"),
			width),
		fit(fmt.Sprintf("  %-24s %-14s %s", "Cost/active hour", modelSignalLatestCostPerActiveHour(signals.DailyMetrics), "active-time normalized"),
			width),
		fit(fmt.Sprintf("  %-24s %-14s %s", "Cache savings", modelSignalTotalCacheSavings(signals.DailyMetrics, signals.ProjectMetrics), "cached-input discount"),
			width),
	)
	lines = append(lines, "", bold("Pressure"))
	lines = append(lines,
		fit(fmt.Sprintf("  %-24s %-14s %s", "Failure pressure", modelSignalLatestFailurePressure(signals.DailyMetrics), "failed model/tool calls per session"),
			width),
		fit(fmt.Sprintf("  %-24s %-14s %s", "Retry pressure", formatSignalRate(signals.AvgModelCallsPerSession, 2)+"/session", "model calls per session"),
			width),
		fit(fmt.Sprintf("  %-24s %-14s %s", "Model failure rate", modelSignalLatestModelFailureRate(signals.DailyMetrics), "failed model calls"),
			width),
		fit(fmt.Sprintf("  %-24s %-14s %s", "Tool failure rate", formatSignalPercent(signals.ToolFailureRate), "failed tool calls"),
			width),
	)
	lines = append(lines, "", bold("Shape"))
	lines = append(lines,
		fit(fmt.Sprintf("  %-24s %-14s %s", "Cache miss", formatSignalPercent(signals.CacheMissRate), "uncached input share"),
			width),
		fit(fmt.Sprintf("  %-24s %-14s %s", "Reasoning share", formatSignalPercent(signals.ReasoningTokenShare), "reasoning output share"),
			width),
		fit(fmt.Sprintf("  %-24s %-14s %s", "Output/input", formatSignalRate(signals.OutputExpansionRate, 2)+"x", "generation expansion"),
			width),
		fit(fmt.Sprintf("  %-24s %-14s %s", "Tool dependency", formatSignalPercent(signals.ToolDependencyRate), "sessions with tools"),
			width),
	)
	if len(signals.Trend) > 0 {
		lines = append(lines, "", bold("Recent Trend"))
		lines = append(lines, fit(fmt.Sprintf("  %-10s %8s %10s %10s %9s %9s %s",
			"Date", "Sessions", "Tokens", "Tok/s", "Failure", "CacheMiss", "Note"), width))
		for _, point := range recentTrendPoints(signals.Trend, 8) {
			note := ""
			if point.LowSample {
				note = "low sample"
			}
			lines = append(lines, fit(fmt.Sprintf("  %-10s %8s %10s %10s %9s %9s %s",
				truncate(point.Date, 10),
				formatInt(int64(point.SessionCount)),
				formatInt(point.TotalTokens),
				formatSignalRate(point.ModelThroughputTokensPerSecond, 1),
				formatSignalPercent(point.ToolFailureRate),
				formatSignalPercent(point.CacheMissRate),
				note,
			), width))
		}
	}
	return lines
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
	lines = append(lines, fit(fmt.Sprintf("  %-14s %-18s %-20s %8s %12s %12s %10s %8s %-8s %-10s %s",
		"Source", "Project", "Model", "Samples", "P90/P50", "P10/P50", "Out tok/s", "Failure", "Health", "Confidence", "Reason"), width))
	for _, row := range topSignalCohorts(rows, 8) {
		metric := cohortCurrentMetric(row)
		lines = append(lines, fit(fmt.Sprintf("  %-14s %-18s %-20s %8s %12s %12s %10s %8s %-8s %-10s %s",
			truncate(modelSignalSourceName(row.SourceLabel, row.AgentName, row.AgentKind, row.SourceKey), 14),
			truncate(shortPath(row.ProjectPath, 18), 18),
			truncate(modelProviderLabel(row.ModelProvider, row.Model), 20),
			formatInt(int64(metric.SessionCount)),
			formatLatencyPair(metric),
			formatThroughputPair(metric),
			formatSignalRate(metric.ModelThroughputOutputTokensPerSecond, 1),
			formatFailurePressure(metric),
			truncate(modelSignalSeverityLabel(row.Drift.Severity), 8),
			truncate(modelSignalConfidence(row.Drift.Confidence), 10),
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
	lines = append(lines, fit(fmt.Sprintf("  %-10s %8s %9s %9s %9s %8s %12s %12s %8s %8s %6s %-8s %s",
		"Date", "Sessions", "Cost", "Cost/S", "Cost/H", "Saved", "P90/P50", "P10/P50", "Retry", "Failure", "Risk", "Health", "Reason"), width))
	for _, row := range recentDailyMetrics(rows, 8) {
		lines = append(lines, fit(fmt.Sprintf("  %-10s %8s %9s %9s %9s %8s %12s %12s %8s %8s %6s %-8s %s",
			truncate(row.Date, 10),
			formatInt(int64(row.SessionCount)),
			formatCost(row.EstimatedCostUSD),
			formatOptionalSignalCost(row.CostPerSession),
			formatOptionalSignalCost(row.CostPerActiveHour),
			formatOptionalSignalCost(row.CacheSavingsUSD),
			formatLatencyPair(row.ModelSignalsMetricSet),
			formatThroughputPair(row.ModelSignalsMetricSet),
			formatSignalRate(row.AvgModelCallsPerSession, 2),
			formatFailurePressure(row.ModelSignalsMetricSet),
			formatSignalRate(row.DegradationRiskScore, 2),
			truncate(modelSignalSeverityLabel(row.Drift.Severity), 8),
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
	lines = append(lines, fit(fmt.Sprintf("  %-22s %8s %-20s %9s %8s %-8s %12s %12s %8s %6s %s",
		"Project", "Sessions", "Dominant model", "Cost", "Saved", "Health", "P90/P50", "P10/P50", "Failure", "Risk", "Reason"), width))
	for _, row := range limitProjectMetrics(rows, 8) {
		metric := projectMetricCurrent(row)
		lines = append(lines, fit(fmt.Sprintf("  %-22s %8s %-20s %9s %8s %-8s %12s %12s %8s %6s %s",
			truncate(shortPath(row.ProjectPath, 22), 22),
			formatInt(int64(row.SessionCount)),
			truncate(projectModelMix(row), 20),
			formatCost(metric.EstimatedCostUSD),
			formatOptionalSignalCost(metric.CacheSavingsUSD),
			truncate(modelSignalSeverityLabel(row.Drift.Severity), 8),
			formatLatencyPair(metric),
			formatThroughputPair(metric),
			formatFailurePressure(metric),
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
	lines = append(lines, fit(fmt.Sprintf("  %-24s %8s %7s %7s %10s %12s %12s %-8s %-10s %s",
		"Project", "Sessions", "Models", "Sources", "Tokens", "P90/P50", "P10/P50", "Health", "Confidence", "Reason"), width))
	for _, row := range limitProjectHotspots(rows, 8) {
		metric := projectHotspotCurrent(row)
		lines = append(lines, fit(fmt.Sprintf("  %-24s %8s %7s %7s %10s %12s %12s %-8s %-10s %s",
			truncate(shortPath(row.ProjectPath, 24), 24),
			formatInt(int64(row.SessionCount)),
			formatInt(int64(row.ModelCount)),
			formatInt(int64(row.SourceCount)),
			formatInt(row.TotalTokens),
			formatLatencyPair(metric),
			formatThroughputPair(metric),
			truncate(modelSignalSeverityLabel(row.Drift.Severity), 8),
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
	lines = append(lines, fit(fmt.Sprintf("  %-18s %-24s %-8s %-10s %6s %-8s %-10s %-10s %s",
		"Source", "Model", "Health", "Confidence", "Risk", "RiskLvl", "P90", "P10 tok/s", "Reason"), width))
	for _, row := range limitMatrixRows(rows, 6) {
		for _, cell := range limitMatrixCells(row.Cells, 4) {
			lines = append(lines, fit(fmt.Sprintf("  %-18s %-24s %-8s %-10s %6s %-8s %-10s %-10s %s",
				truncate(modelSignalSourceName(row.SourceLabel, row.AgentName, row.AgentKind, row.SourceKey), 18),
				truncate(modelProviderLabel(cell.ModelProvider, cell.Model), 24),
				truncate(modelSignalSeverityLabel(cell.Severity), 8),
				truncate(modelSignalConfidence(cell.Confidence), 10),
				formatSignalRate(cell.Current.DegradationRiskScore, 2),
				truncate(modelSignalRiskLevel(cell.Current.DegradationRiskScore), 8),
				formatLatencyPer1K(cell.Current),
				formatSignalRate(p10Throughput(cell.Current), 1),
				truncate(matrixCellReason(cell), 36),
			), width))
		}
		if len(row.Cells) > 4 {
			lines = append(lines, fit(fmt.Sprintf("  %-18s %s",
				truncate(modelSignalSourceName(row.SourceLabel, row.AgentName, row.AgentKind, row.SourceKey), 18),
				fmt.Sprintf("+%d more model cells", len(row.Cells)-4),
			), width))
		}
	}
	return lines
}

func appendModelSignalAnomalyLines(lines []string, rows []agentmodel.ModelSignalsAnomalySession, width int) []string {
	lines = append(lines, "", bold("Anomaly Sessions"))
	if len(rows) == 0 {
		return append(lines, "No anomaly sessions.")
	}
	lines = append(lines, fit(fmt.Sprintf("  %-13s %-12s %-18s %-18s %9s %6s %8s %8s %9s %8s %-11s %s",
		"Session", "Source", "Project", "Model", "Tokens", "Failed", "Output", "Reason", "Cache", "Tok/s", "Started", "Signal"), width))
	for _, row := range limitAnomalies(rows, 8) {
		lines = append(lines, fit(fmt.Sprintf("  %-13s %-12s %-18s %-18s %9s %6s %8s %8s %9s %8s %-11s %s",
			truncate(anomalySessionLabel(row), 13),
			truncate(modelSignalSourceName(row.SourceLabel, row.AgentName, row.AgentKind, row.SourceKey), 12),
			truncate(shortPath(row.ProjectPath, 18), 18),
			truncate(empty(row.Model, "unknown"), 18),
			formatInt(row.TotalTokens),
			formatInt(int64(row.FailedToolCalls)),
			formatSignalRate(row.OutputExpansionRate, 2)+"x",
			formatSignalPercent(row.ReasoningTokenShare),
			formatSignalPercent(row.CacheMissRate),
			formatSignalRate(row.ModelThroughputTokensPerSecond, 1),
			formatTime(row.StartedAt),
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

func projectHotspotCurrent(row agentmodel.ModelSignalsProjectHotspot) agentmodel.ModelSignalsMetricSet {
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

func formatLatencyPair(metric agentmodel.ModelSignalsMetricSet) string {
	p90 := metric.P90ModelLatencyMsPer1kOutputTokens
	if p90 <= 0 {
		p90 = metric.ModelLatencyMsPer1kOutputTokens
	}
	p50 := metric.P50ModelLatencyMsPer1kOutputTokens
	if p50 <= 0 {
		p50 = metric.ModelLatencyMsPer1kOutputTokens
	}
	if p50 <= 0 {
		p50 = p90
	}
	if p90 <= 0 {
		p90 = p50
	}
	if p90 <= 0 && p50 <= 0 {
		return "-"
	}
	return formatSignalRate(p90, 0) + "/" + formatSignalRate(p50, 0)
}

func p10Throughput(metric agentmodel.ModelSignalsMetricSet) float64 {
	if metric.P10ModelThroughputTokensPerSecond > 0 {
		return metric.P10ModelThroughputTokensPerSecond
	}
	return metric.ModelThroughputTokensPerSecond
}

func formatThroughputPair(metric agentmodel.ModelSignalsMetricSet) string {
	p10 := p10Throughput(metric)
	p50 := metric.P50ModelThroughputTokensPerSecond
	if p50 <= 0 {
		p50 = metric.ModelThroughputTokensPerSecond
	}
	if p50 <= 0 {
		p50 = p10
	}
	if p10 <= 0 {
		p10 = p50
	}
	if p10 <= 0 && p50 <= 0 {
		return "-"
	}
	return formatSignalRate(p10, 1) + "/" + formatSignalRate(p50, 1)
}

func formatFailurePressure(metric agentmodel.ModelSignalsMetricSet) string {
	value := metric.FailurePressure
	if value <= 0 && metric.SessionCount > 0 {
		value = float64(metric.FailedModelCalls+metric.FailedToolCalls) / float64(metric.SessionCount)
	}
	return formatSignalRate(value, 2)
}

func matrixCellReason(cell agentmodel.ModelSignalsMatrixCell) string {
	if strings.TrimSpace(cell.KeyReason) != "" {
		return cell.KeyReason
	}
	return modelSignalDriftReason(cell.Drift)
}

func modelSignalRiskLevel(score float64) string {
	switch {
	case score >= 0.75:
		return "high"
	case score >= 0.45:
		return "elevated"
	case score >= 0.20:
		return "watch"
	default:
		return "low"
	}
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

func recentTrendPoints(rows []agentmodel.ModelSignalsTrendPoint, limit int) []agentmodel.ModelSignalsTrendPoint {
	result := append([]agentmodel.ModelSignalsTrendPoint(nil), rows...)
	sort.SliceStable(result, func(i, j int) bool { return result[i].Date > result[j].Date })
	return limitSlice(result, limit)
}

func modelSignalBestLatency(rows []agentmodel.ModelSignalsDailyMetric, p90 bool) string {
	for _, row := range recentDailyMetrics(rows, len(rows)) {
		value := row.ModelLatencyMsPer1kOutputTokens
		if p90 {
			if row.P90ModelLatencyMsPer1kOutputTokens > 0 {
				value = row.P90ModelLatencyMsPer1kOutputTokens
			}
		} else if row.P50ModelLatencyMsPer1kOutputTokens > 0 {
			value = row.P50ModelLatencyMsPer1kOutputTokens
		}
		if value > 0 {
			return formatSignalRate(value, 0) + " ms/1k"
		}
	}
	return "-"
}

func modelSignalWorstThroughput(rows []agentmodel.ModelSignalsDailyMetric, p10 bool) string {
	for _, row := range recentDailyMetrics(rows, len(rows)) {
		value := row.ModelThroughputTokensPerSecond
		if p10 && row.P10ModelThroughputTokensPerSecond > 0 {
			value = row.P10ModelThroughputTokensPerSecond
		}
		if value > 0 {
			return formatSignalRate(value, 1) + " tok/s"
		}
	}
	return "-"
}

func modelSignalTotalCost(signals agentmodel.ModelSignals) string {
	var total float64
	var found bool
	for _, row := range signals.DailyMetrics {
		if row.EstimatedCostUSD != nil {
			total += *row.EstimatedCostUSD
			found = true
		}
	}
	if !found {
		for _, row := range signals.ProjectMetrics {
			metric := projectMetricCurrent(row)
			if metric.EstimatedCostUSD != nil {
				total += *metric.EstimatedCostUSD
				found = true
			}
		}
	}
	if !found {
		return "-"
	}
	return formatCost(&total)
}

func modelSignalTotalCacheSavings(daily []agentmodel.ModelSignalsDailyMetric, projects []agentmodel.ModelSignalsProjectMetric) string {
	var total float64
	var found bool
	for _, row := range daily {
		if row.CacheSavingsUSD != nil {
			total += *row.CacheSavingsUSD
			found = true
		}
	}
	if !found {
		for _, row := range projects {
			metric := projectMetricCurrent(row)
			if metric.CacheSavingsUSD != nil {
				total += *metric.CacheSavingsUSD
				found = true
			}
		}
	}
	if !found {
		return "-"
	}
	return formatCost(&total)
}

func modelSignalLatestCostPerSession(rows []agentmodel.ModelSignalsDailyMetric) string {
	for _, row := range recentDailyMetrics(rows, len(rows)) {
		if row.CostPerSession != nil {
			return formatCost(row.CostPerSession)
		}
	}
	return "-"
}

func modelSignalLatestCostPerActiveHour(rows []agentmodel.ModelSignalsDailyMetric) string {
	for _, row := range recentDailyMetrics(rows, len(rows)) {
		if row.CostPerActiveHour != nil {
			return formatCost(row.CostPerActiveHour)
		}
	}
	return "-"
}

func modelSignalLatestFailurePressure(rows []agentmodel.ModelSignalsDailyMetric) string {
	for _, row := range recentDailyMetrics(rows, len(rows)) {
		if row.SessionCount > 0 || row.FailurePressure > 0 {
			return formatFailurePressure(row.ModelSignalsMetricSet)
		}
	}
	return "-"
}

func modelSignalLatestModelFailureRate(rows []agentmodel.ModelSignalsDailyMetric) string {
	for _, row := range recentDailyMetrics(rows, len(rows)) {
		if row.ModelCalls > 0 {
			return formatSignalPercent(float64(row.FailedModelCalls) / float64(row.ModelCalls))
		}
	}
	return "-"
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
