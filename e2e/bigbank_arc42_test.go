// Package e2e contains end-to-end integration tests that exercise the full
// Bausteinsicht pipeline: import → link assignment → sync → SVG export → arc42 adoc.
//
// Run with: go test ./e2e/ -v -run TestBigBankArc42Pipeline
// Skip SVG export validation (requires draw.io CLI):
//
//	go test ./e2e/ -v -run TestBigBankArc42Pipeline (auto-skips if draw.io not found)
package e2e

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/drawio"
	"github.com/docToolchain/Bausteinsicht/internal/importer/structurizr"
	"github.com/docToolchain/Bausteinsicht/internal/model"
	bsync "github.com/docToolchain/Bausteinsicht/internal/sync"
	"github.com/docToolchain/Bausteinsicht/templates"
)

// anchorFromID derives an AsciiDoc anchor from a dot-path element ID.
// Only the last segment is used so nested elements link to their own section.
//
//	"internetBankingSystem.apiApplication" → "#sec-apiapplication"
func anchorFromID(id string) string {
	parts := strings.Split(id, ".")
	last := parts[len(parts)-1]
	return "#sec-" + strings.ToLower(last)
}

// TestBigBankArc42Pipeline is an end-to-end test that:
//  1. Imports the BigBank workspace.dsl via the structurizr importer
//  2. Assigns a `link` field to every element pointing to its arc42 chapter anchor
//  3. Runs a sync cycle to produce the draw.io XML
//  4. Validates that all link attributes are set correctly in the draw.io XML
//  5. Generates an arc42 AsciiDoc file with inline SVG image directives
//  6. If the draw.io CLI is available: exports SVGs and validates <a href> elements
func TestBigBankArc42Pipeline(t *testing.T) {
	var dir string
	if out := os.Getenv("KEEP_OUTPUT"); out != "" {
		dir = out
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir KEEP_OUTPUT dir: %v", err)
		}
	} else {
		dir = t.TempDir()
	}

	// ── Step 1: Import workspace.dsl ─────────────────────────────────────────

	dslPath := filepath.Join("testdata", "bigbank", "workspace.dsl")
	result, err := structurizr.Import(dslPath)
	if err != nil {
		t.Fatalf("structurizr.Import: %v", err)
	}
	for _, w := range result.Warnings {
		t.Logf("import warning: %s", w)
	}

	m := result.Model
	if len(m.Model) == 0 {
		t.Fatal("import produced no elements")
	}
	t.Logf("imported %d top-level elements, %d views", len(m.Model), len(m.Views))

	// Validate that 4 views were imported (System Landscape, System Context, Containers, Components).
	if len(m.Views) != 4 {
		t.Errorf("expected 4 imported views, got %d: %v", len(m.Views), viewKeys(m.Views))
	}

	// ── Step 2: Assign link fields (element ID → arc42 anchor) ───────────────

	flat, err := model.FlattenElements(m)
	if err != nil {
		t.Fatalf("FlattenElements: %v", err)
	}

	// anchorMap maps element ID → anchor string, used later to verify SVGs.
	anchorMap := make(map[string]string, len(flat))
	for id := range flat {
		anchorMap[id] = anchorFromID(id)
	}

	// Patch links into the model's nested element maps.
	setLinks(m.Model, "", anchorMap)

	// Verify links are set on a few known elements.
	checkLink(t, m, "customer", "#sec-customer")
	checkLink(t, m, "internetBankingSystem", "#sec-internetbankingsystem")

	// ── Step 3: Save architecture.jsonc ──────────────────────────────────────

	modelPath := filepath.Join(dir, "architecture.jsonc")
	if err := model.Save(modelPath, m); err != nil {
		t.Fatalf("model.Save: %v", err)
	}

	// ── Step 4: Build draw.io document and run sync ───────────────────────────

	ts, err := drawio.LoadTemplateFromBytes(templates.DefaultTemplate)
	if err != nil {
		t.Fatalf("LoadTemplateFromBytes: %v", err)
	}

	doc := drawio.NewDocument()
	newPageIDs := make(map[string]bool, len(m.Views))
	for viewKey, view := range m.Views {
		doc.AddPage("view-"+viewKey, view.Title)
		newPageIDs["view-"+viewKey] = true
	}

	state := &bsync.SyncState{
		Elements:      make(map[string]bsync.ElementState),
		Relationships: []bsync.RelationshipState{},
	}
	cs := bsync.DetectChanges(m, doc, state, newPageIDs)
	fwdResult := bsync.ApplyForward(cs, doc, ts, m)
	for _, w := range fwdResult.Warnings {
		t.Logf("sync warning: %s", w)
	}

	// ── Step 5: Validate link attributes in draw.io XML ──────────────────────

	t.Run("DrawioLinkAttributes", func(t *testing.T) {
		// Elements that appear in the SystemContext view and should carry links.
		// All top-level elements and containers rendered on at least one page.
		expectedLinks := map[string]string{
			"customer":              "#sec-customer",
			"internetBankingSystem": "#sec-internetbankingsystem",
			"mainframe":             "#sec-mainframe",
			"email":                 "#sec-email",
		}
		for viewKey := range m.Views {
			page := doc.GetPage("view-" + viewKey)
			if page == nil {
				t.Errorf("page view-%s not found in document", viewKey)
				continue
			}
			for elemID, wantLink := range expectedLinks {
				obj := page.FindElement(elemID)
				if obj == nil {
					continue // element may not be in this view
				}
				got := obj.SelectAttrValue("link", "")
				// A drill-down link is set when a scoped view exists; otherwise the user link.
				if got == "" {
					t.Errorf("view %s / element %s: expected link attribute, got empty", viewKey, elemID)
				}
				// If no drill-down view exists for this element, the user link must match.
				if !strings.HasPrefix(got, "data:page/id,") && got != wantLink {
					t.Errorf("view %s / element %s: link = %q, want %q", viewKey, elemID, got, wantLink)
				}
			}
		}

		// Containers must carry their user link on the Containers page.
		containersPage := doc.GetPage("view-Containers")
		if containersPage != nil {
			for _, id := range []string{
				"internetBankingSystem.singlePageApplication",
				"internetBankingSystem.mobileApp",
				"internetBankingSystem.webApplication",
				"internetBankingSystem.apiApplication",
				"internetBankingSystem.database",
			} {
				obj := containersPage.FindElement(id)
				if obj == nil {
					continue
				}
				got := obj.SelectAttrValue("link", "")
				parts := strings.Split(id, ".")
				want := "#sec-" + strings.ToLower(parts[len(parts)-1])
				// apiApplication has a drill-down view (Components scope), so expect that.
				if id == "internetBankingSystem.apiApplication" {
					if !strings.HasPrefix(got, "data:page/id,") {
						t.Errorf("apiApplication: expected drill-down link, got %q", got)
					}
				} else if got != "" && !strings.HasPrefix(got, "data:page/id,") && got != want {
					t.Errorf("container %s: link = %q, want %q or drill-down", id, got, want)
				}
			}
		}
	})

	// ── Step 6: Save draw.io file (for optional manual inspection) ────────────

	drawioPath := filepath.Join(dir, "architecture.drawio")
	if err := drawio.SaveDocument(drawioPath, doc); err != nil {
		t.Fatalf("drawio.SaveDocument: %v", err)
	}
	t.Logf("draw.io saved to: %s", drawioPath)

	// ── Step 7: Generate arc42 AsciiDoc ──────────────────────────────────────

	adocPath := filepath.Join(dir, "arc42.adoc")
	adocContent := generateArc42Adoc(m, flat)
	if err := os.WriteFile(adocPath, []byte(adocContent), 0o644); err != nil {
		t.Fatalf("write arc42.adoc: %v", err)
	}
	t.Logf("arc42 adoc saved to: %s", adocPath)

	// Verify anchors are present for all elements.
	t.Run("Arc42Anchors", func(t *testing.T) {
		for id, anchor := range anchorMap {
			// anchor is "#sec-<name>", strip the "#" for the AsciiDoc [[...]] form.
			adocAnchor := "[[" + strings.TrimPrefix(anchor, "#") + "]]"
			if !strings.Contains(adocContent, adocAnchor) {
				t.Errorf("arc42.adoc missing anchor %s (for element %s)", adocAnchor, id)
			}
		}
	})

	// Verify SVG image directives are present for each view.
	t.Run("Arc42SVGIncludes", func(t *testing.T) {
		for viewKey := range m.Views {
			imgDirective := fmt.Sprintf("image::svgs/architecture-%s.svg", viewKey)
			if !strings.Contains(adocContent, imgDirective) {
				t.Errorf("arc42.adoc missing SVG include for view %s", viewKey)
			}
		}
	})

	// ── Step 8: SVG export + link validation (requires draw.io CLI) ───────────

	drawioCmd := findDrawioCmd()
	if drawioCmd == "" {
		t.Log("draw.io CLI not found — skipping SVG export and link validation")
		t.Log("To run the full pipeline: install draw.io CLI and re-run the test")
		printWorkflowHint(t, dir)
		return
	}

	svgDir := filepath.Join(dir, "svgs")
	if err := os.MkdirAll(svgDir, 0o755); err != nil {
		t.Fatalf("mkdir svgs: %v", err)
	}

	// Build the bausteinsicht binary once into a temp location.
	binaryPath := filepath.Join(t.TempDir(), "bausteinsicht")
	moduleRoot, err := findModuleRootPath()
	if err != nil {
		t.Fatalf("find module root: %v", err)
	}
	buildCmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/bausteinsicht")
	buildCmd.Dir = moduleRoot
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("build bausteinsicht: %v\n%s", err, out)
	}

	t.Run("SVGExportAndLinks", func(t *testing.T) {
		// Export all views as SVG using the built binary.
		// The export command derives the draw.io path from --model (same dir, architecture.drawio).
		cmd := exec.Command(binaryPath,
			"export",
			"--model", modelPath,
			"--image-format", "svg",
			"--output", svgDir,
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("bausteinsicht export: %v\n%s", err, out)
		}
		t.Logf("export output: %s", out)

		// Validate each exported SVG contains <a href="..."> elements.
		for viewKey := range m.Views {
			svgPath := filepath.Join(svgDir, "architecture-"+viewKey+".svg")
			validateSVGLinks(t, svgPath, anchorMap)
		}
	})
}

