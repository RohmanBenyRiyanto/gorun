package setup

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/RohmanBenyRiyanto/gorun/internal/engine"
)

// runWizard collects answers interactively, reusing the same prompt
// helpers the rest of gorun's interactive commands use (engine.PromptInput,
// engine.ConfirmPrompt) rather than inventing new ones.
func runWizard() answers {
	engine.PrintBoldCard("GORUN SETUP")

	var a answers
	a.Name = promptDefault("Project name", "", defaultProjectName())
	a.Usage = engine.PromptInput("Usage banner", "optional, shown in gorun --help")
	a.AppEnv = promptDefault("Environment", "gates seed run's production guard", "local")

	if engine.ConfirmPrompt("Configure MySQL?") {
		a.MySQL = wizardDB("mysql", "3306")
	}
	if engine.ConfirmPrompt("Configure PostgreSQL?") {
		a.PostgreSQL = wizardDB("postgresql", "5432")
	}

	switch {
	case !a.MySQL.Configure && !a.PostgreSQL.Configure:
		engine.PrintWarning("No database engine configured - db/migrate/seed/table commands won't have anything to connect to until you edit .gorun/config.yaml by hand.")
	case a.MySQL.Configure && a.PostgreSQL.Configure:
		fmt.Println()
		a.MultiDB = engine.ConfirmPrompt("Both engines are configured - let commands choose between them at runtime (multi_db)?")
		if !a.MultiDB {
			engine.PrintWarning("multi_db left off - db/migrate/seed/table commands will refuse to run until you either set multi_db: true or remove one engine from .gorun/config.yaml.")
		}
	}

	return a
}

func wizardDB(engineName, defaultPort string) dbAnswers {
	fmt.Println()
	engine.PrintInfo("%s connection", engineName)

	host := promptDefault("Host", "", "127.0.0.1")
	port := promptDefault("Port", "", defaultPort)
	user := promptDefault("User", "", "root")
	password := promptDefault("Password", "prefer an env reference like ${DB_PASSWORD} over a literal value", "${DB_PASSWORD}")
	database := promptDefault("Database name", "", defaultProjectName())
	// gorun appends the engine name itself when resolving these at
	// runtime (MigrationManager/SeederManager.Get*Path) - "database/migrations"
	// here becomes "database/migrations/mysql", not
	// "database/migrations/mysql/mysql".
	migrationPath := promptDefault("Migration path", "engine name appended automatically", "database/migrations")
	seederPath := promptDefault("Seeder path", "engine name appended automatically", "database/seeders")

	return dbAnswers{
		Configure:     true,
		Host:          host,
		Port:          port,
		User:          user,
		Password:      password,
		DatabaseName:  database,
		MigrationPath: migrationPath,
		SeederPath:    seederPath,
	}
}

// promptDefault is engine.PromptInput with a fallback value when the user
// just presses enter.
func promptDefault(label, hint, def string) string {
	if def != "" {
		if hint != "" {
			hint = hint + ", default: " + def
		} else {
			hint = "default: " + def
		}
	}
	if v := engine.PromptInput(label, hint); v != "" {
		return v
	}
	return def
}

// defaultProjectName suggests the current directory's base name, the
// same convention `npm init`/`go mod init` default to.
func defaultProjectName() string {
	dir, err := os.Getwd()
	if err != nil {
		return "myapp"
	}
	name := filepath.Base(dir)
	if name == "" || name == "." || name == "/" {
		return "myapp"
	}
	return name
}
