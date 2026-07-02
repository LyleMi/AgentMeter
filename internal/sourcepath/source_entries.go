package sourcepath

import (
	"strings"

	"github.com/LyleMi/AgentMeter/internal/model"
)

func SourceEntriesFromPaths(paths []string, enabled bool) []model.SourceEntry {
	normalized := NormalizeList(paths)
	entries := make([]model.SourceEntry, 0, len(normalized))
	for _, path := range normalized {
		entries = append(entries, model.SourceEntry{Path: path, Enabled: enabled})
	}
	return entries
}

func NormalizeSourceEntries(entries []model.SourceEntry) []model.SourceEntry {
	seen := map[string]struct{}{}
	result := make([]model.SourceEntry, 0, len(entries))
	for _, entry := range entries {
		cleaned := Normalize(entry.Path)
		if cleaned == "" {
			continue
		}
		key := Key(cleaned)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, model.SourceEntry{
			Path:    cleaned,
			Enabled: entry.Enabled,
			Label:   strings.TrimSpace(entry.Label),
		})
	}
	return result
}

func SourceEntryPaths(entries []model.SourceEntry) []string {
	normalized := NormalizeSourceEntries(entries)
	paths := make([]string, 0, len(normalized))
	for _, entry := range normalized {
		paths = append(paths, entry.Path)
	}
	return paths
}

func EnabledSourceEntryPaths(entries []model.SourceEntry) []string {
	var paths []string
	for _, entry := range NormalizeSourceEntries(entries) {
		if entry.Enabled {
			paths = append(paths, entry.Path)
		}
	}
	return NormalizeList(paths)
}

func EnabledSourceEntries(entries []model.SourceEntry) []model.SourceEntry {
	var result []model.SourceEntry
	for _, entry := range NormalizeSourceEntries(entries) {
		if entry.Enabled {
			result = append(result, entry)
		}
	}
	return result
}
