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
	where, args = appendSourceFilter(where, args, filters.Agent)
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
		LIMIT ?`, strings.Join(where, " AND "))
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
		if err := rows.Scan(&item.ToolCallID, &item.Severity, &item.RiskCount, &ruleIDs, &latestStartedAt); err != nil {
			return nil, err
		}
		item.RuleIDs = splitSortedCSV(ruleIDs)
		result = append(result, item)
	}
	return result, rows.Err()
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
