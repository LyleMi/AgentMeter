package agentresources

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
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
	codexKind = "codex"
	codexName = "Codex"
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
		if agent.Kind == codexKind {
			result.Warnings = append(result.Warnings, agent.Warnings...)
		}
		if agent.Kind != codexKind || !agent.Exists {
			continue
		}
		var warnings []string
		result.Skills, warnings = codexSkills(filepath.Join(agent.RootPath, "skills"))
		result.Warnings = append(result.Warnings, warnings...)
		result.MCPServers, warnings = codexMCPServers(agent.ConfigPath)
		result.Warnings = append(result.Warnings, warnings...)
		result.Memories, warnings = codexMemories(filepath.Join(agent.RootPath, "memories"))
		result.Warnings = append(result.Warnings, warnings...)
		return result, nil
	}
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
	agent, err := requireCodexAgentForKind(request.AgentKind)
	if err != nil {
		return model.AgentResourceOperationResult{}, err
	}
	if request.Enabled {
		err = setCodexSkillEnabled(agent.RootPath, request, true)
	} else {
		err = setCodexSkillEnabled(agent.RootPath, request, false)
	}
	if err != nil {
		return model.AgentResourceOperationResult{}, err
	}
	return operationResult(context.Background())
}

func SetMCPServerEnabled(_ context.Context, request model.AgentResourceToggleRequest) (model.AgentResourceOperationResult, error) {
	agent, err := requireCodexAgentForKind(request.AgentKind)
	if err != nil {
		return model.AgentResourceOperationResult{}, err
	}
	name := strings.TrimSpace(request.Name)
	if name == "" {
		return model.AgentResourceOperationResult{}, BadRequest("MCP server name is required")
	}
	if err := ensurePathInside(agent.ConfigPath, agent.RootPath); err != nil {
		return model.AgentResourceOperationResult{}, err
	}
	content, err := os.ReadFile(agent.ConfigPath)
	if err != nil {
		return model.AgentResourceOperationResult{}, err
	}
	var root map[string]any
	if err := toml.Unmarshal(content, &root); err != nil {
		return model.AgentResourceOperationResult{}, err
	}
	rawServers, _ := root["mcp_servers"].(map[string]any)
	raw, ok := rawServers[name]
	if !ok {
		return model.AgentResourceOperationResult{}, NotFound("MCP server was not found")
	}
	if _, ok := raw.(map[string]any); !ok {
		return model.AgentResourceOperationResult{}, BadRequest("MCP server configuration is not editable")
	}
	updated, err := setMCPEnabledInTOML(content, name, request.Enabled)
	if err != nil {
		return model.AgentResourceOperationResult{}, err
	}
	if err := os.WriteFile(agent.ConfigPath, updated, 0o644); err != nil {
		return model.AgentResourceOperationResult{}, err
	}
	return operationResult(context.Background())
}

func MemoryDetail(_ context.Context, agentKind, path, relativePath string) (model.AgentMemoryDetail, error) {
	agent, err := requireCodexAgentForKind(agentKind)
	if err != nil {
		return model.AgentMemoryDetail{}, err
	}
	memoryPath, err := resolveCodexMemoryPath(agent.RootPath, path, relativePath)
	if err != nil {
		return model.AgentMemoryDetail{}, err
	}
	info, err := os.Stat(memoryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return model.AgentMemoryDetail{}, NotFound("memory file was not found")
		}
		return model.AgentMemoryDetail{}, err
	}
	if info.IsDir() {
		return model.AgentMemoryDetail{}, BadRequest("memory path must be a file")
	}
	content, err := os.ReadFile(memoryPath)
	if err != nil {
		return model.AgentMemoryDetail{}, err
	}
	rel := relativePathFromRoot(filepath.Join(agent.RootPath, "memories"), memoryPath)
	name := strings.TrimSuffix(filepath.Base(memoryPath), filepath.Ext(memoryPath))
	return model.AgentMemoryDetail{
		AgentMemoryResource: model.AgentMemoryResource{
			AgentKind:    codexKind,
			Name:         name,
			Title:        firstNonEmpty(markdownTitle(content), name),
			Path:         memoryPath,
			RelativePath: rel,
			Kind:         memoryKind(rel),
			Preview:      textPreview(content, 260),
			CanEdit:      true,
			SizeBytes:    info.Size(),
			ModifiedAt:   info.ModTime().UTC(),
		},
		Content: string(content),
	}, nil
}

