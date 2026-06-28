package cli

import (
	"bytes"
	"strings"
	"testing"

	"AgentMeter/internal/model"
)

type fakePrivacyRegistry struct {
	targets      []string
	statuses     map[string]model.PrivacyConfigStatus
	profileCalls []fakeProfileCall
	applyCalls   []fakeApplyCall
	editCalls    []fakeEditCall
}

type fakeProfileCall struct {
	target  string
	profile string
}

type fakeApplyCall struct {
	target string
	ids    []string
}

type fakeEditCall struct {
	target string
	edits  []model.PrivacyConfigEdit
}

func newFakePrivacyRegistry() *fakePrivacyRegistry {
	statuses := map[string]model.PrivacyConfigStatus{
		"codex": {
			Target:     "codex",
			Name:       "Codex",
			ConfigPath: "codex.toml",
			Exists:     true,
			Summary:    model.PrivacyConfigSummary{Score: 50, Total: 2, Hardened: 1, Attention: 1},
			Settings: []model.PrivacyConfigSetting{{
				ID:           "web_search",
				Status:       "attention",
				Configured:   true,
				CurrentValue: "enabled",
				StrictValue:  "disabled",
			}},
		},
		"gemini": {
			Target:     "gemini",
			Name:       "Gemini CLI",
			ConfigPath: "gemini.json",
			Summary:    model.PrivacyConfigSummary{Score: 100, Total: 1, Hardened: 1},
		},
	}
	return &fakePrivacyRegistry{
		targets:  []string{"codex", "gemini"},
		statuses: statuses,
	}
}

func (r *fakePrivacyRegistry) Targets() []string {
	return append([]string(nil), r.targets...)
}

func (r *fakePrivacyRegistry) Status(target string) (model.PrivacyConfigStatus, error) {
	return r.statuses[target], nil
}

func (r *fakePrivacyRegistry) Apply(target string, settingIDs []string) (model.PrivacyConfigApplyResult, error) {
	r.applyCalls = append(r.applyCalls, fakeApplyCall{target: target, ids: append([]string(nil), settingIDs...)})
	return model.PrivacyConfigApplyResult{
		Status: r.statuses[target],
		Changed: []model.PrivacyConfigChange{{
			ID:     "selected",
			Before: nil,
			After:  len(settingIDs),
		}},
	}, nil
}

func (r *fakePrivacyRegistry) ApplyChanges(target string, changes []model.PrivacyConfigEdit) (model.PrivacyConfigApplyResult, error) {
	r.editCalls = append(r.editCalls, fakeEditCall{target: target, edits: append([]model.PrivacyConfigEdit(nil), changes...)})
	return model.PrivacyConfigApplyResult{
		Status: r.statuses[target],
		Changed: []model.PrivacyConfigChange{{
			ID:     changes[0].ID,
			Before: "old",
			After:  changes[0].Value,
		}},
	}, nil
}

func (r *fakePrivacyRegistry) ApplyProfile(target, profile string) (model.PrivacyConfigApplyResult, error) {
	r.profileCalls = append(r.profileCalls, fakeProfileCall{target: target, profile: profile})
	return model.PrivacyConfigApplyResult{
		Status: r.statuses[target],
		Changed: []model.PrivacyConfigChange{{
			ID:     "profile",
			Before: "old",
			After:  profile,
		}},
		BackupPath: target + ".bak",
	}, nil
}

func TestPrivacyApplyDefaultsToRecommendedProfile(t *testing.T) {
	registry := newFakePrivacyRegistry()
	var stdout, stderr bytes.Buffer

	code := runWithPrivacyRegistry([]string{"privacy", "apply", "codex"}, &stdout, &stderr, registry)

	if code != ExitOK {
		t.Fatalf("exit code = %d, stderr=%s", code, stderr.String())
	}
	if len(registry.profileCalls) != 1 || registry.profileCalls[0].target != "codex" || registry.profileCalls[0].profile != "recommended" {
		t.Fatalf("profile calls = %#v", registry.profileCalls)
	}
	if !strings.Contains(stdout.String(), "Applied recommended profile") {
		t.Fatalf("stdout did not describe profile apply:\n%s", stdout.String())
	}
}

