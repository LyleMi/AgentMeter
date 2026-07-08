package agentresources

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/LyleMi/AgentMeter/internal/model"
)

func setSkillMarkdownEnabled(skillsRoot string, request model.AgentResourceToggleRequest, enabled bool) error {
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
		return renameDisabledSkillFile(active, disabled)
	}
	return renameActiveSkillFile(active, disabled)
}

func renameDisabledSkillFile(active, disabled string) error {
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

func renameActiveSkillFile(active, disabled string) error {
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
