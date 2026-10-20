// Package ai provides AI-powered curation, synthesis, and grounded question answering for OKF bundles.
package ai

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/abcubed3/okf/pkg/assembly"
	"github.com/abcubed3/okf/pkg/bundle"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// AskOptions defines settings for answering questions over an OKF bundle.
type AskOptions struct {
	// Model specifies the Gemini model (defaults to GEMINI_MODEL or "gemini-2.5-flash").
	Model string
	// APIKey is the Google Gemini API Key (defaults to GEMINI_API_KEY environment variable).
	APIKey string
	// Depth is the BFS graph traversal depth (defaults to 2).
	Depth int
	// MaxChars is the character budget for assembled context (defaults to 16000).
	MaxChars int
	// SeedConcept explicitly targets a starting concept ID instead of auto-matching.
	SeedConcept string
}

// AskResult contains the grounded answer and citations.
type AskResult struct {
	// Query is the user's natural language question.
	Query string
	// Answer is the synthesized, grounded answer.
	Answer string
	// SeedConcept is the primary concept node from which context was assembled.
	SeedConcept string
	// ReferencedConcepts lists all concept IDs pulled into the context graph.
	ReferencedConcepts []string
	// Attestations lists any sanctioned computation queries found in the assembled graph.
	Attestations []string
	// AssembledContext is the raw XML/Markdown context passed to the LLM.
	AssembledContext string
}

// ConceptMatch holds match score for seed concept discovery.
type ConceptMatch struct {
	Concept *bundle.Concept
	Score   int
}

