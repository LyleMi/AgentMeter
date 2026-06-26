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

func TestResolveSourceDetectsWorkBuddyBeforeBroadSessionRoots(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, ".workbuddy")
	for _, path := range []string{
		filepath.Join(root, "projects"),
		filepath.Join(root, "sessions"),
	} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	rootSpec := ResolveSource(root)
	if rootSpec.Kind != "workbuddy" || rootSpec.Name != "WorkBuddy" {
		t.Fatalf("root spec = %+v", rootSpec)
	}

	sessionsSpec := ResolveSource(filepath.Join(root, "sessions"))
	if sessionsSpec.Kind != "workbuddy" || sessionsSpec.RootPath != root {
		t.Fatalf("sessions spec = %+v", sessionsSpec)
	}

	projectsSpec := ResolveSource(filepath.Join(root, "projects"))
	if projectsSpec.Kind != "workbuddy" || projectsSpec.RootPath != root {
		t.Fatalf("projects spec = %+v", projectsSpec)
	}
}
