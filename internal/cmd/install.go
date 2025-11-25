package cmd

import (
	"bufio"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/storrealbac/mpm/internal/models"
	"github.com/storrealbac/mpm/internal/modrinth"
	"github.com/storrealbac/mpm/internal/server"
	"github.com/storrealbac/mpm/internal/ui"
	"github.com/spf13/cobra"
)

var (
	pluginsDir string
	force      bool
)

var installCmd = &cobra.Command{
	Use:   "install [plugin...]",
	Short: "Install plugins from Modrinth",
	Long: `Install plugins defined in package.yml or specified as arguments from Modrinth.
If arguments are specified, searches for and downloads the latest compatible version and adds it to package.yml.`,
	RunE: runInstall,
}

func init() {
	installCmd.Flags().StringVar(&pluginsDir, "dir", "plugins", "Directory where plugins will be saved")
	installCmd.Flags().BoolVar(&force, "force", false, "Force re-download if already exists")

	// Set usage template (simplified)
	// Set usage template (simplified)
	installCmd.SetUsageTemplate(fmt.Sprintf(`%s
  {{.UseLine}}

%s
{{.Flags.FlagUsages | trimTrailingWhitespaces}}
`,
		ui.SectionStyle.Render("Usage:"),
		ui.SectionStyle.Render("Flags:"),
	))
}

func runInstall(cmd *cobra.Command, args []string) error {
	// Create plugins directory
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		return fmt.Errorf("could not create directory %s: %w", pluginsDir, err)
	}

	client := modrinth.NewClient()

	// If arguments provided, install specific plugins
	if len(args) > 0 {
		// When installing specific plugins, we don't check/download the server jar
		// We just need the server version for compatibility checking
		pkg, err := models.LoadPackageFromFile("package.yml")
		var serverVersion string
		if err == nil {
			serverVersion = pkg.Server.MinecraftVersion
		}
		return installSpecificPlugins(client, args, serverVersion)
	}

	// If no arguments, install from package.yml (full install)
	// This includes verifying the server jar
	pkg, err := models.LoadPackageFromFile("package.yml")
	var serverVersion string
	if err == nil {
		serverVersion = pkg.Server.MinecraftVersion
		if pkg.Server.Type != "" {
			ui.PrintInfo("Verifying server %s %s...", pkg.Server.Type, pkg.Server.MinecraftVersion)

			shouldDownload := true
			if !force {
				if _, err := os.Stat("server.jar"); err == nil {
					ui.PrintSuccess("Server already exists (server.jar)")
					shouldDownload = false
				}
			}

			if shouldDownload {
				downloader, err := server.GetDownloader(pkg.Server.Type)
				if err != nil {
					ui.PrintWarning("Could not get downloader for %s: %v", pkg.Server.Type, err)
				} else {
					// Use current directory for server.jar
					_, err := downloader.Download(pkg.Server.MinecraftVersion, pkg.Server.Build, ".")
					if err != nil {
						ui.PrintError("Error downloading server: %v", err)
					} else {
						ui.PrintSuccess("Server ready (server.jar)")
					}
				}
			}
		}
	}

	return installFromPackage(client, serverVersion)
}

