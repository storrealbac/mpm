package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/storrealbac/mpm/internal/models"
	"github.com/storrealbac/mpm/internal/ui"
)

var runCmd = &cobra.Command{
	Use:   "run <script-name>",
	Short: "Run a custom script defined in package.yml",
	Long: `Execute a custom script defined in the scripts section of package.yml.

Example:
  mpm run build
  mpm run test`,
	Args: cobra.ExactArgs(1),
	RunE: runScript,
}

func init() {
	rootCmd.AddCommand(runCmd)
}

func runScript(cmd *cobra.Command, args []string) error {
	scriptName := args[0]

	// Load package.yml
	pkg, err := models.LoadPackageFromFile("package.yml")
	if err != nil {
		return fmt.Errorf("failed to load package.yml: %w", err)
	}

	// Check if scripts section exists
	if pkg.Scripts == nil || len(pkg.Scripts) == 0 {
		return fmt.Errorf("no scripts defined in package.yml")
	}

	// Find the script
	scriptCommand, exists := pkg.Scripts[scriptName]
	if !exists {
		return fmt.Errorf("script '%s' not found in package.yml\n\nAvailable scripts:\n%s",
			scriptName, formatAvailableScripts(pkg.Scripts))
	}

	// Display what we're running
	ui.PrintInfo("Running script: %s", scriptName)
	fmt.Printf("  Command: %s\n\n", scriptCommand)

	// Execute the script
	return executeCommand(scriptCommand)
}

func formatAvailableScripts(scripts map[string]string) string {
	var sb strings.Builder
	for name, command := range scripts {
		sb.WriteString(fmt.Sprintf("  • %s\n    %s\n", name, command))
	}
	return sb.String()
}

func executeCommand(command string) error {
	var cmd *exec.Cmd

	// Use appropriate shell based on OS
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}

	// Set up stdout/stderr
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Run the command
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("script failed with exit code %d", exitErr.ExitCode())
		}
		return fmt.Errorf("failed to execute script: %w", err)
	}

	return nil
}
