package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/RohmanBenyRiyanto/gorun/internal/engine"
	"github.com/urfave/cli/v3"
)

// TestCommand implements `gorun app test` - wraps `go test` with the
// usual verbose/race/coverage/timeout flags.
type TestCommand struct {
	config *engine.Config
}

// NewTestCommand builds a TestCommand and prints its banner.
func NewTestCommand(config *engine.Config) *TestCommand {
	engine.PrintBoldCard("APPLICATION COMMANDS:TEST")
	return &TestCommand{
		config: config,
	}
}

func (tc *TestCommand) Handle(ctx context.Context, cmd *cli.Command) error {
	pkg := cmd.String("package")
	verbose := cmd.Bool("verbose")
	race := cmd.Bool("race")
	coverage := cmd.Bool("coverage")
	coverprofile := cmd.String("coverprofile")
	timeout := cmd.Int("timeout")
	short := cmd.Bool("short")

	root, err := engine.GetProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to get project root: %w", err)
	}

	engine.PrintSectionHeader("Running Tests")

	testArgs := []string{"test"}

	if verbose {
		testArgs = append(testArgs, "-v")
	}

	if race {
		testArgs = append(testArgs, "-race")
	}

	if coverage {
		testArgs = append(testArgs, "-cover")
	}

	if coverprofile != "" {
		testArgs = append(testArgs, "-coverprofile="+coverprofile)
	}

	if timeout > 0 {
		testArgs = append(testArgs, fmt.Sprintf("-timeout=%ds", timeout))
	}

	if short {
		testArgs = append(testArgs, "-short")
	}

	if pkg != "" {
		testArgs = append(testArgs, pkg)
	} else {
		testArgs = append(testArgs, "./...")
	}

	testInfo := []string{
		fmt.Sprintf("Package|%s", pkg),
		fmt.Sprintf("Verbose|%t", verbose),
		fmt.Sprintf("Race Detector|%t", race),
		fmt.Sprintf("Coverage|%t", coverage),
	}

	if coverprofile != "" {
		testInfo = append(testInfo, fmt.Sprintf("Cover Profile|%s", coverprofile))
	}

	if timeout > 0 {
		testInfo = append(testInfo, fmt.Sprintf("Timeout|%ds", timeout))
	}

	if short {
		testInfo = append(testInfo, "Short Mode|enabled")
	}

	engine.PrintKeyValueTable(testInfo, engine.ColorFile, engine.ColorComment)
	fmt.Println()

	testCmd := exec.CommandContext(ctx, "go", testArgs...)
	testCmd.Dir = root
	testCmd.Stdout = os.Stdout
	testCmd.Stderr = os.Stderr

	if err := testCmd.Run(); err != nil {
		return fmt.Errorf("tests failed: %w", err)
	}

	engine.PrintSuccess("Tests completed successfully!")
	return nil
}
