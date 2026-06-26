package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"AgentMeter/internal/db"
)

func TestStartupAddsDetectedAgentDefaultsToExistingSourcePaths(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	codex := filepath.Join(home, ".codex")
	claude := filepath.Join(home, ".claude")
	codebuddy := filepath.Join(home, ".codebuddy")
	workbuddy := filepath.Join(home, ".workbuddy")
	for _, path := range []string{codex, claude, codebuddy, workbuddy} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	dbPath := filepath.Join(dir, "agentmeter.sqlite")
	conn, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := setSourcePaths(ctx, conn, []string{codex, claude}); err != nil {
		t.Fatal(err)
	}
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}

	first := &App{dbPath: dbPath}
	if err := first.Startup(ctx); err != nil {
		t.Fatal(err)
	}
	settings, err := first.GetSettings()
	if err != nil {
		t.Fatal(err)
	}
	if !containsSourcePath(settings.SourcePaths, codebuddy) {
		t.Fatalf("source paths should include codebuddy: %v", settings.SourcePaths)
	}
	if !containsSourcePath(settings.SourcePaths, workbuddy) {
		t.Fatalf("source paths should include workbuddy: %v", settings.SourcePaths)
	}

	withoutDetectedAgents := strings.Join([]string{codex, claude}, "\n")
	if _, err := first.SaveSettings(withoutDetectedAgents); err != nil {
		t.Fatal(err)
	}
	first.Shutdown(ctx)

	second := &App{dbPath: dbPath}
	if err := second.Startup(ctx); err != nil {
		t.Fatal(err)
	}
	defer second.Shutdown(ctx)
	settings, err = second.GetSettings()
	if err != nil {
		t.Fatal(err)
	}
	if containsSourcePath(settings.SourcePaths, codebuddy) {
		t.Fatalf("codebuddy should stay removed after user save: %v", settings.SourcePaths)
	}
	if containsSourcePath(settings.SourcePaths, workbuddy) {
		t.Fatalf("workbuddy should stay removed after user save: %v", settings.SourcePaths)
	}
}

func TestMergeAutoDefaultSourcePathsLeavesCustomOnlyConfigAlone(t *testing.T) {
	custom := filepath.Join("workspace", "sessions")
	codex := filepath.Join("home", ".codex")
	codebuddy := filepath.Join("home", ".codebuddy")

	merged, autoDefaults, changed := mergeAutoDefaultSourcePaths(
		[]string{custom},
		[]string{codex, codebuddy},
		nil,
		[]string{codex, codebuddy},
	)
	if changed {
		t.Fatalf("changed custom-only config: %v", merged)
	}
	if !sameSourcePaths(merged, []string{custom}) {
		t.Fatalf("merged = %v", merged)
	}
	if len(autoDefaults) != 0 {
		t.Fatalf("autoDefaults = %v", autoDefaults)
	}
}
