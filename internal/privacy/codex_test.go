package privacy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"AgentMeter/internal/model"
)

func TestCodexStatusMissingConfig(t *testing.T) {
	codexHome := filepath.Join(t.TempDir(), "codex-home")
	t.Setenv("CODEX_HOME", codexHome)

	status, err := NewCodexAdapter().Status()
	if err != nil {
		t.Fatal(err)
	}
	if status.Exists {
		t.Fatal("missing config should report exists=false")
	}
	if status.ConfigPath != filepath.Join(codexHome, "config.toml") {
		t.Fatalf("config path = %q", status.ConfigPath)
	}
	if len(status.Settings) != len(codexSettingDefinitions) {
		t.Fatalf("settings = %d, want %d", len(status.Settings), len(codexSettingDefinitions))
	}
	if settingStatus(status.Settings, "analytics.enabled") != statusAttention {
		t.Fatalf("analytics should need explicit hardening")
	}
	if settingStatus(status.Settings, "history.persistence") != statusAttention {
		t.Fatalf("history persistence should need explicit hardening")
	}
	if settingStatus(status.Settings, "web_search") != statusAttention {
		t.Fatalf("web search should need explicit hardening")
	}
	if status.Summary.Total != len(codexSettingDefinitions) {
		t.Fatalf("summary total = %d", status.Summary.Total)
	}
}

func TestCodexStatusEvaluatesUnsafeExplicitValues(t *testing.T) {
	content := []byte(`
web_search = true

[analytics]
enabled = true

[otel]
exporter = "otlp"
metrics_exporter = "statsig"
log_user_prompt = true

[history]
persistence = "save-all"

[shell_environment_policy]
inherit = "all"
ignore_default_excludes = true
`)

	status := buildCodexStatus(filepath.Join("codex", "config.toml"), true, content, nil)

	for _, id := range []string{
		"web_search",
		"analytics.enabled",
		"otel.exporter",
		"otel.metrics_exporter",
		"otel.log_user_prompt",
		"history.persistence",
		"shell_environment_policy.inherit",
		"shell_environment_policy.ignore_default_excludes",
	} {
		if got := settingStatus(status.Settings, id); got != statusAttention {
			t.Fatalf("%s status = %q, want attention", id, got)
		}
	}
}

