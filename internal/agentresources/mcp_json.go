package agentresources

import (
	"os"
	"path/filepath"

	"github.com/LyleMi/AgentMeter/internal/model"
)

type jsonMCPConfig struct {
	path string
	root map[string]any
}

func setGeminiMCPEnabled(agent model.AgentResourceAgent, name string, enabled bool) error {
	config, err := loadJSONMCPConfig(agent.ConfigPath, agent.RootPath, "Gemini settings were not found")
	if err != nil {
		return err
	}
	if _, err := config.server(name); err != nil {
		return err
	}
	mcp := config.object("mcp")
	if enabled {
		mcp["excluded"] = removeStringFromAnyList(mcp["excluded"], name)
		if _, tracksAllowedServers := mcp["allowed"]; tracksAllowedServers {
			mcp["allowed"] = addStringToAnyList(mcp["allowed"], name)
		}
	} else {
		mcp["excluded"] = addStringToAnyList(mcp["excluded"], name)
	}
	return config.write()
}

func setJSONMCPEnabled(agent model.AgentResourceAgent, name string, enabled bool) error {
	configPath, safetyRoot := jsonMCPConfigLocation(agent)
	config, err := loadJSONMCPConfig(configPath, safetyRoot, "MCP config was not found")
	if err != nil {
		return err
	}
	table, err := config.editableServer(name)
	if err != nil {
		return err
	}
	if !setJSONServerEnabled(table, enabled) {
		return Unsupported("MCP server does not expose a supported enable field")
	}
	return config.write()
}

func jsonMCPConfigLocation(agent model.AgentResourceAgent) (string, string) {
	if agent.Kind == "claude" {
		path := claudeMCPConfigPath()
		if stat, err := os.Stat(path); err == nil && !stat.IsDir() {
			return path, filepath.Dir(path)
		}
	}
	return agent.ConfigPath, agent.RootPath
}

func loadJSONMCPConfig(path, safetyRoot, missingMessage string) (jsonMCPConfig, error) {
	if err := ensurePathInside(path, safetyRoot); err != nil {
		return jsonMCPConfig{}, err
	}
	root, exists, err := readJSONSettings(path)
	if err != nil && !os.IsNotExist(err) {
		return jsonMCPConfig{}, err
	}
	if !exists {
		return jsonMCPConfig{}, NotFound(missingMessage)
	}
	return jsonMCPConfig{path: path, root: root}, nil
}

func (c jsonMCPConfig) server(name string) (any, error) {
	servers, _ := c.root[agentResourceMCPServers].(map[string]any)
	server, ok := servers[name]
	if !ok {
		return nil, NotFound("MCP server was not found")
	}
	return server, nil
}

func (c jsonMCPConfig) editableServer(name string) (map[string]any, error) {
	server, err := c.server(name)
	if err != nil {
		return nil, err
	}
	table, ok := server.(map[string]any)
	if !ok {
		return nil, BadRequest("MCP server configuration is not editable")
	}
	return table, nil
}

func (c jsonMCPConfig) object(key string) map[string]any {
	value, _ := c.root[key].(map[string]any)
	if value == nil {
		value = map[string]any{}
		c.root[key] = value
	}
	return value
}

func (c jsonMCPConfig) write() error {
	return writeJSONSettings(c.path, c.root)
}

func setJSONServerEnabled(server map[string]any, enabled bool) bool {
	if _, ok := server["enabled"].(bool); ok {
		server["enabled"] = enabled
		return true
	}
	if _, ok := server["disabled"].(bool); ok {
		server["disabled"] = !enabled
		return true
	}
	return false
}
