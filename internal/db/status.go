package db

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

// StatusCommand implements `gorun db status` - checking connectivity for
// one engine (if given as an arg) or every configured engine.
type StatusCommand struct {
	config *engine.Config
}

// NewStatusCommand builds a StatusCommand and prints its banner.
func NewStatusCommand(config *engine.Config) *StatusCommand {
	engine.PrintBoldCard("DATABASE COMMANDS STATUS")
	return &StatusCommand{config: config}
}

func (c *StatusCommand) Handle(ctx context.Context, cmd *cli.Command) error {
	executor := NewStatusExecutor(c.config)

	args := make([]string, 0)
	if cmd.Args().Len() > 0 {
		args = append(args, cmd.Args().First())
	}

	return executor.Execute(args)
}

// StatusExecutor does StatusCommand's actual connection checks, caching
// results per engine within one run.
type StatusExecutor struct {
	config    *engine.Config
	dbManager *engine.DatabaseManager
	cache     map[engine.DatabaseType]*DatabaseStatus
}

// NewStatusExecutor builds a StatusExecutor backed by config.
func NewStatusExecutor(config *engine.Config) *StatusExecutor {
	return &StatusExecutor{
		config:    config,
		dbManager: engine.NewDatabaseManager(config),
		cache:     make(map[engine.DatabaseType]*DatabaseStatus),
	}
}

// DatabaseStatus is one engine's connectivity check result, ready to
// print as a table.
type DatabaseStatus struct {
	Type        engine.DatabaseType
	Host        string
	Port        string
	User        string
	Database    string
	Version     string
	ServerTime  string
	CurrentUser string
	Hostname    string
	Collation   string
	Timezone    string
	Connected   bool
	Error       error
}

// Execute checks the engine named in args[0], or every configured engine
// if args is empty.
func (se *StatusExecutor) Execute(args []string) error {
	engine.PrintBoldCard("DATABASE STATUS CHECK")

	if len(args) == 0 {
		return se.checkAllDatabases()
	}

	dbType := engine.DatabaseType(args[0])
	if err := se.dbManager.ValidateDatabaseType(dbType); err != nil {
		engine.PrintError("Invalid database type: %s", args[0])
		engine.PrintInfo("Available types: mysql, postgresql")
		return err
	}

	return se.checkSingleDatabase(dbType)
}

func (se *StatusExecutor) checkAllDatabases() error {
	engine.PrintInfo("Checking status for all configured databases...")
	fmt.Println()

	for _, dbInfo := range se.dbManager.GetAvailableDatabases() {
		if err := se.checkSingleDatabase(dbInfo.Type); err != nil {
			return err
		}
		fmt.Println()
	}

	return nil
}

func (se *StatusExecutor) checkSingleDatabase(dbType engine.DatabaseType) error {
	engine.PrintInfo("Connecting to %s database...", dbType)

	status := se.getDatabaseStatus(dbType)
	if !status.Connected {
		engine.PrintError("Unable to connect to %s server", dbType)
		if status.Error != nil {
			engine.PrintError("Error: %v", status.Error)
		}
		return status.Error
	}

	se.printStatusTable(status)
	engine.PrintSuccess("Connected successfully to %s server at '%s:%s'",
		dbType, status.Host, status.Port)

	return nil
}

func (se *StatusExecutor) getDatabaseStatus(dbType engine.DatabaseType) *DatabaseStatus {
	if cached, exists := se.cache[dbType]; exists {
		return cached
	}

	status := &DatabaseStatus{
		Type:      dbType,
		Host:      se.dbManager.GetConfiguredHost(dbType),
		Port:      se.dbManager.GetConfiguredPort(dbType),
		User:      se.dbManager.GetConfiguredUser(dbType),
		Database:  se.dbManager.GetConfiguredDatabase(dbType),
		Hostname:  getSystemHostname(),
		Connected: false,
	}

	gormDB, sqlDB, err := se.dbManager.InitializeDatabase(dbType, se.config)
	if err != nil {
		status.Error = err
		se.cache[dbType] = status
		return status
	}
	defer func() { _ = sqlDB.Close() }()

	if err := sqlDB.Ping(); err != nil {
		status.Error = err
		se.cache[dbType] = status
		return status
	}

	status.Connected = true
	se.populateDatabaseInfo(gormDB, status)
	se.cache[dbType] = status

	return status
}

