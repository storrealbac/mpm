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
