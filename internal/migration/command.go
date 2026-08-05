package migration

import (
	"context"

	"github.com/RohmanBenyRiyanto/gorun/internal/engine"
	"github.com/urfave/cli/v3"
)

// Flag names match `gorun seed run` so both commands stay consistent.
func dbSelectionFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    "type",
			Aliases: []string{"engine"},
			Usage:   "Database engine type (mysql, postgresql) - skips the engine prompt",
		},
		&cli.StringFlag{
			Name:    "database",
			Aliases: []string{"db"},
			Usage:   "Database name - skips the database-selection prompt",
		},
	}
}

// For subcommands that never connect to a named database, e.g. `make`.
func dbEngineOnlyFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    "type",
			Aliases: []string{"engine"},
			Usage:   "Database engine type (mysql, postgresql) - skips the engine prompt",
		},
	}
}

// Command builds the "migrate" command group: run/status/make/rollback/
// reset/refresh/fresh, plus its own help subcommand.
func Command(config *engine.Config) *cli.Command {
	return &cli.Command{
		Name:    "migrate",
		Aliases: []string{"migration"},
		Usage:   "Database migration commands",
		Commands: []*cli.Command{
			{
				Name:    "help",
				Aliases: []string{"h"},
				Usage:   "Show migration commands help",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					helper := NewMigrationHelper()
					helper.ShowHelp()
					return nil
				},
			},
			{
				Name:    "run",
				Aliases: []string{"migrate"},
				Usage:   "Run pending migrations",
				Flags: append(dbSelectionFlags(),
					&cli.BoolFlag{
						Name:    "force",
						Aliases: []string{"f"},
						Usage:   "Run without confirmation",
					},
					&cli.StringFlag{
						Name:    "path",
						Aliases: []string{"p"},
						Usage:   "Run migrations only from specific path",
					},
					&cli.StringFlag{
						Name:    "file",
						Aliases: []string{"F"},
						Usage:   "Run specific migration file (name without extension)",
					},
					&cli.BoolFlag{
						Name:  "pretend",
						Usage: "Only show SQL to be executed",
					},
					&cli.BoolFlag{
						Name:  "step",
						Usage: "Run as individual steps for partial rollback",
					},
				),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return NewRunCommand(config).Handle(ctx, cmd)
				},
			},
			{
				Name:  "status",
				Usage: "Show migration status",
				Flags: dbSelectionFlags(),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return NewStatusCommand(config).Handle(ctx, cmd)
				},
			},
			{
				Name:  "make",
				Usage: "Create new migration file",
				Flags: append(dbEngineOnlyFlags(),
					&cli.StringFlag{
						Name:    "create",
						Aliases: []string{"c"},
						Usage:   "Create new table",
					},
					&cli.StringFlag{
						Name:    "table",
						Aliases: []string{"t"},
						Usage:   "Modify existing table",
					},
					&cli.StringFlag{
						Name:    "path",
						Aliases: []string{"p"},
						Usage:   "Location to store migration file",
					},
					&cli.BoolFlag{
						Name:  "realpath",
						Usage: "Use absolute path",
					},
					&cli.BoolFlag{
						Name:  "fullpath",
						Usage: "Show full path after creation",
					},
				),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					if cmd.Args().Len() == 0 {
						return cli.Exit("migration name is required", 1)
					}
					return NewMakeCommand(config).Handle(ctx, cmd)
				},
			},
			{
				Name:  "rollback",
				Usage: "Rollback the last migration",
				Flags: append(dbSelectionFlags(),
					&cli.IntFlag{
						Name:    "step",
						Aliases: []string{"s"},
						Usage:   "Number of batches to rollback",
						Value:   1,
					},
					&cli.StringFlag{
						Name:    "path",
						Aliases: []string{"p"},
						Usage:   "Limit rollback to specific path",
					},
					&cli.BoolFlag{
						Name:    "force",
						Aliases: []string{"f"},
						Usage:   "Run without confirmation",
					},
				),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return NewRollbackCommand(config).Handle(ctx, cmd)
				},
			},
			{
				Name:  "reset",
				Usage: "Rollback all migrations",
				Flags: append(dbSelectionFlags(),
					&cli.BoolFlag{
						Name:    "force",
						Aliases: []string{"f"},
						Usage:   "Run without confirmation",
					},
				),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return NewResetCommand(config).Handle(ctx, cmd)
				},
			},
			{
				Name:  "refresh",
				Usage: "Reset and re-run all migrations",
				Flags: append(dbSelectionFlags(),
					&cli.BoolFlag{
						Name:  "seed",
						Usage: "Run seeders after migration",
					},
					&cli.IntFlag{
						Name:    "step",
						Aliases: []string{"s"},
						Usage:   "Run migrations per step",
					},
					&cli.BoolFlag{
						Name:    "force",
						Aliases: []string{"f"},
						Usage:   "Run without confirmation",
					},
					&cli.StringFlag{
						Name:    "file",
						Aliases: []string{"F"},
						Usage:   "Run specific migration file (name without extension)",
					},
				),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return NewRefreshCommand(config).Handle(ctx, cmd)
				},
			},
			{
				Name:  "fresh",
				Usage: "Drop all tables and re-run migrations",
				Flags: append(dbSelectionFlags(),
					&cli.BoolFlag{
						Name:  "seed",
						Usage: "Run seeders after migration",
					},
					&cli.BoolFlag{
						Name:  "drop-views",
						Usage: "Drop all views",
					},
					&cli.BoolFlag{
						Name:  "drop-types",
						Usage: "Drop custom types (PostgreSQL)",
					},
					&cli.BoolFlag{
						Name:    "force",
						Aliases: []string{"f"},
						Usage:   "Run without confirmation",
					},
					&cli.StringFlag{
						Name:    "file",
						Aliases: []string{"F"},
						Usage:   "Run specific migration file (name without extension)",
					},
				),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return NewFreshCommand(config).Handle(ctx, cmd)
				},
			},
		},
	}
}
