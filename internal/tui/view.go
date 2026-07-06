package tui

import (
	"fmt"
	"strings"
	"time"

	agentmodel "github.com/LyleMi/AgentMeter/internal/model"
	"github.com/LyleMi/AgentMeter/internal/viewmodel"
)

const (
	defaultWidth  = 100
	defaultHeight = 30
)

func (s *state) view() string {
	width := s.width
	if width <= 0 {
		width = defaultWidth
	}

	lines := []string{
		fit(s.headerLine(), width),
		fit(s.navLine(), width),
		separator(width),
		fit(s.statusLine(), width),
	}

	content := s.content()
	contentHeight := s.contentHeight()
	if len(content) > contentHeight {
		content = content[:contentHeight]
	}
	for len(content) < contentHeight {
		content = append(content, "")
	}
	for _, line := range content {
		lines = append(lines, fit(line, width))
	}

	lines = append(lines, separator(width), fit(s.footerLine(), width))
	return strings.Join(lines, "\n")
}

func (s *state) headerLine() string {
	parts := []string{bold("AgentMeter"), dim(s.page.title())}
	parts = append(parts, s.pageHeaderParts()...)
	parts = append(parts, s.headerFilterParts()...)
	return strings.Join(parts, "  ")
}

func (s *state) pageHeaderParts() []string {
	switch s.page {
	case pageOverview:
		return []string{
			fmt.Sprintf("%s sessions", formatInt(int64(s.overview.TotalSessions))),
			fmt.Sprintf("%s tokens", formatInt(s.overview.TotalTokens)),
			formatCost(s.overview.EstimatedCostUSD),
		}
	case pageTime:
		return []string{
			"tab " + s.timeTab.title(),
			fmt.Sprintf("%s sessions", formatInt(int64(s.overview.TotalSessions))),
			"wall " + formatDuration(s.overview.TotalWallDurationMS),
			"active " + formatPercent(ratio(float64(s.overview.TotalActiveDurationMS), float64(s.overview.TotalWallDurationMS))),
		}
	case pageTokens:
		return []string{
			"tab " + s.tokensTab.title(),
			fmt.Sprintf("%s sessions", formatInt(int64(s.tokens.TotalSessions))),
			fmt.Sprintf("%s tokens", formatInt(s.tokens.TotalTokens)),
			"cache " + formatPercent(s.tokens.CacheUtilizationRate),
		}
	case pageModelSignals:
		return s.modelSignalsHeaderParts()
	case pageModelRisk:
		return s.modelRiskHeaderParts()
	case pageSessions:
		return []string{fmt.Sprintf("%s sessions loaded", formatInt(int64(len(s.sessions))))}
	case pageTools:
		return s.toolsHeaderParts()
	case pageToolCalls:
		return s.toolCallsHeaderParts()
	case pageToolCallDetail:
		return s.toolCallDetailHeaderParts()
	case pageAudit:
		return []string{
			fmt.Sprintf("%s findings", formatInt(int64(s.audit.TotalFindings))),
			fmt.Sprintf("%s critical", formatInt(int64(s.audit.CriticalFindings))),
			fmt.Sprintf("%s high", formatInt(int64(s.audit.HighFindings))),
		}
	case pageAuditFindings:
		return []string{fmt.Sprintf("%s findings loaded", formatInt(int64(len(s.findings))))}
	case pageAuditDetail:
		return s.auditDetailHeaderParts()
	case pagePrivacy:
		return s.privacyHeaderParts()
	case pageSettings:
		return s.settingsHeaderParts()
	default:
		return nil
	}
}

func (s *state) modelSignalsHeaderParts() []string {
	summary := s.signals.HealthSummary
	return []string{
		"tab " + s.modelSignalsTab.title(),
		"health " + modelSignalSeverityLabel(summary.Severity),
		fmt.Sprintf("%s sessions", formatInt(int64(s.signals.TotalSessions))),
		fmt.Sprintf("%s calls", formatInt(int64(s.signals.TotalModelCalls))),
	}
}

