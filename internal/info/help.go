package info

import (
	"fmt"
	"strings"

	"github.com/RohmanBenyRiyanto/gorun/internal/engine"
	"github.com/RohmanBenyRiyanto/gorun/internal/version"
)

// InfoHelper prints the top-level help/version/project-info/commands
// banners shown by `gorun help`, `gorun version`, `gorun info`, and
// `gorun commands` (and their root-level -h/-v/-i/-c flag equivalents).
type InfoHelper struct{}

// commandTableHeaders is the pipe-delimited header row shared by every
// per-group command table this file renders.
const commandTableHeaders = "Command|Description|Key Options"

// NewInfoHelper builds an InfoHelper.
func NewInfoHelper() *InfoHelper {
	return &InfoHelper{}
}

// ShowVersion prints the toolkit version banner.
func (h *InfoHelper) ShowVersion() {
	engine.PrintBoldCard("GORUN CLI TOOLKIT - Version Information")

	versionInfo := []string{
		fmt.Sprintf("Toolkit Version%s  : %s%s",
			engine.ColorReset,
			engine.ColorGreen,
			version.Get()),
		fmt.Sprintf("Database Drivers%s : %s%s",
			engine.ColorReset,
			engine.ColorCyan,
			"MySQL, PostgreSQL"),
		fmt.Sprintf("Go Version%s       : %s%s",
			engine.ColorReset,
			engine.ColorYellow,
			"1.21+"),
		fmt.Sprintf("Build Status%s     : %s%s",
			engine.ColorReset,
			engine.ColorGreen,
			"Stable"),
	}

	for _, info := range versionInfo {
		fmt.Printf("  %s%s\n", engine.ColorFunction, info)
	}
	fmt.Println()

	engine.PrintDivider()
	engine.PrintNormal("• Latest stable release with full feature support")
	engine.PrintNormal("• Compatible with Go modules and modern Go practices")
	engine.PrintNormal("• Supports both MySQL and PostgreSQL databases")
	engine.PrintNormal("• Regular updates and security patches included")
	fmt.Println()
}

// ShowHelp prints the full top-level help screen: usage, global options,
// basic examples, command categories, and advanced usage patterns.
func (h *InfoHelper) ShowHelp() {
	engine.PrintBoldCard("GORUN CLI TOOLKIT - Command Help")

	engine.PrintTextH1("USAGE:")
	fmt.Printf(
		"  %s%s %s%s %s%s%s\n",
		engine.ColorKeyword, "gorun",
		engine.ColorOrange, "<command>",
		engine.ColorComment, "[options]",
		engine.ColorReset,
	)

	engine.PrintDivider()
	h.ShowGlobalOptions()
	h.ShowBasicExamples()
	h.ShowCommandCategories()
	h.ShowAdvancedUsage()
}

// ShowGlobalOptions prints the table of root-level flags (--help,
// --version, --commands, --info, plus a few reserved-but-not-yet-wired-up
// ones like --env/--debug/--config/--quiet).
func (h *InfoHelper) ShowGlobalOptions() {
	engine.PrintTextH1("Global Options")

	headers := "Option|Alias|Description"
	rows := ""
	options := []struct {
		Option      string
		Alias       string
		Description string
	}{
		{"--help", "-h", "Show help message for any command"},
		{"--version", "-v", "Display version and build information"},
		{"--commands", "-c", "List all available commands with descriptions"},
		{"--info", "-i", "Show detailed project information"},
		{"--env", "", "Set environment (dev/stage/prod)"},
		{"--debug", "", "Enable debug mode with verbose output"},
		{"--config", "", "Specify custom configuration file path"},
		{"--quiet", "-q", "Suppress non-essential output"},
	}

	for _, opt := range options {
		option := engine.Keyword(opt.Option)
		alias := ""
		if opt.Alias != "" {
			alias = engine.Keyword(opt.Alias)
		}
		desc := engine.Comment(opt.Description)

		rows += fmt.Sprintf("%s|%s|%s\n", option, alias, desc)
	}

	table := engine.ParseTable(headers, rows)
	table.SetColumnConfig(0, engine.ColumnConfig{
		HeaderAlign:  engine.AlignLeft,
		ContentAlign: engine.AlignLeft,
		MinWidth:     12,
		MaxWidth:     15,
	})
	table.SetColumnConfig(1, engine.ColumnConfig{
		HeaderAlign:  engine.AlignCenter,
		ContentAlign: engine.AlignCenter,
		MinWidth:     6,
		MaxWidth:     8,
	})
	table.DrawHorizontal()
	fmt.Println()
}

