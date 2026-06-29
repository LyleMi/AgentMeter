package query

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"

	"github.com/LyleMi/AgentMeter/internal/model"
	"github.com/LyleMi/AgentMeter/internal/sourcepath"
)

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
		COALESCE(SUM(COALESCE(tu.context_compression_tokens, 0)), 0),
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

	costs := newCostAccumulator[string](s.pricingCalculator(ctx))
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
			&bucket.ProjectPath,
			&bucket.Date,
			&pricingModel,
			&bucket.SessionCount,
			&bucket.TotalTokens,
			&bucket.InputTokens,
			&bucket.CachedInputTokens,
			&bucket.OutputTokens,
			&bucket.ReasoningOutputTokens,
			&bucket.ContextCompressionTokens,
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
				ProjectPath:        bucket.ProjectPath,
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
		target.ContextCompressionTokens += bucket.ContextCompressionTokens

		usage := model.Usage{
			Model:                    pricingModel,
			InputTokens:              bucket.InputTokens,
			CachedInputTokens:        bucket.CachedInputTokens,
			OutputTokens:             bucket.OutputTokens,
			ReasoningOutputTokens:    bucket.ReasoningOutputTokens,
			ContextCompressionTokens: bucket.ContextCompressionTokens,
			TotalTokens:              bucket.TotalTokens,
			Source:                   usageSource,
		}
		if cost, unpriced := costs.add(key, usage); unpriced {
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
		bucket.CacheUtilizationRate = cacheUtilizationRate(bucket.InputTokens, bucket.CachedInputTokens)
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
				'' AS model, '' AS project_path, '' AS day, ` + usageSessionModelExpr + ` AS pricing_model`,
			groupSQL: "src.id, " + usageSessionModelExpr,
			orderSQL: "src.name ASC, SUM(COALESCE(tu.total_tokens, 0)) DESC",
		}, nil
	case "model":
		return usageBreakdownShape{
			groupBy: "model",
			selectSQL: `0 AS source_id, '' AS source_root_path, '' AS source_sessions_path, '' AS agent_kind, '' AS agent_name,
				` + usageSessionModelExpr + ` AS model, '' AS project_path, '' AS day, ` + usageSessionModelExpr + ` AS pricing_model`,
			groupSQL: usageSessionModelExpr,
			orderSQL: "SUM(COALESCE(tu.total_tokens, 0)) DESC, " + usageSessionModelExpr + " ASC",
		}, nil
	case "agent,model":
		return usageBreakdownShape{
			groupBy: "agent,model",
			selectSQL: `src.id, src.root_path, src.sessions_path, src.kind, src.name,
				` + usageSessionModelExpr + ` AS model, '' AS project_path, '' AS day, ` + usageSessionModelExpr + ` AS pricing_model`,
			groupSQL: "src.id, " + usageSessionModelExpr,
			orderSQL: "SUM(COALESCE(tu.total_tokens, 0)) DESC, src.name ASC, " + usageSessionModelExpr + " ASC",
		}, nil
	case "project":
		return usageBreakdownShape{
			groupBy: "project",
			selectSQL: `0 AS source_id, '' AS source_root_path, '' AS source_sessions_path, '' AS agent_kind, '' AS agent_name,
				'' AS model, s.project_path AS project_path, '' AS day, ` + usageSessionModelExpr + ` AS pricing_model`,
			groupSQL: "s.project_path, " + usageSessionModelExpr,
			orderSQL: "SUM(COALESCE(tu.total_tokens, 0)) DESC, s.project_path ASC",
		}, nil
	case "day":
		return usageBreakdownShape{
			groupBy: "day",
			selectSQL: `0 AS source_id, '' AS source_root_path, '' AS source_sessions_path, '' AS agent_kind, '' AS agent_name,
				'' AS model, '' AS project_path, substr(s.started_at, 1, 10) AS day, ` + usageSessionModelExpr + ` AS pricing_model`,
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
	case "project":
		return sourcepath.Key(sourcepath.Normalize(bucket.ProjectPath))
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
		case "project":
			if left.TotalTokens != right.TotalTokens {
				return left.TotalTokens > right.TotalTokens
			}
			return sourcepath.Key(sourcepath.Normalize(left.ProjectPath)) < sourcepath.Key(sourcepath.Normalize(right.ProjectPath))
		default:
			if left.TotalTokens != right.TotalTokens {
				return left.TotalTokens > right.TotalTokens
			}
			return left.Model < right.Model
		}
	})
}
