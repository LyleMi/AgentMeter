package privacy

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"AgentMeter/internal/model"
)

const (
	statusHardened  = "hardened"
	statusImplicit  = "implicit"
	statusAttention = "attention"
)

type CodexAdapter struct {
	Now func() time.Time
}

type settingDefinition struct {
	ID          string
	Group       string
	Title       string
	Description string
	Table       string
	Key         string
	Desired     configValue
	DefaultSafe bool
	Impact      string
}

type configValue struct {
	Kind   string
	Bool   bool
	String string
	Raw    string
}

func NewCodexAdapter() CodexAdapter {
	return CodexAdapter{Now: time.Now}
}

func (a CodexAdapter) Status() (model.PrivacyConfigStatus, error) {
	path, err := codexConfigPath()
	if err != nil {
		return model.PrivacyConfigStatus{}, err
	}
	content, exists, err := readOptionalFile(path)
	if err != nil {
		return model.PrivacyConfigStatus{}, err
	}
	return buildCodexStatus(path, exists, content, nil), nil
}

func (a CodexAdapter) Apply(settingIDs []string) (model.PrivacyConfigApplyResult, error) {
	path, err := codexConfigPath()
	if err != nil {
		return model.PrivacyConfigApplyResult{}, err
	}
	original, exists, err := readOptionalFile(path)
	if err != nil {
		return model.PrivacyConfigApplyResult{}, err
	}

	selected, warnings := selectedSettingDefinitions(settingIDs)
	changes := plannedChanges(original, selected)
	result := model.PrivacyConfigApplyResult{
		Changed:  changes,
		Warnings: warnings,
	}
	if len(changes) == 0 {
		result.Status = buildCodexStatus(path, exists, original, warnings)
		return result, nil
	}

	updated := applySettingsToContent(original, selected)
	if bytes.Equal(updated, original) {
		result.Status = buildCodexStatus(path, exists, original, warnings)
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

	result.Status = buildCodexStatus(path, true, updated, warnings)
	return result, nil
}

func (a CodexAdapter) ApplyChanges(edits []model.PrivacyConfigEdit) (model.PrivacyConfigApplyResult, error) {
	path, err := codexConfigPath()
	if err != nil {
		return model.PrivacyConfigApplyResult{}, err
	}
	original, exists, err := readOptionalFile(path)
	if err != nil {
		return model.PrivacyConfigApplyResult{}, err
	}

	updated, changes, warnings, err := applyCodexEditsToContent(original, edits)
	if err != nil {
		return model.PrivacyConfigApplyResult{}, err
	}
	result := model.PrivacyConfigApplyResult{
		Changed:  changes,
		Warnings: warnings,
	}
	if len(changes) == 0 || bytes.Equal(updated, original) {
		result.Status = buildCodexStatus(path, exists, original, warnings)
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

	result.Status = buildCodexStatus(path, true, updated, warnings)
	return result, nil
}

func (a CodexAdapter) now() time.Time {
	if a.Now != nil {
		return a.Now()
	}
	return time.Now()
}

func codexConfigPath() (string, error) {
	if root := strings.TrimSpace(os.Getenv("CODEX_HOME")); root != "" {
		return filepath.Join(filepath.Clean(root), "config.toml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codex", "config.toml"), nil
}

func readOptionalFile(path string) ([]byte, bool, error) {
	content, err := os.ReadFile(path)
	if err == nil {
		return content, true, nil
	}
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	return nil, false, err
}

func buildCodexStatus(path string, exists bool, content []byte, warnings []string) model.PrivacyConfigStatus {
	doc := parseTOML(content)
	settings := make([]model.PrivacyConfigSetting, 0, len(codexSettingDefinitions))
	summary := model.PrivacyConfigSummary{Total: len(codexSettingDefinitions)}
	for _, definition := range codexSettingDefinitions {
		current, ok := doc.Value(definition.FullKey())
		status := statusAttention
		var currentValue any
		if ok {
			currentValue = current.JSON()
			if current.Equal(definition.Desired) {
				status = statusHardened
			}
		} else if definition.DefaultSafe {
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

		settings = append(settings, model.PrivacyConfigSetting{
			ID:               definition.ID,
			Group:            definition.Group,
			Title:            definition.Title,
			Description:      definition.Description,
			Key:              definition.FullKey(),
			DesiredValue:     definition.Desired.JSON(),
			StrictValue:      definition.Desired.JSON(),
			ValueType:        definition.Desired.ValueType(),
			Configured:       ok,
			SupportsUnset:    true,
			CurrentValue:     currentValue,
			Status:           status,
			Impact:           definition.Impact,
			CanApply:         true,
		})
	}
	if summary.Total > 0 {
		summary.Score = ((summary.Hardened + summary.Implicit) * 100) / summary.Total
	}
	return model.PrivacyConfigStatus{
		Target:     "codex",
		Name:       "Codex",
		ConfigPath: path,
		Exists:     exists,
		Summary:    summary,
		Settings:   settings,
		Warnings:   warnings,
	}
}

func selectedSettingDefinitions(settingIDs []string) ([]settingDefinition, []string) {
	if len(settingIDs) == 0 {
		return append([]settingDefinition(nil), codexSettingDefinitions...), nil
	}
	ids := map[string]struct{}{}
	for _, id := range settingIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			ids[id] = struct{}{}
		}
	}
	if len(ids) == 0 {
		return append([]settingDefinition(nil), codexSettingDefinitions...), nil
	}

	var selected []settingDefinition
	for _, definition := range codexSettingDefinitions {
		if _, ok := ids[definition.ID]; ok {
			selected = append(selected, definition)
			delete(ids, definition.ID)
		}
	}
	warnings := unknownSettingWarnings(ids)
	return selected, warnings
}

func unknownSettingWarnings(ids map[string]struct{}) []string {
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
		warnings = append(warnings, fmt.Sprintf("unknown Codex privacy setting %q was ignored", id))
	}
	return warnings
}

func plannedChanges(content []byte, selected []settingDefinition) []model.PrivacyConfigChange {
	doc := parseTOML(content)
	changes := make([]model.PrivacyConfigChange, 0, len(selected))
	for _, definition := range selected {
		current, ok := doc.Value(definition.FullKey())
		if ok && current.Equal(definition.Desired) {
			continue
		}
		var before any
		if ok {
			before = current.JSON()
		}
		changes = append(changes, model.PrivacyConfigChange{
			ID:     definition.ID,
			Key:    definition.FullKey(),
			Before: before,
			After:  definition.Desired.JSON(),
		})
	}
	return changes
}

func applySettingsToContent(content []byte, selected []settingDefinition) []byte {
	updated := content
	for _, definition := range selected {
		doc := parseTOML(updated)
		current, ok := doc.Value(definition.FullKey())
		if ok && current.Equal(definition.Desired) {
			continue
		}
		doc.Set(definition.Table, definition.Key, definition.Desired)
		updated = doc.Bytes()
	}
	return updated
}

func applyCodexEditsToContent(content []byte, edits []model.PrivacyConfigEdit) ([]byte, []model.PrivacyConfigChange, []string, error) {
	updated := content
	changes := make([]model.PrivacyConfigChange, 0, len(edits))
	unknown := map[string]struct{}{}
	definitions := codexDefinitionByID()

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
			return nil, nil, nil, fmt.Errorf("invalid Codex privacy change op %q for %q", edit.Op, edit.ID)
		}

		doc := parseTOML(updated)
		current, configured := doc.Value(definition.FullKey())
		var before any
		if configured {
			before = current.JSON()
		}

		switch op {
		case "set":
			value, err := configValueFromJSON(definition, edit.Value)
			if err != nil {
				return nil, nil, nil, err
			}
			if configured && current.Equal(value) {
				continue
			}
			doc.Set(definition.Table, definition.Key, value)
			updated = doc.Bytes()
			changes = append(changes, model.PrivacyConfigChange{
				ID:     definition.ID,
				Key:    definition.FullKey(),
				Before: before,
				After:  value.JSON(),
			})
		case "unset":
			if !configured {
				continue
			}
			doc.Unset(definition.Table, definition.Key)
			updated = doc.Bytes()
			changes = append(changes, model.PrivacyConfigChange{
				ID:     definition.ID,
				Key:    definition.FullKey(),
				Before: before,
				After:  nil,
			})
		}
	}

	return updated, changes, unknownSettingWarnings(unknown), nil
}

