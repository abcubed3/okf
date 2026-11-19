package sync

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abcubed3/okf/pkg/bundle"
)

type syncRoundTripFunc func(*http.Request) (*http.Response, error)

func (f syncRoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestEnvelopeRoundTripAndQuarantine(t *testing.T) {
	c := &bundle.Concept{
		ID:   "tables/users",
		Path: "tables/users.md",
		Frontmatter: bundle.Frontmatter{
			Type:  "PostgreSQL Table",
			Title: "Users",
			Verified: bundle.ActorInfoList{
				{By: "human:alice", At: "2026-11-01T00:00:00Z"},
			},
		},
		Body: "Stores user rows.\n",
	}

	raw, err := SerializeConceptEnvelope(c)
	if err != nil {
		t.Fatalf("SerializeConceptEnvelope failed: %v", err)
	}

	// Simulate Google Docs BOM + smart quotes inside YAML block
	corruptedQuotes := "\uFEFF" + strings.ReplaceAll(raw, "Users", "\u201cUsers\u201d")
	parsed, hasFm, err := ParseConceptEnvelope("tables/users", "tables/users.md", corruptedQuotes)
	if err != nil || !hasFm {
		t.Fatalf("expected clean parse with BOM + smart quotes, got hasFm=%v err=%v", hasFm, err)
	}
	if parsed.Frontmatter.Title != "Users" {
		t.Errorf("expected normalized title 'Users', got %q", parsed.Frontmatter.Title)
	}

	// Legacy doc without okf-frontmatter
	legacy, hasFm, err := ParseConceptEnvelope("tables/users", "tables/users.md", "# Plain Body\nHello")
	if err != nil || hasFm || !strings.Contains(legacy.Body, "Plain Body") {
		t.Fatalf("expected legacy body parse, got hasFm=%v err=%v body=%q", hasFm, err, legacy.Body)
	}

	// Malformed YAML inside okf-frontmatter -> quarantine error
	badYAML := "```yaml okf-frontmatter\nverified: [\n```\nBody\n"
	if _, hasFm, err := ParseConceptEnvelope("tables/users", "tables/users.md", badYAML); err == nil || !hasFm {
		t.Errorf("expected quarantine error for invalid YAML, got hasFm=%v err=%v", hasFm, err)
	}
}

func TestStateScrubsLegacyStubIDs(t *testing.T) {
	dir := t.TempDir()
	sm := NewStateManager(dir)
	sm.SetExternalID("tables/users", "notion", "notion-tables/users")
	sm.SetContentHash("tables/users", "notion", "deadbeef")
	sm.SetExternalID("tables/users", "confluence", "12345")
	sm.SetContentHash("tables/users", "confluence", "cafebabe")
	if err := sm.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	sm2 := NewStateManager(dir)
	if err := sm2.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if got := sm2.GetExternalID("tables/users", "notion"); got != "" {
		t.Errorf("expected legacy notion stub ID to be scrubbed, got %q", got)
	}
	if got := sm2.GetContentHash("tables/users", "notion"); got != "" {
		t.Errorf("expected legacy notion hash to be scrubbed, got %q", got)
	}
	if got := sm2.GetExternalID("tables/users", "confluence"); got != "12345" {
		t.Errorf("expected real confluence ID '12345' to be preserved, got %q", got)
	}
}

func TestConfigEnvExpansionAndIncludeFilter(t *testing.T) {
	t.Setenv("OKF_NOTION_TOKEN", "secret-notion-tok")
	t.Setenv("JIRA_API_TOK", "secret-jira-tok")

	cfgPath := filepath.Join(t.TempDir(), "okf.yaml")
	cfgYAML := `
connectors:
  notion:
    parent_id: "parent-1"
    include:
      types: ["Metric"]
      paths: ["metrics/**"]
  jira:
    domain: "acme"
    token: "${JIRA_API_TOK}"
    project_key: "ENG"
`
	if err := os.WriteFile(cfgPath, []byte(cfgYAML), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.Connectors.Notion.Token != "secret-notion-tok" {
		t.Errorf("expected OKF_NOTION_TOKEN fallback, got %q", cfg.Connectors.Notion.Token)
	}
	if cfg.Connectors.Jira.Token != "secret-jira-tok" {
		t.Errorf("expected ${JIRA_API_TOK} expansion, got %q", cfg.Connectors.Jira.Token)
	}

	metric := &bundle.Concept{ID: "metrics/arr", Path: "metrics/arr.md", Frontmatter: bundle.Frontmatter{Type: "Metric"}}
	table := &bundle.Concept{ID: "tables/users", Path: "tables/users.md", Frontmatter: bundle.Frontmatter{Type: "Table"}}
	if !cfg.Connectors.Notion.Include.Matches(metric) {
		t.Errorf("expected metric to match include filter")
	}
	if cfg.Connectors.Notion.Include.Matches(table) {
		t.Errorf("expected table not to match include filter")
	}
}

func TestNotionPushAndPull(t *testing.T) {
	dir := t.TempDir()
	sm := NewStateManager(dir)
	nc := NewNotionConnector(sm)
	nc.client = newRateLimitedClient(&http.Client{
		Transport: syncRoundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch {
			case r.Method == http.MethodPost && r.URL.Path == "/v1/pages":
				return jsonResponse(200, `{"object":"page","id":"page-1","last_edited_time":"2026-11-14T10:00:00Z"}`), nil
			case r.Method == http.MethodPost && r.URL.Path == "/v1/search":
				return jsonResponse(200, `{"object":"list","results":[{"object":"page","id":"page-1","last_edited_time":"2026-11-14T11:00:00Z","properties":{"okf_concept_id":{"type":"rich_text","rich_text":[{"plain_text":"metrics/arr"}]}}}],"has_more":false}`), nil
			case r.Method == http.MethodGet && r.URL.Path == "/v1/blocks/page-1/children":
				return jsonResponse(200, `{"object":"list","results":[{"object":"block","id":"b1","type":"code","code":{"rich_text":[{"plain_text":"type: Metric\ntitle: ARR\n"}]}},{"object":"block","id":"b2","type":"paragraph","paragraph":{"rich_text":[{"plain_text":"Updated from Notion"}]}}],"has_more":false}`), nil
			default:
				return jsonResponse(404, `{}`), nil
			}
		}),
	}, 0)

	cfg := &Config{}
	cfg.Connectors.Notion = &NotionConfig{Token: "tok", ParentID: "p1", BaseURL: "https://api.notion.com/v1"}
	if err := nc.Initialize(context.Background(), cfg); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	c := &bundle.Concept{ID: "metrics/arr", Path: "metrics/arr.md", Frontmatter: bundle.Frontmatter{Type: "Metric", Title: "ARR"}, Body: "Initial"}
	if err := nc.Push(context.Background(), []*bundle.Concept{c}); err != nil {
		t.Fatalf("Push failed: %v", err)
	}
	if sm.GetExternalID("metrics/arr", "notion") != "page-1" {
		t.Fatalf("expected external ID page-1")
	}

	pulled, err := nc.Pull(context.Background())
	if err != nil {
		t.Fatalf("Pull failed: %v", err)
	}
	if len(pulled) != 1 || !strings.Contains(pulled[0].Body, "Updated from Notion") {
		t.Errorf("expected pulled updated concept, got %+v", pulled)
	}
	// Second pull with same last_edited_time should skip
	pulled2, err := nc.Pull(context.Background())
	if err != nil || len(pulled2) != 0 {
		t.Errorf("expected second pull to skip unchanged revision, got %d (%v)", len(pulled2), err)
	}
}

