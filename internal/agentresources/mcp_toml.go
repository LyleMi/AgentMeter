package agentresources

import (
	"strconv"
	"strings"
)

func setMCPEnabledInTOML(content []byte, name string, enabled bool) ([]byte, error) {
	lines, separator := splitTOMLLines(content)
	start, end, ok := findTOMLTable(lines, "[mcp_servers."+name+"]")
	if !ok {
		return nil, BadRequest("MCP server table uses an unsupported TOML table style")
	}
	valueLine := "enabled = " + strconv.FormatBool(enabled)
	if replaceTOMLValue(lines, start+1, end, "enabled", valueLine) {
		return []byte(strings.Join(lines, separator)), nil
	}
	lines = insertTOMLLine(lines, start+1, valueLine)
	return []byte(strings.Join(lines, separator)), nil
}

func splitTOMLLines(content []byte) ([]string, string) {
	text := string(content)
	if strings.Contains(text, "\r\n") {
		return strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n"), "\r\n"
	}
	return strings.Split(text, "\n"), "\n"
}

func findTOMLTable(lines []string, header string) (int, int, bool) {
	start := -1
	for index, line := range lines {
		if strings.TrimSpace(line) == header {
			start = index
			break
		}
	}
	if start < 0 {
		return 0, 0, false
	}
	for index := start + 1; index < len(lines); index++ {
		if isTOMLTableHeader(lines[index]) {
			return start, index, true
		}
	}
	return start, len(lines), true
}

func isTOMLTableHeader(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")
}

func replaceTOMLValue(lines []string, start, end int, key, valueLine string) bool {
	for index := start; index < end; index++ {
		lineKey, _, ok := strings.Cut(strings.TrimSpace(lines[index]), "=")
		if !ok || strings.TrimSpace(lineKey) != key {
			continue
		}
		indentLength := len(lines[index]) - len(strings.TrimLeft(lines[index], " \t"))
		lines[index] = lines[index][:indentLength] + valueLine
		return true
	}
	return false
}

func insertTOMLLine(lines []string, index int, line string) []string {
	updated := make([]string, 0, len(lines)+1)
	updated = append(updated, lines[:index]...)
	updated = append(updated, line)
	return append(updated, lines[index:]...)
}
