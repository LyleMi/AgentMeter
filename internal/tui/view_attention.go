package tui

import (
	"fmt"
	"strings"

	agentmodel "github.com/LyleMi/AgentMeter/internal/model"
)

func (s *state) attentionLines() []string {
	response := s.attention
	lines := []string{
		bold("Selected period"),
		fmt.Sprintf("Current  %s -> %s", shortAttentionTime(response.Window.Current.From), shortAttentionTime(response.Window.Current.To)),
		fmt.Sprintf("Baseline %s -> %s", shortAttentionTime(response.Window.Baseline.From), shortAttentionTime(response.Window.Baseline.To)),
		"",
		bold("Status snapshot"),
		fmt.Sprintf("Sessions %-10s %s", formatInt(int64(response.Snapshot.Sessions.Current)), attentionChange(response.Snapshot.Sessions)),
		fmt.Sprintf("Tokens   %-10s %s", formatInt(int64(response.Snapshot.Tokens.Current)), attentionChange(response.Snapshot.Tokens)),
		fmt.Sprintf("Cost     %-10s %s", formatCostValue(response.Snapshot.CostUSD.Current), attentionChange(response.Snapshot.CostUSD)),
		fmt.Sprintf("Active   %-10s %s", formatDuration(int64(response.Snapshot.ActiveTime.Current)), attentionChange(response.Snapshot.ActiveTime)),
		fmt.Sprintf("Tools    %-10s %s", formatInt(int64(response.Snapshot.ToolCalls.Current)), attentionChange(response.Snapshot.ToolCalls)),
		"",
		fmt.Sprintf("%s  %s critical  %s warning", bold("Needs attention"), formatInt(int64(response.Counts.Critical)), formatInt(int64(response.Counts.Warning))),
	}
	if len(response.Items) == 0 {
		return append(lines, success("No issues need attention in the current range."))
	}
	lines = append(lines, fmt.Sprintf("%-3s %-9s %-24s %-46s", "", "Severity", "Subject", "Reason"))
	start, end := attentionVisibleWindow(len(response.Items), s.selected, max(1, s.visibleListRows()-11))
	for index := start; index < end; index++ {
		item := response.Items[index]
		cursor := " "
		if index == s.selected {
			cursor = ">"
		}
		severity := strings.ToUpper(item.Severity)
		if item.Severity == "critical" {
			severity = danger(severity)
		} else {
			severity = warning(severity)
		}
		lines = append(lines, fmt.Sprintf("%-3s %-18s %-24s %-46s",
			cursor, severity, truncate(item.Subject, 24), truncate(item.Reason, 46)))
	}
	return lines
}

func attentionVisibleWindow(count, selected, visible int) (int, int) {
	if visible >= count {
		return 0, count
	}
	start := selected - visible + 1
	if start < 0 {
		start = 0
	}
	end := start + visible
	if end > count {
		end = count
		start = end - visible
	}
	return start, end
}

func shortAttentionTime(value string) string {
	if len(value) >= 10 {
		return value[:10]
	}
	return value
}

func attentionChange(metric agentmodel.AttentionMetric) string {
	if metric.ChangePct == nil {
		return dim("no baseline")
	}
	return fmt.Sprintf("%+.0f%% vs baseline", *metric.ChangePct*100)
}
