package gorun

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/RohmanBenyRiyanto/gorun/internal/app"
	"github.com/RohmanBenyRiyanto/gorun/internal/db"
	"github.com/RohmanBenyRiyanto/gorun/internal/engine"
	"github.com/RohmanBenyRiyanto/gorun/internal/info"
	"github.com/RohmanBenyRiyanto/gorun/internal/migration"
	"github.com/RohmanBenyRiyanto/gorun/internal/seed"
	"github.com/RohmanBenyRiyanto/gorun/internal/table"
	cli "github.com/urfave/cli/v3"
)

// Config is the one piece of state every gorun command shares - database
// connections, the seeders you want `seed run`/`seed list` to know about,
// and a couple of cosmetic fields. Pass it to New; zero-value fields get
// sane defaults there (see New).
type Config = engine.Config

// DBConnConfig holds connection settings for one database engine (MySQL
// or PostgreSQL) - see Config.MySQL / Config.PostgreSQL.
type DBConnConfig = engine.DBConnConfig

// Seeder is one unit of seed data - implement this for each seeder in
// your project and register it through a SeederRegistry.
type Seeder = engine.Seeder

// SeederRegistry tells `seed run`/`seed list` which seeders exist and
// what order to run them in. gorun ships no seeders of its own, only the
// machinery to run them - implement this for your project's seeders and
// set it via Config.MySQLSeeders / Config.PostgreSQLSeeders.
type SeederRegistry = engine.SeederRegistry

// New builds the full gorun command tree from cfg, ready to run:
//
//	cmd := gorun.New(cfg)
//	err := cmd.Run(context.Background(), os.Args)
//
// Zero-value fields in cfg are defaulted here rather than treated as
// errors: Name -> "gorun", Usage -> a generic one-line description,
// MySQL.Charset -> "utf8mb4", MySQL.Loc -> "Local", PostgreSQL.SslMode ->
// "disable", PostgreSQL.TimeZone -> "UTC". Leaving MySQL or PostgreSQL
// entirely unset just means commands targeting that engine will fail to
// connect - it's not an error by itself.
//
// One process-wide side effect: New overrides the package-level
// cli.HelpPrinter so `--help` (and the bare `gorun` invocation) shows a
// project-info banner instead of urfave/cli's default template. If your
// process builds other cli.Command trees of its own alongside this one,
// they'll pick up the same help printer.
func New(cfg Config) *cli.Command {
	applyDefaults(&cfg)

	infoHelper := info.NewInfoHelper()

	cli.HelpPrinter = func(w io.Writer, templ string, data any) {
		infoHelper.ShowProjectInfo()
	}

	root := &cli.Command{
		Name:  cfg.Name,
		Usage: cfg.Usage,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "commands",
				Aliases: []string{"c"},
				Usage:   "List all available commands with detailed descriptions",
				Action: func(ctx context.Context, cmd *cli.Command, b bool) error {
					infoHelper.ListCommands()
					os.Exit(0)
					return nil
				},
			},
			&cli.BoolFlag{
				Name:    "info",
				Aliases: []string{"i"},
				Usage:   "Show comprehensive project information and environment details",
				Action: func(ctx context.Context, cmd *cli.Command, b bool) error {
					infoHelper.ShowProjectInfo()
					os.Exit(0)
					return nil
				},
			},
			&cli.BoolFlag{
				Name:    "version",
				Aliases: []string{"v"},
				Usage:   "Display toolkit version and build information",
				Action: func(ctx context.Context, cmd *cli.Command, b bool) error {
					infoHelper.ShowVersion()
					os.Exit(0)
					return nil
				},
			},
		},
		Commands: []*cli.Command{
			app.Command(&cfg),
			db.Command(&cfg),
			table.Command(&cfg),
			seed.Command(&cfg),
			migration.Command(&cfg),
			{
				Name:    "help",
				Aliases: []string{"h"},
				Usage:   "Show comprehensive help and usage information",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					infoHelper.ShowHelp()
					return nil
				},
			},
			{
				Name:    "commands",
				Aliases: []string{"list", "ls"},
				Usage:   "List all available commands (alternative to -c flag)",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					infoHelper.ListCommands()
					return nil
				},
			},
			{
				Name:    "version",
				Aliases: []string{"ver"},
				Usage:   "Show version information (alternative to -v flag)",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					infoHelper.ShowVersion()
					return nil
				},
			},
			{
				Name:    "info",
				Aliases: []string{"project", "status"},
				Usage:   "Show project information (alternative to -i flag)",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					infoHelper.ShowProjectInfo()
					return nil
				},
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Bool("commands") {
				infoHelper.ListCommands()
				return nil
			}
			if cmd.Bool("info") {
				infoHelper.ShowProjectInfo()
				return nil
			}
			if cmd.Bool("version") {
				infoHelper.ShowVersion()
				return nil
			}

			infoHelper.ShowHelp()
			return nil
		},
	}

	return root
}

// applyDefaults fills zero-value Config fields with the values documented
// on New.
func applyDefaults(cfg *Config) {
	if cfg.Name == "" {
		cfg.Name = "gorun"
	}
	if cfg.Usage == "" {
		cfg.Usage = "Go Development Toolkit"
	}
	if cfg.MySQL.Charset == "" {
		cfg.MySQL.Charset = "utf8mb4"
	}
	if cfg.MySQL.Loc == "" {
		cfg.MySQL.Loc = "Local"
	}
	if cfg.PostgreSQL.SslMode == "" {
		cfg.PostgreSQL.SslMode = "disable"
	}
	if cfg.PostgreSQL.TimeZone == "" {
		cfg.PostgreSQL.TimeZone = "UTC"
	}
	if cfg.ServerEntrypoint == "" {
		cfg.ServerEntrypoint = filepath.Join("cmd", "server", "main.go")
	}
}
