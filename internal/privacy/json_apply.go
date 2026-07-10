package privacy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/LyleMi/AgentMeter/internal/jsonc"
	"github.com/LyleMi/AgentMeter/internal/model"
)

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
	Recommended bool
}

type jsonPrivacyAdapter struct {
	target       string
	name         string
	agentName    string
	definitions  []jsonSettingDefinition
	settingsPath func() (string, error)
	now          func() time.Time
}

func (a jsonPrivacyAdapter) status() (model.PrivacyConfigStatus, error) {
	path, err := a.settingsPath()
	if err != nil {
		return model.PrivacyConfigStatus{}, err
	}
	content, exists, err := readOptionalFile(path)
	if err != nil {
		return model.PrivacyConfigStatus{}, err
	}
	return a.buildStatus(privacyConfigFile{path: path, exists: exists, content: content}), nil
}

func (a jsonPrivacyAdapter) apply(settingIDs []string) (model.PrivacyConfigApplyResult, error) {
	path, err := a.settingsPath()
	if err != nil {
		return model.PrivacyConfigApplyResult{}, err
	}
	return applyJSONSettingsMutation(path, a.now, a.buildStatus, func(root map[string]any) ([]model.PrivacyConfigChange, []string, error) {
		selected, warnings := selectedJSONSettingDefinitions(settingIDs, a.definitions, a.agentName)
		changes := plannedJSONChanges(root, selected)
		if len(changes) > 0 {
			applyJSONSettings(root, selected)
		}
		return changes, warnings, nil
	})
}

func (a jsonPrivacyAdapter) applyChanges(edits []model.PrivacyConfigEdit) (model.PrivacyConfigApplyResult, error) {
	path, err := a.settingsPath()
	if err != nil {
		return model.PrivacyConfigApplyResult{}, err
	}
	return applyJSONSettingsMutation(path, a.now, a.buildStatus, func(root map[string]any) ([]model.PrivacyConfigChange, []string, error) {
		return applyJSONEdits(root, edits, a.definitions, a.agentName)
	})
}

func (a jsonPrivacyAdapter) applyProfile(profile string) (model.PrivacyConfigApplyResult, error) {
	normalized, err := normalizePrivacyProfile(profile)
	if err != nil {
		return model.PrivacyConfigApplyResult{}, err
	}
	path, err := a.settingsPath()
	if err != nil {
		return model.PrivacyConfigApplyResult{}, err
	}
	return applyJSONSettingsMutation(path, a.now, a.buildStatus, func(root map[string]any) ([]model.PrivacyConfigChange, []string, error) {
		return applyJSONProfile(root, normalized, a.definitions), nil, nil
	})
}

func (a jsonPrivacyAdapter) buildStatus(file privacyConfigFile) model.PrivacyConfigStatus {
	root, canApply := parseJSONStatusFile(&file, a.agentName)
	settings := make([]model.PrivacyConfigSetting, 0, len(a.definitions))
	summary := model.PrivacyConfigSummary{Total: len(a.definitions)}
	for _, definition := range a.definitions {
		setting := buildJSONPrivacySetting(root, definition, canApply)
		addJSONStatusSummary(&summary, setting.Status)
		settings = append(settings, setting)
	}
	if summary.Total > 0 {
		summary.Score = ((summary.Hardened + summary.Implicit) * 100) / summary.Total
	}
	return model.PrivacyConfigStatus{
		Target:     a.target,
		Name:       a.name,
		ConfigPath: file.path,
		Exists:     file.exists,
		Profiles:   privacyConfigProfiles(),
		Summary:    summary,
		Settings:   settings,
		Warnings:   file.warnings,
	}
}

func parseJSONStatusFile(file *privacyConfigFile, agentName string) (map[string]any, bool) {
	root, err := parseJSONSettings(file.content)
	if err == nil {
		return root, true
	}
	file.warnings = append(file.warnings, fmt.Sprintf("%s settings.json could not be parsed: %v", agentName, err))
	return map[string]any{}, false
}

func buildJSONPrivacySetting(root map[string]any, definition jsonSettingDefinition, canApply bool) model.PrivacyConfigSetting {
	current, configured := nestedJSONValue(root, definition.Key)
	status := jsonPrivacySettingStatus(current, configured, definition, canApply)
	strict := definition.Desired
	if definition.MergeArray {
		strict = jsonSettingAfter(current, configured, definition)
	}
	var currentValue any
	if configured {
		currentValue = current
	}
	return model.PrivacyConfigSetting{
		ID:            definition.ID,
		Group:         definition.Group,
		Title:         definition.Title,
		Description:   definition.Description,
		Key:           definition.Key,
		DesiredValue:  definition.Desired,
		StrictValue:   strict,
		ProfileValues: privacyProfileValues(definition.Recommended, strict, strict),
		ValueType:     jsonValueType(definition.Desired),
		Configured:    configured,
		SupportsUnset: canApply,
		CurrentValue:  currentValue,
		Status:        status,
		Impact:        definition.Impact,
		CanApply:      canApply,
	}
}

func jsonPrivacySettingStatus(current any, configured bool, definition jsonSettingDefinition, canApply bool) string {
	if configured && jsonSettingHardened(current, true, definition) {
		return statusHardened
	}
	if !configured && definition.DefaultSafe && canApply {
		return statusImplicit
	}
	return statusAttention
}

