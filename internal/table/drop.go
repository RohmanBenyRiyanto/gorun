package table

import (
	"context"
	"fmt"
	"time"

	"github.com/RohmanBenyRiyanto/gorun/internal/engine"
	"github.com/urfave/cli/v3"
	"gorm.io/gorm"
)

// DropCommand implements `gorun table drop`, with an interactive
// multi-table picker when no flags are given, or a direct flag-driven
// mode otherwise.
type DropCommand struct {
	config *engine.Config
}

// NewDropCommand builds a DropCommand.
func NewDropCommand(config *engine.Config) *DropCommand {
	return &DropCommand{config: config}
}

// Handle runs the interactive wizard if none of --name/--database/--type
// were given, otherwise the direct command-mode flow.
func (c *DropCommand) Handle(ctx context.Context, cmd *cli.Command) error {
	if !cmd.IsSet("name") && !cmd.IsSet("database") && !cmd.IsSet("type") {
		return c.runInteractiveMode()
	}
	return c.runCommandMode(cmd)
}

func (c *DropCommand) runInteractiveMode() error {
	engine.PrintBoldCard("TABLE DROP WIZARD (INTERACTIVE MODE)")
	engine.PrintWarning("WARNING: This operation is irreversible!")

	dbManager := engine.NewDatabaseManager(c.config)

	dbType := dbManager.PromptDatabaseSelection()

	gormDB, sqlDB, err := dbManager.InitializeDatabase(dbType, c.config)
	if err != nil {
		return fmt.Errorf("failed to connect to DB: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	dbName, err := c.promptDatabaseSelection(gormDB, dbType, dbManager)
	if err != nil {
		return err
	}

	gormDB, sqlDB, err = dbManager.InitializeDatabaseWithName(dbType, dbName, c.config)
	if err != nil {
		return fmt.Errorf("failed to connect to DB '%s': %w", dbName, err)
	}
	defer func() { _ = sqlDB.Close() }()

	tables, err := dbManager.GetTablesWithMetadata(gormDB, dbType, dbName)
	if err != nil {
		return fmt.Errorf("failed to list tables: %w", err)
	}
	if len(tables) == 0 {
		engine.PrintInfo("No tables found in database '%s'", dbName)
		return nil
	}

	tableNames := make([]string, len(tables))
	for i, table := range tables {
		tableNames[i] = table.Name
	}

	selectedTables, err := engine.PromptMultiSelectStrings(tableNames, "Select Tables to Drop")
	if err != nil {
		return fmt.Errorf("failed to get table selection: %w", err)
	}

	if len(selectedTables) == 0 {
		engine.PrintInfo("No tables selected")
		return nil
	}

	if !confirmDropTables(dbManager, dbType, dbName, selectedTables) {
		engine.PrintInfo("Operation cancelled")
		return nil
	}

	for _, tableName := range selectedTables {
		if err := dropTable(gormDB, dbType, tableName); err != nil {
			engine.PrintError("Failed to drop table '%s': %v", tableName, err)
			return err
		}
	}

	engine.PrintSuccess("Successfully dropped %d table(s)", len(selectedTables))
	return nil
}

func (c *DropCommand) runCommandMode(cmd *cli.Command) error {
	engine.PrintBoldCard("TABLE DROP (COMMAND MODE)")

	dbManager := engine.NewDatabaseManager(c.config)

	dbType := engine.DatabaseType(cmd.String("type"))
	if err := dbManager.ValidateDatabaseType(dbType); err != nil {
		return fmt.Errorf("invalid database type: %w", err)
	}

	dbName := cmd.String("database")
	if dbName == "" {
		return fmt.Errorf("database name is required")
	}

	gormDB, sqlDB, err := dbManager.InitializeDatabaseWithName(dbType, dbName, c.config)
	if err != nil {
		return fmt.Errorf("failed to connect to DB '%s': %w", dbName, err)
	}
	defer func() { _ = sqlDB.Close() }()

	tableName := cmd.String("name")
	if tableName == "" {
		return fmt.Errorf("table name is required")
	}

	if !cmd.Bool("force") {
		dbManager.PrintTargetSummary(dbName, dbType)
		if !engine.ConfirmPrompt(fmt.Sprintf("Are you sure you want to drop table '%s'?", tableName)) {
			engine.PrintInfo("Operation cancelled")
			return nil
		}
	}

	if err := dropTable(gormDB, dbType, tableName); err != nil {
		return fmt.Errorf("failed to drop table '%s': %w", tableName, err)
	}

	engine.PrintSuccess("Successfully dropped table '%s' from database '%s'", tableName, dbName)
	return nil
}

func dropTable(gormDB *gorm.DB, dbType engine.DatabaseType, tableName string) error {
	fmt.Println()
	engine.FancyProgressBar(fmt.Sprintf("Dropping table '%s'", tableName), 300*time.Millisecond)

	query := fmt.Sprintf("DROP TABLE IF EXISTS %s", formatTableName(dbType, tableName))
	if err := gormDB.Exec(query).Error; err != nil {
		return fmt.Errorf("failed to execute DROP TABLE: %w", err)
	}

	engine.PrintSuccess("Table '%s' dropped successfully", tableName)
	fmt.Println()
	return nil
}

func formatTableName(dbType engine.DatabaseType, tableName string) string {
	switch dbType {
	case engine.MySQL:
		return fmt.Sprintf("`%s`", tableName)
	case engine.PostgreSQL:
		return fmt.Sprintf("\"%s\"", tableName)
	default:
		return tableName
	}
}

func confirmDropTables(dbManager *engine.DatabaseManager, dbType engine.DatabaseType, dbName string, tables []string) bool {
	dbManager.PrintTargetSummary(dbName, dbType)
	engine.PrintTextH1("⚠️  DANGER ZONE ⚠️")
	engine.PrintWarning("The following tables will be PERMANENTLY DELETED:")
	fmt.Println()

	for i, tableName := range tables {
		engine.PrintError("  %d. %s", i+1, tableName)
	}

	fmt.Println()
	engine.PrintWarning("This action cannot be undone!")
	engine.PrintWarning("All data in these tables will be lost forever!")
	fmt.Println()

	return engine.ConfirmPrompt(fmt.Sprintf("⚠️  Confirm DROP of %d table(s)?", len(tables)))
}

func (c *DropCommand) promptDatabaseSelection(gormDB *gorm.DB, dbType engine.DatabaseType, dbManager *engine.DatabaseManager) (string, error) {
	databases, err := dbManager.ListDatabases(gormDB, dbType)
	if err != nil {
		return "", fmt.Errorf("failed to list databases: %w", err)
	}

	if len(databases) == 0 {
		return "", fmt.Errorf("no databases available")
	}

	selectedDB, err := engine.PromptSingleSelectString(databases, "Available Databases")
	if err != nil {
		return "", fmt.Errorf("database selection failed: %w", err)
	}

	return selectedDB, nil
}
