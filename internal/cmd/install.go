package cmd

import (
	"bufio"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/storrealbac/mpm/internal/models"
	"github.com/storrealbac/mpm/internal/server"
	"github.com/storrealbac/mpm/internal/sources"
	"github.com/storrealbac/mpm/internal/ui"
	"github.com/storrealbac/mpm/internal/utils"
	"github.com/spf13/cobra"
)

var (
	pluginsDir    string
	force         bool
	pluginSource  string // "modrinth", "hangar", or "auto" (default)
)

var installCmd = &cobra.Command{
	Use:   "install [plugin...]",
	Short: "Install plugins from Modrinth or Hangar",
	Long: `Install plugins defined in package.yml or specified as arguments from Modrinth or Hangar.
If arguments are specified, searches for and downloads the latest compatible version and adds it to package.yml.
Use --source flag to specify the plugin source (modrinth, hangar, or auto).`,
	RunE: runInstall,
}

func init() {
	installCmd.Flags().StringVar(&pluginsDir, "dir", "plugins", "Directory where plugins will be saved")
	installCmd.Flags().BoolVar(&force, "force", false, "Force re-download if already exists")
	installCmd.Flags().StringVar(&pluginSource, "source", "auto", "Plugin source: modrinth, hangar, or auto (searches both)")

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

	modrinthClient := sources.NewModrinthClient()
	hangarClient := sources.NewHangarClient()

	// If arguments provided, install specific plugins
	if len(args) > 0 {
		// When installing specific plugins, we don't check/download the server jar
		// We just need the server version for compatibility checking
		pkg, err := models.LoadPackageFromFile("package.yml")
		var serverVersion string
		var serverType string
		if err == nil {
			serverVersion = pkg.Server.MinecraftVersion
			serverType = pkg.Server.Type
		}
		return installSpecificPlugins(modrinthClient, hangarClient, args, serverVersion, serverType)
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

	return installFromPackage(modrinthClient, hangarClient, serverVersion, pkg.Server.Type)
}

type pluginSearchResult struct {
	name       string
	id         string
	source     string // "modrinth" or "hangar"
	distance   int
	desc       string
}

func installSpecificPlugins(modrinthClient *sources.ModrinthClient, hangarClient *sources.HangarClient, plugins []string, serverVersion string, serverType string) error {
	pkg, err := models.LoadPackageFromFile("package.yml")
	if err != nil {
		return fmt.Errorf("package.yml not found, run 'mpm init' first")
	}

	lockFile, err := models.LoadPackageLockFromFile("package-lock.yml")
	if err != nil {
		return fmt.Errorf("error loading package-lock.yml: %w", err)
	}

	serverType = strings.ToLower(serverType)

	for _, query := range plugins {
		// Search both APIs based on source flag
		searchModrinth := pluginSource == "auto" || pluginSource == "modrinth"
		searchHangar := pluginSource == "auto" || pluginSource == "hangar"

		var results []pluginSearchResult

		// Search Hangar
		if searchHangar {
			hangarProjects, err := hangarClient.SearchProjects(query, serverType, 25)
			if err == nil {
				for _, p := range hangarProjects {
					fullSlug := fmt.Sprintf("%s/%s", p.Namespace.Owner, p.Namespace.Slug)
					distance := utils.LevenshteinDistance(query, p.Name)
					results = append(results, pluginSearchResult{
						name:     p.Name,
						id:       fullSlug,
						source:   "hangar",
						distance: distance,
						desc:     p.Description,
					})
				}
			}
		}

		// Search Modrinth
		if searchModrinth {
			modrinthProjects, err := modrinthClient.SearchProjects(query, serverType, false)
			if err == nil {
				for _, p := range modrinthProjects {
					distance := utils.LevenshteinDistance(query, p.Title)
					results = append(results, pluginSearchResult{
						name:     p.Title,
						id:       p.Slug,
						source:   "modrinth",
						distance: distance,
						desc:     p.Description,
					})
				}
			}
		}

		if len(results) == 0 {
			ui.PrintWarning("No results found for '%s'", query)
			continue
		}

		// Sort by edit distance (lower is better)
		sort.Slice(results, func(i, j int) bool {
			return results[i].distance < results[j].distance
		})

		// Check for exact match (distance 0)
		var selected *pluginSearchResult
		if results[0].distance == 0 {
			selected = &results[0]
			ui.PrintSuccess("Found: %s (%s) from %s", selected.name, selected.id, selected.source)
		} else {
			// Show top 6 results by edit distance
			fmt.Println("Did you mean:")

			displayResults := results
			if len(displayResults) > 6 {
				displayResults = displayResults[:6]
			}

			for i, r := range displayResults {
				fmt.Printf("   %d. [%s] %s (%s) - %s\n", i+1, strings.ToUpper(r.source), r.name, r.id, r.desc)
			}

			fmt.Printf("\nSelect a number (1-%d) or press Enter to cancel: ", len(displayResults))
			reader := bufio.NewReader(os.Stdin)
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)

			var choice int
			if _, err := fmt.Sscanf(input, "%d", &choice); err == nil && choice >= 1 && choice <= len(displayResults) {
				selected = &displayResults[choice-1]
			} else {
				ui.PrintInfo("Operation cancelled.")
				continue
			}
		}

		// Install the selected plugin
		if selected.source == "hangar" {
			if err := installFromHangar(hangarClient, selected.id, serverVersion, serverType, pkg, lockFile); err != nil {
				ui.PrintError("Failed to install '%s': %v", selected.name, err)
				continue
			}
		} else {
			if err := installFromModrinth(modrinthClient, selected.id, serverVersion, serverType, pkg, lockFile); err != nil {
				ui.PrintError("Failed to install '%s': %v", selected.name, err)
				continue
			}
		}
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

func installFromModrinth(client *sources.ModrinthClient, pluginID, serverVersion, serverType string, pkg *models.Package, lockFile *models.PackageLock) error {
	// Get project info first to get the correct name
	project, err := client.GetProject(pluginID)
	if err != nil {
		return fmt.Errorf("error getting project: %v", err)
	}

	// Get versions
	versions, err := client.GetProjectVersions(pluginID, serverVersion)
	if err != nil {
		return fmt.Errorf("error getting versions: %v", err)
	}

	if len(versions) == 0 {
		ui.PrintWarning("No compatible versions found for platform '%s'", serverType)

		// Try to find versions from alternative platforms
		alternativeVersions := findAlternativeModrinthVersions(client, pluginID, serverVersion, serverType)
		if len(alternativeVersions) == 0 {
			return fmt.Errorf("no versions found for any platform")
		}

		// Let user choose from alternatives
		selectedVersion := promptAlternativeVersionSelection(alternativeVersions)
		if selectedVersion == nil {
			return fmt.Errorf("installation cancelled by user")
		}

		versions = []sources.ModrinthVersion{*selectedVersion}
	}

	// Take latest version
	latestVersion := versions[0]

	// Find primary file
	var fileToDownload *sources.ModrinthFile
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
		return fmt.Errorf("no downloadable file found")
	}

	// Download
	if err := downloadFileModrinth(client, fileToDownload.URL, fileToDownload.Filename, pluginsDir, fileToDownload.Hashes["sha512"], -1); err != nil {
		return fmt.Errorf("error downloading: %v", err)
	}

	// Add to package.yml
	newPlugin := models.Plugin{
		Name:       project.Title,
		Version:    latestVersion.VersionNumber,
		ModrinthID: pluginID,
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
	lockFile.Plugins[pluginID] = models.PluginLock{
		Name:    newPlugin.Name,
		Version: latestVersion.VersionNumber,
		Hash:    fileToDownload.Hashes["sha512"],
	}

	ui.PrintSuccess("Installed %s %s from Modrinth", newPlugin.Name, newPlugin.Version)
	return nil
}

func installFromHangar(client *sources.HangarClient, pluginID, serverVersion, serverType string, pkg *models.Package, lockFile *models.PackageLock) error {
	// Parse owner/slug
	parts := strings.SplitN(pluginID, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid Hangar ID format")
	}
	owner, slug := parts[0], parts[1]

	// Get project to get the plugin name
	project, err := client.GetProject(owner, slug)
	if err != nil {
		return fmt.Errorf("error getting project: %v", err)
	}

	// Get versions
	versions, err := client.GetProjectVersions(owner, slug, serverVersion, serverType)
	if err != nil {
		return fmt.Errorf("error getting versions: %v", err)
	}

	if len(versions) == 0 {
		ui.PrintWarning("No compatible versions found for platform '%s'", serverType)

		// Try to find versions from alternative platforms
		alternativeVersions := findAlternativeHangarVersions(client, owner, slug, serverVersion, serverType, project.SupportedPlatforms)
		if len(alternativeVersions) == 0 {
			return fmt.Errorf("no versions found for any platform")
		}

		// Let user choose from alternatives
		selectedVersion := promptAlternativeHangarVersionSelection(alternativeVersions)
		if selectedVersion == nil {
			return fmt.Errorf("installation cancelled by user")
		}

		versions = []sources.HangarVersion{selectedVersion.version}
		serverType = selectedVersion.platform // Update serverType to match the selected platform
	}

	// Take latest version
	latestVersion := versions[0]

	// Get download URL and hash
	downloadURL, hash, err := sources.GetDownloadURL(&latestVersion, serverType)
	if err != nil {
		return fmt.Errorf("error getting download URL: %v", err)
	}

	// Get filename
	filename := sources.GetFilename(&latestVersion, serverType)

	// Download
	if err := downloadFileHangar(client, downloadURL, filename, pluginsDir, hash, -1); err != nil {
		return fmt.Errorf("error downloading: %v", err)
	}

	// Add to package.yml
	newPlugin := models.Plugin{
		Name:     project.Name,
		Version:  latestVersion.Name,
		HangarID: pluginID,
	}

	// Check if exists and update
	found := false
	for i, p := range pkg.Plugins {
		if p.HangarID == newPlugin.HangarID {
			pkg.Plugins[i] = newPlugin
			found = true
			break
		}
	}
	if !found {
		pkg.Plugins = append(pkg.Plugins, newPlugin)
	}

	// Save hash to package-lock.yml
	lockFile.Plugins[pluginID] = models.PluginLock{
		Name:    newPlugin.Name,
		Version: latestVersion.Name,
		Hash:    hash,
	}

	ui.PrintSuccess("Installed %s %s from Hangar", newPlugin.Name, newPlugin.Version)
	return nil
}

func installFromPackage(modrinthClient *sources.ModrinthClient, hangarClient *sources.HangarClient, serverVersion string, serverType string) error {
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
		source         string // "modrinth" or "hangar"
		// Modrinth fields
		targetVersion  *sources.ModrinthVersion
		fileToDownload *sources.ModrinthFile
		// Hangar fields
		hangarVersion   *sources.HangarVersion
		hangarURL       string
		hangarHash      string
		hangarFilename  string
		index          int
	}

	var tasks []downloadTask

	// Fetch metadata for all plugins first
	for i, plugin := range pkg.Plugins {
		// Determine plugin source
		if plugin.ModrinthID != "" {
			ui.PrintStep(i+1, len(pkg.Plugins), "Checking: %s (Modrinth: %s)", plugin.Name, plugin.ModrinthID)

			// Get project versions
			versions, err := modrinthClient.GetProjectVersions(plugin.ModrinthID, serverVersion)
			if err != nil {
				ui.PrintError("Error getting info: %v", err)
				continue
			}

			if len(versions) == 0 {
				ui.PrintWarning("No compatible versions found for platform '%s'", serverType)

				// Try to find versions from alternative platforms
				alternativeVersions := findAlternativeModrinthVersions(modrinthClient, plugin.ModrinthID, serverVersion, serverType)
				if len(alternativeVersions) > 0 {
					// For batch install, automatically select the first alternative
					selectedAlt := alternativeVersions[0]
					ui.PrintInfo("Using version %s from platform %s for %s", selectedAlt.version.VersionNumber, strings.ToUpper(selectedAlt.platform), plugin.Name)
					versions = []sources.ModrinthVersion{selectedAlt.version}
				} else {
					ui.PrintError("No versions available for %s on any platform", plugin.Name)
					continue
				}
			}

			// Find specific version if required, or latest
			var targetVersion *sources.ModrinthVersion
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
			var fileToDownload *sources.ModrinthFile
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
				source:         "modrinth",
				targetVersion:  targetVersion,
				fileToDownload: fileToDownload,
				index:          i,
			})
		} else if plugin.HangarID != "" {
			ui.PrintStep(i+1, len(pkg.Plugins), "Checking: %s (Hangar: %s)", plugin.Name, plugin.HangarID)

			// Parse owner/slug from HangarID
			parts := strings.SplitN(plugin.HangarID, "/", 2)
			if len(parts) != 2 {
				ui.PrintError("Invalid Hangar ID format for %s (expected owner/slug)", plugin.Name)
				continue
			}
			owner, slug := parts[0], parts[1]

			// Get project versions
			versions, err := hangarClient.GetProjectVersions(owner, slug, serverVersion, serverType)
			if err != nil {
				ui.PrintError("Error getting info: %v", err)
				continue
			}

			if len(versions) == 0 {
				ui.PrintWarning("No compatible versions found for platform '%s'", serverType)

				// Get project to access supported platforms
				project, err := hangarClient.GetProject(owner, slug)
				if err != nil {
					ui.PrintError("Error getting project info for %s: %v", plugin.Name, err)
					continue
				}

				// Try to find versions from alternative platforms
				alternativeVersions := findAlternativeHangarVersions(hangarClient, owner, slug, serverVersion, serverType, project.SupportedPlatforms)
				if len(alternativeVersions) > 0 {
					// For batch install, automatically select the first alternative
					selectedAlt := alternativeVersions[0]
					ui.PrintInfo("Using version %s from platform %s for %s", selectedAlt.version.Name, strings.ToUpper(selectedAlt.platform), plugin.Name)
					versions = []sources.HangarVersion{selectedAlt.version}
					serverType = selectedAlt.platform // Update serverType for this plugin
				} else {
					ui.PrintError("No versions available for %s on any platform", plugin.Name)
					continue
				}
			}

			// Find specific version if required, or latest
			var targetVersion *sources.HangarVersion
			if plugin.Version != "" && plugin.Version != "latest" {
				for _, v := range versions {
					if v.Name == plugin.Version {
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

			// Get download URL
			downloadURL, hash, err := sources.GetDownloadURL(targetVersion, serverType)
			if err != nil {
				ui.PrintError("No downloadable files for %s: %v", plugin.Name, err)
				continue
			}

			filename := sources.GetFilename(targetVersion, serverType)

			tasks = append(tasks, downloadTask{
				plugin:         plugin,
				source:         "hangar",
				hangarVersion:  targetVersion,
				hangarURL:      downloadURL,
				hangarHash:     hash,
				hangarFilename: filename,
				index:          i,
			})
		} else {
			ui.PrintWarning("Plugin %s has no Modrinth or Hangar ID, skipping", plugin.Name)
		}
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

			var err error
			var lockKey string
			var version string
			var hash string

			// Download based on source
			if t.source == "modrinth" {
				err = downloadFileModrinth(modrinthClient, t.fileToDownload.URL, t.fileToDownload.Filename, pluginsDir, t.fileToDownload.Hashes["sha512"], barID)
				lockKey = t.plugin.ModrinthID
				version = t.targetVersion.VersionNumber
				hash = t.fileToDownload.Hashes["sha512"]
			} else if t.source == "hangar" {
				err = downloadFileHangar(hangarClient, t.hangarURL, t.hangarFilename, pluginsDir, t.hangarHash, barID)
				lockKey = t.plugin.HangarID
				version = t.hangarVersion.Name
				hash = t.hangarHash
			}

			if err != nil {
				mutex.Lock()
				errors = append(errors, fmt.Errorf("error downloading %s: %v", t.plugin.Name, err))
				mutex.Unlock()
				return
			}

			// Save to lock file
			mutex.Lock()
			lockFile.Plugins[lockKey] = models.PluginLock{
				Name:    t.plugin.Name,
				Version: version,
				Hash:    hash,
			}
			mutex.Unlock()
		}(task, i)
	}

	wg.Wait()
	ui.CloseMultiBar()

	// Print success messages
	fmt.Println()
	for _, task := range tasks {
		if task.source == "modrinth" {
			ui.PrintSuccess("Installed: %s v%s (Modrinth)", task.plugin.Name, task.targetVersion.VersionNumber)
		} else if task.source == "hangar" {
			ui.PrintSuccess("Installed: %s v%s (Hangar)", task.plugin.Name, task.hangarVersion.Name)
		}
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

func downloadFileModrinth(client *sources.ModrinthClient, url, filename, destDir, expectedHash string, progressBarID int) error {
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

func downloadFileHangar(client *sources.HangarClient, url, filename, destDir, expectedHash string, progressBarID int) error {
	destPath := filepath.Join(destDir, filename)

	if !force {
		if _, err := os.Stat(destPath); err == nil {
			if progressBarID >= 0 {
				ui.SetBarTotal(progressBarID, 1) // Set total to 1 for 100% progress
				ui.UpdateBar(progressBarID, 1)   // Set progress to 100%
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

	// Hash calculation (Hangar uses SHA256)
	hasher := sha256.New()

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

// alternativeVersionInfo holds information about an alternative platform version
type alternativeVersionInfo struct {
	version  sources.ModrinthVersion
	platform string
}

// findAlternativeModrinthVersions searches for versions from alternative platforms
func findAlternativeModrinthVersions(client *sources.ModrinthClient, pluginID, serverVersion, currentPlatform string) []alternativeVersionInfo {
	alternatives := []alternativeVersionInfo{}

	// Define platform search order based on current platform
	var platformsToTry []string
	switch strings.ToLower(currentPlatform) {
	case "paper", "purpur", "folia":
		platformsToTry = []string{"paper", "spigot", "bukkit", "purpur", "folia"}
	case "spigot":
		platformsToTry = []string{"spigot", "bukkit", "paper"}
	case "bukkit":
		platformsToTry = []string{"bukkit", "spigot", "paper"}
	case "velocity":
		platformsToTry = []string{"velocity"}
	case "waterfall":
		platformsToTry = []string{"bungeecord"}
	case "sponge":
		platformsToTry = []string{"sponge"}
	default:
		platformsToTry = []string{"paper", "spigot", "bukkit", "velocity", "bungeecord", "sponge"}
	}

	// Try each platform
	for _, platform := range platformsToTry {
		if strings.ToLower(platform) == strings.ToLower(currentPlatform) {
			continue // Skip current platform as we already checked it
		}

		// Get versions without filtering by game version first
		allVersions, err := client.GetProjectVersions(pluginID, "")
		if err != nil {
			continue
		}

		// Filter versions by platform manually
		for _, version := range allVersions {
			// Check if version supports this platform
			hasCompatibleLoader := false
			for _, loader := range version.Loaders {
				if strings.EqualFold(loader, platform) {
					hasCompatibleLoader = true
					break
				}
			}

			if !hasCompatibleLoader {
				continue
			}

			// Check if version supports the server version (if specified)
			if serverVersion != "" {
				hasGameVersion := false
				for _, gv := range version.GameVersions {
					if gv == serverVersion {
						hasGameVersion = true
						break
					}
				}
				if !hasGameVersion {
					continue
				}
			}

			alternatives = append(alternatives, alternativeVersionInfo{
				version:  version,
				platform: platform,
			})
		}

		// If we found alternatives, stop searching
		if len(alternatives) > 0 {
			break
		}
	}

	return alternatives
}

// promptAlternativeVersionSelection shows alternatives and lets user choose
func promptAlternativeVersionSelection(alternatives []alternativeVersionInfo) *sources.ModrinthVersion {
	ui.PrintInfo("Found versions from alternative platforms:")
	fmt.Println()

	// Group by platform and show max 3 per platform
	platformGroups := make(map[string][]alternativeVersionInfo)
	for _, alt := range alternatives {
		platformGroups[alt.platform] = append(platformGroups[alt.platform], alt)
	}

	options := []alternativeVersionInfo{}
	idx := 1
	for platform, versions := range platformGroups {
		fmt.Printf("  Platform: %s\n", strings.ToUpper(platform))
		limit := len(versions)
		if limit > 3 {
			limit = 3
		}
		for i := 0; i < limit; i++ {
			v := versions[i]
			fmt.Printf("    %d. %s (v%s) - Game versions: %s\n", idx, v.version.Name, v.version.VersionNumber, strings.Join(v.version.GameVersions, ", "))
			options = append(options, v)
			idx++
		}
		fmt.Println()
	}

	fmt.Printf("Select a version (1-%d) or press Enter to cancel: ", len(options))
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	var choice int
	if _, err := fmt.Sscanf(input, "%d", &choice); err == nil && choice >= 1 && choice <= len(options) {
		selected := options[choice-1]
		ui.PrintSuccess("Selected: %s from platform %s", selected.version.Name, strings.ToUpper(selected.platform))
		return &selected.version
	}

	return nil
}

// hangarAlternativeVersionInfo holds information about an alternative Hangar platform version
type hangarAlternativeVersionInfo struct {
	version  sources.HangarVersion
	platform string
}

// findAlternativeHangarVersions searches for versions from alternative Hangar platforms
func findAlternativeHangarVersions(client *sources.HangarClient, owner, slug, serverVersion, currentPlatform string, supportedPlatforms map[string][]string) []hangarAlternativeVersionInfo {
	alternatives := []hangarAlternativeVersionInfo{}

	// Get available platforms from supportedPlatforms
	availablePlatforms := []string{}
	for platform := range supportedPlatforms {
		if !strings.EqualFold(platform, currentPlatform) {
			availablePlatforms = append(availablePlatforms, strings.ToLower(platform))
		}
	}

	if len(availablePlatforms) == 0 {
		return alternatives
	}

	ui.PrintInfo("Searching in alternative platforms: %s", strings.Join(availablePlatforms, ", "))

	// Try each available platform
	for _, platform := range availablePlatforms {
		versions, err := client.GetProjectVersions(owner, slug, serverVersion, platform)
		if err != nil {
			continue
		}

		for _, version := range versions {
			alternatives = append(alternatives, hangarAlternativeVersionInfo{
				version:  version,
				platform: platform,
			})
		}

		// If we found alternatives, stop searching
		if len(alternatives) > 0 {
			break
		}
	}

	return alternatives
}

// promptAlternativeHangarVersionSelection shows Hangar alternatives and lets user choose
func promptAlternativeHangarVersionSelection(alternatives []hangarAlternativeVersionInfo) *hangarAlternativeVersionInfo {
	ui.PrintInfo("Found versions from alternative platforms:")
	fmt.Println()

	// Group by platform and show max 3 per platform
	platformGroups := make(map[string][]hangarAlternativeVersionInfo)
	for _, alt := range alternatives {
		platformGroups[alt.platform] = append(platformGroups[alt.platform], alt)
	}

	options := []hangarAlternativeVersionInfo{}
	idx := 1
	for platform, versions := range platformGroups {
		fmt.Printf("  Platform: %s\n", strings.ToUpper(platform))
		limit := len(versions)
		if limit > 3 {
			limit = 3
		}
		for i := 0; i < limit; i++ {
			v := versions[i]
			gameVersions := []string{}
			if deps, ok := v.version.PlatformDependencies[strings.ToUpper(platform)]; ok {
				gameVersions = deps
			}
			fmt.Printf("    %d. %s - Game versions: %s\n", idx, v.version.Name, strings.Join(gameVersions, ", "))
			options = append(options, v)
			idx++
		}
		fmt.Println()
	}

	fmt.Printf("Select a version (1-%d) or press Enter to cancel: ", len(options))
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	var choice int
	if _, err := fmt.Sscanf(input, "%d", &choice); err == nil && choice >= 1 && choice <= len(options) {
		selected := options[choice-1]
		ui.PrintSuccess("Selected: %s from platform %s", selected.version.Name, strings.ToUpper(selected.platform))
		return &selected
	}

	return nil
}
