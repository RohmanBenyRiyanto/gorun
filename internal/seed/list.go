package seed

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/RohmanBenyRiyanto/gorun/internal/engine"
	"github.com/urfave/cli/v3"
)

// ListCommand implements `gorun seed list`.
type ListCommand struct {
	config *engine.Config
}

// NewListCommand builds a ListCommand.
func NewListCommand(config *engine.Config) *ListCommand {
	return &ListCommand{
		config: config,
	}
}

// Handle runs the interactive engine-selection flow if --type wasn't
// passed, otherwise the direct command-line flow.
func (lc *ListCommand) Handle(ctx context.Context, cmd *cli.Command) error {
	if !cmd.IsSet("type") {
		return lc.runInteractiveMode()
	}
	return lc.runCommandMode(cmd)
}

func (lc *ListCommand) runInteractiveMode() error {
	engine.PrintBoldCard("SEEDER LIST (INTERACTIVE MODE)")

	dbManager := engine.NewDatabaseManager(lc.config)
	dbType := dbManager.PromptDatabaseSelection()

	return lc.listSeeders(dbType, "")
}

func (lc *ListCommand) runCommandMode(cmd *cli.Command) error {
	engine.PrintBoldCard("SEEDER LIST (COMMAND MODE)")

	dbManager := engine.NewDatabaseManager(lc.config)
	dbType := engine.DatabaseType(cmd.String("type"))

	if err := dbManager.ValidateDatabaseType(dbType); err != nil {
		return fmt.Errorf("invalid database type: %w", err)
	}

	return lc.listSeeders(dbType, cmd.String("database"))
}

// seederRegistryFor returns the SeederRegistry the caller wired in via
// Config.MySQLSeeders / Config.PostgreSQLSeeders for dbType.
//
// This is the one deviation from a pure mechanical port: the source tool
// hardcoded imports of its own project's concrete seeder packages
// (infrastructure/persistence/seeders/mysql and .../postgresql). Those are
// project-specific generated code that has no place in a portable gorun
// package, so the registry is now something every consumer supplies
// through Config instead.
func seederRegistryFor(config *engine.Config, dbType engine.DatabaseType) engine.SeederRegistry {
	switch dbType {
	case engine.MySQL:
		return config.MySQLSeeders
	case engine.PostgreSQL:
		return config.PostgreSQLSeeders
	default:
		return nil
	}
}

func (lc *ListCommand) listSeeders(dbType engine.DatabaseType, dbName string) error {
	engine.PrintSectionHeader("SEEDER LISTING")

	dbManager := engine.NewDatabaseManager(lc.config)

	engine.FancyProgressBar("Initializing database connection", 100*time.Millisecond)

	var sqlDB *sql.DB
	var err error

	if dbName != "" {
		_, sqlDB, err = dbManager.InitializeDatabaseWithName(dbType, dbName, lc.config)
	} else {
		_, sqlDB, err = dbManager.InitializeDatabase(dbType, lc.config)
	}

	if err != nil {
		engine.PrintError("Failed to initialize database: %v", err)
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	registry := seederRegistryFor(lc.config, dbType)
	if registry == nil {
		engine.PrintError("No seeder registry configured for %s (set Config.MySQLSeeders / Config.PostgreSQLSeeders)", dbType)
		return fmt.Errorf("no seeder registry configured for database type %s - if you're using gorun as a library, set Config.MySQLSeeders/PostgreSQLSeeders; if you're using the global gorun binary, set runner_path in .gorun/config.yaml (see `gorun setup`)", dbType)
	}

	seederManager := engine.NewSeederManager(dbManager, dbType, lc.config, registry)
	if err := seederManager.RegisterAll(); err != nil {
		engine.PrintError("Failed to register seeders: %v", err)
		return fmt.Errorf("failed to register seeders: %w", err)
	}

	engine.PrintTextH1("Available Seeders")
	fmt.Println()
	engine.PrintDebug("Database Type: %s", dbType)
	if dbName != "" {
		engine.PrintInfo("Database Name: %s", dbName)
	}

	names := seederManager.GetSeederNames()
	if len(names) == 0 {
		engine.PrintWarning("No seeders found in: %s", seederManager.GetSeederPath())
		return nil
	}

	table := engine.NewTable([]string{"No", "Seeder Name", "Status"})
	for i, name := range names {
		table.AddRow([]string{
			fmt.Sprintf("%d", i+1),
			name,
			engine.Green("Ready"),
		})
	}

	table.SetColumnConfig(0, engine.ColumnConfig{
		HeaderAlign:  engine.AlignCenter,
		ContentAlign: engine.AlignCenter,
		MinWidth:     5,
	})
	table.SetColumnConfig(1, engine.ColumnConfig{
		HeaderAlign:  engine.AlignLeft,
		ContentAlign: engine.AlignLeft,
		MinWidth:     30,
	})
	table.SetColumnConfig(2, engine.ColumnConfig{
		HeaderAlign:  engine.AlignCenter,
		ContentAlign: engine.AlignCenter,
		MinWidth:     10,
	})

	table.DrawHorizontal()

	engine.PrintDivider()
	engine.PrintKeyValueTable([]string{
		fmt.Sprintf("Database Type|%s", dbType),
		fmt.Sprintf("Seeder Path|%s", seederManager.GetSeederPath()),
		fmt.Sprintf("Total Seeders|%d", len(names)),
	}, engine.ColorCyan, engine.ColorForeground)

	return nil
}
