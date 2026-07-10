package agentresources

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/LyleMi/AgentMeter/internal/model"
)

type skillResourceScan struct {
	Agent        model.AgentResourceAgent
	Root         string
	ResourceType string
	InspectLabel string
	ScanLabel    string
	Match        func(fs.DirEntry) (bool, bool)
	Update       func(*model.AgentSkillResource, string)
}

type skillResourceWalker struct {
	scan     skillResourceScan
	items    []model.AgentSkillResource
	warnings []string
}

func scanSkillResourceFiles(scan skillResourceScan) ([]model.AgentSkillResource, []string) {
	if stat, err := os.Stat(scan.Root); err != nil || !stat.IsDir() {
		return []model.AgentSkillResource{}, nil
	}
	walker := skillResourceWalker{
		scan:     scan,
		items:    []model.AgentSkillResource{},
		warnings: []string{},
	}
	if err := filepath.WalkDir(scan.Root, walker.visit); err != nil {
		walker.warn("Unable to scan "+scan.ScanLabel, err)
	}
	return walker.items, walker.warnings
}

func (w *skillResourceWalker) visit(path string, entry fs.DirEntry, walkErr error) error {
	if walkErr != nil {
		w.warn("Unable to inspect "+w.scan.InspectLabel+" path "+path, walkErr)
		return nil
	}
	if entry.IsDir() {
		if entry.Name() == ".git" {
			return filepath.SkipDir
		}
		return nil
	}
	include, enabled := w.scan.Match(entry)
	if !include {
		return nil
	}
	item, warning := skillResourceFromFile(w.scan.Agent, w.scan.Root, path, w.scan.ResourceType, enabled)
	if warning != "" {
		w.warnings = append(w.warnings, warning)
		return nil
	}
	if w.scan.Update != nil {
		w.scan.Update(&item, path)
	}
	w.items = append(w.items, item)
	return nil
}

func (w *skillResourceWalker) warn(message string, err error) {
	w.warnings = append(w.warnings, message+": "+err.Error())
}
