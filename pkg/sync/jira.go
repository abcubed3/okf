package sync

import (
	"context"
	"fmt"
	"log"

	"github.com/abcubed3/okf/pkg/bundle"
)

type JiraConnector struct {
	state  *StateManager
	config *JiraConfig
}

func NewJiraConnector(state *StateManager) *JiraConnector {
	return &JiraConnector{
		state: state,
	}
}

func (c *JiraConnector) Name() string {
	return "jira"
}

func (c *JiraConnector) Initialize(ctx context.Context, cfg *Config) error {
	if cfg.Connectors.Jira == nil {
		return fmt.Errorf("jira configuration missing")
	}
	c.config = cfg.Connectors.Jira
	return nil
}

func (c *JiraConnector) Push(ctx context.Context, concept *bundle.Concept) error {
	if c.config == nil {
		return nil
	}
	extID := c.state.GetExternalID(concept.ID, c.Name())
	if extID != "" {
		return nil
	}
	log.Printf("[jira] Pushing concept %s...", concept.ID)
	c.state.SetExternalID(concept.ID, c.Name(), fmt.Sprintf("jira-%s", concept.ID))
	return nil
}

func (c *JiraConnector) Pull(ctx context.Context) ([]*bundle.Concept, error) {
	if c.config == nil {
		return nil, nil
	}
	return nil, nil
}
