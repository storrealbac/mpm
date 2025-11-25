package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/storrealbac/mpm/internal/models"
	"github.com/storrealbac/mpm/internal/ui"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the Minecraft server",
	Long: `Start the Minecraft server using the configuration from package.yml.

The server will be started with the command specified in server.start_command,
or with a default command based on the server type if not specified.

After the server starts, any commands listed in startup_commands will be
executed in the server console in order.`,
	RunE: serveServer,
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func serveServer(cmd *cobra.Command, args []string) error {
	// Load package.yml
	pkg, err := models.LoadPackageFromFile("package.yml")
	if err != nil {
		return fmt.Errorf("failed to load package.yml: %w", err)
	}

	// Find the server jar file
	serverJar, err := findServerJar()
	if err != nil {
		return fmt.Errorf("server jar not found: %w\nRun 'mpm install' to download the server", err)
	}

	// Get or build the start command
	startCommand := pkg.Server.StartCommand
	if startCommand == "" {
		startCommand = buildDefaultStartCommand(serverJar)
	}

	ui.PrintInfo("Starting %s server (Minecraft %s)", pkg.Server.Type, pkg.Server.MinecraftVersion)
	fmt.Printf("  Command: %s\n", startCommand)

	// Check if we have startup commands to execute
	hasStartupCommands := len(pkg.StartupCommands) > 0
	if hasStartupCommands {
		ui.PrintInfo("Will execute %d startup command(s) after server starts", len(pkg.StartupCommands))
	}
	fmt.Println()

	// Start the server
	return startServerWithCommands(startCommand, pkg.StartupCommands)
}

func findServerJar() (string, error) {
	// Look for server jar files in common locations
	possibleNames := []string{"server.jar", "paper.jar", "purpur.jar", "folia.jar", "spigot.jar"}

	for _, name := range possibleNames {
		if _, err := os.Stat(name); err == nil {
			return name, nil
		}
	}

	// Also check in current directory for any .jar that might be the server
	files, err := filepath.Glob("*.jar")
	if err == nil && len(files) > 0 {
		// Return the first jar file found
		return files[0], nil
	}

	return "", fmt.Errorf("no server jar file found")
}

func buildDefaultStartCommand(serverJar string) string {
	// Default Java command with reasonable memory allocation
	// Users can override this in package.yml with server.start_command
	return fmt.Sprintf("java -Xms2G -Xmx4G -jar %s nogui", serverJar)
}

func startServerWithCommands(startCommand string, startupCommands []string) error {
	var serverCmd *exec.Cmd

	// Use appropriate shell based on OS
	if runtime.GOOS == "windows" {
		serverCmd = exec.Command("cmd", "/C", startCommand)
	} else {
		serverCmd = exec.Command("sh", "-c", startCommand)
	}

	// If we have startup commands, we need to monitor output and pipe commands
	if len(startupCommands) > 0 {
		stdin, err := serverCmd.StdinPipe()
		if err != nil {
			return fmt.Errorf("failed to create stdin pipe: %w", err)
		}

		stdout, err := serverCmd.StdoutPipe()
		if err != nil {
			return fmt.Errorf("failed to create stdout pipe: %w", err)
		}

		// Stderr still goes directly to terminal
		serverCmd.Stderr = os.Stderr

		// Start the server
		if err := serverCmd.Start(); err != nil {
			return fmt.Errorf("failed to start server: %w", err)
		}

		// Monitor server output and execute startup commands when ready
		serverReady := make(chan bool, 1)
		go monitorServerOutput(stdout, serverReady)
		go executeStartupCommands(stdin, startupCommands, serverReady)

		// Wait for server to finish (user stops it)
		return serverCmd.Wait()
	}

	// No startup commands, just run the server normally
	serverCmd.Stdout = os.Stdout
	serverCmd.Stderr = os.Stderr
	serverCmd.Stdin = os.Stdin

	if err := serverCmd.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return serverCmd.Wait()
}

func monitorServerOutput(stdout io.ReadCloser, serverReady chan bool) {
	scanner := bufio.NewScanner(stdout)
	sentReady := false

	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println(line) // Print to terminal

		// Detect when server is ready
		// Common patterns: "Done (X.XXXs)!" or "Server started"
		if !sentReady && (strings.Contains(line, "Done (") || strings.Contains(line, "Server started")) {
			serverReady <- true
			sentReady = true
		}
	}

	// If we never detected server ready (unexpected), send it anyway after scan ends
	if !sentReady {
		serverReady <- true
	}
}

func executeStartupCommands(stdin io.WriteCloser, commands []string, serverReady chan bool) {
	// Wait for server to be ready
	<-serverReady

	// Small additional delay to ensure server is fully initialized
	time.Sleep(2 * time.Second)

	ui.PrintInfo("Executing startup commands...")

	writer := bufio.NewWriter(stdin)
	for i, command := range commands {
		ui.PrintStep(i+1, len(commands), "%s", command)
		if _, err := fmt.Fprintf(writer, "%s\n", command); err != nil {
			ui.PrintError("Failed to execute command '%s': %v", command, err)
			continue
		}
		if err := writer.Flush(); err != nil {
			ui.PrintError("Failed to flush command '%s': %v", command, err)
			continue
		}
		time.Sleep(100 * time.Millisecond) // Small delay between commands
	}

	ui.PrintSuccess("Startup commands completed")
	fmt.Println()
}
