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
		if cmd, quit, handled := s.handleRuneKey(k.ch); handled {
			return cmd, quit
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
		return s.openSelection(), false
	case keyEsc:
		return s.back(), false
	}
	return nil, false
}

func (s *state) handleRuneKey(ch rune) (command, bool, bool) {
	if ch == 'q' || ch == 'Q' {
		return nil, true, true
	}
	if target, ok := pageShortcut(ch); ok {
		return s.switchPage(target), false, true
	}
	if cmd, ok := s.handleGlobalRuneKey(ch); ok {
		return cmd, false, true
	}
	if cmd, ok := s.handleScopedFilterKey(ch); ok {
		return cmd, false, true
	}
	if cmd, ok := s.handlePageActionKey(ch); ok {
		return cmd, false, true
	}
	if cmd, ok := s.handleTabKey(ch); ok {
		return cmd, false, true
	}
	return nil, false, false
}

func pageShortcut(ch rune) (page, bool) {
	switch ch {
	case '1', 'o', 'O':
		return pageOverview, true
	case '2':
		return pageTime, true
	case '3', 'n', 'N':
		return pageTokens, true
	case '4', 'm', 'M':
		return pageModelSignals, true
	case '5', 'x', 'X':
		return pageModelRisk, true
	case '6', 's', 'S':
		return pageSessions, true
	case '7', 't', 'T':
		return pageTools, true
	case '8', 'a':
		return pageAudit, true
	case '9', 'p', 'P':
		return pagePrivacy, true
	case '0', 'g', 'G':
		return pageSettings, true
	default:
		return pageOverview, false
	}
}

func (s *state) handleGlobalRuneKey(ch rune) (command, bool) {
	switch ch {
	case 'r', 'R':
		return s.refresh(), true
	case 'i':
		return s.index(false), true
	case 'I':
		return s.index(true), true
	case 'j', 'J':
		s.move(1)
		return nil, true
	case 'k', 'K':
		s.move(-1)
		return nil, true
	case 'b', 'B':
		return s.back(), true
	default:
		return nil, false
	}
}

func (s *state) handleScopedFilterKey(ch rune) (command, bool) {
	switch ch {
	case 'u':
		return s.cycleAgentFilter()
	case 'v', 'V':
		return s.cycleModelOrCommandFilter()
	case 'w', 'W':
		if s.isUsageScopePage() {
			return s.cycleUsageProject(), true
		}
	case 'e', 'E':
		return s.cycleRangeFilter()
	case 'U':
		return s.clearActiveFilters()
	}
	return nil, false
}

func (s *state) cycleAgentFilter() (command, bool) {
	if s.isUsageScopePage() {
		return s.cycleUsageAgent(), true
	}
	if s.isAuditPage() {
		return s.cycleAuditAgent(), true
	}
	if s.page == pageTools || s.page == pageToolCalls {
		return s.cycleToolAgent(), true
	}
	return nil, false
}

func (s *state) cycleModelOrCommandFilter() (command, bool) {
	if s.isUsageScopePage() {
		return s.cycleUsageModel(), true
	}
	if s.page == pageAuditFindings {
		return s.cycleAuditSeverity(), true
	}
	if s.page == pageTools && s.toolsTab == toolsTabShell {
		return s.cycleToolCommand(), true
	}
	return nil, false
}

func (s *state) cycleRangeFilter() (command, bool) {
	if s.isUsageScopePage() {
		return s.cycleUsageRange(), true
	}
	if s.page == pageToolCalls || (s.page == pageTools && (s.toolsTab == toolsTabShell || s.toolsTab == toolsTabCalls)) {
		return s.cycleToolRange(), true
	}
	return nil, false
}

func (s *state) clearActiveFilters() (command, bool) {
	if s.isUsageScopePage() {
		return s.clearUsageScope(), true
	}
	if s.isAuditPage() {
		return s.clearAuditFilters(), true
	}
	if s.page == pageTools || s.page == pageToolCalls {
		return s.clearToolFilters(), true
	}
	return nil, false
}

func (s *state) handlePageActionKey(ch rune) (command, bool) {
	switch ch {
	case 'c', 'C':
		return s.handleCallsOrCategoryKey()
	case 'd', 'D':
		return s.handleDetailSortKey()
	case 'y', 'Y':
		if s.page == pageAuditFindings {
			return s.cycleAuditShell(), true
		}
	case 'f', 'F':
		if s.page == pageAudit {
			return s.switchPage(pageAuditFindings), true
		}
	}
	return nil, false
}

