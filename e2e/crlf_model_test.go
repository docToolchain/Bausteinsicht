package e2e

// #729: a model with CRLF line endings stays CRLF when commands insert text.

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestCRLFModelKeepsItsLineEndings(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")
	path := filepath.Join(dir, "architecture.jsonc")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data = bytes.ReplaceAll(data, []byte("\n"), []byte("\r\n"))
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	runCLI(t, bin, dir, "add", "element", "--id", "crlfx", "--kind", "actor", "--title", "X")
	runCLI(t, bin, dir, "add", "relationship", "--from", "crlfx", "--to", "customer", "--label", "knows")
	runCLI(t, bin, dir, "add", "view", "crlfview", "--title", "V", "--include", "crlfx")

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bare := bytes.Count(after, []byte("\n")) - bytes.Count(after, []byte("\r\n")); bare != 0 {
		t.Errorf("%d line(s) with a bare LF in a CRLF model", bare)
	}
	runCLI(t, bin, dir, "validate")
}
