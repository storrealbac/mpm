package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/storrealbac/mpm/internal/ui"
)

var rootCmd = &cobra.Command{
	Use:   "mpm",
	Short: ui.MPMStyle.Render("mpm") + " - Minecraft Plugin Manager - Manage your server plugins with ease",
	Long: ui.MPMStyle.Render("mpm") + ` is a CLI tool to manage Minecraft server plugins using the Modrinth API.
It allows you to install, update, and remove plugins, as well as manage the server jar itself.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Set up clean help template
	rootCmd.SetUsageTemplate(fmt.Sprintf(`%s
  {{.UseLine}}

%s{{range .Commands}}{{if .IsAvailableCommand}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}

%s
{{.Flags.FlagUsages | trimTrailingWhitespaces}}

Use "{{.CommandPath}} [command] --help" for more information about a command.
`,
		ui.SectionStyle.Render("Usage:"),
		ui.SectionStyle.Render("Available Commands:"),
		ui.SectionStyle.Render("Flags:"),
	))

	// Use default help command
	// The custom help command was causing banner duplication

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(uninstallCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(validateCmd)
}
