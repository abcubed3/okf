package lsp

import (
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestNewServer(t *testing.T) {
	s := NewServer()
	if s == nil {
		t.Fatal("expected non-nil LSP server instance")
	}
	handler := s.buildHandler()
	if handler == nil {
		t.Fatal("expected non-nil protocol handler")
	}
}

func TestUriToPath(t *testing.T) {
	tests := []struct {
		uri      string
		expected string
	}{
		{"file:///path/to/concept.md", "/path/to/concept.md"},
		{"file://localhost/path/to/concept.md", "/path/to/concept.md"},
	}

	for _, tt := range tests {
		got := uriToPath(tt.uri)
		if got != tt.expected {
			t.Errorf("uriToPath(%q) = %q, expected %q", tt.uri, got, tt.expected)
		}
	}
}

func TestServer_Initialize(t *testing.T) {
	s := NewServer()
	rootURI := "file:///tmp/sample-bundle"
	res, err := s.initialize(nil, &protocol.InitializeParams{
		RootURI: &rootURI,
	})
	if err != nil {
		t.Fatalf("initialize failed: %v", err)
	}

	initRes, ok := res.(protocol.InitializeResult)
	if !ok {
		t.Fatalf("expected InitializeResult, got %T", res)
	}

	if initRes.ServerInfo == nil || initRes.ServerInfo.Name != "okf-lsp" {
		t.Errorf("unexpected ServerInfo: %v", initRes.ServerInfo)
	}

	if s.workspaceRoot != "/tmp/sample-bundle" {
		t.Errorf("expected workspaceRoot to be set, got %q", s.workspaceRoot)
	}
}
