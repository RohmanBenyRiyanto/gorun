package seed

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/RohmanBenyRiyanto/gorun/internal/engine"
	"github.com/urfave/cli/v3"
	"gorm.io/gorm"
)

// RunCommand implements `gorun seed run`.
type RunCommand struct {
	config *engine.Config
}

// NewRunCommand builds a RunCommand.
func NewRunCommand(config *engine.Config) *RunCommand {
	return &RunCommand{config: config}
}

// isProdEnv reports whether env names a production environment. Mirrors
// the source tool's configs.App.IsProd(), just reading from
// Config.AppEnv instead of a project-specific App struct.
func isProdEnv(env string) bool {
	return env == "prod" || env == "production"
}

// Handle guards against running against production, resolves the target
// database, then runs one seeder (--class/--seeder) or every registered
// one.
func (rc *RunCommand) Handle(ctx context.Context, cmd *cli.Command) error {
	// Seeders aren't meant to be re-run casually against a live database.
	if isProdEnv(rc.config.AppEnv) && !cmd.Bool("force") {
		return fmt.Errorf("refusing to run seeders against production (app.env=%s) without --force", rc.config.AppEnv)
	}

	dbManager := engine.NewDatabaseManager(rc.config)

	dbType, err := dbManager.ResolveDatabaseType(cmd)
	if err != nil {
		return err
	}

	gormDB, sqlDB, err := dbManager.InitializeDatabase(dbType, rc.config)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	dbSelector := engine.NewDatabaseSelector(dbManager)
	dbName, err := dbSelector.ResolveDatabaseName(cmd, gormDB, dbType, "seed")
	if err != nil {
		return err
	}

	if dbName != "" {
		gormDB, sqlDB, err = dbManager.InitializeDatabaseWithName(dbType, dbName, rc.config)
		if err != nil {
			return fmt.Errorf("failed to connect to database %s: %w", dbName, err)
		}
		defer func() { _ = sqlDB.Close() }()
	}

	registry := seederRegistryFor(rc.config, dbType)
	if registry == nil {
		return fmt.Errorf("no seeder registry configured for database type %s - if you're using gorun as a library, set Config.MySQLSeeders/PostgreSQLSeeders; if you're using the global gorun binary, set runner_path in .gorun/config.yaml (see `gorun setup`)", dbType)
	}

	seederManager := engine.NewSeederManager(dbManager, dbType, rc.config, registry)
	seederManager.Options.Transaction = cmd.Bool("transaction")
	seederManager.Options.StopOnError = cmd.Bool("stop-on-error")
	seederManager.Options.Only = cmd.StringSlice("only")
	seederManager.Options.Except = cmd.StringSlice("except")

	seederName := rc.getSeederName(cmd)

	return rc.runSeeders(seederManager, gormDB, sqlDB, seederName)
}

func (rc *RunCommand) getSeederName(cmd *cli.Command) string {
	if seederName := cmd.String("class"); seederName != "" {
		return seederName
	}
	return cmd.String("seeder")
}

func (rc *RunCommand) runSeeders(seederManager *engine.SeederManager, gormDB *gorm.DB, sqlDB *sql.DB, seederName string) error {
	if seederName != "" {
		engine.PrintInfo("Running single seeder: %s", seederName)
		if err := seederManager.RunSeeders(gormDB, sqlDB, seederName); err != nil {
			return fmt.Errorf("failed to run seeder '%s': %w", seederName, err)
		}
		engine.PrintSuccess("Seeder '%s' completed successfully", seederName)
		return nil
	}

	engine.PrintInfo("Running all registered seeders")
	if err := seederManager.RunSeeders(gormDB, sqlDB, ""); err != nil {
		return fmt.Errorf("failed to run seeders: %w", err)
	}
	engine.PrintSuccess("All seeders completed successfully")
	return nil
}
