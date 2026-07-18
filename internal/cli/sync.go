package cli

import (
	"flag"
	"fmt"
	"path/filepath"

	"github.com/abcubed3/okf/pkg/parser"
	okfsync "github.com/abcubed3/okf/pkg/sync"
)

// RunSync handles the 'sync' subcommand to start the bi-directional knowledge sync daemon.
func RunSync(args []string) error {
	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	bundlePath := fs.String("bundle", ".", "Path to the OKF knowledge bundle directory")
	configPath := fs.String("config", "okf.yaml", "Path to the sync configuration file")
	daemon := fs.Bool("daemon", false, "Run continuously as a daemon")
	interval := fs.Int("interval", 300, "Sync interval in seconds (when running as daemon)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	localPath, cleanup, err := parser.ResolvePath(*bundlePath)
	if err != nil {
		return fmt.Errorf("failed to resolve bundle path: %w", err)
	}
	defer cleanup()

	absBundlePath, err := filepath.Abs(localPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path for bundle: %w", err)
	}

	fmt.Printf("Starting OKF Sync Engine for bundle at %s (Config: %s)\n", absBundlePath, *configPath)
	if err := okfsync.Run(absBundlePath, *configPath, *daemon, *interval); err != nil {
		return fmt.Errorf("sync engine failed: %w", err)
	}

	return nil
}
