package agentresources

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/LyleMi/AgentMeter/internal/model"
	"github.com/LyleMi/AgentMeter/internal/platform"
	"github.com/LyleMi/AgentMeter/internal/sourcepath"
	"github.com/pelletier/go-toml/v2"
)

const (
	codexKind               = "codex"
	codexName               = "Codex"
	agentResourceSkills     = "skills"
	agentResourceMCPServers = "mcpServers"
	agentResourceMemories   = "memories"
)

func Overview(_ context.Context) (model.AgentResourceOverview, error) {
	result := model.AgentResourceOverview{
		Agents:     agentResourceAgents(),
		Skills:     []model.AgentSkillResource{},
		MCPServers: []model.AgentMCPServerResource{},
		Memories:   []model.AgentMemoryResource{},
		Warnings:   []string{},
	}
	for _, agent := range result.Agents {
		result.Warnings = append(result.Warnings, agent.Warnings...)
		if !agent.Exists {
			continue
		}
		var warnings []string
		skills, warnings := skillsForAgent(agent)
		result.Skills = append(result.Skills, skills...)
		result.Warnings = append(result.Warnings, warnings...)
		servers, warnings := mcpServersForAgent(agent)
		result.MCPServers = append(result.MCPServers, servers...)
		result.Warnings = append(result.Warnings, warnings...)
		memories, warnings := memoriesForAgent(agent)
		result.Memories = append(result.Memories, memories...)
		result.Warnings = append(result.Warnings, warnings...)
	}
	sortAgentResources(result.Skills, result.MCPServers, result.Memories)
	return result, nil
}

type ResourceError struct {
	Status  int
	Message string
}

func (e ResourceError) Error() string {
	return e.Message
}

func BadRequest(message string) error {
	return ResourceError{Status: 400, Message: message}
}

func NotFound(message string) error {
	return ResourceError{Status: 404, Message: message}
}

func Unsupported(message string) error {
	return ResourceError{Status: 400, Message: message}
}

func SetSkillEnabled(_ context.Context, request model.AgentResourceToggleRequest) (model.AgentResourceOperationResult, error) {
	agent, err := requireAgentForKind(request.AgentKind)
	if err != nil {
		return model.AgentResourceOperationResult{}, err
	}
	switch agent.Kind {
	case codexKind:
		err = setCodexSkillEnabled(agent.RootPath, request, request.Enabled)
	case "gemini", "claude", "codebuddy", "workbuddy":
		err = setPackageSkillEnabled(agent, request, request.Enabled)
	default:
		err = Unsupported("skill toggles are not supported for " + agent.Kind)
	}
	if err != nil {
		return model.AgentResourceOperationResult{}, err
	}
	return operationResult(context.Background())
}

func SetMCPServerEnabled(_ context.Context, request model.AgentResourceToggleRequest) (model.AgentResourceOperationResult, error) {
	agent, err := requireAgentForKind(request.AgentKind)
	if err != nil {
		return model.AgentResourceOperationResult{}, err
	}
	name := strings.TrimSpace(request.Name)
	if name == "" {
		return model.AgentResourceOperationResult{}, BadRequest("MCP server name is required")
	}
	if err := setMCPServerEnabledForAgent(agent, name, request.Enabled); err != nil {
		return model.AgentResourceOperationResult{}, err
	}
	return operationResult(context.Background())
}

func setMCPServerEnabledForAgent(agent model.AgentResourceAgent, name string, enabled bool) error {
	switch agent.Kind {
	case "gemini":
		return setGeminiMCPEnabled(agent, name, enabled)
	case codexKind:
		return setCodexMCPEnabled(agent, name, enabled)
	default:
		return setJSONMCPEnabled(agent, name, enabled)
	}
}