// ShowBasicExamples prints a handful of getting-started example
// invocations.
func (h *InfoHelper) ShowBasicExamples() {
	engine.PrintTextH1("Basic Usage Examples")

	engine.PrintInfo("Getting Started:")
	examples := []string{
		"gorun help|Show comprehensive help information",
		"gorun version|Display current version and build details",
		"gorun commands|List all available commands by category",
		"gorun info|Show project configuration and environment",
		"gorun --env=dev info|Show project info for development environment",
	}
	engine.PrintKeyValueTable(examples, engine.ColorComment, engine.ColorGray)

	fmt.Println()
	engine.PrintInfo("Command Structure:")
	structureExamples := []string{
		"gorun <category> <action>|Standard command format",
		"gorun db create|Database category, create action",
		"gorun migrate run|Migration category, run action",
		"gorun app build --output=myapp|With options and flags",
	}
	engine.PrintKeyValueTable(structureExamples, engine.ColorComment, engine.ColorGray)
	fmt.Println()
}

// ShowCommandCategories prints a table of the five command groups (db,
// table, migrate, seed, app) with their subcommands.
func (h *InfoHelper) ShowCommandCategories() {
	engine.PrintTextH1("Command Categories")

	headers := "Category|Commands|Description"
	rows := ""
	categories := []struct {
		Category    string
		Commands    string
		Description string
	}{
		{"db", "create, drop, list, status, truncate", "Database management and operations"},
		{"table", "create, drop, list, truncate", "Table operations and schema management"},
		{"migrate", "run, status, make, rollback, reset, refresh, fresh", "Database schema versioning and migrations"},
		{"seed", "run, make, list", "Database seeding and test data management"},
		{"app", "build, serve, test, clean, install, version, status", "Application lifecycle and deployment"},
	}

	for _, cat := range categories {
		category := engine.Orange(cat.Category)
		commands := engine.LightGray(cat.Commands)
		desc := engine.Comment(cat.Description)

		rows += fmt.Sprintf("%s|%s|%s\n", category, commands, desc)
	}

	table := engine.ParseTable(headers, rows)
	table.SetColumnConfig(0, engine.ColumnConfig{
		HeaderAlign:  engine.AlignLeft,
		ContentAlign: engine.AlignLeft,
		MinWidth:     10,
		MaxWidth:     12,
	})
	table.SetColumnConfig(1, engine.ColumnConfig{
		HeaderAlign:  engine.AlignLeft,
		ContentAlign: engine.AlignLeft,
		MinWidth:     35,
		MaxWidth:     40,
	})
	table.DrawHorizontal()

	fmt.Println()
	engine.PrintInfo("Get detailed help for any category:")
	categoryHelp := []string{
		"gorun db help|Show all database commands with examples",
		"gorun migrate help|Show migration commands and workflows",
		"gorun app help|Show application management commands",
	}
	engine.PrintKeyValueTable(categoryHelp, engine.ColorComment, engine.ColorGray)
	fmt.Println()
}

// ShowAdvancedUsage prints example patterns for environment-specific runs,
// chaining commands, and production-safety checks.
func (h *InfoHelper) ShowAdvancedUsage() {
	engine.PrintTextH1("Advanced Usage Patterns")

	engine.PrintInfo("Environment-specific Operations:")
	envExamples := []string{
		"gorun --env=prod migrate run|Run migrations in production",
		"gorun --env=dev seed run|Seed development database",
		"gorun --debug db status|Debug database connection issues",
	}
	engine.PrintKeyValueTable(envExamples, engine.ColorComment, engine.ColorGray)

	fmt.Println()
	engine.PrintInfo("Chaining Operations:")
	chainExamples := []string{
		"gorun db create && gorun migrate run|Create database then run migrations",
		"gorun migrate fresh --seed|Drop all tables, run migrations, then seed",
		"gorun app clean && gorun app build|Clean then rebuild application",
	}
	engine.PrintKeyValueTable(chainExamples, engine.ColorComment, engine.ColorGray)

	fmt.Println()
	engine.PrintInfo("Production Safety:")
	safetyExamples := []string{
		"gorun --env=prod migrate status|Check migration status before deploy",
		"gorun db create --charset=utf8mb4|Specify charset for international support",
		"gorun migrate run --pretend|Preview SQL before execution",
	}
	engine.PrintKeyValueTable(safetyExamples, engine.ColorComment, engine.ColorGray)
	fmt.Println()
}

