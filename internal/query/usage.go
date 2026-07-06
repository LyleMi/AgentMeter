package query

import (
	"context"
	"strings"

	"github.com/LyleMi/AgentMeter/internal/model"
)

const usageSessionModelExpr = "COALESCE(NULLIF(tu.model, ''), s.model)"
const usageCostColumns = usageSessionModelExpr + `, tu.input_tokens, tu.cached_input_tokens, tu.output_tokens, tu.reasoning_output_tokens, tu.total_tokens, tu.source`

func analyticsSessionWhere(filters model.AnalyticsFilters) ([]string, []any) {
	where := []string{"1 = 1"}
	args := []any{}
	return appendAnalyticsFilters(where, args, filters, usageAnalyticsFilterSQLScope())
}

func analyticsUsageWhere(filters model.AnalyticsFilters) ([]string, []any) {
	where := []string{"tu.owner_kind = 'session'"}
	args := []any{}
	return appendAnalyticsFilters(where, args, filters, usageAnalyticsFilterSQLScope())
}

func usageAnalyticsFilterSQLScope() analyticsFilterSQLScope {
	return analyticsFilterSQLScope{
		sourceAlias: "src",
		modelExpr:   usageSessionModelExpr,
		startedExpr: "s.started_at",
	}
}

func appendUsageMetricFilter(where []string) []string {
	return append(where, `(tu.input_tokens > 0 OR tu.cached_input_tokens > 0 OR tu.output_tokens > 0 OR tu.reasoning_output_tokens > 0 OR tu.context_compression_tokens > 0 OR tu.total_tokens > 0)`)
}

func (s *Service) TokenAnalytics(ctx context.Context) (model.TokenAnalytics, error) {
	return s.TokenAnalyticsWithFilters(ctx, model.AnalyticsFilters{})
}

func (s *Service) TokenAnalyticsWithFilters(ctx context.Context, filters model.AnalyticsFilters) (model.TokenAnalytics, error) {
	var result model.TokenAnalytics
	totalSessions, err := s.analyticsSessionCount(ctx, filters)
	if err != nil {
		return result, err
	}
	result.TotalSessions = totalSessions

	usage, err := s.usageTotals(ctx, filters)
	if err != nil {
		return result, err
	}
	result.TotalInputTokens = usage.InputTokens
	result.TotalCachedInputTokens = usage.CachedInputTokens
	result.TotalOutputTokens = usage.OutputTokens
	result.TotalReasoningTokens = usage.ReasoningOutputTokens
	result.TotalContextCompressionTokens = usage.ContextCompressionTokens
	result.TotalTokens = usage.TotalTokens
	result.CacheUtilizationRate = cacheUtilizationRate(result.TotalInputTokens, result.TotalCachedInputTokens)
	result.EstimatedCostUSD, result.UnpricedCount, err = s.totalCostWithFilters(ctx, filters)
	if err != nil {
		return result, err
	}
	result.CacheHitTrend, err = s.cacheHitTrendWithFilters(ctx, filters)
	if err != nil {
		return result, err
	}
	result.ModelUsage, err = s.modelUsageWithFilters(ctx, filters)
	if err != nil {
		return result, err
	}
	result.AgentUsage, err = s.agentUsageWithFilters(ctx, filters)
	if err != nil {
		return result, err
	}
	result.RecentSessions, err = s.analyticsSessions(ctx, filters, 8, "s.started_at DESC, s.id DESC", false)
	if err != nil {
		return result, err
	}
	result.HighTokenSessions, err = s.highTokenSessionsWithFilters(ctx, filters)
	normalizeTokenAnalyticsSlices(&result)
	return result, err
}

func (s *Service) analyticsSessionCount(ctx context.Context, filters model.AnalyticsFilters) (int, error) {
	where, args := analyticsSessionWhere(filters)
	var count int
	err := s.conn.QueryRowContext(ctx, `SELECT COUNT(DISTINCT s.id)
		FROM sessions s
		JOIN sources src ON src.id = s.source_id
		LEFT JOIN token_usage tu ON tu.owner_kind = 'session' AND tu.owner_id = s.id
		WHERE `+strings.Join(where, " AND "), args...).Scan(&count)
	return count, err
}

func (s *Service) usageTotals(ctx context.Context, filters model.AnalyticsFilters) (model.Usage, error) {
	where, args := analyticsUsageWhere(filters)
	var usage model.Usage
	err := s.conn.QueryRowContext(ctx, `SELECT
		COALESCE(SUM(tu.input_tokens), 0),
		COALESCE(SUM(tu.cached_input_tokens), 0),
		COALESCE(SUM(tu.output_tokens), 0),
		COALESCE(SUM(tu.reasoning_output_tokens), 0),
		COALESCE(SUM(tu.context_compression_tokens), 0),
		COALESCE(SUM(tu.total_tokens), 0)
		FROM token_usage tu
		JOIN sessions s ON s.id = tu.owner_id
		JOIN sources src ON src.id = s.source_id
		WHERE `+strings.Join(where, " AND "), args...).
		Scan(&usage.InputTokens, &usage.CachedInputTokens, &usage.OutputTokens, &usage.ReasoningOutputTokens, &usage.ContextCompressionTokens, &usage.TotalTokens)
	return usage, err
}

func normalizeTokenAnalyticsSlices(result *model.TokenAnalytics) {
	result.ModelUsage = nonNilSlice(result.ModelUsage)
	result.AgentUsage = nonNilSlice(result.AgentUsage)
	result.RecentSessions = nonNilSlice(result.RecentSessions)
	result.HighTokenSessions = nonNilSlice(result.HighTokenSessions)
	result.CacheHitTrend = nonNilSlice(result.CacheHitTrend)
}