func installSpecificPlugins(client *modrinth.Client, plugins []string, serverVersion string) error {
	pkg, err := models.LoadPackageFromFile("package.yml")
	if err != nil {
		// If not exists, fail and ask for init
		return fmt.Errorf("package.yml not found, run 'mpm init' first")
	}

	// Load package-lock.yml
	lockFile, err := models.LoadPackageLockFromFile("package-lock.yml")
	if err != nil {
		return fmt.Errorf("error loading package-lock.yml: %w", err)
	}

	serverType := strings.ToLower(pkg.Server.Type)

	for _, query := range plugins {
		ui.PrintInfo("Searching '%s' on Modrinth for %s server...", query, pkg.Server.Type)

		// Try strict search first (exact platform match)
		projects, err := client.SearchProjects(query, serverType, true)
		if err != nil {
			return fmt.Errorf("error searching %s: %w", query, err)
		}

		// If no strict results and server type is set, try broader search
		if len(projects) == 0 && serverType != "" {
			ui.PrintWarning("No %s-specific plugins found for '%s'", pkg.Server.Type, query)
			ui.PrintInfo("Searching for compatible alternatives...")
			projects, err = client.SearchProjects(query, serverType, false)
			if err != nil {
				return fmt.Errorf("error searching %s: %w", query, err)
			}
		}

		if len(projects) == 0 {
			ui.PrintWarning("No results found for '%s'", query)
			continue
		}

		// Search for exact slug match first
		var project *modrinth.Project
		for i := range projects {
			if strings.EqualFold(projects[i].Slug, query) {
				project = &projects[i]
				break
			}
		}

		// If no exact match, show suggestions
		if project == nil {
			ui.PrintWarning("No exact match found for '%s'.", query)
			fmt.Println("Did you mean:")

			limit := 3
			if len(projects) < limit {
				limit = len(projects)
			}

			for i := 0; i < limit; i++ {
				p := projects[i]
				fmt.Printf("   %d. %s (%s) - %s\n", i+1, p.Title, p.Slug, p.Description)
			}

			fmt.Print("\nSelect a number (1-3) or press Enter to cancel: ")
			reader := bufio.NewReader(os.Stdin)
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)

			var choice int
			if _, err := fmt.Sscanf(input, "%d", &choice); err == nil && choice >= 1 && choice <= limit {
				project = &projects[choice-1]
			} else {
				ui.PrintInfo("Operation cancelled.")
				continue
			}
		}
		ui.PrintSuccess("Found: %s (%s)", project.Title, project.Slug)

		// Check plugin compatibility with server type
		if serverType != "" {
			compatible, exactMatch := modrinth.IsPluginCompatible(project, serverType)

			if !compatible {
				ui.PrintWarning("WARNING: '%s' is NOT compatible with %s server!", project.Title, pkg.Server.Type)
				ui.PrintError("This plugin will likely not work. Installation cancelled.")
				continue
			}

			if !exactMatch {
				ui.PrintWarning("WARNING: '%s' is not specifically designed for %s!", project.Title, pkg.Server.Type)
				fmt.Printf("Plugin categories: %v\n", project.Categories)

				if serverType == "folia" {
					ui.PrintWarning("Folia has significant threading changes. Paper/Spigot plugins may crash or corrupt data!")
				}

				fmt.Print("\nThis plugin may not work correctly. Do you want to continue? (y/N): ")
				reader := bufio.NewReader(os.Stdin)
				response, _ := reader.ReadString('\n')
				response = strings.TrimSpace(strings.ToLower(response))

				if response != "y" && response != "yes" {
					ui.PrintInfo("Installation cancelled.")
					continue
				}
			}
		}

		// Get versions
		// Try to filter by server version if exists
		versions, err := client.GetProjectVersions(project.Slug, serverVersion)
		if err != nil {
			ui.PrintError("Error getting versions for %s: %v", project.Title, err)
			continue
		}

		if len(versions) == 0 {
			if serverVersion != "" {
				ui.PrintWarning("No versions found compatible with Minecraft %s for %s.", serverVersion, project.Title)
				ui.PrintInfo("Searching for any available version...")

				// Try searching without filter
				allVersions, err := client.GetProjectVersions(project.Slug, "")
				if err == nil && len(allVersions) > 0 {
					versions = allVersions
					ui.PrintWarning("Installing latest available version (%s). It might not be compatible!", versions[0].VersionNumber)
				} else {
					ui.PrintError("No versions available for %s", project.Title)
					continue
				}
			} else {
				ui.PrintError("No versions available for %s", project.Title)
				continue
			}
		}

		// Take latest version (usually first in list)
		latestVersion := versions[0]

		// Find primary file
		var fileToDownload *modrinth.File
		for _, f := range latestVersion.Files {
			if f.Primary {
				fileToDownload = &f
				break
			}
		}
		if fileToDownload == nil && len(latestVersion.Files) > 0 {
			fileToDownload = &latestVersion.Files[0]
		}

		if fileToDownload == nil {
			ui.PrintError("No downloadable file found for version %s", latestVersion.VersionNumber)
			continue
		}

		// Download
		if err := downloadFile(client, fileToDownload.URL, fileToDownload.Filename, pluginsDir, fileToDownload.Hashes["sha512"], -1); err != nil {
			ui.PrintError("Error downloading %s: %v", fileToDownload.Filename, err)
			continue
		}

		// Add to package.yml
		newPlugin := models.Plugin{
			Name:       project.Title,
			Version:    latestVersion.VersionNumber,
			ModrinthID: project.Slug, // Use slug as stable ID
		}

		// Check if exists and update
		found := false
		for i, p := range pkg.Plugins {
			if p.ModrinthID == newPlugin.ModrinthID {
				pkg.Plugins[i] = newPlugin
				found = true
				break
			}
		}
		if !found {
			pkg.Plugins = append(pkg.Plugins, newPlugin)
		}

		// Save hash to package-lock.yml
		lockFile.Plugins[project.Slug] = models.PluginLock{
			Name:    project.Title,
			Version: latestVersion.VersionNumber,
			Hash:    fileToDownload.Hashes["sha512"],
		}

		ui.PrintSuccess("Installed %s %s", newPlugin.Name, newPlugin.Version)
	}

	// Save updated package.yml
	if err := pkg.SaveToFile("package.yml"); err != nil {
		return fmt.Errorf("error saving package.yml: %w", err)
	}

	// Save package-lock.yml
	if err := lockFile.SaveToFile("package-lock.yml"); err != nil {
		return fmt.Errorf("error saving package-lock.yml: %w", err)
	}

	return nil
}

