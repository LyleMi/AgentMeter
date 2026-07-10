package query

import (
	"context"
	"sort"

	"github.com/LyleMi/AgentMeter/internal/model"
)

func (s *Service) dailyUsage(ctx context.Context) ([]model.DailyUsage, error) {
	return s.dailyUsageWithFilters(ctx, model.AnalyticsFilters{})
}

func (s *Service) dailyUsageWithFilters(ctx context.Context, filters model.AnalyticsFilters) ([]model.DailyUsage, error) {
	where, args := analyticsSessionWhere(filters)
	rows, err := s.conn.QueryContext(ctx, `SELECT * FROM (
		SELECT substr(s.started_at, 1, 10) AS day,
			COUNT(*) AS session_count,
			COALESCE(SUM(tu.total_tokens), 0) AS total_tokens,
			COALESCE(SUM(tu.input_tokens), 0) AS input_tokens,
			COALESCE(SUM(tu.cached_input_tokens), 0) AS cached_input_tokens,
			COALESCE(SUM(tu.output_tokens), 0) AS output_tokens,
			COALESCE(SUM(tu.context_compression_tokens), 0) AS context_compression_tokens,
			COALESCE(SUM((SELECT COUNT(*) FROM tool_calls tc WHERE tc.session_id = s.id)), 0) AS tool_calls
		FROM sessions s
		JOIN sources src ON src.id = s.source_id
		LEFT JOIN token_usage tu ON tu.owner_kind = 'session' AND tu.owner_id = s.id
		WHERE `+whereClause(where)+`
		GROUP BY day
		ORDER BY day DESC
		LIMIT 30
	) ORDER BY day ASC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []model.DailyUsage
	for rows.Next() {
		var item model.DailyUsage
		if err := rows.Scan(&item.Date, &item.SessionCount, &item.TotalTokens, &item.InputTokens, &item.CachedInputTokens, &item.OutputTokens, &item.ContextCompressionTokens, &item.ToolCalls); err != nil {
			return nil, err
		}
		item.CacheUtilizationRate = cacheUtilizationRate(item.InputTokens, item.CachedInputTokens)
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	calculator := s.pricingCalculator(ctx)
	costs, err := s.dailyCostsWithFilters(ctx, calculator, filters)
	if err != nil {
		return nil, err
	}
	for index := range result {
		if cost, ok := costs[result[index].Date]; ok {
			result[index].EstimatedCostUSD = &cost
		}
	}
	return result, nil
}

func (s *Service) cacheHitTrendWithFilters(ctx context.Context, filters model.AnalyticsFilters) ([]model.CacheHitTrendPoint, error) {
	daily, err := s.dailyUsageWithFilters(ctx, filters)
	if err != nil {
		return nil, err
	}
	return cacheHitTrendFromDailyUsage(daily), nil
}

func cacheHitTrendFromDailyUsage(daily []model.DailyUsage) []model.CacheHitTrendPoint {
	daily = fillDailyUsageGaps(daily)
	if len(daily) == 0 {
		return []model.CacheHitTrendPoint{}
	}
	lowVolumeThreshold := cacheTrendLowInputThreshold(daily)
	result := make([]model.CacheHitTrendPoint, 0, len(daily))
	for index, item := range daily {
		rollingInput, rollingCached := rollingCacheTrendWindow(daily, index)
		point := model.CacheHitTrendPoint{
			Date:              item.Date,
			SessionCount:      item.SessionCount,
			TotalTokens:       item.TotalTokens,
			InputTokens:       item.InputTokens,
			CachedInputTokens: item.CachedInputTokens,
			HasUsage:          item.SessionCount > 0 || item.TotalTokens > 0 || item.InputTokens > 0 || item.CachedInputTokens > 0 || item.ContextCompressionTokens > 0,
			LowInputVolume:    item.InputTokens > 0 && lowVolumeThreshold > 0 && item.InputTokens < lowVolumeThreshold,
		}
		point.CacheUtilizationRate = cacheUtilizationRate(item.InputTokens, item.CachedInputTokens)
		point.RollingCacheUtilizationRate = cacheUtilizationRate(rollingInput, rollingCached)
		result = append(result, point)
	}
	return result
}

func rollingCacheTrendWindow(daily []model.DailyUsage, index int) (int64, int64) {
	start := index - 6
	if start < 0 {
		start = 0
	}
	var inputTokens int64
	var cachedInputTokens int64
	for cursor := start; cursor <= index; cursor++ {
		inputTokens += daily[cursor].InputTokens
		cachedInputTokens += daily[cursor].CachedInputTokens
	}
	return inputTokens, cachedInputTokens
}

func cacheTrendLowInputThreshold(daily []model.DailyUsage) int64 {
	var values []int64
	for _, item := range daily {
		if item.InputTokens > 0 {
			values = append(values, item.InputTokens)
		}
	}
	if len(values) == 0 {
		return 0
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	median := values[len(values)/2]
	if len(values)%2 == 0 {
		median = (values[len(values)/2-1] + values[len(values)/2]) / 2
	}
	threshold := median / 4
	if threshold < 1_000 {
		return 1_000
	}
	return threshold
}

func fillDailyUsageGaps(daily []model.DailyUsage) []model.DailyUsage {
	return fillAnalyticsDateGaps(
		daily,
		func(item model.DailyUsage) string { return item.Date },
		func(date string) model.DailyUsage { return model.DailyUsage{Date: date} },
	)
}
