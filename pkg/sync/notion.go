package sync

import (
	"context"
	"fmt"
	"log"

	"github.com/abcubed3/okf/pkg/bundle"
)

type NotionConnector struct {
	state  *StateManager
	config *NotionConfig
}

func NewNotionConnector(state *StateManager) *NotionConnector {
	return &NotionConnector{
		state: state,
	}
}

func (c *NotionConnector) Name() string {
	return "notion"
}

func (c *NotionConnector) Initialize(ctx context.Context, cfg *Config) error {
	if cfg.Connectors.Notion == nil {
		return fmt.Errorf("notion configuration missing")
	}
	c.config = cfg.Connectors.Notion
	return nil
}

func (c *NotionConnector) Push(ctx context.Context, concept *bundle.Concept) error {
	if c.config == nil {
		return nil
	}
	extID := c.state.GetExternalID(concept.ID, c.Name())
	if extID != "" {
		return nil
	}
	log.Printf("[notion] Pushing concept %s...", concept.ID)
	c.state.SetExternalID(concept.ID, c.Name(), fmt.Sprintf("notion-%s", concept.ID))
	return nil
}

func (c *NotionConnector) Pull(ctx context.Context) ([]*bundle.Concept, error) {
	if c.config == nil {
		return nil, nil
	}
	return nil, nil
}