func (s *state) modelRiskHeaderParts() []string {
	rows := buildModelRiskRows(s.signals)
	top := modelRiskTopRow(rows)
	return []string{
		fmt.Sprintf("%s rows", formatInt(int64(len(rows)))),
		"top " + formatPercent(top.Score),
		modelRiskLevel(top.Score),
	}
}

func (s *state) toolsHeaderParts() []string {
	parts := []string{"tab " + s.toolsTab.title(), fmt.Sprintf("%s tools", formatInt(int64(len(s.tools))))}
	if s.toolsTab == toolsTabShell || s.toolsTab == toolsTabCalls {
		parts = append(parts, fmt.Sprintf("%s calls", formatInt(int64(len(s.toolCalls)))))
	}
	if filters := toolActiveFilterLabel(s); filters != "" {
		parts = append(parts, filters)
	}
	return parts
}

func (s *state) toolCallsHeaderParts() []string {
	parts := []string{fmt.Sprintf("%s calls", formatInt(int64(len(s.toolCalls))))}
	if strings.TrimSpace(s.toolCallTool) != "" {
		parts = append(parts, "tool "+s.toolCallTool)
	}
	parts = append(parts, toolCallSortLabel(s.toolCallSort))
	if filters := toolActiveFilterLabel(s); filters != "" {
		parts = append(parts, filters)
	}
	return parts
}

func (s *state) toolCallDetailHeaderParts() []string {
	if s.toolCall == nil {
		return nil
	}
	return []string{empty(s.toolCall.ToolName, "unknown"), "#" + formatInt(s.toolCall.ID)}
}

func (s *state) auditDetailHeaderParts() []string {
	if s.finding == nil {
		return nil
	}
	return []string{empty(s.finding.RuleID, "finding"), "#" + formatInt(s.finding.ID), empty(s.finding.Severity, "unknown")}
}

func (s *state) privacyHeaderParts() []string {
	parts := []string{fmt.Sprintf("%s targets", formatInt(int64(len(s.privacy))))}
	if status := s.selectedPrivacyStatus(); status != nil {
		parts = append(parts, "selected "+privacyDisplayName(*status))
	}
	return parts
}

func (s *state) settingsHeaderParts() []string {
	if len(s.settings.SourceEntries) == 0 {
		return nil
	}
	return []string{fmt.Sprintf("%s sources", formatInt(int64(len(s.settings.SourceEntries))))}
}

func (s *state) headerFilterParts() []string {
	parts := []string{}
	if s.isUsageScopePage() {
		if scope := s.usageScopeLabel(); scope != "" {
			parts = append(parts, scope)
		}
	}
	if s.isAuditPage() {
		if filters := s.auditFilterLabel(); filters != "" {
			parts = append(parts, filters)
		}
	}
	return parts
}

func (s *state) navLine() string {
	items := []struct {
		key   string
		page  page
		label string
	}{
		{"1", pageOverview, "Overview"},
		{"2", pageTime, "Time"},
		{"3", pageTokens, "Tokens"},
		{"4", pageModelSignals, "Model Signals"},
		{"5", pageModelRisk, "Model Risk"},
		{"6", pageSessions, "Sessions"},
		{"7", pageTools, "Tools"},
		{"8", pageAudit, "Audit"},
		{"9", pagePrivacy, "Agent Privacy"},
		{"0", pageSettings, "Settings"},
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		label := item.key + " " + item.label
		if s.page == item.page ||
			(s.page == pageSessionDetail && item.page == pageSessions) ||
			((s.page == pageToolCalls || s.page == pageToolCallDetail) && item.page == pageTools) ||
			((s.page == pageAuditFindings || s.page == pageAuditDetail) && item.page == pageAudit) {
			label = inverse(" " + label + " ")
		}
		parts = append(parts, label)
	}
	return strings.Join(parts, "  ")
}

func (s *state) statusLine() string {
	if s.privacyApplying {
		return accent(s.status)
	}
	if s.privacyPending != nil {
		return accent(s.status)
	}
	if s.indexing {
		return accent(s.status)
	}
	if s.loading {
		return accent("loading " + s.page.title() + "...")
	}
	if s.err != nil {
		return danger(s.err.Error())
	}
	if s.status != "" {
		return s.status
	}
	return dim("Ready")
}

