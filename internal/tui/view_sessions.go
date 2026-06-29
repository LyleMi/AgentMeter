package tui

import (
	"fmt"
	"strconv"

	agentmodel "github.com/LyleMi/AgentMeter/internal/model"
)

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

func sessionHeader(width int) string {
	return fit("  Started          Source     Model              Tokens       Cost     Tools  Project", width)
}

func sessionRow(item agentmodel.Session, selected bool, width int) string {
	prefix := "  "
	if selected {
		prefix = "> "
	}
	return fit(fmt.Sprintf("%s%-16s %-10s %-18s %10s %10s %5s  %s",
		prefix,
		formatTime(item.StartedAt),
		truncate(sessionSourceName(item), 10),
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
		"Source: " + sessionSourceName(session) + "  Family: " + empty(session.AgentKind, "unknown") + "  Agent: " + empty(session.AgentName, "unknown"),
		"Source root: " + empty(session.SourceRootPath, "unknown"),
		"Source sessions: " + empty(session.SourceSessionsPath, "unknown"),
		"Raw source: " + empty(session.RawSourcePath, "unknown"),
		"Model: " + empty(session.Model, "unknown"),
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
