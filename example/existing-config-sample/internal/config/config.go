// Package config stands in for a project's own pre-existing config
// system - the kind almost every real backend already has. gorun neither
// provides nor requires this; it's here to show how a project that
// already has one of these wires it into gorun instead of maintaining
// connection settings twice.
//
// Non-secret settings load from app.yaml. The database password is
// deliberately never read from that file - only from an environment
// variable (DB_MYSQL_PASSWORD), a common convention for exactly this
// reason: a value that sensitive has no business sitting in a committed
// file.
package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config is this project's own resolved configuration - the thing every
// package in a real app would import and pass around, entirely separate
// from gorun.Config.
type Config struct {
	App struct {
		Env string `yaml:"env"`
	} `yaml:"app"`
	Database struct {
		MySQL struct {
			Host   string `yaml:"host"`
			Port   string `yaml:"port"`
			User   string `yaml:"user"`
			DBName string `yaml:"db_name"`
		} `yaml:"mysql"`
	} `yaml:"database"`
}

// Load reads path (app.yaml) into a Config.
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// MySQLPassword reads DB_MYSQL_PASSWORD from the environment - see the
// package doc comment for why this, alone, never comes from app.yaml.
func (c *Config) MySQLPassword() string {
	return os.Getenv("DB_MYSQL_PASSWORD")
}
