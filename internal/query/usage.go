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
		COALESCE(SUM(tu.total_tokens), 0), COALESCE(SUM(tu.input_tokens), 0), COALESCE(SUM(tu.output_tokens), 0),
		COALESCE(SUM(tu.cached_input_tokens), 0), COALESCE(SUM(tu.reasoning_output_tokens), 0), COALESCE(MAX(tu.source), 'unknown')
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
		var cached, reasoning int64
		var source string
		if err := rows.Scan(&item.Model, &item.SessionCount, &item.TotalTokens, &item.InputTokens, &item.OutputTokens, &cached, &reasoning, &source); err != nil {
			return nil, err
		}
		usage := model.Usage{
			Model:                 item.Model,
			InputTokens:           item.InputTokens,
			CachedInputTokens:     cached,
			OutputTokens:          item.OutputTokens,
			ReasoningOutputTokens: reasoning,
			TotalTokens:           item.TotalTokens,
			Source:                source,
		}
		item.EstimatedCostUSD, item.Unpriced = calculator.Compute(usage)
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Service) agentUsage(ctx context.Context) ([]model.AgentUsage, error) {
	rows, err := s.conn.QueryContext(ctx, `SELECT src.kind, src.name, COUNT(*),
		COALESCE(SUM(tu.total_tokens), 0), COALESCE(SUM(tu.input_tokens), 0), COALESCE(SUM(tu.output_tokens), 0),
		COALESCE(SUM((SELECT COUNT(*) FROM tool_calls tc WHERE tc.session_id = s.id)), 0)
		FROM sessions s
		JOIN sources src ON src.id = s.source_id
		LEFT JOIN token_usage tu ON tu.owner_kind = 'session' AND tu.owner_id = s.id
		GROUP BY src.kind, src.name
		ORDER BY COUNT(*) DESC, src.name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []model.AgentUsage
	for rows.Next() {
		var item model.AgentUsage
		if err := rows.Scan(&item.AgentKind, &item.AgentName, &item.SessionCount, &item.TotalTokens, &item.InputTokens, &item.OutputTokens, &item.ToolCalls); err != nil {
			return nil, err
		}
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
		key := sourceKey(result[index].AgentKind, result[index].AgentName)
		if cost, ok := costs[key]; ok {
			result[index].EstimatedCostUSD = &cost
		}
		result[index].Unpriced = unpriced[key]
	}
	return result, nil
}

func (s *Service) agentCosts(ctx context.Context, calculator pricing.Calculator) (map[string]float64, map[string]bool, error) {
	rows, err := s.conn.QueryContext(ctx, `SELECT src.kind, src.name, tu.model,
		COALESCE(SUM(tu.input_tokens), 0), COALESCE(SUM(tu.cached_input_tokens), 0), COALESCE(SUM(tu.output_tokens), 0),
		COALESCE(SUM(tu.reasoning_output_tokens), 0), COALESCE(SUM(tu.total_tokens), 0), COALESCE(MAX(tu.source), 'unknown')
		FROM sessions s
		JOIN sources src ON src.id = s.source_id
		JOIN token_usage tu ON tu.owner_kind = 'session' AND tu.owner_id = s.id
		GROUP BY src.kind, src.name, tu.model`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	costs := map[string]float64{}
	unpriced := map[string]bool{}
	for rows.Next() {
		var kind, name string
		var usage model.Usage
		if err := rows.Scan(&kind, &name, &usage.Model, &usage.InputTokens, &usage.CachedInputTokens, &usage.OutputTokens, &usage.ReasoningOutputTokens, &usage.TotalTokens, &usage.Source); err != nil {
			return nil, nil, err
		}
		key := sourceKey(kind, name)
		if cost, isUnpriced := calculator.Compute(usage); isUnpriced {
			unpriced[key] = true
		} else if cost != nil {
			costs[key] += *cost
		}
	}
	return costs, unpriced, rows.Err()
}

func sourceKey(kind, name string) string {
	return kind + "\x00" + name
}