// ShowProjectInfo prints the project name/root/environment (read from
// go.mod and .env in the current working directory) plus a feature
// summary - `gorun info` and the default help screen shown by
// cli.HelpPrinter.
func (h *InfoHelper) ShowProjectInfo() {
	engine.PrintBoldCard("GORUN CLI TOOLKIT - Project Information")

	root, _ := engine.GetProjectRoot()
	name, _ := engine.GetGoModuleName()
	env := engine.GetAppEnv()

	envColor := engine.ColorYellow
	envStatus := "Development"
	switch env {
	case "production", "prod":
		envColor = engine.ColorGreen
		envStatus = "Production"
	case "staging", "stage":
		envColor = engine.ColorPurple
		envStatus = "Staging"
	case "development", "dev":
		envColor = engine.ColorCyan
		envStatus = "Development"
	case "testing", "test":
		envColor = engine.ColorBlue
		envStatus = "Testing"
	case "local", "localhost":
		envColor = engine.ColorSoftPurple
		envStatus = "Localhost"
	}

	engine.PrintTextH1("Project Details")
	infoItems := []string{
		fmt.Sprintf("Project Name|%s%s%s", engine.ColorCyan, name, engine.ColorReset),
		fmt.Sprintf("Root Directory|%s%s%s", engine.ColorForeground, root, engine.ColorReset),
		fmt.Sprintf("Environment|%s%s%s", envColor, envStatus, engine.ColorReset),
		fmt.Sprintf("Database Support|%sMySQL, PostgreSQL%s", engine.ColorGreen, engine.ColorReset),
		fmt.Sprintf("CLI Version|%s%s%s", engine.ColorGreen, version.Get(), engine.ColorReset),
	}

	table := engine.NewTable([]string{"Property", "Value"})
	for _, item := range infoItems {
		parts := strings.SplitN(item, "|", 2)
		table.AddRow([]string{
			fmt.Sprintf("%s%s%s", engine.ColorFunction, parts[0], engine.ColorReset),
			parts[1],
		})
	}

	table.SetColumnConfig(0, engine.ColumnConfig{
		HeaderAlign:  engine.AlignLeft,
		ContentAlign: engine.AlignLeft,
		MinWidth:     18,
	})

	table.SetColumnConfig(1, engine.ColumnConfig{
		HeaderAlign:  engine.AlignLeft,
		ContentAlign: engine.AlignLeft,
		MinWidth:     40,
	})

	table.DrawHorizontal()

	engine.PrintDivider()
	engine.PrintTextH1("Environment Configuration")

	fmt.Printf("Current environment is set to %s%s%s mode.\n", envColor, envStatus, engine.ColorReset)
	fmt.Println()

	switch envStatus {
	case "Production":
		engine.PrintWarning("Production Environment Active")
		engine.PrintNormal("• Extra confirmation required for destructive operations")
		engine.PrintNormal("• Debug mode disabled by default")
		engine.PrintNormal("• security measures enabled")
	case "Staging":
		engine.PrintInfo("Staging Environment Active")
		engine.PrintNormal("• Similar to production with additional logging")
		engine.PrintNormal("• Safe for integration testing")
		engine.PrintNormal("• Performance monitoring enabled")
	case "Development":
		engine.PrintInfo("Development Environment Active")
		engine.PrintNormal("• Debug mode enabled by default")
		engine.PrintNormal("• security measures disabled")
		engine.PrintNormal("• Performance monitoring disabled")
	case "Testing":
		engine.PrintInfo("Testing Environment Active")
		engine.PrintNormal("• Debug mode enabled by default")
		engine.PrintNormal("• security measures disabled")
		engine.PrintNormal("• Performance monitoring disabled")
	case "Localhost":
		engine.PrintInfo("Localhost Environment Active")
		engine.PrintNormal("• Debug mode enabled by default")
		engine.PrintNormal("• security measures disabled")
		engine.PrintNormal("• Performance monitoring disabled")
	default:
		engine.PrintInfo("Unknown Environment Active")
		engine.PrintNormal("• Debug mode enabled by default")
		engine.PrintNormal("• security measures disabled")
		engine.PrintNormal("• Performance monitoring disabled")
	}

	engine.PrintDivider()
	engine.PrintTextH1("Available Features")

	features := []string{
		"✓ Database Management|Create, drop, and manage databases",
		"✓ Table Operations|Full table lifecycle management",
		"✓ Schema Migrations|Version-controlled database changes",
		"✓ Data Seeding|Test and initial data population",
		"✓ Application Tools|Build, test, and deployment utilities",
		"✓ Multi-DB Support|MySQL and PostgreSQL compatibility",
		"✓ Environment Control|Development, staging, production modes",
		"✓ Safety Features|Confirmation prompts and rollback capabilities",
	}

	engine.PrintKeyValueTable(features, engine.ColorGreen, engine.ColorReset)
	fmt.Println()
}

