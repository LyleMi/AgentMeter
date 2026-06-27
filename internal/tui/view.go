package tui

import (
	"fmt"
	"strconv"
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
		fit(bold("AgentMeter")+" "+dim(s.page.title()), width),
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

func (s *state) navLine() string {
	items := []struct {
		key   string
		page  page
		label string
	}{
		{"1", pageOverview, "Overview"},
		{"2", pageSessions, "Sessions"},
		{"3", pageTools, "Tools"},
		{"4", pageSettings, "Settings"},
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		label := item.key + " " + item.label
		if s.page == item.page || (s.page == pageSessionDetail && item.page == pageSessions) {
			label = inverse(" " + label + " ")
		}
		parts = append(parts, label)
	}
	return strings.Join(parts, "  ")
}

func (s *state) statusLine() string {
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
	if s.page == pageSessionDetail {
		return dim("Keys: b/esc back  up/down scroll  r refresh  i update index  I rebuild index  q quit")
	}
	return dim("Keys: 1-4 switch  tab cycle  up/down select  enter detail  r refresh  i update index  I rebuild index  q quit")
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
	case pageSettings:
		return s.settingsViewportLines()
	default:
		return []string{"Unknown page"}
	}
}

func (s *state) overviewLines() []string {
	o := s.overview
	lines := []string{
		bold("Totals"),
		fmt.Sprintf("Sessions: %-12s Tokens: %-14s Cost: %s", formatInt(int64(o.TotalSessions)), formatInt(o.TotalTokens), formatCost(o.EstimatedCostUSD)),
		fmt.Sprintf("Input: %-15s Cached input: %-8s Output: %-12s Reasoning: %s",
			formatInt(o.TotalInputTokens), formatInt(o.TotalCachedInputTokens), formatInt(o.TotalOutputTokens), formatInt(o.TotalReasoningTokens)),
		fmt.Sprintf("Wall time: %-12s Active time: %-12s Tool calls: %-8s Unpriced sessions: %s",
			formatDuration(o.TotalWallDurationMS), formatDuration(o.TotalActiveDurationMS), formatInt(int64(o.TotalToolCalls)), formatInt(int64(o.UnpricedSessions))),
		"",
		bold("Top Models"),
	}
	if len(o.ModelUsage) == 0 {
		lines = append(lines, "No model usage yet.")
	} else {
		lines = append(lines, fmt.Sprintf("%-28s %8s %12s %12s", "Model", "Sessions", "Tokens", "Cost"))
		for _, item := range limitSlice(o.ModelUsage, 6) {
			lines = append(lines, fmt.Sprintf("%-28s %8s %12s %12s",
				truncate(empty(item.Model, "unknown"), 28),
				formatInt(int64(item.SessionCount)),
				formatInt(item.TotalTokens),
				formatCost(item.EstimatedCostUSD),
			))
		}
	}
	lines = append(lines, "", bold("Recent Sessions"))
	if len(o.RecentSessions) == 0 {
		lines = append(lines, "No sessions indexed yet. Press i to update the index.")
		return lines
	}
	lines = append(lines, sessionHeader(s.width))
	for _, item := range limitSlice(o.RecentSessions, 6) {
		lines = append(lines, sessionRow(item, false, s.width))
	}
	return lines
}

func (s *state) sessionLines() []string {
	lines := []string{bold("Sessions")}
	if len(s.sessions) == 0 {
		if s.loading {
			return lines
		}
		return append(lines, "No sessions found. Press i to update the index.")
	}
	lines = append(lines, sessionHeader(s.width))
	visible := s.contentHeight() - len(lines)
	if visible < 1 {
		visible = 1
	}
	if s.scroll > len(s.sessions)-1 {
		s.scroll = len(s.sessions) - 1
	}
	end := s.scroll + visible
	if end > len(s.sessions) {
		end = len(s.sessions)
	}
	for i := s.scroll; i < end; i++ {
		lines = append(lines, sessionRow(s.sessions[i], i == s.selected, s.width))
	}
	return lines
}

