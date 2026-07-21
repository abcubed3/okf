package mcp

import (
	"context"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// newTestSession creates an in-memory connected MCP session against the sample testdata.
// It returns the session and a cancel function; the caller must call cancel() to clean up.
func newTestSession(t *testing.T) (*sdk.ClientSession, context.CancelFunc) {
	t.Helper()
	srv, err := NewMCPServer("../../../testdata/bundles/sample")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	clientTransport, serverTransport := sdk.NewInMemoryTransports()
	ctx, cancel := context.WithCancel(context.Background())

	go func() { _ = srv.sdkServer.Run(ctx, serverTransport) }()

	client := sdk.NewClient(&sdk.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		cancel()
		t.Fatalf("failed to connect client: %v", err)
	}
	t.Cleanup(func() {
		_ = session.Close()
		cancel()
	})
	return session, cancel
}

func TestMCPServer(t *testing.T) {
	// Create a real server pointing to our testdata sample
	srv, err := NewMCPServer("../../../testdata/bundles/sample")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Connect client and server via in-memory transport
	clientTransport, serverTransport := sdk.NewInMemoryTransports()

	// Start server in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serverErrChan := make(chan error, 1)
	go func() {
		serverErrChan <- srv.sdkServer.Run(ctx, serverTransport)
	}()

	// Start client
	client := sdk.NewClient(&sdk.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("failed to connect client: %v", err)
	}
	defer session.Close()

	// 1. Test Tools list
	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("failed to list tools: %v", err)
	}
	if len(tools.Tools) != 4 {
		t.Errorf("expected 4 tools, got %d", len(tools.Tools))
	}

	// Verify tool names
	expectedTools := map[string]bool{
		"list_concepts":    true,
		"search_concepts":  true,
		"get_concept":      true,
		"assemble_context": true,
	}
	for _, tl := range tools.Tools {
		if !expectedTools[tl.Name] {
			t.Errorf("unexpected tool: %s", tl.Name)
		}
	}

	// 2. Test Resources list
	resources, err := session.ListResources(ctx, nil)
	if err != nil {
		t.Fatalf("failed to list resources: %v", err)
	}
	foundIndex := false
	for _, res := range resources.Resources {
		if res.URI == "okf://index" {
			foundIndex = true
			break
		}
	}
	if !foundIndex {
		t.Errorf("expected to find okf://index resource")
	}

	// 3. Test Read resource
	resResult, err := session.ReadResource(ctx, &sdk.ReadResourceParams{URI: "okf://index"})
	if err != nil {
		t.Fatalf("failed to read resource okf://index: %v", err)
	}
	if len(resResult.Contents) != 1 || !strings.Contains(resResult.Contents[0].Text, "OKF Bundle Index") {
		t.Errorf("unexpected content for okf://index resource")
	}

	// 4. Test Resource template read (e.g. okf://concept/tables/users)
	conceptRes, err := session.ReadResource(ctx, &sdk.ReadResourceParams{URI: "okf://concept/tables/users"})
	if err != nil {
		t.Fatalf("failed to read concept resource: %v", err)
	}
	if len(conceptRes.Contents) != 1 || !strings.Contains(conceptRes.Contents[0].Text, "Users Table") {
		t.Errorf("unexpected content for concept resource tables/users")
	}

	// 5. Test Prompts list
	prompts, err := session.ListPrompts(ctx, nil)
	if err != nil {
		t.Fatalf("failed to list prompts: %v", err)
	}
	if len(prompts.Prompts) != 1 || prompts.Prompts[0].Name != "okf_concept_context" {
		t.Errorf("expected 1 prompt okf_concept_context, got: %v", prompts.Prompts)
	}

	// 6. Test Get prompt
	promptResult, err := session.GetPrompt(ctx, &sdk.GetPromptParams{
		Name: "okf_concept_context",
		Arguments: map[string]string{
			"id":    "tables/users",
			"depth": "1",
		},
	})
	if err != nil {
		t.Fatalf("failed to get prompt: %v", err)
	}
	if len(promptResult.Messages) != 1 || !strings.Contains(promptResult.Messages[0].Content.(*sdk.TextContent).Text, "Analyze the following assembled context") {
		t.Errorf("unexpected prompt messages content: %v", promptResult.Messages)
	}

	// 7. Test CallTool - list_concepts
	listCallParams := &sdk.CallToolParams{
		Name: "list_concepts",
	}
	listCallResult, err := session.CallTool(ctx, listCallParams)
	if err != nil {
		t.Fatalf("failed to call list_concepts: %v", err)
	}
	if !strings.Contains(listCallResult.Content[0].(*sdk.TextContent).Text, "Found") {
		t.Errorf("unexpected list_concepts output: %s", listCallResult.Content[0].(*sdk.TextContent).Text)
	}

	// 8. Test CallTool - search_concepts (filter by type)
	searchCallParams := &sdk.CallToolParams{
		Name: "search_concepts",
		Arguments: map[string]any{
			"type": "BigQuery Table",
		},
	}
	searchCallResult, err := session.CallTool(ctx, searchCallParams)
	if err != nil {
		t.Fatalf("failed to call search_concepts: %v", err)
	}
	if !strings.Contains(searchCallResult.Content[0].(*sdk.TextContent).Text, "Found") {
		t.Errorf("unexpected search_concepts output: %s", searchCallResult.Content[0].(*sdk.TextContent).Text)
	}
}

