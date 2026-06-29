package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	agentmodel "github.com/LyleMi/AgentMeter/internal/model"
)

type appService interface {
	GetOverview() (agentmodel.Overview, error)
	GetOverviewWithFilters(agentmodel.AnalyticsFilters) (agentmodel.Overview, error)
	GetTokenAnalyticsWithFilters(agentmodel.AnalyticsFilters) (agentmodel.TokenAnalytics, error)
	GetUsageBreakdown(groupBy string, filters agentmodel.AnalyticsFilters) (agentmodel.UsageBreakdown, error)
	GetModelSignalsWithFilters(agentmodel.AnalyticsFilters) (agentmodel.ModelSignals, error)
	ListSessions(agentmodel.SessionFilters) ([]agentmodel.Session, error)
	GetSessionDetail(id int64) (agentmodel.SessionDetail, error)
	ListTools(agentmodel.ToolFilters) ([]agentmodel.ToolStat, error)
	ListToolCalls(agentmodel.ToolCallFilters) ([]agentmodel.ToolCall, error)
	GetAuditSummaryWithFilters(agentmodel.AuditFindingFilters) (agentmodel.AuditSummary, error)
	ListAuditFindings(agentmodel.AuditFindingFilters) ([]agentmodel.AuditFinding, error)
	GetAuditFinding(id int64) (agentmodel.AuditFinding, error)
	GetSettings() (agentmodel.Settings, error)
	GetPrivacyConfigs() ([]agentmodel.PrivacyConfigStatus, error)
	IndexNow(rebuild bool) (agentmodel.IndexResult, error)
}

type privacyProfileApplier interface {
	ApplyPrivacyProfile(target, profile string) (agentmodel.PrivacyConfigApplyResult, error)
}

type page int

const (
	pageOverview page = iota
	pageTime
	pageTokens
	pageModelSignals
	pageModelRisk
	pageSessions
	pageSessionDetail
	pageTools
	pageToolCalls
	pageToolCallDetail
	pageAudit
	pageAuditFindings
	pageAuditDetail
	pageSettings
	pagePrivacy
)

type timeTab int

const (
	timeTabSummary timeTab = iota
	timeTabSources
	timeTabTools
	timeTabSessions
)

var timeTabs = []timeTab{
	timeTabSummary,
	timeTabSources,
	timeTabTools,
	timeTabSessions,
}

func (t timeTab) title() string {
	switch t {
	case timeTabSummary:
		return "Summary"
	case timeTabSources:
		return "Sources"
	case timeTabTools:
		return "Tools"
	case timeTabSessions:
		return "Slow Sessions"
	default:
		return "Summary"
	}
}

type tokensTab int

const (
	tokensTabSummary tokensTab = iota
	tokensTabTrends
	tokensTabBreakdown
	tokensTabSessions
)

var tokensTabs = []tokensTab{
	tokensTabSummary,
	tokensTabTrends,
	tokensTabBreakdown,
	tokensTabSessions,
}

func (t tokensTab) title() string {
	switch t {
	case tokensTabSummary:
		return "Summary"
	case tokensTabTrends:
		return "Trends"
	case tokensTabBreakdown:
		return "Breakdown"
	case tokensTabSessions:
		return "Sessions"
	default:
		return "Summary"
	}
}

const tokenBreakdownGlobal = "global"

var tokenBreakdownGroups = []string{
	tokenBreakdownGlobal,
	"agent",
	"model",
	"agent,model",
	"project",
	"day",
}

func tokenBreakdownGroupTitle(group string) string {
	switch group {
	case "agent":
		return "Source"
	case "model":
		return "Model"
	case "agent,model":
		return "Source + Model"
	case "project":
		return "Project"
	case "day":
		return "Day"
	default:
		return "Global"
	}
}

type toolsTab int

const (
	toolsTabOverview toolsTab = iota
	toolsTabSummary
	toolsTabShell
	toolsTabCalls
)

var toolsTabs = []toolsTab{
	toolsTabOverview,
	toolsTabSummary,
	toolsTabShell,
	toolsTabCalls,
}

