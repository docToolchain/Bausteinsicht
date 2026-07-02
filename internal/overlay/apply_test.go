package overlay

import (
	"os"
	"strings"
	"testing"

	"github.com/beevik/etree"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

func drawioWithBareCells(cells map[string]string) string {
	root := `<?xml version="1.0"?><mxfile><diagram><mxGraphModel><root>`
	root += `<mxCell id="0"/><mxCell id="1" parent="0"/>`
	for id, style := range cells {
		root += `<mxCell id="` + id + `" style="` + style + `" vertex="1" parent="1"/>`
	}
	root += `</root></mxGraphModel></diagram></mxfile>`
	return root
}

func drawioWithObjects(objects map[string]string) string {
	out := `<?xml version="1.0"?><mxfile><diagram><mxGraphModel><root>`
	out += `<mxCell id="0"/><mxCell id="1" parent="0"/>`
	for bsID, style := range objects {
		out += `<object bausteinsicht_id="` + bsID + `" id="` + bsID + `-obj"><mxCell style="` + style + `" vertex="1" parent="1"/></object>`
	}
	out += `</root></mxGraphModel></diagram></mxfile>`
	return out
}

func writeTempDrawio(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "*.drawio")
	if err != nil {
		t.Fatalf("createTemp: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	return f.Name()
}

func readCellStyle(t *testing.T, path, cellID string) string {
	t.Helper()
	doc := etree.NewDocument()
	if err := doc.ReadFromFile(path); err != nil {
		t.Fatalf("readCellStyle: %v", err)
	}
	for _, cell := range doc.Root().FindElements(".//mxGraphModel/root/mxCell") {
		if cell.SelectAttrValue("id", "") == cellID {
			return cell.SelectAttrValue("style", "")
		}
	}
	for _, obj := range doc.Root().FindElements(".//mxGraphModel/root/object") {
		if obj.SelectAttrValue("bausteinsicht_id", "") == cellID {
			inner := obj.FindElement("mxCell")
			if inner != nil {
				return inner.SelectAttrValue("style", "")
			}
		}
	}
	t.Fatalf("readCellStyle: cell %q not found in %s", cellID, path)
	return ""
}

func readCellAttr(t *testing.T, path, cellID, attr string) string {
	t.Helper()
	doc := etree.NewDocument()
	if err := doc.ReadFromFile(path); err != nil {
		t.Fatalf("readCellAttr: %v", err)
	}
	for _, cell := range doc.Root().FindElements(".//mxGraphModel/root/mxCell") {
		if cell.SelectAttrValue("id", "") == cellID {
			return cell.SelectAttrValue(attr, "")
		}
	}
	for _, obj := range doc.Root().FindElements(".//mxGraphModel/root/object") {
		if obj.SelectAttrValue("bausteinsicht_id", "") == cellID {
			inner := obj.FindElement("mxCell")
			if inner != nil {
				return inner.SelectAttrValue(attr, "")
			}
		}
	}
	return ""
}

func metricsFor(elements map[string]float64) *MetricsFile {
	var ms []ElementMetric
	for id, val := range elements {
		ms = append(ms, ElementMetric{ElementID: id, Values: map[string]float64{"coverage": val}})
	}
	return &MetricsFile{Metrics: ms}
}

// ─── parseStyleParts ─────────────────────────────────────────────────────────

func TestParseStyleParts_Empty(t *testing.T) {
	if parts := parseStyleParts(""); len(parts) != 0 {
		t.Errorf("expected empty, got %v", parts)
	}
}

func TestParseStyleParts_SinglePart(t *testing.T) {
	parts := parseStyleParts("fillColor=#ff0000")
	if len(parts) != 1 || parts[0] != "fillColor=#ff0000" {
		t.Errorf("expected single part, got %v", parts)
	}
}

func TestParseStyleParts_MultiPart(t *testing.T) {
	parts := parseStyleParts("rounded=1;fillColor=#ff0000;strokeColor=#000000;")
	if len(parts) != 3 {
		t.Errorf("expected 3 parts, got %v", parts)
	}
	if parts[0] != "rounded=1" || parts[1] != "fillColor=#ff0000" || parts[2] != "strokeColor=#000000" {
		t.Errorf("unexpected parts: %v", parts)
	}
}

func TestParseStyleParts_TrailingSemicolon(t *testing.T) {
	// Trailing semicolon should not produce an empty last part.
	parts := parseStyleParts("a=1;b=2;")
	if len(parts) != 2 {
		t.Errorf("expected 2 parts, got %v", parts)
	}
}

// ─── startsWithKey ───────────────────────────────────────────────────────────

func TestStartsWithKey(t *testing.T) {
	if !startsWithKey("fillColor=#abc", "fillColor") {
		t.Error("expected true for fillColor prefix")
	}
	if startsWithKey("strokeColor=#abc", "fillColor") {
		t.Error("expected false for strokeColor")
	}
	if startsWithKey("fill", "fillColor") {
		t.Error("expected false for shorter string")
	}
}

// ─── extractFillColor ────────────────────────────────────────────────────────

func TestExtractFillColor_Present(t *testing.T) {
	got := extractFillColor("rounded=1;fillColor=#ff0000;strokeColor=#000")
	if got != "#ff0000" {
		t.Errorf("expected #ff0000, got %q", got)
	}
}

func TestExtractFillColor_Absent(t *testing.T) {
	got := extractFillColor("rounded=1;strokeColor=#000")
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestExtractFillColor_Empty(t *testing.T) {
	if got := extractFillColor(""); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

// ─── updateStyleFill ─────────────────────────────────────────────────────────

func TestUpdateStyleFill_Replace(t *testing.T) {
	style := updateStyleFill("rounded=1;fillColor=#ff0000;", "#00ff00")
	if !strings.Contains(style, "fillColor=#00ff00") {
		t.Errorf("expected new fillColor, got %q", style)
	}
	if strings.Contains(style, "#ff0000") {
		t.Errorf("old fillColor should be gone, got %q", style)
	}
}

func TestUpdateStyleFill_Append(t *testing.T) {
	style := updateStyleFill("rounded=1;", "#00ff00")
	if !strings.Contains(style, "fillColor=#00ff00") {
		t.Errorf("expected fillColor appended, got %q", style)
	}
}

func TestUpdateStyleFill_EmptyStyle(t *testing.T) {
	style := updateStyleFill("", "#abc123")
	if style != "fillColor=#abc123" {
		t.Errorf("expected bare fillColor, got %q", style)
	}
}

// ─── applyColor ──────────────────────────────────────────────────────────────

func TestApplyColor_SetsStyleAndOriginalFill(t *testing.T) {
	cell := etree.NewElement("mxCell")
	cell.CreateAttr("style", "rounded=1;fillColor=#aabbcc;")

	applyColor(cell, 1.0, DefaultColorScheme) // 1.0 → red bucket

	style := cell.SelectAttrValue("style", "")
	if !strings.Contains(style, "fillColor=") {
		t.Errorf("expected fillColor in style, got %q", style)
	}
	orig := cell.SelectAttrValue(OriginalFillAttr, "")
	if orig != "#aabbcc" {
		t.Errorf("expected original fill #aabbcc, got %q", orig)
	}
}

func TestApplyColor_NoExistingFill_UsesWhiteAsOriginal(t *testing.T) {
	cell := etree.NewElement("mxCell")
	cell.CreateAttr("style", "rounded=1;")

	applyColor(cell, 0.0, DefaultColorScheme)

	orig := cell.SelectAttrValue(OriginalFillAttr, "")
	if orig != "#ffffff" {
		t.Errorf("expected #ffffff as default original, got %q", orig)
	}
}

func TestApplyColor_IdempotentOriginalFill(t *testing.T) {
	// Calling applyColor twice must not overwrite data-original-fill.
	cell := etree.NewElement("mxCell")
	cell.CreateAttr("style", "fillColor=#aabbcc;")

	applyColor(cell, 0.5, DefaultColorScheme)
	applyColor(cell, 1.0, DefaultColorScheme) // second call with different value

	orig := cell.SelectAttrValue(OriginalFillAttr, "")
	if orig != "#aabbcc" {
		t.Errorf("original fill overwritten on second call: got %q", orig)
	}
}

// ─── Apply — bare mxCell ─────────────────────────────────────────────────────

func TestApply_BareMxCell(t *testing.T) {
	path := writeTempDrawio(t, drawioWithBareCells(map[string]string{
		"svc-a": "rounded=1;fillColor=#ffffff;",
		"svc-b": "fillColor=#ffffff;",
	}))

	metrics := metricsFor(map[string]float64{
		"svc-a": 100, // best
		"svc-b": 0,   // worst
	})

	if err := Apply(path, metrics, "coverage", DefaultColorScheme); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	styleA := readCellStyle(t, path, "svc-a")
	styleB := readCellStyle(t, path, "svc-b")

	if !strings.Contains(styleA, "fillColor=") {
		t.Errorf("svc-a: expected fillColor in style, got %q", styleA)
	}
	if !strings.Contains(styleB, "fillColor=") {
		t.Errorf("svc-b: expected fillColor in style, got %q", styleB)
	}
	// The heatmap uses a temperature scale: normalized=1.0 → RED (hot), 0.0 → GREEN (cool).
	// coverage=100 normalizes to 1.0 (top of range) → RED; coverage=0 normalizes to 0.0 → GREEN.
	if !strings.Contains(styleA, DefaultColorScheme.Red) {
		t.Errorf("svc-a: expected red %s (normalized 1.0), got style %q", DefaultColorScheme.Red, styleA)
	}
	if !strings.Contains(styleB, DefaultColorScheme.Green) {
		t.Errorf("svc-b: expected green %s (normalized 0.0), got style %q", DefaultColorScheme.Green, styleB)
	}
}

// ─── Apply — <object>-wrapped elements (bausteinsicht format) ─────────────────

func TestApply_ObjectWrapped(t *testing.T) {
	path := writeTempDrawio(t, drawioWithObjects(map[string]string{
		"customer":   "fillColor=#ffffff;",
		"onlineshop": "fillColor=#ffffff;",
	}))

	metrics := metricsFor(map[string]float64{
		"customer":   100,
		"onlineshop": 0,
	})

	if err := Apply(path, metrics, "coverage", DefaultColorScheme); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	styleCustomer := readCellStyle(t, path, "customer")
	styleShop := readCellStyle(t, path, "onlineshop")

	// Temperature scale: normalized=1.0 → RED, 0.0 → GREEN.
	if !strings.Contains(styleCustomer, DefaultColorScheme.Red) {
		t.Errorf("customer: expected red %s (normalized 1.0), got %q", DefaultColorScheme.Red, styleCustomer)
	}
	if !strings.Contains(styleShop, DefaultColorScheme.Green) {
		t.Errorf("onlineshop: expected green %s (normalized 0.0), got %q", DefaultColorScheme.Green, styleShop)
	}
}

func TestApply_ObjectWrapped_SetsOriginalFill(t *testing.T) {
	path := writeTempDrawio(t, drawioWithObjects(map[string]string{
		"api": "rounded=1;fillColor=#aabbcc;",
	}))

	metrics := metricsFor(map[string]float64{"api": 50})

	if err := Apply(path, metrics, "coverage", DefaultColorScheme); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	orig := readCellAttr(t, path, "api", OriginalFillAttr)
	if orig != "#aabbcc" {
		t.Errorf("expected original fill #aabbcc, got %q", orig)
	}
}

// ─── Apply — unknown element ID is silently skipped ──────────────────────────

func TestApply_UnknownElementSkipped(t *testing.T) {
	path := writeTempDrawio(t, drawioWithBareCells(map[string]string{
		"known": "fillColor=#ffffff;",
	}))

	metrics := metricsFor(map[string]float64{"known": 50, "ghost": 50})

	if err := Apply(path, metrics, "coverage", DefaultColorScheme); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	// "known" must be updated; "ghost" is silently ignored (no crash).
	styleKnown := readCellStyle(t, path, "known")
	if !strings.Contains(styleKnown, "fillColor=") {
		t.Errorf("known: expected fillColor in style, got %q", styleKnown)
	}
}

// ─── Remove — bare mxCell ─────────────────────────────────────────────────────

func TestRemove_BareMxCell(t *testing.T) {
	// Pre-apply so data-original-fill is present.
	path := writeTempDrawio(t, drawioWithBareCells(map[string]string{
		"svc": "fillColor=#ffffff;",
	}))
	metrics := metricsFor(map[string]float64{"svc": 80})
	if err := Apply(path, metrics, "coverage", DefaultColorScheme); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	// Remove the overlay.
	if err := Remove(path); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	style := readCellStyle(t, path, "svc")
	if strings.Contains(style, DefaultColorScheme.Green) {
		t.Errorf("Remove: heatmap color still present after remove: %q", style)
	}
	orig := readCellAttr(t, path, "svc", OriginalFillAttr)
	if orig != "" {
		t.Errorf("Remove: data-original-fill still present: %q", orig)
	}
}

// ─── Remove — <object>-wrapped elements ──────────────────────────────────────

func TestRemove_ObjectWrapped(t *testing.T) {
	// Two elements so the span > 0 and both colors are applied.
	path := writeTempDrawio(t, drawioWithObjects(map[string]string{
		"api":  "fillColor=#ffffff;",
		"svc2": "fillColor=#ffffff;",
	}))
	metrics := metricsFor(map[string]float64{"api": 100, "svc2": 0})
	if err := Apply(path, metrics, "coverage", DefaultColorScheme); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if err := Remove(path); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	// After remove, neither heatmap color should remain.
	styleAPI := readCellStyle(t, path, "api")
	if strings.Contains(styleAPI, DefaultColorScheme.Red) {
		t.Errorf("Remove: RED heatmap color still present on api: %q", styleAPI)
	}
	orig := readCellAttr(t, path, "api", OriginalFillAttr)
	if orig != "" {
		t.Errorf("Remove: data-original-fill still present on api: %q", orig)
	}
}

// ─── Apply+Remove roundtrip: fill color is fully restored ────────────────────

func TestApplyRemove_Roundtrip_RestoresFillColor(t *testing.T) {
	const originalFill = "#aabbcc"
	path := writeTempDrawio(t, drawioWithObjects(map[string]string{
		"elem": "rounded=1;fillColor=" + originalFill + ";strokeColor=#000;",
	}))

	metrics := metricsFor(map[string]float64{"elem": 100})
	if err := Apply(path, metrics, "coverage", DefaultColorScheme); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if err := Remove(path); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	style := readCellStyle(t, path, "elem")
	if !strings.Contains(style, "fillColor="+originalFill) {
		t.Errorf("roundtrip: expected original fill %s restored, got %q", originalFill, style)
	}
}

// ─── Apply — missing metrics file ────────────────────────────────────────────

func TestApply_InvalidMetricKey(t *testing.T) {
	path := writeTempDrawio(t, drawioWithBareCells(map[string]string{"a": ""}))
	metrics := metricsFor(map[string]float64{"a": 50})

	err := Apply(path, metrics, "nonexistent_key", DefaultColorScheme)
	if err == nil {
		t.Error("expected error for unknown metric key, got nil")
	}
}
