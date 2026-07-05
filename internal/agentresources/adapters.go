package agentresources

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/LyleMi/AgentMeter/internal/model"
)

func skillsForAgent(agent model.AgentResourceAgent) ([]model.AgentSkillResource, []string) {
	switch agent.Kind {
	case codexKind:
		return codexSkills(filepath.Join(agent.RootPath, "skills"))
	case "gemini":
		return packageSkills(agent, filepath.Join(agent.RootPath, "skills"))
	case "claude", "codebuddy", "workbuddy":
		items, warnings := packageSkills(agent, filepath.Join(agent.RootPath, "skills"))
		commands, commandWarnings := markdownSkillResources(agent, filepath.Join(agent.RootPath, "commands"), "command")
		agents, agentWarnings := markdownSkillResources(agent, filepath.Join(agent.RootPath, "agents"), "subagent")
		items = append(items, commands...)
		items = append(items, agents...)
		warnings = append(warnings, commandWarnings...)
		warnings = append(warnings, agentWarnings...)
		return items, warnings
	case "cursor":
		return cursorRules(agent)
	default:
		return []model.AgentSkillResource{}, nil
	}
}

func mcpServersForAgent(agent model.AgentResourceAgent) ([]model.AgentMCPServerResource, []string) {
	switch agent.Kind {
	case codexKind:
		return codexMCPServers(agent.ConfigPath)
	case "gemini":
		return geminiMCPServers(agent)
	case "claude":
		return claudeMCPServers(agent)
	case "codebuddy", "workbuddy", "cursor":
		return jsonMCPServers(agent, agent.ConfigPath)
	default:
		return []model.AgentMCPServerResource{}, nil
	}
}

func memoriesForAgent(agent model.AgentResourceAgent) ([]model.AgentMemoryResource, []string) {
	switch agent.Kind {
	case codexKind:
		return codexMemories(filepath.Join(agent.RootPath, "memories"))
	case "gemini":
		return singleInstructionMemory(agent, "GEMINI.md", "primary")
	case "claude":
		return appendMarkdownResources(agent, []markdownResourceSpec{
			{Root: agent.RootPath, RelativePath: "CLAUDE.md", Kind: "primary", CanEdit: true},
			{Root: filepath.Join(agent.RootPath, "commands"), Kind: "command", CanEdit: true},
			{Root: filepath.Join(agent.RootPath, "agents"), Kind: "subagent", CanEdit: true},
		})
	case "codebuddy":
		return appendMarkdownResources(agent, []markdownResourceSpec{
			{Root: agent.RootPath, RelativePath: "CODEBUDDY.md", Kind: "primary", CanEdit: true},
			{Root: filepath.Join(agent.RootPath, "commands"), Kind: "command", CanEdit: true},
			{Root: filepath.Join(agent.RootPath, "agents"), Kind: "subagent", CanEdit: true},
		})
	case "workbuddy":
		return appendMarkdownResources(agent, []markdownResourceSpec{
			{Root: agent.RootPath, RelativePath: "WORKBUDDY.md", Kind: "primary", CanEdit: true},
			{Root: filepath.Join(agent.RootPath, "commands"), Kind: "command", CanEdit: true},
			{Root: filepath.Join(agent.RootPath, "agents"), Kind: "subagent", CanEdit: true},
		})
	case "cursor":
		return appendMarkdownResources(agent, []markdownResourceSpec{
			{Root: filepath.Join(agent.RootPath, "rules"), Kind: "rule", CanEdit: true},
		})
	default:
		return []model.AgentMemoryResource{}, nil
	}
}

func sortAgentResources(skills []model.AgentSkillResource, servers []model.AgentMCPServerResource, memories []model.AgentMemoryResource) {
	sort.SliceStable(skills, func(i, j int) bool {
		if skills[i].AgentKind != skills[j].AgentKind {
			return skills[i].AgentKind < skills[j].AgentKind
		}
		if skills[i].System != skills[j].System {
			return !skills[i].System
		}
		if skills[i].ResourceType != skills[j].ResourceType {
			return skills[i].ResourceType < skills[j].ResourceType
		}
		return strings.ToLower(skills[i].Name) < strings.ToLower(skills[j].Name)
	})
	sort.SliceStable(servers, func(i, j int) bool {
		if servers[i].AgentKind != servers[j].AgentKind {
			return servers[i].AgentKind < servers[j].AgentKind
		}
		return strings.ToLower(servers[i].Name) < strings.ToLower(servers[j].Name)
	})
	sort.SliceStable(memories, func(i, j int) bool {
		if memories[i].AgentKind != memories[j].AgentKind {
			return memories[i].AgentKind < memories[j].AgentKind
		}
		return strings.ToLower(memories[i].RelativePath) < strings.ToLower(memories[j].RelativePath)
	})
}

