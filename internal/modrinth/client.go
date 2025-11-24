package modrinth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const (
	BaseURL = "https://api.modrinth.com/v2"
)

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{},
	}
}

type SearchResponse struct {
	Hits      []Project `json:"hits"`
	Offset    int       `json:"offset"`
	Limit     int       `json:"limit"`
	TotalHits int       `json:"total_hits"`
}

type Project struct {
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

type Version struct {
	ID            string   `json:"id"`
	ProjectID     string   `json:"project_id"`
	AuthorID      string   `json:"author_id"`
	Name          string   `json:"name"`
	VersionNumber string   `json:"version_number"`
	GameVersions  []string `json:"game_versions"`
	Loaders       []string `json:"loaders"`
	Files         []File   `json:"files"`
}

type File struct {
	Hashes   map[string]string `json:"hashes"`
	URL      string            `json:"url"`
	Filename string            `json:"filename"`
	Primary  bool              `json:"primary"`
	Size     int               `json:"size"`
}

// SearchProjects busca proyectos en Modrinth
func (c *Client) SearchProjects(query string) ([]Project, error) {
	// Facets para filtrar por categorías de servidor (Bukkit, Spigot, Paper, etc.)
	// Documentación: https://docs.modrinth.com/api/operations/searchprojects/
	// Usamos OR logic para incluir cualquiera de estas categorías.
	facets := `[["categories:bukkit", "categories:folia", "categories:paper", "categories:purpur", "categories:spigot", "categories:sponge"]]`
	encodedQuery := url.QueryEscape(query)
	encodedFacets := url.QueryEscape(facets)
	url := fmt.Sprintf("%s/search?query=%s&facets=%s", BaseURL, encodedQuery, encodedFacets)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %d", resp.StatusCode)
	}

	var searchResp SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, err
	}

	return searchResp.Hits, nil
}

// GetProjectVersions obtiene las versiones de un proyecto, opcionalmente filtrando por versión de juego
func (c *Client) GetProjectVersions(idOrSlug string, gameVersion string) ([]Version, error) {
	reqUrl := fmt.Sprintf("%s/project/%s/version", BaseURL, idOrSlug)
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

	var versions []Version
	if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
		return nil, err
	}

	return versions, nil
}

// DownloadFile descarga un archivo desde una URL
// DownloadFile downloads a file from a URL and returns a reader.
// It now returns the body directly, the caller is responsible for wrapping it with a progress bar if needed,
// OR we can handle it here. To keep it simple and reusable, let's return the size too.
func (c *Client) DownloadFile(url string) (io.ReadCloser, int64, error) {
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
