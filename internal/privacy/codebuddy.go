package privacy

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"AgentMeter/internal/model"
)

type CodeBuddyAdapter struct {
	Now        func() time.Time
	ConfigPath string
}

func NewCodeBuddyAdapter() CodeBuddyAdapter {
	return CodeBuddyAdapter{Now: time.Now}
}

func (a CodeBuddyAdapter) Status() (model.PrivacyConfigStatus, error) {
	path, err := a.settingsPath()
	if err != nil {
		return model.PrivacyConfigStatus{}, err
	}
	content, exists, err := readOptionalFile(path)
	if err != nil {
		return model.PrivacyConfigStatus{}, err
	}
	return buildCodeBuddyStatus(path, exists, content, nil), nil
}

func (a CodeBuddyAdapter) Apply(settingIDs []string) (model.PrivacyConfigApplyResult, error) {
	path, err := a.settingsPath()
	if err != nil {
		return model.PrivacyConfigApplyResult{}, err
	}
	original, exists, err := readOptionalFile(path)
	if err != nil {
		return model.PrivacyConfigApplyResult{}, err
	}
	root, err := parseJSONSettings(original)
	if err != nil {
		return model.PrivacyConfigApplyResult{}, err
	}

	selected, warnings := selectedCodeBuddySettingDefinitions(settingIDs)
	changes := plannedJSONChanges(root, selected)
	result := model.PrivacyConfigApplyResult{
		Changed:  changes,
		Warnings: warnings,
	}
	if len(changes) == 0 {
		result.Status = buildCodeBuddyStatus(path, exists, original, warnings)
		return result, nil
	}

	applyJSONSettings(root, selected)
	updated, err := marshalJSONSettings(root)
	if err != nil {
		return model.PrivacyConfigApplyResult{}, err
	}
	if bytes.Equal(updated, original) {
		result.Status = buildCodeBuddyStatus(path, exists, original, warnings)
		return result, nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return model.PrivacyConfigApplyResult{}, err
	}
	perm := os.FileMode(0o644)
	if exists {
		stat, err := os.Stat(path)
		if err != nil {
			return model.PrivacyConfigApplyResult{}, err
		}
		perm = stat.Mode().Perm()
		backupPath := backupConfigPath(path, a.now())
		if err := os.WriteFile(backupPath, original, perm); err != nil {
			return model.PrivacyConfigApplyResult{}, err
		}
		result.BackupPath = backupPath
	}
	if err := os.WriteFile(path, updated, perm); err != nil {
		return model.PrivacyConfigApplyResult{}, err
	}

	result.Status = buildCodeBuddyStatus(path, true, updated, warnings)
	return result, nil
}

func (a CodeBuddyAdapter) ApplyChanges(edits []model.PrivacyConfigEdit) (model.PrivacyConfigApplyResult, error) {
	path, err := a.settingsPath()
	if err != nil {
		return model.PrivacyConfigApplyResult{}, err
	}
	original, exists, err := readOptionalFile(path)
	if err != nil {
		return model.PrivacyConfigApplyResult{}, err
	}
	root, err := parseJSONSettings(original)
	if err != nil {
		return model.PrivacyConfigApplyResult{}, err
	}

	changes, warnings, err := applyCodeBuddyEdits(root, edits)
	if err != nil {
		return model.PrivacyConfigApplyResult{}, err
	}
	result := model.PrivacyConfigApplyResult{
		Changed:  changes,
		Warnings: warnings,
	}
	if len(changes) == 0 {
		result.Status = buildCodeBuddyStatus(path, exists, original, warnings)
		return result, nil
	}

	updated, err := marshalJSONSettings(root)
	if err != nil {
		return model.PrivacyConfigApplyResult{}, err
	}
	if bytes.Equal(updated, original) {
		result.Status = buildCodeBuddyStatus(path, exists, original, warnings)
		return result, nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return model.PrivacyConfigApplyResult{}, err
	}
	perm := os.FileMode(0o644)
	if exists {
		stat, err := os.Stat(path)
		if err != nil {
			return model.PrivacyConfigApplyResult{}, err
		}
		perm = stat.Mode().Perm()
		backupPath := backupConfigPath(path, a.now())
		if err := os.WriteFile(backupPath, original, perm); err != nil {
			return model.PrivacyConfigApplyResult{}, err
		}
		result.BackupPath = backupPath
	}
	if err := os.WriteFile(path, updated, perm); err != nil {
		return model.PrivacyConfigApplyResult{}, err
	}

	result.Status = buildCodeBuddyStatus(path, true, updated, warnings)
	return result, nil
}

