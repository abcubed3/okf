package cli

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"

	"github.com/abcubed3/okf/pkg/config"
	"github.com/abcubed3/okf/pkg/parser"
	"github.com/abcubed3/okf/pkg/validator"
)

// RunLint executes the linting check on a specified target bundle path.
func RunLint(args []string) error {
	bundlePath := "."
	host := "http://localhost:8080"
	apiKey := ""

	fs := flag.NewFlagSet("lint", flag.ContinueOnError)
	fs.StringVar(&host, "host", host, "OKF Hub host URL for remote resolution")
	fs.StringVar(&apiKey, "api-key", "", "OKF Hub API Key for remote resolution")
	if err := fs.Parse(args); err != nil {
		return err
	}

	apiKey = config.GetAPIKey(apiKey)

	if fs.NArg() > 0 {
		bundlePath = fs.Arg(0)
	}

	localPath, cleanup, err := parser.ResolvePath(bundlePath)
	if err != nil {
		return fmt.Errorf("failed to resolve bundle path: %w", err)
	}
	defer cleanup()

	absPath, err := filepath.Abs(localPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	fmt.Printf("Linting OKF bundle at: %s...\n", absPath)

	b, err := parser.ParseBundle(context.Background(), absPath)
	if err != nil {
		return fmt.Errorf("failed to parse bundle: %w", err)
	}

	opts := validator.Options{
		Host:   host,
		APIKey: apiKey,
	}
	issues := validator.ValidateBundle(b, opts)

	errorsCount := 0
	warningsCount := 0

	for _, issue := range issues {
		severityPrefix := ""
		if issue.Severity == validator.SeverityError {
			severityPrefix = "[ERROR]"
			errorsCount++
		} else {
			severityPrefix = "[WARN] "
			warningsCount++
		}

		fmt.Printf("  %s %s: %s\n", severityPrefix, issue.Path, issue.Message)
	}

	fmt.Println()
	if errorsCount > 0 || warningsCount > 0 {
		fmt.Printf("Validation complete: %d errors, %d warnings found.\n", errorsCount, warningsCount)
	} else {
		fmt.Println("Validation complete: OKF bundle is perfectly valid! 🎉")
	}

	if errorsCount > 0 {
		return fmt.Errorf("validation failed with %d errors", errorsCount)
	}
	return nil
}
