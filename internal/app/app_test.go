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
	"AgentMeter/internal/sourcepath"
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

func TestPrivacyClaudeHTTPApplyEmptyBodyAppliesAll(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), ".claude", "settings.json")
	t.Setenv("AGENTMETER_CLAUDE_SETTINGS_PATH", configPath)

	mux := http.NewServeMux()
	RegisterHTTPHandlers(mux, &App{}, fstest.MapFS{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/privacy/claude/apply", nil)
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var result model.PrivacyConfigApplyResult
	if err := json.NewDecoder(recorder.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.Status.Target != "claude" || !result.Status.Exists {
		t.Fatalf("status should report created Claude config: %#v", result.Status)
	}
	if len(result.Changed) == 0 {
		t.Fatal("empty body should apply all supported settings")
	}
	if _, err := os.Stat(configPath); err != nil {
		t.Fatal(err)
	}
}

func TestPrivacyCodeBuddyHTTPApplyEmptyBodyAppliesAll(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), ".codebuddy", "settings.json")
	t.Setenv("AGENTMETER_CODEBUDDY_SETTINGS_PATH", configPath)

	mux := http.NewServeMux()
	RegisterHTTPHandlers(mux, &App{}, fstest.MapFS{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/privacy/codebuddy/apply", nil)
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var result model.PrivacyConfigApplyResult
	if err := json.NewDecoder(recorder.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.Status.Target != "codebuddy" || !result.Status.Exists {
		t.Fatalf("status should report created CodeBuddy config: %#v", result.Status)
	}
	if len(result.Changed) == 0 {
		t.Fatal("empty body should apply all supported settings")
	}
	if _, err := os.Stat(configPath); err != nil {
		t.Fatal(err)
	}
}

func TestPrivacyCodexHTTPChangesAppliesEditableChanges(t *testing.T) {
	codexHome := filepath.Join(t.TempDir(), "codex-home")
	t.Setenv("CODEX_HOME", codexHome)
	if err := os.MkdirAll(codexHome, 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(codexHome, "config.toml")
	if err := os.WriteFile(configPath, []byte("[analytics]\nenabled = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	RegisterHTTPHandlers(mux, &App{}, fstest.MapFS{})

	recorder := httptest.NewRecorder()
	body := strings.NewReader(`{"changes":[{"id":"analytics.enabled","op":"set","value":false}]}`)
	request := httptest.NewRequest(http.MethodPost, "/api/privacy/codex/changes", body)
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var result model.PrivacyConfigApplyResult
	if err := json.NewDecoder(recorder.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if len(result.Changed) != 1 {
		t.Fatalf("changed = %d, want 1: %#v", len(result.Changed), result.Changed)
	}
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "enabled = false") {
		t.Fatalf("config was not updated:\n%s", content)
	}
}

func TestPrivacyGeminiHTTPChangesAppliesEditableChanges(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), ".gemini", "settings.json")
	t.Setenv("AGENTMETER_GEMINI_SETTINGS_PATH", configPath)
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(`{"privacy":{"usageStatisticsEnabled":true}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	RegisterHTTPHandlers(mux, &App{}, fstest.MapFS{})

	recorder := httptest.NewRecorder()
	body := strings.NewReader(`{"changes":[{"id":"privacy.usageStatisticsEnabled","op":"set","value":false}]}`)
	request := httptest.NewRequest(http.MethodPost, "/api/privacy/gemini/changes", body)
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var result model.PrivacyConfigApplyResult
	if err := json.NewDecoder(recorder.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if len(result.Changed) != 1 {
		t.Fatalf("changed = %d, want 1: %#v", len(result.Changed), result.Changed)
	}
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var saved map[string]any
	if err := json.Unmarshal(content, &saved); err != nil {
		t.Fatal(err)
	}
	privacy := saved["privacy"].(map[string]any)
	if privacy["usageStatisticsEnabled"] != false {
		t.Fatalf("config was not updated: %#v", saved)
	}
}

func TestPrivacyClaudeHTTPChangesAppliesEditableChanges(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), ".claude", "settings.json")
	t.Setenv("AGENTMETER_CLAUDE_SETTINGS_PATH", configPath)
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(`{"env":{"DISABLE_TELEMETRY":"0"}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	RegisterHTTPHandlers(mux, &App{}, fstest.MapFS{})

	recorder := httptest.NewRecorder()
	body := strings.NewReader(`{"changes":[{"id":"env.DISABLE_TELEMETRY","op":"set","value":"1"}]}`)
	request := httptest.NewRequest(http.MethodPost, "/api/privacy/claude/changes", body)
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var result model.PrivacyConfigApplyResult
	if err := json.NewDecoder(recorder.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if len(result.Changed) != 1 {
		t.Fatalf("changed = %d, want 1: %#v", len(result.Changed), result.Changed)
	}
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var saved map[string]any
	if err := json.Unmarshal(content, &saved); err != nil {
		t.Fatal(err)
	}
	env := saved["env"].(map[string]any)
	if env["DISABLE_TELEMETRY"] != "1" {
		t.Fatalf("config was not updated: %#v", saved)
	}
}

func TestPrivacyCodeBuddyHTTPChangesAppliesEditableChanges(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), ".codebuddy", "settings.json")
	t.Setenv("AGENTMETER_CODEBUDDY_SETTINGS_PATH", configPath)
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(`{"cleanupPeriodDays":30}`), 0o644); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	RegisterHTTPHandlers(mux, &App{}, fstest.MapFS{})

	recorder := httptest.NewRecorder()
	body := strings.NewReader(`{"changes":[{"id":"cleanupPeriodDays","op":"set","value":7}]}`)
	request := httptest.NewRequest(http.MethodPost, "/api/privacy/codebuddy/changes", body)
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var result model.PrivacyConfigApplyResult
	if err := json.NewDecoder(recorder.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if len(result.Changed) != 1 {
		t.Fatalf("changed = %d, want 1: %#v", len(result.Changed), result.Changed)
	}
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var saved map[string]any
	if err := json.Unmarshal(content, &saved); err != nil {
		t.Fatal(err)
	}
	if saved["cleanupPeriodDays"] != float64(7) {
		t.Fatalf("config was not updated: %#v", saved)
	}
}

func TestPrivacyHTTPUnsupportedTargetReturnsNotFound(t *testing.T) {
	mux := http.NewServeMux()
	RegisterHTTPHandlers(mux, &App{}, fstest.MapFS{})

	recorder := httptest.NewRecorder()
	body := strings.NewReader(`{`)
	request := httptest.NewRequest(http.MethodPost, "/api/privacy/unknown/changes", body)
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "unsupported privacy target: unknown") {
		t.Fatalf("body should explain unsupported target: %s", recorder.Body.String())
	}
}

func TestUnknownAPIRouteDoesNotServeFrontendIndex(t *testing.T) {
	mux := http.NewServeMux()
	RegisterHTTPHandlers(mux, &App{}, fstest.MapFS{
		"index.html": {Data: []byte("<!doctype html><title>AgentMeter</title>")},
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/privacy/codex/missing", nil)
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.Contains(contentType, "application/json") {
		t.Fatalf("content type = %q, want json", contentType)
	}
	if strings.Contains(recorder.Body.String(), "<!doctype html>") {
		t.Fatalf("unknown API route served frontend index: %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "api route not found") {
		t.Fatalf("body should explain missing API route: %s", recorder.Body.String())
	}
}

func containsExactSourcePath(paths []string, path string) bool {
	key := sourcepath.Key(sourcepath.Normalize(path))
	for _, candidate := range sourcepath.NormalizeList(paths) {
		if sourcepath.Key(candidate) == key {
			return true
		}
	}
	return false
}
