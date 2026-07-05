package agentresources

import (
	"context"
	"encoding/json"
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
	writeFile(t, filepath.Join(root, "memories", "extensions", "ad_hoc", "instructions.md"), "# Ad hoc instructions\n")
	writeFile(t, filepath.Join(root, "memories", "skills", "parallel-worker-orchestration", "SKILL.md"), "# Skill instructions\n")

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
	for _, memory := range overview.Memories {
		if strings.HasPrefix(memory.RelativePath, "extensions/") || strings.HasPrefix(memory.RelativePath, "skills/") {
			t.Fatalf("resource file should not be reported as memory: %+v", memory)
		}
	}
	if len(overview.Agents) < 4 {
		t.Fatalf("expected known non-Codex agents to be listed: %+v", overview.Agents)
	}
	claude := findAgent(t, overview, "claude")
	if len(claude.Unsupported) != 0 || len(claude.Supports) == 0 {
		t.Fatalf("non-Codex agent should expose supported inventory areas: %+v", claude)
	}
}

func TestOverviewDiscoversGeminiResourcesAndTogglesMCP(t *testing.T) {
	dir := t.TempDir()
	isolateAgentHomes(t, dir)
	root := filepath.Join(dir, ".gemini")
	t.Setenv("AGENTMETER_GEMINI_SETTINGS_PATH", filepath.Join(root, "settings.json"))
	writeFile(t, filepath.Join(root, "skills", "planner", "SKILL.md"), "---\nname: planner\ndescription: Plans work.\n---\n# Planner\n")
	writeFile(t, filepath.Join(root, "GEMINI.md"), "# Gemini\n\nPrefer local context.\n")
	writeFile(t, filepath.Join(root, "settings.json"), `{
  // JSONC is accepted.
  "mcpServers": {
    "filesystem": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem"],
      "env": { "TOKEN": "secret" }
    }
  },
  "mcp": {
    "allowed": [],
    "excluded": ["filesystem"]
  }
}`)

	overview, err := Overview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	gemini := findAgent(t, overview, "gemini")
	if !gemini.Exists {
		t.Fatalf("gemini should exist: %+v", gemini)
	}
	skill := findAgentSkill(t, overview, "gemini", "planner")
	if skill.ResourceType != "skill" || !skill.CanToggle {
		t.Fatalf("gemini skill = %+v", skill)
	}
	memory := findMemory(t, overview, "gemini", "GEMINI.md")
	if memory.Kind != "primary" || !memory.CanEdit || memory.Preview != "Prefer local context." {
		t.Fatalf("gemini memory = %+v", memory)
	}
	server := findAgentMCP(t, overview, "gemini", "filesystem")
	if server.Enabled || !server.CanToggle || server.Status != "disabled" {
		t.Fatalf("gemini server = %+v", server)
	}

	result, err := SetMCPServerEnabled(context.Background(), model.AgentResourceToggleRequest{
		AgentKind: "gemini",
		Name:      "filesystem",
		Enabled:   true,
	})
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(root, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var saved map[string]any
	if err := json.Unmarshal(content, &saved); err != nil {
		t.Fatal(err)
	}
	mcp := saved["mcp"].(map[string]any)
	if containsJSONListValue(mcp["excluded"], "filesystem") {
		t.Fatalf("filesystem should be removed from excluded: %s", content)
	}
	if !containsJSONListValue(mcp["allowed"], "filesystem") {
		t.Fatalf("filesystem should be added to allowed when allowed exists: %s", content)
	}
	if server := findAgentMCP(t, result.Overview, "gemini", "filesystem"); !server.Enabled {
		t.Fatalf("server should be enabled after toggle: %+v", server)
	}
}

func TestOverviewDiscoversClaudeCommandsSubagentsAndUserMCP(t *testing.T) {
	dir := t.TempDir()
	isolateAgentHomes(t, dir)
	home := filepath.Join(dir, "home")
	root := filepath.Join(dir, ".claude")
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("AGENTMETER_CLAUDE_SETTINGS_PATH", filepath.Join(root, "settings.json"))
	writeFile(t, filepath.Join(root, "skills", "review", "SKILL.md"), "---\nname: review\n---\n# Review\n")
	writeFile(t, filepath.Join(root, "commands", "lint.md"), "# Lint\n\nRun the lint command.\n")
	writeFile(t, filepath.Join(root, "agents", "reviewer.md"), "---\nname: reviewer\n---\n# Reviewer\n")
	writeFile(t, filepath.Join(root, "CLAUDE.md"), "# Claude\n\nUse repository instructions.\n")
	writeFile(t, filepath.Join(home, ".claude.json"), `{
  "mcpServers": {
    "notes": {
      "command": "node",
      "args": ["notes.js"],
      "enabled": false
    }
  }
}`)

	overview, err := Overview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := findAgentSkill(t, overview, "claude", "review"); got.ResourceType != "skill" {
		t.Fatalf("claude skill = %+v", got)
	}
	if got := findAgentSkill(t, overview, "claude", "lint"); got.ResourceType != "command" || got.CanToggle {
		t.Fatalf("claude command = %+v", got)
	}
	if got := findAgentSkill(t, overview, "claude", "reviewer"); got.ResourceType != "subagent" || got.CanToggle {
		t.Fatalf("claude subagent = %+v", got)
	}
	if server := findAgentMCP(t, overview, "claude", "notes"); server.Enabled || !server.CanToggle || server.ConfigPath != filepath.Join(home, ".claude.json") {
		t.Fatalf("claude MCP = %+v", server)
	}
	detail, err := MemoryDetail(context.Background(), "claude", "", "commands/lint.md")
	if err != nil {
		t.Fatal(err)
	}
	if detail.Kind != "command" || !strings.Contains(detail.Content, "lint command") {
		t.Fatalf("command detail = %+v", detail)
	}
	updated, err := UpdateMemory(context.Background(), model.AgentMemoryUpdateRequest{
		AgentKind:    "claude",
		RelativePath: "commands/lint.md",
		Content:      "# Lint\n\nUpdated.\n",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Preview != "Updated." {
		t.Fatalf("updated command = %+v", updated)
	}
}

func TestOverviewDiscoversCodeBuddyWorkBuddyAndCursorResources(t *testing.T) {
	dir := t.TempDir()
	isolateAgentHomes(t, dir)
	codebuddy := filepath.Join(dir, ".codebuddy")
	workbuddy := filepath.Join(dir, ".workbuddy")
	cursor := filepath.Join(dir, ".cursor")
	writeFile(t, filepath.Join(codebuddy, "settings.json"), `{"mcpServers":{"git":{"command":"git-mcp","disabled":true}}}`)
	writeFile(t, filepath.Join(codebuddy, "CODEBUDDY.md"), "# CodeBuddy\n\nCodeBuddy instructions.\n")
	writeFile(t, filepath.Join(workbuddy, "settings.json"), `{"mcpServers":{"shell":{"command":"shell-mcp","enabled":true}}}`)
	writeFile(t, filepath.Join(workbuddy, "skills", "ops", "SKILL.md.disabled"), "---\nname: ops\n---\n# Ops\n")
	writeFile(t, filepath.Join(cursor, "mcp.json"), `{"mcpServers":{"sqlite":{"command":"sqlite-mcp"}}}`)
	writeFile(t, filepath.Join(cursor, "rules", "style.mdc"), "# Style\n\nUse project style.\n")

	overview, err := Overview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if server := findAgentMCP(t, overview, "codebuddy", "git"); server.Enabled || !server.CanToggle {
		t.Fatalf("codebuddy MCP = %+v", server)
	}
	if server := findAgentMCP(t, overview, "workbuddy", "shell"); !server.Enabled || !server.CanToggle {
		t.Fatalf("workbuddy MCP = %+v", server)
	}
	if skill := findAgentSkill(t, overview, "workbuddy", "ops"); skill.Enabled || !skill.CanToggle {
		t.Fatalf("workbuddy disabled skill = %+v", skill)
	}
	if rule := findAgentSkill(t, overview, "cursor", "style"); rule.ResourceType != "rule" || rule.CanToggle {
		t.Fatalf("cursor rule skill resource = %+v", rule)
	}
	if memory := findMemory(t, overview, "cursor", "style.mdc"); memory.Kind != "rule" || !memory.CanEdit {
		t.Fatalf("cursor rule memory resource = %+v", memory)
	}
}

func TestGenericMemoryRejectsTraversalAndUnsupportedPaths(t *testing.T) {
	dir := t.TempDir()
	isolateAgentHomes(t, dir)
	root := filepath.Join(dir, ".gemini")
	t.Setenv("AGENTMETER_GEMINI_SETTINGS_PATH", filepath.Join(root, "settings.json"))
	writeFile(t, filepath.Join(root, "GEMINI.md"), "# Gemini\n")

	if _, err := MemoryDetail(context.Background(), "gemini", "", "../outside.md"); err == nil {
		t.Fatal("expected traversal to be rejected")
	}
	if _, err := MemoryDetail(context.Background(), "gemini", "", "settings.json"); err == nil {
		t.Fatal("expected unsupported path to be rejected")
	}
}

func TestMalformedJSONSettingsProducesWarning(t *testing.T) {
	dir := t.TempDir()
	isolateAgentHomes(t, dir)
	root := filepath.Join(dir, ".gemini")
	t.Setenv("AGENTMETER_GEMINI_SETTINGS_PATH", filepath.Join(root, "settings.json"))
	writeFile(t, filepath.Join(root, "settings.json"), `{"mcpServers":`)

	overview, err := Overview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(overview.Warnings) == 0 {
		t.Fatalf("expected malformed settings warning: %+v", overview)
	}
}

func TestOverviewEncodesEmptyResourceCollectionsAsArrays(t *testing.T) {
	dir := t.TempDir()
	isolateAgentHomes(t, dir)
	root := filepath.Join(dir, ".codex")
	t.Setenv("CODEX_HOME", root)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}

	overview, err := Overview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(overview)
	if err != nil {
		t.Fatal(err)
	}
	text := string(payload)
	for _, unexpected := range []string{
		`"skills":null`,
		`"mcpServers":null`,
		`"memories":null`,
		`"warnings":null`,
	} {
		if strings.Contains(text, unexpected) {
			t.Fatalf("overview should encode empty collections as arrays, got %s", text)
		}
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
	writeFile(t, filepath.Join(root, "memories", "extensions", "ad_hoc", "instructions.md"), "# Ad hoc instructions\n")
	writeFile(t, filepath.Join(root, "memories", "skills", "parallel-worker-orchestration", "SKILL.md"), "# Skill instructions\n")

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
	_, err = MemoryDetail(context.Background(), "codex", "", "skills/parallel-worker-orchestration/SKILL.md")
	if err == nil {
		t.Fatal("expected skill resource file to be rejected as memory")
	}
	_, err = MemoryDetail(context.Background(), "codex", "", "extensions/ad_hoc/instructions.md")
	if err == nil {
		t.Fatal("expected extension resource file to be rejected as memory")
	}
	_, err = UpdateMemory(context.Background(), modelMemoryUpdate("skills/parallel-worker-orchestration/SKILL.md", "# Updated\n"))
	if err == nil {
		t.Fatal("expected skill resource file update to be rejected as memory")
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

func findAgentSkill(t *testing.T, overview model.AgentResourceOverview, agentKind, name string) model.AgentSkillResource {
	t.Helper()
	for _, skill := range overview.Skills {
		if skill.AgentKind == agentKind && skill.Name == name {
			return skill
		}
	}
	t.Fatalf("skill %s/%q missing: %+v", agentKind, name, overview.Skills)
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

func findAgentMCP(t *testing.T, overview model.AgentResourceOverview, agentKind, name string) model.AgentMCPServerResource {
	t.Helper()
	for _, server := range overview.MCPServers {
		if server.AgentKind == agentKind && server.Name == name {
			return server
		}
	}
	t.Fatalf("mcp server %s/%q missing: %+v", agentKind, name, overview.MCPServers)
	return model.AgentMCPServerResource{}
}

func findMemory(t *testing.T, overview model.AgentResourceOverview, agentKind, rel string) model.AgentMemoryResource {
	t.Helper()
	for _, memory := range overview.Memories {
		if memory.AgentKind == agentKind && memory.RelativePath == rel {
			return memory
		}
	}
	t.Fatalf("memory %s/%q missing: %+v", agentKind, rel, overview.Memories)
	return model.AgentMemoryResource{}
}

func containsJSONListValue(value any, want string) bool {
	raw, ok := value.([]any)
	if !ok {
		return false
	}
	for _, item := range raw {
		if item == want {
			return true
		}
	}
	return false
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
