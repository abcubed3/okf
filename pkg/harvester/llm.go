package harvester

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/abcubed3/okf/pkg/bundle"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// LLMClassifier uses the Gemini API to extract structured YAML frontmatter metadata from raw markdown.
type LLMClassifier struct {
	client *genai.Client
	model  *genai.GenerativeModel
}

// NewLLMClassifier initializes a new Gemini-backed metadata classifier.
func NewLLMClassifier(ctx context.Context, apiKey string) (*LLMClassifier, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %w", err)
	}

	// Use Gemini Flash for fast, low-cost structured outputs
	model := client.GenerativeModel("gemini-1.5-flash")
	model.ResponseMIMEType = "application/json"
	model.ResponseSchema = &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"type": {
				Type:        genai.TypeString,
				Description: "The concept type, e.g. 'Playbook', 'Database Table', 'API Endpoint', 'Metric', or 'Documentation'",
			},
			"title": {
				Type:        genai.TypeString,
				Description: "Human-readable title of the document",
			},
			"description": {
				Type:        genai.TypeString,
				Description: "Short summary description of the document's purpose",
			},
			"tags": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type: genai.TypeString,
				},
				Description: "Labels for categorization and searching",
			},
		},
		Required: []string{"type", "title", "description"},
	}

	return &LLMClassifier{
		client: client,
		model:  model,
	}, nil
}

// Classify takes the document title and body, and returns extracted OKF Frontmatter.
func (c *LLMClassifier) Classify(ctx context.Context, title, body string) (bundle.Frontmatter, error) {
	// Truncate body to save tokens if it's too large, we just need enough to classify
	if len(body) > 4000 {
		body = body[:4000]
	}

	prompt := fmt.Sprintf("Classify the following web documentation into an OKF Frontmatter metadata structure.\n\nTitle: %s\n\nContent:\n%s", title, body)

	resp, err := c.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return bundle.Frontmatter{}, fmt.Errorf("generation error: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return bundle.Frontmatter{}, fmt.Errorf("empty response from LLM")
	}

	part := resp.Candidates[0].Content.Parts[0]
	text, ok := part.(genai.Text)
	if !ok {
		return bundle.Frontmatter{}, fmt.Errorf("expected text response but got %T", part)
	}

	var fm bundle.Frontmatter
	if err := json.Unmarshal([]byte(text), &fm); err != nil {
		return bundle.Frontmatter{}, fmt.Errorf("json unmarshal error: %w", err)
	}

	return fm, nil
}

// Close closes the underlying client connection.
func (c *LLMClassifier) Close() error {
	return c.client.Close()
}