func packageSkills(agent model.AgentResourceAgent, root string) ([]model.AgentSkillResource, []string) {
	return scanSkillResourceFiles(agent, root, "skill", agent.Name+" skill", agent.Name+" skills", func(entry fs.DirEntry) (bool, bool) {
		enabled := strings.EqualFold(entry.Name(), "SKILL.md")
		if !enabled && !strings.EqualFold(entry.Name(), "SKILL.md.disabled") {
			return false, false
		}
		return true, enabled
	}, nil)
}

func skillResourceFromFile(agent model.AgentResourceAgent, root, path, resourceType string, enabled bool) (model.AgentSkillResource, string) {
	info, err := os.Stat(path)
	if err != nil {
		return model.AgentSkillResource{}, "Unable to inspect " + resourceType + " file " + path + ": " + err.Error()
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return model.AgentSkillResource{}, "Unable to read " + resourceType + " file " + path + ": " + err.Error()
	}
	resourcePath := filepath.Dir(path)
	rel := relativePath(root, resourcePath)
	meta := skillMetadata(content)
	name := firstNonEmpty(meta["name"], strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)), filepath.Base(resourcePath))
	if strings.EqualFold(filepath.Base(path), "SKILL.md") || strings.EqualFold(filepath.Base(path), "SKILL.md.disabled") {
		name = firstNonEmpty(meta["name"], filepath.Base(resourcePath))
	}
	return model.AgentSkillResource{
		AgentKind:    agent.Kind,
		ResourceType: resourceType,
		Name:         name,
		Title:        firstNonEmpty(markdownTitle(content), name),
		Description:  meta["description"],
		Path:         resourcePath,
		RelativePath: rel,
		System:       strings.HasPrefix(filepath.ToSlash(rel), ".system/"),
		Enabled:      enabled,
		CanToggle:    resourceType == "skill" && !strings.HasPrefix(filepath.ToSlash(rel), ".system/"),
		Status:       enabledStatus(enabled),
		SizeBytes:    info.Size(),
		ModifiedAt:   info.ModTime().UTC(),
	}, ""
}

func markdownSkillResources(agent model.AgentResourceAgent, root, resourceType string) ([]model.AgentSkillResource, []string) {
	return scanSkillResourceFiles(agent, root, resourceType, agent.Name+" "+resourceType, agent.Name+" "+resourceType+" resources", func(entry fs.DirEntry) (bool, bool) {
		if !strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
			return false, false
		}
		return true, true
	}, func(item *model.AgentSkillResource, path string) {
		item.Path = path
		item.RelativePath = relativePath(root, path)
		item.CanToggle = false
		item.Status = "configured"
	})
}

func cursorRules(agent model.AgentResourceAgent) ([]model.AgentSkillResource, []string) {
	root := filepath.Join(agent.RootPath, "rules")
	return scanSkillResourceFiles(agent, root, "rule", "Cursor rule", "Cursor rules", func(entry fs.DirEntry) (bool, bool) {
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".md" && ext != ".mdc" {
			return false, false
		}
		return true, true
	}, func(item *model.AgentSkillResource, path string) {
		item.CanToggle = false
		item.Path = path
		item.RelativePath = relativePath(root, path)
	})
}