// FindSeedConcepts scores concepts against the search query to find the best entry point for BFS graph traversal.
func FindSeedConcepts(b *bundle.Bundle, query string) []*bundle.Concept {
	if b == nil || len(b.Concepts) == 0 {
		return nil
	}

	terms := strings.Fields(strings.ToLower(query))
	if len(terms) == 0 {
		return nil
	}

	var matches []ConceptMatch

	for _, c := range b.Concepts {
		score := 0
		idLower := strings.ToLower(c.ID)
		titleLower := strings.ToLower(c.Frontmatter.Title)
		descLower := strings.ToLower(c.Frontmatter.Desc)
		typeLower := strings.ToLower(c.Frontmatter.Type)
		resourceLower := strings.ToLower(c.Frontmatter.Resource)

		for _, term := range terms {
			// Skip very short stopwords
			if len(term) < 2 {
				continue
			}

			if strings.Contains(idLower, term) {
				score += 15
			}
			if strings.Contains(titleLower, term) {
				score += 12
			}
			for _, tag := range c.Frontmatter.Tags {
				if strings.EqualFold(tag, term) {
					score += 10
				}
			}
			if strings.Contains(descLower, term) {
				score += 6
			}
			if strings.Contains(typeLower, term) {
				score += 4
			}
			if strings.Contains(resourceLower, term) {
				score += 8
			}
		}

		if score > 0 {
			matches = append(matches, ConceptMatch{Concept: c, Score: score})
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	results := make([]*bundle.Concept, 0, len(matches))
	for _, m := range matches {
		results = append(results, m.Concept)
	}
	return results
}

// AskBundle executes a grounded question-answering workflow over an OKF bundle.
func AskBundle(ctx context.Context, b *bundle.Bundle, query string, opts AskOptions) (*AskResult, error) {
	if b == nil {
		return nil, fmt.Errorf("bundle cannot be nil")
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	seedID := strings.TrimSpace(opts.SeedConcept)
	if seedID == "" {
		candidates := FindSeedConcepts(b, query)
		if len(candidates) == 0 {
			// Fallback: pick the first concept if any exist
			for id := range b.Concepts {
				seedID = id
				break
			}
		} else {
			seedID = candidates[0].ID
		}
	}

	if seedID == "" {
		return nil, fmt.Errorf("no concepts found in bundle to answer query")
	}

	depth := opts.Depth
	if depth <= 0 {
		depth = 2
	}
	maxChars := opts.MaxChars
	if maxChars <= 0 {
		maxChars = 16000
	}

	// 1. Graph assembly
	graph := assembly.BuildGraph(b)
	assemblyOpts := assembly.AssemblyOptions{
		MaxDepth:      depth,
		MaxCharacters: maxChars,
		MaxTokens:     maxChars / 4,
		Direction:     assembly.DirectionBidirectional,
		Format:        "xml",
	}

	res, err := assembly.AssembleContext(graph, seedID, assemblyOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to assemble context for concept %q: %w", seedID, err)
	}

	var referencedIDs []string
	var attestations []string
	for _, node := range res.Nodes {
		referencedIDs = append(referencedIDs, node.ID)
		if c, exists := b.Concepts[node.ID]; exists {
			if c.Frontmatter.Attestation != nil && c.Frontmatter.Attestation.Query != "" {
				attestations = append(attestations, c.Frontmatter.Attestation.Query)
			}
		}
	}

	// 2. Resolve API key & Model
	apiKey := opts.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}

	modelName := opts.Model
	if modelName == "" {
		modelName = os.Getenv("GEMINI_MODEL")
	}
	if modelName == "" {
		modelName = "gemini-3.8-flash"
	}

	// If no API key is provided, return grounded offline assembly result
	if apiKey == "" {
		var offlineAns strings.Builder
		offlineAns.WriteString(fmt.Sprintf("### Grounded Context for: %q\n\n", query))
		offlineAns.WriteString(fmt.Sprintf("*Primary Entry Concept:* `%s`\n", seedID))
		offlineAns.WriteString(fmt.Sprintf("*Referenced Concepts:* `%s`\n\n", strings.Join(referencedIDs, "`, `")))

		if len(attestations) > 0 {
			offlineAns.WriteString("#### 📐 Sanctioned Computation Query (Attested):\n```sql\n")
			offlineAns.WriteString(strings.TrimSpace(attestations[0]))
			offlineAns.WriteString("\n```\n\n")
		}

		offlineAns.WriteString("*(Set GEMINI_API_KEY or pass --api-key to enable full generative AI answer synthesis)*\n")

		return &AskResult{
			Query:              query,
			Answer:             offlineAns.String(),
			SeedConcept:        seedID,
			ReferencedConcepts: referencedIDs,
			Attestations:       attestations,
			AssembledContext:   res.Context,
		}, nil
	}

	// 3. Generative AI Answer Synthesis via Gemini
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}
	defer client.Close()

	model := client.GenerativeModel(modelName)

	prompt := fmt.Sprintf(`You are an authoritative AI Knowledge Steward answering questions about an organization's data, metrics, database schemas, and playbooks.
You must answer STRICTLY based on the provided Open Knowledge Format (OKF) graph context below.

CRITICAL RULES:
1. ONLY make claims backed by the concepts in the assembled context. Do NOT invent columns, table names, or business logic.
2. If an attested query (SQL computation) exists in the context for this metric or calculation, YOU MUST provide the verbatim query and identify the attester and target engine.
3. Cite the exact concept IDs (e.g. [metrics/mrr]) you relied upon.
4. Mention the trust status and freshness (e.g., whether it is verified by a human or stale).
5. If the context does not contain enough information to answer, state clearly what is missing.

USER QUESTION: %s

ASSEMBLED OKF CONTEXT:
%s
`, query, res.Context)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("failed to generate answer from Gemini: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response received from Gemini")
	}

	var answerText strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		if txt, ok := part.(genai.Text); ok {
			answerText.WriteString(string(txt))
		}
	}

	return &AskResult{
		Query:              query,
		Answer:             answerText.String(),
		SeedConcept:        seedID,
		ReferencedConcepts: referencedIDs,
		Attestations:       attestations,
		AssembledContext:   res.Context,
	}, nil
}
