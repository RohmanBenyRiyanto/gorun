package db

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/RohmanBenyRiyanto/gorun/internal/engine"
	"github.com/urfave/cli/v3"
	"gorm.io/gorm"
)

// DropCommand implements `gorun db drop`, including the interactive
// multi-database drop flow when no name is given.
type DropCommand struct {
	config *engine.Config
}

// NewDropCommand builds a DropCommand and prints its banner.
func NewDropCommand(config *engine.Config) *DropCommand {
	engine.PrintBoldCard("DATABASE COMMANDS DROP")
	return &DropCommand{config: config}
}

func (c *DropCommand) Handle(ctx context.Context, cmd *cli.Command) error {
	dbManager := engine.NewDatabaseManager(c.config)

	var dbType engine.DatabaseType
	if typeFlag := cmd.String("type"); typeFlag != "" {
		dbType = engine.DatabaseType(typeFlag)
		if err := dbManager.ValidateDatabaseType(dbType); err != nil {
			return fmt.Errorf("invalid database type: %w", err)
		}
	} else {
		dbType = dbManager.PromptDatabaseSelection()
	}

	gormDB, sqlDB, err := dbManager.InitializeDatabase(dbType, c.config)
	if err != nil {
		return fmt.Errorf("failed to initialize database connection: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	var dbName string
	if nameFlag := cmd.String("name"); nameFlag != "" {
		dbName = nameFlag
	} else if cmd.Args().Len() > 0 {
		dbName = cmd.Args().First()
	}

	force := cmd.Bool("force")

	if dbName != "" {
		engine.PrintInfo("Database name provided: %s", dbName)
		return c.dropSingleDatabase(dbManager, gormDB, dbName, dbType, force)
	}

	return c.handleInteractiveDrop(dbManager, gormDB, dbType, force)
}

func (c *DropCommand) dropSingleDatabase(dbManager *engine.DatabaseManager, gormDB *gorm.DB, dbName string, dbType engine.DatabaseType, force bool) error {
	engine.PrintDivider()
	engine.PrintInfo("Checking database existence...")

	exists, err := checkDatabaseExists(gormDB, dbName, dbType)
	if err != nil {
		return fmt.Errorf("failed to check database existence: %w", err)
	}

	if !exists {
		engine.PrintWarning("Database '%s' does not exist.", dbName)
		return nil
	}

	size, err := getDatabaseSize(gormDB, dbName, dbType)
	if err != nil {
		engine.PrintWarning("Could not determine database size: %v", err)
	} else {
		engine.PrintInfo("Database size: %s", size)
	}

	if !force {
		dbManager.PrintTargetSummary(dbName, dbType)
		if !engine.ConfirmPrompt(fmt.Sprintf("Are you sure you want to drop database '%s'", dbName)) {
			engine.PrintInfo("Operation cancelled by user.")
			return nil
		}
	}

	engine.FancyProgressBar(fmt.Sprintf("Dropping database '%s'", dbName), 1*time.Second)
	if err := dropDatabase(gormDB, dbName, dbType); err != nil {
		return fmt.Errorf("failed to drop database: %w", err)
	}

	engine.PrintSuccess("Database successfully dropped: %s", dbName)
	return nil
}

func (c *DropCommand) handleInteractiveDrop(dbManager *engine.DatabaseManager, gormDB *gorm.DB, dbType engine.DatabaseType, force bool) error {
	databases, err := listDatabasesWithSizes(gormDB, dbType)
	if err != nil {
		return fmt.Errorf("failed to list databases: %w", err)
	}

	if len(databases) == 0 {
		engine.PrintWarning("No databases found to drop.")
		return nil
	}

	displayDatabaseTable(databases)

	selectedDbs, err := c.promptDatabaseSelection(databases)
	if err != nil {
		return err
	}

	if len(selectedDbs) == 0 {
		engine.PrintInfo("No databases selected for dropping.")
		return nil
	}

	if !force {
		dbManager.PrintTargetSummary(fmt.Sprintf("%d database(s) selected, listed below", len(selectedDbs)), dbType)
		engine.PrintWarning("You are about to drop %d database(s):", len(selectedDbs))
		for _, db := range selectedDbs {
			engine.PrintWarning(" - %s (%s)", db.Name, db.Size)
		}

		if !engine.ConfirmPrompt("Are you sure you want to continue") {
			engine.PrintInfo("Operation cancelled by user.")
			return nil
		}
	}

	for _, db := range selectedDbs {
		engine.PrintDivider()
		engine.FancyProgressBar(fmt.Sprintf("Dropping database '%s'", db.Name), 1*time.Second)
		if err := dropDatabase(gormDB, db.Name, dbType); err != nil {
			engine.PrintError("Failed to drop database %s: %v", db.Name, err)
			engine.PrintDivider()
			continue
		}
		engine.PrintSuccess("Successfully dropped database: %s", db.Name)
		engine.PrintDivider()
	}

	return nil
}

func (c *DropCommand) promptDatabaseSelection(databases []DatabaseInfo) ([]DatabaseInfo, error) {
	reader := bufio.NewReader(os.Stdin)

	engine.PrintOptionPrompt("Select databases to drop (e.g. 1, 3-5, or * for all)", "")

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return nil, nil
	}

	if input == "*" {
		return databases, nil
	}

	var selected []DatabaseInfo
	parts := strings.Split(input, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid range format: %s", part)
			}

			start, err := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid start of range: %s", rangeParts[0])
			}

			end, err := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid end of range: %s", rangeParts[1])
			}

			if start > end {
				start, end = end, start
			}

			for i := start; i <= end; i++ {
				if i > 0 && i <= len(databases) {
					selected = append(selected, databases[i-1])
				}
			}
		} else {
			num, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid selection: %s", part)
			}

			if num > 0 && num <= len(databases) {
				selected = append(selected, databases[num-1])
			}
		}
	}

	return removeDuplicates(selected), nil
}