func (t toolsTab) title() string {
	switch t {
	case toolsTabSummary:
		return "Summary"
	case toolsTabShell:
		return "Shell"
	case toolsTabCalls:
		return "Calls"
	default:
		return "Overview"
	}
}

type usageRange int

const (
	usageRangeAll usageRange = iota
	usageRangeDay
	usageRangeWeek
	usageRangeMonth
)

var usageRanges = []usageRange{
	usageRangeAll,
	usageRangeDay,
	usageRangeWeek,
	usageRangeMonth,
}

func (r usageRange) title() string {
	switch r {
	case usageRangeDay:
		return "1 day"
	case usageRangeWeek:
		return "7 days"
	case usageRangeMonth:
		return "30 days"
	default:
		return "All"
	}
}

type modelSignalsTab int

const (
	modelSignalsTabCharts modelSignalsTab = iota
	modelSignalsTabOverview
	modelSignalsTabDaily
	modelSignalsTabCohorts
	modelSignalsTabMatrix
	modelSignalsTabProjects
	modelSignalsTabAnomalies
)

var modelSignalsTabs = []modelSignalsTab{
	modelSignalsTabCharts,
	modelSignalsTabOverview,
	modelSignalsTabDaily,
	modelSignalsTabCohorts,
	modelSignalsTabMatrix,
	modelSignalsTabProjects,
	modelSignalsTabAnomalies,
}

func (t modelSignalsTab) title() string {
	switch t {
	case modelSignalsTabCharts:
		return "Charts"
	case modelSignalsTabOverview:
		return "Overview"
	case modelSignalsTabDaily:
		return "Daily"
	case modelSignalsTabCohorts:
		return "Cohorts"
	case modelSignalsTabMatrix:
		return "Matrix"
	case modelSignalsTabProjects:
		return "Projects"
	case modelSignalsTabAnomalies:
		return "Anomalies"
	default:
		return "Charts"
	}
}

func (p page) title() string {
	switch p {
	case pageOverview:
		return "Overview"
	case pageTime:
		return "Time"
	case pageTokens:
		return "Tokens"
	case pageModelSignals:
		return "Model Signals"
	case pageModelRisk:
		return "Model Risk"
	case pageSessions:
		return "Sessions"
	case pageSessionDetail:
		return "Session Detail"
	case pageTools:
		return "Tools"
	case pageToolCalls:
		return "Tool Calls"
	case pageToolCallDetail:
		return "Tool Call Detail"
	case pageAudit:
		return "Audit"
	case pageAuditFindings:
		return "Audit Findings"
	case pageAuditDetail:
		return "Audit Detail"
	case pageSettings:
		return "Settings"
	case pagePrivacy:
		return "Agent Privacy"
	default:
		return "Unknown"
	}
}

type keyType int

const (
	keyUnknown keyType = iota
	keyRune
	keyEnter
	keyEsc
	keyCtrlC
	keyTab
	keyShiftTab
	keyUp
	keyDown
	keyLeft
	keyRight
	keyPageUp
	keyPageDown
	keyHome
	keyEnd
)

type keyMsg struct {
	typ keyType
	ch  rune
}

type loadMsg struct {
	seq           int
	page          page
	overview      agentmodel.Overview
	scopeOverview agentmodel.Overview
	scopeProjects agentmodel.UsageBreakdown
	tokens        agentmodel.TokenAnalytics
	breakdown     agentmodel.UsageBreakdown
	sessions      []agentmodel.Session
	detail        agentmodel.SessionDetail
	tools         []agentmodel.ToolStat
	toolCalls     []agentmodel.ToolCall
	toolCall      *agentmodel.ToolCall
	signals       agentmodel.ModelSignals
	audit         agentmodel.AuditSummary
	findings      []agentmodel.AuditFinding
	finding       *agentmodel.AuditFinding
	auditSession  *agentmodel.SessionDetail
	settings      agentmodel.Settings
	privacy       []agentmodel.PrivacyConfigStatus
	err           error
}

type indexMsg struct {
	result  agentmodel.IndexResult
	rebuild bool
	err     error
}

type privacyProfileAction struct {
	target     string
	targetName string
	profile    string
}

