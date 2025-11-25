package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/storrealbac/mpm/internal/ui"
)

// Downloader define la interfaz para descargar jars de servidor
type Downloader interface {
	Download(version, build string, outputDir string) (string, error)
}

func GetDownloader(serverType string) (Downloader, error) {
	switch strings.ToLower(serverType) {
	case "paper", "velocity", "waterfall":
		return &PaperDownloader{Project: strings.ToLower(serverType)}, nil
	case "purpur":
		return &PurpurDownloader{}, nil
	case "folia":
		return &FoliaDownloader{}, nil
	case "spigot":
		return &SpigotDownloader{}, nil
	case "bukkit":
		return &BukkitDownloader{}, nil
	case "sponge":
		return &SpongeDownloader{}, nil
	default:
		return nil, fmt.Errorf("tipo de servidor no soportado: %s", serverType)
	}
}

// --- PaperMC Implementation ---

type PaperDownloader struct {
	Project string // paper, velocity, waterfall
}

func (p *PaperDownloader) Download(version, build string, outputDir string) (string, error) {
	// 1. Si build es "latest", obtener el último build
	if build == "" || build == "latest" {
		latestBuild, err := p.getLatestBuild(version)
		if err != nil {
			return "", err
		}
		build = latestBuild
	}

	// 2. Construir URL
	// https://api.papermc.io/v2/projects/{project}/versions/{version}/builds/{build}/downloads/{download}
	fileName := fmt.Sprintf("%s-%s-%s.jar", p.Project, version, build)
	url := fmt.Sprintf("https://api.papermc.io/v2/projects/%s/versions/%s/builds/%s/downloads/%s", p.Project, version, build, fileName)

	// 3. Descargar
	return downloadFile(url, outputDir, "server.jar") // Siempre guardamos como server.jar para consistencia? O conservamos nombre?
	// El usuario pidió "descargue tambien el .jar del servidor en la carpeta".
	// Usualmente se renombra a server.jar para que los scripts de inicio no cambien.
}

func (p *PaperDownloader) getLatestBuild(version string) (string, error) {
	url := fmt.Sprintf("https://api.papermc.io/v2/projects/%s/versions/%s/builds", p.Project, version)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("error API PaperMC: %d", resp.StatusCode)
	}

	var result struct {
		Builds []struct {
			Build int `json:"build"`
		} `json:"builds"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Builds) == 0 {
		return "", fmt.Errorf("no se encontraron builds para %s %s", p.Project, version)
	}

	// El último en la lista es el más reciente
	return fmt.Sprintf("%d", result.Builds[len(result.Builds)-1].Build), nil
}

// --- Purpur Implementation ---

type PurpurDownloader struct{}

func (p *PurpurDownloader) Download(version, build string, outputDir string) (string, error) {
	if build == "" || build == "latest" {
		build = "latest"
	}

	// https://api.purpurmc.org/v2/purpur/{version}/{build}/download
	url := fmt.Sprintf("https://api.purpurmc.org/v2/purpur/%s/%s/download", version, build)

	return downloadFile(url, outputDir, "server.jar")
}

// --- Helper ---

func downloadFile(url, outputDir, fileName string) (string, error) {
	destPath := filepath.Join(outputDir, fileName)
	fmt.Printf("Downloading server from %s...\n", url)

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("download error: %d", resp.StatusCode)
	}

	if resp.ContentLength <= 0 {
		fmt.Println("Warning: Content length unknown, progress bar might not work.")
	}

	out, err := os.Create(destPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	// Progress bar
	counter := &ui.WriteCounter{Total: uint64(resp.ContentLength)}
	if _, err = io.Copy(out, io.TeeReader(resp.Body, counter)); err != nil {
		return "", err
	}

	// Force 100% if total was unknown or just to be sure
	if resp.ContentLength <= 0 {
		fmt.Printf("\r[%s] 100.00%%", strings.Repeat("=", 40))
	}
	fmt.Println() // New line after progress bar

	return destPath, nil
}

// --- Folia Implementation ---

type FoliaDownloader struct{}

func (f *FoliaDownloader) Download(version, build string, outputDir string) (string, error) {
	// Folia uses PaperMC API similar to Paper
	// 1. If build is "latest", get the latest build
	if build == "" || build == "latest" {
		latestBuild, err := f.getLatestBuild(version)
		if err != nil {
			return "", err
		}
		build = latestBuild
	}

	// 2. Build URL
	fileName := fmt.Sprintf("folia-%s-%s.jar", version, build)
	url := fmt.Sprintf("https://api.papermc.io/v2/projects/folia/versions/%s/builds/%s/downloads/%s", version, build, fileName)

	// 3. Download
	return downloadFile(url, outputDir, "server.jar")
}

func (f *FoliaDownloader) getLatestBuild(version string) (string, error) {
	url := fmt.Sprintf("https://api.papermc.io/v2/projects/folia/versions/%s/builds", version)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("error API PaperMC (Folia): %d", resp.StatusCode)
	}

	var result struct {
		Builds []struct {
			Build int `json:"build"`
		} `json:"builds"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Builds) == 0 {
		return "", fmt.Errorf("no se encontraron builds para Folia %s", version)
	}

	// Last in the list is the most recent
	return fmt.Sprintf("%d", result.Builds[len(result.Builds)-1].Build), nil
}

