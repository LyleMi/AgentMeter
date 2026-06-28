package query

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"AgentMeter/internal/model"
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
	var overview model.Overview
	err := s.conn.QueryRowContext(ctx, `SELECT
		COUNT(*),
		COALESCE(SUM(wall_duration_ms), 0),
		COALESCE(SUM(active_duration_ms), 0)
		FROM sessions`).Scan(&overview.TotalSessions, &overview.TotalWallDurationMS, &overview.TotalActiveDurationMS)
	if err != nil {
		return overview, err
	}
	err = s.conn.QueryRowContext(ctx, `SELECT
		COALESCE(SUM(input_tokens), 0),
		COALESCE(SUM(cached_input_tokens), 0),
		COALESCE(SUM(output_tokens), 0),
		COALESCE(SUM(reasoning_output_tokens), 0),
		COALESCE(SUM(total_tokens), 0)
		FROM token_usage WHERE owner_kind = 'session'`).
		Scan(&overview.TotalInputTokens, &overview.TotalCachedInputTokens, &overview.TotalOutputTokens, &overview.TotalReasoningTokens, &overview.TotalTokens)
	if err != nil {
		return overview, err
	}
	if err := s.conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM tool_calls`).Scan(&overview.TotalToolCalls); err != nil {
		return overview, err
	}
	overview.EstimatedCostUSD, overview.UnpricedSessions, err = s.totalCost(ctx)
	if err != nil {
		return overview, err
	}
	overview.DailyUsage, err = s.dailyUsage(ctx)
	if err != nil {
		return overview, err
	}
	overview.ModelUsage, err = s.modelUsage(ctx)
	if err != nil {
		return overview, err
	}
	overview.AgentUsage, err = s.agentUsage(ctx)
	if err != nil {
		return overview, err
	}
	overview.RecentSessions, err = s.Sessions(ctx, model.SessionFilters{Limit: 6})
	return overview, err
}

func (s *Service) Sessions(ctx context.Context, filters model.SessionFilters) ([]model.Session, error) {
	where := []string{"1 = 1"}
	args := []any{}
	if strings.TrimSpace(filters.Search) != "" {
		search := "%" + strings.TrimSpace(filters.Search) + "%"
		where = append(where, `(s.session_key LIKE ? OR s.codex_session_id LIKE ? OR s.project_path LIKE ? OR s.model LIKE ? OR sf.path LIKE ? OR src.kind LIKE ? OR src.name LIKE ?)`)
		args = append(args, search, search, search, search, search, search, search)
	}
	if strings.TrimSpace(filters.Model) != "" {
		where = append(where, `s.model = ?`)
		args = append(args, strings.TrimSpace(filters.Model))
	}
	if strings.TrimSpace(filters.Agent) != "" {
		where = append(where, `src.kind = ?`)
		args = append(args, strings.TrimSpace(filters.Agent))
	}
	limit, offset := clampLimitOffset(filters.Limit, filters.Offset, 200, 500)
	args = append(args, limit, offset)
	query := fmt.Sprintf(`%s
		WHERE %s
		ORDER BY s.started_at DESC
		LIMIT ? OFFSET ?`, sessionSelect, strings.Join(where, " AND "))
	return s.scanSessions(ctx, query, args...)
}

func (s *Service) SessionDetail(ctx context.Context, id int64) (model.SessionDetail, error) {
	session, err := s.sessionByID(ctx, id)
	if err != nil {
		return model.SessionDetail{}, err
	}
	events, err := s.events(ctx, id)
	if err != nil {
		return model.SessionDetail{}, err
	}
	modelCalls, err := s.modelCalls(ctx, id)
	if err != nil {
		return model.SessionDetail{}, err
	}
	toolCalls, err := s.toolCalls(ctx, id)
	if err != nil {
		return model.SessionDetail{}, err
	}
	return model.SessionDetail{
		Session:    session,
		Events:     events,
		ModelCalls: modelCalls,
		ToolCalls:  toolCalls,
	}, nil
}

func (s *Service) Tools(ctx context.Context, filters model.ToolFilters) ([]model.ToolStat, error) {
	where := []string{"1 = 1"}
	args := []any{}
	if strings.TrimSpace(filters.Agent) != "" {
		where = append(where, "src.kind = ?")
		args = append(args, strings.TrimSpace(filters.Agent))
	}
	query := fmt.Sprintf(`SELECT
		tc.tool_name,
		COUNT(*),
		SUM(CASE WHEN tc.status IN ('completed', 'success') THEN 1 ELSE 0 END),
		SUM(CASE WHEN tc.status IN ('completed', 'success') THEN 0 ELSE 1 END),
		COALESCE(SUM(tc.duration_ms), 0),
		COALESCE(AVG(tc.duration_ms), 0)
		FROM tool_calls tc
		JOIN sessions sess ON sess.id = tc.session_id
		JOIN sources src ON src.id = sess.source_id
		WHERE %s
		GROUP BY tc.tool_name
		ORDER BY COUNT(*) DESC, tc.tool_name ASC`, strings.Join(where, " AND "))
	rows, err := s.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []model.ToolStat
	for rows.Next() {
		var item model.ToolStat
		if err := rows.Scan(&item.ToolName, &item.Calls, &item.SuccessCalls, &item.FailedCalls, &item.TotalDurationMS, &item.AvgDurationMS); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Service) ToolCalls(ctx context.Context, filters model.ToolCallFilters) ([]model.ToolCall, error) {
	where := []string{"1 = 1"}
	args := []any{}
	if strings.TrimSpace(filters.ToolName) != "" {
		where = append(where, "tc.tool_name = ?")
		args = append(args, strings.TrimSpace(filters.ToolName))
	}
	if strings.TrimSpace(filters.Agent) != "" {
		where = append(where, "src.kind = ?")
		args = append(args, strings.TrimSpace(filters.Agent))
	}
	if strings.TrimSpace(filters.StartedFrom) != "" {
		where = append(where, "tc.started_at >= ?")
		args = append(args, strings.TrimSpace(filters.StartedFrom))
	}
	if strings.TrimSpace(filters.StartedTo) != "" {
		where = append(where, "tc.started_at <= ?")
		args = append(args, strings.TrimSpace(filters.StartedTo))
	}
	limit, offset := clampLimitOffset(filters.Limit, filters.Offset, 500, 1000)
	orderBy := "tc.started_at DESC, tc.id DESC"
	switch strings.TrimSpace(filters.Sort) {
	case "duration_desc":
		orderBy = "tc.duration_ms DESC, tc.started_at DESC, tc.id DESC"
	case "duration_asc":
		orderBy = "tc.duration_ms ASC, tc.started_at DESC, tc.id DESC"
	}
	args = append(args, limit, offset)
	query := fmt.Sprintf(`%s
		WHERE %s
		ORDER BY %s
		LIMIT ? OFFSET ?`, toolCallSelect, strings.Join(where, " AND "), orderBy)
	return s.scanToolCalls(ctx, query, args...)
}

func (s *Service) AuditSummary(ctx context.Context) (model.AuditSummary, error) {
	var summary model.AuditSummary
	err := s.conn.QueryRowContext(ctx, `SELECT
		COUNT(*),
		COALESCE(SUM(CASE WHEN severity = 'critical' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN severity = 'high' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN severity = 'medium' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN severity = 'low' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN category = 'command' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN category = 'privacy' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN category = 'egress' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN category = 'file' THEN 1 ELSE 0 END), 0),
		COUNT(DISTINCT session_id)
		FROM audit_findings`).Scan(
		&summary.TotalFindings,
		&summary.CriticalFindings,
		&summary.HighFindings,
		&summary.MediumFindings,
		&summary.LowFindings,
		&summary.CommandFindings,
		&summary.PrivacyFindings,
		&summary.EgressFindings,
		&summary.FileFindings,
		&summary.SessionsWithFindings,
	)
	if err != nil {
		return summary, err
	}
	summary.RecentFindings, err = s.AuditFindings(ctx, model.AuditFindingFilters{Limit: 8})
	if summary.RecentFindings == nil {
		summary.RecentFindings = []model.AuditFinding{}
	}
	return summary, err
}

func (s *Service) AuditFindings(ctx context.Context, filters model.AuditFindingFilters) ([]model.AuditFinding, error) {
	where := []string{"1 = 1"}
	args := []any{}
	if strings.TrimSpace(filters.Category) != "" {
		where = append(where, "af.category = ?")
		args = append(args, strings.TrimSpace(filters.Category))
	}
	if strings.TrimSpace(filters.Severity) != "" {
		where = append(where, "af.severity = ?")
		args = append(args, strings.TrimSpace(filters.Severity))
	}
	if strings.TrimSpace(filters.ShellFamily) != "" {
		where = append(where, "af.shell_family = ?")
		args = append(args, strings.TrimSpace(filters.ShellFamily))
	}
	if strings.TrimSpace(filters.Search) != "" {
		search := "%" + strings.TrimSpace(filters.Search) + "%"
		where = append(where, `(af.title LIKE ? OR af.description LIKE ? OR af.evidence LIKE ? OR af.command LIKE ? OR af.rule_id LIKE ? OR sess.session_key LIKE ? OR sess.project_path LIKE ? OR sf.path LIKE ?)`)
		args = append(args, search, search, search, search, search, search, search, search)
	}
	limit, offset := clampLimitOffset(filters.Limit, filters.Offset, 500, 1000)
	args = append(args, limit, offset)
	query := fmt.Sprintf(`%s
		WHERE %s
		ORDER BY af.timestamp DESC, af.id DESC
		LIMIT ? OFFSET ?`, auditFindingSelect, strings.Join(where, " AND "))
	return s.scanAuditFindings(ctx, query, args...)
}

func (s *Service) sessionByID(ctx context.Context, id int64) (model.Session, error) {
	query := sessionSelect + ` WHERE s.id = ?`
	sessions, err := s.scanSessions(ctx, query, id)
	if err != nil {
		return model.Session{}, err
	}
	if len(sessions) == 0 {
		return model.Session{}, sql.ErrNoRows
	}
	return sessions[0], nil
}
