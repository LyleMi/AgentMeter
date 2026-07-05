package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/LyleMi/AgentMeter/internal/model"
	"github.com/LyleMi/AgentMeter/internal/privacy"
	"github.com/LyleMi/AgentMeter/internal/sourcepath"
)

func (a *App) PrivacyTargets() []string {
	return privacy.DefaultRegistry().Targets()
}

func (a *App) SupportsPrivacyTarget(target string) bool {
	return privacy.DefaultRegistry().Supports(target)
}

func (a *App) GetPrivacyConfigs() ([]model.PrivacyConfigStatus, error) {
	statuses, err := privacy.DefaultRegistry().Statuses()
	if err != nil {
		return nil, err
	}
	for index := range statuses {
		statuses[index] = a.addPrivacySourceMetadata(statuses[index], "")
	}
	return statuses, nil
}

func (a *App) GetPrivacyConfig(target string) (model.PrivacyConfigStatus, error) {
	return a.GetPrivacyConfigForSource(target, "")
}

func (a *App) GetPrivacyConfigForSource(target, sourceKey string) (model.PrivacyConfigStatus, error) {
	adapter, err := a.privacyAdapterForSource(target, sourceKey)
	if err != nil {
		return model.PrivacyConfigStatus{}, err
	}
	status, err := adapter.Status()
	if err != nil {
		return status, err
	}
	return a.addPrivacySourceMetadata(status, sourceKey), nil
}

func (a *App) ApplyPrivacyConfig(target string, settingIDs []string) (model.PrivacyConfigApplyResult, error) {
	return a.ApplyPrivacyConfigForSource(target, "", settingIDs)
}

func (a *App) ApplyPrivacyConfigForSource(target, sourceKey string, settingIDs []string) (model.PrivacyConfigApplyResult, error) {
	adapter, err := a.privacyAdapterForSource(target, sourceKey)
	if err != nil {
		return model.PrivacyConfigApplyResult{}, err
	}
	result, err := adapter.Apply(settingIDs)
	if err != nil {
		return result, err
	}
	return a.addPrivacySourceMetadataToResult(result, sourceKey), nil
}

func (a *App) ApplyPrivacyConfigChanges(target string, changes []model.PrivacyConfigEdit) (model.PrivacyConfigApplyResult, error) {
	return a.ApplyPrivacyConfigChangesForSource(target, "", changes)
}

func (a *App) ApplyPrivacyConfigChangesForSource(target, sourceKey string, changes []model.PrivacyConfigEdit) (model.PrivacyConfigApplyResult, error) {
	adapter, err := a.privacyAdapterForSource(target, sourceKey)
	if err != nil {
		return model.PrivacyConfigApplyResult{}, err
	}
	result, err := adapter.ApplyChanges(changes)
	if err != nil {
		return result, err
	}
	return a.addPrivacySourceMetadataToResult(result, sourceKey), nil
}

func (a *App) ApplyPrivacyProfile(target, profile string) (model.PrivacyConfigApplyResult, error) {
	return a.ApplyPrivacyProfileForSource(target, "", profile)
}

func (a *App) ApplyPrivacyProfileForSource(target, sourceKey, profile string) (model.PrivacyConfigApplyResult, error) {
	adapter, err := a.privacyAdapterForSource(target, sourceKey)
	if err != nil {
		return model.PrivacyConfigApplyResult{}, err
	}
	result, err := adapter.ApplyProfile(profile)
	if err != nil {
		return result, err
	}
	return a.addPrivacySourceMetadataToResult(result, sourceKey), nil
}

func (a *App) addPrivacySourceMetadataToResult(result model.PrivacyConfigApplyResult, selectedSourceKey string) model.PrivacyConfigApplyResult {
	result.Status = a.addPrivacySourceMetadata(result.Status, selectedSourceKey)
	for _, warning := range result.Status.Warnings {
		if !containsString(result.Warnings, warning) {
			result.Warnings = append(result.Warnings, warning)
		}
	}
	return result
}

func (a *App) addPrivacySourceMetadata(status model.PrivacyConfigStatus, selectedSourceKey string) model.PrivacyConfigStatus {
	if a.conn == nil {
		return status
	}
	kind := strings.TrimSpace(status.Target)
	if kind == "" {
		return status
	}
	options, err := a.privacySourceOptions(kind)
	if err != nil || len(options) == 0 {
		return status
	}
	status.SourceOptions = options
	status.SelectedSourceKey = matchingPrivacySourceKey(options, selectedSourceKey, status.ConfigPath)
	if privacySupportsSourceWrites(kind) {
		return status
	}
	if len(options) <= 1 {
		return status
	}
	label := privacyTargetLabel(kind)
	path := strings.TrimSpace(status.ConfigPath)
	if path == "" {
		path = "the configured target path"
	}
	warning := "Multiple " + label + "-like sources are indexed. This privacy target manages only " + path + ". Source-specific privacy writes are not enabled yet."
	if !containsString(status.Warnings, warning) {
		status.Warnings = append(status.Warnings, warning)
	}
	return status
}

