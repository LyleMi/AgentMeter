package query

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/LyleMi/AgentMeter/internal/model"
)

func (s *Service) Sessions(ctx context.Context, filters model.SessionFilters) ([]model.Session, error) {
	where := []string{"1 = 1"}
	args := []any{}
	if strings.TrimSpace(filters.Search) != "" {
		search := "%" + strings.TrimSpace(filters.Search) + "%"
		where = append(where, `(s.session_key LIKE ? OR s.codex_session_id LIKE ? OR s.project_path LIKE ? OR s.model LIKE ? OR sf.path LIKE ? OR src.kind LIKE ? OR src.name LIKE ? OR src.root_path LIKE ? OR src.sessions_path LIKE ?)`)
		args = append(args, search, search, search, search, search, search, search, search, search)
	}
	if strings.TrimSpace(filters.Model) != "" {
		where = append(where, `s.model = ?`)
		args = append(args, strings.TrimSpace(filters.Model))
	}
	where, args = appendSourceFilter(where, args, filters.Agent)
	limit, offset := clampLimitOffset(filters.Limit, filters.Offset, 200, 500)
	args = append(args, limit, offset)
	query := fmt.Sprintf(`%s
		WHERE %s
		ORDER BY s.started_at DESC
		LIMIT ? OFFSET ?`, sessionSelect, whereClause(where))
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
