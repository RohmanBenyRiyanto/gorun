package migration

import (
	"context"
	"fmt"

	"github.com/RohmanBenyRiyanto/gorun/internal/engine"
	"github.com/urfave/cli/v3"
)

// ResetCommand implements `gorun migrate reset` - rolling back every
// applied migration.
type ResetCommand struct {
	config *engine.Config
}

// NewResetCommand builds a ResetCommand and prints its banner.
func NewResetCommand(config *engine.Config) *ResetCommand {
	engine.PrintBoldCard("MIGRATION COMMANDS:RESET")
	return &ResetCommand{
		config: config,
	}
}

// Handle resolves the target database, confirms unless --force, and rolls
// back every applied migration via MigrationManager.Reset.
func (rc *ResetCommand) Handle(ctx context.Context, cmd *cli.Command) error {
	dbManager := engine.NewDatabaseManager(rc.config)
	dbSelector := engine.NewDatabaseSelector(dbManager)

	dbType, err := dbManager.ResolveDatabaseType(cmd)
	if err != nil {
		return err
	}

	gormDB, sqlDB, err := dbManager.InitializeDatabase(dbType, rc.config)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	dbName, err := dbSelector.ResolveDatabaseName(cmd, gormDB, dbType, "reset migrations")
	if err != nil {
		return err
	}

	mm := engine.NewMigrationManager(dbManager, rc.config)
	if err := mm.SetDatabaseName(dbName).InitializeDatabase(dbType); err != nil {
		return fmt.Errorf("failed to initialize migration manager: %w", err)
	}
	defer func() { _ = mm.Close() }()

	options := engine.MigrationOptions{
		Force:    cmd.Bool("force"),
		Database: dbName,
	}

	mm.SetOptions(options)

	if !options.Force {
		currentMigrations, err := mm.GetCurrentMigrations()
		if err != nil {
			return fmt.Errorf("failed to get current migrations: %w", err)
		}

		if len(currentMigrations) == 0 {
			engine.PrintInfo("No migrations to reset")
			return nil
		}

		dbManager.PrintTargetSummary(dbName, dbType)
		engine.PrintWarning("About to reset ALL %d migrations", len(currentMigrations))
		confirmed := engine.ConfirmPrompt("Continue with reset?")
		if !confirmed {
			engine.PrintInfo("Reset cancelled")
			return nil
		}
	}

	if err := mm.Reset(); err != nil {
		return fmt.Errorf("failed to reset migrations: %w", err)
	}

	engine.PrintSuccess("All migrations reset successfully!")

	return nil
}
