package query

import (
	"sort"
	"strings"

	"github.com/LyleMi/AgentMeter/internal/model"
)

type modelSignalDriftRuleKind int

const (
	modelSignalDriftRelativeIncrease modelSignalDriftRuleKind = iota
	modelSignalDriftRelativeDecrease
	modelSignalDriftAbsoluteIncrease
)

type modelSignalDriftField int

const (
	modelSignalDriftP90Latency modelSignalDriftField = iota
	modelSignalDriftLatency
	modelSignalDriftP10Throughput
	modelSignalDriftOutputThroughput
	modelSignalDriftToolFailureRate
	modelSignalDriftFailurePressure
	modelSignalDriftCacheMissRate
	modelSignalDriftAvgModelCalls
	modelSignalDriftOutputExpansion
	modelSignalDriftReasoningOverhead
	modelSignalDriftDegradationRisk
)

type modelSignalDriftRule struct {
	kind              modelSignalDriftRuleKind
	field             modelSignalDriftField
	key               string
	label             string
	direction         string
	reason            string
	warningThreshold  float64
	criticalThreshold float64
	minimumThreshold  float64
}

var modelSignalDriftRules = []modelSignalDriftRule{
	{
		kind:              modelSignalDriftRelativeIncrease,
		field:             modelSignalDriftP90Latency,
		key:               "p90ModelLatencyMsPer1kOutputTokens",
		label:             "p90 model latency per 1k output tokens",
		direction:         "higher_worse",
		reason:            "model latency increased",
		warningThreshold:  0.5,
		criticalThreshold: 1.0,
		minimumThreshold:  250,
	},
	{
		kind:              modelSignalDriftRelativeIncrease,
		field:             modelSignalDriftLatency,
		key:               "modelLatencyMsPer1kOutputTokens",
		label:             "model latency per 1k output tokens",
		direction:         "higher_worse",
		reason:            "model latency increased",
		warningThreshold:  0.5,
		criticalThreshold: 1.0,
		minimumThreshold:  250,
	},
	{
		kind:              modelSignalDriftRelativeDecrease,
		field:             modelSignalDriftP10Throughput,
		key:               "p10ModelThroughputTokensPerSecond",
		label:             "p10 model throughput",
		direction:         "lower_worse",
		reason:            "output throughput dropped",
		warningThreshold:  0.25,
		criticalThreshold: 0.5,
		minimumThreshold:  25,
	},
	{
		kind:              modelSignalDriftRelativeDecrease,
		field:             modelSignalDriftOutputThroughput,
		key:               "modelThroughputOutputTokensPerSecond",
		label:             "model output throughput",
		direction:         "lower_worse",
		reason:            "output throughput dropped",
		warningThreshold:  0.25,
		criticalThreshold: 0.5,
		minimumThreshold:  25,
	},
	{
		kind:              modelSignalDriftAbsoluteIncrease,
		field:             modelSignalDriftToolFailureRate,
		key:               "toolFailureRate",
		label:             "tool failure rate",
		direction:         "higher_downstream_symptom",
		reason:            "tool failure rate increased",
		warningThreshold:  0.10,
		criticalThreshold: 0.25,
		minimumThreshold:  0.10,
	},
	{
		kind:              modelSignalDriftAbsoluteIncrease,
		field:             modelSignalDriftFailurePressure,
		key:               "failurePressure",
		label:             "failure pressure",
		direction:         "higher_worse",
		reason:            "failure pressure increased",
		warningThreshold:  0.10,
		criticalThreshold: 0.25,
		minimumThreshold:  0.10,
	},
	{
		kind:              modelSignalDriftAbsoluteIncrease,
		field:             modelSignalDriftCacheMissRate,
		key:               "cacheMissRate",
		label:             "cache miss rate",
		direction:         "higher_symptom",
		reason:            "cache miss rate increased",
		warningThreshold:  0.20,
		criticalThreshold: 0.40,
		minimumThreshold:  0.50,
	},
	{
		kind:              modelSignalDriftRelativeIncrease,
		field:             modelSignalDriftAvgModelCalls,
		key:               "avgModelCallsPerSession",
		label:             "model calls per session",
		direction:         "higher_retry_loop_symptom",
		reason:            "model calls per session increased",
		warningThreshold:  0.5,
		criticalThreshold: 1.0,
		minimumThreshold:  0.5,
	},
	{
		kind:              modelSignalDriftRelativeIncrease,
		field:             modelSignalDriftOutputExpansion,
		key:               "outputExpansionRate",
		label:             "output expansion rate",
		direction:         "behavior_higher",
		reason:            "output expansion increased",
		warningThreshold:  1.0,
		criticalThreshold: 2.0,
		minimumThreshold:  1.0,
	},
	{
		kind:              modelSignalDriftAbsoluteIncrease,
		field:             modelSignalDriftReasoningOverhead,
		key:               "reasoningOverheadRate",
		label:             "reasoning overhead rate",
		direction:         "cost_shape_review",
		reason:            "reasoning overhead increased",
		warningThreshold:  0.50,
		criticalThreshold: 1.00,
		minimumThreshold:  0.50,
	},
	{
		kind:              modelSignalDriftAbsoluteIncrease,
		field:             modelSignalDriftDegradationRisk,
		key:               "degradationRiskScore",
		label:             "model quality risk score",
		direction:         "higher_worse",
		reason:            "model quality risk increased",
		warningThreshold:  0.15,
		criticalThreshold: 0.30,
		minimumThreshold:  0.30,
	},
}

