package sync

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/abcubed3/okf/pkg/bundle"
)

// ─── HashConcept tests ────────────────────────────────────────────────────────

func TestHashConcept_Deterministic(t *testing.T) {
	c := &bundle.Concept{
		ID:   "tables/users",
		Path: "tables/users.md",
		Frontmatter: bundle.Frontmatter{
			Type:  "PostgreSQL Table",
			Title: "Users",
			Desc:  "User accounts.",
			Tags:  []string{"database", "table"},
		},
		Body: "# Users\n\nStores user accounts.",
	}

	h1, err := HashConcept(c)
	if err != nil {
		t.Fatalf("HashConcept error: %v", err)
	}
	h2, err := HashConcept(c)
	if err != nil {
		t.Fatalf("HashConcept error on second call: %v", err)
	}
	if h1 != h2 {
		t.Errorf("HashConcept is non-deterministic: %q != %q", h1, h2)
	}
	if len(h1) != 64 {
		t.Errorf("expected 64-char hex digest, got %d chars", len(h1))
	}
}

func TestHashConcept_ChangesOnBodyEdit(t *testing.T) {
	base := &bundle.Concept{
		ID:          "tables/users",
		Frontmatter: bundle.Frontmatter{Type: "Table"},
		Body:        "original body",
	}
	modified := &bundle.Concept{
		ID:          "tables/users",
		Frontmatter: bundle.Frontmatter{Type: "Table"},
		Body:        "modified body",
	}

	h1, _ := HashConcept(base)
	h2, _ := HashConcept(modified)
	if h1 == h2 {
		t.Error("expected different hashes for different bodies")
	}
}

func TestHashConcept_ChangesOnFrontmatterEdit(t *testing.T) {
	base := &bundle.Concept{
		Frontmatter: bundle.Frontmatter{Type: "Table", Title: "Users"},
		Body:        "body",
	}
	modified := &bundle.Concept{
		Frontmatter: bundle.Frontmatter{Type: "Table", Title: "Customers"},
		Body:        "body",
	}

	h1, _ := HashConcept(base)
	h2, _ := HashConcept(modified)
	if h1 == h2 {
		t.Error("expected different hashes when frontmatter title changes")
	}
}

// ─── StateManager tests ───────────────────────────────────────────────────────

func newTestStateManager(t *testing.T) (*StateManager, string) {
	t.Helper()
	dir := t.TempDir()
	sm := NewStateManager(dir)
	return sm, dir
}

func TestNewStateManager_StatePath(t *testing.T) {
	dir := t.TempDir()
	sm := NewStateManager(dir)
	expected := filepath.Join(dir, ".okf", "state.json")
	if sm.path != expected {
		t.Errorf("expected state path %q, got %q", expected, sm.path)
	}
}

func TestStateManager_LoadMissingFile(t *testing.T) {
	sm, _ := newTestStateManager(t)
	// No state file yet — Load should not error.
	if err := sm.Load(); err != nil {
		t.Errorf("Load on missing file should return nil, got: %v", err)
	}
}

func TestStateManager_SaveAndLoad(t *testing.T) {
	sm, _ := newTestStateManager(t)

	sm.SetExternalID("tables/users", "notion", "notion-abc123")
	sm.SetContentHash("tables/users", "notion", "deadbeef")
	sm.SetLastPulledHash("tables/users", "pullhash1")
	sm.UpdateLastSync(time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC))

	if err := sm.Save(); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	// Load into a fresh manager pointing at the same path.
	sm2 := NewStateManager(filepath.Dir(filepath.Dir(sm.path)))
	if err := sm2.Load(); err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if got := sm2.GetExternalID("tables/users", "notion"); got != "notion-abc123" {
		t.Errorf("expected external ID 'notion-abc123', got %q", got)
	}
	if got := sm2.GetContentHash("tables/users", "notion"); got != "deadbeef" {
		t.Errorf("expected content hash 'deadbeef', got %q", got)
	}
	if got := sm2.GetLastPulledHash("tables/users"); got != "pullhash1" {
		t.Errorf("expected pull hash 'pullhash1', got %q", got)
	}
	if got := sm2.GetLastSync(); !got.Equal(time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)) {
		t.Errorf("unexpected LastSync: %v", got)
	}
}

