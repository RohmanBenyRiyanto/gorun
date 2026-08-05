package engine

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/urfave/cli/v3"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

// DatabaseType is a supported database engine identifier ("mysql" or
// "postgresql").
type DatabaseType string

const (
	MySQL      DatabaseType = "mysql"
	PostgreSQL DatabaseType = "postgresql"
)

// DatabaseInfo describes one supported engine for display purposes (the
// engine-selection prompt, status tables, etc).
type DatabaseInfo struct {
	Type        DatabaseType
	DisplayName string
	Driver      string
	Description string
}

// TableInfo describes one table's metadata as reported by
// DatabaseManager.GetTablesWithMetadata.
type TableInfo struct {
	Number      int     // Display number
	Name        string  // Table name
	Rows        int64   // Estimated row count
	Size        string  // Formatted size
	DataSize    string  // Data size only
	IndexSize   string  // Index size
	Engine      string  // Storage engine (MySQL)
	Collation   string  // Table collation
	Schema      string  // Schema name
	Description *string // Table description/comment
	Columns     int     // Number of columns
	CreateTime  string  // Creation time
	UpdateTime  string  // Last update time
}

// DatabaseConfig bundles a DatabaseInfo with the Config and connection
// function needed to open it - see DatabaseManager.GetDatabaseConfig.
type DatabaseConfig struct {
	Info     DatabaseInfo
	Config   *Config
	InitFunc func(*Config) (*gorm.DB, *sql.DB, error)
}

// DatabaseManager is the shared entry point for engine selection and
// connection setup - every command that touches a database goes through
// one of these. Create it with NewDatabaseManager.
type DatabaseManager struct {
	availableDatabases []DatabaseInfo
	selectedConfig     *DatabaseConfig
	config             *Config
}

// NewDatabaseManager builds a DatabaseManager backed by config, aware of
// only the engine(s) config actually configures - not every engine gorun
// knows how to speak to. That's what lets ResolveDatabaseType/
// PromptDatabaseSelection auto-pick a lone configured engine instead of
// asking, and what lets a two-engines-but-not-MultiDB project be caught
// as a misconfiguration instead of silently offering a picker.
func NewDatabaseManager(config *Config) *DatabaseManager {
	dm := &DatabaseManager{config: config}

	if config.MySQL.IsConfigured() {
		dm.availableDatabases = append(dm.availableDatabases, DatabaseInfo{
			Type:        MySQL,
			DisplayName: "MySQL",
			Driver:      "github.com/go-sql-driver/mysql",
			Description: "MySQL engine",
		})
	}
	if config.PostgreSQL.IsConfigured() {
		dm.availableDatabases = append(dm.availableDatabases, DatabaseInfo{
			Type:        PostgreSQL,
			DisplayName: "PostgreSQL",
			Driver:      "github.com/lib/pq",
			Description: "PostgreSQL engine",
		})
	}

	return dm
}

// PromptDatabaseSelection asks the user to pick MySQL or PostgreSQL
// interactively.
func (dm *DatabaseManager) PromptDatabaseSelection() DatabaseType {
	return dm.PromptDatabaseSelectionWithTitle("SELECT DATABASE ENGINE:")
}

// ResolveDatabaseType reads the "type" flag off cmd if set - validating
// both that it's a real engine and that this project actually configures
// it - otherwise falls back to PromptDatabaseSelection, which handles the
// zero/one/many-configured-engines cases on its own. An explicit --type
// is accepted even when MultiDB is off: typing --type on the command line
// is itself an unambiguous, deliberate choice, unlike an interactive
// picker (or a bare fallback) that could hand you the wrong engine
// without you really registering that this project has two.
func (dm *DatabaseManager) ResolveDatabaseType(cmd *cli.Command) (DatabaseType, error) {
	if s := cmd.String("type"); s != "" {
		dbType := DatabaseType(s)
		if !dbType.IsValid() {
			return "", fmt.Errorf("invalid database engine: %s", s)
		}
		if err := dm.ValidateDatabaseType(dbType); err != nil {
			return "", fmt.Errorf("%s is not configured in this project (see .gorun/config.yaml)", s)
		}
		return dbType, nil
	}
	return dm.PromptDatabaseSelection(), nil
}

