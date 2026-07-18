// Package server implements a Model Context Protocol (MCP) server for OKF.
// It exposes resources, prompts, and tools to search and assemble context from the concept graph.
package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/abcubed3/okf/pkg/assembly"
	"github.com/abcubed3/okf/pkg/bundle"
	"github.com/abcubed3/okf/pkg/parser"
	"github.com/fsnotify/fsnotify"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gopkg.in/yaml.v3"
)

// Server wraps the MCP Server implementation and references the OKF bundle/graph.
type Server struct {
	// bundlePath is the absolute path to the bundle directory.
	bundlePath string
	// sdkServer is the underlying Model Context Protocol SDK server instance.
	sdkServer *mcp.Server
	// Token is an optional authentication token for SSE transport.
	Token string

	mu sync.RWMutex
	// bundle is the parsed OKF Bundle.
	bundle *bundle.Bundle
	// graph is the built relationship graph of all concepts.
	graph *assembly.ConceptGraph
	// searchTexts caches the lowercase full text for each concept to speed up searches.
	searchTexts map[string]string
}

// NewMCPServer parses the OKF bundle and initializes the MCP Server.
func NewMCPServer(bundlePath string) (*Server, error) {
	absPath, err := filepath.Abs(bundlePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path of bundle: %w", err)
	}

	b, err := parser.ParseBundle(context.Background(), absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OKF bundle: %w", err)
	}
	g := assembly.BuildGraph(b)

	sdkServer := mcp.NewServer(
		&mcp.Implementation{
			Name:    "okf-server",
			Version: "1.0.0",
		},
		nil,
	)

	s := &Server{
		bundlePath: absPath,
		sdkServer:  sdkServer,
		bundle:     b,
		graph:      g,
	}

	s.rebuildSearchIndex()
	s.registerCapabilities()

	return s, nil
}

// registerCapabilities registers resources, tools, and prompts with the MCP Server.
func (s *Server) registerCapabilities() {
	// 1. Resources: okf://index
	s.sdkServer.AddResource(&mcp.Resource{
		URI:         "okf://index",
		Name:        "OKF Bundle Index",
		MIMEType:    "text/markdown",
		Description: "List of all concepts in the OKF bundle",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		s.mu.RLock()
		defer s.mu.RUnlock()

		var indexBuilder strings.Builder
		indexBuilder.WriteString("# OKF Bundle Index\n\n")
		for _, c := range s.bundle.Concepts {
			indexBuilder.WriteString(fmt.Sprintf("- [%s](okf://concept/%s) (%s): %s\n", c.Frontmatter.Title, c.ID, c.Frontmatter.Type, c.Frontmatter.Desc))
		}
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      "okf://index",
					MIMEType: "text/markdown",
					Text:     indexBuilder.String(),
				},
			},
		}, nil
	})

	// 2. Resource Template: okf://concept/{id}
	s.sdkServer.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "okf://concept/{+id}",
		Name:        "OKF Concept Document",
		MIMEType:    "text/markdown",
		Description: "Raw OKF concept document including frontmatter and body",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		s.mu.RLock()
		defer s.mu.RUnlock()

		uri := req.Params.URI
		id := strings.TrimPrefix(uri, "okf://concept/")
		concept, exists := s.bundle.GetConcept(id)
		if !exists {
			return nil, fmt.Errorf("concept %q not found", id)
		}
		content := formatRawConcept(concept)
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      uri,
					MIMEType: "text/markdown",
					Text:     content,
				},
			},
		}, nil
	})

	// 3. Prompts: okf_concept_context
	s.sdkServer.AddPrompt(&mcp.Prompt{
		Name:        "okf_concept_context",
		Description: "Generate context for a concept and ask the model to analyze it",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "id",
				Description: "The ID of the concept to start the analysis from (e.g. tables/users)",
				Required:    true,
			},
			{
				Name:        "depth",
				Description: "Optional depth of links to traverse (default 2)",
				Required:    false,
			},
		},
	}, func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		s.mu.RLock()
		defer s.mu.RUnlock()

		id, ok := req.Params.Arguments["id"]
		if !ok || id == "" {
			return nil, fmt.Errorf("argument 'id' is required")
		}

		depthVal := 2
		if depthStr, ok := req.Params.Arguments["depth"]; ok && depthStr != "" {
			var d int
			if _, err := fmt.Sscanf(depthStr, "%d", &d); err == nil {
				depthVal = d
			}
		}

		opts := assembly.DefaultOptions()
		opts.MaxDepth = depthVal

		ctxStr, err := assembly.AssembleContext(s.graph, id, opts)
		if err != nil {
			return nil, err
		}

		promptText := fmt.Sprintf("Analyze the following assembled context related to concept %q:\n\n%s\n\nProvide a comprehensive summary of this concept, its metadata, and how it relates to other concepts in the graph.", id, ctxStr)

		return &mcp.GetPromptResult{
			Description: fmt.Sprintf("Analysis prompt for concept %q", id),
			Messages: []*mcp.PromptMessage{
				{
					Role: mcp.Role("user"),
					Content: &mcp.TextContent{
						Text: promptText,
					},
				},
			},
		}, nil
	})

	// 4. Tools
	mcp.AddTool(s.sdkServer, &mcp.Tool{
		Name:        "list_concepts",
		Description: "List all concepts in the OKF bundle with their types and descriptions.",
	}, s.handleListConcepts)

	mcp.AddTool(s.sdkServer, &mcp.Tool{
		Name:        "search_concepts",
		Description: "Search concepts in the OKF bundle by query (matching ID, title, description, or tags) or filter by type or tag.",
	}, s.handleSearchConcepts)

	mcp.AddTool(s.sdkServer, &mcp.Tool{
		Name:        "get_concept",
		Description: "Retrieve a specific OKF concept document by its unique ID (e.g. tables/users).",
	}, s.handleGetConcept)

	mcp.AddTool(s.sdkServer, &mcp.Tool{
		Name:        "assemble_context",
		Description: "Build a pruned, cohesive context subset of the concept graph starting from a concept ID, traversing related links.",
	}, s.handleAssembleContext)
}

