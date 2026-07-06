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
		s.handleLoadMsg(m)
	case indexMsg:
		return s.handleIndexMsg(m), false
	case privacyProfileMsg:
		s.handlePrivacyProfileMsg(m)
	case resizeMsg:
		s.handleResizeMsg(m)
	}
	return nil, false
}

func (s *state) handleLoadMsg(m loadMsg) {
	if m.seq != s.loadSeq {
		return
	}
	s.loading = false
	s.err = m.err
	if m.err != nil {
		s.status = "load failed: " + m.err.Error()
		return
	}
	s.applyLoadMsg(m)
}

func (s *state) applyLoadMsg(m loadMsg) {
	switch m.page {
	case pageOverview, pageTime:
		s.overview = m.overview
		s.mergeScopeOptions(m.scopeOverview, m.scopeProjects)
	case pageTokens:
		s.applyTokenLoad(m)
	case pageModelSignals, pageModelRisk:
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
		s.applyToolsLoad(m)
	case pageToolCalls:
		s.toolCalls = m.toolCalls
		s.clampSelection(len(s.toolCalls))
	case pageToolCallDetail:
		s.applyToolCallDetailLoad(m)
	case pageAudit:
		s.audit = m.audit
		s.clampSelection(len(s.audit.RecentFindings))
		s.mergeScopeOptions(m.scopeOverview, agentmodel.UsageBreakdown{})
	case pageAuditFindings:
		s.findings = m.findings
		s.clampSelection(len(s.findings))
		s.mergeScopeOptions(m.scopeOverview, agentmodel.UsageBreakdown{})
	case pageAuditDetail:
		s.applyAuditDetailLoad(m)
	case pageSettings:
		s.settings = m.settings
	case pagePrivacy:
		s.privacy = m.privacy
		s.clampPrivacyTarget()
	}
}

func (s *state) applyTokenLoad(m loadMsg) {
	s.tokens = m.tokens
	s.breakdown = m.breakdown
	s.mergeScopeOptions(m.scopeOverview, m.scopeProjects)
}

func (s *state) applyToolsLoad(m loadMsg) {
	s.tools = m.tools
	s.toolCalls = m.toolCalls
	s.clampSelection(len(s.tools))
	if s.toolsTab == toolsTabShell || s.toolsTab == toolsTabCalls {
		s.clampSelection(len(s.toolCalls))
	}
	s.mergeScopeOptions(m.scopeOverview, agentmodel.UsageBreakdown{})
}

func (s *state) applyToolCallDetailLoad(m loadMsg) {
	s.toolCalls = m.toolCalls
	if m.toolCall == nil {
		return
	}
	call := *m.toolCall
	s.toolCall = &call
	s.scroll = 0
}

func (s *state) applyAuditDetailLoad(m loadMsg) {
	if m.finding != nil {
		finding := *m.finding
		s.finding = &finding
		s.scroll = 0
	}
	if m.auditSession != nil {
		session := *m.auditSession
		s.auditSession = &session
		return
	}
	s.auditSession = nil
}

func (s *state) handleIndexMsg(m indexMsg) command {
	s.indexing = false
	if m.err != nil {
		s.err = m.err
		s.status = "index failed: " + m.err.Error()
		return nil
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
	return s.refresh()
}

func (s *state) handlePrivacyProfileMsg(m privacyProfileMsg) {
	s.privacyApplying = false
	s.privacyPending = nil
	if m.err != nil {
		s.err = m.err
		s.status = "privacy profile failed: " + m.err.Error()
		return
	}
	s.err = nil
	s.mergePrivacyStatus(m.result.Status, m.target)
	s.status = privacyApplyStatus(m.profile, m.targetName, m.result)
}

func (s *state) handleResizeMsg(m resizeMsg) {
	if m.width > 0 {
		s.width = m.width
	}
	if m.height > 0 {
		s.height = m.height
	}
	s.ensureVisible()
}