func setCodexMCPEnabled(agent model.AgentResourceAgent, name string, enabled bool) error {
	if err := ensurePathInside(agent.ConfigPath, agent.RootPath); err != nil {
		return err
	}
	content, err := os.ReadFile(agent.ConfigPath)
	if err != nil {
		return err
	}
	var root map[string]any
	if err := toml.Unmarshal(content, &root); err != nil {
		return err
	}
	rawServers, _ := root["mcp_servers"].(map[string]any)
	raw, ok := rawServers[name]
	if !ok {
		return NotFound("MCP server was not found")
	}
	if _, ok := raw.(map[string]any); !ok {
		return BadRequest("MCP server configuration is not editable")
	}
	updated, err := setMCPEnabledInTOML(content, name, enabled)
	if err != nil {
		return err
	}
	if err := os.WriteFile(agent.ConfigPath, updated, 0o644); err != nil {
		return err
	}
	return nil
}

func operationResult(ctx context.Context) (model.AgentResourceOperationResult, error) {
	overview, err := Overview(ctx)
	if err != nil {
		return model.AgentResourceOperationResult{}, err
	}
	return model.AgentResourceOperationResult{Overview: overview, Warnings: overview.Warnings}, nil
}

func requireAgentForKind(kind string) (model.AgentResourceAgent, error) {
	kind = strings.ToLower(strings.TrimSpace(kind))
	if kind == "" {
		kind = codexKind
	}
	for _, agent := range agentResourceAgents() {
		if agent.Kind == kind {
			if !agent.Exists {
				return model.AgentResourceAgent{}, NotFound(agent.Name + " home is not available")
			}
			return agent, nil
		}
	}
	return model.AgentResourceAgent{}, Unsupported("resource writes are not supported for " + kind)
}

func agentResourceAgents() []model.AgentResourceAgent {
	definitions := []struct {
		kind       string
		name       string
		root       string
		configPath string
		supports   []string
	}{
		{codexKind, codexName, codexRoot(), filepath.Join(codexRoot(), "config.toml"), []string{agentResourceSkills, agentResourceMCPServers, agentResourceMemories}},
		{"gemini", "Gemini CLI", jsonAgentRoot("AGENTMETER_GEMINI_SETTINGS_PATH", "", ".gemini"), jsonAgentConfigPath("AGENTMETER_GEMINI_SETTINGS_PATH", "", ".gemini"), []string{agentResourceSkills, agentResourceMCPServers, agentResourceMemories}},
		{"claude", "Claude Code", jsonAgentRoot("AGENTMETER_CLAUDE_SETTINGS_PATH", "CLAUDE_CONFIG_DIR", ".claude"), jsonAgentConfigPath("AGENTMETER_CLAUDE_SETTINGS_PATH", "CLAUDE_CONFIG_DIR", ".claude"), []string{agentResourceSkills, agentResourceMCPServers, agentResourceMemories}},
		{"codebuddy", "CodeBuddy", jsonAgentRoot("AGENTMETER_CODEBUDDY_SETTINGS_PATH", "CODEBUDDY_CONFIG_DIR", ".codebuddy"), jsonAgentConfigPath("AGENTMETER_CODEBUDDY_SETTINGS_PATH", "CODEBUDDY_CONFIG_DIR", ".codebuddy"), []string{agentResourceSkills, agentResourceMCPServers, agentResourceMemories}},
		{"workbuddy", "WorkBuddy", envOrHomeRoot("WORKBUDDY_CONFIG_DIR", ".workbuddy"), filepath.Join(envOrHomeRoot("WORKBUDDY_CONFIG_DIR", ".workbuddy"), "settings.json"), []string{agentResourceSkills, agentResourceMCPServers, agentResourceMemories}},
		{"cursor", "Cursor", envOrHomeRoot("CURSOR_HOME", ".cursor"), filepath.Join(envOrHomeRoot("CURSOR_HOME", ".cursor"), "mcp.json"), []string{"rules", agentResourceMCPServers}},
	}
	agents := make([]model.AgentResourceAgent, 0, len(definitions))
	for _, definition := range definitions {
		agent := model.AgentResourceAgent{
			Kind:        definition.kind,
			Name:        definition.name,
			RootPath:    sourcepath.Normalize(definition.root),
			ConfigPath:  sourcepath.Normalize(definition.configPath),
			Warnings:    []string{},
			Supports:    append([]string{}, definition.supports...),
			Unsupported: []string{},
		}
		if stat, err := os.Stat(agent.RootPath); err == nil && stat.IsDir() {
			agent.Exists = true
		} else if err != nil {
			agent.Warnings = append(agent.Warnings, definition.name+" home is not available: "+err.Error())
		}
		agents = append(agents, agent)
	}
	return agents
}

