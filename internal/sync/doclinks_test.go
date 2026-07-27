package sync

import (
	"testing"

	"github.com/beevik/etree"
	"github.com/docToolchain/Bausteinsicht/internal/drawio"
	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// findObjectByID returns the <object> with the given draw.io cell id, or nil.
func findObjectByID(root *etree.Element, id string) *etree.Element {
	for _, obj := range root.SelectElements("object") {
		if obj.SelectAttrValue("id", "") == id {
			return obj
		}
	}
	return nil
}

// --- Element-level doc-link icons (#543) ---

func TestSynchronizeDocLinkIcons_LinkIcon(t *testing.T) {
	ts := minimalTemplates(t)
	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"api": {Kind: "container", Title: "API", Link: "architecture.adoc#sec-backend"},
		},
	}
	doc := emptyDoc()

	Run(m, doc, emptyState(), ts, nil)

	page := requireFirstPage(t, doc)
	icon := findObjectByID(page.Root(), "doclink-api-link")
	if icon == nil {
		t.Fatal("expected a doclink-api-link icon object")
	}
	if got := icon.SelectAttrValue("link", ""); got != "architecture.adoc#sec-backend" {
		t.Errorf("expected link %q, got %q", "architecture.adoc#sec-backend", got)
	}
	// The element's own shape link is untouched by the icon (ADR-009 priority
	// rule for drill-down vs. user link is unaffected by this new icon).
	elemObj := findObjectByID(page.Root(), "api")
	if elemObj == nil {
		t.Fatal("expected the element's own object to exist")
	}
	if got := elemObj.SelectAttrValue("link", ""); got != "architecture.adoc#sec-backend" {
		t.Errorf("expected element shape to keep its own link %q, got %q", "architecture.adoc#sec-backend", got)
	}
}

func TestSynchronizeDocLinkIcons_NoLinkNoIcon(t *testing.T) {
	ts := minimalTemplates(t)
	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"api": {Kind: "container", Title: "API"},
		},
	}
	doc := emptyDoc()

	Run(m, doc, emptyState(), ts, nil)

	page := requireFirstPage(t, doc)
	if icon := findObjectByID(page.Root(), "doclink-api-link"); icon != nil {
		t.Error("expected no doclink-api-link icon when Link is unset")
	}
}

func TestSynchronizeDocLinkIcons_DecisionClickableWhenFileKnown(t *testing.T) {
	ts := minimalTemplates(t)
	m := &model.BausteinsichtModel{
		Specification: model.Specification{
			Decisions: []model.DecisionRecord{
				{ID: "ADR-001", Title: "Use JSONC", Status: model.ADRActive, FilePath: "ADRs/ADR-001-DSL-Format.html"},
			},
		},
		Model: map[string]model.Element{
			"api": {Kind: "container", Title: "API", Decisions: []string{"ADR-001"}},
		},
	}
	doc := emptyDoc()

	Run(m, doc, emptyState(), ts, nil)

	page := requireFirstPage(t, doc)
	icon := findObjectByID(page.Root(), "doclink-api-decision-0")
	if icon == nil {
		t.Fatal("expected a doclink-api-decision-0 icon object")
	}
	if got := icon.SelectAttrValue("link", ""); got != "ADRs/ADR-001-DSL-Format.html" {
		t.Errorf("expected link %q, got %q", "ADRs/ADR-001-DSL-Format.html", got)
	}
}

