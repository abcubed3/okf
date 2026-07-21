package cli

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"
	"strings"

	"github.com/abcubed3/okf/pkg/config"
	"github.com/abcubed3/okf/pkg/parser"
	"github.com/abcubed3/okf/pkg/publish"
)

// RunPush handles the push CLI command for iterative/incremental updates
func RunPush(args []string) error {
	bundlePath := "."
	host := "http://localhost:8080"
	apiKey := ""

	fs := flag.NewFlagSet("push", flag.ContinueOnError)
	fs.StringVar(&host, "host", host, "OKF Hub host URL")
	fs.StringVar(&apiKey, "api-key", "", "OKF Hub API Key (overrides env and config)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() > 0 {
		bundlePath = fs.Arg(0)
	}

	apiKey = config.GetAPIKey(apiKey)
	if apiKey == "" {
		return fmt.Errorf("API Key is required for pushing. Use --api-key, OKF_HUB_API_KEY, or run `okf auth login`")
	}

	// Resolve bundle
	localPath, cleanup, err := parser.ResolvePath(bundlePath)
	if err != nil {
		return fmt.Errorf("failed to resolve bundle path: %w", err)
	}
	defer cleanup()

	absPath, err := filepath.Abs(localPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	fmt.Printf("Pushing incremental OKF bundle at: %s to %s...\n", absPath, host)

	// Parse local bundle
	b, err := parser.ParseBundle(context.Background(), absPath)
	if err != nil {
		return fmt.Errorf("failed to parse bundle: %w", err)
	}

	bundleName := filepath.Base(b.Path)

	// For an incremental push, we construct a patch payload
	// In a full implementation, we would diff against the remote bundle.
	// For this prototype, we send the parsed concepts to the /api/bundles/push endpoint as a patch.
	payload := publish.PublishPayload{
		Bundle: publish.BundlePayload{
			ID:          bundleName,
			Name:        bundleName,
			Description: "Incremental push for: " + bundleName,
			Concepts:    make([]publish.PublishConcept, 0, len(b.Concepts)),
		},
	}

	for id, c := range b.Concepts {
		payload.Bundle.Concepts = append(payload.Bundle.Concepts, publish.PublishConcept{
			ID:          id,
			Type:        c.Frontmatter.Type,
			Title:       c.Frontmatter.Title,
			Description: c.Frontmatter.Desc,
			Resource:    c.Frontmatter.Resource,
		})
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to serialize push payload: %w", err)
	}

	url := fmt.Sprintf("%s/api/bundles/push", host)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("network error during push: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		if len(body) > 0 {
			return fmt.Errorf("server returned error %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
		}
		return fmt.Errorf("server returned error code: %d", resp.StatusCode)
	}

	fmt.Println("🚀 Successfully pushed incremental updates to the OKF Hub!")
	return nil
}
