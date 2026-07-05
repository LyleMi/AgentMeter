package audit

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/LyleMi/AgentMeter/internal/model"
)

type ShellFamily string

const (
	ShellPowerShell ShellFamily = "powershell"
	ShellCmd        ShellFamily = "cmd"
	ShellPosix      ShellFamily = "posix"
	ShellUnknown    ShellFamily = "unknown"
)

type CommandInfo struct {
	ToolName string      `json:"toolName"`
	Command  string      `json:"command"`
	Family   ShellFamily `json:"family"`
	Source   string      `json:"source"`
}

type CommandRisk struct {
	RuleID   string `json:"ruleId"`
	Category string `json:"category"`
	Severity string `json:"severity"`
	Title    string `json:"title"`
}

func IsShellTool(toolName string) bool {
	switch normalizedShellTool(toolName) {
	case "shell_command", "bash", "zsh", "sh", "powershell", "powershell.exe", "pwsh", "pwsh.exe", "cmd", "cmd.exe":
		return true
	default:
		return false
	}
}

func ExtractShellCommand(call model.ToolCall) (CommandInfo, bool) {
	if !IsShellTool(call.ToolName) {
		return CommandInfo{ToolName: call.ToolName, Family: ShellUnknown}, false
	}

	candidates := []struct {
		source           string
		text             string
		allowRawFallback bool
	}{
		{source: "raw_start_event_json", text: call.RawStartEventJSON, allowRawFallback: false},
		{source: "input_summary", text: call.InputSummary, allowRawFallback: true},
		{source: "raw_start_event_summary", text: call.RawStartEventSummary, allowRawFallback: true},
	}

	for _, candidate := range candidates {
		command := extractCommandText(candidate.text, candidate.allowRawFallback)
		if command == "" {
			continue
		}
		return CommandInfo{
			ToolName: call.ToolName,
			Command:  command,
			Family:   ClassifyShellFamily(call.ToolName, command),
			Source:   candidate.source,
		}, true
	}

	return CommandInfo{ToolName: call.ToolName, Family: ClassifyShellFamily(call.ToolName, "")}, false
}

func ExtractCommandText(text string) string {
	return extractCommandText(text, true)
}

func ClassifyShellFamily(toolName string, command string) ShellFamily {
	if family := shellFamilyFromTool(toolName); family != ShellUnknown {
		return family
	}
	if family := shellFamilyFromCommand(command); family != ShellUnknown {
		return family
	}
	return ShellUnknown
}

func ClassifyCommandRisks(command CommandInfo) []CommandRisk {
	if strings.TrimSpace(command.Command) == "" {
		return nil
	}
	risks := make([]CommandRisk, 0, len(commandRiskRules))
	for _, rule := range commandRiskRules {
		if rule.pattern.MatchString(command.Command) {
			risks = append(risks, CommandRisk{
				RuleID:   rule.ruleID,
				Category: rule.category,
				Severity: rule.severity,
				Title:    rule.title,
			})
		}
	}
	return risks
}

func normalizedShellTool(toolName string) string {
	name := strings.ToLower(strings.TrimSpace(toolName))
	if strings.HasSuffix(name, ".shell_command") {
		return "shell_command"
	}
	switch name {
	case "cmd.exe", "powershell.exe", "pwsh.exe":
		return name
	}
	if dot := strings.LastIndex(name, "."); dot >= 0 {
		name = name[dot+1:]
	}
	return name
}

func shellFamilyFromTool(toolName string) ShellFamily {
	switch normalizedShellTool(toolName) {
	case "bash", "zsh", "sh":
		return ShellPosix
	case "powershell", "powershell.exe", "pwsh", "pwsh.exe":
		return ShellPowerShell
	case "cmd", "cmd.exe":
		return ShellCmd
	default:
		return ShellUnknown
	}
}