func (a *App) privacyAdapterForSource(target, sourceKey string) (privacy.Adapter, error) {
	normalized := strings.ToLower(strings.TrimSpace(target))
	cleanSourceKey := strings.TrimSpace(sourceKey)
	if cleanSourceKey == "" {
		return privacy.DefaultRegistry().Adapter(normalized)
	}
	if !privacySupportsSourceWrites(normalized) {
		return nil, privacySourceUnsupportedError{Target: normalized}
	}
	source, err := a.privacySourceByKey(normalized, cleanSourceKey)
	if err != nil {
		return nil, err
	}
	switch normalized {
	case "codex":
		return privacy.NewCodexAdapterForConfigPath(codexPrivacyConfigPathForRoot(source.RootPath)), nil
	case "gemini":
		adapter := privacy.NewGeminiAdapter()
		adapter.ConfigPath = settingsJSONPrivacyConfigPathForRoot(source.RootPath)
		return adapter, nil
	case "claude":
		adapter := privacy.NewClaudeAdapter()
		adapter.ConfigPath = settingsJSONPrivacyConfigPathForRoot(source.RootPath)
		return adapter, nil
	case "codebuddy":
		adapter := privacy.NewCodeBuddyAdapter()
		adapter.ConfigPath = settingsJSONPrivacyConfigPathForRoot(source.RootPath)
		return adapter, nil
	default:
		return nil, privacySourceUnsupportedError{Target: normalized}
	}
}

func privacySupportsSourceWrites(target string) bool {
	switch strings.ToLower(strings.TrimSpace(target)) {
	case "codex", "gemini", "claude", "codebuddy":
		return true
	default:
		return false
	}
}

type privacySourceUnsupportedError struct {
	Target string
}

func (e privacySourceUnsupportedError) Error() string {
	return fmt.Sprintf("source-specific privacy writes are not supported for %s", e.Target)
}

type privacySourceNotFoundError struct {
	Target    string
	SourceKey string
}

func (e privacySourceNotFoundError) Error() string {
	return fmt.Sprintf("privacy source %q was not found for %s", e.SourceKey, e.Target)
}

type privacySourceKeyError struct {
	SourceKey string
}

func (e privacySourceKeyError) Error() string {
	return fmt.Sprintf("invalid privacy source key %q", e.SourceKey)
}

func (a *App) privacySourceOptions(kind string) ([]model.PrivacyConfigSourceOption, error) {
	rows, err := a.conn.QueryContext(appContext(a.ctx), `SELECT id, kind, name, root_path, sessions_path, platform, created_at, updated_at FROM sources WHERE kind = ? ORDER BY name, root_path, id`, strings.TrimSpace(kind))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var options []model.PrivacyConfigSourceOption
	for rows.Next() {
		var source model.Source
		var created, updated string
		if err := rows.Scan(&source.ID, &source.Kind, &source.Name, &source.RootPath, &source.SessionsPath, &source.Platform, &created, &updated); err != nil {
			return nil, err
		}
		label := strings.TrimSpace(source.Name)
		if label == "" {
			label = source.Kind
		}
		options = append(options, model.PrivacyConfigSourceOption{
			SourceID:   source.ID,
			SourceKey:  privacySourceKey(source.ID),
			Label:      label,
			RootPath:   source.RootPath,
			ConfigPath: privacyConfigPathForSource(source),
		})
	}
	return options, rows.Err()
}

func (a *App) privacySourceByKey(kind, sourceKey string) (model.Source, error) {
	if a.conn == nil {
		return model.Source{}, errors.New("app database is not ready")
	}
	sourceID, err := privacySourceID(sourceKey)
	if err != nil {
		return model.Source{}, err
	}
	var source model.Source
	var created, updated string
	row := a.conn.QueryRowContext(appContext(a.ctx), `SELECT id, kind, name, root_path, sessions_path, platform, created_at, updated_at FROM sources WHERE id = ? AND kind = ?`, sourceID, strings.TrimSpace(kind))
	if err := row.Scan(&source.ID, &source.Kind, &source.Name, &source.RootPath, &source.SessionsPath, &source.Platform, &created, &updated); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Source{}, privacySourceNotFoundError{Target: kind, SourceKey: sourceKey}
		}
		return model.Source{}, err
	}
	return source, nil
}

func privacySourceID(sourceKey string) (int64, error) {
	value := strings.TrimSpace(sourceKey)
	idText, ok := strings.CutPrefix(value, "source:")
	if !ok {
		return 0, privacySourceKeyError{SourceKey: sourceKey}
	}
	id, err := strconv.ParseInt(strings.TrimSpace(idText), 10, 64)
	if err != nil || id <= 0 {
		return 0, privacySourceKeyError{SourceKey: sourceKey}
	}
	return id, nil
}

func privacySourceKey(sourceID int64) string {
	if sourceID <= 0 {
		return ""
	}
	return "source:" + strconv.FormatInt(sourceID, 10)
}

