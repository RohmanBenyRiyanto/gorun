package app

import (
	"context"

	"github.com/RohmanBenyRiyanto/gorun/internal/engine"
	"github.com/urfave/cli/v3"
)

// Command builds the "app" command group: build/serve/test/clean/install/
// status/version, plus its own help subcommand.
func Command(config *engine.Config) *cli.Command {
	return &cli.Command{
		Name:    "app",
		Aliases: []string{"application"},
		Usage:   "Application management commands",
		Commands: []*cli.Command{
			{
				Name:    "help",
				Aliases: []string{"h"},
				Usage:   "Show application commands help",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					helper := NewAppHelper()
					helper.ShowHelp()
					return nil
				},
			},
			{
				Name:    "build",
				Aliases: []string{"compile"},
				Usage:   "Build the application",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "Output binary name",
						Value:   "app",
					},
					&cli.StringFlag{
						Name:  "os",
						Usage: "Target operating system (linux, windows, darwin)",
					},
					&cli.StringFlag{
						Name:  "arch",
						Usage: "Target architecture (amd64, arm64, 386)",
					},
					&cli.BoolFlag{
						Name:  "race",
						Usage: "Enable race detector",
					},
					&cli.BoolFlag{
						Name:  "docker",
						Usage: "Build for docker container",
					},
					&cli.StringFlag{
						Name:  "ldflags",
						Usage: "Pass 'flag' to the Go linker",
					},
					&cli.StringFlag{
						Name:  "tags",
						Usage: "Build tags to pass to Go",
					},
					&cli.BoolFlag{
						Name:  "verbose",
						Usage: "Enable verbose build output",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return NewBuildCommand(config).Handle(ctx, cmd)
				},
			},
			{
				Name:    "status",
				Aliases: []string{"info"},
				Usage:   "Show application status and information",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "detailed",
						Usage: "Show detailed information",
					},
					&cli.BoolFlag{
						Name:  "json",
						Usage: "Output in JSON format",
					},
					&cli.BoolFlag{
						Name:  "health",
						Usage: "Check application health",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return NewStatusCommand(config).Handle(ctx, cmd)
				},
			},
			{
				Name:    "serve",
				Aliases: []string{"start", "run"},
				Usage:   "Start the application server",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "port",
						Aliases: []string{"p"},
						Usage:   "Port to serve on",
					},
					&cli.StringFlag{
						Name:  "host",
						Usage: "Host to bind to",
						Value: "localhost",
					},
					&cli.BoolFlag{
						Name:  "dev",
						Usage: "Enable development mode",
					},
					&cli.BoolFlag{
						Name:  "watch",
						Usage: "Enable hot reload (requires air)",
					},
					&cli.StringFlag{
						Name:  "env",
						Usage: "Environment (development, staging, production)",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return NewServeCommand(config).Handle(ctx, cmd)
				},
			},
			{
				Name:    "test",
				Aliases: []string{"t"},
				Usage:   "Run application tests",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "package",
						Aliases: []string{"pkg"},
						Usage:   "Run tests for specific package",
					},
					&cli.BoolFlag{
						Name:  "verbose",
						Usage: "Enable verbose test output",
					},
					&cli.BoolFlag{
						Name:  "race",
						Usage: "Enable race detector",
					},
					&cli.BoolFlag{
						Name:  "coverage",
						Usage: "Enable coverage analysis",
					},
					&cli.StringFlag{
						Name:  "coverprofile",
						Usage: "Write coverage profile to file",
					},
					&cli.IntFlag{
						Name:  "timeout",
						Usage: "Test timeout in seconds",
						Value: 30,
					},
					&cli.BoolFlag{
						Name:  "short",
						Usage: "Run short tests only",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return NewTestCommand(config).Handle(ctx, cmd)
				},
			},
			{
				Name:    "clean",
				Aliases: []string{"clear"},
				Usage:   "Clean build artifacts and temporary files",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "cache",
						Usage: "Clean Go build cache",
					},
					&cli.BoolFlag{
						Name:  "modules",
						Usage: "Clean module cache",
					},
					&cli.BoolFlag{
						Name:  "logs",
						Usage: "Clean log files",
					},
					&cli.BoolFlag{
						Name:  "all",
						Usage: "Clean everything",
					},
					&cli.BoolFlag{
						Name:    "force",
						Aliases: []string{"f"},
						Usage:   "Force clean without confirmation",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return NewCleanCommand(config).Handle(ctx, cmd)
				},
			},
			{
				Name:    "install",
				Aliases: []string{"deps"},
				Usage:   "Install application dependencies",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "update",
						Usage: "Update dependencies to latest versions",
					},
					&cli.BoolFlag{
						Name:  "tidy",
						Usage: "Run go mod tidy",
					},
					&cli.BoolFlag{
						Name:  "vendor",
						Usage: "Download dependencies to vendor directory",
					},
					&cli.BoolFlag{
						Name:  "verify",
						Usage: "Verify dependencies",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return NewInstallCommand(config).Handle(ctx, cmd)
				},
			},
			{
				Name:    "version",
				Aliases: []string{"v"},
				Usage:   "Show application version information",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "json",
						Usage: "Output in JSON format",
					},
					&cli.BoolFlag{
						Name:  "short",
						Usage: "Show version only",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return NewVersionCommand(config).Handle(ctx, cmd)
				},
			},
		},
	}
}