func shellFamilyFromCommand(command string) ShellFamily {
	trimmed := strings.TrimSpace(command)
	lower := strings.ToLower(trimmed)
	switch {
	case strings.HasPrefix(lower, "powershell ") || strings.HasPrefix(lower, "powershell.exe ") ||
		strings.HasPrefix(lower, "pwsh ") || strings.HasPrefix(lower, "pwsh.exe "):
		return ShellPowerShell
	case strings.HasPrefix(lower, "cmd /") || strings.HasPrefix(lower, "cmd.exe /"):
		return ShellCmd
	}

	powerShellClues := []string{
		"set-executionpolicy", "invoke-webrequest", "invoke-restmethod", "invoke-expression",
		"remove-item", "get-childitem", "set-itemproperty", "new-itemproperty", "remove-itemproperty",
		"start-process", "$env:", " env:", "hklm:", "hkcu:", "-executionpolicy",
	}
	for _, clue := range powerShellClues {
		if strings.Contains(lower, clue) {
			return ShellPowerShell
		}
	}

	cmdClues := []string{
		"del /", "rmdir /", "rd /", "copy ", "xcopy ", "robocopy ", "%userprofile%", "%appdata%",
		"reg add", "reg delete", "sc.exe ", "sc create", "sc delete", "net start", "net stop",
	}
	for _, clue := range cmdClues {
		if strings.Contains(lower, clue) {
			return ShellCmd
		}
	}

	posixClues := []string{
		"sudo ", "doas ", "su -", "rm -", "chmod ", "chown ", "curl ", "wget ", "apt-get ",
		"apt ", "brew ", "export ", "$home", "~/", "#!/bin/sh", "#!/usr/bin/env bash",
	}
	for _, clue := range posixClues {
		if strings.Contains(lower, clue) {
			return ShellPosix
		}
	}

	return ShellUnknown
}

func extractCommandText(text string, allowRawFallback bool) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}

	current := trimmed
	for depth := 0; depth < 3; depth++ {
		var value any
		if err := json.Unmarshal([]byte(current), &value); err == nil {
			if command := commandFromValue(value, 0); command != "" {
				return command
			}
			if unwrapped, ok := value.(string); ok {
				current = strings.TrimSpace(unwrapped)
				if current == "" {
					return ""
				}
				continue
			}
			return ""
		}
		if unquoted, err := strconv.Unquote(current); err == nil && unquoted != current {
			current = strings.TrimSpace(unquoted)
			continue
		}
		break
	}

	if command := commandFromLooseFields(current); command != "" {
		return command
	}
	if allowRawFallback && !looksStructured(current) {
		return current
	}
	return ""
}

var commandFieldNames = []string{"command", "cmd", "script", "arguments", "input"}

func commandFromValue(value any, depth int) string {
	if depth > 8 || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case map[string]any:
		for _, fieldName := range commandFieldNames {
			if fieldValue, ok := lookupField(typed, fieldName); ok {
				if command := commandFromFieldValue(fieldValue, depth+1); command != "" {
					return command
				}
			}
		}
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if command := commandFromValue(typed[key], depth+1); command != "" {
				return command
			}
		}
	case []any:
		for _, item := range typed {
			if command := commandFromValue(item, depth+1); command != "" {
				return command
			}
		}
	}
	return ""
}

func commandFromFieldValue(value any, depth int) string {
	if depth > 8 || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return commandFromFieldString(typed, depth+1)
	case map[string]any:
		return commandFromValue(typed, depth+1)
	case []any:
		if command := joinScalarArray(typed); command != "" {
			return command
		}
		return commandFromValue(typed, depth+1)
	case []string:
		return strings.TrimSpace(strings.Join(typed, " "))
	case json.Number:
		return typed.String()
	case float64, bool:
		return strings.TrimSpace(fmt.Sprint(typed))
	default:
		return ""
	}
}

