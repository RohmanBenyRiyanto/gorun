package migration

import (
	"context"
	"fmt"
	"strings"

	"github.com/RohmanBenyRiyanto/gorun/internal/engine"
	"github.com/urfave/cli/v3"
)

// RunCommand implements `gorun migrate run` - applying pending migrations.
type RunCommand struct {
	config *engine.Config
}

// NewRunCommand builds a RunCommand and prints its banner.
func NewRunCommand(config *engine.Config) *RunCommand {
	engine.PrintBoldCard("MIGRATION COMMANDS:RUN")
	return &RunCommand{
		config: config,
	}
}

// Handle resolves the target database, then either runs one specific
// migration (--file) or every pending one, confirming first unless
// --force.
func (rc *RunCommand) Handle(ctx context.Context, cmd *cli.Command) error {
	dbManager := engine.NewDatabaseManager(rc.config)
	dbSelector := engine.NewDatabaseSelector(dbManager)
	migrationUtils := NewMigrationUtils()

	dbType, err := dbManager.ResolveDatabaseType(cmd)
	if err != nil {
		return err
	}

	gormDB, sqlDB, err := dbManager.InitializeDatabase(dbType, rc.config)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	dbName, err := dbSelector.ResolveDatabaseName(cmd, gormDB, dbType, "run migrations")
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
		File:     cmd.String("file"),
		Pretend:  cmd.Bool("pretend"),
		Step:     cmd.Int("step"),
		Database: dbName,
	}

	mm.SetOptions(options)

	engine.PrintDebug("File: %v", options.File)

	if options.File != "" {
		return rc.handleSpecificMigration(mm, dbType, dbName, migrationUtils)
	}

	return rc.handleNormalMigration(mm, dbType, dbName, migrationUtils)
}

func (rc *RunCommand) handleSpecificMigration(mm *engine.MigrationManager, dbType engine.DatabaseType, dbName string, utils *MigrationUtils) error {
	statuses, err := utils.GetMigrationDetails(mm)
	if err != nil {
		return fmt.Errorf("failed to get migration details: %w", err)
	}

	fmt.Println()

	var targetFile string
	for _, status := range statuses {
		if strings.Contains(status.Name, mm.Options.File) {
			targetFile = status.Name
			break
		}
	}

	if targetFile == "" {
		return fmt.Errorf("no migration file matching '%s' found", mm.Options.File)
	}

	if err := mm.MigrateSpecific(mm.Options.File, mm.Options.Force); err != nil {
		return fmt.Errorf("failed to run specific migration: %w", err)
	}

	fmt.Println()
	engine.PrintSuccess("Specific migration '%s' completed successfully!", targetFile)

	statuses, _ = utils.GetMigrationDetails(mm)
	utils.DisplayMigrationStatus(dbType, dbName, statuses)

	return nil
}

func (rc *RunCommand) handleNormalMigration(mm *engine.MigrationManager, dbType engine.DatabaseType, dbName string, utils *MigrationUtils) error {
	statuses, err := utils.GetMigrationDetails(mm)
	if err != nil {
		return fmt.Errorf("failed to get migration details: %w", err)
	}

	if !mm.Options.Force {
		var pendingMigrations []string
		for _, migration := range statuses {
			if migration.Status == "Pending" {
				pendingMigrations = append(pendingMigrations, migration.Name)
			}
		}

		if len(pendingMigrations) > 0 {
			engine.PrintWarning("Pending migrations to run (%d):", len(pendingMigrations))
			for _, name := range pendingMigrations {
				engine.PrintNormal("%s", "- "+name)
			}

			confirmed := engine.ConfirmPrompt("Run these migrations?")
			if !confirmed {
				engine.PrintInfo("Migration cancelled")
				return nil
			}
		}
	}

	if err := mm.Migrate(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	engine.PrintSuccess("Migrations completed successfully!")
	statuses, _ = utils.GetMigrationDetails(mm)
	utils.DisplayMigrationStatus(dbType, dbName, statuses)

	return nil
}