// --- Spigot Implementation ---

type SpigotDownloader struct{}

func (s *SpigotDownloader) Download(version, build string, outputDir string) (string, error) {
	// Spigot doesn't have an official API for downloading pre-built jars
	// Users typically need to use BuildTools
	// However, we can use the GetBukkit.org API which provides pre-built Spigot jars
	url := fmt.Sprintf("https://download.getbukkit.org/spigot/spigot-%s.jar", version)

	return downloadFile(url, outputDir, "server.jar")
}

// --- Bukkit Implementation ---

type BukkitDownloader struct{}

func (b *BukkitDownloader) Download(version, build string, outputDir string) (string, error) {
	// Bukkit/CraftBukkit can be downloaded from GetBukkit.org
	url := fmt.Sprintf("https://download.getbukkit.org/craftbukkit/craftbukkit-%s.jar", version)

	return downloadFile(url, outputDir, "server.jar")
}

// --- Sponge Implementation ---

type SpongeDownloader struct{}

func (s *SpongeDownloader) Download(version, build string, outputDir string) (string, error) {
	// SpongeVanilla or SpongeForge download
	// Sponge has different versions based on Minecraft version
	// We'll use the SpongeVanilla API
	// Note: This is a simplified implementation
	// For production, you'd want to query the Sponge API to get the correct build

	if build == "" || build == "latest" {
		// Get latest recommended build
		latestBuild, err := s.getLatestBuild(version)
		if err != nil {
			return "", err
		}
		build = latestBuild
	}

	url := fmt.Sprintf("https://repo.spongepowered.org/repository/maven-releases/org/spongepowered/spongevanilla/%s/spongevanilla-%s.jar", build, build)

	return downloadFile(url, outputDir, "server.jar")
}

func (s *SpongeDownloader) getLatestBuild(version string) (string, error) {
	// For Sponge, we need to determine the API version based on Minecraft version
	// This is a simplified mapping - in production you'd query their API
	// Common mappings:
	// MC 1.16.5 -> API 8
	// MC 1.18.2 -> API 9
	// MC 1.19.4 -> API 10
	// MC 1.20.x -> API 11

	apiVersion := "11.0.0" // Default to latest API

	// Basic version mapping
	if strings.HasPrefix(version, "1.16") {
		apiVersion = "8.2.0"
	} else if strings.HasPrefix(version, "1.18") {
		apiVersion = "9.0.0"
	} else if strings.HasPrefix(version, "1.19") {
		apiVersion = "10.0.0"
	} else if strings.HasPrefix(version, "1.20") || strings.HasPrefix(version, "1.21") {
		apiVersion = "11.0.0"
	}

	return apiVersion, nil
}
