package setup

import (
	"context"
	"fmt"
	"os"

	"github.com/RohmanBenyRiyanto/gorun/internal/engine"
	"github.com/RohmanBenyRiyanto/gorun/internal/projectconfig"
	"github.com/urfave/cli/v3"
	"golang.org/x/term"
)

// Command builds `gorun setup`. It's deliberately not part of the
// gorun.New command tree - setup's whole job is producing a Config,
// which gorun.New needs handed to it already built - so cmd/gorun wires
// this in as a special case before Config even exists.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "setup",
		Usage: "Make the current directory a gorun project (writes .gorun/config.yaml)",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "yes",
				Usage: "Skip interactive prompts - answer entirely from flags/env, failing loudly on anything required that's missing",
			},
			&cli.BoolFlag{
				Name:  "interactive",
				Usage: "Force prompts even when stdin isn't detected as a terminal",
			},
			&cli.BoolFlag{
				Name:  "force",
				Usage: "Overwrite an existing .gorun/config.yaml instead of refusing",
			},
			&cli.StringFlag{Name: "name", Usage: "Project name shown in gorun's own help/usage output"},
			&cli.StringFlag{Name: "usage", Usage: "One-line usage banner shown alongside --name"},
			&cli.StringFlag{Name: "app-env", Usage: "Environment name (gates seed run's production guard)"},
			&cli.StringFlag{Name: "extends", Usage: "Path to another config file this one inherits defaults from"},
			&cli.BoolFlag{Name: "multi-db", Usage: "Required alongside both --mysql-host and --postgresql-host - confirms this project deliberately uses both engines"},

			&cli.StringFlag{Name: "mysql-host", Usage: "MySQL: configuring this engine at all is triggered by setting this flag"},
			&cli.StringFlag{Name: "mysql-port", Value: "3306"},
			&cli.StringFlag{Name: "mysql-user"},
			&cli.StringFlag{Name: "mysql-password"},
			&cli.StringFlag{Name: "mysql-database"},
			&cli.StringFlag{Name: "mysql-migration-path", Value: "database/migrations", Usage: "gorun appends the engine name itself (-> database/migrations/mysql) - don't include it here"},
			&cli.StringFlag{Name: "mysql-seeder-path", Value: "database/seeders", Usage: "gorun appends the engine name itself (-> database/seeders/mysql) - don't include it here"},

			&cli.StringFlag{Name: "postgresql-host", Usage: "PostgreSQL: configuring this engine at all is triggered by setting this flag"},
			&cli.StringFlag{Name: "postgresql-port", Value: "5432"},
			&cli.StringFlag{Name: "postgresql-user"},
			&cli.StringFlag{Name: "postgresql-password"},
			&cli.StringFlag{Name: "postgresql-database"},
			&cli.StringFlag{Name: "postgresql-migration-path", Value: "database/migrations", Usage: "gorun appends the engine name itself (-> database/migrations/postgresql) - don't include it here"},
			&cli.StringFlag{Name: "postgresql-seeder-path", Value: "database/seeders", Usage: "gorun appends the engine name itself (-> database/seeders/postgresql) - don't include it here"},
		},
		Action: run,
	}
}

func run(ctx context.Context, cmd *cli.Command) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("resolving current directory: %w", err)
	}

	existing := dir + "/" + projectconfig.ConfigRelPath
	if _, err := os.Stat(existing); err == nil && !cmd.Bool("force") {
		return fmt.Errorf("%s already exists - pass --force to overwrite it", projectconfig.ConfigRelPath)
	}

	if cmd.Bool("yes") && cmd.Bool("interactive") {
		return fmt.Errorf("--yes and --interactive can't both be set")
	}
	interactive := cmd.Bool("interactive") || (!cmd.Bool("yes") && term.IsTerminal(int(os.Stdin.Fd())))

	var a answers
	if interactive {
		a = runWizard()
	} else {
		var err error
		a, err = answersFromFlags(cmd)
		if err != nil {
			return err
		}
	}

	if err := writeConfig(dir, a); err != nil {
		return err
	}
	if err := scaffoldDirs(dir, a); err != nil {
		return err
	}
	if err := scaffoldRunner(dir, a); err != nil {
		return err
	}
	if err := scaffoldWrapper(dir, a); err != nil {
		return err
	}

	engine.PrintSuccess("Wrote %s", projectconfig.ConfigRelPath)
	if a.wantsRunner() {
		engine.PrintInfo("Scaffolded %s - add your seeders there, then `gorun seed run` picks them up.", runnerPath+"/main.go")
		engine.PrintInfo("Scaffolded ./gorun (and gorun.bat for Windows) - a shortcut for `go run ./cmd/gorun-runner`.")
	}
	engine.PrintInfo("Run `gorun db status` to check connectivity, or `gorun help` to see everything else.")
	return nil
}
