package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/LyleMi/AgentMeter/internal/db"
	"github.com/LyleMi/AgentMeter/internal/model"
	"github.com/LyleMi/AgentMeter/internal/sourcepath"
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
		{Path: enabled, Enabled: true, Label: "Nightly"},
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
	if settings.SourceEntries[0].Label != "Nightly" {
		t.Fatalf("source label was not preserved: %+v", settings.SourceEntries)
	}
	if containsSourcePath(settings.SourcePaths, disabled) {
		t.Fatalf("disabled source should not be active: %v", settings.SourcePaths)
	}
	if strings.Contains(settings.SourcePath, disabled) {
		t.Fatalf("disabled source leaked into sourcePath: %q", settings.SourcePath)
	}
}

func TestPrivacyStatusIncludesSourceOptionsWhenMultipleSourcesAreIndexed(t *testing.T) {
	tests := []struct {
		target     string
		stableDir  string
		nightlyDir string
		envKey     string
		envValue   func(string) string
	}{
		{
			target:     "codex",
			stableDir:  ".codex",
			nightlyDir: ".ycodex",
			envKey:     "CODEX_HOME",
			envValue:   func(root string) string { return root },
		},
		{
			target:     "gemini",
			stableDir:  ".gemini",
			nightlyDir: ".ygemini",
			envKey:     "AGENTMETER_GEMINI_SETTINGS_PATH",
			envValue:   settingsJSONPrivacyConfigPathForRoot,
		},
		{
			target:     "claude",
			stableDir:  ".claude",
			nightlyDir: ".yclaude",
			envKey:     "AGENTMETER_CLAUDE_SETTINGS_PATH",
			envValue:   settingsJSONPrivacyConfigPathForRoot,
		},
		{
			target:     "codebuddy",
			stableDir:  ".codebuddy",
			nightlyDir: ".ycodebuddy",
			envKey:     "AGENTMETER_CODEBUDDY_SETTINGS_PATH",
			envValue:   settingsJSONPrivacyConfigPathForRoot,
		},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			ctx := context.Background()
			dir := t.TempDir()
			dbPath := filepath.Join(dir, "agentmeter.sqlite")
			stableRoot := filepath.Join(dir, tt.stableDir)
			nightlyRoot := filepath.Join(dir, tt.nightlyDir)
			t.Setenv(tt.envKey, tt.envValue(stableRoot))

			conn, err := db.Open(dbPath)
			if err != nil {
				t.Fatal(err)
			}
			stable, err := db.EnsureSource(ctx, conn, db.SourceInput{
				Kind:         tt.target,
				Name:         privacyTargetLabel(tt.target),
				RootPath:     stableRoot,
				SessionsPath: filepath.Join(stableRoot, "sessions"),
				Platform:     "test",
			})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := db.EnsureSource(ctx, conn, db.SourceInput{
				Kind:         tt.target,
				Name:         privacyTargetLabel(tt.target) + " nightly",
				RootPath:     nightlyRoot,
				SessionsPath: filepath.Join(nightlyRoot, "sessions"),
				Platform:     "test",
			}); err != nil {
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

			status, err := app.GetPrivacyConfig(tt.target)
			if err != nil {
				t.Fatal(err)
			}
			if len(status.SourceOptions) != 2 {
				t.Fatalf("source options = %#v", status.SourceOptions)
			}
			if status.SelectedSourceKey != fmt.Sprintf("source:%d", stable.ID) {
				t.Fatalf("selected source key = %q, want source:%d", status.SelectedSourceKey, stable.ID)
			}
			if status.SourceOptions[0].ConfigPath == "" || status.SourceOptions[1].ConfigPath == "" {
				t.Fatalf("source option config paths should be populated: %#v", status.SourceOptions)
			}
			for _, warning := range status.Warnings {
				if strings.Contains(warning, "Source-specific privacy writes are not enabled") {
					t.Fatalf("unexpected source-specific warning after source options were enabled: %q", warning)
				}
			}
		})
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

func TestAgentResourcesHTTPReturnsCodexOverview(t *testing.T) {
	dir := t.TempDir()
	isolateResourceAgentHomes(t, dir)
	root := filepath.Join(dir, ".codex")
	t.Setenv("CODEX_HOME", root)
	if err := os.MkdirAll(filepath.Join(root, "skills", "sample"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "skills", "sample", "SKILL.md"), []byte("---\nname: sample\ndescription: Sample skill.\n---\n# Sample\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	app := &App{dbPath: filepath.Join(t.TempDir(), "agentmeter.sqlite")}
	defer app.Shutdown(context.Background())
	mux := http.NewServeMux()
	RegisterHTTPHandlers(mux, app, fstest.MapFS{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/agent-resources", nil)
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var overview model.AgentResourceOverview
	if err := json.NewDecoder(recorder.Body).Decode(&overview); err != nil {
		t.Fatal(err)
	}
	codex := findResourceAgent(t, overview, "codex")
	if !codex.Exists {
		t.Fatalf("agent overview = %+v", overview.Agents)
	}
	if len(overview.Agents) < 4 {
		t.Fatalf("expected known agents in overview: %+v", overview.Agents)
	}
	if len(overview.Skills) != 1 || overview.Skills[0].Name != "sample" {
		t.Fatalf("skills = %+v", overview.Skills)
	}
}

func TestAgentResourcesHTTPMissingHomeIsReadOnlyShape(t *testing.T) {
	dir := t.TempDir()
	isolateResourceAgentHomes(t, dir)
	t.Setenv("CODEX_HOME", filepath.Join(dir, "missing"))

	app := &App{dbPath: filepath.Join(t.TempDir(), "agentmeter.sqlite")}
	defer app.Shutdown(context.Background())
	mux := http.NewServeMux()
	RegisterHTTPHandlers(mux, app, fstest.MapFS{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/agent-resources", nil)
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var overview model.AgentResourceOverview
	if err := json.NewDecoder(recorder.Body).Decode(&overview); err != nil {
		t.Fatal(err)
	}
	codex := findResourceAgent(t, overview, "codex")
	if codex.Exists {
		t.Fatalf("agent overview = %+v", overview.Agents)
	}
	if overview.Skills == nil || overview.MCPServers == nil || overview.Memories == nil {
		t.Fatalf("resource arrays should be non-null: %+v", overview)
	}
}

func TestAgentResourcesHTTPCanToggleCodexSkill(t *testing.T) {
	dir := t.TempDir()
	isolateResourceAgentHomes(t, dir)
	root := filepath.Join(dir, ".codex")
	t.Setenv("CODEX_HOME", root)
	if err := os.MkdirAll(filepath.Join(root, "skills", "sample"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "skills", "sample", "SKILL.md"), []byte("---\nname: sample\n---\n# Sample\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	app := &App{dbPath: filepath.Join(t.TempDir(), "agentmeter.sqlite")}
	defer app.Shutdown(context.Background())
	mux := http.NewServeMux()
	RegisterHTTPHandlers(mux, app, fstest.MapFS{})

	recorder := httptest.NewRecorder()
	body := strings.NewReader(`{"agentKind":"codex","relativePath":"sample","enabled":false}`)
	request := httptest.NewRequest(http.MethodPost, "/api/agent-resources/skills/enabled", body)
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if _, err := os.Stat(filepath.Join(root, "skills", "sample", "SKILL.md.disabled")); err != nil {
		t.Fatal(err)
	}
}

func TestAgentResourcesHTTPCanReadAndUpdateMemory(t *testing.T) {
	dir := t.TempDir()
	isolateResourceAgentHomes(t, dir)
	root := filepath.Join(dir, ".codex")
	t.Setenv("CODEX_HOME", root)
	memoryPath := filepath.Join(root, "memories", "MEMORY.md")
	if err := os.MkdirAll(filepath.Dir(memoryPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(memoryPath, []byte("# Memory\n\nInitial.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	app := &App{dbPath: filepath.Join(t.TempDir(), "agentmeter.sqlite")}
	defer app.Shutdown(context.Background())
	mux := http.NewServeMux()
	RegisterHTTPHandlers(mux, app, fstest.MapFS{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/agent-resources/memories/detail?agentKind=codex&relativePath=MEMORY.md", nil)
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var detail model.AgentMemoryDetail
	if err := json.NewDecoder(recorder.Body).Decode(&detail); err != nil {
		t.Fatal(err)
	}
	if detail.Content != "# Memory\n\nInitial.\n" || !detail.CanEdit {
		t.Fatalf("detail = %+v", detail)
	}

	recorder = httptest.NewRecorder()
	body := strings.NewReader(`{"agentKind":"codex","relativePath":"MEMORY.md","content":"# Memory\n\nUpdated.\n"}`)
	request = httptest.NewRequest(http.MethodPost, "/api/agent-resources/memories/detail", body)
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	content, err := os.ReadFile(memoryPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "# Memory\n\nUpdated.\n" {
		t.Fatalf("memory content = %q", content)
	}
}

func TestAgentResourcesHTTPRejectsMemoryTraversal(t *testing.T) {
	dir := t.TempDir()
	isolateResourceAgentHomes(t, dir)
	root := filepath.Join(dir, ".codex")
	t.Setenv("CODEX_HOME", root)
	if err := os.MkdirAll(filepath.Join(root, "memories"), 0o755); err != nil {
		t.Fatal(err)
	}

	app := &App{dbPath: filepath.Join(t.TempDir(), "agentmeter.sqlite")}
	defer app.Shutdown(context.Background())
	mux := http.NewServeMux()
	RegisterHTTPHandlers(mux, app, fstest.MapFS{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/agent-resources/memories/detail?agentKind=codex&relativePath=../config.toml", nil)
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
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

func TestPrivacyGeminiHTTPApplyRecommendedProfile(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), ".gemini", "settings.json")
	t.Setenv("AGENTMETER_GEMINI_SETTINGS_PATH", configPath)
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(`{
  "privacy": { "usageStatisticsEnabled": true },
  "telemetry": { "enabled": true, "traces": true, "logPrompts": true },
  "general": { "sessionRetention": { "maxAge": "7d" } }
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	RegisterHTTPHandlers(mux, &App{}, fstest.MapFS{})

	recorder := httptest.NewRecorder()
	body := strings.NewReader(`{"profile":"recommended"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/privacy/gemini/profile", body)
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var result model.PrivacyConfigApplyResult
	if err := json.NewDecoder(recorder.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.Status.Target != "gemini" || len(result.Status.Profiles) != 3 {
		t.Fatalf("status should include Gemini profile metadata: %#v", result.Status)
	}
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var saved map[string]any
	if err := json.Unmarshal(content, &saved); err != nil {
		t.Fatal(err)
	}
	if saved["privacy"].(map[string]any)["usageStatisticsEnabled"] != false {
		t.Fatalf("recommended profile did not disable usage stats: %#v", saved)
	}
	retention := saved["general"].(map[string]any)["sessionRetention"].(map[string]any)
	if _, ok := retention["maxAge"]; ok {
		t.Fatalf("recommended profile should leave retention unset/default: %#v", saved)
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

func TestPrivacyCodexHTTPChangesCanTargetIndexedSource(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "agentmeter.sqlite")
	stableRoot := filepath.Join(dir, ".codex")
	nightlyRoot := filepath.Join(dir, ".ycodex")
	t.Setenv("CODEX_HOME", stableRoot)
	for _, root := range []string{stableRoot, nightlyRoot} {
		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "config.toml"), []byte("[analytics]\nenabled = true\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	conn, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.EnsureSource(ctx, conn, db.SourceInput{
		Kind:         "codex",
		Name:         "Codex",
		RootPath:     stableRoot,
		SessionsPath: filepath.Join(stableRoot, "sessions"),
		Platform:     "test",
	}); err != nil {
		t.Fatal(err)
	}
	nightly, err := db.EnsureSource(ctx, conn, db.SourceInput{
		Kind:         "codex",
		Name:         "Codex nightly",
		RootPath:     nightlyRoot,
		SessionsPath: filepath.Join(nightlyRoot, "sessions"),
		Platform:     "test",
	})
	if err != nil {
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

	mux := http.NewServeMux()
	RegisterHTTPHandlers(mux, app, fstest.MapFS{})

	recorder := httptest.NewRecorder()
	body := strings.NewReader(fmt.Sprintf(`{"sourceKey":"source:%d","changes":[{"id":"analytics.enabled","op":"set","value":false}]}`, nightly.ID))
	request := httptest.NewRequest(http.MethodPost, "/api/privacy/codex/changes", body)
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var result model.PrivacyConfigApplyResult
	if err := json.NewDecoder(recorder.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.Status.SelectedSourceKey != fmt.Sprintf("source:%d", nightly.ID) {
		t.Fatalf("selected source key = %q, want source:%d", result.Status.SelectedSourceKey, nightly.ID)
	}
	stableContent, err := os.ReadFile(filepath.Join(stableRoot, "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(stableContent), "enabled = true") {
		t.Fatalf("stable config should not be updated:\n%s", stableContent)
	}
	nightlyContent, err := os.ReadFile(filepath.Join(nightlyRoot, "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(nightlyContent), "enabled = false") {
		t.Fatalf("nightly config was not updated:\n%s", nightlyContent)
	}
}

func TestPrivacyJSONHTTPChangesCanTargetIndexedSource(t *testing.T) {
	tests := []struct {
		target     string
		stableDir  string
		nightlyDir string
		envKey     string
		original   string
		changeJSON string
		want       string
	}{
		{
			target:     "gemini",
			stableDir:  ".gemini",
			nightlyDir: ".ygemini",
			envKey:     "AGENTMETER_GEMINI_SETTINGS_PATH",
			original:   `{"privacy":{"usageStatisticsEnabled":true}}`,
			changeJSON: `{"id":"privacy.usageStatisticsEnabled","op":"set","value":false}`,
			want:       `"usageStatisticsEnabled": false`,
		},
		{
			target:     "claude",
			stableDir:  ".claude",
			nightlyDir: ".yclaude",
			envKey:     "AGENTMETER_CLAUDE_SETTINGS_PATH",
			original:   `{"env":{"DISABLE_TELEMETRY":"0"}}`,
			changeJSON: `{"id":"env.DISABLE_TELEMETRY","op":"set","value":"1"}`,
			want:       `"DISABLE_TELEMETRY": "1"`,
		},
		{
			target:     "codebuddy",
			stableDir:  ".codebuddy",
			nightlyDir: ".ycodebuddy",
			envKey:     "AGENTMETER_CODEBUDDY_SETTINGS_PATH",
			original:   `{"cleanupPeriodDays":30}`,
			changeJSON: `{"id":"cleanupPeriodDays","op":"set","value":7}`,
			want:       `"cleanupPeriodDays": 7`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			ctx := context.Background()
			dir := t.TempDir()
			dbPath := filepath.Join(dir, "agentmeter.sqlite")
			stableRoot := filepath.Join(dir, tt.stableDir)
			nightlyRoot := filepath.Join(dir, tt.nightlyDir)
			stableConfig := settingsJSONPrivacyConfigPathForRoot(stableRoot)
			nightlyConfig := settingsJSONPrivacyConfigPathForRoot(nightlyRoot)
			t.Setenv(tt.envKey, stableConfig)
			for _, path := range []string{stableConfig, nightlyConfig} {
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(tt.original), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			conn, err := db.Open(dbPath)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := db.EnsureSource(ctx, conn, db.SourceInput{
				Kind:         tt.target,
				Name:         privacyTargetLabel(tt.target),
				RootPath:     stableRoot,
				SessionsPath: filepath.Join(stableRoot, "sessions"),
				Platform:     "test",
			}); err != nil {
				t.Fatal(err)
			}
			nightly, err := db.EnsureSource(ctx, conn, db.SourceInput{
				Kind:         tt.target,
				Name:         privacyTargetLabel(tt.target) + " nightly",
				RootPath:     nightlyRoot,
				SessionsPath: filepath.Join(nightlyRoot, "sessions"),
				Platform:     "test",
			})
			if err != nil {
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

			mux := http.NewServeMux()
			RegisterHTTPHandlers(mux, app, fstest.MapFS{})

			recorder := httptest.NewRecorder()
			body := strings.NewReader(fmt.Sprintf(`{"sourceKey":"source:%d","changes":[%s]}`, nightly.ID, tt.changeJSON))
			request := httptest.NewRequest(http.MethodPost, "/api/privacy/"+tt.target+"/changes", body)
			mux.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
			}
			var result model.PrivacyConfigApplyResult
			if err := json.NewDecoder(recorder.Body).Decode(&result); err != nil {
				t.Fatal(err)
			}
			if result.Status.SelectedSourceKey != fmt.Sprintf("source:%d", nightly.ID) {
				t.Fatalf("selected source key = %q, want source:%d", result.Status.SelectedSourceKey, nightly.ID)
			}
			stableContent, err := os.ReadFile(stableConfig)
			if err != nil {
				t.Fatal(err)
			}
			if string(stableContent) != tt.original {
				t.Fatalf("stable config should not be updated:\n%s", stableContent)
			}
			nightlyContent, err := os.ReadFile(nightlyConfig)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(nightlyContent), tt.want) {
				t.Fatalf("nightly config was not updated:\n%s", nightlyContent)
			}
		})
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

func TestPricingHTTPSavesCustomModel(t *testing.T) {
	app := &App{dbPath: filepath.Join(t.TempDir(), "agentmeter.sqlite")}
	defer app.Shutdown(context.Background())
	mux := http.NewServeMux()
	RegisterHTTPHandlers(mux, app, fstest.MapFS{})

	recorder := httptest.NewRecorder()
	body := strings.NewReader(`{"model":"codex-auto-review","inputPer1m":9,"cachedInputPer1m":1,"outputPer1m":20}`)
	request := httptest.NewRequest(http.MethodPost, "/api/pricing", body)
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var saved model.PricingModel
	if err := json.NewDecoder(recorder.Body).Decode(&saved); err != nil {
		t.Fatal(err)
	}
	if !saved.IsCustom || saved.NormalizedModel != "codex-auto-review" || saved.InputPer1M != 9 || saved.OutputPer1M != 20 {
		t.Fatalf("saved pricing = %+v", saved)
	}

	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/api/pricing", nil)
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var models []model.PricingModel
	if err := json.NewDecoder(recorder.Body).Decode(&models); err != nil {
		t.Fatal(err)
	}
	if !hasPricingModel(models, "codex-auto-review") {
		t.Fatalf("custom model missing from pricing list: %+v", models)
	}
}

func TestPricingHTTPRejectsInvalidCustomModel(t *testing.T) {
	app := &App{dbPath: filepath.Join(t.TempDir(), "agentmeter.sqlite")}
	defer app.Shutdown(context.Background())
	mux := http.NewServeMux()
	RegisterHTTPHandlers(mux, app, fstest.MapFS{})

	recorder := httptest.NewRecorder()
	body := strings.NewReader(`{"model":"","inputPer1m":-1}`)
	request := httptest.NewRequest(http.MethodPost, "/api/pricing", body)
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
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

func hasPricingModel(models []model.PricingModel, normalized string) bool {
	for _, item := range models {
		if item.NormalizedModel == normalized {
			return true
		}
	}
	return false
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

func isolateResourceAgentHomes(t *testing.T, dir string) {
	t.Helper()
	home := filepath.Join(dir, "home")
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("AGENTMETER_GEMINI_SETTINGS_PATH", filepath.Join(dir, ".gemini", "settings.json"))
	t.Setenv("AGENTMETER_CLAUDE_SETTINGS_PATH", filepath.Join(dir, ".claude", "settings.json"))
	t.Setenv("AGENTMETER_CODEBUDDY_SETTINGS_PATH", filepath.Join(dir, ".codebuddy", "settings.json"))
	t.Setenv("WORKBUDDY_CONFIG_DIR", filepath.Join(dir, ".workbuddy"))
	t.Setenv("CURSOR_HOME", filepath.Join(dir, ".cursor"))
}

func findResourceAgent(t *testing.T, overview model.AgentResourceOverview, kind string) model.AgentResourceAgent {
	t.Helper()
	for _, agent := range overview.Agents {
		if agent.Kind == kind {
			return agent
		}
	}
	t.Fatalf("agent %q missing: %+v", kind, overview.Agents)
	return model.AgentResourceAgent{}
}
