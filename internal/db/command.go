package db

import (
	"context"

	"github.com/RohmanBenyRiyanto/gorun/internal/engine"
	"github.com/urfave/cli/v3"
)

// Command builds the "db" command group: create/drop/list/status/truncate,
// plus its own help subcommand.
func Command(config *engine.Config) *cli.Command {
	return &cli.Command{
		Name:  "db",
		Usage: "Database management commands",
		Commands: []*cli.Command{
			{
				Name:    "help",
				Aliases: []string{"h"},
				Usage:   "Show database commands help",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					NewDBHelper().ShowHelp()
					return nil
				},
			},
			{
				Name:  "create",
				Usage: "Create a new database",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return NewCreateCommand(config).Handle(ctx, cmd)
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "name",
						Aliases: []string{"n"},
						Usage:   "Specify the database name",
					},
					&cli.StringFlag{
						Name:    "type",
						Aliases: []string{"t"},
						Usage:   "Specify database type (mysql/postgresql)",
					},
					&cli.StringFlag{
						Name:  "charset",
						Usage: "Specify character set (MySQL only)",
					},
					&cli.StringFlag{
						Name:  "collation",
						Usage: "Specify collation",
					},
					&cli.StringFlag{
						Name:  "encoding",
						Usage: "Specify encoding (PostgreSQL only)",
					},
				},
			},
			{
				Name:  "drop",
				Usage: "Drop an existing database",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return NewDropCommand(config).Handle(ctx, cmd)
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "name",
						Aliases: []string{"n"},
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
				Name:    "list",
				Aliases: []string{"ls"},
				Usage:   "List all databases",
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
				Name:  "status",
				Usage: "Show database connection status",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return NewStatusCommand(config).Handle(ctx, cmd)
				},
			},
			{
				Name:  "truncate",
				Usage: "Truncate all tables in a database",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return NewTruncateCommand(config).Handle(ctx, cmd)
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "name",
						Aliases: []string{"n"},
						Usage:   "Specify the database name",
					},
				},
			},
		},
	}
}
