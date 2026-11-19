package validator

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/abcubed3/okf/pkg/bundle"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestHubURIValidationAndIndexParseError(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Query().Get("bundle") != "acme/core" {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader("")),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"concepts":[{"id":"tables/users"}]}`)),
			}, nil
		}),
	}

	b := bundle.NewBundle(t.TempDir())
	b.IndexFiles["index.md"] = &bundle.Concept{
		ID:         "index",
		Path:       "index.md",
		ParseError: "unclosed frontmatter '---' fence",
	}
	b.Concepts["docs/ok"] = &bundle.Concept{
		ID:          "docs/ok",
		Path:        "docs/ok.md",
		Frontmatter: bundle.Frontmatter{Type: "Doc", Title: "OK", Desc: "Desc"},
		Body:        "See [users](hub://acme/core/tables/users.md) and [bad](hub://single) and [missing](hub://acme/core/tables/orders).",
	}

	issues := ValidateBundle(b, Options{Host: "https://hub.example.com", HTTPClient: client})
	var hasIdxErr, hasMalformedErr, hasMissingErr bool
	for _, iss := range issues {
		if iss.Path == "index.md" && iss.Severity == SeverityError && strings.Contains(iss.Message, "unclosed") {
			hasIdxErr = true
		}
		if iss.Severity == SeverityError && strings.Contains(iss.Message, "malformed hub:// URI") {
			hasMalformedErr = true
		}
		if iss.Severity == SeverityError && strings.Contains(iss.Message, "tables/orders") {
			hasMissingErr = true
		}
		if strings.Contains(iss.Message, "tables/users") {
			t.Errorf("valid nested hub:// concept should not fail: %+v", iss)
		}
	}
	if !hasIdxErr || !hasMalformedErr || !hasMissingErr {
		t.Errorf("missing expected issues: idx=%v malformed=%v missing=%v; all=%+v", hasIdxErr, hasMalformedErr, hasMissingErr, issues)
	}

	// Unreachable host should emit SeverityWarning, not SeverityError, for valid hub:// URIs
	offlineClient := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return nil, errors.New("connection refused")
		}),
	}
	offlineIssues := ValidateBundle(b, Options{Host: "https://hub.example.com", HTTPClient: offlineClient})
	for _, iss := range offlineIssues {
		if strings.Contains(iss.Message, "tables/users") && iss.Severity != SeverityWarning {
			t.Errorf("expected SeverityWarning when hub is unreachable, got %+v", iss)
		}
	}
}