type privacyProfileMsg struct {
	target     string
	targetName string
	profile    string
	result     agentmodel.PrivacyConfigApplyResult
	err        error
}

type resizeMsg struct {
	width  int
	height int
}

type message interface{}

type command func(context.Context, chan<- message)

type state struct {
	service appService

	page     page
	previous page

	width  int
	height int

	loadSeq int
	loading bool
	err     error
	status  string

	selected int
	scroll   int

	overview      agentmodel.Overview
	scopeOverview agentmodel.Overview
	scopeProjects agentmodel.UsageBreakdown
	tokens        agentmodel.TokenAnalytics
	breakdown     agentmodel.UsageBreakdown
	sessions      []agentmodel.Session
	detail        *agentmodel.SessionDetail
	tools         []agentmodel.ToolStat
	toolCalls     []agentmodel.ToolCall
	toolCall      *agentmodel.ToolCall
	signals       agentmodel.ModelSignals
	audit         agentmodel.AuditSummary
	findings      []agentmodel.AuditFinding
	finding       *agentmodel.AuditFinding
	auditSession  *agentmodel.SessionDetail
	settings      agentmodel.Settings
	privacy       []agentmodel.PrivacyConfigStatus

	indexing  bool
	lastIndex *agentmodel.IndexResult

	toolCallTool string
	toolCallSort string
	toolsTab     toolsTab
	toolAgent    string
	toolRange    usageRange
	toolCommand  string

	privacyTarget   int
	privacyPending  *privacyProfileAction
	privacyApplying bool

	modelSignalsTab     modelSignalsTab
	timeTab             timeTab
	tokensTab           tokensTab
	tokenBreakdownGroup string
	usageAgent          string
	usageModel          string
	usageProject        string
	usageRange          usageRange
	auditAgent          string
	auditCategory       string
	auditSeverity       string
	auditShell          string
}

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

func (s *state) analyticsFilters() agentmodel.AnalyticsFilters {
	filters := agentmodel.AnalyticsFilters{
		Agent:   strings.TrimSpace(s.usageAgent),
		Model:   strings.TrimSpace(s.usageModel),
		Project: strings.TrimSpace(s.usageProject),
	}
	switch s.usageRange {
	case usageRangeDay:
		filters.StartedFrom = time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC3339)
	case usageRangeWeek:
		filters.StartedFrom = time.Now().AddDate(0, 0, -7).UTC().Format(time.RFC3339)
	case usageRangeMonth:
		filters.StartedFrom = time.Now().AddDate(0, 0, -30).UTC().Format(time.RFC3339)
	}
	return filters
}

func (s *state) loadUsageScopeOptions(filters agentmodel.AnalyticsFilters, fallback agentmodel.Overview) (agentmodel.Overview, agentmodel.UsageBreakdown) {
	overview := fallback
	var projects agentmodel.UsageBreakdown
	if hasAnalyticsFilters(filters) {
		if value, err := s.service.GetOverview(); err == nil {
			overview = value
		}
	}
	if len(overview.AgentUsage) == 0 && len(overview.ModelUsage) == 0 && len(overview.RecentSessions) == 0 {
		if value, err := s.service.GetOverview(); err == nil {
			overview = value
		}
	}
	if value, err := s.service.GetUsageBreakdown("project", agentmodel.AnalyticsFilters{}); err == nil {
		projects = value
	}
	return overview, projects
}

func hasAnalyticsFilters(filters agentmodel.AnalyticsFilters) bool {
	return strings.TrimSpace(filters.Agent) != "" ||
		strings.TrimSpace(filters.Model) != "" ||
		strings.TrimSpace(filters.Project) != "" ||
		strings.TrimSpace(filters.StartedFrom) != "" ||
		strings.TrimSpace(filters.StartedTo) != ""
}

func (s *state) mergeScopeOptions(overview agentmodel.Overview, projects agentmodel.UsageBreakdown) {
	if len(overview.AgentUsage) > 0 || len(overview.ModelUsage) > 0 || len(overview.RecentSessions) > 0 || len(overview.SlowSessions) > 0 {
		s.scopeOverview = overview
	}
	if len(projects.Buckets) > 0 {
		s.scopeProjects = projects
	}
}

