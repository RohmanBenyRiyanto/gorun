package table

import (
	"context"

	"github.com/RohmanBenyRiyanto/gorun/internal/engine"
	"github.com/urfave/cli/v3"
)

// Command builds the "table" command group: create/drop/list/truncate,
// plus its own help subcommand.
func Command(config *engine.Config) *cli.Command {
	return &cli.Command{
		Name:  "table",
		Usage: "Table management commands",
		Commands: []*cli.Command{
			{
				Name:    "help",
				Aliases: []string{"h"},
				Usage:   "Show table commands help",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					NewTableHelper().ShowHelp()
					return nil
				},
			},
			{
				Name:  "create",
				Usage: "Create a new table",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return NewCreateCommand(config).Handle(ctx, cmd)
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "name",
						Aliases: []string{"n"},
						Usage:   "Specify the table name",
					},
					&cli.StringFlag{
						Name:    "database",
						Aliases: []string{"d"},
						Usage:   "Specify the database name",
					},
					&cli.StringFlag{
						Name:    "type",
						Aliases: []string{"t"},
						Usage:   "Specify database type (mysql/postgresql)",
					},
					&cli.StringFlag{
						Name:  "schema",
						Usage: "Path to table schema file",
					},
					&cli.BoolFlag{
						Name:    "force",
						Aliases: []string{"f"},
						Usage:   "Overwrite if table exists",
					},
				},
			},
			{
				Name:  "drop",
				Usage: "Drop an existing table",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return NewDropCommand(config).Handle(ctx, cmd)
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "name",
						Aliases: []string{"n"},
						Usage:   "Specify the table name",
					},
					&cli.StringFlag{
						Name:    "database",
						Aliases: []string{"d"},
						Usage:   "Specify the database name",
					},
					&cli.StringFlag{
						Name:    "type",
						Aliases: []string{"t"},
						Usage:   "Specify database type (mysql/postgresql)",
					},
					&cli.BoolFlag{
						Name:    "force",
						Aliases: []string{"f"},
						Usage:   "Skip confirmation prompt",
					},
				},
			},
			{
				Name:  "list",
				Usage: "List tables in a database",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return NewListCommand(config).Handle(ctx, cmd)
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "type",
						Aliases: []string{"t"},
						Usage:   "Specify database type (mysql/postgresql)",
					},
				},
			},
			{
				Name:  "truncate",
				Usage: "Truncate table(s)",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return NewTruncateCommand(config).Handle(ctx, cmd)
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "name",
						Aliases: []string{"n"},
						Usage:   "Specify the table name (for non-interactive mode)",
					},
					&cli.StringFlag{
						Name:    "database",
						Aliases: []string{"d"},
						Usage:   "Specify the database name",
					},
					&cli.StringFlag{
						Name:    "type",
						Aliases: []string{"t"},
						Usage:   "Specify database type (mysql/postgresql)",
					},
					&cli.BoolFlag{
						Name:    "force",
						Aliases: []string{"f"},
						Usage:   "Skip confirmation prompt",
					},
				},
			},
		},
	}
}
