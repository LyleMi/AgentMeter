package privacy

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func readOptionalFile(path string) ([]byte, bool, error) {
	content, err := os.ReadFile(path)
	if err == nil {
		return content, true, nil
	}
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	return nil, false, err
}

func writeUpdatedConfig(path string, original, updated []byte, exists bool, now func() time.Time) (string, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	perm := os.FileMode(0o644)
	var backupPath string
	if exists {
		stat, err := os.Stat(path)
		if err != nil {
			return "", err
		}
		perm = stat.Mode().Perm()
		backupPath = backupConfigPath(path, callNow(now))
		if err := os.WriteFile(backupPath, original, perm); err != nil {
			return "", err
		}
	}
	if err := os.WriteFile(path, updated, perm); err != nil {
		return "", err
	}
	return backupPath, nil
}

func backupConfigPath(path string, now time.Time) string {
	stamp := now.UTC().Format("20060102T150405.000000000Z")
	return filepath.Join(filepath.Dir(path), fmt.Sprintf("%s.%s.bak", filepath.Base(path), stamp))
}

func callNow(now func() time.Time) time.Time {
	if now != nil {
		return now()
	}
	return time.Now()
}
