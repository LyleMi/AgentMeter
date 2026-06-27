package query

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"AgentMeter/internal/db"
	"AgentMeter/internal/model"
	"AgentMeter/internal/pricing"
)

type Service struct {
	conn *sql.DB
}

func New(conn *sql.DB) *Service {
	return &Service{conn: conn}
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
	limit := filters.Limit
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	offset := filters.Offset
	if offset < 0 {
		offset = 0
	}
	args = append(args, limit, offset)
	query := fmt.Sprintf(`SELECT
		s.id, s.source_id, s.source_file_id, src.kind, src.name, COALESCE(NULLIF(s.session_key, ''), s.codex_session_id), s.codex_session_id, s.project_path, s.model, s.model_provider, s.originator, s.thread_source,
		s.agent_nickname, s.agent_role, s.started_at, s.ended_at, s.wall_duration_ms, s.active_duration_ms, s.model_duration_ms,
		s.tool_duration_ms, s.idle_duration_ms, s.event_count, s.parse_status,
		COALESCE(tu.model, s.model), COALESCE(tu.input_tokens, 0), COALESCE(tu.cached_input_tokens, 0), COALESCE(tu.output_tokens, 0),
		COALESCE(tu.reasoning_output_tokens, 0), COALESCE(tu.total_tokens, 0), COALESCE(tu.source, 'unknown'),
		(SELECT COUNT(*) FROM tool_calls tc WHERE tc.session_id = s.id) AS tool_call_count,
		sf.path, sf.scan_status, sf.error
		FROM sessions s
		JOIN sources src ON src.id = s.source_id
		JOIN source_files sf ON sf.id = s.source_file_id
		LEFT JOIN token_usage tu ON tu.owner_kind = 'session' AND tu.owner_id = s.id
		WHERE %s
		ORDER BY s.started_at DESC
		LIMIT ? OFFSET ?`, strings.Join(where, " AND "))
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

func (s *Service) Tools(ctx context.Context) ([]model.ToolStat, error) {
	rows, err := s.conn.QueryContext(ctx, `SELECT
		tool_name,
		COUNT(*),
		SUM(CASE WHEN status IN ('completed', 'success') THEN 1 ELSE 0 END),
		SUM(CASE WHEN status IN ('completed', 'success') THEN 0 ELSE 1 END),
		COALESCE(SUM(duration_ms), 0),
		COALESCE(AVG(duration_ms), 0)
		FROM tool_calls
		GROUP BY tool_name
		ORDER BY COUNT(*) DESC, tool_name ASC`)
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
	limit := filters.Limit
	if limit <= 0 || limit > 1000 {
		limit = 500
	}
	offset := filters.Offset
	if offset < 0 {
		offset = 0
	}
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

func (s *Service) sessionByID(ctx context.Context, id int64) (model.Session, error) {
	query := `SELECT
		s.id, s.source_id, s.source_file_id, src.kind, src.name, COALESCE(NULLIF(s.session_key, ''), s.codex_session_id), s.codex_session_id, s.project_path, s.model, s.model_provider, s.originator, s.thread_source,
		s.agent_nickname, s.agent_role, s.started_at, s.ended_at, s.wall_duration_ms, s.active_duration_ms, s.model_duration_ms,
		s.tool_duration_ms, s.idle_duration_ms, s.event_count, s.parse_status,
		COALESCE(tu.model, s.model), COALESCE(tu.input_tokens, 0), COALESCE(tu.cached_input_tokens, 0), COALESCE(tu.output_tokens, 0),
		COALESCE(tu.reasoning_output_tokens, 0), COALESCE(tu.total_tokens, 0), COALESCE(tu.source, 'unknown'),
		(SELECT COUNT(*) FROM tool_calls tc WHERE tc.session_id = s.id) AS tool_call_count,
		sf.path, sf.scan_status, sf.error
		FROM sessions s
		JOIN sources src ON src.id = s.source_id
		JOIN source_files sf ON sf.id = s.source_file_id
		LEFT JOIN token_usage tu ON tu.owner_kind = 'session' AND tu.owner_id = s.id
		WHERE s.id = ?`
	sessions, err := s.scanSessions(ctx, query, id)
	if err != nil {
		return model.Session{}, err
	}
	if len(sessions) == 0 {
		return model.Session{}, sql.ErrNoRows
	}
	return sessions[0], nil
}

func (s *Service) events(ctx context.Context, sessionID int64) ([]model.Event, error) {
	rows, err := s.conn.QueryContext(ctx, `SELECT id, session_id, source_file_id, source_line, timestamp, kind, raw_type, summary, raw_json
		FROM events WHERE session_id = ? ORDER BY timestamp, source_line`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []model.Event
	for rows.Next() {
		var item model.Event
		var ts string
		if err := rows.Scan(&item.ID, &item.SessionID, &item.SourceFileID, &item.SourceLine, &ts, &item.Kind, &item.RawType, &item.Summary, &item.RawJSON); err != nil {
			return nil, err
		}
		item.Timestamp = db.ParseTime(ts)
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Service) modelCalls(ctx context.Context, sessionID int64) ([]model.ModelCall, error) {
	rows, err := s.conn.QueryContext(ctx, `SELECT id, session_id, started_at, ended_at, duration_ms, model, provider, status,
		input_tokens, cached_input_tokens, output_tokens, reasoning_output_tokens, total_tokens, cost_usd
		FROM model_calls WHERE session_id = ? ORDER BY started_at, id`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []model.ModelCall
	for rows.Next() {
		var item model.ModelCall
		var started, ended string
		var cost sql.NullFloat64
		if err := rows.Scan(&item.ID, &item.SessionID, &started, &ended, &item.DurationMS, &item.Model, &item.Provider, &item.Status,
			&item.InputTokens, &item.CachedInputTokens, &item.OutputTokens, &item.ReasoningOutputTokens, &item.TotalTokens, &cost); err != nil {
			return nil, err
		}
		item.StartedAt = db.ParseTime(started)
		item.EndedAt = db.ParseTime(ended)
		if cost.Valid {
			item.CostUSD = &cost.Float64
		} else if item.TotalTokens > 0 {
			item.Unpriced = true
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Service) toolCalls(ctx context.Context, sessionID int64) ([]model.ToolCall, error) {
	return s.scanToolCalls(ctx, toolCallSelect+` WHERE tc.session_id = ? ORDER BY tc.started_at, tc.id`, sessionID)
}

const toolCallSelect = `SELECT
		tc.id, tc.session_id, tc.started_at, tc.ended_at, tc.duration_ms, tc.tool_name, tc.status, tc.input_summary, tc.output_summary, tc.error,
		tc.raw_event_id, tc.call_id, tc.raw_start_event_id, tc.raw_end_event_id,
		COALESCE(start_event.source_line, 0), COALESCE(end_event.source_line, 0),
		COALESCE(start_event.raw_type, ''), COALESCE(end_event.raw_type, ''),
		COALESCE(start_event.summary, ''), COALESCE(end_event.summary, ''),
		COALESCE(start_event.raw_json, ''), COALESCE(end_event.raw_json, ''),
		COALESCE(NULLIF(sess.session_key, ''), sess.codex_session_id), sess.codex_session_id, sess.project_path,
		src.kind, src.name, sf.path
	FROM tool_calls tc
	JOIN sessions sess ON sess.id = tc.session_id
	JOIN sources src ON src.id = sess.source_id
	JOIN source_files sf ON sf.id = sess.source_file_id
	LEFT JOIN events start_event ON start_event.id = CASE WHEN tc.raw_start_event_id != 0 THEN tc.raw_start_event_id ELSE tc.raw_event_id END
	LEFT JOIN events end_event ON end_event.id = tc.raw_end_event_id`

func (s *Service) scanToolCalls(ctx context.Context, query string, args ...any) ([]model.ToolCall, error) {
	rows, err := s.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []model.ToolCall
	for rows.Next() {
		var item model.ToolCall
		var started, ended string
		if err := rows.Scan(
			&item.ID,
			&item.SessionID,
			&started,
			&ended,
			&item.DurationMS,
			&item.ToolName,
			&item.Status,
			&item.InputSummary,
			&item.OutputSummary,
			&item.Error,
			&item.RawEventID,
			&item.CallID,
			&item.RawStartEventID,
			&item.RawEndEventID,
			&item.RawStartEventLine,
			&item.RawEndEventLine,
			&item.RawStartEventType,
			&item.RawEndEventType,
			&item.RawStartEventSummary,
			&item.RawEndEventSummary,
			&item.RawStartEventJSON,
			&item.RawEndEventJSON,
			&item.SessionKey,
			&item.CodexSessionID,
			&item.ProjectPath,
			&item.AgentKind,
			&item.AgentName,
			&item.RawSourcePath,
		); err != nil {
			return nil, err
		}
		item.StartedAt = db.ParseTime(started)
		item.EndedAt = db.ParseTime(ended)
		item.RawEventLine = item.RawStartEventLine
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Service) scanSessions(ctx context.Context, query string, args ...any) ([]model.Session, error) {
	rows, err := s.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []model.Session
	for rows.Next() {
		var item model.Session
		var started, ended string
		if err := rows.Scan(
			&item.ID,
			&item.SourceID,
			&item.SourceFileID,
			&item.AgentKind,
			&item.AgentName,
			&item.SessionKey,
			&item.CodexSessionID,
			&item.ProjectPath,
			&item.Model,
			&item.ModelProvider,
			&item.Originator,
			&item.ThreadSource,
			&item.AgentNickname,
			&item.AgentRole,
			&started,
			&ended,
			&item.WallDurationMS,
			&item.ActiveDurationMS,
			&item.ModelDurationMS,
			&item.ToolDurationMS,
			&item.IdleDurationMS,
			&item.EventCount,
			&item.ParseStatus,
			&item.TokenUsage.Model,
			&item.TokenUsage.InputTokens,
			&item.TokenUsage.CachedInputTokens,
			&item.TokenUsage.OutputTokens,
			&item.TokenUsage.ReasoningOutputTokens,
			&item.TokenUsage.TotalTokens,
			&item.TokenUsage.Source,
			&item.ToolCallCount,
			&item.RawSourcePath,
			&item.LastIndexedScanStatus,
			&item.LastIndexedScanMessage,
		); err != nil {
			return nil, err
		}
		item.StartedAt = db.ParseTime(started)
		item.EndedAt = db.ParseTime(ended)
		if item.SessionKey == "" {
			item.SessionKey = item.CodexSessionID
		}
		if item.AgentName == "" {
			item.AgentName = item.AgentKind
		}
		cost, unpriced := pricing.Compute(s.conn, item.TokenUsage)
		item.TokenUsage.CostUSD = cost
		item.TokenUsage.Unpriced = unpriced
		item.EstimatedCostUSD = cost
		item.Unpriced = unpriced
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Service) totalCost(ctx context.Context) (*float64, int, error) {
	rows, err := s.conn.QueryContext(ctx, `SELECT model, input_tokens, cached_input_tokens, output_tokens, reasoning_output_tokens, total_tokens, source
		FROM token_usage WHERE owner_kind = 'session'`)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var total float64
	var hasCost bool
	var unpriced int
	for rows.Next() {
		var usage model.Usage
		if err := rows.Scan(&usage.Model, &usage.InputTokens, &usage.CachedInputTokens, &usage.OutputTokens, &usage.ReasoningOutputTokens, &usage.TotalTokens, &usage.Source); err != nil {
			return nil, 0, err
		}
		cost, isUnpriced := pricing.Compute(s.conn, usage)
		if isUnpriced {
			unpriced++
			continue
		}
		total += *cost
		hasCost = true
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if !hasCost {
		return nil, unpriced, nil
	}
	return &total, unpriced, nil
}

func (s *Service) dailyUsage(ctx context.Context) ([]model.DailyUsage, error) {
	rows, err := s.conn.QueryContext(ctx, `SELECT * FROM (
		SELECT substr(s.started_at, 1, 10) AS day,
			COUNT(*) AS session_count,
			COALESCE(SUM(tu.total_tokens), 0) AS total_tokens,
			COALESCE(SUM(tu.input_tokens), 0) AS input_tokens,
			COALESCE(SUM(tu.output_tokens), 0) AS output_tokens,
			COALESCE(SUM((SELECT COUNT(*) FROM tool_calls tc WHERE tc.session_id = s.id)), 0) AS tool_calls
		FROM sessions s
		LEFT JOIN token_usage tu ON tu.owner_kind = 'session' AND tu.owner_id = s.id
		GROUP BY day
		ORDER BY day DESC
		LIMIT 30
	) ORDER BY day ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []model.DailyUsage
	for rows.Next() {
		var item model.DailyUsage
		if err := rows.Scan(&item.Date, &item.SessionCount, &item.TotalTokens, &item.InputTokens, &item.OutputTokens, &item.ToolCalls); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	costs, err := s.dailyCosts(ctx)
	if err != nil {
		return nil, err
	}
	for index := range result {
		if cost, ok := costs[result[index].Date]; ok {
			result[index].EstimatedCostUSD = &cost
		}
	}
	return result, nil
}

func (s *Service) dailyCosts(ctx context.Context) (map[string]float64, error) {
	rows, err := s.conn.QueryContext(ctx, `SELECT substr(s.started_at, 1, 10) AS day, tu.model,
		COALESCE(SUM(tu.input_tokens), 0), COALESCE(SUM(tu.cached_input_tokens), 0), COALESCE(SUM(tu.output_tokens), 0),
		COALESCE(SUM(tu.reasoning_output_tokens), 0), COALESCE(SUM(tu.total_tokens), 0), COALESCE(MAX(tu.source), 'unknown')
		FROM sessions s
		JOIN token_usage tu ON tu.owner_kind = 'session' AND tu.owner_id = s.id
		GROUP BY day, tu.model`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]float64{}
	for rows.Next() {
		var day string
		var usage model.Usage
		if err := rows.Scan(&day, &usage.Model, &usage.InputTokens, &usage.CachedInputTokens, &usage.OutputTokens, &usage.ReasoningOutputTokens, &usage.TotalTokens, &usage.Source); err != nil {
			return nil, err
		}
		if cost, unpriced := pricing.Compute(s.conn, usage); !unpriced {
			result[day] += *cost
		}
	}
	return result, rows.Err()
}

func (s *Service) modelUsage(ctx context.Context) ([]model.ModelUsage, error) {
	rows, err := s.conn.QueryContext(ctx, `SELECT tu.model, COUNT(*),
		COALESCE(SUM(tu.total_tokens), 0), COALESCE(SUM(tu.input_tokens), 0), COALESCE(SUM(tu.output_tokens), 0),
		COALESCE(SUM(tu.cached_input_tokens), 0), COALESCE(SUM(tu.reasoning_output_tokens), 0), COALESCE(MAX(tu.source), 'unknown')
		FROM token_usage tu
		WHERE tu.owner_kind = 'session'
		GROUP BY tu.model
		ORDER BY SUM(tu.total_tokens) DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []model.ModelUsage
	for rows.Next() {
		var item model.ModelUsage
		var cached, reasoning int64
		var source string
		if err := rows.Scan(&item.Model, &item.SessionCount, &item.TotalTokens, &item.InputTokens, &item.OutputTokens, &cached, &reasoning, &source); err != nil {
			return nil, err
		}
		usage := model.Usage{
			Model:                 item.Model,
			InputTokens:           item.InputTokens,
			CachedInputTokens:     cached,
			OutputTokens:          item.OutputTokens,
			ReasoningOutputTokens: reasoning,
			TotalTokens:           item.TotalTokens,
			Source:                source,
		}
		item.EstimatedCostUSD, item.Unpriced = pricing.Compute(s.conn, usage)
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Service) agentUsage(ctx context.Context) ([]model.AgentUsage, error) {
	rows, err := s.conn.QueryContext(ctx, `SELECT src.kind, src.name, COUNT(*),
		COALESCE(SUM(tu.total_tokens), 0), COALESCE(SUM(tu.input_tokens), 0), COALESCE(SUM(tu.output_tokens), 0),
		COALESCE(SUM((SELECT COUNT(*) FROM tool_calls tc WHERE tc.session_id = s.id)), 0)
		FROM sessions s
		JOIN sources src ON src.id = s.source_id
		LEFT JOIN token_usage tu ON tu.owner_kind = 'session' AND tu.owner_id = s.id
		GROUP BY src.kind, src.name
		ORDER BY COUNT(*) DESC, src.name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []model.AgentUsage
	for rows.Next() {
		var item model.AgentUsage
		if err := rows.Scan(&item.AgentKind, &item.AgentName, &item.SessionCount, &item.TotalTokens, &item.InputTokens, &item.OutputTokens, &item.ToolCalls); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	costs, unpriced, err := s.agentCosts(ctx)
	if err != nil {
		return nil, err
	}
	for index := range result {
		key := sourceKey(result[index].AgentKind, result[index].AgentName)
		if cost, ok := costs[key]; ok {
			result[index].EstimatedCostUSD = &cost
		}
		result[index].Unpriced = unpriced[key]
	}
	return result, nil
}

func (s *Service) agentCosts(ctx context.Context) (map[string]float64, map[string]bool, error) {
	rows, err := s.conn.QueryContext(ctx, `SELECT src.kind, src.name, tu.model,
		COALESCE(SUM(tu.input_tokens), 0), COALESCE(SUM(tu.cached_input_tokens), 0), COALESCE(SUM(tu.output_tokens), 0),
		COALESCE(SUM(tu.reasoning_output_tokens), 0), COALESCE(SUM(tu.total_tokens), 0), COALESCE(MAX(tu.source), 'unknown')
		FROM sessions s
		JOIN sources src ON src.id = s.source_id
		JOIN token_usage tu ON tu.owner_kind = 'session' AND tu.owner_id = s.id
		GROUP BY src.kind, src.name, tu.model`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	costs := map[string]float64{}
	unpriced := map[string]bool{}
	for rows.Next() {
		var kind, name string
		var usage model.Usage
		if err := rows.Scan(&kind, &name, &usage.Model, &usage.InputTokens, &usage.CachedInputTokens, &usage.OutputTokens, &usage.ReasoningOutputTokens, &usage.TotalTokens, &usage.Source); err != nil {
			return nil, nil, err
		}
		key := sourceKey(kind, name)
		if cost, isUnpriced := pricing.Compute(s.conn, usage); isUnpriced {
			unpriced[key] = true
		} else {
			costs[key] += *cost
		}
	}
	return costs, unpriced, rows.Err()
}

func sourceKey(kind, name string) string {
	return kind + "\x00" + name
}
