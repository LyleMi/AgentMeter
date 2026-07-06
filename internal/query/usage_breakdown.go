package query

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"

	"github.com/LyleMi/AgentMeter/internal/model"
	"github.com/LyleMi/AgentMeter/internal/pricing"
	"github.com/LyleMi/AgentMeter/internal/sourcepath"
)

type usageBreakdownShape struct {
	groupBy   string
	selectSQL string
	groupSQL  string
	orderSQL  string
}

type usageBreakdownRow struct {
	bucket       model.UsageBreakdownBucket
	pricingModel string
	usageSource  string
}

type usageBreakdownBuilder struct {
	shape        usageBreakdownShape
	costs        *costAccumulator[string]
	bucketsByKey map[string]*model.UsageBreakdownBucket
}

func (s *Service) UsageBreakdown(ctx context.Context, groupBy string, filters model.AnalyticsFilters) (model.UsageBreakdown, error) {
	shape, err := usageBreakdownShapeFor(groupBy)
	if err != nil {
		return model.UsageBreakdown{}, err
	}
	where, args := analyticsSessionWhere(filters)
	rows, err := s.conn.QueryContext(ctx, usageBreakdownQuery(shape, where), args...)
	if err != nil {
		return model.UsageBreakdown{}, err
	}
	defer rows.Close()

	builder := newUsageBreakdownBuilder(shape, s.pricingCalculator(ctx))
	for rows.Next() {
		row, err := scanUsageBreakdownRow(rows)
		if err != nil {
			return model.UsageBreakdown{}, err
		}
		builder.add(row)
	}
	if err := rows.Err(); err != nil {
		return model.UsageBreakdown{}, err
	}
	return builder.result(), nil
}

func usageBreakdownQuery(shape usageBreakdownShape, where []string) string {
	return `SELECT ` + shape.selectSQL + `,
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
		WHERE ` + whereClause(where) + `
		GROUP BY ` + shape.groupSQL + `, s.id, tu.id
		ORDER BY ` + shape.orderSQL
}

type usageBreakdownScanner interface {
	Scan(dest ...any) error
}

func scanUsageBreakdownRow(scanner usageBreakdownScanner) (usageBreakdownRow, error) {
	var row usageBreakdownRow
	err := scanner.Scan(
		&row.bucket.SourceID,
		&row.bucket.SourceRootPath,
		&row.bucket.SourceSessionsPath,
		&row.bucket.AgentKind,
		&row.bucket.AgentName,
		&row.bucket.Model,
		&row.bucket.ProjectPath,
		&row.bucket.Date,
		&row.pricingModel,
		&row.bucket.SessionCount,
		&row.bucket.TotalTokens,
		&row.bucket.InputTokens,
		&row.bucket.CachedInputTokens,
		&row.bucket.OutputTokens,
		&row.bucket.ReasoningOutputTokens,
		&row.bucket.ContextCompressionTokens,
		&row.usageSource,
	)
	return row, err
}

func newUsageBreakdownBuilder(shape usageBreakdownShape, calculator pricing.Calculator) usageBreakdownBuilder {
	return usageBreakdownBuilder{
		shape:        shape,
		costs:        newCostAccumulator[string](calculator),
		bucketsByKey: map[string]*model.UsageBreakdownBucket{},
	}
}

func (b *usageBreakdownBuilder) add(row usageBreakdownRow) {
	key := usageBreakdownBucketKey(b.shape.groupBy, row.bucket)
	target := b.bucketFor(key, row.bucket)
	addUsageBreakdownTokens(target, row.bucket)
	b.addCost(key, row)
}

func (b *usageBreakdownBuilder) bucketFor(key string, bucket model.UsageBreakdownBucket) *model.UsageBreakdownBucket {
	target := b.bucketsByKey[key]
	if target != nil {
		return target
	}
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
	b.bucketsByKey[key] = target
	return target
}

func addUsageBreakdownTokens(target *model.UsageBreakdownBucket, bucket model.UsageBreakdownBucket) {
	target.SessionCount += bucket.SessionCount
	target.TotalTokens += bucket.TotalTokens
	target.InputTokens += bucket.InputTokens
	target.CachedInputTokens += bucket.CachedInputTokens
	target.OutputTokens += bucket.OutputTokens
	target.ReasoningOutputTokens += bucket.ReasoningOutputTokens
	target.ContextCompressionTokens += bucket.ContextCompressionTokens
}

func (b *usageBreakdownBuilder) addCost(key string, row usageBreakdownRow) {
	usage := model.Usage{
		Model:                    row.pricingModel,
		InputTokens:              row.bucket.InputTokens,
		CachedInputTokens:        row.bucket.CachedInputTokens,
		OutputTokens:             row.bucket.OutputTokens,
		ReasoningOutputTokens:    row.bucket.ReasoningOutputTokens,
		ContextCompressionTokens: row.bucket.ContextCompressionTokens,
		TotalTokens:              row.bucket.TotalTokens,
		Source:                   row.usageSource,
	}
	target := b.bucketsByKey[key]
	if cost, unpriced := b.costs.add(key, usage); unpriced {
		target.Unpriced = true
	} else if cost != nil {
		addCost(&target.EstimatedCostUSD, *cost)
	}
}

func (b *usageBreakdownBuilder) result() model.UsageBreakdown {
	result := model.UsageBreakdown{GroupBy: b.shape.groupBy, Buckets: []model.UsageBreakdownBucket{}}
	for _, bucket := range b.bucketsByKey {
		bucket.CacheUtilizationRate = cacheUtilizationRate(bucket.InputTokens, bucket.CachedInputTokens)
		result.Buckets = append(result.Buckets, *bucket)
	}
	sortUsageBreakdownBuckets(result.Buckets, b.shape.groupBy)
	return result
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
