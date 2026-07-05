package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/LyleMi/AgentMeter/internal/agent"
	"github.com/LyleMi/AgentMeter/internal/agentresources"
	"github.com/LyleMi/AgentMeter/internal/db"
	"github.com/LyleMi/AgentMeter/internal/ingest"
	"github.com/LyleMi/AgentMeter/internal/model"
	"github.com/LyleMi/AgentMeter/internal/platform"
	"github.com/LyleMi/AgentMeter/internal/pricing"
	"github.com/LyleMi/AgentMeter/internal/privacy"
	"github.com/LyleMi/AgentMeter/internal/query"
	"github.com/LyleMi/AgentMeter/internal/sourcepath"
)

type App struct {
	ctx        context.Context
	dbPath     string
	conn       *sql.DB
	indexer    *ingest.Indexer
	query      *query.Service
	startupMu  sync.Mutex
	indexMu    sync.Mutex
	lastResult *model.IndexResult
	lastStart  *time.Time
}

const (
	sourceEntriesConfigKey             = "source_entries"
	sourceEntriesAutoDefaultsConfigKey = "source_entries_auto_defaults"
)

func New() (*App, error) {
	dbPath, err := platform.DefaultDatabasePath()
	if err != nil {
		return nil, err
	}
	return &App{dbPath: dbPath}, nil
}

func (a *App) Startup(ctx context.Context) error {
	a.startupMu.Lock()
	defer a.startupMu.Unlock()
	if a.conn != nil {
		a.ctx = ctx
		return nil
	}
	conn, err := db.Open(a.dbPath)
	if err != nil {
		return err
	}
	a.ctx = ctx
	a.conn = conn
	a.indexer = ingest.New(conn, a.dbPath)
	a.query = query.New(conn)
	if err := a.ensureSourcePathsConfig(ctx, conn); err != nil {
		return err
	}
	return nil
}

func (a *App) Shutdown(_ context.Context) {
	_ = db.Close(a.conn)
}

func (a *App) GetSettings() (model.Settings, error) {
	if err := a.ensureReady(); err != nil {
		return model.Settings{}, err
	}
	sourceEntries, err := a.configuredSourceEntries(a.ctx, a.conn)
	if err != nil {
		return model.Settings{}, err
	}
	if len(sourceEntries) == 0 {
		sourceEntries = sourcepath.SourceEntriesFromPaths(platform.DefaultAgentSourcePaths(), true)
	}
	sourcePaths := sourcepath.EnabledSourceEntryPaths(sourceEntries)
	sourcePath := strings.Join(sourcePaths, "\n")
	defaultSourcePaths := platform.DefaultAgentSourcePaths()
	models, err := pricing.List(a.ctx, a.conn)
	if err != nil {
		return model.Settings{}, err
	}
	return model.Settings{
		SourcePath:         sourcePath,
		SourcePaths:        sourcePaths,
		SourceEntries:      sourceEntries,
		DefaultSourcePath:  strings.Join(defaultSourcePaths, "\n"),
		DefaultSourcePaths: defaultSourcePaths,
		DatabasePath:       a.dbPath,
		PricingModels:      models,
		LastIndexStartedAt: a.lastStart,
		LastIndexResult:    a.lastResult,
	}, nil
}

func (a *App) GetAgentResources() (model.AgentResourceOverview, error) {
	if err := a.ensureReady(); err != nil {
		return model.AgentResourceOverview{}, err
	}
	return agentresources.Overview(a.ctx)
}

func (a *App) SetAgentSkillEnabled(request model.AgentResourceToggleRequest) (model.AgentResourceOperationResult, error) {
	if err := a.ensureReady(); err != nil {
		return model.AgentResourceOperationResult{}, err
	}
	return agentresources.SetSkillEnabled(a.ctx, request)
}

func (a *App) SetAgentMCPServerEnabled(request model.AgentResourceToggleRequest) (model.AgentResourceOperationResult, error) {
	if err := a.ensureReady(); err != nil {
		return model.AgentResourceOperationResult{}, err
	}
	return agentresources.SetMCPServerEnabled(a.ctx, request)
}

func (a *App) GetAgentMemoryDetail(agentKind, path, relativePath string) (model.AgentMemoryDetail, error) {
	if err := a.ensureReady(); err != nil {
		return model.AgentMemoryDetail{}, err
	}
	return agentresources.MemoryDetail(a.ctx, agentKind, path, relativePath)
}

func (a *App) UpdateAgentMemory(request model.AgentMemoryUpdateRequest) (model.AgentMemoryDetail, error) {
	if err := a.ensureReady(); err != nil {
		return model.AgentMemoryDetail{}, err
	}
	return agentresources.UpdateMemory(a.ctx, request)
}