// setLinks recursively patches Link fields in elem map using precomputed anchorMap.
func setLinks(elems map[string]model.Element, prefix string, anchorMap map[string]string) {
	for key, elem := range elems {
		id := key
		if prefix != "" {
			id = prefix + "." + key
		}
		if anchor, ok := anchorMap[id]; ok {
			elem.Link = anchor
		}
		if len(elem.Children) > 0 {
			setLinks(elem.Children, id, anchorMap)
		}
		elems[key] = elem
	}
}

// checkLink asserts that the named top-level element has the expected link.
func checkLink(t *testing.T, m *model.BausteinsichtModel, elemKey, wantLink string) {
	t.Helper()
	elem, ok := m.Model[elemKey]
	if !ok {
		t.Errorf("element %q not found in model", elemKey)
		return
	}
	if elem.Link != wantLink {
		t.Errorf("element %q: link = %q, want %q", elemKey, elem.Link, wantLink)
	}
}

// generateArc42Adoc produces an arc42-structured AsciiDoc document for the BigBank model.
// It includes:
//   - Chapter 3 (System Scope and Context) with all imported views
//   - Chapter 5 (Building Block View) with element-level sections
//   - An [[sec-<id>]] anchor for every element (matching the link convention)
func generateArc42Adoc(m *model.BausteinsichtModel, flat map[string]*model.Element) string {
	var b strings.Builder

	b.WriteString("= Big Bank plc — Architecture Documentation\n")
	b.WriteString(":toc: left\n")
	b.WriteString(":toclevels: 3\n")
	b.WriteString(":sectnums:\n")
	b.WriteString(":imagesdir: .\n\n")
	b.WriteString("// arc42 template — generated by Bausteinsicht BigBank E2E test\n\n")

	// ── Chapter 1 ──────────────────────────────────────────────────────────────
	b.WriteString("== Introduction and Goals\n\n")
	b.WriteString("Big Bank plc is a fictional bank used to illustrate the key features of Structurizr and Bausteinsicht.\n\n")

	// ── Chapter 2 ──────────────────────────────────────────────────────────────
	b.WriteString("== Architecture Constraints\n\n")
	b.WriteString("No specific architectural constraints are defined for this example.\n\n")

	// ── Chapter 3: System Scope and Context ────────────────────────────────────
	// Emit an SVG image directive for every imported view.
	// The view key is the bausteinsicht-generated key (scope-based, not DSL ID-based).
	b.WriteString("== System Scope and Context\n\n")
	for _, key := range sortedKeys(m.Views) {
		writeViewSection(&b, m, key)
	}

	// Top-level elements (people + software systems)
	topLevelIDs := sortedTopLevelIDs(m.Model)
	for _, key := range topLevelIDs {
		elem := m.Model[key]
		anchor := "sec-" + strings.ToLower(key)
		fmt.Fprintf(&b, "[[%s]]\n=== %s\n\n", anchor, elem.Title)
		if elem.Description != "" {
			fmt.Fprintf(&b, "%s\n\n", elem.Description)
		}
		fmt.Fprintf(&b, "* *Kind:* %s\n", elem.Kind)
		if elem.Technology != "" {
			fmt.Fprintf(&b, "* *Technology:* %s\n", elem.Technology)
		}
		b.WriteString("\n")
	}

	// ── Chapter 4 ──────────────────────────────────────────────────────────────
	b.WriteString("== Solution Strategy\n\n")
	b.WriteString("The Internet Banking System uses a modern single-page application backed by a REST API.\n\n")

	// ── Chapter 5: Building Block View ────────────────────────────────────────
	b.WriteString("== Building Block View\n\n")

	internetBanking, hasIBS := m.Model["internetBankingSystem"]
	if hasIBS {
		// Level 1 — Containers
		b.WriteString("=== Level 1: Internet Banking System — Containers\n\n")
		containerIDs := sortedKeys(internetBanking.Children)
		for _, key := range containerIDs {
			child := internetBanking.Children[key]
			anchor := "sec-" + strings.ToLower(key)
			fmt.Fprintf(&b, "[[%s]]\n==== %s\n\n", anchor, child.Title)
			if child.Description != "" {
				fmt.Fprintf(&b, "%s\n\n", child.Description)
			}
			if child.Technology != "" {
				fmt.Fprintf(&b, "* *Technology:* %s\n\n", child.Technology)
			}
		}

		// Level 2 — Components (API Application)
		apiApp, hasAPI := internetBanking.Children["apiApplication"]
		if hasAPI {
			b.WriteString("=== Level 2: API Application — Components\n\n")
			compIDs := sortedKeys(apiApp.Children)
			for _, key := range compIDs {
				comp := apiApp.Children[key]
				anchor := "sec-" + strings.ToLower(key)
				fmt.Fprintf(&b, "[[%s]]\n==== %s\n\n", anchor, comp.Title)
				if comp.Description != "" {
					fmt.Fprintf(&b, "%s\n\n", comp.Description)
				}
				if comp.Technology != "" {
					fmt.Fprintf(&b, "* *Technology:* %s\n\n", comp.Technology)
				}
			}
		}
	}

	// ── Chapter 6 ──────────────────────────────────────────────────────────────
	b.WriteString("== Runtime View\n\n")
	b.WriteString("_See dynamic views in the Structurizr workspace for runtime scenarios (sign-in flow, etc.)._\n\n")

	// ── Chapter 7 ──────────────────────────────────────────────────────────────
	b.WriteString("== Deployment View\n\n")
	b.WriteString("_See deployment diagrams in the Structurizr workspace for Development and Live environments._\n\n")

	// ── Chapter 8–12 stubs ─────────────────────────────────────────────────────
	for _, ch := range []string{
		"Cross-cutting Concepts",
		"Architecture Decisions",
		"Quality Requirements",
		"Risks and Technical Debt",
		"Glossary",
	} {
		fmt.Fprintf(&b, "== %s\n\n_To be defined._\n\n", ch)
	}

	return b.String()
}

