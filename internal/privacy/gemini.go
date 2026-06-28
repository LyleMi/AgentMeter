package privacy

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"AgentMeter/internal/model"
)

type GeminiAdapter struct {
	Now        func() time.Time
	ConfigPath string
}

type jsonSettingDefinition struct {
	ID          string
	Group       string
	Title       string
	Description string
	Key         string
	Desired     any
	DefaultSafe bool
	Impact      string
	MergeArray  bool
}

func NewGeminiAdapter() GeminiAdapter {
	return GeminiAdapter{Now: time.Now}
}

func (a GeminiAdapter) Status() (model.PrivacyConfigStatus, error) {
	path, err := a.settingsPath()
	if err != nil {
		return model.PrivacyConfigStatus{}, err
	}
	content, exists, err := readOptionalFile(path)
	if err != nil {
		return model.PrivacyConfigStatus{}, err
	}
	return buildGeminiStatus(path, exists, content, nil), nil
}

func (a GeminiAdapter) Apply(settingIDs []string) (model.PrivacyConfigApplyResult, error) {
	path, err := a.settingsPath()
	if err != nil {
		return model.PrivacyConfigApplyResult{}, err
	}
	return applyJSONSettingsMutation(path, a.now, buildGeminiStatus, func(root map[string]any) ([]model.PrivacyConfigChange, []string, error) {
		selected, warnings := selectedJSONSettingDefinitions(settingIDs, geminiSettingDefinitions, "Gemini")
		changes := plannedJSONChanges(root, selected)
		if len(changes) > 0 {
			applyJSONSettings(root, selected)
		}
		return changes, warnings, nil
	})
}

func (a GeminiAdapter) ApplyChanges(edits []model.PrivacyConfigEdit) (model.PrivacyConfigApplyResult, error) {
	path, err := a.settingsPath()
	if err != nil {
		return model.PrivacyConfigApplyResult{}, err
	}
	return applyJSONSettingsMutation(path, a.now, buildGeminiStatus, func(root map[string]any) ([]model.PrivacyConfigChange, []string, error) {
		return applyJSONEdits(root, edits, geminiSettingDefinitions, "Gemini")
	})
}

func (a GeminiAdapter) now() time.Time {
	if a.Now != nil {
		return a.Now()
	}
	return time.Now()
}

func (a GeminiAdapter) settingsPath() (string, error) {
	if strings.TrimSpace(a.ConfigPath) != "" {
		return filepath.Clean(a.ConfigPath), nil
	}
	return geminiSettingsPath()
}

func geminiSettingsPath() (string, error) {
	if path := strings.TrimSpace(os.Getenv("AGENTMETER_GEMINI_SETTINGS_PATH")); path != "" {
		return filepath.Clean(path), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".gemini", "settings.json"), nil
}

func buildGeminiStatus(path string, exists bool, content []byte, warnings []string) model.PrivacyConfigStatus {
	root, err := parseJSONSettings(content)
	canApply := true
	if err != nil {
		canApply = false
		warnings = append(warnings, fmt.Sprintf("Gemini settings.json could not be parsed: %v", err))
		root = map[string]any{}
	}

	settings := make([]model.PrivacyConfigSetting, 0, len(geminiSettingDefinitions))
	summary := model.PrivacyConfigSummary{Total: len(geminiSettingDefinitions)}
	for _, definition := range geminiSettingDefinitions {
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
		Target:     "gemini",
		Name:       "Gemini CLI",
		ConfigPath: path,
		Exists:     exists,
		Summary:    summary,
		Settings:   settings,
		Warnings:   warnings,
	}
}

func parseJSONSettings(content []byte) (map[string]any, error) {
	if strings.TrimSpace(string(content)) == "" {
		return map[string]any{}, nil
	}
	var value any
	decoder := json.NewDecoder(strings.NewReader(stripJSONComments(string(content))))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New("settings file contains trailing JSON data")
		}
		return nil, err
	}
	root, ok := value.(map[string]any)
	if !ok {
		return nil, errors.New("settings file is not a JSON object")
	}
	return root, nil
}

func stripJSONComments(content string) string {
	var builder strings.Builder
	inString := false
	escaped := false
	for index := 0; index < len(content); index++ {
		ch := content[index]
		if escaped {
			builder.WriteByte(ch)
			escaped = false
			continue
		}
		if inString {
			builder.WriteByte(ch)
			if ch == '\\' {
				escaped = true
			} else if ch == '"' {
				inString = false
			}
			continue
		}
		if ch == '"' {
			inString = true
			builder.WriteByte(ch)
			continue
		}
		if ch == '/' && index+1 < len(content) {
			next := content[index+1]
			if next == '/' {
				index += 2
				for index < len(content) && content[index] != '\n' && content[index] != '\r' {
					index++
				}
				if index < len(content) {
					builder.WriteByte(content[index])
				}
				continue
			}
			if next == '*' {
				index += 2
				for index+1 < len(content) && !(content[index] == '*' && content[index+1] == '/') {
					if content[index] == '\n' || content[index] == '\r' {
						builder.WriteByte(content[index])
					}
					index++
				}
				if index+1 < len(content) {
					index++
				}
				continue
			}
		}
		builder.WriteByte(ch)
	}
	return builder.String()
}