func scanSkillResourceFiles(agent model.AgentResourceAgent, root, resourceType, inspectLabel, scanLabel string, match func(fs.DirEntry) (bool, bool), update func(*model.AgentSkillResource, string)) ([]model.AgentSkillResource, []string) {
	if stat, err := os.Stat(root); err != nil || !stat.IsDir() {
		return []model.AgentSkillResource{}, nil
	}
	items := []model.AgentSkillResource{}
	warnings := []string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			warnings = append(warnings, "Unable to inspect "+inspectLabel+" path "+path+": "+err.Error())
			return nil
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		include, enabled := match(entry)
		if !include {
			return nil
		}
		item, warning := skillResourceFromFile(agent, root, path, resourceType, enabled)
		if warning != "" {
			warnings = append(warnings, warning)
			return nil
		}
		if update != nil {
			update(&item, path)
		}
		items = append(items, item)
		return nil
	})
	if err != nil {
		warnings = append(warnings, "Unable to scan "+scanLabel+": "+err.Error())
	}
	return items, warnings
}

type markdownResourceSpec struct {
	Root         string
	RelativePath string
	Kind         string
	CanEdit      bool
}

func singleInstructionMemory(agent model.AgentResourceAgent, name, kind string) ([]model.AgentMemoryResource, []string) {
	return appendMarkdownResources(agent, []markdownResourceSpec{{Root: agent.RootPath, RelativePath: name, Kind: kind, CanEdit: true}})
}

func appendMarkdownResources(agent model.AgentResourceAgent, specs []markdownResourceSpec) ([]model.AgentMemoryResource, []string) {
	items := []model.AgentMemoryResource{}
	warnings := []string{}
	for _, spec := range specs {
		if spec.RelativePath != "" {
			path := filepath.Join(spec.Root, filepath.FromSlash(spec.RelativePath))
			if item, ok, warning := memoryResourceFromFile(agent, spec.Root, path, spec.Kind, spec.CanEdit); warning != "" {
				warnings = append(warnings, warning)
			} else if ok {
				items = append(items, item)
			}
			continue
		}
		if stat, err := os.Stat(spec.Root); err != nil || !stat.IsDir() {
			continue
		}
		err := filepath.WalkDir(spec.Root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				warnings = append(warnings, "Unable to inspect "+agent.Name+" markdown path "+path+": "+err.Error())
				return nil
			}
			if entry.IsDir() {
				if entry.Name() == ".git" {
					return filepath.SkipDir
				}
				return nil
			}
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if ext != ".md" && !(spec.Kind == "rule" && ext == ".mdc") {
				return nil
			}
			if item, ok, warning := memoryResourceFromFile(agent, spec.Root, path, spec.Kind, spec.CanEdit); warning != "" {
				warnings = append(warnings, warning)
			} else if ok {
				items = append(items, item)
			}
			return nil
		})
		if err != nil {
			warnings = append(warnings, "Unable to scan "+agent.Name+" markdown resources: "+err.Error())
		}
	}
	return items, warnings
}

func memoryResourceFromFile(agent model.AgentResourceAgent, root, path, kind string, canEdit bool) (model.AgentMemoryResource, bool, string) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return model.AgentMemoryResource{}, false, ""
	}
	if err != nil {
		return model.AgentMemoryResource{}, false, "Unable to inspect memory file " + path + ": " + err.Error()
	}
	if info.IsDir() {
		return model.AgentMemoryResource{}, false, ""
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return model.AgentMemoryResource{}, false, "Unable to read memory file " + path + ": " + err.Error()
	}
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	rel := relativePath(root, path)
	return model.AgentMemoryResource{
		AgentKind:    agent.Kind,
		Name:         name,
		Title:        firstNonEmpty(markdownTitle(content), name),
		Path:         path,
		RelativePath: rel,
		Kind:         kind,
		Preview:      textPreview(content, 260),
		CanEdit:      canEdit,
		SizeBytes:    info.Size(),
		ModifiedAt:   info.ModTime().UTC(),
	}, true, ""
}

func genericMemoryDetail(agent model.AgentResourceAgent, path, relativePath string) (model.AgentMemoryDetail, error) {
	memoryPath, root, kind, err := resolveGenericMemoryPath(agent, path, relativePath)
	if err != nil {
		return model.AgentMemoryDetail{}, err
	}
	item, ok, warning := memoryResourceFromFile(agent, root, memoryPath, kind, true)
	if warning != "" {
		return model.AgentMemoryDetail{}, errors.New(warning)
	}
	if !ok {
		return model.AgentMemoryDetail{}, NotFound("memory file was not found")
	}
	content, err := os.ReadFile(memoryPath)
	if err != nil {
		return model.AgentMemoryDetail{}, err
	}
	return model.AgentMemoryDetail{AgentMemoryResource: item, Content: string(content)}, nil
}

