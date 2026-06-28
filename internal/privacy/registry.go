package privacy

import (
	"errors"
	"fmt"

	"AgentMeter/internal/model"
)

type Adapter interface {
	Status() (model.PrivacyConfigStatus, error)
	Apply([]string) (model.PrivacyConfigApplyResult, error)
	ApplyChanges([]model.PrivacyConfigEdit) (model.PrivacyConfigApplyResult, error)
	ApplyProfile(string) (model.PrivacyConfigApplyResult, error)
}

type AdapterFactory func() Adapter

type Registry struct {
	order    []string
	adapters map[string]AdapterFactory
}

type UnsupportedTargetError struct {
	Target string
}

func (e UnsupportedTargetError) Error() string {
	return fmt.Sprintf("unsupported privacy target: %s", e.Target)
}

func DefaultRegistry() Registry {
	return NewRegistry(map[string]AdapterFactory{
		"codex":     func() Adapter { return NewCodexAdapter() },
		"gemini":    func() Adapter { return NewGeminiAdapter() },
		"claude":    func() Adapter { return NewClaudeAdapter() },
		"codebuddy": func() Adapter { return NewCodeBuddyAdapter() },
	}, []string{"codex", "gemini", "claude", "codebuddy"})
}

func NewRegistry(adapters map[string]AdapterFactory, order []string) Registry {
	copied := make(map[string]AdapterFactory, len(adapters))
	for target, factory := range adapters {
		copied[normalizeTarget(target)] = factory
	}
	return Registry{
		order:    append([]string(nil), order...),
		adapters: copied,
	}
}

func (r Registry) Targets() []string {
	targets := make([]string, 0, len(r.adapters))
	seen := map[string]struct{}{}
	for _, target := range r.order {
		target = normalizeTarget(target)
		if _, ok := r.adapters[target]; !ok {
			continue
		}
		targets = append(targets, target)
		seen[target] = struct{}{}
	}
	for target := range r.adapters {
		if _, ok := seen[target]; !ok {
			targets = append(targets, target)
		}
	}
	return targets
}

func (r Registry) Adapter(target string) (Adapter, error) {
	normalized := normalizeTarget(target)
	factory, ok := r.adapters[normalized]
	if !ok || factory == nil {
		return nil, UnsupportedTargetError{Target: target}
	}
	return factory(), nil
}

func (r Registry) Supports(target string) bool {
	factory, ok := r.adapters[normalizeTarget(target)]
	return ok && factory != nil
}

func (r Registry) Status(target string) (model.PrivacyConfigStatus, error) {
	adapter, err := r.Adapter(target)
	if err != nil {
		return model.PrivacyConfigStatus{}, err
	}
	return adapter.Status()
}

func (r Registry) Statuses() ([]model.PrivacyConfigStatus, error) {
	targets := r.Targets()
	statuses := make([]model.PrivacyConfigStatus, 0, len(targets))
	for _, target := range targets {
		status, err := r.Status(target)
		if err != nil {
			return nil, err
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}

func (r Registry) Apply(target string, settingIDs []string) (model.PrivacyConfigApplyResult, error) {
	adapter, err := r.Adapter(target)
	if err != nil {
		return model.PrivacyConfigApplyResult{}, err
	}
	return adapter.Apply(settingIDs)
}

func (r Registry) ApplyChanges(target string, changes []model.PrivacyConfigEdit) (model.PrivacyConfigApplyResult, error) {
	adapter, err := r.Adapter(target)
	if err != nil {
		return model.PrivacyConfigApplyResult{}, err
	}
	return adapter.ApplyChanges(changes)
}

func (r Registry) ApplyProfile(target, profile string) (model.PrivacyConfigApplyResult, error) {
	adapter, err := r.Adapter(target)
	if err != nil {
		return model.PrivacyConfigApplyResult{}, err
	}
	return adapter.ApplyProfile(profile)
}

func IsUnsupportedTarget(err error) bool {
	var targetErr UnsupportedTargetError
	return errors.As(err, &targetErr)
}

func normalizeTarget(target string) string {
	return target
}