func plannedJSONChanges(root map[string]any, selected []jsonSettingDefinition) []model.PrivacyConfigChange {
	changes := make([]model.PrivacyConfigChange, 0, len(selected))
	for _, definition := range selected {
		current, ok := nestedJSONValue(root, definition.Key)
		if jsonSettingHardened(current, ok, definition) {
			continue
		}
		var before any
		if ok {
			before = current
		}
		changes = append(changes, model.PrivacyConfigChange{
			ID:     definition.ID,
			Key:    definition.Key,
			Before: before,
			After:  jsonSettingAfter(current, ok, definition),
		})
	}
	return changes
}

func applyJSONSettings(root map[string]any, selected []jsonSettingDefinition) {
	for _, definition := range selected {
		current, ok := nestedJSONValue(root, definition.Key)
		if jsonSettingHardened(current, ok, definition) {
			continue
		}
		setNestedJSONValue(root, definition.Key, jsonSettingAfter(current, ok, definition))
	}
}

func jsonSettingHardened(current any, ok bool, definition jsonSettingDefinition) bool {
	if !ok {
		return false
	}
	if definition.MergeArray {
		return stringArrayContainsAll(current, desiredStrings(definition.Desired))
	}
	return jsonValuesEqual(current, definition.Desired)
}

func jsonSettingAfter(current any, ok bool, definition jsonSettingDefinition) any {
	if definition.MergeArray && ok {
		return mergedStringArray(current, desiredStrings(definition.Desired))
	}
	return cloneJSONValue(definition.Desired)
}

func jsonValuesEqual(left any, right any) bool {
	if leftNumber, ok := jsonNumberValue(left); ok {
		rightNumber, rightOK := jsonNumberValue(right)
		return rightOK && leftNumber == rightNumber
	}
	return reflect.DeepEqual(left, right)
}

func jsonNumberValue(value any) (float64, bool) {
	switch typed := value.(type) {
	case json.Number:
		number, err := typed.Float64()
		return number, err == nil
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int8:
		return float64(typed), true
	case int16:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case uint:
		return float64(typed), true
	case uint8:
		return float64(typed), true
	case uint16:
		return float64(typed), true
	case uint32:
		return float64(typed), true
	case uint64:
		return float64(typed), true
	default:
		return 0, false
	}
}

func nestedJSONValue(root map[string]any, key string) (any, bool) {
	parts := strings.Split(key, ".")
	var current any = root
	for _, part := range parts {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = object[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func setNestedJSONValue(root map[string]any, key string, value any) {
	parts := strings.Split(key, ".")
	current := root
	for _, part := range parts[:len(parts)-1] {
		next, ok := current[part].(map[string]any)
		if !ok {
			next = map[string]any{}
			current[part] = next
		}
		current = next
	}
	current[parts[len(parts)-1]] = cloneJSONValue(value)
}

func unsetNestedJSONValue(root map[string]any, key string) bool {
	parts := strings.Split(key, ".")
	current := root
	for _, part := range parts[:len(parts)-1] {
		next, ok := current[part].(map[string]any)
		if !ok {
			return false
		}
		current = next
	}
	leaf := parts[len(parts)-1]
	if _, ok := current[leaf]; !ok {
		return false
	}
	delete(current, leaf)
	return true
}

func jsonValueType(value any) string {
	switch value.(type) {
	case bool:
		return "bool"
	case string:
		return "string"
	case json.Number, float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return "number"
	case []string, []any:
		return "stringArray"
	default:
		return "string"
	}
}

func editableJSONNumber(value any) (any, bool) {
	switch typed := value.(type) {
	case json.Number:
		number, err := typed.Float64()
		if err != nil {
			return nil, false
		}
		return number, true
	case float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return typed, true
	default:
		return nil, false
	}
}

func desiredStrings(value any) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if value, ok := item.(string); ok {
				result = append(result, value)
			}
		}
		return result
	default:
		return nil
	}
}

func stringArrayContainsAll(current any, desired []string) bool {
	if len(desired) == 0 {
		return true
	}
	present := map[string]struct{}{}
	switch typed := current.(type) {
	case []any:
		for _, item := range typed {
			if value, ok := item.(string); ok {
				present[value] = struct{}{}
			}
		}
	case []string:
		for _, value := range typed {
			present[value] = struct{}{}
		}
	default:
		return false
	}
	for _, value := range desired {
		if _, ok := present[value]; !ok {
			return false
		}
	}
	return true
}

