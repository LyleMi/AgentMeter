package privacy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/LyleMi/AgentMeter/internal/model"
)

func TestCodeBuddyStatusMissingConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), ".codebuddy", "settings.json")
	t.Setenv("AGENTMETER_CODEBUDDY_SETTINGS_PATH", configPath)

	status, err := NewCodeBuddyAdapter().Status()
	if err != nil {
		t.Fatal(err)
	}
	if status.Exists {
		t.Fatal("missing config should report exists=false")
	}
	if status.Target != "codebuddy" || status.Name != "CodeBuddy Code/IDE" {
		t.Fatalf("target = %q, name = %q", status.Target, status.Name)
	}
	if status.ConfigPath != configPath {
		t.Fatalf("config path = %q", status.ConfigPath)
	}
	if len(status.Settings) != len(codeBuddySettingDefinitions) {
		t.Fatalf("settings = %d, want %d", len(status.Settings), len(codeBuddySettingDefinitions))
	}
	if settingStatus(status.Settings, "env.DISABLE_TELEMETRY") != statusAttention {
		t.Fatalf("telemetry should need explicit hardening")
	}
	if settingStatus(status.Settings, "env.OTEL_LOG_USER_PROMPTS") != statusImplicit {
		t.Fatalf("prompt recording should be default-safe")
	}
	if settingStatus(status.Settings, "cleanupPeriodDays") != statusAttention {
		t.Fatalf("retention should need explicit hardening")
	}
	if settingStatus(status.Settings, "permissions.defaultMode") != statusImplicit {
		t.Fatalf("default permission mode should be default-safe")
	}
}

func TestCodeBuddySettingsPathResolvesConfigDir(t *testing.T) {
	configDir := filepath.Join(t.TempDir(), "codebuddy-config")
	t.Setenv("AGENTMETER_CODEBUDDY_SETTINGS_PATH", "")
	t.Setenv("CODEBUDDY_CONFIG_DIR", configDir)

	status, err := NewCodeBuddyAdapter().Status()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(configDir, "settings.json"); status.ConfigPath != want {
		t.Fatalf("config path = %q, want %q", status.ConfigPath, want)
	}
}

func TestCodeBuddySettingsPathAgentMeterOverrideWins(t *testing.T) {
	dir := t.TempDir()
	overridePath := filepath.Join(dir, "override", "settings.json")
	configDir := filepath.Join(dir, "codebuddy-config")
	t.Setenv("AGENTMETER_CODEBUDDY_SETTINGS_PATH", overridePath)
	t.Setenv("CODEBUDDY_CONFIG_DIR", configDir)

	status, err := NewCodeBuddyAdapter().Status()
	if err != nil {
		t.Fatal(err)
	}
	if status.ConfigPath != overridePath {
		t.Fatalf("config path = %q, want %q", status.ConfigPath, overridePath)
	}
}

func TestCodeBuddyStatusReadsJSONCAndEvaluatesUnsafeValues(t *testing.T) {
	content := []byte(`{
  // CodeBuddy accepts settings.json-style JSONC through AgentMeter's shared parser.
  "env": {
    "DISABLE_TELEMETRY": "0",
    "OTEL_LOG_TOOL_CONTENT": "1"
  },
  "cleanupPeriodDays": 30,
  "memory": { "autoMemoryEnabled": true },
  "permissions": {
    "defaultMode": "bypassPermissions",
    "deny": ["Read(./.env)"]
  },
  "trustAll": true,
  "enableAllProjectMcpServers": true
}`)

	status := buildCodeBuddyStatus(filepath.Join("codebuddy", "settings.json"), true, content, nil)
	for _, id := range []string{
		"env.DISABLE_TELEMETRY",
		"env.OTEL_LOG_TOOL_CONTENT",
		"cleanupPeriodDays",
		"memory.autoMemoryEnabled",
		"permissions.defaultMode",
		"permissions.deny",
		"trustAll",
		"enableAllProjectMcpServers",
	} {
		if got := settingStatus(status.Settings, id); got != statusAttention {
			t.Fatalf("%s status = %q, want attention", id, got)
		}
	}
}

