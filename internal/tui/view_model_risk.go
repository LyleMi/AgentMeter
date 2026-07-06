package tui

import (
	"fmt"
	"sort"
	"strings"

	agentmodel "github.com/LyleMi/AgentMeter/internal/model"
)

type modelRiskRow struct {
	Key             string
	Source          string
	SourceSecondary string
	Model           string
	Provider        string
	Score           float64
	Level           string
	Reason          string
	Confidence      string
	SampleNote      string
	SessionCount    int
	ModelCalls      int
	TotalTokens     int64
	CohortCount     int
	Drivers         []modelRiskDriver
}

type modelRiskDriver struct {
	Key          string
	Label        string
	Value        string
	Contribution float64
	Weight       float64
	Level        string
	Explanation  string
}

type modelRiskDriverInput struct {
	key          string
	label        string
	value        float64
	formatter    func(float64) string
	contribution float64
	weight       float64
	explanation  string
}

func (s *state) modelRiskViewportLines() []string {
	lines := modelRiskLines(s.signals, s.width)
	return s.viewportLines(lines)
}

func modelRiskLines(signals agentmodel.ModelSignals, width int) []string {
	rows := buildModelRiskRows(signals)
	lines := []string{
		bold("Model Risk"),
		dim("Quality-risk score is symptom triage, not proof of routing, throttling, or substitution."),
	}
	if len(rows) == 0 {
		return append(lines, "", "No model quality risk rows match the current scope.")
	}

	top := rows[0]
	high := countRiskRows(rows, "high")
	elevated := countRiskRows(rows, "elevated")
	lowConfidence := countLowConfidenceRiskRows(rows)
	lines = append(lines, "",
		bold("Summary"),
		fmt.Sprintf("Highest risk: %-8s %-9s Source: %-18s Model: %s",
			formatPercent(top.Score),
			top.Level,
			truncate(top.Source, 18),
			truncate(modelProviderLabel(top.Provider, top.Model), 28),
		),
		fmt.Sprintf("Rows: %-8s Elevated+: %-8s High: %-8s Low confidence: %s",
			formatInt(int64(len(rows))),
			formatInt(int64(high+elevated)),
			formatInt(int64(high)),
			formatInt(int64(lowConfidence)),
		),
		"Top reason: "+fit(top.Reason, width-12),
	)

	lines = append(lines, "", bold("Score Drivers"))
	if len(top.Drivers) == 0 {
		lines = append(lines, "No dominant risk driver.")
	} else {
		lines = append(lines, fit(fmt.Sprintf("  %-22s %-12s %-10s %-9s %s",
			"Driver", "Value", "Contrib", "Level", "Explanation"), width))
		for _, driver := range top.Drivers {
			lines = append(lines, fit(fmt.Sprintf("  %-22s %-12s %-10s %-9s %s",
				truncate(driver.Label, 22),
				truncate(driver.Value, 12),
				formatPercent(driver.Contribution),
				truncate(driver.Level, 9),
				truncate(driver.Explanation, 46),
			), width))
		}
	}

	lines = append(lines, "", bold("Risk Explanations"))
	lines = append(lines, fit(fmt.Sprintf("  %-16s %-24s %8s %-9s %-10s %8s %8s %10s %s",
		"Source", "Model", "Risk", "Level", "Confidence", "Sessions", "Calls", "Tokens", "Reason"), width))
	for _, row := range limitSlice(rows, 16) {
		lines = append(lines, fit(fmt.Sprintf("  %-16s %-24s %8s %-9s %-10s %8s %8s %10s %s",
			truncate(row.Source, 16),
			truncate(modelProviderLabel(row.Provider, row.Model), 24),
			formatPercent(row.Score),
			truncate(row.Level, 9),
			truncate(row.Confidence, 10),
			formatInt(int64(row.SessionCount)),
			formatInt(int64(row.ModelCalls)),
			formatInt(row.TotalTokens),
			truncate(row.Reason, 42),
		), width))
		if len(row.Drivers) > 0 {
			lines = append(lines, fit("    drivers: "+strings.Join(modelRiskDriverLabels(row.Drivers), "; "), width))
		}
		if strings.TrimSpace(row.SampleNote) != "" {
			lines = append(lines, fit("    sample: "+row.SampleNote, width))
		}
	}
	return lines
}