func (s *state) footerLine() string {
	var text string
	switch s.page {
	case pageOverview:
		text = "Keys: u source  v model  w project  e range  U clear scope  tab cycle  up/down scroll  r refresh  i update index  I rebuild index  q quit"
	case pageTime:
		text = "Keys: [/]/h/l time tabs  u/v/w/e scope  U clear  up/down scroll  tab cycle pages  r refresh  i update index  I rebuild index  q quit"
	case pageTokens:
		if s.tokensTab == tokensTabBreakdown {
			text = "Keys: [/]/h/l token tabs  d group  u/v/w/e scope  U clear  up/down scroll  r refresh  i update index  I rebuild index  q quit"
			break
		}
		text = "Keys: [/]/h/l token tabs  u/v/w/e scope  U clear  up/down scroll  tab cycle pages  r refresh  i update index  I rebuild index  q quit"
	case pageSessionDetail:
		text = "Keys: b/esc back  up/down scroll  r refresh  i update index  I rebuild index  q quit"
	case pageSessions:
		text = "Keys: enter detail  up/down select  tab cycle  r refresh  i update index  I rebuild index  q quit"
	case pageTools:
		if s.toolsTab == toolsTabShell {
			text = "Keys: [/]/h/l tool tabs  enter detail  v command  u source  e range  d sort  U clear  up/down select  r refresh  i update index  I rebuild index  q quit"
			break
		}
		if s.toolsTab == toolsTabCalls {
			text = "Keys: [/]/h/l tool tabs  enter detail  u source  e range  d sort  U clear  up/down select  r refresh  i update index  I rebuild index  q quit"
			break
		}
		text = "Keys: [/]/h/l tool tabs  enter calls  c all calls  u source  U clear  up/down select  tab cycle  r refresh  i update index  I rebuild index  q quit"
	case pageToolCalls:
		text = "Keys: enter detail  b/esc tools  u source  e range  d sort  U clear  up/down select  r refresh  i update index  I rebuild index  q quit"
	case pageToolCallDetail:
		text = "Keys: b/esc back  up/down scroll  r refresh  i update index  I rebuild index  q quit"
	case pageModelSignals:
		text = "Keys: [/]/h/l signal tabs  u/v/w/e scope  U clear  up/down scroll  tab cycle pages  r refresh  i update index  I rebuild index  q quit"
	case pageModelRisk:
		text = "Keys: u/v/w/e scope  U clear  up/down scroll  tab cycle pages  r refresh  i update index  I rebuild index  q quit"
	case pageAudit:
		text = "Keys: enter detail  f findings  u source  U clear filters  up/down select  tab cycle  r refresh  i update index  I rebuild index  q quit"
	case pageAuditFindings:
		text = "Keys: enter detail  c category  v severity  y shell  u source  U clear  b/esc summary  up/down select  r refresh  i update index  I rebuild index  q quit"
	case pageAuditDetail:
		text = "Keys: b/esc back  u source  U clear filters  up/down scroll  r refresh  i update index  I rebuild index  q quit"
	case pagePrivacy:
		if s.privacyPending != nil {
			text = "Keys: enter write profile  esc cancel  q quit"
			break
		}
		text = "Keys: up/down target  enter recommended  A strict  u defaults  pgup/pgdn detail  r refresh  q quit"
	case pageSettings:
		text = "Keys: up/down scroll  tab cycle  r refresh  i update index  I rebuild index  q quit"
	default:
		text = "Keys: 1-9/0 switch  tab cycle  up/down select/scroll  r refresh  i update index  I rebuild index  q quit"
	}
	if position := s.positionLabel(); position != "" {
		text += "  " + position
	}
	return dim(text)
}

func (s *state) positionLabel() string {
	count := s.itemCount()
	if count <= 0 {
		return ""
	}
	if s.isListPage() {
		visible := s.visibleListRows()
		if count <= visible {
			return ""
		}
		start := s.scroll + 1
		end := s.scroll + visible
		if end > count {
			end = count
		}
		return fmt.Sprintf("Rows %s-%s/%s", formatInt(int64(start)), formatInt(int64(end)), formatInt(int64(count)))
	}
	switch s.page {
	case pageSessionDetail, pageToolCallDetail, pageModelSignals, pageTime, pageTokens, pageModelRisk, pageAuditDetail, pageSettings:
		visible := s.contentHeight()
		if count <= visible {
			return ""
		}
		start := s.scroll + 1
		end := s.scroll + visible
		if end > count {
			end = count
		}
		return fmt.Sprintf("Lines %s-%s/%s", formatInt(int64(start)), formatInt(int64(end)), formatInt(int64(count)))
	default:
		return ""
	}
}

