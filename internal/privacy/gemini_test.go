package privacy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"AgentMeter/internal/model"
)

func TestGeminiStatusMissingConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), ".gemini", "settings.json")
	adapter := NewGeminiAdapter()
	adapter.ConfigPath = configPath

	status, err := adapter.Status()
	if err != nil {
		t.Fatal(err)
	}
	if status.Exists {
		t.Fatal("missing config should report exists=false")
	}
	if status.Target != "gemini" || status.Name != "Gemini CLI" {
		t.Fatalf("target = %q, name = %q", status.Target, status.Name)
	}
	if status.ConfigPath != configPath {
		t.Fatalf("config path = %q", status.ConfigPath)
	}
	if len(status.Settings) != len(geminiSettingDefinitions) {
		t.Fatalf("settings = %d, want %d", len(status.Settings), len(geminiSettingDefinitions))
	}
	if settingStatus(status.Settings, "privacy.usageStatisticsEnabled") != statusAttention {
		t.Fatalf("usage statistics should need explicit hardening")
	}
	if settingStatus(status.Settings, "telemetry.enabled") != statusImplicit {
		t.Fatalf("telemetry.enabled should be default-safe")
	}
	if settingStatus(status.Settings, "tools.exclude.web") != statusAttention {
		t.Fatalf("web tools should need explicit hardening")
	}
}

func TestGeminiStatusReadsJSONCAndEvaluatesUnsafeValues(t *testing.T) {
	content := []byte(`{
  // Gemini CLI accepts comments in settings.json.
  "privacy": { "usageStatisticsEnabled": true },
  "telemetry": {
    "enabled": true,
    "traces": true,
    "logPrompts": true
  },
  "tools": { "exclude": ["write_file"] },
  "advanced": { "ignoreLocalEnv": false }
}`)

	status := buildGeminiStatus(filepath.Join("gemini", "settings.json"), true, content, nil)

	for _, id := range []string{
		"privacy.usageStatisticsEnabled",
		"telemetry.enabled",
		"telemetry.traces",
		"telemetry.logPrompts",
		"tools.exclude.web",
		"advanced.ignoreLocalEnv",
	} {
		if got := settingStatus(status.Settings, id); got != statusAttention {
			t.Fatalf("%s status = %q, want attention", id, got)
		}
	}
}

func TestGeminiApplyCreatesSettingsAndMergesWebToolExcludes(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), ".gemini", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	original := []byte(`{
  // keep existing exclusions
  "tools": { "exclude": ["write_file"] },
  "privacy": { "usageStatisticsEnabled": true }
}`)
	if err := os.WriteFile(configPath, original, 0o600); err != nil {
		t.Fatal(err)
	}

	adapter := NewGeminiAdapter()
	adapter.ConfigPath = configPath
	adapter.Now = func() time.Time {
		return time.Date(2026, 6, 28, 1, 2, 3, 4, time.UTC)
	}
	result, err := adapter.Apply([]string{"privacy.usageStatisticsEnabled", "tools.exclude.web"})
	if err != nil {
		t.Fatal(err)
	}
	wantBackup := filepath.Join(filepath.Dir(configPath), "settings.json.20260628T010203.000000004Z.bak")
	if result.BackupPath != wantBackup {
		t.Fatalf("backup path = %q, want %q", result.BackupPath, wantBackup)
	}
	if len(result.Changed) != 2 {
		t.Fatalf("changed = %d, want 2: %#v", len(result.Changed), result.Changed)
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var saved map[string]any
	if err := json.Unmarshal(content, &saved); err != nil {
		t.Fatalf("updated settings should be valid JSON: %v\n%s", err, content)
	}
	privacy := saved["privacy"].(map[string]any)
	if privacy["usageStatisticsEnabled"] != false {
		t.Fatalf("usage stats should be disabled: %#v", privacy)
	}
	tools := saved["tools"].(map[string]any)
	exclude := tools["exclude"].([]any)
	for _, want := range []string{"write_file", "google_web_search", "web_fetch"} {
		if !jsonArrayContainsString(exclude, want) {
			t.Fatalf("tools.exclude missing %q: %#v", want, exclude)
		}
	}
	if settingStatus(result.Status.Settings, "privacy.usageStatisticsEnabled") != statusHardened {
		t.Fatalf("usage statistics should be hardened after apply")
	}
	if settingStatus(result.Status.Settings, "tools.exclude.web") != statusHardened {
		t.Fatalf("web tools should be hardened after apply")
	}
}