func compareModelSignalDrift(current, baseline model.ModelSignalsMetricSet) model.ModelSignalsDrift {
	drift := model.ModelSignalsDrift{
		Severity:   modelSignalSeverityHealthy,
		Confidence: modelSignalConfidenceHigh,
		Reasons:    []string{},
		Metrics:    []model.ModelSignalsDriftMetric{},
	}
	if note := modelSignalSampleNote(current, baseline); note != "" {
		drift.Severity = modelSignalSeverityUnknown
		drift.Confidence = modelSignalConfidenceLow
		drift.SampleNote = note
		drift.Reasons = append(drift.Reasons, note)
		return drift
	}

	for _, rule := range modelSignalDriftRules {
		rule.addTo(&drift, current, baseline)
	}

	return drift
}

func (rule modelSignalDriftRule) addTo(drift *model.ModelSignalsDrift, current, baseline model.ModelSignalsMetricSet) {
	currentValue := rule.field.value(current)
	baselineValue := rule.field.value(baseline)
	metric, ok := rule.metric(currentValue, baselineValue)
	if !ok {
		return
	}
	appendModelSignalDriftMetric(drift, metric, rule.reason)
}

func (rule modelSignalDriftRule) metric(current, baseline float64) (model.ModelSignalsDriftMetric, bool) {
	delta := current - baseline
	deltaPct := safeDeltaPct(current, baseline)
	severity, ok := rule.severity(current, baseline, delta, deltaPct)
	if !ok {
		return model.ModelSignalsDriftMetric{}, false
	}
	return model.ModelSignalsDriftMetric{
		Key:       rule.key,
		Label:     rule.label,
		Direction: rule.direction,
		Severity:  severity,
		Current:   current,
		Baseline:  baseline,
		Delta:     delta,
		DeltaPct:  deltaPct,
	}, true
}

func (rule modelSignalDriftRule) severity(current, baseline, delta, deltaPct float64) (string, bool) {
	switch rule.kind {
	case modelSignalDriftRelativeIncrease:
		return rule.relativeIncreaseSeverity(current, baseline, delta, deltaPct)
	case modelSignalDriftRelativeDecrease:
		return rule.relativeDecreaseSeverity(current, baseline, delta, deltaPct)
	case modelSignalDriftAbsoluteIncrease:
		return rule.absoluteIncreaseSeverity(current, baseline, delta)
	default:
		return "", false
	}
}

func (rule modelSignalDriftRule) relativeIncreaseSeverity(current, baseline, delta, deltaPct float64) (string, bool) {
	if baseline <= 0 || current <= baseline || delta < rule.minimumThreshold {
		return "", false
	}
	if deltaPct >= rule.criticalThreshold {
		return modelSignalSeverityCritical, true
	}
	if deltaPct >= rule.warningThreshold {
		return modelSignalSeverityWarning, true
	}
	return "", false
}

func (rule modelSignalDriftRule) relativeDecreaseSeverity(current, baseline, delta, deltaPct float64) (string, bool) {
	if baseline <= 0 || current >= baseline || -delta < rule.minimumThreshold {
		return "", false
	}
	if deltaPct <= -rule.criticalThreshold {
		return modelSignalSeverityCritical, true
	}
	if deltaPct <= -rule.warningThreshold {
		return modelSignalSeverityWarning, true
	}
	return "", false
}

