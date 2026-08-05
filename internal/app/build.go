package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/RohmanBenyRiyanto/gorun/internal/engine"
	"github.com/urfave/cli/v3"
)

// BuildCommand implements `gorun app build` - compiling
// Config.ServerEntrypoint via `go build`.
type BuildCommand struct {
	config *engine.Config
}

// NewBuildCommand builds a BuildCommand and prints its banner.
func NewBuildCommand(config *engine.Config) *BuildCommand {
	engine.PrintBoldCard("APPLICATION COMMANDS:BUILD")
	return &BuildCommand{
		config: config,
	}
}

// Handle runs the build: resolves flags, shells out to `go build` with the
// resulting args/env, and reports the output binary's size and location.
func (bc *BuildCommand) Handle(ctx context.Context, cmd *cli.Command) error {
	start := time.Now()

	root, err := engine.GetProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to get project root: %w", err)
	}

	moduleName, err := engine.GetGoModuleName()
	if err != nil {
		return fmt.Errorf("failed to get module name: %w", err)
	}

	output := cmd.String("output")
	targetOS := cmd.String("os")
	targetArch := cmd.String("arch")
	enableRace := cmd.Bool("race")
	dockerBuild := cmd.Bool("docker")
	ldflags := cmd.String("ldflags")
	buildTags := cmd.String("tags")
	verbose := cmd.Bool("verbose")

	if targetOS == "" {
		targetOS = runtime.GOOS
	}
	if targetArch == "" {
		targetArch = runtime.GOARCH
	}

	if dockerBuild {
		targetOS = "linux"
		if targetArch == "" {
			targetArch = "amd64"
		}
	}

	mainPath := filepath.Join(root, bc.config.ServerEntrypoint)
	if _, err := os.Stat(mainPath); os.IsNotExist(err) {
		return fmt.Errorf("main.go not found at %s", mainPath)
	}

	buildDir := filepath.Join(root, "build")
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return fmt.Errorf("failed to create build directory: %w", err)
	}

	outputPath := filepath.Join(buildDir, output)
	if targetOS == "windows" && !strings.HasSuffix(outputPath, ".exe") {
		outputPath += ".exe"
	}

	engine.PrintSectionHeader("Building Application")

	buildInfo := []string{
		fmt.Sprintf("Module|%s", moduleName),
		fmt.Sprintf("Source|%s", mainPath),
		fmt.Sprintf("Output|%s", outputPath),
		fmt.Sprintf("Target OS|%s", targetOS),
		fmt.Sprintf("Target Arch|%s", targetArch),
	}

	if enableRace {
		buildInfo = append(buildInfo, "Race Detector|enabled")
	}
	if dockerBuild {
		buildInfo = append(buildInfo, "Docker Build|enabled")
	}
	if ldflags != "" {
		buildInfo = append(buildInfo, fmt.Sprintf("Ldflags|%s", ldflags))
	}
	if buildTags != "" {
		buildInfo = append(buildInfo, fmt.Sprintf("Build Tags|%s", buildTags))
	}

	engine.PrintKeyValueTable(buildInfo, engine.ColorFile, engine.ColorComment)
	fmt.Println()

	buildArgs := []string{"build"}

	if verbose {
		buildArgs = append(buildArgs, "-v")
	}

	if enableRace && targetOS == runtime.GOOS && targetArch == runtime.GOARCH {
		buildArgs = append(buildArgs, "-race")
	}

	if ldflags != "" {
		buildArgs = append(buildArgs, "-ldflags", ldflags)
	} else {
		defaultLdflags := "-s -w"
		buildArgs = append(buildArgs, "-ldflags", defaultLdflags)
	}

	if buildTags != "" {
		buildArgs = append(buildArgs, "-tags", buildTags)
	}

	buildArgs = append(buildArgs, "-o", outputPath, mainPath)

	buildCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go engine.FancyProgressBarContext(buildCtx, "Building application", "")

	cmd_exec := exec.CommandContext(ctx, "go", buildArgs...)
	cmd_exec.Dir = root
	cmd_exec.Env = append(os.Environ(),
		fmt.Sprintf("GOOS=%s", targetOS),
		fmt.Sprintf("GOARCH=%s", targetArch),
		"CGO_ENABLED=0",
	)

	if verbose {
		cmd_exec.Stdout = os.Stdout
		cmd_exec.Stderr = os.Stderr
	} else {
		output_bytes, err := cmd_exec.CombinedOutput()
		if err != nil {
			cancel()
			engine.PrintError("Build failed: %v", err)
			if len(output_bytes) > 0 {
				fmt.Println(string(output_bytes))
			}
			return err
		}
	}

	if verbose {
		err = cmd_exec.Run()
	} else {
		_, err = cmd_exec.CombinedOutput()
	}

	cancel()

	if err != nil {
		engine.PrintError("Build failed: %v", err)
		return err
	}

	fileInfo, err := os.Stat(outputPath)
	if err != nil {
		return fmt.Errorf("failed to get build info: %w", err)
	}

	duration := time.Since(start)
	size := float64(fileInfo.Size()) / 1024 / 1024 // MB

	fmt.Println()
	engine.PrintSuccess("Build completed successfully!")

	summary := []string{
		fmt.Sprintf("Output File|%s", outputPath),
		fmt.Sprintf("File Size|%.2f MB", size),
		fmt.Sprintf("Build Time|%v", duration.Round(time.Millisecond)),
		fmt.Sprintf("Target|%s/%s", targetOS, targetArch),
	}

	engine.PrintKeyValueTable(summary, engine.ColorFile, engine.ColorComment)

	if targetOS == runtime.GOOS && targetArch == runtime.GOARCH {
		fmt.Println()
		engine.PrintTextNote("You can run the binary with: %s", outputPath)
	}

	return nil
}
