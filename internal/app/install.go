package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/RohmanBenyRiyanto/gorun/internal/engine"
	"github.com/urfave/cli/v3"
)

// InstallCommand implements `gorun app install` - wraps `go mod`
// download/tidy/vendor/verify/get -u.
type InstallCommand struct {
	config *engine.Config
}

// NewInstallCommand builds an InstallCommand and prints its banner.
func NewInstallCommand(config *engine.Config) *InstallCommand {
	engine.PrintBoldCard("APPLICATION COMMANDS:INSTALL")
	return &InstallCommand{
		config: config,
	}
}

func (ic *InstallCommand) Handle(ctx context.Context, cmd *cli.Command) error {
	update := cmd.Bool("update")
	tidy := cmd.Bool("tidy")
	vendor := cmd.Bool("vendor")
	verify := cmd.Bool("verify")

	engine.PrintSectionHeader("Installing Dependencies")

	actions := []string{}
	if update {
		actions = append(actions, "Update dependencies to latest versions")
	}
	if tidy {
		actions = append(actions, "Run go mod tidy")
	}
	if vendor {
		actions = append(actions, "Download dependencies to vendor directory")
	}
	if verify {
		actions = append(actions, "Verify dependencies")
	}

	if len(actions) == 0 {
		actions = append(actions, "Run go mod download")
	}

	engine.PrintInfo("Will perform the following actions:")
	for _, action := range actions {
		fmt.Printf("  - %s\n", action)
	}
	fmt.Println()

	if update {
		if err := ic.updateDependencies(); err != nil {
			return fmt.Errorf("failed to update dependencies: %w", err)
		}
	}

	if tidy {
		if err := ic.runModTidy(); err != nil {
			return fmt.Errorf("failed to run go mod tidy: %w", err)
		}
	}

	if vendor {
		if err := ic.vendorDependencies(); err != nil {
			return fmt.Errorf("failed to vendor dependencies: %w", err)
		}
	}

	if verify {
		if err := ic.verifyDependencies(); err != nil {
			return fmt.Errorf("failed to verify dependencies: %w", err)
		}
	}

	if !update && !tidy && !vendor && !verify {
		if err := ic.downloadDependencies(); err != nil {
			return fmt.Errorf("failed to download dependencies: %w", err)
		}
	}

	engine.PrintSuccess("Dependency installation completed successfully!")
	return nil
}

func (ic *InstallCommand) updateDependencies() error {
	engine.PrintInfo("Updating dependencies...")
	cmd := exec.Command("go", "get", "-u", "./...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (ic *InstallCommand) runModTidy() error {
	engine.PrintInfo("Running go mod tidy...")
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (ic *InstallCommand) vendorDependencies() error {
	engine.PrintInfo("Vendoring dependencies...")
	cmd := exec.Command("go", "mod", "vendor")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (ic *InstallCommand) verifyDependencies() error {
	engine.PrintInfo("Verifying dependencies...")
	cmd := exec.Command("go", "mod", "verify")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (ic *InstallCommand) downloadDependencies() error {
	engine.PrintInfo("Downloading dependencies...")
	cmd := exec.Command("go", "mod", "download")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
