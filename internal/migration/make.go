package migration

import (
	"context"
	"fmt"
	"strings"

	"github.com/RohmanBenyRiyanto/gorun/internal/engine"
	"github.com/urfave/cli/v3"
)

// MakeCommand implements `gorun migrate make` - scaffolding a new
// timestamped migration file.
type MakeCommand struct {
	config *engine.Config
}

// NewMakeCommand builds a MakeCommand and prints its banner.
func NewMakeCommand(config *engine.Config) *MakeCommand {
	engine.PrintBoldCard("MIGRATION COMMANDS:MAKE")
	return &MakeCommand{
		config: config,
	}
}

// Handle resolves the engine, normalizes the migration name, and writes
// the new file via MigrationManager.MakeMigration.
func (mc *MakeCommand) Handle(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() == 0 {
		return fmt.Errorf("migration name is required")
	}

	dbManager := engine.NewDatabaseManager(mc.config)
	dbType, err := dbManager.ResolveDatabaseType(cmd)
	if err != nil {
		return err
	}

	_, sqlDB, err := dbManager.InitializeDatabase(dbType, mc.config)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	mm := engine.NewMigrationManager(dbManager, mc.config)
	if err := mm.InitializeDatabase(dbType); err != nil {
		return fmt.Errorf("failed to initialize migration manager: %w", err)
	}
	defer func() { _ = mm.Close() }()

	name := strings.TrimSpace(cmd.Args().First())
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "_")

	options := engine.MigrationOptions{
		Create:   cmd.String("create"),
		Table:    cmd.String("table"),
		Path:     cmd.String("path"),
		RealPath: cmd.Bool("realpath"),
		FullPath: cmd.Bool("fullpath"),
	}

	mm.SetOptions(options)

	if err := mm.MakeMigration(name); err != nil {
		return fmt.Errorf("failed to create migration: %w", err)
	}

	engine.PrintSuccess("Migration created successfully!")
	return nil
}
