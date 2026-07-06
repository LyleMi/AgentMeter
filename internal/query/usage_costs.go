package query

import (
	"context"
	"database/sql"

	"github.com/LyleMi/AgentMeter/internal/model"
	"github.com/LyleMi/AgentMeter/internal/pricing"
)

func (s *Service) pricingCalculator(ctx context.Context) pricing.Calculator {
	calculator, err := pricing.LoadCalculator(ctx, s.conn)
	if err != nil {
		return pricing.Calculator{}
	}
	return calculator
}

func (s *Service) totalCost(ctx context.Context) (*float64, int, error) {
	return s.totalCostWithFilters(ctx, model.AnalyticsFilters{})
}

func (s *Service) totalCostWithFilters(ctx context.Context, filters model.AnalyticsFilters) (*float64, int, error) {
	calculator := s.pricingCalculator(ctx)
	where, args := analyticsUsageWhere(filters)
	rows, err := s.conn.QueryContext(ctx, `SELECT `+usageCostColumns+`
		FROM token_usage tu
		JOIN sessions s ON s.id = tu.owner_id
		JOIN sources src ON src.id = s.source_id
		WHERE `+whereClause(where), args...)
	if err != nil {
		return nil, 0, err
	}

	accumulator, err := scanUsageCosts(rows, calculator, func(rows *sql.Rows) (struct{}, model.Usage, error) {
		var usage model.Usage
		err := scanUsageCostColumns(rows, &usage)
		return struct{}{}, usage, err
	})
	if err != nil {
		return nil, 0, err
	}
	return accumulator.totalCost(), accumulator.unpricedCount, nil
}

func (s *Service) dailyCosts(ctx context.Context, calculator pricing.Calculator) (map[string]float64, error) {
	return s.dailyCostsWithFilters(ctx, calculator, model.AnalyticsFilters{})
}

func (s *Service) dailyCostsWithFilters(ctx context.Context, calculator pricing.Calculator, filters model.AnalyticsFilters) (map[string]float64, error) {
	where, args := analyticsUsageWhere(filters)
	rows, err := s.conn.QueryContext(ctx, `SELECT substr(s.started_at, 1, 10) AS day, `+usageCostColumns+`
		FROM sessions s
		JOIN sources src ON src.id = s.source_id
		JOIN token_usage tu ON tu.owner_kind = 'session' AND tu.owner_id = s.id
		WHERE `+whereClause(where), args...)
	if err != nil {
		return nil, err
	}

	accumulator, err := scanUsageCosts(rows, calculator, func(rows *sql.Rows) (string, model.Usage, error) {
		var day string
		var usage model.Usage
		err := rows.Scan(&day, &usage.Model, &usage.InputTokens, &usage.CachedInputTokens, &usage.OutputTokens, &usage.ReasoningOutputTokens, &usage.TotalTokens, &usage.Source)
		return day, usage, err
	})
	if err != nil {
		return nil, err
	}
	return accumulator.costs, nil
}

func (s *Service) modelCostsWithFilters(ctx context.Context, calculator pricing.Calculator, filters model.AnalyticsFilters) (map[string]float64, map[string]bool, error) {
	where, args := analyticsUsageWhere(filters)
	where = appendUsageMetricFilter(where)
	rows, err := s.conn.QueryContext(ctx, `SELECT `+usageCostColumns+`
		FROM token_usage tu
		JOIN sessions s ON s.id = tu.owner_id
		JOIN sources src ON src.id = s.source_id
		WHERE `+whereClause(where), args...)
	if err != nil {
		return nil, nil, err
	}

	accumulator, err := scanUsageCosts(rows, calculator, func(rows *sql.Rows) (string, model.Usage, error) {
		var usage model.Usage
		err := scanUsageCostColumns(rows, &usage)
		return usage.Model, usage, err
	})
	if err != nil {
		return nil, nil, err
	}
	return accumulator.costs, accumulator.unpriced, nil
}

func (s *Service) agentCosts(ctx context.Context, calculator pricing.Calculator) (map[int64]float64, map[int64]bool, error) {
	return s.agentCostsWithFilters(ctx, calculator, model.AnalyticsFilters{})
}

func (s *Service) agentCostsWithFilters(ctx context.Context, calculator pricing.Calculator, filters model.AnalyticsFilters) (map[int64]float64, map[int64]bool, error) {
	where, args := analyticsUsageWhere(filters)
	rows, err := s.conn.QueryContext(ctx, `SELECT src.id, `+usageCostColumns+`
		FROM sessions s
		JOIN sources src ON src.id = s.source_id
		JOIN token_usage tu ON tu.owner_kind = 'session' AND tu.owner_id = s.id
		WHERE `+whereClause(where), args...)
	if err != nil {
		return nil, nil, err
	}

	accumulator, err := scanUsageCosts(rows, calculator, func(rows *sql.Rows) (int64, model.Usage, error) {
		var sourceID int64
		var usage model.Usage
		err := rows.Scan(&sourceID, &usage.Model, &usage.InputTokens, &usage.CachedInputTokens, &usage.OutputTokens, &usage.ReasoningOutputTokens, &usage.TotalTokens, &usage.Source)
		return sourceID, usage, err
	})
	if err != nil {
		return nil, nil, err
	}
	return accumulator.costs, accumulator.unpriced, nil
}

type costAccumulator[K comparable] struct {
	calculator    pricing.Calculator
	costs         map[K]float64
	unpriced      map[K]bool
	total         float64
	hasCost       bool
	unpricedCount int
}

func newCostAccumulator[K comparable](calculator pricing.Calculator) *costAccumulator[K] {
	return &costAccumulator[K]{
		calculator: calculator,
		costs:      map[K]float64{},
		unpriced:   map[K]bool{},
	}
}

func (a *costAccumulator[K]) add(key K, usage model.Usage) (*float64, bool) {
	cost, unpriced := a.calculator.Compute(usage)
	if unpriced {
		a.unpriced[key] = true
		a.unpricedCount++
		return nil, true
	}
	if cost != nil {
		a.costs[key] += *cost
		a.total += *cost
		a.hasCost = true
	}
	return cost, false
}

func (a *costAccumulator[K]) totalCost() *float64 {
	if !a.hasCost {
		return nil
	}
	return &a.total
}

func scanUsageCosts[K comparable](
	rows *sql.Rows,
	calculator pricing.Calculator,
	scan func(*sql.Rows) (K, model.Usage, error),
) (*costAccumulator[K], error) {
	defer rows.Close()
	accumulator := newCostAccumulator[K](calculator)
	for rows.Next() {
		key, usage, err := scan(rows)
		if err != nil {
			return nil, err
		}
		accumulator.add(key, usage)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return accumulator, nil
}

func scanUsageCostColumns(rows *sql.Rows, usage *model.Usage) error {
	return rows.Scan(
		&usage.Model,
		&usage.InputTokens,
		&usage.CachedInputTokens,
		&usage.OutputTokens,
		&usage.ReasoningOutputTokens,
		&usage.TotalTokens,
		&usage.Source,
	)
}
