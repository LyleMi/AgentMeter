package tui

import (
	"context"
	"fmt"
	"strings"

	agentmodel "github.com/LyleMi/AgentMeter/internal/model"
)

func (s *state) load(target page) command {
	s.loadSeq++
	seq := s.loadSeq
	s.loading = true
	s.err = nil
	analyticsFilters := s.analyticsFilters()
	auditSummaryFilters := s.auditSummaryFilters()
	auditFilters := s.auditFilters()
	toolCallFilters := s.toolCallFilters()
	toolExplorerFilters := s.toolExplorerFilters()
	toolFilters := agentmodel.ToolFilters{Agent: strings.TrimSpace(s.toolAgent)}
	toolsTab := s.toolsTab
	toolCommand := s.toolCommand
	breakdownGroup := s.tokenBreakdownGroup
	return func(ctx context.Context, ch chan<- message) {
		msg := loadMsg{seq: seq, page: target}
		switch target {
		case pageOverview:
			msg.overview, msg.err = s.service.GetOverviewWithFilters(analyticsFilters)
			if msg.err == nil {
				msg.scopeOverview, msg.scopeProjects = s.loadUsageScopeOptions(analyticsFilters, msg.overview)
			}
		case pageTime:
			msg.overview, msg.err = s.service.GetOverviewWithFilters(analyticsFilters)
			if msg.err == nil {
				msg.scopeOverview, msg.scopeProjects = s.loadUsageScopeOptions(analyticsFilters, msg.overview)
			}
		case pageTokens:
			msg.tokens, msg.err = s.service.GetTokenAnalyticsWithFilters(analyticsFilters)
			if msg.err == nil && breakdownGroup != tokenBreakdownGlobal {
				msg.breakdown, msg.err = s.service.GetUsageBreakdown(breakdownGroup, analyticsFilters)
			}
			if msg.err == nil {
				msg.scopeOverview, msg.scopeProjects = s.loadUsageScopeOptions(analyticsFilters, agentmodel.Overview{})
			}
		case pageSessions:
			msg.sessions, msg.err = s.service.ListSessions(agentmodel.SessionFilters{Limit: 200})
		case pageTools:
			msg.tools, msg.err = s.service.ListTools(toolFilters)
			if msg.err == nil {
				if value, err := s.service.GetOverview(); err == nil {
					msg.scopeOverview = value
				}
			}
			if msg.err == nil && (toolsTab == toolsTabShell || toolsTab == toolsTabCalls) {
				msg.toolCalls, msg.err = listToolCallsForToolsContext(s.service, toolExplorerFilters, msg.tools, toolsTab, toolCommand)
			}
		case pageToolCalls:
			msg.toolCalls, msg.err = s.service.ListToolCalls(toolCallFilters)
		case pageModelSignals:
			msg.signals, msg.err = s.service.GetModelSignalsWithFilters(analyticsFilters)
			if msg.err == nil {
				msg.scopeOverview, msg.scopeProjects = s.loadUsageScopeOptions(analyticsFilters, agentmodel.Overview{})
			}
		case pageModelRisk:
			msg.signals, msg.err = s.service.GetModelSignalsWithFilters(analyticsFilters)
			if msg.err == nil {
				msg.scopeOverview, msg.scopeProjects = s.loadUsageScopeOptions(analyticsFilters, agentmodel.Overview{})
			}
		case pageAudit:
			msg.audit, msg.err = s.service.GetAuditSummaryWithFilters(auditSummaryFilters)
			if msg.err == nil {
				if value, err := s.service.GetOverview(); err == nil {
					msg.scopeOverview = value
				}
			}
		case pageAuditFindings:
			msg.findings, msg.err = s.service.ListAuditFindings(auditFilters)
			if msg.err == nil {
				if value, err := s.service.GetOverview(); err == nil {
					msg.scopeOverview = value
				}
			}
		case pageSettings:
			msg.settings, msg.err = s.service.GetSettings()
		case pagePrivacy:
			msg.privacy, msg.err = s.service.GetPrivacyConfigs()
		default:
			msg.err = fmt.Errorf("unsupported page: %s", target.title())
		}
		sendMessage(ctx, ch, msg)
	}
}