func TestPrivacyApplyAllUsesProfileForEveryTarget(t *testing.T) {
	registry := newFakePrivacyRegistry()
	var stdout, stderr bytes.Buffer

	code := runWithPrivacyRegistry([]string{"privacy", "apply", "all", "strict"}, &stdout, &stderr, registry)

	if code != ExitOK {
		t.Fatalf("exit code = %d, stderr=%s", code, stderr.String())
	}
	if len(registry.profileCalls) != 2 {
		t.Fatalf("profile calls = %#v", registry.profileCalls)
	}
	if registry.profileCalls[0].target != "codex" || registry.profileCalls[0].profile != "strict" {
		t.Fatalf("first profile call = %#v", registry.profileCalls[0])
	}
	if registry.profileCalls[1].target != "gemini" || registry.profileCalls[1].profile != "strict" {
		t.Fatalf("second profile call = %#v", registry.profileCalls[1])
	}
}

func TestPrivacyApplySettingIDs(t *testing.T) {
	registry := newFakePrivacyRegistry()
	var stdout, stderr bytes.Buffer

	code := runWithPrivacyRegistry([]string{"privacy", "apply", "codex", "analytics.enabled", "web_search"}, &stdout, &stderr, registry)

	if code != ExitOK {
		t.Fatalf("exit code = %d, stderr=%s", code, stderr.String())
	}
	if len(registry.applyCalls) != 1 {
		t.Fatalf("apply calls = %#v", registry.applyCalls)
	}
	got := registry.applyCalls[0]
	if got.target != "codex" || len(got.ids) != 2 || got.ids[0] != "analytics.enabled" || got.ids[1] != "web_search" {
		t.Fatalf("apply call = %#v", got)
	}
}

func TestPrivacySetParsesJSONValues(t *testing.T) {
	registry := newFakePrivacyRegistry()
	var stdout, stderr bytes.Buffer

	code := runWithPrivacyRegistry([]string{"privacy", "set", "codex", "analytics.enabled", "false"}, &stdout, &stderr, registry)

	if code != ExitOK {
		t.Fatalf("exit code = %d, stderr=%s", code, stderr.String())
	}
	if len(registry.editCalls) != 1 || len(registry.editCalls[0].edits) != 1 {
		t.Fatalf("edit calls = %#v", registry.editCalls)
	}
	edit := registry.editCalls[0].edits[0]
	value, ok := edit.Value.(bool)
	if edit.ID != "analytics.enabled" || edit.Op != "set" || !ok || value {
		t.Fatalf("edit = %#v", edit)
	}
}

func TestPrivacyUnsetBuildsUnsetEdit(t *testing.T) {
	registry := newFakePrivacyRegistry()
	var stdout, stderr bytes.Buffer

	code := runWithPrivacyRegistry([]string{"privacy", "unset", "gemini", "tools.exclude.web"}, &stdout, &stderr, registry)

	if code != ExitOK {
		t.Fatalf("exit code = %d, stderr=%s", code, stderr.String())
	}
	if len(registry.editCalls) != 1 || len(registry.editCalls[0].edits) != 1 {
		t.Fatalf("edit calls = %#v", registry.editCalls)
	}
	edit := registry.editCalls[0].edits[0]
	if registry.editCalls[0].target != "gemini" || edit.ID != "tools.exclude.web" || edit.Op != "unset" {
		t.Fatalf("edit call = %#v", registry.editCalls[0])
	}
}

func TestPrivacySettingsPrintsSettingIDs(t *testing.T) {
	registry := newFakePrivacyRegistry()
	var stdout, stderr bytes.Buffer

	code := runWithPrivacyRegistry([]string{"privacy", "settings", "codex"}, &stdout, &stderr, registry)

	if code != ExitOK {
		t.Fatalf("exit code = %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "web_search") || !strings.Contains(stdout.String(), "current=\"enabled\"") {
		t.Fatalf("stdout missing setting details:\n%s", stdout.String())
	}
}
