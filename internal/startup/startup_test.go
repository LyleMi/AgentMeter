package startup

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNeedsNPMInstall(t *testing.T) {
	frontendDir := t.TempDir()
	if install, err := NeedsNPMInstall(frontendDir); err != nil || !install {
		t.Fatalf("missing node_modules install = %v, err = %v; want true, nil", install, err)
	}

	nodeModules := filepath.Join(frontendDir, "node_modules")
	mustMkdir(t, nodeModules)
	packageLock := mustWrite(t, frontendDir, "package-lock.json")
	nodeModulesLock := mustWrite(t, nodeModules, ".package-lock.json")

	older := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	newer := older.Add(time.Hour)
	mustChtimes(t, packageLock, older)
	mustChtimes(t, nodeModulesLock, newer)
	if install, err := NeedsNPMInstall(frontendDir); err != nil || install {
		t.Fatalf("current node_modules install = %v, err = %v; want false, nil", install, err)
	}

	mustChtimes(t, packageLock, newer.Add(time.Hour))
	if install, err := NeedsNPMInstall(frontendDir); err != nil || !install {
		t.Fatalf("stale node_modules install = %v, err = %v; want true, nil", install, err)
	}
}

func TestNeedsFrontendBuild(t *testing.T) {
	frontendDir := t.TempDir()
	if build, err := NeedsFrontendBuild(frontendDir, false); err != nil || !build {
		t.Fatalf("missing dist build = %v, err = %v; want true, nil", build, err)
	}

	distIndex := mustWrite(t, filepath.Join(frontendDir, "dist"), "index.html")
	srcFile := mustWrite(t, filepath.Join(frontendDir, "src"), "main.ts")
	packageJSON := mustWrite(t, frontendDir, "package.json")

	base := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	mustChtimes(t, srcFile, base)
	mustChtimes(t, packageJSON, base)
	mustChtimes(t, distIndex, base.Add(time.Hour))
	if build, err := NeedsFrontendBuild(frontendDir, false); err != nil || build {
		t.Fatalf("fresh dist build = %v, err = %v; want false, nil", build, err)
	}

	mustChtimes(t, packageJSON, base.Add(2*time.Hour))
	if build, err := NeedsFrontendBuild(frontendDir, false); err != nil || !build {
		t.Fatalf("stale dist build = %v, err = %v; want true, nil", build, err)
	}

	if build, err := NeedsFrontendBuild(frontendDir, true); err != nil || !build {
		t.Fatalf("forced build = %v, err = %v; want true, nil", build, err)
	}
}

func TestFindRepoRoot(t *testing.T) {
	repoRoot := t.TempDir()
	mustWrite(t, repoRoot, "go.mod")
	mustWrite(t, filepath.Join(repoRoot, "frontend"), "package.json")
	nested := filepath.Join(repoRoot, "internal", "startup")
	mustMkdir(t, nested)

	got, err := FindRepoRoot(nested)
	if err != nil {
		t.Fatal(err)
	}
	if got != repoRoot {
		t.Fatalf("repo root = %q; want %q", got, repoRoot)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, dir, name string) string {
	t.Helper()
	mustMkdir(t, dir)
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func mustChtimes(t *testing.T, path string, ts time.Time) {
	t.Helper()
	if err := os.Chtimes(path, ts, ts); err != nil {
		t.Fatal(err)
	}
}