func UpdateMemory(_ context.Context, request model.AgentMemoryUpdateRequest) (model.AgentMemoryDetail, error) {
	agent, err := requireCodexAgentForKind(request.AgentKind)
	if err != nil {
		return model.AgentMemoryDetail{}, err
	}
	memoryPath, err := resolveCodexMemoryPath(agent.RootPath, request.Path, request.RelativePath)
	if err != nil {
		return model.AgentMemoryDetail{}, err
	}
	if !strings.EqualFold(filepath.Ext(memoryPath), ".md") {
		return model.AgentMemoryDetail{}, BadRequest("memory path must be a markdown file")
	}
	if err := os.MkdirAll(filepath.Dir(memoryPath), 0o755); err != nil {
		return model.AgentMemoryDetail{}, err
	}
	if err := os.WriteFile(memoryPath, []byte(request.Content), 0o644); err != nil {
		return model.AgentMemoryDetail{}, err
	}
	return MemoryDetail(context.Background(), codexKind, memoryPath, "")
}

func operationResult(ctx context.Context) (model.AgentResourceOperationResult, error) {
	overview, err := Overview(ctx)
	if err != nil {
		return model.AgentResourceOperationResult{}, err
	}
	return model.AgentResourceOperationResult{Overview: overview, Warnings: overview.Warnings}, nil
}

func requireCodexAgentForKind(kind string) (model.AgentResourceAgent, error) {
	kind = strings.ToLower(strings.TrimSpace(kind))
	if kind == "" {
		kind = codexKind
	}
	if kind != codexKind {
		return model.AgentResourceAgent{}, Unsupported("resource writes are not supported for " + kind)
	}
	for _, agent := range agentResourceAgents() {
		if agent.Kind == codexKind {
			if !agent.Exists {
				return model.AgentResourceAgent{}, NotFound("Codex home is not available")
			}
			return agent, nil
		}
	}
	return model.AgentResourceAgent{}, NotFound("Codex home is not available")
}

