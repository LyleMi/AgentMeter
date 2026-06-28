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

func TestResolveSourceDetectsKnownAndVariantRoots(t *testing.T) {
	dir := t.TempDir()
	tests := []struct {
		name     string
		root     string
		child    string
		wantKind string
		wantName string
	}{
		{name: "codex exact", root: ".codex", child: "sessions", wantKind: "codex", wantName: "Codex"},
		{name: "codex variant", root: ".ycodex", child: "sessions", wantKind: "codex", wantName: "Codex (.ycodex)"},
		{name: "claude exact", root: ".claude", child: "projects", wantKind: "claude", wantName: "Claude Code"},
		{name: "claude variant", root: ".xclaude", child: "projects", wantKind: "claude", wantName: "Claude Code (.xclaude)"},
		{name: "codebuddy variant", root: ".xcodebuddy", child: "projects", wantKind: "codebuddy", wantName: "CodeBuddy (.xcodebuddy)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := filepath.Join(dir, tt.root)
			if err := os.MkdirAll(filepath.Join(root, tt.child), 0o755); err != nil {
				t.Fatal(err)
			}
			spec := ResolveSource(root)
			if spec.Kind != tt.wantKind || spec.Name != tt.wantName || spec.RootPath != root || spec.SessionsPath != root {
				t.Fatalf("root spec = %+v", spec)
			}
		})
	}
}

func TestResolveSourceDetectsVariantChildPathWithoutCodexMisclassification(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, ".xcodebuddy")
	sessions := filepath.Join(root, "sessions")
	if err := os.MkdirAll(sessions, 0o755); err != nil {
		t.Fatal(err)
	}

	spec := ResolveSource(sessions)
	if spec.Kind != "codebuddy" || spec.RootPath != root || spec.SessionsPath != sessions {
		t.Fatalf("sessions spec = %+v", spec)
	}
}

func TestResolveSourceKeepsGenericSessionsAsJSONL(t *testing.T) {
	dir := t.TempDir()
	sessions := filepath.Join(dir, "logs", "sessions")
	if err := os.MkdirAll(sessions, 0o755); err != nil {
		t.Fatal(err)
	}

	spec := ResolveSource(sessions)
	if spec.Kind != "jsonl" || spec.RootPath != sessions || spec.SessionsPath != sessions {
		t.Fatalf("generic sessions spec = %+v", spec)
	}
}
