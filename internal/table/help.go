package table

import (
	"fmt"

	"github.com/RohmanBenyRiyanto/gorun/internal/engine"
)

// TableHelper prints `gorun table help`'s detailed usage listing.
type TableHelper struct{}

// NewTableHelper builds a TableHelper.
func NewTableHelper() *TableHelper {
	return &TableHelper{}
}

// ShowHelp prints usage, an available-commands table, and detailed
// per-subcommand examples.
func (h *TableHelper) ShowHelp() {
	engine.PrintBoldCard("TABLE COMMANDS HELP")

	engine.PrintTextH1("USAGE:")
	fmt.Printf(
		"  %s%s %s%s %s%s %s%s%s\n",
		engine.ColorKeyword, "gorun",
		engine.ColorOrange, "table",
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
		{"create", "Create a new table"},
		{"drop", "Drop an existing table"},
		{"list (ls)", "List all tables in a database"},
		{"truncate", "Truncate a table"},
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
		engine.ColorOrange, "table",
		engine.ColorOrange, "create",
		engine.ColorComment, "[name]",
		engine.ColorComment, "[options]",
		engine.ColorReset,
	)
	engine.PrintCodeBlock(createCmdStr, "Create a new table")
	fmt.Println()
	engine.PrintInfo("Options:")
	engine.PrintKeyValueTable([]string{
		"-n, --name|Specify the table name (required)",
		"-d, --database|Specify the database name",
		"-t, --type|Specify database type (mysql/postgresql)",
		"--schema|Path to table schema file",
		"-f, --force|Overwrite if table exists",
	}, engine.ColorLightGray, engine.ColorComment)
	fmt.Println()
	engine.PrintInfo("Examples:")
	engine.PrintKeyValueTable([]string{
		"gorun table create -n users|Create table with interactive prompts",
		"gorun table create -n products -d mydb -t mysql|Create table in specific MySQL database",
		"gorun table create -n logs --schema schema.json|Create table using schema file",
	}, engine.ColorLightGray, engine.ColorComment)

	dropCmdStr := fmt.Sprintf(
		"%s%s %s%s %s%s %s%s %s%s%s",
		engine.ColorKeyword, "gorun",
		engine.ColorOrange, "table",
		engine.ColorOrange, "drop",
		engine.ColorComment, "[name]",
		engine.ColorComment, "[options]",
		engine.ColorReset,
	)
	engine.PrintCodeBlock(dropCmdStr, "Drop an existing table")
	fmt.Println()
	engine.PrintInfo("Options:")
	engine.PrintKeyValueTable([]string{
		"-n, --name|Specify the table name (required)",
		"-d, --database|Specify the database name",
		"-t, --type|Specify database type (mysql/postgresql)",
		"-f, --force|Skip confirmation prompt",
	}, engine.ColorLightGray, engine.ColorComment)
	fmt.Println()
	engine.PrintWarning("This action cannot be undone!")
	fmt.Println()
	engine.PrintInfo("Examples:")
	engine.PrintKeyValueTable([]string{
		"gorun table drop -n temp_data|Drop table with confirmation",
		"gorun table drop -n old_logs -f|Force drop without confirmation",
	}, engine.ColorLightGray, engine.ColorComment)

	listCmdStr := fmt.Sprintf(
		"%s%s %s%s %s%s %s%s%s",
		engine.ColorKeyword, "gorun",
		engine.ColorOrange, "table",
		engine.ColorOrange, "list",
		engine.ColorComment, "[options]",
		engine.ColorReset,
	)
	engine.PrintCodeBlock(listCmdStr, "List all tables in a database")
	fmt.Println()
	engine.PrintInfo("Options:")
	engine.PrintKeyValueTable([]string{
		"-d, --database|Specify the database name",
		"-t, --type|Specify database type (mysql/postgresql)",
		"--details|Show detailed table information",
	}, engine.ColorLightGray, engine.ColorComment)
	fmt.Println()
	engine.PrintInfo("Aliases:")
	engine.PrintKeyValueTable([]string{
		"ls|Shortcut for list command",
	}, engine.ColorLightGray, engine.ColorComment)
	fmt.Println()
	engine.PrintInfo("Examples:")
	engine.PrintKeyValueTable([]string{
		"gorun table list|List tables with interactive prompts",
		"gorun table ls -d mydb --details|List tables with detailed info",
	}, engine.ColorLightGray, engine.ColorComment)

	truncateCmdStr := fmt.Sprintf(
		"%s%s %s%s %s%s %s%s %s%s%s",
		engine.ColorKeyword, "gorun",
		engine.ColorOrange, "table",
		engine.ColorOrange, "truncate",
		engine.ColorComment, "[name]",
		engine.ColorComment, "[options]",
		engine.ColorReset,
	)
	engine.PrintCodeBlock(truncateCmdStr, "Truncate a table")
	fmt.Println()
	engine.PrintInfo("Options:")
	engine.PrintKeyValueTable([]string{
		"-n, --name|Specify the table name (required)",
		"-d, --database|Specify the database name",
		"-t, --type|Specify database type (mysql/postgresql)",
		"-f, --force|Skip confirmation prompt",
	}, engine.ColorLightGray, engine.ColorComment)
	fmt.Println()
	engine.PrintWarning("This will delete all data in the table!")
	fmt.Println()
	engine.PrintInfo("Examples:")
	engine.PrintKeyValueTable([]string{
		"gorun table truncate -n logs|Truncate table with confirmation",
		"gorun table truncate -n sessions -f|Force truncate without confirmation",
	}, engine.ColorLightGray, engine.ColorComment)

	engine.PrintDivider()
	engine.PrintWarning("Important Notes")
	fmt.Println()
	engine.PrintNormal("• Always backup your data before destructive operations")
	engine.PrintNormal("• Table names are case-sensitive in some database systems")
	engine.PrintNormal("• Some operations may require special privileges")
	engine.PrintNormal("• The 'truncate' command will reset auto-increment counters")
	engine.PrintNormal("• Use --force flag carefully as it skips confirmation prompts")
	fmt.Println()
}