func installFromPackage(client *modrinth.Client, serverVersion string) error {
	pkg, err := models.LoadPackageFromFile("package.yml")
	if err != nil {
		return fmt.Errorf("could not read package.yml: %w", err)
	}

	// Load package-lock.yml
	lockFile, err := models.LoadPackageLockFromFile("package-lock.yml")
	if err != nil {
		return fmt.Errorf("error loading package-lock.yml: %w", err)
	}

	ui.PrintHeader("Installing %d plugins for server '%s'...", len(pkg.Plugins), pkg.Name)

	// Prepare download tasks
	type downloadTask struct {
		plugin         models.Plugin
		targetVersion  *modrinth.Version
		fileToDownload *modrinth.File
		index          int
	}

	var tasks []downloadTask

	// Fetch metadata for all plugins first
	for i, plugin := range pkg.Plugins {
		ui.PrintStep(i+1, len(pkg.Plugins), "Checking: %s (ID: %s)", plugin.Name, plugin.ModrinthID)

		// Get project versions
		versions, err := client.GetProjectVersions(plugin.ModrinthID, serverVersion)
		if err != nil {
			ui.PrintError("Error getting info: %v", err)
			continue
		}

		if len(versions) == 0 {
			if serverVersion != "" {
				ui.PrintWarning("No versions found compatible with Minecraft %s for %s.", serverVersion, plugin.Name)
				ui.PrintInfo("Searching for any available version...")

				allVersions, err := client.GetProjectVersions(plugin.ModrinthID, "")
				if err == nil && len(allVersions) > 0 {
					versions = allVersions
					ui.PrintWarning("Installing latest available version (%s). It might not be compatible!", versions[0].VersionNumber)
				} else {
					ui.PrintError("No versions available for %s", plugin.Name)
					continue
				}
			} else {
				ui.PrintError("No versions available for %s", plugin.Name)
				continue
			}
		}

		// Find specific version if required, or latest
		var targetVersion *modrinth.Version
		if plugin.Version != "" && plugin.Version != "latest" {
			for _, v := range versions {
				if v.VersionNumber == plugin.Version {
					targetVersion = &v
					break
				}
			}
		} else if len(versions) > 0 {
			targetVersion = &versions[0]
		}

		if targetVersion == nil {
			ui.PrintError("Version %s not found for %s", plugin.Version, plugin.Name)
			continue
		}

		// Find file
		var fileToDownload *modrinth.File
		for _, f := range targetVersion.Files {
			if f.Primary {
				fileToDownload = &f
				break
			}
		}
		if fileToDownload == nil && len(targetVersion.Files) > 0 {
			fileToDownload = &targetVersion.Files[0]
		}

		if fileToDownload == nil {
			ui.PrintError("No downloadable files for %s", plugin.Name)
			continue
		}

		tasks = append(tasks, downloadTask{
			plugin:         plugin,
			targetVersion:  targetVersion,
			fileToDownload: fileToDownload,
			index:          i,
		})
	}

	// Download with concurrency limit of 5
	fmt.Println()
	ui.PrintInfo("Downloading %d plugins (5 concurrent downloads)...", len(tasks))
	fmt.Println()

	// Initialize multi-bar progress
	ui.InitMultiBar()

	// Create progress bars for all tasks
	taskBars := make(map[int]int) // task index -> bar ID
	for i, task := range tasks {
		barID := ui.AddBar(task.plugin.Name, 0) // Size will be set during download
		taskBars[i] = barID
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5) // Limit to 5 concurrent downloads
	var mutex sync.Mutex
	errors := make([]error, 0)

	for i, task := range tasks {
		wg.Add(1)
		go func(t downloadTask, taskIdx int) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			barID := taskBars[taskIdx]

			// Download
			err := downloadFile(client, t.fileToDownload.URL, t.fileToDownload.Filename, pluginsDir, t.fileToDownload.Hashes["sha512"], barID)
			if err != nil {
				mutex.Lock()
				errors = append(errors, fmt.Errorf("error downloading %s: %v", t.plugin.Name, err))
				mutex.Unlock()
				return
			}

			// Save to lock file
			mutex.Lock()
			lockFile.Plugins[t.plugin.ModrinthID] = models.PluginLock{
				Name:    t.plugin.Name,
				Version: t.targetVersion.VersionNumber,
				Hash:    t.fileToDownload.Hashes["sha512"],
			}
			mutex.Unlock()
		}(task, i)
	}

	wg.Wait()
	ui.CloseMultiBar()

	// Print success messages
	fmt.Println()
	for _, task := range tasks {
		ui.PrintSuccess("Installed: %s v%s", task.plugin.Name, task.targetVersion.VersionNumber)
	}

	// Report errors
	if len(errors) > 0 {
		fmt.Println()
		ui.PrintWarning("Some downloads failed:")
		for _, err := range errors {
			ui.PrintError("%v", err)
		}
	}

	// Save package-lock.yml
	if err := lockFile.SaveToFile("package-lock.yml"); err != nil {
		return fmt.Errorf("error saving package-lock.yml: %w", err)
	}

	return nil
}

