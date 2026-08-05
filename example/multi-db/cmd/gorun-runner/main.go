// Command gorun-runner is what the globally-installed gorun binary hands
// `gorun seed ...` off to - it can't carry this project's real seeders
// itself (a YAML file can't express a Go interface implementation), so
// this small entrypoint does: it loads the same .gorun/config.yaml,
// attaches your seeders, and runs the normal gorun command tree. You
// won't normally run this directly.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/RohmanBenyRiyanto/gorun"
)

func main() {
	cfg, err := gorun.LoadConfigFile(".gorun/config.yaml")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// TODO: register your seeders, e.g.:
	// cfg.MySQLSeeders = myapp.MySQLSeeders{}

	cmd := gorun.New(cfg)
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
