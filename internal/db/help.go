package db

import (
	"fmt"

	"github.com/RohmanBenyRiyanto/gorun/internal/engine"
)

// DBHelper prints `gorun db help`'s detailed usage listing.
type DBHelper struct{}

// NewDBHelper builds a DBHelper.
func NewDBHelper() *DBHelper {
	return &DBHelper{}
}

// ShowHelp prints usage, an available-commands table, and detailed
// per-subcommand examples.
func (h *DBHelper) ShowHelp() {
	engine.PrintBoldCard("DATABASE COMMANDS HELP")

	engine.PrintTextH1("USAGE:")
	fmt.Printf(
		"  %s%s %s%s %s%s %s%s%s\n",
		engine.ColorKeyword, "gorun",
		engine.ColorOrange, "db",
		engine.ColorComment, "[command]",
		engine.ColorComment, "[options]",
		engine.ColorReset,
	)

	engine.PrintDivider()
	engine.PrintTextH1("Available Commands")

	headers := "No|Command|Description"
	rows := ""
	commands := []struct {
		Name        string
		Description string
	}{
		{"create", "Create a new database"},
		{"drop", "Drop an existing database"},
		{"list (ls)", "List all databases"},
		{"status", "Show database connection status"},
		{"truncate", "Truncate all tables in a database"},
		{"help", "Show this help message"},
	}

	for i, cmd := range commands {
		num := engine.LightGray(fmt.Sprintf("%d", i+1))
		name := engine.Orange(cmd.Name)
		desc := engine.Comment(cmd.Description)

		rows += fmt.Sprintf("%s|%s|%s\n", num, name, desc)
	}

	table := engine.ParseTable(headers, rows)
	table.SetColumnConfig(0, engine.ColumnConfig{
		HeaderAlign:  engine.AlignCenter,
		ContentAlign: engine.AlignCenter,
		MinWidth:     3,
		MaxWidth:     3,
	})
	table.DrawHorizontal()

	engine.PrintDivider()
	engine.PrintTextH1("Detailed Usage")

	createCmdStr := fmt.Sprintf(
		"%s%s %s%s %s%s %s%s %s%s%s",
		engine.ColorKeyword, "gorun",
		engine.ColorOrange, "db",
		engine.ColorOrange, "create",
		engine.ColorComment, "[name]",
		engine.ColorComment, "[options]",
		engine.ColorReset,
	)
	engine.PrintCodeBlock(createCmdStr, "Create a new database")
	fmt.Println()
	engine.PrintInfo("Options:")
	engine.PrintKeyValueTable([]string{
		"-n, --name|Specify the database name",
		"-t, --type|Specify database type (mysql/postgresql)",
		"--charset|Specify character set (MySQL only)",
		"--collation|Specify collation",
		"--encoding|Specify encoding (PostgreSQL only)",
	}, engine.ColorLightGray, engine.ColorComment)
	fmt.Println()
	engine.PrintInfo("Examples:")
	engine.PrintKeyValueTable([]string{
		"gorun db create my_db|Create database with interactive prompts",
		"gorun db create -n my_db -t mysql|Create MySQL database named 'my_db'",
		"gorun db create -n utf8_db --charset utf8mb4|Create with specific charset",
	}, engine.ColorLightGray, engine.ColorComment)

	dropCmdStr := fmt.Sprintf(
		"%s%s %s%s %s%s %s%s %s%s%s",
		engine.ColorKeyword, "gorun",
		engine.ColorOrange, "db",
		engine.ColorOrange, "drop",
		engine.ColorComment, "[name]",
		engine.ColorComment, "[options]",
		engine.ColorReset,
	)
	engine.PrintCodeBlock(dropCmdStr, "Drop an existing database")
	fmt.Println()
	engine.PrintInfo("Options:")
	engine.PrintKeyValueTable([]string{
		"-n, --name|Specify the database name",
		"-t, --type|Specify database type (mysql/postgresql)",
		"-f, --force|Skip confirmation prompt",
	}, engine.ColorLightGray, engine.ColorComment)
	engine.PrintWarning("This action cannot be undone!")
	fmt.Println()
	engine.PrintInfo("Examples:")
	engine.PrintKeyValueTable([]string{
		"gorun db drop my_db|Drop database with confirmation",
		"gorun db drop -n my_db -f|Force drop without confirmation",
	}, engine.ColorLightGray, engine.ColorComment)

	listCmdStr := fmt.Sprintf(
		"%s%s %s%s %s%s %s%s%s",
		engine.ColorKeyword, "gorun",
		engine.ColorOrange, "db",
		engine.ColorOrange, "list",
		engine.ColorComment, "[options]",
		engine.ColorReset,
	)
	engine.PrintCodeBlock(listCmdStr, "List all databases")
	fmt.Println()
	engine.PrintInfo("Options:")
	engine.PrintKeyValueTable([]string{
		"-t, --type|Specify database type (mysql/postgresql)",
	}, engine.ColorLightGray, engine.ColorComment)
	engine.PrintInfo("Aliases:")
	engine.PrintKeyValueTable([]string{
		"ls|Shortcut for list command",
	}, engine.ColorLightGray, engine.ColorComment)
	fmt.Println()
	engine.PrintInfo("Examples:")
	engine.PrintKeyValueTable([]string{
		"gorun db list|List all databases (interactive type selection)",
		"gorun db ls -t postgresql|List PostgreSQL databases",
	}, engine.ColorLightGray, engine.ColorComment)

	statusCmdStr := fmt.Sprintf(
		"%s%s %s%s %s%s%s",
		engine.ColorKeyword, "gorun",
		engine.ColorOrange, "db",
		engine.ColorOrange, "status",
		engine.ColorReset,
	)
	engine.PrintCodeBlock(statusCmdStr, "Show database connection status")
	engine.PrintInfo("Output includes:")
	engine.PrintKeyValueTable([]string{
		"Connection status|Whether connection is successful",
		"Database version|Version of the database server",
		"Connection details|Host, port, and database name",
	}, engine.ColorLightGray, engine.ColorComment)

	truncateCmdStr := fmt.Sprintf(
		"%s%s %s%s %s%s %s%s %s%s%s",
		engine.ColorKeyword, "gorun",
		engine.ColorOrange, "db",
		engine.ColorOrange, "truncate",
		engine.ColorComment, "[name]",
		engine.ColorComment, "[options]",
		engine.ColorReset,
	)
	engine.PrintCodeBlock(truncateCmdStr, "Truncate all tables in a database")
	fmt.Println()
	engine.PrintInfo("Options:")
	engine.PrintKeyValueTable([]string{
		"-n, --name|Specify the database name",
		"-f, --force|Skip confirmation prompt",
	}, engine.ColorLightGray, engine.ColorComment)
	fmt.Println()
	engine.PrintWarning("This will delete all data in all tables!")
	fmt.Println()
	engine.PrintInfo("Examples:")
	engine.PrintKeyValueTable([]string{
		"gorun db truncate my_db|Truncate tables with confirmation",
		"gorun db truncate -n my_db -f|Force truncate without confirmation",
	}, engine.ColorLightGray, engine.ColorComment)

	engine.PrintDivider()
	engine.PrintWarning("Important Notes")
	fmt.Println()
	engine.PrintNormal("• Always backup your database before destructive operations")
	engine.PrintNormal("• Database names are case-sensitive in some database systems")
	engine.PrintNormal("• Some operations may require special privileges")
	engine.PrintNormal("• The 'truncate' command will reset all auto-increment counters")
	engine.PrintNormal("• Use --force flag carefully as it skips all confirmation prompts")
	fmt.Println()
}