// PromptDatabaseSelectionWithTitle is PromptDatabaseSelection with a
// custom prompt title. It only actually prompts when there's a genuine
// choice to make:
//
//   - No engine configured: exits with an error - there's nothing to
//     select from, so showing an empty menu would just be confusing.
//   - Exactly one engine configured: returns it immediately, no prompt.
//     This is the common case, and it shouldn't cost an extra keystroke
//     just because gorun also knows how to speak to a second engine you
//     don't use.
//   - Two engines configured but Config.MultiDB is false: exits with an
//     error rather than silently offering a picker - see MultiDB's doc
//     comment for why this is treated as a misconfiguration to fix, not
//     a feature to fall into by accident.
//   - Two engines configured and MultiDB is true: prompts, as before.
func (dm *DatabaseManager) PromptDatabaseSelectionWithTitle(title string) DatabaseType {
	switch {
	case len(dm.availableDatabases) == 0:
		PrintError("No database engine configured. Run `gorun setup` or edit .gorun/config.yaml.")
		os.Exit(1)
	case len(dm.availableDatabases) == 1:
		selected := dm.availableDatabases[0]
		PrintInfo("Using %s (the only database engine configured)", selected.DisplayName)
		return selected.Type
	case !dm.config.MultiDB:
		PrintError("Multiple database engines are configured (%s) but multi_db is not enabled in .gorun/config.yaml.", dm.configuredEngineNames())
		PrintInfo("Set multi_db: true if this project genuinely needs both, or remove the one you don't use.")
		os.Exit(1)
	}

	fmt.Println()
	PrintTextH1(title)
	fmt.Println()

	for i, dbInfo := range dm.availableDatabases {
		PrintOption(i+1, dbInfo.DisplayName)
	}

	fmt.Println()
	PrintOptionPrompt("Select database number", fmt.Sprintf("1-%d", len(dm.availableDatabases)))

	reader := stdinReader
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	choice, err := strconv.Atoi(input)

	fmt.Println()
	if err != nil || choice < 1 || choice > len(dm.availableDatabases) {
		PrintError("Invalid selection. Exiting.")
		os.Exit(1)
	}

	selectedDB := dm.availableDatabases[choice-1]
	port := dm.getConfiguredPort(selectedDB.Type)
	PrintSuccess("Selected database: %s (Port: %s)", selectedDB.DisplayName, port)
	return selectedDB.Type
}

// configuredEngineNames joins the display names of every configured
// engine, for the MultiDB-not-enabled error message.
func (dm *DatabaseManager) configuredEngineNames() string {
	names := make([]string, len(dm.availableDatabases))
	for i, info := range dm.availableDatabases {
		names[i] = info.DisplayName
	}
	return strings.Join(names, ", ")
}

// InitializeDatabase opens a connection to dbType's server without
// selecting a specific database (MySQL: no schema; PostgreSQL: the
// "postgres" maintenance database) - used for operations like listing or
// creating databases.
func (dm *DatabaseManager) InitializeDatabase(dbType DatabaseType, config *Config) (*gorm.DB, *sql.DB, error) {
	PrintInfo("Initializing %s system connection...", dbType)
	PrintDivider()

	switch dbType {
	case MySQL:
		return dm.initMySQLSystemDB(config)
	case PostgreSQL:
		return dm.initPostgreSQLSystemDB(config)
	default:
		return nil, nil, fmt.Errorf("unsupported database type: %s", dbType)
	}
}

// InitializeDatabaseWithName opens a connection to a specific database on
// dbType's server.
func (dm *DatabaseManager) InitializeDatabaseWithName(dbType DatabaseType, dbName string, config *Config) (*gorm.DB, *sql.DB, error) {
	PrintInfo("Initializing %s connection to database: %s", dbType, dbName)
	PrintDivider()

	switch dbType {
	case MySQL:
		return dm.initMySQLDB(config, dbName)
	case PostgreSQL:
		return dm.initPostgreSQLDB(config, dbName)
	default:
		return nil, nil, fmt.Errorf("unsupported database type: %s", dbType)
	}

}

func (dm *DatabaseManager) initMySQLSystemDB(config *Config) (*gorm.DB, *sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/?charset=utf8mb4&parseTime=True&loc=Local",
		config.MySQL.User,
		config.MySQL.Password,
		config.MySQL.Host,
		config.MySQL.Port,
	)

	sqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open MySQL connection: %w", err)
	}

	sqlDB.SetMaxOpenConns(config.MySQL.MaxOpenConns)
	sqlDB.SetMaxIdleConns(config.MySQL.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(config.MySQL.ConnMaxLifetime)

	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, nil, fmt.Errorf("failed to ping MySQL: %w", err)
	}

	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn: sqlDB,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})

	if err != nil {
		_ = sqlDB.Close()
		return nil, nil, fmt.Errorf("failed to initialize GORM: %w", err)
	}

	return gormDB, sqlDB, nil
}