func TestSynchronizeDocLinkIcons_DecisionNotClickableWithoutFile(t *testing.T) {
	ts := minimalTemplates(t)
	m := &model.BausteinsichtModel{
		Specification: model.Specification{
			Decisions: []model.DecisionRecord{
				{ID: "ADR-001", Title: "Use JSONC", Status: model.ADRActive}, // no FilePath
			},
		},
		Model: map[string]model.Element{
			"api": {Kind: "container", Title: "API", Decisions: []string{"ADR-001"}},
		},
	}
	doc := emptyDoc()

	Run(m, doc, emptyState(), ts, nil)

	page := requireFirstPage(t, doc)
	icon := findObjectByID(page.Root(), "doclink-api-decision-0")
	if icon == nil {
		t.Fatal("expected a doclink-api-decision-0 icon object even without a file (visible, non-clickable)")
	}
	if got := icon.SelectAttrValue("link", ""); got != "" {
		t.Errorf("expected no link attribute when the ADR has no file, got %q", got)
	}
}

func TestSynchronizeDocLinkIcons_DocLinksTyped(t *testing.T) {
	ts := minimalTemplates(t)
	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"api": {
				Kind:  "container",
				Title: "API",
				DocLinks: []model.DocLink{
					{Type: "arc42", Href: "docs/arc42/05-building-blocks.html#api", Title: "Baustein API"},
					{Type: "persona", Href: "docs/prd.html#felix"},
				},
			},
		},
	}
	doc := emptyDoc()

	Run(m, doc, emptyState(), ts, nil)

	page := requireFirstPage(t, doc)

	icon0 := findObjectByID(page.Root(), "doclink-api-doc-0")
	if icon0 == nil {
		t.Fatal("expected a doclink-api-doc-0 icon object")
	}
	if got := icon0.SelectAttrValue("link", ""); got != "docs/arc42/05-building-blocks.html#api" {
		t.Errorf("doc-0: expected link %q, got %q", "docs/arc42/05-building-blocks.html#api", got)
	}
	if got := icon0.SelectAttrValue("tooltip", ""); got != "Baustein API" {
		t.Errorf("doc-0: expected tooltip %q, got %q", "Baustein API", got)
	}

	icon1 := findObjectByID(page.Root(), "doclink-api-doc-1")
	if icon1 == nil {
		t.Fatal("expected a doclink-api-doc-1 icon object")
	}
	if got := icon1.SelectAttrValue("link", ""); got != "docs/prd.html#felix" {
		t.Errorf("doc-1: expected link %q, got %q", "docs/prd.html#felix", got)
	}
	// Title omitted → tooltip falls back to the type.
	if got := icon1.SelectAttrValue("tooltip", ""); got != "persona" {
		t.Errorf("doc-1: expected fallback tooltip %q, got %q", "persona", got)
	}
}

func TestSynchronizeDocLinkIcons_StaleIconsRemovedOnResync(t *testing.T) {
	ts := minimalTemplates(t)
	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"api": {Kind: "container", Title: "API", Link: "architecture.adoc#sec-backend"},
		},
	}
	doc := emptyDoc()

	r1 := Run(m, doc, emptyState(), ts, nil)
	page := requireFirstPage(t, doc)
	if findObjectByID(page.Root(), "doclink-api-link") == nil {
		t.Fatal("round 1: expected the link icon to exist")
	}

	// Remove the Link in the model and re-sync from the state round 1 produced.
	state1 := stateWithElem("api", "API", "", "")
	m.Model["api"] = model.Element{Kind: "container", Title: "API"}
	Run(m, doc, state1, ts, nil)

	if icon := findObjectByID(page.Root(), "doclink-api-link"); icon != nil {
		t.Error("round 2: expected the stale link icon to be removed")
	}
	_ = r1
}

// --- View-level doc-link icons (#543) ---

func viewModelWithDocLinks(docLinks []model.DocLink) *model.BausteinsichtModel {
	return &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{"container": {Notation: "Container"}},
		},
		Model: map[string]model.Element{
			"api": {Kind: "container", Title: "API"},
		},
		Views: map[string]model.View{
			"overview": {
				Title:    "Overview",
				Include:  []string{"api"},
				Layout:   "layered",
				DocLinks: docLinks,
			},
		},
	}
}

