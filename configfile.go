package gorun

import (
	"fmt"
	"time"

	"github.com/RohmanBenyRiyanto/gorun/internal/projectconfig"
)

// LoadConfigFile reads a .gorun/config.yaml-style file - following its
// extends chain and resolving ${VAR} references from the environment -
// into a Config. It's the same loader the globally-installed gorun
// binary uses, exposed here so a project-local entrypoint (see
// Config.RunnerPath) can build the exact same Config instead of
// duplicating connection settings by hand:
//
//	cfg, err := gorun.LoadConfigFile(".gorun/config.yaml")
//	cfg.MySQLSeeders = myapp.SeederRegistry{}  // MultiDB, seeders etc.
//	cmd := gorun.New(cfg)
//
// MySQLSeeders/PostgreSQLSeeders are never set by this function - a YAML
// file can't express a Go interface implementation, so wire those in
// yourself afterward.
func LoadConfigFile(path string) (Config, error) {
	f, err := projectconfig.Load(path)
	if err != nil {
		return Config{}, err
	}
	return configFromFile(f)
}

// DiscoverConfigFile walks upward from dir looking for
// .gorun/config.yaml, the same way the globally-installed gorun binary
// does - check the current directory, then its parent, and so on up to
// the filesystem root. Returns the found path and true, or "" and false.
func DiscoverConfigFile(dir string) (string, bool) {
	return projectconfig.Discover(dir)
}

// configFromFile converts a fully-merged projectconfig.File into a
// Config. The only real work is ConnMaxLifetime, a plain string in YAML
// (YAML has no duration type) and a time.Duration here.
func configFromFile(f *projectconfig.File) (Config, error) {
	cfg := Config{
		Name:             f.Name,
		Usage:            f.Usage,
		AppEnv:           f.AppEnv,
		MultiDB:          f.MultiDB,
		RunnerPath:       f.RunnerPath,
		ServerEntrypoint: f.ServerEntrypoint,
	}

	mysql, err := dbConnConfigFromSection(f.MySQL)
	if err != nil {
		return Config{}, fmt.Errorf("mysql: %w", err)
	}
	cfg.MySQL = mysql

	pg, err := dbConnConfigFromSection(f.PostgreSQL)
	if err != nil {
		return Config{}, fmt.Errorf("postgresql: %w", err)
	}
	cfg.PostgreSQL = pg

	return cfg, nil
}

func dbConnConfigFromSection(s *projectconfig.DBSection) (DBConnConfig, error) {
	if s == nil {
		return DBConnConfig{}, nil
	}

	lifetime, err := parseDuration(s.ConnMaxLifetime)
	if err != nil {
		return DBConnConfig{}, fmt.Errorf("conn_max_lifetime %q: %w", s.ConnMaxLifetime, err)
	}

	parseTime := false
	if s.ParseTime != nil {
		parseTime = *s.ParseTime
	}

	return DBConnConfig{
		Host:            s.Host,
		Port:            s.Port,
		User:            s.User,
		Password:        s.Password,
		DatabaseName:    s.DatabaseName,
		Charset:         s.Charset,
		ParseTime:       parseTime,
		Loc:             s.Loc,
		SslMode:         s.SslMode,
		TimeZone:        s.TimeZone,
		MaxOpenConns:    s.MaxOpenConns,
		MaxIdleConns:    s.MaxIdleConns,
		ConnMaxLifetime: lifetime,
		MigrationPath:   s.MigrationPath,
		SeederPath:      s.SeederPath,
	}, nil
}

func parseDuration(raw string) (time.Duration, error) {
	if raw == "" {
		return 0, nil
	}
	return time.ParseDuration(raw)
}