func (s *Service) highTokenSessions(ctx context.Context) ([]model.Session, error) {
	return s.highTokenSessionsWithFilters(ctx, model.AnalyticsFilters{})
}

func (s *Service) highTokenSessionsWithFilters(ctx context.Context, filters model.AnalyticsFilters) ([]model.Session, error) {
	return s.analyticsSessions(ctx, filters, 8, "COALESCE(tu.total_tokens, 0) DESC, s.started_at DESC, s.id DESC", true)
}

func (s *Service) analyticsSessions(ctx context.Context, filters model.AnalyticsFilters, limit int, orderBy string, requireTokens bool) ([]model.Session, error) {
	where, args := analyticsSessionWhere(filters)
	if requireTokens {
		where = append(where, "COALESCE(tu.total_tokens, 0) > 0")
	}
	args = append(args, limit)
	query := sessionSelect + `
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY ` + orderBy + `
		LIMIT ?`
	return s.scanSessions(ctx, query, args...)
}

func (s *Service) modelUsage(ctx context.Context) ([]model.ModelUsage, error) {
	return s.modelUsageWithFilters(ctx, model.AnalyticsFilters{})
}

func (s *Service) modelUsageWithFilters(ctx context.Context, filters model.AnalyticsFilters) ([]model.ModelUsage, error) {
	calculator := s.pricingCalculator(ctx)
	costs, unpriced, err := s.modelCostsWithFilters(ctx, calculator, filters)
	if err != nil {
		return nil, err
	}
	where, args := analyticsUsageWhere(filters)
	where = appendUsageMetricFilter(where)
	rows, err := s.conn.QueryContext(ctx, `SELECT `+usageSessionModelExpr+`, COUNT(*),
		COALESCE(SUM(tu.total_tokens), 0), COALESCE(SUM(tu.input_tokens), 0), COALESCE(SUM(tu.cached_input_tokens), 0),
		COALESCE(SUM(tu.output_tokens), 0), COALESCE(SUM(tu.reasoning_output_tokens), 0), COALESCE(SUM(tu.context_compression_tokens), 0)
		FROM token_usage tu
		JOIN sessions s ON s.id = tu.owner_id
		JOIN sources src ON src.id = s.source_id
		WHERE `+strings.Join(where, " AND ")+`
		GROUP BY `+usageSessionModelExpr+`
		ORDER BY SUM(tu.total_tokens) DESC, `+usageSessionModelExpr+` ASC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []model.ModelUsage
	for rows.Next() {
		var item model.ModelUsage
		if err := rows.Scan(
			&item.Model,
			&item.SessionCount,
			&item.TotalTokens,
			&item.InputTokens,
			&item.CachedInputTokens,
			&item.OutputTokens,
			&item.ReasoningOutputTokens,
			&item.ContextCompressionTokens,
		); err != nil {
			return nil, err
		}
		if cost, ok := costs[item.Model]; ok {
			item.EstimatedCostUSD = &cost
		}
		item.Unpriced = unpriced[item.Model]
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Service) agentUsage(ctx context.Context) ([]model.AgentUsage, error) {
	return s.agentUsageWithFilters(ctx, model.AnalyticsFilters{})
}

func (s *Service) agentUsageWithFilters(ctx context.Context, filters model.AnalyticsFilters) ([]model.AgentUsage, error) {
	where, args := analyticsSessionWhere(filters)
	rows, err := s.conn.QueryContext(ctx, `SELECT src.id, src.root_path, src.sessions_path, src.kind, src.name, COUNT(*),
		COALESCE(SUM(tu.total_tokens), 0), COALESCE(SUM(tu.input_tokens), 0), COALESCE(SUM(tu.cached_input_tokens), 0),
		COALESCE(SUM(tu.output_tokens), 0), COALESCE(SUM(tu.reasoning_output_tokens), 0), COALESCE(SUM(tu.context_compression_tokens), 0),
		COALESCE(SUM((SELECT COUNT(*) FROM tool_calls tc WHERE tc.session_id = s.id)), 0)
		FROM sessions s
		JOIN sources src ON src.id = s.source_id
		LEFT JOIN token_usage tu ON tu.owner_kind = 'session' AND tu.owner_id = s.id
		WHERE `+strings.Join(where, " AND ")+`
		GROUP BY src.id
		ORDER BY COUNT(*) DESC, src.name ASC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []model.AgentUsage
	for rows.Next() {
		var item model.AgentUsage
		if err := rows.Scan(
			&item.SourceID,
			&item.SourceRootPath,
			&item.SourceSessionsPath,
			&item.AgentKind,
			&item.AgentName,
			&item.SessionCount,
			&item.TotalTokens,
			&item.InputTokens,
			&item.CachedInputTokens,
			&item.OutputTokens,
			&item.ReasoningOutputTokens,
			&item.ContextCompressionTokens,
			&item.ToolCalls,
		); err != nil {
			return nil, err
		}
		item.SourceKey = sourceInstanceKey(item.SourceID)
		item.SourceLabel = item.AgentName
		item.CacheUtilizationRate = cacheUtilizationRate(item.InputTokens, item.CachedInputTokens)
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	calculator := s.pricingCalculator(ctx)
	costs, unpriced, err := s.agentCostsWithFilters(ctx, calculator, filters)
	if err != nil {
		return nil, err
	}
	for index := range result {
		key := result[index].SourceID
		if cost, ok := costs[key]; ok {
			result[index].EstimatedCostUSD = &cost
		}
		result[index].Unpriced = unpriced[key]
	}
	return result, nil
}