// StartStdio starts the server using the standard input/output transport.
func (s *Server) StartStdio(ctx context.Context) error {
	if err := s.StartWatcher(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to start file watcher: %v\n", err)
	}
	transport := &mcp.StdioTransport{}
	return s.sdkServer.Run(ctx, transport)
}

// StartSSE starts the server using HTTP Server-Sent Events (SSE) transport.
func (s *Server) StartSSE(addr string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := s.StartWatcher(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to start file watcher: %v\n", err)
	}

	handler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		return s.sdkServer
	}, nil)

	// Secure SSE Handler with CORS boundaries and Bearer Token authentication
	secureHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. CORS check
		origin := r.Header.Get("Origin")
		if origin != "" {
			// Restrict origin to localhost / local IP addresses
			if strings.HasPrefix(origin, "http://localhost") || strings.HasPrefix(origin, "http://127.0.0.1") ||
				strings.HasPrefix(origin, "https://localhost") || strings.HasPrefix(origin, "https://127.0.0.1") {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			} else {
				http.Error(w, "Forbidden origin", http.StatusForbidden)
				return
			}
		}

		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusOK)
			return
		}

		// 2. Token authentication validation
		if s.Token != "" {
			reqToken := r.URL.Query().Get("token")
			if reqToken == "" {
				authHeader := r.Header.Get("Authorization")
				if strings.HasPrefix(authHeader, "Bearer ") {
					reqToken = strings.TrimPrefix(authHeader, "Bearer ")
				}
			}

			if reqToken != s.Token {
				http.Error(w, "Unauthorized: invalid or missing token", http.StatusUnauthorized)
				return
			}
		}

		handler.ServeHTTP(w, r)
	})

	server := &http.Server{
		Addr:    addr,
		Handler: secureHandler,
	}

	return server.ListenAndServe()
}

// StartWatcher sets up fsnotify to monitor bundle updates in the background.
func (s *Server) StartWatcher(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	err = filepath.Walk(s.bundlePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
				return filepath.SkipDir
			}
			return watcher.Add(path)
		}
		return nil
	})
	if err != nil {
		watcher.Close()
		return err
	}

	go func() {
		defer watcher.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if filepath.Ext(event.Name) == ".md" {
					if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Remove) {
						s.reloadBundle()
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				fmt.Fprintf(os.Stderr, "Watcher error: %v\n", err)
			}
		}
	}()

	return nil
}

