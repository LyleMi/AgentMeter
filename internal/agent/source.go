package agent

import (
	"os"
	"path/filepath"
	"strings"
)

type SourceSpec struct {
	Kind         string
	Name         string
	RootPath     string
	SessionsPath string
}

type UsageSource struct {
	Dir         string
	DedupeScope string
}

func ResolveSource(path string) SourceSpec {
	cleaned := filepath.Clean(path)
	parent := filepath.Dir(cleaned)
	base := strings.ToLower(filepath.Base(cleaned))

	if isCodexRoot(cleaned) {
		return SourceSpec{Kind: "codex", Name: "Codex", RootPath: cleaned, SessionsPath: cleaned}
	}
	if (base == "sessions" || base == "archived_sessions") && isCodexRoot(parent) {
		return SourceSpec{Kind: "codex", Name: "Codex", RootPath: parent, SessionsPath: cleaned}
	}
	if isClaudeRoot(cleaned) {
		return SourceSpec{Kind: "claude", Name: "Claude Code", RootPath: cleaned, SessionsPath: cleaned}
	}
	if base == "projects" && isClaudeRoot(parent) {
		return SourceSpec{Kind: "claude", Name: "Claude Code", RootPath: parent, SessionsPath: cleaned}
	}
	return SourceSpec{Kind: "jsonl", Name: "Generic JSONL", RootPath: cleaned, SessionsPath: cleaned}
}

func UsageSources(spec SourceSpec) []UsageSource {
	switch spec.Kind {
	case "codex":
		if filepath.Clean(spec.RootPath) != filepath.Clean(spec.SessionsPath) {
			return []UsageSource{{Dir: spec.SessionsPath, DedupeScope: spec.SessionsPath}}
		}
		sessions := filepath.Join(spec.RootPath, "sessions")
		archived := filepath.Join(spec.RootPath, "archived_sessions")
		sources := make([]UsageSource, 0, 2)
		if isDir(sessions) {
			sources = append(sources, UsageSource{Dir: sessions, DedupeScope: spec.RootPath})
		}
		if isDir(archived) {
			sources = append(sources, UsageSource{Dir: archived, DedupeScope: spec.RootPath})
		}
		if len(sources) > 0 {
			return sources
		}
	case "claude":
		if filepath.Clean(spec.RootPath) == filepath.Clean(spec.SessionsPath) {
			projects := filepath.Join(spec.RootPath, "projects")
			if isDir(projects) {
				return []UsageSource{{Dir: projects, DedupeScope: spec.RootPath}}
			}
		}
	}
	return []UsageSource{{Dir: spec.SessionsPath, DedupeScope: spec.SessionsPath}}
}

func isCodexRoot(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	return base == ".codex" || isDir(filepath.Join(path, "sessions")) || isDir(filepath.Join(path, "archived_sessions"))
}

func isClaudeRoot(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	return base == ".claude" || isDir(filepath.Join(path, "projects"))
}

func isDir(path string) bool {
	stat, err := os.Stat(path)
	return err == nil && stat.IsDir()
}
