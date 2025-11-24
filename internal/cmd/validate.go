package cmd

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/storrealbac/mpm/internal/models"
	"github.com/storrealbac/mpm/internal/ui"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate installation",
	Long:  `Checks if all plugins defined in package.yml are present in the plugins/ directory.`,
	RunE:  runValidate,
}

func init() {
	// Set usage template (simplified)
	// Set usage template (simplified)
	validateCmd.SetUsageTemplate(fmt.Sprintf(`%s
  {{.UseLine}}

%s
{{.Flags.FlagUsages | trimTrailingWhitespaces}}
`,
		ui.SectionStyle.Render("Usage:"),
		ui.SectionStyle.Render("Flags:"),
	))
}

func runValidate(cmd *cobra.Command, args []string) error {
	pkg, err := models.LoadPackageFromFile("package.yml")
	if err != nil {
		return fmt.Errorf("could not read package.yml: %w", err)
	}

	// Load package-lock.yml
	lockFile, err := models.LoadPackageLockFromFile("package-lock.yml")
	if err != nil {
		return fmt.Errorf("error loading package-lock.yml: %w", err)
	}

	pluginsDir := "plugins"
	missingCount := 0
	installedCount := 0
	invalidCount := 0

	ui.PrintHeader("Validation Report")

	// Create table
	table := ui.NewTable("PLUGIN", "STATUS", "DETAILS")

	for _, plugin := range pkg.Plugins {
		found := false
		var matchedFile os.DirEntry
		files, _ := os.ReadDir(pluginsDir)

		// Normalize plugin name: remove special chars, replace spaces with dashes, lowercase
		normalizedPluginName := normalizePluginName(plugin.Name)

		for _, file := range files {
			normalizedFileName := normalizePluginName(file.Name())
			// Check if filename starts with normalized plugin name
			if strings.HasPrefix(normalizedFileName, normalizedPluginName) {
				found = true
				matchedFile = file
				break
			}
		}

		var status, details string
		if !found {
			status = ui.CreateStatusBadge("MISSING")
			details = fmt.Sprintf("v%s required", plugin.Version)
			missingCount++
		} else {
			// Validate Checksum if available in lock file
			if pluginLock, exists := lockFile.Plugins[plugin.ModrinthID]; exists && pluginLock.Hash != "" {
				fullPath := filepath.Join(pluginsDir, matchedFile.Name())
				valid, err := validateChecksum(fullPath, pluginLock.Hash)
				if err != nil {
					status = ui.CreateStatusBadge("ERROR")
					details = fmt.Sprintf("Error reading file: %v", err)
					invalidCount++
				} else if !valid {
					status = ui.CreateStatusBadge("INVALID")
					details = "Checksum mismatch"
					invalidCount++
				} else {
					status = ui.CreateStatusBadge("OK")
					details = "Verified (SHA512)"
					installedCount++
				}
			} else {
				status = ui.CreateStatusBadge("OK")
				details = "Installed (No hash)"
				installedCount++
			}
		}

		table.AddRow(plugin.Name, status, details)
	}

	// Render the table
	fmt.Println(table.Render())
	fmt.Println()

	// Progress bar
	totalPlugins := len(pkg.Plugins)
	progressBar := ui.CreateProgressBar(installedCount, totalPlugins, 15)
	fmt.Printf("%s\n\n", progressBar)

	// Summary
	if missingCount > 0 || invalidCount > 0 {
		if missingCount > 0 {
			ui.PrintWarning("Validation failed: %d plugins missing.", missingCount)
		}
		if invalidCount > 0 {
			ui.PrintError("Validation failed: %d plugins have invalid checksums.", invalidCount)
		}
		ui.PrintInfo("Run 'mpm install' to fix.")
		return fmt.Errorf("validation failed")
	}

	ui.PrintSuccess("Validation successful: All %d plugins installed and verified.", totalPlugins)
	return nil
}

func validateChecksum(filePath, expectedHash string) (bool, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	hasher := sha512.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return false, err
	}

	calculatedHash := hex.EncodeToString(hasher.Sum(nil))
	return strings.EqualFold(calculatedHash, expectedHash), nil
}
