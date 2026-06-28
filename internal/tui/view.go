package tui

import (
	"fmt"
	"strings"
	"time"

	agentmodel "AgentMeter/internal/model"
	"AgentMeter/internal/viewmodel"
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
	switch s.page {
	case pageOverview:
		parts = append(parts,
			fmt.Sprintf("%s sessions", formatInt(int64(s.overview.TotalSessions))),
			fmt.Sprintf("%s tokens", formatInt(s.overview.TotalTokens)),
			formatCost(s.overview.EstimatedCostUSD),
		)
	case pageSessions:
		parts = append(parts, fmt.Sprintf("%s sessions loaded", formatInt(int64(len(s.sessions)))))
	case pageTools:
		parts = append(parts, fmt.Sprintf("%s tools", formatInt(int64(len(s.tools)))))
	case pageToolCalls:
		parts = append(parts, fmt.Sprintf("%s calls", formatInt(int64(len(s.toolCalls)))))
		if strings.TrimSpace(s.toolCallTool) != "" {
			parts = append(parts, "tool "+s.toolCallTool)
		}
		parts = append(parts, toolCallSortLabel(s.toolCallSort))
	case pageToolCallDetail:
		if s.toolCall != nil {
			parts = append(parts, empty(s.toolCall.ToolName, "unknown"), "#"+formatInt(s.toolCall.ID))
		}
	case pageModelSignals:
		summary := s.signals.HealthSummary
		parts = append(parts,
			"health "+modelSignalSeverityLabel(summary.Severity),
			fmt.Sprintf("%s sessions", formatInt(int64(s.signals.TotalSessions))),
			fmt.Sprintf("%s calls", formatInt(int64(s.signals.TotalModelCalls))),
		)
	case pagePrivacy:
		parts = append(parts, fmt.Sprintf("%s targets", formatInt(int64(len(s.privacy)))))
		if status := s.selectedPrivacyStatus(); status != nil {
			parts = append(parts, "selected "+privacyDisplayName(*status))
		}
	case pageSettings:
		if len(s.settings.SourceEntries) > 0 {
			parts = append(parts, fmt.Sprintf("%s sources", formatInt(int64(len(s.settings.SourceEntries)))))
		}
	}
	return strings.Join(parts, "  ")
}

func (s *state) navLine() string {
	items := []struct {
		key   string
		page  page
		label string
	}{
		{"1", pageOverview, "Overview"},
		{"2", pageSessions, "Sessions"},
		{"3", pageTools, "Tools"},
		{"m", pageModelSignals, "Model Signals"},
		{"4", pageSettings, "Settings"},
		{"5", pagePrivacy, "Agent Privacy"},
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		label := item.key + " " + item.label
		if s.page == item.page || (s.page == pageSessionDetail && item.page == pageSessions) || ((s.page == pageToolCalls || s.page == pageToolCallDetail) && item.page == pageTools) {
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
	case pageSessionDetail:
		text = "Keys: b/esc back  up/down scroll  r refresh  i update index  I rebuild index  q quit"
	case pageSessions:
		text = "Keys: enter detail  up/down select  tab cycle  r refresh  i update index  I rebuild index  q quit"
	case pageTools:
		text = "Keys: enter calls  c all calls  up/down select  tab cycle  r refresh  i update index  I rebuild index  q quit"
	case pageToolCalls:
		text = "Keys: enter detail  b/esc tools  d sort  up/down select  r refresh  i update index  I rebuild index  q quit"
	case pageToolCallDetail:
		text = "Keys: b/esc calls  up/down scroll  r refresh  i update index  I rebuild index  q quit"
	case pageModelSignals:
		text = "Keys: up/down scroll  tab cycle  r refresh  i update index  I rebuild index  q quit"
	case pagePrivacy:
		if s.privacyPending != nil {
			text = "Keys: enter write profile  esc cancel  q quit"
			break
		}
		text = "Keys: up/down target  enter recommended  A strict  u defaults  pgup/pgdn detail  r refresh  q quit"
	case pageSettings:
		text = "Keys: up/down scroll  tab cycle  r refresh  i update index  I rebuild index  q quit"
	default:
		text = "Keys: 1-5/m switch  tab cycle  up/down select/scroll  r refresh  i update index  I rebuild index  q quit"
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
	case pageSessionDetail, pageToolCallDetail, pageModelSignals, pageSettings:
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
	case pageModelSignals:
		return s.modelSignalViewportLines()
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

func formatDuration(ms int64) string {
	return viewmodel.FormatDuration(float64(ms))
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