func (a CodeBuddyAdapter) now() time.Time {
	if a.Now != nil {
		return a.Now()
	}
	return time.Now()
}

func (a CodeBuddyAdapter) settingsPath() (string, error) {
	if strings.TrimSpace(a.ConfigPath) != "" {
		return filepath.Clean(a.ConfigPath), nil
	}
	return codeBuddySettingsPath()
}

func codeBuddySettingsPath() (string, error) {
	if path := strings.TrimSpace(os.Getenv("AGENTMETER_CODEBUDDY_SETTINGS_PATH")); path != "" {
		return filepath.Clean(path), nil
	}
	if dir := strings.TrimSpace(os.Getenv("CODEBUDDY_CONFIG_DIR")); dir != "" {
		return filepath.Join(filepath.Clean(dir), "settings.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codebuddy", "settings.json"), nil
}

func buildCodeBuddyStatus(path string, exists bool, content []byte, warnings []string) model.PrivacyConfigStatus {
	root, err := parseJSONSettings(content)
	canApply := true
	if err != nil {
		canApply = false
		warnings = append(warnings, fmt.Sprintf("CodeBuddy settings.json could not be parsed: %v", err))
		root = map[string]any{}
	}

	settings := make([]model.PrivacyConfigSetting, 0, len(codeBuddySettingDefinitions))
	summary := model.PrivacyConfigSummary{Total: len(codeBuddySettingDefinitions)}
	for _, definition := range codeBuddySettingDefinitions {
		current, ok := nestedJSONValue(root, definition.Key)
		status := statusAttention
		var currentValue any
		if ok {
			currentValue = current
			if jsonSettingHardened(current, ok, definition) {
				status = statusHardened
			}
		} else if definition.DefaultSafe && canApply {
			status = statusImplicit
		}

		switch status {
		case statusHardened:
			summary.Hardened++
		case statusImplicit:
			summary.Implicit++
		default:
			summary.Attention++
		}

		strict := definition.Desired
		if definition.MergeArray {
			strict = jsonSettingAfter(current, ok, definition)
		}
		settings = append(settings, model.PrivacyConfigSetting{
			ID:            definition.ID,
			Group:         definition.Group,
			Title:         definition.Title,
			Description:   definition.Description,
			Key:           definition.Key,
			DesiredValue:  definition.Desired,
			StrictValue:   strict,
			ValueType:     jsonValueType(definition.Desired),
			Configured:    ok,
			SupportsUnset: canApply,
			CurrentValue:  currentValue,
			Status:        status,
			Impact:        definition.Impact,
			CanApply:      canApply,
		})
	}
	if summary.Total > 0 {
		summary.Score = ((summary.Hardened + summary.Implicit) * 100) / summary.Total
	}
	return model.PrivacyConfigStatus{
		Target:     "codebuddy",
		Name:       "CodeBuddy Code/IDE",
		ConfigPath: path,
		Exists:     exists,
		Summary:    summary,
		Settings:   settings,
		Warnings:   warnings,
	}
}

func selectedCodeBuddySettingDefinitions(settingIDs []string) ([]jsonSettingDefinition, []string) {
	if len(settingIDs) == 0 {
		return append([]jsonSettingDefinition(nil), codeBuddySettingDefinitions...), nil
	}
	ids := map[string]struct{}{}
	for _, id := range settingIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			ids[id] = struct{}{}
		}
	}
	if len(ids) == 0 {
		return append([]jsonSettingDefinition(nil), codeBuddySettingDefinitions...), nil
	}

	var selected []jsonSettingDefinition
	for _, definition := range codeBuddySettingDefinitions {
		if _, ok := ids[definition.ID]; ok {
			selected = append(selected, definition)
			delete(ids, definition.ID)
		}
	}
	return selected, unknownCodeBuddySettingWarnings(ids)
}

