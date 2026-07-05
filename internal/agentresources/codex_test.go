package agentresources

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestOverviewReturnsWarningWhenCodexHomeMissing(t *testing.T) {
	root := filepath.Join(t.TempDir(), "missing-codex")
	t.Setenv("CODEX_HOME", root)

	overview, err := Overview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(overview.Agents) != 1 || overview.Agents[0].Exists {
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
	root := t.TempDir()
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
	if overview.Skills[0].Name != "writer" || overview.Skills[0].System {
		t.Fatalf("user skill should sort before system skill: %+v", overview.Skills)
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
	if len(server.EnvKeys) != 2 || server.EnvKeys[0] != "SECRET_TOKEN" || server.EnvKeys[1] != "VISIBLE_NAME" {
		t.Fatalf("env keys should be names only and sorted: %+v", server.EnvKeys)
	}
	if len(overview.Memories) != 1 {
		t.Fatalf("memories = %+v", overview.Memories)
	}
	if overview.Memories[0].Kind != "primary" || overview.Memories[0].Preview != "Keep responses direct." {
		t.Fatalf("memory = %+v", overview.Memories[0])
	}
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
