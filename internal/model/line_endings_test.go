package model

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const lfDoc = "{\n  \"model\": {\n    \"a\": {\n      \"kind\": \"system\"\n    }\n  },\n  \"relationships\": [\n    {\"from\": \"a\", \"to\": \"a\"}\n  ]\n}\n"

func toCRLF(s string) string { return strings.ReplaceAll(s, "\n", "\r\n") }

func bareLF(data []byte) int {
	return bytes.Count(data, []byte("\n")) - bytes.Count(data, []byte("\r\n"))
}

// Text that is inserted into a file with CRLF line endings has CRLF line endings
// too, also inside a value that spans several lines; a file with LF endings stays
// free of CR (#729).
func TestInsert_FollowsTheLineEndingsOfTheFile(t *testing.T) {
	multiLine := "{\n    \"kind\": \"system\",\n    \"title\": \"B\"\n  }"
	for name, doc := range map[string]string{"CRLF": toCRLF(lfDoc), "LF": lfDoc} {
		out, err := InsertObjectEntry([]byte(doc), []string{"model"}, "b", multiLine)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		out, err = AppendArrayEntry(out, []string{"relationships"}, "{\"from\": \"a\",\n \"to\": \"b\"}")
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if name == "CRLF" && bareLF(out) != 0 {
			t.Errorf("CRLF: %d line(s) with a bare LF in\n%q", bareLF(out), out)
		}
		if name == "LF" && bytes.Contains(out, []byte("\r")) {
			t.Errorf("LF: a CR was inserted into\n%q", out)
		}
		if _, err := ParseModel(out); err != nil {
			t.Errorf("%s: the result does not parse: %v", name, err)
		}
	}
}

// A file with both kinds of line endings gets the one that most of its lines have.
func TestInsert_MixedFileFollowsTheMajority(t *testing.T) {
	doc := toCRLF(lfDoc)
	doc = strings.Replace(doc, "\r\n", "\n", 1) // one LF-only line in a CRLF file
	out, err := InsertObjectEntry([]byte(doc), []string{"model"}, "b", `{"kind": "system"}`)
	if err != nil {
		t.Fatal(err)
	}
	if got := bareLF(out) - bareLF([]byte(doc)); got != 0 {
		t.Errorf("the inserted lines must use CRLF, %d bare LF added", got)
	}
}

// The fallback that rewrites the whole file keeps the line endings too.
func TestSave_KeepsCRLF(t *testing.T) {
	path := filepath.Join(t.TempDir(), "m.jsonc")
	doc := toCRLF("// header\n{\n  \"specification\": {\"elements\": {\"system\": {\"notation\": \"S\"}}},\n  \"model\": {\"a\": {\"kind\": \"system\", \"title\": \"A\"}}\n}\n")
	if err := os.WriteFile(path, []byte(doc), 0o600); err != nil {
		t.Fatal(err)
	}
	m, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := Save(path, m); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if bareLF(data) != 0 || !bytes.Contains(data, []byte("\r\n")) {
		t.Errorf("a CRLF file must stay CRLF after Save, got %d bare LF in\n%q", bareLF(data), data)
	}
}