func TestJiraOptInCreateAndSearchJQL(t *testing.T) {
	dir := t.TempDir()
	sm := NewStateManager(dir)
	jc := NewJiraConnector(sm)

	var created int
	var usedSearchJQL bool
	jc.client = newRateLimitedClient(&http.Client{
		Transport: syncRoundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch {
			case r.Method == http.MethodPost && r.URL.Path == "/rest/api/3/issue":
				created++
				return jsonResponse(201, `{"key":"ENG-1"}`), nil
			case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/properties/okf_concept_id"):
				return jsonResponse(200, `{}`), nil
			case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/rest/api/3/issue/ENG-1"):
				return jsonResponse(200, `{"fields":{"updated":"2026-11-14T10:00:00Z"}}`), nil
			case r.Method == http.MethodPost && r.URL.Path == "/rest/api/3/search/jql":
				usedSearchJQL = true
				var body map[string]interface{}
				_ = json.NewDecoder(r.Body).Decode(&body)
				return jsonResponse(200, `{"issues":[],"nextPageToken":""}`), nil
			default:
				return jsonResponse(404, `{}`), nil
			}
		}),
	}, 0)

	cfg := &Config{}
	cfg.Connectors.Jira = &JiraConfig{Domain: "acme", Token: "tok", ProjectKey: "ENG", BaseURL: "https://acme.atlassian.net/rest/api/3"}
	if err := jc.Initialize(context.Background(), cfg); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	c := &bundle.Concept{ID: "tables/users", Path: "tables/users.md", Frontmatter: bundle.Frontmatter{Type: "Table", Title: "Users"}}
	// With push_new=false and no include filter, new untracked concept must NOT create a Jira issue
	if err := jc.Push(context.Background(), []*bundle.Concept{c}); err != nil {
		t.Fatalf("Push failed: %v", err)
	}
	if created != 0 {
		t.Errorf("expected 0 Jira issues created when push_new=false and no include filter, got %d", created)
	}

	// With push_new=true, it creates the issue
	cfg.Connectors.Jira.PushNew = true
	if err := jc.Push(context.Background(), []*bundle.Concept{c}); err != nil {
		t.Fatalf("Push failed: %v", err)
	}
	if created != 1 || sm.GetExternalID("tables/users", "jira") != "ENG-1" {
		t.Errorf("expected 1 Jira issue created (ENG-1), got created=%d ext=%q", created, sm.GetExternalID("tables/users", "jira"))
	}

	if _, err := jc.Pull(context.Background()); err != nil {
		t.Fatalf("Pull failed: %v", err)
	}
	if !usedSearchJQL {
		t.Errorf("expected Jira Pull to use /rest/api/3/search/jql")
	}
}