func (s *state) detailLines() []string {
	if s.detail == nil {
		return []string{bold("Session Detail")}
	}
	lines := sessionDetailLines(*s.detail, s.width)
	height := s.contentHeight()
	if s.scroll >= len(lines) {
		s.scroll = len(lines) - 1
	}
	if s.scroll < 0 {
		s.scroll = 0
	}
	end := s.scroll + height
	if end > len(lines) {
		end = len(lines)
	}
	return lines[s.scroll:end]
}

func (s *state) toolLines() []string {
	lines := []string{bold("Tools")}
	if len(s.tools) == 0 {
		if s.loading {
			return lines
		}
		return append(lines, "No tool calls found.")
	}
	lines = append(lines, fmt.Sprintf("  %-30s %8s %8s %8s %12s %12s", "Tool", "Calls", "Success", "Failed", "Total", "Average"))
	visible := s.contentHeight() - len(lines)
	if visible < 1 {
		visible = 1
	}
	end := s.scroll + visible
	if end > len(s.tools) {
		end = len(s.tools)
	}
	for i := s.scroll; i < end; i++ {
		item := s.tools[i]
		prefix := "  "
		if i == s.selected {
			prefix = "> "
		}
		lines = append(lines, fmt.Sprintf("%s%-30s %8s %8s %8s %12s %12s",
			prefix,
			truncate(empty(item.ToolName, "unknown"), 30),
			formatInt(int64(item.Calls)),
			formatInt(int64(item.SuccessCalls)),
			formatInt(int64(item.FailedCalls)),
			formatDuration(item.TotalDurationMS),
			formatDuration(int64(item.AvgDurationMS)),
		))
	}
	return lines
}

func (s *state) settingsViewportLines() []string {
	lines := settingsLines(s.settings, s.width)
	height := s.contentHeight()
	if s.scroll >= len(lines) {
		s.scroll = len(lines) - 1
	}
	if s.scroll < 0 {
		s.scroll = 0
	}
	end := s.scroll + height
	if end > len(lines) {
		end = len(lines)
	}
	return lines[s.scroll:end]
}

func sessionHeader(width int) string {
	return fit("  Started          Agent      Model              Tokens       Cost     Tools  Project", width)
}

func sessionRow(item agentmodel.Session, selected bool, width int) string {
	prefix := "  "
	if selected {
		prefix = "> "
	}
	return fit(fmt.Sprintf("%s%-16s %-10s %-18s %10s %10s %5s  %s",
		prefix,
		formatTime(item.StartedAt),
		truncate(empty(item.AgentName, item.AgentKind), 10),
		truncate(empty(item.Model, "unknown"), 18),
		formatInt(item.TokenUsage.TotalTokens),
		formatCost(item.EstimatedCostUSD),
		formatInt(int64(item.ToolCallCount)),
		shortPath(item.ProjectPath, 30),
	), width)
}

