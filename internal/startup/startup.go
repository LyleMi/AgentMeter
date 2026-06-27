package startup

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

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
		if fileExists(filepath.Join(current, "go.mod")) && fileExists(filepath.Join(current, "frontend", "package.json")) {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("could not find repository root from %s", start)
		}
		current = parent
	}
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
