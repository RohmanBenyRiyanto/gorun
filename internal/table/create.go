package table

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/RohmanBenyRiyanto/gorun/internal/engine"
	"github.com/urfave/cli/v3"
	"gorm.io/gorm"
)

// CreateCommand implements `gorun table create`, with an interactive
// wizard (template picker + SQL preview) when no flags are given, or a
// direct flag-driven mode otherwise.
type CreateCommand struct {
	config *engine.Config
}

// NewCreateCommand builds a CreateCommand.
func NewCreateCommand(config *engine.Config) *CreateCommand {
	return &CreateCommand{config: config}
}

// Handle runs the interactive wizard if none of --name/--database/--type/
// --schema were given, otherwise the direct command-mode flow.
func (c *CreateCommand) Handle(ctx context.Context, cmd *cli.Command) error {
	if !cmd.IsSet("name") && !cmd.IsSet("database") && !cmd.IsSet("type") && !cmd.IsSet("schema") {
		return c.runInteractiveMode()
	}
	return c.runCommandMode(cmd)
}

func (c *CreateCommand) runInteractiveMode() error {
	engine.PrintBoldCard("TABLE CREATION WIZARD (INTERACTIVE MODE)")

	dbManager := engine.NewDatabaseManager(c.config)

	dbType, err := c.promptDatabaseType(dbManager)
	if err != nil {
		return err
	}

	gormDB, sqlDB, err := dbManager.InitializeDatabase(dbType, c.config)
	if err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	dbName, err := c.promptDatabaseSelection(gormDB, dbType, dbManager)
	if err != nil {
		return err
	}

	gormDB, sqlDB, err = dbManager.InitializeDatabaseWithName(dbType, dbName, c.config)
	if err != nil {
		return fmt.Errorf("failed to connect to database '%s': %w", dbName, err)
	}
	defer func() { _ = sqlDB.Close() }()

	tableName, err := c.promptTableName(gormDB, dbType, dbName)
	if err != nil {
		return err
	}

	sql, err := c.promptTableTemplate(tableName, dbType)
	if err != nil {
		return err
	}

	if !c.confirmExecution(sql) {
		engine.PrintWarning("Table creation cancelled")
		return nil
	}

	if err := c.executeCreateTable(gormDB, sql); err != nil {
		return fmt.Errorf("table creation failed: %w", err)
	}

	engine.PrintSuccess("Table '%s' created successfully in database '%s'", tableName, dbName)
	return nil
}

func (c *CreateCommand) runCommandMode(cmd *cli.Command) error {
	engine.PrintBoldCard("TABLE CREATION (COMMAND MODE)")

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

	var dbName string
	if dbFlag := cmd.String("database"); dbFlag != "" {
		dbName = dbFlag
	} else {
		gormDB, sqlDB, err := dbManager.InitializeDatabase(dbType, c.config)
		if err != nil {
			return fmt.Errorf("failed to initialize database connection: %w", err)
		}
		defer func() { _ = sqlDB.Close() }()

		databases, err := dbManager.ListDatabases(gormDB, dbType)
		if err != nil {
			return fmt.Errorf("failed to list databases: %w", err)
		}
		if len(databases) == 0 {
			return fmt.Errorf("no databases available")
		}
		dbName = databases[0]
	}

	gormDB, sqlDB, err := dbManager.InitializeDatabaseWithName(dbType, dbName, c.config)
	if err != nil {
		return fmt.Errorf("failed to connect to database '%s': %w", dbName, err)
	}
	defer func() { _ = sqlDB.Close() }()

	tableName := cmd.String("name")
	if tableName == "" {
		return fmt.Errorf("table name is required")
	}
	if !isValidTableName(tableName) {
		return fmt.Errorf("invalid table name (alphanumeric and underscores only)")
	}

	exists, err := c.checkTableExists(gormDB, dbType, dbName, tableName)
	if err != nil {
		return fmt.Errorf("failed to check table existence: %w", err)
	}

	if exists && !cmd.Bool("force") {
		return fmt.Errorf("table '%s' already exists (use --force to overwrite)", tableName)
	}

	if exists {
		if err := c.dropTable(gormDB, dbType, tableName); err != nil {
			return fmt.Errorf("failed to drop existing table: %w", err)
		}
	}

	var sql string
	if schemaFile := cmd.String("schema"); schemaFile != "" {
		sqlBytes, err := os.ReadFile(schemaFile)
		if err != nil {
			return fmt.Errorf("failed to read schema file: %w", err)
		}
		sql = string(sqlBytes)
	} else {
		sql = c.generateEmptyTableSQL(tableName, dbType)
	}

	if err := c.executeCreateTable(gormDB, sql); err != nil {
		return fmt.Errorf("table creation failed: %w", err)
	}

	engine.PrintSuccess("Table '%s' created successfully in database '%s'", tableName, dbName)
	return nil
}

