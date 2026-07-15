package harvester

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abcubed3/okf/pkg/bundle"
)

func TestOpenAPIHarvester_V3(t *testing.T) {
	specContent := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      summary: List users
      description: Returns a list of users.
      parameters:
        - name: limit
          in: query
          description: Max users to return
          required: false
          schema:
            type: integer
      responses:
        '200':
          description: A list of users
          content:
            application/json:
              schema:
                type: array
                items:
                  type: object
                  properties:
                    id:
                      type: string
  /users/{id}:
    post:
      summary: Create or update user
      requestBody:
        description: User data
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                name:
                  type: string
      responses:
        '200':
          description: Success
`

	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "openapi.yaml")
	if err := os.WriteFile(specFile, []byte(specContent), 0644); err != nil {
		t.Fatalf("failed to write mock spec: %v", err)
	}

	h := NewOpenAPIHarvester(specFile)
	concepts, err := h.Harvest(context.Background())
	if err != nil {
		t.Fatalf("Harvest failed: %v", err)
	}

	if len(concepts) != 2 {
		t.Fatalf("expected 2 concepts, got %d", len(concepts))
	}

	var getUsersConcept *bundle.Concept
	var postUserConcept *bundle.Concept
	for _, c := range concepts {
		if c.ID == "endpoints/get-users" {
			getUsersConcept = c
		} else if c.ID == "endpoints/post-users-id" {
			postUserConcept = c
		}
	}

	if getUsersConcept == nil {
		t.Fatal("get-users concept not found")
	}
	if postUserConcept == nil {
		t.Fatal("post-users-id concept not found")
	}

	// Verify GET Users Details
	if getUsersConcept.Frontmatter.Type != "API Endpoint" {
		t.Errorf("expected type 'API Endpoint', got %q", getUsersConcept.Frontmatter.Type)
	}
	if getUsersConcept.Frontmatter.Title != "GET /users" {
		t.Errorf("expected title 'GET /users', got %q", getUsersConcept.Frontmatter.Title)
	}
	if !strings.Contains(getUsersConcept.Body, "limit | query | integer | No | Max users to return") {
		t.Errorf("expected parameters table in body, got: %s", getUsersConcept.Body)
	}

	// Verify POST User Details
	if !strings.Contains(postUserConcept.Body, "## Request Body") {
		t.Errorf("expected Request Body section, got: %s", postUserConcept.Body)
	}
	if !strings.Contains(postUserConcept.Body, "- **Type**: `object` (application/json)") {
		t.Errorf("expected body type in POST concept body, got: %s", postUserConcept.Body)
	}
}

func TestOpenAPIHarvester_V2(t *testing.T) {
	// Use JSON format to avoid YAML unmarshal issues with openapi3.Types in parameters.
	specContent := `{
  "swagger": "2.0",
  "info": { "title": "Test API v2", "version": "1.0.0" },
  "paths": {
    "/pets": {
      "get": {
        "summary": "List pets",
        "parameters": [
          {
            "name": "tags",
            "in": "query",
            "type": "string",
            "required": false
          }
        ],
        "responses": {
          "200": {
            "description": "list of pets",
            "schema": { "type": "array", "items": { "type": "string" } }
          }
        }
      }
    }
  }
}`

	tmpDir := t.TempDir()
	// Write with .json extension so openapi3 loader picks it up via JSON unmarshal.
	specFile := filepath.Join(tmpDir, "swagger.json")
	if err := os.WriteFile(specFile, []byte(specContent), 0644); err != nil {
		t.Fatalf("failed to write mock spec: %v", err)
	}

	h := NewOpenAPIHarvester(specFile)
	concepts, err := h.Harvest(context.Background())
	if err != nil {
		t.Fatalf("Harvest failed: %v", err)
	}

	if len(concepts) != 1 {
		t.Fatalf("expected 1 concept, got %d", len(concepts))
	}

	c := concepts[0]
	if c.ID != "endpoints/get-pets" {
		t.Errorf("expected concept ID 'endpoints/get-pets', got %q", c.ID)
	}
	if c.Frontmatter.Title != "GET /pets" {
		t.Errorf("expected title 'GET /pets', got %q", c.Frontmatter.Title)
	}
	if !strings.Contains(c.Body, "tags | query | string | No") {
		t.Errorf("expected param tags details in body, got: %s", c.Body)
	}
	if !strings.Contains(c.Body, "200 | list of pets | []string") {
		t.Errorf("expected responses table in body, got: %s", c.Body)
	}
}
