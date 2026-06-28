package query

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"

	"AgentMeter/internal/model"
	"AgentMeter/internal/pricing"
)

const usageSessionModelExpr = "COALESCE(NULLIF(tu.model, ''), s.model)"

func (s *Service) pricingCalculator(ctx context.Context) pricing.Calculator {
	calculator, err := pricing.LoadCalculator(ctx, s.conn)
	if err != nil {
		return pricing.Calculator{}
	}
	return calculator
}

func analyticsSessionWhere(filters model.AnalyticsFilters) ([]string, []any) {
	where := []string{"1 = 1"}
	args := []any{}
	return appendAnalyticsFilters(where, args, filters, "src", usageSessionModelExpr, "s.started_at")
}

func analyticsUsageWhere(filters model.AnalyticsFilters) ([]string, []any) {
	where := []string{"tu.owner_kind = 'session'"}
	args := []any{}
	return appendAnalyticsFilters(where, args, filters, "src", usageSessionModelExpr, "s.started_at")
}

func appendBillableUsageFilter(where []string) []string {
	return append(where, `(tu.input_tokens > 0 OR tu.cached_input_tokens > 0 OR tu.output_tokens > 0 OR tu.reasoning_output_tokens > 0 OR tu.total_tokens > 0)`)
}

func (s *Service) totalCost(ctx context.Context) (*float64, int, error) {
	return s.totalCostWithFilters(ctx, model.AnalyticsFilters{})
}