func codexDefinitionByID() map[string]settingDefinition {
	definitions := make(map[string]settingDefinition, len(codexSettingDefinitions))
	for _, definition := range codexSettingDefinitions {
		definitions[definition.ID] = definition
	}
	return definitions
}

func configValueFromJSON(definition settingDefinition, value any) (configValue, error) {
	switch definition.Desired.Kind {
	case "bool":
		typed, ok := value.(bool)
		if !ok {
			return configValue{}, fmt.Errorf("Codex privacy setting %q requires a bool value", definition.ID)
		}
		return boolValue(typed), nil
	case "string":
		typed, ok := value.(string)
		if !ok {
			return configValue{}, fmt.Errorf("Codex privacy setting %q requires a string value", definition.ID)
		}
		return stringValue(typed), nil
	default:
		return configValue{}, fmt.Errorf("Codex privacy setting %q does not support editable values", definition.ID)
	}
}

func backupConfigPath(path string, now time.Time) string {
	stamp := now.UTC().Format("20060102T150405.000000000Z")
	return filepath.Join(filepath.Dir(path), fmt.Sprintf("%s.%s.bak", filepath.Base(path), stamp))
}

func boolValue(value bool) configValue {
	return configValue{Kind: "bool", Bool: value}
}

func stringValue(value string) configValue {
	return configValue{Kind: "string", String: value}
}