func removeDuplicates(dbs []DatabaseInfo) []DatabaseInfo {
	seen := make(map[string]bool)
	var result []DatabaseInfo

	for _, db := range dbs {
		if !seen[db.Name] {
			seen[db.Name] = true
			result = append(result, db)
		}
	}

	return result
}

// DropDatabaseWithDefaults drops dbName non-interactively, with no
// confirmation prompt - exported for callers that need to drop a database
// programmatically rather than through the CLI flow.
func DropDatabaseWithDefaults(dbName string, dbType engine.DatabaseType, config *engine.Config) error {
	dbManager := engine.NewDatabaseManager(config)
	gormDB, sqlDB, err := dbManager.InitializeDatabase(dbType, config)
	if err != nil {
		return err
	}
	defer func() { _ = sqlDB.Close() }()

	exists, err := checkDatabaseExists(gormDB, dbName, dbType)
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("database '%s' does not exist", dbName)
	}

	return dropDatabase(gormDB, dbName, dbType)
}

func checkDatabaseExists(gormDB *gorm.DB, dbName string, dbType engine.DatabaseType) (bool, error) {
	var exists bool
	var query string

	switch dbType {
	case engine.MySQL:
		query = "SELECT COUNT(*) FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME = ?"
	case engine.PostgreSQL:
		query = "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = ?)"
	}

	err := gormDB.Raw(query, dbName).Scan(&exists).Error
	return exists, err
}

func dropDatabase(gormDB *gorm.DB, dbName string, dbType engine.DatabaseType) error {
	var query string
	switch dbType {
	case engine.MySQL:
		query = fmt.Sprintf("DROP DATABASE `%s`", dbName)
	case engine.PostgreSQL:
		terminateQuery := `
			SELECT pg_terminate_backend(pid)
			FROM pg_stat_activity
			WHERE datname = ? AND pid <> pg_backend_pid();
		`

		if err := gormDB.Exec(terminateQuery, dbName).Error; err != nil {
			return fmt.Errorf("failed to terminate connections: %w", err)
		}

		query = fmt.Sprintf("DROP DATABASE \"%s\"", dbName)
	default:
		return fmt.Errorf("unsupported database type: %v", dbType)
	}

	return gormDB.Exec(query).Error
}
