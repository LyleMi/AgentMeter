package tui

import (
	"fmt"
	"strconv"
	"strings"

	"AgentMeter/internal/audit"
	agentmodel "AgentMeter/internal/model"
)

func (s *state) toolLines() []string {
	switch s.toolsTab {
	case toolsTabSummary:
		return s.toolSummaryLines()
	case toolsTabShell, toolsTabCalls:
		return s.toolExplorerLines()
	default:
		return s.toolOverviewLines()
	}
}

func (s *state) toolOverviewLines() []string {
	lines := []string{
		bold("Tools"),
		toolTabLine(s.toolsTab, s.width),
		toolFilterLine(s),
	}
	if len(s.tools) == 0 {
		if s.loading {
			return lines
		}
		return append(lines, "No tool calls found.")
	}

	totalCalls := 0
	failedCalls := 0
	totalDuration := int64(0)
	for _, item := range s.tools {
		totalCalls += item.Calls
		failedCalls += item.FailedCalls
		totalDuration += item.TotalDurationMS
	}
	avgDuration := float64(0)
	if totalCalls > 0 {
		avgDuration = float64(totalDuration) / float64(totalCalls)
	}
	lines = append(lines,
		bold("Top Tools"),
		fmt.Sprintf("  %-30s %8s %8s %8s %12s %12s", "Tool", "Calls", "Success", "Failed", "Total", "Average"),
	)
	lines = s.appendToolRows(lines, visibleToolRows(s.tools, s.scroll, s.contentHeight()-len(lines)), -1)
	return append(lines, "",
		bold("Activity Summary"),
		fmt.Sprintf("Total calls: %-10s Tools used: %-8s Failed/pending: %-8s Average: %s",
			formatInt(int64(totalCalls)),
			formatInt(int64(len(s.tools))),
			formatInt(int64(failedCalls)),
			formatDurationFloat(avgDuration),
		),
	)
}

