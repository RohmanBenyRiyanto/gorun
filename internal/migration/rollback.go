package migration

import (
	"context"
	"fmt"

	"github.com/RohmanBenyRiyanto/gorun/internal/engine"
	"github.com/urfave/cli/v3"
)

// RollbackCommand implements `gorun migrate rollback` - undoing the last
// batch (or --step batches) of applied migrations.
type RollbackCommand struct {
	config *engine.Config
}

// NewRollbackCommand builds a RollbackCommand and prints its banner.
func NewRollbackCommand(config *engine.Config) *RollbackCommand {
	engine.PrintBoldCard("MIGRATION COMMANDS:ROLLBACK")
	return &RollbackCommand{
		config: config,
	}
}

// Handle resolves the target database, confirms unless --force, and rolls
// back via MigrationManager.Rollback.
func (rc *RollbackCommand) Handle(ctx context.Context, cmd *cli.Command) error {
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

	dbName, err := dbSelector.ResolveDatabaseName(cmd, gormDB, dbType, "rollback migrations")
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
		Path:     cmd.String("path"),
		Step:     cmd.Int("step"),
		Database: dbName,
	}

	mm.SetOptions(options)

	if !options.Force {
		lastBatch, err := mm.GetLastBatch()
		if err != nil {
			return fmt.Errorf("failed to get last batch: %w", err)
		}

		if lastBatch == 0 {
			engine.PrintInfo("No migrations to rollback")
			return nil
		}

		dbManager.PrintTargetSummary(dbName, dbType)
		engine.PrintWarning("About to rollback %d steps from batch %d", options.Step, lastBatch)
		confirmed := engine.ConfirmPrompt("Continue with rollback?")
		if !confirmed {
			engine.PrintInfo("Rollback cancelled")
			return nil
		}
	}

	if err := mm.Rollback(options.Step); err != nil {
		return fmt.Errorf("failed to rollback migrations: %w", err)
	}

	engine.PrintSuccess("Rollback completed successfully!")

	return nil
}
