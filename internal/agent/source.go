package agent

import (
	"os"
	"path/filepath"
	"strings"

	"AgentMeter/internal/sourcepath"
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
}

func ResolveSource(path string) SourceSpec {
	cleaned := sourcepath.Normalize(path)
	if cleaned == "" {
		cleaned = filepath.Clean(path)
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
		if !sourcepath.Equal(spec.RootPath, spec.SessionsPath) {
			return []UsageSource{{Dir: spec.SessionsPath, DedupeScope: spec.SessionsPath}}
		}
		sessions := filepath.Join(spec.RootPath, "sessions")
		archived := filepath.Join(spec.RootPath, "archived_sessions")
		sources := make([]UsageSource, 0, 2)
		if isDir(sessions) {
			sources = append(sources, UsageSource{Dir: sessions, DedupeScope: spec.RootPath})
		}
		if isDir(archived) {
			sources = append(sources, UsageSource{Dir: archived, DedupeScope: spec.RootPath})
		}
		if len(sources) > 0 {
			return sources
		}
	case "claude":
		if sourcepath.Equal(spec.RootPath, spec.SessionsPath) {
			projects := filepath.Join(spec.RootPath, "projects")
			if isDir(projects) {
				return []UsageSource{{Dir: projects, DedupeScope: spec.RootPath}}
			}
		}
	case "codebuddy":
		if sourcepath.Equal(spec.RootPath, spec.SessionsPath) {
			projects := filepath.Join(spec.RootPath, "projects")
			if isDir(projects) {
				return []UsageSource{{Dir: projects, DedupeScope: spec.RootPath}}
			}
			sessions := filepath.Join(spec.RootPath, "sessions")
			if isDir(sessions) {
				return []UsageSource{{Dir: sessions, DedupeScope: spec.RootPath}}
			}
		}
	case "workbuddy":
		if sourcepath.Equal(spec.RootPath, spec.SessionsPath) ||
			strings.EqualFold(filepath.Base(spec.SessionsPath), "sessions") {
			projects := filepath.Join(spec.RootPath, "projects")
			if isDir(projects) {
				return []UsageSource{{Dir: projects, DedupeScope: spec.RootPath}}
			}
			sessions := filepath.Join(spec.RootPath, "sessions")
			if isDir(sessions) {
				return []UsageSource{{Dir: sessions, DedupeScope: spec.RootPath}}
			}
		}
	}
	return []UsageSource{{Dir: spec.SessionsPath, DedupeScope: spec.SessionsPath}}
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