func (dm *DatabaseManager) initMySQLDB(config *Config, dbName string) (*gorm.DB, *sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		config.MySQL.User,
		config.MySQL.Password,
		config.MySQL.Host,
		config.MySQL.Port,
		dbName,
	)

	sqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open MySQL connection: %w", err)
	}

	sqlDB.SetMaxOpenConns(config.MySQL.MaxOpenConns)
	sqlDB.SetMaxIdleConns(config.MySQL.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(config.MySQL.ConnMaxLifetime)

	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, nil, fmt.Errorf("failed to ping MySQL: %w", err)
	}

	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn: sqlDB,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})

	if err != nil {
		_ = sqlDB.Close()
		return nil, nil, fmt.Errorf("failed to initialize GORM: %w", err)
	}

	return gormDB, sqlDB, nil
}

func (dm *DatabaseManager) initPostgreSQLSystemDB(config *Config) (*gorm.DB, *sql.DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=postgres port=%s sslmode=%s TimeZone=%s",
		config.PostgreSQL.Host,
		config.PostgreSQL.User,
		config.PostgreSQL.Password,
		config.PostgreSQL.Port,
		config.PostgreSQL.SslMode,
		config.PostgreSQL.TimeZone,
	)

	sqlDB, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open PostgreSQL connection: %w", err)
	}

	sqlDB.SetMaxOpenConns(config.PostgreSQL.MaxOpenConns)
	sqlDB.SetMaxIdleConns(config.PostgreSQL.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(config.PostgreSQL.ConnMaxLifetime)

	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: sqlDB,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})

	if err != nil {
		_ = sqlDB.Close()
		return nil, nil, fmt.Errorf("failed to initialize GORM: %w", err)
	}

	return gormDB, sqlDB, nil
}

func (dm *DatabaseManager) initPostgreSQLDB(config *Config, dbName string) (*gorm.DB, *sql.DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		config.PostgreSQL.Host,
		config.PostgreSQL.User,
		config.PostgreSQL.Password,
		dbName,
		config.PostgreSQL.Port,
		config.PostgreSQL.SslMode,
		config.PostgreSQL.TimeZone,
	)

	sqlDB, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open PostgreSQL connection: %w", err)
	}

	sqlDB.SetMaxOpenConns(config.PostgreSQL.MaxOpenConns)
	sqlDB.SetMaxIdleConns(config.PostgreSQL.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(config.PostgreSQL.ConnMaxLifetime)

	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: sqlDB,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})

	if err != nil {
		_ = sqlDB.Close()
		return nil, nil, fmt.Errorf("failed to initialize GORM: %w", err)
	}

	return gormDB, sqlDB, nil
}

func (dm *DatabaseManager) getConfiguredPort(dbType DatabaseType) string {
	switch dbType {
	case MySQL:
		return dm.config.MySQL.Port
	case PostgreSQL:
		return dm.config.PostgreSQL.Port
	default:
		return "N/A"
	}
}

func (dm *DatabaseManager) getConfiguredHost(dbType DatabaseType) string {
	switch dbType {
	case MySQL:
		return dm.config.MySQL.Host
	case PostgreSQL:
		return dm.config.PostgreSQL.Host
	default:
		return "N/A"
	}
}

func (dm *DatabaseManager) getConfiguredDatabase(dbType DatabaseType) string {
	switch dbType {
	case MySQL:
		return dm.config.MySQL.DatabaseName
	case PostgreSQL:
		return dm.config.PostgreSQL.DatabaseName
	default:
		return "N/A"
	}
}

// PrintTargetSummary prints exactly what a command is about to touch -
// engine, host:port, database - right before its own ConfirmPrompt.
// Single-engine/single-database auto-selection (see ResolveDatabaseType
// and DatabaseSelector.SelectDatabase) means a destructive command can
// now reach its confirmation with zero prompts before it, so that
// confirmation is the one remaining place a human notices "wait, that's
// not the server I meant" before anything actually happens - a bare
// "drop database 'X'?" doesn't give them enough to check.
func (dm *DatabaseManager) PrintTargetSummary(database string, dbType DatabaseType) {
	table := NewTable([]string{"Property", "Value"})
	table.AddRow([]string{"Engine", string(dbType)})
	table.AddRow([]string{"Host", fmt.Sprintf("%s:%s", dm.getConfiguredHost(dbType), dm.getConfiguredPort(dbType))})
	table.AddRow([]string{"Database", database})
	table.DrawVertical()
}