func (s *Service) totalCostWithFilters(ctx context.Context, filters model.AnalyticsFilters) (*float64, int, error) {
	calculator := s.pricingCalculator(ctx)
	where, args := analyticsUsageWhere(filters)
	rows, err := s.conn.QueryContext(ctx, `SELECT `+usageSessionModelExpr+`, tu.input_tokens, tu.cached_input_tokens, tu.output_tokens, tu.reasoning_output_tokens, tu.total_tokens, tu.source
		FROM token_usage tu
		JOIN sessions s ON s.id = tu.owner_id
		JOIN sources src ON src.id = s.source_id
		WHERE `+strings.Join(where, " AND "), args...)
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
	result.TotalTokens = usage.TotalTokens
	if result.TotalInputTokens > 0 {
		result.CacheUtilizationRate = float64(result.TotalCachedInputTokens) / float64(result.TotalInputTokens)
	}
	result.EstimatedCostUSD, result.UnpricedCount, err = s.totalCostWithFilters(ctx, filters)
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
		COALESCE(SUM(tu.total_tokens), 0)
		FROM token_usage tu
		JOIN sessions s ON s.id = tu.owner_id
		JOIN sources src ON src.id = s.source_id
		WHERE `+strings.Join(where, " AND "), args...).
		Scan(&usage.InputTokens, &usage.CachedInputTokens, &usage.OutputTokens, &usage.ReasoningOutputTokens, &usage.TotalTokens)
	return usage, err
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
			COALESCE(SUM(tu.output_tokens), 0) AS output_tokens,
			COALESCE(SUM((SELECT COUNT(*) FROM tool_calls tc WHERE tc.session_id = s.id)), 0) AS tool_calls
		FROM sessions s
		JOIN sources src ON src.id = s.source_id
		LEFT JOIN token_usage tu ON tu.owner_kind = 'session' AND tu.owner_id = s.id
		WHERE `+strings.Join(where, " AND ")+`
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
		if err := rows.Scan(&item.Date, &item.SessionCount, &item.TotalTokens, &item.InputTokens, &item.OutputTokens, &item.ToolCalls); err != nil {
			return nil, err
		}
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

func (s *Service) dailyCosts(ctx context.Context, calculator pricing.Calculator) (map[string]float64, error) {
	return s.dailyCostsWithFilters(ctx, calculator, model.AnalyticsFilters{})
}

func (s *Service) dailyCostsWithFilters(ctx context.Context, calculator pricing.Calculator, filters model.AnalyticsFilters) (map[string]float64, error) {
	where, args := analyticsUsageWhere(filters)
	rows, err := s.conn.QueryContext(ctx, `SELECT substr(s.started_at, 1, 10) AS day, `+usageSessionModelExpr+`,
		tu.input_tokens, tu.cached_input_tokens, tu.output_tokens, tu.reasoning_output_tokens, tu.total_tokens, tu.source
		FROM sessions s
		JOIN sources src ON src.id = s.source_id
		JOIN token_usage tu ON tu.owner_kind = 'session' AND tu.owner_id = s.id
		WHERE `+strings.Join(where, " AND "), args...)
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
	return s.modelUsageWithFilters(ctx, model.AnalyticsFilters{})
}

func (s *Service) modelUsageWithFilters(ctx context.Context, filters model.AnalyticsFilters) ([]model.ModelUsage, error) {
	calculator := s.pricingCalculator(ctx)
	costs, unpriced, err := s.modelCostsWithFilters(ctx, calculator, filters)
	if err != nil {
		return nil, err
	}
	where, args := analyticsUsageWhere(filters)
	where = appendBillableUsageFilter(where)
	rows, err := s.conn.QueryContext(ctx, `SELECT `+usageSessionModelExpr+`, COUNT(*),
		COALESCE(SUM(tu.total_tokens), 0), COALESCE(SUM(tu.input_tokens), 0), COALESCE(SUM(tu.cached_input_tokens), 0),
		COALESCE(SUM(tu.output_tokens), 0), COALESCE(SUM(tu.reasoning_output_tokens), 0)
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

func (s *Service) modelCostsWithFilters(ctx context.Context, calculator pricing.Calculator, filters model.AnalyticsFilters) (map[string]float64, map[string]bool, error) {
	where, args := analyticsUsageWhere(filters)
	where = appendBillableUsageFilter(where)
	rows, err := s.conn.QueryContext(ctx, `SELECT `+usageSessionModelExpr+`,
		tu.input_tokens, tu.cached_input_tokens, tu.output_tokens, tu.reasoning_output_tokens, tu.total_tokens, tu.source
		FROM token_usage tu
		JOIN sessions s ON s.id = tu.owner_id
		JOIN sources src ON src.id = s.source_id
		WHERE `+strings.Join(where, " AND "), args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	costs := map[string]float64{}
	unpriced := map[string]bool{}
	for rows.Next() {
		var usage model.Usage
		if err := rows.Scan(&usage.Model, &usage.InputTokens, &usage.CachedInputTokens, &usage.OutputTokens, &usage.ReasoningOutputTokens, &usage.TotalTokens, &usage.Source); err != nil {
			return nil, nil, err
		}
		if cost, isUnpriced := calculator.Compute(usage); isUnpriced {
			unpriced[usage.Model] = true
		} else if cost != nil {
			costs[usage.Model] += *cost
		}
	}
	return costs, unpriced, rows.Err()
}

func (s *Service) agentUsage(ctx context.Context) ([]model.AgentUsage, error) {
	return s.agentUsageWithFilters(ctx, model.AnalyticsFilters{})
}

func (s *Service) agentUsageWithFilters(ctx context.Context, filters model.AnalyticsFilters) ([]model.AgentUsage, error) {
	where, args := analyticsSessionWhere(filters)
	rows, err := s.conn.QueryContext(ctx, `SELECT src.id, src.root_path, src.sessions_path, src.kind, src.name, COUNT(*),
		COALESCE(SUM(tu.total_tokens), 0), COALESCE(SUM(tu.input_tokens), 0), COALESCE(SUM(tu.cached_input_tokens), 0),
		COALESCE(SUM(tu.output_tokens), 0), COALESCE(SUM(tu.reasoning_output_tokens), 0),
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

func (s *Service) agentCosts(ctx context.Context, calculator pricing.Calculator) (map[int64]float64, map[int64]bool, error) {
	return s.agentCostsWithFilters(ctx, calculator, model.AnalyticsFilters{})
}

func (s *Service) agentCostsWithFilters(ctx context.Context, calculator pricing.Calculator, filters model.AnalyticsFilters) (map[int64]float64, map[int64]bool, error) {
	where, args := analyticsUsageWhere(filters)
	rows, err := s.conn.QueryContext(ctx, `SELECT src.id, `+usageSessionModelExpr+`,
		tu.input_tokens, tu.cached_input_tokens, tu.output_tokens, tu.reasoning_output_tokens, tu.total_tokens, tu.source
		FROM sessions s
		JOIN sources src ON src.id = s.source_id
		JOIN token_usage tu ON tu.owner_kind = 'session' AND tu.owner_id = s.id
		WHERE `+strings.Join(where, " AND "), args...)
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

type usageBreakdownShape struct {
	groupBy   string
	selectSQL string
	groupSQL  string
	orderSQL  string
}

func (s *Service) UsageBreakdown(ctx context.Context, groupBy string, filters model.AnalyticsFilters) (model.UsageBreakdown, error) {
	shape, err := usageBreakdownShapeFor(groupBy)
	if err != nil {
		return model.UsageBreakdown{}, err
	}
	where, args := analyticsSessionWhere(filters)
	query := `SELECT ` + shape.selectSQL + `,
		COUNT(DISTINCT s.id),
		COALESCE(SUM(COALESCE(tu.total_tokens, 0)), 0),
		COALESCE(SUM(COALESCE(tu.input_tokens, 0)), 0),
		COALESCE(SUM(COALESCE(tu.cached_input_tokens, 0)), 0),
		COALESCE(SUM(COALESCE(tu.output_tokens, 0)), 0),
		COALESCE(SUM(COALESCE(tu.reasoning_output_tokens, 0)), 0),
		COALESCE(MAX(tu.source), 'unknown')
		FROM sessions s
		JOIN sources src ON src.id = s.source_id
		LEFT JOIN token_usage tu ON tu.owner_kind = 'session' AND tu.owner_id = s.id
		WHERE ` + strings.Join(where, " AND ") + `
		GROUP BY ` + shape.groupSQL + `, s.id, tu.id
		ORDER BY ` + shape.orderSQL
	rows, err := s.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return model.UsageBreakdown{}, err
	}
	defer rows.Close()

	calculator := s.pricingCalculator(ctx)
	bucketsByKey := map[string]*model.UsageBreakdownBucket{}
	for rows.Next() {
		var bucket model.UsageBreakdownBucket
		var pricingModel string
		var usageSource string
		if err := rows.Scan(
			&bucket.SourceID,
			&bucket.SourceRootPath,
			&bucket.SourceSessionsPath,
			&bucket.AgentKind,
			&bucket.AgentName,
			&bucket.Model,
			&bucket.Date,
			&pricingModel,
			&bucket.SessionCount,
			&bucket.TotalTokens,
			&bucket.InputTokens,
			&bucket.CachedInputTokens,
			&bucket.OutputTokens,
			&bucket.ReasoningOutputTokens,
			&usageSource,
		); err != nil {
			return model.UsageBreakdown{}, err
		}
		key := usageBreakdownBucketKey(shape.groupBy, bucket)
		target := bucketsByKey[key]
		if target == nil {
			target = &model.UsageBreakdownBucket{
				SourceID:           bucket.SourceID,
				SourceRootPath:     bucket.SourceRootPath,
				SourceSessionsPath: bucket.SourceSessionsPath,
				AgentKind:          bucket.AgentKind,
				AgentName:          bucket.AgentName,
				Model:              bucket.Model,
				Date:               bucket.Date,
			}
			fillBreakdownSourceIdentity(target)
			bucketsByKey[key] = target
		}
		target.SessionCount += bucket.SessionCount
		target.TotalTokens += bucket.TotalTokens
		target.InputTokens += bucket.InputTokens
		target.CachedInputTokens += bucket.CachedInputTokens
		target.OutputTokens += bucket.OutputTokens
		target.ReasoningOutputTokens += bucket.ReasoningOutputTokens

		usage := model.Usage{
			Model:                 pricingModel,
			InputTokens:           bucket.InputTokens,
			CachedInputTokens:     bucket.CachedInputTokens,
			OutputTokens:          bucket.OutputTokens,
			ReasoningOutputTokens: bucket.ReasoningOutputTokens,
			TotalTokens:           bucket.TotalTokens,
			Source:                usageSource,
		}
		if cost, unpriced := calculator.Compute(usage); unpriced {
			target.Unpriced = true
		} else if cost != nil {
			addCost(&target.EstimatedCostUSD, *cost)
		}
	}
	if err := rows.Err(); err != nil {
		return model.UsageBreakdown{}, err
	}

	result := model.UsageBreakdown{GroupBy: shape.groupBy, Buckets: []model.UsageBreakdownBucket{}}
	for _, bucket := range bucketsByKey {
		if bucket.InputTokens > 0 {
			bucket.CacheUtilizationRate = float64(bucket.CachedInputTokens) / float64(bucket.InputTokens)
		}
		result.Buckets = append(result.Buckets, *bucket)
	}
	sortUsageBreakdownBuckets(result.Buckets, shape.groupBy)
	return result, nil
}

func usageBreakdownShapeFor(groupBy string) (usageBreakdownShape, error) {
	normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(groupBy), " ", ""))
	switch normalized {
	case "agent":
		return usageBreakdownShape{
			groupBy: "agent",
			selectSQL: `src.id, src.root_path, src.sessions_path, src.kind, src.name,
				'' AS model, '' AS day, ` + usageSessionModelExpr + ` AS pricing_model`,
			groupSQL: "src.id, " + usageSessionModelExpr,
			orderSQL: "src.name ASC, SUM(COALESCE(tu.total_tokens, 0)) DESC",
		}, nil
	case "model":
		return usageBreakdownShape{
			groupBy: "model",
			selectSQL: `0 AS source_id, '' AS source_root_path, '' AS source_sessions_path, '' AS agent_kind, '' AS agent_name,
				` + usageSessionModelExpr + ` AS model, '' AS day, ` + usageSessionModelExpr + ` AS pricing_model`,
			groupSQL: usageSessionModelExpr,
			orderSQL: "SUM(COALESCE(tu.total_tokens, 0)) DESC, " + usageSessionModelExpr + " ASC",
		}, nil
	case "agent,model":
		return usageBreakdownShape{
			groupBy: "agent,model",
			selectSQL: `src.id, src.root_path, src.sessions_path, src.kind, src.name,
				` + usageSessionModelExpr + ` AS model, '' AS day, ` + usageSessionModelExpr + ` AS pricing_model`,
			groupSQL: "src.id, " + usageSessionModelExpr,
			orderSQL: "SUM(COALESCE(tu.total_tokens, 0)) DESC, src.name ASC, " + usageSessionModelExpr + " ASC",
		}, nil
	case "day":
		return usageBreakdownShape{
			groupBy: "day",
			selectSQL: `0 AS source_id, '' AS source_root_path, '' AS source_sessions_path, '' AS agent_kind, '' AS agent_name,
				'' AS model, substr(s.started_at, 1, 10) AS day, ` + usageSessionModelExpr + ` AS pricing_model`,
			groupSQL: "substr(s.started_at, 1, 10), " + usageSessionModelExpr,
			orderSQL: "day ASC, SUM(COALESCE(tu.total_tokens, 0)) DESC",
		}, nil
	default:
		return usageBreakdownShape{}, errors.New("unsupported usage breakdown groupBy: " + groupBy)
	}
}

func usageBreakdownBucketKey(groupBy string, bucket model.UsageBreakdownBucket) string {
	switch groupBy {
	case "agent":
		return strconv.FormatInt(bucket.SourceID, 10)
	case "model":
		return bucket.Model
	case "agent,model":
		return strconv.FormatInt(bucket.SourceID, 10) + "\x00" + bucket.Model
	case "day":
		return bucket.Date
	default:
		return bucket.Model
	}
}

func fillBreakdownSourceIdentity(item *model.UsageBreakdownBucket) {
	if item.SourceID <= 0 {
		return
	}
	item.SourceKey, item.SourceLabel = sourceIdentity(item.SourceID, item.AgentName, item.AgentKind)
}

func addCost(target **float64, cost float64) {
	if *target == nil {
		value := 0.0
		*target = &value
	}
	**target += cost
}

func sortUsageBreakdownBuckets(buckets []model.UsageBreakdownBucket, groupBy string) {
	sort.Slice(buckets, func(i, j int) bool {
		left := buckets[i]
		right := buckets[j]
		switch groupBy {
		case "day":
			if left.Date != right.Date {
				return left.Date < right.Date
			}
			return left.TotalTokens > right.TotalTokens
		case "agent":
			if left.SessionCount != right.SessionCount {
				return left.SessionCount > right.SessionCount
			}
			if left.SourceLabel != right.SourceLabel {
				return left.SourceLabel < right.SourceLabel
			}
			return left.SourceID < right.SourceID
		case "agent,model":
			if left.TotalTokens != right.TotalTokens {
				return left.TotalTokens > right.TotalTokens
			}
			if left.SourceLabel != right.SourceLabel {
				return left.SourceLabel < right.SourceLabel
			}
			return left.Model < right.Model
		default:
			if left.TotalTokens != right.TotalTokens {
				return left.TotalTokens > right.TotalTokens
			}
			return left.Model < right.Model
		}
	})
}
