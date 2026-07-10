package agentresources

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/LyleMi/AgentMeter/internal/model"
)

func TestSetJSONMCPEnabledSupportsEnabledAndDisabledFields(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "settings.json")
	writeFile(t, path, `{"mcpServers":{"first":{"enabled":true},"second":{"disabled":true}}}`)
	agent := model.AgentResourceAgent{Kind: "codebuddy", RootPath: root, ConfigPath: path}

	if err := setJSONMCPEnabled(agent, "first", false); err != nil {
		t.Fatal(err)
	}
	if err := setJSONMCPEnabled(agent, "second", true); err != nil {
		t.Fatal(err)
	}
	rootObject := readJSONObject(t, path)
	servers := rootObject[agentResourceMCPServers].(map[string]any)
	if servers["first"].(map[string]any)["enabled"] != false {
		t.Fatal("enabled field was not updated")
	}
	if servers["second"].(map[string]any)["disabled"] != false {
		t.Fatal("disabled field was not inverted")
	}
}

func TestSetJSONMCPEnabledRejectsUnsupportedServer(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "settings.json")
	writeFile(t, path, `{"mcpServers":{"server":{"command":"mcp"}}}`)
	agent := model.AgentResourceAgent{Kind: "codebuddy", RootPath: root, ConfigPath: path}

	if err := setJSONMCPEnabled(agent, "server", false); err == nil {
		t.Fatal("expected unsupported toggle error")
	}
}

func readJSONObject(t *testing.T, path string) map[string]any {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	if err := json.Unmarshal(content, &value); err != nil {
		t.Fatal(err)
	}
	return value
}