func (s *state) isUsageScopePage() bool {
	switch s.page {
	case pageOverview, pageTime, pageTokens, pageModelSignals, pageModelRisk:
		return true
	default:
		return false
	}
}

func (s *state) cycleUsageAgent() command {
	options := usageAgentOptions(s.scopeOverview)
	next, label := cycleStringOption(s.usageAgent, options)
	s.usageAgent = next
	s.selected = 0
	s.scroll = 0
	s.status = "source filter: " + label
	return s.load(s.page)
}

func (s *state) cycleUsageModel() command {
	options := usageModelOptions(s.scopeOverview)
	next, label := cycleStringOption(s.usageModel, options)
	s.usageModel = next
	s.selected = 0
	s.scroll = 0
	s.status = "model filter: " + label
	return s.load(s.page)
}

func (s *state) cycleUsageProject() command {
	options := usageProjectOptions(s.scopeProjects, s.scopeOverview)
	next, label := cycleStringOption(s.usageProject, options)
	s.usageProject = next
	s.selected = 0
	s.scroll = 0
	s.status = "project filter: " + label
	return s.load(s.page)
}

func (s *state) cycleUsageRange() command {
	s.usageRange = cycleTab(s.usageRange, usageRanges, 1)
	s.selected = 0
	s.scroll = 0
	s.status = "range filter: " + s.usageRange.title()
	return s.load(s.page)
}

func (s *state) clearUsageScope() command {
	if s.usageAgent == "" && s.usageModel == "" && s.usageProject == "" && s.usageRange == usageRangeAll {
		s.status = "usage scope already clear"
		return nil
	}
	s.usageAgent = ""
	s.usageModel = ""
	s.usageProject = ""
	s.usageRange = usageRangeAll
	s.selected = 0
	s.scroll = 0
	s.status = "usage scope cleared"
	return s.load(s.page)
}

func (s *state) isAuditPage() bool {
	switch s.page {
	case pageAudit, pageAuditFindings, pageAuditDetail:
		return true
	default:
		return false
	}
}

func (s *state) auditSummaryFilters() agentmodel.AuditFindingFilters {
	return agentmodel.AuditFindingFilters{Agent: strings.TrimSpace(s.auditAgent)}
}

func (s *state) auditFilters() agentmodel.AuditFindingFilters {
	return agentmodel.AuditFindingFilters{
		Agent:       strings.TrimSpace(s.auditAgent),
		Category:    strings.TrimSpace(s.auditCategory),
		Severity:    strings.TrimSpace(s.auditSeverity),
		ShellFamily: strings.TrimSpace(s.auditShell),
		Limit:       500,
	}
}

func (s *state) cycleAuditAgent() command {
	options := usageAgentOptions(s.scopeOverview)
	next, label := cycleStringOption(s.auditAgent, options)
	s.auditAgent = next
	s.selected = 0
	s.scroll = 0
	s.status = "audit source filter: " + label
	return s.reloadAuditPage()
}

func (s *state) cycleAuditCategory() command {
	next, label := cycleStringOption(s.auditCategory, auditCategoryOptions())
	s.auditCategory = next
	s.selected = 0
	s.scroll = 0
	s.status = "audit category filter: " + label
	return s.load(pageAuditFindings)
}

func (s *state) cycleAuditSeverity() command {
	next, label := cycleStringOption(s.auditSeverity, auditSeverityOptions())
	s.auditSeverity = next
	s.selected = 0
	s.scroll = 0
	s.status = "audit severity filter: " + label
	return s.load(pageAuditFindings)
}

func (s *state) cycleAuditShell() command {
	next, label := cycleStringOption(s.auditShell, auditShellOptions())
	s.auditShell = next
	s.selected = 0
	s.scroll = 0
	s.status = "audit shell filter: " + label
	return s.load(pageAuditFindings)
}

func (s *state) clearAuditFilters() command {
	if s.auditAgent == "" && s.auditCategory == "" && s.auditSeverity == "" && s.auditShell == "" {
		s.status = "audit filters already clear"
		return nil
	}
	s.auditAgent = ""
	s.auditCategory = ""
	s.auditSeverity = ""
	s.auditShell = ""
	s.selected = 0
	s.scroll = 0
	s.status = "audit filters cleared"
	return s.reloadAuditPage()
}