func codexRoot() string {
	if value := strings.TrimSpace(os.Getenv("CODEX_HOME")); value != "" {
		return sourcepath.Normalize(value)
	}
	return sourcepath.Normalize(platform.DefaultCodexRoot())
}

func codexMCPServers(configPath string) ([]model.AgentMCPServerResource, []string) {
	content, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		return []model.AgentMCPServerResource{}, nil
	}
	if err != nil {
		return []model.AgentMCPServerResource{}, []string{"Unable to read Codex config: " + err.Error()}
	}
	var root map[string]any
	if err := toml.Unmarshal(content, &root); err != nil {
		return []model.AgentMCPServerResource{}, []string{"Unable to parse Codex config: " + err.Error()}
	}
	rawServers, _ := root["mcp_servers"].(map[string]any)
	servers := make([]model.AgentMCPServerResource, 0, len(rawServers))
	for name, raw := range rawServers {
		table, _ := raw.(map[string]any)
		command := stringValue(table["command"])
		args := stringSlice(table["args"])
		envKeys := mapKeys(table["env"])
		status := "configured"
		warning := ""
		if strings.TrimSpace(command) == "" {
			status = "incomplete"
			warning = "command is not configured"
		}
		enabled := boolValue(table["enabled"], true)
		if !enabled && status == "configured" {
			status = "disabled"
		}
		servers = append(servers, model.AgentMCPServerResource{
			AgentKind:  codexKind,
			Name:       name,
			Command:    command,
			Args:       args,
			EnvKeys:    envKeys,
			ConfigPath: configPath,
			Enabled:    enabled && status == "configured",
			CanToggle:  true,
			Status:     status,
			Warning:    warning,
		})
	}
	sort.Slice(servers, func(i, j int) bool {
		return strings.ToLower(servers[i].Name) < strings.ToLower(servers[j].Name)
	})
	return servers, nil
}

func markdownTitle(content []byte) string {
	lines := strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")
	inFrontmatter := false
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if index == 0 && trimmed == "---" {
			inFrontmatter = true
			continue
		}
		if inFrontmatter {
			if trimmed == "---" {
				inFrontmatter = false
			}
			continue
		}
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
		}
	}
	return ""
}

func textPreview(content []byte, limit int) string {
	text := strings.ReplaceAll(string(content), "\r\n", "\n")
	var parts []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts = append(parts, line)
		if len(strings.Join(parts, " ")) >= limit {
			break
		}
	}
	preview := strings.Join(parts, " ")
	if len(preview) <= limit {
		return preview
	}
	return strings.TrimSpace(preview[:limit]) + "..."
}

func relativePath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func relativePathFromRoot(root, path string) string {
	return relativePath(root, path)
}

func setCodexSkillEnabled(root string, request model.AgentResourceToggleRequest, enabled bool) error {
	skillsRoot := filepath.Join(root, agentResourceSkills)
	return setSkillMarkdownEnabled(skillsRoot, request, enabled)
}

func resolveCodexMemoryPath(root, path, rel string) (string, error) {
	memoriesRoot := filepath.Join(root, agentResourceMemories)
	return resolvePathInRoot(memoriesRoot, path, rel)
}