func (dm *DatabaseManager) getConfiguredUser(dbType DatabaseType) string {
	switch dbType {
	case MySQL:
		return dm.config.MySQL.User
	case PostgreSQL:
		return dm.config.PostgreSQL.User
	default:
		return "N/A"
	}
}

// GetDatabaseConfig resolves dbType into a DatabaseConfig (info + init
// function) and remembers it as the manager's currently selected config.
func (dm *DatabaseManager) GetDatabaseConfig(dbType DatabaseType, config *Config) (*DatabaseConfig, error) {
	var dbInfo DatabaseInfo
	var initFunc func(*Config) (*gorm.DB, *sql.DB, error)

	for _, info := range dm.availableDatabases {
		if info.Type == dbType {
			dbInfo = info
			break
		}
	}

	if dbInfo.Type == "" {
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}

	switch dbType {
	case MySQL:
		initFunc = dm.initMySQLSystemDB
	case PostgreSQL:
		initFunc = dm.initPostgreSQLSystemDB
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}

	dbConfig := &DatabaseConfig{
		Info:     dbInfo,
		Config:   config,
		InitFunc: initFunc,
	}

	dm.selectedConfig = dbConfig
	return dbConfig, nil
}

// GetSelectedConfig returns whatever DatabaseConfig GetDatabaseConfig last
// resolved, or nil if it hasn't been called yet.
func (dm *DatabaseManager) GetSelectedConfig() *DatabaseConfig {
	return dm.selectedConfig
}

// GetAvailableDatabases returns whichever engine(s) this project actually
// configures - not necessarily both, even though gorun knows how to speak
// to both.
func (dm *DatabaseManager) GetAvailableDatabases() []DatabaseInfo {
	return dm.availableDatabases
}

// GetDatabaseInfo looks up the static DatabaseInfo for dbType.
func (dm *DatabaseManager) GetDatabaseInfo(dbType DatabaseType) (DatabaseInfo, error) {
	for _, info := range dm.availableDatabases {
		if info.Type == dbType {
			return info, nil
		}
	}
	return DatabaseInfo{}, fmt.Errorf("database type %s not found", dbType)
}

// GetDriverImport returns the blank-import driver line to embed in
// generated seeder files.
func (dm *DatabaseManager) GetDriverImport(dbType DatabaseType) string {
	switch dbType {
	case MySQL:
		return `_ "github.com/go-sql-driver/mysql"`
	case PostgreSQL:
		return `_ "github.com/lib/pq"`
	default:
		return ""
	}
}

// GetFolderName returns dbType's on-disk folder name (currently just its
// string form).
func (dm *DatabaseManager) GetFolderName(dbType DatabaseType) string {
	return string(dbType)
}

// ValidateDatabaseType returns an error unless dbType is one of the known
// engines.
func (dm *DatabaseManager) ValidateDatabaseType(dbType DatabaseType) error {
	for _, info := range dm.availableDatabases {
		if info.Type == dbType {
			return nil
		}
	}
	return fmt.Errorf("unsupported database type: %s", dbType)
}

// PrintDatabaseInfo prints a summary table for dbType.
func (dm *DatabaseManager) PrintDatabaseInfo(dbType DatabaseType) {
	info, err := dm.GetDatabaseInfo(dbType)
	if err != nil {
		PrintError("Database type not found: %s", dbType)
		return
	}

	PrintSectionHeader(fmt.Sprintf("%s DATABASE INFORMATION", strings.ToUpper(string(info.Type))))

	table := NewTable([]string{"Property", "Value"})

	table.AddRow([]string{"Type", string(info.Type)})
	table.AddRow([]string{"Display Name", info.DisplayName})
	table.AddRow([]string{"Driver", info.Driver})
	table.AddRow([]string{"Host", dm.getConfiguredHost(dbType)})
	table.AddRow([]string{"Port", dm.getConfiguredPort(dbType)})
	table.AddRow([]string{"Database", dm.getConfiguredDatabase(dbType)})
	table.AddRow([]string{"User", dm.getConfiguredUser(dbType)})
	table.AddRow([]string{"Description", info.Description})

	table.DrawVertical()

	fmt.Println()
}

