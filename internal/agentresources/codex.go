package agentresources

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
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
	root := codexRoot()
	agent := model.AgentResourceAgent{
		Kind:       codexKind,
		Name:       codexName,
		RootPath:   root,
		ConfigPath: filepath.Join(root, "config.toml"),
		Warnings:   []string{},
	}
	if stat, err := os.Stat(root); err == nil && stat.IsDir() {
		agent.Exists = true
	} else if err != nil {
		agent.Warnings = append(agent.Warnings, "Codex home is not available: "+err.Error())
	}

	result := model.AgentResourceOverview{
		Agents:   []model.AgentResourceAgent{agent},
		Warnings: append([]string{}, agent.Warnings...),
	}
	if !agent.Exists {
		result.Skills = []model.AgentSkillResource{}
		result.MCPServers = []model.AgentMCPServerResource{}
		result.Memories = []model.AgentMemoryResource{}
		return result, nil
	}

	var warnings []string
	result.Skills, warnings = codexSkills(filepath.Join(root, "skills"))
	result.Warnings = append(result.Warnings, warnings...)
	result.MCPServers, warnings = codexMCPServers(agent.ConfigPath)
	result.Warnings = append(result.Warnings, warnings...)
	result.Memories, warnings = codexMemories(filepath.Join(root, "memories"))
	result.Warnings = append(result.Warnings, warnings...)
	return result, nil
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
	var items []model.AgentSkillResource
	var warnings []string
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
		if !strings.EqualFold(entry.Name(), "SKILL.md") {
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
		servers = append(servers, model.AgentMCPServerResource{
			AgentKind:  codexKind,
			Name:       name,
			Command:    command,
			Args:       args,
			EnvKeys:    envKeys,
			ConfigPath: configPath,
			Enabled:    status == "configured",
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
	var items []model.AgentMemoryResource
	var warnings []string
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
