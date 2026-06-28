package platform

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"AgentMeter/internal/agent"
	"AgentMeter/internal/sourcepath"
)

type SourceCandidate struct {
	Path       string
	Kind       string
	Name       string
	AutoReason string
}

func DefaultCodexRoot() string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".codex")
	}
	return ".codex"
}

func DefaultCodexSessionsPath() string {
	return filepath.Join(DefaultCodexRoot(), "sessions")
}

func DefaultCodexSourcePath() string {
	return DefaultCodexRoot()
}

func DefaultClaudeRoot() string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".claude")
	}
	return ".claude"
}

func DefaultCodeBuddyRoot() string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".codebuddy")
	}
	return ".codebuddy"
}

func DefaultWorkBuddyRoot() string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".workbuddy")
	}
	return ".workbuddy"
}

func DefaultCursorRoot() string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".cursor")
	}
	return ".cursor"
}

func DefaultAgentSourceCandidates() []string {
	candidates := DiscoverAgentSourceCandidates()
	paths := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		paths = append(paths, candidate.Path)
	}
	return sourcepath.NormalizeList(paths)
}

func DiscoverAgentSourceCandidates() []SourceCandidate {
	var candidates []SourceCandidate
	add := func(path, reason string) {
		path = sourcepath.Normalize(path)
		if path == "" {
			return
		}
		spec := agent.ResolveSource(path)
		kind, name := spec.Kind, spec.Name
		if kind == "jsonl" {
			kind, name = inferCandidateFamily(path)
		}
		candidates = append(candidates, SourceCandidate{Path: path, Kind: kind, Name: name, AutoReason: reason})
	}

	add(DefaultCodexRoot(), "default")
	add(DefaultClaudeRoot(), "default")
	add(DefaultCodeBuddyRoot(), "default")
	add(DefaultWorkBuddyRoot(), "default")
	add(DefaultCursorRoot(), "default")

	for _, env := range []string{"CODEX_HOME", "CLAUDE_CONFIG_DIR", "CODEBUDDY_CONFIG_DIR", "WORKBUDDY_CONFIG_DIR", "CURSOR_HOME"} {
		if value := strings.TrimSpace(os.Getenv(env)); value != "" {
			add(value, "env:"+env)
		}
	}

	if home, err := os.UserHomeDir(); err == nil && home != "" {
		for _, path := range homeAgentVariants(home) {
			add(path, "home")
		}
	}
	return uniqueCandidates(candidates)
}

func DefaultAgentSourcePaths() []string {
	candidates := DefaultAgentSourceCandidates()
	var existing []string
	for _, candidate := range candidates {
		if stat, err := os.Stat(candidate); err == nil && stat.IsDir() {
			existing = append(existing, candidate)
		}
	}
	if len(existing) > 0 {
		return sourcepath.NormalizeList(existing)
	}
	return sourcepath.NormalizeList(candidates[:1])
}

func homeAgentVariants(home string) []string {
	entries, err := os.ReadDir(home)
	if err != nil {
		return nil
	}
	var paths []string
	for _, entry := range entries {
		if !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		name := strings.ToLower(entry.Name())
		if !containsKnownAgentToken(name) {
			continue
		}
		path := filepath.Join(home, entry.Name())
		spec := agent.ResolveSource(path)
		if spec.Kind == "jsonl" {
			continue
		}
		paths = append(paths, path)
	}
	return paths
}

func containsKnownAgentToken(name string) bool {
	for _, token := range []string{"codex", "claude", "codebuddy", "workbuddy", "cursor"} {
		if strings.Contains(name, token) {
			return true
		}
	}
	return false
}

func inferCandidateFamily(path string) (string, string) {
	name := strings.ToLower(filepath.Base(path))
	switch {
	case strings.Contains(name, "codebuddy"):
		return "codebuddy", "CodeBuddy"
	case strings.Contains(name, "workbuddy"):
		return "workbuddy", "WorkBuddy"
	case strings.Contains(name, "claude"):
		return "claude", "Claude Code"
	case strings.Contains(name, "codex"):
		return "codex", "Codex"
	case strings.Contains(name, "cursor"):
		return "cursor", "Cursor"
	default:
		return "jsonl", "Generic JSONL"
	}
}

func uniqueCandidates(candidates []SourceCandidate) []SourceCandidate {
	seen := map[string]struct{}{}
	result := make([]SourceCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		path := sourcepath.Normalize(candidate.Path)
		if path == "" {
			continue
		}
		key := sourcepath.Key(path)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		candidate.Path = path
		result = append(result, candidate)
	}
	return result
}

func DefaultAgentSourcePath() string {
	return strings.Join(DefaultAgentSourcePaths(), "\n")
}

func DefaultDatabasePath() (string, error) {
	base := ""
	switch runtime.GOOS {
	case "windows":
		base = os.Getenv("LOCALAPPDATA")
		if base == "" {
			base = os.Getenv("APPDATA")
		}
		if base != "" {
			base = filepath.Join(base, "AgentMeter")
		}
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, "Library", "Application Support", "AgentMeter")
	default:
		if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
			base = filepath.Join(xdg, "AgentMeter")
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			base = filepath.Join(home, ".local", "share", "AgentMeter")
		}
	}
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".agentmeter")
	}
	if err := os.MkdirAll(base, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(base, "agentmeter.sqlite"), nil
}

func PlatformName() string {
	return runtime.GOOS
}
