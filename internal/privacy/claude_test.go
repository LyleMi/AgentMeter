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

func TestClaudeStatusMissingConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), ".claude", "settings.json")
	t.Setenv("AGENTMETER_CLAUDE_SETTINGS_PATH", configPath)

	status, err := NewClaudeAdapter().Status()
	if err != nil {
		t.Fatal(err)
	}
	if status.Exists {
		t.Fatal("missing config should report exists=false")
	}
	if status.Target != "claude" || status.Name != "Claude Code" {
		t.Fatalf("target = %q, name = %q", status.Target, status.Name)
	}
	if status.ConfigPath != configPath {
		t.Fatalf("config path = %q", status.ConfigPath)
	}
	if len(status.Settings) != len(claudeSettingDefinitions) {
		t.Fatalf("settings = %d, want %d", len(status.Settings), len(claudeSettingDefinitions))
	}
	if status.Summary.Total != len(claudeSettingDefinitions) {
		t.Fatalf("summary total = %d", status.Summary.Total)
	}
	if settingStatus(status.Settings, "env.DISABLE_TELEMETRY") != statusAttention {
		t.Fatalf("telemetry should need explicit hardening")
	}
	if settingStatus(status.Settings, "permissions.deny") != statusAttention {
		t.Fatalf("permissions deny should need explicit hardening")
	}
}

func TestClaudeSettingsPathResolvesClaudeConfigDir(t *testing.T) {
	configDir := filepath.Join(t.TempDir(), "claude-config")
	t.Setenv("AGENTMETER_CLAUDE_SETTINGS_PATH", "")
	t.Setenv("CLAUDE_CONFIG_DIR", configDir)

	status, err := NewClaudeAdapter().Status()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(configDir, "settings.json"); status.ConfigPath != want {
		t.Fatalf("config path = %q, want %q", status.ConfigPath, want)
	}
}

func TestClaudeSettingsPathAgentMeterOverrideWins(t *testing.T) {
	dir := t.TempDir()
	overridePath := filepath.Join(dir, "override", "settings.json")
	configDir := filepath.Join(dir, "claude-config")
	t.Setenv("AGENTMETER_CLAUDE_SETTINGS_PATH", overridePath)
	t.Setenv("CLAUDE_CONFIG_DIR", configDir)

	status, err := NewClaudeAdapter().Status()
	if err != nil {
		t.Fatal(err)
	}
	if status.ConfigPath != overridePath {
		t.Fatalf("config path = %q, want %q", status.ConfigPath, overridePath)
	}
}

func TestClaudeStatusExistingConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	content := []byte(`{
  "env": { "DISABLE_TELEMETRY": "1" },
  "permissions": { "deny": ["Bash"] },
  "attribution": { "sessionUrl": true }
}`)
	if err := os.WriteFile(configPath, content, 0o644); err != nil {
		t.Fatal(err)
	}

	adapter := NewClaudeAdapter()
	adapter.ConfigPath = configPath
	status, err := adapter.Status()
	if err != nil {
		t.Fatal(err)
	}
	if !status.Exists {
		t.Fatal("existing config should report exists=true")
	}
	if settingStatus(status.Settings, "env.DISABLE_TELEMETRY") != statusHardened {
		t.Fatalf("telemetry should be hardened")
	}
	if settingStatus(status.Settings, "attribution.sessionUrl") != statusAttention {
		t.Fatalf("session URL attribution should need hardening")
	}
	setting := findSetting(status.Settings, "permissions.deny")
	if setting == nil {
		t.Fatal("permissions.deny setting missing")
	}
	strict, ok := setting.StrictValue.([]any)
	if !ok {
		t.Fatalf("strict value = %#v", setting.StrictValue)
	}
	for _, want := range []string{"Bash", "WebFetch"} {
		if !jsonArrayContainsString(strict, want) {
			t.Fatalf("strict value missing %q: %#v", want, strict)
		}
	}
}