func unknownCodeBuddySettingWarnings(ids map[string]struct{}) []string {
	if len(ids) == 0 {
		return nil
	}
	unknown := make([]string, 0, len(ids))
	for id := range ids {
		unknown = append(unknown, id)
	}
	sort.Strings(unknown)
	warnings := make([]string, 0, len(unknown))
	for _, id := range unknown {
		warnings = append(warnings, fmt.Sprintf("unknown CodeBuddy privacy setting %q was ignored", id))
	}
	return warnings
}

func applyCodeBuddyEdits(root map[string]any, edits []model.PrivacyConfigEdit) ([]model.PrivacyConfigChange, []string, error) {
	changes := make([]model.PrivacyConfigChange, 0, len(edits))
	unknown := map[string]struct{}{}
	definitions := codeBuddyDefinitionByID()

	for _, edit := range edits {
		id := strings.TrimSpace(edit.ID)
		definition, ok := definitions[id]
		if !ok {
			if id != "" {
				unknown[id] = struct{}{}
			}
			continue
		}

		op := strings.TrimSpace(strings.ToLower(edit.Op))
		if op != "set" && op != "unset" {
			return nil, nil, fmt.Errorf("invalid CodeBuddy privacy change op %q for %q", edit.Op, edit.ID)
		}

		current, configured := nestedJSONValue(root, definition.Key)
		var before any
		if configured {
			before = current
		}

		switch op {
		case "set":
			value, err := editableCodeBuddyJSONValue(definition, edit.Value)
			if err != nil {
				return nil, nil, err
			}
			if configured && jsonValuesEqual(current, value) {
				continue
			}
			setNestedJSONValue(root, definition.Key, value)
			changes = append(changes, model.PrivacyConfigChange{
				ID:     definition.ID,
				Key:    definition.Key,
				Before: before,
				After:  cloneJSONValue(value),
			})
		case "unset":
			if !configured {
				continue
			}
			unsetNestedJSONValue(root, definition.Key)
			changes = append(changes, model.PrivacyConfigChange{
				ID:     definition.ID,
				Key:    definition.Key,
				Before: before,
				After:  nil,
			})
		}
	}

	return changes, unknownCodeBuddySettingWarnings(unknown), nil
}

func codeBuddyDefinitionByID() map[string]jsonSettingDefinition {
	definitions := make(map[string]jsonSettingDefinition, len(codeBuddySettingDefinitions))
	for _, definition := range codeBuddySettingDefinitions {
		definitions[definition.ID] = definition
	}
	return definitions
}

func editableCodeBuddyJSONValue(definition jsonSettingDefinition, value any) (any, error) {
	switch jsonValueType(definition.Desired) {
	case "bool":
		typed, ok := value.(bool)
		if !ok {
			return nil, fmt.Errorf("CodeBuddy privacy setting %q requires a bool value", definition.ID)
		}
		return typed, nil
	case "string":
		typed, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("CodeBuddy privacy setting %q requires a string value", definition.ID)
		}
		return typed, nil
	case "number":
		typed, ok := editableJSONNumber(value)
		if !ok {
			return nil, fmt.Errorf("CodeBuddy privacy setting %q requires a number value", definition.ID)
		}
		return typed, nil
	case "stringArray":
		typed, ok := value.([]any)
		if !ok {
			return nil, fmt.Errorf("CodeBuddy privacy setting %q requires a stringArray value", definition.ID)
		}
		result := make([]any, 0, len(typed))
		for _, item := range typed {
			text, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("CodeBuddy privacy setting %q requires a stringArray value", definition.ID)
			}
			result = append(result, text)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("CodeBuddy privacy setting %q does not support editable values", definition.ID)
	}
}

