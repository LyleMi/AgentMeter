package privacy

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"AgentMeter/internal/model"
)

type ClaudeAdapter struct {
	Now        func() time.Time
	ConfigPath string
}

func NewClaudeAdapter() ClaudeAdapter {
	return ClaudeAdapter{Now: time.Now}
}

func (a ClaudeAdapter) Status() (model.PrivacyConfigStatus, error) {
	path, err := a.settingsPath()
	if err != nil {
		return model.PrivacyConfigStatus{}, err
	}
	content, exists, err := readOptionalFile(path)
	if err != nil {
		return model.PrivacyConfigStatus{}, err
	}
	return buildClaudeStatus(path, exists, content, nil), nil
}

func (a ClaudeAdapter) Apply(settingIDs []string) (model.PrivacyConfigApplyResult, error) {
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

	selected, warnings := selectedClaudeSettingDefinitions(settingIDs)
	changes := plannedJSONChanges(root, selected)
	result := model.PrivacyConfigApplyResult{
		Changed:  changes,
		Warnings: warnings,
	}
	if len(changes) == 0 {
		result.Status = buildClaudeStatus(path, exists, original, warnings)
		return result, nil
	}

	applyJSONSettings(root, selected)
	updated, err := marshalJSONSettings(root)
	if err != nil {
		return model.PrivacyConfigApplyResult{}, err
	}
	if bytes.Equal(updated, original) {
		result.Status = buildClaudeStatus(path, exists, original, warnings)
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

	result.Status = buildClaudeStatus(path, true, updated, warnings)
	return result, nil
}

func (a ClaudeAdapter) ApplyChanges(edits []model.PrivacyConfigEdit) (model.PrivacyConfigApplyResult, error) {
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

	changes, warnings, err := applyClaudeEdits(root, edits)
	if err != nil {
		return model.PrivacyConfigApplyResult{}, err
	}
	result := model.PrivacyConfigApplyResult{
		Changed:  changes,
		Warnings: warnings,
	}
	if len(changes) == 0 {
		result.Status = buildClaudeStatus(path, exists, original, warnings)
		return result, nil
	}

	updated, err := marshalJSONSettings(root)
	if err != nil {
		return model.PrivacyConfigApplyResult{}, err
	}
	if bytes.Equal(updated, original) {
		result.Status = buildClaudeStatus(path, exists, original, warnings)
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

	result.Status = buildClaudeStatus(path, true, updated, warnings)
	return result, nil
}

func (a ClaudeAdapter) now() time.Time {
	if a.Now != nil {
		return a.Now()
	}
	return time.Now()
}

func (a ClaudeAdapter) settingsPath() (string, error) {
	if strings.TrimSpace(a.ConfigPath) != "" {
		return filepath.Clean(a.ConfigPath), nil
	}
	return claudeSettingsPath()
}

func claudeSettingsPath() (string, error) {
	if path := strings.TrimSpace(os.Getenv("AGENTMETER_CLAUDE_SETTINGS_PATH")); path != "" {
		return filepath.Clean(path), nil
	}
	if dir := strings.TrimSpace(os.Getenv("CLAUDE_CONFIG_DIR")); dir != "" {
		return filepath.Join(filepath.Clean(dir), "settings.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "settings.json"), nil
}

func buildClaudeStatus(path string, exists bool, content []byte, warnings []string) model.PrivacyConfigStatus {
	root, err := parseJSONSettings(content)
	canApply := true
	if err != nil {
		canApply = false
		warnings = append(warnings, fmt.Sprintf("Claude Code settings.json could not be parsed: %v", err))
		root = map[string]any{}
	}

	settings := make([]model.PrivacyConfigSetting, 0, len(claudeSettingDefinitions))
	summary := model.PrivacyConfigSummary{Total: len(claudeSettingDefinitions)}
	for _, definition := range claudeSettingDefinitions {
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
		Target:     "claude",
		Name:       "Claude Code",
		ConfigPath: path,
		Exists:     exists,
		Summary:    summary,
		Settings:   settings,
		Warnings:   warnings,
	}
}

func selectedClaudeSettingDefinitions(settingIDs []string) ([]jsonSettingDefinition, []string) {
	if len(settingIDs) == 0 {
		return append([]jsonSettingDefinition(nil), claudeSettingDefinitions...), nil
	}
	ids := map[string]struct{}{}
	for _, id := range settingIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			ids[id] = struct{}{}
		}
	}
	if len(ids) == 0 {
		return append([]jsonSettingDefinition(nil), claudeSettingDefinitions...), nil
	}

	var selected []jsonSettingDefinition
	for _, definition := range claudeSettingDefinitions {
		if _, ok := ids[definition.ID]; ok {
			selected = append(selected, definition)
			delete(ids, definition.ID)
		}
	}
	return selected, unknownClaudeSettingWarnings(ids)
}

func unknownClaudeSettingWarnings(ids map[string]struct{}) []string {
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
		warnings = append(warnings, fmt.Sprintf("unknown Claude Code privacy setting %q was ignored", id))
	}
	return warnings
}

func applyClaudeEdits(root map[string]any, edits []model.PrivacyConfigEdit) ([]model.PrivacyConfigChange, []string, error) {
	changes := make([]model.PrivacyConfigChange, 0, len(edits))
	unknown := map[string]struct{}{}
	definitions := claudeDefinitionByID()

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
			return nil, nil, fmt.Errorf("invalid Claude Code privacy change op %q for %q", edit.Op, edit.ID)
		}

		current, configured := nestedJSONValue(root, definition.Key)
		var before any
		if configured {
			before = current
		}

		switch op {
		case "set":
			value, err := editableClaudeJSONValue(definition, edit.Value)
			if err != nil {
				return nil, nil, err
			}
			if configured && reflect.DeepEqual(current, value) {
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

	return changes, unknownClaudeSettingWarnings(unknown), nil
}

func claudeDefinitionByID() map[string]jsonSettingDefinition {
	definitions := make(map[string]jsonSettingDefinition, len(claudeSettingDefinitions))
	for _, definition := range claudeSettingDefinitions {
		definitions[definition.ID] = definition
	}
	return definitions
}

func editableClaudeJSONValue(definition jsonSettingDefinition, value any) (any, error) {
	switch jsonValueType(definition.Desired) {
	case "bool":
		typed, ok := value.(bool)
		if !ok {
			return nil, fmt.Errorf("Claude Code privacy setting %q requires a bool value", definition.ID)
		}
		return typed, nil
	case "string":
		typed, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("Claude Code privacy setting %q requires a string value", definition.ID)
		}
		return typed, nil
	case "stringArray":
		typed, ok := value.([]any)
		if !ok {
			return nil, fmt.Errorf("Claude Code privacy setting %q requires a stringArray value", definition.ID)
		}
		result := make([]any, 0, len(typed))
		for _, item := range typed {
			text, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("Claude Code privacy setting %q requires a stringArray value", definition.ID)
			}
			result = append(result, text)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("Claude Code privacy setting %q does not support editable values", definition.ID)
	}
}

var claudeSettingDefinitions = []jsonSettingDefinition{
	{
		ID:          "env.DISABLE_TELEMETRY",
		Group:       "Telemetry",
		Title:       "Telemetry",
		Description: "Disables Claude Code telemetry emission through the user settings environment.",
		Key:         "env.DISABLE_TELEMETRY",
		Desired:     "1",
		DefaultSafe: false,
		Impact:      "Prevents Claude Code telemetry from being enabled through this user settings file.",
	},
	{
		ID:          "env.DISABLE_ERROR_REPORTING",
		Group:       "Telemetry",
		Title:       "Error reporting",
		Description: "Disables Claude Code error reporting through the user settings environment.",
		Key:         "env.DISABLE_ERROR_REPORTING",
		Desired:     "1",
		DefaultSafe: false,
		Impact:      "Reduces diagnostic error payloads sent from Claude Code.",
	},
	{
		ID:          "env.DISABLE_FEEDBACK_COMMAND",
		Group:       "Feedback",
		Title:       "Feedback command",
		Description: "Disables the Claude Code feedback command.",
		Key:         "env.DISABLE_FEEDBACK_COMMAND",
		Desired:     "1",
		DefaultSafe: false,
		Impact:      "Prevents feedback command submissions from this environment.",
	},
	{
		ID:          "env.CLAUDE_CODE_DISABLE_FEEDBACK_SURVEY",
		Group:       "Feedback",
		Title:       "Feedback survey",
		Description: "Disables Claude Code feedback survey prompts.",
		Key:         "env.CLAUDE_CODE_DISABLE_FEEDBACK_SURVEY",
		Desired:     "1",
		DefaultSafe: false,
		Impact:      "Avoids survey flows that may send user feedback outside the machine.",
	},
	{
		ID:          "env.CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC",
		Group:       "Network",
		Title:       "Nonessential traffic",
		Description: "Disables nonessential Claude Code network traffic through the user settings environment.",
		Key:         "env.CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC",
		Desired:     "1",
		DefaultSafe: false,
		Impact:      "Limits Claude Code to essential service traffic where supported.",
	},
	{
		ID:          "env.CLAUDE_CODE_SUBPROCESS_ENV_SCRUB",
		Group:       "Environment",
		Title:       "Subprocess environment scrub",
		Description: "Requests subprocess environment scrubbing for Claude Code tool execution.",
		Key:         "env.CLAUDE_CODE_SUBPROCESS_ENV_SCRUB",
		Desired:     "1",
		DefaultSafe: false,
		Impact:      "Reduces accidental exposure of parent-process environment variables to subprocess tools.",
	},
	{
		ID:          "attribution.commit",
		Group:       "Attribution",
		Title:       "Commit attribution",
		Description: "Removes generated commit attribution text.",
		Key:         "attribution.commit",
		Desired:     "",
		DefaultSafe: false,
		Impact:      "Avoids adding Claude attribution metadata to generated commit messages.",
	},
	{
		ID:          "attribution.pr",
		Group:       "Attribution",
		Title:       "Pull request attribution",
		Description: "Removes generated pull request attribution text.",
		Key:         "attribution.pr",
		Desired:     "",
		DefaultSafe: false,
		Impact:      "Avoids adding Claude attribution metadata to generated pull request text.",
	},
	{
		ID:          "attribution.sessionUrl",
		Group:       "Attribution",
		Title:       "Session URL attribution",
		Description: "Disables Claude Code session URL attribution.",
		Key:         "attribution.sessionUrl",
		Desired:     false,
		DefaultSafe: false,
		Impact:      "Avoids adding shareable session URLs to generated attribution.",
	},
	{
		ID:          "fileCheckpointingEnabled",
		Group:       "Local retention",
		Title:       "File checkpointing",
		Description: "Disables Claude Code file checkpointing.",
		Key:         "fileCheckpointingEnabled",
		Desired:     false,
		DefaultSafe: false,
		Impact:      "Reduces local recovery snapshots of edited files.",
	},
	{
		ID:          "autoMemoryEnabled",
		Group:       "Memory",
		Title:       "Auto memory",
		Description: "Disables Claude Code automatic memory behavior.",
		Key:         "autoMemoryEnabled",
		Desired:     false,
		DefaultSafe: false,
		Impact:      "Avoids automatic durable memory extraction from conversations.",
	},
	{
		ID:          "disableArtifact",
		Group:       "Artifacts",
		Title:       "Artifacts",
		Description: "Disables Claude Code artifact generation.",
		Key:         "disableArtifact",
		Desired:     true,
		DefaultSafe: false,
		Impact:      "Reduces creation of additional generated artifact content outside normal chat output.",
	},
	{
		ID:          "disableClaudeAiConnectors",
		Group:       "Connectors",
		Title:       "Claude AI connectors",
		Description: "Disables Claude AI connector integrations.",
		Key:         "disableClaudeAiConnectors",
		Desired:     true,
		DefaultSafe: false,
		Impact:      "Prevents connector integrations from being available by default.",
	},
	{
		ID:          "permissions.deny",
		Group:       "Permissions",
		Title:       "Deny WebFetch",
		Description: "Adds WebFetch to the Claude Code permissions deny list.",
		Key:         "permissions.deny",
		Desired:     []string{"WebFetch"},
		DefaultSafe: false,
		Impact:      "Prevents Claude Code from using WebFetch unless this deny rule is removed.",
		MergeArray:  true,
	},
}