func updateGenericMemory(agent model.AgentResourceAgent, request model.AgentMemoryUpdateRequest) (model.AgentMemoryDetail, error) {
	memoryPath, _, _, err := resolveGenericMemoryPath(agent, request.Path, request.RelativePath)
	if err != nil {
		return model.AgentMemoryDetail{}, err
	}
	if !strings.EqualFold(filepath.Ext(memoryPath), ".md") && !strings.EqualFold(filepath.Ext(memoryPath), ".mdc") {
		return model.AgentMemoryDetail{}, BadRequest("memory path must be a markdown file")
	}
	if err := os.MkdirAll(filepath.Dir(memoryPath), 0o755); err != nil {
		return model.AgentMemoryDetail{}, err
	}
	if err := os.WriteFile(memoryPath, []byte(request.Content), 0o644); err != nil {
		return model.AgentMemoryDetail{}, err
	}
	return genericMemoryDetail(agent, memoryPath, "")
}

func resolveGenericMemoryPath(agent model.AgentResourceAgent, path, rel string) (string, string, string, error) {
	candidate, err := resolvePathInRoot(agent.RootPath, path, rel)
	if err != nil {
		return "", "", "", err
	}
	relRoot := filepath.ToSlash(relativePath(agent.RootPath, candidate))
	if relRoot == "." || strings.HasPrefix(relRoot, "../") || filepath.IsAbs(relRoot) {
		return "", "", "", BadRequest("path is outside the known agent resource root")
	}
	switch agent.Kind {
	case "gemini":
		if relRoot == "GEMINI.md" {
			return candidate, agent.RootPath, "primary", nil
		}
	case "claude":
		return resolveMarkdownKind(agent, candidate, relRoot, "CLAUDE.md")
	case "codebuddy":
		return resolveMarkdownKind(agent, candidate, relRoot, "CODEBUDDY.md")
	case "workbuddy":
		return resolveMarkdownKind(agent, candidate, relRoot, "WORKBUDDY.md")
	case "cursor":
		if strings.HasPrefix(relRoot, "rules/") && (strings.EqualFold(filepath.Ext(candidate), ".md") || strings.EqualFold(filepath.Ext(candidate), ".mdc")) {
			return candidate, filepath.Join(agent.RootPath, "rules"), "rule", nil
		}
	}
	return "", "", "", BadRequest("memory path is not a supported " + agent.Name + " markdown resource")
}

func resolveMarkdownKind(agent model.AgentResourceAgent, candidate, relRoot, primaryName string) (string, string, string, error) {
	if relRoot == primaryName {
		return candidate, agent.RootPath, "primary", nil
	}
	if strings.HasPrefix(relRoot, "commands/") && strings.EqualFold(filepath.Ext(candidate), ".md") {
		return candidate, filepath.Join(agent.RootPath, "commands"), "command", nil
	}
	if strings.HasPrefix(relRoot, "agents/") && strings.EqualFold(filepath.Ext(candidate), ".md") {
		return candidate, filepath.Join(agent.RootPath, "agents"), "subagent", nil
	}
	return "", "", "", BadRequest("memory path is not a supported " + agent.Name + " markdown resource")
}

