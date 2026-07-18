// Package sync implements the OKF synchronization engine and connectors for external systems.
package sync

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/abcubed3/okf/pkg/bundle"
	"github.com/abcubed3/okf/pkg/harvester"
	"github.com/abcubed3/okf/pkg/parser"
)

// conceptEntry pairs a parsed concept with its pre-computed content hash.
type conceptEntry struct {
	concept *bundle.Concept
	hash    string
}

// Engine orchestrates the sync process across all registered connectors.
type Engine struct {
	bundlePath string
	config     *Config
	state      *StateManager
	connectors []Connector
}

// Run is the entrypoint for starting the sync engine.
// It loads config and state, registers all connectors, and starts the sync loop.
func Run(bundlePath, configPath string, daemon bool, interval int) error {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load sync configuration: %w", err)
	}

	state := NewStateManager(bundlePath)
	if err := state.Load(); err != nil {
		log.Printf("Warning: failed to load state, starting fresh: %v", err)
	}

	eng := &Engine{
		bundlePath: bundlePath,
		config:     cfg,
		state:      state,
	}

	// Register connectors in priority order.
	eng.Register(NewGoogleDriveConnector(state))
	eng.Register(NewNotionConnector(state))
	eng.Register(NewConfluenceConnector(state))
	eng.Register(NewJiraConnector(state))
	eng.Register(NewGitConnector(state))

	return eng.Start(daemon, interval)
}

// Register adds a Connector to the engine's active connector list.
func (e *Engine) Register(c Connector) {
	e.connectors = append(e.connectors, c)
}

// Start initialises all connectors and then either runs once or loops as a daemon.
func (e *Engine) Start(daemon bool, interval int) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	for _, c := range e.connectors {
		if err := c.Initialize(ctx, e.config); err != nil {
			log.Printf("Warning: Failed to initialize connector %s (might be unconfigured): %v", c.Name(), err)
		}
	}

	if !daemon {
		return e.syncOnce(ctx)
	}

	log.Printf("Starting OKF Sync daemon (Interval: %ds). Press Ctrl+C to stop.", interval)
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	// Run immediately, then on each tick.
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

// syncOnce runs a single full push→pull sync cycle across all connectors.
func (e *Engine) syncOnce(ctx context.Context) error {
	log.Println("Starting sync cycle...")

	// 1. Parse the local bundle.
	parsedBundle, err := parser.ParseBundle(ctx, e.bundlePath)
	if err != nil {
		return fmt.Errorf("failed to parse local bundle: %w", err)
	}

	// Pre-compute content hashes for all valid local concepts.
	// Done once here — cost is O(concepts), not O(concepts × connectors).
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

	// 2. Push to each connector, skipping unchanged concepts (per-connector).
	for _, c := range e.connectors {
		e.pushToConnector(ctx, c, localConcepts)
	}

	// 3. Pull from each connector, skipping concepts whose content hasn't changed.
	for _, c := range e.connectors {
		e.pullFromConnector(ctx, c)
	}

	// 4. Persist state.
	e.state.UpdateLastSync(time.Now().UTC())
	if err := e.state.Save(); err != nil {
		log.Printf("Failed to save sync state: %v", err)
	}

	log.Println("Sync cycle complete.")
	return nil
}

// pushToConnector pushes only the concepts whose content hash has changed since
// the last successful push to this connector. On success it commits the new hash.
func (e *Engine) pushToConnector(ctx context.Context, c Connector, entries []conceptEntry) {
	// Filter to concepts that need pushing for this specific connector.
	var toSync []*bundle.Concept
	var syncHashes []string

	for _, entry := range entries {
		stored := e.state.GetContentHash(entry.concept.ID, c.Name())
		if stored == entry.hash {
			// Content identical to last push — nothing to do.
			continue
		}
		toSync = append(toSync, entry.concept)
		syncHashes = append(syncHashes, entry.hash)
	}

	if len(toSync) == 0 {
		log.Printf("[%s] All concepts up-to-date. Skipping push.", c.Name())
		return
	}

	log.Printf("[%s] Pushing %d changed concept(s)...", c.Name(), len(toSync))
	if err := c.Push(ctx, toSync); err != nil {
		log.Printf("[%s] Failed to push concepts: %v", c.Name(), err)
		// Do NOT update hashes — the push failed, so we must retry next cycle.
		return
	}

	// Only commit the new hashes after a confirmed successful push.
	for i, concept := range toSync {
		e.state.SetContentHash(concept.ID, c.Name(), syncHashes[i])
	}
	log.Printf("[%s] Push complete. %d concept(s) updated.", c.Name(), len(toSync))
}

// pullFromConnector fetches remote updates and writes only concepts whose content
// has changed since the last pull (pull-side deduplication).
func (e *Engine) pullFromConnector(ctx context.Context, c Connector) {
	pulled, err := c.Pull(ctx)
	if err != nil {
		log.Printf("[%s] Failed to pull updates: %v", c.Name(), err)
		return
	}

	if len(pulled) == 0 {
		return
	}

	// Filter out concepts whose content matches what we last wrote.
	var toWrite []*bundle.Concept
	var writeHashes []string

	for _, concept := range pulled {
		hash, err := HashConcept(concept)
		if err != nil {
			log.Printf("[%s] Warning: failed to hash pulled concept %s, writing anyway: %v", c.Name(), concept.ID, err)
			toWrite = append(toWrite, concept)
			writeHashes = append(writeHashes, "")
			continue
		}
		if e.state.GetLastPulledHash(concept.ID) == hash {
			// Content is identical to what's already on disk — skip the write.
			continue
		}
		toWrite = append(toWrite, concept)
		writeHashes = append(writeHashes, hash)
	}

	if len(toWrite) == 0 {
		log.Printf("[%s] Pulled %d concept(s), all already up-to-date on disk.", c.Name(), len(pulled))
		return
	}

	log.Printf("[%s] Writing %d updated concept(s) to local bundle...", c.Name(), len(toWrite))
	if err := harvester.WriteConcepts(toWrite, e.bundlePath); err != nil {
		log.Printf("[%s] Failed to write pulled concepts: %v", c.Name(), err)
		// Do NOT commit hashes — next cycle will retry.
		return
	}

	// Commit pull-side hashes only after successful write.
	for i, concept := range toWrite {
		if writeHashes[i] != "" {
			e.state.SetLastPulledHash(concept.ID, writeHashes[i])
		}
	}
}