func addJSONStatusSummary(summary *model.PrivacyConfigSummary, status string) {
	switch status {
	case statusHardened:
		summary.Hardened++
	case statusImplicit:
		summary.Implicit++
	default:
		summary.Attention++
	}
}

func jsonSettingsPath(configPath, overrideEnv, configDirEnv, homeDirName string) (string, error) {
	if strings.TrimSpace(configPath) != "" {
		return filepath.Clean(configPath), nil
	}
	if path := strings.TrimSpace(os.Getenv(overrideEnv)); path != "" {
		return filepath.Clean(path), nil
	}
	if configDirEnv != "" {
		if dir := strings.TrimSpace(os.Getenv(configDirEnv)); dir != "" {
			return filepath.Join(filepath.Clean(dir), "settings.json"), nil
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, homeDirName, "settings.json"), nil
}

func parseJSONSettings(content []byte) (map[string]any, error) {
	return jsonc.ParseObject(content)
}

func applyJSONSettingsMutation(
	path string,
	now func() time.Time,
	buildStatus func(privacyConfigFile) model.PrivacyConfigStatus,
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
		result.Status = buildStatus(privacyConfigFile{path: path, exists: exists, content: original, warnings: warnings})
		return result, nil
	}

	updated, err := marshalJSONSettings(root)
	if err != nil {
		return model.PrivacyConfigApplyResult{}, err
	}
	if bytes.Equal(updated, original) {
		result.Status = buildStatus(privacyConfigFile{path: path, exists: exists, content: original, warnings: warnings})
		return result, nil
	}

	backupPath, err := writeUpdatedConfig(path, original, updated, exists, now)
	if err != nil {
		return model.PrivacyConfigApplyResult{}, err
	}
	result.BackupPath = backupPath
	result.Status = buildStatus(privacyConfigFile{path: path, exists: true, content: updated, warnings: warnings})
	return result, nil
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

func applyJSONProfile(root map[string]any, profile string, definitions []jsonSettingDefinition) []model.PrivacyConfigChange {
	changes := make([]model.PrivacyConfigChange, 0, len(definitions))
	for _, definition := range definitions {
		current, configured := nestedJSONValue(root, definition.Key)
		var before any
		if configured {
			before = current
		}

		switch privacyProfileOperation(profile, definition.Recommended) {
		case privacyProfileOpSet:
			if jsonSettingHardened(current, configured, definition) {
				continue
			}
			after := jsonSettingAfter(current, configured, definition)
			setNestedJSONValue(root, definition.Key, after)
			changes = append(changes, model.PrivacyConfigChange{
				ID:     definition.ID,
				Key:    definition.Key,
				Before: before,
				After:  cloneJSONValue(after),
			})
		case privacyProfileOpUnset:
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
	return changes
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
		definition, ok := definitionsByID[strings.TrimSpace(edit.ID)]
		if !ok {
			if id := strings.TrimSpace(edit.ID); id != "" {
				unknown[id] = struct{}{}
			}
			continue
		}
		change, err := applyJSONEdit(root, definition, edit, agentName)
		if err != nil {
			return nil, nil, err
		}
		if change != nil {
			changes = append(changes, *change)
		}
	}

	return changes, unknownJSONSettingWarnings(unknown, agentName), nil
}

func applyJSONEdit(root map[string]any, definition jsonSettingDefinition, edit model.PrivacyConfigEdit, agentName string) (*model.PrivacyConfigChange, error) {
	current, configured := nestedJSONValue(root, definition.Key)
	context := jsonEditContext{
		root:       root,
		definition: definition,
		current:    current,
		configured: configured,
		agentName:  agentName,
	}
	switch strings.TrimSpace(strings.ToLower(edit.Op)) {
	case privacyProfileOpSet:
		return context.set(edit.Value)
	case privacyProfileOpUnset:
		return context.unset(), nil
	default:
		return nil, invalidJSONEditOpError(agentName, edit.Op, edit.ID)
	}
}

type jsonEditContext struct {
	root       map[string]any
	definition jsonSettingDefinition
	current    any
	configured bool
	agentName  string
}

func (c jsonEditContext) set(rawValue any) (*model.PrivacyConfigChange, error) {
	value, err := editableJSONValue(c.definition, rawValue, c.agentName)
	if err != nil {
		return nil, err
	}
	if c.configured && jsonValuesEqual(c.current, value) {
		return nil, nil
	}
	setNestedJSONValue(c.root, c.definition.Key, value)
	change := jsonPrivacyChange(c.definition, configuredJSONValue(c.current, c.configured), cloneJSONValue(value))
	return &change, nil
}

func (c jsonEditContext) unset() *model.PrivacyConfigChange {
	if !c.configured {
		return nil
	}
	unsetNestedJSONValue(c.root, c.definition.Key)
	change := jsonPrivacyChange(c.definition, c.current, nil)
	return &change
}

func configuredJSONValue(value any, configured bool) any {
	if !configured {
		return nil
	}
	return value
}

func jsonPrivacyChange(definition jsonSettingDefinition, before, after any) model.PrivacyConfigChange {
	return model.PrivacyConfigChange{
		ID:     definition.ID,
		Key:    definition.Key,
		Before: before,
		After:  after,
	}
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
