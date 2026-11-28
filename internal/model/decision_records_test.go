package model

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func decisionModel(d DecisionRecord) *BausteinsichtModel {
	return &BausteinsichtModel{
		Specification: Specification{
			Elements:  map[string]ElementKind{"system": {Notation: "S"}},
			Decisions: []DecisionRecord{d},
		},
		Model: map[string]Element{"s": {Kind: "system", Title: "S", Decisions: []string{d.ID}}},
	}
}

func hasMessage(ws []ValidationWarning, parts ...string) bool {
	for _, w := range ws {
		ok := true
		for _, p := range parts {
			ok = ok && strings.Contains(w.Path+" "+w.Message, p)
		}
		if ok {
			return true
		}
	}
	return false
}

// The status of a decision is one of the known ones (accepted is the usual ADR
// status); anything else is reported (#707).
func TestValidate_DecisionStatus(t *testing.T) {
	for _, status := range []ADRStatus{"proposed", "accepted", "active", "deprecated", "superseded", ""} {
		r := ValidateWithWarnings(decisionModel(DecisionRecord{ID: "ADR-1", Title: "T", Status: status}))
		if hasMessage(r.Warnings, "status") {
			t.Errorf("status %q is known and must not be reported: %v", status, r.Warnings)
		}
	}
	r := ValidateWithWarnings(decisionModel(DecisionRecord{ID: "ADR-1", Title: "T", Status: "foo"}))
	if !hasMessage(r.Warnings, "ADR-1", `"foo"`, "status") {
		t.Errorf("want a warning about the status \"foo\", got %v", r.Warnings)
	}
}

func TestValidate_DecisionDate(t *testing.T) {
	r := ValidateWithWarnings(decisionModel(DecisionRecord{ID: "ADR-1", Title: "T", Date: "2026-01-31"}))
	if hasMessage(r.Warnings, "date") {
		t.Errorf("a valid date is not reported: %v", r.Warnings)
	}
	r = ValidateWithWarnings(decisionModel(DecisionRecord{ID: "ADR-1", Title: "T", Date: "bad-date"}))
	if !hasMessage(r.Warnings, "ADR-1", `"bad-date"`, "date") {
		t.Errorf("want a warning about the date, got %v", r.Warnings)
	}
}

// The file of a decision is a path inside the project: absolute paths and paths
// that leave the project are errors (#707).
func TestValidate_DecisionFileMustStayInTheProject(t *testing.T) {
	for _, file := range []string{"../etc/passwd", "docs/../../x.adoc", "/etc/passwd", `C:\Windows\x.adoc`} {
		r := ValidateWithWarnings(decisionModel(DecisionRecord{ID: "ADR-1", Title: "T", FilePath: file}))
		found := false
		for _, e := range r.Errors {
			found = found || (strings.Contains(e.Message, "outside the project") && strings.Contains(e.Path, "ADR-1"))
		}
		if !found {
			t.Errorf("file %q: want an error \"outside the project\", got %v", file, r.Errors)
		}
	}
	for _, file := range []string{"docs/adr1.adoc", "./docs/adr1.adoc", "docs/../adr1.adoc", "https://example.com/adr1"} {
		r := ValidateWithWarnings(decisionModel(DecisionRecord{ID: "ADR-1", Title: "T", FilePath: file}))
		for _, e := range r.Errors {
			if strings.Contains(e.Message, "outside the project") {
				t.Errorf("file %q is inside the project or a URL, got error %v", file, e)
			}
		}
	}
}

// A file that does not exist is reported once the model knows where it lies (#707).
func TestValidate_DecisionFileMustExist(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs", "adr1.adoc"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	m := decisionModel(DecisionRecord{ID: "ADR-1", Title: "T", FilePath: "docs/adr1.adoc"})
	m.BaseDir = dir
	if r := ValidateWithWarnings(m); hasMessage(r.Warnings, "does not exist") {
		t.Errorf("the file exists: %v", r.Warnings)
	}
	m = decisionModel(DecisionRecord{ID: "ADR-2", Title: "T", FilePath: "docs/nope.adoc"})
	m.BaseDir = dir
	if r := ValidateWithWarnings(m); !hasMessage(r.Warnings, "ADR-2", "docs/nope.adoc", "does not exist") {
		t.Errorf("want a warning for the missing file, got %v", r.Warnings)
	}
	// without a base directory (a model parsed from text) nothing is checked
	m = decisionModel(DecisionRecord{ID: "ADR-3", Title: "T", FilePath: "docs/nope.adoc"})
	if r := ValidateWithWarnings(m); hasMessage(r.Warnings, "does not exist") {
		t.Errorf("no base directory, no file check: %v", r.Warnings)
	}
}

// Load tells the model where it lies, so that the files can be found.
func TestLoad_SetsTheBaseDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "m.jsonc")
	if err := os.WriteFile(path, []byte(`{"specification":{"elements":{"system":{"notation":"S"}}},"model":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	m, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if m.BaseDir != dir {
		t.Errorf("want BaseDir %q, got %q", dir, m.BaseDir)
	}
}

func TestDecisionBadgeColor_Accepted(t *testing.T) {
	if DecisionBadgeColor("accepted") != DecisionBadgeColor(ADRActive) {
		t.Error("an accepted decision has the colour of an active one")
	}
}
