package db

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/RohmanBenyRiyanto/gorun/internal/engine"
	"github.com/urfave/cli/v3"
	"gorm.io/gorm"
)

// TruncateCommand implements `gorun db truncate` - clearing every table in
// a database, in foreign-key-safe order.
type TruncateCommand struct {
	config *engine.Config
}

// NewTruncateCommand builds a TruncateCommand and prints its banner.
func NewTruncateCommand(config *engine.Config) *TruncateCommand {
	engine.PrintBoldCard("DATABASE COMMANDS TRUNCATE")
	return &TruncateCommand{config: config}
}

func (c *TruncateCommand) Handle(ctx context.Context, cmd *cli.Command) error {
	dbManager := engine.NewDatabaseManager(c.config)
	dbType := dbManager.PromptDatabaseSelection()
	gormDB, sqlDB, err := dbManager.InitializeDatabase(dbType, c.config)
	if err != nil {
		return fmt.Errorf("failed to initialize database connection: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	truncator := NewDatabaseTruncator(dbManager, gormDB, dbType)

	var dbName string
	if nameFlag := cmd.String("name"); nameFlag != "" {
		dbName = nameFlag
	} else if cmd.Args().Len() > 0 {
		dbName = cmd.Args().First()
	}

	if dbName != "" {
		engine.PrintInfo("Database name provided: %s", dbName)
		return truncator.TruncateDatabase(dbName)
	}

	return truncator.HandleInteractiveTruncate()
}

// DatabaseTruncator truncates every table in one database, computing a
// foreign-key-safe order first so referenced tables go last.
type DatabaseTruncator struct {
	dbManager *engine.DatabaseManager
	gormDB    *gorm.DB
	dbType    engine.DatabaseType
}

// NewDatabaseTruncator builds a DatabaseTruncator for an already-open
// connection.
func NewDatabaseTruncator(dbManager *engine.DatabaseManager, gormDB *gorm.DB, dbType engine.DatabaseType) *DatabaseTruncator {
	return &DatabaseTruncator{
		dbManager: dbManager,
		gormDB:    gormDB,
		dbType:    dbType,
	}
}

// TableInfo is one table's row count and size, as reported by
// listTablesWithMetadata.
type TableInfo struct {
	Name      string
	RowCount  int64
	SizeBytes int64
}

// ForeignKeyInfo is one foreign key constraint, used to topologically sort
// tables before truncating so a referenced table is cleared after
// whatever references it.
type ForeignKeyInfo struct {
	TableName        string
	ConstraintName   string
	ReferencedTable  string
	ColumnName       string
	ReferencedColumn string
}

// HandleInteractiveTruncate lists databases and prompts for one to
// truncate.
func (dt *DatabaseTruncator) HandleInteractiveTruncate() error {
	databases, err := listDatabasesWithSizes(dt.gormDB, dt.dbType)
	if err != nil {
		return fmt.Errorf("failed to list databases: %w", err)
	}

	if len(databases) == 0 {
		engine.PrintWarning("No databases found to truncate.")
		return nil
	}

	displayDatabaseTable(databases)

	selectedDb, err := promptSingleDatabaseSelection(databases, "truncate")
	if err != nil {
		return err
	}

	if selectedDb == nil {
		engine.PrintInfo("No database selected for truncation.")
		return nil
	}

	return dt.TruncateDatabase(selectedDb.Name)
}

// TruncateDatabase truncates every table in dbName, after confirming with
// the user (see confirmTruncation).
func (dt *DatabaseTruncator) TruncateDatabase(dbName string) error {
	exists, err := dt.checkDatabaseExists(dbName)
	if err != nil {
		return fmt.Errorf("failed to check database existence: %w", err)
	}

	if !exists {
		engine.PrintWarning("Database '%s' does not exist.", dbName)
		return nil
	}

	tables, err := dt.listTablesWithMetadata(dbName)
	if err != nil {
		return fmt.Errorf("failed to list tables: %w", err)
	}

	if len(tables) == 0 {
		engine.PrintWarning("No tables found in database '%s'", dbName)
		return nil
	}

	foreignKeys, err := dt.getForeignKeyConstraints(dbName)
	if err != nil {
		return fmt.Errorf("failed to analyze foreign key constraints: %w", err)
	}

	sortedTables := dt.sortTablesByDependency(tables, foreignKeys)

	dt.displayTablesWithMetadata(sortedTables)

	if !dt.confirmTruncation(dbName, sortedTables) {
		engine.PrintInfo("Operation cancelled by user.")
		return nil
	}

	return dt.performTruncation(dbName, sortedTables)
}

func (dt *DatabaseTruncator) listTablesWithMetadata(dbName string) ([]TableInfo, error) {
	var tables []TableInfo
	var query string

	switch dt.dbType {
	case engine.MySQL:
		if err := dt.gormDB.Exec(fmt.Sprintf("USE `%s`", dbName)).Error; err != nil {
			return nil, err
		}

		query = `
			SELECT 
				t.table_name,
				COALESCE(t.table_rows, 0) as row_count,
				COALESCE(t.data_length + t.index_length, 0) as size_bytes
			FROM information_schema.tables t
			WHERE t.table_schema = ? 
			AND t.table_type = 'BASE TABLE'
			ORDER BY t.table_name
		`
	case engine.PostgreSQL:
		query = `
			SELECT 
				t.table_name,
				COALESCE(s.n_tup_ins - s.n_tup_del, 0) as row_count,
				COALESCE(pg_total_relation_size(c.oid), 0) as size_bytes
			FROM information_schema.tables t
			LEFT JOIN pg_class c ON c.relname = t.table_name
			LEFT JOIN pg_stat_user_tables s ON s.relname = t.table_name
			WHERE t.table_schema = 'public' 
			AND t.table_catalog = $1
			AND t.table_type = 'BASE TABLE'
			ORDER BY t.table_name
		`
	}

	rows, err := dt.gormDB.Raw(query, dbName).Rows()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var table TableInfo
		if err := rows.Scan(&table.Name, &table.RowCount, &table.SizeBytes); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}

	return tables, nil
}

func (dt *DatabaseTruncator) getForeignKeyConstraints(dbName string) ([]ForeignKeyInfo, error) {
	var constraints []ForeignKeyInfo
	var query string

	switch dt.dbType {
	case engine.MySQL:
		query = `
			SELECT 
				kcu.table_name,
				kcu.constraint_name,
				kcu.referenced_table_name,
				kcu.column_name,
				kcu.referenced_column_name
			FROM information_schema.key_column_usage kcu
			JOIN information_schema.referential_constraints rc 
				ON kcu.constraint_name = rc.constraint_name
			WHERE kcu.table_schema = ? 
			AND kcu.referenced_table_name IS NOT NULL
			ORDER BY kcu.table_name, kcu.constraint_name
		`
	case engine.PostgreSQL:
		query = `
			SELECT 
				tc.table_name,
				tc.constraint_name,
				ccu.table_name as referenced_table,
				kcu.column_name,
				ccu.column_name as referenced_column
			FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage kcu 
				ON tc.constraint_name = kcu.constraint_name
			JOIN information_schema.constraint_column_usage ccu 
				ON ccu.constraint_name = tc.constraint_name
			WHERE tc.constraint_type = 'FOREIGN KEY'
			AND tc.table_catalog = $1
			ORDER BY tc.table_name, tc.constraint_name
		`
	}

	rows, err := dt.gormDB.Raw(query, dbName).Rows()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var fk ForeignKeyInfo
		if err := rows.Scan(&fk.TableName, &fk.ConstraintName, &fk.ReferencedTable, &fk.ColumnName, &fk.ReferencedColumn); err != nil {
			return nil, err
		}
		constraints = append(constraints, fk)
	}

	return constraints, nil
}

