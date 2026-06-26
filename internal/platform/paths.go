package platform

import (
	"os"
	"path/filepath"
	"runtime"
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

func DefaultDatabasePath() (string, error) {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		base = os.Getenv("APPDATA")
	}
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".agentmeter")
	} else {
		base = filepath.Join(base, "AgentMeter")
	}
	if err := os.MkdirAll(base, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(base, "agentmeter.sqlite"), nil
}

func PlatformName() string {
	return runtime.GOOS
}