func setPackageSkillEnabled(agent model.AgentResourceAgent, request model.AgentResourceToggleRequest, enabled bool) error {
	skillsRoot := filepath.Join(agent.RootPath, "skills")
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

func geminiMCPServers(agent model.AgentResourceAgent) ([]model.AgentMCPServerResource, []string) {
	root, exists, err := readJSONSettings(agent.ConfigPath)
	if os.IsNotExist(err) || !exists {
		return []model.AgentMCPServerResource{}, nil
	}
	if err != nil {
		return []model.AgentMCPServerResource{}, []string{"Unable to parse Gemini settings: " + err.Error()}
	}
	rawServers, _ := root["mcpServers"].(map[string]any)
	mcp, _ := root["mcp"].(map[string]any)
	excluded := stringSetFromAny(mcp["excluded"])
	allowed, hasAllowed := stringSetFromAnyWithPresence(mcp["allowed"])
	return mcpResourcesFromMap(agent, rawServers, agent.ConfigPath, func(name string, table map[string]any, status string) (bool, bool, string) {
		enabled := !excluded[name] && (!hasAllowed || allowed[name])
		if !enabled && status == "configured" {
			status = "disabled"
		}
		return enabled, true, status
	}), nil
}

func claudeMCPServers(agent model.AgentResourceAgent) ([]model.AgentMCPServerResource, []string) {
	configPath := claudeMCPConfigPath()
	if stat, err := os.Stat(configPath); err == nil && !stat.IsDir() {
		return jsonMCPServers(model.AgentResourceAgent{Kind: agent.Kind, Name: agent.Name, RootPath: filepath.Dir(configPath), ConfigPath: configPath}, configPath)
	}
	return jsonMCPServers(agent, agent.ConfigPath)
}

func jsonMCPServers(agent model.AgentResourceAgent, configPath string) ([]model.AgentMCPServerResource, []string) {
	root, exists, err := readJSONSettings(configPath)
	if os.IsNotExist(err) || !exists {
		return []model.AgentMCPServerResource{}, nil
	}
	if err != nil {
		return []model.AgentMCPServerResource{}, []string{"Unable to parse " + agent.Name + " MCP config: " + err.Error()}
	}
	rawServers, _ := root["mcpServers"].(map[string]any)
	return mcpResourcesFromMap(agent, rawServers, configPath, func(_ string, table map[string]any, status string) (bool, bool, string) {
		enabled := true
		canToggle := false
		if _, ok := table["enabled"].(bool); ok {
			enabled = boolValue(table["enabled"], true)
			canToggle = true
		} else if _, ok := table["disabled"].(bool); ok {
			enabled = !boolValue(table["disabled"], false)
			canToggle = true
		}
		if !enabled && status == "configured" {
			status = "disabled"
		}
		return enabled && status == "configured", canToggle, status
	}), nil
}

func mcpResourcesFromMap(agent model.AgentResourceAgent, rawServers map[string]any, configPath string, state func(string, map[string]any, string) (bool, bool, string)) []model.AgentMCPServerResource {
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
		enabled, canToggle, status := state(name, table, status)
		servers = append(servers, model.AgentMCPServerResource{
			AgentKind:  agent.Kind,
			Name:       name,
			Command:    command,
			Args:       args,
			EnvKeys:    envKeys,
			ConfigPath: configPath,
			Enabled:    enabled,
			CanToggle:  canToggle,
			Status:     status,
			Warning:    warning,
		})
	}
	return servers
}

func setGeminiMCPEnabled(agent model.AgentResourceAgent, name string, enabled bool) error {
	if err := ensurePathInside(agent.ConfigPath, agent.RootPath); err != nil {
		return err
	}
	root, exists, err := readJSONSettings(agent.ConfigPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if !exists {
		return NotFound("Gemini settings were not found")
	}
	servers, _ := root["mcpServers"].(map[string]any)
	if _, ok := servers[name]; !ok {
		return NotFound("MCP server was not found")
	}
	mcp, _ := root["mcp"].(map[string]any)
	if mcp == nil {
		mcp = map[string]any{}
		root["mcp"] = mcp
	}
	if enabled {
		mcp["excluded"] = removeStringFromAnyList(mcp["excluded"], name)
		if _, ok := mcp["allowed"]; ok {
			mcp["allowed"] = addStringToAnyList(mcp["allowed"], name)
		}
	} else {
		mcp["excluded"] = addStringToAnyList(mcp["excluded"], name)
	}
	return writeJSONSettings(agent.ConfigPath, root)
}

func setJSONMCPEnabled(agent model.AgentResourceAgent, name string, enabled bool) error {
	configPath := agent.ConfigPath
	rootForSafety := agent.RootPath
	if agent.Kind == "claude" {
		if stat, err := os.Stat(claudeMCPConfigPath()); err == nil && !stat.IsDir() {
			configPath = claudeMCPConfigPath()
			rootForSafety = filepath.Dir(configPath)
		}
	}
	if err := ensurePathInside(configPath, rootForSafety); err != nil {
		return err
	}
	root, exists, err := readJSONSettings(configPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if !exists {
		return NotFound("MCP config was not found")
	}
	servers, _ := root["mcpServers"].(map[string]any)
	raw, ok := servers[name]
	if !ok {
		return NotFound("MCP server was not found")
	}
	table, ok := raw.(map[string]any)
	if !ok {
		return BadRequest("MCP server configuration is not editable")
	}
	if _, ok := table["enabled"].(bool); ok {
		table["enabled"] = enabled
		return writeJSONSettings(configPath, root)
	}
	if _, ok := table["disabled"].(bool); ok {
		table["disabled"] = !enabled
		return writeJSONSettings(configPath, root)
	}
	return Unsupported("MCP server does not expose a supported enable field")
}

func readJSONSettings(path string) (map[string]any, bool, error) {
	content, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]any{}, false, err
	}
	if err != nil {
		return nil, false, err
	}
	root, err := parseJSONCObject(content)
	if err != nil {
		return nil, true, err
	}
	return root, true, nil
}