func TestSynchronizeViewDocLinkIcons_Basic(t *testing.T) {
	ts := minimalTemplates(t)
	m := viewModelWithDocLinks([]model.DocLink{
		{Type: "prd", Href: "docs/prd.html", Title: "Product Requirements"},
	})
	doc := drawio.NewDocument()
	doc.AddPage("view-overview", "Overview")

	Run(m, doc, emptyState(), ts, nil)

	page := requirePage(t, doc, "view-overview")
	icon := findObjectByID(page.Root(), "doclink-view-overview-0")
	if icon == nil {
		t.Fatal("expected a doclink-view-overview-0 icon object")
	}
	if got := icon.SelectAttrValue("link", ""); got != "docs/prd.html" {
		t.Errorf("expected link %q, got %q", "docs/prd.html", got)
	}
	cell := icon.SelectElement("mxCell")
	if cell == nil {
		t.Fatal("expected the icon to wrap an mxCell")
	}
	if got := cell.SelectAttrValue("parent", ""); got != "1" {
		t.Errorf("expected the view-level icon to be page-absolute (parent=1), got %q", got)
	}
}

func TestSynchronizeViewDocLinkIcons_NoDocLinksNoIcons(t *testing.T) {
	ts := minimalTemplates(t)
	m := viewModelWithDocLinks(nil)
	doc := drawio.NewDocument()
	doc.AddPage("view-overview", "Overview")

	Run(m, doc, emptyState(), ts, nil)

	page := requirePage(t, doc, "view-overview")
	for _, obj := range page.Root().SelectElements("object") {
		id := obj.SelectAttrValue("id", "")
		if len(id) >= len("doclink-view-") && id[:len("doclink-view-")] == "doclink-view-" {
			t.Errorf("expected no view-level doc-link icons, found %q", id)
		}
	}
}

func TestSynchronizeViewDocLinkIcons_PositionStableAcrossSyncs(t *testing.T) {
	ts := minimalTemplates(t)
	m := viewModelWithDocLinks([]model.DocLink{
		{Type: "prd", Href: "docs/prd.html"},
	})
	doc := drawio.NewDocument()
	doc.AddPage("view-overview", "Overview")

	Run(m, doc, emptyState(), ts, nil)
	page := requirePage(t, doc, "view-overview")
	icon1 := findObjectByID(page.Root(), "doclink-view-overview-0")
	if icon1 == nil {
		t.Fatal("round 1: expected the icon to exist")
	}
	x1 := icon1.SelectElement("mxCell").SelectElement("mxGeometry").SelectAttrValue("x", "")
	y1 := icon1.SelectElement("mxCell").SelectElement("mxGeometry").SelectAttrValue("y", "")

	// Second sync: change the doc link's title (content changes) but not the
	// row's existence — the position must not drift.
	m.Views["overview"] = model.View{
		Title:    "Overview",
		Include:  []string{"api"},
		Layout:   "layered",
		DocLinks: []model.DocLink{{Type: "prd", Href: "docs/prd.html", Title: "Updated Title"}},
	}
	state1 := emptyState()
	state1.Elements["api"] = ElementState{Title: "API", Kind: "container"}
	Run(m, doc, state1, ts, nil)

	icon2 := findObjectByID(page.Root(), "doclink-view-overview-0")
	if icon2 == nil {
		t.Fatal("round 2: expected the icon to still exist")
	}
	x2 := icon2.SelectElement("mxCell").SelectElement("mxGeometry").SelectAttrValue("x", "")
	y2 := icon2.SelectElement("mxCell").SelectElement("mxGeometry").SelectAttrValue("y", "")

	if x1 != x2 || y1 != y2 {
		t.Errorf("expected stable position across syncs, round1=(%s,%s) round2=(%s,%s)", x1, y1, x2, y2)
	}
	if got := icon2.SelectAttrValue("tooltip", ""); got != "Updated Title" {
		t.Errorf("expected updated tooltip %q, got %q", "Updated Title", got)
	}
}
