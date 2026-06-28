package platform

import (
	"os"
	"path/filepath"
	"testing"

	"AgentMeter/internal/sourcepath"
)

func TestDefaultAgentSourcePathsDiscoversHomeVariantsConservatively(t *testing.T) {
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	ycodex := filepath.Join(home, ".ycodex")
	xclaude := filepath.Join(home, ".xclaude")
	xcodebuddy := filepath.Join(home, ".xcodebuddy")
	genericSessions := filepath.Join(home, "logs", "sessions")
	for _, path := range []string{
		filepath.Join(ycodex, "sessions"),
		filepath.Join(xclaude, "projects"),
		filepath.Join(xcodebuddy, "sessions"),
		genericSessions,
	} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	paths := DefaultAgentSourcePaths()
	for _, want := range []string{ycodex, xclaude, xcodebuddy} {
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
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("CODEX_HOME", codexHome)

	candidates := DefaultAgentSourceCandidates()
	if !hasPath(candidates, codexHome) {
		t.Fatalf("env root missing from candidates: %v", candidates)
	}
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