func (a *App) SaveSourceSettings(sourceEntries []model.SourceEntry) (model.Settings, error) {
	if err := a.ensureReady(); err != nil {
		return model.Settings{}, err
	}
	if err := setSourceEntries(a.ctx, a.conn, sourceEntries); err != nil {
		return model.Settings{}, err
	}
	return a.GetSettings()
}

func (a *App) IndexNow(rebuild bool) (model.IndexResult, error) {
	if err := a.ensureReady(); err != nil {
		return model.IndexResult{}, err
	}
	a.indexMu.Lock()
	defer a.indexMu.Unlock()

	settings, err := a.GetSettings()
	if err != nil {
		return model.IndexResult{}, err
	}
	start := time.Now().UTC()
	a.lastStart = &start
	result, err := a.indexer.IndexEntries(a.ctx, sourcepath.EnabledSourceEntries(settings.SourceEntries), rebuild)
	if err != nil {
		return result, err
	}
	a.lastResult = &result
	encoded, _ := json.Marshal(result)
	_ = db.SetConfig(a.ctx, a.conn, "last_index_result", string(encoded))
	return result, nil
}

func (a *App) configuredSourceEntries(ctx context.Context, conn *sql.DB) ([]model.SourceEntry, error) {
	if encoded, ok, err := db.GetConfig(ctx, conn, sourceEntriesConfigKey); err != nil {
		return nil, err
	} else if ok && strings.TrimSpace(encoded) != "" {
		var entries []model.SourceEntry
		if err := json.Unmarshal([]byte(encoded), &entries); err == nil {
			return sourcepath.NormalizeSourceEntries(entries), nil
		}
	}
	return nil, nil
}

func (a *App) ensureSourcePathsConfig(ctx context.Context, conn *sql.DB) error {
	_, hasSourceEntries, err := db.GetConfig(ctx, conn, sourceEntriesConfigKey)
	if err != nil {
		return err
	}
	sourceEntries, err := a.configuredSourceEntries(ctx, conn)
	if err != nil {
		return err
	}
	defaultSourcePaths := platform.DefaultAgentSourcePaths()
	if len(sourceEntries) == 0 {
		if err := setSourceEntries(ctx, conn, sourcepath.SourceEntriesFromPaths(defaultSourcePaths, true)); err != nil {
			return err
		}
		return setAutoDefaultSourcePaths(ctx, conn, defaultSourcePaths)
	}

	autoDefaults, hasAutoDefaults, err := getAutoDefaultSourcePaths(ctx, conn)
	if err != nil {
		return err
	}
	candidates := platform.DefaultAgentSourceCandidates()
	if !hasAutoDefaults {
		autoDefaults = configuredDefaultSourcePaths(sourcepath.SourceEntryPaths(sourceEntries), candidates)
	}

	merged, nextAutoDefaults, changed := mergeAutoDefaultSourcePaths(sourcepath.SourceEntryPaths(sourceEntries), defaultSourcePaths, autoDefaults, candidates)
	if !hasSourceEntries || changed {
		if err := setSourceEntries(ctx, conn, mergeSourceEntriesForPaths(sourceEntries, merged)); err != nil {
			return err
		}
	}
	if !hasAutoDefaults || !sameSourcePaths(autoDefaults, nextAutoDefaults) {
		return setAutoDefaultSourcePaths(ctx, conn, nextAutoDefaults)
	}
	return nil
}

func setSourceEntries(ctx context.Context, conn *sql.DB, sourceEntries []model.SourceEntry) error {
	normalized := sourcepath.NormalizeSourceEntries(sourceEntries)
	encodedEntries, err := json.Marshal(normalized)
	if err != nil {
		return err
	}
	if err := db.SetConfig(ctx, conn, sourceEntriesConfigKey, string(encodedEntries)); err != nil {
		return err
	}
	return nil
}

func getAutoDefaultSourcePaths(ctx context.Context, conn *sql.DB) ([]string, bool, error) {
	encoded, ok, err := db.GetConfig(ctx, conn, sourceEntriesAutoDefaultsConfigKey)
	if err != nil || !ok {
		return nil, ok, err
	}
	var paths []string
	if err := json.Unmarshal([]byte(encoded), &paths); err == nil {
		return sourcepath.NormalizeList(paths), true, nil
	}
	return nil, true, nil
}

func setAutoDefaultSourcePaths(ctx context.Context, conn *sql.DB, paths []string) error {
	encoded, err := json.Marshal(sourcepath.NormalizeList(paths))
	if err != nil {
		return err
	}
	return db.SetConfig(ctx, conn, sourceEntriesAutoDefaultsConfigKey, string(encoded))
}