func (s *state) toolSummaryLines() []string {
	lines := []string{
		bold("Tool Summary"),
		toolTabLine(s.toolsTab, s.width),
		toolFilterLine(s),
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
	return s.appendToolRows(lines, s.tools[s.scroll:end], -1)
}

func visibleToolRows(rows []agentmodel.ToolStat, scroll, visible int) []agentmodel.ToolStat {
	if visible < 1 {
		visible = 1
	}
	if scroll < 0 {
		scroll = 0
	}
	if scroll > len(rows) {
		scroll = len(rows)
	}
	end := scroll + visible
	if end > len(rows) {
		end = len(rows)
	}
	return rows[scroll:end]
}

func (s *state) appendToolRows(lines []string, rows []agentmodel.ToolStat, limit int) []string {
	if limit >= 0 {
		rows = limitSlice(rows, limit)
	}
	for offset, item := range rows {
		i := s.scroll + offset
		if limit >= 0 {
			i = offset
		}
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

func (s *state) toolExplorerLines() []string {
	title := "Recent Tool Calls"
	if s.toolsTab == toolsTabShell {
		title = "Shell Commands"
	}
	lines := []string{
		bold(title),
		toolTabLine(s.toolsTab, s.width),
		toolFilterLine(s),
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
		if s.toolsTab == toolsTabShell {
			command := invokedToolCommand(s.toolCalls[i])
			if command != "" {
				lines = append(lines, fit("    command: "+command, s.width))
			}
		}
	}
	return lines
}

func toolTabLine(active toolsTab, width int) string {
	labels := make([]string, 0, len(toolsTabs))
	for _, tab := range toolsTabs {
		label := tab.title()
		if tab == active {
			label = inverse(" " + label + " ")
		}
		labels = append(labels, label)
	}
	return fit("Tabs: "+strings.Join(labels, "  "), width)
}

func toolFilterLine(s *state) string {
	parts := []string{}
	if strings.TrimSpace(s.toolAgent) != "" {
		parts = append(parts, "source "+filterLabel(s.toolAgent, usageAgentOptions(s.scopeOverview)))
	}
	if s.toolsTab == toolsTabShell || s.toolsTab == toolsTabCalls {
		parts = append([]string{"Sort: " + toolCallSortLabel(s.toolCallSort)}, parts...)
		if s.toolRange != usageRangeAll {
			parts = append(parts, "range "+s.toolRange.title())
		}
	}
	if strings.TrimSpace(s.toolCommand) != "" && (s.toolsTab == toolsTabShell || s.toolsTab == toolsTabCalls) {
		parts = append(parts, "command "+s.toolCommand)
	}
	if len(parts) == 0 {
		parts = append(parts, "Source: All")
	}
	return fit(strings.Join(parts, "  "), s.width)
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

func isShellToolName(toolName string) bool {
	normalized := strings.ToLower(strings.TrimSpace(toolName))
	if normalized == "" {
		return false
	}
	switch normalized {
	case "bash", "cmd", "cmd.exe", "powershell", "powershell.exe", "pwsh", "pwsh.exe", "sh", "shell", "shell_command", "terminal", "zsh":
		return true
	}
	if strings.Contains(normalized, "shell_command") {
		return true
	}
	for _, token := range strings.FieldsFunc(normalized, func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9')
	}) {
		switch token {
		case "bash", "cmd", "powershell", "pwsh", "sh", "shell", "terminal", "zsh":
			return true
		}
	}
	return false
}

func invokedToolCommand(call agentmodel.ToolCall) string {
	return commandNameFromText(toolCommandSummary(call), 0)
}

func toolCommandSummary(call agentmodel.ToolCall) string {
	if info, ok := audit.ExtractShellCommand(call); ok && strings.TrimSpace(info.Command) != "" {
		return strings.TrimSpace(info.Command)
	}
	for _, value := range []string{
		audit.ExtractCommandText(call.RawStartEventJSON),
		audit.ExtractCommandText(call.InputSummary),
		audit.ExtractCommandText(call.RawStartEventSummary),
	} {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return toolCallPrimarySummary(call)
}

func commandNameFromText(command string, depth int) string {
	if depth > 4 {
		return ""
	}
	tokens := tokenizeCommand(command)
	for _, segment := range commandSegments(tokens) {
		if name := commandNameFromSegment(segment, depth); name != "" {
			return name
		}
	}
	return ""
}

func commandNameFromSegment(segment []string, depth int) string {
	tokens := append([]string(nil), segment...)
	for len(tokens) > 0 {
		token := cleanCommandToken(tokens[0])
		tokens = tokens[1:]
		if token == "" || isEnvironmentAssignment(token) || isRedirectionToken(token) {
			continue
		}
		name := normalizeExecutableName(token)
		if name == "" {
			continue
		}
		switch name {
		case "bash", "sh", "zsh":
			if nested := nestedShellCommand(tokens, "-c"); nested != "" {
				if inner := commandNameFromText(nested, depth+1); inner != "" {
					return inner
				}
			}
			return name
		case "cmd":
			if nested := nestedShellCommand(tokens, "/c"); nested != "" {
				if inner := commandNameFromText(nested, depth+1); inner != "" {
					return inner
				}
			}
			return name
		case "powershell", "pwsh":
			if nested := nestedShellCommand(tokens, "-command"); nested != "" {
				if inner := commandNameFromText(nested, depth+1); inner != "" {
					return inner
				}
			}
			return name
		case "env":
			stripWrapperPrefix(name, &tokens)
			continue
		case "sudo", "doas", "time", "command", "builtin", "exec", "nice", "nohup":
			stripWrapperPrefix(name, &tokens)
			continue
		case "cd", "echo", "set", "export", "true", "false":
			return ""
		default:
			return name
		}
	}
	return ""
}

func tokenizeCommand(command string) []string {
	var tokens []string
	var current strings.Builder
	var quote rune
	escaping := false
	push := func() {
		if current.Len() > 0 {
			tokens = append(tokens, current.String())
			current.Reset()
		}
	}
	runes := []rune(command)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if escaping {
			current.WriteRune(r)
			escaping = false
			continue
		}
		if quote != 0 {
			if r == '\\' {
				escaping = true
				continue
			}
			if r == quote {
				quote = 0
				continue
			}
			current.WriteRune(r)
			continue
		}
		switch r {
		case '\'', '"':
			quote = r
		case ' ', '\t', '\n', '\r':
			push()
		case '&':
			if i+1 < len(runes) && runes[i+1] == '&' {
				push()
				tokens = append(tokens, "&&")
				i++
			} else {
				current.WriteRune(r)
			}
		case '|':
			push()
			if i+1 < len(runes) && runes[i+1] == '|' {
				tokens = append(tokens, "||")
				i++
			} else {
				tokens = append(tokens, string(r))
			}
		case ';':
			push()
			tokens = append(tokens, string(r))
		default:
			current.WriteRune(r)
		}
	}
	push()
	return tokens
}

func commandSegments(tokens []string) [][]string {
	var segments [][]string
	var current []string
	for _, token := range tokens {
		if token == ";" || token == "|" || token == "&&" || token == "||" {
			if len(current) > 0 {
				segments = append(segments, current)
				current = nil
			}
			continue
		}
		current = append(current, token)
	}
	if len(current) > 0 {
		segments = append(segments, current)
	}
	return segments
}

func nestedShellCommand(tokens []string, flag string) string {
	flag = strings.ToLower(flag)
	for i, token := range tokens {
		token = strings.ToLower(cleanCommandToken(token))
		if token == flag || (flag == "-c" && strings.HasPrefix(token, "-") && strings.Contains(token, "c")) || (flag == "-command" && (token == "-c" || token == "/c")) || (flag == "/c" && token == "/k") {
			return strings.Join(tokens[i+1:], " ")
		}
	}
	return ""
}

func stripWrapperPrefix(wrapper string, tokens *[]string) {
	values := *tokens
	switch wrapper {
	case "env":
		for len(values) > 0 {
			token := cleanCommandToken(values[0])
			if strings.HasPrefix(token, "-") || isEnvironmentAssignment(token) {
				values = values[1:]
				continue
			}
			break
		}
	case "sudo", "doas":
		for len(values) > 0 {
			option := cleanCommandToken(values[0])
			if !strings.HasPrefix(option, "-") {
				break
			}
			values = values[1:]
			if option == "-g" || option == "-h" || option == "-p" || option == "-u" || option == "-C" {
				if len(values) > 0 {
					values = values[1:]
				}
			}
		}
	case "nice":
		for len(values) > 0 {
			option := cleanCommandToken(values[0])
			if !strings.HasPrefix(option, "-") {
				break
			}
			values = values[1:]
			if option == "-n" && len(values) > 0 {
				values = values[1:]
			}
		}
	default:
		for len(values) > 0 && strings.HasPrefix(cleanCommandToken(values[0]), "-") {
			values = values[1:]
		}
	}
	*tokens = values
}

func cleanCommandToken(token string) string {
	return strings.Trim(strings.TrimSpace(token), " \t\r\n()[]{}&,")
}

func normalizeExecutableName(token string) string {
	token = strings.TrimLeft(cleanCommandToken(token), ".\\/")
	if token == "" {
		return ""
	}
	parts := strings.FieldsFunc(token, func(r rune) bool { return r == '\\' || r == '/' })
	if len(parts) > 0 {
		token = parts[len(parts)-1]
	}
	token = strings.ToLower(token)
	for _, suffix := range []string{".exe", ".cmd", ".bat", ".ps1", ".sh"} {
		token = strings.TrimSuffix(token, suffix)
	}
	if token == "py" || strings.HasPrefix(token, "python") {
		return "python"
	}
	if strings.HasPrefix(token, "pip") {
		return "pip"
	}
	if token == "nodejs" {
		return "node"
	}
	return token
}

func isEnvironmentAssignment(token string) bool {
	if strings.HasPrefix(strings.ToLower(token), "$env:") {
		return strings.Contains(token, "=")
	}
	if index := strings.Index(token, "="); index > 0 {
		first := token[0]
		return (first >= 'A' && first <= 'Z') || (first >= 'a' && first <= 'z') || first == '_'
	}
	return false
}

func isRedirectionToken(token string) bool {
	token = strings.TrimSpace(token)
	return strings.HasPrefix(token, ">") || strings.HasPrefix(token, "<") || strings.Contains(token, ">") && len(token) <= 3
}

func toolCommandOptions(calls []agentmodel.ToolCall) []stringOption {
	seen := map[string]string{}
	for _, call := range calls {
		if command := invokedToolCommand(call); command != "" {
			seen[command] = command
		}
	}
	return sortedStringOptions(seen)
}