func (v configValue) Equal(other configValue) bool {
	if v.Kind != other.Kind {
		return false
	}
	switch v.Kind {
	case "bool":
		return v.Bool == other.Bool
	case "string":
		return v.String == other.String
	default:
		return strings.TrimSpace(v.Raw) == strings.TrimSpace(other.Raw)
	}
}

func (v configValue) JSON() any {
	switch v.Kind {
	case "bool":
		return v.Bool
	case "string":
		return v.String
	default:
		return strings.TrimSpace(v.Raw)
	}
}

func (v configValue) TOML() string {
	switch v.Kind {
	case "bool":
		if v.Bool {
			return "true"
		}
		return "false"
	case "string":
		return strconv.Quote(v.String)
	default:
		return strings.TrimSpace(v.Raw)
	}
}

func (v configValue) ValueType() string {
	switch v.Kind {
	case "bool":
		return "bool"
	case "string":
		return "string"
	default:
		return "string"
	}
}

func (d settingDefinition) FullKey() string {
	if d.Table == "" {
		return d.Key
	}
	return d.Table + "." + d.Key
}

var codexSettingDefinitions = []settingDefinition{
	{
		ID:          "analytics.enabled",
		Group:       "Telemetry",
		Title:       "Analytics",
		Description: "Disables Codex analytics collection in the user config.",
		Table:       "analytics",
		Key:         "enabled",
		Desired:     boolValue(false),
		DefaultSafe: false,
		Impact:      "Keeps analytics disabled explicitly.",
	},
	{
		ID:          "otel.exporter",
		Group:       "Telemetry",
		Title:       "OpenTelemetry exporter",
		Description: "Disables the general OpenTelemetry exporter.",
		Table:       "otel",
		Key:         "exporter",
		Desired:     stringValue("none"),
		DefaultSafe: true,
		Impact:      "Prevents telemetry export from the Codex process.",
	},
	{
		ID:          "otel.trace_exporter",
		Group:       "Telemetry",
		Title:       "Trace exporter",
		Description: "Disables OpenTelemetry trace export.",
		Table:       "otel",
		Key:         "trace_exporter",
		Desired:     stringValue("none"),
		DefaultSafe: true,
		Impact:      "Prevents trace spans from leaving the machine.",
	},
	{
		ID:          "otel.metrics_exporter",
		Group:       "Telemetry",
		Title:       "Metrics exporter",
		Description: "Disables OpenTelemetry metrics export.",
		Table:       "otel",
		Key:         "metrics_exporter",
		Desired:     stringValue("none"),
		DefaultSafe: false,
		Impact:      "Prevents metrics export from the Codex process.",
	},
	{
		ID:          "otel.log_user_prompt",
		Group:       "Telemetry",
		Title:       "Prompt logging",
		Description: "Keeps user prompt logging disabled for telemetry.",
		Table:       "otel",
		Key:         "log_user_prompt",
		Desired:     boolValue(false),
		DefaultSafe: true,
		Impact:      "Avoids including prompt text in telemetry logs.",
	},
	{
		ID:          "web_search",
		Group:       "Network",
		Title:       "Web search",
		Description: "Disables Codex web search from user config.",
		Key:         "web_search",
		Desired:     stringValue("disabled"),
		DefaultSafe: false,
		Impact:      "Keeps prompts and search queries from using web search by default.",
	},
	{
		ID:          "history.persistence",
		Group:       "Local history",
		Title:       "Conversation history",
		Description: "Disables local Codex history persistence.",
		Table:       "history",
		Key:         "persistence",
		Desired:     stringValue("none"),
		DefaultSafe: false,
		Impact:      "Reduces local retention of prompts and responses.",
	},
	{
		ID:          "features.memories",
		Group:       "Memory",
		Title:       "Memory feature",
		Description: "Disables the Codex memory feature.",
		Table:       "features",
		Key:         "memories",
		Desired:     boolValue(false),
		DefaultSafe: true,
		Impact:      "Keeps durable memory features disabled explicitly.",
	},
	{
		ID:          "memories.generate_memories",
		Group:       "Memory",
		Title:       "Generate memories",
		Description: "Prevents Codex from generating memories.",
		Table:       "memories",
		Key:         "generate_memories",
		Desired:     boolValue(false),
		DefaultSafe: true,
		Impact:      "Avoids creating durable memory records from conversations.",
	},
	{
		ID:          "memories.use_memories",
		Group:       "Memory",
		Title:       "Use memories",
		Description: "Prevents Codex from using saved memories.",
		Table:       "memories",
		Key:         "use_memories",
		Desired:     boolValue(false),
		DefaultSafe: true,
		Impact:      "Avoids injecting saved memories into future context.",
	},
	{
		ID:          "memories.disable_on_external_context",
		Group:       "Memory",
		Title:       "External context memory guard",
		Description: "Keeps memories disabled when external context is present.",
		Table:       "memories",
		Key:         "disable_on_external_context",
		Desired:     boolValue(true),
		DefaultSafe: true,
		Impact:      "Reduces memory use when outside context may be present.",
	},
	{
		ID:          "sandbox_workspace_write.network_access",
		Group:       "Network",
		Title:       "Workspace network access",
		Description: "Disables network access for workspace-write sandbox mode.",
		Table:       "sandbox_workspace_write",
		Key:         "network_access",
		Desired:     boolValue(false),
		DefaultSafe: true,
		Impact:      "Keeps sandboxed commands offline unless explicitly changed later.",
	},
	{
		ID:          "shell_environment_policy.inherit",
		Group:       "Environment",
		Title:       "Shell environment inheritance",
		Description: "Limits inherited shell environment variables to Codex core defaults.",
		Table:       "shell_environment_policy",
		Key:         "inherit",
		Desired:     stringValue("core"),
		DefaultSafe: false,
		Impact:      "Reduces accidental exposure of environment variables to shell commands.",
	},
	{
		ID:          "shell_environment_policy.ignore_default_excludes",
		Group:       "Environment",
		Title:       "Default environment excludes",
		Description: "Keeps Codex default environment-variable excludes active.",
		Table:       "shell_environment_policy",
		Key:         "ignore_default_excludes",
		Desired:     boolValue(false),
		DefaultSafe: true,
		Impact:      "Preserves default filtering for sensitive environment variables.",
	},
}