func mergeAutoDefaultSourcePaths(sourcePaths, defaultSourcePaths, autoDefaults, candidates []string) ([]string, []string, bool) {
	merged := sourcepath.NormalizeList(sourcePaths)
	defaults := sourcepath.NormalizeList(defaultSourcePaths)
	nextAutoDefaults := sourcepath.NormalizeList(autoDefaults)
	if len(merged) == 0 {
		return defaults, defaults, len(defaults) > 0
	}
	if !hasSourcePathOverlap(merged, candidates) {
		return merged, nextAutoDefaults, false
	}

	changed := false
	for _, path := range defaults {
		if containsSourcePath(merged, path) {
			nextAutoDefaults = appendSourcePath(nextAutoDefaults, path)
			continue
		}
		if containsSourcePath(nextAutoDefaults, path) {
			continue
		}
		merged = appendSourcePath(merged, path)
		nextAutoDefaults = appendSourcePath(nextAutoDefaults, path)
		changed = true
	}
	return merged, nextAutoDefaults, changed
}

func configuredDefaultSourcePaths(sourcePaths, candidates []string) []string {
	var result []string
	for _, path := range sourcepath.NormalizeList(sourcePaths) {
		if containsSourcePath(candidates, path) {
			result = appendSourcePath(result, path)
		}
	}
	return result
}

func hasSourcePathOverlap(left, right []string) bool {
	for _, path := range sourcepath.NormalizeList(left) {
		if containsSourcePath(right, path) {
			return true
		}
	}
	return false
}

func containsSourcePath(paths []string, path string) bool {
	key := comparableSourcePathKey(path)
	for _, candidate := range sourcepath.NormalizeList(paths) {
		if comparableSourcePathKey(candidate) == key {
			return true
		}
	}
	return false
}

func appendSourcePath(paths []string, path string) []string {
	return sourcepath.NormalizeList(append(paths, path))
}

func mergeSourceEntriesForPaths(entries []model.SourceEntry, paths []string) []model.SourceEntry {
	merged := sourcepath.NormalizeSourceEntries(entries)
	for _, path := range sourcepath.NormalizeList(paths) {
		if !containsSourcePath(sourcepath.SourceEntryPaths(merged), path) {
			merged = append(merged, model.SourceEntry{Path: path, Enabled: true})
		}
	}
	return sourcepath.NormalizeSourceEntries(merged)
}

func sameSourcePaths(left, right []string) bool {
	left = sourcepath.NormalizeList(left)
	right = sourcepath.NormalizeList(right)
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if sourcepath.Key(left[index]) != sourcepath.Key(right[index]) {
			return false
		}
	}
	return true
}

func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func parseSourcePathList(value string) []string {
	var raw []string
	for _, line := range strings.FieldsFunc(value, func(r rune) bool {
		return r == '\n' || r == '\r' || r == ';'
	}) {
		raw = append(raw, line)
	}
	return sourcepath.NormalizeList(raw)
}

func comparableSourcePathKey(path string) string {
	cleaned := sourcepath.Normalize(path)
	if cleaned == "." || cleaned == "" {
		return sourcepath.Key(cleaned)
	}
	spec := agent.ResolveSource(cleaned)
	if spec.Kind != "jsonl" && spec.RootPath != "" {
		return sourcepath.Key(sourcepath.Normalize(spec.RootPath))
	}
	return sourcepath.Key(cleaned)
}

func (a *App) GetOverview() (model.Overview, error) {
	return a.GetOverviewWithFilters(model.AnalyticsFilters{})
}

func (a *App) GetOverviewWithFilters(filters model.AnalyticsFilters) (model.Overview, error) {
	if err := a.ensureReady(); err != nil {
		return model.Overview{}, err
	}
	return a.query.OverviewWithFilters(a.ctx, filters)
}

func (a *App) GetTokenAnalytics() (model.TokenAnalytics, error) {
	return a.GetTokenAnalyticsWithFilters(model.AnalyticsFilters{})
}

func (a *App) GetTokenAnalyticsWithFilters(filters model.AnalyticsFilters) (model.TokenAnalytics, error) {
	if err := a.ensureReady(); err != nil {
		return model.TokenAnalytics{}, err
	}
	return a.query.TokenAnalyticsWithFilters(a.ctx, filters)
}

func (a *App) GetModelSignalsWithFilters(filters model.AnalyticsFilters) (model.ModelSignals, error) {
	if err := a.ensureReady(); err != nil {
		return model.ModelSignals{}, err
	}
	return a.query.ModelSignalsWithFilters(a.ctx, filters)
}

func (a *App) GetUsageBreakdown(groupBy string, filters model.AnalyticsFilters) (model.UsageBreakdown, error) {
	if err := a.ensureReady(); err != nil {
		return model.UsageBreakdown{}, err
	}
	return a.query.UsageBreakdown(a.ctx, groupBy, filters)
}

