package pull

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ConceptResponse matches the Hub API concept structure
type ConceptResponse struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Resource    string `json:"resource"`
}

// PullBundle downloads a remote bundle into the local .okf_cache directory
func PullBundle(hubURI, host, apiKey string) error {
	// Parse hub://<bundle_id>
	if !strings.HasPrefix(hubURI, "hub://") {
		return fmt.Errorf("invalid URI format: must start with hub://")
	}

	bundleID := strings.TrimPrefix(hubURI, "hub://")
	// If it has multiple segments, assume the first segment is the bundle ID for now.
	// Actually, based on previous design, bundleID can be "namespace/bundle" like "stripe/api"
	// So we pass the entire remainder as the bundleID.

	apiURL := fmt.Sprintf("%s/api/concepts?bundle=%s", host, url.QueryEscape(bundleID))

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	fmt.Printf("Pulling OKF bundle '%s' from Hub...\n", bundleID)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("network error during pull: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("hub returned error code: %d", resp.StatusCode)
	}

	var concepts []ConceptResponse
	if err := json.NewDecoder(resp.Body).Decode(&concepts); err != nil {
		return fmt.Errorf("failed to parse hub response: %w", err)
	}

	if len(concepts) == 0 {
		fmt.Printf("Warning: Bundle '%s' pulled successfully but contains no concepts.\n", bundleID)
		return nil
	}

	// Create local directory in the current working directory
	cacheDir := bundleID
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write each concept as a markdown file
	for _, c := range concepts {
		// Frontmatter struct for yaml encoding
		fm := struct {
			Type        string `yaml:"type"`
			Title       string `yaml:"title,omitempty"`
			Description string `yaml:"description,omitempty"`
			Resource    string `yaml:"resource,omitempty"`
		}{
			Type:        c.Type,
			Title:       c.Title,
			Description: c.Description,
			Resource:    c.Resource,
		}

		yamlData, err := yaml.Marshal(fm)
		if err != nil {
			fmt.Printf("Failed to marshal frontmatter for concept %s: %v\n", c.ID, err)
			continue
		}

		content := fmt.Sprintf("---\n%s---\n\n# %s\n\n%s\n", yamlData, c.Title, c.Description)
		filePath := filepath.Join(cacheDir, c.ID+".md")

		// Create subdirectories if concept ID has slashes
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			fmt.Printf("Failed to create directory for concept %s: %v\n", c.ID, err)
			continue
		}

		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			fmt.Printf("Failed to write concept file %s: %v\n", c.ID, err)
		}
	}

	fmt.Printf("🎉 Successfully pulled bundle '%s' to %s/\n", bundleID, cacheDir)
	return nil
}
