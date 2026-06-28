package tui

import "fmt"

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