func resolvePathInRoot(root, path, rel string) (string, error) {
	root = filepath.Clean(root)
	candidate := strings.TrimSpace(path)
	if candidate == "" {
		if strings.TrimSpace(rel) == "" {
			return "", BadRequest("path or relativePath is required")
		}
		if filepath.IsAbs(rel) {
			return "", BadRequest("relativePath must not be absolute")
		}
		candidate = filepath.Join(root, filepath.FromSlash(rel))
	}
	candidate = filepath.Clean(candidate)
	if err := ensurePathInside(candidate, root); err != nil {
		return "", err
	}
	return candidate, nil
}

func ensurePathInside(path, root string) error {
	if strings.TrimSpace(path) == "" || strings.TrimSpace(root) == "" {
		return BadRequest("path and root are required")
	}
	absRoot, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return err
	}
	absPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return err
	}
	rootKey := comparablePath(absRoot)
	pathKey := comparablePath(absPath)
	if pathKey == rootKey || strings.HasPrefix(pathKey, rootKey+string(os.PathSeparator)) {
		return nil
	}
	return BadRequest("path is outside the known agent resource root")
}

func comparablePath(path string) string {
	cleaned := filepath.Clean(path)
	if runtime.GOOS == "windows" {
		return strings.ToLower(cleaned)
	}
	return cleaned
}

func enabledStatus(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}

func boolValue(value any, fallback bool) bool {
	typed, ok := value.(bool)
	if !ok {
		return fallback
	}
	return typed
}

func jsonAgentRoot(overrideEnv, configDirEnv, homeDirName string) string {
	if path := strings.TrimSpace(os.Getenv(overrideEnv)); path != "" {
		return filepath.Dir(filepath.Clean(path))
	}
	return envOrHomeRoot(configDirEnv, homeDirName)
}

func jsonAgentConfigPath(overrideEnv, configDirEnv, homeDirName string) string {
	if path := strings.TrimSpace(os.Getenv(overrideEnv)); path != "" {
		return filepath.Clean(path)
	}
	return filepath.Join(envOrHomeRoot(configDirEnv, homeDirName), "settings.json")
}

func envOrHomeRoot(envName, homeDirName string) string {
	if envName != "" {
		if value := strings.TrimSpace(os.Getenv(envName)); value != "" {
			return filepath.Clean(value)
		}
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, homeDirName)
	}
	return homeDirName
}

func memoryKind(rel string) string {
	rel = filepath.ToSlash(rel)
	switch {
	case rel == "MEMORY.md":
		return "primary"
	case rel == "memory_summary.md":
		return "summary"
	case rel == "raw_memories.md":
		return "raw"
	case strings.HasPrefix(rel, "rollout_summaries/"):
		return "rollout"
	default:
		return "markdown"
	}
}

func shouldSkipCodexMemoryDir(rel, name string) bool {
	if name == ".git" {
		return true
	}
	rel = filepath.ToSlash(rel)
	return rel == "extensions" || strings.HasPrefix(rel, "extensions/") ||
		rel == agentResourceSkills || strings.HasPrefix(rel, agentResourceSkills+"/")
}

func isCodexMemoryFile(rel string) bool {
	rel = filepath.ToSlash(filepath.Clean(rel))
	if !strings.EqualFold(filepath.Ext(rel), ".md") {
		return false
	}
	return !(rel == "extensions" || strings.HasPrefix(rel, "extensions/") ||
		rel == agentResourceSkills || strings.HasPrefix(rel, agentResourceSkills+"/") ||
		rel == ".git" || strings.HasPrefix(rel, ".git/"))
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return ""
	}
}

func stringSlice(value any) []string {
	raw, ok := value.([]any)
	if !ok {
		return []string{}
	}
	values := make([]string, 0, len(raw))
	for _, item := range raw {
		switch typed := item.(type) {
		case string:
			values = append(values, typed)
		case int64:
			values = append(values, strconv.FormatInt(typed, 10))
		case float64:
			values = append(values, strconv.FormatFloat(typed, 'f', -1, 64))
		}
	}
	return values
}

func mapKeys(value any) []string {
	raw, ok := value.(map[string]any)
	if !ok {
		return []string{}
	}
	keys := make([]string, 0, len(raw))
	for key := range raw {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
