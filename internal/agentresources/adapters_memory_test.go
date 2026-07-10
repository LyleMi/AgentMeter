package agentresources

import (
	"path/filepath"
	"testing"

	"github.com/LyleMi/AgentMeter/internal/model"
)

func TestResolveGenericMemoryPathRules(t *testing.T) {
	root := t.TempDir()
	tests := []struct {
		name      string
		agentKind string
		rel       string
		wantRoot  string
		wantKind  string
	}{
		{name: "gemini primary", agentKind: "gemini", rel: "GEMINI.md", wantRoot: root, wantKind: "primary"},
		{name: "claude command", agentKind: "claude", rel: "commands/lint.md", wantRoot: filepath.Join(root, "commands"), wantKind: "command"},
		{name: "codebuddy subagent", agentKind: "codebuddy", rel: "agents/review.md", wantRoot: filepath.Join(root, "agents"), wantKind: "subagent"},
		{name: "workbuddy primary", agentKind: "workbuddy", rel: "WORKBUDDY.md", wantRoot: root, wantKind: "primary"},
		{name: "cursor rule", agentKind: "cursor", rel: "rules/style.mdc", wantRoot: filepath.Join(root, "rules"), wantKind: "rule"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := model.AgentResourceAgent{Kind: tt.agentKind, Name: tt.agentKind, RootPath: root}
			location, err := resolveGenericMemoryPath(agent, "", tt.rel)
			if err != nil {
				t.Fatal(err)
			}
			if location.root != tt.wantRoot || location.kind != tt.wantKind {
				t.Fatalf("location = %+v, want root %q kind %q", location, tt.wantRoot, tt.wantKind)
			}
		})
	}
}

func TestResolveGenericMemoryPathRejectsUnsupportedLocations(t *testing.T) {
	root := t.TempDir()
	tests := []struct {
		agentKind string
		rel       string
	}{
		{agentKind: "gemini", rel: "commands/lint.md"},
		{agentKind: "claude", rel: "commands/lint.txt"},
		{agentKind: "cursor", rel: "rules/style.txt"},
		{agentKind: "workbuddy", rel: "other/note.md"},
	}

	for _, tt := range tests {
		t.Run(tt.agentKind+"/"+tt.rel, func(t *testing.T) {
			agent := model.AgentResourceAgent{Kind: tt.agentKind, Name: tt.agentKind, RootPath: root}
			if _, err := resolveGenericMemoryPath(agent, "", tt.rel); err == nil {
				t.Fatal("expected unsupported location error")
			}
		})
	}
}
