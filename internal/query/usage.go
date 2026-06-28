package query

import (
	"context"

	"AgentMeter/internal/model"
	"AgentMeter/internal/pricing"
)

func (s *Service) pricingCalculator(ctx context.Context) pricing.Calculator {
	calculator, err := pricing.LoadCalculator(ctx, s.conn)
	if err != nil {
		return pricing.Calculator{}
	}
	return calculator
}

func (s *Service) totalCost(ctx context.Context) (*float64, int, error) {
	calculator := s.pricingCalculator(ctx)
	rows, err := s.conn.QueryContext(ctx, `SELECT model, input_tokens, cached_input_tokens, output_tokens, reasoning_output_tokens, total_tokens, source
		FROM token_usage WHERE owner_kind = 'session'`)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var total float64
	var hasCost bool
	var unpriced int
	for rows.Next() {
		var usage model.Usage
		if err := rows.Scan(&usage.Model, &usage.InputTokens, &usage.CachedInputTokens, &usage.OutputTokens, &usage.ReasoningOutputTokens, &usage.TotalTokens, &usage.Source); err != nil {
			return nil, 0, err
		}
		cost, isUnpriced := calculator.Compute(usage)
		if isUnpriced {
			unpriced++
			continue
		}
		if cost != nil {
			total += *cost
			hasCost = true
		}
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if !hasCost {
		return nil, unpriced, nil
	}
	return &total, unpriced, nil
}

func (s *Service) TokenAnalytics(ctx context.Context) (model.TokenAnalytics, error) {
	var result model.TokenAnalytics
	if err := s.conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM sessions`).Scan(&result.TotalSessions); err != nil {
		return result, err
	}
	err := s.conn.QueryRowContext(ctx, `SELECT
		COALESCE(SUM(input_tokens), 0),
		COALESCE(SUM(cached_input_tokens), 0),
		COALESCE(SUM(output_tokens), 0),
		COALESCE(SUM(reasoning_output_tokens), 0),
		COALESCE(SUM(total_tokens), 0)
		FROM token_usage WHERE owner_kind = 'session'`).
		Scan(&result.TotalInputTokens, &result.TotalCachedInputTokens, &result.TotalOutputTokens, &result.TotalReasoningTokens, &result.TotalTokens)
	if err != nil {
		return result, err
	}
	if result.TotalInputTokens > 0 {
		result.CacheUtilizationRate = float64(result.TotalCachedInputTokens) / float64(result.TotalInputTokens)
	}
	result.EstimatedCostUSD, result.UnpricedCount, err = s.totalCost(ctx)
	if err != nil {
		return result, err
	}
	result.ModelUsage, err = s.modelUsage(ctx)
	if err != nil {
		return result, err
	}
	result.AgentUsage, err = s.agentUsage(ctx)
	if err != nil {
		return result, err
	}
	result.RecentSessions, err = s.Sessions(ctx, model.SessionFilters{Limit: 8})
	if err != nil {
		return result, err
	}
	result.HighTokenSessions, err = s.highTokenSessions(ctx)
	normalizeTokenAnalyticsSlices(&result)
	return result, err
}

func normalizeTokenAnalyticsSlices(result *model.TokenAnalytics) {
	if result.ModelUsage == nil {
		result.ModelUsage = []model.ModelUsage{}
	}
	if result.AgentUsage == nil {
		result.AgentUsage = []model.AgentUsage{}
	}
	if result.RecentSessions == nil {
		result.RecentSessions = []model.Session{}
	}
	if result.HighTokenSessions == nil {
		result.HighTokenSessions = []model.Session{}
	}
}

func (s *Service) highTokenSessions(ctx context.Context) ([]model.Session, error) {
	return s.scanSessions(ctx, sessionSelect+` WHERE COALESCE(tu.total_tokens, 0) > 0
		ORDER BY COALESCE(tu.total_tokens, 0) DESC, s.started_at DESC, s.id DESC
		LIMIT 8`)
}

func (s *Service) dailyUsage(ctx context.Context) ([]model.DailyUsage, error) {
	rows, err := s.conn.QueryContext(ctx, `SELECT * FROM (
		SELECT substr(s.started_at, 1, 10) AS day,
			COUNT(*) AS session_count,
			COALESCE(SUM(tu.total_tokens), 0) AS total_tokens,
			COALESCE(SUM(tu.input_tokens), 0) AS input_tokens,
			COALESCE(SUM(tu.output_tokens), 0) AS output_tokens,
			COALESCE(SUM((SELECT COUNT(*) FROM tool_calls tc WHERE tc.session_id = s.id)), 0) AS tool_calls
		FROM sessions s
		LEFT JOIN token_usage tu ON tu.owner_kind = 'session' AND tu.owner_id = s.id
		GROUP BY day
		ORDER BY day DESC
		LIMIT 30
	) ORDER BY day ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []model.DailyUsage
	for rows.Next() {
		var item model.DailyUsage
		if err := rows.Scan(&item.Date, &item.SessionCount, &item.TotalTokens, &item.InputTokens, &item.OutputTokens, &item.ToolCalls); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	calculator := s.pricingCalculator(ctx)
	costs, err := s.dailyCosts(ctx, calculator)
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

func (s *Service) dailyCosts(ctx context.Context, calculator pricing.Calculator) (map[string]float64, error) {
	rows, err := s.conn.QueryContext(ctx, `SELECT substr(s.started_at, 1, 10) AS day, tu.model,
		COALESCE(SUM(tu.input_tokens), 0), COALESCE(SUM(tu.cached_input_tokens), 0), COALESCE(SUM(tu.output_tokens), 0),
		COALESCE(SUM(tu.reasoning_output_tokens), 0), COALESCE(SUM(tu.total_tokens), 0), COALESCE(MAX(tu.source), 'unknown')
		FROM sessions s
		JOIN token_usage tu ON tu.owner_kind = 'session' AND tu.owner_id = s.id
		GROUP BY day, tu.model`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]float64{}
	for rows.Next() {
		var day string
		var usage model.Usage
		if err := rows.Scan(&day, &usage.Model, &usage.InputTokens, &usage.CachedInputTokens, &usage.OutputTokens, &usage.ReasoningOutputTokens, &usage.TotalTokens, &usage.Source); err != nil {
			return nil, err
		}
		if cost, unpriced := calculator.Compute(usage); !unpriced && cost != nil {
			result[day] += *cost
		}
	}
	return result, rows.Err()
}

func (s *Service) modelUsage(ctx context.Context) ([]model.ModelUsage, error) {
	calculator := s.pricingCalculator(ctx)
	rows, err := s.conn.QueryContext(ctx, `SELECT tu.model, COUNT(*),
		COALESCE(SUM(tu.total_tokens), 0), COALESCE(SUM(tu.input_tokens), 0), COALESCE(SUM(tu.cached_input_tokens), 0),
		COALESCE(SUM(tu.output_tokens), 0), COALESCE(SUM(tu.reasoning_output_tokens), 0), COALESCE(MAX(tu.source), 'unknown')
		FROM token_usage tu
		WHERE tu.owner_kind = 'session' AND (
			tu.input_tokens > 0 OR tu.cached_input_tokens > 0 OR tu.output_tokens > 0 OR tu.reasoning_output_tokens > 0 OR tu.total_tokens > 0
		)
		GROUP BY tu.model
		ORDER BY SUM(tu.total_tokens) DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []model.ModelUsage
	for rows.Next() {
		var item model.ModelUsage
		var source string
		if err := rows.Scan(
			&item.Model,
			&item.SessionCount,
			&item.TotalTokens,
			&item.InputTokens,
			&item.CachedInputTokens,
			&item.OutputTokens,
			&item.ReasoningOutputTokens,
			&source,
		); err != nil {
			return nil, err
		}
		usage := model.Usage{
			Model:                 item.Model,
			InputTokens:           item.InputTokens,
			CachedInputTokens:     item.CachedInputTokens,
			OutputTokens:          item.OutputTokens,
			ReasoningOutputTokens: item.ReasoningOutputTokens,
			TotalTokens:           item.TotalTokens,
			Source:                source,
		}
		item.EstimatedCostUSD, item.Unpriced = calculator.Compute(usage)
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Service) agentUsage(ctx context.Context) ([]model.AgentUsage, error) {
	rows, err := s.conn.QueryContext(ctx, `SELECT src.id, src.root_path, src.sessions_path, src.kind, src.name, COUNT(*),
		COALESCE(SUM(tu.total_tokens), 0), COALESCE(SUM(tu.input_tokens), 0), COALESCE(SUM(tu.cached_input_tokens), 0),
		COALESCE(SUM(tu.output_tokens), 0), COALESCE(SUM(tu.reasoning_output_tokens), 0),
		COALESCE(SUM((SELECT COUNT(*) FROM tool_calls tc WHERE tc.session_id = s.id)), 0)
		FROM sessions s
		JOIN sources src ON src.id = s.source_id
		LEFT JOIN token_usage tu ON tu.owner_kind = 'session' AND tu.owner_id = s.id
		GROUP BY src.id
		ORDER BY COUNT(*) DESC, src.name ASC`)
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
			&item.ToolCalls,
		); err != nil {
			return nil, err
		}
		item.SourceKey = sourceInstanceKey(item.SourceID)
		item.SourceLabel = item.AgentName
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	calculator := s.pricingCalculator(ctx)
	costs, unpriced, err := s.agentCosts(ctx, calculator)
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

func (s *Service) agentCosts(ctx context.Context, calculator pricing.Calculator) (map[int64]float64, map[int64]bool, error) {
	rows, err := s.conn.QueryContext(ctx, `SELECT src.id, tu.model,
		COALESCE(SUM(tu.input_tokens), 0), COALESCE(SUM(tu.cached_input_tokens), 0), COALESCE(SUM(tu.output_tokens), 0),
		COALESCE(SUM(tu.reasoning_output_tokens), 0), COALESCE(SUM(tu.total_tokens), 0), COALESCE(MAX(tu.source), 'unknown')
		FROM sessions s
		JOIN sources src ON src.id = s.source_id
		JOIN token_usage tu ON tu.owner_kind = 'session' AND tu.owner_id = s.id
		GROUP BY src.id, tu.model`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	costs := map[int64]float64{}
	unpriced := map[int64]bool{}
	for rows.Next() {
		var sourceID int64
		var usage model.Usage
		if err := rows.Scan(&sourceID, &usage.Model, &usage.InputTokens, &usage.CachedInputTokens, &usage.OutputTokens, &usage.ReasoningOutputTokens, &usage.TotalTokens, &usage.Source); err != nil {
			return nil, nil, err
		}
		if cost, isUnpriced := calculator.Compute(usage); isUnpriced {
			unpriced[sourceID] = true
		} else if cost != nil {
			costs[sourceID] += *cost
		}
	}
	return costs, unpriced, rows.Err()
}
