package publish

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/abcubed3/okf/pkg/parser"
)

// PublishPayload matches the protobuf structure for the wire format
type PublishPayload struct {
	Bundle BundlePayload `json:"bundle"`
}

type BundlePayload struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Concepts    []PublishConcept `json:"concepts"`
}

type PublishConcept struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Resource    string `json:"resource"`
}

// PublishBundle validates, packages, and pushes a bundle to the OKF Hub.
func PublishBundle(bundlePath, host, apiKey string) error {
	localPath, cleanup, err := parser.ResolvePath(bundlePath)
	if err != nil {
		return fmt.Errorf("failed to resolve bundle path: %w", err)
	}
	defer cleanup()

	absPath, err := filepath.Abs(localPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	fmt.Printf("Publishing OKF bundle at: %s to %s...\n", absPath, host)

	// 1. Parse the bundle locally
	b, err := parser.ParseBundle(context.Background(), absPath)
	if err != nil {
		return fmt.Errorf("failed to parse bundle: %w", err)
	}

	bundleName := filepath.Base(b.Path)

	// 2. Map to PublishPayload (matches proto structure)
	payload := PublishPayload{
		Bundle: BundlePayload{
			ID:          bundleName, // Using the bundle name as the ID in the prototype
			Name:        bundleName,
			Description: "Published OKF Bundle: " + bundleName,
			Concepts:    make([]PublishConcept, 0, len(b.Concepts)),
		},
	}

	for id, c := range b.Concepts {
		payload.Bundle.Concepts = append(payload.Bundle.Concepts, PublishConcept{
			ID:          id,
			Type:        c.Frontmatter.Type,
			Title:       c.Frontmatter.Title,
			Description: c.Frontmatter.Desc,
			Resource:    c.Frontmatter.Resource,
		})
	}

	// 3. Serialize payload
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to serialize payload: %w", err)
	}

	// 4. Send HTTP POST with a 30-second timeout to prevent infinite hangs.
	url := fmt.Sprintf("%s/api/bundles", host)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("network error during publish: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		if len(body) > 0 {
			return fmt.Errorf("server returned error %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
		}
		return fmt.Errorf("server returned error code: %d", resp.StatusCode)
	}

	fmt.Println("🎉 Successfully published bundle to the OKF Hub!")
	return nil
}
