package tui

func (s *state) itemCount() int {
	switch s.page {
	case pageSessions:
		return len(s.sessions)
	case pageTools:
		if s.toolsTab == toolsTabShell || s.toolsTab == toolsTabCalls {
			return len(s.toolCalls)
		}
		return len(s.tools)
	case pageToolCalls:
		return len(s.toolCalls)
	case pageAudit:
		return len(s.audit.RecentFindings)
	case pageAuditFindings:
		return len(s.findings)
	case pageSessionDetail:
		if s.detail == nil {
			return 0
		}
		return len(sessionDetailLines(*s.detail, s.width))
	case pageToolCallDetail:
		if s.toolCall == nil {
			return 0
		}
		return len(toolCallDetailLines(*s.toolCall, s.width))
	case pageModelSignals:
		return len(modelSignalLines(s.signals, s.width, s.modelSignalsTab))
	case pageTime:
		return len(timeLines(s.overview, s.width, s.timeTab))
	case pageTokens:
		return len(tokenLines(s.tokens, s.breakdown, s.width, s.tokensTab, s.tokenBreakdownGroup))
	case pageModelRisk:
		return len(modelRiskLines(s.signals, s.width))
	case pageAuditDetail:
		return len(auditDetailLines(s.finding, s.auditSession, s.width))
	case pageSettings:
		return len(settingsLines(s.settings, s.width))
	case pagePrivacy:
		if status := s.selectedPrivacyStatus(); status != nil {
			return len(privacyDetailLines(*status, s.width))
		}
		return 0
	default:
		return 0
	}
}

func (s *state) pageStep() int {
	if s.isListPage() {
		return s.visibleListRows()
	}
	step := s.contentHeight() - 2
	if step < 1 {
		return 1
	}
	return step
}

func (s *state) isListPage() bool {
	switch s.page {
	case pageSessions, pageTools, pageToolCalls, pageAudit, pageAuditFindings:
		return true
	default:
		return false
	}
}

func (s *state) visibleListRows() int {
	visible := s.contentHeight() - s.listHeaderLines()
	if visible < 1 {
		return 1
	}
	return visible
}

func (s *state) listHeaderLines() int {
	switch s.page {
	case pageTools, pageToolCalls:
		if s.page == pageTools && s.toolsTab == toolsTabOverview {
			return 5
		}
		return 3
	case pageAudit:
		return 8
	case pageAuditFindings:
		return 3
	default:
		return 2
	}
}

func (s *state) move(delta int) {
	if delta == 0 {
		return
	}
	if s.page == pageSessionDetail || s.page == pageToolCallDetail || s.page == pageModelSignals || s.page == pageTime || s.page == pageTokens || s.page == pageModelRisk || s.page == pageAuditDetail || s.page == pageSettings || s.page == pagePrivacy {
		maxScroll := s.maxScroll()
		s.scroll += delta
		if s.scroll < 0 {
			s.scroll = 0
		}
		if s.scroll > maxScroll {
			s.scroll = maxScroll
		}
		return
	}
	s.moveTo(s.selected + delta)
}

func (s *state) moveTo(index int) {
	count := s.itemCount()
	if count <= 0 {
		s.selected = 0
		s.scroll = 0
		return
	}
	if index < 0 {
		index = 0
	}
	if index >= count {
		index = count - 1
	}
	s.selected = index
	s.ensureVisible()
}

func (s *state) clampSelection(count int) {
	if count <= 0 {
		s.selected = 0
		s.scroll = 0
		return
	}
	if s.selected >= count {
		s.selected = count - 1
	}
	if s.selected < 0 {
		s.selected = 0
	}
	s.ensureVisible()
}

func (s *state) ensureVisible() {
	if s.page == pageSessionDetail || s.page == pageToolCallDetail || s.page == pageModelSignals || s.page == pageTime || s.page == pageTokens || s.page == pageModelRisk || s.page == pageAuditDetail || s.page == pageSettings || s.page == pagePrivacy {
		maxScroll := s.maxScroll()
		if s.scroll > maxScroll {
			s.scroll = maxScroll
		}
		if s.scroll < 0 {
			s.scroll = 0
		}
		return
	}
	visible := s.visibleListRows()
	if s.selected < s.scroll {
		s.scroll = s.selected
	}
	if s.selected >= s.scroll+visible {
		s.scroll = s.selected - visible + 1
	}
}

func (s *state) maxScroll() int {
	if s.page == pagePrivacy {
		return s.privacyMaxScroll()
	}
	max := s.itemCount() - s.contentHeight()
	if max < 0 {
		return 0
	}
	return max
}
