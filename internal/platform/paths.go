package platform

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

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

func DefaultAgentSourcePaths() []string {
	candidates := []string{DefaultCodexRoot(), DefaultClaudeRoot()}
	var existing []string
	for _, candidate := range candidates {
		if stat, err := os.Stat(candidate); err == nil && stat.IsDir() {
			existing = append(existing, candidate)
		}
	}
	if len(existing) > 0 {
		return existing
	}
	return candidates[:1]
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
