package platform

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LyleMi/AgentMeter/internal/sourcepath"
)

func TestDefaultAgentSourcePathsDiscoversHomeVariantsConservatively(t *testing.T) {
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	ycodex := filepath.Join(home, ".ycodex")
	xclaude := filepath.Join(home, ".xclaude")
	xcodebuddy := filepath.Join(home, ".xcodebuddy")
	xcursor := filepath.Join(home, ".xcursor")
	genericSessions := filepath.Join(home, "logs", "sessions")
	for _, path := range []string{
		filepath.Join(ycodex, "sessions"),
		filepath.Join(xclaude, "projects"),
		filepath.Join(xcodebuddy, "sessions"),
		filepath.Join(xcursor, "projects"),
		genericSessions,
	} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	paths := DefaultAgentSourcePaths()
	for _, want := range []string{ycodex, xclaude, xcodebuddy, xcursor} {
		if !hasPath(paths, want) {
			t.Fatalf("paths missing %s: %v", want, paths)
		}
	}
	if hasPath(paths, filepath.Join(home, "logs")) || hasPath(paths, genericSessions) {
		t.Fatalf("generic sessions should not be auto-discovered: %v", paths)
	}
}

func TestDiscoverAgentSourceCandidatesIncludesEnvironmentRoots(t *testing.T) {
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	codexHome := filepath.Join(dir, "codex-env")
	cursorHome := filepath.Join(dir, "cursor-env")
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("CODEX_HOME", codexHome)
	t.Setenv("CURSOR_HOME", cursorHome)

	candidates := DefaultAgentSourceCandidates()
	if !hasPath(candidates, codexHome) {
		t.Fatalf("env root missing from candidates: %v", candidates)
	}
	if !hasPath(candidates, cursorHome) {
		t.Fatalf("cursor env root missing from candidates: %v", candidates)
	}
}

func TestDiscoverAgentSourceCandidatesIncludesDefaultCursorRoot(t *testing.T) {
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	cursorRoot := filepath.Join(home, ".cursor")

	candidates := DiscoverAgentSourceCandidates()
	for _, candidate := range candidates {
		if sourcepath.Key(candidate.Path) != sourcepath.Key(sourcepath.Normalize(cursorRoot)) {
			continue
		}
		if candidate.Kind != "cursor" || candidate.Name != "Cursor" {
			t.Fatalf("cursor candidate = %+v", candidate)
		}
		return
	}
	t.Fatalf("default cursor root missing from candidates: %+v", candidates)
}

func hasPath(paths []string, path string) bool {
	key := sourcepath.Key(sourcepath.Normalize(path))
	for _, candidate := range sourcepath.NormalizeList(paths) {
		if sourcepath.Key(candidate) == key {
			return true
		}
	}
	return false
}
