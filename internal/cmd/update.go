package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/storrealbac/mpm/internal/models"
	"github.com/storrealbac/mpm/internal/modrinth"
	"github.com/storrealbac/mpm/internal/ui"
	"github.com/spf13/cobra"
)

var checkOnly bool

var updateCmd = &cobra.Command{
	Use:   "update [plugin...]",
	Short: "Update plugins to new versions",
	Long: `Updates plugins to their latest versions according to package.yml.
The --check option only shows which plugins need updates.`,
	RunE: runUpdate,
}

func init() {
	updateCmd.Flags().BoolVar(&checkOnly, "check", false, "Only check for updates without installing")

	// Set usage template (simplified)
	// Set usage template (simplified)
	updateCmd.SetUsageTemplate(fmt.Sprintf(`%s
  {{.UseLine}}

%s
{{.Flags.FlagUsages | trimTrailingWhitespaces}}
`,
		ui.SectionStyle.Render("Usage:"),
		ui.SectionStyle.Render("Flags:"),
	))
}

func runUpdate(cmd *cobra.Command, args []string) error {
	pkg, err := models.LoadPackageFromFile("package.yml")
	if err != nil {
		return fmt.Errorf("could not read package.yml: %w", err)
	}

	// Load package-lock.yml
	lockFile, err := models.LoadPackageLockFromFile("package-lock.yml")
	if err != nil {
		return fmt.Errorf("error loading package-lock.yml: %w", err)
	}

	client := modrinth.NewClient()
	updatesFound := false

	ui.PrintHeader("Checking for updates...")

	for i, plugin := range pkg.Plugins {
		// If arguments specified, only update those
		if len(args) > 0 {
			found := false
			for _, arg := range args {
				if strings.EqualFold(plugin.Name, arg) || strings.EqualFold(plugin.ModrinthID, arg) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		versions, err := client.GetProjectVersions(plugin.ModrinthID, pkg.Server.MinecraftVersion)
		if err != nil {
			ui.PrintError("Error getting versions for %s: %v", plugin.Name, err)
			continue
		}

		if len(versions) == 0 {
			continue
		}

		latest := versions[0]
		if latest.VersionNumber != plugin.Version {
			updatesFound = true
			ui.PrintInfo("Update available for %s: %s -> %s", plugin.Name, plugin.Version, latest.VersionNumber)

			if !checkOnly {
				// Download new version
				var fileToDownload *modrinth.File
				for _, f := range latest.Files {
					if f.Primary {
						fileToDownload = &f
						break
					}
				}
				if fileToDownload == nil && len(latest.Files) > 0 {
					fileToDownload = &latest.Files[0]
				}

				if fileToDownload != nil {
					// Delete old version (simple heuristic)
					// This is risky if we don't know the exact old filename.
					// For now, just download the new one.
					// TODO: Improve old version cleanup.

					ui.PrintInfo("Downloading %s...", latest.VersionNumber)
					// Use global pluginsDir if accessible, otherwise hardcode "plugins" or read from flag
					// Since updateCmd doesn't have dir flag, assume "plugins"
					pluginsDir := "plugins"
					destPath := filepath.Join(pluginsDir, fileToDownload.Filename)

					reader, size, err := client.DownloadFile(fileToDownload.URL)
					if err == nil {
						defer reader.Close()
						file, err := os.Create(destPath)
						if err == nil {
							defer file.Close()

							counter := &ui.WriteCounter{Total: uint64(size)}
							io.Copy(file, io.TeeReader(reader, counter))

							// Force 100% if total was unknown
							if size <= 0 {
								fmt.Printf("\r[%s] 100.00%%", strings.Repeat("=", 40))
							}
							fmt.Println()

							ui.PrintSuccess("Updated to %s", latest.VersionNumber)

							// Update model
							pkg.Plugins[i].Version = latest.VersionNumber

							// Save hash to package-lock.yml
							lockFile.Plugins[plugin.ModrinthID] = models.PluginLock{
								Name:    plugin.Name,
								Version: latest.VersionNumber,
								Hash:    fileToDownload.Hashes["sha512"],
							}
						} else {
							ui.PrintError("Error creating file: %v", err)
						}
					} else {
						ui.PrintError("Error downloading: %v", err)
					}
				}
			}
		} else {
			if len(args) > 0 {
				ui.PrintSuccess("%s is up to date (%s)", plugin.Name, plugin.Version)
			}
		}
	}

	if !updatesFound {
		ui.PrintSuccess("All plugins are up to date.")
	} else if !checkOnly {
		if err := pkg.SaveToFile("package.yml"); err != nil {
			return fmt.Errorf("error saving package.yml: %w", err)
		}
		if err := lockFile.SaveToFile("package-lock.yml"); err != nil {
			return fmt.Errorf("error saving package-lock.yml: %w", err)
		}
		ui.PrintSuccess("package.yml and package-lock.yml updated.")
	}

	return nil
}