func (a *App) ListSessions(filters model.SessionFilters) ([]model.Session, error) {
	if err := a.ensureReady(); err != nil {
		return nil, err
	}
	return a.query.Sessions(a.ctx, filters)
}

func (a *App) GetSessionDetail(id int64) (model.SessionDetail, error) {
	if err := a.ensureReady(); err != nil {
		return model.SessionDetail{}, err
	}
	return a.query.SessionDetail(a.ctx, id)
}

func (a *App) GetTools() ([]model.ToolStat, error) {
	return a.ListTools(model.ToolFilters{})
}

func (a *App) ListTools(filters model.ToolFilters) ([]model.ToolStat, error) {
	if err := a.ensureReady(); err != nil {
		return nil, err
	}
	return a.query.Tools(a.ctx, filters)
}

func (a *App) ListToolCalls(filters model.ToolCallFilters) ([]model.ToolCall, error) {
	if err := a.ensureReady(); err != nil {
		return nil, err
	}
	return a.query.ToolCalls(a.ctx, filters)
}

func (a *App) PromptSuggestions(filters model.PromptSuggestionFilters) ([]model.PromptSuggestion, error) {
	if err := a.ensureReady(); err != nil {
		return nil, err
	}
	return a.query.PromptSuggestions(a.ctx, filters)
}

func (a *App) SavedPrompts() ([]model.SavedPrompt, error) {
	if err := a.ensureReady(); err != nil {
		return nil, err
	}
	return a.query.SavedPrompts(a.ctx)
}

func (a *App) SavePrompt(input model.SavedPromptInput) (model.SavedPrompt, error) {
	if err := a.ensureReady(); err != nil {
		return model.SavedPrompt{}, err
	}
	return a.query.SavePrompt(a.ctx, input)
}

func (a *App) UpdateSavedPrompt(id int64, input model.SavedPromptInput) (model.SavedPrompt, error) {
	if err := a.ensureReady(); err != nil {
		return model.SavedPrompt{}, err
	}
	return a.query.UpdateSavedPrompt(a.ctx, id, input)
}

func (a *App) DeleteSavedPrompt(id int64) error {
	if err := a.ensureReady(); err != nil {
		return err
	}
	return a.query.DeleteSavedPrompt(a.ctx, id)
}

func (a *App) RecordPromptCopy(id int64) (model.SavedPrompt, error) {
	if err := a.ensureReady(); err != nil {
		return model.SavedPrompt{}, err
	}
	return a.query.RecordPromptCopy(a.ctx, id)
}

func (a *App) IgnorePromptSuggestion(key string) error {
	if err := a.ensureReady(); err != nil {
		return err
	}
	return a.query.IgnorePromptSuggestion(a.ctx, key)
}

func (a *App) UnignorePromptSuggestion(key string) error {
	if err := a.ensureReady(); err != nil {
		return err
	}
	return a.query.UnignorePromptSuggestion(a.ctx, key)
}

func (a *App) GetAuditSummary() (model.AuditSummary, error) {
	return a.GetAuditSummaryWithFilters(model.AuditFindingFilters{})
}

func (a *App) GetAuditSummaryWithFilters(filters model.AuditFindingFilters) (model.AuditSummary, error) {
	if err := a.ensureReady(); err != nil {
		return model.AuditSummary{}, err
	}
	return a.query.AuditSummaryWithFilters(a.ctx, filters)
}

func (a *App) ListAuditFindings(filters model.AuditFindingFilters) ([]model.AuditFinding, error) {
	if err := a.ensureReady(); err != nil {
		return nil, err
	}
	return a.query.AuditFindings(a.ctx, filters)
}

func (a *App) GetAuditFinding(id int64) (model.AuditFinding, error) {
	if err := a.ensureReady(); err != nil {
		return model.AuditFinding{}, err
	}
	return a.query.AuditFinding(a.ctx, id)
}

func (a *App) GetPricingModels() ([]model.PricingModel, error) {
	if err := a.ensureReady(); err != nil {
		return nil, err
	}
	return pricing.List(a.ctx, a.conn)
}

func (a *App) SavePricingModel(input model.PricingModelInput) (model.PricingModel, error) {
	if err := a.ensureReady(); err != nil {
		return model.PricingModel{}, err
	}
	return pricing.UpsertCustom(a.ctx, a.conn, input)
}

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

func (a *App) ensureReady() error {
	if a.conn == nil {
		if a.ctx == nil {
			a.ctx = context.Background()
		}
		return a.Startup(a.ctx)
	}
	if a.query == nil || a.indexer == nil {
		return errors.New("app services are not initialized")
	}
	return nil
}
