package app

import (
	"fmt"

	"github.com/RohmanBenyRiyanto/gorun/internal/engine"
)

// AppHelper prints `gorun app help`'s detailed usage listing.
type AppHelper struct{}

// NewAppHelper builds an AppHelper.
func NewAppHelper() *AppHelper {
	return &AppHelper{}
}

// ShowHelp prints usage, an available-commands table, and detailed
// per-subcommand examples.
func (h *AppHelper) ShowHelp() {
	engine.PrintBoldCard("APPLICATION COMMANDS HELP")

	engine.PrintTextH1("USAGE:")
	fmt.Printf(
		"  %s%s %s%s %s%s %s%s%s\n",
		engine.ColorKeyword, "gorun",
		engine.ColorOrange, "app",
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
		{"build", "Build the application binary"},
		{"status", "Check application status and information"},
		{"serve", "Start the application server"},
		{"test", "Run application tests"},
		{"clean", "Clean build artifacts and temporary files"},
		{"install", "Install application dependencies"},
		{"version", "Show application version information"},
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

	buildCmdStr := fmt.Sprintf(
		"%s%s %s%s %s%s %s%s%s",
		engine.ColorKeyword, "gorun",
		engine.ColorOrange, "app",
		engine.ColorOrange, "build",
		engine.ColorComment, "[options]",
		engine.ColorReset,
	)
	engine.PrintCodeBlock(buildCmdStr, "Build the application binary")
	fmt.Println()
	engine.PrintInfo("Options:")
	engine.PrintKeyValueTable([]string{
		"-o, --output|Output binary name (default: app)",
		"--os|Target operating system (linux, windows, darwin)",
		"--arch|Target architecture (amd64, arm64, 386)",
		"--race|Enable race detector",
		"--docker|Build for docker container",
		"--ldflags|Pass 'flag' to the Go linker",
		"--tags|Build tags to pass to Go",
		"--verbose|Enable verbose build output",
	}, engine.ColorLightGray, engine.ColorComment)
	fmt.Println()
	engine.PrintInfo("Aliases: compile")
	fmt.Println()
	engine.PrintInfo("Examples:")
	engine.PrintKeyValueTable([]string{
		"gorun app build|Build with default settings",
		"gorun app build -o myapp --os linux|Build for Linux with custom name",
		"gorun app build --race --verbose|Build with race detector and verbose output",
		"gorun app build --docker|Build optimized for Docker container",
	}, engine.ColorLightGray, engine.ColorComment)

	statusCmdStr := fmt.Sprintf(
		"%s%s %s%s %s%s %s%s%s",
		engine.ColorKeyword, "gorun",
		engine.ColorOrange, "app",
		engine.ColorOrange, "status",
		engine.ColorComment, "[options]",
		engine.ColorReset,
	)
	engine.PrintCodeBlock(statusCmdStr, "Show application status and information")
	fmt.Println()
	engine.PrintInfo("Options:")
	engine.PrintKeyValueTable([]string{
		"--detailed|Show detailed information",
		"--json|Output in JSON format",
		"--health|Check application health",
	}, engine.ColorLightGray, engine.ColorComment)
	fmt.Println()
	engine.PrintInfo("Aliases: info")
	fmt.Println()
	engine.PrintInfo("Examples:")
	engine.PrintKeyValueTable([]string{
		"gorun app status|Show basic status information",
		"gorun app status --detailed|Show comprehensive status details",
		"gorun app status --json|Output status in JSON format",
		"gorun app status --health|Check application health status",
	}, engine.ColorLightGray, engine.ColorComment)

	serveCmdStr := fmt.Sprintf(
		"%s%s %s%s %s%s %s%s%s",
		engine.ColorKeyword, "gorun",
		engine.ColorOrange, "app",
		engine.ColorOrange, "serve",
		engine.ColorComment, "[options]",
		engine.ColorReset,
	)
	engine.PrintCodeBlock(serveCmdStr, "Start the application server")
	fmt.Println()
	engine.PrintInfo("Options:")
	engine.PrintKeyValueTable([]string{
		"-p, --port|Port to serve on",
		"--host|Host to bind to (default: localhost)",
		"--dev|Enable development mode",
		"--watch|Enable hot reload (requires air)",
		"--env|Environment (development, staging, production)",
	}, engine.ColorLightGray, engine.ColorComment)
	fmt.Println()
	engine.PrintInfo("Aliases: start, run")
	fmt.Println()
	engine.PrintInfo("Examples:")
	engine.PrintKeyValueTable([]string{
		"gorun app serve|Start server with default settings",
		"gorun app serve -p 8080|Start server on port 8080",
		"gorun app serve --dev --watch|Start in development mode with hot reload",
		"gorun app serve --env production|Start in production environment",
	}, engine.ColorLightGray, engine.ColorComment)

	testCmdStr := fmt.Sprintf(
		"%s%s %s%s %s%s %s%s%s",
		engine.ColorKeyword, "gorun",
		engine.ColorOrange, "app",
		engine.ColorOrange, "test",
		engine.ColorComment, "[options]",
		engine.ColorReset,
	)
	engine.PrintCodeBlock(testCmdStr, "Run application tests")
	fmt.Println()
	engine.PrintInfo("Options:")
	engine.PrintKeyValueTable([]string{
		"--package, --pkg|Run tests for specific package",
		"--verbose|Enable verbose test output",
		"--race|Enable race detector",
		"--coverage|Enable coverage analysis",
		"--coverprofile|Write coverage profile to file",
		"--timeout|Test timeout in seconds (default: 30)",
		"--short|Run short tests only",
	}, engine.ColorLightGray, engine.ColorComment)
	fmt.Println()
	engine.PrintInfo("Aliases: t")
	fmt.Println()
	engine.PrintInfo("Examples:")
	engine.PrintKeyValueTable([]string{
		"gorun app test|Run all tests",
		"gorun app test --verbose --coverage|Run tests with verbose output and coverage",
		"gorun app test --pkg ./handlers|Run tests for specific package",
		"gorun app test --race --timeout 60|Run tests with race detector and custom timeout",
	}, engine.ColorLightGray, engine.ColorComment)

	cleanCmdStr := fmt.Sprintf(
		"%s%s %s%s %s%s %s%s%s",
		engine.ColorKeyword, "gorun",
		engine.ColorOrange, "app",
		engine.ColorOrange, "clean",
		engine.ColorComment, "[options]",
		engine.ColorReset,
	)
	engine.PrintCodeBlock(cleanCmdStr, "Clean build artifacts and temporary files")
	fmt.Println()
	engine.PrintInfo("Options:")
	engine.PrintKeyValueTable([]string{
		"--cache|Clean Go build cache",
		"--modules|Clean module cache",
		"--logs|Clean log files",
		"--all|Clean everything",
		"-f, --force|Force clean without confirmation",
	}, engine.ColorLightGray, engine.ColorComment)
	fmt.Println()
	engine.PrintInfo("Aliases: clear")
	engine.PrintWarning("Some operations may require confirmation unless --force is used")
	fmt.Println()
	engine.PrintInfo("Examples:")
	engine.PrintKeyValueTable([]string{
		"gorun app clean|Clean build artifacts with confirmation",
		"gorun app clean --all -f|Force clean everything without confirmation",
		"gorun app clean --cache --logs|Clean build cache and log files",
		"gorun app clean --modules|Clean Go module cache",
	}, engine.ColorLightGray, engine.ColorComment)

	installCmdStr := fmt.Sprintf(
		"%s%s %s%s %s%s %s%s%s",
		engine.ColorKeyword, "gorun",
		engine.ColorOrange, "app",
		engine.ColorOrange, "install",
		engine.ColorComment, "[options]",
		engine.ColorReset,
	)
	engine.PrintCodeBlock(installCmdStr, "Install application dependencies")
	fmt.Println()
	engine.PrintInfo("Options:")
	engine.PrintKeyValueTable([]string{
		"--update|Update dependencies to latest versions",
		"--tidy|Run go mod tidy",
		"--vendor|Download dependencies to vendor directory",
		"--verify|Verify dependencies",
	}, engine.ColorLightGray, engine.ColorComment)
	fmt.Println()
	engine.PrintInfo("Aliases: deps")
	fmt.Println()
	engine.PrintInfo("Examples:")
	engine.PrintKeyValueTable([]string{
		"gorun app install|Install dependencies",
		"gorun app install --update --tidy|Update and tidy dependencies",
		"gorun app install --vendor|Install dependencies to vendor directory",
		"gorun app install --verify|Install and verify dependencies",
	}, engine.ColorLightGray, engine.ColorComment)

	versionCmdStr := fmt.Sprintf(
		"%s%s %s%s %s%s %s%s%s",
		engine.ColorKeyword, "gorun",
		engine.ColorOrange, "app",
		engine.ColorOrange, "version",
		engine.ColorComment, "[options]",
		engine.ColorReset,
	)
	engine.PrintCodeBlock(versionCmdStr, "Show application version information")
	fmt.Println()
	engine.PrintInfo("Options:")
	engine.PrintKeyValueTable([]string{
		"--json|Output in JSON format",
		"--short|Show version only",
	}, engine.ColorLightGray, engine.ColorComment)
	fmt.Println()
	engine.PrintInfo("Aliases:")
	fmt.Println()
	engine.PrintInfo("Examples:")
	engine.PrintKeyValueTable([]string{
		"gorun app version|Show full version information",
		"gorun app version --short|Show version number only",
		"gorun app version --json|Show version information in JSON format",
	}, engine.ColorLightGray, engine.ColorComment)

	engine.PrintDivider()
	engine.PrintWarning("Important Notes")
	fmt.Println()
	engine.PrintNormal("• Use --dev flag when developing to enable development features")
	engine.PrintNormal("• Always run tests before building for production")
	engine.PrintNormal("• The --watch flag requires 'air' to be installed for hot reload")
	engine.PrintNormal("• Clean operations with --all flag will remove all build artifacts")
	engine.PrintNormal("• Use --force flag carefully as it skips confirmation prompts")
	engine.PrintNormal("• Environment variables can override command line flags")
	fmt.Println()
}
