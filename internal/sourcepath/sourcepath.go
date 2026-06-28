package sourcepath

import (
	"path/filepath"
	"runtime"
	"strings"
)

func Normalize(path string) string {
	cleaned := strings.TrimSpace(path)
	if cleaned == "" {
		return ""
	}
	return filepath.Clean(cleaned)
}

func Equal(left, right string) bool {
	return Key(Normalize(left)) == Key(Normalize(right))
}

func Key(path string) string {
	if runtime.GOOS == "windows" {
		return strings.ToLower(path)
	}
	return path
}

func NormalizeList(paths []string) []string {
	seen := map[string]struct{}{}
	var result []string
	for _, path := range paths {
		cleaned := Normalize(path)
		if cleaned == "" {
			continue
		}
		key := Key(cleaned)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, cleaned)
	}
	return result
}

func DedupeKey(scope, relative string) string {
	return Normalize(scope) + "\x00" + Normalize(relative)
}