func TestCodexApplyCreatesConfigInTempCodexHome(t *testing.T) {
	codexHome := filepath.Join(t.TempDir(), "codex-home")
	t.Setenv("CODEX_HOME", codexHome)

	result, err := NewCodexAdapter().Apply([]string{"analytics.enabled", "history.persistence"})
	if err != nil {
		t.Fatal(err)
	}
	if result.BackupPath != "" {
		t.Fatalf("new config should not create backup, got %q", result.BackupPath)
	}
	if len(result.Changed) != 2 {
		t.Fatalf("changed = %d, want 2: %#v", len(result.Changed), result.Changed)
	}
	content, err := os.ReadFile(filepath.Join(codexHome, "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	for _, want := range []string{
		"[analytics]",
		"enabled = false",
		"[history]",
		`persistence = "none"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("config does not contain %q:\n%s", want, text)
		}
	}
	if settingStatus(result.Status.Settings, "analytics.enabled") != statusHardened {
		t.Fatalf("analytics should be hardened after apply")
	}
	if settingStatus(result.Status.Settings, "history.persistence") != statusHardened {
		t.Fatalf("history should be hardened after apply")
	}
}

func TestCodexApplyWritesWebSearchDisabledString(t *testing.T) {
	codexHome := filepath.Join(t.TempDir(), "codex-home")
	t.Setenv("CODEX_HOME", codexHome)

	result, err := NewCodexAdapter().Apply([]string{"web_search"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Changed) != 1 {
		t.Fatalf("changed = %d, want 1: %#v", len(result.Changed), result.Changed)
	}
	content, err := os.ReadFile(filepath.Join(codexHome, "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), `web_search = "disabled"`) {
		t.Fatalf("web_search should use documented string enum:\n%s", content)
	}
	if settingStatus(result.Status.Settings, "web_search") != statusHardened {
		t.Fatalf("web search should be hardened after apply")
	}
}

func TestCodexApplyCreatesBackupForExistingConfig(t *testing.T) {
	codexHome := t.TempDir()
	t.Setenv("CODEX_HOME", codexHome)
	configPath := filepath.Join(codexHome, "config.toml")
	original := []byte("[analytics]\nenabled = true\n")
	if err := os.WriteFile(configPath, original, 0o600); err != nil {
		t.Fatal(err)
	}

	adapter := NewCodexAdapter()
	adapter.Now = func() time.Time {
		return time.Date(2026, 6, 28, 1, 2, 3, 4, time.UTC)
	}
	result, err := adapter.Apply([]string{"analytics.enabled"})
	if err != nil {
		t.Fatal(err)
	}
	wantBackup := filepath.Join(codexHome, "config.toml.20260628T010203.000000004Z.bak")
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
	updated, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updated), "enabled = false") {
		t.Fatalf("updated config did not harden analytics:\n%s", updated)
	}
}

func TestCodexApplyChangesSetsCustomValue(t *testing.T) {
	codexHome := t.TempDir()
	t.Setenv("CODEX_HOME", codexHome)
	configPath := filepath.Join(codexHome, "config.toml")
	if err := os.WriteFile(configPath, []byte("[analytics]\nenabled = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := NewCodexAdapter().ApplyChanges([]model.PrivacyConfigEdit{
		{ID: "analytics.enabled", Op: "set", Value: false},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Changed) != 1 {
		t.Fatalf("changed = %d, want 1: %#v", len(result.Changed), result.Changed)
	}
	if result.Changed[0].Before != true || result.Changed[0].After != false {
		t.Fatalf("changed = %#v", result.Changed[0])
	}
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "enabled = false") {
		t.Fatalf("custom value was not written:\n%s", content)
	}
	setting := findSetting(result.Status.Settings, "analytics.enabled")
	if setting == nil {
		t.Fatal("analytics setting missing")
	}
	if setting.ValueType != "bool" || !setting.Configured || !setting.SupportsUnset || setting.StrictValue != false {
		t.Fatalf("setting metadata = %#v", setting)
	}
}

func TestCodexApplyChangesUnsetsValue(t *testing.T) {
	codexHome := t.TempDir()
	t.Setenv("CODEX_HOME", codexHome)
	configPath := filepath.Join(codexHome, "config.toml")
	original := strings.Join([]string{
		"# existing config",
		"web_search = \"disabled\" # remove only this line",
		`model = "gpt-5"`,
		"",
		"[analytics]",
		"enabled = false",
		"",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := NewCodexAdapter().ApplyChanges([]model.PrivacyConfigEdit{
		{ID: "web_search", Op: "unset"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Changed) != 1 {
		t.Fatalf("changed = %d, want 1: %#v", len(result.Changed), result.Changed)
	}
	if result.Changed[0].Before != "disabled" || result.Changed[0].After != nil {
		t.Fatalf("changed = %#v", result.Changed[0])
	}
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	if strings.Contains(text, "web_search") {
		t.Fatalf("web_search should be removed:\n%s", text)
	}
	for _, want := range []string{"# existing config", `model = "gpt-5"`, "[analytics]", "enabled = false"} {
		if !strings.Contains(text, want) {
			t.Fatalf("updated config lost %q:\n%s", want, text)
		}
	}
	setting := findSetting(result.Status.Settings, "web_search")
	if setting == nil || setting.Configured {
		t.Fatalf("web_search should be unconfigured after unset: %#v", setting)
	}
}

func TestCodexApplyPreservesUnrelatedConfigLines(t *testing.T) {
	codexHome := t.TempDir()
	t.Setenv("CODEX_HOME", codexHome)
	configPath := filepath.Join(codexHome, "config.toml")
	original := strings.Join([]string{
		"# existing config",
		`model = "gpt-5"`,
		"",
		"[model_providers.openai]",
		`base_url = "https://example.test/v1"`,
		"",
		"[analytics]",
		"# keep this comment",
		`other = "kept"`,
		"enabled = true # inline comment",
		"",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := NewCodexAdapter().Apply([]string{"analytics.enabled"})
	if err != nil {
		t.Fatal(err)
	}
	updatedBytes, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	updated := string(updatedBytes)
	for _, want := range []string{
		"# existing config",
		`model = "gpt-5"`,
		"[model_providers.openai]",
		`base_url = "https://example.test/v1"`,
		"# keep this comment",
		`other = "kept"`,
		"enabled = false # inline comment",
	} {
		if !strings.Contains(updated, want) {
			t.Fatalf("updated config lost %q:\n%s", want, updated)
		}
	}
}

func settingStatus(settings []model.PrivacyConfigSetting, id string) string {
	for _, setting := range settings {
		if setting.ID == id {
			return setting.Status
		}
	}
	return ""
}

func findSetting(settings []model.PrivacyConfigSetting, id string) *model.PrivacyConfigSetting {
	for index := range settings {
		if settings[index].ID == id {
			return &settings[index]
		}
	}
	return nil
}
