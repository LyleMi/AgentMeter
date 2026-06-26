package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveSourceDetectsCodeBuddyBeforeBroadSessionRoots(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, ".codebuddy")
	for _, path := range []string{
		filepath.Join(root, "projects"),
		filepath.Join(root, "sessions"),
	} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	rootSpec := ResolveSource(root)
	if rootSpec.Kind != "codebuddy" || rootSpec.Name != "CodeBuddy" {
		t.Fatalf("root spec = %+v", rootSpec)
	}

	sessionsSpec := ResolveSource(filepath.Join(root, "sessions"))
	if sessionsSpec.Kind != "codebuddy" || sessionsSpec.RootPath != root {
		t.Fatalf("sessions spec = %+v", sessionsSpec)
	}
}
