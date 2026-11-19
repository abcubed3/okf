package bundle

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDiffV02FieldsAndIndexFiles(t *testing.T) {
	a := NewBundle("/a")
	a.Concepts["comp/rev"] = &Concept{
		ID:   "comp/rev",
		Path: "comp/rev.md",
		Frontmatter: Frontmatter{
			Type:        "Attested Computation",
			Title:       "Revenue",
			Status:      ConceptLifecycleDraft,
			StaleAfter:  "2026-12-01T00:00:00Z",
			Runtime:     "postgres",
			Computation: "queries/rev.sql",
			Parameters:  []Parameter{{Name: "days", Type: "int", Required: true}},
			Executor:    &Executor{Resource: "exec.sh", Receipt: []string{"rows"}},
			Attester:    &Attester{Resource: "check.py"},
			Verified:    ActorInfoList{{By: "human:alice", At: "2026-11-01T00:00:00Z"}},
		},
		Body: "body",
	}
	a.IndexFiles["index.md"] = &Concept{
		ID:          "index",
		Path:        "index.md",
		Frontmatter: Frontmatter{OKFVersion: "0.2", Title: "Root A"},
		Body:        "Root",
	}

	b := NewBundle("/b")
	b.Concepts["comp/rev"] = &Concept{
		ID:   "comp/rev",
		Path: "comp/rev.md",
		Frontmatter: Frontmatter{
			Type:        "Attested Computation",
			Title:       "Revenue",
			Status:      ConceptLifecycleStable,
			StaleAfter:  "2026-12-31T00:00:00Z",
			Runtime:     "bigquery",
			Computation: "queries/rev_v2.sql",
			Parameters:  []Parameter{{Name: "days", Type: "int", Required: false}},
			Executor:    &Executor{Resource: "exec_v2.sh", Receipt: []string{"rows", "hash"}},
			Attester:    &Attester{Resource: "check_v2.py"},
			Verified:    ActorInfoList{{By: "human:bob", At: "2026-11-02T00:00:00Z"}},
		},
		Body: "body",
	}
	b.IndexFiles["index.md"] = &Concept{
		ID:          "index",
		Path:        "index.md",
		Frontmatter: Frontmatter{OKFVersion: "0.2", Title: "Root B"},
		Body:        "Root",
	}

	d, err := Diff(a, b)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}
	if len(d.Changes) != 2 {
		t.Fatalf("expected 2 changes (concept + index.md), got %d", len(d.Changes))
	}

	var compDiff *ConceptDiff
	for i := range d.Changes {
		if d.Changes[i].ID == "comp/rev" {
			compDiff = &d.Changes[i]
		}
	}
	if compDiff == nil {
		t.Fatalf("expected comp/rev diff")
	}
	for _, key := range []string{"status", "stale_after", "runtime", "computation", "parameters", "executor", "attester", "verified"} {
		if _, ok := compDiff.FmDiffs[key]; !ok {
			t.Errorf("expected FmDiffs[%q] to be recorded", key)
		}
	}
}

