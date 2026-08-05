package engine

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// GetProjectRoot walks up from the current working directory to find the
// nearest ancestor containing a go.mod file.
func GetProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached root
		}
		dir = parent
	}
	return "", errors.New("go.mod not found")
}

// GetGoModuleName reads the module path out of the project's go.mod.
func GetGoModuleName() (string, error) {
	root, err := GetProjectRoot()
	if err != nil {
		return "", err
	}
	modFile := filepath.Join(root, "go.mod")

	file, err := os.Open(modFile)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.Fields(line)[1], nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to read go.mod: %w", err)
	}
	return "", errors.New("module name not found")
}

// GetAppEnv reads APP_ENV out of a .env file in the current working
// directory, or "" if there isn't one. This is separate from
// Config.AppEnv - it's used by the display/help commands (project info,
// banners), not by anything that gates behavior.
func GetAppEnv() string {
	envMap, err := ReadEnvFile(".env")
	if err != nil {
		return ""
	}
	return envMap["APP_ENV"]
}

// GetCallerScript returns the invoking binary's base name (os.Args[0]).
func GetCallerScript() string {
	return filepath.Base(os.Args[0])
}

// GetBaseMigrationPathFromEnv reads MIGRATION_PATH from the project's
// .env file, defaulting to "database/migrations" under the project root
// if unset. Relative values are resolved against the project root.
func GetBaseMigrationPathFromEnv() (string, error) {
	root, err := GetProjectRoot()
	if err != nil {
		return "", err
	}
	envMap, _ := ReadEnvFile(filepath.Join(root, ".env"))

	path := envMap["MIGRATION_PATH"]
	if path == "" {
		return filepath.Join(root, "database/migrations"), nil
	}
	if filepath.IsAbs(path) {
		return path, nil
	}
	return filepath.Join(root, path), nil
}

// GetBaseSeederPathFromEnv reads SEEDER_PATH from the project's .env file,
// defaulting to "database/seeders" under the project root if unset.
// Relative values are resolved against the project root.
func GetBaseSeederPathFromEnv() (string, error) {
	root, err := GetProjectRoot()
	if err != nil {
		return "", err
	}
	envMap, _ := ReadEnvFile(filepath.Join(root, ".env"))

	path := envMap["SEEDER_PATH"]
	if path == "" {
		return filepath.Join(root, "database/seeders"), nil
	}
	if filepath.IsAbs(path) {
		return path, nil
	}
	return filepath.Join(root, path), nil
}

// NormalizePath resolves path to an absolute, cleaned form.
func NormalizePath(path string) (string, error) {
	if !filepath.IsAbs(path) {
		abs, err := filepath.Abs(path)
		if err != nil {
			return "", err
		}
		path = abs
	}
	return filepath.Clean(path), nil
}

// ReadEnvFile does a minimal parse of a .env-style file (KEY=value,
// "#" comments, quoted values) into a map. Missing/unreadable files return
// an empty map plus the underlying error - callers generally treat that
// as "no overrides" rather than a hard failure.
func ReadEnvFile(path string) (map[string]string, error) {
	env := make(map[string]string)

	file, err := os.Open(path)
	if err != nil {
		return env, err
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(strings.SplitN(parts[1], "#", 2)[0])
		value = strings.Trim(value, `"`)
		env[key] = value
	}
	return env, scanner.Err()
}

// RunCommand runs cmdStr through the shell (bash -c, or cmd /C on
// Windows) in dir (defaulting to the current working directory), with env
// appended to the inherited environment, and returns its captured
// stdout/stderr.
func RunCommand(cmdStr string, dir string, env []string) (stdout string, stderr string, err error) {
	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", cmdStr)
	} else {
		cmd = exec.Command("bash", "-c", cmdStr)
	}

	if dir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", "", fmt.Errorf("failed to get working directory: %w", err)
		}
		dir = wd
	}
	cmd.Dir = dir

	if env != nil {
		cmd.Env = append(os.Environ(), env...)
	}

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err = cmd.Run()
	return strings.TrimSpace(outBuf.String()), strings.TrimSpace(errBuf.String()), err
}
