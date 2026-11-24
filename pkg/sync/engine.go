package sync

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"syscall"
	"time"

	"github.com/abcubed3/okf/pkg/bundle"
	"github.com/abcubed3/okf/pkg/parser"
)

// conceptEntry pairs a parsed concept with its pre-computed content hash.
type conceptEntry struct {
	concept *bundle.Concept
	hash    string
}

// SyncOptions configures a sync engine run.
type SyncOptions struct {
	BundlePath string
	ConfigPath string
	Daemon     bool
	Interval   int
	DryRun     bool
}

// Engine orchestrates the sync process across all registered connectors.
type Engine struct {
	bundlePath string
	config     *Config
	state      *StateManager
	connectors []Connector
	dryRun     bool
	daemon     bool
}

// Run is the entrypoint for starting the sync engine.
// It loads config and state, registers configured connectors, and starts the sync loop.
func Run(bundlePath, configPath string, daemon bool, interval int) error {
	return RunWithOptions(SyncOptions{
		BundlePath: bundlePath,
		ConfigPath: configPath,
		Daemon:     daemon,
		Interval:   interval,
	})
}

// RunWithOptions starts the sync engine with explicit SyncOptions (including DryRun).
func RunWithOptions(opts SyncOptions) error {
	cfg, err := LoadConfig(opts.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load sync configuration: %w", err)
	}

	state := NewStateManager(opts.BundlePath)
	if err := state.Load(); err != nil {
		log.Printf("Warning: failed to load state, starting fresh: %v", err)
	}

	eng := &Engine{
		bundlePath: opts.BundlePath,
		config:     cfg,
		state:      state,
		dryRun:     opts.DryRun,
		daemon:     opts.Daemon,
	}

	// Register only configured connectors in priority order.
	if cfg.Connectors.GoogleDrive != nil {
		gd := NewGoogleDriveConnector(state)
		gd.daemon = opts.Daemon
		eng.Register(gd)
	}
	if cfg.Connectors.Notion != nil {
		eng.Register(NewNotionConnector(state))
	}
	if cfg.Connectors.Confluence != nil {
		eng.Register(NewConfluenceConnector(state))
	}
	if cfg.Connectors.Jira != nil {
		eng.Register(NewJiraConnector(state))
	}
	if cfg.Connectors.Git != nil {
		eng.Register(NewGitConnector(state))
	}

	return eng.Start(opts.Daemon, opts.Interval)
}

// Register adds a Connector to the engine's active connector list.
func (e *Engine) Register(c Connector) {
	e.connectors = append(e.connectors, c)
}

// Start initialises all configured connectors and then either runs once or loops as a daemon.
func (e *Engine) Start(daemon bool, interval int) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	for _, c := range e.connectors {
		if err := c.Initialize(ctx, e.config); err != nil {
			return fmt.Errorf("failed to initialize configured connector %s: %w", c.Name(), err)
		}
	}

	if !daemon {
		return e.syncOnce(ctx)
	}

	log.Printf("Starting OKF Sync daemon (Interval: %ds). Press Ctrl+C to stop.", interval)
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	if err := e.syncOnce(ctx); err != nil {
		log.Printf("Error during initial sync: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			log.Println("OKF Sync daemon shutting down gracefully...")
			return nil
		case <-ticker.C:
			if err := e.syncOnce(ctx); err != nil {
				log.Printf("Error during periodic sync: %v", err)
			}
		}
	}
}

func (e *Engine) includeFilterFor(name string) *IncludeFilter {
	if e.config == nil {
		return nil
	}
	switch name {
	case "google_drive":
		if e.config.Connectors.GoogleDrive != nil {
			return e.config.Connectors.GoogleDrive.Include
		}
	case "notion":
		if e.config.Connectors.Notion != nil {
			return e.config.Connectors.Notion.Include
		}
	case "confluence":
		if e.config.Connectors.Confluence != nil {
			return e.config.Connectors.Confluence.Include
		}
	case "jira":
		if e.config.Connectors.Jira != nil {
			return e.config.Connectors.Jira.Include
		}
	case "git":
		if e.config.Connectors.Git != nil {
			return e.config.Connectors.Git.Include
		}
	}
	return nil
}

// syncOnce runs a single full push->pull sync cycle across all configured connectors.
func (e *Engine) syncOnce(ctx context.Context) error {
	log.Println("Starting sync cycle...")

	parsedBundle, err := parser.ParseBundle(ctx, e.bundlePath)
	if err != nil {
		return fmt.Errorf("failed to parse local bundle: %w", err)
	}

	var localConcepts []conceptEntry
	for _, concept := range parsedBundle.Concepts {
		if concept.ParseError != "" {
			continue
		}
		hash, err := HashConcept(concept)
		if err != nil {
			log.Printf("Warning: failed to hash concept %s, skipping: %v", concept.ID, err)
			continue
		}
		localConcepts = append(localConcepts, conceptEntry{concept, hash})
	}

	for _, c := range e.connectors {
		e.pushToConnector(ctx, c, localConcepts)
	}

	for _, c := range e.connectors {
		e.pullFromConnector(ctx, c, parsedBundle)
	}

	if !e.dryRun {
		e.state.UpdateLastSync(time.Now().UTC())
		if err := e.state.Save(); err != nil {
			log.Printf("Failed to save sync state: %v", err)
		}
	}

	log.Println("Sync cycle complete.")
	return nil
}

