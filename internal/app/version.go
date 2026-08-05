package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/RohmanBenyRiyanto/gorun/internal/engine"
	"github.com/urfave/cli/v3"
)

// VersionCommand implements `gorun app version` - app name/version plus
// Go and git build metadata.
type VersionCommand struct {
	config *engine.Config
}

// VersionInfo is VersionCommand's collected snapshot, printable as text or
// JSON.
type VersionInfo struct {
	Application struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"application"`
	Build struct {
		Commit string `json:"commit"`
		Date   string `json:"date"`
		Branch string `json:"branch"`
	} `json:"build"`
	Go struct {
		Version string `json:"version"`
		OS      string `json:"os"`
		Arch    string `json:"arch"`
	} `json:"go"`
}

// NewVersionCommand builds a VersionCommand and prints its banner.
func NewVersionCommand(config *engine.Config) *VersionCommand {
	engine.PrintBoldCard("APPLICATION COMMANDS:VERSION")
	return &VersionCommand{
		config: config,
	}
}

func (vc *VersionCommand) Handle(ctx context.Context, cmd *cli.Command) error {
	jsonOutput := cmd.Bool("json")
	short := cmd.Bool("short")

	version := vc.collectVersionInfo()

	if jsonOutput {
		return vc.outputJSON(version)
	}

	return vc.outputFormatted(version, short)
}

func (vc *VersionCommand) collectVersionInfo() *VersionInfo {
	version := &VersionInfo{}

	version.Application.Name = vc.getAppName()
	version.Application.Version = vc.getAppVersion()

	version.Build = vc.getBuildInfo()

	version.Go.Version = runtime.Version()
	version.Go.OS = runtime.GOOS
	version.Go.Arch = runtime.GOARCH

	return version
}

func (vc *VersionCommand) outputJSON(version *VersionInfo) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(version)
}

func (vc *VersionCommand) outputFormatted(version *VersionInfo, short bool) error {
	if short {
		fmt.Println(version.Application.Version)
		return nil
	}

	engine.PrintSectionHeader("Version Information")

	engine.PrintTextH1("Application")
	appData := []string{
		fmt.Sprintf("Name|%s", version.Application.Name),
		fmt.Sprintf("Version|%s", version.Application.Version),
	}
	engine.PrintKeyValueTable(appData, engine.ColorFile, engine.ColorComment)

	fmt.Println()

	engine.PrintTextH1("Build")
	buildData := []string{
		fmt.Sprintf("Commit|%s", version.Build.Commit),
		fmt.Sprintf("Date|%s", version.Build.Date),
		fmt.Sprintf("Branch|%s", version.Build.Branch),
	}
	engine.PrintKeyValueTable(buildData, engine.ColorFile, engine.ColorComment)

	fmt.Println()

	engine.PrintTextH1("Go")
	goData := []string{
		fmt.Sprintf("Version|%s", version.Go.Version),
		fmt.Sprintf("OS|%s", version.Go.OS),
		fmt.Sprintf("Arch|%s", version.Go.Arch),
	}
	engine.PrintKeyValueTable(goData, engine.ColorFile, engine.ColorComment)

	return nil
}

func (vc *VersionCommand) getAppName() string {
	if moduleName, err := engine.GetGoModuleName(); err == nil {
		parts := strings.Split(moduleName, "/")
		return parts[len(parts)-1]
	}
	return unknownValue
}

func (vc *VersionCommand) getAppVersion() string {
	if version := os.Getenv("APP_VERSION"); version != "" {
		return version
	}

	if commit, err := vc.getGitCommit(); err == nil {
		return "git-" + commit
	}

	return "development"
}

func (vc *VersionCommand) getBuildInfo() struct {
	Commit string `json:"commit"`
	Date   string `json:"date"`
	Branch string `json:"branch"`
} {
	commit, _ := vc.getGitCommit()
	date, _ := vc.getGitDate()
	branch, _ := vc.getGitBranch()

	return struct {
		Commit string `json:"commit"`
		Date   string `json:"date"`
		Branch string `json:"branch"`
	}{
		Commit: commit,
		Date:   date,
		Branch: branch,
	}
}

func (vc *VersionCommand) getGitCommit() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (vc *VersionCommand) getGitDate() (string, error) {
	cmd := exec.Command("git", "log", "-1", "--format=%cd", "--date=short")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (vc *VersionCommand) getGitBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
