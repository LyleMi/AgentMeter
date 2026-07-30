package query

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/LyleMi/AgentMeter/internal/db"
	"github.com/LyleMi/AgentMeter/internal/model"
)

func (s *Service) ModelSignals(ctx context.Context) (model.ModelSignals, error) {
	return s.ModelSignalsWithFilters(ctx, model.AnalyticsFilters{})
}

func (s *Service) ModelSignalsWithFilters(ctx context.Context, filters model.AnalyticsFilters) (model.ModelSignals, error) {
	if strings.TrimSpace(filters.StartedFrom) == "" && strings.TrimSpace(filters.StartedTo) == "" {
		allMetrics, err := s.modelSignalSessionMetrics(ctx, filters)
		if err != nil {
			return model.ModelSignals{}, err
		}
		anchor := latestModelSignalMetricStart(allMetrics)
		windows := modelSignalHealthWindowBounds(anchor)
		result := buildModelSignalsForWindows(allMetrics, allMetrics, windows)
		normalizeModelSignalsSlices(&result)
		return result, nil
	}
	windows, _, expandedFilters := selectedModelSignalHealthWindows(filters, time.Now().UTC())
	allMetrics, err := s.modelSignalSessionMetrics(ctx, expandedFilters)
	if err != nil {
		return model.ModelSignals{}, err
	}
	currentMetrics := modelSignalMetricsInWindow(allMetrics, windows.currentFrom, windows.anchor)
	result := buildModelSignalsForWindows(currentMetrics, allMetrics, windows)
	normalizeModelSignalsSlices(&result)
	return result, nil
}

func equalModelSignalHealthWindows(anchor time.Time, duration time.Duration) modelSignalHealthWindows {
	if anchor.IsZero() {
		return modelSignalHealthWindows{}
	}
	currentFrom := anchor.Add(-duration)
	baselineFrom := currentFrom.Add(-duration)
	return modelSignalHealthWindows{
		anchor:       anchor,
		currentFrom:  currentFrom,
		baselineFrom: baselineFrom,
		current: model.ModelSignalsWindow{
			From: db.FormatTime(currentFrom),
			To:   db.FormatTime(anchor),
		},
		baseline: model.ModelSignalsWindow{
			From: db.FormatTime(baselineFrom),
			To:   db.FormatTime(currentFrom),
		},
	}
}

func buildModelSignals(metrics []modelSignalSessionMetric) model.ModelSignals {
	return buildModelSignalsForWindows(metrics, metrics, modelSignalHealthWindowBounds(latestModelSignalMetricStart(metrics)))
}

func buildModelSignalsForWindows(currentMetrics, allMetrics []modelSignalSessionMetric, windows modelSignalHealthWindows) model.ModelSignals {
	var result model.ModelSignals
	aggregates := aggregateModelSignalMetrics(currentMetrics)
	applyModelSignalsTotals(&result, aggregates.totals.metricSet())
	result.ModelBreakdown = buildModelSignalsBreakdown(aggregates.breakdowns)
	result.Trend = buildModelSignalsTrend(aggregates.trendByDay)

	result.AnomalySessions = rankModelSignalAnomalies(currentMetrics, 8)
	result.DailyMetrics = buildModelSignalDailyMetricsForPeriod(currentMetrics, allMetrics)
	result.HealthSummary, result.Cohorts, result.Matrix, result.ProjectHotspots, result.ProjectMetrics = buildModelSignalHealthReadModelsWithWindows(allMetrics, windows)
	return result
}

func selectedModelSignalHealthWindows(filters model.AnalyticsFilters, now time.Time) (modelSignalHealthWindows, model.AnalyticsFilters, model.AnalyticsFilters) {
	to := modelSignalFilterBoundary(filters.StartedTo, now, true)
	from := modelSignalFilterBoundary(filters.StartedFrom, to.AddDate(0, 0, -7), false)
	if !from.Before(to) {
		from = to.AddDate(0, 0, -7)
	}
	baselineFrom := from.Add(-to.Sub(from))
	windows := modelSignalHealthWindows{
		anchor:       to,
		currentFrom:  from,
		baselineFrom: baselineFrom,
		current: model.ModelSignalsWindow{
			From: db.FormatTime(from),
			To:   db.FormatTime(to),
		},
		baseline: model.ModelSignalsWindow{
			From: db.FormatTime(baselineFrom),
			To:   db.FormatTime(from),
		},
	}
	current := filters
	current.StartedFrom = from.Format(time.RFC3339Nano)
	current.StartedTo = to.Format(time.RFC3339Nano)
	expanded := filters
	expanded.StartedFrom = baselineFrom.Format(time.RFC3339Nano)
	expanded.StartedTo = to.Format(time.RFC3339Nano)
	return windows, current, expanded
}

func modelSignalFilterBoundary(value string, fallback time.Time, end bool) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback.UTC()
	}
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed.UTC()
	}
	if parsed, err := time.ParseInLocation(analyticsDateOnlyLayout, value, time.UTC); err == nil {
		if end {
			return parsed.AddDate(0, 0, 1)
		}
		return parsed
	}
	return fallback.UTC()
}

