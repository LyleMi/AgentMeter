package startup

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNeedsPNPMInstall(t *testing.T) {
	frontendDir := t.TempDir()
	if install, err := NeedsPNPMInstall(frontendDir); err != nil || !install {
		t.Fatalf("missing node_modules install = %v, err = %v; want true, nil", install, err)
	}

	nodeModules := filepath.Join(frontendDir, "node_modules")
	mustMkdir(t, nodeModules)
	pnpmLock := mustWrite(t, frontendDir, "pnpm-lock.yaml")
	nodeModulesLock := mustWrite(t, filepath.Join(nodeModules, ".pnpm"), "lock.yaml")

	older := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	newer := older.Add(time.Hour)
	mustChtimes(t, pnpmLock, older)
	mustChtimes(t, nodeModulesLock, newer)
	if install, err := NeedsPNPMInstall(frontendDir); err != nil || install {
		t.Fatalf("current node_modules install = %v, err = %v; want false, nil", install, err)
	}

	mustChtimes(t, pnpmLock, newer.Add(time.Hour))
	if install, err := NeedsPNPMInstall(frontendDir); err != nil || !install {
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
	mustWriteContent(t, repoRoot, "go.mod", "module github.com/LyleMi/AgentMeter\n")
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

func TestFindRepoRootRejectsDifferentModule(t *testing.T) {
	repoRoot := t.TempDir()
	mustWriteContent(t, repoRoot, "go.mod", "module example.com/not-agentmeter\n")
	mustWrite(t, filepath.Join(repoRoot, "frontend"), "package.json")
	nested := filepath.Join(repoRoot, "internal", "startup")
	mustMkdir(t, nested)

	if got, err := FindRepoRoot(nested); err == nil {
		t.Fatalf("repo root = %q, want error for a different module", got)
	}
}

func TestPrepareWebAssetsUsesExistingStaticDirWithoutRepoRoot(t *testing.T) {
	root := t.TempDir()
	distIndex := mustWrite(t, filepath.Join(root, "frontend", "dist"), "index.html")
	t.Chdir(root)

	got, err := PrepareWebAssets("frontend/dist", false)
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.Abs(filepath.Dir(distIndex))
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("static dir = %q; want %q", got, want)
	}
}

func TestPrepareEmbeddedFrontendSourceCopiesBundledFrontend(t *testing.T) {
	cacheBase := t.TempDir()
	t.Setenv("HOME", cacheBase)
	t.Setenv("LOCALAPPDATA", cacheBase)
	t.Setenv("XDG_CACHE_HOME", cacheBase)

	cacheRoot, err := PrepareEmbeddedFrontendSource()
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		filepath.Join(cacheRoot, "frontend", "package.json"),
		filepath.Join(cacheRoot, "frontend", "pnpm-lock.yaml"),
		filepath.Join(cacheRoot, "frontend", "pnpm-workspace.yaml"),
		filepath.Join(cacheRoot, "frontend", "src", "main.ts"),
		filepath.Join(cacheRoot, "frontend", "public", "favicon.png"),
	} {
		if !fileExists(path) {
			t.Fatalf("expected embedded frontend file %s", path)
		}
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

func mustWriteContent(t *testing.T, dir, name, content string) string {
	t.Helper()
	mustMkdir(t, dir)
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
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