func TestGeminiApplyChangesSetsCustomValue(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), ".gemini", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(`{"tools":{"exclude":["write_file"]}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	adapter := NewGeminiAdapter()
	adapter.ConfigPath = configPath
	result, err := adapter.ApplyChanges([]model.PrivacyConfigEdit{
		{ID: "tools.exclude.web", Op: "set", Value: []any{"google_web_search"}},
	})
	if err != nil {
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
	exclude := saved["tools"].(map[string]any)["exclude"].([]any)
	if len(exclude) != 1 || exclude[0] != "google_web_search" {
		t.Fatalf("tools.exclude should be replaced by custom value: %#v", exclude)
	}
	setting := findSetting(result.Status.Settings, "tools.exclude.web")
	if setting == nil {
		t.Fatal("tools.exclude.web setting missing")
	}
	if setting.ValueType != "stringArray" || !setting.Configured || !setting.SupportsUnset {
		t.Fatalf("setting metadata = %#v", setting)
	}
}

func TestGeminiStatusStrictArrayPreservesExistingExcludes(t *testing.T) {
	content := []byte(`{
  "tools": { "exclude": ["write_file"] }
}`)

	status := buildGeminiStatus(filepath.Join("gemini", "settings.json"), true, content, nil)
	setting := findSetting(status.Settings, "tools.exclude.web")
	if setting == nil {
		t.Fatal("tools.exclude.web setting missing")
	}
	strict, ok := setting.StrictValue.([]any)
	if !ok {
		t.Fatalf("strict value = %#v", setting.StrictValue)
	}
	for _, want := range []string{"write_file", "google_web_search", "web_fetch"} {
		if !jsonArrayContainsString(strict, want) {
			t.Fatalf("strict value missing %q: %#v", want, strict)
		}
	}
}

func TestGeminiApplyChangesUnsetsNestedKey(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), ".gemini", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	original := []byte(`{
  "general": {
    "sessionRetention": {
      "enabled": true,
      "maxAge": "14d"
    }
  }
}`)
	if err := os.WriteFile(configPath, original, 0o644); err != nil {
		t.Fatal(err)
	}

	adapter := NewGeminiAdapter()
	adapter.ConfigPath = configPath
	result, err := adapter.ApplyChanges([]model.PrivacyConfigEdit{
		{ID: "general.sessionRetention.maxAge", Op: "unset"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Changed) != 1 {
		t.Fatalf("changed = %d, want 1: %#v", len(result.Changed), result.Changed)
	}
	if result.Changed[0].Before != "14d" || result.Changed[0].After != nil {
		t.Fatalf("changed = %#v", result.Changed[0])
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var saved map[string]any
	if err := json.Unmarshal(content, &saved); err != nil {
		t.Fatal(err)
	}
	retention := saved["general"].(map[string]any)["sessionRetention"].(map[string]any)
	if _, ok := retention["maxAge"]; ok {
		t.Fatalf("maxAge should be removed: %#v", retention)
	}
	if retention["enabled"] != true {
		t.Fatalf("sibling nested value should be preserved: %#v", retention)
	}
	setting := findSetting(result.Status.Settings, "general.sessionRetention.maxAge")
	if setting == nil || setting.Configured {
		t.Fatalf("maxAge should be unconfigured after unset: %#v", setting)
	}
}

func TestGeminiApplyRejectsInvalidSettings(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), ".gemini", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(`{"privacy":`), 0o644); err != nil {
		t.Fatal(err)
	}

	adapter := NewGeminiAdapter()
	adapter.ConfigPath = configPath
	_, err := adapter.Apply([]string{"privacy.usageStatisticsEnabled"})
	if err == nil {
		t.Fatal("invalid settings should not be overwritten")
	}
	content, readErr := os.ReadFile(configPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !strings.Contains(string(content), `{"privacy":`) {
		t.Fatalf("invalid file should be preserved, got %q", content)
	}
}

func jsonArrayContainsString(values []any, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