func (se *StatusExecutor) populateDatabaseInfo(gormDB *gorm.DB, status *DatabaseStatus) {
	queryMap := se.getDatabaseQueries(status.Type)

	if version, err := se.querySingleValue(gormDB, queryMap["version"]); err == nil {
		status.Version = se.formatVersion(status.Type, version)
	}

	if serverTime, err := se.queryTimeValue(gormDB, queryMap["serverTime"]); err == nil {
		status.ServerTime = serverTime.Format("2006-01-02 15:04:05")
	}

	if currentUser, err := se.querySingleValue(gormDB, queryMap["currentUser"]); err == nil {
		status.CurrentUser = currentUser
	}

	if collation, err := se.querySingleValue(gormDB, queryMap["collation"]); err == nil {
		status.Collation = collation
	}

	if timezone, err := se.querySingleValue(gormDB, queryMap["timezone"]); err == nil {
		status.Timezone = timezone
	}
}

func (se *StatusExecutor) getDatabaseQueries(dbType engine.DatabaseType) map[string]string {
	return map[string]string{
		"version":     se.getVersionQuery(dbType),
		"serverTime":  se.getServerTimeQuery(dbType),
		"currentUser": se.getCurrentUserQuery(dbType),
		"collation":   se.getCollationQuery(dbType),
		"timezone":    se.getTimezoneQuery(dbType),
	}
}

func (se *StatusExecutor) getVersionQuery(dbType engine.DatabaseType) string {
	if dbType == engine.MySQL {
		return "SELECT VERSION()"
	}
	return "SELECT version()"
}

func (se *StatusExecutor) getServerTimeQuery(dbType engine.DatabaseType) string {
	if dbType == engine.MySQL {
		return "SELECT NOW()"
	}
	return "SELECT now()"
}

func (se *StatusExecutor) getCurrentUserQuery(dbType engine.DatabaseType) string {
	if dbType == engine.MySQL {
		return "SELECT USER()"
	}
	return "SELECT current_user"
}

func (se *StatusExecutor) getCollationQuery(dbType engine.DatabaseType) string {
	if dbType == engine.MySQL {
		return "SELECT @@collation_database"
	}
	return "SHOW LC_COLLATE"
}

func (se *StatusExecutor) getTimezoneQuery(dbType engine.DatabaseType) string {
	if dbType == engine.MySQL {
		return "SELECT @@time_zone"
	}
	return "SHOW TIMEZONE"
}

func (se *StatusExecutor) querySingleValue(gormDB *gorm.DB, query string) (string, error) {
	var result string
	err := gormDB.Raw(query).Scan(&result).Error
	return result, err
}

func (se *StatusExecutor) queryTimeValue(gormDB *gorm.DB, query string) (time.Time, error) {
	var result time.Time
	err := gormDB.Raw(query).Scan(&result).Error
	return result, err
}

func (se *StatusExecutor) formatVersion(dbType engine.DatabaseType, version string) string {
	if dbType == engine.PostgreSQL {
		re := regexp.MustCompile(`PostgreSQL \d+\.\d+`)
		return re.FindString(version)
	}
	return version
}

func (se *StatusExecutor) printStatusTable(status *DatabaseStatus) {
	engine.PrintSectionHeader(fmt.Sprintf("%s DATABASE STATUS", strings.ToUpper(string(status.Type))))

	table := engine.NewTable([]string{"Key", "Value"})

	data := []struct {
		Key   string
		Value string
	}{
		{"Type", string(status.Type)},
		{"Host", status.Host},
		{"Port", status.Port},
		{"User", status.User},
		{"Database", status.Database},
		{"Current User", status.CurrentUser},
		{"Server Time", status.ServerTime},
		{"Version", status.Version},
		{"Hostname", status.Hostname},
		{"Collation", status.Collation},
		{"Timezone", status.Timezone},
	}

	for _, item := range data {
		table.AddRow([]string{item.Key, item.Value})
	}

	table.DrawVertical()
}

func getSystemHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}