// PrintAvailableDatabases prints a table of every known engine and its
// configured host/port/user.
func (dm *DatabaseManager) PrintAvailableDatabases() {
	PrintSectionHeader("AVAILABLE DATABASE TYPES")

	table := NewTable([]string{
		"No", "Database", "Type", "Host:Port", "User", "Description",
	})

	for i, info := range dm.availableDatabases {
		host := dm.getConfiguredHost(info.Type)
		port := dm.getConfiguredPort(info.Type)
		user := dm.getConfiguredUser(info.Type)

		row := []string{
			fmt.Sprintf("%02d", i+1),
			info.DisplayName,
			string(info.Type),
			fmt.Sprintf("%s:%s", host, port),
			user,
			info.Description,
		}

		table.AddRow(row)
	}

	table.SetColumnConfig(0, ColumnConfig{HeaderAlign: AlignCenter, ContentAlign: AlignCenter, MinWidth: 3}) // No
	table.SetColumnConfig(1, ColumnConfig{HeaderAlign: AlignLeft, ContentAlign: AlignLeft, MinWidth: 15})    // Database
	table.SetColumnConfig(2, ColumnConfig{HeaderAlign: AlignCenter, ContentAlign: AlignCenter, MinWidth: 8}) // Type
	table.SetColumnConfig(3, ColumnConfig{HeaderAlign: AlignLeft, ContentAlign: AlignLeft, MinWidth: 15})    // Host:Port
	table.SetColumnConfig(4, ColumnConfig{HeaderAlign: AlignLeft, ContentAlign: AlignLeft, MinWidth: 10})    // User
	table.SetColumnConfig(5, ColumnConfig{HeaderAlign: AlignLeft, ContentAlign: AlignLeft, MinWidth: 20})    // Description

	table.DrawHorizontal()
	fmt.Println()
}

// TestConnection opens a connection to dbType and pings it (twice - once
// on the raw *sql.DB, once through GORM's own handle) to confirm both
// layers are healthy.
func (dm *DatabaseManager) TestConnection(dbType DatabaseType, config *Config) error {
	host := dm.getConfiguredHost(dbType)
	port := dm.getConfiguredPort(dbType)

	PrintInfo("Testing connection to %s at %s:%s", dbType, host, port)

	db, sqlDB, err := dm.InitializeDatabase(dbType, config)
	if err != nil {
		PrintError("Connection failed: %v", err)
		return err
	}
	defer func() { _ = sqlDB.Close() }()

	if err := sqlDB.Ping(); err != nil {
		PrintError("Ping failed: %v", err)
		return err
	}

	sqlDB2, err := db.DB()
	if err != nil {
		PrintError("Failed to get underlying sql.DB: %v", err)
		return err
	}

	if err := sqlDB2.Ping(); err != nil {
		PrintError("GORM ping failed: %v", err)
		return err
	}

	PrintSuccess("Connection successful to %s at %s:%s", dbType, host, port)
	return nil
}

// GetConnectionString renders a redacted (password masked) connection
// string for display purposes.
func (dm *DatabaseManager) GetConnectionString(dbType DatabaseType, config *Config) string {
	switch dbType {
	case MySQL:
		return fmt.Sprintf("mysql://%s:***@%s:%s/",
			config.MySQL.User,
			config.MySQL.Host,
			config.MySQL.Port)
	case PostgreSQL:
		return fmt.Sprintf("postgresql://%s:***@%s:%s/postgres",
			config.PostgreSQL.User,
			config.PostgreSQL.Host,
			config.PostgreSQL.Port)
	default:
		return "unknown database type"
	}
}

// GetDatabaseStatus collects dbType's configured connection details into a
// display-friendly map (used by PrintDatabaseStatus and `db status`).
func (dm *DatabaseManager) GetDatabaseStatus(dbType DatabaseType) map[string]interface{} {
	status := map[string]interface{}{
		"type":       string(dbType),
		"host":       dm.getConfiguredHost(dbType),
		"port":       dm.getConfiguredPort(dbType),
		"database":   dm.getConfiguredDatabase(dbType),
		"user":       dm.getConfiguredUser(dbType),
		"connection": dm.GetConnectionString(dbType, dm.config),
		"driver":     dm.GetDriverImport(dbType),
	}

	switch dbType {
	case MySQL:
		status["charset"] = dm.config.MySQL.Charset
		status["parse_time"] = dm.config.MySQL.ParseTime
		status["loc"] = dm.config.MySQL.Loc
		status["max_open_conns"] = dm.config.MySQL.MaxOpenConns
		status["max_idle_conns"] = dm.config.MySQL.MaxIdleConns
		status["conn_max_lifetime"] = dm.config.MySQL.ConnMaxLifetime
	case PostgreSQL:
		status["ssl_mode"] = dm.config.PostgreSQL.SslMode
		status["timezone"] = dm.config.PostgreSQL.TimeZone
		status["max_open_conns"] = dm.config.PostgreSQL.MaxOpenConns
		status["max_idle_conns"] = dm.config.PostgreSQL.MaxIdleConns
		status["conn_max_lifetime"] = dm.config.PostgreSQL.ConnMaxLifetime
	}

	return status
}

