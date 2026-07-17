package sync

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/abcubed3/okf/pkg/harvester"
	"github.com/abcubed3/okf/pkg/parser"
)

// Engine orchestrates the sync process.
type Engine struct {
	bundlePath string
	config     *Config
	state      *StateManager
	connectors []Connector
}

// Run is the entrypoint for starting the sync engine.
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
	
	// Register connectors here
	eng.Register(NewGoogleDriveConnector(state))
	eng.Register(NewNotionConnector(state))
	eng.Register(NewConfluenceConnector(state))
	eng.Register(NewJiraConnector(state))
	eng.Register(NewGitConnector(state))

	return eng.Start(daemon, interval)
}

func (e *Engine) Register(c Connector) {
	e.connectors = append(e.connectors, c)
}

func (e *Engine) Start(daemon bool, interval int) error {
	ctx := context.Background()
	
	// Initialize connectors
	for _, c := range e.connectors {
		if err := c.Initialize(ctx, e.config); err != nil {
			log.Printf("Warning: Failed to initialize connector %s (might be unconfigured): %v", c.Name(), err)
		}
	}

	if !daemon {
		return e.syncOnce(ctx)
	}

	log.Printf("Starting OKF Sync daemon (Interval: %ds)", interval)
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	// Initial sync
	if err := e.syncOnce(ctx); err != nil {
		log.Printf("Error during initial sync: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := e.syncOnce(ctx); err != nil {
				log.Printf("Error during periodic sync: %v", err)
			}
		}
	}
}

func (e *Engine) syncOnce(ctx context.Context) error {
	log.Println("Starting sync cycle...")

	// 1. Parse local bundle
	bundle, err := parser.ParseBundle(e.bundlePath)
	if err != nil {
		return fmt.Errorf("failed to parse local bundle: %w", err)
	}

	// 2. Push to all connectors
	for _, c := range e.connectors {
		for id, concept := range bundle.Concepts {
			if concept.ParseError != "" {
				continue
			}
			
			if err := c.Push(ctx, concept); err != nil {
				log.Printf("[%s] Failed to push concept %s: %v", c.Name(), id, err)
			}
		}
	}

	// 3. Pull from all connectors
	for _, c := range e.connectors {
		pulled, err := c.Pull(ctx)
		if err != nil {
			log.Printf("[%s] Failed to pull updates: %v", c.Name(), err)
			continue
		}
		if len(pulled) > 0 {
			log.Printf("[%s] Saving %d pulled concepts to local bundle...", c.Name(), len(pulled))
			if err := harvester.WriteConcepts(pulled, e.bundlePath); err != nil {
				log.Printf("[%s] Failed to write pulled concepts: %v", c.Name(), err)
			}
		}
	}

	// Update last sync state
	e.state.UpdateLastSync(time.Now())
	if err := e.state.Save(); err != nil {
		log.Printf("Failed to save sync state: %v", err)
	}

	log.Println("Sync cycle complete.")
	return nil
}
