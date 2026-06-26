package ingest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindJSONLFilesUsesCodexHomeSourcesWithActiveCopyWinning(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "codex")
	activeDuplicate := filepath.Join(root, "sessions", "project", "duplicate.jsonl")
	archivedDuplicate := filepath.Join(root, "archived_sessions", "project", "duplicate.jsonl")
	archivedOnly := filepath.Join(root, "archived_sessions", "project", "archived-only.jsonl")
	ignored := filepath.Join(root, "sessions", "project", "notes.txt")
	for _, path := range []string{activeDuplicate, archivedDuplicate, archivedOnly, ignored} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("{}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	files, err := findJSONLFiles(root)
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 2 {
		t.Fatalf("files = %v", files)
	}
	got := map[string]bool{}
	for _, file := range files {
		got[file] = true
	}
	if !got[activeDuplicate] {
		t.Fatalf("missing active duplicate: %v", files)
	}
	if !got[archivedOnly] {
		t.Fatalf("missing archived-only file: %v", files)
	}
	if got[archivedDuplicate] {
		t.Fatalf("archived duplicate should be skipped: %v", files)
	}
}

func TestFindJSONLFilesKeepsDirectDirectoryMode(t *testing.T) {
	dir := t.TempDir()
	run := filepath.Join(dir, "logs", "run.jsonl")
	if err := os.MkdirAll(filepath.Dir(run), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(run, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := findJSONLFiles(filepath.Join(dir, "logs"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0] != run {
		t.Fatalf("files = %v", files)
	}
}

func TestFindJSONLFilesUsesClaudeProjectsDirectory(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, ".claude")
	run := filepath.Join(root, "projects", "-workspace-project", "run.jsonl")
	ignored := filepath.Join(root, "todos", "run.jsonl")
	for _, path := range []string{run, ignored} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("{}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	files, err := findJSONLFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0] != run {
		t.Fatalf("files = %v", files)
	}
}

func TestFindJSONLFilesUsesCodeBuddyProjectsDirectory(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, ".codebuddy")
	run := filepath.Join(root, "projects", "d-tools-project", "run.jsonl")
	ignored := filepath.Join(root, "sessions", "run.jsonl")
	for _, path := range []string{run, ignored} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("{}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	files, err := findJSONLFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0] != run {
		t.Fatalf("files = %v", files)
	}
}
