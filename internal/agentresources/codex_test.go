package agentresources

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LyleMi/AgentMeter/internal/model"
)

func TestOverviewReturnsWarningWhenCodexHomeMissing(t *testing.T) {
	dir := t.TempDir()
	isolateAgentHomes(t, dir)
	root := filepath.Join(dir, "missing-codex")
	t.Setenv("CODEX_HOME", root)

	overview, err := Overview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	agent := findAgent(t, overview, "codex")
	if agent.Exists {
		t.Fatalf("agent status = %+v", overview.Agents)
	}
	if len(overview.Warnings) == 0 {
		t.Fatalf("expected warning for missing home: %+v", overview)
	}
	if len(overview.Skills) != 0 || len(overview.MCPServers) != 0 || len(overview.Memories) != 0 {
		t.Fatalf("missing home should return empty resources: %+v", overview)
	}
}

func TestOverviewDiscoversCodexResources(t *testing.T) {
	dir := t.TempDir()
	isolateAgentHomes(t, dir)
	root := filepath.Join(dir, ".codex")
	t.Setenv("CODEX_HOME", root)
	writeFile(t, filepath.Join(root, "skills", "writer", "SKILL.md"), `---
name: writer
description: Writes concise local docs.
---

# Writer Skill

Body.
`)
	writeFile(t, filepath.Join(root, "skills", ".system", "imagegen", "SKILL.md"), `---
name: imagegen
description: Generates images.
---

# Imagegen
`)
	writeFile(t, filepath.Join(root, "config.toml"), `
[mcp_servers.node_repl]
command = 'node-repl.exe'
args = ['--stdio']

[mcp_servers.node_repl.env]
SECRET_TOKEN = 'do-not-return'
VISIBLE_NAME = 'node'
`)
	writeFile(t, filepath.Join(root, "memories", "MEMORY.md"), "# Memory\n\nKeep responses direct.\n")
	writeFile(t, filepath.Join(root, "memories", ".git", "ignored.md"), "# ignored\n")

	overview, err := Overview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(overview.Skills) != 2 {
		t.Fatalf("skills = %+v", overview.Skills)
	}
	if overview.Skills[0].Name != "writer" || overview.Skills[0].System || !overview.Skills[0].Enabled {
		t.Fatalf("user skill should sort before system skill: %+v", overview.Skills)
	}
	if !overview.Skills[0].CanToggle || overview.Skills[1].CanToggle {
		t.Fatalf("skill toggle capability = %+v", overview.Skills)
	}
	if overview.Skills[0].Description != "Writes concise local docs." || overview.Skills[0].Title != "Writer Skill" {
		t.Fatalf("skill metadata = %+v", overview.Skills[0])
	}
	if len(overview.MCPServers) != 1 {
		t.Fatalf("mcp servers = %+v", overview.MCPServers)
	}
	server := overview.MCPServers[0]
	if server.Name != "node_repl" || server.Command != "node-repl.exe" || !server.Enabled {
		t.Fatalf("server = %+v", server)
	}
	if !server.CanToggle {
		t.Fatalf("server should be toggleable: %+v", server)
	}
	if len(server.EnvKeys) != 2 || server.EnvKeys[0] != "SECRET_TOKEN" || server.EnvKeys[1] != "VISIBLE_NAME" {
		t.Fatalf("env keys should be names only and sorted: %+v", server.EnvKeys)
	}
	if len(overview.Memories) != 1 {
		t.Fatalf("memories = %+v", overview.Memories)
	}
	if overview.Memories[0].Kind != "primary" || overview.Memories[0].Preview != "Keep responses direct." {
		t.Fatalf("memory = %+v", overview.Memories[0])
	}
	if !overview.Memories[0].CanEdit {
		t.Fatalf("memory should be editable: %+v", overview.Memories[0])
	}
	if len(overview.Agents) < 4 {
		t.Fatalf("expected known non-Codex agents to be listed: %+v", overview.Agents)
	}
	claude := findAgent(t, overview, "claude")
	if len(claude.Unsupported) == 0 || len(claude.Warnings) == 0 {
		t.Fatalf("unsupported non-Codex agent should explain empty resources: %+v", claude)
	}
}

func TestSetSkillEnabledRenamesCodexSkillFile(t *testing.T) {
	dir := t.TempDir()
	isolateAgentHomes(t, dir)
	root := filepath.Join(dir, ".codex")
	t.Setenv("CODEX_HOME", root)
	writeFile(t, filepath.Join(root, "skills", "writer", "SKILL.md"), "---\nname: writer\n---\n# Writer\n")

	result, err := SetSkillEnabled(context.Background(), modelToggle("writer", "writer", false))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "skills", "writer", "SKILL.md.disabled")); err != nil {
		t.Fatal(err)
	}
	if skill := findSkill(t, result.Overview, "writer"); skill.Enabled {
		t.Fatalf("skill should be disabled: %+v", skill)
	}

	result, err = SetSkillEnabled(context.Background(), modelToggle("writer", "writer", true))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "skills", "writer", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	if skill := findSkill(t, result.Overview, "writer"); !skill.Enabled {
		t.Fatalf("skill should be enabled: %+v", skill)
	}
}

