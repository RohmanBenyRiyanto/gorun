package migration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/RohmanBenyRiyanto/gorun/internal/engine"
	"github.com/urfave/cli/v3"
)

// RefreshCommand implements `gorun migrate refresh` - rolling back
// everything applied and re-running from scratch, without dropping the
// database itself (see FreshCommand for that).
type RefreshCommand struct {
	config *engine.Config
}

// NewRefreshCommand builds a RefreshCommand and prints its banner.
func NewRefreshCommand(config *engine.Config) *RefreshCommand {
	engine.PrintBoldCard("MIGRATION COMMANDS:REFRESH")
	return &RefreshCommand{
		config: config,
	}
}

// Handle resolves the target database, then either refreshes everything
// (mm.Refresh) or, with --file set, refreshes and re-applies just that one
// migration. Runs seeders afterward if --seed was passed, via
// Config.RunnerPath (see the gorun README's "Using it from the global
// CLI: RunnerPath") - --seed errors immediately if it's unset rather than
// guessing at a path.
func (rc *RefreshCommand) Handle(ctx context.Context, cmd *cli.Command) error {
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

	dbName, err := dbSelector.ResolveDatabaseName(cmd, gormDB, dbType, "refresh migrations")
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
		Step:     cmd.Int("step"),
		Seed:     cmd.Bool("seed"),
		File:     cmd.String("file"),
		Database: dbName,
	}

	mm.SetOptions(options)

	if options.File != "" {
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

		if err := mm.Refresh(); err != nil {
			return fmt.Errorf("failed to refresh migrations: %w", err)
		}

		if err := mm.MigrateSpecific(targetFile, options.Force); err != nil {
			return fmt.Errorf("failed to run specific migration: %w", err)
		}

		fmt.Println()
		engine.PrintSuccess("Specific migration '%s' completed successfully!", targetFile)
	} else {
		if !options.Force {
			currentMigrations, err := mm.GetCurrentMigrations()
			if err != nil {
				return fmt.Errorf("failed to get current migrations: %w", err)
			}

			if len(currentMigrations) == 0 {
				engine.PrintInfo("No migrations to refresh")
				return nil
			}

			engine.PrintWarning("About to refresh %d migrations", len(currentMigrations))
			confirmed := engine.ConfirmPrompt("Continue with refresh?")
			if !confirmed {
				engine.PrintInfo("Refresh cancelled")
				return nil
			}
		}

		if err := mm.Refresh(); err != nil {
			return fmt.Errorf("failed to refresh migrations: %w", err)
		}
	}

	if options.Seed {
		if rc.config.RunnerPath == "" {
			return fmt.Errorf("--seed needs Config.RunnerPath set (runner_path in .gorun/config.yaml, see `gorun setup`) - or skip --seed and run `gorun seed run` yourself afterward")
		}

		engine.PrintInfo("Preparing to run seeders...")

		seedArgs := []string{
			"seed",
			"run",
			"--type=" + string(dbType),
			"--database=" + dbName,
		}

		cmdArgs := append([]string{"run", rc.config.RunnerPath}, seedArgs...)

		engine.PrintInfo("Executing: go %s", strings.Join(cmdArgs, " "))

		seedCmd := exec.Command("go", cmdArgs...)
		seedCmd.Stdout = os.Stdout
		seedCmd.Stderr = os.Stderr

		if err := seedCmd.Run(); err != nil {
			return fmt.Errorf("seed command failed: %w", err)
		}

		engine.PrintSuccess("Seeders completed successfully!")
	}

	engine.PrintSuccess("Database refreshed successfully!")
	return nil
}
