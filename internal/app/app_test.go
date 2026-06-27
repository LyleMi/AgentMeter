package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"AgentMeter/internal/db"
	"AgentMeter/internal/model"
)

func TestStartupAddsDetectedAgentDefaultsToExistingSourceEntries(t *testing.T) {
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
	if err := setSourceEntries(ctx, conn, []model.SourceEntry{
		{Path: codex, Enabled: true},
		{Path: claude, Enabled: true},
	}); err != nil {
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

	if _, err := first.SaveSourceSettings([]model.SourceEntry{
		{Path: codex, Enabled: true},
		{Path: claude, Enabled: true},
	}); err != nil {
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

func TestStartupAddsDetectedAgentDefaultsWhenCodexSessionsPathIsConfigured(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	codex := filepath.Join(home, ".codex")
	codexSessions := filepath.Join(codex, "sessions")
	codebuddy := filepath.Join(home, ".codebuddy")
	workbuddy := filepath.Join(home, ".workbuddy")
	for _, path := range []string{codexSessions, codebuddy, workbuddy} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	dbPath := filepath.Join(dir, "agentmeter.sqlite")
	conn, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := setSourceEntries(ctx, conn, []model.SourceEntry{{Path: codexSessions, Enabled: true}}); err != nil {
		t.Fatal(err)
	}
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}

	app := &App{dbPath: dbPath}
	if err := app.Startup(ctx); err != nil {
		t.Fatal(err)
	}
	defer app.Shutdown(ctx)

	settings, err := app.GetSettings()
	if err != nil {
		t.Fatal(err)
	}
	if !containsSourcePath(settings.SourcePaths, codebuddy) {
		t.Fatalf("source paths should include codebuddy: %v", settings.SourcePaths)
	}
	if !containsSourcePath(settings.SourcePaths, workbuddy) {
		t.Fatalf("source paths should include workbuddy: %v", settings.SourcePaths)
	}
	if !containsExactSourcePath(settings.SourcePaths, codexSessions) {
		t.Fatalf("existing codex sessions source should not be rewritten unexpectedly: %v", settings.SourcePaths)
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

func TestSaveSourceSettingsKeepsDisabledEntriesOutOfActivePaths(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "agentmeter.sqlite")
	enabled := filepath.Join(dir, "enabled-source")
	disabled := filepath.Join(dir, "disabled-source")

	app := &App{dbPath: dbPath}
	if err := app.Startup(ctx); err != nil {
		t.Fatal(err)
	}
	defer app.Shutdown(ctx)

	settings, err := app.SaveSourceSettings([]model.SourceEntry{
		{Path: enabled, Enabled: true},
		{Path: disabled, Enabled: false},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(settings.SourceEntries) != 2 {
		t.Fatalf("source entries = %v", settings.SourceEntries)
	}
	if !containsSourcePath(settings.SourcePaths, enabled) {
		t.Fatalf("enabled source should be active: %v", settings.SourcePaths)
	}
	if containsSourcePath(settings.SourcePaths, disabled) {
		t.Fatalf("disabled source should not be active: %v", settings.SourcePaths)
	}
	if strings.Contains(settings.SourcePath, disabled) {
		t.Fatalf("disabled source leaked into sourcePath: %q", settings.SourcePath)
	}
}

func TestPrivacyCodexHTTPApplyEmptyBodyAppliesAll(t *testing.T) {
	codexHome := filepath.Join(t.TempDir(), "codex-home")
	t.Setenv("CODEX_HOME", codexHome)

	mux := http.NewServeMux()
	RegisterHTTPHandlers(mux, &App{}, fstest.MapFS{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/privacy/codex/apply", nil)
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var result model.PrivacyConfigApplyResult
	if err := json.NewDecoder(recorder.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if !result.Status.Exists {
		t.Fatalf("status should report created config: %#v", result.Status)
	}
	if len(result.Changed) == 0 {
		t.Fatal("empty body should apply all supported settings")
	}
	if _, err := os.Stat(filepath.Join(codexHome, "config.toml")); err != nil {
		t.Fatal(err)
	}
}

func TestPrivacyGeminiHTTPApplyEmptyBodyAppliesAll(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), ".gemini", "settings.json")
	t.Setenv("AGENTMETER_GEMINI_SETTINGS_PATH", configPath)

	mux := http.NewServeMux()
	RegisterHTTPHandlers(mux, &App{}, fstest.MapFS{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/privacy/gemini/apply", nil)
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var result model.PrivacyConfigApplyResult
	if err := json.NewDecoder(recorder.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.Status.Target != "gemini" || !result.Status.Exists {
		t.Fatalf("status should report created Gemini config: %#v", result.Status)
	}
	if len(result.Changed) == 0 {
		t.Fatal("empty body should apply all supported settings")
	}
	if _, err := os.Stat(configPath); err != nil {
		t.Fatal(err)
	}
}

func containsExactSourcePath(paths []string, path string) bool {
	key := sourcePathKey(filepath.Clean(path))
	for _, candidate := range normalizeSourcePaths(paths) {
		if sourcePathKey(candidate) == key {
			return true
		}
	}
	return false
}
