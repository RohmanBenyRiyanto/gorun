package db

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/RohmanBenyRiyanto/gorun/internal/engine"
	"gorm.io/gorm"
)

// DatabaseInfo is one database's display row (number/name/size) for the
// `db` package's interactive selection tables.
type DatabaseInfo struct {
	Number int
	Name   string
	Size   string
}

func displayDatabaseTable(databases []DatabaseInfo) {
	engine.PrintInfo("Available Databases:")

	table := engine.NewTable([]string{"No", "Database Name", "Size"})

	for _, db := range databases {
		table.AddRow([]string{
			fmt.Sprintf("%2d.", db.Number),
			db.Name,
			db.Size,
		})
	}

	table.SetColumnConfig(0, engine.ColumnConfig{
		HeaderAlign:  engine.AlignCenter,
		ContentAlign: engine.AlignCenter,
		MinWidth:     3,
		MaxWidth:     3,
	})
	table.SetColumnConfig(1, engine.ColumnConfig{
		HeaderAlign:  engine.AlignLeft,
		ContentAlign: engine.AlignLeft,
		MinWidth:     20,
	})
	table.SetColumnConfig(2, engine.ColumnConfig{
		HeaderAlign:  engine.AlignRight,
		ContentAlign: engine.AlignRight,
		MinWidth:     10,
	})

	table.DrawHorizontal()
	engine.PrintDivider()
}

func listDatabasesWithSizes(gormDB *gorm.DB, dbType engine.DatabaseType) ([]DatabaseInfo, error) {
	var databases []DatabaseInfo

	dbNames, err := listDatabaseNames(gormDB, dbType)
	if err != nil {
		return nil, err
	}

	for i, name := range dbNames {
		size, err := getDatabaseSize(gormDB, name, dbType)
		if err != nil {
			size = "unknown"
		}
		databases = append(databases, DatabaseInfo{
			Number: i + 1,
			Name:   name,
			Size:   size,
		})
	}

	return databases, nil
}

func listDatabaseNames(gormDB *gorm.DB, dbType engine.DatabaseType) ([]string, error) {
	var query string
	switch dbType {
	case engine.MySQL:
		query = "SHOW DATABASES"
	case engine.PostgreSQL:
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
		if dbType == engine.MySQL && isSystemDatabase(dbName) {
			continue
		}
		databases = append(databases, dbName)
	}

	return databases, nil
}

func isSystemDatabase(dbName string) bool {
	systemDBs := map[string]bool{
		"information_schema": true,
		"mysql":              true,
		"performance_schema": true,
		"sys":                true,
	}
	return systemDBs[dbName]
}

func getDatabaseSize(gormDB *gorm.DB, dbName string, dbType engine.DatabaseType) (string, error) {
	var query string
	var size string

	switch dbType {
	case engine.MySQL:
		query = `SELECT ROUND(SUM(data_length + index_length) / 1024 / 1024, 2) AS 'size_mb' 
			FROM information_schema.tables 
			WHERE table_schema = ?`
	case engine.PostgreSQL:
		query = "SELECT pg_size_pretty(pg_database_size(?))"
	default:
		return "", fmt.Errorf("unsupported database type")
	}

	err := gormDB.Raw(query, dbName).Scan(&size).Error
	return size, err
}

func promptSingleDatabaseSelection(databases []DatabaseInfo, action string) (*DatabaseInfo, error) {
	reader := bufio.NewReader(os.Stdin)
	engine.PrintOptionPrompt(fmt.Sprintf("Select database to %s (enter number)", action), "")
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