func matchingPrivacySourceKey(options []model.PrivacyConfigSourceOption, selectedSourceKey, configPath string) string {
	selectedSourceKey = strings.TrimSpace(selectedSourceKey)
	if selectedSourceKey != "" {
		for _, option := range options {
			if option.SourceKey == selectedSourceKey {
				return selectedSourceKey
			}
		}
	}
	for _, option := range options {
		if sourcepath.Equal(option.ConfigPath, configPath) {
			return option.SourceKey
		}
	}
	return ""
}

func privacyConfigPathForSource(source model.Source) string {
	switch strings.ToLower(strings.TrimSpace(source.Kind)) {
	case "codex":
		return codexPrivacyConfigPathForRoot(source.RootPath)
	case "gemini", "claude", "codebuddy":
		return settingsJSONPrivacyConfigPathForRoot(source.RootPath)
	default:
		return ""
	}
}

func codexPrivacyConfigPathForRoot(root string) string {
	if strings.TrimSpace(root) == "" {
		return ""
	}
	return filepath.Join(filepath.Clean(root), "config.toml")
}

func settingsJSONPrivacyConfigPathForRoot(root string) string {
	if strings.TrimSpace(root) == "" {
		return ""
	}
	return filepath.Join(filepath.Clean(root), "settings.json")
}

func appContext(ctx context.Context) context.Context {
	if ctx != nil {
		return ctx
	}
	return context.Background()
}

func privacyTargetLabel(target string) string {
	switch strings.ToLower(strings.TrimSpace(target)) {
	case "codex":
		return "Codex"
	case "claude":
		return "Claude"
	case "codebuddy":
		return "CodeBuddy"
	case "gemini":
		return "Gemini"
	default:
		return target
	}
}

func (a *App) GetCodexPrivacyConfig() (model.PrivacyConfigStatus, error) {
	return a.GetPrivacyConfig("codex")
}

func (a *App) ApplyCodexPrivacyConfig(settingIDs []string) (model.PrivacyConfigApplyResult, error) {
	return a.ApplyPrivacyConfig("codex", settingIDs)
}

func (a *App) ApplyCodexPrivacyConfigChanges(changes []model.PrivacyConfigEdit) (model.PrivacyConfigApplyResult, error) {
	return a.ApplyPrivacyConfigChanges("codex", changes)
}

func (a *App) ApplyCodexPrivacyProfile(profile string) (model.PrivacyConfigApplyResult, error) {
	return a.ApplyPrivacyProfile("codex", profile)
}

func (a *App) GetGeminiPrivacyConfig() (model.PrivacyConfigStatus, error) {
	return a.GetPrivacyConfig("gemini")
}

func (a *App) ApplyGeminiPrivacyConfig(settingIDs []string) (model.PrivacyConfigApplyResult, error) {
	return a.ApplyPrivacyConfig("gemini", settingIDs)
}

func (a *App) ApplyGeminiPrivacyConfigChanges(changes []model.PrivacyConfigEdit) (model.PrivacyConfigApplyResult, error) {
	return a.ApplyPrivacyConfigChanges("gemini", changes)
}

func (a *App) ApplyGeminiPrivacyProfile(profile string) (model.PrivacyConfigApplyResult, error) {
	return a.ApplyPrivacyProfile("gemini", profile)
}

func (a *App) GetClaudePrivacyConfig() (model.PrivacyConfigStatus, error) {
	return a.GetPrivacyConfig("claude")
}

func (a *App) ApplyClaudePrivacyConfig(settingIDs []string) (model.PrivacyConfigApplyResult, error) {
	return a.ApplyPrivacyConfig("claude", settingIDs)
}

func (a *App) ApplyClaudePrivacyConfigChanges(changes []model.PrivacyConfigEdit) (model.PrivacyConfigApplyResult, error) {
	return a.ApplyPrivacyConfigChanges("claude", changes)
}

func (a *App) ApplyClaudePrivacyProfile(profile string) (model.PrivacyConfigApplyResult, error) {
	return a.ApplyPrivacyProfile("claude", profile)
}

func (a *App) GetCodeBuddyPrivacyConfig() (model.PrivacyConfigStatus, error) {
	return a.GetPrivacyConfig("codebuddy")
}

func (a *App) ApplyCodeBuddyPrivacyConfig(settingIDs []string) (model.PrivacyConfigApplyResult, error) {
	return a.ApplyPrivacyConfig("codebuddy", settingIDs)
}

func (a *App) ApplyCodeBuddyPrivacyConfigChanges(changes []model.PrivacyConfigEdit) (model.PrivacyConfigApplyResult, error) {
	return a.ApplyPrivacyConfigChanges("codebuddy", changes)
}

func (a *App) ApplyCodeBuddyPrivacyProfile(profile string) (model.PrivacyConfigApplyResult, error) {
	return a.ApplyPrivacyProfile("codebuddy", profile)
}
