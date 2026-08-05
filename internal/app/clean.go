package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/RohmanBenyRiyanto/gorun/internal/engine"
	"github.com/urfave/cli/v3"
)

// CleanCommand implements `gorun app clean` - removing the Go build/module
// cache and stale log files.
type CleanCommand struct {
	config *engine.Config
}

// NewCleanCommand builds a CleanCommand and prints its banner.
func NewCleanCommand(config *engine.Config) *CleanCommand {
	engine.PrintBoldCard("APPLICATION COMMANDS:CLEAN")
	return &CleanCommand{
		config: config,
	}
}

func (cc *CleanCommand) Handle(ctx context.Context, cmd *cli.Command) error {
	cleanCache := cmd.Bool("cache")
	cleanModules := cmd.Bool("modules")
	cleanLogs := cmd.Bool("logs")
	cleanAll := cmd.Bool("all")
	force := cmd.Bool("force")

	root, err := engine.GetProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to get project root: %w", err)
	}

	if cleanAll {
		cleanCache = true
		cleanModules = true
		cleanLogs = true
	}

	if !cleanCache && !cleanModules && !cleanLogs {
		cleanCache = true
		cleanModules = true
		cleanLogs = true
	}

	engine.PrintSectionHeader("Cleaning Project")

	cleanItems := []string{}
	if cleanCache {
		cleanItems = append(cleanItems, "Go build cache")
	}
	if cleanModules {
		cleanItems = append(cleanItems, "Go module cache")
	}
	if cleanLogs {
		cleanItems = append(cleanItems, "Log files")
	}

	engine.PrintInfo("Will clean the following:")
	for _, item := range cleanItems {
		fmt.Printf("  - %s\n", item)
	}
	fmt.Println()

	if !force {
		confirmed := engine.ConfirmPrompt("Continue with clean?")
		if !confirmed {
			engine.PrintInfo("Clean cancelled")
			return nil
		}
	}

	if cleanCache {
		if err := cc.cleanBuildCache(); err != nil {
			return fmt.Errorf("failed to clean build cache: %w", err)
		}
	}

	if cleanModules {
		if err := cc.cleanModuleCache(); err != nil {
			return fmt.Errorf("failed to clean module cache: %w", err)
		}
	}

	if cleanLogs {
		if err := cc.cleanLogFiles(root); err != nil {
			return fmt.Errorf("failed to clean log files: %w", err)
		}
	}

	engine.PrintSuccess("Clean completed successfully!")
	return nil
}

func (cc *CleanCommand) cleanBuildCache() error {
	engine.PrintInfo("Cleaning Go build cache...")
	cmd := exec.Command("go", "clean", "-cache")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (cc *CleanCommand) cleanModuleCache() error {
	engine.PrintInfo("Cleaning Go module cache...")
	cmd := exec.Command("go", "clean", "-modcache")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (cc *CleanCommand) cleanLogFiles(root string) error {
	engine.PrintInfo("Cleaning log files...")
	logDir := filepath.Join(root, "logs")

	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		engine.PrintInfo("No logs directory found")
		return nil
	}

	files, err := os.ReadDir(logDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".log") {
			filePath := filepath.Join(logDir, file.Name())
			if err := os.Remove(filePath); err != nil {
				return err
			}
			engine.PrintInfo("Removed log file: %s", file.Name())
		}
	}

	return nil
}
