package engine

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/urfave/cli/v3"
	"gorm.io/gorm"
)

// DatabaseSelectorInfo is one database's display row in the interactive
// selection table (name + human-readable size).
type DatabaseSelectorInfo struct {
	Number int
	Name   string
	Size   string
}

// DatabaseSelector prompts the user to pick a specific database (as
// opposed to DatabaseManager, which picks the engine).
type DatabaseSelector struct {
	dbManager *DatabaseManager
}

// NewDatabaseSelector builds a DatabaseSelector backed by dbManager.
func NewDatabaseSelector(dbManager *DatabaseManager) *DatabaseSelector {
	return &DatabaseSelector{
		dbManager: dbManager,
	}
}

// ResolveDatabaseName reads the "database" flag off cmd if set, otherwise
// defers to SelectDatabase - the standard pattern for commands that accept
// --database/--db to skip both the config lookup and the prompt.
func (ds *DatabaseSelector) ResolveDatabaseName(cmd *cli.Command, gormDB *gorm.DB, dbType DatabaseType, action string) (string, error) {
	if s := cmd.String("database"); s != "" {
		return s, nil
	}
	return ds.SelectDatabase(gormDB, dbType, action)
}

// SelectDatabase resolves which specific database on dbType's server to
// operate on. If the project's own config already names one
// (DBConnConfig.DatabaseName - the common case: one project, one
// database, decided once at setup time rather than re-picked from a
// server-wide list on every single command, the way Laravel's
// DB_DATABASE works) that's used directly, no round-trip to list
// anything. Only falls back to listing every database on the server and
// prompting when config leaves DatabaseName empty.
func (ds *DatabaseSelector) SelectDatabase(gormDB *gorm.DB, dbType DatabaseType, action string) (string, error) {
	if configured := ds.dbManager.getConfiguredDatabase(dbType); configured != "" {
		return configured, nil
	}

	databases, err := ds.listDatabasesWithSizes(gormDB, dbType)
	if err != nil {
		return "", fmt.Errorf("failed to list databases: %w", err)
	}

	if len(databases) == 0 {
		return "", fmt.Errorf("no databases found for %s", dbType)
	}

	ds.displayDatabaseTable(databases)

	selectedDB, err := ds.promptSingleDatabaseSelection(databases, action)
	if err != nil {
		return "", err
	}

	if selectedDB == nil {
		return "", fmt.Errorf("no database selected")
	}

	PrintSuccess("Selected database: %s", selectedDB.Name)
	return selectedDB.Name, nil
}

func (ds *DatabaseSelector) displayDatabaseTable(databases []DatabaseSelectorInfo) {
	PrintTextH1("Available Databases:")

	table := NewTable([]string{"No", "Database Name", "Size"})

	for _, db := range databases {
		table.AddRow([]string{
			fmt.Sprintf("%2d.", db.Number),
			db.Name,
			db.Size,
		})
	}

	table.SetColumnConfig(0, ColumnConfig{HeaderAlign: AlignCenter, ContentAlign: AlignCenter, MinWidth: 3, MaxWidth: 3}) // No
	table.SetColumnConfig(1, ColumnConfig{HeaderAlign: AlignLeft, ContentAlign: AlignLeft, MinWidth: 20})                 // Database Name
	table.SetColumnConfig(2, ColumnConfig{HeaderAlign: AlignLeft, ContentAlign: AlignLeft, MinWidth: 10})                 // Size

	table.DrawHorizontal()
}

func (ds *DatabaseSelector) listDatabasesWithSizes(gormDB *gorm.DB, dbType DatabaseType) ([]DatabaseSelectorInfo, error) {
	var databases []DatabaseSelectorInfo

	dbNames, err := ds.listDatabaseNames(gormDB, dbType)
	if err != nil {
		return nil, err
	}

	for i, name := range dbNames {
		size, err := ds.getDatabaseSize(gormDB, name, dbType)
		if err != nil {
			size = "unknown"
		}
		databases = append(databases, DatabaseSelectorInfo{
			Number: i + 1,
			Name:   name,
			Size:   size,
		})
	}

	return databases, nil
}

func (ds *DatabaseSelector) listDatabaseNames(gormDB *gorm.DB, dbType DatabaseType) ([]string, error) {
	var query string
	switch dbType {
	case MySQL:
		query = "SHOW DATABASES"
	case PostgreSQL:
		query = "SELECT datname FROM pg_database WHERE datistemplate = false"
	default:
		return nil, fmt.Errorf("unsupported database type")
	}

	rows, err := gormDB.Raw(query).Rows()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var databases []string
	for rows.Next() {
		var dbName string
		if err := rows.Scan(&dbName); err != nil {
			return nil, err
		}
		if dbType == MySQL && ds.isSystemDatabase(dbName) {
			continue
		}
		databases = append(databases, dbName)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return databases, nil
}

func (ds *DatabaseSelector) isSystemDatabase(dbName string) bool {
	systemDBs := map[string]bool{
		"information_schema": true,
		"mysql":              true,
		"performance_schema": true,
		"sys":                true,
	}
	return systemDBs[dbName]
}

func (ds *DatabaseSelector) getDatabaseSize(gormDB *gorm.DB, dbName string, dbType DatabaseType) (string, error) {
	var query string
	var size string

	switch dbType {
	case MySQL:
		query = `SELECT ROUND(SUM(data_length + index_length) / 1024 / 1024, 2) AS 'size_mb' 
			FROM information_schema.tables 
			WHERE table_schema = ?`
	case PostgreSQL:
		query = "SELECT pg_size_pretty(pg_database_size(?))"
	default:
		return "", fmt.Errorf("unsupported database type")
	}

	err := gormDB.Raw(query, dbName).Scan(&size).Error
	if err != nil {
		// Size is cosmetic display info for `db list`/`db status`; a failed
		// lookup degrades to "unknown" instead of failing the whole command.
		return "unknown", nil //nolint:nilerr
	}

	if dbType == MySQL && size != "" && size != "unknown" {
		if _, err := strconv.ParseFloat(size, 64); err == nil {
			size += " MB"
		}
	}

	return size, nil
}

func (ds *DatabaseSelector) promptSingleDatabaseSelection(databases []DatabaseSelectorInfo, action string) (*DatabaseSelectorInfo, error) {
	reader := stdinReader
	PrintOptionPrompt(fmt.Sprintf("Select database to %s (enter number)", action), "")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return nil, nil
	}

	num, err := strconv.Atoi(input)
	if err != nil {
		return nil, fmt.Errorf("invalid selection: %s", input)
	}

	if num > 0 && num <= len(databases) {
		return &databases[num-1], nil
	}

	return nil, fmt.Errorf("selection out of range")
}
