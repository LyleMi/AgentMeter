package tui

var pageFooterText = map[page]string{
	pageAttention:      "Keys: enter review  u/v/w/e filters  U reset to 7 days  up/down select  [/]/h/l task tabs  r refresh  i update index  q quit",
	pageOverview:       "Keys: [/]/h/l Usage/Time/Models/Tools  u/v/w/e scope  U reset to 7 days  up/down  r refresh  i update index  q quit",
	pageTime:           "Keys: [/]/h/l time tabs  u/v/w/e scope  U clear  up/down scroll  tab cycle pages  r refresh  i update index  q quit",
	pageSessionDetail:  "Keys: b/esc back  up/down scroll  r refresh  i update index  q quit",
	pageSessions:       "Keys: enter detail  u/v/w/e scope  U reset to 7 days  up/down select  tab cycle  r refresh  i update index  q quit",
	pageToolCalls:      "Keys: enter detail  b/esc tools  u source  e range  d sort  U clear  up/down select  r refresh  i update index  q quit",
	pageToolCallDetail: "Keys: b/esc back  up/down scroll  r refresh  i update index  q quit",
	pageModelSignals:   "Keys: [/]/h/l signal tabs  u/v/w/e scope  U clear  up/down scroll  tab cycle pages  r refresh  i update index  q quit",
	pageModelRisk:      "Keys: u/v/w/e scope  U clear  up/down scroll  tab cycle pages  r refresh  i update index  q quit",
	pageAudit:          "Keys: [/]/h/l Audit/Privacy  e range  U reset to 7 days  enter detail  f findings  up/down  r refresh  i update index  q quit",
	pageAuditFindings:  "Keys: enter detail  c category  v severity  y shell  u source  e range  U reset  b/esc summary  up/down select  r refresh  i update index  q quit",
	pageAuditDetail:    "Keys: b/esc back  u source  e range  U reset filters  up/down scroll  r refresh  i update index  q quit",
	pageSettings:       "Keys: up/down scroll  tab cycle  r refresh  i update index  I rebuild index  q quit",
}

const (
	defaultFooterText        = "Keys: 1-5 switch tasks  tab cycle  up/down select/scroll  r refresh  i update index  q quit"
	tokensFooterText         = "Keys: [/]/h/l token tabs  u/v/w/e scope  U clear  up/down scroll  tab cycle pages  r refresh  i update index  q quit"
	tokenBreakdownFooterText = "Keys: [/]/h/l token tabs  d group  u/v/w/e scope  U clear  up/down scroll  r refresh  i update index  q quit"
	toolsFooterText          = "Keys: [/]/h/l tool tabs  enter calls  c all calls  u source  U clear  up/down select  tab cycle  r refresh  i update index  q quit"
	toolShellFooterText      = "Keys: [/]/h/l tool tabs  enter detail  v command  u source  e range  d sort  U clear  up/down select  r refresh  i update index  q quit"
	toolCallsFooterText      = "Keys: [/]/h/l tool tabs  enter detail  u source  e range  d sort  U clear  up/down select  r refresh  i update index  q quit"
	privacyFooterText        = "Keys: up/down target  enter recommended  A strict  u defaults  pgup/pgdn detail  r refresh  q quit"
	privacyPendingFooterText = "Keys: enter write profile  esc cancel  q quit"
)

func (s *state) footerLine() string {
	text := s.footerText()
	if position := s.positionLabel(); position != "" {
		text += "  " + position
	}
	return dim(text)
}

func (s *state) footerText() string {
	switch s.page {
	case pageTokens:
		if s.tokensTab == tokensTabBreakdown {
			return tokenBreakdownFooterText
		}
		return tokensFooterText
	case pageTools:
		return s.toolsFooterText()
	case pagePrivacy:
		if s.privacyPending != nil {
			return privacyPendingFooterText
		}
		return privacyFooterText
	default:
		return footerTextForPage(s.page)
	}
}

func (s *state) toolsFooterText() string {
	switch s.toolsTab {
	case toolsTabShell:
		return toolShellFooterText
	case toolsTabCalls:
		return toolCallsFooterText
	default:
		return toolsFooterText
	}
}

func footerTextForPage(current page) string {
	if text, ok := pageFooterText[current]; ok {
		return text
	}
	return defaultFooterText
}