func (s *state) usageScopeLabel() string {
	parts := []string{}
	if strings.TrimSpace(s.usageAgent) != "" {
		parts = append(parts, "source "+filterLabel(s.usageAgent, usageAgentOptions(s.scopeOverview)))
	}
	if strings.TrimSpace(s.usageModel) != "" {
		parts = append(parts, "model "+filterLabel(s.usageModel, usageModelOptions(s.scopeOverview)))
	}
	if strings.TrimSpace(s.usageProject) != "" {
		parts = append(parts, "project "+shortPath(s.usageProject, 28))
	}
	if s.usageRange != usageRangeAll {
		parts = append(parts, "range "+s.usageRange.title())
	}
	if len(parts) == 0 {
		return ""
	}
	return "scope " + strings.Join(parts, ", ")
}

func filterLabel(value string, options []stringOption) string {
	for _, option := range options {
		if option.value == value {
			return option.label
		}
	}
	return value
}

func (s *state) auditFilterLabel() string {
	parts := []string{}
	if strings.TrimSpace(s.auditAgent) != "" {
		parts = append(parts, "source "+filterLabel(s.auditAgent, usageAgentOptions(s.scopeOverview)))
	}
	if strings.TrimSpace(s.auditCategory) != "" {
		parts = append(parts, "category "+filterLabel(s.auditCategory, auditCategoryOptions()))
	}
	if strings.TrimSpace(s.auditSeverity) != "" {
		parts = append(parts, "severity "+filterLabel(s.auditSeverity, auditSeverityOptions()))
	}
	if strings.TrimSpace(s.auditShell) != "" {
		parts = append(parts, "shell "+filterLabel(s.auditShell, auditShellOptions()))
	}
	if len(parts) == 0 {
		return ""
	}
	return "filters " + strings.Join(parts, ", ")
}

func toolActiveFilterLabel(s *state) string {
	parts := []string{}
	if strings.TrimSpace(s.toolAgent) != "" {
		parts = append(parts, "source "+filterLabel(s.toolAgent, usageAgentOptions(s.scopeOverview)))
	}
	if s.toolRange != usageRangeAll && (s.page == pageToolCalls || s.toolsTab == toolsTabShell || s.toolsTab == toolsTabCalls) {
		parts = append(parts, "range "+s.toolRange.title())
	}
	if strings.TrimSpace(s.toolCommand) != "" && s.page == pageTools && (s.toolsTab == toolsTabShell || s.toolsTab == toolsTabCalls) {
		parts = append(parts, "command "+s.toolCommand)
	}
	if len(parts) == 0 {
		return ""
	}
	return "filters " + strings.Join(parts, ", ")
}

func (s *state) contentHeight() int {
	height := s.height
	if height <= 0 {
		height = defaultHeight
	}
	contentHeight := height - 6
	if contentHeight < 4 {
		return 4
	}
	return contentHeight
}

func (s *state) content() []string {
	switch s.page {
	case pageOverview:
		return s.overviewLines()
	case pageTime:
		return s.timeViewportLines()
	case pageTokens:
		return s.tokenViewportLines()
	case pageModelSignals:
		return s.modelSignalViewportLines()
	case pageModelRisk:
		return s.modelRiskViewportLines()
	case pageSessions:
		return s.sessionLines()
	case pageSessionDetail:
		return s.detailLines()
	case pageTools:
		return s.toolLines()
	case pageToolCalls:
		return s.toolCallLines()
	case pageToolCallDetail:
		return s.toolCallDetailViewportLines()
	case pageAudit:
		return s.auditLines()
	case pageAuditFindings:
		return s.auditFindingLines()
	case pageAuditDetail:
		return s.auditDetailViewportLines()
	case pageSettings:
		return s.settingsViewportLines()
	case pagePrivacy:
		return s.privacyViewportLines()
	default:
		return []string{"Unknown page"}
	}
}