func (c *CreateCommand) promptDatabaseType(dbManager *engine.DatabaseManager) (engine.DatabaseType, error) {
	for {
		dbType := dbManager.PromptDatabaseSelection()
		if dbType == engine.MySQL || dbType == engine.PostgreSQL {
			return dbType, nil
		}
		engine.PrintError("Invalid database type selected")
		if !engine.ConfirmPrompt("Continue?") {
			return "", fmt.Errorf("operation cancelled by user")
		}
	}
}

func (c *CreateCommand) promptDatabaseSelection(gormDB *gorm.DB, dbType engine.DatabaseType, dbManager *engine.DatabaseManager) (string, error) {
	databases, err := dbManager.ListDatabases(gormDB, dbType)
	if err != nil {
		return "", fmt.Errorf("failed to list databases: %w", err)
	}

	if len(databases) == 0 {
		return "", fmt.Errorf("no databases available")
	}

	selected := engine.PromptNumberSelection("Available Databases:", databases, "")
	engine.PrintSuccess("Selected database: %s", selected)
	if selected == "" {
		if !engine.ConfirmPrompt("Continue?") {
			return "", fmt.Errorf("operation cancelled by user")
		}
		return "", fmt.Errorf("invalid database selection")
	}

	return selected, nil
}

func (c *CreateCommand) promptTableName(gormDB *gorm.DB, dbType engine.DatabaseType, dbName string) (string, error) {
	for {
		fmt.Println()
		input := engine.PromptInput("Enter table name", "")

		tableName := strings.TrimSpace(input)
		if tableName == "" {
			engine.PrintError("Table name cannot be empty")
			continue
		}

		if !isValidTableName(tableName) {
			engine.PrintError("Invalid table name (alphanumeric and underscores only)")
			continue
		}

		exists, err := c.checkTableExists(gormDB, dbType, dbName, tableName)
		if err != nil {
			return "", fmt.Errorf("failed to check table existence: %w", err)
		}
		if exists {
			engine.PrintError("Table '%s' already exists", tableName)
			if !engine.ConfirmPrompt("Continue?") {
				return "", fmt.Errorf("operation cancelled by user")
			}
			continue
		}

		return tableName, nil
	}
}

