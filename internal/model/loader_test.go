package model

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_ValidModelWithComments(t *testing.T) {
	m, err := Load("testdata/valid-model.jsonc")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if m.Specification.Elements["system"].Notation != "System" {
		t.Errorf("expected element notation 'System', got '%s'", m.Specification.Elements["system"].Notation)
	}
	if m.Model["mySystem"].Title != "My System" {
		t.Errorf("expected title 'My System', got '%s'", m.Model["mySystem"].Title)
	}
}

func TestLoad_ValidModelWithTrailingCommas(t *testing.T) {
	m, err := Load("testdata/valid-model.jsonc")
	if err != nil {
		t.Fatalf("expected no error loading model with trailing commas, got: %v", err)
	}
	if len(m.Model) != 2 {
		t.Errorf("expected 2 model elements, got %d", len(m.Model))
	}
}

func TestLoad_MinimalModel(t *testing.T) {
	m, err := Load("testdata/minimal-model.jsonc")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(m.Specification.Elements) != 1 {
		t.Errorf("expected 1 element kind, got %d", len(m.Specification.Elements))
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load("testdata/nonexistent.jsonc")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	_, err := Load("testdata/invalid-json.jsonc")
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestSave_RoundTrip(t *testing.T) {
	original, err := Load("testdata/minimal-model.jsonc")
	if err != nil {
		t.Fatalf("failed to load original: %v", err)
	}

	tmp := filepath.Join(t.TempDir(), "model.jsonc")
	if err := Save(tmp, original); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	reloaded, err := Load(tmp)
	if err != nil {
		t.Fatalf("failed to reload: %v", err)
	}
	if reloaded.Model["root"].Title != original.Model["root"].Title {
		t.Errorf("round-trip mismatch: expected '%s', got '%s'", original.Model["root"].Title, reloaded.Model["root"].Title)
	}
}

func TestSave_AtomicWrite(t *testing.T) {
	original, err := Load("testdata/minimal-model.jsonc")
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}

	dir := t.TempDir()
	target := filepath.Join(dir, "model.jsonc")
	if err := Save(target, original); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	// Temp file should have been cleaned up
	tmpFile := target + ".tmp"
	if _, err := os.Stat(tmpFile); !os.IsNotExist(err) {
		t.Error("expected .tmp file to be removed after atomic write")
	}
}

func TestAutoDetect_FindsJSONCFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "model.jsonc")

	original, err := Load("testdata/minimal-model.jsonc")
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}
	if err := Save(target, original); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	found, err := AutoDetect(dir)
	if err != nil {
		t.Fatalf("expected to find .jsonc file, got error: %v", err)
	}
	if found != target {
		t.Errorf("expected '%s', got '%s'", target, found)
	}
}

func TestAutoDetect_NoJSONCFile(t *testing.T) {
	dir := t.TempDir()
	_, err := AutoDetect(dir)
	if err == nil {
		t.Error("expected error when no .jsonc file found")
	}
}

func TestStripJSONC_RemovesLineComments(t *testing.T) {
	input := []byte(`{
  "key": "value" // comment
}`)
	out := StripJSONC(input)
	expected := `{
  "key": "value"
}`
	if string(out) != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, string(out))
	}
}

func TestStripJSONC_PreservesCommentInString(t *testing.T) {
	input := []byte(`{"key": "value // not a comment"}`)
	out := StripJSONC(input)
	if string(out) != `{"key": "value // not a comment"}` {
		t.Errorf("comment inside string should not be stripped, got: %s", string(out))
	}
}

func TestStripJSONC_RemovesBlockComments(t *testing.T) {
	input := []byte(`{
  /* this is a block comment */
  "key": "value"
}`)
	out := StripJSONC(input)
	expected := `{

  "key": "value"
}`
	if string(out) != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, string(out))
	}
}

func TestStripJSONC_RemovesInlineBlockComment(t *testing.T) {
	input := []byte(`{"key": /* comment */ "value"}`)
	out := StripJSONC(input)
	expected := `{"key":  "value"}`
	if string(out) != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, string(out))
	}
}

func TestStripJSONC_PreservesBlockCommentInString(t *testing.T) {
	input := []byte(`{"key": "value /* not a comment */"}`)
	out := StripJSONC(input)
	if string(out) != `{"key": "value /* not a comment */"}` {
		t.Errorf("block comment inside string should not be stripped, got: %s", string(out))
	}
}

func TestStripJSONC_MultilineBlockComment(t *testing.T) {
	input := []byte(`{
  /* multi
     line
     comment */
  "key": "value"
}`)
	out := StripJSONC(input)
	expected := `{

  "key": "value"
}`
	if string(out) != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, string(out))
	}
}