func sessionDetailLines(detail agentmodel.SessionDetail, width int) []string {
	session := detail.Session
	lines := []string{
		bold("Session"),
		"ID: " + strconv.FormatInt(session.ID, 10) + "  Label: " + sessionLabel(session),
		"Agent: " + empty(session.AgentName, session.AgentKind) + "  Model: " + empty(session.Model, "unknown"),
		"Project: " + empty(session.ProjectPath, "unknown"),
		"Started: " + formatFullTime(session.StartedAt) + "  Ended: " + formatFullTime(session.EndedAt),
		"Wall: " + formatDuration(session.WallDurationMS) + "  Active: " + formatDuration(session.ActiveDurationMS) + "  Model: " + formatDuration(session.ModelDurationMS) + "  Tools: " + formatDuration(session.ToolDurationMS),
		"Tokens: " + formatInt(session.TokenUsage.TotalTokens) + "  Input: " + formatInt(session.TokenUsage.InputTokens) + "  Output: " + formatInt(session.TokenUsage.OutputTokens) + "  Cost: " + formatCost(session.EstimatedCostUSD),
		bold("Model Calls"),
	}
	if len(detail.ModelCalls) == 0 {
		lines = append(lines, "No model calls.")
	} else {
		lines = append(lines, fmt.Sprintf("%-16s %-18s %10s %10s %10s", "Started", "Model", "Duration", "Tokens", "Cost"))
		for _, call := range detail.ModelCalls {
			lines = append(lines, fmt.Sprintf("%-16s %-18s %10s %10s %10s",
				formatTime(call.StartedAt),
				truncate(empty(call.Model, "unknown"), 18),
				formatDuration(call.DurationMS),
				formatInt(call.TotalTokens),
				formatCost(call.CostUSD),
			))
		}
	}
	lines = append(lines, bold("Tool Calls"))
	if len(detail.ToolCalls) == 0 {
		lines = append(lines, "No tool calls.")
	} else {
		lines = append(lines, fmt.Sprintf("%-16s %-24s %-10s %10s %s", "Started", "Tool", "Status", "Duration", "Input"))
		for _, call := range detail.ToolCalls {
			lines = append(lines, fit(fmt.Sprintf("%-16s %-24s %-10s %10s %s",
				formatTime(call.StartedAt),
				truncate(empty(call.ToolName, "unknown"), 24),
				truncate(empty(call.Status, "unknown"), 10),
				formatDuration(call.DurationMS),
				truncate(call.InputSummary, width-68),
			), width))
		}
	}
	lines = append(lines, bold("Events"))
	if len(detail.Events) == 0 {
		lines = append(lines, "No events.")
		return lines
	}
	lines = append(lines, fmt.Sprintf("%-16s %-12s %s", "Time", "Kind", "Summary"))
	for _, event := range detail.Events {
		lines = append(lines, fit(fmt.Sprintf("%-16s %-12s %s",
			formatTime(event.Timestamp),
			truncate(empty(event.Kind, event.RawType), 12),
			event.Summary,
		), width))
	}
	return lines
}

func settingsLines(settings agentmodel.Settings, width int) []string {
	lines := []string{
		bold("Settings"),
		"Database: " + empty(settings.DatabasePath, "unknown"),
		"",
		bold("Source Paths"),
	}
	if len(settings.SourceEntries) == 0 {
		lines = append(lines, "No source paths configured.")
	} else {
		for _, entry := range settings.SourceEntries {
			state := "disabled"
			if entry.Enabled {
				state = "enabled "
			}
			lines = append(lines, fmt.Sprintf("[%s] %s", state, entry.Path))
		}
	}
	lines = append(lines, "", bold("Last Index"))
	if settings.LastIndexStartedAt == nil {
		lines = append(lines, "No index run recorded.")
	} else {
		lines = append(lines, "Started: "+formatFullTime(*settings.LastIndexStartedAt))
	}
	if settings.LastIndexResult != nil {
		result := settings.LastIndexResult
		lines = append(lines, fmt.Sprintf("Files seen: %s  Indexed: %s  Skipped: %s  Failed: %s  Sessions: %s  Duration: %s",
			formatInt(int64(result.FilesSeen)),
			formatInt(int64(result.Indexed)),
			formatInt(int64(result.Skipped)),
			formatInt(int64(result.Failed)),
			formatInt(int64(result.Sessions)),
			formatDuration(result.DurationMS),
		))
		if len(result.Warnings) > 0 {
			lines = append(lines, "Warnings:")
			for _, warning := range result.Warnings {
				lines = append(lines, fit("- "+warning, width))
			}
		}
	}
	lines = append(lines, "", bold("Pricing Models"))
	if len(settings.PricingModels) == 0 {
		lines = append(lines, "No pricing models configured.")
		return lines
	}
	lines = append(lines, fmt.Sprintf("%-28s %12s %12s %12s", "Model", "Input/1M", "Cached/1M", "Output/1M"))
	for _, item := range settings.PricingModels {
		lines = append(lines, fmt.Sprintf("%-28s %12.4f %12.4f %12.4f",
			truncate(item.Model, 28),
			item.InputPer1M,
			item.CachedInputPer1M,
			item.OutputPer1M,
		))
	}
	return lines
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

func limitSlice[T any](items []T, limit int) []T {
	if limit < 0 || len(items) <= limit {
		return items
	}
	return items[:limit]
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
