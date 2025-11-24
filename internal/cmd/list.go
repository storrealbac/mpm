package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/storrealbac/mpm/internal/models"
	"github.com/storrealbac/mpm/internal/ui"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed plugins",
	Long: `Lists all plugins defined in package.yml and their installation status.

Examples:
  mpm list`,
	RunE: runList,
}

func init() {
	// Set usage template (removing duplication from Long field)
	// Set usage template (removing duplication from Long field)
	listCmd.SetUsageTemplate(fmt.Sprintf(`%s
  {{.UseLine}}

%s
{{.Flags.FlagUsages | trimTrailingWhitespaces}}
`,
		ui.SectionStyle.Render("Usage:"),
		ui.SectionStyle.Render("Flags:"),
	))
}

func runList(cmd *cobra.Command, args []string) error {
	pkg, err := models.LoadPackageFromFile("package.yml")
	if err != nil {
		return fmt.Errorf("could not read package.yml: %w", err)
	}

	ui.PrintHeader("Plugin List")

	// Create table
	table := ui.NewTable("NAME", "VERSION", "STATUS")

	for _, plugin := range pkg.Plugins {
		var status string
		// Check if installed using normalized name matching
		found := false
		files, _ := os.ReadDir("plugins")

		// Normalize plugin name: remove special chars, replace spaces with dashes, lowercase
		normalizedPluginName := normalizePluginName(plugin.Name)

		for _, file := range files {
			normalizedFileName := normalizePluginName(file.Name())
			// Check if filename starts with normalized plugin name
			if strings.HasPrefix(normalizedFileName, normalizedPluginName) {
				found = true
				break
			}
		}

		if found {
			status = ui.CreateStatusBadge("INSTALLED")
		} else {
			status = ui.CreateStatusBadge("MISSING")
		}

		// Add data (table handles styling internally)
		table.AddRow(plugin.Name, plugin.Version, status)
	}

	// Render the table
	fmt.Println(table.Render())
	fmt.Println()

	// Summary
	totalPlugins := len(pkg.Plugins)
	summary := fmt.Sprintf("Total plugins: %d", totalPlugins)
	ui.PrintInfo(summary)

	return nil
}