func (rule modelSignalDriftRule) absoluteIncreaseSeverity(current, baseline, delta float64) (string, bool) {
	if current <= baseline || current < rule.minimumThreshold {
		return "", false
	}
	if delta >= rule.criticalThreshold {
		return modelSignalSeverityCritical, true
	}
	if delta >= rule.warningThreshold {
		return modelSignalSeverityWarning, true
	}
	return "", false
}

func (field modelSignalDriftField) value(item model.ModelSignalsMetricSet) float64 {
	switch field {
	case modelSignalDriftP90Latency:
		return item.P90ModelLatencyMsPer1kOutputTokens
	case modelSignalDriftLatency:
		return item.ModelLatencyMsPer1kOutputTokens
	case modelSignalDriftP10Throughput:
		return item.P10ModelThroughputTokensPerSecond
	case modelSignalDriftOutputThroughput:
		return item.ModelThroughputOutputTokensPerSecond
	case modelSignalDriftToolFailureRate:
		return item.ToolFailureRate
	case modelSignalDriftFailurePressure:
		return item.FailurePressure
	case modelSignalDriftCacheMissRate:
		return item.CacheMissRate
	case modelSignalDriftAvgModelCalls:
		return item.AvgModelCallsPerSession
	case modelSignalDriftOutputExpansion:
		return item.OutputExpansionRate
	case modelSignalDriftReasoningOverhead:
		return item.ReasoningOverheadRate
	case modelSignalDriftDegradationRisk:
		return item.DegradationRiskScore
	default:
		return 0
	}
}

func modelSignalSampleNote(current, baseline model.ModelSignalsMetricSet) string {
	switch {
	case current.SessionCount == 0 && baseline.SessionCount == 0:
		return "missing current and baseline windows"
	case current.SessionCount == 0:
		return "missing current window"
	case baseline.SessionCount == 0:
		return "missing baseline window"
	case current.ModelCalls == 0 || baseline.ModelCalls == 0:
		return "missing model call samples"
	case current.SessionCount < 2 || baseline.SessionCount < 2:
		return "low current or baseline sample"
	default:
		return ""
	}
}

func appendModelSignalDriftMetric(drift *model.ModelSignalsDrift, metric model.ModelSignalsDriftMetric, reason string) {
	drift.Metrics = append(drift.Metrics, metric)
	drift.Reasons = appendUniqueString(drift.Reasons, reason)
	drift.Severity = worseModelSignalSeverity(drift.Severity, metric.Severity)
}

func modelSignalMetricSetLowSample(item model.ModelSignalsMetricSet) bool {
	return item.SessionCount > 0 && (item.SessionCount < 3 || item.ModelCalls < 3 || item.ModelDurationMS <= 0)
}

func modelSignalSeverityRank(severity string) int {
	switch severity {
	case modelSignalSeverityCritical:
		return 0
	case modelSignalSeverityWarning:
		return 1
	case modelSignalSeverityUnknown:
		return 2
	case modelSignalSeverityHealthy:
		return 3
	default:
		return 4
	}
}

func worseModelSignalSeverity(left, right string) string {
	if left == "" {
		return right
	}
	if right == "" {
		return left
	}
	if modelSignalSeverityRank(right) < modelSignalSeverityRank(left) {
		return right
	}
	return left
}

func addModelSignalReasonCounts(counts map[string]int, reasons []string) {
	for _, reason := range reasons {
		if strings.TrimSpace(reason) == "" {
			continue
		}
		counts[reason]++
	}
}

func topModelSignalReasons(counts map[string]int, limit int) []string {
	if len(counts) == 0 || limit <= 0 {
		return []string{}
	}
	reasons := make([]string, 0, len(counts))
	for reason := range counts {
		reasons = append(reasons, reason)
	}
	sort.Slice(reasons, func(i, j int) bool {
		if counts[reasons[i]] != counts[reasons[j]] {
			return counts[reasons[i]] > counts[reasons[j]]
		}
		return reasons[i] < reasons[j]
	})
	if len(reasons) > limit {
		reasons = reasons[:limit]
	}
	return reasons
}

func firstModelSignalReason(reasons []string) string {
	if len(reasons) == 0 {
		return ""
	}
	return reasons[0]
}

func appendUniqueString(values []string, value string) []string {
	for _, candidate := range values {
		if candidate == value {
			return values
		}
	}
	return append(values, value)
}
