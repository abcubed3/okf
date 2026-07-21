// Package lsp implements an OKF Language Server Protocol (LSP) server.
// It provides real-time diagnostics for OKF concept markdown files as they
// are edited inside any LSP-compatible editor (VS Code, Neovim, etc.).
package lsp

import (
	stdcontext "context"
	"net/url"
	"path/filepath"
	"strings"
	"sync"

	"github.com/abcubed3/okf/pkg/bundle"
	"github.com/abcubed3/okf/pkg/parser"
	"github.com/abcubed3/okf/pkg/validator"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/tliron/glsp/server"
)

// Server holds all LSP server state. Each field is safe for concurrent access
// via the embedded mutex. Using a struct eliminates the package-level global
// variable anti-pattern and makes the server testable in isolation.
type Server struct {
	mu            sync.RWMutex
	workspaceRoot string
	activeBundle  *bundle.Bundle
}

// NewServer allocates a new LSP Server instance.
func NewServer() *Server {
	return &Server{}
}

// Run starts the LSP server on Standard I/O. This call blocks until the editor
// closes the connection.
func Run() {
	s := NewServer()
	handler := s.buildHandler()
	srv := server.NewServer(handler, "okf-lsp", true)
	_ = srv.RunStdio()
}

// buildHandler constructs the protocol.Handler, binding each LSP method to a
// method on this Server instance (avoiding package-level globals).
func (s *Server) buildHandler() *protocol.Handler {
	return &protocol.Handler{
		Initialize:            s.initialize,
		Initialized:           s.initialized,
		Shutdown:              shutdown,
		SetTrace:              setTrace,
		TextDocumentDidOpen:   s.textDocumentDidOpen,
		TextDocumentDidChange: s.textDocumentDidChange,
		TextDocumentDidSave:   s.textDocumentDidSave,
		TextDocumentDidClose:  s.textDocumentDidClose,
	}
}

// initialize handles the LSP 'initialize' request.
// It advertises the server capabilities and captures the workspace root URI.
func (s *Server) initialize(context *glsp.Context, params *protocol.InitializeParams) (any, error) {
	capabilities := protocol.ServerCapabilities{
		TextDocumentSync: protocol.TextDocumentSyncKindFull,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if params.RootURI != nil {
		s.workspaceRoot = uriToPath(*params.RootURI)
	} else if params.RootPath != nil {
		s.workspaceRoot = *params.RootPath
	}

	return protocol.InitializeResult{
		Capabilities: capabilities,
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    "okf-lsp",
			Version: func(v string) *string { return &v }("0.1.0"),
		},
	}, nil
}

// initialized handles the LSP 'initialized' notification sent after initialize completes.
// It performs an initial bundle parse so diagnostics are ready from the moment the editor opens.
func (s *Server) initialized(context *glsp.Context, params *protocol.InitializedParams) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.workspaceRoot != "" {
		if b, err := parser.ParseBundle(stdcontext.Background(), s.workspaceRoot); err == nil {
			s.activeBundle = b
		}
	}
	return nil
}

// shutdown handles the LSP 'shutdown' request.
func shutdown(context *glsp.Context) error {
	protocol.SetTraceValue(protocol.TraceValueOff)
	return nil
}

// setTrace handles the LSP '$/setTrace' notification.
func setTrace(context *glsp.Context, params *protocol.SetTraceParams) error {
	protocol.SetTraceValue(params.Value)
	return nil
}

func (s *Server) textDocumentDidOpen(context *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
	return s.parseAndValidate(context, params.TextDocument.URI, params.TextDocument.Text)
}

func (s *Server) textDocumentDidChange(context *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
	if len(params.ContentChanges) == 0 {
		return nil
	}
	change, ok := params.ContentChanges[0].(protocol.TextDocumentContentChangeEventWhole)
	if !ok {
		return nil
	}
	return s.parseAndValidate(context, params.TextDocument.URI, change.Text)
}

func (s *Server) textDocumentDidSave(context *glsp.Context, params *protocol.DidSaveTextDocumentParams) error {
	if params.Text != nil {
		return s.parseAndValidate(context, params.TextDocument.URI, *params.Text)
	}
	return nil
}

func (s *Server) textDocumentDidClose(context *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
	context.Notify(protocol.ServerTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
		URI:         params.TextDocument.URI,
		Diagnostics: []protocol.Diagnostic{},
	})
	return nil
}

// parseAndValidate parses the given concept file content, updates the in-memory
// bundle, validates the single concept, and publishes diagnostics to the editor.
func (s *Server) parseAndValidate(context *glsp.Context, uri string, content string) error {
	s.mu.RLock()
	root := s.workspaceRoot
	s.mu.RUnlock()

	if root == "" {
		return nil
	}

	filePath := uriToPath(uri)
	relPath, err := filepath.Rel(root, filePath)
	if err != nil {
		return nil
	}
	conceptID := strings.TrimSuffix(filepath.ToSlash(relPath), ".md")

	c, err := parser.ParseConceptReader(strings.NewReader(content), relPath, conceptID)
	if err != nil {
		c = &bundle.Concept{
			ID:         conceptID,
			Path:       relPath,
			ParseError: err.Error(),
		}
	}

	s.mu.Lock()
	if s.activeBundle == nil {
		s.activeBundle = bundle.NewBundle(root)
	}
	s.activeBundle.Concepts[conceptID] = c
	b := s.activeBundle
	s.mu.Unlock()

	issues := validator.ValidateConcept(c, b, validator.Options{}, make(map[string]map[string]bool))

	var diagnostics []protocol.Diagnostic
	for _, issue := range issues {
		severity := protocol.DiagnosticSeverityError
		if issue.Severity == validator.SeverityWarning {
			severity = protocol.DiagnosticSeverityWarning
		}
		var line protocol.UInteger = 0
		diagnostic := protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{Line: line, Character: 0},
				End:   protocol.Position{Line: line, Character: 100},
			},
			Severity: &severity,
			Source:   func(s string) *string { return &s }("okf"),
			Message:  issue.Message,
		}
		diagnostics = append(diagnostics, diagnostic)
	}

	if diagnostics == nil {
		diagnostics = []protocol.Diagnostic{}
	}

	context.Notify(protocol.ServerTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diagnostics,
	})

	return nil
}

func uriToPath(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		path := strings.TrimPrefix(uri, "file://")
		return path
	}
	return u.Path
}