func joinScalarArray(values []any) string {
	if len(values) == 0 {
		return ""
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		switch typed := value.(type) {
		case string:
			parts = append(parts, typed)
		case json.Number:
			parts = append(parts, typed.String())
		case float64, bool:
			parts = append(parts, fmt.Sprint(typed))
		default:
			return ""
		}
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func commandFromFieldString(value string, depth int) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if depth <= 8 && looksStructured(trimmed) {
		var nested any
		if err := json.Unmarshal([]byte(trimmed), &nested); err == nil {
			if command := commandFromValue(nested, depth+1); command != "" {
				return command
			}
		}
	}
	if unquoted, err := strconv.Unquote(trimmed); err == nil && unquoted != trimmed {
		if command := commandFromFieldString(unquoted, depth+1); command != "" {
			return command
		}
	}
	return trimmed
}

func lookupField(values map[string]any, fieldName string) (any, bool) {
	if value, ok := values[fieldName]; ok {
		return value, true
	}
	for key, value := range values {
		if strings.EqualFold(key, fieldName) {
			return value, true
		}
	}
	return nil, false
}

var looseCommandFieldPattern = regexp.MustCompile(`(?is)\b(command|cmd|script|arguments|input)\b\s*[:=]\s*(?:"([^"]*)"|'([^']*)'|([^\r\n,}]+))`)

func commandFromLooseFields(text string) string {
	match := looseCommandFieldPattern.FindStringSubmatch(text)
	if len(match) == 0 {
		return ""
	}
	for i := 2; i < len(match); i++ {
		if strings.TrimSpace(match[i]) != "" {
			return commandFromFieldString(match[i], 0)
		}
	}
	return ""
}

func looksStructured(text string) bool {
	trimmed := strings.TrimSpace(text)
	return strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[")
}

type commandRiskRule struct {
	ruleID   string
	category string
	severity string
	title    string
	pattern  *regexp.Regexp
}

var commandRiskRules = []commandRiskRule{
	{
		ruleID:   "shell.destructive-delete",
		category: CategoryFile,
		severity: SeverityHigh,
		title:    "Destructive delete command",
		pattern:  regexp.MustCompile(`(?i)(^|[;&|]\s*)(((sudo|doas)\s+)?rm\s+-[^\r\n;&|]*[rf][^\r\n;&|]*\b|find\b[^\r\n;&|]*\s-delete\b|del(?:ete)?\s+(?:/[a-z]+\s+)*[^\r\n;&|]+|rmdir\s+/s\b|rd\s+/s\b|remove-item\b[^\r\n;&|]*-(recurse|force)\b)`),
	},
	{
		ruleID:   "shell.privilege",
		category: CategoryCommand,
		severity: SeverityMedium,
		title:    "Privilege or admin elevation",
		pattern:  regexp.MustCompile(`(?i)(^|[;&|]\s*)(sudo|doas|su)\b|\brunas\b|\bstart-process\b[^\r\n;&|]*\b-verb\s+runas\b|\btakeown\b|\bicacls\b[^\r\n;&|]*\b/grant\b|\bnet\s+localgroup\s+administrators\b`),
	},
	{
		ruleID:   "shell.download-and-execute",
		category: CategoryEgress,
		severity: SeverityCritical,
		title:    "Download and execute pipeline",
		pattern:  regexp.MustCompile(`(?i)((curl|wget|invoke-webrequest|invoke-restmethod|\biwr\b|\birm\b)[^\r\n]*(\|\s*(sh|bash|zsh|powershell|pwsh|iex|invoke-expression)\b|\biex\b|\binvoke-expression\b))|(\b(iex|invoke-expression)\b[^\r\n]*(iwr|irm|invoke-webrequest|invoke-restmethod|downloadstring))`),
	},
	{
		ruleID:   "shell.obfuscated-execution",
		category: CategoryCommand,
		severity: SeverityHigh,
		title:    "Obfuscated shell execution",
		pattern:  regexp.MustCompile(`(?i)\b(powershell|powershell\.exe|pwsh|pwsh\.exe)\b[^\r\n;&|]*\s-(encodedcommand|enc|e)\b|\bfrombase64string\s*\(|\bbase64\s+(-d|--decode)\b[^\r\n]*\|\s*(sh|bash|zsh|powershell|pwsh)\b`),
	},
	{
		ruleID:   "shell.secret-file-read",
		category: CategoryFile,
		severity: SeverityHigh,
		title:    "Secret file read",
		pattern:  regexp.MustCompile(`(?i)(^|[;&|]\s*)(cat|type|get-content|gc|more|less)\b[^\r\n;&|]*(\.env\b|id_(rsa|dsa|ecdsa|ed25519)\b|\.aws[\\/]+credentials\b|credentials\.json\b|secret(s)?\.(json|ya?ml|txt|env)\b|\.npmrc\b|\.pypirc\b)`),
	},
	{
		ruleID:   "shell.environment-dump",
		category: CategoryCommand,
		severity: SeverityMedium,
		title:    "Environment variable dump",
		pattern:  regexp.MustCompile(`(?i)(^\s*(env|printenv)\s*($|[>#|&;]))|(^\s*set\s*($|[>#|&;]))|\b(get-childitem|gci|dir|ls|get-item)\s+(-path\s+)?env:`),
	},
	{
		ruleID:   "shell.network-transfer",
		category: CategoryEgress,
		severity: SeverityMedium,
		title:    "Network transfer command",
		pattern:  regexp.MustCompile(`(?i)(^|[;&|]\s*)(curl|wget|scp|rsync|ftp|tftp)\b|\b(invoke-webrequest|invoke-restmethod|iwr|irm)\b|\bbitsadmin\b[^\r\n;&|]*\b/transfer\b|\bcertutil\b[^\r\n;&|]*\b-urlcache\b`),
	},
	{
		ruleID:   "shell.package-install",
		category: CategoryCommand,
		severity: SeverityMedium,
		title:    "Package installation command",
		pattern:  regexp.MustCompile(`(?i)(^|[;&|]\s*)((npm|pnpm|yarn)\s+(install|add|global\s+add)\b|pip3?\s+install\b|python(3)?\s+-m\s+pip\s+install\b|apt(-get)?\s+install\b|brew\s+install\b|choco\s+install\b|winget\s+install\b|scoop\s+install\b|go\s+install\b|cargo\s+install\b|gem\s+install\b|nuget\s+install\b|dotnet\s+tool\s+install\b)`),
	},
	{
		ruleID:   "shell.git-force-push",
		category: CategoryEgress,
		severity: SeverityHigh,
		title:    "Git force push",
		pattern:  regexp.MustCompile(`(?i)(^|[;&|]\s*)git\s+push\b[^\r\n;&|]*(--force(-with-lease)?\b|\s-f\b)`),
	},
	{
		ruleID:   "shell.windows-system-change",
		category: CategoryCommand,
		severity: SeverityHigh,
		title:    "Windows execution policy, registry, or service change",
		pattern:  regexp.MustCompile(`(?i)\bset-executionpolicy\b|\b-executionpolicy\s+(bypass|unrestricted|remotesigned)\b|\breg(\.exe)?\s+(add|delete|import|load|unload)\b|\b(new-itemproperty|set-itemproperty|remove-itemproperty)\b[^\r\n;&|]*(hklm:|hkcu:|registry::|hkey_local_machine|hkey_current_user)|\b(new-service|set-service|start-service|stop-service|restart-service|remove-service)\b|\bsc(\.exe)?\s+(create|delete|config|start|stop)\b|\bnet\s+(start|stop)\b`),
	},
	{
		ruleID:   "shell.persistence",
		category: CategoryCommand,
		severity: SeverityHigh,
		title:    "Persistence mechanism change",
		pattern:  regexp.MustCompile(`(?i)\bschtasks(\.exe)?\b[^\r\n;&|]*/create\b|\bcrontab\b[^\r\n]*(-e|-l|\s-)|\bsystemctl\b[^\r\n;&|]*\benable\b|\blaunchctl\b[^\r\n;&|]*\b(load|bootstrap|enable)\b|\breg(\.exe)?\s+add\b[^\r\n;&|]*(\\run(once)?\b|currentversion\\run)|\b(new-itemproperty|set-itemproperty)\b[^\r\n;&|]*(\\run(once)?\b|currentversion\\run|hklm:[^;&|\r\n]*\\run|hkcu:[^;&|\r\n]*\\run)`),
	},
	{
		ruleID:   "shell.destructive-disk",
		category: CategoryFile,
		severity: SeverityCritical,
		title:    "Destructive disk command",
		pattern:  regexp.MustCompile(`(?i)(^|[;&|]\s*)(((sudo|doas)\s+)?dd\b[^\r\n;&|]*\bof=|((sudo|doas)\s+)?mkfs(\.[a-z0-9]+)?\b|format(\.com)?\s+[a-z]:|\bdiskpart\b)`),
	},
	{
		ruleID:   "shell.defense-evasion",
		category: CategoryCommand,
		severity: SeverityHigh,
		title:    "Security control or logging disabled",
		pattern:  regexp.MustCompile(`(?i)\bset-mppreference\b[^\r\n;&|]*(^|[\s;&|])-disable(realtime|ioav|behavior|script)monitoring\s+\$?true\b|\bnetsh\s+advfirewall\b[^\r\n;&|]*\bstate\s+off\b|\bset-netfirewallprofile\b[^\r\n;&|]*\b-enabled\s+(false|\$false)\b|\bwevtutil\b\s+cl\b|\bauditpol\b[^\r\n;&|]*/clear\b|\b(sc|sc\.exe|net)\s+(stop|config)\s+(windefend|mpssvc|eventlog|sense|wdnissvc)\b|\bsystemctl\b[^\r\n;&|]*\b(stop|disable|mask)\b[^\r\n;&|]*(firewalld|ufw|auditd|rsyslog|syslog|falcon|defender|mdatp)\b|\bufw\s+disable\b`),
	},
}
