package query

import (
	"math"
	"sort"

	"AgentMeter/internal/db"
	"AgentMeter/internal/model"
)

func rankModelSignalAnomalies(metrics []modelSignalSessionMetric, limit int) []model.ModelSignalsAnomalySession {
	anomalies := make([]model.ModelSignalsAnomalySession, 0, len(metrics))
	for _, metric := range metrics {
		item := modelSignalsAnomalyFromMetric(metric)
		if len(item.ReasonLabels) == 0 {
			continue
		}
		anomalies = append(anomalies, item)
	}
	sort.Slice(anomalies, func(i, j int) bool {
		if anomalies[i].Score != anomalies[j].Score {
			return anomalies[i].Score > anomalies[j].Score
		}
		if !anomalies[i].StartedAt.Equal(anomalies[j].StartedAt) {
			return anomalies[i].StartedAt.After(anomalies[j].StartedAt)
		}
		return anomalies[i].SessionID > anomalies[j].SessionID
	})
	if len(anomalies) > limit {
		anomalies = anomalies[:limit]
	}
	return anomalies
}

func modelSignalsAnomalyFromMetric(metric modelSignalSessionMetric) model.ModelSignalsAnomalySession {
	sourceKey, sourceLabel := sourceIdentity(metric.SourceID, metric.AgentName, metric.AgentKind)
	item := model.ModelSignalsAnomalySession{
		SessionID:                            metric.SessionID,
		SourceID:                             metric.SourceID,
		SourceKey:                            sourceKey,
		SourceLabel:                          sourceLabel,
		SourceRootPath:                       metric.SourceRootPath,
		SourceSessionsPath:                   metric.SourceSessionsPath,
		AgentKind:                            metric.AgentKind,
		AgentName:                            metric.AgentName,
		SessionKey:                           metric.SessionKey,
		CodexSessionID:                       metric.CodexSessionID,
		ProjectPath:                          metric.ProjectPath,
		Model:                                metric.Model,
		StartedAt:                            db.ParseTime(metric.StartedAt),
		RawSourcePath:                        metric.RawSourcePath,
		ModelCalls:                           metric.ModelCalls,
		ToolCalls:                            metric.ToolCalls,
		FailedToolCalls:                      metric.FailedToolCalls,
		TotalTokens:                          metric.TotalTokens,
		InputTokens:                          metric.InputTokens,
		CachedInputTokens:                    metric.CachedInputTokens,
		OutputTokens:                         metric.OutputTokens,
		ReasoningOutputTokens:                metric.ReasoningOutputTokens,
		ModelDurationMS:                      metric.ModelDurationMS,
		OutputExpansionRate:                  safeRate(metric.OutputTokens, metric.InputTokens),
		CacheMissRate:                        cacheMissRate(metric.InputTokens, metric.CachedInputTokens),
		ModelThroughputTokensPerSecond:       throughputPerSecond(metric.TotalTokens, metric.ModelDurationMS),
		ModelThroughputOutputTokensPerSecond: throughputPerSecond(metric.OutputTokens, metric.ModelDurationMS),
		ToolFailureRate:                      safeRateInt(metric.FailedToolCalls, metric.ToolCalls),
		ReasonLabels:                         []string{},
	}
	item.VisibleOutputTokens = metric.VisibleOutputTokens
	item.BillableOutputTokens = metric.BillableOutputTokens
	item.ReasoningTokenShare, item.ReasoningOverheadRate = reasoningRates(metric.ReasoningOutputTokens, metric.VisibleOutputTokens, metric.BillableOutputTokens)
	if item.OutputExpansionRate >= 3 {
		item.ReasonLabels = append(item.ReasonLabels, "high output/input ratio")
		item.Score += math.Min(item.OutputExpansionRate/3, 5)
	}
	if item.TotalTokens > 0 && item.ModelDurationMS > 0 && item.ModelThroughputTokensPerSecond < 500 {
		item.ReasonLabels = append(item.ReasonLabels, "slow model throughput")
		item.Score += (500 - item.ModelThroughputTokensPerSecond) / 500
	}
	if item.FailedToolCalls >= 2 || (item.ToolCalls > 0 && item.ToolFailureRate >= 0.5) {
		item.ReasonLabels = append(item.ReasonLabels, "failed tool calls")
		item.Score += float64(item.FailedToolCalls) + item.ToolFailureRate
	}
	if item.InputTokens > 0 && item.CacheMissRate >= 0.8 {
		item.ReasonLabels = append(item.ReasonLabels, "high cache miss")
		item.Score += item.CacheMissRate
	}
	return item
}
