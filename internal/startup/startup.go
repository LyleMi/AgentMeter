package startup

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/LyleMi/AgentMeter/frontend"
)

const (
	ModulePath       = "github.com/LyleMi/AgentMeter"
	DefaultStaticDir = "frontend/dist"
)

func PrepareWebAssets(staticDir string, forceBuild bool) (string, error) {
	staticDir = strings.TrimSpace(staticDir)
	if staticDir == "" {
		staticDir = DefaultStaticDir
	}

	if repoRoot, err := FindRepoRoot("."); err == nil {
		if err := EnsureWebAssets(repoRoot, forceBuild); err != nil {
			return "", err
		}
		return filepath.Abs(resolveStaticDir(repoRoot, staticDir))
	}

	staticPath, err := filepath.Abs(staticDir)
	if err != nil {
		return "", err
	}
	if !forceBuild && HasFrontendBuild(staticPath) {
		return staticPath, nil
	}
	if !isDefaultStaticDir(staticDir) {
		return "", fmt.Errorf("static directory %s is not a built frontend and no AgentMeter source checkout was found", staticPath)
	}

	cacheRoot, err := PrepareEmbeddedFrontendSource()
	if err != nil {
		return "", err
	}
	if err := EnsureWebAssets(cacheRoot, forceBuild); err != nil {
		return "", err
	}
	return filepath.Join(cacheRoot, filepath.FromSlash(DefaultStaticDir)), nil
}

func FindRepoRoot(start string) (string, error) {
	current, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(current)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		current = filepath.Dir(current)
	}

	for {
		if isAgentMeterRepoRoot(current) {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("could not find repository root from %s", start)
		}
		current = parent
	}
}

func HasFrontendBuild(staticDir string) bool {
	return fileExists(filepath.Join(staticDir, "index.html"))
}

func EnsureWebAssets(repoRoot string, forceBuild bool) error {
	frontendDir := filepath.Join(repoRoot, "frontend")

	install, err := NeedsNPMInstall(frontendDir)
	if err != nil {
		return err
	}
	build, err := NeedsFrontendBuild(frontendDir, forceBuild)
	if err != nil {
		return err
	}
	if !install && !build {
		return nil
	}
	if _, err := exec.LookPath("npm"); err != nil {
		return errors.New("npm was not found on PATH")
	}

	if install {
		fmt.Fprintln(os.Stdout, "Installing frontend dependencies...")
		if err := run(frontendDir, "npm", "ci"); err != nil {
			return err
		}
	}
	if build {
		fmt.Fprintln(os.Stdout, "Building frontend...")
		if err := run(frontendDir, "npm", "run", "build"); err != nil {
			return err
		}
	}
	return nil
}

func NeedsNPMInstall(frontendDir string) (bool, error) {
	nodeModules := filepath.Join(frontendDir, "node_modules")
	if !dirExists(nodeModules) {
		return true, nil
	}

	packageLock := filepath.Join(frontendDir, "package-lock.json")
	if !fileExists(packageLock) {
		return false, nil
	}

	nodeModulesLock := filepath.Join(nodeModules, ".package-lock.json")
	if !fileExists(nodeModulesLock) {
		return true, nil
	}

	packageLockInfo, err := os.Stat(packageLock)
	if err != nil {
		return false, err
	}
	nodeModulesLockInfo, err := os.Stat(nodeModulesLock)
	if err != nil {
		return false, err
	}
	return packageLockInfo.ModTime().After(nodeModulesLockInfo.ModTime()), nil
}

func NeedsFrontendBuild(frontendDir string, forceBuild bool) (bool, error) {
	if forceBuild {
		return true, nil
	}

	distIndex := filepath.Join(frontendDir, "dist", "index.html")
	distInfo, err := os.Stat(distIndex)
	if errors.Is(err, os.ErrNotExist) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	distTime := distInfo.ModTime()

	inputs := []string{
		"src",
		"public",
		"index.html",
		"package.json",
		"package-lock.json",
		"tsconfig.json",
		"tsconfig.node.json",
		"vite.config.ts",
	}
	for _, input := range inputs {
		newest, err := newestWriteTime(filepath.Join(frontendDir, input))
		if err != nil {
			return false, err
		}
		if newest.After(distTime) {
			return true, nil
		}
	}
	return false, nil
}