func TestStateManager_CreatesDotOkfDirectory(t *testing.T) {
	sm, dir := newTestStateManager(t)
	sm.SetExternalID("foo", "bar", "baz")
	if err := sm.Save(); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	dotOkf := filepath.Join(dir, ".okf")
	info, err := os.Stat(dotOkf)
	if err != nil {
		t.Fatalf("expected .okf directory to exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected .okf to be a directory")
	}

	stateFile := filepath.Join(dotOkf, "state.json")
	if _, err := os.Stat(stateFile); err != nil {
		t.Errorf("expected state.json to exist: %v", err)
	}
}

func TestStateManager_GetExternalID_Missing(t *testing.T) {
	sm, _ := newTestStateManager(t)
	if got := sm.GetExternalID("nonexistent", "notion"); got != "" {
		t.Errorf("expected empty string for missing concept, got %q", got)
	}
}

func TestStateManager_GetContentHash_Missing(t *testing.T) {
	sm, _ := newTestStateManager(t)
	if got := sm.GetContentHash("nonexistent", "notion"); got != "" {
		t.Errorf("expected empty string for missing hash, got %q", got)
	}
}

func TestStateManager_GetLastPulledHash_Missing(t *testing.T) {
	sm, _ := newTestStateManager(t)
	if got := sm.GetLastPulledHash("nonexistent"); got != "" {
		t.Errorf("expected empty string for missing pull hash, got %q", got)
	}
}

func TestStateManager_GetLastSync_Zero(t *testing.T) {
	sm, _ := newTestStateManager(t)
	if !sm.GetLastSync().IsZero() {
		t.Error("expected zero LastSync on fresh StateManager")
	}
}

func TestStateManager_PerConnectorHashes(t *testing.T) {
	sm, _ := newTestStateManager(t)

	// Set different hashes for the same concept across two connectors
	sm.SetContentHash("tables/users", "notion", "hash-notion")
	sm.SetContentHash("tables/users", "confluence", "hash-confluence")

	if got := sm.GetContentHash("tables/users", "notion"); got != "hash-notion" {
		t.Errorf("notion hash: expected 'hash-notion', got %q", got)
	}
	if got := sm.GetContentHash("tables/users", "confluence"); got != "hash-confluence" {
		t.Errorf("confluence hash: expected 'hash-confluence', got %q", got)
	}

	// Updating one should not affect the other
	sm.SetContentHash("tables/users", "notion", "hash-notion-v2")
	if got := sm.GetContentHash("tables/users", "confluence"); got != "hash-confluence" {
		t.Errorf("confluence hash changed unexpectedly: %q", got)
	}
}

func TestStateManager_AtomicSave(t *testing.T) {
	sm, dir := newTestStateManager(t)
	sm.SetExternalID("test/concept", "git", "git-id-1")

	if err := sm.Save(); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	// Verify no .tmp file was left behind
	tmpPath := sm.path + ".tmp"
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Errorf("expected .tmp file to be cleaned up after Save")
	}

	// Verify the JSON is valid and well-formed
	data, err := os.ReadFile(filepath.Join(dir, ".okf", "state.json"))
	if err != nil {
		t.Fatalf("could not read state.json: %v", err)
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		t.Errorf("state.json is not valid JSON: %v", err)
	}
}

func TestStateManager_LastPushedTimestamp(t *testing.T) {
	sm, _ := newTestStateManager(t)

	before := time.Now().UTC().Add(-time.Second)
	sm.SetContentHash("tables/orders", "gdrive", "somehash")
	after := time.Now().UTC().Add(time.Second)

	sm.mu.RLock()
	cs := sm.state.Concepts["tables/orders"]
	lp := cs.LastPushed["gdrive"]
	sm.mu.RUnlock()

	if lp.Before(before) || lp.After(after) {
		t.Errorf("LastPushed timestamp %v is not within expected range [%v, %v]", lp, before, after)
	}
}

// ─── Engine integration test ──────────────────────────────────────────────────

// mockConnector records which concepts it received in Push and returns none from Pull.
type mockConnector struct {
	name    string
	pushed  []*bundle.Concept
	pullRet []*bundle.Concept
}

