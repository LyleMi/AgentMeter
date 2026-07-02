package tui

func (s *state) handleKey(k keyMsg) (command, bool) {
	if k.typ == keyCtrlC {
		return nil, true
	}
	if s.page == pagePrivacy {
		if cmd, quit, handled := s.handlePrivacyKey(k); handled {
			return cmd, quit
		}
	}
	if k.typ == keyRune {
		switch k.ch {
		case 'q', 'Q':
			return nil, true
		case '1', 'o', 'O':
			return s.switchPage(pageOverview), false
		case '2':
			return s.switchPage(pageTime), false
		case '3', 'n', 'N':
			return s.switchPage(pageTokens), false
		case '4', 'm', 'M':
			return s.switchPage(pageModelSignals), false
		case '5', 'x', 'X':
			return s.switchPage(pageModelRisk), false
		case '6', 's', 'S':
			return s.switchPage(pageSessions), false
		case '7', 't', 'T':
			return s.switchPage(pageTools), false
		case '8', 'a':
			return s.switchPage(pageAudit), false
		case '9', 'p', 'P':
			return s.switchPage(pagePrivacy), false
		case '0', 'g', 'G':
			return s.switchPage(pageSettings), false
		case 'r', 'R':
			return s.refresh(), false
		case 'c', 'C':
			if s.page == pageTools {
				if s.toolsTab == toolsTabOverview || s.toolsTab == toolsTabSummary {
					return s.openToolCalls(""), false
				}
				s.toolsTab = toolsTabCalls
				s.selected = 0
				s.scroll = 0
				s.status = "tools tab: Calls"
				return s.load(pageTools), false
			}
			if s.page == pageAuditFindings {
				return s.cycleAuditCategory(), false
			}
		case 'd', 'D':
			if s.page == pageToolCalls || (s.page == pageTools && (s.toolsTab == toolsTabShell || s.toolsTab == toolsTabCalls)) {
				return s.cycleToolCallSort(), false
			}
			if s.page == pageTokens && s.tokensTab == tokensTabBreakdown {
				return s.cycleTokenBreakdownGroup(), false
			}
		case 'u':
			if s.isUsageScopePage() {
				return s.cycleUsageAgent(), false
			}
			if s.isAuditPage() {
				return s.cycleAuditAgent(), false
			}
			if s.page == pageTools || s.page == pageToolCalls {
				return s.cycleToolAgent(), false
			}
		case 'v', 'V':
			if s.isUsageScopePage() {
				return s.cycleUsageModel(), false
			}
			if s.page == pageAuditFindings {
				return s.cycleAuditSeverity(), false
			}
			if s.page == pageTools && s.toolsTab == toolsTabShell {
				return s.cycleToolCommand(), false
			}
		case 'w', 'W':
			if s.isUsageScopePage() {
				return s.cycleUsageProject(), false
			}
		case 'e', 'E':
			if s.isUsageScopePage() {
				return s.cycleUsageRange(), false
			}
			if s.page == pageToolCalls || (s.page == pageTools && (s.toolsTab == toolsTabShell || s.toolsTab == toolsTabCalls)) {
				return s.cycleToolRange(), false
			}
		case 'U':
			if s.isUsageScopePage() {
				return s.clearUsageScope(), false
			}
			if s.isAuditPage() {
				return s.clearAuditFilters(), false
			}
			if s.page == pageTools || s.page == pageToolCalls {
				return s.clearToolFilters(), false
			}
		case 'y', 'Y':
			if s.page == pageAuditFindings {
				return s.cycleAuditShell(), false
			}
		case 'f', 'F':
			if s.page == pageAudit {
				return s.switchPage(pageAuditFindings), false
			}
		case 'i':
			return s.index(false), false
		case 'I':
			return s.index(true), false
		case 'j', 'J':
			s.move(1)
		case 'k', 'K':
			s.move(-1)
		case 'b', 'B':
			if s.page == pageSessionDetail {
				return s.switchPage(pageSessions), false
			}
			if s.page == pageToolCallDetail {
				return s.switchPage(s.toolCallBackPage()), false
			}
			if s.page == pageToolCalls {
				return s.switchPage(pageTools), false
			}
			if s.page == pageAuditFindings {
				return s.switchPage(pageAudit), false
			}
			if s.page == pageAuditDetail {
				return s.switchPage(s.previous), false
			}
		case '[', 'h', 'H':
			if s.page == pageModelSignals {
				s.cycleModelSignalsTab(-1)
			}
			if s.page == pageTime {
				s.cycleTimeTab(-1)
			}
			if s.page == pageTokens {
				s.cycleTokensTab(-1)
			}
			if s.page == pageTools {
				return s.cycleToolsTab(-1), false
			}
		case ']', 'l', 'L':
			if s.page == pageModelSignals {
				s.cycleModelSignalsTab(1)
			}
			if s.page == pageTime {
				s.cycleTimeTab(1)
			}
			if s.page == pageTokens {
				s.cycleTokensTab(1)
			}
			if s.page == pageTools {
				return s.cycleToolsTab(1), false
			}
		}
	}

	switch k.typ {
	case keyTab, keyRight:
		return s.switchPage(s.nextPage()), false
	case keyShiftTab, keyLeft:
		return s.switchPage(s.previousPage()), false
	case keyUp:
		s.move(-1)
	case keyDown:
		s.move(1)
	case keyPageUp:
		s.move(-s.pageStep())
	case keyPageDown:
		s.move(s.pageStep())
	case keyHome:
		s.moveTo(0)
	case keyEnd:
		s.moveTo(s.itemCount() - 1)
	case keyEnter:
		if s.page == pageSessions && len(s.sessions) > 0 {
			id := s.sessions[s.selected].ID
			s.previous = pageSessions
			s.page = pageSessionDetail
			s.selected = 0
			s.scroll = 0
			s.detail = nil
			return s.loadDetail(id), false
		}
		if s.page == pageTools && (s.toolsTab == toolsTabOverview || s.toolsTab == toolsTabSummary) && len(s.tools) > 0 {
			return s.openToolCalls(s.tools[s.selected].ToolName), false
		}
		if s.page == pageTools && (s.toolsTab == toolsTabShell || s.toolsTab == toolsTabCalls) && len(s.toolCalls) > 0 {
			call := s.toolCalls[s.selected]
			s.previous = pageTools
			s.page = pageToolCallDetail
			s.scroll = 0
			s.toolCall = &call
		}
		if s.page == pageToolCalls && len(s.toolCalls) > 0 {
			call := s.toolCalls[s.selected]
			s.previous = pageToolCalls
			s.page = pageToolCallDetail
			s.scroll = 0
			s.toolCall = &call
		}
		if s.page == pageAudit && len(s.audit.RecentFindings) > 0 {
			finding := s.audit.RecentFindings[s.selected]
			s.previous = pageAudit
			s.page = pageAuditDetail
			s.selected = 0
			s.scroll = 0
			s.finding = nil
			s.auditSession = nil
			return s.loadAuditDetail(finding.ID), false
		}
		if s.page == pageAuditFindings && len(s.findings) > 0 {
			finding := s.findings[s.selected]
			s.previous = pageAuditFindings
			s.page = pageAuditDetail
			s.selected = 0
			s.scroll = 0
			s.finding = nil
			s.auditSession = nil
			return s.loadAuditDetail(finding.ID), false
		}
	case keyEsc:
		if s.page == pageSessionDetail {
			return s.switchPage(s.previous), false
		}
		if s.page == pageToolCallDetail {
			return s.switchPage(s.toolCallBackPage()), false
		}
		if s.page == pageToolCalls {
			return s.switchPage(pageTools), false
		}
		if s.page == pageAuditFindings {
			return s.switchPage(pageAudit), false
		}
		if s.page == pageAuditDetail {
			return s.switchPage(s.previous), false
		}
	}
	return nil, false
}

