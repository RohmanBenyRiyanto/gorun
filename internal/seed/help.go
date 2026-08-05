package seed

import (
	"fmt"
	"strings"

	"github.com/RohmanBenyRiyanto/gorun/internal/engine"
)

// SeedHelper prints `gorun seed help`'s detailed usage listing.
type SeedHelper struct{}

// NewSeedHelper builds a SeedHelper.
func NewSeedHelper() *SeedHelper {
	return &SeedHelper{}
}

// ShowHelp prints usage, an available-commands table, and detailed
// per-subcommand examples.
func (sh *SeedHelper) ShowHelp() {
	engine.PrintBoldCard("SEEDER COMMANDS HELP")

	engine.PrintTextH1("USAGE:")
	fmt.Printf(
		"  %s%s %s%s %s%s %s%s%s\n",
		engine.ColorKeyword, "gorun",
		engine.ColorOrange, "seed",
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
		{"run", "Execute database seeders"},
		{"make", "Create new seeder file"},
		{"list", "List available seeders"},
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

	runCmdGroupStr := fmt.Sprintf(
		"%s%s %s%s %s%s %s%s%s\n%s%s %s%s %s%s %s%s %s%s%s",
		engine.ColorKeyword, "gorun",
		engine.ColorOrange, "seed    ",
		engine.ColorComment, "[command]",
		engine.ColorComment, "[options]",
		engine.ColorReset,
		engine.ColorKeyword, "gorun",
		engine.ColorOrange, "seed",
		engine.ColorOrange, "run",
		engine.ColorComment, "[options]",
		engine.ColorComment, "[--class <name>]",
		engine.ColorReset,
	)

	engine.PrintCodeBlock(runCmdGroupStr, "Execute database seeders")
	engine.PrintInfo("Options:")
	engine.PrintKeyValueTable([]string{
		"--class, -c|Run specific seeder class",
		"--seeder, -s|Alias for --class",
		"--database, -db|Specify database connection",
		"--force, -f|Force run in production environment",
		"--only, -o|Run only these seeders (comma separated)",
		"--except, -e|Exclude these seeders (comma separated)",
		"--transaction, -t|Run seeders in a transaction (default: true)",
		"--stop-on-error|Stop on the first failing seeder (default: true; false runs all, reports combined failures)",
	}, engine.ColorLightGray, engine.ColorComment)

	makeCmdStr := fmt.Sprintf(
		"%s%s %s%s %s%s %s%s %s%s%s",
		engine.ColorKeyword, "gorun",
		engine.ColorOrange, "seed",
		engine.ColorOrange, "make",
		engine.ColorComment, "<name>",
		engine.ColorComment, "[options]",
		engine.ColorReset,
	)

	engine.PrintCodeBlock(makeCmdStr, "Create new seeder file")
	engine.PrintInfo("Options:")
	engine.PrintKeyValueTable([]string{
		"--path, -p|Location to store seeder file",
		"--realpath|Use absolute path",
		"--fullpath|Show full path after creation",
		"--model|Generate model with seeder",
		"--table|Specify table name for model",
	}, engine.ColorLightGray, engine.ColorComment)

	listCmdStr := fmt.Sprintf(
		"%s%s %s%s %s%s %s%s%s",
		engine.ColorKeyword, "gorun",
		engine.ColorOrange, "seed",
		engine.ColorOrange, "list",
		engine.ColorComment, "[options]",
		engine.ColorReset,
	)

	engine.PrintCodeBlock(listCmdStr, "List available seeders")
	engine.PrintInfo("Options:")
	engine.PrintKeyValueTable([]string{
		"--details|Show detailed information",
	}, engine.ColorLightGray, engine.ColorComment)

	engine.PrintDivider()
	engine.PrintWarning("Important Notes")
	fmt.Println()
	engine.PrintNormal("• Seeders are typically used to populate database with test data")
	engine.PrintNormal("• Be cautious when running seeders in production environment")
	engine.PrintNormal("• Use --transaction flag to ensure data consistency")
	engine.PrintNormal("• Production runs require --force (app.env=prod/production is blocked otherwise)")
	engine.PrintNormal("• Seeders should be idempotent - safe to run multiple times")
	fmt.Println()
}

// PrintColoredCodeBlock prints a sample "gorun seed [options]" /
// "gorun seed run [options]" code block. Not currently called from
// ShowHelp (which builds its own per-subcommand blocks inline) - kept for
// callers that want just this one snippet.
func PrintColoredCodeBlock() {
	code := []string{
		fmt.Sprintf(
			"%s%s %s%s %s%s%s",
			engine.ColorKeyword, "gorun",
			engine.ColorOrange, "seed",
			engine.ColorComment, "[options]",
			engine.ColorReset,
		),
		fmt.Sprintf(
			"%s%s %s%s %s%s %s%s%s",
			engine.ColorKeyword, "gorun",
			engine.ColorOrange, "seed",
			engine.ColorFunction, "run",
			engine.ColorComment, "[options]",
			engine.ColorReset,
		),
	}

	coloredCode := strings.Join(code, "\n")
	engine.PrintCodeBlock(coloredCode, "Execute database seeders")
}
