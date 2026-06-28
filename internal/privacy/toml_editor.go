package privacy

import (
	"strconv"
	"strings"
)

type tomlDocument struct {
	lines        []string
	newline      string
	finalNewline bool
	values       map[string]configValue
	positions    map[string]int
	sections     map[string]sectionRange
}

type sectionRange struct {
	Start int
	End   int
}

func parseTOML(content []byte) tomlDocument {
	text := string(content)
	newline := "\n"
	if strings.Contains(text, "\r\n") {
		newline = "\r\n"
	}
	normalized := strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
	finalNewline := strings.HasSuffix(normalized, "\n")
	lines := []string{}
	if normalized != "" {
		lines = strings.Split(normalized, "\n")
		if finalNewline {
			lines = lines[:len(lines)-1]
		}
	}

	doc := tomlDocument{
		lines:        lines,
		newline:      newline,
		finalNewline: finalNewline,
		values:       map[string]configValue{},
		positions:    map[string]int{},
		sections:     map[string]sectionRange{},
	}
	doc.index()
	return doc
}

func (d *tomlDocument) index() {
	d.values = map[string]configValue{}
	d.positions = map[string]int{}
	d.sections = map[string]sectionRange{}

	currentSection := ""
	currentStart := -1
	for index, line := range d.lines {
		if section, ok := parseSectionName(line); ok {
			if currentStart >= 0 {
				d.sections[currentSection] = sectionRange{Start: currentStart, End: index}
			}
			currentSection = section
			currentStart = index
			continue
		}
		key, raw, ok := parseAssignment(line)
		if !ok {
			continue
		}
		fullKey := fullTOMLKey(currentSection, key)
		d.values[fullKey] = parseConfigValue(raw)
		d.positions[fullKey] = index
	}
	if currentStart >= 0 {
		d.sections[currentSection] = sectionRange{Start: currentStart, End: len(d.lines)}
	}
}

func (d tomlDocument) Value(fullKey string) (configValue, bool) {
	value, ok := d.values[fullKey]
	return value, ok
}

func (d *tomlDocument) Set(table, key string, value configValue) {
	fullKey := key
	if table != "" {
		fullKey = table + "." + key
	}
	if position, ok := d.positions[fullKey]; ok {
		d.lines[position] = replaceAssignmentValue(d.lines[position], value.TOML())
		d.index()
		return
	}

	line := key + " = " + value.TOML()
	if table == "" {
		insertAt := d.firstSectionIndex()
		d.insertLine(insertAt, line)
		d.index()
		return
	}

	if section, ok := d.sections[table]; ok {
		insertAt := section.End
		for insertAt > section.Start+1 && strings.TrimSpace(d.lines[insertAt-1]) == "" {
			insertAt--
		}
		d.insertLine(insertAt, line)
		d.index()
		return
	}

	if len(d.lines) > 0 && strings.TrimSpace(d.lines[len(d.lines)-1]) != "" {
		d.lines = append(d.lines, "")
	}
	d.lines = append(d.lines, "["+table+"]", line)
	d.finalNewline = true
	d.index()
}

func (d *tomlDocument) Unset(table, key string) {
	fullKey := key
	if table != "" {
		fullKey = table + "." + key
	}
	position, ok := d.positions[fullKey]
	if !ok {
		return
	}
	d.lines = append(d.lines[:position], d.lines[position+1:]...)
	d.index()
}

func (d *tomlDocument) insertLine(index int, line string) {
	if index < 0 || index > len(d.lines) {
		index = len(d.lines)
	}
	d.lines = append(d.lines, "")
	copy(d.lines[index+1:], d.lines[index:])
	d.lines[index] = line
	d.finalNewline = true
}

func (d tomlDocument) firstSectionIndex() int {
	for index, line := range d.lines {
		if _, ok := parseSectionName(line); ok {
			return index
		}
	}
	return len(d.lines)
}

func (d tomlDocument) Bytes() []byte {
	if len(d.lines) == 0 {
		return nil
	}
	text := strings.Join(d.lines, d.newline)
	if d.finalNewline {
		text += d.newline
	}
	return []byte(text)
}

func parseSectionName(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "[[") {
		return "", false
	}
	end := strings.Index(trimmed, "]")
	if end <= 1 {
		return "", false
	}
	name := strings.TrimSpace(trimmed[1:end])
	if name == "" || strings.Contains(name, "[") {
		return "", false
	}
	return name, true
}

func parseAssignment(line string) (string, string, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return "", "", false
	}
	eq := indexUnquoted(line, '=')
	if eq < 0 {
		return "", "", false
	}
	key := strings.TrimSpace(line[:eq])
	if key == "" {
		return "", "", false
	}
	return unquoteBareKey(key), line[eq+1:], true
}

func fullTOMLKey(section, key string) string {
	if section == "" {
		return key
	}
	return section + "." + key
}

func unquoteBareKey(key string) string {
	key = strings.TrimSpace(key)
	if len(key) >= 2 {
		if key[0] == '"' && key[len(key)-1] == '"' {
			if unquoted, err := strconv.Unquote(key); err == nil {
				return unquoted
			}
		}
		if key[0] == '\'' && key[len(key)-1] == '\'' {
			return key[1 : len(key)-1]
		}
	}
	return key
}

func parseConfigValue(raw string) configValue {
	value, _ := splitValueComment(raw)
	trimmed := strings.TrimSpace(value)
	switch strings.ToLower(trimmed) {
	case "true":
		return boolValue(true)
	case "false":
		return boolValue(false)
	}
	if len(trimmed) >= 2 {
		if trimmed[0] == '"' {
			if unquoted, err := strconv.Unquote(trimmed); err == nil {
				return stringValue(unquoted)
			}
		}
		if trimmed[0] == '\'' && trimmed[len(trimmed)-1] == '\'' {
			return stringValue(trimmed[1 : len(trimmed)-1])
		}
	}
	return configValue{Kind: "raw", Raw: trimmed}
}

func replaceAssignmentValue(line, value string) string {
	eq := indexUnquoted(line, '=')
	if eq < 0 {
		return line
	}
	_, comment := splitValueComment(line[eq+1:])
	left := strings.TrimRight(line[:eq], " \t")
	return left + " = " + value + comment
}

func splitValueComment(raw string) (string, string) {
	inSingle := false
	inDouble := false
	escaped := false
	for index, r := range raw {
		switch {
		case escaped:
			escaped = false
		case inDouble && r == '\\':
			escaped = true
		case !inDouble && r == '\'':
			inSingle = !inSingle
		case !inSingle && r == '"':
			inDouble = !inDouble
		case !inSingle && !inDouble && r == '#':
			commentStart := index
			for commentStart > 0 {
				previous := raw[commentStart-1]
				if previous != ' ' && previous != '\t' {
					break
				}
				commentStart--
			}
			return raw[:commentStart], raw[commentStart:]
		}
	}
	return raw, ""
}

func indexUnquoted(line string, target rune) int {
	inSingle := false
	inDouble := false
	escaped := false
	for index, r := range line {
		switch {
		case escaped:
			escaped = false
		case inDouble && r == '\\':
			escaped = true
		case !inDouble && r == '\'':
			inSingle = !inSingle
		case !inSingle && r == '"':
			inDouble = !inDouble
		case !inSingle && !inDouble && r == target:
			return index
		}
	}
	return -1
}
