package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/abcubed3/okf/pkg/bundle"
	"github.com/google/generative-ai-go/genai"
	"golang.org/x/sync/errgroup"
	"google.golang.org/api/option"
)

// CurateOptions contains settings for AI curation.
type CurateOptions struct {
	Model  string
	APIKey string
}

type curationResponse struct {
	Description string `json:"description"`
	Body        string `json:"body"`
}

// CurateConcepts iterates through raw concepts, prompts Gemini for business descriptions,
// and mutates the concept bodies and descriptions in place.
func CurateConcepts(ctx context.Context, concepts []*bundle.Concept, opts CurateOptions) error {
	apiKey := opts.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	if apiKey == "" {
		return fmt.Errorf("GEMINI_API_KEY environment variable or --api-key flag is required for AI curation")
	}

	modelName := opts.Model
	if modelName == "" {
		modelName = "gemini-2.5-flash"
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return fmt.Errorf("failed to create Gemini client: %w", err)
	}
	defer client.Close()

	var g errgroup.Group
	g.SetLimit(5) // Limit concurrency to respect typical rate limits

	for i := range concepts {
		c := concepts[i] // Use pointer reference

		g.Go(func() error {
			model := client.GenerativeModel(modelName)
			model.ResponseMIMEType = "application/json"
			model.ResponseSchema = &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"description": {
						Type:        genai.TypeString,
						Description: "A business-friendly description (1-2 sentences) of the concept.",
					},
					"body": {
						Type:        genai.TypeString,
						Description: "A detailed markdown body explaining the schema structure and potential use cases. MUST BE VALID MARKDOWN.",
					},
				},
				Required: []string{"description", "body"},
			}

			prompt := fmt.Sprintf(
				"You are an expert data steward. I will provide you with a raw database schema extracted as an OKF concept.\nTitle: %s\nRaw Description: %s\nBody (Schema): %s\n\nPlease write a business-friendly description and a more detailed markdown body explaining the schema structure and potential use cases.",
				c.Frontmatter.Title, c.Frontmatter.Desc, c.Body,
			)

			resp, err := model.GenerateContent(ctx, genai.Text(prompt))
			if err != nil {
				return fmt.Errorf("failed to generate content for concept %q: %w", c.ID, err)
			}

			if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
				return fmt.Errorf("empty response from Gemini for concept %q", c.ID)
			}

			part := resp.Candidates[0].Content.Parts[0]
			if txtPart, ok := part.(genai.Text); ok {
				var parsed curationResponse
				if err := json.Unmarshal([]byte(txtPart), &parsed); err != nil {
					return fmt.Errorf("failed to parse structured JSON output for concept %q: %w", c.ID, err)
				}
				c.Frontmatter.Desc = parsed.Description
				c.Body = parsed.Body
			} else {
				return fmt.Errorf("unexpected response part type for concept %q", c.ID)
			}

			return nil
		})
	}

	return g.Wait()
}
