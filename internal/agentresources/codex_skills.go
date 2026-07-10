package agentresources

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/LyleMi/AgentMeter/internal/model"
)

type codexSkillScan struct {
	root     string
	items    []model.AgentSkillResource
	warnings []string
}

func codexSkills(root string) ([]model.AgentSkillResource, []string) {
	if stat, err := os.Stat(root); err != nil || !stat.IsDir() {
		return []model.AgentSkillResource{}, nil
	}
	scan := codexSkillScan{root: root, items: []model.AgentSkillResource{}, warnings: []string{}}
	if err := filepath.WalkDir(root, scan.visit); err != nil {
		scan.warnings = append(scan.warnings, "Unable to scan Codex skills: "+err.Error())
	}
	sort.Slice(scan.items, func(i, j int) bool {
		if scan.items[i].System != scan.items[j].System {
			return !scan.items[i].System
		}
		return strings.ToLower(scan.items[i].Name) < strings.ToLower(scan.items[j].Name)
	})
	return scan.items, scan.warnings
}

func (s *codexSkillScan) visit(path string, entry fs.DirEntry, walkErr error) error {
	if walkErr != nil {
		s.warnings = append(s.warnings, "Unable to inspect skill path "+path+": "+walkErr.Error())
		return nil
	}
	if entry.IsDir() {
		if entry.Name() == ".git" {
			return filepath.SkipDir
		}
		return nil
	}
	enabled, isSkill := codexSkillFileState(entry.Name())
	if !isSkill {
		return nil
	}
	item, err := readCodexSkill(s.root, path, entry, enabled)
	if err != nil {
		s.warnings = append(s.warnings, err.Error())
		return nil
	}
	s.items = append(s.items, item)
	return nil
}

func codexSkillFileState(name string) (bool, bool) {
	if strings.EqualFold(name, "SKILL.md") {
		return true, true
	}
	return false, strings.EqualFold(name, "SKILL.md.disabled")
}

func readCodexSkill(root, path string, entry fs.DirEntry, enabled bool) (model.AgentSkillResource, error) {
	info, err := entry.Info()
	if err != nil {
		return model.AgentSkillResource{}, fmt.Errorf("Unable to inspect skill file %s: %w", path, err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return model.AgentSkillResource{}, fmt.Errorf("Unable to read skill file %s: %w", path, err)
	}
	dir := filepath.Dir(path)
	rel := relativePath(root, dir)
	meta := skillMetadata(content)
	name := firstNonEmpty(meta["name"], filepath.Base(dir))
	system := strings.HasPrefix(filepath.ToSlash(rel), ".system/")
	return model.AgentSkillResource{
		AgentKind:    codexKind,
		ResourceType: "skill",
		Name:         name,
		Title:        firstNonEmpty(markdownTitle(content), name),
		Description:  meta["description"],
		Path:         dir,
		RelativePath: rel,
		System:       system,
		Enabled:      enabled,
		CanToggle:    !system,
		Status:       enabledStatus(enabled),
		SizeBytes:    info.Size(),
		ModifiedAt:   info.ModTime().UTC(),
	}, nil
}

func skillMetadata(content []byte) map[string]string {
	meta := map[string]string{}
	lines := skillFrontmatterLines(content)
	if lines == nil {
		return meta
	}
	for index := 1; index < len(lines); index++ {
		if strings.TrimSpace(lines[index]) == "---" {
			break
		}
		if key, value, ok := skillMetadataEntry(lines, &index); ok {
			meta[key] = value
		}
	}
	return meta
}

func skillFrontmatterLines(content []byte) []string {
	trimmed := bytes.TrimLeft(content, "\xef\xbb\xbf\r\n\t ")
	if !bytes.HasPrefix(trimmed, []byte("---")) {
		return nil
	}
	return strings.Split(strings.ReplaceAll(string(trimmed), "\r\n", "\n"), "\n")
}

func skillMetadataEntry(lines []string, index *int) (string, string, bool) {
	key, value, ok := strings.Cut(lines[*index], ":")
	key = strings.TrimSpace(key)
	if !ok || key == "" {
		return "", "", false
	}
	cleanValue := strings.TrimSpace(value)
	if cleanValue == "|" || cleanValue == ">" {
		return key, skillMetadataBlock(lines, index), true
	}
	return key, strings.Trim(cleanValue, `"'`), true
}

func skillMetadataBlock(lines []string, index *int) string {
	var block []string
	for *index+1 < len(lines) {
		next := lines[*index+1]
		trimmed := strings.TrimSpace(next)
		if trimmed == "---" || (trimmed != "" && !strings.HasPrefix(next, " ") && !strings.HasPrefix(next, "\t")) {
			break
		}
		block = append(block, trimmed)
		*index++
	}
	return strings.Join(nonEmptyStrings(block), " ")
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