func buildModelRiskRows(signals agentmodel.ModelSignals) []modelRiskRow {
	var rows []modelRiskRow
	for _, matrixRow := range signals.Matrix {
		source := modelSignalSourceName(matrixRow.SourceLabel, matrixRow.AgentName, matrixRow.AgentKind, matrixRow.SourceKey)
		secondary := sourceContext(matrixRow.AgentKind, matrixRow.AgentName, matrixRow.SourceRootPath, matrixRow.SourceSessionsPath)
		for _, cell := range matrixRow.Cells {
			current := cell.Current
			score := agentmodel.ClampRiskScore(current.DegradationRiskScore)
			drivers := buildModelRiskDrivers(current)
			sort.SliceStable(drivers, func(i, j int) bool {
				if drivers[i].Contribution == drivers[j].Contribution {
					return drivers[i].Label < drivers[j].Label
				}
				return drivers[i].Contribution > drivers[j].Contribution
			})
			drivers = positiveRiskDrivers(drivers)
			reason := modelRiskPrimaryReason(cell, drivers)
			sample := strings.TrimSpace(cell.Drift.SampleNote)
			if sample == "" {
				sample = "Sample confidence is normal"
			}
			sessionCount := cell.SessionCount
			if sessionCount == 0 {
				sessionCount = current.SessionCount
			}
			modelCalls := cell.ModelCalls
			if modelCalls == 0 {
				modelCalls = current.ModelCalls
			}
			totalTokens := cell.TotalTokens
			if totalTokens == 0 {
				totalTokens = current.TotalTokens
			}
			key := matrixRow.SourceKey + ":" + cell.ModelProvider + ":" + cell.Model
			rows = append(rows, modelRiskRow{
				Key:             key,
				Source:          source,
				SourceSecondary: secondary,
				Model:           empty(cell.Model, "unknown"),
				Provider:        cell.ModelProvider,
				Score:           score,
				Level:           modelRiskLevel(score),
				Reason:          reason,
				Confidence:      modelSignalConfidence(cell.Confidence),
				SampleNote:      sample,
				SessionCount:    sessionCount,
				ModelCalls:      modelCalls,
				TotalTokens:     totalTokens,
				CohortCount:     cell.CohortCount,
				Drivers:         limitSlice(drivers, 4),
			})
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Score != rows[j].Score {
			return rows[i].Score > rows[j].Score
		}
		if rows[i].SessionCount != rows[j].SessionCount {
			return rows[i].SessionCount > rows[j].SessionCount
		}
		return rows[i].Source < rows[j].Source
	})
	return rows
}

func modelRiskPrimaryReason(cell agentmodel.ModelSignalsMatrixCell, drivers []modelRiskDriver) string {
	for _, reason := range cell.Drift.Reasons {
		if strings.TrimSpace(reason) != "" {
			return strings.TrimSpace(reason)
		}
	}
	if strings.TrimSpace(cell.KeyReason) != "" {
		return strings.TrimSpace(cell.KeyReason)
	}
	if len(drivers) > 0 {
		return drivers[0].Label
	}
	return "No dominant risk driver"
}

