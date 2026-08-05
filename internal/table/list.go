package table

import (
	"context"
	"fmt"

	"github.com/RohmanBenyRiyanto/gorun/internal/engine"
	"github.com/urfave/cli/v3"
)

// ListCommand implements `gorun table list` - browsing and multi-selecting
// tables in a chosen database, with per-table row/size/column metadata.
type ListCommand struct {
	config *engine.Config
}

// NewListCommand builds a ListCommand.
func NewListCommand(config *engine.Config) *ListCommand {
	return &ListCommand{
		config: config,
	}
}

// Handle prompts for engine and database, lists tables with metadata, and
// lets the user select some to inspect further.
func (c *ListCommand) Handle(ctx context.Context, cmd *cli.Command) error {
	dbManager := engine.NewDatabaseManager(c.config)
	dbType := dbManager.PromptDatabaseSelection()

	gormDB, sqlDB, err := dbManager.InitializeDatabase(dbType, c.config)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	selector := engine.NewDatabaseSelector(dbManager)
	dbName, err := selector.SelectDatabase(gormDB, dbType, "list tables from")
	if err != nil {
		return fmt.Errorf("failed to select database: %w", err)
	}

	gormDB, sqlDB, err = dbManager.InitializeDatabaseWithName(dbType, dbName, c.config)
	if err != nil {
		return fmt.Errorf("failed to reinitialize database with selected DB name: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	tables, err := dbManager.GetTablesWithMetadata(gormDB, dbType, dbName)
	if err != nil {
		return fmt.Errorf("failed to get tables: %w", err)
	}

	var selectItems []engine.MultiSelectItem
	for _, table := range tables {
		description := fmt.Sprintf("Rows: %s | Size: %s | Columns: %d",
			formatNumber(table.Rows),
			table.Size,
			table.Columns)

		if table.Description != nil && *table.Description != "" {
			description += fmt.Sprintf("\nDescription: %s", *table.Description)
		}

		selectItems = append(selectItems, engine.SimpleSelectItem{
			Name:        table.Name,
			Description: description,
		})
	}

	config := engine.DefaultMultiSelectConfig()
	config.Title = fmt.Sprintf("TABLES IN '%s' (%d tables)", dbName, len(tables))
	config.Prompt = "Select tables (e.g., 1, 3-5, * for all, or leave blank)"
	config.ShowNumbers = true
	config.ShowDescriptions = true
	config.PageSize = 20
	config.AllowEmpty = true

	selected, err := engine.PromptMultiSelect(selectItems, config)
	if err != nil {
		return fmt.Errorf("failed to select tables: %w", err)
	}

	if len(selected) > 0 {
		var selectedTables []engine.TableInfo
		for _, item := range selected {
			for _, table := range tables {
				if table.Name == item.GetDisplayName() {
					selectedTables = append(selectedTables, table)
					break
				}
			}
		}
		displaySelectedTables(selectedTables)
	}

	return nil
}

func formatNumber(n int64) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	}
	if n >= 1000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}

func displaySelectedTables(tables []engine.TableInfo) {
	engine.PrintDivider()
	engine.PrintSuccess("Selected Tables (%d):", len(tables))

	table := engine.NewTable([]string{
		"Table Name", "Rows", "Size", "Data Size", "Index Size", "Columns", "Schema",
	})

	for _, tbl := range tables {
		row := []string{
			tbl.Name,
			formatNumber(tbl.Rows),
			tbl.Size,
			tbl.DataSize,
			tbl.IndexSize,
			fmt.Sprintf("%d", tbl.Columns),
			tbl.Schema,
		}
		table.AddRow(row)
	}

	table.SetColumnConfig(0, engine.ColumnConfig{HeaderAlign: engine.AlignLeft, ContentAlign: engine.AlignLeft, MinWidth: 20})
	table.SetColumnConfig(1, engine.ColumnConfig{HeaderAlign: engine.AlignLeft, ContentAlign: engine.AlignLeft, MinWidth: 8})
	table.SetColumnConfig(2, engine.ColumnConfig{HeaderAlign: engine.AlignCenter, ContentAlign: engine.AlignCenter, MinWidth: 8})
	table.SetColumnConfig(3, engine.ColumnConfig{HeaderAlign: engine.AlignCenter, ContentAlign: engine.AlignCenter, MinWidth: 10})
	table.SetColumnConfig(4, engine.ColumnConfig{HeaderAlign: engine.AlignCenter, ContentAlign: engine.AlignCenter, MinWidth: 10})
	table.SetColumnConfig(5, engine.ColumnConfig{HeaderAlign: engine.AlignLeft, ContentAlign: engine.AlignLeft, MinWidth: 8})
	table.SetColumnConfig(6, engine.ColumnConfig{HeaderAlign: engine.AlignLeft, ContentAlign: engine.AlignLeft, MinWidth: 10})

	table.DrawHorizontal()

	var hasDescriptions bool
	for _, table := range tables {
		if table.Description != nil && *table.Description != "" {
			hasDescriptions = true
			break
		}
	}

	if hasDescriptions {
		engine.PrintInfo("\nTable Descriptions:")
		for _, table := range tables {
			if table.Description != nil && *table.Description != "" {
				engine.PrintInfo("\n%s:", table.Name)
				engine.PrintInfo("  %s", *table.Description)
			}
		}
	}
}
