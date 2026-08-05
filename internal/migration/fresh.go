package migration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/RohmanBenyRiyanto/gorun/internal/engine"
	"github.com/urfave/cli/v3"
)

// FreshCommand implements `gorun migrate fresh` - dropping and recreating
// the whole database, then re-running every migration.
type FreshCommand struct {
	config *engine.Config
}

// NewFreshCommand builds a FreshCommand and prints its banner.
func NewFreshCommand(config *engine.Config) *FreshCommand {
	engine.PrintBoldCard("MIGRATION COMMANDS:FRESH")
	return &FreshCommand{
		config: config,
	}
}

// Handle resolves the target database, confirms the drop unless --force,
// then either re-runs everything (mm.Fresh) or, with --file set, rolls
// back and re-applies just that one migration. Runs seeders afterward if
// --seed was passed.
func (fc *FreshCommand) Handle(ctx context.Context, cmd *cli.Command) error {
	dbManager := engine.NewDatabaseManager(fc.config)
	dbSelector := engine.NewDatabaseSelector(dbManager)
	migrationUtils := NewMigrationUtils()

	dbType, err := dbManager.ResolveDatabaseType(cmd)
	if err != nil {
		return err
	}

	gormDB, sqlDB, err := dbManager.InitializeDatabase(dbType, fc.config)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	dbName, err := dbSelector.ResolveDatabaseName(cmd, gormDB, dbType, "fresh migrations")
	if err != nil {
		return err
	}

	mm := engine.NewMigrationManager(dbManager, fc.config)
	if err := mm.SetDatabaseName(dbName).InitializeDatabase(dbType); err != nil {
		return fmt.Errorf("failed to initialize migration manager: %w", err)
	}
	defer func() { _ = mm.Close() }()

	options := engine.MigrationOptions{
		Force:     cmd.Bool("force"),
		Seed:      cmd.Bool("seed"),
		DropViews: cmd.Bool("drop-views"),
		DropTypes: cmd.Bool("drop-types"),
		File:      cmd.String("file"),
		Database:  dbName,
	}

	mm.SetOptions(options)

	if !options.Force {
		dbManager.PrintTargetSummary(dbName, dbType)
		if options.File != "" {
			engine.PrintWarning("About to FRESH database '%s' and run specific migration '%s' - this will DROP and RECREATE the database", dbName, options.File)
		} else {
			engine.PrintWarning("About to FRESH database '%s' - this will DROP and RECREATE the database", dbName)
		}
		confirmed := engine.ConfirmPrompt("Continue with fresh?")
		if !confirmed {
			engine.PrintInfo("Fresh cancelled")
			return nil
		}
	}

	fmt.Println()

	if options.File != "" {
		return fc.handleSpecificFileFresh(mm, migrationUtils, options)
	} else {
		if err := mm.Fresh(); err != nil {
			return fmt.Errorf("failed to fresh database: %w", err)
		}
	}

	if options.Seed {
		if err := fc.runSeeders(dbType, dbName); err != nil {
			return err
		}
	}

	engine.PrintSuccess("Database fresh completed successfully!")
	return nil
}

func (fc *FreshCommand) handleSpecificFileFresh(mm *engine.MigrationManager, migrationUtils *MigrationUtils, options engine.MigrationOptions) error {
	statuses, err := migrationUtils.GetMigrationDetails(mm)
	if err != nil {
		return fmt.Errorf("failed to get migration details: %w", err)
	}

	var targetFile string
	for _, status := range statuses {
		if strings.Contains(status.Name, options.File) {
			targetFile = status.Name
			break
		}
	}

	if targetFile == "" {
		return fmt.Errorf("no migration file matching '%s' found", options.File)
	}

	currentMigrations, err := mm.GetCurrentMigrations()
	if err != nil {
		return fmt.Errorf("failed to get current migrations: %w", err)
	}

	var isApplied bool
	for _, migration := range currentMigrations {
		if migration == targetFile {
			isApplied = true
			break
		}
	}

	if isApplied {
		engine.PrintInfo("Rolling back specific migration '%s'...", targetFile)
		if err := fc.rollbackSpecificMigration(mm, targetFile); err != nil {
			return fmt.Errorf("failed to rollback specific migration: %w", err)
		}
		engine.PrintSuccess("Successfully rolled back migration '%s'", targetFile)
	}

	engine.PrintInfo("Applying specific migration '%s'...", targetFile)
	if err := mm.MigrateSpecific(targetFile, true); err != nil { // force=true: skip normal checks, we already handled rollback above
		return fmt.Errorf("failed to run specific migration: %w", err)
	}

	fmt.Println()
	engine.PrintSuccess("Specific migration '%s' completed successfully!", targetFile)

	if options.Seed {
		if err := fc.runSeeders(mm.GetDatabaseType(), options.Database); err != nil {
			return err
		}
	}

	return nil
}

func (fc *FreshCommand) rollbackSpecificMigration(mm *engine.MigrationManager, migrationName string) error {
	migrationPath := mm.GetMigrationPath()
	filePath := filepath.Join(migrationPath, migrationName+".sql")

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		engine.PrintWarning("Migration file not found, removing from database records only")
		if err := mm.GetGormDB().Where("migration = ?", migrationName).Delete(&engine.Migrations{}).Error; err != nil {
			return fmt.Errorf("failed to remove migration record: %w", err)
		}
		return nil
	}

	var batch int
	if err := mm.GetGormDB().Model(&engine.Migrations{}).
		Select("batch").
		Where("migration = ?", migrationName).
		Scan(&batch).Error; err != nil {
		return fmt.Errorf("failed to get migration batch: %w", err)
	}

	if err := mm.RunMigrationFile(filePath, "down", batch); err != nil {
		return fmt.Errorf("failed to run down migration: %w", err)
	}

	return nil
}

// runSeeders shells out to `go run <Config.RunnerPath> seed run`, the
// same delegation cmd/gorun's own seed commands use (see the gorun
// README's "Using it from the global CLI: RunnerPath") - set it via
// `gorun setup` or by hand in .gorun/config.yaml. Requires RunnerPath
// specifically rather than trying MySQLSeeders/PostgreSQLSeeders
// in-process: --seed's whole job is producing a fresh, isolated `go run`
// invocation the same way `gorun seed run` on its own would, not
// reusing this process's already-open connection.
func (fc *FreshCommand) runSeeders(dbType engine.DatabaseType, dbName string) error {
	if fc.config.RunnerPath == "" {
		return fmt.Errorf("--seed needs Config.RunnerPath set (runner_path in .gorun/config.yaml, see `gorun setup`) - or skip --seed and run `gorun seed run` yourself afterward")
	}

	engine.PrintInfo("Preparing to run seeders...")

	seedArgs := []string{
		"seed",
		"run",
		"--type=" + string(dbType),
		"--database=" + dbName,
	}

	cmdArgs := append([]string{"run", fc.config.RunnerPath}, seedArgs...)

	engine.PrintInfo("Executing: go %s", strings.Join(cmdArgs, " "))

	seedCmd := exec.Command("go", cmdArgs...)
	seedCmd.Stdout = os.Stdout
	seedCmd.Stderr = os.Stderr

	if err := seedCmd.Run(); err != nil {
		return fmt.Errorf("seed command failed: %w", err)
	}

	engine.PrintSuccess("Seeders completed successfully!")
	return nil
}