func TestSetSkillEnabledRejectsNonCodexAgentKind(t *testing.T) {
	dir := t.TempDir()
	isolateAgentHomes(t, dir)
	root := filepath.Join(dir, ".codex")
	t.Setenv("CODEX_HOME", root)
	writeFile(t, filepath.Join(root, "skills", "writer", "SKILL.md"), "---\nname: writer\n---\n# Writer\n")

	_, err := SetSkillEnabled(context.Background(), model.AgentResourceToggleRequest{
		AgentKind:    "gemini",
		RelativePath: "writer",
		Enabled:      false,
	})
	if err == nil {
		t.Fatal("expected non-Codex skill toggle to be rejected")
	}
	if _, err := os.Stat(filepath.Join(root, "skills", "writer", "SKILL.md")); err != nil {
		t.Fatalf("Codex skill should not be changed after rejected non-Codex toggle: %v", err)
	}
}

func TestSetSkillEnabledRejectsPathOutsideSkillsRoot(t *testing.T) {
	dir := t.TempDir()
	isolateAgentHomes(t, dir)
	root := filepath.Join(dir, ".codex")
	t.Setenv("CODEX_HOME", root)
	if err := os.MkdirAll(filepath.Join(root, "skills"), 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := SetSkillEnabled(context.Background(), modelToggle("bad", "../outside", false))
	if err == nil {
		t.Fatal("expected outside path to be rejected")
	}
}

func TestSetMCPServerEnabledWritesCodexEnabledFlag(t *testing.T) {
	dir := t.TempDir()
	isolateAgentHomes(t, dir)
	root := filepath.Join(dir, ".codex")
	t.Setenv("CODEX_HOME", root)
	writeFile(t, filepath.Join(root, "config.toml"), `
[mcp_servers.node_repl]
command = 'node'
args = ['server.js']
`)

	result, err := SetMCPServerEnabled(context.Background(), modelToggle("node_repl", "", false))
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(root, "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "enabled = false") {
		t.Fatalf("config should contain disabled MCP flag:\n%s", content)
	}
	if server := findMCP(t, result.Overview, "node_repl"); server.Enabled {
		t.Fatalf("server should be disabled: %+v", server)
	}
}

func TestMemoryDetailAndUpdateValidateCodexMemoryRoot(t *testing.T) {
	dir := t.TempDir()
	isolateAgentHomes(t, dir)
	root := filepath.Join(dir, ".codex")
	t.Setenv("CODEX_HOME", root)
	writeFile(t, filepath.Join(root, "memories", "MEMORY.md"), "# Memory\n\nInitial.\n")

	detail, err := MemoryDetail(context.Background(), "codex", "", "MEMORY.md")
	if err != nil {
		t.Fatal(err)
	}
	if detail.Content != "# Memory\n\nInitial.\n" {
		t.Fatalf("content = %q", detail.Content)
	}
	updated, err := UpdateMemory(context.Background(), modelMemoryUpdate("MEMORY.md", "# Memory\n\nUpdated.\n"))
	if err != nil {
		t.Fatal(err)
	}
	if updated.Preview != "Updated." {
		t.Fatalf("updated memory = %+v", updated)
	}
	_, err = MemoryDetail(context.Background(), "codex", "", "../config.toml")
	if err == nil {
		t.Fatal("expected traversal to be rejected")
	}
}

func isolateAgentHomes(t *testing.T, dir string) {
	t.Helper()
	home := filepath.Join(dir, "home")
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("AGENTMETER_GEMINI_SETTINGS_PATH", filepath.Join(dir, ".gemini", "settings.json"))
	t.Setenv("AGENTMETER_CLAUDE_SETTINGS_PATH", filepath.Join(dir, ".claude", "settings.json"))
	t.Setenv("AGENTMETER_CODEBUDDY_SETTINGS_PATH", filepath.Join(dir, ".codebuddy", "settings.json"))
	t.Setenv("WORKBUDDY_CONFIG_DIR", filepath.Join(dir, ".workbuddy"))
	t.Setenv("CURSOR_HOME", filepath.Join(dir, ".cursor"))
}

func findAgent(t *testing.T, overview model.AgentResourceOverview, kind string) model.AgentResourceAgent {
	t.Helper()
	for _, agent := range overview.Agents {
		if agent.Kind == kind {
			return agent
		}
	}
	t.Fatalf("agent %q missing: %+v", kind, overview.Agents)
	return model.AgentResourceAgent{}
}

func findSkill(t *testing.T, overview model.AgentResourceOverview, name string) model.AgentSkillResource {
	t.Helper()
	for _, skill := range overview.Skills {
		if skill.Name == name {
			return skill
		}
	}
	t.Fatalf("skill %q missing: %+v", name, overview.Skills)
	return model.AgentSkillResource{}
}

func findMCP(t *testing.T, overview model.AgentResourceOverview, name string) model.AgentMCPServerResource {
	t.Helper()
	for _, server := range overview.MCPServers {
		if server.Name == name {
			return server
		}
	}
	t.Fatalf("mcp server %q missing: %+v", name, overview.MCPServers)
	return model.AgentMCPServerResource{}
}

func modelToggle(name, rel string, enabled bool) model.AgentResourceToggleRequest {
	return model.AgentResourceToggleRequest{AgentKind: "codex", Name: name, RelativePath: rel, Enabled: enabled}
}

func modelMemoryUpdate(rel, content string) model.AgentMemoryUpdateRequest {
	return model.AgentMemoryUpdateRequest{AgentKind: "codex", RelativePath: rel, Content: content}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
