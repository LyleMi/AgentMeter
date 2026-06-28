package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"runtime"
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
	result, err := a.indexer.IndexPaths(a.ctx, settings.SourcePaths, rebuild)
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
		return normalizeSourcePaths(paths), true, nil
	}
	return nil, true, nil
}

func setAutoDefaultSourcePaths(ctx context.Context, conn *sql.DB, paths []string) error {
	encoded, err := json.Marshal(normalizeSourcePaths(paths))
	if err != nil {
		return err
	}
	return db.SetConfig(ctx, conn, sourceEntriesAutoDefaultsConfigKey, string(encoded))
}

func mergeAutoDefaultSourcePaths(sourcePaths, defaultSourcePaths, autoDefaults, candidates []string) ([]string, []string, bool) {
	merged := normalizeSourcePaths(sourcePaths)
	defaults := normalizeSourcePaths(defaultSourcePaths)
	nextAutoDefaults := normalizeSourcePaths(autoDefaults)
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
	for _, path := range normalizeSourcePaths(sourcePaths) {
		if containsSourcePath(candidates, path) {
			result = appendSourcePath(result, path)
		}
	}
	return result
}

func hasSourcePathOverlap(left, right []string) bool {
	for _, path := range normalizeSourcePaths(left) {
		if containsSourcePath(right, path) {
			return true
		}
	}
	return false
}

func containsSourcePath(paths []string, path string) bool {
	key := comparableSourcePathKey(path)
	for _, candidate := range normalizeSourcePaths(paths) {
		if comparableSourcePathKey(candidate) == key {
			return true
		}
	}
	return false
}

func appendSourcePath(paths []string, path string) []string {
	return normalizeSourcePaths(append(paths, path))
}

func sourceEntriesFromPaths(paths []string, enabled bool) []model.SourceEntry {
	normalized := normalizeSourcePaths(paths)
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
		cleaned := strings.TrimSpace(entry.Path)
		if cleaned == "" {
			continue
		}
		cleaned = filepath.Clean(cleaned)
		key := sourcePathKey(cleaned)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, model.SourceEntry{Path: cleaned, Enabled: entry.Enabled})
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
	return normalizeSourcePaths(paths)
}

func mergeSourceEntriesForPaths(entries []model.SourceEntry, paths []string) []model.SourceEntry {
	merged := normalizeSourceEntries(entries)
	for _, path := range normalizeSourcePaths(paths) {
		if !containsSourcePath(sourceEntryPaths(merged), path) {
			merged = append(merged, model.SourceEntry{Path: path, Enabled: true})
		}
	}
	return normalizeSourceEntries(merged)
}

func sameSourcePaths(left, right []string) bool {
	left = normalizeSourcePaths(left)
	right = normalizeSourcePaths(right)
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if sourcePathKey(left[index]) != sourcePathKey(right[index]) {
			return false
		}
	}
	return true
}

func parseSourcePathList(value string) []string {
	var raw []string
	for _, line := range strings.FieldsFunc(value, func(r rune) bool {
		return r == '\n' || r == '\r' || r == ';'
	}) {
		raw = append(raw, line)
	}
	return normalizeSourcePaths(raw)
}

func normalizeSourcePaths(paths []string) []string {
	seen := map[string]struct{}{}
	var result []string
	for _, path := range paths {
		cleaned := strings.TrimSpace(path)
		if cleaned == "" {
			continue
		}
		cleaned = filepath.Clean(cleaned)
		key := sourcePathKey(cleaned)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, cleaned)
	}
	return result
}

func sourcePathKey(path string) string {
	if runtime.GOOS == "windows" {
		return strings.ToLower(path)
	}
	return path
}

func comparableSourcePathKey(path string) string {
	cleaned := filepath.Clean(strings.TrimSpace(path))
	if cleaned == "." || cleaned == "" {
		return sourcePathKey(cleaned)
	}
	spec := agent.ResolveSource(cleaned)
	if spec.Kind != "jsonl" && spec.RootPath != "" {
		return sourcePathKey(filepath.Clean(spec.RootPath))
	}
	return sourcePathKey(cleaned)
}

func (a *App) GetOverview() (model.Overview, error) {
	if err := a.ensureReady(); err != nil {
		return model.Overview{}, err
	}
	return a.query.Overview(a.ctx)
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
	if err := a.ensureReady(); err != nil {
		return model.AuditSummary{}, err
	}
	return a.query.AuditSummary(a.ctx)
}

func (a *App) ListAuditFindings(filters model.AuditFindingFilters) ([]model.AuditFinding, error) {
	if err := a.ensureReady(); err != nil {
		return nil, err
	}
	return a.query.AuditFindings(a.ctx, filters)
}

func (a *App) GetPricingModels() ([]model.PricingModel, error) {
	if err := a.ensureReady(); err != nil {
		return nil, err
	}
	return pricing.List(a.ctx, a.conn)
}

func (a *App) GetCodexPrivacyConfig() (model.PrivacyConfigStatus, error) {
	return privacy.NewCodexAdapter().Status()
}

func (a *App) ApplyCodexPrivacyConfig(settingIDs []string) (model.PrivacyConfigApplyResult, error) {
	return privacy.NewCodexAdapter().Apply(settingIDs)
}

func (a *App) ApplyCodexPrivacyConfigChanges(changes []model.PrivacyConfigEdit) (model.PrivacyConfigApplyResult, error) {
	return privacy.NewCodexAdapter().ApplyChanges(changes)
}

func (a *App) GetGeminiPrivacyConfig() (model.PrivacyConfigStatus, error) {
	return privacy.NewGeminiAdapter().Status()
}

func (a *App) ApplyGeminiPrivacyConfig(settingIDs []string) (model.PrivacyConfigApplyResult, error) {
	return privacy.NewGeminiAdapter().Apply(settingIDs)
}

func (a *App) ApplyGeminiPrivacyConfigChanges(changes []model.PrivacyConfigEdit) (model.PrivacyConfigApplyResult, error) {
	return privacy.NewGeminiAdapter().ApplyChanges(changes)
}

func (a *App) GetClaudePrivacyConfig() (model.PrivacyConfigStatus, error) {
	return privacy.NewClaudeAdapter().Status()
}

func (a *App) ApplyClaudePrivacyConfig(settingIDs []string) (model.PrivacyConfigApplyResult, error) {
	return privacy.NewClaudeAdapter().Apply(settingIDs)
}

func (a *App) ApplyClaudePrivacyConfigChanges(changes []model.PrivacyConfigEdit) (model.PrivacyConfigApplyResult, error) {
	return privacy.NewClaudeAdapter().ApplyChanges(changes)
}

func (a *App) GetCodeBuddyPrivacyConfig() (model.PrivacyConfigStatus, error) {
	return privacy.NewCodeBuddyAdapter().Status()
}

func (a *App) ApplyCodeBuddyPrivacyConfig(settingIDs []string) (model.PrivacyConfigApplyResult, error) {
	return privacy.NewCodeBuddyAdapter().Apply(settingIDs)
}

func (a *App) ApplyCodeBuddyPrivacyConfigChanges(changes []model.PrivacyConfigEdit) (model.PrivacyConfigApplyResult, error) {
	return privacy.NewCodeBuddyAdapter().ApplyChanges(changes)
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