func (s *state) refresh() command {
	switch s.page {
	case pageSessionDetail:
		if s.detail == nil {
			return nil
		}
		return s.loadDetail(s.detail.Session.ID)
	case pageToolCallDetail:
		if s.toolCall == nil {
			return nil
		}
		return s.loadToolCallDetail(s.toolCall.ID)
	case pageAuditDetail:
		if s.finding == nil {
			return nil
		}
		return s.loadAuditDetail(s.finding.ID)
	default:
		return s.load(s.page)
	}
}

func (s *state) loadToolCallDetail(id int64) command {
	s.loadSeq++
	seq := s.loadSeq
	s.loading = true
	s.err = nil
	toolCallFilters := s.toolCallFilters()
	toolExplorerFilters := s.toolExplorerFilters()
	toolsTab := s.toolsTab
	toolCommand := s.toolCommand
	tools := append([]agentmodel.ToolStat(nil), s.tools...)
	previous := s.previous
	return func(ctx context.Context, ch chan<- message) {
		var calls []agentmodel.ToolCall
		var err error
		if previous == pageTools && (toolsTab == toolsTabShell || toolsTab == toolsTabCalls) {
			calls, err = listToolCallsForToolsContext(s.service, toolExplorerFilters, tools, toolsTab, toolCommand)
		} else {
			calls, err = s.service.ListToolCalls(toolCallFilters)
		}
		msg := loadMsg{seq: seq, page: pageToolCallDetail, toolCalls: calls, err: err}
		if err == nil {
			for i := range calls {
				if calls[i].ID == id {
					call := calls[i]
					msg.toolCall = &call
					break
				}
			}
			if msg.toolCall == nil {
				msg.err = fmt.Errorf("tool call %d not found", id)
			}
		}
		sendMessage(ctx, ch, msg)
	}
}

func (s *state) loadDetail(id int64) command {
	s.loadSeq++
	seq := s.loadSeq
	s.loading = true
	s.err = nil
	return func(ctx context.Context, ch chan<- message) {
		detail, err := s.service.GetSessionDetail(id)
		sendMessage(ctx, ch, loadMsg{
			seq:    seq,
			page:   pageSessionDetail,
			detail: detail,
			err:    err,
		})
	}
}

func (s *state) loadAuditDetail(id int64) command {
	s.loadSeq++
	seq := s.loadSeq
	s.loading = true
	s.err = nil
	auditAgent := s.auditAgent
	return func(ctx context.Context, ch chan<- message) {
		finding, err := s.service.GetAuditFinding(id)
		msg := loadMsg{seq: seq, page: pageAuditDetail, err: err}
		if err == nil {
			if strings.TrimSpace(auditAgent) != "" && !auditFindingMatchesAgent(finding, auditAgent) {
				msg.err = fmt.Errorf("audit finding %d does not match source filter %q", id, auditAgent)
				sendMessage(ctx, ch, msg)
				return
			}
			msg.finding = &finding
			if finding.SessionID > 0 {
				detail, detailErr := s.service.GetSessionDetail(finding.SessionID)
				if detailErr == nil {
					msg.auditSession = &detail
				}
			}
		}
		sendMessage(ctx, ch, msg)
	}
}

func (s *state) index(rebuild bool) command {
	if s.indexing {
		s.status = "index already running"
		return nil
	}
	s.indexing = true
	if rebuild {
		s.status = "rebuilding index..."
	} else {
		s.status = "updating index..."
	}
	return func(ctx context.Context, ch chan<- message) {
		result, err := s.service.IndexNow(rebuild)
		sendMessage(ctx, ch, indexMsg{result: result, rebuild: rebuild, err: err})
	}
}

func sendMessage(ctx context.Context, ch chan<- message, msg message) {
	select {
	case <-ctx.Done():
	case ch <- msg:
	}
}
