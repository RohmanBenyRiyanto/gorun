package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/RohmanBenyRiyanto/gorun/internal/engine"
	"github.com/urfave/cli/v3"
)

// ServeCommand implements `gorun app serve` - running
// Config.ServerEntrypoint directly, or through `air` when hot reload is
// requested.
type ServeCommand struct {
	config *engine.Config
}

// NewServeCommand builds a ServeCommand and prints its banner.
func NewServeCommand(config *engine.Config) *ServeCommand {
	engine.PrintBoldCard("APPLICATION COMMANDS:SERVE")
	return &ServeCommand{
		config: config,
	}
}

func (sc *ServeCommand) Handle(ctx context.Context, cmd *cli.Command) error {
	port := cmd.String("port")
	host := cmd.String("host")
	devMode := cmd.Bool("dev")
	watch := cmd.Bool("watch")
	env := cmd.String("env")

	root, err := engine.GetProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to get project root: %w", err)
	}

	envMap, err := engine.ReadEnvFile(filepath.Join(root, ".env"))
	if err != nil {
		engine.PrintWarning("Could not read .env file: %v", err)
		envMap = make(map[string]string)
	}

	if port == "" {
		if envPort := envMap["APP_PORT"]; envPort != "" {
			port = envPort
		} else {
			port = "8080"
		}
	}

	if host == "" {
		if envHost := envMap["APP_HOST"]; envHost != "" {
			host = envHost
		} else {
			host = "localhost"
		}
	}

	if env == "" {
		if envEnv := envMap["APP_ENV"]; envEnv != "" {
			env = envEnv
		} else {
			env = "development"
		}
	}

	if devMode {
		env = "development"
		if !watch {
			watch = true
		}
	}

	engine.PrintSectionHeader("Starting Application Server")

	serverInfo := []string{
		fmt.Sprintf("Environment|%s", env),
		fmt.Sprintf("Host|%s", host),
		fmt.Sprintf("Port|%s", port),
		fmt.Sprintf("URL|http://%s:%s", host, port),
	}

	if devMode {
		serverInfo = append(serverInfo, "Development Mode|enabled")
	}

	if watch {
		serverInfo = append(serverInfo, "Hot Reload|enabled (using air)")
	}

	engine.PrintKeyValueTable(serverInfo, engine.ColorFile, engine.ColorComment)
	fmt.Println()

	if watch {
		return sc.serveWithHotReload(ctx, root, env, host, port)
	}

	return sc.serveRegular(ctx, root, env, host, port)
}

func (sc *ServeCommand) serveRegular(ctx context.Context, root, env, host, port string) error {
	cmd := exec.Command("go", "run", "./"+sc.config.ServerEntrypoint)
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("APP_ENV=%s", env),
		fmt.Sprintf("APP_HOST=%s", host),
		fmt.Sprintf("APP_PORT=%s", port),
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	engine.PrintSuccess("Server started! Press Ctrl+C to stop")

	select {
	case <-ctx.Done():
		engine.PrintInfo("Shutting down server...")
	case sig := <-sigChan:
		engine.PrintInfo("Received signal %v, shutting down server...", sig)
	}

	if err := cmd.Process.Kill(); err != nil {
		return fmt.Errorf("failed to kill server process: %w", err)
	}

	return nil
}

func (sc *ServeCommand) serveWithHotReload(ctx context.Context, root, env, host, port string) error {
	cmd := exec.Command("air")
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("APP_ENV=%s", env),
		fmt.Sprintf("APP_HOST=%s", host),
		fmt.Sprintf("APP_PORT=%s", port),
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start air: %w", err)
	}

	engine.PrintSuccess("Server started with hot reload! Press Ctrl+C to stop")

	select {
	case <-ctx.Done():
		engine.PrintInfo("Shutting down server from context cancel...")
	case sig := <-sigChan:
		engine.PrintInfo("Received signal %v, shutting down server...", sig)
	}

	_ = cmd.Wait()

	engine.PrintInfo("Server process stopped successfully")

	// Brief delay to let the OS fully release the port before returning
	time.Sleep(1 * time.Second)

	return nil
}