func (e *Engine) pushToConnector(ctx context.Context, c Connector, entries []conceptEntry) {
	filter := e.includeFilterFor(c.Name())
	var toSync []*bundle.Concept
	var syncHashes []string

	for _, entry := range entries {
		if !filter.Matches(entry.concept) {
			continue
		}
		stored := e.state.GetContentHash(entry.concept.ID, c.Name())
		if stored == entry.hash {
			continue
		}
		toSync = append(toSync, entry.concept)
		syncHashes = append(syncHashes, entry.hash)
	}

	if len(toSync) == 0 {
		log.Printf("[%s] All concepts up-to-date. Skipping push.", c.Name())
		return
	}

	if e.dryRun {
		log.Printf("[%s] [dry-run] Would push %d concept(s).", c.Name(), len(toSync))
		return
	}

	log.Printf("[%s] Pushing %d changed concept(s)...", c.Name(), len(toSync))
	if err := c.Push(ctx, toSync); err != nil {
		log.Printf("[%s] Failed to push concepts: %v", c.Name(), err)
		return
	}

	for i, concept := range toSync {
		if e.state.GetContentHash(concept.ID, c.Name()) != syncHashes[i] {
			e.state.SetContentHash(concept.ID, c.Name(), syncHashes[i])
		}
	}
	log.Printf("[%s] Push complete. %d concept(s) updated.", c.Name(), len(toSync))
}

func (e *Engine) pullFromConnector(ctx context.Context, c Connector, localBundle *bundle.Bundle) {
	pulled, err := c.Pull(ctx)
	if err != nil {
		log.Printf("[%s] Failed to pull updates: %v", c.Name(), err)
		return
	}
	if len(pulled) == 0 {
		return
	}

	writeBundle := bundle.NewBundle(e.bundlePath)
	var writeHashes = make(map[string]string)

	for _, concept := range pulled {
		if concept == nil || concept.ID == "" {
			continue
		}
		if concept.Path == "" {
			concept.Path = filepath.FromSlash(concept.ID) + ".md"
		}

		// Merge legacy body onto existing local frontmatter if pulled doc had no okf-frontmatter block,
		// and clear Verified if remote content changed without a new verifier entry.
		if localBundle != nil {
			if localC, exists := localBundle.GetConcept(concept.ID); exists && localC != nil {
				if concept.Frontmatter.Type == "" && localC.Frontmatter.Type != "" {
					mergedFm := localC.Frontmatter
					concept.Frontmatter = mergedFm
				}
				contentChanged := concept.Body != localC.Body || concept.Frontmatter.Computation != localC.Frontmatter.Computation
				if contentChanged && reflect.DeepEqual(concept.Frontmatter.Verified, localC.Frontmatter.Verified) {
					// Per OKF v0.3 §5.3, preserve historical verification records while advancing
					// generated.at so TrustTier() automatically demotes until re-verified.
					by := "process:okf-sync-" + c.Name()
					if concept.Frontmatter.Generated != nil && concept.Frontmatter.Generated.By != "" {
						by = concept.Frontmatter.Generated.By
					}
					concept.Frontmatter.Generated = &bundle.ActorInfo{
						By: by,
						At: time.Now().UTC().Format(time.RFC3339),
					}
				}
			}
		}

		hash, err := HashConcept(concept)
		if err != nil {
			log.Printf("[%s] Warning: failed to hash pulled concept %s, writing anyway: %v", c.Name(), concept.ID, err)
		} else if e.state.GetLastPulledHashFor(concept.ID, c.Name()) == hash {
			continue
		}

		writeBundle.Concepts[concept.ID] = concept
		writeHashes[concept.ID] = hash
	}

	if len(writeBundle.Concepts) == 0 {
		log.Printf("[%s] Pulled %d concept(s), all already up-to-date on disk.", c.Name(), len(pulled))
		return
	}

	if e.dryRun {
		log.Printf("[%s] [dry-run] Would write %d pulled concept(s) to disk.", c.Name(), len(writeBundle.Concepts))
		return
	}

	log.Printf("[%s] Writing %d updated concept(s) to local bundle...", c.Name(), len(writeBundle.Concepts))
	if err := bundle.WriteBundle(writeBundle, e.bundlePath); err != nil {
		log.Printf("[%s] Failed to write pulled concepts: %v", c.Name(), err)
		return
	}

	for id, hash := range writeHashes {
		if hash != "" {
			e.state.SetLastPulledHashFor(id, c.Name(), hash)
		}
	}
}
