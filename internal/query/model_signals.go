package query

import (
	"context"
	"math"
	"sort"
	"strings"
	"time"

	"AgentMeter/internal/db"
	"AgentMeter/internal/model"
)

type modelSignalSessionMetric struct {
	SessionID          int64
	SourceID           int64
	SourceRootPath     string
	SourceSessionsPath string
	AgentKind          string
	AgentName          string
	SessionKey         string
	CodexSessionID     string
	ProjectPath        string
	Model              string
	StartedAt          string
	Day                string
	RawSourcePath      string

	InputTokens           int64
	CachedInputTokens     int64
	OutputTokens          int64
	ReasoningOutputTokens int64
	TotalTokens           int64
	ModelDurationMS       int64
	ModelCalls            int
	ToolCalls             int
	FailedToolCalls       int
}

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

func (s *Service) modelSignalSessionMetrics(ctx context.Context, filters model.AnalyticsFilters) ([]modelSignalSessionMetric, error) {
	where, args := analyticsSessionWhere(filters)
	rows, err := s.conn.QueryContext(ctx, `SELECT
		s.id,
		s.source_id,
		src.root_path,
		src.sessions_path,
		src.kind,
		src.name,
		COALESCE(NULLIF(s.session_key, ''), s.codex_session_id),
		s.codex_session_id,
		s.project_path,
		`+usageSessionModelExpr+`,
		s.started_at,
		substr(s.started_at, 1, 10),
		sf.path,
		COALESCE(tu.input_tokens, 0),
		COALESCE(tu.cached_input_tokens, 0),
		COALESCE(tu.output_tokens, 0),
		COALESCE(tu.reasoning_output_tokens, 0),
		COALESCE(tu.total_tokens, 0),
		CASE
			WHEN COALESCE((SELECT COUNT(*) FROM model_calls mc WHERE mc.session_id = s.id), 0) > 0
				THEN COALESCE((SELECT SUM(CASE WHEN mc.duration_ms > 0 THEN mc.duration_ms ELSE 0 END) FROM model_calls mc WHERE mc.session_id = s.id), 0)
			WHEN s.model_duration_ms > 0 THEN s.model_duration_ms
			ELSE 0
		END,
		COALESCE((SELECT COUNT(*) FROM model_calls mc WHERE mc.session_id = s.id), 0),
		COALESCE((SELECT COUNT(*) FROM tool_calls tc WHERE tc.session_id = s.id), 0),
		COALESCE((SELECT COUNT(*) FROM tool_calls tc WHERE tc.session_id = s.id AND tc.status NOT IN ('completed', 'success')), 0)
		FROM sessions s
		JOIN sources src ON src.id = s.source_id
		JOIN source_files sf ON sf.id = s.source_file_id
		LEFT JOIN token_usage tu ON tu.owner_kind = 'session' AND tu.owner_id = s.id
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY s.started_at ASC, s.id ASC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []modelSignalSessionMetric
	for rows.Next() {
		var item modelSignalSessionMetric
		if err := rows.Scan(
			&item.SessionID,
			&item.SourceID,
			&item.SourceRootPath,
			&item.SourceSessionsPath,
			&item.AgentKind,
			&item.AgentName,
			&item.SessionKey,
			&item.CodexSessionID,
			&item.ProjectPath,
			&item.Model,
			&item.StartedAt,
			&item.Day,
			&item.RawSourcePath,
			&item.InputTokens,
			&item.CachedInputTokens,
			&item.OutputTokens,
			&item.ReasoningOutputTokens,
			&item.TotalTokens,
			&item.ModelDurationMS,
			&item.ModelCalls,
			&item.ToolCalls,
			&item.FailedToolCalls,
		); err != nil {
			return nil, err
		}
		if item.AgentName == "" {
			item.AgentName = item.AgentKind
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func buildModelSignals(metrics []modelSignalSessionMetric) model.ModelSignals {
	var result model.ModelSignals
	breakdowns := map[string]*model.ModelSignalsBreakdown{}
	breakdownSessionsWithTools := map[string]int{}
	trendByDay := map[string]*model.ModelSignalsTrendPoint{}
	trendSessionsWithTools := map[string]int{}
	sessionsWithTools := 0

	for _, metric := range metrics {
		result.TotalSessions++
		result.TotalModelCalls += metric.ModelCalls
		result.TotalToolCalls += metric.ToolCalls
		result.FailedToolCalls += metric.FailedToolCalls
		if metric.ToolCalls > 0 {
			sessionsWithTools++
		}

		breakdown := breakdowns[metric.Model]
		if breakdown == nil {
			breakdown = &model.ModelSignalsBreakdown{Model: metric.Model}
			breakdowns[metric.Model] = breakdown
		}
		breakdown.SessionCount++
		breakdown.ModelCalls += metric.ModelCalls
		breakdown.ToolCalls += metric.ToolCalls
		breakdown.FailedToolCalls += metric.FailedToolCalls
		if metric.ToolCalls > 0 {
			breakdownSessionsWithTools[metric.Model]++
		}
		addTokenAndDurationTotals(
			&breakdown.TotalTokens,
			&breakdown.InputTokens,
			&breakdown.CachedInputTokens,
			&breakdown.OutputTokens,
			&breakdown.ReasoningOutputTokens,
			&breakdown.ModelDurationMS,
			metric,
		)

		if metric.Day != "" {
			point := trendByDay[metric.Day]
			if point == nil {
				point = &model.ModelSignalsTrendPoint{Date: metric.Day}
				trendByDay[metric.Day] = point
			}
			point.SessionCount++
			point.ModelCalls += metric.ModelCalls
			point.ToolCalls += metric.ToolCalls
			point.FailedToolCalls += metric.FailedToolCalls
			point.TotalTokens += metric.TotalTokens
			point.InputTokens += metric.InputTokens
			point.CachedInputTokens += metric.CachedInputTokens
			point.OutputTokens += metric.OutputTokens
			point.ReasoningOutputTokens += metric.ReasoningOutputTokens
			point.ModelDurationMS += metric.ModelDurationMS
			if metric.ToolCalls > 0 {
				trendSessionsWithTools[metric.Day]++
			}
		}
	}

	var totalTokens, inputTokens, cachedInputTokens, outputTokens, reasoningOutputTokens, modelDurationMS int64
	for _, metric := range metrics {
		totalTokens += metric.TotalTokens
		inputTokens += metric.InputTokens
		cachedInputTokens += metric.CachedInputTokens
		outputTokens += metric.OutputTokens
		reasoningOutputTokens += metric.ReasoningOutputTokens
		modelDurationMS += metric.ModelDurationMS
	}
	result.ToolFailureRate = safeRateInt(result.FailedToolCalls, result.TotalToolCalls)
	result.ToolDependencyRate = safeRateInt(sessionsWithTools, result.TotalSessions)
	result.AvgModelCallsPerSession = safeRateInt(result.TotalModelCalls, result.TotalSessions)
	result.OutputExpansionRate = safeRate(outputTokens, inputTokens)
	result.ReasoningTokenShare = clamp01(safeRate(reasoningOutputTokens, outputTokens))
	result.CacheMissRate = cacheMissRate(inputTokens, cachedInputTokens)
	result.ModelThroughputTokensPerSecond = throughputPerSecond(totalTokens, modelDurationMS)
	result.ModelThroughputOutputTokensPerSecond = throughputPerSecond(outputTokens, modelDurationMS)

	for _, breakdown := range breakdowns {
		applyModelSignalsBreakdownRates(breakdown, breakdownSessionsWithTools[breakdown.Model])
		result.ModelBreakdown = append(result.ModelBreakdown, *breakdown)
	}
	sort.Slice(result.ModelBreakdown, func(i, j int) bool {
		left := result.ModelBreakdown[i]
		right := result.ModelBreakdown[j]
		if left.TotalTokens != right.TotalTokens {
			return left.TotalTokens > right.TotalTokens
		}
		return left.Model < right.Model
	})

	for _, point := range trendByDay {
		point.ToolDependencyRate = safeRateInt(trendSessionsWithTools[point.Date], point.SessionCount)
		applyModelSignalsTrendRates(point)
		result.Trend = append(result.Trend, *point)
	}
	sort.Slice(result.Trend, func(i, j int) bool { return result.Trend[i].Date < result.Trend[j].Date })
	if len(result.Trend) > 30 {
		result.Trend = result.Trend[len(result.Trend)-30:]
	}
	result.Trend = fillModelSignalsTrendGaps(result.Trend)
	applyModelSignalsRollingRates(result.Trend)

	result.AnomalySessions = rankModelSignalAnomalies(metrics, 8)
	return result
}

func addTokenAndDurationTotals(totalTokens, inputTokens, cachedInputTokens, outputTokens, reasoningOutputTokens, modelDurationMS *int64, metric modelSignalSessionMetric) {
	*totalTokens += metric.TotalTokens
	*inputTokens += metric.InputTokens
	*cachedInputTokens += metric.CachedInputTokens
	*outputTokens += metric.OutputTokens
	*reasoningOutputTokens += metric.ReasoningOutputTokens
	*modelDurationMS += metric.ModelDurationMS
}

func applyModelSignalsBreakdownRates(item *model.ModelSignalsBreakdown, sessionsWithTools int) {
	item.ToolFailureRate = safeRateInt(item.FailedToolCalls, item.ToolCalls)
	item.ToolDependencyRate = safeRateInt(sessionsWithTools, item.SessionCount)
	item.AvgModelCallsPerSession = safeRateInt(item.ModelCalls, item.SessionCount)
	item.OutputExpansionRate = safeRate(item.OutputTokens, item.InputTokens)
	item.ReasoningTokenShare = clamp01(safeRate(item.ReasoningOutputTokens, item.OutputTokens))
	item.CacheMissRate = cacheMissRate(item.InputTokens, item.CachedInputTokens)
	item.ModelThroughputTokensPerSecond = throughputPerSecond(item.TotalTokens, item.ModelDurationMS)
	item.ModelThroughputOutputTokensPerSecond = throughputPerSecond(item.OutputTokens, item.ModelDurationMS)
}

func applyModelSignalsTrendRates(point *model.ModelSignalsTrendPoint) {
	point.ToolFailureRate = safeRateInt(point.FailedToolCalls, point.ToolCalls)
	point.OutputExpansionRate = safeRate(point.OutputTokens, point.InputTokens)
	point.ReasoningTokenShare = clamp01(safeRate(point.ReasoningOutputTokens, point.OutputTokens))
	point.CacheMissRate = cacheMissRate(point.InputTokens, point.CachedInputTokens)
	point.ModelThroughputTokensPerSecond = throughputPerSecond(point.TotalTokens, point.ModelDurationMS)
	point.ModelThroughputOutputTokensPerSecond = throughputPerSecond(point.OutputTokens, point.ModelDurationMS)
	point.LowSample = point.SessionCount > 0 && (point.SessionCount < 3 || point.ModelCalls < 3 || point.ModelDurationMS <= 0)
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
		ReasoningTokenShare:                  clamp01(safeRate(metric.ReasoningOutputTokens, metric.OutputTokens)),
		CacheMissRate:                        cacheMissRate(metric.InputTokens, metric.CachedInputTokens),
		ModelThroughputTokensPerSecond:       throughputPerSecond(metric.TotalTokens, metric.ModelDurationMS),
		ModelThroughputOutputTokensPerSecond: throughputPerSecond(metric.OutputTokens, metric.ModelDurationMS),
		ToolFailureRate:                      safeRateInt(metric.FailedToolCalls, metric.ToolCalls),
		ReasonLabels:                         []string{},
	}
	if item.ReasoningTokenShare >= 0.5 {
		item.ReasonLabels = append(item.ReasonLabels, "high reasoning share")
		item.Score += item.ReasoningTokenShare * 2
	}
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
}

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

func cacheMissRate(inputTokens, cachedInputTokens int64) float64 {
	if inputTokens <= 0 {
		return 0
	}
	return clamp01(float64(inputTokens-cachedInputTokens) / float64(inputTokens))
}

func throughputPerSecond(tokens, durationMS int64) float64 {
	if tokens <= 0 || durationMS <= 0 {
		return 0
	}
	return float64(tokens) / (float64(durationMS) / 1000)
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
