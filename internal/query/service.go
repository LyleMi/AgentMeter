package query

import (
	"context"
	"database/sql"
	"strings"

	"github.com/LyleMi/AgentMeter/internal/model"
)

type Service struct {
	conn *sql.DB
}

func New(conn *sql.DB) *Service {
	return &Service{conn: conn}
}

func clampLimitOffset(limit, offset, defaultLimit, maxLimit int) (int, int) {
	if limit <= 0 || limit > maxLimit {
		limit = defaultLimit
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func (s *Service) Overview(ctx context.Context) (model.Overview, error) {
	return s.OverviewWithFilters(ctx, model.AnalyticsFilters{})
}

func (s *Service) OverviewWithFilters(ctx context.Context, filters model.AnalyticsFilters) (model.Overview, error) {
	var overview model.Overview
	where, args := analyticsSessionWhere(filters)
	err := s.conn.QueryRowContext(ctx, `SELECT
		COUNT(*),
		COALESCE(SUM(s.wall_duration_ms), 0),
		COALESCE(SUM(s.active_duration_ms), 0),
		COALESCE(SUM(s.model_duration_ms), 0),
		COALESCE(SUM(s.tool_duration_ms), 0),
		COALESCE(SUM(s.idle_duration_ms), 0)
		FROM sessions s
		JOIN sources src ON src.id = s.source_id
		LEFT JOIN token_usage tu ON tu.owner_kind = 'session' AND tu.owner_id = s.id
		WHERE `+strings.Join(where, " AND "), args...).Scan(
		&overview.TotalSessions,
		&overview.TotalWallDurationMS,
		&overview.TotalActiveDurationMS,
		&overview.TotalModelDurationMS,
		&overview.TotalToolDurationMS,
		&overview.TotalIdleDurationMS,
	)
	if err != nil {
		return overview, err
	}
	usage, err := s.usageTotals(ctx, filters)
	if err != nil {
		return overview, err
	}
	overview.TotalInputTokens = usage.InputTokens
	overview.TotalCachedInputTokens = usage.CachedInputTokens
	overview.TotalOutputTokens = usage.OutputTokens
	overview.TotalReasoningTokens = usage.ReasoningOutputTokens
	overview.TotalContextCompressionTokens = usage.ContextCompressionTokens
	overview.TotalTokens = usage.TotalTokens
	overview.TotalToolCalls, err = s.toolCallCount(ctx, filters)
	if err != nil {
		return overview, err
	}
	overview.SuspectedNetworkToolDurationMS, overview.SuspectedNetworkToolCalls, err = s.suspectedNetworkToolTotalsWithFilters(ctx, overview.TotalToolDurationMS, filters)
	if err != nil {
		return overview, err
	}
	overview.EstimatedCostUSD, overview.UnpricedSessions, err = s.totalCostWithFilters(ctx, filters)
	if err != nil {
		return overview, err
	}
	overview.DailyUsage, err = s.dailyUsageWithFilters(ctx, filters)
	if err != nil {
		return overview, err
	}
	overview.CacheHitTrend = cacheHitTrendFromDailyUsage(overview.DailyUsage)
	overview.ModelUsage, err = s.modelUsageWithFilters(ctx, filters)
	if err != nil {
		return overview, err
	}
	overview.AgentUsage, err = s.agentUsageWithFilters(ctx, filters)
	if err != nil {
		return overview, err
	}
	overview.ToolTimeLeaders, err = s.toolTimeLeadersWithFilters(ctx, filters)
	if err != nil {
		return overview, err
	}
	overview.AgentTimeUsage, err = s.agentTimeUsageWithFilters(ctx, filters)
	if err != nil {
		return overview, err
	}
	overview.ModelTimeUsage, err = s.modelTimeUsageWithFilters(ctx, filters)
	if err != nil {
		return overview, err
	}
	overview.RecentSessions, err = s.analyticsSessions(ctx, filters, 6, "s.started_at DESC, s.id DESC", false)
	if err != nil {
		return overview, err
	}
	overview.SlowSessions, err = s.slowSessionsWithFilters(ctx, filters)
	normalizeOverviewSlices(&overview)
	return overview, err
}

func normalizeOverviewSlices(overview *model.Overview) {
	overview.DailyUsage = nonNilSlice(overview.DailyUsage)
	overview.CacheHitTrend = nonNilSlice(overview.CacheHitTrend)
	overview.ModelUsage = nonNilSlice(overview.ModelUsage)
	overview.AgentUsage = nonNilSlice(overview.AgentUsage)
	overview.ToolTimeLeaders = nonNilSlice(overview.ToolTimeLeaders)
	overview.AgentTimeUsage = nonNilSlice(overview.AgentTimeUsage)
	overview.ModelTimeUsage = nonNilSlice(overview.ModelTimeUsage)
	overview.RecentSessions = nonNilSlice(overview.RecentSessions)
	overview.SlowSessions = nonNilSlice(overview.SlowSessions)
}
