package export

import (
	"encoding/json"
	"testing"

	"github.com/abcubed3/okf/pkg/bundle"
)

func TestExportBundleToJSONLD(t *testing.T) {
	b := bundle.NewBundle("/test")
	b.Concepts["tables/users"] = &bundle.Concept{
		ID:   "tables/users",
		Path: "tables/users.md",
		Frontmatter: bundle.Frontmatter{
			Type:      "PostgreSQL Table",
			Title:     "Users Table",
			Desc:      "Stores user accounts",
			Resource:  "users",
			Tags:      []string{"db", "user"},
			Timestamp: "2026-07-15T00:00:00Z",
		},
		Body: `
# Users Table
Stores user accounts.

## Schema
| Column | Type | Nullable | Default | Description |
| ------ | ---- | -------- | ------- | ----------- |
| id | integer | NO | nextval('users_id_seq') | Unique ID |
| email | character varying | NO | | User email |
`,
	}

	b.Concepts["endpoints/create-user"] = &bundle.Concept{
		ID:   "endpoints/create-user",
		Path: "endpoints/create-user.md",
		Frontmatter: bundle.Frontmatter{
			Type:      "API Endpoint",
			Title:     "POST /users",
			Desc:      "Creates a new user",
			Resource:  "/users",
			Tags:      []string{"api"},
			Timestamp: "2026-07-15T00:00:00Z",
		},
		Body: `# POST /users`,
	}

	b.Concepts["playbooks/deploy"] = &bundle.Concept{
		ID:   "playbooks/deploy",
		Path: "playbooks/deploy.md",
		Frontmatter: bundle.Frontmatter{
			Type:      "Playbook",
			Title:     "Deployment Playbook",
			Desc:      "How to deploy the service",
			Tags:      []string{"ops"},
			Timestamp: "2026-07-15T00:00:00Z",
		},
		Body: `# Deploying`,
	}

	bytes, err := ExportBundleToJSONLD(b)
	if err != nil {
		t.Fatalf("failed to export bundle: %v", err)
	}

	var graph Graph
	if err := json.Unmarshal(bytes, &graph); err != nil {
		t.Fatalf("failed to unmarshal JSON-LD output: %v", err)
	}

	if graph.Context != "https://schema.org" {
		t.Errorf("expected context 'https://schema.org', got %q", graph.Context)
	}

	if len(graph.Graph) != 3 {
		t.Fatalf("expected 3 items in graph, got %d", len(graph.Graph))
	}

	var verifiedDataset, verifiedWebAPI, verifiedArticle bool

	for _, item := range graph.Graph {
		m, ok := item.(map[string]interface{})
		if !ok {
			t.Fatalf("expected graph item to be a map")
		}

		itemType, _ := m["@type"].(string)
		switch itemType {
		case "Dataset":
			verifiedDataset = true
			if m["name"] != "Users Table" {
				t.Errorf("expected Dataset name 'Users Table', got %v", m["name"])
			}
			vars, ok := m["variableMeasured"].([]interface{})
			if !ok || len(vars) != 2 {
				t.Errorf("expected 2 columns in variableMeasured, got %v", m["variableMeasured"])
			} else {
				col1 := vars[0].(map[string]interface{})
				if col1["name"] != "id" || col1["valueReference"] != "integer" || col1["description"] != "Unique ID" {
					t.Errorf("unexpected column 1 values: %v", col1)
				}
			}
		case "WebAPI":
			verifiedWebAPI = true
			if m["name"] != "POST /users" {
				t.Errorf("expected WebAPI name 'POST /users', got %v", m["name"])
			}
		case "TechArticle":
			verifiedArticle = true
			if m["name"] != "Deployment Playbook" {
				t.Errorf("expected TechArticle name 'Deployment Playbook', got %v", m["name"])
			}
		default:
			t.Errorf("unexpected item type in graph: %s", itemType)
		}
	}

	if !verifiedDataset {
		t.Error("Dataset mapping was not verified")
	}
	if !verifiedWebAPI {
		t.Error("WebAPI mapping was not verified")
	}
	if !verifiedArticle {
		t.Error("TechArticle mapping was not verified")
	}
}

func TestParseColumnsFromMarkdown(t *testing.T) {
	body := `
| Column | Type | Nullable | Default | Description |
|---|---|---|---|---|
| id | int | NO | NULL | user ID |
| name | varchar(255) | YES | NULL | user name |
`
	cols := parseColumnsFromMarkdown(body)
	if len(cols) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(cols))
	}

	if cols[0].Name != "id" || cols[0].ValueReference != "int" || cols[0].Description != "user ID" {
		t.Errorf("unexpected column 0: %+v", cols[0])
	}
	if cols[1].Name != "name" || cols[1].ValueReference != "varchar(255)" || cols[1].Description != "user name" {
		t.Errorf("unexpected column 1: %+v", cols[1])
	}
}
