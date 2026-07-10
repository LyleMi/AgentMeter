package agentresources

import (
	"strings"
	"testing"
)

func TestSetMCPEnabledInTOML(t *testing.T) {
	tests := []struct {
		name    string
		content string
		enabled bool
		want    string
	}{
		{
			name:    "inserts before table body",
			content: "[mcp_servers.node]\ncommand = 'node'\n",
			enabled: false,
			want:    "[mcp_servers.node]\nenabled = false\ncommand = 'node'\n",
		},
		{
			name:    "replaces indented value",
			content: "[mcp_servers.node]\n  enabled = false\ncommand = 'node'\n",
			enabled: true,
			want:    "[mcp_servers.node]\n  enabled = true\ncommand = 'node'\n",
		},
		{
			name:    "preserves CRLF",
			content: "[mcp_servers.node]\r\ncommand = 'node'\r\n[mcp_servers.other]\r\ncommand = 'other'\r\n",
			enabled: true,
			want:    "[mcp_servers.node]\r\nenabled = true\r\ncommand = 'node'\r\n[mcp_servers.other]\r\ncommand = 'other'\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := setMCPEnabledInTOML([]byte(tt.content), "node", tt.enabled)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Fatalf("updated TOML = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSetMCPEnabledInTOMLRejectsUnsupportedTable(t *testing.T) {
	_, err := setMCPEnabledInTOML([]byte("[mcp_servers.'node']\ncommand = 'node'\n"), "node", true)
	if err == nil || !strings.Contains(err.Error(), "unsupported TOML table style") {
		t.Fatalf("error = %v", err)
	}
}
