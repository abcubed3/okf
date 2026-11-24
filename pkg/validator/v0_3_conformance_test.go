package validator

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abcubed3/okf/pkg/parser"
)

func TestValidateV03ConformanceRules(t *testing.T) {
	root := t.TempDir()
	writeFile := func(relPath, content string) {
		t.Helper()
		full := filepath.Join(root, filepath.FromSlash(relPath))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Coexisting repository files must be ignored without errors
	writeFile("README.md", "# Repo README\n")
	writeFile("AGENTS.md", "# Agent Rules\n")
	writeFile("SKILL.md", "---\nname: companion-skill\ndescription: Skill manifest without type\n---\n# Skill\n")

	// Valid root index.md with v0.3 bundle metadata
	writeFile("index.md", `---
okf_version: "0.3"
title: "Acme Finance Bundle"
description: "Curated metrics and attested computations."
license: "CC-BY-4.0"
---
# Acme Finance Bundle

- [Net Revenue](metrics/net-revenue.md) - Active metric
`)

	// Invalid nested index.md with frontmatter -> ERROR
	writeFile("metrics/index.md", "---\ntitle: Forbidden Nested Frontmatter\n---\n# Metrics\n")

	// Invalid log.md with frontmatter -> ERROR
	writeFile("log.md", "---\ntitle: Bad Log\n---\n# Update Log\n\n## Nov 18 2026\n\n- Added metrics.\n")

	// Active replacement metric with footnote & GFM table link
	writeFile("metrics/net-revenue.md", `---
type: Metric
title: Net Revenue
description: Active net revenue metric.
sources:
  - id: rev-policy
    resource: https://example.com/policy
    last_modified: 2026-11-01
generated:
  by: reference_agent/gemini
  at: 2026-11-19T12:00:00Z
verified:
  - by: human:alice
    at: 2026-11-18T09:00:00Z
---
# Net Revenue

Defined by policy.[^rev-policy] Unmatched footnote.[^missing-id]

| Related | Link |
| --- | --- |
| Computation | [Comp](../computations/rev.md) |

[^rev-policy]: Policy
`)

	// Deprecated concept with valid superseded_by
	writeFile("metrics/old-revenue.md", `---
type: Metric
title: Old Revenue
description: Superseded metric stub.
status: deprecated
superseded_by: /metrics/net-revenue.md
---
# Old Revenue
`)

	// Attested Computation missing computation body/path and missing parameter type
	writeFile("computations/rev.md", `---
type: Attested Computation
title: Revenue Query
description: Computes revenue.
runtime: bigquery
parameters:
  - name: year
---
# Overview
No computation fence here.
`)

	b, err := parser.ParseBundle(context.Background(), root)
	if err != nil {
		t.Fatalf("ParseBundle failed: %v", err)
	}

	issues := ValidateBundle(b, Options{})

	hasIssue := func(pathSub, msgSub string, sev IssueSeverity) bool {
		for _, iss := range issues {
			if strings.Contains(filepath.ToSlash(iss.Path), pathSub) &&
				strings.Contains(iss.Message, msgSub) &&
				iss.Severity == sev {
				return true
			}
		}
		return false
	}

	if !hasIssue("metrics/index.md", "nested index.md must not contain YAML frontmatter", SeverityError) {
		t.Errorf("expected nested index.md frontmatter ERROR, got: %+v", issues)
	}
	if !hasIssue("log.md", "log.md must not contain YAML frontmatter", SeverityError) {
		t.Errorf("expected log.md frontmatter ERROR, got: %+v", issues)
	}
	if !hasIssue("log.md", "ISO 8601 YYYY-MM-DD", SeverityWarning) {
		t.Errorf("expected log.md date heading WARNING, got: %+v", issues)
	}
	if !hasIssue("metrics/net-revenue.md", "missing time and UTC offset", SeverityWarning) {
		t.Errorf("expected date-only RFC3339 WARNING on sources[0].last_modified, got: %+v", issues)
	}
	if !hasIssue("metrics/net-revenue.md", "older than 'generated.at'", SeverityWarning) {
		t.Errorf("expected outdated verification WARNING, got: %+v", issues)
	}
	if !hasIssue("metrics/net-revenue.md", "footnote '[^missing-id]'", SeverityWarning) {
		t.Errorf("expected unmatched footnote WARNING, got: %+v", issues)
	}
	if !hasIssue("metrics/old-revenue.md", "superseded by '/metrics/net-revenue.md'", SeverityWarning) {
		t.Errorf("expected deprecation with superseded_by WARNING, got: %+v", issues)
	}
	if !hasIssue("computations/rev.md", "parameters[0] is missing required 'type'", SeverityError) {
		t.Errorf("expected parameter missing type ERROR, got: %+v", issues)
	}
	if !hasIssue("computations/rev.md", "neither an inline '# Computation'", SeverityWarning) {
		t.Errorf("expected missing computation locator WARNING, got: %+v", issues)
	}

	// Ensure README.md, AGENTS.md, SKILL.md caused zero issues
	for _, iss := range issues {
		base := filepath.Base(iss.Path)
		if base == "README.md" || base == "AGENTS.md" || base == "SKILL.md" {
			t.Errorf("unexpected issue on skipped non-concept file %q: %+v", iss.Path, iss)
		}
	}
}

func TestV03Validator_FootnotesLogOrderCyclesAndMissingAt(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("index.md", "---\nokf_version: \"0.3\"\n---\n# Test\n")
	write("log.md", "# Log\n\n## 2026-01-01\n- Older entry first\n\n## 2026-11-18\n- Newer entry second\n")

	write("metrics/no_sources.md", `---
type: Metric
title: Footnote Without Sources
description: Uses footnote without declaring sources.
---

Claim with unmatched footnote.[^missing-src]
Inline multi-backtick span: `+"``code ` [^ignored-span]``"+`

~~~
[^ignored-tilde-fence]
~~~
`)

	write("metrics/dup_source.md", `---
type: Metric
title: Duplicate Source ID
description: Has duplicate source IDs and missing verified.at.
sources:
  - id: s1
    resource: https://example.com/1
  - id: s1
    resource: https://example.com/2
generated:
  by: okf-harvester/v0.3
  at: 2026-11-18T10:00:00Z
verified:
  - by: human:alice
---

Body.[^s1]

[^s1]: Source 1
`)

	write("metrics/a.md", `---
type: Metric
title: Metric A
description: Deprecated A.
status: deprecated
superseded_by: /metrics/b.md
---
# A
`)
	write("metrics/b.md", `---
type: Metric
title: Metric B
description: Deprecated B.
status: deprecated
superseded_by: /metrics/a.md
---
# B
`)

	b, err := parser.ParseBundle(context.Background(), root)
	if err != nil {
		t.Fatalf("ParseBundle failed: %v", err)
	}

	issues := ValidateBundle(b, Options{})
	var allMsgs []string
	for _, iss := range issues {
		allMsgs = append(allMsgs, iss.Path+": "+iss.Message)
	}
	joined := strings.Join(allMsgs, "\n")

	for _, want := range []string{
		"reverse chronological order",
		"footnote '[^missing-src]' does not match any 'sources[].id' entry",
		`duplicate 'sources[].id' value "s1"`,
		"verified[0] record is missing 'at' timestamp",
		"'superseded_by' chain forms a cycle",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("expected issue containing %q, got:\n%s", want, joined)
		}
	}

	for _, unwanted := range []string{
		"ignored-span",
		"ignored-tilde-fence",
		"older than 'generated.at'",
	} {
		if strings.Contains(joined, unwanted) {
			t.Errorf("did not expect issue containing %q, got:\n%s", unwanted, joined)
		}
	}
}
