package seed

import (
	"context"

	"github.com/RohmanBenyRiyanto/gorun/internal/engine"
	"github.com/urfave/cli/v3"
)

// Command builds the "seed" command group: run/make/list, plus its own
// help subcommand.
func Command(config *engine.Config) *cli.Command {
	return &cli.Command{
		Name:    "seed",
		Aliases: []string{"seeder"},
		Usage:   "Database seeding operations",
		Commands: []*cli.Command{
			{
				Name:    "help",
				Aliases: []string{"h"},
				Usage:   "Show migration commands help",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					helper := NewSeedHelper()
					helper.ShowHelp()
					return nil
				},
			},
			{
				Name:    "run",
				Aliases: []string{"execute"},
				Usage:   "Run database seeders",
				// Flags are read in RunCommand.Handle; see script_seeder.go for how they're applied.
				Flags: []cli.Flag{
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
					&cli.StringFlag{
						Name:    "class",
						Aliases: []string{"c"},
						Usage:   "Run specific seeder class",
					},
					&cli.StringFlag{
						Name:    "seeder",
						Aliases: []string{"s"},
						Usage:   "Same as --class",
					},
					&cli.BoolFlag{
						Name:    "force",
						Aliases: []string{"f"},
						Usage:   "Force run against production (app.env=prod/production)",
					},
					&cli.BoolFlag{
						Name:    "transaction",
						Aliases: []string{"t"},
						Usage:   "Wrap each seeder's Run in a DB transaction",
						Value:   true,
					},
					&cli.BoolFlag{
						Name:  "stop-on-error",
						Usage: "Stop on the first failing seeder (false: run all, report combined failures at the end)",
						Value: true,
					},
					&cli.StringSliceFlag{
						Name:    "only",
						Aliases: []string{"o"},
						Usage:   "Run only these seeders (comma separated) - ignored if empty",
					},
					&cli.StringSliceFlag{
						Name:    "except",
						Aliases: []string{"e"},
						Usage:   "Exclude these seeders (comma separated) - ignored if --only is set",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return NewRunCommand(config).Handle(ctx, cmd)
				},
			},
			{
				Name:    "make",
				Aliases: []string{"create"},
				Usage:   "Create new seeder file",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "path",
						Aliases: []string{"p"},
						Usage:   "Location to store seeder file",
					},
					&cli.BoolFlag{
						Name:  "realpath",
						Usage: "Use absolute path",
					},
					&cli.BoolFlag{
						Name:  "fullpath",
						Usage: "Show full path after creation",
					},
					&cli.BoolFlag{
						Name:  "model",
						Usage: "Generate model with seeder",
					},
					&cli.StringFlag{
						Name:  "table",
						Usage: "Specify table name for model",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					if cmd.Args().Len() == 0 {
						return cli.Exit("seeder name is required", 1)
					}
					return NewMakeCommand(config).Handle(ctx, cmd)
				},
			},
			{
				Name:    "list",
				Aliases: []string{"ls"},
				Usage:   "List available seeders",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "details",
						Usage: "Show detailed information",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return NewListCommand(config).Handle(ctx, cmd)
				},
			},
		},
	}
}
