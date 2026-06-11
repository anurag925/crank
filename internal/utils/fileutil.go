package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EnsureDir creates a directory (and any parents) if it does not exist.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

// WriteFile writes content to path, creating parent directories as needed.
func WriteFile(path, content string) error {
	if err := EnsureDir(filepath.Dir(path)); err != nil {
		return fmt.Errorf("create dir %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// PathExists reports whether the given filesystem path exists.
func PathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// IsEmptyDir reports whether the given directory exists and is empty.
func IsEmptyDir(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, err
	}
	return len(entries) == 0, nil
}

// ToPackageName converts a project name like "my-app" or "my_app" into a Go-friendly
// package segment such as "myapp". It lower-cases the result and strips characters that are
// not valid Go identifiers.
func ToPackageName(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_' || r == ' ':
			// skip separators
		}
	}
	return b.String()
}
