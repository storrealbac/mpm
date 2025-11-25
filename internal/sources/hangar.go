package sources

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	HangarBaseURL = "https://hangar.papermc.io/api/v1"
)

type HangarClient struct {
	httpClient *http.Client
}

func NewHangarClient() *HangarClient {
	return &HangarClient{
		httpClient: &http.Client{},
	}
}

// SearchResponse represents the paginated response from the Hangar API
type HangarSearchResponse struct {
	Result     []HangarProject    `json:"result"`
	Pagination HangarPagination   `json:"pagination"`
}

type HangarPagination struct {
	Count  int `json:"count"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// Project represents a Hangar project
type HangarProject struct {
	Name               string              `json:"name"`
	Namespace          HangarNamespace     `json:"namespace"`
	Description        string              `json:"description"`
	Category           string              `json:"category"`
	Stats              HangarStats         `json:"stats"`
	SupportedPlatforms map[string][]string `json:"supportedPlatforms"` // Platform -> versions
}

type HangarNamespace struct {
	Owner string `json:"owner"`
	Slug  string `json:"slug"`
}

type HangarStats struct {
	Downloads int `json:"downloads"`
	Views     int `json:"views"`
	Stars     int `json:"stars"`
}

// VersionsResponse represents the paginated versions response
type HangarVersionsResponse struct {
	Result     []HangarVersion  `json:"result"`
	Pagination HangarPagination `json:"pagination"`
}

// Version represents a project version
type HangarVersion struct {
	Name                          string                            `json:"name"`
	CreatedAt                     string                            `json:"createdAt"`
	Description                   string                            `json:"description"`
	Downloads                     map[string]HangarVersionDownload  `json:"downloads"` // Platform -> download info
	PlatformDependencies          map[string][]string               `json:"platformDependencies"` // Platform -> versions
	PlatformDependenciesFormatted map[string][]string               `json:"platformDependenciesFormatted"` // Platform -> version ranges
}

type HangarVersionDownload struct {
	FileInfo    *HangarFileInfo `json:"fileInfo"`
	DownloadURL string          `json:"downloadUrl"`
	ExternalURL string          `json:"externalUrl"`
}

type HangarFileInfo struct {
	Name       string `json:"name"`
	SizeBytes  int64  `json:"sizeBytes"`
	Sha256Hash string `json:"sha256Hash"`
}

// SearchProjects searches for projects on Hangar
// query: search query string
// serverType: optional server platform to filter results (paper, velocity, waterfall)
// limit: maximum number of results to return (default 25)
func (c *HangarClient) SearchProjects(query string, serverType string, limit int) ([]HangarProject, error) {
	if limit <= 0 {
		limit = 25
	}

	// Build the URL with query parameters
	params := url.Values{}
	params.Add("q", query)
	params.Add("limit", fmt.Sprintf("%d", limit))
	params.Add("offset", "0")

	// Hangar uses platform filtering in the query parameter
	if serverType != "" {
		platform := mapServerTypeToPlatform(serverType)
		if platform != "" {
			params.Add("platform", platform)
		}
	}

	reqURL := fmt.Sprintf("%s/projects?%s", HangarBaseURL, params.Encode())

	resp, err := c.httpClient.Get(reqURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	var searchResp HangarSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, err
	}

	return searchResp.Result, nil
}

// GetProject retrieves a specific project by owner and slug
func (c *HangarClient) GetProject(owner, slug string) (*HangarProject, error) {
	reqURL := fmt.Sprintf("%s/projects/%s/%s", HangarBaseURL, owner, slug)

	resp, err := c.httpClient.Get(reqURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	var project HangarProject
	if err := json.NewDecoder(resp.Body).Decode(&project); err != nil {
		return nil, err
	}

	return &project, nil
}

// GetProjectVersions retrieves versions for a project
// owner: project owner username
// slug: project slug
// gameVersion: optional Minecraft version filter (e.g., "1.20.4") - filtered client-side
// platform: optional platform filter (e.g., "PAPER", "VELOCITY") - filtered client-side
func (c *HangarClient) GetProjectVersions(owner, slug, gameVersion, platform string) ([]HangarVersion, error) {
	params := url.Values{}
	params.Add("limit", "25")
	params.Add("offset", "0")

	reqURL := fmt.Sprintf("%s/projects/%s/%s/versions?%s", HangarBaseURL, owner, slug, params.Encode())

	resp, err := c.httpClient.Get(reqURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	var versionsResp HangarVersionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&versionsResp); err != nil {
		return nil, err
	}

	// Filter client-side by platform and game version
	filtered := make([]HangarVersion, 0)
	platformUpper := strings.ToUpper(platform)

	for _, version := range versionsResp.Result {
		// Filter by platform if specified
		if platform != "" {
			if _, ok := version.PlatformDependencies[platformUpper]; !ok {
				continue // Skip if platform not supported
			}
		}

		// Filter by game version if specified
		if gameVersion != "" && platform != "" {
			versions, ok := version.PlatformDependencies[platformUpper]
			if !ok {
				continue
			}

			// Check if gameVersion is in the list of supported versions
			found := false
			for _, v := range versions {
				if v == gameVersion {
					found = true
					break
				}
			}

			if !found {
				continue // Skip if this game version is not supported
			}
		}

		filtered = append(filtered, version)
	}

	return filtered, nil
}

// DownloadFile downloads a file from a URL and returns a reader
func (c *HangarClient) DownloadFile(downloadURL string) (io.ReadCloser, int64, error) {
	resp, err := c.httpClient.Get(downloadURL)
	if err != nil {
		return nil, 0, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, 0, fmt.Errorf("API error: %d", resp.StatusCode)
	}

	return resp.Body, resp.ContentLength, nil
}

// IsPluginCompatible checks if a plugin's platforms match the server type
func HangarIsPluginCompatible(project *HangarProject, serverType string) (compatible bool, exactMatch bool) {
	if serverType == "" {
		return true, true
	}

	platform := mapServerTypeToPlatform(serverType)
	if platform == "" {
		return false, false
	}

	// Check if the project supports this platform
	if versions, ok := project.SupportedPlatforms[platform]; ok && len(versions) > 0 {
		return true, true
	}

	// Check for compatible platforms based on server type
	switch serverType {
	case "paper":
		// Paper is compatible with Bukkit/Spigot plugins
		if _, ok := project.SupportedPlatforms["PAPER"]; ok {
			return true, true
		}
		// Some plugins might only list generic support
		return false, false

	case "velocity":
		// Velocity only works with Velocity plugins
		if _, ok := project.SupportedPlatforms["VELOCITY"]; ok {
			return true, true
		}
		return false, false

	case "waterfall":
		// Waterfall is compatible with BungeeCord plugins
		if _, ok := project.SupportedPlatforms["WATERFALL"]; ok {
			return true, true
		}
		return false, false

	case "purpur", "folia":
		// These are Paper forks, compatible with Paper plugins
		if _, ok := project.SupportedPlatforms["PAPER"]; ok {
			return true, false // Compatible but not exact match
		}
		return false, false

	case "spigot", "bukkit":
		// These can use Paper plugins in many cases
		if _, ok := project.SupportedPlatforms["PAPER"]; ok {
			return true, false
		}
		return false, false
	}

	return false, false
}

// mapServerTypeToPlatform converts mpm server types to Hangar platform names
func mapServerTypeToPlatform(serverType string) string {
	switch strings.ToLower(serverType) {
	case "paper", "purpur", "folia", "spigot", "bukkit":
		return "PAPER"
	case "velocity":
		return "VELOCITY"
	case "waterfall":
		return "WATERFALL"
	default:
		return ""
	}
}

// GetDownloadURL returns the download URL for a version based on platform
func GetDownloadURL(version *HangarVersion, serverType string) (string, string, error) {
	platform := mapServerTypeToPlatform(serverType)
	if platform == "" {
		return "", "", fmt.Errorf("unsupported server type: %s", serverType)
	}

	download, ok := version.Downloads[platform]
	if !ok {
		return "", "", fmt.Errorf("no download found for platform %s", platform)
	}

	hash := ""
	if download.FileInfo != nil {
		hash = download.FileInfo.Sha256Hash
	}

	if download.DownloadURL != "" {
		return download.DownloadURL, hash, nil
	}
	if download.ExternalURL != "" {
		return download.ExternalURL, hash, nil
	}

	return "", "", fmt.Errorf("no download URL found for platform %s", platform)
}

// GetFilename returns the filename for a version download
func GetFilename(version *HangarVersion, serverType string) string {
	platform := mapServerTypeToPlatform(serverType)

	if download, ok := version.Downloads[platform]; ok {
		if download.FileInfo != nil && download.FileInfo.Name != "" {
			return download.FileInfo.Name
		}
		// Extract filename from URL if FileInfo is null
		var url string
		if download.DownloadURL != "" {
			url = download.DownloadURL
		} else if download.ExternalURL != "" {
			url = download.ExternalURL
		}

		if url != "" {
			parts := strings.Split(url, "/")
			if len(parts) > 0 {
				filename := parts[len(parts)-1]
				if filename != "" && strings.HasSuffix(filename, ".jar") {
					return filename
				}
			}
		}
	}

	// Fallback
	return fmt.Sprintf("plugin-%s.jar", version.Name)
}
