package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/storrealbac/mpm/internal/models"
	"github.com/storrealbac/mpm/internal/ui"
	"github.com/spf13/cobra"
)

var interactive bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Creates a new package.yml file",
	Long: `Creates a package.yml template file in the current directory to define the Minecraft server's plugins/mods.

Examples:
  mpm init                    # Create with default settings
  mpm init -i                 # Create in interactive mode

Flags:
  -h, --help          help for init
  -i, --interactive   Interactive mode to configure the server`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Interactive mode to configure the server")

	// Set usage template (content moved to Long field)
	// Set usage template (content moved to Long field)
	initCmd.SetUsageTemplate(fmt.Sprintf(`%s
  {{.UseLine}}

%s
{{.Flags.FlagUsages | trimTrailingWhitespaces}}
`,
		ui.SectionStyle.Render("Usage:"),
		ui.SectionStyle.Render("Flags:"),
	))
}

func runInit(cmd *cobra.Command, args []string) error {
	filename := "package.yml"

	// Check if already exists
	if _, err := os.Stat(filename); err == nil {
		ui.PrintWarning("File %s already exists. Overwrite? (y/N): ", filename)

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response != "y" && response != "yes" {
			ui.PrintInfo("Operation cancelled.")
			return nil
		}
	}

	pkg := &models.Package{
		Name:    "my-minecraft-server",
		Version: "1.0.0",
		Server: models.ServerConfig{
			Type:             "paper",
			MinecraftVersion: "1.20.4",
			Build:            "latest",
			StartCommand:     "java -Xms3G -Xmx4G -jar server.jar nogui",
		},
		Scripts: map[string]string{
			"clean": "rm -rf logs/*.log",
		},
		StartupCommands: []string{
			"say Server started successfully!",
		},
		Plugins: []models.Plugin{},
	}

	// Interactive mode
	if interactive {
		reader := bufio.NewReader(os.Stdin)

		ui.PrintHeader("Interactive Setup")

		fmt.Printf("%s", ui.InfoStyle.Render("Project Name: "))
		name, _ := reader.ReadString('\n')
		name = strings.TrimSpace(name)
		if name != "" {
			pkg.Name = name
		}

		fmt.Printf("%s", ui.InfoStyle.Render("Project Version: "))
		version, _ := reader.ReadString('\n')
		version = strings.TrimSpace(version)
		if version != "" {
			pkg.Version = version
		}

		fmt.Printf("%s", ui.InfoStyle.Render("Server Type (paper, purpur, folia, spigot, bukkit, sponge, velocity, waterfall) [paper]: "))
		srvType, _ := reader.ReadString('\n')
		srvType = strings.TrimSpace(srvType)
		if srvType != "" {
			pkg.Server.Type = srvType
		}

		fmt.Printf("%s", ui.InfoStyle.Render(fmt.Sprintf("Minecraft Version [%s]: ", pkg.Server.MinecraftVersion)))
		mcVer, _ := reader.ReadString('\n')
		mcVer = strings.TrimSpace(mcVer)
		if mcVer != "" {
			pkg.Server.MinecraftVersion = mcVer
		}

		ui.PrintInfo("A default configuration has been created. You can edit it later in package.yml")
	}

	err := pkg.SaveToFile(filename)
	if err != nil {
		return fmt.Errorf("error creating %s: %w", filename, err)
	}

	ui.PrintSuccess("File %s created successfully!", filename)
	ui.PrintInfo("Edit it to add your plugins and then run 'mpm install'")

	return nil
}