func buildModelRiskDrivers(metric agentmodel.ModelSignalsMetricSet) []modelRiskDriver {
	return []modelRiskDriver{
		newModelRiskDriver(modelRiskDriverInput{
			key:          "latency",
			label:        "Tail latency",
			value:        firstPositiveFloat(metric.P90ModelLatencyMsPer1kOutputTokens, metric.ModelLatencyMsPer1kOutputTokens),
			formatter:    func(value float64) string { return formatSignalRate(value, 0) + " ms/1k" },
			contribution: agentmodel.RiskThresholdScore(firstPositiveFloat(metric.P90ModelLatencyMsPer1kOutputTokens, metric.ModelLatencyMsPer1kOutputTokens), 8000, 20000) * 0.24,
			weight:       0.24,
			explanation:  "Tail responses are slow after normalizing by generated output.",
		}),
		newModelRiskDriver(modelRiskDriverInput{
			key:          "throughput",
			label:        "Slow-floor throughput",
			value:        firstPositiveFloat(metric.P10ModelThroughputTokensPerSecond, metric.ModelThroughputOutputTokensPerSecond, metric.ModelThroughputTokensPerSecond),
			formatter:    func(value float64) string { return formatSignalRate(value, 1) + " tok/s" },
			contribution: agentmodel.InverseRiskThresholdScore(firstPositiveFloat(metric.P10ModelThroughputTokensPerSecond, metric.ModelThroughputOutputTokensPerSecond, metric.ModelThroughputTokensPerSecond), 40, 12) * 0.24,
			weight:       0.24,
			explanation:  "Observed token throughput is below the expected floor.",
		}),
		newModelRiskDriver(modelRiskDriverInput{
			key:          "failurePressure",
			label:        "Failure pressure",
			value:        metric.FailurePressure,
			formatter:    func(value float64) string { return formatSignalRate(value, 2) + "/session" },
			contribution: agentmodel.RiskRangeScore(metric.FailurePressure, 0.05, 0.95) * 0.18,
			weight:       0.18,
			explanation:  "Model or tool failures are concentrated per session.",
		}),
		newModelRiskDriver(modelRiskDriverInput{
			key:          "toolFailureRate",
			label:        "Tool failures",
			value:        metric.ToolFailureRate,
			formatter:    formatSignalPercent,
			contribution: agentmodel.RiskRangeScore(metric.ToolFailureRate, 0.08, 0.42) * 0.10,
			weight:       0.10,
			explanation:  "Tool failures are taking a larger share of tool calls.",
		}),
		newModelRiskDriver(modelRiskDriverInput{
			key:          "cacheMiss",
			label:        "Cache misses",
			value:        metric.CacheMissRate,
			formatter:    formatSignalPercent,
			contribution: agentmodel.RiskRangeScore(metric.CacheMissRate, 0.70, 0.30) * 0.08,
			weight:       0.08,
			explanation:  "A high uncached input share can make the same work slower or more expensive.",
		}),
		newModelRiskDriver(modelRiskDriverInput{
			key:          "retryPressure",
			label:        "Retry pressure",
			value:        metric.AvgModelCallsPerSession,
			formatter:    func(value float64) string { return formatSignalRate(value, 2) + "/session" },
			contribution: agentmodel.RiskRangeScore(metric.AvgModelCallsPerSession, 1.5, 2.5) * 0.07,
			weight:       0.07,
			explanation:  "More model calls per session can indicate repair loops or unstable responses.",
		}),
		newModelRiskDriver(modelRiskDriverInput{
			key:          "outputExpansion",
			label:        "Output expansion",
			value:        metric.OutputExpansionRate,
			formatter:    func(value float64) string { return formatSignalRate(value, 2) + "x" },
			contribution: agentmodel.RiskRangeScore(metric.OutputExpansionRate, 3.0, 5.0) * 0.05,
			weight:       0.05,
			explanation:  "Generated output is large relative to input, changing latency and cost.",
		}),
		newModelRiskDriver(modelRiskDriverInput{
			key:          "reasoningOverhead",
			label:        "Reasoning overhead",
			value:        metric.ReasoningOverheadRate,
			formatter:    func(value float64) string { return formatSignalRate(value, 2) + "x" },
			contribution: agentmodel.RiskRangeScore(metric.ReasoningOverheadRate, 1.0, 4.0) * 0.04,
			weight:       0.04,
			explanation:  "Hidden reasoning output is high relative to visible output.",
		}),
	}
}

func newModelRiskDriver(input modelRiskDriverInput) modelRiskDriver {
	contribution := agentmodel.ClampRiskScore(input.contribution)
	normalized := 0.0
	if input.weight > 0 {
		normalized = contribution / input.weight
	}
	return modelRiskDriver{
		Key:          input.key,
		Label:        input.label,
		Value:        input.formatter(input.value),
		Contribution: contribution,
		Weight:       input.weight,
		Level:        modelRiskLevel(normalized),
		Explanation:  input.explanation,
	}
}

func positiveRiskDrivers(drivers []modelRiskDriver) []modelRiskDriver {
	result := make([]modelRiskDriver, 0, len(drivers))
	for _, driver := range drivers {
		if driver.Contribution > 0 {
			result = append(result, driver)
		}
	}
	return result
}

func modelRiskDriverLabels(drivers []modelRiskDriver) []string {
	labels := make([]string, 0, len(drivers))
	for _, driver := range drivers {
		labels = append(labels, driver.Label+" "+formatPercent(driver.Contribution))
	}
	return labels
}

func modelRiskTopRow(rows []modelRiskRow) modelRiskRow {
	if len(rows) == 0 {
		return modelRiskRow{}
	}
	return rows[0]
}

func countRiskRows(rows []modelRiskRow, level string) int {
	count := 0
	for _, row := range rows {
		if row.Level == level {
			count++
		}
	}
	return count
}

func countLowConfidenceRiskRows(rows []modelRiskRow) int {
	count := 0
	for _, row := range rows {
		if strings.EqualFold(row.Confidence, "low") {
			count++
		}
	}
	return count
}

func modelRiskLevel(score float64) string {
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

func firstPositiveFloat(values ...float64) float64 {
	for _, value := range values {
		if finiteSignal(value) && value > 0 {
			return value
		}
	}
	return 0
}
