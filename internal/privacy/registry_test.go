package privacy

import (
	"errors"
	"testing"

	"AgentMeter/internal/model"
)

type registryFakeAdapter struct {
	target       string
	statusCalls  int
	applyCalls   int
	changesCalls int
}

func (a *registryFakeAdapter) Status() (model.PrivacyConfigStatus, error) {
	a.statusCalls++
	return model.PrivacyConfigStatus{Target: a.target}, nil
}

func (a *registryFakeAdapter) Apply(settingIDs []string) (model.PrivacyConfigApplyResult, error) {
	a.applyCalls++
	return model.PrivacyConfigApplyResult{
		Status: model.PrivacyConfigStatus{Target: a.target},
		Changed: []model.PrivacyConfigChange{{
			ID:    "selected",
			After: len(settingIDs),
		}},
	}, nil
}

func (a *registryFakeAdapter) ApplyChanges(changes []model.PrivacyConfigEdit) (model.PrivacyConfigApplyResult, error) {
	a.changesCalls++
	return model.PrivacyConfigApplyResult{
		Status:  model.PrivacyConfigStatus{Target: a.target},
		Changed: make([]model.PrivacyConfigChange, 0, len(changes)),
	}, nil
}

func TestRegistryDispatchesByTarget(t *testing.T) {
	gemini := &registryFakeAdapter{target: "gemini"}
	claude := &registryFakeAdapter{target: "claude"}
	registry := NewRegistry(map[string]AdapterFactory{
		"gemini": func() Adapter { return gemini },
		"claude": func() Adapter { return claude },
	}, []string{"claude", "gemini"})

	statuses, err := registry.Statuses()
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 2 || statuses[0].Target != "claude" || statuses[1].Target != "gemini" {
		t.Fatalf("statuses = %#v", statuses)
	}

	result, err := registry.Apply("gemini", []string{"privacy.usageStatisticsEnabled"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status.Target != "gemini" || gemini.applyCalls != 1 || claude.applyCalls != 0 {
		t.Fatalf("registry did not dispatch to gemini: result=%#v gemini=%d claude=%d", result, gemini.applyCalls, claude.applyCalls)
	}
}

func TestRegistryUnsupportedTarget(t *testing.T) {
	registry := NewRegistry(map[string]AdapterFactory{
		"codex": func() Adapter { return &registryFakeAdapter{target: "codex"} },
	}, []string{"codex"})

	_, err := registry.Status("unknown")
	if err == nil {
		t.Fatal("expected unsupported target error")
	}
	var targetErr UnsupportedTargetError
	if !errors.As(err, &targetErr) || targetErr.Target != "unknown" {
		t.Fatalf("err = %T %v", err, err)
	}
	if !IsUnsupportedTarget(err) {
		t.Fatalf("IsUnsupportedTarget(%v) = false", err)
	}
}
