package tui

import (
	"fmt"

	agentmodel "github.com/LyleMi/AgentMeter/internal/model"
)

func newState(service appService, width, height int) *state {
	if width <= 0 {
		width = defaultWidth
	}
	if height <= 0 {
		height = defaultHeight
	}
	return &state{
		service:             service,
		page:                pageOverview,
		width:               width,
		height:              height,
		tokenBreakdownGroup: tokenBreakdownGlobal,
	}
}

func (s *state) init() command {
	return s.load(pageOverview)
}

func (s *state) update(msg message) (command, bool) {
	switch m := msg.(type) {
	case keyMsg:
		return s.handleKey(m)
	case loadMsg:
		if m.seq != s.loadSeq {
			return nil, false
		}
		s.loading = false
		s.err = m.err
		if m.err != nil {
			s.status = "load failed: " + m.err.Error()
			return nil, false
		}
		switch m.page {
		case pageOverview:
			s.overview = m.overview
			s.mergeScopeOptions(m.scopeOverview, m.scopeProjects)
		case pageTime:
			s.overview = m.overview
			s.mergeScopeOptions(m.scopeOverview, m.scopeProjects)
		case pageTokens:
			s.tokens = m.tokens
			s.breakdown = m.breakdown
			s.mergeScopeOptions(m.scopeOverview, m.scopeProjects)
		case pageModelSignals:
			s.signals = m.signals
			s.mergeScopeOptions(m.scopeOverview, m.scopeProjects)
		case pageModelRisk:
			s.signals = m.signals
			s.mergeScopeOptions(m.scopeOverview, m.scopeProjects)
		case pageSessions:
			s.sessions = m.sessions
			s.clampSelection(len(s.sessions))
		case pageSessionDetail:
			detail := m.detail
			s.detail = &detail
			s.scroll = 0
		case pageTools:
			s.tools = m.tools
			s.toolCalls = m.toolCalls
			s.clampSelection(len(s.tools))
			if s.toolsTab == toolsTabShell || s.toolsTab == toolsTabCalls {
				s.clampSelection(len(s.toolCalls))
			}
			s.mergeScopeOptions(m.scopeOverview, agentmodel.UsageBreakdown{})
		case pageToolCalls:
			s.toolCalls = m.toolCalls
			s.clampSelection(len(s.toolCalls))
		case pageToolCallDetail:
			s.toolCalls = m.toolCalls
			if m.toolCall != nil {
				call := *m.toolCall
				s.toolCall = &call
				s.scroll = 0
			}
		case pageAudit:
			s.audit = m.audit
			s.clampSelection(len(s.audit.RecentFindings))
			s.mergeScopeOptions(m.scopeOverview, agentmodel.UsageBreakdown{})
		case pageAuditFindings:
			s.findings = m.findings
			s.clampSelection(len(s.findings))
			s.mergeScopeOptions(m.scopeOverview, agentmodel.UsageBreakdown{})
		case pageAuditDetail:
			if m.finding != nil {
				finding := *m.finding
				s.finding = &finding
				s.scroll = 0
			}
			if m.auditSession != nil {
				session := *m.auditSession
				s.auditSession = &session
			} else {
				s.auditSession = nil
			}
		case pageSettings:
			s.settings = m.settings
		case pagePrivacy:
			s.privacy = m.privacy
			s.clampPrivacyTarget()
		}
	case indexMsg:
		s.indexing = false
		if m.err != nil {
			s.err = m.err
			s.status = "index failed: " + m.err.Error()
			return nil, false
		}
		s.err = nil
		result := m.result
		s.lastIndex = &result
		mode := "index"
		if m.rebuild {
			mode = "rebuild index"
		}
		s.status = fmt.Sprintf("%s complete: %d indexed, %d skipped, %d failed, %d sessions",
			mode, result.Indexed, result.Skipped, result.Failed, result.Sessions)
		return s.refresh(), false
	case privacyProfileMsg:
		s.privacyApplying = false
		s.privacyPending = nil
		if m.err != nil {
			s.err = m.err
			s.status = "privacy profile failed: " + m.err.Error()
			return nil, false
		}
		s.err = nil
		s.mergePrivacyStatus(m.result.Status, m.target)
		s.status = privacyApplyStatus(m.profile, m.targetName, m.result)
		return nil, false
	case resizeMsg:
		if m.width > 0 {
			s.width = m.width
		}
		if m.height > 0 {
			s.height = m.height
		}
		s.ensureVisible()
	}
	return nil, false
}