func (dt *DatabaseTruncator) sortTablesByDependency(tables []TableInfo, foreignKeys []ForeignKeyInfo) []TableInfo {
	dependencies := make(map[string][]string)
	inDegree := make(map[string]int)

	for _, table := range tables {
		dependencies[table.Name] = []string{}
		inDegree[table.Name] = 0
	}

	for _, fk := range foreignKeys {
		if _, exists := dependencies[fk.ReferencedTable]; exists {
			dependencies[fk.ReferencedTable] = append(dependencies[fk.ReferencedTable], fk.TableName)
			inDegree[fk.TableName]++
		}
	}

	var result []TableInfo
	queue := []string{}

	for tableName, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, tableName)
		}
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, table := range tables {
			if table.Name == current {
				result = append(result, table)
				break
			}
		}

		for _, dependent := range dependencies[current] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	if len(result) != len(tables) {
		engine.PrintWarning("Circular dependency detected. Using fallback ordering.")
		return tables
	}

	return result
}

func (dt *DatabaseTruncator) performTruncation(dbName string, tables []TableInfo) error {
	switch dt.dbType {
	case engine.MySQL:
		return dt.truncateMySQLTables(dbName, tables)
	case engine.PostgreSQL:
		return dt.truncatePostgreSQLTables(tables)
	default:
		return fmt.Errorf("unsupported database type: %v", dt.dbType)
	}
}

