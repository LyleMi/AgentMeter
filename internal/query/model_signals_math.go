package query

import (
	"math"
	"sort"
	"strings"

	"AgentMeter/internal/model"
)

func safeRate(numerator, denominator int64) float64 {
	if denominator <= 0 || numerator <= 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

func safeRateInt(numerator, denominator int) float64 {
	if denominator <= 0 || numerator <= 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

func reasoningOutputSemantics(inputTokens, cachedInputTokens, outputTokens, reasoningOutputTokens, totalTokens int64, modelName string) (visibleOutputTokens int64, billableOutputTokens int64, reasoningTokenShare float64, reasoningOverheadRate float64) {
	if outputTokens <= 0 {
		return 0, 0, 0, 0
	}
	if reasoningOutputTokens <= 0 {
		return outputTokens, outputTokens, 0, 0
	}
	separateReasoning := reasoningOutputAppearsSeparate(inputTokens, cachedInputTokens, outputTokens, reasoningOutputTokens, totalTokens, modelName)
	visibleOutputTokens, billableOutputTokens = reasoningOutputDenominators(outputTokens, reasoningOutputTokens, separateReasoning)
	reasoningTokenShare, reasoningOverheadRate = reasoningRates(reasoningOutputTokens, visibleOutputTokens, billableOutputTokens)
	return visibleOutputTokens, billableOutputTokens, reasoningTokenShare, reasoningOverheadRate
}

func reasoningOutputAppearsSeparate(inputTokens, cachedInputTokens, outputTokens, reasoningOutputTokens, totalTokens int64, modelName string) bool {
	if outputTokens <= 0 || reasoningOutputTokens <= 0 || totalTokens <= 0 {
		return false
	}
	if reasoningOutputTokens > outputTokens {
		return true
	}
	normalizedModel := strings.ToLower(strings.TrimSpace(modelName))
	normalizedModel = strings.TrimPrefix(normalizedModel, "models/")
	if !strings.Contains(normalizedModel, "gemini") {
		return false
	}
	promptTokens := inputTokens
	if cachedInputTokens > inputTokens {
		promptTokens += cachedInputTokens
	}
	return totalTokens >= promptTokens+outputTokens+reasoningOutputTokens
}

func reasoningOutputDenominators(outputTokens, reasoningOutputTokens int64, separateReasoning bool) (visibleOutputTokens int64, billableOutputTokens int64) {
	if outputTokens <= 0 {
		return 0, 0
	}
	if reasoningOutputTokens <= 0 {
		return outputTokens, outputTokens
	}
	if separateReasoning {
		return outputTokens, outputTokens + reasoningOutputTokens
	}
	visibleOutputTokens = outputTokens - reasoningOutputTokens
	if visibleOutputTokens < 0 {
		visibleOutputTokens = 0
	}
	return visibleOutputTokens, outputTokens
}

func reasoningRates(reasoningOutputTokens, visibleOutputTokens, billableOutputTokens int64) (reasoningTokenShare float64, reasoningOverheadRate float64) {
	reasoningTokenShare = clamp01(safeRate(reasoningOutputTokens, billableOutputTokens))
	reasoningOverheadRate = safeRate(reasoningOutputTokens, visibleOutputTokens)
	return reasoningTokenShare, reasoningOverheadRate
}

func cacheMissRate(inputTokens, cachedInputTokens int64) float64 {
	denominator := cacheInputDenominator(inputTokens, cachedInputTokens)
	if denominator <= 0 {
		return 0
	}
	uncachedInputTokens := inputTokens
	if cachedInputTokens <= inputTokens {
		uncachedInputTokens = inputTokens - cachedInputTokens
	}
	return clamp01(float64(uncachedInputTokens) / float64(denominator))
}

func cacheUtilizationRate(inputTokens, cachedInputTokens int64) float64 {
	denominator := cacheInputDenominator(inputTokens, cachedInputTokens)
	if denominator <= 0 || cachedInputTokens <= 0 {
		return 0
	}
	return clamp01(float64(cachedInputTokens) / float64(denominator))
}

func cacheInputDenominator(inputTokens, cachedInputTokens int64) int64 {
	if inputTokens <= 0 {
		if cachedInputTokens > 0 {
			return cachedInputTokens
		}
		return 0
	}
	if cachedInputTokens > inputTokens {
		return inputTokens + cachedInputTokens
	}
	return inputTokens
}

func throughputPerSecond(tokens, durationMS int64) float64 {
	if tokens <= 0 || durationMS <= 0 {
		return 0
	}
	return float64(tokens) / (float64(durationMS) / 1000)
}

func modelLatencyMSPer1kOutputTokens(outputTokens, durationMS int64) float64 {
	if outputTokens <= 0 || durationMS <= 0 {
		return 0
	}
	return float64(durationMS) / float64(outputTokens) * 1000
}

func modelSignalDegradationRiskScore(item model.ModelSignalsMetricSet) float64 {
	if item.SessionCount <= 0 || item.ModelCalls <= 0 {
		return 0
	}
	latency := firstPositiveFloat(item.P90ModelLatencyMsPer1kOutputTokens, item.ModelLatencyMsPer1kOutputTokens)
	throughput := firstPositiveFloat(
		item.P10ModelThroughputTokensPerSecond,
		item.ModelThroughputOutputTokensPerSecond,
		item.ModelThroughputTokensPerSecond,
	)
	score := 0.0
	score += thresholdScore(latency, 8_000, 20_000) * 0.24
	score += inverseThresholdScore(throughput, 40, 12) * 0.24
	score += rangeScore(item.FailurePressure, 0.05, 0.95) * 0.18
	score += rangeScore(item.ToolFailureRate, 0.08, 0.42) * 0.10
	score += rangeScore(item.CacheMissRate, 0.70, 0.30) * 0.08
	score += rangeScore(item.AvgModelCallsPerSession, 1.5, 2.5) * 0.07
	score += rangeScore(item.OutputExpansionRate, 3.0, 5.0) * 0.05
	score += rangeScore(item.ReasoningOverheadRate, 1.0, 4.0) * 0.04
	return clamp01(score)
}

func thresholdScore(value, warning, critical float64) float64 {
	if value <= warning || warning >= critical {
		return 0
	}
	if value >= critical {
		return 1
	}
	return clamp01((value - warning) / (critical - warning))
}

func inverseThresholdScore(value, warning, critical float64) float64 {
	if value <= 0 || warning <= critical {
		return 0
	}
	if value >= warning {
		return 0
	}
	if value <= critical {
		return 1
	}
	return clamp01((warning - value) / (warning - critical))
}

func rangeScore(value, start, span float64) float64 {
	if value <= start || span <= 0 {
		return 0
	}
	return clamp01((value - start) / span)
}

func firstPositiveFloat(values ...float64) float64 {
	for _, value := range values {
		if value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0) {
			return value
		}
	}
	return 0
}

func percentileNearest(values []float64, percentile float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := make([]float64, 0, len(values))
	for _, value := range values {
		if value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0) {
			sorted = append(sorted, value)
		}
	}
	if len(sorted) == 0 {
		return 0
	}
	sort.Float64s(sorted)
	if percentile <= 0 {
		return sorted[0]
	}
	if percentile >= 1 {
		return sorted[len(sorted)-1]
	}
	index := int(math.Ceil(percentile*float64(len(sorted)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

func safeDeltaPct(current, baseline float64) float64 {
	if baseline <= 0 {
		return 0
	}
	value := (current - baseline) / baseline
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	return value
}

func clamp01(value float64) float64 {
	if value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}
