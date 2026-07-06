package query

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/LyleMi/AgentMeter/internal/model"
)

func (s *Service) Tools(ctx context.Context, filters model.ToolFilters) ([]model.ToolStat, error) {
	where := []string{"1 = 1"}
	args := []any{}
	where, args = appendSourceFilter(where, args, filters.Agent)
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
		ORDER BY COUNT(*) DESC, tc.tool_name ASC`, whereClause(where))
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
	if filters.Shell {
		where = append(where, shellToolSQLPredicate("tc.tool_name"))
	}
	where, args = appendSourceFilter(where, args, filters.Agent)
	if strings.TrimSpace(filters.StartedFrom) != "" {
		where = append(where, "tc.started_at >= ?")
		args = append(args, strings.TrimSpace(filters.StartedFrom))
	}
	if strings.TrimSpace(filters.StartedTo) != "" {
		where = append(where, "tc.started_at <= ?")
		args = append(args, strings.TrimSpace(filters.StartedTo))
	}
	if filters.RiskOnly {
		where = append(where, "COALESCE(risk.risk_count, 0) > 0")
	}
	limit, offset := clampLimitOffset(filters.Limit, filters.Offset, 500, 1000)
	orderBy := "tc.started_at DESC, tc.id DESC"
	switch strings.TrimSpace(filters.Sort) {
	case "duration_desc":
		orderBy = "tc.duration_ms DESC, tc.started_at DESC, tc.id DESC"
	case "duration_asc":
		orderBy = "tc.duration_ms ASC, tc.started_at DESC, tc.id DESC"
	case "risk_desc":
		orderBy = "CASE WHEN COALESCE(risk.risk_count, 0) > 0 THEN risk.risk_score ELSE 1 END DESC, tc.started_at DESC, tc.id DESC"
	case "risk_asc":
		orderBy = "CASE WHEN COALESCE(risk.risk_count, 0) > 0 THEN risk.risk_score ELSE 1 END ASC, tc.started_at DESC, tc.id DESC"
	}
	args = append(args, limit, offset)
	if filters.IncludeRisk || filters.Shell || filters.RiskOnly || strings.HasPrefix(strings.TrimSpace(filters.Sort), "risk_") {
		query := fmt.Sprintf(`%s
		WHERE %s
		ORDER BY %s
		LIMIT ? OFFSET ?`, toolCallWithRiskSelect, whereClause(where), orderBy)
		return s.scanToolCallsWithRisk(ctx, query, args...)
	}
	query := fmt.Sprintf(`%s
		WHERE %s
		ORDER BY %s
		LIMIT ? OFFSET ?`, toolCallSelect, whereClause(where), orderBy)
	return s.scanToolCalls(ctx, query, args...)
}

func (s *Service) ToolCallRisks(ctx context.Context, filters model.ToolCallRiskFilters) ([]model.ToolCallRiskSummary, error) {
	where := []string{"af.tool_call_id > 0"}
	args := []any{}
	where, args = appendSourceFilter(where, args, filters.Agent)
	if strings.TrimSpace(filters.StartedFrom) != "" {
		where = append(where, "tc.started_at >= ?")
		args = append(args, strings.TrimSpace(filters.StartedFrom))
	}
	if strings.TrimSpace(filters.StartedTo) != "" {
		where = append(where, "tc.started_at <= ?")
		args = append(args, strings.TrimSpace(filters.StartedTo))
	}
	limit, _ := clampLimitOffset(filters.Limit, 0, 500, 1000)
	args = append(args, limit)
	query := fmt.Sprintf(`SELECT
		af.tool_call_id,
		CASE MAX(CASE af.severity
			WHEN 'critical' THEN 4
			WHEN 'high' THEN 3
			WHEN 'medium' THEN 2
			WHEN 'low' THEN 1
			ELSE 0
		END)
			WHEN 4 THEN 'critical'
			WHEN 3 THEN 'high'
			WHEN 2 THEN 'medium'
			WHEN 1 THEN 'low'
			ELSE ''
		END AS severity,
		MIN(100, CASE MAX(CASE af.severity
			WHEN 'critical' THEN 4
			WHEN 'high' THEN 3
			WHEN 'medium' THEN 2
			WHEN 'low' THEN 1
			ELSE 0
		END)
			WHEN 4 THEN 90
			WHEN 3 THEN 70
			WHEN 2 THEN 45
			WHEN 1 THEN 20
			ELSE 0
		END + ((COUNT(DISTINCT af.rule_id) - 1) * 5)) AS risk_score,
		COUNT(*) AS risk_count,
		GROUP_CONCAT(DISTINCT af.rule_id) AS rule_ids,
		MAX(tc.started_at) AS latest_started_at
		FROM audit_findings af
		JOIN tool_calls tc ON tc.id = af.tool_call_id
		JOIN sessions sess ON sess.id = af.session_id
		JOIN sources src ON src.id = sess.source_id
		WHERE %s
		GROUP BY af.tool_call_id
		ORDER BY latest_started_at DESC, af.tool_call_id DESC
		LIMIT ?`, whereClause(where))
	rows, err := s.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.ToolCallRiskSummary
	for rows.Next() {
		var item model.ToolCallRiskSummary
		var ruleIDs string
		var latestStartedAt string
		if err := rows.Scan(&item.ToolCallID, &item.Severity, &item.RiskScore, &item.RiskCount, &ruleIDs, &latestStartedAt); err != nil {
			return nil, err
		}
		item.RuleIDs = splitSortedCSV(ruleIDs)
		result = append(result, item)
	}
	return result, rows.Err()
}

func shellToolSQLPredicate(column string) string {
	lower := "LOWER(TRIM(" + column + "))"
	return "(" + lower + " IN ('shell_command', 'bash', 'zsh', 'sh', 'powershell', 'powershell.exe', 'pwsh', 'pwsh.exe', 'cmd', 'cmd.exe', 'shell', 'terminal') OR " + lower + " LIKE '%.shell_command' OR " + lower + " LIKE '%shell_command%')"
}

func splitSortedCSV(value string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, raw := range strings.Split(value, ",") {
		item := strings.TrimSpace(raw)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		result = append(result, item)
	}
	sort.Strings(result)
	return result
}