func downloadFile(client *modrinth.Client, url, filename, destDir, expectedHash string, progressBarID int) error {
	destPath := filepath.Join(destDir, filename)

	if !force {
		if _, err := os.Stat(destPath); err == nil {
			if progressBarID >= 0 {
				ui.SetBarTotal(progressBarID, 1) // Set total to 1 for 100% progress
				ui.UpdateBar(progressBarID, 1) // Set progress to 100%
				ui.FinishBar(progressBarID)
			}
			return nil
		}
	}

	reader, size, err := client.DownloadFile(url)
	if err != nil {
		return err
	}
	defer reader.Close()

	// Update progress bar total if we know the size
	if progressBarID >= 0 && size > 0 {
		ui.SetBarTotal(progressBarID, uint64(size))
	}

	// Create temporary file
	tmpFile, err := os.CreateTemp(destDir, "mpm-download-*.tmp")
	if err != nil {
		return fmt.Errorf("could not create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name()) // Clean up temp file on error/exit
	defer tmpFile.Close()

	// Hash calculation
	hasher := sha512.New()

	// Progress tracking
	var counter io.Writer
	if progressBarID >= 0 {
		// Update multi-bar progress
		counter = &MultiBarWriter{
			BarID: progressBarID,
			Total: uint64(size),
		}
	} else {
		// Fallback to old style
		counter = &ui.WriteCounter{Total: uint64(size)}
	}

	multiWriter := io.MultiWriter(tmpFile, hasher, counter)

	if _, err = io.Copy(multiWriter, reader); err != nil {
		return err
	}

	// Mark as finished
	if progressBarID >= 0 {
		ui.FinishBar(progressBarID)
	} else {
		// Force 100% if total was unknown (old style)
		if size <= 0 {
			fmt.Printf("\r[%s] 100.00%%", strings.Repeat("=", 40))
		}
		fmt.Println() // New line
	}

	// Verify Checksum
	if expectedHash != "" {
		calculatedHash := hex.EncodeToString(hasher.Sum(nil))
		if !strings.EqualFold(calculatedHash, expectedHash) {
			return fmt.Errorf("checksum mismatch for %s:\nExpected: %s\nActual:   %s", filename, expectedHash, calculatedHash)
		}
	}

	// Close temp file before moving
	tmpFile.Close()

	// Move temp file to destination
	if err := os.Rename(tmpFile.Name(), destPath); err != nil {
		// Fallback copy if rename fails (e.g. cross-device)
		src, err := os.Open(tmpFile.Name())
		if err != nil {
			return err
		}
		defer src.Close()

		dst, err := os.Create(destPath)
		if err != nil {
			return err
		}
		defer dst.Close()

		if _, err := io.Copy(dst, src); err != nil {
			return err
		}
	}

	return nil
}

// MultiBarWriter updates a specific progress bar
type MultiBarWriter struct {
	BarID   int
	Total   uint64
	Written uint64
}

func (w *MultiBarWriter) Write(p []byte) (int, error) {
	n := len(p)
	w.Written += uint64(n)
	ui.UpdateBar(w.BarID, w.Written)
	return n, nil
}