func TestCodeBuddyApplyEmptyCreatesSettings(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), ".codebuddy", "settings.json")
	adapter := NewCodeBuddyAdapter()
	adapter.ConfigPath = configPath

	result, err := adapter.Apply(nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.BackupPath != "" {
		t.Fatalf("new config should not create backup, got %q", result.BackupPath)
	}
	if len(result.Changed) != len(codeBuddySettingDefinitions) {
		t.Fatalf("changed = %d, want %d", len(result.Changed), len(codeBuddySettingDefinitions))
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var saved map[string]any
	if err := json.Unmarshal(content, &saved); err != nil {
		t.Fatalf("updated settings should be valid JSON: %v\n%s", err, content)
	}
	env := saved["env"].(map[string]any)
	if env["DISABLE_TELEMETRY"] != "1" {
		t.Fatalf("telemetry env should be set: %#v", env)
	}
	if saved["cleanupPeriodDays"] != float64(7) {
		t.Fatalf("cleanupPeriodDays should be numeric 7: %#v", saved["cleanupPeriodDays"])
	}
	permissions := saved["permissions"].(map[string]any)
	if permissions["defaultMode"] != "default" {
		t.Fatalf("default mode should be default: %#v", permissions)
	}
	deny := permissions["deny"].([]any)
	if !jsonArrayContainsString(deny, "WebFetch") || !jsonArrayContainsString(deny, "Read(~/.ssh/**)") {
		t.Fatalf("permissions.deny missing strict values: %#v", deny)
	}
	memory := saved["memory"].(map[string]any)
	if memory["autoMemoryEnabled"] != false {
		t.Fatalf("auto memory should be disabled: %#v", memory)
	}
}

func TestCodeBuddyApplySelectedMergesDenyAndBacksUp(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), ".codebuddy", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	original := []byte(`{
  "permissions": {
    "allow": ["Read"],
    "deny": ["Read(./.env)"]
  },
  "env": { "DISABLE_TELEMETRY": "0" },
  "cleanupPeriodDays": 30
}`)
	if err := os.WriteFile(configPath, original, 0o600); err != nil {
		t.Fatal(err)
	}

	adapter := NewCodeBuddyAdapter()
	adapter.ConfigPath = configPath
	adapter.Now = func() time.Time {
		return time.Date(2026, 6, 28, 1, 2, 3, 4, time.UTC)
	}
	result, err := adapter.Apply([]string{"permissions.deny", "env.DISABLE_TELEMETRY", "cleanupPeriodDays"})
	if err != nil {
		t.Fatal(err)
	}
	wantBackup := filepath.Join(filepath.Dir(configPath), "settings.json.20260628T010203.000000004Z.bak")
	if result.BackupPath != wantBackup {
		t.Fatalf("backup path = %q, want %q", result.BackupPath, wantBackup)
	}
	backup, err := os.ReadFile(result.BackupPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(backup) != string(original) {
		t.Fatalf("backup = %q, want original %q", backup, original)
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var saved map[string]any
	if err := json.Unmarshal(content, &saved); err != nil {
		t.Fatalf("updated settings should be valid JSON: %v\n%s", err, content)
	}
	permissions := saved["permissions"].(map[string]any)
	if !jsonArrayContainsString(permissions["allow"].([]any), "Read") {
		t.Fatalf("permissions.allow should be preserved: %#v", permissions)
	}
	deny := permissions["deny"].([]any)
	for _, want := range []string{"Read(./.env)", "WebFetch", "WebSearch", "Read(~/.aws/**)"} {
		if !jsonArrayContainsString(deny, want) {
			t.Fatalf("permissions.deny missing %q: %#v", want, deny)
		}
	}
	env := saved["env"].(map[string]any)
	if env["DISABLE_TELEMETRY"] != "1" {
		t.Fatalf("telemetry env should be hardened: %#v", env)
	}
	if saved["cleanupPeriodDays"] != float64(7) {
		t.Fatalf("cleanupPeriodDays should be hardened: %#v", saved["cleanupPeriodDays"])
	}
}

func TestCodeBuddyApplyProfileRecommendedSetsTelemetryReportingAndUnsetsLocalControls(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), ".codebuddy", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	original := []byte(`{
  "env": {
    "DISABLE_TELEMETRY": "0",
    "DISABLE_ERROR_REPORTING": "0",
    "DISABLE_AUTOUPDATER": "1"
  },
  "autoUpdates": false,
  "cleanupPeriodDays": 7,
  "memory": { "autoMemoryEnabled": false },
  "permissions": { "deny": ["WebFetch"] }
}`)
	if err := os.WriteFile(configPath, original, 0o644); err != nil {
		t.Fatal(err)
	}

	adapter := NewCodeBuddyAdapter()
	adapter.ConfigPath = configPath
	result, err := adapter.ApplyProfile("recommended")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Changed) == 0 {
		t.Fatal("recommended profile should change explicit telemetry/network/local settings")
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
	wantEnv := map[string]any{
		"DISABLE_TELEMETRY":               "1",
		"CODEBUDDY_CODE_ENABLE_TELEMETRY": "0",
		"CLAUDE_CODE_ENABLE_TELEMETRY":    "0",
		"OTEL_TRACES_EXPORTER":            "none",
		"OTEL_LOG_USER_PROMPTS":           "0",
		"OTEL_LOG_TOOL_DETAILS":           "0",
		"OTEL_LOG_TOOL_CONTENT":           "0",
		"OTEL_LOG_RAW_API_BODIES":         "0",
		"DISABLE_ERROR_REPORTING":         "1",
		"DISABLE_FEEDBACK_COMMAND":        "1",
	}
	for key, want := range wantEnv {
		if env[key] != want {
			t.Fatalf("recommended profile should set env.%s=%#v: %#v", key, want, saved)
		}
	}
	if _, ok := env["DISABLE_AUTOUPDATER"]; ok {
		t.Fatalf("recommended profile should leave network controls unset/default: %#v", saved)
	}
	for _, key := range []string{"autoUpdates", "cleanupPeriodDays"} {
		if _, ok := saved[key]; ok {
			t.Fatalf("recommended profile should leave %s unset/default: %#v", key, saved)
		}
	}
	if _, ok := saved["memory"].(map[string]any)["autoMemoryEnabled"]; ok {
		t.Fatalf("recommended profile should leave memory unset/default: %#v", saved)
	}
	if _, ok := saved["permissions"].(map[string]any)["deny"]; ok {
		t.Fatalf("recommended profile should leave permissions unset/default: %#v", saved)
	}
}

func TestCodeBuddyStatusStrictArrayPreservesExistingDenyRules(t *testing.T) {
	content := []byte(`{
  "permissions": { "deny": ["Bash(rm:*)"] }
}`)

	status := buildCodeBuddyStatus(filepath.Join("codebuddy", "settings.json"), true, content, nil)
	setting := findSetting(status.Settings, "permissions.deny")
	if setting == nil {
		t.Fatal("permissions.deny setting missing")
	}
	strict, ok := setting.StrictValue.([]any)
	if !ok {
		t.Fatalf("strict value = %#v", setting.StrictValue)
	}
	for _, want := range []string{"Bash(rm:*)", "WebFetch", "WebSearch"} {
		if !jsonArrayContainsString(strict, want) {
			t.Fatalf("strict value missing %q: %#v", want, strict)
		}
	}
}

func TestCodeBuddyApplyChangesSetsNumberAndUnsetsValues(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), ".codebuddy", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	original := []byte(`{
  "cleanupPeriodDays": 30,
  "trustAll": true,
  "includeCoAuthoredBy": true
}`)
	if err := os.WriteFile(configPath, original, 0o644); err != nil {
		t.Fatal(err)
	}

	adapter := NewCodeBuddyAdapter()
	adapter.ConfigPath = configPath
	result, err := adapter.ApplyChanges([]model.PrivacyConfigEdit{
		{ID: "cleanupPeriodDays", Op: "set", Value: float64(14)},
		{ID: "trustAll", Op: "set", Value: false},
		{ID: "includeCoAuthoredBy", Op: "unset"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Changed) != 3 {
		t.Fatalf("changed = %d, want 3: %#v", len(result.Changed), result.Changed)
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var saved map[string]any
	if err := json.Unmarshal(content, &saved); err != nil {
		t.Fatal(err)
	}
	if saved["cleanupPeriodDays"] != float64(14) {
		t.Fatalf("cleanupPeriodDays should be 14: %#v", saved)
	}
	if saved["trustAll"] != false {
		t.Fatalf("trustAll should be false: %#v", saved)
	}
	if _, ok := saved["includeCoAuthoredBy"]; ok {
		t.Fatalf("includeCoAuthoredBy should be removed: %#v", saved)
	}
}

func TestCodeBuddyApplyChangesRejectsInvalidNumber(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), ".codebuddy", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	original := []byte(`{"cleanupPeriodDays": 30}`)
	if err := os.WriteFile(configPath, original, 0o644); err != nil {
		t.Fatal(err)
	}

	adapter := NewCodeBuddyAdapter()
	adapter.ConfigPath = configPath
	_, err := adapter.ApplyChanges([]model.PrivacyConfigEdit{
		{ID: "cleanupPeriodDays", Op: "set", Value: "7"},
	})
	if err == nil {
		t.Fatal("invalid number edit should be rejected")
	}
	if !strings.Contains(err.Error(), "requires a number value") {
		t.Fatalf("error = %v", err)
	}
	content, readErr := os.ReadFile(configPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(content) != string(original) {
		t.Fatalf("invalid edit should preserve file, got %q", content)
	}
}

func TestCodeBuddyInvalidJSONStatusAndApplyPreservesFile(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), ".codebuddy", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	invalid := []byte(`{"env":`)
	if err := os.WriteFile(configPath, invalid, 0o644); err != nil {
		t.Fatal(err)
	}

	adapter := NewCodeBuddyAdapter()
	adapter.ConfigPath = configPath
	status, err := adapter.Status()
	if err != nil {
		t.Fatal(err)
	}
	if len(status.Warnings) != 1 || !strings.Contains(status.Warnings[0], "could not be parsed") {
		t.Fatalf("warnings = %#v", status.Warnings)
	}
	setting := findSetting(status.Settings, "env.DISABLE_TELEMETRY")
	if setting == nil {
		t.Fatal("telemetry setting missing")
	}
	if setting.CanApply || setting.SupportsUnset {
		t.Fatalf("invalid JSON setting should be non-applicable: %#v", setting)
	}

	_, err = adapter.Apply([]string{"env.DISABLE_TELEMETRY"})
	if err == nil {
		t.Fatal("invalid settings should not be overwritten")
	}
	content, readErr := os.ReadFile(configPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(content) != string(invalid) {
		t.Fatalf("invalid file should be preserved, got %q", content)
	}
}
