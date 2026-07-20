package cli

import (
	"flag"
	"fmt"

	"github.com/abcubed3/okf/pkg/config"
	"github.com/abcubed3/okf/pkg/publish"
	"github.com/abcubed3/okf/pkg/pull"
)

// RunPublish handles the publish CLI command
func RunPublish(args []string) error {
	bundlePath := "."
	host := "http://localhost:8080"
	apiKey := ""

	fs := flag.NewFlagSet("publish", flag.ContinueOnError)
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
		return fmt.Errorf("API Key is required for publishing. Use --api-key, OKF_HUB_API_KEY, or run `okf auth login`")
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
	apiKey := ""

	fs := flag.NewFlagSet("pull", flag.ContinueOnError)
	fs.StringVar(&host, "host", host, "OKF Hub host URL")
	fs.StringVar(&apiKey, "api-key", "", "OKF Hub API Key (overrides env and config)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() > 0 {
		hubURI = fs.Arg(0)
	}

	if hubURI == "" {
		return fmt.Errorf("a hub:// URI is required. Example: okf pull hub://stripe/api")
	}

	apiKey = config.GetAPIKey(apiKey)

	if err := pull.PullBundle(hubURI, host, apiKey); err != nil {
		return fmt.Errorf("error pulling bundle: %w", err)
	}
	return nil
}
