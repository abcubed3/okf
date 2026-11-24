package links

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseBundleID(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"stripe/api", "stripe/api", false},
		{"hub://stripe/api", "stripe/api", false},
		{"/stripe/api/", "stripe/api", false},
		{"stripe", "", true},
		{"stripe/api/extra", "", true},
		{"/api", "", true},
		{"", "", true},
	}
	for _, tc := range cases {
		got, err := ParseBundleID(tc.in)
		if (err != nil) != tc.wantErr {
			t.Errorf("ParseBundleID(%q) err=%v, wantErr=%v", tc.in, err, tc.wantErr)
		}
		if got != tc.want {
			t.Errorf("ParseBundleID(%q)=%q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestParseHubURI(t *testing.T) {
	cases := []struct {
		in          string
		wantBundle  string
		wantConcept string
		wantErr     bool
	}{
		{"hub://stripe/api", "stripe/api", "", false},
		{"hub://stripe/api/charges/create", "stripe/api", "charges/create", false},
		{"hub://stripe/api/charges/create.md#params", "stripe/api", "charges/create", false},
		{"hub://onlyone", "", "", true},
		{"hub://stripe/api/", "", "", true},
		{"hub://stripe/../secret", "", "", true},
		{"stripe/api", "", "", true},
	}
	for _, tc := range cases {
		b, c, err := ParseHubURI(tc.in)
		if (err != nil) != tc.wantErr {
			t.Errorf("ParseHubURI(%q) err=%v, wantErr=%v", tc.in, err, tc.wantErr)
		}
		if b != tc.wantBundle || c != tc.wantConcept {
			t.Errorf("ParseHubURI(%q)=(%q, %q), want (%q, %q)", tc.in, b, c, tc.wantBundle, tc.wantConcept)
		}
	}
}

func TestResolveBundlePath_V03AndLegacyFallback(t *testing.T) {
	root := t.TempDir()
	refFile := filepath.Join(root, "references", "skills", "run-on-bq.md")
	if err := os.MkdirAll(filepath.Dir(refFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(refFile, []byte("# Run\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// 1. Leading slash resolves from bundle root in v0.3
	got, escaped := ResolveBundlePath(root, "computations/revenue.md", "/references/skills/run-on-bq.md#sec", "0.3")
	if escaped || got != "references/skills/run-on-bq.md" {
		t.Errorf("v0.3 root-anchored: got (%q, %v)", got, escaped)
	}

	// 2. Bare path in v0.3 resolves strictly relative to containing file's directory
	gotV03Bare, escaped := ResolveBundlePath(root, "computations/revenue.md", "references/skills/run-on-bq.md", "0.3")
	if escaped || gotV03Bare != "computations/references/skills/run-on-bq.md" {
		t.Errorf("v0.3 bare relative: got (%q, %v)", gotV03Bare, escaped)
	}

	// 3. Bare path in legacy v0.2 falls back to bundle root when file exists there
	gotV02Bare, escaped := ResolveBundlePath(root, "computations/revenue.md", "references/skills/run-on-bq.md", "0.2")
	if escaped || gotV02Bare != "references/skills/run-on-bq.md" {
		t.Errorf("v0.2 fallback: got (%q, %v)", gotV02Bare, escaped)
	}

	// 4. Escaping "../" above root is flagged
	_, escaped = ResolveBundlePath(root, "computations/revenue.md", "../../outside.md", "0.3")
	if !escaped {
		t.Errorf("expected escaped=true for ../../outside.md")
	}

	// 5. Multiple leading slashes normalize without a leading "/"
	gotMultiSlash, escaped := ResolveBundlePath(root, "computations/revenue.md", "///references/skills/run-on-bq.md", "0.3")
	if escaped || gotMultiSlash != "references/skills/run-on-bq.md" {
		t.Errorf("multiple leading slashes: got (%q, %v)", gotMultiSlash, escaped)
	}

	// 6. Protocol-relative URLs (//example.com/...) are treated as external scheme URIs
	if !IsExternal("//example.com/spec.md") || !IsSchemeURI("//example.com/spec.md") {
		t.Errorf("expected protocol-relative URL //example.com/spec.md to be classified as external")
	}
}
