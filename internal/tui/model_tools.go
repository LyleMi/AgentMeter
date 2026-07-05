package tui

import (
	"sort"
	"strings"
	"time"

	agentmodel "github.com/LyleMi/AgentMeter/internal/model"
)

func (s *state) toolCallFilters() agentmodel.ToolCallFilters {
	filters := s.toolExplorerFilters()
	filters.ToolName = s.toolCallTool
	filters.Limit = 200
	return filters
}

func (s *state) toolExplorerFilters() agentmodel.ToolCallFilters {
	filters := agentmodel.ToolCallFilters{
		Agent: strings.TrimSpace(s.toolAgent),
		Sort:  s.toolCallSort,
		Limit: 500,
	}
	switch s.toolRange {
	case usageRangeDay:
		filters.StartedFrom = time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC3339)
	case usageRangeWeek:
		filters.StartedFrom = time.Now().AddDate(0, 0, -7).UTC().Format(time.RFC3339)
	case usageRangeMonth:
		filters.StartedFrom = time.Now().AddDate(0, 0, -30).UTC().Format(time.RFC3339)
	}
	return agentmodel.ToolCallFilters{
		Agent:       filters.Agent,
		StartedFrom: filters.StartedFrom,
		Sort:        filters.Sort,
		Limit:       filters.Limit,
	}
}

func listToolCallsForToolsContext(service appService, filters agentmodel.ToolCallFilters, tools []agentmodel.ToolStat, tab toolsTab, command string) ([]agentmodel.ToolCall, error) {
	if tab != toolsTabShell {
		calls, err := service.ListToolCalls(filters)
		if err != nil {
			return nil, err
		}
		return filterToolCallsForToolsTab(calls, tab, command), nil
	}
	limit := filters.Limit
	if limit <= 0 {
		limit = 500
	}
	filters.Shell = true
	filters.IncludeRisk = true
	filters.Limit = limit
	calls, err := service.ListToolCalls(filters)
	if err != nil {
		return nil, err
	}
	calls = filterToolCallsForToolsTab(calls, tab, command)
	return limitSlice(calls, limit), nil
}

func shellToolNames(tools []agentmodel.ToolStat) []string {
	seen := map[string]bool{}
	names := []string{}
	for _, tool := range tools {
		name := strings.TrimSpace(tool.ToolName)
		if name == "" || seen[name] || !isShellToolName(name) {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func uniqueToolCalls(calls []agentmodel.ToolCall) []agentmodel.ToolCall {
	seen := map[int64]bool{}
	result := make([]agentmodel.ToolCall, 0, len(calls))
	for _, call := range calls {
		if call.ID > 0 {
			if seen[call.ID] {
				continue
			}
			seen[call.ID] = true
		}
		result = append(result, call)
	}
	return result
}

func sortToolCalls(calls []agentmodel.ToolCall, sortMode string) {
	sort.SliceStable(calls, func(i, j int) bool {
		left, right := calls[i], calls[j]
		switch strings.TrimSpace(sortMode) {
		case "duration_desc":
			if left.DurationMS != right.DurationMS {
				return left.DurationMS > right.DurationMS
			}
		case "duration_asc":
			if left.DurationMS != right.DurationMS {
				return left.DurationMS < right.DurationMS
			}
		}
		if !left.StartedAt.Equal(right.StartedAt) {
			return left.StartedAt.After(right.StartedAt)
		}
		return left.ID > right.ID
	})
}

func filterToolCallsForToolsTab(calls []agentmodel.ToolCall, tab toolsTab, command string) []agentmodel.ToolCall {
	if tab != toolsTabShell && strings.TrimSpace(command) == "" {
		return calls
	}
	result := make([]agentmodel.ToolCall, 0, len(calls))
	for _, call := range calls {
		if tab == toolsTabShell && !isShellToolName(call.ToolName) {
			continue
		}
		if strings.TrimSpace(command) != "" && invokedToolCommand(call) != command {
			continue
		}
		result = append(result, call)
	}
	return result
}

func (s *state) openToolCalls(toolName string) command {
	s.previous = pageTools
	s.page = pageToolCalls
	s.selected = 0
	s.scroll = 0
	s.toolCall = nil
	s.toolCallTool = toolName
	return s.load(pageToolCalls)
}

func (s *state) toolCallBackPage() page {
	if s.previous == pageTools {
		return pageTools
	}
	return pageToolCalls
}

func (s *state) cycleToolCallSort() command {
	switch s.toolCallSort {
	case "":
		s.toolCallSort = "duration_desc"
	case "duration_desc":
		s.toolCallSort = "duration_asc"
	case "duration_asc":
		s.toolCallSort = "risk_desc"
	case "risk_desc":
		s.toolCallSort = "risk_asc"
	default:
		s.toolCallSort = ""
	}
	s.selected = 0
	s.scroll = 0
	s.status = "tool calls sorted by " + toolCallSortLabel(s.toolCallSort)
	if s.page == pageTools {
		return s.load(pageTools)
	}
	return s.load(pageToolCalls)
}

func (s *state) cycleToolAgent() command {
	options := usageAgentOptions(s.scopeOverview)
	next, label := cycleStringOption(s.toolAgent, options)
	s.toolAgent = next
	s.selected = 0
	s.scroll = 0
	s.status = "tool source filter: " + label
	return s.load(toolListRootPage(s.page))
}

func (s *state) cycleToolRange() command {
	s.toolRange = cycleTab(s.toolRange, usageRanges, 1)
	s.selected = 0
	s.scroll = 0
	s.status = "tool range filter: " + s.toolRange.title()
	return s.load(toolListRootPage(s.page))
}

func (s *state) cycleToolCommand() command {
	options := toolCommandOptions(s.toolCalls)
	next, label := cycleStringOption(s.toolCommand, options)
	s.toolCommand = next
	s.selected = 0
	s.scroll = 0
	s.status = "shell command filter: " + label
	return s.load(pageTools)
}

func (s *state) clearToolFilters() command {
	if s.toolAgent == "" && s.toolRange == usageRangeAll && s.toolCommand == "" {
		s.status = "tool filters already clear"
		return nil
	}
	s.toolAgent = ""
	s.toolRange = usageRangeAll
	s.toolCommand = ""
	s.selected = 0
	s.scroll = 0
	s.status = "tool filters cleared"
	return s.load(toolListRootPage(s.page))
}

func toolListRootPage(current page) page {
	if current == pageToolCalls || current == pageToolCallDetail {
		return pageToolCalls
	}
	return current
}
