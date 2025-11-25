package sources

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const (
	ModrinthBaseURL = "https://api.modrinth.com/v2"
)

type ModrinthClient struct {
	httpClient *http.Client
}

func NewModrinthClient() *ModrinthClient {
	return &ModrinthClient{
		httpClient: &http.Client{},
	}
}

type ModrinthSearchResponse struct {
	Hits      []ModrinthProject `json:"hits"`
	Offset    int       `json:"offset"`
	Limit     int       `json:"limit"`
	TotalHits int       `json:"total_hits"`
}

type ModrinthProject struct {
	Slug        string   `json:"slug"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Categories  []string `json:"categories"`
	ClientSide  string   `json:"client_side"`
	ServerSide  string   `json:"server_side"`
	ProjectID   string   `json:"project_id"`
	Author      string   `json:"author"`
	Versions    []string `json:"versions"`
}

type ModrinthVersion struct {
	ID            string   `json:"id"`
	ProjectID     string   `json:"project_id"`
	AuthorID      string   `json:"author_id"`
	Name          string   `json:"name"`
	VersionNumber string   `json:"version_number"`
	GameVersions  []string `json:"game_versions"`
	Loaders       []string `json:"loaders"`
	Files         []ModrinthFile   `json:"files"`
}

type ModrinthFile struct {
	Hashes   map[string]string `json:"hashes"`
	URL      string            `json:"url"`
	Filename string            `json:"filename"`
	Primary  bool              `json:"primary"`
	Size     int               `json:"size"`
}

// SearchProjects busca proyectos en Modrinth
// serverType: optional server platform to filter results (paper, folia, velocity, etc.)
// strict: if true, only return plugins that exactly match the server type
func (c *ModrinthClient) SearchProjects(query string, serverType string, strict bool) ([]ModrinthProject, error) {
	// Build facets based on server type and strict mode
	facets := buildSearchFacets(serverType, strict)

	encodedQuery := url.QueryEscape(query)
	encodedFacets := url.QueryEscape(facets)
	url := fmt.Sprintf("%s/search?query=%s&facets=%s", ModrinthBaseURL, encodedQuery, encodedFacets)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %d", resp.StatusCode)
	}

	var searchResp ModrinthSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, err
	}

	return searchResp.Hits, nil
}

// buildSearchFacets creates Modrinth API facets based on server type
func buildSearchFacets(serverType string, strict bool) string {
	// Map server types to appropriate Modrinth categories
	// Reference: https://docs.modrinth.com/api/operations/searchprojects/

	switch serverType {
	case "velocity":
		if strict {
			return `[["categories:velocity"]]`
		}
		// Velocity proxies can only use velocity plugins
		return `[["categories:velocity"]]`

	case "waterfall":
		if strict {
			return `[["categories:bungeecord"]]`
		}
		// Waterfall is BungeeCord compatible
		return `[["categories:bungeecord"]]`

	case "folia":
		if strict {
			// Only Folia-specific plugins
			return `[["categories:folia"]]`
		}
		// Fallback: Folia plugins or general Bukkit API plugins (with warning)
		return `[["categories:folia", "categories:paper", "categories:purpur", "categories:spigot", "categories:bukkit"]]`

	case "paper":
		if strict {
			return `[["categories:paper"]]`
		}
		// Paper is compatible with Paper, Spigot, and Bukkit plugins
		return `[["categories:paper", "categories:spigot", "categories:bukkit"]]`

	case "purpur":
		if strict {
			return `[["categories:purpur"]]`
		}
		// Purpur is Paper fork, compatible with Paper, Spigot, and Bukkit
		return `[["categories:purpur", "categories:paper", "categories:spigot", "categories:bukkit"]]`

	case "spigot":
		if strict {
			return `[["categories:spigot"]]`
		}
		// Spigot is compatible with Spigot and Bukkit plugins
		return `[["categories:spigot", "categories:bukkit"]]`

	case "bukkit":
		if strict {
			return `[["categories:bukkit"]]`
		}
		// Bukkit is the base API
		return `[["categories:bukkit"]]`

	case "sponge":
		if strict {
			return `[["categories:sponge"]]`
		}
		// Sponge has its own plugin API, not compatible with others
		return `[["categories:sponge"]]`

	default:
		// If no server type specified, include all server-side plugins
		return `[["categories:bukkit", "categories:folia", "categories:paper", "categories:purpur", "categories:spigot", "categories:sponge", "categories:velocity", "categories:bungeecord"]]`
	}
}

// IsPluginCompatible checks if a plugin's categories match the server type
func ModrinthIsPluginCompatible(project *ModrinthProject, serverType string) (compatible bool, exactMatch bool) {
	if serverType == "" {
		return true, true
	}

	categories := project.Categories

	// Check for exact category match
	for _, cat := range categories {
		if cat == serverType {
			return true, true
		}
	}

	// Check compatibility based on server type
	switch serverType {
	case "folia":
		// Folia needs explicit Folia support
		// Paper/Spigot/Bukkit plugins may not work due to threading changes
		for _, cat := range categories {
			if cat == "folia" {
				return true, true
			}
		}
		// Other categories are not compatible
		return false, false

	case "paper":
		// Paper is compatible with Paper, Spigot, and Bukkit
		for _, cat := range categories {
			if cat == "paper" {
				return true, true
			}
			if cat == "spigot" || cat == "bukkit" {
				return true, false
			}
		}

	case "purpur":
		// Purpur is Paper fork
		for _, cat := range categories {
			if cat == "purpur" {
				return true, true
			}
			if cat == "paper" || cat == "spigot" || cat == "bukkit" {
				return true, false
			}
		}

	case "spigot":
		// Spigot is compatible with Spigot and Bukkit
		for _, cat := range categories {
			if cat == "spigot" {
				return true, true
			}
			if cat == "bukkit" {
				return true, false
			}
		}

	case "bukkit":
		// Bukkit is the base
		for _, cat := range categories {
			if cat == "bukkit" {
				return true, true
			}
		}

	case "velocity":
		// Velocity plugins only
		for _, cat := range categories {
			if cat == "velocity" {
				return true, true
			}
		}

	case "waterfall":
		// Waterfall is BungeeCord compatible
		for _, cat := range categories {
			if cat == "bungeecord" {
				return true, true
			}
		}

	case "sponge":
		// Sponge plugins only
		for _, cat := range categories {
			if cat == "sponge" {
				return true, true
			}
		}
	}

	return false, false
}

// GetProject retrieves project information by ID or slug
func (c *ModrinthClient) GetProject(idOrSlug string) (*ModrinthProject, error) {
	reqUrl := fmt.Sprintf("%s/project/%s", ModrinthBaseURL, idOrSlug)

	resp, err := c.httpClient.Get(reqUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %d", resp.StatusCode)
	}

	var project ModrinthProject
	if err := json.NewDecoder(resp.Body).Decode(&project); err != nil {
		return nil, err
	}

	return &project, nil
}

// GetProjectVersions obtiene las versiones de un proyecto, opcionalmente filtrando por versión de juego
func (c *ModrinthClient) GetProjectVersions(idOrSlug string, gameVersion string) ([]ModrinthVersion, error) {
	reqUrl := fmt.Sprintf("%s/project/%s/version", ModrinthBaseURL, idOrSlug)
	if gameVersion != "" {
		// game_versions=["1.20.1"]
		encodedVersion := fmt.Sprintf(`["%s"]`, gameVersion)
		reqUrl = fmt.Sprintf("%s?game_versions=%s", reqUrl, url.QueryEscape(encodedVersion))
	}

	resp, err := c.httpClient.Get(reqUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %d", resp.StatusCode)
	}

	var versions []ModrinthVersion
	if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
		return nil, err
	}

	return versions, nil
}

// DownloadFile descarga un archivo desde una URL
// DownloadFile downloads a file from a URL and returns a reader.
// It now returns the body directly, the caller is responsible for wrapping it with a progress bar if needed,
// OR we can handle it here. To keep it simple and reusable, let's return the size too.
func (c *ModrinthClient) DownloadFile(url string) (io.ReadCloser, int64, error) {
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, 0, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, 0, fmt.Errorf("API error: %d", resp.StatusCode)
	}

	return resp.Body, resp.ContentLength, nil
}