func modelSignalMetricsInWindow(metrics []modelSignalSessionMetric, from, to time.Time) []modelSignalSessionMetric {
	result := make([]modelSignalSessionMetric, 0, len(metrics))
	for _, metric := range metrics {
		started := db.ParseTime(metric.StartedAt)
		if !started.IsZero() && !started.Before(from) && !started.After(to) {
			result = append(result, metric)
		}
	}
	return result
}

type modelSignalAggregates struct {
	totals     modelSignalMetricAccumulator
	breakdowns map[string]*modelSignalMetricAccumulator
	trendByDay map[string]*modelSignalMetricAccumulator
}

func aggregateModelSignalMetrics(metrics []modelSignalSessionMetric) modelSignalAggregates {
	aggregates := modelSignalAggregates{
		breakdowns: map[string]*modelSignalMetricAccumulator{},
		trendByDay: map[string]*modelSignalMetricAccumulator{},
	}
	for _, metric := range metrics {
		aggregates.totals.add(metric)
		accumulatorFor(aggregates.breakdowns, metric.Model).add(metric)
		if metric.Day != "" {
			accumulatorFor(aggregates.trendByDay, metric.Day).add(metric)
		}
	}
	return aggregates
}

func accumulatorFor(values map[string]*modelSignalMetricAccumulator, key string) *modelSignalMetricAccumulator {
	if values[key] == nil {
		values[key] = &modelSignalMetricAccumulator{}
	}
	return values[key]
}

func buildModelSignalsBreakdown(breakdowns map[string]*modelSignalMetricAccumulator) []model.ModelSignalsBreakdown {
	result := make([]model.ModelSignalsBreakdown, 0, len(breakdowns))
	for modelName, breakdown := range breakdowns {
		result = append(result, modelSignalsBreakdownFromMetricSet(modelName, breakdown.metricSet()))
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].TotalTokens != result[j].TotalTokens {
			return result[i].TotalTokens > result[j].TotalTokens
		}
		return result[i].Model < result[j].Model
	})
	return result
}

func buildModelSignalsTrend(trends map[string]*modelSignalMetricAccumulator) []model.ModelSignalsTrendPoint {
	result := make([]model.ModelSignalsTrendPoint, 0, len(trends))
	for day, point := range trends {
		result = append(result, modelSignalsTrendPointFromMetricSet(day, point.metricSet()))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Date < result[j].Date })
	if len(result) > 30 {
		result = result[len(result)-30:]
	}
	result = fillModelSignalsTrendGaps(result)
	applyModelSignalsRollingRates(result)
	return result
}

func applyModelSignalsTotals(result *model.ModelSignals, totals model.ModelSignalsMetricSet) {
	result.TotalSessions = totals.SessionCount
	result.TotalModelCalls = totals.ModelCalls
	result.TotalToolCalls = totals.ToolCalls
	result.FailedToolCalls = totals.FailedToolCalls
	result.ToolFailureRate = totals.ToolFailureRate
	result.ToolDependencyRate = totals.ToolDependencyRate
	result.AvgModelCallsPerSession = totals.AvgModelCallsPerSession
	result.OutputExpansionRate = totals.OutputExpansionRate
	result.ReasoningTokenShare = totals.ReasoningTokenShare
	result.ReasoningOverheadRate = totals.ReasoningOverheadRate
	result.VisibleOutputTokens = totals.VisibleOutputTokens
	result.BillableOutputTokens = totals.BillableOutputTokens
	result.CacheMissRate = totals.CacheMissRate
	result.ModelThroughputTokensPerSecond = totals.ModelThroughputTokensPerSecond
	result.ModelThroughputOutputTokensPerSecond = totals.ModelThroughputOutputTokensPerSecond
}

func modelSignalsBreakdownFromMetricSet(modelName string, item model.ModelSignalsMetricSet) model.ModelSignalsBreakdown {
	return model.ModelSignalsBreakdown{
		Model:                                modelName,
		SessionCount:                         item.SessionCount,
		ModelCalls:                           item.ModelCalls,
		ToolCalls:                            item.ToolCalls,
		FailedToolCalls:                      item.FailedToolCalls,
		TotalTokens:                          item.TotalTokens,
		InputTokens:                          item.InputTokens,
		CachedInputTokens:                    item.CachedInputTokens,
		OutputTokens:                         item.OutputTokens,
		ReasoningOutputTokens:                item.ReasoningOutputTokens,
		VisibleOutputTokens:                  item.VisibleOutputTokens,
		BillableOutputTokens:                 item.BillableOutputTokens,
		ModelDurationMS:                      item.ModelDurationMS,
		ToolFailureRate:                      item.ToolFailureRate,
		ToolDependencyRate:                   item.ToolDependencyRate,
		AvgModelCallsPerSession:              item.AvgModelCallsPerSession,
		OutputExpansionRate:                  item.OutputExpansionRate,
		ReasoningTokenShare:                  item.ReasoningTokenShare,
		ReasoningOverheadRate:                item.ReasoningOverheadRate,
		CacheMissRate:                        item.CacheMissRate,
		ModelThroughputTokensPerSecond:       item.ModelThroughputTokensPerSecond,
		ModelThroughputOutputTokensPerSecond: item.ModelThroughputOutputTokensPerSecond,
	}
}