func (dt *DatabaseTruncator) truncateMySQLTables(dbName string, tables []TableInfo) error {
	tx := dt.gormDB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	if err := tx.Exec("SET FOREIGN_KEY_CHECKS = 0").Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to disable foreign key checks: %w", err)
	}

	for i := len(tables) - 1; i >= 0; i-- {
		table := tables[i]
		engine.FancyProgressBar(fmt.Sprintf("Truncating table '%s'", table.Name), 300*time.Millisecond)

		query := fmt.Sprintf("TRUNCATE TABLE `%s`.`%s`", dbName, table.Name)
		if err := tx.Exec(query).Error; err != nil {
			engine.PrintError("Failed to truncate table %s: %v", table.Name, err)
			continue
		}
		engine.PrintSuccess("Successfully truncated table: %s (%d rows)", table.Name, table.RowCount)
	}

	if err := tx.Exec("SET FOREIGN_KEY_CHECKS = 1").Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to enable foreign key checks: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (dt *DatabaseTruncator) truncatePostgreSQLTables(tables []TableInfo) error {
	tx := dt.gormDB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	var tableNames []string
	for _, table := range tables {
		tableNames = append(tableNames, fmt.Sprintf(`"%s"`, table.Name))
	}

	engine.FancyProgressBar("Truncating all tables with CASCADE", 500*time.Millisecond)

	query := fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", strings.Join(tableNames, ", "))
	if err := tx.Exec(query).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to truncate tables: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	for _, table := range tables {
		engine.PrintSuccess("Successfully truncated table: %s (%d rows)", table.Name, table.RowCount)
	}

	return nil
}

func (dt *DatabaseTruncator) checkDatabaseExists(dbName string) (bool, error) {
	var exists bool
	var query string

	switch dt.dbType {
	case engine.MySQL:
		query = "SELECT COUNT(*) > 0 FROM information_schema.schemata WHERE schema_name = ?"
	case engine.PostgreSQL:
		query = "SELECT COUNT(*) > 0 FROM pg_database WHERE datname = $1"
	default:
		return false, fmt.Errorf("unsupported database type: %v", dt.dbType)
	}

	err := dt.gormDB.Raw(query, dbName).Scan(&exists).Error
	return exists, err
}

func (dt *DatabaseTruncator) displayTablesWithMetadata(tables []TableInfo) {
	engine.PrintDivider()
	engine.PrintInfo("Tables in selected database (sorted by dependency):")

	table := engine.NewTable([]string{"No", "Table Name", "Row Count", "Size (MB)", "Status"})

	for i, tbl := range tables {
		sizeMB := float64(tbl.SizeBytes) / (1024 * 1024)
		status := "Ready"
		if tbl.RowCount == 0 {
			status = "Empty"
		}

		table.AddRow([]string{
			fmt.Sprintf("%d", i+1),
			tbl.Name,
			fmt.Sprintf("%d", tbl.RowCount),
			fmt.Sprintf("%.2f", sizeMB),
			status,
		})
	}

	table.SetColumnConfig(0, engine.ColumnConfig{HeaderAlign: engine.AlignCenter, ContentAlign: engine.AlignCenter, MinWidth: 3})
	table.SetColumnConfig(1, engine.ColumnConfig{HeaderAlign: engine.AlignLeft, ContentAlign: engine.AlignLeft, MinWidth: 20})
	table.SetColumnConfig(2, engine.ColumnConfig{HeaderAlign: engine.AlignRight, ContentAlign: engine.AlignRight, MinWidth: 10})
	table.SetColumnConfig(3, engine.ColumnConfig{HeaderAlign: engine.AlignRight, ContentAlign: engine.AlignRight, MinWidth: 10})
	table.SetColumnConfig(4, engine.ColumnConfig{HeaderAlign: engine.AlignCenter, ContentAlign: engine.AlignCenter, MinWidth: 8})

	table.DrawHorizontal()
	engine.PrintDivider()

}

func (dt *DatabaseTruncator) confirmTruncation(dbName string, tables []TableInfo) bool {
	reader := bufio.NewReader(os.Stdin)

	var totalRows int64
	var totalSize int64
	for _, table := range tables {
		totalRows += table.RowCount
		totalSize += table.SizeBytes
	}

	dt.dbManager.PrintTargetSummary(dbName, dt.dbType)
	engine.PrintWarning("DANGER: You are about to TRUNCATE ALL TABLES in database '%s'", dbName)
	engine.PrintInfo("This will permanently DELETE ALL DATA from %d tables", len(tables))
	engine.PrintInfo("Total rows to be deleted: %d", totalRows)
	engine.PrintInfo("Total size to be freed: %.2f MB", float64(totalSize)/(1024*1024))
	engine.PrintWarning("Tables to be truncated:")

	for _, table := range tables {
		engine.PrintDebug("  - %s (%d rows)", table.Name, table.RowCount)
	}

	engine.PrintWarning("⚠️  THIS OPERATION CANNOT BE UNDONE!")
	engine.PrintOptionPrompt("Type 'CONFIRM' to proceed or anything else to cancel:", "")

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	return input == "CONFIRM"
}