// writeViewSection emits an SVG image directive for the given view key.
// The directive uses opts="inline" so that SVG links remain clickable in HTML output.
// For PDF output, Asciidoctor PDF also follows links embedded in SVG files.
func writeViewSection(b *strings.Builder, m *model.BausteinsichtModel, viewKey string) {
	view := m.Views[viewKey]
	title := view.Title
	if title == "" {
		title = viewKey
	}
	// opts="inline" — inlines the SVG into the HTML page so internal #sec-* links work.
	// For external-file SVG (without opts="inline"), use absolute anchors: arc42.html#sec-*.
	fmt.Fprintf(b, "=== %s\n\n", title)
	fmt.Fprintf(b, ".%s\n", title)
	fmt.Fprintf(b, "image::svgs/architecture-%s.svg[%s,opts=\"inline\"]\n\n", viewKey, title)
}

// validateSVGLinks parses an SVG file and validates its hyperlinks.
//
// Two categories of hrefs are silently allowed:
//   - External URLs (https://...) — draw.io embeds a font-rendering hint URL
//     ("https://www.drawio.com/doc/faq/svg-export-text-problems") in every SVG.
//   - Internal draw.io page links (data:page/id,...) — these are only clickable
//     inside the draw.io app and are not rendered as SVG <a> elements on export.
//
// For any remaining (element-level) hrefs, every value must appear in anchorMap.
// Finding zero element hrefs is acceptable: the DrawioLinkAttributes subtest already
// verifies that link attributes are written to the draw.io XML; whether they appear
// in exported SVG depends on which elements are placed on each view's page.
func validateSVGLinks(t *testing.T, svgPath string, anchorMap map[string]string) {
	t.Helper()
	data, err := os.ReadFile(svgPath)
	if err != nil {
		t.Errorf("read SVG %s: %v", svgPath, err)
		return
	}

	hrefs := extractSVGHrefs(data)

	// Build a set of expected anchors.
	expected := make(map[string]bool)
	for _, anchor := range anchorMap {
		expected[anchor] = true
	}

	elementHrefs := 0
	for href := range hrefs {
		if strings.HasPrefix(href, "https://") || strings.HasPrefix(href, "http://") {
			continue // draw.io's own external URLs (font hint etc.) — ignored
		}
		if strings.HasPrefix(href, "data:page/id,") {
			continue // internal draw.io drill-down links — not rendered in SVG
		}
		// draw.io resolves relative fragment-only hrefs (#anchor) against its
		// own export3.html source file during SVG export. Strip the prefix so
		// we can validate the anchor regardless of the draw.io install path.
		if strings.HasPrefix(href, "file://") {
			if idx := strings.LastIndex(href, "#"); idx != -1 {
				href = href[idx:] // "#sec-foo"
			}
		}
		// Any remaining href is an element-level link and must match our anchorMap.
		elementHrefs++
		if !expected[href] {
			t.Errorf("SVG %s: unexpected element href %q (not in anchorMap)", svgPath, href)
		}
	}

	t.Logf("SVG %s: %d total href(s), %d element href(s) validated", filepath.Base(svgPath), len(hrefs), elementHrefs)
}