func TestClaudeApplyEmptyCreatesSettings(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), ".claude", "settings.json")
	adapter := NewClaudeAdapter()
	adapter.ConfigPath = configPath

	result, err := adapter.Apply(nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.BackupPath != "" {
		t.Fatalf("new config should not create backup, got %q", result.BackupPath)
	}
	if len(result.Changed) != len(claudeSettingDefinitions) {
		t.Fatalf("changed = %d, want %d", len(result.Changed), len(claudeSettingDefinitions))
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
	if _, ok := env["CLAUDE_CODE_SKIP_PROMPT_HISTORY"]; ok {
		t.Fatalf("prompt history skip should not be set: %#v", env)
	}
	if _, ok := saved["skipWebFetchPreflight"]; ok {
		t.Fatalf("WebFetch preflight skipping should not be applied by default: %#v", saved)
	}
	permissions := saved["permissions"].(map[string]any)
	deny := permissions["deny"].([]any)
	if !jsonArrayContainsString(deny, "WebFetch") {
		t.Fatalf("permissions.deny missing WebFetch: %#v", deny)
	}
}

func TestClaudeApplySelectedMergesDenyAndBacksUp(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	original := []byte(`{
  "permissions": {
    "allow": ["Bash"],
    "deny": ["Read"]
  },
  "env": { "DISABLE_TELEMETRY": "0" }
}`)
	if err := os.WriteFile(configPath, original, 0o600); err != nil {
		t.Fatal(err)
	}

	adapter := NewClaudeAdapter()
	adapter.ConfigPath = configPath
	adapter.Now = func() time.Time {
		return time.Date(2026, 6, 28, 1, 2, 3, 4, time.UTC)
	}
	result, err := adapter.Apply([]string{"permissions.deny", "env.DISABLE_TELEMETRY"})
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
	allow := permissions["allow"].([]any)
	if !jsonArrayContainsString(allow, "Bash") {
		t.Fatalf("permissions.allow should be preserved: %#v", permissions)
	}
	deny := permissions["deny"].([]any)
	for _, want := range []string{"Read", "WebFetch"} {
		if !jsonArrayContainsString(deny, want) {
			t.Fatalf("permissions.deny missing %q: %#v", want, deny)
		}
	}
	env := saved["env"].(map[string]any)
	if env["DISABLE_TELEMETRY"] != "1" {
		t.Fatalf("telemetry env should be hardened: %#v", env)
	}
}

func TestClaudeApplyChangesSetsAndUnsetsValues(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	original := []byte(`{
  "attribution": {
    "commit": "Generated by Claude",
    "sessionUrl": true
  },
  "disableArtifact": false
}`)
	if err := os.WriteFile(configPath, original, 0o644); err != nil {
		t.Fatal(err)
	}

	adapter := NewClaudeAdapter()
	adapter.ConfigPath = configPath
	result, err := adapter.ApplyChanges([]model.PrivacyConfigEdit{
		{ID: "disableArtifact", Op: "set", Value: true},
		{ID: "attribution.commit", Op: "unset"},
		{ID: "permissions.deny", Op: "set", Value: []any{"Bash"}},
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
	if saved["disableArtifact"] != true {
		t.Fatalf("disableArtifact should be true: %#v", saved)
	}
	attribution := saved["attribution"].(map[string]any)
	if _, ok := attribution["commit"]; ok {
		t.Fatalf("commit attribution should be removed: %#v", attribution)
	}
	if attribution["sessionUrl"] != true {
		t.Fatalf("sibling attribution value should be preserved: %#v", attribution)
	}
	deny := saved["permissions"].(map[string]any)["deny"].([]any)
	if len(deny) != 1 || deny[0] != "Bash" {
		t.Fatalf("permissions.deny should be replaced by custom value: %#v", deny)
	}
}

func TestClaudeApplyChangesRejectsInvalidType(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	original := []byte(`{"disableArtifact": false}`)
	if err := os.WriteFile(configPath, original, 0o644); err != nil {
		t.Fatal(err)
	}

	adapter := NewClaudeAdapter()
	adapter.ConfigPath = configPath
	_, err := adapter.ApplyChanges([]model.PrivacyConfigEdit{
		{ID: "disableArtifact", Op: "set", Value: "true"},
	})
	if err == nil {
		t.Fatal("invalid bool edit should be rejected")
	}
	if !strings.Contains(err.Error(), "requires a bool value") {
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

func TestClaudeInvalidJSONStatusAndApplyPreservesFile(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	invalid := []byte(`{"env":`)
	if err := os.WriteFile(configPath, invalid, 0o644); err != nil {
		t.Fatal(err)
	}

	adapter := NewClaudeAdapter()
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
