package query

import (
	"context"
	"database/sql"
	"strconv"

	"github.com/LyleMi/AgentMeter/internal/db"
	"github.com/LyleMi/AgentMeter/internal/model"
)

func sourceInstanceKey(id int64) string {
	if id <= 0 {
		return ""
	}
	return "source:" + strconv.FormatInt(id, 10)
}

func sourceIdentity(sourceID int64, agentName, agentKind string) (string, string) {
	label := agentName
	if label == "" {
		label = agentKind
	}
	return sourceInstanceKey(sourceID), label
}

func fillSessionSourceIdentity(item *model.Session) {
	item.SourceKey, item.SourceLabel = sourceIdentity(item.SourceID, item.AgentName, item.AgentKind)
}

func fillToolCallSourceIdentity(item *model.ToolCall) {
	item.SourceKey, item.SourceLabel = sourceIdentity(item.SourceID, item.AgentName, item.AgentKind)
}

func fillAuditFindingSourceIdentity(item *model.AuditFinding) {
	item.SourceKey, item.SourceLabel = sourceIdentity(item.SourceID, item.AgentName, item.AgentKind)
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
	calculator := s.pricingCalculator(ctx)
	rows, err := s.conn.QueryContext(ctx, `SELECT id, session_id, started_at, ended_at, duration_ms, model, provider, status,
		input_tokens, cached_input_tokens, output_tokens, reasoning_output_tokens, context_compression_tokens, total_tokens, cost_usd
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
			&item.InputTokens, &item.CachedInputTokens, &item.OutputTokens, &item.ReasoningOutputTokens, &item.ContextCompressionTokens, &item.TotalTokens, &cost); err != nil {
			return nil, err
		}
		item.StartedAt = db.ParseTime(started)
		item.EndedAt = db.ParseTime(ended)
		currentCost, unpriced := calculator.Compute(model.Usage{
			Model:                    item.Model,
			InputTokens:              item.InputTokens,
			CachedInputTokens:        item.CachedInputTokens,
			OutputTokens:             item.OutputTokens,
			ReasoningOutputTokens:    item.ReasoningOutputTokens,
			ContextCompressionTokens: item.ContextCompressionTokens,
			TotalTokens:              item.TotalTokens,
			Source:                   "model_call",
		})
		if currentCost != nil || unpriced {
			item.CostUSD = currentCost
			item.Unpriced = unpriced
		} else if cost.Valid {
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

func (s *Service) scanToolCalls(ctx context.Context, query string, args ...any) ([]model.ToolCall, error) {
	return scanQueryRows(ctx, s.conn, query, scanToolCall, args...)
}

func scanQueryRows[T any](ctx context.Context, conn *sql.DB, query string, scan func(*sql.Rows) (T, error), args ...any) ([]T, error) {
	rows, err := conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []T
	for rows.Next() {
		item, err := scan(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Service) scanToolCallsWithRisk(ctx context.Context, query string, args ...any) ([]model.ToolCall, error) {
	return scanQueryRows(ctx, s.conn, query, scanToolCallWithRisk, args...)
}

func (s *Service) scanAuditFindings(ctx context.Context, query string, args ...any) ([]model.AuditFinding, error) {
	return scanQueryRows(ctx, s.conn, query, scanAuditFinding, args...)
}

func scanToolCall(rows *sql.Rows) (model.ToolCall, error) {
	var item model.ToolCall
	var started, ended string
	if err := rows.Scan(
		&item.ID,
		&item.SessionID,
		&item.SourceID,
		&item.SourceRootPath,
		&item.SourceSessionsPath,
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
		return model.ToolCall{}, err
	}
	prepareScannedToolCall(&item, started, ended)
	return item, nil
}

func scanToolCallWithRisk(rows *sql.Rows) (model.ToolCall, error) {
	var item model.ToolCall
	var started, ended, ruleIDs string
	if err := rows.Scan(
		&item.ID,
		&item.SessionID,
		&item.SourceID,
		&item.SourceRootPath,
		&item.SourceSessionsPath,
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
		&item.RiskScore,
		&item.RiskSeverity,
		&item.RiskCount,
		&ruleIDs,
	); err != nil {
		return model.ToolCall{}, err
	}
	prepareScannedToolCall(&item, started, ended)
	item.RiskRuleIDs = splitSortedCSV(ruleIDs)
	return item, nil
}

func prepareScannedToolCall(item *model.ToolCall, started, ended string) {
	item.StartedAt = db.ParseTime(started)
	item.EndedAt = db.ParseTime(ended)
	fillToolCallSourceIdentity(item)
	item.RawEventLine = item.RawStartEventLine
}

func scanAuditFinding(rows *sql.Rows) (model.AuditFinding, error) {
	var item model.AuditFinding
	var ts, created string
	if err := rows.Scan(
		&item.ID,
		&item.SessionID,
		&item.SourceID,
		&item.SourceRootPath,
		&item.SourceSessionsPath,
		&item.ToolCallID,
		&item.SourceFileID,
		&item.RawEventID,
		&item.SourceLine,
		&ts,
		&item.Source,
		&item.EventType,
		&item.Category,
		&item.Severity,
		&item.RuleID,
		&item.Title,
		&item.Description,
		&item.Evidence,
		&item.Command,
		&item.ShellFamily,
		&item.Platform,
		&item.Decision,
		&created,
		&item.SessionKey,
		&item.CodexSessionID,
		&item.ProjectPath,
		&item.AgentKind,
		&item.AgentName,
		&item.RawSourcePath,
	); err != nil {
		return model.AuditFinding{}, err
	}
	item.Timestamp = db.ParseTime(ts)
	item.CreatedAt = db.ParseTime(created)
	if item.AgentName == "" {
		item.AgentName = item.AgentKind
	}
	fillAuditFindingSourceIdentity(&item)
	return item, nil
}

func (s *Service) scanSessions(ctx context.Context, query string, args ...any) ([]model.Session, error) {
	calculator := s.pricingCalculator(ctx)

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
			&item.SourceRootPath,
			&item.SourceSessionsPath,
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
			&item.TokenUsage.ContextCompressionTokens,
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
		fillSessionSourceIdentity(&item)
		cost, unpriced := calculator.Compute(item.TokenUsage)
		item.TokenUsage.CostUSD = cost
		item.TokenUsage.Unpriced = unpriced
		item.EstimatedCostUSD = cost
		item.Unpriced = unpriced
		result = append(result, item)
	}
	return result, rows.Err()
}