func PrepareEmbeddedFrontendSource() (string, error) {
	cacheRoot, err := EmbeddedFrontendCacheRoot()
	if err != nil {
		return "", err
	}
	frontendDir := filepath.Join(cacheRoot, "frontend")
	marker := filepath.Join(frontendDir, ".agentmeter-source-ready")
	if fileExists(marker) && fileExists(filepath.Join(frontendDir, "package.json")) {
		return cacheRoot, nil
	}
	if err := os.MkdirAll(frontendDir, 0o755); err != nil {
		return "", err
	}
	if err := copyEmbeddedFrontend(frontendDir); err != nil {
		return "", err
	}
	if err := os.WriteFile(marker, []byte(embeddedFrontendCacheKey()), 0o644); err != nil {
		return "", err
	}
	return cacheRoot, nil
}

func EmbeddedFrontendCacheRoot() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil || strings.TrimSpace(base) == "" {
		base = os.TempDir()
	}
	if strings.TrimSpace(base) == "" {
		return "", errors.New("could not determine a cache directory")
	}
	return filepath.Join(base, "AgentMeter", "frontend-src", embeddedFrontendCacheKey()), nil
}

func OpenBrowserAfterDelay(target string, delay time.Duration) {
	go func() {
		time.Sleep(delay)
		if err := OpenBrowser(target); err != nil {
			fmt.Fprintf(os.Stderr, "open browser: %v\n", err)
		}
	}()
}

func OpenBrowser(target string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	case "darwin":
		cmd = exec.Command("open", target)
	default:
		cmd = exec.Command("xdg-open", target)
	}
	return cmd.Start()
}

func resolveStaticDir(base, staticDir string) string {
	if filepath.IsAbs(staticDir) {
		return filepath.Clean(staticDir)
	}
	return filepath.Join(base, filepath.FromSlash(staticDir))
}

func isDefaultStaticDir(staticDir string) bool {
	return filepath.Clean(filepath.FromSlash(staticDir)) == filepath.Clean(filepath.FromSlash(DefaultStaticDir))
}

func isAgentMeterRepoRoot(dir string) bool {
	if !fileExists(filepath.Join(dir, "go.mod")) || !fileExists(filepath.Join(dir, "frontend", "package.json")) {
		return false
	}
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")) == ModulePath
		}
	}
	return false
}

func copyEmbeddedFrontend(targetDir string) error {
	return fs.WalkDir(frontend.SourceFS, ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == "." {
			return nil
		}

		targetPath := filepath.Join(targetDir, filepath.FromSlash(path))
		if entry.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}
		data, err := fs.ReadFile(frontend.SourceFS, path)
		if err != nil {
			return err
		}
		return os.WriteFile(targetPath, data, 0o644)
	})
}

func embeddedFrontendCacheKey() string {
	key := "devel"
	if info, ok := debug.ReadBuildInfo(); ok {
		if version := strings.TrimSpace(info.Main.Version); version != "" && version != "(devel)" {
			key = version
		}
		if sum := strings.TrimSpace(info.Main.Sum); sum != "" {
			key += "-" + sum
		}
	}
	if sourceHash := embeddedFrontendSourceHash(); sourceHash != "" {
		key += "-" + sourceHash
	}
	return sanitizeCacheKey(key)
}

func embeddedFrontendSourceHash() string {
	hash := sha256.New()
	err := fs.WalkDir(frontend.SourceFS, ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		data, err := fs.ReadFile(frontend.SourceFS, path)
		if err != nil {
			return err
		}
		hash.Write([]byte(path))
		hash.Write([]byte{0})
		hash.Write(data)
		hash.Write([]byte{0})
		return nil
	})
	if err != nil {
		return ""
	}
	return hex.EncodeToString(hash.Sum(nil))[:12]
}

func sanitizeCacheKey(value string) string {
	var builder strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '.', r == '-', r == '_':
			builder.WriteRune(r)
		default:
			builder.WriteRune('_')
		}
	}
	if builder.Len() == 0 {
		return "devel"
	}
	return builder.String()
}

func run(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func newestWriteTime(root string) (time.Time, error) {
	info, err := os.Stat(root)
	if errors.Is(err, os.ErrNotExist) {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}
	if !info.IsDir() {
		return info.ModTime(), nil
	}

	var newest time.Time
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.ModTime().After(newest) {
			newest = info.ModTime()
		}
		return nil
	})
	if err != nil {
		return time.Time{}, err
	}
	return newest, nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