func mergedStringArray(current any, desired []string) []any {
	result := []any{}
	present := map[string]struct{}{}
	if typed, ok := current.([]any); ok {
		result = append(result, typed...)
		for _, item := range typed {
			if value, ok := item.(string); ok {
				present[value] = struct{}{}
			}
		}
	} else if typed, ok := current.([]string); ok {
		for _, value := range typed {
			result = append(result, value)
			present[value] = struct{}{}
		}
	}
	for _, value := range desired {
		if _, ok := present[value]; ok {
			continue
		}
		result = append(result, value)
	}
	return result
}

func cloneJSONValue(value any) any {
	switch typed := value.(type) {
	case []string:
		result := make([]any, 0, len(typed))
		for _, item := range typed {
			result = append(result, item)
		}
		return result
	case []any:
		return append([]any(nil), typed...)
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, item := range typed {
			result[key] = cloneJSONValue(item)
		}
		return result
	default:
		return typed
	}
}

func marshalJSONSettings(root map[string]any) ([]byte, error) {
	content, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(content, '\n'), nil
}

var geminiSettingDefinitions = []jsonSettingDefinition{
	{
		ID:          "privacy.usageStatisticsEnabled",
		Group:       "Usage",
		Title:       "Usage statistics",
		Description: "Opts out of Gemini CLI usage statistics collection.",
		Key:         "privacy.usageStatisticsEnabled",
		Desired:     false,
		DefaultSafe: false,
		Impact:      "Prevents Gemini CLI usage statistics from being sent to Google.",
	},
	{
		ID:          "telemetry.enabled",
		Group:       "Telemetry",
		Title:       "OpenTelemetry",
		Description: "Disables Gemini CLI OpenTelemetry emission.",
		Key:         "telemetry.enabled",
		Desired:     false,
		DefaultSafe: true,
		Impact:      "Prevents telemetry logs, metrics, and traces from being exported.",
	},
	{
		ID:          "telemetry.traces",
		Group:       "Telemetry",
		Title:       "Detailed traces",
		Description: "Disables detailed telemetry traces.",
		Key:         "telemetry.traces",
		Desired:     false,
		DefaultSafe: true,
		Impact:      "Avoids detailed attributes such as tool output and file-read trace data.",
	},
	{
		ID:          "telemetry.logPrompts",
		Group:       "Telemetry",
		Title:       "Prompt logging",
		Description: "Prevents prompts from being included in telemetry logs.",
		Key:         "telemetry.logPrompts",
		Desired:     false,
		DefaultSafe: false,
		Impact:      "Keeps prompt text out of telemetry if telemetry is enabled later.",
	},
	{
		ID:          "general.logRagSnippets",
		Group:       "Telemetry",
		Title:       "RAG snippet logging",
		Description: "Disables local logging of full RAG snippets.",
		Key:         "general.logRagSnippets",
		Desired:     false,
		DefaultSafe: true,
		Impact:      "Avoids writing retrieved code customization snippets to local debug logs.",
	},
	{
		ID:          "general.checkpointing.enabled",
		Group:       "Local retention",
		Title:       "Session checkpointing",
		Description: "Disables session checkpointing.",
		Key:         "general.checkpointing.enabled",
		Desired:     false,
		DefaultSafe: true,
		Impact:      "Avoids extra recovery snapshots of working state.",
	},
	{
		ID:          "general.sessionRetention.enabled",
		Group:       "Local retention",
		Title:       "Session cleanup",
		Description: "Keeps automatic session cleanup enabled.",
		Key:         "general.sessionRetention.enabled",
		Desired:     true,
		DefaultSafe: true,
		Impact:      "Ensures old Gemini CLI chats are eligible for automatic cleanup.",
	},
	{
		ID:          "general.sessionRetention.maxAge",
		Group:       "Local retention",
		Title:       "Chat retention window",
		Description: "Reduces the Gemini CLI chat retention window.",
		Key:         "general.sessionRetention.maxAge",
		Desired:     "7d",
		DefaultSafe: false,
		Impact:      "Keeps fewer local chat records than the default 30 day retention window.",
	},
	{
		ID:          "tools.sandboxNetworkAccess",
		Group:       "Network",
		Title:       "Sandbox network access",
		Description: "Disables network access inside the Gemini CLI sandbox.",
		Key:         "tools.sandboxNetworkAccess",
		Desired:     false,
		DefaultSafe: true,
		Impact:      "Keeps sandboxed tool execution offline unless explicitly changed later.",
	},
	{
		ID:          "tools.exclude.web",
		Group:       "Network",
		Title:       "Web tools",
		Description: "Excludes Gemini CLI web search and web fetch tools.",
		Key:         "tools.exclude",
		Desired:     []string{"google_web_search", "web_fetch"},
		DefaultSafe: false,
		Impact:      "Prevents the model from using built-in web search or URL fetch tools by default.",
		MergeArray:  true,
	},
	{
		ID:          "experimental.directWebFetch",
		Group:       "Network",
		Title:       "Direct web fetch",
		Description: "Disables direct web fetch behavior.",
		Key:         "experimental.directWebFetch",
		Desired:     false,
		DefaultSafe: true,
		Impact:      "Avoids web fetch paths that bypass LLM summarization.",
	},
	{
		ID:          "advanced.ignoreLocalEnv",
		Group:       "Environment",
		Title:       "Local .env loading",
		Description: "Ignores generic project .env files.",
		Key:         "advanced.ignoreLocalEnv",
		Desired:     true,
		DefaultSafe: false,
		Impact:      "Reduces accidental loading of project secrets into the Gemini CLI process.",
	},
	{
		ID:          "security.environmentVariableRedaction.enabled",
		Group:       "Environment",
		Title:       "Environment variable redaction",
		Description: "Enables redaction for sensitive environment variables.",
		Key:         "security.environmentVariableRedaction.enabled",
		Desired:     true,
		DefaultSafe: false,
		Impact:      "Redacts environment variables that may contain secrets.",
	},
	{
		ID:          "security.disableYoloMode",
		Group:       "Approval",
		Title:       "YOLO mode",
		Description: "Disables YOLO mode even when requested by flag.",
		Key:         "security.disableYoloMode",
		Desired:     true,
		DefaultSafe: false,
		Impact:      "Prevents broad automatic approval from being enabled accidentally.",
	},
	{
		ID:          "security.disableAlwaysAllow",
		Group:       "Approval",
		Title:       "Always allow",
		Description: "Disables persistent Always allow choices.",
		Key:         "security.disableAlwaysAllow",
		Desired:     true,
		DefaultSafe: false,
		Impact:      "Reduces long-lived tool approvals that can leak into future sessions.",
	},
	{
		ID:          "security.enablePermanentToolApproval",
		Group:       "Approval",
		Title:       "Permanent tool approval",
		Description: "Disables permanent tool approval.",
		Key:         "security.enablePermanentToolApproval",
		Desired:     false,
		DefaultSafe: true,
		Impact:      "Avoids future-session approvals being added from confirmation dialogs.",
	},
	{
		ID:          "security.blockGitExtensions",
		Group:       "Extensions",
		Title:       "Git extensions",
		Description: "Blocks installing and loading extensions from Git.",
		Key:         "security.blockGitExtensions",
		Desired:     true,
		DefaultSafe: false,
		Impact:      "Reduces exposure to remote extension code and extension-provided tools.",
	},
	{
		ID:          "agents.browser.confirmSensitiveActions",
		Group:       "Browser",
		Title:       "Sensitive browser actions",
		Description: "Requires confirmation for sensitive browser actions.",
		Key:         "agents.browser.confirmSensitiveActions",
		Desired:     true,
		DefaultSafe: false,
		Impact:      "Requires manual confirmation before filling forms or running browser scripts.",
	},
	{
		ID:          "agents.browser.blockFileUploads",
		Group:       "Browser",
		Title:       "Browser file uploads",
		Description: "Blocks file uploads from the browser agent.",
		Key:         "agents.browser.blockFileUploads",
		Desired:     true,
		DefaultSafe: false,
		Impact:      "Prevents browser automation from uploading local files.",
	},
	{
		ID:          "experimental.voiceMode",
		Group:       "Voice",
		Title:       "Voice mode",
		Description: "Disables experimental voice mode.",
		Key:         "experimental.voiceMode",
		Desired:     false,
		DefaultSafe: true,
		Impact:      "Avoids voice workflows that may send recordings to a cloud transcription backend.",
	},
	{
		ID:          "experimental.autoMemory",
		Group:       "Memory",
		Title:       "Auto Memory",
		Description: "Disables automatic memory and skill extraction from past sessions.",
		Key:         "experimental.autoMemory",
		Desired:     false,
		DefaultSafe: true,
		Impact:      "Prevents background model calls over selected local transcript content.",
	},
	{
		ID:          "context.loadMemoryFromIncludeDirectories",
		Group:       "Memory",
		Title:       "Include-directory memory",
		Description: "Disables loading memory files from include directories.",
		Key:         "context.loadMemoryFromIncludeDirectories",
		Desired:     false,
		DefaultSafe: true,
		Impact:      "Keeps /memory reload scoped to the current directory by default.",
	},
	{
		ID:          "skills.enabled",
		Group:       "Memory",
		Title:       "Agent skills",
		Description: "Disables Gemini CLI agent skills.",
		Key:         "skills.enabled",
		Desired:     false,
		DefaultSafe: false,
		Impact:      "Avoids injecting local skill instructions into future agent context.",
	},
}
