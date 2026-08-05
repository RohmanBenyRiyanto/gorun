package table

import (
	"context"
	"fmt"

	"github.com/RohmanBenyRiyanto/gorun/internal/engine"
	"github.com/urfave/cli/v3"
	"gorm.io/gorm"
)

// TruncateCommand implements `gorun table truncate` - clearing one or more
// individual tables (as opposed to `db truncate`, which clears every
// table in a database).
type TruncateCommand struct {
	config *engine.Config
}

// NewTruncateCommand builds a TruncateCommand.
func NewTruncateCommand(config *engine.Config) *TruncateCommand {
	return &TruncateCommand{
		config: config,
	}
}

// Handle truncates a single named table (--name) or, interactively, any
// number of tables the user multi-selects.
func (c *TruncateCommand) Handle(ctx context.Context, cmd *cli.Command) error {
	engine.PrintBoldCard("TABLE TRUNCATE WIZARD")
	engine.PrintDivider()
	engine.PrintWarning("WARNING: This will permanently delete all data in the selected tables!")
	engine.PrintDivider()

	dbManager := engine.NewDatabaseManager(c.config)

	if cmd.String("name") != "" {
		return c.truncateSingleTable(dbManager, cmd)
	}

	return c.truncateInteractive(dbManager)
}

func (c *TruncateCommand) truncateSingleTable(dbManager *engine.DatabaseManager, cmd *cli.Command) error {
	dbType := engine.DatabaseType(cmd.String("type"))
	if dbType == "" {
		dbType = dbManager.PromptDatabaseSelection()
	}

	dbName := cmd.String("database")
	if dbName == "" {
		dbName = dbManager.GetConfiguredDatabase(dbType)
	}

	gormDB, sqlDB, err := dbManager.InitializeDatabaseWithName(dbType, dbName, c.config)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	tableName := cmd.String("name")
	if !cmd.Bool("force") {
		dbManager.PrintTargetSummary(dbName, dbType)
		if !engine.ConfirmPrompt(fmt.Sprintf("Truncate table %s? This cannot be undone!", tableName)) {
			engine.PrintInfo("Operation cancelled")
			return nil
		}
	}

	return c.executeTruncate(gormDB, []string{tableName})
}

func (c *TruncateCommand) truncateInteractive(dbManager *engine.DatabaseManager) error {
	dbType := dbManager.PromptDatabaseSelection()

	gormDB, sqlDB, err := dbManager.InitializeDatabase(dbType, c.config)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	selector := engine.NewDatabaseSelector(dbManager)
	dbName, err := selector.SelectDatabase(gormDB, dbType, "truncate tables from")
	if err != nil {
		return fmt.Errorf("failed to select database: %w", err)
	}

	gormDB, sqlDB, err = dbManager.InitializeDatabaseWithName(dbType, dbName, c.config)
	if err != nil {
		return fmt.Errorf("failed to reinitialize database: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	tables, err := dbManager.GetTablesWithMetadata(gormDB, dbType, dbName)
	if err != nil {
		return fmt.Errorf("failed to get tables: %w", err)
	}

	if len(tables) == 0 {
		engine.PrintInfo("No tables found in database '%s'", dbName)
		return nil
	}

	var selectItems []engine.MultiSelectItem
	for _, table := range tables {
		selectItems = append(selectItems, engine.SimpleSelectItem{
			Name:        table.Name,
			Description: fmt.Sprintf("Rows: %s | Size: %s", formatNumber(table.Rows), table.Size),
		})
	}

	config := engine.DefaultMultiSelectConfig()
	config.Title = fmt.Sprintf("SELECT TABLES TO TRUNCATE (%s)", dbName)
	config.Prompt = "Select tables to truncate (e.g., 1, 3-5)"
	config.AllowEmpty = false

	selected, err := engine.PromptMultiSelect(selectItems, config)
	if err != nil {
		return fmt.Errorf("table selection failed: %w", err)
	}

	var tableNames []string
	for _, item := range selected {
		tableNames = append(tableNames, item.GetDisplayName())
	}

	dbManager.PrintTargetSummary(dbName, dbType)
	if !engine.PromptConfirmSelection(selected, "truncate") {
		engine.PrintInfo("Operation cancelled")
		return nil
	}

	return c.executeTruncate(gormDB, tableNames)
}

func (c *TruncateCommand) executeTruncate(gormDB *gorm.DB, tableNames []string) error {
	fmt.Println()

	if gormDB.Name() == "mysql" {
		if err := gormDB.Exec("SET FOREIGN_KEY_CHECKS = 0").Error; err != nil {
			return fmt.Errorf("failed to disable foreign key checks: %w", err)
		}
		defer gormDB.Exec("SET FOREIGN_KEY_CHECKS = 1")
	}

	for _, tableName := range tableNames {
		var sql string
		switch gormDB.Name() {
		case "mysql":
			sql = fmt.Sprintf("TRUNCATE TABLE `%s`", tableName)
		case "postgres":
			sql = fmt.Sprintf("TRUNCATE TABLE \"%s\" RESTART IDENTITY CASCADE", tableName)
		default:
			sql = fmt.Sprintf("TRUNCATE TABLE %s", tableName)
		}

		if err := gormDB.Exec(sql).Error; err != nil {
			return fmt.Errorf("failed to truncate table %s: %w", tableName, err)
		}
		engine.PrintSuccess("Truncated table: %s", tableName)
	}

	fmt.Println()
	engine.PrintSuccess("Successfully truncated %d tables", len(tableNames))
	return nil
}