// reloadBundle parses the bundle root directory again to load modifications.
func (s *Server) reloadBundle() {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, err := parser.ParseBundle(context.Background(), s.bundlePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to reload bundle: %v\n", err)
		return
	}
	s.bundle = b
	s.graph = assembly.BuildGraph(b)
	s.rebuildSearchIndex()
	fmt.Fprintln(os.Stderr, "Successfully reloaded OKF bundle in memory!")
}

// rebuildSearchIndex builds lowercase text caches to perform fast full-text querying.
func (s *Server) rebuildSearchIndex() {
	searchTexts := make(map[string]string)
	for id, c := range s.bundle.Concepts {
		var builder strings.Builder
		builder.WriteString(id)
		builder.WriteByte(' ')
		builder.WriteString(c.Frontmatter.Title)
		builder.WriteByte(' ')
		builder.WriteString(c.Frontmatter.Desc)
		builder.WriteByte(' ')
		builder.WriteString(c.Body)
		for _, tag := range c.Frontmatter.Tags {
			builder.WriteByte(' ')
			builder.WriteString(tag)
		}

		searchTexts[id] = strings.ToLower(builder.String())
	}
	s.searchTexts = searchTexts
}

// tokenize splits a text string into lowercase alphanumeric words.
func tokenize(text string) []string {
	lower := strings.ToLower(text)
	var tokens []string
	var current strings.Builder
	for _, r := range lower {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			current.WriteRune(r)
		} else {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	return tokens
}

// formatRawConcept formats the raw concept markdown representation (re-marshaled YAML frontmatter + body).
func formatRawConcept(c *bundle.Concept) string {
	fmBytes, err := yaml.Marshal(c.Frontmatter)
	if err != nil {
		return c.Body
	}
	return fmt.Sprintf("---\n%s---\n%s", string(fmBytes), c.Body)
}

// --- Tool Handlers ---

// ListConceptsArgs represents arguments for the list_concepts tool (currently empty).
type ListConceptsArgs struct{}

// ConceptSummary provides a concise overview of a concept.
type ConceptSummary struct {
	ID          string   `json:"id"`
	Type        string   `json:"type"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Tags        []string `json:"tags,omitempty"`
}

// ListConceptsResult encapsulates the search response containing lists of concepts.
type ListConceptsResult struct {
	Concepts []ConceptSummary `json:"concepts"`
}

// handleListConcepts returns a summary list of all available concepts in the bundle.
func (s *Server) handleListConcepts(ctx context.Context, req *mcp.CallToolRequest, args ListConceptsArgs) (*mcp.CallToolResult, ListConceptsResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var list []ConceptSummary
	for id, c := range s.bundle.Concepts {
		list = append(list, ConceptSummary{
			ID:          id,
			Type:        c.Frontmatter.Type,
			Title:       c.Frontmatter.Title,
			Description: c.Frontmatter.Desc,
			Tags:        c.Frontmatter.Tags,
		})
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: fmt.Sprintf("Found %d concepts in the bundle.", len(list)),
			},
		},
	}, ListConceptsResult{Concepts: list}, nil
}

// SearchConceptsArgs defines search criteria for searching/filtering concepts.
type SearchConceptsArgs struct {
	Query string `json:"query,omitempty" jsonschema:"optional keyword query to match against ID, title, description, or tags"`
	Type  string `json:"type,omitempty" jsonschema:"optional concept type to filter by (e.g. BigQuery Table)"`
	Tag   string `json:"tag,omitempty" jsonschema:"optional tag to filter by"`
}

// SearchConceptsResult encapsulates the search outcomes.
type SearchConceptsResult struct {
	Concepts []ConceptSummary `json:"concepts"`
}

// handleSearchConcepts executes searching and filtering based on the provided SearchConceptsArgs.
func (s *Server) handleSearchConcepts(ctx context.Context, req *mcp.CallToolRequest, args SearchConceptsArgs) (*mcp.CallToolResult, SearchConceptsResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []ConceptSummary
	qTokens := tokenize(args.Query)
	filterType := strings.ToLower(args.Type)
	filterTag := strings.ToLower(args.Tag)

	for id, c := range s.bundle.Concepts {
		if filterType != "" && strings.ToLower(c.Frontmatter.Type) != filterType {
			continue
		}

		if filterTag != "" {
			tagFound := false
			for _, t := range c.Frontmatter.Tags {
				if strings.ToLower(t) == filterTag {
					tagFound = true
					break
				}
			}
			if !tagFound {
				continue
			}
		}

		// Perform high-performance text search
		if len(qTokens) > 0 {
			matchAll := true
			textToSearch := s.searchTexts[id]
			for _, qTok := range qTokens {
				if !strings.Contains(textToSearch, qTok) {
					matchAll = false
					break
				}
			}
			if !matchAll {
				continue
			}
		}

		results = append(results, ConceptSummary{
			ID:          id,
			Type:        c.Frontmatter.Type,
			Title:       c.Frontmatter.Title,
			Description: c.Frontmatter.Desc,
			Tags:        c.Frontmatter.Tags,
		})
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: fmt.Sprintf("Found %d search results.", len(results)),
			},
		},
	}, SearchConceptsResult{Concepts: results}, nil
}

// GetConceptArgs represents the target concept ID parameter.
type GetConceptArgs struct {
	ID string `json:"id" jsonschema:"the unique ID of the concept (e.g. tables/users)"`
}

// GetConceptResult provides the full content of a requested concept.
type GetConceptResult struct {
	ID          string      `json:"id"`
	Path        string      `json:"path"`
	Frontmatter interface{} `json:"frontmatter"`
	Body        string      `json:"body"`
}

// handleGetConcept retrieves a specific OKF concept document by its unique ID.
func (s *Server) handleGetConcept(ctx context.Context, req *mcp.CallToolRequest, args GetConceptArgs) (*mcp.CallToolResult, GetConceptResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	c, exists := s.bundle.GetConcept(args.ID)
	if !exists {
		return nil, GetConceptResult{}, fmt.Errorf("concept %q not found", args.ID)
	}

	rawContent := formatRawConcept(c)

	return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: rawContent,
				},
			},
		}, GetConceptResult{
			ID:          c.ID,
			Path:        c.Path,
			Frontmatter: c.Frontmatter,
			Body:        c.Body,
		}, nil
}

// AssembleContextArgs represents the parameters configuring relationship graph context building.
type AssembleContextArgs struct {
	ID        string  `json:"id" jsonschema:"the unique starting ID of the concept"`
	Depth     *int    `json:"depth,omitempty" jsonschema:"optional maximum depth of links to traverse (default 2)"`
	Direction *string `json:"direction,omitempty" jsonschema:"optional direction of links to follow: outbound, inbound, bidirectional (default bidirectional)"`
	MaxChars  *int    `json:"max_chars,omitempty" jsonschema:"optional maximum character budget (default 16000)"`
	Format    *string `json:"format,omitempty" jsonschema:"optional format of assembled context: xml or markdown (default xml)"`
}

// AssembleContextResult contains the output text of the assembled context.
type AssembleContextResult struct {
	Context string `json:"context"`
}

// handleAssembleContext builds a pruned, cohesive context subset starting from a concept ID.
func (s *Server) handleAssembleContext(ctx context.Context, req *mcp.CallToolRequest, args AssembleContextArgs) (*mcp.CallToolResult, AssembleContextResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	opts := assembly.DefaultOptions()

	if args.Depth != nil {
		opts.MaxDepth = *args.Depth
	}
	if args.Direction != nil {
		opts.Direction = assembly.Direction(*args.Direction)
	}
	if args.MaxChars != nil {
		opts.MaxCharacters = *args.MaxChars
		opts.MaxTokens = *args.MaxChars / 4
	}
	if args.Format != nil {
		opts.Format = *args.Format
	}

	ctxStr, err := assembly.AssembleContext(s.graph, args.ID, opts)
	if err != nil {
		return nil, AssembleContextResult{}, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: ctxStr,
			},
		},
	}, AssembleContextResult{Context: ctxStr}, nil
}
