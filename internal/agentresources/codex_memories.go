package agentresources

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/LyleMi/AgentMeter/internal/model"
)

func MemoryDetail(_ context.Context, agentKind, path, relativePath string) (model.AgentMemoryDetail, error) {
	agent, err := requireAgentForKind(agentKind)
	if err != nil {
		return model.AgentMemoryDetail{}, err
	}
	if agent.Kind != codexKind {
		return genericMemoryDetail(agent, path, relativePath)
	}
	return codexMemoryDetail(agent.RootPath, path, relativePath)
}

func codexMemoryDetail(root, path, relativePath string) (model.AgentMemoryDetail, error) {
	memoryPath, err := resolveCodexMemoryPath(root, path, relativePath)
	if err != nil {
		return model.AgentMemoryDetail{}, err
	}
	rel := relativePathFromRoot(filepath.Join(root, agentResourceMemories), memoryPath)
	if !isCodexMemoryFile(rel) {
		return model.AgentMemoryDetail{}, BadRequest("memory path is not a supported Codex memory file")
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
	return model.AgentMemoryDetail{
		AgentMemoryResource: newCodexMemoryResource(memoryPath, rel, info, content),
		Content:             string(content),
	}, nil
}

func UpdateMemory(_ context.Context, request model.AgentMemoryUpdateRequest) (model.AgentMemoryDetail, error) {
	agent, err := requireAgentForKind(request.AgentKind)
	if err != nil {
		return model.AgentMemoryDetail{}, err
	}
	if agent.Kind != codexKind {
		return updateGenericMemory(agent, request)
	}
	return updateCodexMemory(agent.RootPath, request)
}

func updateCodexMemory(root string, request model.AgentMemoryUpdateRequest) (model.AgentMemoryDetail, error) {
	memoryPath, err := resolveCodexMemoryPath(root, request.Path, request.RelativePath)
	if err != nil {
		return model.AgentMemoryDetail{}, err
	}
	if !strings.EqualFold(filepath.Ext(memoryPath), ".md") {
		return model.AgentMemoryDetail{}, BadRequest("memory path must be a markdown file")
	}
	rel := relativePathFromRoot(filepath.Join(root, agentResourceMemories), memoryPath)
	if !isCodexMemoryFile(rel) {
		return model.AgentMemoryDetail{}, BadRequest("memory path is not a supported Codex memory file")
	}
	if err := os.MkdirAll(filepath.Dir(memoryPath), 0o755); err != nil {
		return model.AgentMemoryDetail{}, err
	}
	if err := os.WriteFile(memoryPath, []byte(request.Content), 0o644); err != nil {
		return model.AgentMemoryDetail{}, err
	}
	return codexMemoryDetail(root, memoryPath, "")
}

type codexMemoryWalker struct {
	root     string
	items    []model.AgentMemoryResource
	warnings []string
}

func codexMemories(root string) ([]model.AgentMemoryResource, []string) {
	if stat, err := os.Stat(root); err != nil || !stat.IsDir() {
		return []model.AgentMemoryResource{}, nil
	}
	walker := codexMemoryWalker{
		root:     root,
		items:    []model.AgentMemoryResource{},
		warnings: []string{},
	}
	if err := filepath.WalkDir(root, walker.visit); err != nil {
		walker.warn("Unable to scan Codex memories", err)
	}
	sort.Slice(walker.items, func(i, j int) bool {
		return strings.ToLower(walker.items[i].RelativePath) < strings.ToLower(walker.items[j].RelativePath)
	})
	return walker.items, walker.warnings
}

func (w *codexMemoryWalker) visit(path string, entry fs.DirEntry, walkErr error) error {
	if walkErr != nil {
		w.warn("Unable to inspect memory path "+path, walkErr)
		return nil
	}
	rel := relativePath(w.root, path)
	if entry.IsDir() {
		if shouldSkipCodexMemoryDir(rel, entry.Name()) {
			return filepath.SkipDir
		}
		return nil
	}
	if !isCodexMemoryFile(rel) {
		return nil
	}
	info, err := entry.Info()
	if err != nil {
		w.warn("Unable to inspect memory file "+path, err)
		return nil
	}
	content, err := os.ReadFile(path)
	if err != nil {
		w.warn("Unable to read memory file "+path, err)
		return nil
	}
	w.items = append(w.items, newCodexMemoryResource(path, rel, info, content))
	return nil
}

func (w *codexMemoryWalker) warn(message string, err error) {
	w.warnings = append(w.warnings, message+": "+err.Error())
}

func newCodexMemoryResource(path, rel string, info fs.FileInfo, content []byte) model.AgentMemoryResource {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return model.AgentMemoryResource{
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
	}
}
