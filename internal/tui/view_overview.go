package tui

import (
	"fmt"

	"github.com/LyleMi/AgentMeter/internal/viewmodel"
)

func (s *state) overviewLines() []string {
	o := s.overview
	lines := []string{
		bold("Totals"),
		fmt.Sprintf("Sessions: %-12s Tokens: %-14s Cost: %s", formatInt(int64(o.TotalSessions)), formatInt(o.TotalTokens), formatCost(o.EstimatedCostUSD)),
		fmt.Sprintf("Input: %-15s Cached input: %-8s Output: %-12s Reasoning: %s",
			formatInt(o.TotalInputTokens), formatInt(o.TotalCachedInputTokens), formatInt(o.TotalOutputTokens), formatInt(o.TotalReasoningTokens)),
		fmt.Sprintf("Wall: %-15s Active: %-12s Model: %-12s Tools: %-12s Idle: %s",
			formatDuration(o.TotalWallDurationMS), formatDuration(o.TotalActiveDurationMS), formatDuration(o.TotalModelDurationMS), formatDuration(o.TotalToolDurationMS), formatDuration(o.TotalIdleDurationMS)),
		fmt.Sprintf("Tool calls: %-10s Network-suspect: %-10s Unpriced sessions: %s",
			formatInt(int64(o.TotalToolCalls)), formatInt(int64(o.SuspectedNetworkToolCalls)), formatInt(int64(o.UnpricedSessions))),
		"",
	}
	lines = append(lines, bold("Efficiency"))
	metrics := viewmodel.DerivedOverviewMetrics(o)
	if len(metrics) == 0 {
		lines = append(lines, "No derived metrics yet.")
	} else {
		for index := 0; index < len(metrics); index += 2 {
			left := metrics[index]
			if index+1 >= len(metrics) {
				lines = append(lines, fmt.Sprintf("%-24s %12s", left.Label, left.Value))
				continue
			}
			right := metrics[index+1]
			lines = append(lines, fmt.Sprintf("%-24s %12s    %-24s %12s", left.Label, left.Value, right.Label, right.Value))
		}
	}
	lines = append(lines, "", bold("Top Models"))
	if len(o.ModelUsage) == 0 {
		lines = append(lines, "No model usage yet.")
	} else {
		lines = append(lines, fmt.Sprintf("%-26s %8s %11s %11s %11s %12s", "Model", "Sessions", "Input", "Output", "Tokens", "Cost"))
		for _, item := range limitSlice(o.ModelUsage, 6) {
			lines = append(lines, fmt.Sprintf("%-26s %8s %11s %11s %11s %12s",
				truncate(empty(item.Model, "unknown"), 26),
				formatInt(int64(item.SessionCount)),
				formatInt(item.InputTokens),
				formatInt(item.OutputTokens),
				formatInt(item.TotalTokens),
				formatCost(item.EstimatedCostUSD),
			))
		}
	}
	if len(o.AgentUsage) > 0 {
		lines = append(lines, "", bold("Top Agents"))
		lines = append(lines, fmt.Sprintf("%-18s %-26s %8s %12s %7s %10s", "Source", "Family/Path", "Sessions", "Tokens", "Tools", "Cost"))
		for _, item := range limitSlice(o.AgentUsage, 4) {
			lines = append(lines, fmt.Sprintf("%-18s %-26s %8s %12s %7s %10s",
				truncate(agentUsageSourceName(item), 18),
				truncate(agentUsageContext(item), 26),
				formatInt(int64(item.SessionCount)),
				formatInt(item.TotalTokens),
				formatInt(int64(item.ToolCalls)),
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

func limitSlice[T any](items []T, limit int) []T {
	if limit < 0 || len(items) <= limit {
		return items
	}
	return items[:limit]
}
