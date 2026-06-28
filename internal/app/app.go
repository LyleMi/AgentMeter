package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"AgentMeter/internal/agent"
	"AgentMeter/internal/db"
	"AgentMeter/internal/ingest"
	"AgentMeter/internal/model"
	"AgentMeter/internal/platform"
	"AgentMeter/internal/pricing"
	"AgentMeter/internal/privacy"
	"AgentMeter/internal/query"
	"AgentMeter/internal/sourcepath"
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
		sourceEntries = sourceEntriesFromPaths(platform.DefaultAgentSourcePaths(), true)
	}
	sourcePaths := enabledSourceEntryPaths(sourceEntries)
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
	result, err := a.indexer.IndexEntries(a.ctx, enabledSourceEntries(settings.SourceEntries), rebuild)
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
			return normalizeSourceEntries(entries), nil
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
		if err := setSourceEntries(ctx, conn, sourceEntriesFromPaths(defaultSourcePaths, true)); err != nil {
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
		autoDefaults = configuredDefaultSourcePaths(sourceEntryPaths(sourceEntries), candidates)
	}

	merged, nextAutoDefaults, changed := mergeAutoDefaultSourcePaths(sourceEntryPaths(sourceEntries), defaultSourcePaths, autoDefaults, candidates)
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
	normalized := normalizeSourceEntries(sourceEntries)
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

func sourceEntriesFromPaths(paths []string, enabled bool) []model.SourceEntry {
	normalized := sourcepath.NormalizeList(paths)
	entries := make([]model.SourceEntry, 0, len(normalized))
	for _, path := range normalized {
		entries = append(entries, model.SourceEntry{Path: path, Enabled: enabled})
	}
	return entries
}

func normalizeSourceEntries(entries []model.SourceEntry) []model.SourceEntry {
	seen := map[string]struct{}{}
	result := make([]model.SourceEntry, 0, len(entries))
	for _, entry := range entries {
		cleaned := sourcepath.Normalize(entry.Path)
		if cleaned == "" {
			continue
		}
		key := sourcepath.Key(cleaned)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, model.SourceEntry{Path: cleaned, Enabled: entry.Enabled, Label: strings.TrimSpace(entry.Label)})
	}
	return result
}

func sourceEntryPaths(entries []model.SourceEntry) []string {
	paths := make([]string, 0, len(entries))
	for _, entry := range normalizeSourceEntries(entries) {
		paths = append(paths, entry.Path)
	}
	return paths
}

func enabledSourceEntryPaths(entries []model.SourceEntry) []string {
	var paths []string
	for _, entry := range normalizeSourceEntries(entries) {
		if entry.Enabled {
			paths = append(paths, entry.Path)
		}
	}
	return sourcepath.NormalizeList(paths)
}

func enabledSourceEntries(entries []model.SourceEntry) []model.SourceEntry {
	var result []model.SourceEntry
	for _, entry := range normalizeSourceEntries(entries) {
		if entry.Enabled {
			result = append(result, entry)
		}
	}
	return result
}

func mergeSourceEntriesForPaths(entries []model.SourceEntry, paths []string) []model.SourceEntry {
	merged := normalizeSourceEntries(entries)
	for _, path := range sourcepath.NormalizeList(paths) {
		if !containsSourcePath(sourceEntryPaths(merged), path) {
			merged = append(merged, model.SourceEntry{Path: path, Enabled: true})
		}
	}
	return normalizeSourceEntries(merged)
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
	if err := a.ensureReady(); err != nil {
		return model.Overview{}, err
	}
	return a.query.Overview(a.ctx)
}

func (a *App) GetTokenAnalytics() (model.TokenAnalytics, error) {
	if err := a.ensureReady(); err != nil {
		return model.TokenAnalytics{}, err
	}
	return a.query.TokenAnalytics(a.ctx)
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
		statuses[index] = a.addPrivacySourceWarning(statuses[index])
	}
	return statuses, nil
}

func (a *App) GetPrivacyConfig(target string) (model.PrivacyConfigStatus, error) {
	status, err := privacy.DefaultRegistry().Status(target)
	if err != nil {
		return status, err
	}
	return a.addPrivacySourceWarning(status), nil
}

func (a *App) ApplyPrivacyConfig(target string, settingIDs []string) (model.PrivacyConfigApplyResult, error) {
	result, err := privacy.DefaultRegistry().Apply(target, settingIDs)
	if err != nil {
		return result, err
	}
	return a.addPrivacySourceWarningToResult(result), nil
}

func (a *App) ApplyPrivacyConfigChanges(target string, changes []model.PrivacyConfigEdit) (model.PrivacyConfigApplyResult, error) {
	result, err := privacy.DefaultRegistry().ApplyChanges(target, changes)
	if err != nil {
		return result, err
	}
	return a.addPrivacySourceWarningToResult(result), nil
}

func (a *App) ApplyPrivacyProfile(target, profile string) (model.PrivacyConfigApplyResult, error) {
	result, err := privacy.DefaultRegistry().ApplyProfile(target, profile)
	if err != nil {
		return result, err
	}
	return a.addPrivacySourceWarningToResult(result), nil
}

func (a *App) addPrivacySourceWarningToResult(result model.PrivacyConfigApplyResult) model.PrivacyConfigApplyResult {
	result.Status = a.addPrivacySourceWarning(result.Status)
	for _, warning := range result.Status.Warnings {
		if !containsString(result.Warnings, warning) {
			result.Warnings = append(result.Warnings, warning)
		}
	}
	return result
}

func (a *App) addPrivacySourceWarning(status model.PrivacyConfigStatus) model.PrivacyConfigStatus {
	if a.conn == nil {
		return status
	}
	kind := strings.TrimSpace(status.Target)
	if kind == "" {
		return status
	}
	var count int
	if err := a.conn.QueryRowContext(a.ctx, `SELECT COUNT(*) FROM sources WHERE kind = ?`, kind).Scan(&count); err != nil || count <= 1 {
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
