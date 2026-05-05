package lsp

import (
	"encoding/json"
	"testing"
)

func TestURIToPath(t *testing.T) {
	tests := []struct {
		uri      string
		expected string
	}{
		{"file:///home/user/arch.jsonc", "/home/user/arch.jsonc"},
		{"/home/user/arch.jsonc", "/home/user/arch.jsonc"},
	}

	for _, tt := range tests {
		result := URIToPath(tt.uri)
		if result != tt.expected {
			t.Errorf("URIToPath(%q) = %q, want %q", tt.uri, result, tt.expected)
		}
	}
}

func TestHandleInitialize(t *testing.T) {
	server := NewServer()
	params := &InitializeParams{RootPath: "/tmp"}

	response := server.handleInitialize(1, params)
	if response == nil {
		t.Error("expected non-nil response")
	}

	msg, ok := response.(*JSONRPCMessage)
	if !ok {
		t.Fatalf("expected JSONRPCMessage, got %T", response)
	}

	if msg.JSONRPC != "2.0" {
		t.Errorf("expected JSONRPC 2.0, got %s", msg.JSONRPC)
	}

	if msg.ID != 1 {
		t.Errorf("expected ID 1, got %v", msg.ID)
	}

	result, ok := msg.Result.(InitializeResult)
	if !ok {
		t.Fatalf("expected InitializeResult, got %T", msg.Result)
	}

	if result.Capabilities.TextDocumentSync != 2 {
		t.Errorf("expected TextDocumentSync 2, got %d", result.Capabilities.TextDocumentSync)
	}

	if !result.Capabilities.DiagnosticProvider {
		t.Error("expected DiagnosticProvider to be true")
	}
}

func TestHandleDidOpen(t *testing.T) {
	server := NewServer()
	uri := "file:///tmp/architecture.jsonc"
	content := `{"model": {}}`

	server.handleDidOpen(&uri, &content)

	doc, ok := server.documents[uri]
	if !ok {
		t.Error("expected document to be stored")
	}

	if doc.Content != content {
		t.Errorf("expected content %q, got %q", content, doc.Content)
	}

	if doc.Version != 1 {
		t.Errorf("expected version 1, got %d", doc.Version)
	}
}

func TestHandleDidChange(t *testing.T) {
	server := NewServer()
	uri := "file:///tmp/architecture.jsonc"
	originalContent := `{"model": {}}`
	newContent := `{"model": {"svc": {}}}`

	// First open the document
	server.handleDidOpen(&uri, &originalContent)

	// Then change it
	server.handleDidChange(&uri, 2, newContent)

	doc := server.documents[uri]
	if doc.Content != newContent {
		t.Errorf("expected content %q, got %q", newContent, doc.Content)
	}

	if doc.Version != 2 {
		t.Errorf("expected version 2, got %d", doc.Version)
	}
}

func TestMessageParsing(t *testing.T) {
	jsonStr := `{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "initialize",
		"params": {"rootPath": "/tmp"}
	}`

	var msg JSONRPCMessage
	if err := json.Unmarshal([]byte(jsonStr), &msg); err != nil {
		t.Fatalf("failed to parse message: %v", err)
	}

	if msg.JSONRPC != "2.0" {
		t.Errorf("expected JSONRPC 2.0, got %s", msg.JSONRPC)
	}

	if msg.Method != "initialize" {
		t.Errorf("expected method initialize, got %s", msg.Method)
	}

	// JSON unmarshals numeric IDs as float64
	if id, ok := msg.ID.(float64); !ok || id != 1.0 {
		t.Errorf("expected ID 1.0, got %v", msg.ID)
	}
}
