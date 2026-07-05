package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/LyleMi/AgentMeter/internal/agent"
	"github.com/LyleMi/AgentMeter/internal/db"
	"github.com/LyleMi/AgentMeter/internal/model"
	"github.com/LyleMi/AgentMeter/internal/platform"
	"github.com/LyleMi/AgentMeter/internal/pricing"
	"github.com/LyleMi/AgentMeter/internal/sourcepath"
)

const (
	sourceEntriesConfigKey             = "source_entries"
	sourceEntriesAutoDefaultsConfigKey = "source_entries_auto_defaults"
)

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