func agentResourceAgents() []model.AgentResourceAgent {
	definitions := []struct {
		kind       string
		name       string
		root       string
		configPath string
		supports   []string
	}{
		{codexKind, codexName, codexRoot(), filepath.Join(codexRoot(), "config.toml"), []string{"skills", "mcpServers", "memories"}},
		{"gemini", "Gemini CLI", jsonAgentRoot("AGENTMETER_GEMINI_SETTINGS_PATH", "", ".gemini"), jsonAgentConfigPath("AGENTMETER_GEMINI_SETTINGS_PATH", "", ".gemini"), nil},
		{"claude", "Claude Code", jsonAgentRoot("AGENTMETER_CLAUDE_SETTINGS_PATH", "CLAUDE_CONFIG_DIR", ".claude"), jsonAgentConfigPath("AGENTMETER_CLAUDE_SETTINGS_PATH", "CLAUDE_CONFIG_DIR", ".claude"), nil},
		{"codebuddy", "CodeBuddy", jsonAgentRoot("AGENTMETER_CODEBUDDY_SETTINGS_PATH", "CODEBUDDY_CONFIG_DIR", ".codebuddy"), jsonAgentConfigPath("AGENTMETER_CODEBUDDY_SETTINGS_PATH", "CODEBUDDY_CONFIG_DIR", ".codebuddy"), nil},
		{"workbuddy", "WorkBuddy", envOrHomeRoot("WORKBUDDY_CONFIG_DIR", ".workbuddy"), filepath.Join(envOrHomeRoot("WORKBUDDY_CONFIG_DIR", ".workbuddy"), "settings.json"), nil},
		{"cursor", "Cursor", envOrHomeRoot("CURSOR_HOME", ".cursor"), filepath.Join(envOrHomeRoot("CURSOR_HOME", ".cursor"), "settings.json"), nil},
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
		if definition.kind != codexKind {
			agent.Unsupported = []string{"skills", "mcpServers", "memories"}
			agent.Warnings = append(agent.Warnings, definition.name+" resource inventory is not implemented yet; resources are intentionally empty.")
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

func codexSkills(root string) ([]model.AgentSkillResource, []string) {
	if stat, err := os.Stat(root); err != nil || !stat.IsDir() {
		return []model.AgentSkillResource{}, nil
	}
	items := []model.AgentSkillResource{}
	warnings := []string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			warnings = append(warnings, "Unable to inspect skill path "+path+": "+err.Error())
			return nil
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		enabled := strings.EqualFold(entry.Name(), "SKILL.md")
		if !enabled && !strings.EqualFold(entry.Name(), "SKILL.md.disabled") {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			warnings = append(warnings, "Unable to inspect skill file "+path+": "+err.Error())
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			warnings = append(warnings, "Unable to read skill file "+path+": "+err.Error())
			return nil
		}
		dir := filepath.Dir(path)
		rel := relativePath(root, dir)
		meta := skillMetadata(content)
		name := firstNonEmpty(meta["name"], filepath.Base(dir))
		items = append(items, model.AgentSkillResource{
			AgentKind:    codexKind,
			Name:         name,
			Title:        firstNonEmpty(markdownTitle(content), name),
			Description:  meta["description"],
			Path:         dir,
			RelativePath: rel,
			System:       strings.HasPrefix(filepath.ToSlash(rel), ".system/"),
			Enabled:      enabled,
			CanToggle:    !strings.HasPrefix(filepath.ToSlash(rel), ".system/"),
			Status:       enabledStatus(enabled),
			SizeBytes:    info.Size(),
			ModifiedAt:   info.ModTime().UTC(),
		})
		return nil
	})
	if err != nil {
		warnings = append(warnings, "Unable to scan Codex skills: "+err.Error())
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].System != items[j].System {
			return !items[i].System
		}
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	return items, warnings
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

func codexMemories(root string) ([]model.AgentMemoryResource, []string) {
	if stat, err := os.Stat(root); err != nil || !stat.IsDir() {
		return []model.AgentMemoryResource{}, nil
	}
	items := []model.AgentMemoryResource{}
	warnings := []string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			warnings = append(warnings, "Unable to inspect memory path "+path+": "+err.Error())
			return nil
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			warnings = append(warnings, "Unable to inspect memory file "+path+": "+err.Error())
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			warnings = append(warnings, "Unable to read memory file "+path+": "+err.Error())
			return nil
		}
		rel := relativePath(root, path)
		name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		items = append(items, model.AgentMemoryResource{
			AgentKind:    codexKind,
			Name:         name,
			Title:        firstNonEmpty(markdownTitle(content), name),
			Path:         path,
			RelativePath: rel,
			Kind:         memoryKind(rel),
			Preview:      textPreview(content, 260),
			CanEdit:      true,
			SizeBytes:    info.Size(),
			ModifiedAt:   info.ModTime().UTC(),
		})
		return nil
	})
	if err != nil {
		warnings = append(warnings, "Unable to scan Codex memories: "+err.Error())
	}
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].RelativePath) < strings.ToLower(items[j].RelativePath)
	})
	return items, warnings
}

