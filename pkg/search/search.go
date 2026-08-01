package search

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type BundleResult struct {
	ID          string   `json:"id"`
	URN         string   `json:"urn,omitempty"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Namespace   string   `json:"namespace"`
	IsPrivate   bool     `json:"is_private"`
	Version     string   `json:"version,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// SearchBundles queries the OKF Hub for bundles matching query and optional tag filters.
func SearchBundles(query, tag, host, apiKey string) ([]BundleResult, error) {
	return SearchBundlesWithClient(query, tag, host, apiKey, &http.Client{Timeout: 10 * time.Second})
}

// SearchBundlesWithClient allows querying using a custom http.Client.
func SearchBundlesWithClient(query, tag, host, apiKey string, client *http.Client) ([]BundleResult, error) {
	if host == "" {
		host = os.Getenv("OKF_HUB_HOST")
		if host == "" {
			host = "http://localhost:8080"
		}
	}
	host = strings.TrimSuffix(host, "/")

	params := url.Values{}
	if query != "" {
		params.Set("q", query)
	}
	if tag != "" {
		params.Set("tag", tag)
	}

	apiURL := fmt.Sprintf("%s/api/bundles?%s", host, params.Encode())

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error during search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hub returned status code: %d", resp.StatusCode)
	}

	var bundles []BundleResult
	if err := json.NewDecoder(resp.Body).Decode(&bundles); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	return bundles, nil
}
