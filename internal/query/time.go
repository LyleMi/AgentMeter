package query

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/LyleMi/AgentMeter/internal/model"
)

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

func (s *Service) toolCallCount(ctx context.Context, filters model.AnalyticsFilters) (int, error) {
	where, args := analyticsSessionWhere(filters)
	var count int
	err := s.conn.QueryRowContext(ctx, `SELECT COUNT(*)
		FROM tool_calls tc
		JOIN sessions s ON s.id = tc.session_id
		JOIN sources src ON src.id = s.source_id
		LEFT JOIN token_usage tu ON tu.owner_kind = 'session' AND tu.owner_id = s.id
		WHERE `+whereClause(where), args...).Scan(&count)
	return count, err
}

func (s *Service) suspectedNetworkToolTotals(ctx context.Context, totalToolDurationMS int64) (int64, int, error) {
	return s.suspectedNetworkToolTotalsWithFilters(ctx, totalToolDurationMS, model.AnalyticsFilters{})
}

func (s *Service) suspectedNetworkToolTotalsWithFilters(ctx context.Context, totalToolDurationMS int64, filters model.AnalyticsFilters) (int64, int, error) {
	var duration int64
	var calls int
	where, args := analyticsSessionWhere(filters)
	where = append(where, suspectedNetworkToolCondition("tc"))
	query := fmt.Sprintf(`SELECT
		COALESCE(SUM(CASE WHEN tc.duration_ms > 0 THEN tc.duration_ms ELSE 0 END), 0),
		COUNT(*)
		FROM tool_calls tc
		JOIN sessions s ON s.id = tc.session_id
		JOIN sources src ON src.id = s.source_id
		LEFT JOIN token_usage tu ON tu.owner_kind = 'session' AND tu.owner_id = s.id
		WHERE %s`, whereClause(where))
	if err := s.conn.QueryRowContext(ctx, query, args...).Scan(&duration, &calls); err != nil {
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
	return s.toolTimeLeadersWithFilters(ctx, model.AnalyticsFilters{})
}

func (s *Service) toolTimeLeadersWithFilters(ctx context.Context, filters model.AnalyticsFilters) ([]model.ToolTimeUsage, error) {
	networkCondition := suspectedNetworkToolCondition("tc")
	where, args := analyticsSessionWhere(filters)
	query := fmt.Sprintf(`SELECT
		tc.tool_name,
		COUNT(*),
		SUM(CASE WHEN tc.status IN ('completed', 'success') THEN 1 ELSE 0 END),
		SUM(CASE WHEN tc.status IN ('completed', 'success') THEN 0 ELSE 1 END),
		COALESCE(SUM(tc.duration_ms), 0),
		COALESCE(AVG(tc.duration_ms), 0),
		COALESCE(MAX(tc.duration_ms), 0),
		MAX(CASE WHEN %s THEN 1 ELSE 0 END)
		FROM tool_calls tc
		JOIN sessions s ON s.id = tc.session_id
		JOIN sources src ON src.id = s.source_id
		LEFT JOIN token_usage tu ON tu.owner_kind = 'session' AND tu.owner_id = s.id
		WHERE %s
		GROUP BY tc.tool_name
		ORDER BY SUM(tc.duration_ms) DESC, tc.tool_name ASC
		LIMIT 8`, networkCondition, whereClause(where))
	return scanQueryRows(ctx, s.conn, query, scanToolTimeUsage, args...)
}

func (s *Service) agentTimeUsage(ctx context.Context) ([]model.AgentTimeUsage, error) {
	return s.agentTimeUsageWithFilters(ctx, model.AnalyticsFilters{})
}

func (s *Service) agentTimeUsageWithFilters(ctx context.Context, filters model.AnalyticsFilters) ([]model.AgentTimeUsage, error) {
	networkCondition := suspectedNetworkToolCondition("tc")
	where, args := analyticsSessionWhere(filters)
	query := fmt.Sprintf(`SELECT
		src.id,
		src.root_path,
		src.sessions_path,
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
		LEFT JOIN token_usage tu ON tu.owner_kind = 'session' AND tu.owner_id = s.id
		WHERE %s
		GROUP BY src.id
		ORDER BY SUM(s.wall_duration_ms) DESC, src.name ASC`, networkCondition, whereClause(where))
	return scanQueryRows(ctx, s.conn, query, scanAgentTimeUsage, args...)
}

func (s *Service) modelTimeUsage(ctx context.Context) ([]model.ModelTimeUsage, error) {
	return s.modelTimeUsageWithFilters(ctx, model.AnalyticsFilters{})
}

func (s *Service) modelTimeUsageWithFilters(ctx context.Context, filters model.AnalyticsFilters) ([]model.ModelTimeUsage, error) {
	where, args := analyticsSessionWhere(filters)
	query := `SELECT
		s.model,
		COUNT(*),
		COALESCE(SUM(tu.total_tokens), 0),
		COALESCE(SUM(s.wall_duration_ms), 0),
		COALESCE(SUM(s.active_duration_ms), 0),
		COALESCE(SUM(s.model_duration_ms), 0),
		COALESCE(SUM(s.tool_duration_ms), 0),
		COALESCE(SUM(s.idle_duration_ms), 0)
		FROM sessions s
		JOIN sources src ON src.id = s.source_id
		LEFT JOIN token_usage tu ON tu.owner_kind = 'session' AND tu.owner_id = s.id
		WHERE ` + whereClause(where) + `
		GROUP BY s.model
		ORDER BY SUM(s.wall_duration_ms) DESC, s.model ASC`
	return scanQueryRows(ctx, s.conn, query, scanModelTimeUsage, args...)
}

func scanToolTimeUsage(rows *sql.Rows) (model.ToolTimeUsage, error) {
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
		return model.ToolTimeUsage{}, err
	}
	item.SuspectedNetwork = suspectedNetwork != 0
	return item, nil
}

func scanAgentTimeUsage(rows *sql.Rows) (model.AgentTimeUsage, error) {
	var item model.AgentTimeUsage
	if err := rows.Scan(
		&item.SourceID,
		&item.SourceRootPath,
		&item.SourceSessionsPath,
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
		return model.AgentTimeUsage{}, err
	}
	item.SourceKey = sourceInstanceKey(item.SourceID)
	item.SourceLabel = item.AgentName
	if item.SuspectedNetworkToolDurationMS < 0 || item.ToolDurationMS < 0 {
		item.SuspectedNetworkToolDurationMS = 0
	} else if item.SuspectedNetworkToolDurationMS > item.ToolDurationMS {
		item.SuspectedNetworkToolDurationMS = item.ToolDurationMS
	}
	return item, nil
}

func scanModelTimeUsage(rows *sql.Rows) (model.ModelTimeUsage, error) {
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
		return model.ModelTimeUsage{}, err
	}
	return item, nil
}

func (s *Service) slowSessions(ctx context.Context) ([]model.Session, error) {
	return s.slowSessionsWithFilters(ctx, model.AnalyticsFilters{})
}

func (s *Service) slowSessionsWithFilters(ctx context.Context, filters model.AnalyticsFilters) ([]model.Session, error) {
	return s.analyticsSessions(ctx, filters, 8, "s.wall_duration_ms DESC, s.started_at DESC, s.id DESC", false)
}
