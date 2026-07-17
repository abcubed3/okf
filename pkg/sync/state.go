// Package sync implements the OKF synchronization engine and connectors for external systems.
package sync

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/abcubed3/okf/pkg/bundle"
	"gopkg.in/yaml.v3"
)

// stateDirName is the hidden subdirectory inside the bundle root used for OKF internal metadata.
// Keeping state here avoids polluting the concept tree and prevents ParseBundle from picking it up.
const stateDirName = ".okf"

// stateFileName is the filename of the persisted sync state inside stateDirName.
const stateFileName = "state.json"

// ConceptState holds per-concept, per-connector synchronization metadata.
type ConceptState struct {
	// ExternalIDs maps connector name → the external system's ID for this concept
	// (e.g. a Confluence page ID, Notion page ID, Google Drive file ID).
	ExternalIDs map[string]string `json:"external_ids,omitempty"`

	// ContentHashes maps connector name → SHA-256 hex digest of the concept content
	// at the time of the last successful push to that connector.
	// An entry is only written after a confirmed successful push, so a missing or
	// mismatched hash triggers a re-push on the next sync cycle.
	ContentHashes map[string]string `json:"content_hashes,omitempty"`

	// LastPushed maps connector name → timestamp of the last successful push.
	LastPushed map[string]time.Time `json:"last_pushed,omitempty"`

	// LastPulledHash is the SHA-256 of the concept content the last time it was
	// written to disk from a pull operation. Used to skip writing unchanged files.
	LastPulledHash string `json:"last_pulled_hash,omitempty"`
}

// State is the full persisted sync state for the bundle.
type State struct {
	// LastSync is the time the last full sync cycle completed.
	LastSync time.Time `json:"last_sync"`

	// Concepts maps OKF concept ID → per-concept sync state.
	Concepts map[string]*ConceptState `json:"concepts"`
}

// StateManager manages loading, mutation, and persistence of sync state.
// All methods are safe for concurrent use.
type StateManager struct {
	path  string
	state *State
	mu    sync.RWMutex
}

// NewStateManager creates a StateManager whose state file lives in
// <bundlePath>/.okf/state.json.
func NewStateManager(bundlePath string) *StateManager {
	stateDir := filepath.Join(bundlePath, stateDirName)
	return &StateManager{
		path: filepath.Join(stateDir, stateFileName),
		state: &State{
			Concepts: make(map[string]*ConceptState),
		},
	}
}

// Load reads the persisted state from disk. Missing file is treated as a clean slate.
func (sm *StateManager) Load() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	data, err := os.ReadFile(sm.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // First run — start fresh
		}
		return fmt.Errorf("failed to read sync state: %w", err)
	}

	if err := json.Unmarshal(data, sm.state); err != nil {
		return fmt.Errorf("failed to parse sync state: %w", err)
	}

	// Ensure the top-level map is always initialised (handles legacy state files).
	if sm.state.Concepts == nil {
		sm.state.Concepts = make(map[string]*ConceptState)
	}
	return nil
}

// Save atomically writes the current state to disk.
func (sm *StateManager) Save() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Ensure the .okf/ directory exists.
	if err := os.MkdirAll(filepath.Dir(sm.path), 0o755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	data, err := json.MarshalIndent(sm.state, "", "  ")
	if err != nil {
		return err
	}

	// Write to a temp file first, then rename for atomicity.
	tmp := sm.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("failed to write sync state: %w", err)
	}
	return os.Rename(tmp, sm.path)
}

// ─── Concept accessors ────────────────────────────────────────────────────────

// conceptState returns the ConceptState for the given ID, creating it if absent.
// Must be called with at least a read lock held (callers that mutate must hold write lock).
func (sm *StateManager) conceptState(conceptID string) *ConceptState {
	cs, ok := sm.state.Concepts[conceptID]
	if !ok {
		cs = &ConceptState{
			ExternalIDs:   make(map[string]string),
			ContentHashes: make(map[string]string),
			LastPushed:    make(map[string]time.Time),
		}
		sm.state.Concepts[conceptID] = cs
	}
	// Ensure sub-maps are initialised (handles partially-migrated state files).
	if cs.ExternalIDs == nil {
		cs.ExternalIDs = make(map[string]string)
	}
	if cs.ContentHashes == nil {
		cs.ContentHashes = make(map[string]string)
	}
	if cs.LastPushed == nil {
		cs.LastPushed = make(map[string]time.Time)
	}
	return cs
}

// GetExternalID returns the external system ID for a concept+connector pair.
// Returns an empty string if none has been recorded.
func (sm *StateManager) GetExternalID(conceptID, connectorName string) string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	if cs, ok := sm.state.Concepts[conceptID]; ok {
		return cs.ExternalIDs[connectorName]
	}
	return ""
}

// SetExternalID records the external system ID for a concept+connector pair.
func (sm *StateManager) SetExternalID(conceptID, connectorName, externalID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.conceptState(conceptID).ExternalIDs[connectorName] = externalID
}

// GetContentHash returns the stored content hash for a concept+connector pair.
// Returns an empty string if no successful push has been recorded.
func (sm *StateManager) GetContentHash(conceptID, connectorName string) string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	if cs, ok := sm.state.Concepts[conceptID]; ok {
		return cs.ContentHashes[connectorName]
	}
	return ""
}

// SetContentHash records the content hash after a successful push to a connector.
func (sm *StateManager) SetContentHash(conceptID, connectorName, hash string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	cs := sm.conceptState(conceptID)
	cs.ContentHashes[connectorName] = hash
	cs.LastPushed[connectorName] = time.Now().UTC()
}

// GetLastPulledHash returns the stored hash from the last time this concept was
// written to disk from a pull operation.
func (sm *StateManager) GetLastPulledHash(conceptID string) string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	if cs, ok := sm.state.Concepts[conceptID]; ok {
		return cs.LastPulledHash
	}
	return ""
}

// SetLastPulledHash records the hash of a concept after it has been written to disk.
func (sm *StateManager) SetLastPulledHash(conceptID, hash string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.conceptState(conceptID).LastPulledHash = hash
}

// ─── Global accessors ─────────────────────────────────────────────────────────

// GetLastSync returns the timestamp of the last completed full sync cycle.
func (sm *StateManager) GetLastSync() time.Time {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state.LastSync
}

// UpdateLastSync records the time a sync cycle completed.
func (sm *StateManager) UpdateLastSync(t time.Time) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.state.LastSync = t
}

// ─── Hashing ──────────────────────────────────────────────────────────────────

// HashConcept computes a deterministic SHA-256 digest over a concept's content.
// It marshals the frontmatter with yaml.Marshal (which produces stable, sorted
// output) and appends the raw body bytes, giving a stable fingerprint that is
// independent of filesystem metadata, timestamps, or map-iteration order.
func HashConcept(c *bundle.Concept) (string, error) {
	fmBytes, err := yaml.Marshal(c.Frontmatter)
	if err != nil {
		return "", fmt.Errorf("failed to marshal frontmatter for hashing: %w", err)
	}
	h := sha256.New()
	h.Write(fmBytes)
	h.Write([]byte(c.Body))
	return hex.EncodeToString(h.Sum(nil)), nil
}