func (s *state) cycleTimeTab(delta int) {
	s.timeTab = cycleTab(s.timeTab, timeTabs, delta)
	s.scroll = 0
	s.status = "time tab: " + s.timeTab.title()
}

func (s *state) cycleTokensTab(delta int) {
	s.tokensTab = cycleTab(s.tokensTab, tokensTabs, delta)
	s.scroll = 0
	s.status = "tokens tab: " + s.tokensTab.title()
}

func (s *state) cycleToolsTab(delta int) command {
	s.toolsTab = cycleTab(s.toolsTab, toolsTabs, delta)
	s.selected = 0
	s.scroll = 0
	s.status = "tools tab: " + s.toolsTab.title()
	return s.load(pageTools)
}

func cycleTab[T comparable](current T, tabs []T, delta int) T {
	if len(tabs) == 0 || delta == 0 {
		return current
	}
	index := 0
	for i, tab := range tabs {
		if tab == current {
			index = i
			break
		}
	}
	index = (index + delta) % len(tabs)
	if index < 0 {
		index += len(tabs)
	}
	return tabs[index]
}

func (s *state) cycleModelSignalsTab(delta int) {
	if delta == 0 {
		return
	}
	index := 0
	for i, tab := range modelSignalsTabs {
		if tab == s.modelSignalsTab {
			index = i
			break
		}
	}
	index = (index + delta) % len(modelSignalsTabs)
	if index < 0 {
		index += len(modelSignalsTabs)
	}
	s.modelSignalsTab = modelSignalsTabs[index]
	s.scroll = 0
	s.status = "model signals tab: " + s.modelSignalsTab.title()
}

func (s *state) switchPage(target page) command {
	if target == pageSessionDetail {
		target = pageSessions
	}
	if target == pageToolCallDetail {
		target = pageToolCalls
	}
	if target == pageAuditDetail {
		target = pageAudit
	}
	if target == s.page && !s.loading {
		return nil
	}
	s.page = target
	s.selected = 0
	s.scroll = 0
	s.detail = nil
	s.toolCall = nil
	s.finding = nil
	s.auditSession = nil
	if target != pagePrivacy {
		s.privacyPending = nil
	}
	return s.load(target)
}

func (s *state) nextPage() page {
	switch s.page {
	case pageOverview:
		return pageTime
	case pageTime:
		return pageTokens
	case pageTokens:
		return pageModelSignals
	case pageModelSignals:
		return pageModelRisk
	case pageModelRisk:
		return pageSessions
	case pageSessions, pageSessionDetail:
		return pageTools
	case pageTools, pageToolCalls, pageToolCallDetail:
		return pageAudit
	case pageAudit, pageAuditFindings, pageAuditDetail:
		return pagePrivacy
	case pagePrivacy:
		return pageSettings
	case pageSettings:
		return pageOverview
	default:
		return pageOverview
	}
}

func (s *state) previousPage() page {
	switch s.page {
	case pageOverview:
		return pageSettings
	case pageTime:
		return pageOverview
	case pageTokens:
		return pageTime
	case pageModelSignals:
		return pageTokens
	case pageModelRisk:
		return pageModelSignals
	case pageSessions, pageSessionDetail:
		return pageModelRisk
	case pageToolCalls, pageToolCallDetail:
		return pageTools
	case pageTools:
		return pageSessions
	case pageAuditFindings, pageAuditDetail:
		return pageAudit
	case pageAudit:
		return pageTools
	case pagePrivacy:
		return pageAudit
	case pageSettings:
		return pagePrivacy
	default:
		return pageSettings
	}
}
