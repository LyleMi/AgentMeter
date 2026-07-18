package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LyleMi/AgentMeter/internal/model"
)

func TestScanSourceDirectoriesReportsSizeFilesAndState(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "sessions", "2026")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "config.json"), []byte("12345"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "session.jsonl"), []byte("1234567"), 0o644); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(root, "missing")

	result := scanSourceDirectories([]model.SourceEntry{
		{Path: root, Label: "Codex", Enabled: true},
		{Path: missing, Label: "Missing", Enabled: false},
	})

	if result.TotalSizeBytes != 12 || result.TotalFileCount != 2 {
		t.Fatalf("totals = %d bytes, %d files", result.TotalSizeBytes, result.TotalFileCount)
	}
	if len(result.Directories) != 2 {
		t.Fatalf("directories = %d", len(result.Directories))
	}
	directory := result.Directories[0]
	if !directory.Exists || directory.SizeBytes != 12 || directory.FileCount != 2 {
		t.Fatalf("directory = %+v", directory)
	}
	if directory.Label != "Codex" || !directory.Enabled {
		t.Fatalf("directory identity = %+v", directory)
	}
	if result.Directories[1].Exists || result.Directories[1].Error != "" {
		t.Fatalf("missing directory = %+v", result.Directories[1])
	}
}

func TestScanSourceDirectoryRejectsFilePath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	if err := os.WriteFile(path, []byte("session"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := scanSourceDirectory(model.SourceEntry{Path: path, Enabled: true})
	if result.Exists || result.Error != "source path is not a directory" {
		t.Fatalf("result = %+v", result)
	}
}
