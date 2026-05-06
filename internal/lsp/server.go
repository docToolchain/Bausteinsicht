package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

type Server struct {
	documents map[string]*Document
	workDir   string
	modelPath string
}

type Document struct {
	URI      string
	Content  string
	Version  int
	Text     string
	Filename string
}

type JSONRPCMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  interface{}     `json:"result,omitempty"`
	Error   interface{}     `json:"error,omitempty"`
}

type InitializeParams struct {
	RootPath  string `json:"rootPath"`
	RootURI   string `json:"rootUri"`
	Workspace struct {
		WorkspaceFolders []struct {
			URI  string `json:"uri"`
			Name string `json:"name"`
		} `json:"workspaceFolders"`
	} `json:"workspace"`
}

type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
}

type ServerCapabilities struct {
	TextDocumentSync int    `json:"textDocumentSync"`
	DiagnosticProvider bool `json:"diagnosticProvider"`
	CodeLensProvider struct {
		CodeLensOptions
	} `json:"codeLensProvider"`
}

type CodeLensOptions struct {
	ResolveProvider bool `json:"resolveProvider"`
}

func NewServer() *Server {
	return &Server{
		documents: make(map[string]*Document),
		workDir:   ".",
	}
}

func (s *Server) Run() error {
	return s.readMessages()
}

func (s *Server) readMessages() error {
	reader := bufio.NewReader(os.Stdin)

	for {
		// Read headers
		headers := make(map[string]string)
		for {
			line, err := reader.ReadString('\n')
			if err != nil && err != io.EOF {
				return err
			}
			line = strings.TrimSpace(line)
			if line == "" {
				break
			}
			parts := strings.Split(line, ": ")
			if len(parts) == 2 {
				headers[parts[0]] = parts[1]
			}
		}

		// Read body
		contentLength := headers["Content-Length"]
		if contentLength == "" {
			continue
		}

		length, err := strconv.Atoi(contentLength)
		if err != nil {
			continue
		}

		body := make([]byte, length)
		_, err = reader.Read(body)
		if err != nil && err != io.EOF {
			return err
		}

		// Parse and handle message
		var msg JSONRPCMessage
		if err := json.Unmarshal(body, &msg); err != nil {
			log.Printf("Failed to parse message: %v", err)
			continue
		}

		// Handle message
		response := s.handleMessage(&msg)
		if response != nil {
			s.sendMessage(response)
		}
	}
}

func (s *Server) handleMessage(msg *JSONRPCMessage) interface{} {
	switch msg.Method {
	case "initialize":
		var params InitializeParams
		if err := json.Unmarshal(msg.Params, &params); err != nil {
			return nil
		}
		return s.handleInitialize(msg.ID, &params)

	case "initialized":
		return nil

	case "textDocument/didOpen":
		var params struct {
			TextDocument struct {
				URI  string `json:"uri"`
				Text string `json:"text"`
			} `json:"textDocument"`
		}
		if err := json.Unmarshal(msg.Params, &params); err != nil {
			return nil
		}
		s.handleDidOpen(&params.TextDocument.URI, &params.TextDocument.Text)
		return nil

	case "textDocument/didChange":
		var params struct {
			TextDocument struct {
				URI     string `json:"uri"`
				Version int    `json:"version"`
			} `json:"textDocument"`
			ContentChanges []struct {
				Text string `json:"text"`
			} `json:"contentChanges"`
		}
		if err := json.Unmarshal(msg.Params, &params); err != nil {
			return nil
		}
		if len(params.ContentChanges) > 0 {
			s.handleDidChange(&params.TextDocument.URI, params.TextDocument.Version, params.ContentChanges[0].Text)
		}
		return nil

	case "textDocument/didSave":
		var params struct {
			TextDocument struct {
				URI string `json:"uri"`
			} `json:"textDocument"`
		}
		if err := json.Unmarshal(msg.Params, &params); err != nil {
			return nil
		}
		s.handleDidSave(&params.TextDocument.URI)
		return nil

	case "shutdown":
		// Send response before exiting (LSP spec requirement)
		response := &JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Result:  map[string]interface{}{},
		}
		s.sendMessage(response)
		os.Exit(0)

	default:
		return nil
	}

	return nil
}

func (s *Server) handleInitialize(id interface{}, params *InitializeParams) interface{} {
	if params.RootPath != "" {
		s.workDir = params.RootPath
	}
	// Auto-detect model file
	s.detectModel()

	return &JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      id,
		Result: InitializeResult{
			Capabilities: ServerCapabilities{
				TextDocumentSync: 2, // Full document sync
				DiagnosticProvider: true,
				CodeLensProvider: struct {
					CodeLensOptions
				}{CodeLensOptions{ResolveProvider: false}},
			},
		},
	}
}

func (s *Server) handleDidOpen(uri *string, text *string) {
	if uri == nil || text == nil {
		return
	}

	filename := URIToPath(*uri)
	doc := &Document{
		URI:      *uri,
		Content:  *text,
		Text:     *text,
		Version:  1,
		Filename: filename,
	}
	s.documents[*uri] = doc

	// Publish diagnostics if this is the model file
	if s.isModelFile(filename) {
		s.publishDiagnostics(uri)
	}
}

func (s *Server) handleDidChange(uri *string, version int, text string) {
	if uri == nil {
		return
	}

	doc, ok := s.documents[*uri]
	if !ok {
		return
	}

	doc.Content = text
	doc.Text = text
	doc.Version = version

	if s.isModelFile(doc.Filename) {
		s.publishDiagnostics(uri)
	}
}

func (s *Server) handleDidSave(uri *string) {
	if uri == nil {
		return
	}

	doc, ok := s.documents[*uri]
	if !ok {
		return
	}

	if s.isModelFile(doc.Filename) {
		s.publishDiagnostics(uri)
	}
}

func (s *Server) publishDiagnostics(uri *string) {
	doc, ok := s.documents[*uri]
	if !ok || doc == nil {
		return
	}

	diags := ValidateDocument(doc, s.workDir)

	params := map[string]interface{}{
		"uri":         *uri,
		"diagnostics": diags,
	}
	paramsData, _ := json.Marshal(params)

	msg := &JSONRPCMessage{
		JSONRPC: "2.0",
		Method:  "textDocument/publishDiagnostics",
		Params:  paramsData,
	}

	s.sendMessage(msg)
}

func (s *Server) sendMessage(msg interface{}) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal message: %v", err)
		return
	}

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	_, _ = os.Stdout.WriteString(header)
	_, _ = os.Stdout.Write(data)
}

func (s *Server) detectModel() {
	// Look for architecture.jsonc in work directory
	modelPath := filepath.Join(s.workDir, "architecture.jsonc")
	if _, err := os.Stat(modelPath); err == nil {
		s.modelPath = modelPath
	}
}

func (s *Server) isModelFile(filename string) bool {
	base := filepath.Base(filename)
	return strings.Contains(base, "architecture") && strings.HasSuffix(base, ".jsonc")
}

func URIToPath(uri string) string {
	// Parse URI to handle cross-platform paths and URL-encoded characters
	u, err := url.Parse(uri)
	if err != nil {
		// Fall back to simple prefix removal on parse error
		if strings.HasPrefix(uri, "file://") {
			return uri[7:]
		}
		return uri
	}

	// Extract path from parsed URI
	path := filepath.FromSlash(u.Path)

	// On Windows, remove leading slash from absolute paths (C:/path not /C:/path)
	if runtime.GOOS == "windows" && len(path) > 0 && path[0] == '/' && len(path) > 2 && path[2] == ':' {
		path = path[1:]
	}

	return path
}
