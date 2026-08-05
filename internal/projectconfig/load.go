package projectconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Load reads the config file at path - following its extends chain, if
// any - and returns the fully-merged File. Env-var references (${VAR})
// are resolved along the way, so secrets never need to sit in the file
// itself. This package deliberately doesn't know about gorun.Config - see
// the root package's LoadConfigFile, which converts this into one; kept
// separate so this package doesn't import the one that imports it.
func Load(path string) (*File, error) {
	return loadFile(path, map[string]struct{}{})
}

// loadFile reads and parses a single config file, then - if it declares
// extends - recursively loads and merges its base file underneath it.
// visited tracks the absolute paths already seen in this chain so a
// config that (directly or indirectly) extends itself is reported as an
// error instead of recursing forever.
func loadFile(path string, visited map[string]struct{}) (*File, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolving %q: %w", path, err)
	}
	if _, seen := visited[abs]; seen {
		return nil, fmt.Errorf("extends cycle detected at %s", abs)
	}
	visited[abs] = struct{}{}

	raw, err := os.ReadFile(abs)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", abs, err)
	}

	var f File
	if err := yaml.Unmarshal(interpolateEnv(raw), &f); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", abs, err)
	}

	if f.Extends == "" {
		return &f, nil
	}

	basePath, err := expandPath(f.Extends, filepath.Dir(abs))
	if err != nil {
		return nil, fmt.Errorf("resolving extends %q in %s: %w", f.Extends, abs, err)
	}
	base, err := loadFile(basePath, visited)
	if err != nil {
		return nil, fmt.Errorf("loading extends %q from %s: %w", f.Extends, abs, err)
	}

	return mergeFile(base, &f), nil
}

// expandPath resolves a config-file-supplied path (extends, mainly)
// against the directory of the file that referenced it: ~ expands to the
// user's home directory, an absolute path is used as-is, and anything
// else is treated as relative to baseDir rather than the process's
// current working directory - so `extends: ../shared.yaml` means "next
// to this file", not "next to wherever gorun happened to be invoked
// from".
func expandPath(raw, baseDir string) (string, error) {
	if strings.HasPrefix(raw, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolving ~: %w", err)
		}
		raw = filepath.Join(home, strings.TrimPrefix(raw, "~"))
	}
	if filepath.IsAbs(raw) {
		return raw, nil
	}
	return filepath.Join(baseDir, raw), nil
}

// mergeFile layers overlay on top of base: any field overlay leaves at
// its zero value falls back to base's value. overlay's own Extends is
// intentionally dropped from the result - it's already been consumed by
// the time this is called.
func mergeFile(base, overlay *File) *File {
	merged := *base
	if overlay.Name != "" {
		merged.Name = overlay.Name
	}
	if overlay.Usage != "" {
		merged.Usage = overlay.Usage
	}
	if overlay.AppEnv != "" {
		merged.AppEnv = overlay.AppEnv
	}
	if overlay.RunnerPath != "" {
		merged.RunnerPath = overlay.RunnerPath
	}
	if overlay.ServerEntrypoint != "" {
		merged.ServerEntrypoint = overlay.ServerEntrypoint
	}
	merged.MultiDB = base.MultiDB || overlay.MultiDB
	merged.MySQL = mergeDBSection(base.MySQL, overlay.MySQL)
	merged.PostgreSQL = mergeDBSection(base.PostgreSQL, overlay.PostgreSQL)
	merged.Extends = ""
	return &merged
}

// mergeDBSection layers overlay on top of base field by field. Either
// side may be nil - nil is simply "no settings from this layer".
func mergeDBSection(base, overlay *DBSection) *DBSection {
	if base == nil && overlay == nil {
		return nil
	}
	if base == nil {
		copied := *overlay
		return &copied
	}
	if overlay == nil {
		copied := *base
		return &copied
	}

	merged := *base
	if overlay.Host != "" {
		merged.Host = overlay.Host
	}
	if overlay.Port != "" {
		merged.Port = overlay.Port
	}
	if overlay.User != "" {
		merged.User = overlay.User
	}
	if overlay.Password != "" {
		merged.Password = overlay.Password
	}
	if overlay.DatabaseName != "" {
		merged.DatabaseName = overlay.DatabaseName
	}
	if overlay.Charset != "" {
		merged.Charset = overlay.Charset
	}
	if overlay.ParseTime != nil {
		merged.ParseTime = overlay.ParseTime
	}
	if overlay.Loc != "" {
		merged.Loc = overlay.Loc
	}
	if overlay.SslMode != "" {
		merged.SslMode = overlay.SslMode
	}
	if overlay.TimeZone != "" {
		merged.TimeZone = overlay.TimeZone
	}
	if overlay.MaxOpenConns != 0 {
		merged.MaxOpenConns = overlay.MaxOpenConns
	}
	if overlay.MaxIdleConns != 0 {
		merged.MaxIdleConns = overlay.MaxIdleConns
	}
	if overlay.ConnMaxLifetime != "" {
		merged.ConnMaxLifetime = overlay.ConnMaxLifetime
	}
	if overlay.MigrationPath != "" {
		merged.MigrationPath = overlay.MigrationPath
	}
	if overlay.SeederPath != "" {
		merged.SeederPath = overlay.SeederPath
	}
	return &merged
}