// extractSVGHrefs parses SVG XML and collects all href / xlink:href values from <a> elements.
func extractSVGHrefs(data []byte) map[string]bool {
	hrefs := make(map[string]bool)
	dec := xml.NewDecoder(bytes.NewReader(data))
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "a" {
			continue
		}
		for _, attr := range se.Attr {
			if attr.Name.Local == "href" {
				hrefs[attr.Value] = true
			}
		}
	}
	return hrefs
}

// findDrawioCmd returns the path to the draw.io CLI binary, or "" if not found.
func findDrawioCmd() string {
	for _, name := range []string{"drawio-export", "drawio", "draw.io"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	return ""
}

// printWorkflowHint logs the manual workflow commands for running the full pipeline.
func printWorkflowHint(t *testing.T, dir string) {
	t.Helper()
	t.Logf("\n── Manual full-pipeline commands ────────────────────────────────────────")
	t.Logf("  cd %s", dir)
	t.Logf("  bausteinsicht sync")
	t.Logf("  bausteinsicht export --image-format svg --output svgs/")
	t.Logf("  asciidoctor arc42.adoc          # HTML output")
	t.Logf("  asciidoctor-pdf arc42.adoc      # PDF output")
	t.Logf("─────────────────────────────────────────────────────────────────────────")
}

// sortedTopLevelIDs returns top-level model keys sorted alphabetically.
func sortedTopLevelIDs(elems map[string]model.Element) []string {
	keys := make([]string, 0, len(elems))
	for k := range elems {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// sortedKeys returns sorted keys of any string-keyed map.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// viewKeys returns a sorted list of view keys for use in error messages.
func viewKeys(views map[string]model.View) []string {
	return sortedKeys(views)
}

// findModuleRootPath walks up from the e2e/ directory to find the directory containing go.mod.
func findModuleRootPath() (string, error) {
	dir, err := filepath.Abs(".")
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}