// PrintDatabaseStatus prints GetDatabaseStatus as a table.
func (dm *DatabaseManager) PrintDatabaseStatus(dbType DatabaseType) {
	status := dm.GetDatabaseStatus(dbType)

	PrintSectionHeader(fmt.Sprintf("%s DATABASE STATUS", strings.ToUpper(string(dbType))))

	table := NewTable([]string{"Property", "Value"})

	keyOrder := []string{"type", "host", "port", "database", "user", "connection", "driver"}

	for _, key := range keyOrder {
		if value, exists := status[key]; exists {
			displayKey := dm.formatStatusKey(key)
			table.AddRow([]string{displayKey, fmt.Sprintf("%v", value)})
		}
	}

	for key, value := range status {
		if !dm.isInSlice(key, keyOrder) {
			displayKey := dm.formatStatusKey(key)
			table.AddRow([]string{displayKey, fmt.Sprintf("%v", value)})
		}
	}

	table.DrawVertical()
	fmt.Println()
}

func (dm *DatabaseManager) formatStatusKey(key string) string {
	caser := cases.Title(language.English)
	return caser.String(strings.ReplaceAll(key, "_", " "))
}

func (dm *DatabaseManager) isInSlice(str string, slice []string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// PrintConnectionResult prints a small readiness table after opening a
// connection.
func (dm *DatabaseManager) PrintConnectionResult(dbType DatabaseType, gormDB *gorm.DB, sqlDB *sql.DB) {
	host := dm.getConfiguredHost(dbType)
	port := dm.getConfiguredPort(dbType)

	PrintSuccess("Database connection established successfully!")

	table := NewTable([]string{"Component", "Status", "Details"})

	table.AddRow([]string{
		strings.ToUpper(string(dbType)),
		"Connected",
		fmt.Sprintf("%s:%s → System Connection", host, port),
	})

	table.AddRow([]string{
		"GORM DB",
		dm.getStatusIcon(gormDB != nil),
		"ORM interface ready",
	})

	table.AddRow([]string{
		"SQL DB",
		dm.getStatusIcon(sqlDB != nil),
		"Raw database connection ready",
	})

	table.SetColumnConfig(0, ColumnConfig{HeaderAlign: AlignLeft, ContentAlign: AlignLeft, MinWidth: 12})    // Component
	table.SetColumnConfig(1, ColumnConfig{HeaderAlign: AlignCenter, ContentAlign: AlignCenter, MinWidth: 9}) // Status
	table.SetColumnConfig(2, ColumnConfig{HeaderAlign: AlignLeft, ContentAlign: AlignLeft, MinWidth: 30})    // Details

	table.DrawHorizontal()
	fmt.Println()
}

func (dm *DatabaseManager) getStatusIcon(status bool) string {
	if status {
		return "✓ Ready"
	}
	return "✗ Not Ready"
}

// PrintConnectionSummary prints a table summarizing every connection in
// connections.
func (dm *DatabaseManager) PrintConnectionSummary(connections []DatabaseType) {
	if len(connections) == 0 {
		PrintWarning("No database connections established")
		return
	}

	PrintSectionHeader("DATABASE CONNECTION SUMMARY")

	table := NewTable([]string{"Database", "Host", "Port", "Status", "Connection String"})

	for _, dbType := range connections {
		host := dm.getConfiguredHost(dbType)
		port := dm.getConfiguredPort(dbType)
		connectionString := dm.GetConnectionString(dbType, dm.config)

		table.AddRow([]string{
			strings.ToUpper(string(dbType)),
			host,
			port,
			"Connected",
			connectionString,
		})
	}

	table.SetColumnConfig(0, ColumnConfig{HeaderAlign: AlignLeft, ContentAlign: AlignLeft, MinWidth: 10})     // Database
	table.SetColumnConfig(1, ColumnConfig{HeaderAlign: AlignLeft, ContentAlign: AlignLeft, MinWidth: 12})     // Host
	table.SetColumnConfig(2, ColumnConfig{HeaderAlign: AlignCenter, ContentAlign: AlignCenter, MinWidth: 6})  // Port
	table.SetColumnConfig(3, ColumnConfig{HeaderAlign: AlignCenter, ContentAlign: AlignCenter, MinWidth: 10}) // Status
	table.SetColumnConfig(4, ColumnConfig{HeaderAlign: AlignLeft, ContentAlign: AlignLeft, MinWidth: 30})     // Connection String

	table.DrawHorizontal()
	fmt.Println()
}

// PrintQuickStatus prints a compact per-engine configuration table.
func (dm *DatabaseManager) PrintQuickStatus() {
	PrintSectionHeader("QUICK DATABASE STATUS")

	table := NewTable([]string{"Database", "Host:Port", "User", "Status"})

	for _, info := range dm.availableDatabases {
		host := dm.getConfiguredHost(info.Type)
		port := dm.getConfiguredPort(info.Type)
		user := dm.getConfiguredUser(info.Type)

		table.AddRow([]string{
			info.DisplayName,
			fmt.Sprintf("%s:%s", host, port),
			user,
			"Configured",
		})
	}

	table.SetColumnConfig(0, ColumnConfig{HeaderAlign: AlignLeft, ContentAlign: AlignLeft, MinWidth: 20})     // Database
	table.SetColumnConfig(1, ColumnConfig{HeaderAlign: AlignLeft, ContentAlign: AlignLeft, MinWidth: 18})     // Host:Port
	table.SetColumnConfig(2, ColumnConfig{HeaderAlign: AlignCenter, ContentAlign: AlignCenter, MinWidth: 10}) // User
	table.SetColumnConfig(3, ColumnConfig{HeaderAlign: AlignCenter, ContentAlign: AlignCenter, MinWidth: 12}) // Status

	table.DrawHorizontal()
	fmt.Println()
}

// GetConfiguredHost returns dbType's configured host.
func (dm *DatabaseManager) GetConfiguredHost(dbType DatabaseType) string {
	return dm.getConfiguredHost(dbType)
}

// GetConfiguredPort returns dbType's configured port.
func (dm *DatabaseManager) GetConfiguredPort(dbType DatabaseType) string {
	return dm.getConfiguredPort(dbType)
}

// GetConfiguredDatabase returns dbType's configured database name.
func (dm *DatabaseManager) GetConfiguredDatabase(dbType DatabaseType) string {
	return dm.getConfiguredDatabase(dbType)
}

// GetConfiguredUser returns dbType's configured user.
func (dm *DatabaseManager) GetConfiguredUser(dbType DatabaseType) string {
	return dm.getConfiguredUser(dbType)
}

// ListDatabases lists every non-system database on dbType's server.
func (dm *DatabaseManager) ListDatabases(gormDB *gorm.DB, dbType DatabaseType) ([]string, error) {
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
		if isSystemDatabase(dbType, dbName) {
			continue
		}
		databases = append(databases, dbName)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return databases, nil
}

func isSystemDatabase(dbType DatabaseType, dbName string) bool {
	commonSystemDBs := map[string]bool{
		"template0": true,
		"template1": true,
	}

	if commonSystemDBs[dbName] {
		return true
	}

	switch dbType {
	case MySQL:
		mysqlSystemDBs := map[string]bool{
			"information_schema": true,
			"mysql":              true,
			"performance_schema": true,
			"sys":                true,
		}
		return mysqlSystemDBs[dbName]
	case PostgreSQL:
		postgresSystemDBs := map[string]bool{
			"postgres": true,
		}
		return postgresSystemDBs[dbName]
	}

	return false
}

// GetTablesWithMetadata returns detailed metadata for every table in
// dbName.
func (dm *DatabaseManager) GetTablesWithMetadata(gormDB *gorm.DB, dbType DatabaseType, dbName string) ([]TableInfo, error) {
	switch dbType {
	case MySQL:
		return dm.getMySQLTablesDetailed(gormDB, dbName)
	case PostgreSQL:
		return dm.getPostgreSQLTablesDetailed(gormDB, dbName)
	default:
		return nil, fmt.Errorf("unsupported database type")
	}
}

func (dm *DatabaseManager) getMySQLTablesDetailed(gormDB *gorm.DB, dbName string) ([]TableInfo, error) {
	query := `
		SELECT
			t.table_name,
			t.table_rows,
			ROUND(t.data_length/1024/1024, 2) as data_mb,
			ROUND(t.index_length/1024/1024, 2) as index_mb,
			ROUND((t.data_length + t.index_length)/1024/1024, 2) as total_mb,
			t.engine,
			t.table_collation,
			t.table_schema,
			COALESCE(t.table_comment, ''),
			(SELECT COUNT(*) FROM information_schema.columns c
			 WHERE c.table_schema = t.table_schema AND c.table_name = t.table_name) as column_count,
			COALESCE(t.create_time, ''),
			COALESCE(t.update_time, '')
		FROM information_schema.tables t
		WHERE t.table_schema = ?
		ORDER BY t.table_name
	`

	rows, err := gormDB.Raw(query, dbName).Rows()
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var tables []TableInfo
	for rows.Next() {
		var table TableInfo
		var dataMB, indexMB, totalMB float64
		var description string

		err := rows.Scan(
			&table.Name, &table.Rows, &dataMB, &indexMB, &totalMB,
			&table.Engine, &table.Collation, &table.Schema, &description,
			&table.Columns, &table.CreateTime, &table.UpdateTime,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan table row: %w", err)
		}

		if description != "" {
			table.Description = &description
		}

		table.DataSize = fmt.Sprintf("%.2f MB", dataMB)
		table.IndexSize = fmt.Sprintf("%.2f MB", indexMB)
		table.Size = fmt.Sprintf("%.2f MB", totalMB)

		tables = append(tables, table)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read table rows: %w", err)
	}

	for i := range tables {
		tables[i].Number = i + 1
	}

	return tables, nil
}

func (dm *DatabaseManager) getPostgreSQLTablesDetailed(gormDB *gorm.DB, dbName string) ([]TableInfo, error) {
	var currentDB string
	if err := gormDB.Raw("SELECT current_database()").Scan(&currentDB).Error; err != nil {
		return nil, fmt.Errorf("failed to get current database: %w", err)
	}

	if currentDB != dbName {
		return nil, fmt.Errorf("database mismatch: expected %s, got %s", dbName, currentDB)
	}

	query := `
        SELECT
            t.table_name,
            COALESCE(c.reltuples::bigint, 0) as estimated_rows,
            COALESCE(pg_size_pretty(pg_table_size('"' || t.table_name || '"')), '0 bytes') as data_size,
            COALESCE(pg_size_pretty(pg_indexes_size('"' || t.table_name || '"')), '0 bytes') as index_size,
            COALESCE(pg_size_pretty(pg_total_relation_size('"' || t.table_name || '"')), '0 bytes') as total_size,
            COALESCE(obj_description(c.oid), '') as description,
            (SELECT COUNT(*) FROM information_schema.columns col
             WHERE col.table_catalog = current_database()
             AND col.table_schema = t.table_schema
             AND col.table_name = t.table_name) as column_count,
            t.table_schema
        FROM information_schema.tables t
        LEFT JOIN pg_class c ON c.relname = t.table_name
        LEFT JOIN pg_namespace n ON n.oid = c.relnamespace AND n.nspname = t.table_schema
        WHERE t.table_catalog = current_database()
        AND t.table_type = 'BASE TABLE'
        AND t.table_schema NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
        ORDER BY t.table_schema, t.table_name
    `

	rows, err := gormDB.Raw(query).Rows()
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var tables []TableInfo
	for rows.Next() {
		var table TableInfo
		var description string
		var schema string

		err := rows.Scan(
			&table.Name,
			&table.Rows,
			&table.DataSize,
			&table.IndexSize,
			&table.Size,
			&description,
			&table.Columns,
			&schema,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan table row: %w", err)
		}

		if description != "" {
			table.Description = &description
		}

		table.Schema = schema
		table.Engine = "PostgreSQL"
		table.Collation = "-"

		table.CreateTime = "-"
		table.UpdateTime = "-"

		tables = append(tables, table)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read table rows: %w", err)
	}

	for i := range tables {
		tables[i].Number = i + 1
	}

	return tables, nil
}

func (dt DatabaseType) String() string {
	switch dt {
	case MySQL:
		return "mysql"
	case PostgreSQL:
		return "postgresql"
	default:
		return "unknown"
	}
}

// IsValid reports whether dt is a recognized engine (MySQL or
// PostgreSQL).
func (dt DatabaseType) IsValid() bool {
	return dt == MySQL || dt == PostgreSQL
}
