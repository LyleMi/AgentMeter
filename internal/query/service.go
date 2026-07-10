package query

import (
	"context"
	"database/sql"

	"github.com/LyleMi/AgentMeter/internal/model"
)

type Service struct {
	conn *sql.DB
}

type overviewLoader struct {
	service  *Service
	ctx      context.Context
	filters  model.AnalyticsFilters
	overview *model.Overview
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
	overview, err := s.overviewSessionTotals(ctx, filters)
	if err != nil {
		return overview, err
	}
	loader := overviewLoader{service: s, ctx: ctx, filters: filters, overview: &overview}
	if err := loader.populateTotals(); err != nil {
		return overview, err
	}
	if err := loader.populateUsage(); err != nil {
		return overview, err
	}
	if err := loader.populateTimeUsage(); err != nil {
		return overview, err
	}
	if err := loader.populateSessions(); err != nil {
		return overview, err
	}
	normalizeOverviewSlices(&overview)
	return overview, nil
}

func (s *Service) overviewSessionTotals(ctx context.Context, filters model.AnalyticsFilters) (model.Overview, error) {
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
		WHERE `+whereClause(where), args...).Scan(
		&overview.TotalSessions,
		&overview.TotalWallDurationMS,
		&overview.TotalActiveDurationMS,
		&overview.TotalModelDurationMS,
		&overview.TotalToolDurationMS,
		&overview.TotalIdleDurationMS,
	)
	return overview, err
}

func (l overviewLoader) populateTotals() error {
	usage, err := l.service.usageTotals(l.ctx, l.filters)
	if err != nil {
		return err
	}
	l.overview.TotalInputTokens = usage.InputTokens
	l.overview.TotalCachedInputTokens = usage.CachedInputTokens
	l.overview.TotalOutputTokens = usage.OutputTokens
	l.overview.TotalReasoningTokens = usage.ReasoningOutputTokens
	l.overview.TotalContextCompressionTokens = usage.ContextCompressionTokens
	l.overview.TotalTokens = usage.TotalTokens
	l.overview.TotalToolCalls, err = l.service.toolCallCount(l.ctx, l.filters)
	if err != nil {
		return err
	}
	l.overview.SuspectedNetworkToolDurationMS, l.overview.SuspectedNetworkToolCalls, err = l.service.suspectedNetworkToolTotalsWithFilters(l.ctx, l.overview.TotalToolDurationMS, l.filters)
	if err != nil {
		return err
	}
	l.overview.EstimatedCostUSD, l.overview.UnpricedSessions, err = l.service.totalCostWithFilters(l.ctx, l.filters)
	return err
}

func (l overviewLoader) populateUsage() error {
	var err error
	l.overview.DailyUsage, err = l.service.dailyUsageWithFilters(l.ctx, l.filters)
	if err != nil {
		return err
	}
	l.overview.CacheHitTrend = cacheHitTrendFromDailyUsage(l.overview.DailyUsage)
	l.overview.ModelUsage, err = l.service.modelUsageWithFilters(l.ctx, l.filters)
	if err != nil {
		return err
	}
	l.overview.AgentUsage, err = l.service.agentUsageWithFilters(l.ctx, l.filters)
	return err
}

func (l overviewLoader) populateTimeUsage() error {
	var err error
	l.overview.ToolTimeLeaders, err = l.service.toolTimeLeadersWithFilters(l.ctx, l.filters)
	if err != nil {
		return err
	}
	l.overview.AgentTimeUsage, err = l.service.agentTimeUsageWithFilters(l.ctx, l.filters)
	if err != nil {
		return err
	}
	l.overview.ModelTimeUsage, err = l.service.modelTimeUsageWithFilters(l.ctx, l.filters)
	return err
}

func (l overviewLoader) populateSessions() error {
	var err error
	l.overview.RecentSessions, err = l.service.analyticsSessions(l.ctx, l.filters, 6, "s.started_at DESC, s.id DESC", false)
	if err != nil {
		return err
	}
	l.overview.SlowSessions, err = l.service.slowSessionsWithFilters(l.ctx, l.filters)
	return err
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
