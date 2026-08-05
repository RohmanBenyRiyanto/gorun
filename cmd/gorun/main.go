// Command gorun is the globally-installed CLI: install it once per
// machine with
//
//	go install github.com/RohmanBenyRiyanto/gorun/cmd/gorun@latest
//
// then run `gorun <command>` from inside any project directory that has a
// .gorun/config.yaml (see the gorun README for the file format). Each
// project keeps its own config, discovered by walking up from the current
// directory the same way git finds .git/ - one binary, many projects, no
// `go run ./path/to/main.go` required in any of them.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/RohmanBenyRiyanto/gorun"
	"github.com/RohmanBenyRiyanto/gorun/internal/engine"
	"github.com/RohmanBenyRiyanto/gorun/internal/setup"
	cli "github.com/urfave/cli/v3"
)

// projectScoped are the subcommands that read database/seeder settings
// out of Config and therefore need a project config before they can do
// anything useful. Commands not in this set (help, version, info,
// commands, app) run fine with a zero-value Config.
var projectScoped = map[string]bool{
	"db":      true,
	"migrate": true,
	"seed":    true,
	"table":   true,
}

func main() {
	args := os.Args[1:]

	// setup runs before any Config exists - that's the whole point of it -
	// so it's handled entirely separately from the discover/load/gorun.New
	// path every other command goes through.
	if len(args) > 0 && args[0] == "setup" {
		runSetup(args[1:])
		return
	}

	configPath, explicitConfig, rest := extractConfigFlag(args)

	cfg, found, err := resolveConfig(configPath, explicitConfig)
	if len(rest) > 0 && projectScoped[rest[0]] {
		if !found {
			fmt.Fprintln(os.Stderr, "gorun: no .gorun/config.yaml found in this directory or any parent.")
			fmt.Fprintln(os.Stderr, "Run `gorun setup` here first, or pass --config <path>.")
			os.Exit(1)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "gorun: failed to load config: %v\n", err)
			os.Exit(1)
		}
	}

	// gorun.Config loaded from YAML can never carry a real
	// SeederRegistry - that's Go behavior, not data, and a config file
	// can't express it. If this project's config names a RunnerPath,
	// every `seed` subcommand is handled by that project-local Go
	// entrypoint instead (which builds its own Config with real seeders
	// and calls gorun.New itself), rather than gorun.New's own copy
	// reporting "no registry configured" for every single seed command.
	if len(rest) > 0 && rest[0] == "seed" && cfg.RunnerPath != "" {
		os.Exit(runViaRunner(cfg.RunnerPath, rest))
	}

	cmd := gorun.New(cfg)
	if err := cmd.Run(context.Background(), append([]string{"gorun"}, rest...)); err != nil {
		fmt.Fprintf(os.Stderr, "gorun: %v\n", err)
		os.Exit(1)
	}
}

// runViaRunner execs `go run runnerPath <args...>` with stdio connected
// straight through, and returns its exit code so the caller can just
// os.Exit with it - from the terminal, delegating to the runner is
// invisible, `gorun seed run` behaves the same either way.
func runViaRunner(runnerPath string, args []string) int {
	c := exec.Command("go", append([]string{"run", runnerPath}, args...)...)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr

	if err := c.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "gorun: failed to run %s: %v\n", runnerPath, err)
		return 1
	}
	return 0
}

func runSetup(args []string) {
	root := &cli.Command{
		Name:     "gorun",
		Commands: []*cli.Command{setup.Command()},
	}
	if err := root.Run(context.Background(), append([]string{"gorun", "setup"}, args...)); err != nil {
		fmt.Fprintf(os.Stderr, "gorun setup: %v\n", err)
		os.Exit(1)
	}
}

// resolveConfig loads the config at path when explicit is true, otherwise
// discovers one from the current directory. found reports whether a
// config was located at all (which callers care about even for commands
// that don't strictly require one, so error messages can be accurate);
// err carries any parse failure once a path is known.
func resolveConfig(path string, explicit bool) (cfg gorun.Config, found bool, err error) {
	configPath := path
	if !explicit {
		discovered, ok := gorun.DiscoverConfigFile(".")
		if !ok {
			return gorun.Config{}, false, nil
		}
		configPath = discovered
	}

	loadDotEnv(configPath)

	cfg, err = gorun.LoadConfigFile(configPath)
	return cfg, true, err
}

// loadDotEnv reads a .env file from the project root - two directories
// up from configPath, which is always <root>/.gorun/config.yaml - and
// sets each key as an environment variable, but only if it isn't already
// set, so a real exported environment variable always wins over the
// file. This is what lets .gorun/config.yaml's ${VAR} references (and,
// since runViaRunner's subprocess inherits this process's environment,
// a delegated runner's own env reads too) resolve without requiring the
// user to export everything by hand first. Silently does nothing if
// there's no .env file - a convenience, not a requirement.
func loadDotEnv(configPath string) {
	root := filepath.Dir(filepath.Dir(configPath))
	env, err := engine.ReadEnvFile(filepath.Join(root, ".env"))
	if err != nil {
		return
	}
	for k, v := range env {
		if _, set := os.LookupEnv(k); !set {
			_ = os.Setenv(k, v)
		}
	}
}

// extractConfigFlag pulls a `--config <path>` / `--config=<path>` override
// out of args before urfave/cli ever sees them: which config file to load
// is gorun-the-binary's own concern, decided before the command tree (and
// the Config baked into it) can even be built - no individual subcommand
// should need to know about it.
func extractConfigFlag(args []string) (path string, ok bool, rest []string) {
	rest = make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--config" && i+1 < len(args):
			path = args[i+1]
			ok = true
			i++
		case strings.HasPrefix(arg, "--config="):
			path = strings.TrimPrefix(arg, "--config=")
			ok = true
		default:
			rest = append(rest, arg)
		}
	}
	return path, ok, rest
}
