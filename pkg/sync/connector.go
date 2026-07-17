package sync

import (
	"context"

	"github.com/abcubed3/okf/pkg/bundle"
)

// Connector represents an integration with an external system.
type Connector interface {
	// Initialize sets up the connector with the given configuration.
	Initialize(ctx context.Context, cfg *Config) error
	
	// Push syncs a concept from OKF to the external system.
	Push(ctx context.Context, concept *bundle.Concept) error
	
	// Pull fetches updates from the external system into OKF format.
	Pull(ctx context.Context) ([]*bundle.Concept, error)

	// Name returns the identifier of the connector (e.g. "google_drive").
	Name() string
}
