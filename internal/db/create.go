package db

import (
	"context"
	"fmt"
	"time"

	"github.com/RohmanBenyRiyanto/gorun/internal/engine"
	"github.com/urfave/cli/v3"
	"gorm.io/gorm"
)

// CreateCommand implements `gorun db create`.
type CreateCommand struct {
	config *engine.Config
}

// DatabaseCreationOptions is the charset/collation/encoding picked
// (interactively or via flags) for a new database.
type DatabaseCreationOptions struct {
	Charset   string
	Collation string
	Encoding  string
}

// NewCreateCommand builds a CreateCommand and prints its banner.
func NewCreateCommand(config *engine.Config) *CreateCommand {
	engine.PrintBoldCard("DATABASE COMMANDS CREATE")
	return &CreateCommand{config: config}
}

func (c *CreateCommand) Handle(ctx context.Context, cmd *cli.Command) error {
	engine.PrintInfo("Starting database creation process...")

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

	engine.PrintInfo("Initializing database connection for type: %s", dbType)
	gormDB, sqlDB, err := dbManager.InitializeDatabase(dbType, c.config)
	if err != nil {
		engine.PrintError("Failed to initialize database connection: %v", err)
		return fmt.Errorf("failed to initialize database connection: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	var dbName string
	if nameFlag := cmd.String("name"); nameFlag != "" {
		dbName = nameFlag
	} else if cmd.Args().Len() > 0 {
		dbName = cmd.Args().First()
	}

	if dbName == "" {
		defaultName := getDefaultDatabaseName(dbType, c.config)
		dbName = engine.PromptInput(
			"Enter database name",
			fmt.Sprintf("Leave blank for default (%s)", defaultName),
		)
		if dbName == "" {
			dbName = defaultName
		}
	}

	options := c.getDatabaseCreationOptions(dbType)

	if err := c.processDatabase(gormDB, dbName, dbType, options); err != nil {
		return err
	}

	c.displayCreationSuccess(dbName, options)
	return nil
}

func (c *CreateCommand) getDatabaseCreationOptions(dbType engine.DatabaseType) *DatabaseCreationOptions {
	options := &DatabaseCreationOptions{}
	engine.PrintDivider()

	switch dbType {
	case engine.MySQL:
		engine.PrintInfo("[MySQL Character Set & Collation Configuration]")
		options.Charset = getMySQLCharset(c.config)
		engine.PrintInfo("Default charset: %s", options.Charset)

		customCharset := engine.PromptInput(
			"Enter character set",
			fmt.Sprintf("Leave blank for default (%s)", options.Charset),
		)
		if customCharset != "" {
			options.Charset = customCharset
		}

		defaultCollation := fmt.Sprintf("%s_general_ci", options.Charset)
		customCollation := engine.PromptInput(
			"Enter collation",
			fmt.Sprintf("Leave blank for default (%s)", defaultCollation),
		)
		if customCollation != "" {
			options.Collation = customCollation
		} else {
			options.Collation = defaultCollation
		}

	case engine.PostgreSQL:
		engine.PrintInfo("[PostgreSQL Encoding & Collation Configuration]")
		options.Encoding = "UTF8"
		customEncoding := engine.PromptInput(
			"Enter encoding",
			"Leave blank for default (UTF8)",
		)
		if customEncoding != "" {
			options.Encoding = customEncoding
		}

		options.Collation = "C.UTF-8"
		customCollation := engine.PromptInput(
			"Enter collation",
			"Leave blank for default (C.UTF-8)",
		)
		if customCollation != "" {
			options.Collation = customCollation
		}
	}

	return options
}

func (c *CreateCommand) processDatabase(gormDB *gorm.DB, dbName string, dbType engine.DatabaseType, options *DatabaseCreationOptions) error {
	engine.PrintDivider()
	engine.PrintInfo("Checking database existence for: %s", dbName)

	exists, err := c.checkDatabaseExists(gormDB, dbName, dbType)
	if err != nil {
		return fmt.Errorf("failed to check database existence: %w", err)
	}

	if exists {
		engine.PrintWarning("Database '%s' already exists", dbName)
		if !engine.ConfirmPrompt("Drop and recreate it?") {
			engine.PrintInfo("Operation cancelled by user")
			return fmt.Errorf("operation cancelled")
		}

		if err := c.dropDatabase(gormDB, dbName, dbType); err != nil {
			return fmt.Errorf("failed to drop database: %w", err)
		}
	}

	return c.createDatabase(gormDB, dbName, dbType, options)
}

func (c *CreateCommand) checkDatabaseExists(gormDB *gorm.DB, dbName string, dbType engine.DatabaseType) (bool, error) {
	var query string
	switch dbType {
	case engine.MySQL:
		query = "SELECT SCHEMA_NAME FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME = ?"
	case engine.PostgreSQL:
		query = "SELECT 1 FROM pg_database WHERE datname = ?"
	}

	var result interface{}
	err := gormDB.Raw(query, dbName).Scan(&result).Error
	return result != nil, err
}

func (c *CreateCommand) dropDatabase(gormDB *gorm.DB, dbName string, dbType engine.DatabaseType) error {
	engine.FancyProgressBar(fmt.Sprintf("Dropping database '%s'", dbName), 1*time.Second)

	var query string
	switch dbType {
	case engine.MySQL:
		query = fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", dbName)
	case engine.PostgreSQL:
		query = fmt.Sprintf("DROP DATABASE IF EXISTS \"%s\"", dbName)
	}

	return gormDB.Exec(query).Error
}

func (c *CreateCommand) createDatabase(gormDB *gorm.DB, dbName string, dbType engine.DatabaseType, options *DatabaseCreationOptions) error {
	engine.FancyProgressBar(fmt.Sprintf("Creating database '%s'", dbName), 2*time.Second)

	query := c.buildCreateQuery(dbName, dbType, options)
	return gormDB.Exec(query).Error
}

func (c *CreateCommand) buildCreateQuery(dbName string, dbType engine.DatabaseType, options *DatabaseCreationOptions) string {
	switch dbType {
	case engine.MySQL:
		return fmt.Sprintf("CREATE DATABASE `%s` CHARACTER SET %s COLLATE %s",
			dbName, options.Charset, options.Collation)
	case engine.PostgreSQL:
		return fmt.Sprintf("CREATE DATABASE \"%s\" ENCODING='%s' LC_COLLATE='%s' LC_CTYPE='%s' TEMPLATE=template0",
			dbName, options.Encoding, options.Collation, options.Collation)
	default:
		return ""
	}
}

func (c *CreateCommand) displayCreationSuccess(dbName string, options *DatabaseCreationOptions) {
	engine.PrintDivider()
	engine.PrintSuccess("Database created: %s", dbName)
	if options.Charset != "" {
		engine.PrintSuccess("Charset: %s", options.Charset)
	}
	if options.Encoding != "" {
		engine.PrintSuccess("Encoding: %s", options.Encoding)
	}
	engine.PrintSuccess("Collation: %s", options.Collation)
}

func getDefaultDatabaseName(dbType engine.DatabaseType, config *engine.Config) string {
	switch dbType {
	case engine.MySQL:
		return config.MySQL.DatabaseName
	case engine.PostgreSQL:
		return config.PostgreSQL.DatabaseName
	default:
		return ""
	}
}

func getMySQLCharset(config *engine.Config) string {
	if config.MySQL.Charset != "" {
		return config.MySQL.Charset
	}
	return "utf8mb4"
}
