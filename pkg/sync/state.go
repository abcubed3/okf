package sync

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// State tracks the synchronization state between OKF and external tools.
type State struct {
	// LastSync time across all connectors
	LastSync time.Time `json:"last_sync"`
	
	// Mappings maps OKF concept ID -> Connector Name -> External ID
	Mappings map[string]map[string]string `json:"mappings"`
}

type StateManager struct {
	path  string
	state *State
	mu    sync.RWMutex
}

func NewStateManager(bundlePath string) *StateManager {
	return &StateManager{
		path: filepath.Join(bundlePath, "okf_state.json"),
		state: &State{
			Mappings: make(map[string]map[string]string),
		},
	}
}

func (sm *StateManager) Load() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	data, err := os.ReadFile(sm.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return json.Unmarshal(data, sm.state)
}

func (sm *StateManager) Save() error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	data, err := json.MarshalIndent(sm.state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(sm.path, data, 0644)
}

func (sm *StateManager) GetExternalID(conceptID, connectorName string) string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	if sm.state.Mappings[conceptID] != nil {
		return sm.state.Mappings[conceptID][connectorName]
	}
	return ""
}

func (sm *StateManager) SetExternalID(conceptID, connectorName, externalID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	if sm.state.Mappings[conceptID] == nil {
		sm.state.Mappings[conceptID] = make(map[string]string)
	}
	sm.state.Mappings[conceptID][connectorName] = externalID
}

func (sm *StateManager) UpdateLastSync(t time.Time) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.state.LastSync = t
}
