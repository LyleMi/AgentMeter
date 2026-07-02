package tui

import (
	"context"

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
