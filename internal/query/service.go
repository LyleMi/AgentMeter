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
		COALESCE(SUM(active_duration_ms), 0),
		COALESCE(SUM(model_duration_ms), 0),
		COALESCE(SUM(tool_duration_ms), 0),
		COALESCE(SUM(idle_duration_ms), 0)
		FROM sessions`).Scan(
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
	overview.SuspectedNetworkToolDurationMS, overview.SuspectedNetworkToolCalls, err = s.suspectedNetworkToolTotals(ctx, overview.TotalToolDurationMS)
	if err != nil {
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
	overview.ToolTimeLeaders, err = s.toolTimeLeaders(ctx)
	if err != nil {
		return overview, err
	}
	overview.AgentTimeUsage, err = s.agentTimeUsage(ctx)
	if err != nil {
		return overview, err
	}
	overview.ModelTimeUsage, err = s.modelTimeUsage(ctx)
	if err != nil {
		return overview, err
	}
	overview.RecentSessions, err = s.Sessions(ctx, model.SessionFilters{Limit: 6})
	if err != nil {
		return overview, err
	}
	overview.SlowSessions, err = s.slowSessions(ctx)
	normalizeOverviewSlices(&overview)
	return overview, err
}

func normalizeOverviewSlices(overview *model.Overview) {
	if overview.DailyUsage == nil {
		overview.DailyUsage = []model.DailyUsage{}
	}
	if overview.ModelUsage == nil {
		overview.ModelUsage = []model.ModelUsage{}
	}
	if overview.AgentUsage == nil {
		overview.AgentUsage = []model.AgentUsage{}
	}
	if overview.ToolTimeLeaders == nil {
		overview.ToolTimeLeaders = []model.ToolTimeUsage{}
	}
	if overview.AgentTimeUsage == nil {
		overview.AgentTimeUsage = []model.AgentTimeUsage{}
	}
	if overview.ModelTimeUsage == nil {
		overview.ModelTimeUsage = []model.ModelTimeUsage{}
	}
	if overview.RecentSessions == nil {
		overview.RecentSessions = []model.Session{}
	}
	if overview.SlowSessions == nil {
		overview.SlowSessions = []model.Session{}
	}
}

var suspectedNetworkToolTerms = []string{
	"web",
	"http://",
	"https://",
	"curl",
	"wget",
	"invoke-webrequest",
	"git fetch",
	"git pull",
	"git clone",
	"npm install",
	"npm ci",
	"pnpm install",
	"yarn install",
	"pip install",
	"go get",
	"go mod download",
}

func isSuspectedNetworkTool(toolName, inputSummary string) bool {
	text := strings.ToLower(toolName + " " + inputSummary)
	for _, term := range suspectedNetworkToolTerms {
		if strings.Contains(text, term) {
			return true
		}
	}
	return false
}

func suspectedNetworkToolCondition(alias string) string {
	// This is intentionally conservative: only lower-cased tool name/input text
	// containing obvious network markers or install/fetch commands is counted.
	text := fmt.Sprintf("LOWER(COALESCE(%s.tool_name, '') || ' ' || COALESCE(%s.input_summary, ''))", alias, alias)
	parts := make([]string, 0, len(suspectedNetworkToolTerms))
	for _, term := range suspectedNetworkToolTerms {
		parts = append(parts, fmt.Sprintf("%s LIKE '%%%s%%'", text, strings.ReplaceAll(term, "'", "''")))
	}
	return "(" + strings.Join(parts, " OR ") + ")"
}

func (s *Service) suspectedNetworkToolTotals(ctx context.Context, totalToolDurationMS int64) (int64, int, error) {
	var duration int64
	var calls int
	query := fmt.Sprintf(`SELECT
		COALESCE(SUM(CASE WHEN tc.duration_ms > 0 THEN tc.duration_ms ELSE 0 END), 0),
		COUNT(*)
		FROM tool_calls tc
		WHERE %s`, suspectedNetworkToolCondition("tc"))
	if err := s.conn.QueryRowContext(ctx, query).Scan(&duration, &calls); err != nil {
		return 0, 0, err
	}
	if duration < 0 {
		duration = 0
	}
	if totalToolDurationMS < 0 {
		duration = 0
	} else if duration > totalToolDurationMS {
		duration = totalToolDurationMS
	}
	return duration, calls, nil
}

func (s *Service) toolTimeLeaders(ctx context.Context) ([]model.ToolTimeUsage, error) {
	networkCondition := suspectedNetworkToolCondition("tc")
	rows, err := s.conn.QueryContext(ctx, fmt.Sprintf(`SELECT
		tc.tool_name,
		COUNT(*),
		SUM(CASE WHEN tc.status IN ('completed', 'success') THEN 1 ELSE 0 END),
		SUM(CASE WHEN tc.status IN ('completed', 'success') THEN 0 ELSE 1 END),
		COALESCE(SUM(tc.duration_ms), 0),
		COALESCE(AVG(tc.duration_ms), 0),
		COALESCE(MAX(tc.duration_ms), 0),
		MAX(CASE WHEN %s THEN 1 ELSE 0 END)
		FROM tool_calls tc
		GROUP BY tc.tool_name
		ORDER BY SUM(tc.duration_ms) DESC, tc.tool_name ASC
		LIMIT 8`, networkCondition))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []model.ToolTimeUsage
	for rows.Next() {
		var item model.ToolTimeUsage
		var suspectedNetwork int
		if err := rows.Scan(
			&item.ToolName,
			&item.Calls,
			&item.SuccessCalls,
			&item.FailedCalls,
			&item.TotalDurationMS,
			&item.AvgDurationMS,
			&item.MaxDurationMS,
			&suspectedNetwork,
		); err != nil {
			return nil, err
		}
		item.SuspectedNetwork = suspectedNetwork != 0
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Service) agentTimeUsage(ctx context.Context) ([]model.AgentTimeUsage, error) {
	networkCondition := suspectedNetworkToolCondition("tc")
	rows, err := s.conn.QueryContext(ctx, fmt.Sprintf(`SELECT
		src.kind,
		src.name,
		COUNT(*),
		COALESCE(SUM((SELECT COUNT(*) FROM tool_calls tc WHERE tc.session_id = s.id)), 0),
		COALESCE(SUM(s.wall_duration_ms), 0),
		COALESCE(SUM(s.active_duration_ms), 0),
		COALESCE(SUM(s.model_duration_ms), 0),
		COALESCE(SUM(s.tool_duration_ms), 0),
		COALESCE(SUM(s.idle_duration_ms), 0),
		COALESCE(SUM((
			SELECT COALESCE(SUM(CASE WHEN tc.duration_ms > 0 THEN tc.duration_ms ELSE 0 END), 0)
			FROM tool_calls tc
			WHERE tc.session_id = s.id AND %s
		)), 0)
		FROM sessions s
		JOIN sources src ON src.id = s.source_id
		GROUP BY src.kind, src.name
		ORDER BY SUM(s.wall_duration_ms) DESC, src.name ASC`, networkCondition))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []model.AgentTimeUsage
	for rows.Next() {
		var item model.AgentTimeUsage
		if err := rows.Scan(
			&item.AgentKind,
			&item.AgentName,
			&item.SessionCount,
			&item.ToolCalls,
			&item.WallDurationMS,
			&item.ActiveDurationMS,
			&item.ModelDurationMS,
			&item.ToolDurationMS,
			&item.IdleDurationMS,
			&item.SuspectedNetworkToolDurationMS,
		); err != nil {
			return nil, err
		}
		if item.SuspectedNetworkToolDurationMS < 0 || item.ToolDurationMS < 0 {
			item.SuspectedNetworkToolDurationMS = 0
		} else if item.SuspectedNetworkToolDurationMS > item.ToolDurationMS {
			item.SuspectedNetworkToolDurationMS = item.ToolDurationMS
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Service) modelTimeUsage(ctx context.Context) ([]model.ModelTimeUsage, error) {
	rows, err := s.conn.QueryContext(ctx, `SELECT
		s.model,
		COUNT(*),
		COALESCE(SUM(tu.total_tokens), 0),
		COALESCE(SUM(s.wall_duration_ms), 0),
		COALESCE(SUM(s.active_duration_ms), 0),
		COALESCE(SUM(s.model_duration_ms), 0),
		COALESCE(SUM(s.tool_duration_ms), 0),
		COALESCE(SUM(s.idle_duration_ms), 0)
		FROM sessions s
		LEFT JOIN token_usage tu ON tu.owner_kind = 'session' AND tu.owner_id = s.id
		GROUP BY s.model
		ORDER BY SUM(s.wall_duration_ms) DESC, s.model ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []model.ModelTimeUsage
	for rows.Next() {
		var item model.ModelTimeUsage
		if err := rows.Scan(
			&item.Model,
			&item.SessionCount,
			&item.TotalTokens,
			&item.WallDurationMS,
			&item.ActiveDurationMS,
			&item.ModelDurationMS,
			&item.ToolDurationMS,
			&item.IdleDurationMS,
		); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Service) slowSessions(ctx context.Context) ([]model.Session, error) {
	return s.scanSessions(ctx, sessionSelect+` ORDER BY s.wall_duration_ms DESC, s.started_at DESC, s.id DESC LIMIT 8`)
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