func (c *CreateCommand) promptTableTemplate(tableName string, dbType engine.DatabaseType) (string, error) {
	templates := []struct {
		Name        string
		Description string
		MySQLSQL    string
		PostgreSQL  string
	}{
		{
			Name:        "Users",
			Description: "User accounts with authentication fields",
			MySQLSQL: `CREATE TABLE %s (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  username VARCHAR(50) NOT NULL UNIQUE,
  email VARCHAR(100) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
			PostgreSQL: `CREATE TABLE %s (
  id BIGSERIAL PRIMARY KEY,
  username VARCHAR(50) NOT NULL UNIQUE,
  email VARCHAR(100) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER update_%s_timestamp
BEFORE UPDATE ON %s
FOR EACH ROW EXECUTE FUNCTION update_timestamp();`,
		},
		{
			Name:        "Products",
			Description: "Product catalog for e-commerce",
			MySQLSQL: `CREATE TABLE %s (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  price DECIMAL(10,2) NOT NULL,
  stock INT DEFAULT 0,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;`,
			PostgreSQL: `CREATE TABLE %s (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  price DECIMAL(10,2) NOT NULL,
  stock INT DEFAULT 0,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`,
		},
	}

	options := make([]string, 0, len(templates)+1)
	for _, t := range templates {
		options = append(options, fmt.Sprintf("%s - %s", t.Name, t.Description))
	}
	options = append(options, "Empty table (only ID field)")

	selected := engine.PromptNumberSelection("Select Table Template:", options, "1")

	switch selected {
	case "Empty table (only ID field)":
		return c.generateEmptyTableSQL(tableName, dbType), nil
	default:
		for i, opt := range options {
			if opt == selected {
				template := templates[i]
				var sql string
				switch dbType {
				case engine.MySQL:
					sql = template.MySQLSQL
				case engine.PostgreSQL:
					sql = template.PostgreSQL
				}
				return strings.ReplaceAll(sql, "%s", tableName), nil
			}
		}
	}

	if !engine.ConfirmPrompt("Continue?") {
		return "", fmt.Errorf("operation cancelled by user")
	}

	return "", fmt.Errorf("invalid template selection")
}

func (c *CreateCommand) generateEmptyTableSQL(tableName string, dbType engine.DatabaseType) string {
	switch dbType {
	case engine.MySQL:
		return fmt.Sprintf("CREATE TABLE %s (id BIGINT AUTO_INCREMENT PRIMARY KEY) ENGINE=InnoDB", tableName)
	case engine.PostgreSQL:
		return fmt.Sprintf("CREATE TABLE %s (id BIGSERIAL PRIMARY KEY)", tableName)
	}
	return ""
}

func (c *CreateCommand) confirmExecution(sql string) bool {
	engine.PrintDivider()
	engine.PrintTextH1("SQL Preview:")
	engine.PrintCodeBlock(sql, "Query")

	comfirm := engine.ConfirmPrompt("Execute this SQL?")
	return comfirm
}

func (c *CreateCommand) executeCreateTable(gormDB *gorm.DB, sql string) error {
	statements := strings.Split(sql, ";")

	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		if err := gormDB.Exec(stmt).Error; err != nil {
			return fmt.Errorf("SQL execution failed: %w\nStatement: %s", err, stmt)
		}
	}
	return nil
}

func (c *CreateCommand) checkTableExists(gormDB *gorm.DB, dbType engine.DatabaseType, dbName, tableName string) (bool, error) {
	var query string
	var args []interface{}

	switch dbType {
	case engine.MySQL:
		query = `SELECT COUNT(*) FROM information_schema.tables 
                 WHERE table_schema = ? AND table_name = ?`
		args = []interface{}{dbName, tableName}
	case engine.PostgreSQL:
		query = `SELECT COUNT(*) FROM information_schema.tables 
                 WHERE table_schema = ? AND table_name = ?`
		args = []interface{}{"public", tableName}
	}

	var count int64
	if err := gormDB.Raw(query, args...).Scan(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (c *CreateCommand) dropTable(gormDB *gorm.DB, dbType engine.DatabaseType, tableName string) error {
	engine.FancyProgressBar(fmt.Sprintf("Dropping table '%s'", tableName), 1*time.Second)

	var query string
	switch dbType {
	case engine.MySQL:
		query = fmt.Sprintf("DROP TABLE IF EXISTS `%s`", tableName)
	case engine.PostgreSQL:
		query = fmt.Sprintf("DROP TABLE IF EXISTS \"%s\"", tableName)
	}

	return gormDB.Exec(query).Error
}

func isValidTableName(name string) bool {
	return regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`).MatchString(name)
}
