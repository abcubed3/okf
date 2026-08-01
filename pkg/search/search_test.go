package search

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

type mockTransport struct {
	fn func(req *http.Request) (*http.Response, error)
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.fn(req)
}

func TestSearchBundles(t *testing.T) {
	mockBundles := []BundleResult{
		{
			ID:          "stripe/api",
			Name:        "Stripe API",
			Description: "Stripe integration bundle",
			Namespace:   "stripe",
			IsPrivate:   false,
			Version:     "1.0.0",
			Tags:        []string{"stripe", "payments"},
		},
	}

	data, _ := json.Marshal(mockBundles)

	client := &http.Client{
		Transport: &mockTransport{
			fn: func(req *http.Request) (*http.Response, error) {
				if req.URL.Query().Get("q") != "stripe" {
					t.Errorf("expected query 'stripe', got '%s'", req.URL.Query().Get("q"))
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(data)),
					Header:     make(http.Header),
				}, nil
			},
		},
	}

	results, err := SearchBundlesWithClient("stripe", "", "http://localhost:8080", "test-api-key", client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].ID != "stripe/api" {
		t.Errorf("expected bundle ID 'stripe/api', got '%s'", results[0].ID)
	}
}
