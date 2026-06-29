package query

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/LyleMi/AgentMeter/internal/model"
)

func (s *Service) AuditSummary(ctx context.Context) (model.AuditSummary, error) {
	return s.AuditSummaryWithFilters(ctx, model.AuditFindingFilters{})
}

func (s *Service) AuditSummaryWithFilters(ctx context.Context, filters model.AuditFindingFilters) (model.AuditSummary, error) {
	var summary model.AuditSummary
	where := []string{"1 = 1"}
	args := []any{}
	where, args = appendSourceFilter(where, args, filters.Agent)
	query := fmt.Sprintf(`SELECT
		COUNT(*),
		COALESCE(SUM(CASE WHEN af.severity = 'critical' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN af.severity = 'high' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN af.severity = 'medium' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN af.severity = 'low' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN af.category = 'command' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN af.category = 'privacy' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN af.category = 'egress' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN af.category = 'file' THEN 1 ELSE 0 END), 0),
		COUNT(DISTINCT af.session_id)
		FROM audit_findings af
		JOIN sessions sess ON sess.id = af.session_id
		JOIN sources src ON src.id = sess.source_id
		WHERE %s`, strings.Join(where, " AND "))
	err := s.conn.QueryRowContext(ctx, query, args...).Scan(
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
	summary.RecentFindings, err = s.AuditFindings(ctx, model.AuditFindingFilters{Agent: filters.Agent, Limit: 8})
	summary.RecentFindings = nonNilSlice(summary.RecentFindings)
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
	where, args = appendSourceFilter(where, args, filters.Agent)
	if strings.TrimSpace(filters.Search) != "" {
		search := "%" + strings.TrimSpace(filters.Search) + "%"
		where = append(where, `(af.title LIKE ? OR af.description LIKE ? OR af.evidence LIKE ? OR af.command LIKE ? OR af.rule_id LIKE ? OR sess.session_key LIKE ? OR sess.project_path LIKE ? OR sf.path LIKE ? OR src.root_path LIKE ? OR src.sessions_path LIKE ?)`)
		args = append(args, search, search, search, search, search, search, search, search, search, search)
	}
	limit, offset := clampLimitOffset(filters.Limit, filters.Offset, 500, 1000)
	args = append(args, limit, offset)
	query := fmt.Sprintf(`%s
		WHERE %s
		ORDER BY af.timestamp DESC, af.id DESC
		LIMIT ? OFFSET ?`, auditFindingSelect, strings.Join(where, " AND "))
	return s.scanAuditFindings(ctx, query, args...)
}

func (s *Service) AuditFinding(ctx context.Context, id int64) (model.AuditFinding, error) {
	findings, err := s.scanAuditFindings(ctx, auditFindingSelect+` WHERE af.id = ?`, id)
	if err != nil {
		return model.AuditFinding{}, err
	}
	if len(findings) == 0 {
		return model.AuditFinding{}, sql.ErrNoRows
	}
	return findings[0], nil
}
