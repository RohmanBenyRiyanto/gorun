package migration

import (
	"fmt"
	"strings"

	"github.com/RohmanBenyRiyanto/gorun/internal/engine"
)

// MigrationHelper prints `gorun migrate help`'s detailed usage listing.
type MigrationHelper struct{}

// NewMigrationHelper builds a MigrationHelper.
func NewMigrationHelper() *MigrationHelper {
	return &MigrationHelper{}
}

// ShowHelp prints usage, an available-commands table, and detailed
// per-subcommand examples.
func (mh *MigrationHelper) ShowHelp() {
	engine.PrintBoldCard("MIGRATION COMMANDS HELP")

	engine.PrintTextH1("USAGE:")
	fmt.Printf(
		"  %s%s %s%s %s%s %s%s%s\n",
		engine.ColorKeyword, "gorun",
		engine.ColorOrange, "migrate",
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
		{"run", "Run pending migrations"},
		{"status", "Show migration status"},
		{"make", "Create new migration file"},
		{"rollback", "Rollback the last migration"},
		{"reset", "Rollback all migrations"},
		{"refresh", "Reset and re-run all migrations"},
		{"fresh", "Drop all tables and re-run migrations"},
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

	runCmdGroupStr := fmt.Sprintf(
		"%s%s %s%s %s%s %s%s%s\n%s%s %s%s %s%s %s%s %s%s%s",
		engine.ColorKeyword, "gorun",
		engine.ColorOrange, "migrate    ",
		engine.ColorComment, "[command]",
		engine.ColorComment, "[options]",
		engine.ColorReset,
		engine.ColorKeyword, "gorun",
		engine.ColorOrange, "migrate",
		engine.ColorOrange, "run",
		engine.ColorComment, "[command]",
		engine.ColorComment, "[options]",
		engine.ColorReset,
	)

	engine.PrintCodeBlock(runCmdGroupStr, "Run pending migrations")
	engine.PrintInfo("Options:")
	engine.PrintKeyValueTable([]string{
		"--force, -f|Run without confirmation",
		"--path, -p|Run migrations only from specific path",
		"--file, -f|Run specific migration file (name without extension)",
		"--pretend|Only show SQL to be executed",
		"--step|Run as individual steps for partial rollback",
	}, engine.ColorLightGray, engine.ColorComment)

	statusCmdStr := fmt.Sprintf(
		"%s%s %s%s %s%s %s%s%s",
		engine.ColorKeyword, "gorun",
		engine.ColorOrange, "migrate",
		engine.ColorOrange, "status",
		engine.ColorComment, "[options]",
		engine.ColorReset,
	)

	engine.PrintCodeBlock(statusCmdStr, "Show migration status")
	engine.PrintInfo("Options:")
	engine.PrintKeyValueTable([]string{
		"--path, -p|Show status for specific path only",
		"--format|Output format (text, json)",
	}, engine.ColorLightGray, engine.ColorComment)

	makeCmdStr := fmt.Sprintf(
		"%s%s %s%s %s%s %s%s %s%s%s",
		engine.ColorKeyword, "gorun",
		engine.ColorOrange, "migrate",
		engine.ColorOrange, "make",
		engine.ColorComment, "<name>",
		engine.ColorComment, "[options]",
		engine.ColorReset,
	)

	engine.PrintCodeBlock(makeCmdStr, "Create new migration file")
	engine.PrintInfo("Options:")
	engine.PrintKeyValueTable([]string{
		"-c, --create|Create new table",
		"-t, --table|Modify existing table",
		"-p, --path|Location to store migration file",
		"--realpath|Use absolute path",
		"--fullpath|Show full path after creation",
	}, engine.ColorLightGray, engine.ColorComment)

	rollbackCmdStr := fmt.Sprintf(
		"%s%s %s%s %s%s %s%s%s",
		engine.ColorKeyword, "gorun",
		engine.ColorOrange, "migrate",
		engine.ColorOrange, "rollback",
		engine.ColorComment, "[options]",
		engine.ColorReset,
	)

	engine.PrintCodeBlock(rollbackCmdStr, "Rollback the last migration")
	engine.PrintInfo("Options:")
	engine.PrintKeyValueTable([]string{
		"-s, --step|Number of batches to rollback (default: 1)",
		"-p, --path|Limit rollback to specific path",
		"-f, --force|Run without confirmation",
	}, engine.ColorLightGray, engine.ColorComment)

	resetCmdStr := fmt.Sprintf(
		"%s%s %s%s %s%s %s%s%s",
		engine.ColorKeyword, "gorun",
		engine.ColorOrange, "migrate",
		engine.ColorOrange, "reset",
		engine.ColorComment, "[options]",
		engine.ColorReset,
	)

	engine.PrintCodeBlock(resetCmdStr, "Rollback all migrations")
	engine.PrintInfo("Options:")
	engine.PrintKeyValueTable([]string{
		"-f, --force|Run without confirmation",
	}, engine.ColorLightGray, engine.ColorComment)

	refreshCmdStr := fmt.Sprintf(
		"%s%s %s%s %s%s %s%s%s",
		engine.ColorKeyword, "gorun",
		engine.ColorOrange, "migrate",
		engine.ColorOrange, "refresh",
		engine.ColorComment, "[options]",
		engine.ColorReset,
	)

	engine.PrintCodeBlock(refreshCmdStr, "Reset and re-run all migrations")
	engine.PrintInfo("Options:")
	engine.PrintKeyValueTable([]string{
		"--seed|Run seeders after migration",
		"-s, --step|Run migrations per step",
		"-f, --force|Run without confirmation",
		"-F, --file|Run specific migration file (name without extension)",
	}, engine.ColorLightGray, engine.ColorComment)

	freshCmdStr := fmt.Sprintf(
		"%s%s %s%s %s%s %s%s%s",
		engine.ColorKeyword, "gorun",
		engine.ColorOrange, "migrate",
		engine.ColorOrange, "fresh",
		engine.ColorComment, "[options]",
		engine.ColorReset,
	)

	engine.PrintCodeBlock(freshCmdStr, "Drop all tables and re-run migrations")
	engine.PrintInfo("Options:")
	engine.PrintKeyValueTable([]string{
		"--seed|Run seeders after migration",
		"--drop-views|Drop all views",
		"--drop-types|Drop custom types (PostgreSQL)",
		"-f, --force|Run without confirmation",
		"-F, --file|Run specific migration file (name without extension)",
	}, engine.ColorLightGray, engine.ColorComment)

	engine.PrintDivider()
	engine.PrintWarning("Important Notes")
	fmt.Println()
	engine.PrintNormal("• Always backup your database before running migrations in production")
	engine.PrintNormal("• Use --pretend flag to preview SQL before executing")
	engine.PrintNormal("• Test migrations in development environment first")
	engine.PrintNormal("• Keep migration files in version control")
	engine.PrintNormal("• Never modify existing migration files after deployment")
	fmt.Println()
}

// PrintColoredCodeBlock prints a sample "gorun migrate [options]" /
// "gorun migrate run [options]" code block. Not currently called from
// ShowHelp (which builds its own per-subcommand blocks inline) - kept for
// callers that want just this one snippet.
func PrintColoredCodeBlock() {
	code := []string{
		fmt.Sprintf(
			"%s%s %s%s %s%s%s",
			engine.ColorKeyword, "gorun",
			engine.ColorOrange, "migrate",
			engine.ColorComment, "[options]",
			engine.ColorReset,
		),
		fmt.Sprintf(
			"%s%s %s%s %s%s %s%s%s",
			engine.ColorKeyword, "gorun",
			engine.ColorOrange, "migrate",
			engine.ColorFunction, "run",
			engine.ColorComment, "[options]",
			engine.ColorReset,
		),
	}

	coloredCode := strings.Join(code, "\n")

	engine.PrintCodeBlock(coloredCode, "Run pending migrations")
}
