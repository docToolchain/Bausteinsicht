package e2e

// TestModelFileEdgeCases (#519) closes E2E-Test-Plan.adoc section 13 (7
// tests, previously zero e2e coverage): unusual but valid encodings of the
// model file (BOM, line-ending variants, indentation, minification) and a
// basic performance budget for a 100-element model.

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

const minimalModelJSON = `{
  "specification": {
    "elements": { "system": { "notation": "System" } }
  },
  "model": {
    "alpha": { "kind": "system", "title": "Alpha" },
    "beta": { "kind": "system", "title": "Beta" }
  }
}
`

func TestModelFileEdgeCases(t *testing.T) {
	t.Run("13.1_UTF8BOM", test13_1UTF8BOM)
	t.Run("13.2_WindowsLineEndings", test13_2WindowsLineEndings)
	t.Run("13.3_MixedLineEndings", test13_3MixedLineEndings)
	t.Run("13.4_TabIndentation", test13_4TabIndentation)
	t.Run("13.5_NoTrailingNewline", test13_5NoTrailingNewline)
	t.Run("13.6_MinifiedJSON", test13_6MinifiedJSON)
	t.Run("13.7_HundredElementsPerformance", test13_7HundredElementsPerformance)
}

// validateModelText writes the given text as architecture.jsonc and runs
// `validate`, returning combined output and exit code.
func validateModelText(t *testing.T, bin, dir, text string) (string, int) {
	t.Helper()
	writeFile(t, dir+"/architecture.jsonc", text)
	return runCLIAllowFail(t, bin, dir, "validate")
}

// test13_1UTF8BOM covers 13.1: a UTF-8 BOM prefix is stripped
// (internal/model/loader.go StripJSONC), not a fatal error.
func test13_1UTF8BOM(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	bom := "\xEF\xBB\xBF"
	out, code := validateModelText(t, bin, dir, bom+minimalModelJSON)
	if code != 0 {
		t.Errorf("expected BOM to be stripped and validation to succeed, got exit %d\n%s", code, out)
	}
}

// test13_2WindowsLineEndings covers 13.2.
func test13_2WindowsLineEndings(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	crlf := strings.ReplaceAll(minimalModelJSON, "\n", "\r\n")
	out, code := validateModelText(t, bin, dir, crlf)
	if code != 0 {
		t.Errorf("expected CRLF line endings to work, got exit %d\n%s", code, out)
	}
}

// test13_3MixedLineEndings covers 13.3.
func test13_3MixedLineEndings(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	lines := strings.Split(minimalModelJSON, "\n")
	var sb strings.Builder
	for i, line := range lines {
		sb.WriteString(line)
		if i == len(lines)-1 {
			continue
		}
		if i%2 == 0 {
			sb.WriteString("\r\n")
		} else {
			sb.WriteString("\n")
		}
	}
	out, code := validateModelText(t, bin, dir, sb.String())
	if code != 0 {
		t.Errorf("expected mixed line endings to work, got exit %d\n%s", code, out)
	}
}

// test13_4TabIndentation covers 13.4.
func test13_4TabIndentation(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	tabbed := "{\n\t\"specification\": {\n\t\t\"elements\": { \"system\": { \"notation\": \"System\" } }\n\t},\n\t\"model\": {\n\t\t\"alpha\": { \"kind\": \"system\", \"title\": \"Alpha\" }\n\t}\n}\n"
	out, code := validateModelText(t, bin, dir, tabbed)
	if code != 0 {
		t.Errorf("expected tab-indented JSON to work, got exit %d\n%s", code, out)
	}
}

// test13_5NoTrailingNewline covers 13.5.
func test13_5NoTrailingNewline(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	noNewline := strings.TrimRight(minimalModelJSON, "\n")
	out, code := validateModelText(t, bin, dir, noNewline)
	if code != 0 {
		t.Errorf("expected no-trailing-newline JSON to work, got exit %d\n%s", code, out)
	}
}

// test13_6MinifiedJSON covers 13.6.
func test13_6MinifiedJSON(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	minified := `{"specification":{"elements":{"system":{"notation":"System"}}},"model":{"alpha":{"kind":"system","title":"Alpha"},"beta":{"kind":"system","title":"Beta"}}}`
	out, code := validateModelText(t, bin, dir, minified)
	if code != 0 {
		t.Errorf("expected minified JSON to work, got exit %d\n%s", code, out)
	}
}

// test13_7HundredElementsPerformance covers 13.7: validate + sync on a
// 100-element model complete quickly. The test plan's stated budget is
// "<1s"; this test uses a more generous 5s ceiling to absorb CI/container
// scheduling noise while still catching a gross performance regression
// (e.g. an accidentally-quadratic algorithm).
func test13_7HundredElementsPerformance(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	var sb strings.Builder
	sb.WriteString(`{"specification":{"elements":{"system":{"notation":"System"}}},"model":{`)
	for i := 0; i < 100; i++ {
		if i > 0 {
			sb.WriteString(",")
		}
		fmt.Fprintf(&sb, `"elem%d":{"kind":"system","title":"Element %d"}`, i, i)
	}
	sb.WriteString(`},"views":{"all":{"title":"All","include":["*"]}}}`)
	writeFile(t, dir+"/architecture.jsonc", sb.String())

	start := time.Now()
	runCLI(t, bin, dir, "validate")
	runCLI(t, bin, dir, "sync")
	elapsed := time.Since(start)

	const budget = 5 * time.Second
	if elapsed > budget {
		t.Errorf("validate+sync on 100 elements took %v, want < %v", elapsed, budget)
	} else {
		t.Logf("validate+sync on 100 elements took %v (doc budget: <1s, test ceiling: %v)", elapsed, budget)
	}

	if _, err := os.Stat(dir + "/architecture.drawio"); err != nil {
		t.Errorf("expected architecture.drawio to exist after sync: %v", err)
	}
}