func (s *state) handleCallsOrCategoryKey() (command, bool) {
	if s.page == pageTools {
		if s.toolsTab == toolsTabOverview || s.toolsTab == toolsTabSummary {
			return s.openToolCalls(""), true
		}
		s.toolsTab = toolsTabCalls
		s.selected = 0
		s.scroll = 0
		s.status = "tools tab: Calls"
		return s.load(pageTools), true
	}
	if s.page == pageAuditFindings {
		return s.cycleAuditCategory(), true
	}
	return nil, false
}

func (s *state) handleDetailSortKey() (command, bool) {
	if s.page == pageToolCalls || (s.page == pageTools && (s.toolsTab == toolsTabShell || s.toolsTab == toolsTabCalls)) {
		return s.cycleToolCallSort(), true
	}
	if s.page == pageTokens && s.tokensTab == tokensTabBreakdown {
		return s.cycleTokenBreakdownGroup(), true
	}
	return nil, false
}

func (s *state) handleTabKey(ch rune) (command, bool) {
	switch ch {
	case '[', 'h', 'H':
		return s.cycleActiveTab(-1)
	case ']', 'l', 'L':
		return s.cycleActiveTab(1)
	default:
		return nil, false
	}
}

func (s *state) cycleActiveTab(delta int) (command, bool) {
	switch s.page {
	case pageModelSignals:
		s.cycleModelSignalsTab(delta)
	case pageTime:
		s.cycleTimeTab(delta)
	case pageTokens:
		s.cycleTokensTab(delta)
	case pageTools:
		return s.cycleToolsTab(delta), true
	default:
		return nil, false
	}
	return nil, true
}

func (s *state) openSelection() command {
	switch {
	case s.page == pageSessions && len(s.sessions) > 0:
		return s.openSessionDetail()
	case s.page == pageTools && (s.toolsTab == toolsTabOverview || s.toolsTab == toolsTabSummary) && len(s.tools) > 0:
		return s.openToolCalls(s.tools[s.selected].ToolName)
	case s.page == pageTools && (s.toolsTab == toolsTabShell || s.toolsTab == toolsTabCalls) && len(s.toolCalls) > 0:
		s.openToolCallDetail(pageTools)
	case s.page == pageToolCalls && len(s.toolCalls) > 0:
		s.openToolCallDetail(pageToolCalls)
	case s.page == pageAudit && len(s.audit.RecentFindings) > 0:
		return s.openRecentAuditFinding()
	case s.page == pageAuditFindings && len(s.findings) > 0:
		return s.openAuditFinding()
	}
	return nil
}

func (s *state) openSessionDetail() command {
	id := s.sessions[s.selected].ID
	s.previous = pageSessions
	s.page = pageSessionDetail
	s.selected = 0
	s.scroll = 0
	s.detail = nil
	return s.loadDetail(id)
}

func (s *state) openToolCallDetail(previous page) {
	call := s.toolCalls[s.selected]
	s.previous = previous
	s.page = pageToolCallDetail
	s.scroll = 0
	s.toolCall = &call
}

func (s *state) openRecentAuditFinding() command {
	finding := s.audit.RecentFindings[s.selected]
	return s.openAuditDetail(pageAudit, finding.ID)
}

func (s *state) openAuditFinding() command {
	finding := s.findings[s.selected]
	return s.openAuditDetail(pageAuditFindings, finding.ID)
}

func (s *state) openAuditDetail(previous page, id int64) command {
	s.previous = previous
	s.page = pageAuditDetail
	s.selected = 0
	s.scroll = 0
	s.finding = nil
	s.auditSession = nil
	return s.loadAuditDetail(id)
}

func (s *state) back() command {
	switch s.page {
	case pageSessionDetail:
		return s.switchPage(s.previous)
	case pageToolCallDetail:
		return s.switchPage(s.toolCallBackPage())
	case pageToolCalls:
		return s.switchPage(pageTools)
	case pageAuditFindings:
		return s.switchPage(pageAudit)
	case pageAuditDetail:
		return s.switchPage(s.previous)
	default:
		return nil
	}
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
