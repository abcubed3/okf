package cli

import (
	"flag"
	"fmt"
	"strings"

	"github.com/abcubed3/okf/pkg/config"
	"github.com/abcubed3/okf/pkg/search"
)

// RunSearch handles the search CLI command
func RunSearch(args []string) error {
	query := ""
	tag := ""
	host := "http://localhost:8080"
	apiKey := ""

	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	fs.StringVar(&tag, "tag", "", "Filter search by tag")
	fs.StringVar(&host, "host", host, "OKF Hub host URL")
	fs.StringVar(&apiKey, "api-key", "", "OKF Hub API Key")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() > 0 {
		query = fs.Arg(0)
	}

	apiKey = config.GetAPIKey(apiKey)

	results, err := search.SearchBundles(query, tag, host, apiKey)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(results) == 0 {
		fmt.Println("No matching OKF knowledge bundles found.")
		return nil
	}

	fmt.Printf("🔍 Found %d matching OKF knowledge bundle(s):\n\n", len(results))
	for _, b := range results {
		privacy := "public"
		if b.IsPrivate {
			privacy = "private"
		}
		tagsStr := ""
		if len(b.Tags) > 0 {
			tagsStr = fmt.Sprintf(" [%s]", strings.Join(b.Tags, ", "))
		}
		fmt.Printf("📦 hub://%s (v%s) (%s)%s\n", b.ID, b.Version, privacy, tagsStr)
		fmt.Printf("   %s\n", b.Name)
		fmt.Printf("   %s\n\n", b.Description)
	}

	return nil
}
