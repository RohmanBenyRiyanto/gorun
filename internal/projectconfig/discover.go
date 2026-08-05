package projectconfig

import (
	"os"
	"path/filepath"
)

// ConfigRelPath is where a project's config lives, relative to the
// project root - the file gorun setup writes and Discover looks for.
const ConfigRelPath = ".gorun/config.yaml"

// Discover walks upward from startDir looking for a .gorun/config.yaml,
// the same way git finds .git/ or the Go toolchain finds go.mod: check
// the current directory, then its parent, and so on up to the filesystem
// root. Returns the found path and true, or "" and false if startDir
// isn't inside a gorun project at all.
func Discover(startDir string) (string, bool) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", false
	}

	for {
		candidate := filepath.Join(dir, ConfigRelPath)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, true
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}