func (s *state) reloadAuditPage() command {
	if s.page == pageAuditDetail {
		if s.finding == nil {
			return nil
		}
		return s.loadAuditDetail(s.finding.ID)
	}
	return s.load(auditListRootPage(s.page))
}

func auditListRootPage(current page) page {
	if current == pageAuditDetail {
		return pageAuditFindings
	}
	return current
}

func auditCategoryOptions() []stringOption {
	return []stringOption{
		{value: "command", label: "Command"},
		{value: "privacy", label: "Privacy"},
		{value: "egress", label: "Egress"},
		{value: "file", label: "File"},
	}
}

func auditSeverityOptions() []stringOption {
	return []stringOption{
		{value: "critical", label: "Critical"},
		{value: "high", label: "High"},
		{value: "medium", label: "Medium"},
		{value: "low", label: "Low"},
	}
}

func auditShellOptions() []stringOption {
	return []stringOption{
		{value: "posix", label: "POSIX"},
		{value: "powershell", label: "PowerShell"},
		{value: "cmd", label: "cmd.exe"},
		{value: "unknown", label: "Unknown shell"},
	}
}

type stringOption struct {
	value string
	label string
}

func cycleStringOption(current string, options []stringOption) (string, string) {
	if len(options) == 0 {
		return "", "All"
	}
	all := append([]stringOption{{label: "All"}}, options...)
	index := 0
	for i, option := range all {
		if option.value == current {
			index = i
			break
		}
	}
	next := all[(index+1)%len(all)]
	return next.value, next.label
}

func usageAgentOptions(overview agentmodel.Overview) []stringOption {
	seen := map[string]string{}
	for _, row := range overview.AgentUsage {
		addSourceOption(seen, row.SourceID, row.SourceKey, row.SourceLabel, row.AgentKind, row.AgentName, row.SourceRootPath, row.SourceSessionsPath)
	}
	for _, session := range append(append([]agentmodel.Session{}, overview.RecentSessions...), overview.SlowSessions...) {
		addSourceOption(seen, session.SourceID, session.SourceKey, session.SourceLabel, session.AgentKind, session.AgentName, session.SourceRootPath, session.SourceSessionsPath)
	}
	return sortedStringOptions(seen)
}

func usageModelOptions(overview agentmodel.Overview) []stringOption {
	seen := map[string]string{}
	for _, row := range overview.ModelUsage {
		addValueOption(seen, row.Model, empty(row.Model, "unknown"))
	}
	for _, session := range append(append([]agentmodel.Session{}, overview.RecentSessions...), overview.SlowSessions...) {
		addValueOption(seen, session.Model, empty(session.Model, "unknown"))
	}
	return sortedStringOptions(seen)
}

func usageProjectOptions(projects agentmodel.UsageBreakdown, overview agentmodel.Overview) []stringOption {
	seen := map[string]string{}
	for _, row := range projects.Buckets {
		addValueOption(seen, row.ProjectPath, shortPath(row.ProjectPath, 36))
	}
	for _, session := range append(append([]agentmodel.Session{}, overview.RecentSessions...), overview.SlowSessions...) {
		addValueOption(seen, session.ProjectPath, shortPath(session.ProjectPath, 36))
	}
	return sortedStringOptions(seen)
}

func addSourceOption(seen map[string]string, sourceID int64, sourceKey, sourceLabel, agentKind, agentName, rootPath, sessionsPath string) {
	value := strings.TrimSpace(sourceKey)
	if value == "" && sourceID > 0 {
		value = fmt.Sprintf("source:%d", sourceID)
	}
	if value == "" {
		value = strings.TrimSpace(agentKind)
	}
	if value == "" {
		value = strings.TrimSpace(agentName)
	}
	if value == "" {
		return
	}
	label := sourceDisplayName(sourceLabel, agentName, agentKind, sourceKey)
	context := sourceContext(agentKind, agentName, rootPath, sessionsPath)
	if context != "" && context != label {
		label += " (" + context + ")"
	}
	seen[value] = label
}

