package tui

import (
	"fmt"
	"strings"

	agentmodel "github.com/LyleMi/AgentMeter/internal/model"
)

func (s *state) attentionLines() []string {
	response := s.attention
	lines := []string{
		fmt.Sprintf("Sessions %s  ·  Tokens %s  ·  Cost %s", formatInt(int64(response.Snapshot.Sessions.Current)), formatInt(int64(response.Snapshot.Tokens.Current)), formatCostValue(response.Snapshot.CostUSD.Current)),
		dim(fmt.Sprintf("Active %s  ·  Tools %s  ·  Cost %s", formatDuration(int64(response.Snapshot.ActiveTime.Current)), formatInt(int64(response.Snapshot.ToolCalls.Current)), attentionChange(response.Snapshot.CostUSD))),
		dim(fmt.Sprintf("%s → %s  ·  baseline %s → %s", empty(shortAttentionTime(response.Window.Current.From), "—"), empty(shortAttentionTime(response.Window.Current.To), "—"), empty(shortAttentionTime(response.Window.Baseline.From), "—"), empty(shortAttentionTime(response.Window.Baseline.To), "—"))),
		"",
		fmt.Sprintf("%s  %s critical  %s warning", bold("Needs attention"), formatInt(int64(response.Counts.Critical)), formatInt(int64(response.Counts.Warning))),
	}
	// Short terminals prioritize actionable rows over the snapshot.
	if s.contentHeight() < 8 {
		lines = lines[4:]
	}
	if len(response.Items) == 0 {
		if s.loading {
			return append(lines, dim("Loading attention…"))
		}
		return append(lines, success("No issues need attention in the current range."))
	}
	visible := max(1, (s.contentHeight()-len(lines))/2)
	start, end := attentionVisibleWindow(len(response.Items), s.selected, visible)
	for index := start; index < end; index++ {
		item := response.Items[index]
		cursor := "  "
		if index == s.selected {
			cursor = accent("› ")
		}
		severity := strings.ToUpper(item.Severity)
		if item.Severity == "critical" {
			severity = danger(severity)
		} else {
			severity = warning(severity)
		}
		lines = append(lines, cursor+severity+"  "+item.Subject, "  "+dim(item.Reason))
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