// ListCommands prints every command grouped by category (`gorun commands`
// / `gorun -c`), each with its key options.
func (h *InfoHelper) ListCommands() {
	engine.PrintBoldCard("GORUN CLI TOOLKIT - Available Commands")

	h.showDatabaseCommands()
	h.showTableCommands()
	h.showMigrationCommands()
	h.showSeedCommands()
	h.showApplicationCommands()
	h.showGlobalCommands()
}

func (h *InfoHelper) showDatabaseCommands() {
	engine.PrintSectionHeader("DATABASE COMMANDS")
	engine.PrintNormal("Manage database creation, deletion, and maintenance operations.")
	fmt.Println()

	headers := commandTableHeaders
	rows := ""
	dbCommands := []struct {
		Command     string
		Description string
		Options     string
	}{
		{"db create", "Create a new database with specified settings", "--name, --type, --charset, --collation, --encoding"},
		{"db drop", "Permanently delete an existing database", "--name, --type, --force"},
		{"db list", "Display all available databases", "--type"},
		{"db status", "Check database connectivity and version", "(none)"},
		{"db truncate", "Remove all data from database tables (whole DB, FK-aware order)", "--name (no --type/--force - always prompts for engine)"},
	}

	for _, cmd := range dbCommands {
		command := engine.Orange(cmd.Command)
		desc := engine.Comment(cmd.Description)
		opts := engine.LightGray(cmd.Options)
		rows += fmt.Sprintf("%s|%s|%s\n", command, desc, opts)
	}

	table := engine.ParseTable(headers, rows)
	h.configureCommandTable(table)
	table.DrawHorizontal()

	engine.PrintInfo("Common Database Options:")
	dbOptions := []string{
		"-n, --name|Database name to operate on",
		"-t, --type|Database type (mysql/postgresql) - not available on db truncate",
		"-f, --force|Skip confirmation prompt - not available on db truncate/list/status",
		"--charset|Character set (create, MySQL only)",
		"--collation|Collation (create)",
		"--encoding|Encoding (create, PostgreSQL only)",
	}
	engine.PrintKeyValueTable(dbOptions, engine.ColorLightGray, engine.ColorComment)
	fmt.Println()
}

func (h *InfoHelper) showTableCommands() {
	engine.PrintSectionHeader("TABLE COMMANDS")
	engine.PrintNormal("Handle table creation, modification, and maintenance tasks.")
	fmt.Println()

	headers := commandTableHeaders
	rows := ""
	tableCommands := []struct {
		Command     string
		Description string
		Options     string
	}{
		{"table create", "Create new table with schema definition", "--name, --schema, --database, --type, --force"},
		{"table drop", "Remove existing table and all data", "--name, --database, --type, --force"},
		{"table list", "Show all tables in specified database", "--type"},
		{"table truncate", "Clear all data in ONE table while keeping structure", "--name, --database, --type, --force"},
	}

	for _, cmd := range tableCommands {
		command := engine.Orange(cmd.Command)
		desc := engine.Comment(cmd.Description)
		opts := engine.LightGray(cmd.Options)
		rows += fmt.Sprintf("%s|%s|%s\n", command, desc, opts)
	}

	table := engine.ParseTable(headers, rows)
	h.configureCommandTable(table)
	table.DrawHorizontal()

	engine.PrintInfo("Common Table Options:")
	tableOptions := []string{
		"-n, --name|Table name to operate on",
		"-d, --database|Target database name",
		"-t, --type|Database type (mysql/postgresql) - skips the engine prompt",
		"-f, --force|Skip confirmation prompt (create: overwrite if exists)",
		"--schema|Path to table schema definition file (create only)",
	}
	engine.PrintKeyValueTable(tableOptions, engine.ColorLightGray, engine.ColorComment)
	fmt.Println()
}