func writeJSONSettings(path string, root map[string]any) error {
	content, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	content = append(content, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o644)
}

func parseJSONCObject(content []byte) (map[string]any, error) {
	if strings.TrimSpace(string(content)) == "" {
		return map[string]any{}, nil
	}
	var value any
	decoder := json.NewDecoder(strings.NewReader(stripJSONComments(string(content))))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New("settings file contains trailing JSON data")
		}
		return nil, err
	}
	root, ok := value.(map[string]any)
	if !ok {
		return nil, errors.New("settings file is not a JSON object")
	}
	return root, nil
}

func stripJSONComments(content string) string {
	var builder strings.Builder
	inString := false
	escaped := false
	for index := 0; index < len(content); index++ {
		ch := content[index]
		if escaped {
			builder.WriteByte(ch)
			escaped = false
			continue
		}
		if inString {
			builder.WriteByte(ch)
			if ch == '\\' {
				escaped = true
			} else if ch == '"' {
				inString = false
			}
			continue
		}
		if ch == '"' {
			inString = true
			builder.WriteByte(ch)
			continue
		}
		if ch == '/' && index+1 < len(content) {
			next := content[index+1]
			if next == '/' {
				index += 2
				for index < len(content) && content[index] != '\n' && content[index] != '\r' {
					index++
				}
				if index < len(content) {
					builder.WriteByte(content[index])
				}
				continue
			}
			if next == '*' {
				index += 2
				for index+1 < len(content) && !(content[index] == '*' && content[index+1] == '/') {
					if content[index] == '\n' || content[index] == '\r' {
						builder.WriteByte(content[index])
					}
					index++
				}
				if index+1 < len(content) {
					index++
				}
				continue
			}
		}
		builder.WriteByte(ch)
	}
	return builder.String()
}

func stringSetFromAny(value any) map[string]bool {
	result, _ := stringSetFromAnyWithPresence(value)
	return result
}

func stringSetFromAnyWithPresence(value any) (map[string]bool, bool) {
	result := map[string]bool{}
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			if text, ok := item.(string); ok {
				result[text] = true
			}
		}
		return result, true
	case []string:
		for _, item := range typed {
			result[item] = true
		}
		return result, true
	default:
		return result, false
	}
}

func addStringToAnyList(value any, text string) []any {
	values := anyStringList(value)
	for _, item := range values {
		if item == text {
			return stringsToAnyList(values)
		}
	}
	values = append(values, text)
	sort.Strings(values)
	return stringsToAnyList(values)
}

func removeStringFromAnyList(value any, text string) []any {
	values := anyStringList(value)
	next := values[:0]
	for _, item := range values {
		if item != text {
			next = append(next, item)
		}
	}
	sort.Strings(next)
	return stringsToAnyList(next)
}

func anyStringList(value any) []string {
	var values []string
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			if text, ok := item.(string); ok {
				values = append(values, text)
			}
		}
	case []string:
		values = append(values, typed...)
	}
	return values
}

func stringsToAnyList(values []string) []any {
	result := make([]any, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	return result
}

func claudeMCPConfigPath() string {
	if path := strings.TrimSpace(os.Getenv("AGENTMETER_CLAUDE_JSON_PATH")); path != "" {
		return filepath.Clean(path)
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".claude.json")
	}
	return ".claude.json"
}