func addValueOption(seen map[string]string, value, label string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	label = strings.TrimSpace(label)
	if label == "" {
		label = value
	}
	seen[value] = label
}

func sortedStringOptions(seen map[string]string) []stringOption {
	options := make([]stringOption, 0, len(seen))
	for value, label := range seen {
		options = append(options, stringOption{value: value, label: label})
	}
	sort.SliceStable(options, func(i, j int) bool {
		if options[i].label == options[j].label {
			return options[i].value < options[j].value
		}
		return options[i].label < options[j].label
	})
	return options
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

func (s *state) toolCallFilters() agentmodel.ToolCallFilters {
	filters := s.toolExplorerFilters()
	filters.ToolName = s.toolCallTool
	filters.Limit = 200
	return filters
}

func (s *state) toolExplorerFilters() agentmodel.ToolCallFilters {
	filters := agentmodel.ToolCallFilters{
		Agent: strings.TrimSpace(s.toolAgent),
		Sort:  s.toolCallSort,
		Limit: 500,
	}
	switch s.toolRange {
	case usageRangeDay:
		filters.StartedFrom = time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC3339)
	case usageRangeWeek:
		filters.StartedFrom = time.Now().AddDate(0, 0, -7).UTC().Format(time.RFC3339)
	case usageRangeMonth:
		filters.StartedFrom = time.Now().AddDate(0, 0, -30).UTC().Format(time.RFC3339)
	}
	return agentmodel.ToolCallFilters{
		Agent:       filters.Agent,
		StartedFrom: filters.StartedFrom,
		Sort:        filters.Sort,
		Limit:       filters.Limit,
	}
}

func listToolCallsForToolsContext(service appService, filters agentmodel.ToolCallFilters, tools []agentmodel.ToolStat, tab toolsTab, command string) ([]agentmodel.ToolCall, error) {
	if tab != toolsTabShell {
		calls, err := service.ListToolCalls(filters)
		if err != nil {
			return nil, err
		}
		return filterToolCallsForToolsTab(calls, tab, command), nil
	}
	toolNames := shellToolNames(tools)
	if len(toolNames) == 0 {
		return []agentmodel.ToolCall{}, nil
	}
	limit := filters.Limit
	if limit <= 0 {
		limit = 500
	}
	calls := []agentmodel.ToolCall{}
	for _, toolName := range toolNames {
		toolFilters := filters
		toolFilters.ToolName = toolName
		toolFilters.Limit = limit
		values, err := service.ListToolCalls(toolFilters)
		if err != nil {
			return nil, err
		}
		calls = append(calls, values...)
	}
	calls = uniqueToolCalls(calls)
	sortToolCalls(calls, filters.Sort)
	calls = filterToolCallsForToolsTab(calls, tab, command)
	return limitSlice(calls, limit), nil
}