func (m *mockConnector) Name() string                                  { return m.name }
func (m *mockConnector) Initialize(_ context.Context, _ *Config) error { return nil }
func (m *mockConnector) Push(_ context.Context, concepts []*bundle.Concept) error {
	m.pushed = append(m.pushed, concepts...)
	return nil
}
func (m *mockConnector) Pull(_ context.Context) ([]*bundle.Concept, error) {
	return m.pullRet, nil
}

func TestEngine_SkipsUnchangedConceptsOnSecondSync(t *testing.T) {
	dir := t.TempDir()

	// Write a concept file
	conceptDir := filepath.Join(dir, "tables")
	if err := os.MkdirAll(conceptDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\ntype: Table\ntitle: Users\n---\n# Users\nBody text."
	if err := os.WriteFile(filepath.Join(conceptDir, "users.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	state := NewStateManager(dir)
	mock := &mockConnector{name: "mock"}

	eng := &Engine{
		bundlePath: dir,
		config:     &Config{},
		state:      state,
		connectors: []Connector{mock},
	}

	ctx := context.Background()

	// First sync — concept is new, should be pushed.
	if err := eng.syncOnce(ctx); err != nil {
		t.Fatalf("first syncOnce error: %v", err)
	}
	if len(mock.pushed) != 1 {
		t.Errorf("expected 1 push on first sync, got %d", len(mock.pushed))
	}

	// Reset the mock counter but don't change the file.
	mock.pushed = nil

	// Second sync — nothing changed, should skip push.
	if err := eng.syncOnce(ctx); err != nil {
		t.Fatalf("second syncOnce error: %v", err)
	}
	if len(mock.pushed) != 0 {
		t.Errorf("expected 0 pushes on second sync (no changes), got %d", len(mock.pushed))
	}
}

func TestEngine_PushesAfterContentChange(t *testing.T) {
	dir := t.TempDir()

	conceptDir := filepath.Join(dir, "tables")
	if err := os.MkdirAll(conceptDir, 0o755); err != nil {
		t.Fatal(err)
	}
	filePath := filepath.Join(conceptDir, "users.md")
	writeContent := func(body string) {
		t.Helper()
		content := "---\ntype: Table\ntitle: Users\n---\n" + body
		if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	state := NewStateManager(dir)
	mock := &mockConnector{name: "mock"}
	eng := &Engine{
		bundlePath: dir,
		config:     &Config{},
		state:      state,
		connectors: []Connector{mock},
	}

	ctx := context.Background()

	writeContent("# Users\nOriginal body.")
	if err := eng.syncOnce(ctx); err != nil {
		t.Fatal(err)
	}
	firstPushCount := len(mock.pushed)
	mock.pushed = nil

	// Modify the file content.
	writeContent("# Users\nEdited body — this changed.")
	if err := eng.syncOnce(ctx); err != nil {
		t.Fatal(err)
	}

	if len(mock.pushed) != firstPushCount {
		t.Errorf("expected %d push(es) after content change, got %d", firstPushCount, len(mock.pushed))
	}
}

func TestEngine_PullSideDeduplication(t *testing.T) {
	dir := t.TempDir()

	state := NewStateManager(dir)
	pulled := &bundle.Concept{
		ID:          "pulled/concept",
		Path:        "pulled/concept.md",
		Frontmatter: bundle.Frontmatter{Type: "External"},
		Body:        "# Pulled\nContent from remote.",
	}
	mock := &mockConnector{
		name:    "mock",
		pullRet: []*bundle.Concept{pulled},
	}
	eng := &Engine{
		bundlePath: dir,
		config:     &Config{},
		state:      state,
		connectors: []Connector{mock},
	}

	ctx := context.Background()

	// First pull — concept is new, should be written.
	if err := eng.syncOnce(ctx); err != nil {
		t.Fatalf("first syncOnce error: %v", err)
	}
	writtenPath := filepath.Join(dir, pulled.Path)
	if _, err := os.Stat(writtenPath); err != nil {
		t.Errorf("expected pulled concept file to be written at %s: %v", writtenPath, err)
	}

	// Verify the pull hash was stored.
	hash, _ := HashConcept(pulled)
	if got := state.GetLastPulledHash(pulled.ID); got != hash {
		t.Errorf("expected stored pull hash %q, got %q", hash, got)
	}
}
