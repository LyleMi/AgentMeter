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

	writer := newFittedLineWriter(lines, width)
	appendModelSignalHeaderLines(writer, signals)
	switch tab {
	case modelSignalsTabOverview:
		appendModelSignalOverviewLines(writer, signals)
	case modelSignalsTabDaily:
		appendModelSignalDailyLines(writer, signals.DailyMetrics)
	case modelSignalsTabCohorts:
		appendModelSignalCohortLines(writer, signals.Cohorts)
	case modelSignalsTabMatrix:
		appendModelSignalMatrixLines(writer, signals.Matrix)
	case modelSignalsTabProjects:
		appendModelSignalProjectLines(writer, signals)
	case modelSignalsTabAnomalies:
		appendModelSignalAnomalyLines(writer, signals.AnomalySessions)
	default:
		appendModelSignalMetricExplorerLines(writer, signals)
	}
	return writer.result()
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

func appendModelSignalHeaderLines(w *fittedLineWriter, signals agentmodel.ModelSignals) {
	summary := signals.HealthSummary
	w.append("",
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
		w.appendFit("Top reasons: " + strings.Join(limitStrings(summary.TopReasons, 4), "; "))
	}
}

func appendModelSignalOverviewLines(w *fittedLineWriter, signals agentmodel.ModelSignals) {
	w.append("", bold("Health Overview"))
	appendModelSignalBreakdownLines(w, signals.ModelBreakdown)
	appendModelSignalCohortLines(w, signals.Cohorts)
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

func appendModelSignalMetricExplorerLines(w *fittedLineWriter, signals agentmodel.ModelSignals) {
	w.append("", bold("Metric Explorer"))
	w.append(dim("Terminal view of the Web chart metrics. Switch tabs for source tables."))
	appendMetricExplorerSection(w, "Performance", []metricExplorerRow{
		{label: "P90 latency", value: modelSignalBestLatency(signals.DailyMetrics, true), note: "slower is worse"},
		{label: "P50 latency", value: modelSignalBestLatency(signals.DailyMetrics, false), note: "typical ms/1k output"},
		{label: "P10 throughput", value: modelSignalWorstThroughput(signals.DailyMetrics, true), note: "lower tail tok/s"},
		{label: "Output throughput", value: formatSignalRate(signals.ModelThroughputOutputTokensPerSecond, 1) + " tok/s", note: "visible output speed"},
	})
	appendMetricExplorerSection(w, "Cost", []metricExplorerRow{
		{label: "Estimated cost", value: modelSignalTotalCost(signals), note: "priced indexed usage"},
		{label: "Cost/session", value: modelSignalLatestCostPerSession(signals.DailyMetrics), note: "latest daily cost burn"},
		{label: "Cost/active hour", value: modelSignalLatestCostPerActiveHour(signals.DailyMetrics), note: "active-time normalized"},
		{label: "Cache savings", value: modelSignalTotalCacheSavings(signals.DailyMetrics, signals.ProjectMetrics), note: "cached-input discount"},
	})
	appendMetricExplorerSection(w, "Pressure", []metricExplorerRow{
		{label: "Failure pressure", value: modelSignalLatestFailurePressure(signals.DailyMetrics), note: "failed model/tool calls per session"},
		{label: "Retry pressure", value: formatSignalRate(signals.AvgModelCallsPerSession, 2) + "/session", note: "model calls per session"},
		{label: "Model failure rate", value: modelSignalLatestModelFailureRate(signals.DailyMetrics), note: "failed model calls"},
		{label: "Tool failure rate", value: formatSignalPercent(signals.ToolFailureRate), note: "failed tool calls"},
	})
	appendMetricExplorerSection(w, "Shape", []metricExplorerRow{
		{label: "Cache miss", value: formatSignalPercent(signals.CacheMissRate), note: "uncached input share"},
		{label: "Reasoning share", value: formatSignalPercent(signals.ReasoningTokenShare), note: "reasoning output share"},
		{label: "Output/input", value: formatSignalRate(signals.OutputExpansionRate, 2) + "x", note: "generation expansion"},
		{label: "Tool dependency", value: formatSignalPercent(signals.ToolDependencyRate), note: "sessions with tools"},
	})
	if len(signals.Trend) > 0 {
		w.append("", bold("Recent Trend"))
		w.appendFit(fmt.Sprintf("  %-10s %8s %10s %10s %9s %9s %s",
			"Date", "Sessions", "Tokens", "Tok/s", "Failure", "CacheMiss", "Note"))
		for _, point := range recentTrendPoints(signals.Trend, 8) {
			note := ""
			if point.LowSample {
				note = "low sample"
			}
			w.appendFit(fmt.Sprintf("  %-10s %8s %10s %10s %9s %9s %s",
				truncate(point.Date, 10),
				formatInt(int64(point.SessionCount)),
				formatInt(point.TotalTokens),
				formatSignalRate(point.ModelThroughputTokensPerSecond, 1),
				formatSignalPercent(point.ToolFailureRate),
				formatSignalPercent(point.CacheMissRate),
				note,
			))
		}
	}
}

type metricExplorerRow struct {
	label string
	value string
	note  string
}

func appendMetricExplorerSection(w *fittedLineWriter, title string, rows []metricExplorerRow) {
	w.append("", bold(title))
	for _, row := range rows {
		w.appendFit(fmt.Sprintf("  %-24s %-14s %s", row.label, row.value, row.note))
	}
}

func appendModelSignalBreakdownLines(w *fittedLineWriter, rows []agentmodel.ModelSignalsBreakdown) {
	appendFittedLineSection(w, fittedRowSection[agentmodel.ModelSignalsBreakdown]{
		title: "Model Breakdown",
		empty: "No model breakdown rows.",
		table: fittedRowTable[agentmodel.ModelSignalsBreakdown]{
			header: fmt.Sprintf("  %-24s %8s %8s %11s %9s %9s %9s %10s",
				"Model", "Sessions", "Calls", "Tokens", "Cache", "Reason", "ToolFail", "Tok/s"),
			rows:  rows,
			limit: 8,
			rowLine: func(row agentmodel.ModelSignalsBreakdown) string {
				return fmt.Sprintf("  %-24s %8s %8s %11s %9s %9s %9s %10s",
					truncate(empty(row.Model, "unknown"), 24),
					formatInt(int64(row.SessionCount)),
					formatInt(int64(row.ModelCalls)),
					formatInt(row.TotalTokens),
					formatSignalPercent(row.CacheMissRate),
					formatSignalPercent(row.ReasoningTokenShare),
					formatSignalPercent(row.ToolFailureRate),
					formatSignalRate(row.ModelThroughputTokensPerSecond, 1),
				)
			},
		},
	})
}

func appendModelSignalCohortLines(w *fittedLineWriter, rows []agentmodel.ModelSignalsCohort) {
	appendFittedLineSection(w, fittedRowSection[agentmodel.ModelSignalsCohort]{
		title: "Top Drift Cohorts",
		empty: "No cohort drift rows.",
		table: fittedRowTable[agentmodel.ModelSignalsCohort]{
			header: fmt.Sprintf("  %-14s %-18s %-20s %8s %12s %12s %10s %8s %-8s %-10s %s",
				"Source", "Project", "Model", "Samples", "P90/P50", "P10/P50", "Out tok/s", "Failure", "Health", "Confidence", "Reason"),
			rows:  topSignalCohorts(rows, 8),
			limit: 8,
			rowLine: func(row agentmodel.ModelSignalsCohort) string {
				metric := cohortCurrentMetric(row)
				return fmt.Sprintf("  %-14s %-18s %-20s %8s %12s %12s %10s %8s %-8s %-10s %s",
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
				)
			},
		},
	})
}

func appendModelSignalDailyLines(w *fittedLineWriter, rows []agentmodel.ModelSignalsDailyMetric) {
	appendFittedLineSection(w, fittedRowSection[agentmodel.ModelSignalsDailyMetric]{
		title: "Daily Efficiency",
		empty: "No daily efficiency rows.",
		table: fittedRowTable[agentmodel.ModelSignalsDailyMetric]{
			header: fmt.Sprintf("  %-10s %8s %9s %9s %9s %8s %12s %12s %8s %8s %6s %-8s %s",
				"Date", "Sessions", "Cost", "Cost/S", "Cost/H", "Saved", "P90/P50", "P10/P50", "Retry", "Failure", "Risk", "Health", "Reason"),
			rows:  recentDailyMetrics(rows, 8),
			limit: 8,
			rowLine: func(row agentmodel.ModelSignalsDailyMetric) string {
				return fmt.Sprintf("  %-10s %8s %9s %9s %9s %8s %12s %12s %8s %8s %6s %-8s %s",
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
				)
			},
		},
	})
}

func appendModelSignalProjectLines(w *fittedLineWriter, signals agentmodel.ModelSignals) {
	if len(signals.ProjectMetrics) > 0 {
		appendModelSignalProjectMetricLines(w, signals.ProjectMetrics)
		return
	}
	appendModelSignalProjectHotspotLines(w, signals.ProjectHotspots)
}

func appendModelSignalProjectMetricLines(w *fittedLineWriter, rows []agentmodel.ModelSignalsProjectMetric) {
	appendFittedLineSection(w, fittedRowSection[agentmodel.ModelSignalsProjectMetric]{
		title: "Project Hotspots",
		table: fittedRowTable[agentmodel.ModelSignalsProjectMetric]{
			header: fmt.Sprintf("  %-22s %8s %-20s %9s %8s %-8s %12s %12s %8s %6s %s",
				"Project", "Sessions", "Dominant model", "Cost", "Saved", "Health", "P90/P50", "P10/P50", "Failure", "Risk", "Reason"),
			rows:  rows,
			limit: 8,
			rowLine: func(row agentmodel.ModelSignalsProjectMetric) string {
				metric := projectMetricCurrent(row)
				return fmt.Sprintf("  %-22s %8s %-20s %9s %8s %-8s %12s %12s %8s %6s %s",
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
				)
			},
		},
	})
}

func appendModelSignalProjectHotspotLines(w *fittedLineWriter, rows []agentmodel.ModelSignalsProjectHotspot) {
	appendFittedLineSection(w, fittedRowSection[agentmodel.ModelSignalsProjectHotspot]{
		title: "Project Hotspots",
		empty: "No project hotspot rows.",
		table: fittedRowTable[agentmodel.ModelSignalsProjectHotspot]{
			header: fmt.Sprintf("  %-24s %8s %7s %7s %10s %12s %12s %-8s %-10s %s",
				"Project", "Sessions", "Models", "Sources", "Tokens", "P90/P50", "P10/P50", "Health", "Confidence", "Reason"),
			rows:  rows,
			limit: 8,
			rowLine: func(row agentmodel.ModelSignalsProjectHotspot) string {
				metric := projectHotspotCurrent(row)
				return fmt.Sprintf("  %-24s %8s %7s %7s %10s %12s %12s %-8s %-10s %s",
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
				)
			},
		},
	})
}

func appendModelSignalMatrixLines(w *fittedLineWriter, rows []agentmodel.ModelSignalsMatrixRow) {
	w.append("", bold("Source Model Matrix"))
	if len(rows) == 0 {
		w.append("No source/model matrix rows.")
		return
	}
	w.appendFit(fmt.Sprintf("  %-18s %-24s %-8s %-10s %6s %-8s %-10s %-10s %s",
		"Source", "Model", "Health", "Confidence", "Risk", "RiskLvl", "P90", "P10 tok/s", "Reason"))
	for _, row := range limitMatrixRows(rows, 6) {
		for _, cell := range limitMatrixCells(row.Cells, 4) {
			w.appendFit(fmt.Sprintf("  %-18s %-24s %-8s %-10s %6s %-8s %-10s %-10s %s",
				truncate(modelSignalSourceName(row.SourceLabel, row.AgentName, row.AgentKind, row.SourceKey), 18),
				truncate(modelProviderLabel(cell.ModelProvider, cell.Model), 24),
				truncate(modelSignalSeverityLabel(cell.Severity), 8),
				truncate(modelSignalConfidence(cell.Confidence), 10),
				formatSignalRate(cell.Current.DegradationRiskScore, 2),
				truncate(modelSignalRiskLevel(cell.Current.DegradationRiskScore), 8),
				formatLatencyPer1K(cell.Current),
				formatSignalRate(p10Throughput(cell.Current), 1),
				truncate(matrixCellReason(cell), 36),
			))
		}
		if len(row.Cells) > 4 {
			w.appendFit(fmt.Sprintf("  %-18s %s",
				truncate(modelSignalSourceName(row.SourceLabel, row.AgentName, row.AgentKind, row.SourceKey), 18),
				fmt.Sprintf("+%d more model cells", len(row.Cells)-4),
			))
		}
	}
}

func appendModelSignalAnomalyLines(w *fittedLineWriter, rows []agentmodel.ModelSignalsAnomalySession) {
	appendFittedLineSection(w, fittedRowSection[agentmodel.ModelSignalsAnomalySession]{
		title: "Anomaly Sessions",
		empty: "No anomaly sessions.",
		table: fittedRowTable[agentmodel.ModelSignalsAnomalySession]{
			header: fmt.Sprintf("  %-13s %-12s %-18s %-18s %9s %6s %8s %8s %9s %8s %-11s %s",
				"Session", "Source", "Project", "Model", "Tokens", "Failed", "Output", "Reason", "Cache", "Tok/s", "Started", "Signal"),
			rows:  rows,
			limit: 8,
			rowLine: func(row agentmodel.ModelSignalsAnomalySession) string {
				return fmt.Sprintf("  %-13s %-12s %-18s %-18s %9s %6s %8s %8s %9s %8s %-11s %s",
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
				)
			},
		},
	})
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

func limitMatrixRows(rows []agentmodel.ModelSignalsMatrixRow, limit int) []agentmodel.ModelSignalsMatrixRow {
	return limitSlice(rows, limit)
}

func limitMatrixCells(rows []agentmodel.ModelSignalsMatrixCell, limit int) []agentmodel.ModelSignalsMatrixCell {
	return limitSlice(rows, limit)
}

func limitStrings(values []string, limit int) []string {
	return limitSlice(values, limit)
}
