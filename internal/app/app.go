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

	"AgentMeter/internal/db"
	"AgentMeter/internal/ingest"
	"AgentMeter/internal/model"
	"AgentMeter/internal/platform"
	"AgentMeter/internal/pricing"
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
	sourcePathsConfigKey             = "source_paths"
	legacySourcePathConfigKey        = "source_path"
	sourcePathsAutoDefaultsConfigKey = "source_paths_auto_defaults"
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
	sourcePaths, err := a.configuredSourcePaths(a.ctx, a.conn)
	if err != nil {
		return model.Settings{}, err
	}
	if len(sourcePaths) == 0 {
		sourcePaths = platform.DefaultAgentSourcePaths()
	}
	sourcePath := strings.Join(sourcePaths, "\n")
	defaultSourcePaths := platform.DefaultAgentSourcePaths()
	models, err := pricing.List(a.ctx, a.conn)
	if err != nil {
		return model.Settings{}, err
	}
	return model.Settings{
		SourcePath:         sourcePath,
		SourcePaths:        sourcePaths,
		DefaultSourcePath:  strings.Join(defaultSourcePaths, "\n"),
		DefaultSourcePaths: defaultSourcePaths,
		DatabasePath:       a.dbPath,
		PricingModels:      models,
		LastIndexStartedAt: a.lastStart,
		LastIndexResult:    a.lastResult,
	}, nil
}

func (a *App) SaveSettings(sourcePath string) (model.Settings, error) {
	if err := a.ensureReady(); err != nil {
		return model.Settings{}, err
	}
	paths := parseSourcePathList(sourcePath)
	if err := setSourcePaths(a.ctx, a.conn, paths); err != nil {
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

func (a *App) configuredSourcePaths(ctx context.Context, conn *sql.DB) ([]string, error) {
	if encoded, ok, err := db.GetConfig(ctx, conn, sourcePathsConfigKey); err != nil {
		return nil, err
	} else if ok && strings.TrimSpace(encoded) != "" {
		var paths []string
		if err := json.Unmarshal([]byte(encoded), &paths); err == nil {
			return normalizeSourcePaths(paths), nil
		}
		return parseSourcePathList(encoded), nil
	}
	if legacy, ok, err := db.GetConfig(ctx, conn, legacySourcePathConfigKey); err != nil {
		return nil, err
	} else if ok {
		return parseSourcePathList(legacy), nil
	}
	return nil, nil
}

func (a *App) ensureSourcePathsConfig(ctx context.Context, conn *sql.DB) error {
	_, hasSourcePaths, err := db.GetConfig(ctx, conn, sourcePathsConfigKey)
	if err != nil {
		return err
	}
	sourcePaths, err := a.configuredSourcePaths(ctx, conn)
	if err != nil {
		return err
	}
	defaultSourcePaths := platform.DefaultAgentSourcePaths()
	if len(sourcePaths) == 0 {
		if err := setSourcePaths(ctx, conn, defaultSourcePaths); err != nil {
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
		autoDefaults = configuredDefaultSourcePaths(sourcePaths, candidates)
	}

	merged, nextAutoDefaults, changed := mergeAutoDefaultSourcePaths(sourcePaths, defaultSourcePaths, autoDefaults, candidates)
	if !hasSourcePaths || changed {
		if err := setSourcePaths(ctx, conn, merged); err != nil {
			return err
		}
	}
	if !hasAutoDefaults || !sameSourcePaths(autoDefaults, nextAutoDefaults) {
		return setAutoDefaultSourcePaths(ctx, conn, nextAutoDefaults)
	}
	return nil
}

func setSourcePaths(ctx context.Context, conn *sql.DB, paths []string) error {
	normalized := normalizeSourcePaths(paths)
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return err
	}
	if err := db.SetConfig(ctx, conn, sourcePathsConfigKey, string(encoded)); err != nil {
		return err
	}
	legacy := ""
	if len(normalized) > 0 {
		legacy = normalized[0]
	}
	return db.SetConfig(ctx, conn, legacySourcePathConfigKey, legacy)
}

func getAutoDefaultSourcePaths(ctx context.Context, conn *sql.DB) ([]string, bool, error) {
	encoded, ok, err := db.GetConfig(ctx, conn, sourcePathsAutoDefaultsConfigKey)
	if err != nil || !ok {
		return nil, ok, err
	}
	var paths []string
	if err := json.Unmarshal([]byte(encoded), &paths); err == nil {
		return normalizeSourcePaths(paths), true, nil
	}
	return parseSourcePathList(encoded), true, nil
}

func setAutoDefaultSourcePaths(ctx context.Context, conn *sql.DB, paths []string) error {
	encoded, err := json.Marshal(normalizeSourcePaths(paths))
	if err != nil {
		return err
	}
	return db.SetConfig(ctx, conn, sourcePathsAutoDefaultsConfigKey, string(encoded))
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
	key := sourcePathKey(filepath.Clean(strings.TrimSpace(path)))
	for _, candidate := range normalizeSourcePaths(paths) {
		if sourcePathKey(candidate) == key {
			return true
		}
	}
	return false
}

func appendSourcePath(paths []string, path string) []string {
	return normalizeSourcePaths(append(paths, path))
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
	if err := a.ensureReady(); err != nil {
		return nil, err
	}
	return a.query.Tools(a.ctx)
}

func (a *App) GetPricingModels() ([]model.PricingModel, error) {
	if err := a.ensureReady(); err != nil {
		return nil, err
	}
	return pricing.List(a.ctx, a.conn)
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