func (h *InfoHelper) showMigrationCommands() {
	engine.PrintSectionHeader("MIGRATION COMMANDS")
	engine.PrintNormal("Version control for database schema changes and rollbacks.")
	fmt.Println()

	headers := commandTableHeaders
	rows := ""
	migrationCommands := []struct {
		Command     string
		Description string
		Options     string
	}{
		{"migrate run", "Execute pending database migrations", "--force, --path, --file, --pretend, --step, --type, --database"},
		{"migrate status", "Show current migration state", "--type, --database"},
		{"migrate make", "Generate new migration file", "--create, --table, --path, --realpath, --fullpath, --type"},
		{"migrate rollback", "Undo the last migration batch", "--step, --path, --force, --type, --database"},
		{"migrate reset", "Rollback all applied migrations", "--force, --type, --database"},
		{"migrate refresh", "Reset and re-run all migrations", "--seed, --step, --force, --file, --type, --database"},
		{"migrate fresh", "Drop all tables and re-run migrations", "--seed, --drop-views, --drop-types, --force, --file, --type, --database"},
	}

	for _, cmd := range migrationCommands {
		command := engine.Orange(cmd.Command)
		desc := engine.Comment(cmd.Description)
		opts := engine.LightGray(cmd.Options)
		rows += fmt.Sprintf("%s|%s|%s\n", command, desc, opts)
	}

	table := engine.ParseTable(headers, rows)
	h.configureCommandTable(table)
	table.DrawHorizontal()

	engine.PrintInfo("Common Migration Options:")
	migrationOptions := []string{
		"-p, --path|Custom migrations directory path",
		"-s, --step|Number of migration batches to process (int on rollback/refresh; bool 'run as steps' on run)",
		"-F, --file|Run/refresh/fresh only one specific migration (name without extension)",
		"--pretend|Show SQL without executing changes (run only)",
		"--seed|Run seeders after migration completion (refresh/fresh only)",
		"--drop-views|Drop all views first (fresh only)",
		"--drop-types|Drop custom types, PostgreSQL only (fresh only)",
		"-c, --create|Create new table migration template (make only)",
		"-t, --table|Modify existing table migration template (make only)",
		"--type, --database|Skip the engine/database prompts - needed for non-interactive/CI runs",
	}
	engine.PrintKeyValueTable(migrationOptions, engine.ColorLightGray, engine.ColorComment)
	fmt.Println()
}

func (h *InfoHelper) showSeedCommands() {
	engine.PrintSectionHeader("SEED COMMANDS")
	engine.PrintNormal("Populate databases with test data and initial records.")
	fmt.Println()

	headers := commandTableHeaders
	rows := ""
	seedCommands := []struct {
		Command     string
		Description string
		Options     string
	}{
		{"seed run", "Execute database seeders", "--class, --type, --database, --only"},
		{"seed make", "Create new seeder class file", "--path, --model, --table"},
		{"seed list", "Display available seeder classes", "--details, --path"},
	}

	for _, cmd := range seedCommands {
		command := engine.Orange(cmd.Command)
		desc := engine.Comment(cmd.Description)
		opts := engine.LightGray(cmd.Options)
		rows += fmt.Sprintf("%s|%s|%s\n", command, desc, opts)
	}

	table := engine.ParseTable(headers, rows)
	h.configureCommandTable(table)
	table.DrawHorizontal()

	engine.PrintInfo("Common Seed Options:")
	seedOptions := []string{
		"-c, --class|Run specific seeder class only",
		"--type, --database|Skip the engine/database prompts - needed for non-interactive/CI runs",
		"--only|Run only specified seeders (comma-separated)",
		"--except|Exclude specified seeders from execution",
		"--model|Generate model with seeder (make only)",
		"--transaction|Run in database transaction (default: true)",
		"--stop-on-error|Stop on first failing seeder (default: true)",
		"--force|Required to run against production (app.env=prod/production)",
	}
	engine.PrintKeyValueTable(seedOptions, engine.ColorLightGray, engine.ColorComment)
	fmt.Println()
}

