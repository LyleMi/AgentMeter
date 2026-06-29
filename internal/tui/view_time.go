package tui

import (
	"fmt"
	"sort"
	"strings"

	agentmodel "AgentMeter/internal/model"
)

func (s *state) timeViewportLines() []string {
	lines := timeLines(s.overview, s.width, s.timeTab)
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

func timeLines(overview agentmodel.Overview, width int, tab timeTab) []string {
	lines := []string{
		bold("Time"),
		timeTabLine(tab, width),
	}
	if overview.TotalSessions == 0 {
		return append(lines, "", "No time analysis yet. Press i to update the index.")
	}

	lines = appendTimeKpiLines(lines, overview)
	switch tab {
	case timeTabSources:
		lines = appendTimeSourceLines(lines, overview.AgentTimeUsage, width)
	case timeTabTools:
		lines = appendTimeToolLines(lines, overview.ToolTimeLeaders, width)
	case timeTabSessions:
		lines = appendTimeSessionLines(lines, overview.SlowSessions, width)
	default:
		lines = appendTimeCompositionLines(lines, overview)
		lines = appendTimeAttributionSummaryLines(lines, overview, width)
	}
	return lines
}

func timeTabLine(active timeTab, width int) string {
	labels := make([]string, 0, len(timeTabs))
	for _, tab := range timeTabs {
		label := tab.title()
		if tab == active {
			label = inverse(" " + label + " ")
		}
		labels = append(labels, label)
	}
	return fit("Tabs: "+strings.Join(labels, "  "), width)
}

func appendTimeKpiLines(lines []string, overview agentmodel.Overview) []string {
	wall := overview.TotalWallDurationMS
	active := overview.TotalActiveDurationMS
	tool := overview.TotalToolDurationMS
	network := overview.SuspectedNetworkToolDurationMS
	slowest := "-"
	slowestNote := "No slow sessions"
	if len(overview.SlowSessions) > 0 {
		session := overview.SlowSessions[0]
		slowest = sessionLabel(session)
		slowestNote = formatDuration(session.WallDurationMS) + " wall"
	}
	return append(lines, "",
		bold("Summary"),
		fmt.Sprintf("Wall time: %-14s Sessions: %-10s Slowest: %-18s %s",
			formatDuration(wall),
			formatInt(int64(overview.TotalSessions)),
			truncate(slowest, 18),
			slowestNote,
		),
		fmt.Sprintf("Active share: %-8s Active: %-12s Model: %-12s Tools: %-12s Idle: %s",
			formatPercent(ratio(float64(active), float64(wall))),
			formatDuration(active),
			formatDuration(overview.TotalModelDurationMS),
			formatDuration(tool),
			formatDuration(overview.TotalIdleDurationMS),
		),
		fmt.Sprintf("Tool share: %-10s Network-likely: %-12s across %s calls",
			formatPercent(ratio(float64(tool), float64(wall))),
			formatPercent(ratio(float64(network), float64(wall))),
			formatInt(int64(overview.SuspectedNetworkToolCalls)),
		),
	)
}

func appendTimeCompositionLines(lines []string, overview agentmodel.Overview) []string {
	wall := overview.TotalWallDurationMS
	network := overview.SuspectedNetworkToolDurationMS
	if network > overview.TotalToolDurationMS {
		network = overview.TotalToolDurationMS
	}
	otherTools := overview.TotalToolDurationMS - network
	if otherTools < 0 {
		otherTools = 0
	}
	return append(lines, "",
		bold("Composition"),
		timeCompositionLine("Model", overview.TotalModelDurationMS, wall),
		timeCompositionLine("Suspected network tools", network, wall),
		timeCompositionLine("Other tools", otherTools, wall),
		timeCompositionLine("Idle / unclassified", overview.TotalIdleDurationMS, wall),
	)
}

func timeCompositionLine(label string, value, total int64) string {
	return fmt.Sprintf("  %-28s %12s %8s", label, formatDuration(value), formatPercent(ratio(float64(value), float64(total))))
}

func appendTimeAttributionSummaryLines(lines []string, overview agentmodel.Overview, width int) []string {
	lines = append(lines, "", bold("Source Time Attribution"))
	lines = appendTimeSourceRows(lines, rankedAgentTimeUsage(overview.AgentTimeUsage), 5, width)
	lines = append(lines, "", bold("Model Time Attribution"))
	lines = appendTimeModelRows(lines, rankedModelTimeUsage(overview.ModelTimeUsage), 5, width)
	return lines
}

func appendTimeSourceLines(lines []string, rows []agentmodel.AgentTimeUsage, width int) []string {
	rows = rankedAgentTimeUsage(rows)
	lines = append(lines, "", bold("Source Time Comparison"))
	if len(rows) == 0 {
		return append(lines, "No source time rows.")
	}
	totalSessions := 0
	totalWall := int64(0)
	for _, row := range rows {
		totalSessions += row.SessionCount
		totalWall += row.WallDurationMS
	}
	top := rows[0]
	lines = append(lines,
		fmt.Sprintf("Sources: %-8s Sessions: %-10s Wall: %-12s Top: %s (%s)",
			formatInt(int64(len(rows))),
			formatInt(int64(totalSessions)),
			formatDuration(totalWall),
			truncate(agentTimeSourceName(top), 24),
			formatPercent(ratio(float64(top.WallDurationMS), float64(totalWall))),
		),
	)
	return appendTimeSourceRows(lines, rows, 12, width)
}

func appendTimeSourceRows(lines []string, rows []agentmodel.AgentTimeUsage, limit int, width int) []string {
	if len(rows) == 0 {
		return append(lines, "No source time rows.")
	}
	lines = append(lines, fit(fmt.Sprintf("  %-18s %-25s %8s %7s %11s %11s %11s %11s %9s %11s",
		"Source", "Family/Path", "Sessions", "Calls", "Wall", "Active", "Model", "Tool", "Network", "Idle"), width))
	for _, row := range limitSlice(rows, limit) {
		lines = append(lines, fit(fmt.Sprintf("  %-18s %-25s %8s %7s %11s %11s %11s %11s %9s %11s",
			truncate(agentTimeSourceName(row), 18),
			truncate(sourceContext(row.AgentKind, row.AgentName, row.SourceRootPath, row.SourceSessionsPath), 25),
			formatInt(int64(row.SessionCount)),
			formatInt(int64(row.ToolCalls)),
			formatDuration(row.WallDurationMS),
			formatDuration(row.ActiveDurationMS),
			formatDuration(row.ModelDurationMS),
			formatDuration(row.ToolDurationMS),
			formatDuration(row.SuspectedNetworkToolDurationMS),
			formatDuration(row.IdleDurationMS),
		), width))
	}
	return lines
}

func appendTimeModelRows(lines []string, rows []agentmodel.ModelTimeUsage, limit int, width int) []string {
	if len(rows) == 0 {
		return append(lines, "No model time rows.")
	}
	lines = append(lines, fit(fmt.Sprintf("  %-26s %8s %11s %11s %11s %11s %11s %11s",
		"Model", "Sessions", "Tokens", "Wall", "Active", "Model", "Tool", "Idle"), width))
	for _, row := range limitSlice(rows, limit) {
		lines = append(lines, fit(fmt.Sprintf("  %-26s %8s %11s %11s %11s %11s %11s %11s",
			truncate(empty(row.Model, "unknown"), 26),
			formatInt(int64(row.SessionCount)),
			formatInt(row.TotalTokens),
			formatDuration(row.WallDurationMS),
			formatDuration(row.ActiveDurationMS),
			formatDuration(row.ModelDurationMS),
			formatDuration(row.ToolDurationMS),
			formatDuration(row.IdleDurationMS),
		), width))
	}
	return lines
}

func appendTimeToolLines(lines []string, rows []agentmodel.ToolTimeUsage, width int) []string {
	rows = rankedToolTimeUsage(rows)
	lines = append(lines, "", bold("Tool Duration Leaders"))
	if len(rows) == 0 {
		return append(lines, "No tool duration rows.")
	}
	lines = append(lines, dim("Network-likely is inferred from tool names and shell/network activity."))
	lines = append(lines, fit(fmt.Sprintf("  %-30s %8s %8s %8s %12s %12s %12s %8s",
		"Tool", "Calls", "Success", "Failed", "Total", "Average", "Max", "Network"), width))
	for _, row := range limitSlice(rows, 16) {
		network := "no"
		if row.SuspectedNetwork {
			network = "likely"
		}
		lines = append(lines, fit(fmt.Sprintf("  %-30s %8s %8s %8s %12s %12s %12s %8s",
			truncate(empty(row.ToolName, "unknown"), 30),
			formatInt(int64(row.Calls)),
			formatInt(int64(row.SuccessCalls)),
			formatInt(int64(row.FailedCalls)),
			formatDuration(row.TotalDurationMS),
			formatDurationFloat(row.AvgDurationMS),
			formatDuration(row.MaxDurationMS),
			network,
		), width))
	}
	return lines
}

func appendTimeSessionLines(lines []string, rows []agentmodel.Session, width int) []string {
	lines = append(lines, "", bold("Slow Sessions"))
	if len(rows) == 0 {
		return append(lines, "No slow sessions yet.")
	}
	lines = append(lines, fit("  Started          Source       Model              Wall       Active     Model      Tool       Project / session", width))
	for _, session := range limitSlice(rows, 16) {
		lines = append(lines, fit(fmt.Sprintf("  %-16s %-12s %-18s %10s %10s %10s %10s  %s",
			formatTime(session.StartedAt),
			truncate(sessionSourceName(session), 12),
			truncate(empty(session.Model, "unknown"), 18),
			formatDuration(session.WallDurationMS),
			formatDuration(session.ActiveDurationMS),
			formatDuration(session.ModelDurationMS),
			formatDuration(session.ToolDurationMS),
			truncate(shortPath(session.ProjectPath, 22)+" / "+sessionLabel(session), 44),
		), width))
	}
	return lines
}

func rankedAgentTimeUsage(rows []agentmodel.AgentTimeUsage) []agentmodel.AgentTimeUsage {
	result := append([]agentmodel.AgentTimeUsage(nil), rows...)
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].WallDurationMS == result[j].WallDurationMS {
			return agentTimeSourceName(result[i]) < agentTimeSourceName(result[j])
		}
		return result[i].WallDurationMS > result[j].WallDurationMS
	})
	return result
}

func rankedModelTimeUsage(rows []agentmodel.ModelTimeUsage) []agentmodel.ModelTimeUsage {
	result := append([]agentmodel.ModelTimeUsage(nil), rows...)
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].WallDurationMS == result[j].WallDurationMS {
			return result[i].Model < result[j].Model
		}
		return result[i].WallDurationMS > result[j].WallDurationMS
	})
	return result
}

func rankedToolTimeUsage(rows []agentmodel.ToolTimeUsage) []agentmodel.ToolTimeUsage {
	result := append([]agentmodel.ToolTimeUsage(nil), rows...)
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].TotalDurationMS == result[j].TotalDurationMS {
			return result[i].ToolName < result[j].ToolName
		}
		return result[i].TotalDurationMS > result[j].TotalDurationMS
	})
	return result
}

func agentTimeSourceName(item agentmodel.AgentTimeUsage) string {
	return sourceDisplayName(item.SourceLabel, item.AgentName, item.AgentKind, item.SourceKey)
}
