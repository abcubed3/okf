package lsp

import (
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

var (
	handler       protocol.Handler
	workspaceRoot string
	activeBundle  *bundle.Bundle
	bundleMutex   sync.RWMutex
)

func init() {
	handler = protocol.Handler{
		Initialize:            initialize,
		Initialized:           initialized,
		Shutdown:              shutdown,
		SetTrace:              setTrace,
		TextDocumentDidOpen:    textDocumentDidOpen,
		TextDocumentDidChange:  textDocumentDidChange,
		TextDocumentDidSave:    textDocumentDidSave,
		TextDocumentDidClose:   textDocumentDidClose,
		TextDocumentHover:      textDocumentHover,
		TextDocumentDefinition: textDocumentDefinition,
	}
}

// Run starts the LSP server on Standard I/O
func Run() {
	server := server.NewServer(&handler, "okf-lsp", true)
	server.RunStdio()
}

func initialize(context *glsp.Context, params *protocol.InitializeParams) (any, error) {
	capabilities := handler.CreateServerCapabilities()

	// We support full text document sync for now
	capabilities.TextDocumentSync = protocol.TextDocumentSyncKindFull
	capabilities.HoverProvider = true
	capabilities.DefinitionProvider = true

	if params.RootURI != nil {
		workspaceRoot = strings.TrimPrefix(*params.RootURI, "file://")
	} else if params.RootPath != nil {
		workspaceRoot = *params.RootPath
	}

	return protocol.InitializeResult{
		Capabilities: capabilities,
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    "okf-lsp",
			Version: func(s string) *string { return &s }("0.1.0"),
		},
	}, nil
}

func initialized(context *glsp.Context, params *protocol.InitializedParams) error {
	bundleMutex.Lock()
	defer bundleMutex.Unlock()
	if workspaceRoot != "" {
		b, err := parser.ParseBundle(workspaceRoot)
		if err == nil {
			activeBundle = b
		}
	}
	return nil
}

func shutdown(context *glsp.Context) error {
	protocol.SetTraceValue(protocol.TraceValueOff)
	return nil
}

func setTrace(context *glsp.Context, params *protocol.SetTraceParams) error {
	protocol.SetTraceValue(params.Value)
	return nil
}

func textDocumentDidOpen(context *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
	return parseAndValidate(context, params.TextDocument.URI, params.TextDocument.Text)
}

func textDocumentDidChange(context *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
	if len(params.ContentChanges) > 0 {
		text := params.ContentChanges[0].(protocol.TextDocumentContentChangeEventWhole).Text
		return parseAndValidate(context, params.TextDocument.URI, text)
	}
	return nil
}

func textDocumentDidSave(context *glsp.Context, params *protocol.DidSaveTextDocumentParams) error {
	if params.Text != nil {
		return parseAndValidate(context, params.TextDocument.URI, *params.Text)
	}
	// If text is not provided, we could read from disk, but for full sync it's usually tracked.
	return nil
}

func textDocumentDidClose(context *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
	context.Notify(protocol.ServerTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
		URI:         params.TextDocument.URI,
		Diagnostics: []protocol.Diagnostic{},
	})
	return nil
}

func parseAndValidate(context *glsp.Context, uri string, content string) error {
	if workspaceRoot == "" {
		return nil
	}

	relPath := strings.TrimPrefix(strings.TrimPrefix(uri, "file://"), workspaceRoot+"/")
	conceptID := strings.TrimSuffix(relPath, ".md")

	c, err := parser.ParseConceptReader(strings.NewReader(content), relPath, conceptID)
	if err != nil {
		c = &bundle.Concept{
			ID:         conceptID,
			Path:       relPath,
			ParseError: err.Error(),
		}
	}

	bundleMutex.Lock()
	if activeBundle == nil {
		activeBundle = bundle.NewBundle(workspaceRoot)
	}
	activeBundle.Concepts[conceptID] = c
	b := activeBundle
	bundleMutex.Unlock()

	issues := validator.ValidateBundle(b)

	diagnosticsByURI := make(map[string][]protocol.Diagnostic)
	for _, issue := range issues {
		issueURI := "file://" + filepath.Join(workspaceRoot, issue.Path)
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
		diagnosticsByURI[issueURI] = append(diagnosticsByURI[issueURI], diagnostic)
	}

	for u, diagnostics := range diagnosticsByURI {
		context.Notify(protocol.ServerTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
			URI:         u,
			Diagnostics: diagnostics,
		})
	}

	return nil
}

func textDocumentHover(context *glsp.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	// A real implementation would parse the line content and find the concept ID under the cursor.
	// For simplicity, we just return a placeholder or look up if we know the hovered word.
	// This requires maintaining the text content to extract the word, which we omit for brevity.
	// As a placeholder, we'll return a simple hover if the user hovers over a file we know.
	return nil, nil
}

func textDocumentDefinition(context *glsp.Context, params *protocol.DefinitionParams) (any, error) {
	// A real implementation would extract the concept link under the cursor.
	// For simplicity in this demo, we return nil if no link is found.
	return nil, nil
}