func (h *InfoHelper) showApplicationCommands() {
	engine.PrintSectionHeader("APPLICATION COMMANDS")
	engine.PrintNormal("Build, test, deploy, and manage application lifecycle.")
	fmt.Println()

	headers := commandTableHeaders
	rows := ""
	appCommands := []struct {
		Command     string
		Description string
		Options     string
	}{
		{"app build", "Compile application binary", "--output, --os, --arch, --race, --docker, --ldflags, --tags, --verbose"},
		{"app serve", "Start development server", "--port, --host, --dev, --watch, --env"},
		{"app test", "Run application test suite", "--package, --verbose, --race, --coverage, --coverprofile, --timeout, --short"},
		{"app clean", "Remove build artifacts", "--cache, --modules, --logs, --all, --force"},
		{"app install", "Install project dependencies", "--update, --tidy, --vendor, --verify"},
		{"app status", "Check application health", "--detailed, --json, --health"},
		{"app version", "Show version information", "--json, --short"},
	}

	for _, cmd := range appCommands {
		command := engine.Orange(cmd.Command)
		desc := engine.Comment(cmd.Description)
		opts := engine.LightGray(cmd.Options)
		rows += fmt.Sprintf("%s|%s|%s\n", command, desc, opts)
	}

	table := engine.ParseTable(headers, rows)
	h.configureCommandTable(table)
	table.DrawHorizontal()

	engine.PrintInfo("Common Application Options:")
	appOptions := []string{
		"-o, --output|Output binary filename, default \"app\" (build)",
		"-p, --port|Server port number (serve)",
		"--host|Bind to specific host address, default \"localhost\" (serve)",
		"--dev|Enable development mode (serve)",
		"--watch|Enable hot reload via air, needs .air.toml (serve)",
		"--race|Enable Go race detector (build, test)",
		"--coverage|Generate test coverage reports (test)",
		"--force, -f|Skip confirmation prompt (clean)",
	}
	engine.PrintKeyValueTable(appOptions, engine.ColorLightGray, engine.ColorComment)
	fmt.Println()
}

func (h *InfoHelper) showGlobalCommands() {
	engine.PrintSectionHeader("GLOBAL COMMANDS")
	engine.PrintNormal("General toolkit information and utility commands.")
	fmt.Println()

	headers := commandTableHeaders
	rows := ""
	globalCommands := []struct {
		Command     string
		Description string
		Options     string
	}{
		{"help (h)", "Show comprehensive help information", "(none - takes no flags)"},
		{"version (ver)", "Display toolkit version details", "(none - see 'app version' for --json/--short)"},
		{"commands (list, ls)", "List all available commands (this panel)", "(none - same as root -c/--commands)"},
		{"info (project, status)", "Show project and environment info", "(none - same as root -i/--info)"},
	}

	for _, cmd := range globalCommands {
		command := engine.Orange(cmd.Command)
		desc := engine.Comment(cmd.Description)
		opts := engine.LightGray(cmd.Options)
		rows += fmt.Sprintf("%s|%s|%s\n", command, desc, opts)
	}

	table := engine.ParseTable(headers, rows)
	h.configureCommandTable(table)
	table.DrawHorizontal()

	engine.PrintDivider()
	engine.PrintInfo("Root-level flags (before any command, e.g. `gorun -c`):")
	rootFlags := []string{
		"-c, --commands|List all available commands (this panel)",
		"-i, --info|Show project and environment info",
		"-v, --version|Show toolkit version",
	}
	engine.PrintKeyValueTable(rootFlags, engine.ColorLightGray, engine.ColorComment)
	fmt.Println()

	engine.PrintDivider()
	engine.PrintWarning("Important Usage Notes")
	fmt.Println()
	engine.PrintNormal("• Always backup databases before destructive operations")
	engine.PrintNormal("• Use --pretend to preview SQL before executing a migration")
	engine.PrintNormal("• seed run --force is specifically required to seed app.env=prod/production")
	engine.PrintNormal("• Check migration status before deploying to production")
	engine.PrintNormal("• db truncate always prompts for the DB engine - no --type flag exists for it")
	fmt.Println()

	engine.PrintInfo("For detailed help on any command category, use:")
	categoryHelpExamples := []string{
		"gorun <category> help|Show detailed help for specific category",
		"gorun db help|Complete database commands documentation",
		"gorun migrate help|Migration workflow and examples",
	}
	engine.PrintKeyValueTable(categoryHelpExamples, engine.ColorComment, engine.ColorGray)
	fmt.Println()
}

func (h *InfoHelper) configureCommandTable(table *engine.Table) {
	table.SetColumnConfig(0, engine.ColumnConfig{
		HeaderAlign:  engine.AlignLeft,
		ContentAlign: engine.AlignLeft,
		MinWidth:     18,
		MaxWidth:     20,
	})

	table.SetColumnConfig(1, engine.ColumnConfig{
		HeaderAlign:  engine.AlignLeft,
		ContentAlign: engine.AlignLeft,
		MinWidth:     35,
		MaxWidth:     45,
	})

	table.SetColumnConfig(2, engine.ColumnConfig{
		HeaderAlign:  engine.AlignLeft,
		ContentAlign: engine.AlignLeft,
		MinWidth:     20,
		MaxWidth:     25,
	})
}
