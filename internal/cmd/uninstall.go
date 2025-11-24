package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/storrealbac/mpm/internal/models"
	"github.com/storrealbac/mpm/internal/ui"
	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use: "uninstall [plugin...]",

	Short: "Uninstall plugins",
	Long:  `Removes plugins from the plugins/ directory and package.yml.`,
	RunE:  runUninstall,
}

func init() {
	// Set usage template (simplified)
	// Set usage template (simplified)
	uninstallCmd.SetUsageTemplate(fmt.Sprintf(`%s
  {{.UseLine}}

%s
{{.Flags.FlagUsages | trimTrailingWhitespaces}}
`,
		ui.SectionStyle.Render("Usage:"),
		ui.SectionStyle.Render("Flags:"),
	))
}

func runUninstall(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("you must specify at least one plugin to uninstall")
	}

	pkg, err := models.LoadPackageFromFile("package.yml")
	if err != nil {
		return fmt.Errorf("could not read package.yml: %w", err)
	}

	pluginsDir := "plugins" // Should be configurable or read from root flag

	for _, pluginName := range args {
		// 1. Remove from package.yml
		foundIndex := -1
		for i, p := range pkg.Plugins {
			if strings.EqualFold(p.Name, pluginName) || strings.EqualFold(p.ModrinthID, pluginName) {
				foundIndex = i
				break
			}
		}

		if foundIndex == -1 {
			ui.PrintWarning("Plugin '%s' not found in package.yml", pluginName)
			continue
		}

		// 2. Remove file
		// Heuristic: Search for files starting with the name in plugins/
		// This is risky, ideally we should store the filename in package.yml
		files, err := os.ReadDir(pluginsDir)
		if err == nil {
			for _, file := range files {
				if !file.IsDir() && strings.HasSuffix(file.Name(), ".jar") {
					// Simple heuristic: if filename starts with plugin name (case insensitive)
					if strings.HasPrefix(strings.ToLower(file.Name()), strings.ToLower(pluginName)) {
						fullPath := filepath.Join(pluginsDir, file.Name())
						if err := os.Remove(fullPath); err != nil {
							ui.PrintError("Error deleting %s: %v", file.Name(), err)
						} else {
							ui.PrintSuccess("Deleted file: %s", file.Name())
						}
					}
				}
			}
		}

		// Remove from package.yml
		pkg.Plugins = append(pkg.Plugins[:foundIndex], pkg.Plugins[foundIndex+1:]...)
		ui.PrintSuccess("Removed from package.yml: %s", pluginName)
	}

	if err := pkg.SaveToFile("package.yml"); err != nil {
		return fmt.Errorf("error saving package.yml: %w", err)
	}

	return nil
}