func TestMCPServer_GetConcept(t *testing.T) {
	session, _ := newTestSession(t)
	ctx := context.Background()

	t.Run("existing concept", func(t *testing.T) {
		result, err := session.CallTool(ctx, &sdk.CallToolParams{
			Name:      "get_concept",
			Arguments: map[string]any{"id": "tables/users"},
		})
		if err != nil {
			t.Fatalf("get_concept tool call failed: %v", err)
		}
		if result.IsError {
			t.Fatalf("get_concept returned error: %v", result.Content)
		}
		text := result.Content[0].(*sdk.TextContent).Text
		if !strings.Contains(text, "Users Table") {
			t.Errorf("expected 'Users Table' in response, got: %s", text)
		}
		// Should include frontmatter
		if !strings.Contains(text, "type:") {
			t.Errorf("expected frontmatter in response, got: %s", text)
		}
	})

	t.Run("nonexistent concept", func(t *testing.T) {
		result, err := session.CallTool(ctx, &sdk.CallToolParams{
			Name:      "get_concept",
			Arguments: map[string]any{"id": "nonexistent/concept"},
		})
		if err != nil {
			// Some SDK implementations return protocol errors
			return
		}
		// Either IsError is set or the text contains "not found"
		if !result.IsError {
			text := result.Content[0].(*sdk.TextContent).Text
			if !strings.Contains(strings.ToLower(text), "not found") {
				t.Errorf("expected 'not found' for missing concept, got: %s", text)
			}
		}
	})
}

