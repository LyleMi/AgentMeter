package tui

import (
	"fmt"
	"strconv"
	"strings"

	agentmodel "AgentMeter/internal/model"
)

func (s *state) toolLines() []string {
	lines := []string{
		bold("Tools"),
		dim("Enter opens calls for the selected tool; c shows all recent tool calls."),
	}
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

func (s *state) toolCallLines() []string {
	scope := "all tools"
	if strings.TrimSpace(s.toolCallTool) != "" {
		scope = s.toolCallTool
	}
	lines := []string{
		bold("Tool Calls"),
		fmt.Sprintf("Scope: %s  Sort: %s", scope, toolCallSortLabel(s.toolCallSort)),
	}
	if len(s.toolCalls) == 0 {
		if s.loading {
			return lines
		}
		return append(lines, "No matching tool calls found.")
	}
	lines = append(lines, toolCallHeader(s.width))
	visible := s.contentHeight() - len(lines)
	if visible < 1 {
		visible = 1
	}
	if s.scroll > len(s.toolCalls)-1 {
		s.scroll = len(s.toolCalls) - 1
	}
	end := s.scroll + visible
	if end > len(s.toolCalls) {
		end = len(s.toolCalls)
	}
	for i := s.scroll; i < end; i++ {
		lines = append(lines, toolCallRow(s.toolCalls[i], i == s.selected, s.width))
	}
	return lines
}

func (s *state) toolCallDetailViewportLines() []string {
	if s.toolCall == nil {
		return []string{bold("Tool Call Detail")}
	}
	lines := toolCallDetailLines(*s.toolCall, s.width)
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

func toolCallHeader(width int) string {
	return fit("  Started          Tool               Source       Status       Duration   Session     Input", width)
}

func toolCallRow(call agentmodel.ToolCall, selected bool, width int) string {
	prefix := "  "
	if selected {
		prefix = "> "
	}
	return fit(fmt.Sprintf("%s%-16s %-18s %-12s %-10s %9s %-11s %s",
		prefix,
		formatTime(call.StartedAt),
		truncate(empty(call.ToolName, "unknown"), 18),
		truncate(toolCallSourceName(call), 12),
		truncate(empty(call.Status, "unknown"), 10),
		formatDuration(call.DurationMS),
		truncate(toolCallSessionLabel(call), 11),
		truncate(toolCallPrimarySummary(call), 40),
	), width)
}

func toolCallDetailLines(call agentmodel.ToolCall, width int) []string {
	lines := []string{
		bold("Tool Call"),
		"ID: " + strconv.FormatInt(call.ID, 10) + "  Session: " + toolCallSessionLabel(call),
		"Tool: " + empty(call.ToolName, "unknown") + "  Status: " + empty(call.Status, "unknown") + "  Duration: " + formatDuration(call.DurationMS),
		"Source: " + toolCallSourceName(call) + "  Family: " + empty(call.AgentKind, "unknown") + "  Agent: " + empty(call.AgentName, "unknown"),
		"Started: " + formatFullTime(call.StartedAt) + "  Ended: " + formatFullTime(call.EndedAt),
		"Project: " + empty(call.ProjectPath, "unknown"),
		"Raw source: " + empty(call.RawSourcePath, "unknown"),
	}
	if strings.TrimSpace(call.CallID) != "" {
		lines = append(lines, "Call ID: "+call.CallID)
	}
	lines = appendRawEventLines(lines, call)
	lines = append(lines, bold("Input"))
	lines = appendToolCallValue(lines, call.InputSummary, width)
	lines = append(lines, bold("Output"))
	lines = appendToolCallValue(lines, call.OutputSummary, width)
	if strings.TrimSpace(call.Error) != "" {
		lines = append(lines, bold("Error"))
		lines = appendToolCallValue(lines, call.Error, width)
	}
	if strings.TrimSpace(call.RawStartEventSummary) != "" || strings.TrimSpace(call.RawEndEventSummary) != "" {
		lines = append(lines, bold("Raw Event Summaries"))
		if strings.TrimSpace(call.RawStartEventSummary) != "" {
			lines = append(lines, fit("Start: "+call.RawStartEventSummary, width))
		}
		if strings.TrimSpace(call.RawEndEventSummary) != "" {
			lines = append(lines, fit("End: "+call.RawEndEventSummary, width))
		}
	}
	return lines
}

func appendRawEventLines(lines []string, call agentmodel.ToolCall) []string {
	parts := []string{}
	if call.RawEventID > 0 {
		parts = append(parts, "event #"+strconv.FormatInt(call.RawEventID, 10))
	}
	if call.RawStartEventID > 0 {
		parts = append(parts, "start #"+strconv.FormatInt(call.RawStartEventID, 10))
	}
	if call.RawEndEventID > 0 {
		parts = append(parts, "end #"+strconv.FormatInt(call.RawEndEventID, 10))
	}
	if call.RawEventLine > 0 {
		parts = append(parts, "line "+strconv.Itoa(call.RawEventLine))
	}
	if call.RawStartEventLine > 0 {
		parts = append(parts, "start line "+strconv.Itoa(call.RawStartEventLine))
	}
	if call.RawEndEventLine > 0 {
		parts = append(parts, "end line "+strconv.Itoa(call.RawEndEventLine))
	}
	if len(parts) > 0 {
		lines = append(lines, "Raw events: "+strings.Join(parts, "  "))
	}
	if strings.TrimSpace(call.RawStartEventType) != "" || strings.TrimSpace(call.RawEndEventType) != "" {
		lines = append(lines, "Raw event types: start="+empty(call.RawStartEventType, "-")+"  end="+empty(call.RawEndEventType, "-"))
	}
	return lines
}

func appendToolCallValue(lines []string, value string, width int) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return append(lines, "-")
	}
	for _, line := range strings.Split(value, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lines = append(lines, fit(line, width))
	}
	if len(lines) == 0 {
		return append(lines, "-")
	}
	return lines
}

func toolCallPrimarySummary(call agentmodel.ToolCall) string {
	for _, value := range []string{call.InputSummary, call.OutputSummary, call.Error} {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return "-"
}

func toolCallSortLabel(sort string) string {
	switch strings.TrimSpace(sort) {
	case "duration_desc":
		return "duration high to low"
	case "duration_asc":
		return "duration low to high"
	default:
		return "recent first"
	}
}