func modelSignalsTrendPointFromMetricSet(day string, item model.ModelSignalsMetricSet) model.ModelSignalsTrendPoint {
	return model.ModelSignalsTrendPoint{
		Date:                                 day,
		SessionCount:                         item.SessionCount,
		ModelCalls:                           item.ModelCalls,
		ToolCalls:                            item.ToolCalls,
		FailedToolCalls:                      item.FailedToolCalls,
		TotalTokens:                          item.TotalTokens,
		InputTokens:                          item.InputTokens,
		CachedInputTokens:                    item.CachedInputTokens,
		OutputTokens:                         item.OutputTokens,
		ReasoningOutputTokens:                item.ReasoningOutputTokens,
		VisibleOutputTokens:                  item.VisibleOutputTokens,
		BillableOutputTokens:                 item.BillableOutputTokens,
		ModelDurationMS:                      item.ModelDurationMS,
		OutputExpansionRate:                  item.OutputExpansionRate,
		ReasoningTokenShare:                  item.ReasoningTokenShare,
		ReasoningOverheadRate:                item.ReasoningOverheadRate,
		CacheMissRate:                        item.CacheMissRate,
		ModelThroughputTokensPerSecond:       item.ModelThroughputTokensPerSecond,
		ModelThroughputOutputTokensPerSecond: item.ModelThroughputOutputTokensPerSecond,
		ToolFailureRate:                      item.ToolFailureRate,
		ToolDependencyRate:                   item.ToolDependencyRate,
		LowSample:                            modelSignalMetricSetLowSample(item),
	}
}

func applyModelSignalsRollingRates(points []model.ModelSignalsTrendPoint) {
	for index := range points {
		start := index - 6
		if start < 0 {
			start = 0
		}
		var totalTokens int64
		var modelDurationMS int64
		var toolCalls int
		var failedToolCalls int
		for cursor := start; cursor <= index; cursor++ {
			totalTokens += points[cursor].TotalTokens
			modelDurationMS += points[cursor].ModelDurationMS
			toolCalls += points[cursor].ToolCalls
			failedToolCalls += points[cursor].FailedToolCalls
		}
		points[index].RollingModelThroughputTokensPerSecond = throughputPerSecond(totalTokens, modelDurationMS)
		points[index].RollingToolFailureRate = safeRateInt(failedToolCalls, toolCalls)
	}
}

func fillModelSignalsTrendGaps(points []model.ModelSignalsTrendPoint) []model.ModelSignalsTrendPoint {
	return fillAnalyticsDateGaps(
		points,
		func(point model.ModelSignalsTrendPoint) string { return point.Date },
		func(date string) model.ModelSignalsTrendPoint { return model.ModelSignalsTrendPoint{Date: date} },
	)
}

func normalizeModelSignalsSlices(result *model.ModelSignals) {
	result.Trend = nonNilSlice(result.Trend)
	result.ModelBreakdown = nonNilSlice(result.ModelBreakdown)
	result.AnomalySessions = nonNilSlice(result.AnomalySessions)
	if result.HealthSummary.Severity == "" {
		result.HealthSummary.Severity = modelSignalSeverityUnknown
	}
	result.HealthSummary.TopReasons = nonNilSlice(result.HealthSummary.TopReasons)
	result.Cohorts = nonNilSlice(result.Cohorts)
	for index := range result.Cohorts {
		normalizeModelSignalsDrift(&result.Cohorts[index].Drift)
	}
	result.Matrix = nonNilSlice(result.Matrix)
	for rowIndex := range result.Matrix {
		result.Matrix[rowIndex].Cells = nonNilSlice(result.Matrix[rowIndex].Cells)
		for cellIndex := range result.Matrix[rowIndex].Cells {
			normalizeModelSignalsDrift(&result.Matrix[rowIndex].Cells[cellIndex].Drift)
		}
	}
	result.ProjectHotspots = nonNilSlice(result.ProjectHotspots)
	for index := range result.ProjectHotspots {
		normalizeModelSignalsDrift(&result.ProjectHotspots[index].Drift)
	}
	result.DailyMetrics = nonNilSlice(result.DailyMetrics)
	for index := range result.DailyMetrics {
		normalizeModelSignalsDrift(&result.DailyMetrics[index].Drift)
	}
	result.ProjectMetrics = nonNilSlice(result.ProjectMetrics)
	for index := range result.ProjectMetrics {
		normalizeModelSignalsDrift(&result.ProjectMetrics[index].Drift)
	}
}

func normalizeModelSignalsDrift(drift *model.ModelSignalsDrift) {
	if drift.Severity == "" {
		drift.Severity = modelSignalSeverityUnknown
	}
	if drift.Confidence == "" {
		drift.Confidence = modelSignalConfidenceLow
	}
	drift.Reasons = nonNilSlice(drift.Reasons)
	drift.Metrics = nonNilSlice(drift.Metrics)
}