func TestMergeUnionTrustPrecedenceAndStaleReplacement(t *testing.T) {
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)

	// 1. Non-stale human-reviewed source beats newer unverified target and does not backfill authoritative fields.
	src := NewBundle("/src")
	src.OKFVersion = "0.2"
	src.IndexFiles["index.md"] = &Concept{
		ID:          "index",
		Path:        "index.md",
		Frontmatter: Frontmatter{OKFVersion: "0.2", Title: "Root"},
		Body:        "Index body",
	}
	src.Concepts["metrics/arr"] = &Concept{
		ID:   "metrics/arr",
		Path: "metrics/arr.md",
		Frontmatter: Frontmatter{
			Type:       "Metric",
			Title:      "ARR",
			StaleAfter: "2026-12-01T00:00:00Z",
			Verified:   ActorInfoList{{By: "human:alice", At: "2026-11-01T00:00:00Z"}},
			Timestamp:  "2026-11-01T00:00:00Z",
		},
		Body:      "Human reviewed body",
		Citations: []Citation{{Number: 1, Title: "Src", URI: "https://example.com/src"}},
	}

	// 2. Stale human-reviewed concept yields to newer unverified revision, clearing expired human sign-off.
	src.Concepts["metrics/stale"] = &Concept{
		ID:   "metrics/stale",
		Path: "metrics/stale.md",
		Frontmatter: Frontmatter{
			Type:       "Metric",
			Title:      "Stale Metric",
			StaleAfter: "2026-10-01T00:00:00Z",
			Verified:   ActorInfoList{{By: "human:alice", At: "2026-09-01T00:00:00Z"}},
			Timestamp:  "2026-09-01T00:00:00Z",
		},
		Body: "Stale body",
	}

	tgt := NewBundle("/tgt")
	tgt.Concepts["metrics/arr"] = &Concept{
		ID:   "metrics/arr",
		Path: "metrics/arr.md",
		Frontmatter: Frontmatter{
			Type:        "Metric",
			Title:       "ARR",
			Computation: "evil.sql",
			Resource:    "evil_table",
			Extra:       map[string]interface{}{"unreviewed": true},
			Timestamp:   "2026-11-10T00:00:00Z", // newer, but unverified
		},
		Body:      "Unverified body",
		Citations: []Citation{{Number: 2, Title: "Evil", URI: "https://example.com/evil"}},
	}
	tgt.Concepts["metrics/stale"] = &Concept{
		ID:   "metrics/stale",
		Path: "metrics/stale.md",
		Frontmatter: Frontmatter{
			Type:      "Metric",
			Title:     "Fresh Unverified Metric",
			Timestamp: "2026-11-10T00:00:00Z",
		},
		Body: "Fresh body",
	}

	var warnBuf bytes.Buffer
	merged, err := MergeWithOptions(src, tgt, MergeUnion, MergeOptions{
		Now:        now,
		WarnWriter: &warnBuf,
	})
	if err != nil {
		t.Fatalf("MergeWithOptions failed: %v", err)
	}

	arr := merged.Concepts["metrics/arr"]
	if arr.Body != "Human reviewed body" {
		t.Errorf("expected non-stale human-reviewed body to win, got %q", arr.Body)
	}
	if arr.Frontmatter.Computation != "" || arr.Frontmatter.Resource != "" || len(arr.Frontmatter.Extra) != 0 {
		t.Errorf("expected no authoritative backfill from lower-trust override, got comp=%q res=%q extra=%v",
			arr.Frontmatter.Computation, arr.Frontmatter.Resource, arr.Frontmatter.Extra)
	}
	if len(arr.Citations) != 1 || arr.Citations[0].URI != "https://example.com/src" {
		t.Errorf("expected citations from winning body, got %v", arr.Citations)
	}

	stale := merged.Concepts["metrics/stale"]
	if stale.Body != "Fresh body" {
		t.Errorf("expected newer revision to replace stale human-reviewed concept, got %q", stale.Body)
	}
	if len(stale.Frontmatter.Verified) != 0 {
		t.Errorf("expected expired human sign-off to be cleared, got %v", stale.Frontmatter.Verified)
	}

	if _, ok := merged.IndexFiles["index.md"]; !ok {
		t.Errorf("expected index.md to be preserved in merged bundle")
	}

	warnings := warnBuf.String()
	if !strings.Contains(warnings, "metrics/arr") || !strings.Contains(warnings, "metrics/stale") {
		t.Errorf("expected merge warnings for both concepts, got %q", warnings)
	}

	// Test WriteBundle preserves IndexFiles and does not inject okf-harvester
	outDir := t.TempDir()
	if err := WriteBundle(merged, outDir); err != nil {
		t.Fatalf("WriteBundle failed: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(outDir, "metrics", "arr.md"))
	if err != nil {
		t.Fatalf("failed to read written concept: %v", err)
	}
	if strings.Contains(string(data), "okf-harvester") {
		t.Errorf("WriteBundle must not inject okf-harvester")
	}
	if _, err := os.Stat(filepath.Join(outDir, "index.md")); err != nil {
		t.Errorf("expected index.md to be written: %v", err)
	}
}