func TestStripJSONC_RemovesTrailingCommas(t *testing.T) {
	input := []byte(`{"a": 1, "b": 2,}`)
	out := StripJSONC(input)
	expected := `{"a": 1, "b": 2}`
	if string(out) != expected {
		t.Errorf("expected trailing comma removed, got: %s", string(out))
	}
}

func TestStripJSONC_RemovesTrailingCommaBeforeArrayEnd(t *testing.T) {
	input := []byte(`[1, 2, 3,]`)
	out := StripJSONC(input)
	expected := `[1, 2, 3]`
	if string(out) != expected {
		t.Errorf("expected trailing comma in array removed, got: %s", string(out))
	}
}

func TestStripJSONC_StripsBOM(t *testing.T) {
	bom := []byte{0xEF, 0xBB, 0xBF}
	input := append(bom, []byte(`{"key": "value"}`)...)
	out := StripJSONC(input)
	expected := `{"key": "value"}`
	if string(out) != expected {
		t.Errorf("expected BOM stripped, got: %q", string(out))
	}
}

func TestLoad_ModelWithBOM(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "model.jsonc")

	// Read a valid model, prepend BOM, write it out
	original, err := os.ReadFile("testdata/minimal-model.jsonc")
	if err != nil {
		t.Fatalf("failed to read testdata: %v", err)
	}
	bom := []byte{0xEF, 0xBB, 0xBF}
	withBOM := append(bom, original...)
	if err := os.WriteFile(path, withBOM, 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	m, err := Load(path)
	if err != nil {
		t.Fatalf("expected no error for BOM file, got: %v", err)
	}
	if len(m.Model) == 0 {
		t.Error("expected non-empty model")
	}
}

func TestSave_PreservesPreambleComments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "model.jsonc")

	// Write a model file with preamble comments before the root `{`
	preamble := "// Architecture model for Acme Corp\n// Author: Jane Doe\n"
	original, err := Load("testdata/minimal-model.jsonc")
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}
	// First save without preamble
	if err := Save(path, original); err != nil {
		t.Fatalf("failed to save: %v", err)
	}
	// Manually prepend preamble to simulate a user-edited file
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read: %v", err)
	}
	if err := os.WriteFile(path, append([]byte(preamble), data...), 0644); err != nil {
		t.Fatalf("failed to write with preamble: %v", err)
	}

	// Save again — preamble should be preserved
	if err := Save(path, original); err != nil {
		t.Fatalf("second save failed: %v", err)
	}

	result, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read result: %v", err)
	}
	if string(result[:len(preamble)]) != preamble {
		t.Errorf("preamble lost after save.\nExpected prefix:\n%s\nGot:\n%s", preamble, string(result[:80]))
	}
}

func TestSave_NoPreambleWhenFileStartsWithBrace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "model.jsonc")

	original, err := Load("testdata/minimal-model.jsonc")
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}
	if err := Save(path, original); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	// Save again — file starts with `{`, no preamble should be added
	if err := Save(path, original); err != nil {
		t.Fatalf("second save failed: %v", err)
	}

	result, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read result: %v", err)
	}
	if result[0] != '{' {
		t.Errorf("expected file to start with '{', got '%c'", result[0])
	}
}

func TestLoad_ElementOrder(t *testing.T) {
	m, err := Load("testdata/valid-model.jsonc")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	// valid-model.jsonc defines "system" before "container"
	want := []string{"system", "container"}
	if len(m.ElementOrder) != len(want) {
		t.Fatalf("expected ElementOrder %v, got %v", want, m.ElementOrder)
	}
	for i, w := range want {
		if m.ElementOrder[i] != w {
			t.Errorf("ElementOrder[%d] = %q, want %q", i, m.ElementOrder[i], w)
		}
	}
}

func TestLoad_NullJSONRoot(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "null-model.jsonc")
	if err := os.WriteFile(path, []byte("null"), 0644); err != nil {
		t.Fatalf("failed to write null model: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Error("expected error for null JSON root, got nil")
	}
}

func TestLoad_RejectsOversizedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "huge.jsonc")
	data := make([]byte, MaxModelFileSize+1)
	for i := range data {
		data[i] = ' '
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Error("expected error for oversized file, got nil")
	}
}

func TestAutoDetect_ErrorOnMultipleFiles(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.jsonc", "b.jsonc"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(`{"specification":{}}`), 0600); err != nil {
			t.Fatal(err)
		}
	}
	_, err := AutoDetect(dir)
	if err == nil {
		t.Error("expected error when multiple .jsonc files found without explicit --model")
	}
}
