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

func applyJSONSettingsMutation(
	path string,
	now func() time.Time,
	buildStatus func(string, bool, []byte, []string) model.PrivacyConfigStatus,
	mutate func(map[string]any) ([]model.PrivacyConfigChange, []string, error),
) (model.PrivacyConfigApplyResult, error) {
	original, exists, err := readOptionalFile(path)
	if err != nil {
		return model.PrivacyConfigApplyResult{}, err
	}
	root, err := parseJSONSettings(original)
	if err != nil {
		return model.PrivacyConfigApplyResult{}, err
	}

	changes, warnings, err := mutate(root)
	if err != nil {
		return model.PrivacyConfigApplyResult{}, err
	}
	result := model.PrivacyConfigApplyResult{
		Changed:  changes,
		Warnings: warnings,
	}
	if len(changes) == 0 {
		result.Status = buildStatus(path, exists, original, warnings)
		return result, nil
	}

	updated, err := marshalJSONSettings(root)
	if err != nil {
		return model.PrivacyConfigApplyResult{}, err
	}
	if bytes.Equal(updated, original) {
		result.Status = buildStatus(path, exists, original, warnings)
		return result, nil
	}

	backupPath, err := writeUpdatedConfig(path, original, updated, exists, now)
	if err != nil {
		return model.PrivacyConfigApplyResult{}, err
	}
	result.BackupPath = backupPath
	result.Status = buildStatus(path, true, updated, warnings)
	return result, nil
}

func writeUpdatedConfig(path string, original, updated []byte, exists bool, now func() time.Time) (string, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	perm := os.FileMode(0o644)
	var backupPath string
	if exists {
		stat, err := os.Stat(path)
		if err != nil {
			return "", err
		}
		perm = stat.Mode().Perm()
		backupPath = backupConfigPath(path, callNow(now))
		if err := os.WriteFile(backupPath, original, perm); err != nil {
			return "", err
		}
	}
	if err := os.WriteFile(path, updated, perm); err != nil {
		return "", err
	}
	return backupPath, nil
}

func callNow(now func() time.Time) time.Time {
	if now != nil {
		return now()
	}
	return time.Now()
}

func selectedJSONSettingDefinitions(settingIDs []string, definitions []jsonSettingDefinition, agentName string) ([]jsonSettingDefinition, []string) {
	if len(settingIDs) == 0 {
		return append([]jsonSettingDefinition(nil), definitions...), nil
	}
	ids := map[string]struct{}{}
	for _, id := range settingIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			ids[id] = struct{}{}
		}
	}
	if len(ids) == 0 {
		return append([]jsonSettingDefinition(nil), definitions...), nil
	}

	var selected []jsonSettingDefinition
	for _, definition := range definitions {
		if _, ok := ids[definition.ID]; ok {
			selected = append(selected, definition)
			delete(ids, definition.ID)
		}
	}
	return selected, unknownJSONSettingWarnings(ids, agentName)
}

func unknownJSONSettingWarnings(ids map[string]struct{}, agentName string) []string {
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
		warnings = append(warnings, fmt.Sprintf("unknown %s privacy setting %q was ignored", agentName, id))
	}
	return warnings
}

func applyJSONEdits(root map[string]any, edits []model.PrivacyConfigEdit, definitions []jsonSettingDefinition, agentName string) ([]model.PrivacyConfigChange, []string, error) {
	changes := make([]model.PrivacyConfigChange, 0, len(edits))
	unknown := map[string]struct{}{}
	definitionsByID := jsonDefinitionsByID(definitions)

	for _, edit := range edits {
		id := strings.TrimSpace(edit.ID)
		definition, ok := definitionsByID[id]
		if !ok {
			if id != "" {
				unknown[id] = struct{}{}
			}
			continue
		}

		op := strings.TrimSpace(strings.ToLower(edit.Op))
		if op != "set" && op != "unset" {
			return nil, nil, invalidJSONEditOpError(agentName, edit.Op, edit.ID)
		}

		current, configured := nestedJSONValue(root, definition.Key)
		var before any
		if configured {
			before = current
		}

		switch op {
		case "set":
			value, err := editableJSONValue(definition, edit.Value, agentName)
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

	return changes, unknownJSONSettingWarnings(unknown, agentName), nil
}

func jsonDefinitionsByID(definitions []jsonSettingDefinition) map[string]jsonSettingDefinition {
	byID := make(map[string]jsonSettingDefinition, len(definitions))
	for _, definition := range definitions {
		byID[definition.ID] = definition
	}
	return byID
}

func invalidJSONEditOpError(agentName, op, id string) error {
	return &jsonEditOpError{agentName: agentName, op: op, id: id}
}

type jsonEditOpError struct {
	agentName string
	op        string
	id        string
}

func (e *jsonEditOpError) Error() string {
	return fmt.Sprintf("invalid %s privacy change op %q for %q", e.agentName, e.op, e.id)
}

func editableJSONValue(definition jsonSettingDefinition, value any, agentName string) (any, error) {
	switch jsonValueType(definition.Desired) {
	case "bool":
		typed, ok := value.(bool)
		if !ok {
			return nil, jsonValueTypeError(agentName, definition.ID, "bool")
		}
		return typed, nil
	case "string":
		typed, ok := value.(string)
		if !ok {
			return nil, jsonValueTypeError(agentName, definition.ID, "string")
		}
		return typed, nil
	case "number":
		typed, ok := editableJSONNumber(value)
		if !ok {
			return nil, jsonValueTypeError(agentName, definition.ID, "number")
		}
		return typed, nil
	case "stringArray":
		typed, ok := value.([]any)
		if !ok {
			return nil, jsonValueTypeError(agentName, definition.ID, "stringArray")
		}
		result := make([]any, 0, len(typed))
		for _, item := range typed {
			text, ok := item.(string)
			if !ok {
				return nil, jsonValueTypeError(agentName, definition.ID, "stringArray")
			}
			result = append(result, text)
		}
		return result, nil
	default:
		return nil, jsonUnsupportedEditableValueError(agentName, definition.ID)
	}
}

func jsonValueTypeError(agentName, id, valueType string) error {
	return &jsonEditValueError{agentName: agentName, id: id, valueType: valueType}
}

type jsonEditValueError struct {
	agentName string
	id        string
	valueType string
}

func (e *jsonEditValueError) Error() string {
	return fmt.Sprintf("%s privacy setting %q requires a %s value", e.agentName, e.id, e.valueType)
}

func jsonUnsupportedEditableValueError(agentName, id string) error {
	return &jsonUnsupportedEditableValue{agentName: agentName, id: id}
}

type jsonUnsupportedEditableValue struct {
	agentName string
	id        string
}

func (e *jsonUnsupportedEditableValue) Error() string {
	return fmt.Sprintf("%s privacy setting %q does not support editable values", e.agentName, e.id)
}