func skillMetadata(content []byte) map[string]string {
	meta := map[string]string{}
	trimmed := bytes.TrimLeft(content, "\xef\xbb\xbf\r\n\t ")
	if !bytes.HasPrefix(trimmed, []byte("---")) {
		return meta
	}
	lines := strings.Split(strings.ReplaceAll(string(trimmed), "\r\n", "\n"), "\n")
	for index := 1; index < len(lines); index++ {
		line := lines[index]
		if strings.TrimSpace(line) == "---" {
			break
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		cleanValue := strings.TrimSpace(value)
		if cleanValue == "|" || cleanValue == ">" {
			var block []string
			for index+1 < len(lines) {
				next := lines[index+1]
				if strings.TrimSpace(next) == "---" || (strings.TrimSpace(next) != "" && !strings.HasPrefix(next, " ") && !strings.HasPrefix(next, "\t")) {
					break
				}
				block = append(block, strings.TrimSpace(next))
				index++
			}
			meta[key] = strings.Join(nonEmptyStrings(block), " ")
			continue
		}
		meta[key] = strings.Trim(cleanValue, `"'`)
	}
	return meta
}

func nonEmptyStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			result = append(result, strings.TrimSpace(value))
		}
	}
	return result
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
	skillsRoot := filepath.Join(root, "skills")
	dir, err := resolvePathInRoot(skillsRoot, request.Path, request.RelativePath)
	if err != nil {
		return err
	}
	rel := relativePathFromRoot(skillsRoot, dir)
	if rel == "." || strings.HasPrefix(filepath.ToSlash(rel), ".system/") {
		return Unsupported("system skills cannot be toggled")
	}
	active := filepath.Join(dir, "SKILL.md")
	disabled := filepath.Join(dir, "SKILL.md.disabled")
	if err := ensurePathInside(active, skillsRoot); err != nil {
		return err
	}
	if err := ensurePathInside(disabled, skillsRoot); err != nil {
		return err
	}
	if enabled {
		if _, err := os.Stat(active); err == nil {
			return nil
		}
		if _, err := os.Stat(disabled); err != nil {
			if os.IsNotExist(err) {
				return NotFound("disabled skill file was not found")
			}
			return err
		}
		return os.Rename(disabled, active)
	}
	if _, err := os.Stat(disabled); err == nil {
		return nil
	}
	if _, err := os.Stat(active); err != nil {
		if os.IsNotExist(err) {
			return NotFound("skill file was not found")
		}
		return err
	}
	return os.Rename(active, disabled)
}

func resolveCodexMemoryPath(root, path, rel string) (string, error) {
	memoriesRoot := filepath.Join(root, "memories")
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

func setMCPEnabledInTOML(content []byte, name string, enabled bool) ([]byte, error) {
	text := string(content)
	separator := "\n"
	if strings.Contains(text, "\r\n") {
		separator = "\r\n"
		text = strings.ReplaceAll(text, "\r\n", "\n")
	}
	lines := strings.Split(text, "\n")
	header := "[mcp_servers." + name + "]"
	start := -1
	for index, line := range lines {
		if strings.TrimSpace(line) == header {
			start = index
			break
		}
	}
	if start < 0 {
		return nil, BadRequest("MCP server table uses an unsupported TOML table style")
	}
	end := len(lines)
	for index := start + 1; index < len(lines); index++ {
		trimmed := strings.TrimSpace(lines[index])
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			end = index
			break
		}
	}
	valueLine := "enabled = " + strconv.FormatBool(enabled)
	for index := start + 1; index < end; index++ {
		trimmed := strings.TrimSpace(lines[index])
		key, _, ok := strings.Cut(trimmed, "=")
		if ok && strings.TrimSpace(key) == "enabled" {
			prefix := lines[index][:len(lines[index])-len(strings.TrimLeft(lines[index], " \t"))]
			lines[index] = prefix + valueLine
			return []byte(strings.Join(lines, separator)), nil
		}
	}
	updated := make([]string, 0, len(lines)+1)
	updated = append(updated, lines[:start+1]...)
	updated = append(updated, valueLine)
	updated = append(updated, lines[start+1:]...)
	return []byte(strings.Join(updated, separator)), nil
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
	case strings.HasPrefix(rel, "extensions/"):
		return "extension"
	case strings.HasPrefix(rel, "rollout_summaries/"):
		return "rollout"
	case strings.HasPrefix(rel, "skills/"):
		return "skill"
	default:
		return "markdown"
	}
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
