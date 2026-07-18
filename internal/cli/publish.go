package cli

import (
	"flag"
	"fmt"
	"os"

	"github.com/abcubed3/okf/pkg/publish"
	"github.com/abcubed3/okf/pkg/pull"
)

// RunPublish handles the publish CLI command
func RunPublish(args []string) error {
	bundlePath := "."
	host := "http://localhost:8080"
	apiKey := os.Getenv("OKF_HUB_API_KEY")

	fs := flag.NewFlagSet("publish", flag.ContinueOnError)
	fs.StringVar(&host, "host", host, "OKF Hub host URL")
	fs.StringVar(&apiKey, "api-key", apiKey, "OKF Hub API Key (or set OKF_HUB_API_KEY)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() > 0 {
		bundlePath = fs.Arg(0)
	}

	if apiKey == "" {
		return fmt.Errorf("API Key is required for publishing. Use --api-key or set OKF_HUB_API_KEY environment variable")
	}

	if err := publish.PublishBundle(bundlePath, host, apiKey); err != nil {
		return fmt.Errorf("error publishing bundle: %w", err)
	}
	return nil
}

// RunPull handles the pull CLI command
func RunPull(args []string) error {
	hubURI := ""
	host := "http://localhost:8080"
	apiKey := os.Getenv("OKF_HUB_API_KEY")

	fs := flag.NewFlagSet("pull", flag.ContinueOnError)
	fs.StringVar(&host, "host", host, "OKF Hub host URL")
	fs.StringVar(&apiKey, "api-key", apiKey, "OKF Hub API Key (or set OKF_HUB_API_KEY)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() > 0 {
		hubURI = fs.Arg(0)
	}

	if hubURI == "" {
		return fmt.Errorf("a hub:// URI is required. Example: okf pull hub://stripe/api")
	}

	if err := pull.PullBundle(hubURI, host, apiKey); err != nil {
		return fmt.Errorf("error pulling bundle: %w", err)
	}
	return nil
}