var codeBuddySettingDefinitions = []jsonSettingDefinition{
	{
		ID:          "env.DISABLE_TELEMETRY",
		Group:       "Telemetry",
		Title:       "Telemetry",
		Description: "Disables CodeBuddy Code OpenTelemetry export through the user settings environment.",
		Key:         "env.DISABLE_TELEMETRY",
		Desired:     "1",
		DefaultSafe: false,
		Impact:      "Disables all CodeBuddy telemetry with the documented highest-priority opt-out.",
	},
	{
		ID:          "env.CODEBUDDY_CODE_ENABLE_TELEMETRY",
		Group:       "Telemetry",
		Title:       "Telemetry opt-in",
		Description: "Prevents the CodeBuddy telemetry opt-in environment variable from enabling export later.",
		Key:         "env.CODEBUDDY_CODE_ENABLE_TELEMETRY",
		Desired:     "0",
		DefaultSafe: true,
		Impact:      "Keeps user-level settings from explicitly opting into trace export.",
	},
	{
		ID:          "env.CLAUDE_CODE_ENABLE_TELEMETRY",
		Group:       "Telemetry",
		Title:       "Claude telemetry alias",
		Description: "Prevents the Claude-compatible telemetry opt-in variable from enabling CodeBuddy export.",
		Key:         "env.CLAUDE_CODE_ENABLE_TELEMETRY",
		Desired:     "0",
		DefaultSafe: true,
		Impact:      "Covers CodeBuddy's Claude Code telemetry compatibility environment variable.",
	},
	{
		ID:          "env.OTEL_TRACES_EXPORTER",
		Group:       "Telemetry",
		Title:       "OTel exporter",
		Description: "Disables the OpenTelemetry trace exporter in user settings.",
		Key:         "env.OTEL_TRACES_EXPORTER",
		Desired:     "none",
		DefaultSafe: true,
		Impact:      "Prevents user settings from sending CodeBuddy traces to an external collector.",
	},
	{
		ID:          "env.OTEL_LOG_USER_PROMPTS",
		Group:       "Telemetry",
		Title:       "Prompt recording",
		Description: "Keeps user prompt text out of OpenTelemetry spans.",
		Key:         "env.OTEL_LOG_USER_PROMPTS",
		Desired:     "0",
		DefaultSafe: true,
		Impact:      "Avoids recording full user prompts if telemetry is enabled elsewhere.",
	},
	{
		ID:          "env.OTEL_LOG_TOOL_DETAILS",
		Group:       "Telemetry",
		Title:       "Tool detail recording",
		Description: "Keeps tool parameters out of OpenTelemetry spans.",
		Key:         "env.OTEL_LOG_TOOL_DETAILS",
		Desired:     "0",
		DefaultSafe: true,
		Impact:      "Avoids recording file paths, commands, URLs, searches, and tool input attributes.",
	},
	{
		ID:          "env.OTEL_LOG_TOOL_CONTENT",
		Group:       "Telemetry",
		Title:       "Tool content recording",
		Description: "Keeps full tool input and output out of OpenTelemetry span events.",
		Key:         "env.OTEL_LOG_TOOL_CONTENT",
		Desired:     "0",
		DefaultSafe: true,
		Impact:      "Avoids recording tool inputs and results that can include source code or secrets.",
	},
	{
		ID:          "env.OTEL_LOG_RAW_API_BODIES",
		Group:       "Telemetry",
		Title:       "Raw API body recording",
		Description: "Keeps raw API request and response body recording disabled.",
		Key:         "env.OTEL_LOG_RAW_API_BODIES",
		Desired:     "0",
		DefaultSafe: true,
		Impact:      "Avoids future full request/response body capture if the reserved switch becomes active.",
	},
	{
		ID:          "env.DISABLE_ERROR_REPORTING",
		Group:       "Reporting",
		Title:       "Error reporting",
		Description: "Disables CodeBuddy error reporting through the user settings environment.",
		Key:         "env.DISABLE_ERROR_REPORTING",
		Desired:     "1",
		DefaultSafe: false,
		Impact:      "Reduces diagnostic error payloads sent outside the machine.",
	},
	{
		ID:          "env.DISABLE_FEEDBACK_COMMAND",
		Group:       "Reporting",
		Title:       "Feedback command",
		Description: "Disables the CodeBuddy feedback command.",
		Key:         "env.DISABLE_FEEDBACK_COMMAND",
		Desired:     "1",
		DefaultSafe: false,
		Impact:      "Prevents feedback command submissions from this environment.",
	},
	{
		ID:          "env.DISABLE_AUTOUPDATER",
		Group:       "Network",
		Title:       "Auto updater",
		Description: "Disables CodeBuddy automatic update checks through the user settings environment.",
		Key:         "env.DISABLE_AUTOUPDATER",
		Desired:     "1",
		DefaultSafe: false,
		Impact:      "Avoids background update traffic controlled by this settings file.",
	},
	{
		ID:          "autoUpdates",
		Group:       "Network",
		Title:       "Auto updates",
		Description: "Keeps CodeBuddy auto-update settings disabled.",
		Key:         "autoUpdates",
		Desired:     false,
		DefaultSafe: true,
		Impact:      "Prevents this user settings file from enabling automatic update checks.",
	},
	{
		ID:          "cleanupPeriodDays",
		Group:       "Local retention",
		Title:       "Chat retention window",
		Description: "Reduces the CodeBuddy local chat history retention window.",
		Key:         "cleanupPeriodDays",
		Desired:     7,
		DefaultSafe: false,
		Impact:      "Keeps fewer local chat records than CodeBuddy's documented 30 day default.",
	},
	{
		ID:          "memory.autoMemoryEnabled",
		Group:       "Memory",
		Title:       "Auto memory",
		Description: "Disables CodeBuddy automatic memory storage.",
		Key:         "memory.autoMemoryEnabled",
		Desired:     false,
		DefaultSafe: false,
		Impact:      "Prevents autonomous persistent memory extraction from conversations.",
	},
	{
		ID:          "env.CODEBUDDY_DISABLE_AUTO_MEMORY",
		Group:       "Memory",
		Title:       "Auto memory environment",
		Description: "Disables CodeBuddy automatic memory with the documented environment variable.",
		Key:         "env.CODEBUDDY_DISABLE_AUTO_MEMORY",
		Desired:     "1",
		DefaultSafe: false,
		Impact:      "Provides an environment-level fallback for disabling auto memory.",
	},
	{
		ID:          "memory.memoryExtraction",
		Group:       "Memory",
		Title:       "Memory extraction",
		Description: "Disables background memory extraction.",
		Key:         "memory.memoryExtraction",
		Desired:     false,
		DefaultSafe: true,
		Impact:      "Avoids extracting durable memory from conversations at the end of a session.",
	},
	{
		ID:          "memory.teamMemory.enabled",
		Group:       "Memory",
		Title:       "Team memory",
		Description: "Disables team memory mode from user settings.",
		Key:         "memory.teamMemory.enabled",
		Desired:     false,
		DefaultSafe: true,
		Impact:      "Prevents writing project memories into shared team memory storage from this setting.",
	},
	{
		ID:          "includeCoAuthoredBy",
		Group:       "Attribution",
		Title:       "Commit attribution",
		Description: "Disables CodeBuddy co-authored-by attribution in commits and pull requests.",
		Key:         "includeCoAuthoredBy",
		Desired:     false,
		DefaultSafe: false,
		Impact:      "Avoids adding CodeBuddy attribution metadata to generated git content.",
	},
	{
		ID:          "trustAll",
		Group:       "Trust",
		Title:       "Trust all directories",
		Description: "Keeps CodeBuddy directory trust prompts enabled.",
		Key:         "trustAll",
		Desired:     false,
		DefaultSafe: true,
		Impact:      "Prevents this settings file from globally trusting every working directory.",
	},
	{
		ID:          "permissions.defaultMode",
		Group:       "Permissions",
		Title:       "Default permission mode",
		Description: "Keeps CodeBuddy's default permission mode at the normal reviewed mode.",
		Key:         "permissions.defaultMode",
		Desired:     "default",
		DefaultSafe: true,
		Impact:      "Avoids starting sessions in auto, dontAsk, or bypassPermissions mode.",
	},
	{
		ID:          "permissions.disableBypassPermissionsMode",
		Group:       "Permissions",
		Title:       "Bypass permission mode",
		Description: "Disables activation of CodeBuddy bypassPermissions mode.",
		Key:         "permissions.disableBypassPermissionsMode",
		Desired:     "disable",
		DefaultSafe: false,
		Impact:      "Disables the dangerous skip-permissions mode and related CLI flags.",
	},
	{
		ID:          "permissions.disableAutoMode",
		Group:       "Permissions",
		Title:       "Auto permission mode",
		Description: "Disables activation of CodeBuddy auto permission mode.",
		Key:         "permissions.disableAutoMode",
		Desired:     "disable",
		DefaultSafe: false,
		Impact:      "Prevents automatic permission-mode decisions from this user settings file.",
	},
	{
		ID:          "permissions.deny",
		Group:       "Permissions",
		Title:       "Sensitive and web deny rules",
		Description: "Adds CodeBuddy deny rules for web access, common download commands, and sensitive files.",
		Key:         "permissions.deny",
		Desired: []string{
			"WebFetch",
			"WebSearch",
			"Bash(curl:*)",
			"Bash(wget:*)",
			"Read(./.env)",
			"Read(./.env.*)",
			"Read(./secrets/**)",
			"Read(~/.ssh/**)",
			"Read(~/.aws/**)",
			"Edit(**/*.env)",
			"Edit(**/*.key)",
			"Edit(**/*.pem)",
		},
		DefaultSafe: false,
		Impact:      "Reduces accidental exposure through web tools, download commands, and common secret-bearing files.",
		MergeArray:  true,
	},
	{
		ID:          "disableAllHooks",
		Group:       "Hooks",
		Title:       "Hooks",
		Description: "Disables CodeBuddy hooks from this user settings file.",
		Key:         "disableAllHooks",
		Desired:     true,
		DefaultSafe: false,
		Impact:      "Prevents hook commands from running before or after tool execution.",
	},
	{
		ID:          "allowUntrustedFrontmatterHooks",
		Group:       "Hooks",
		Title:       "Untrusted frontmatter hooks",
		Description: "Keeps untrusted agent and skill frontmatter hooks disabled.",
		Key:         "allowUntrustedFrontmatterHooks",
		Desired:     false,
		DefaultSafe: true,
		Impact:      "Prevents local or marketplace markdown files from silently launching shell commands.",
	},
	{
		ID:          "enableAllProjectMcpServers",
		Group:       "MCP",
		Title:       "Project MCP auto-approval",
		Description: "Disables automatic approval of all project MCP servers.",
		Key:         "enableAllProjectMcpServers",
		Desired:     false,
		DefaultSafe: true,
		Impact:      "Prevents project .mcp.json files from enabling every MCP server without review.",
	},
	{
		ID:          "enabledMcpjsonServers",
		Group:       "MCP",
		Title:       "Approved project MCP servers",
		Description: "Clears explicitly approved project MCP server names from user settings.",
		Key:         "enabledMcpjsonServers",
		Desired:     []string{},
		DefaultSafe: true,
		Impact:      "Requires project MCP servers to be reviewed instead of pre-approved.",
	},
}
