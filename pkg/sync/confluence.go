package sync

import (
	"context"
	"fmt"
	"log"

	"github.com/abcubed3/okf/pkg/bundle"
)

type ConfluenceConnector struct {
	state  *StateManager
	config *ConfluenceConfig
}

func NewConfluenceConnector(state *StateManager) *ConfluenceConnector {
	return &ConfluenceConnector{
		state: state,
	}
}

func (c *ConfluenceConnector) Name() string {
	return "confluence"
}

func (c *ConfluenceConnector) Initialize(ctx context.Context, cfg *Config) error {
	if cfg.Connectors.Confluence == nil {
		return fmt.Errorf("confluence configuration missing")
	}
	c.config = cfg.Connectors.Confluence
	return nil
}

func (c *ConfluenceConnector) Push(ctx context.Context, concepts []*bundle.Concept) error {
	if c.config == nil {
		return nil
	}
	for _, concept := range concepts {
		extID := c.state.GetExternalID(concept.ID, c.Name())
		if extID != "" {
			continue
		}
		log.Printf("[confluence] Pushing concept %s...", concept.ID)
		c.state.SetExternalID(concept.ID, c.Name(), fmt.Sprintf("confluence-%s", concept.ID))
	}
	return nil
}

func (c *ConfluenceConnector) Pull(ctx context.Context) ([]*bundle.Concept, error) {
	if c.config == nil {
		return nil, nil
	}
	return nil, nil
}