func separator(width int) string {
	if width < 20 {
		width = 20
	}
	return strings.Repeat("-", width)
}

func fit(value string, width int) string {
	if width <= 0 {
		width = defaultWidth
	}
	runes := []rune(value)
	if len(runes) <= width {
		return value
	}
	if width <= 3 {
		return string(runes[:width])
	}
	return string(runes[:width-3]) + "..."
}

func truncate(value string, width int) string {
	if width <= 0 {
		return ""
	}
	return fit(value, width)
}

func empty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func formatInt(value int64) string {
	return viewmodel.FormatNumber(value)
}

func formatCost(value *float64) string {
	return viewmodel.FormatCost(value)
}

func formatCostValue(value float64) string {
	return viewmodel.FormatCostValue(value)
}

func formatDuration(ms int64) string {
	return viewmodel.FormatDuration(float64(ms))
}

func formatDurationFloat(ms float64) string {
	return viewmodel.FormatDuration(ms)
}

func formatPercent(value float64) string {
	return viewmodel.FormatPercent(value)
}

func ratio(numerator, denominator float64) float64 {
	if denominator <= 0 {
		return 0
	}
	return numerator / denominator
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return "-"
	}
	return value.Local().Format("01-02 15:04")
}

func formatFullTime(value time.Time) string {
	if value.IsZero() {
		return "-"
	}
	return value.Local().Format("2006-01-02 15:04:05")
}

func shortPath(value string, width int) string {
	if strings.TrimSpace(value) == "" {
		return "unknown"
	}
	return fit(viewmodel.ShortPath(value), width)
}

func sessionLabel(session agentmodel.Session) string {
	return viewmodel.SessionLabel(session)
}

func sessionSourceName(session agentmodel.Session) string {
	return sourceDisplayName(session.SourceLabel, session.AgentName, session.AgentKind, session.SourceKey)
}

func toolCallSourceName(call agentmodel.ToolCall) string {
	return sourceDisplayName(call.SourceLabel, call.AgentName, call.AgentKind, call.SourceKey)
}

func toolCallSessionLabel(call agentmodel.ToolCall) string {
	return viewmodel.SessionLabel(agentmodel.Session{
		ID:             call.SessionID,
		SessionKey:     call.SessionKey,
		CodexSessionID: call.CodexSessionID,
	})
}

func agentUsageSourceName(item agentmodel.AgentUsage) string {
	return sourceDisplayName(item.SourceLabel, item.AgentName, item.AgentKind, item.SourceKey)
}

func agentUsageContext(item agentmodel.AgentUsage) string {
	return sourceContext(item.AgentKind, item.AgentName, item.SourceRootPath, item.SourceSessionsPath)
}

func sourceDisplayName(sourceLabel, agentName, agentKind, sourceKey string) string {
	for _, value := range []string{sourceLabel, agentName, agentKind, sourceKey} {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return "unknown"
}

func sourceContext(agentKind, agentName, rootPath, sessionsPath string) string {
	family := strings.TrimSpace(agentKind)
	if family == "" {
		family = strings.TrimSpace(agentName)
	}
	if family == "" {
		family = "unknown"
	}
	path := strings.TrimSpace(rootPath)
	if path == "" {
		path = strings.TrimSpace(sessionsPath)
	}
	if path == "" {
		return family
	}
	return family + " @ " + viewmodel.ShortPath(path)
}

func bold(value string) string {
	return "\x1b[1m" + value + "\x1b[0m"
}

func dim(value string) string {
	return "\x1b[2m" + value + "\x1b[0m"
}

func inverse(value string) string {
	return "\x1b[7m" + value + "\x1b[0m"
}

func accent(value string) string {
	return "\x1b[36m" + value + "\x1b[0m"
}

func danger(value string) string {
	return "\x1b[31m" + value + "\x1b[0m"
}

func warning(value string) string {
	return "\x1b[33m" + value + "\x1b[0m"
}

func success(value string) string {
	return "\x1b[32m" + value + "\x1b[0m"
}
