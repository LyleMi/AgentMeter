package app

import (
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/LyleMi/AgentMeter/internal/model"
)

func (a *App) GetSourceStorage() (model.SourceStorage, error) {
	if err := a.ensureReady(); err != nil {
		return model.SourceStorage{}, err
	}
	settings, err := a.GetSettings()
	if err != nil {
		return model.SourceStorage{}, err
	}
	return scanSourceDirectories(settings.SourceEntries), nil
}

func scanSourceDirectories(entries []model.SourceEntry) model.SourceStorage {
	result := model.SourceStorage{
		Directories: make([]model.SourceDirectoryStorage, 0, len(entries)),
		ScannedAt:   time.Now().UTC(),
	}
	for _, entry := range entries {
		directory := scanSourceDirectory(entry)
		result.Directories = append(result.Directories, directory)
		result.TotalSizeBytes += directory.SizeBytes
		result.TotalFileCount += directory.FileCount
	}
	return result
}

func scanSourceDirectory(entry model.SourceEntry) model.SourceDirectoryStorage {
	result := model.SourceDirectoryStorage{
		Path:    entry.Path,
		Label:   entry.Label,
		Enabled: entry.Enabled,
	}
	info, err := os.Stat(entry.Path)
	if err != nil {
		if !os.IsNotExist(err) {
			result.Error = err.Error()
		}
		return result
	}
	if !info.IsDir() {
		result.Error = "source path is not a directory"
		return result
	}
	result.Exists = true
	_ = filepath.WalkDir(entry.Path, func(path string, item fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if result.Error == "" {
				result.Error = walkErr.Error()
			}
			return nil
		}
		if item.IsDir() {
			return nil
		}
		itemInfo, err := item.Info()
		if err != nil {
			if result.Error == "" {
				result.Error = err.Error()
			}
			return nil
		}
		result.SizeBytes += itemInfo.Size()
		result.FileCount++
		return nil
	})
	return result
}
