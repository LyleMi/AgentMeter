package query

import (
	"context"
	"sort"
	"time"

	"AgentMeter/internal/model"
)

func (s *Service) ModelSignals(ctx context.Context) (model.ModelSignals, error) {
	return s.ModelSignalsWithFilters(ctx, model.AnalyticsFilters{})
}

func (s *Service) ModelSignalsWithFilters(ctx context.Context, filters model.AnalyticsFilters) (model.ModelSignals, error) {
	metrics, err := s.modelSignalSessionMetrics(ctx, filters)
	if err != nil {
		return model.ModelSignals{}, err
	}
	result := buildModelSignals(metrics)
	normalizeModelSignalsSlices(&result)
	return result, nil
}

func buildModelSignals(metrics []modelSignalSessionMetric) model.ModelSignals {
	var result model.ModelSignals
	var totals modelSignalMetricAccumulator
	breakdowns := map[string]*modelSignalMetricAccumulator{}
	trendByDay := map[string]*modelSignalMetricAccumulator{}

	for _, metric := range metrics {
		totals.add(metric)

		breakdown := breakdowns[metric.Model]
		if breakdown == nil {
			breakdown = &modelSignalMetricAccumulator{}
			breakdowns[metric.Model] = breakdown
		}
		breakdown.add(metric)

		if metric.Day != "" {
			point := trendByDay[metric.Day]
			if point == nil {
				point = &modelSignalMetricAccumulator{}
				trendByDay[metric.Day] = point
			}
			point.add(metric)
		}
	}

	applyModelSignalsTotals(&result, totals.metricSet())

	for modelName, breakdown := range breakdowns {
		result.ModelBreakdown = append(result.ModelBreakdown, modelSignalsBreakdownFromMetricSet(modelName, breakdown.metricSet()))
	}
	sort.Slice(result.ModelBreakdown, func(i, j int) bool {
		left := result.ModelBreakdown[i]
		right := result.ModelBreakdown[j]
		if left.TotalTokens != right.TotalTokens {
			return left.TotalTokens > right.TotalTokens
		}
		return left.Model < right.Model
	})

	for day, point := range trendByDay {
		result.Trend = append(result.Trend, modelSignalsTrendPointFromMetricSet(day, point.metricSet()))
	}
	sort.Slice(result.Trend, func(i, j int) bool { return result.Trend[i].Date < result.Trend[j].Date })
	if len(result.Trend) > 30 {
		result.Trend = result.Trend[len(result.Trend)-30:]
	}
	result.Trend = fillModelSignalsTrendGaps(result.Trend)
	applyModelSignalsRollingRates(result.Trend)

	result.AnomalySessions = rankModelSignalAnomalies(metrics, 8)
	result.DailyMetrics = buildModelSignalDailyMetrics(metrics)
	result.HealthSummary, result.Cohorts, result.Matrix, result.ProjectHotspots, result.ProjectMetrics = buildModelSignalHealthReadModels(metrics)
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
	if len(points) <= 1 {
		return points
	}
	start, err := time.Parse(analyticsDateOnlyLayout, points[0].Date)
	if err != nil {
		return points
	}
	end, err := time.Parse(analyticsDateOnlyLayout, points[len(points)-1].Date)
	if err != nil || end.Before(start) {
		return points
	}
	spanDays := int(end.Sub(start).Hours()/24) + 1
	if spanDays <= len(points) || spanDays > 62 {
		return points
	}
	byDate := make(map[string]model.ModelSignalsTrendPoint, len(points))
	for _, point := range points {
		byDate[point.Date] = point
	}
	filled := make([]model.ModelSignalsTrendPoint, 0, spanDays)
	for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
		date := day.Format(analyticsDateOnlyLayout)
		if point, ok := byDate[date]; ok {
			filled = append(filled, point)
		} else {
			filled = append(filled, model.ModelSignalsTrendPoint{Date: date})
		}
	}
	return filled
}

func normalizeModelSignalsSlices(result *model.ModelSignals) {
	if result.Trend == nil {
		result.Trend = []model.ModelSignalsTrendPoint{}
	}
	if result.ModelBreakdown == nil {
		result.ModelBreakdown = []model.ModelSignalsBreakdown{}
	}
	if result.AnomalySessions == nil {
		result.AnomalySessions = []model.ModelSignalsAnomalySession{}
	}
	if result.HealthSummary.Severity == "" {
		result.HealthSummary.Severity = modelSignalSeverityUnknown
	}
	if result.HealthSummary.TopReasons == nil {
		result.HealthSummary.TopReasons = []string{}
	}
	if result.Cohorts == nil {
		result.Cohorts = []model.ModelSignalsCohort{}
	}
	for index := range result.Cohorts {
		normalizeModelSignalsDrift(&result.Cohorts[index].Drift)
	}
	if result.Matrix == nil {
		result.Matrix = []model.ModelSignalsMatrixRow{}
	}
	for rowIndex := range result.Matrix {
		if result.Matrix[rowIndex].Cells == nil {
			result.Matrix[rowIndex].Cells = []model.ModelSignalsMatrixCell{}
		}
		for cellIndex := range result.Matrix[rowIndex].Cells {
			normalizeModelSignalsDrift(&result.Matrix[rowIndex].Cells[cellIndex].Drift)
		}
	}
	if result.ProjectHotspots == nil {
		result.ProjectHotspots = []model.ModelSignalsProjectHotspot{}
	}
	for index := range result.ProjectHotspots {
		normalizeModelSignalsDrift(&result.ProjectHotspots[index].Drift)
	}
	if result.DailyMetrics == nil {
		result.DailyMetrics = []model.ModelSignalsDailyMetric{}
	}
	for index := range result.DailyMetrics {
		normalizeModelSignalsDrift(&result.DailyMetrics[index].Drift)
	}
	if result.ProjectMetrics == nil {
		result.ProjectMetrics = []model.ModelSignalsProjectMetric{}
	}
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
	if drift.Reasons == nil {
		drift.Reasons = []string{}
	}
	if drift.Metrics == nil {
		drift.Metrics = []model.ModelSignalsDriftMetric{}
	}
}