func shellToolNames(tools []agentmodel.ToolStat) []string {
	seen := map[string]bool{}
	names := []string{}
	for _, tool := range tools {
		name := strings.TrimSpace(tool.ToolName)
		if name == "" || seen[name] || !isShellToolName(name) {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func uniqueToolCalls(calls []agentmodel.ToolCall) []agentmodel.ToolCall {
	seen := map[int64]bool{}
	result := make([]agentmodel.ToolCall, 0, len(calls))
	for _, call := range calls {
		if call.ID > 0 {
			if seen[call.ID] {
				continue
			}
			seen[call.ID] = true
		}
		result = append(result, call)
	}
	return result
}

func sortToolCalls(calls []agentmodel.ToolCall, sortMode string) {
	sort.SliceStable(calls, func(i, j int) bool {
		left, right := calls[i], calls[j]
		switch strings.TrimSpace(sortMode) {
		case "duration_desc":
			if left.DurationMS != right.DurationMS {
				return left.DurationMS > right.DurationMS
			}
		case "duration_asc":
			if left.DurationMS != right.DurationMS {
				return left.DurationMS < right.DurationMS
			}
		}
		if !left.StartedAt.Equal(right.StartedAt) {
			return left.StartedAt.After(right.StartedAt)
		}
		return left.ID > right.ID
	})
}

func filterToolCallsForToolsTab(calls []agentmodel.ToolCall, tab toolsTab, command string) []agentmodel.ToolCall {
	if tab != toolsTabShell && strings.TrimSpace(command) == "" {
		return calls
	}
	result := make([]agentmodel.ToolCall, 0, len(calls))
	for _, call := range calls {
		if tab == toolsTabShell && !isShellToolName(call.ToolName) {
			continue
		}
		if strings.TrimSpace(command) != "" && invokedToolCommand(call) != command {
			continue
		}
		result = append(result, call)
	}
	return result
}

func (s *state) cycleTokenBreakdownGroup() command {
	index := 0
	for i, group := range tokenBreakdownGroups {
		if group == s.tokenBreakdownGroup {
			index = i
			break
		}
	}
	index = (index + 1) % len(tokenBreakdownGroups)
	s.tokenBreakdownGroup = tokenBreakdownGroups[index]
	s.selected = 0
	s.scroll = 0
	s.status = "token breakdown group: " + tokenBreakdownGroupTitle(s.tokenBreakdownGroup)
	return s.load(pageTokens)
}

func (s *state) openToolCalls(toolName string) command {
	s.previous = pageTools
	s.page = pageToolCalls
	s.selected = 0
	s.scroll = 0
	s.toolCall = nil
	s.toolCallTool = toolName
	return s.load(pageToolCalls)
}

func (s *state) toolCallBackPage() page {
	if s.previous == pageTools {
		return pageTools
	}
	return pageToolCalls
}

func (s *state) cycleToolCallSort() command {
	switch s.toolCallSort {
	case "":
		s.toolCallSort = "duration_desc"
	case "duration_desc":
		s.toolCallSort = "duration_asc"
	default:
		s.toolCallSort = ""
	}
	s.selected = 0
	s.scroll = 0
	s.status = "tool calls sorted by " + toolCallSortLabel(s.toolCallSort)
	if s.page == pageTools {
		return s.load(pageTools)
	}
	return s.load(pageToolCalls)
}

func (s *state) cycleToolAgent() command {
	options := usageAgentOptions(s.scopeOverview)
	next, label := cycleStringOption(s.toolAgent, options)
	s.toolAgent = next
	s.selected = 0
	s.scroll = 0
	s.status = "tool source filter: " + label
	return s.load(toolListRootPage(s.page))
}

func (s *state) cycleToolRange() command {
	s.toolRange = cycleTab(s.toolRange, usageRanges, 1)
	s.selected = 0
	s.scroll = 0
	s.status = "tool range filter: " + s.toolRange.title()
	return s.load(toolListRootPage(s.page))
}

func (s *state) cycleToolCommand() command {
	options := toolCommandOptions(s.toolCalls)
	next, label := cycleStringOption(s.toolCommand, options)
	s.toolCommand = next
	s.selected = 0
	s.scroll = 0
	s.status = "shell command filter: " + label
	return s.load(pageTools)
}

func (s *state) clearToolFilters() command {
	if s.toolAgent == "" && s.toolRange == usageRangeAll && s.toolCommand == "" {
		s.status = "tool filters already clear"
		return nil
	}
	s.toolAgent = ""
	s.toolRange = usageRangeAll
	s.toolCommand = ""
	s.selected = 0
	s.scroll = 0
	s.status = "tool filters cleared"
	return s.load(toolListRootPage(s.page))
}

func toolListRootPage(current page) page {
	if current == pageToolCalls || current == pageToolCallDetail {
		return pageToolCalls
	}
	return current
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

func auditFindingMatchesAgent(finding agentmodel.AuditFinding, filter string) bool {
	filter = strings.ToLower(strings.TrimSpace(filter))
	if filter == "" {
		return true
	}
	values := []string{
		finding.SourceKey,
		finding.SourceLabel,
		finding.AgentKind,
		finding.AgentName,
	}
	if finding.SourceID > 0 {
		values = append(values, fmt.Sprintf("source:%d", finding.SourceID))
	}
	for _, value := range values {
		if strings.ToLower(strings.TrimSpace(value)) == filter {
			return true
		}
	}
	return false
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
