// Command gorun-runner is what the globally-installed gorun binary hands
// `gorun seed ...` off to. Unlike example/full-sample's version (and the
// one `gorun setup` scaffolds by default), this one does NOT call
// gorun.LoadConfigFile - it calls this project's own, pre-existing
// config.Load instead, and maps the result into a gorun.Config by hand.
// That's the whole point of this example: when a project already has a
// config system, gorun.Config can be built straight from it, with zero
// YAML of gorun's own involved in this path at all.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/RohmanBenyRiyanto/gorun"
	mysqlseeders "github.com/RohmanBenyRiyanto/gorun-existing-config-sample/database/seeders/mysql"
	"github.com/RohmanBenyRiyanto/gorun-existing-config-sample/internal/config"
)

func main() {
	appCfg, err := config.Load("app.yaml")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	cfg := gorun.Config{
		MySQL: gorun.DBConnConfig{
			Host:          appCfg.Database.MySQL.Host,
			Port:          appCfg.Database.MySQL.Port,
			User:          appCfg.Database.MySQL.User,
			Password:      appCfg.MySQLPassword(), // env only - see config.go
			DatabaseName:  appCfg.Database.MySQL.DBName,
			MigrationPath: "database/migrations",
			SeederPath:    "database/seeders",
		},
		AppEnv:       appCfg.App.Env,
		MySQLSeeders: mysqlseeders.Registry{},
		Name:         "existing-config-sample",
	}

	cmd := gorun.New(cfg)
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
