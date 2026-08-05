package seed

import (
	"context"
	"fmt"

	"github.com/RohmanBenyRiyanto/gorun/internal/engine"
	"github.com/urfave/cli/v3"
)

// MakeCommand implements `gorun seed make` - scaffolding new seeder
// files.
type MakeCommand struct {
	config *engine.Config
}

// NewMakeCommand builds a MakeCommand.
func NewMakeCommand(config *engine.Config) *MakeCommand {
	return &MakeCommand{
		config: config,
	}
}

// Handle creates one seeder file per name argument via
// SeederManager.CreateSeeder - no SeederRegistry is needed here since
// scaffolding a file doesn't run anything.
func (mc *MakeCommand) Handle(ctx context.Context, cmd *cli.Command) error {
	dbManager := engine.NewDatabaseManager(mc.config)
	dbType := dbManager.PromptDatabaseSelection()

	seederManager := engine.NewSeederManager(dbManager, dbType, mc.config, nil)
	seederManager.Options = engine.SeederOptions{
		Path:     cmd.String("path"),
		RealPath: cmd.Bool("realpath"),
		FullPath: cmd.Bool("fullpath"),
		Model:    cmd.Bool("model"),
		Table:    cmd.String("table"),
	}

	for _, name := range cmd.Args().Slice() {
		if err := seederManager.CreateSeeder(name); err != nil {
			return fmt.Errorf("failed to create seeder %s: %w", name, err)
		}
	}

	return nil
}