func TestMCPServer_AssembleContext(t *testing.T) {
	session, _ := newTestSession(t)
	ctx := context.Background()

	t.Run("basic assembly", func(t *testing.T) {
		result, err := session.CallTool(ctx, &sdk.CallToolParams{
			Name: "assemble_context",
			Arguments: map[string]any{
				"id": "tables/users",
			},
		})
		if err != nil {
			t.Fatalf("assemble_context tool call failed: %v", err)
		}
		if result.IsError {
			t.Fatalf("assemble_context returned error: %v", result.Content)
		}
		text := result.Content[0].(*sdk.TextContent).Text
		if text == "" {
			t.Error("expected non-empty assembled context")
		}
	})

	t.Run("with depth constraint", func(t *testing.T) {
		depth := 1
		result, err := session.CallTool(ctx, &sdk.CallToolParams{
			Name: "assemble_context",
			Arguments: map[string]any{
				"id":    "tables/users",
				"depth": float64(depth),
			},
		})
		if err != nil {
			t.Fatalf("assemble_context with depth failed: %v", err)
		}
		if result.IsError {
			t.Fatalf("assemble_context with depth returned error")
		}
	})

	t.Run("with markdown format", func(t *testing.T) {
		format := "markdown"
		result, err := session.CallTool(ctx, &sdk.CallToolParams{
			Name: "assemble_context",
			Arguments: map[string]any{
				"id":     "tables/users",
				"format": format,
			},
		})
		if err != nil {
			t.Fatalf("assemble_context with markdown format failed: %v", err)
		}
		if result.IsError {
			t.Fatalf("assemble_context with markdown format returned error")
		}
	})

	t.Run("with char budget", func(t *testing.T) {
		result, err := session.CallTool(ctx, &sdk.CallToolParams{
			Name: "assemble_context",
			Arguments: map[string]any{
				"id":        "tables/users",
				"max_chars": float64(500),
			},
		})
		if err != nil {
			t.Fatalf("assemble_context with char budget failed: %v", err)
		}
		if result.IsError {
			t.Fatalf("assemble_context with char budget returned error")
		}
		text := result.Content[0].(*sdk.TextContent).Text
		// With a tight char budget, the output should still be valid
		if len(text) == 0 {
			t.Error("expected non-empty response even with tight budget")
		}
	})

	t.Run("nonexistent start concept", func(t *testing.T) {
		result, err := session.CallTool(ctx, &sdk.CallToolParams{
			Name: "assemble_context",
			Arguments: map[string]any{
				"id": "nonexistent/concept",
			},
		})
		if err != nil {
			return // protocol error is acceptable
		}
		// Either the result is an error or has an error message
		if !result.IsError {
			text := result.Content[0].(*sdk.TextContent).Text
			if !strings.Contains(strings.ToLower(text), "not found") && !strings.Contains(strings.ToLower(text), "error") {
				t.Logf("assemble_context nonexistent concept response: %s", text)
			}
		}
	})
}

func TestMCPServer_SearchConcepts(t *testing.T) {
	session, _ := newTestSession(t)
	ctx := context.Background()

	t.Run("search by query keyword", func(t *testing.T) {
		result, err := session.CallTool(ctx, &sdk.CallToolParams{
			Name:      "search_concepts",
			Arguments: map[string]any{"query": "users"},
		})
		if err != nil {
			t.Fatalf("search_concepts by query failed: %v", err)
		}
		if result.IsError {
			t.Fatalf("search_concepts returned error")
		}
		text := result.Content[0].(*sdk.TextContent).Text
		if !strings.Contains(text, "Found") {
			t.Errorf("expected 'Found' in search results, got: %s", text)
		}
	})

	t.Run("search by tag", func(t *testing.T) {
		result, err := session.CallTool(ctx, &sdk.CallToolParams{
			Name:      "search_concepts",
			Arguments: map[string]any{"tag": "ecommerce"},
		})
		if err != nil {
			t.Fatalf("search_concepts by tag failed: %v", err)
		}
		if result.IsError {
			t.Fatalf("search_concepts by tag returned error")
		}
		text := result.Content[0].(*sdk.TextContent).Text
		if !strings.Contains(text, "Found") {
			t.Errorf("expected 'Found' in tag search results, got: %s", text)
		}
	})

	t.Run("search with no matches", func(t *testing.T) {
		result, err := session.CallTool(ctx, &sdk.CallToolParams{
			Name:      "search_concepts",
			Arguments: map[string]any{"query": "zzz_definitely_no_match_xyz"},
		})
		if err != nil {
			t.Fatalf("search_concepts failed: %v", err)
		}
		if result.IsError {
			t.Fatalf("search_concepts returned error for empty results")
		}
		text := result.Content[0].(*sdk.TextContent).Text
		if !strings.Contains(text, "Found 0") {
			t.Errorf("expected 'Found 0' for no-match query, got: %s", text)
		}
	})
}

func TestMCPServer_InvalidBundle(t *testing.T) {
	_, err := NewMCPServer("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Error("expected error when creating server with nonexistent bundle path")
	}
}
