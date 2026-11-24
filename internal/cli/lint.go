package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/abcubed3/okf/pkg/bundle"
	"github.com/abcubed3/okf/pkg/config"
	"github.com/abcubed3/okf/pkg/parser"
	"github.com/abcubed3/okf/pkg/validator"
)

// RunLint executes the linting check on a specified target bundle path.
func RunLint(args []string) error {
	bundlePath := "."
	host := "http://localhost:8080"
	apiKey := ""
	strict := false
	migrateV03 := false
	dryRun := false

	fs := flag.NewFlagSet("lint", flag.ContinueOnError)
	fs.StringVar(&host, "host", host, "OKF Hub host URL for remote resolution")
	fs.StringVar(&apiKey, "api-key", "", "OKF Hub API Key for remote resolution")
	fs.BoolVar(&strict, "strict", false, "Treat validation warnings as errors")
	fs.BoolVar(&migrateV03, "migrate-v03", false, "Migrate bundle in-place to OKF v0.3 conformance before linting")
	fs.BoolVar(&dryRun, "dry-run", false, "Preview --migrate-v03 changes without writing files to disk")
	if err := fs.Parse(args); err != nil {
		return err
	}

	apiKey = config.GetAPIKey(apiKey)

	if fs.NArg() > 0 {
		bundlePath = fs.Arg(0)
	}

	localPath, cleanup, err := parser.ResolvePath(bundlePath)
	if err != nil {
		if !parser.IsGitURL(bundlePath) && errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("failed to parse bundle: %w", err)
		}
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

	if migrateV03 {
		if parser.IsGitURL(bundlePath) && !dryRun {
			return fmt.Errorf("--migrate-v03 cannot write directly to remote Git URL %q (clone locally or pass --dry-run)", bundlePath)
		}
		report, migErr := bundle.MigrateBundleToV03(b, dryRun)
		if migErr != nil {
			return fmt.Errorf("failed to migrate bundle to OKF v0.3: %w", migErr)
		}
		if report.Updated == 0 {
			fmt.Println("Migration check: bundle already conforms to OKF v0.3 conventions (0 files modified).")
		} else {
			prefix := "Migrated"
			if dryRun {
				prefix = "[DRY RUN] Would migrate"
			}
			fmt.Printf("%s %d file(s) to OKF v0.3:\n", prefix, report.Updated)
			for _, f := range report.Files {
				for _, act := range f.Actions {
					fmt.Printf("  - %s: %s\n", f.Path, act)
				}
			}
			if !dryRun {
				b, err = parser.ParseBundle(context.Background(), absPath)
				if err != nil {
					return fmt.Errorf("failed to re-parse migrated bundle: %w", err)
				}
			}
		}
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

	// Summarize OKF v0.3 Trust Signal metrics
	verCount := 0
	staleCount := 0
	deprecatedCount := 0
	now := time.Now()

	for _, c := range b.Concepts {
		if c.IsVerified() {
			verCount++
		}
		if c.IsStale(now) {
			staleCount++
		}
		if strings.EqualFold(string(c.Frontmatter.Status), string(bundle.ConceptLifecycleDeprecated)) {
			deprecatedCount++
		}
	}

	targetVer := b.OKFVersion
	if targetVer == "" {
		targetVer = "0.3 (default)"
	}

	fmt.Println()
	if len(b.SkippedFiles) > 0 {
		fmt.Printf("Bundle Summary [OKF v%s]: %d concepts total (%d verified, %d stale, %d deprecated, %d non-concept .md skipped)\n",
			targetVer, len(b.Concepts), verCount, staleCount, deprecatedCount, len(b.SkippedFiles))
	} else {
		fmt.Printf("Bundle Summary [OKF v%s]: %d concepts total (%d verified, %d stale, %d deprecated)\n",
			targetVer, len(b.Concepts), verCount, staleCount, deprecatedCount)
	}

	if errorsCount > 0 || warningsCount > 0 {
		fmt.Printf("Validation complete: %d errors, %d warnings found.\n", errorsCount, warningsCount)
	} else {
		fmt.Println("Validation complete: OKF bundle is perfectly valid! 🎉")
	}

	if errorsCount > 0 || (strict && warningsCount > 0) {
		if strict && warningsCount > 0 && errorsCount == 0 {
			return fmt.Errorf("validation failed with %d warnings (strict mode)", warningsCount)
		}
		if strict && warningsCount > 0 {
			return fmt.Errorf("validation failed with %d errors and %d warnings (strict mode)", errorsCount, warningsCount)
		}
		return fmt.Errorf("validation failed with %d errors", errorsCount)
	}
	return nil
}
