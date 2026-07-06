package agent

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/LyleMi/AgentMeter/internal/sourcepath"
)

type SourceSpec struct {
	Kind         string
	Name         string
	RootPath     string
	SessionsPath string
}

type UsageSource struct {
	Dir         string
	DedupeScope string
}

type familyRule struct {
	Kind      string
	Name      string
	ExactRoot string
	Token     string
	Children  []string
}

var familyRules = []familyRule{
	{Kind: "codebuddy", Name: "CodeBuddy", ExactRoot: ".codebuddy", Token: "codebuddy", Children: []string{"projects", "sessions"}},
	{Kind: "workbuddy", Name: "WorkBuddy", ExactRoot: ".workbuddy", Token: "workbuddy", Children: []string{"projects", "sessions"}},
	{Kind: "claude", Name: "Claude Code", ExactRoot: ".claude", Token: "claude", Children: []string{"projects"}},
	{Kind: "codex", Name: "Codex", ExactRoot: ".codex", Token: "codex", Children: []string{"sessions", "archived_sessions"}},
	{Kind: "cursor", Name: "Cursor", ExactRoot: ".cursor", Token: "cursor", Children: []string{"projects", "User"}},
}

func ResolveSource(path string) SourceSpec {
	cleaned := sourcepath.Normalize(path)
	if cleaned == "" {
		cleaned = filepath.Clean(path)
	}
	if spec, ok := matchingCursorNestedPath(cleaned); ok {
		return spec
	}
	if rule, ok := matchingChildRule(cleaned); ok {
		root := filepath.Dir(cleaned)
		return SourceSpec{Kind: rule.Kind, Name: displayName(rule, root), RootPath: root, SessionsPath: cleaned}
	}
	if rule, ok := matchingRootRule(cleaned); ok {
		return SourceSpec{Kind: rule.Kind, Name: displayName(rule, cleaned), RootPath: cleaned, SessionsPath: cleaned}
	}
	return SourceSpec{Kind: "jsonl", Name: "Generic JSONL", RootPath: cleaned, SessionsPath: cleaned}
}

func UsageSources(spec SourceSpec) []UsageSource {
	switch spec.Kind {
	case "codex":
		if sources, ok := codexUsageSources(spec); ok {
			return sources
		}
	case "claude":
		return rootChildUsageSources(spec, "projects")
	case "codebuddy":
		return rootChildUsageSources(spec, "projects", "sessions")
	case "workbuddy":
		if workbuddyCanUseRootChildren(spec) {
			return firstExistingRootChildren(spec, "projects", "sessions")
		}
	case "cursor":
		if sources, ok := cursorUsageSources(spec); ok {
			return sources
		}
	}
	return fallbackUsageSources(spec)
}

func codexUsageSources(spec SourceSpec) ([]UsageSource, bool) {
	if !sourcepath.Equal(spec.RootPath, spec.SessionsPath) {
		return fallbackUsageSources(spec), true
	}
	sources := existingRootChildSources(spec, "sessions", "archived_sessions")
	return sources, len(sources) > 0
}

func rootChildUsageSources(spec SourceSpec, children ...string) []UsageSource {
	if sourcepath.Equal(spec.RootPath, spec.SessionsPath) {
		return firstExistingRootChildren(spec, children...)
	}
	return fallbackUsageSources(spec)
}

func firstExistingRootChildren(spec SourceSpec, children ...string) []UsageSource {
	for _, child := range children {
		if source, ok := existingRootChildSource(spec, child); ok {
			return []UsageSource{source}
		}
	}
	return fallbackUsageSources(spec)
}

func existingRootChildSources(spec SourceSpec, children ...string) []UsageSource {
	sources := make([]UsageSource, 0, len(children))
	for _, child := range children {
		if source, ok := existingRootChildSource(spec, child); ok {
			sources = append(sources, source)
		}
	}
	return sources
}

func existingRootChildSource(spec SourceSpec, child string) (UsageSource, bool) {
	return existingUsageSource(filepath.Join(spec.RootPath, child), spec.RootPath)
}

func existingUsageSource(dir string, dedupeScope string) (UsageSource, bool) {
	if !isDir(dir) {
		return UsageSource{}, false
	}
	return UsageSource{Dir: dir, DedupeScope: dedupeScope}, true
}

func workbuddyCanUseRootChildren(spec SourceSpec) bool {
	return sourcepath.Equal(spec.RootPath, spec.SessionsPath) ||
		strings.EqualFold(filepath.Base(spec.SessionsPath), "sessions")
}

