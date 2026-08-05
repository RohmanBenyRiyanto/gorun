package db

import (
	"context"
	"fmt"

	"github.com/RohmanBenyRiyanto/gorun/internal/engine"
	"github.com/urfave/cli/v3"
)

// ListCommand implements `gorun db list`.
type ListCommand struct {
	config *engine.Config
}

// NewListCommand builds a ListCommand and prints its banner.
func NewListCommand(config *engine.Config) *ListCommand {
	engine.PrintBoldCard("DATABASE COMMANDS LIST")
	return &ListCommand{config: config}
}

func (c *ListCommand) Handle(ctx context.Context, cmd *cli.Command) error {
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

	databases, err := listDatabasesWithSizes(gormDB, dbType)
	if err != nil {
		return fmt.Errorf("failed to list databases: %w", err)
	}

	if len(databases) == 0 {
		engine.PrintWarning("No databases found.")
		return nil
	}

	displayDatabaseTable(databases)
	return nil
}
