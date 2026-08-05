package migration

import (
	"context"
	"fmt"
	"time"

	"github.com/RohmanBenyRiyanto/gorun/internal/engine"
	"github.com/urfave/cli/v3"
)

// StatusCommand implements `gorun migrate status` - a Ran/Pending/Missing
// table for every migration, plus an interactive follow-up menu.
type StatusCommand struct {
	config *engine.Config
}

// NewStatusCommand builds a StatusCommand and prints its banner.
func NewStatusCommand(config *engine.Config) *StatusCommand {
	engine.PrintBoldCard("MIGRATION COMMANDS:STATUS")
	return &StatusCommand{
		config: config,
	}
}

// Handle resolves the target database, prints the migration status table,
// and (if there's at least one migration) offers the interactive
// run/clean/view menu via MigrationUtils.PromptMigrationActions.
func (sc *StatusCommand) Handle(ctx context.Context, cmd *cli.Command) error {

	dbManager := engine.NewDatabaseManager(sc.config)
	dbSelector := engine.NewDatabaseSelector(dbManager)
	migrationUtils := NewMigrationUtils()

	dbType, err := dbManager.ResolveDatabaseType(cmd)
	if err != nil {
		return err
	}

	gormDB, sqlDB, err := dbManager.InitializeDatabase(dbType, sc.config)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	dbName, err := dbSelector.ResolveDatabaseName(cmd, gormDB, dbType, "check migration status")
	if err != nil {
		return err
	}

	mm := engine.NewMigrationManager(dbManager, sc.config)
	if err := mm.SetDatabaseName(dbName).InitializeDatabase(dbType); err != nil {
		return fmt.Errorf("failed to initialize migration manager: %w", err)
	}
	defer func() { _ = mm.Close() }()

	_, err = mm.Status()
	if err != nil {
		return fmt.Errorf("failed to get migration status: %w", err)
	}

	migrationStatuses, err := migrationUtils.GetMigrationDetails(mm)
	if err != nil {
		return fmt.Errorf("failed to get migration details: %w", err)
	}

	migrationUtils.DisplayMigrationStatus(dbType, dbName, migrationStatuses)

	if len(migrationStatuses) > 0 {
		return migrationUtils.PromptMigrationActions(mm, migrationStatuses)
	}

	return nil
}

// MigrationStatus is one migration's combined file + database-record
// state, as built by MigrationUtils.GetMigrationDetails.
type MigrationStatus struct {
	Number      int
	Name        string
	Status      string
	Batch       int
	RanAt       time.Time
	FileExists  bool
	FilePath    string
	FileSize    string
	HasUp       bool
	HasDown     bool
	Description string
}