func cursorUsageSources(spec SourceSpec) ([]UsageSource, bool) {
	if sourcepath.Equal(spec.RootPath, spec.SessionsPath) {
		if source, ok := existingRootChildSource(spec, "projects"); ok {
			return []UsageSource{source}, true
		}
		return cursorWorkspaceStorageUsageSource(filepath.Join(spec.RootPath, "User"), spec.RootPath)
	}
	if strings.EqualFold(filepath.Base(spec.SessionsPath), "User") {
		return cursorWorkspaceStorageUsageSource(spec.SessionsPath, spec.RootPath)
	}
	return nil, false
}

func cursorWorkspaceStorageUsageSource(userDir string, root string) ([]UsageSource, bool) {
	source, ok := existingUsageSource(filepath.Join(userDir, "workspaceStorage"), root)
	if !ok {
		return nil, false
	}
	return []UsageSource{source}, true
}

func fallbackUsageSources(spec SourceSpec) []UsageSource {
	return []UsageSource{{Dir: spec.SessionsPath, DedupeScope: spec.SessionsPath}}
}

func matchingCursorNestedPath(path string) (SourceSpec, bool) {
	if root, ok := cursorProjectsRoot(path); ok {
		return SourceSpec{Kind: "cursor", Name: cursorDisplayName(root), RootPath: root, SessionsPath: path}, true
	}
	if root, ok := cursorWorkspaceStorageRoot(path); ok {
		return SourceSpec{Kind: "cursor", Name: cursorDisplayName(root), RootPath: root, SessionsPath: path}, true
	}
	return SourceSpec{}, false
}

func cursorProjectsRoot(path string) (string, bool) {
	for current := filepath.Clean(path); ; current = filepath.Dir(current) {
		if strings.EqualFold(filepath.Base(current), "projects") {
			root := filepath.Dir(current)
			if cursorRootNameMatches(root) {
				return root, true
			}
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
	}
	return "", false
}

func cursorWorkspaceStorageRoot(path string) (string, bool) {
	for current := filepath.Clean(path); ; current = filepath.Dir(current) {
		base := filepath.Base(current)
		if strings.EqualFold(base, "workspaceStorage") || strings.EqualFold(base, "globalStorage") {
			userDir := filepath.Dir(current)
			if !strings.EqualFold(filepath.Base(userDir), "User") {
				continue
			}
			root := filepath.Dir(userDir)
			if cursorRootNameMatches(root) {
				return root, true
			}
			if strings.EqualFold(filepath.Base(root), "data") && cursorRootNameMatches(filepath.Dir(root)) {
				return filepath.Dir(root), true
			}
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
	}
	return "", false
}

func cursorRootNameMatches(path string) bool {
	return rootNameMatches(path, familyRule{ExactRoot: ".cursor", Token: "cursor"})
}

func cursorDisplayName(rootPath string) string {
	base := filepath.Base(rootPath)
	if strings.EqualFold(base, ".cursor") || strings.EqualFold(base, "Cursor") {
		return "Cursor"
	}
	return "Cursor (" + base + ")"
}

func matchingChildRule(path string) (familyRule, bool) {
	parent := filepath.Dir(path)
	base := strings.ToLower(filepath.Base(path))
	for _, rule := range familyRules {
		if !containsString(rule.Children, base) {
			continue
		}
		if rootNameMatches(parent, rule) {
			return rule, true
		}
	}
	return familyRule{}, false
}

func matchingRootRule(path string) (familyRule, bool) {
	for _, rule := range familyRules {
		if !rootNameMatches(path, rule) {
			continue
		}
		if hasFamilyStructure(path, rule) {
			return rule, true
		}
	}
	return familyRule{}, false
}

func rootNameMatches(path string, rule familyRule) bool {
	base := strings.ToLower(filepath.Base(path))
	trimmed := strings.TrimPrefix(base, ".")
	return base == rule.ExactRoot || strings.Contains(trimmed, rule.Token)
}

func hasFamilyStructure(path string, rule familyRule) bool {
	for _, child := range rule.Children {
		if isDir(filepath.Join(path, child)) {
			return true
		}
	}
	return false
}

func displayName(rule familyRule, rootPath string) string {
	if rule.Kind == "cursor" {
		return cursorDisplayName(rootPath)
	}
	base := filepath.Base(rootPath)
	if strings.EqualFold(base, rule.ExactRoot) {
		return rule.Name
	}
	return rule.Name + " (" + base + ")"
}

func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if strings.EqualFold(candidate, value) {
			return true
		}
	}
	return false
}

func isDir(path string) bool {
	stat, err := os.Stat(path)
	return err == nil && stat.IsDir()
}
