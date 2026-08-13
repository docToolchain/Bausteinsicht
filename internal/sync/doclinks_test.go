package sync

import (
	"strconv"
	"strings"
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

// --- Additional coverage: glyph table, geometry, multi-entry, multi-view,
// removal, growth, and the reverse-sync safety property (#543 follow-up) ---

func iconX(t *testing.T, obj *etree.Element) float64 {
	t.Helper()
	cell := obj.SelectElement("mxCell")
	if cell == nil {
		t.Fatal("icon has no mxCell")
	}
	geo := cell.SelectElement("mxGeometry")
	if geo == nil {
		t.Fatal("icon has no mxGeometry")
	}
	return parseFloat(geo.SelectAttrValue("x", "0"))
}

func TestDocLinkGlyph_AllTypesDistinct(t *testing.T) {
	isFallbackType := func(s string) bool { return s == "unknown-type" || s == "" }

	types := []string{"arc42", "adr", "persona", "prd", "spec", "unknown-type", ""}
	seen := make(map[string]string, len(types))
	for _, typ := range types {
		glyph, fill, stroke := docLinkGlyph(typ)
		if glyph == "" {
			t.Errorf("type %q: expected a non-empty glyph", typ)
		}
		if fill == "" || stroke == "" {
			t.Errorf("type %q: expected non-empty fill/stroke colors", typ)
		}
		if other, ok := seen[glyph]; ok && other != typ {
			// "unknown-type" and "" both intentionally fall back to the same
			// generic glyph — that's the one allowed collision.
			if !isFallbackType(typ) || !isFallbackType(other) {
				t.Errorf("type %q: glyph %q collides with type %q", typ, glyph, other)
			}
		}
		seen[glyph] = typ
	}
	// Known types must each get their own distinct glyph (no two conventional
	// types silently sharing an icon).
	knownGlyphs := make(map[string]bool)
	for _, typ := range []string{"arc42", "adr", "persona", "prd", "spec"} {
		glyph, _, _ := docLinkGlyph(typ)
		if knownGlyphs[glyph] {
			t.Errorf("type %q: glyph %q is not unique among the conventional types", typ, glyph)
		}
		knownGlyphs[glyph] = true
	}
}

func TestPlaceElementIconRow_NoOverlap(t *testing.T) {
	root := etree.NewElement("root")
	icons := []docIcon{
		{id: "doclink-x-a", glyph: "a", fill: "#fff", stroke: "#000", href: "a"},
		{id: "doclink-x-b", glyph: "b", fill: "#fff", stroke: "#000", href: "b"},
		{id: "doclink-x-c", glyph: "c", fill: "#fff", stroke: "#000", href: "c"},
	}
	placeElementIconRow(root, "elem1", icons, 200)

	var xs []float64
	for _, icon := range icons {
		obj := findObjectByID(root, icon.id)
		if obj == nil {
			t.Fatalf("expected icon %q to exist", icon.id)
		}
		xs = append(xs, iconX(t, obj))
	}
	for i := 1; i < len(xs); i++ {
		gap := xs[i] - xs[i-1]
		if gap != docIconSize+docIconGap {
			t.Errorf("icon %d: expected x-gap %v from previous icon, got %v (xs=%v)", i, docIconSize+docIconGap, gap, xs)
		}
		if xs[i] < xs[i-1]+docIconSize {
			t.Errorf("icon %d overlaps icon %d: xs=%v, iconSize=%v", i, i-1, xs, docIconSize)
		}
	}
	// The row leaves a docIconGap margin at the parent's right edge (matches
	// the pre-existing AddDecisionBadges/AddStatusBadge margin convention).
	wantEnd := 200.0 - docIconGap
	if last := xs[len(xs)-1]; last+docIconSize != wantEnd {
		t.Errorf("expected the last icon to end at %v (parent width 200 minus the %v margin), got end=%v", wantEnd, docIconGap, last+docIconSize)
	}
}

func TestSynchronizeDocLinkIcons_CombinedOrderAndUniqueness(t *testing.T) {
	ts := minimalTemplates(t)
	m := &model.BausteinsichtModel{
		Specification: model.Specification{
			Decisions: []model.DecisionRecord{
				{ID: "ADR-001", Status: model.ADRActive, FilePath: "adr1.html"},
				{ID: "ADR-002", Status: model.ADRActive, FilePath: "adr2.html"},
			},
		},
		Model: map[string]model.Element{
			"api": {
				Kind:      "container",
				Title:     "API",
				Link:      "architecture.adoc#sec-api",
				Decisions: []string{"ADR-001", "ADR-002"},
				DocLinks: []model.DocLink{
					{Type: "arc42", Href: "arc42.html"},
					{Type: "persona", Href: "persona.html"},
				},
			},
		},
	}
	doc := emptyDoc()

	Run(m, doc, emptyState(), ts, nil)

	page := requireFirstPage(t, doc)
	root := page.Root()

	// All 5 icons must exist, each with a unique id and its own href.
	wantOrder := []struct {
		id   string
		href string
	}{
		{"doclink-api-link", "architecture.adoc#sec-api"},
		{"doclink-api-decision-0", "adr1.html"},
		{"doclink-api-decision-1", "adr2.html"},
		{"doclink-api-doc-0", "arc42.html"},
		{"doclink-api-doc-1", "persona.html"},
	}

	var xs []float64
	for _, want := range wantOrder {
		obj := findObjectByID(root, want.id)
		if obj == nil {
			t.Fatalf("expected icon %q to exist", want.id)
		}
		if got := obj.SelectAttrValue("link", ""); got != want.href {
			t.Errorf("icon %q: expected href %q, got %q", want.id, want.href, got)
		}
		xs = append(xs, iconX(t, obj))
	}

	// Left-to-right rendering order must match wantOrder (link, decisions, docLinks).
	for i := 1; i < len(xs); i++ {
		if xs[i] <= xs[i-1] {
			t.Errorf("icon %d (%s, x=%v) is not to the right of icon %d (%s, x=%v) — order/overlap violated",
				i, wantOrder[i].id, xs[i], i-1, wantOrder[i-1].id, xs[i-1])
		}
	}

	// No stray 6th icon.
	count := 0
	for _, obj := range root.SelectElements("object") {
		if strings.HasPrefix(obj.SelectAttrValue("id", ""), "doclink-api-") {
			count++
		}
	}
	if count != len(wantOrder) {
		t.Errorf("expected exactly %d doclink-api-* icons, got %d", len(wantOrder), count)
	}
}

// TestSynchronizeViewDocLinkIcons_NoOverlapWithMetadataBox is a regression
// test for the icon row landing on the exact same coordinate as the
// metadata box (both previously derived their position from computeMaxY
// independently): with real content on the page, the metadata box gets a
// non-trivial width and the doc-link row must anchor to *its* geometry
// instead of duplicating the computation — otherwise both info boxes are
// created at contentWidth-driven positions, and the icon row (still using
// the old fixed metadataX/computeMaxY math) would sit on top of the
// metadata box's title text.
func TestSynchronizeViewDocLinkIcons_NoOverlapWithMetadataBox(t *testing.T) {
	ts := minimalTemplates(t)
	m := viewModelWithDocLinks([]model.DocLink{
		{Type: "prd", Href: "docs/prd.html", Title: "Product Requirements"},
	})
	doc := drawio.NewDocument()
	doc.AddPage("view-overview", "Overview")

	Run(m, doc, emptyState(), ts, nil)
	page := requirePage(t, doc, "view-overview")
	root := page.Root()

	mx, my, mw, mh, found := infoBoxGeometry(root, metadataPrefix+"overview")
	if !found {
		t.Fatal("expected a metadata box on the page")
	}

	icon := findObjectByID(root, "doclink-view-overview-0")
	if icon == nil {
		t.Fatal("expected the view-level doc-link icon to exist")
	}
	ix := iconX(t, icon)
	iy := icon.SelectElement("mxCell").SelectElement("mxGeometry").SelectAttrValue("y", "")
	iyFloat := parseFloat(iy)

	// The icon must not sit at the metadata box's own top-left corner (the
	// original bug: both computed to the identical (x, y)).
	if ix == mx && iyFloat == my {
		t.Errorf("icon at (%v, %v) collides exactly with the metadata box's origin (%v, %v)", ix, iyFloat, mx, my)
	}
	// It must stay horizontally within the box (right-anchored row), not
	// spill past its right edge.
	if ix < mx || ix+docIconSize > mx+mw {
		t.Errorf("icon x=%v (width %v) falls outside metadata box bounds [%v, %v]", ix, docIconSize, mx, mx+mw)
	}
	// And vertically within the box's own height, near its top edge — it
	// shares the header row, it doesn't sit below the box entirely.
	if iyFloat < my || iyFloat+docIconSize > my+mh {
		t.Errorf("icon y=%v (height %v) falls outside metadata box bounds [%v, %v]", iyFloat, docIconSize, my, my+mh)
	}
}

func TestSynchronizeViewDocLinkIcons_MultipleEntries(t *testing.T) {
	ts := minimalTemplates(t)
	m := viewModelWithDocLinks([]model.DocLink{
		{Type: "prd", Href: "docs/prd.html"},
		{Type: "persona", Href: "docs/persona.html"},
		{Type: "arc42", Href: "docs/arc42.html"},
	})
	doc := drawio.NewDocument()
	doc.AddPage("view-overview", "Overview")

	Run(m, doc, emptyState(), ts, nil)

	page := requirePage(t, doc, "view-overview")
	root := page.Root()

	wantHrefs := []string{"docs/prd.html", "docs/persona.html", "docs/arc42.html"}
	var xs []float64
	for i, href := range wantHrefs {
		id := "doclink-view-overview-" + strconv.Itoa(i)
		obj := findObjectByID(root, id)
		if obj == nil {
			t.Fatalf("expected icon %q to exist", id)
		}
		if got := obj.SelectAttrValue("link", ""); got != href {
			t.Errorf("icon %q: expected href %q, got %q", id, href, got)
		}
		xs = append(xs, iconX(t, obj))
	}
	for i := 1; i < len(xs); i++ {
		if xs[i] < xs[i-1]+docIconSize+docIconGap {
			t.Errorf("view icons %d and %d overlap or are too close: xs=%v", i-1, i, xs)
		}
	}
}

func TestSynchronizeViewDocLinkIcons_MultipleViewsNoCollision(t *testing.T) {
	ts := minimalTemplates(t)
	m := &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{"container": {Notation: "Container"}},
		},
		Model: map[string]model.Element{
			"api": {Kind: "container", Title: "API"},
		},
		Views: map[string]model.View{
			"overview": {
				Title: "Overview", Include: []string{"api"}, Layout: "layered",
				DocLinks: []model.DocLink{{Type: "prd", Href: "overview-prd.html"}},
			},
			"detail": {
				Title: "Detail", Include: []string{"api"}, Layout: "layered",
				DocLinks: []model.DocLink{{Type: "spec", Href: "detail-spec.html"}},
			},
		},
	}
	doc := drawio.NewDocument()
	doc.AddPage("view-overview", "Overview")
	doc.AddPage("view-detail", "Detail")

	Run(m, doc, emptyState(), ts, nil)

	overviewIcon := findObjectByID(requirePage(t, doc, "view-overview").Root(), "doclink-view-overview-0")
	if overviewIcon == nil {
		t.Fatal("expected doclink-view-overview-0 on the overview page")
	}
	if got := overviewIcon.SelectAttrValue("link", ""); got != "overview-prd.html" {
		t.Errorf("overview icon: expected %q, got %q", "overview-prd.html", got)
	}

	detailIcon := findObjectByID(requirePage(t, doc, "view-detail").Root(), "doclink-view-detail-0")
	if detailIcon == nil {
		t.Fatal("expected doclink-view-detail-0 on the detail page")
	}
	if got := detailIcon.SelectAttrValue("link", ""); got != "detail-spec.html" {
		t.Errorf("detail icon: expected %q, got %q", "detail-spec.html", got)
	}

	// The overview page must not also contain the detail view's icon (and vice
	// versa) — the id includes the view id, so no cross-page collision.
	if findObjectByID(requirePage(t, doc, "view-overview").Root(), "doclink-view-detail-0") != nil {
		t.Error("overview page unexpectedly contains the detail view's icon")
	}
}

func TestSynchronizeViewDocLinkIcons_RemovedBetweenSyncs(t *testing.T) {
	ts := minimalTemplates(t)
	m := viewModelWithDocLinks([]model.DocLink{{Type: "prd", Href: "docs/prd.html"}})
	doc := drawio.NewDocument()
	doc.AddPage("view-overview", "Overview")

	Run(m, doc, emptyState(), ts, nil)
	page := requirePage(t, doc, "view-overview")
	if findObjectByID(page.Root(), "doclink-view-overview-0") == nil {
		t.Fatal("round 1: expected the icon to exist")
	}

	// Round 2: the view no longer declares any docLinks.
	m.Views["overview"] = model.View{Title: "Overview", Include: []string{"api"}, Layout: "layered"}
	state1 := emptyState()
	state1.Elements["api"] = ElementState{Title: "API", Kind: "container"}
	Run(m, doc, state1, ts, nil)

	if icon := findObjectByID(page.Root(), "doclink-view-overview-0"); icon != nil {
		t.Error("round 2: expected the removed docLinks entry's icon to be cleaned up")
	}
}

func TestSynchronizeViewDocLinkIcons_CountGrowsRightEdgeStable(t *testing.T) {
	ts := minimalTemplates(t)
	m := viewModelWithDocLinks([]model.DocLink{{Type: "prd", Href: "docs/prd.html"}})
	doc := drawio.NewDocument()
	doc.AddPage("view-overview", "Overview")

	Run(m, doc, emptyState(), ts, nil)
	page := requirePage(t, doc, "view-overview")
	icon0Round1 := findObjectByID(page.Root(), "doclink-view-overview-0")
	if icon0Round1 == nil {
		t.Fatal("round 1: expected icon 0 to exist")
	}
	// The row is anchored to the metadata box's right edge and grows
	// leftward (matching placeElementIconRow's element-icon convention), so
	// with a single icon it sits at that fixed right-edge anchor.
	rightEdgeX := iconX(t, icon0Round1)

	// Round 2: a second docLinks entry is added.
	m.Views["overview"] = model.View{
		Title: "Overview", Include: []string{"api"}, Layout: "layered",
		DocLinks: []model.DocLink{
			{Type: "prd", Href: "docs/prd.html"},
			{Type: "spec", Href: "docs/spec.html"},
		},
	}
	state1 := emptyState()
	state1.Elements["api"] = ElementState{Title: "API", Kind: "container"}
	Run(m, doc, state1, ts, nil)

	// The newly-last icon now occupies the anchor position icon 0 held
	// alone; icon 0 shifts one slot left to make room.
	icon1 := findObjectByID(page.Root(), "doclink-view-overview-1")
	if icon1 == nil {
		t.Fatal("round 2: expected the newly-added icon 1 to exist")
	}
	if x1 := iconX(t, icon1); x1 != rightEdgeX {
		t.Errorf("round 2: expected the last icon to stay at the right-edge anchor %v, got %v", rightEdgeX, x1)
	}

	icon0Round2 := findObjectByID(page.Root(), "doclink-view-overview-0")
	if icon0Round2 == nil {
		t.Fatal("round 2: expected icon 0 to still exist")
	}
	if x0Round2 := iconX(t, icon0Round2); x0Round2 != rightEdgeX-(docIconSize+docIconGap) {
		t.Errorf("round 2: expected icon 0 to shift left by one slot to %v, got %v", rightEdgeX-(docIconSize+docIconGap), x0Round2)
	}
	if got := icon1.SelectAttrValue("link", ""); got != "docs/spec.html" {
		t.Errorf("round 2: icon 1 expected href %q, got %q", "docs/spec.html", got)
	}
	x0Round2 := iconX(t, icon0Round2)
	if x1 := iconX(t, icon1); x1 < x0Round2+docIconSize+docIconGap {
		t.Errorf("round 2: icon 1 (x=%v) overlaps icon 0 (x=%v)", x1, x0Round2)
	}
}

// TestSynchronizeDocLinkIcons_NotImportedByReverseSync is the safety property
// behind the docLinkPrefix exclusions added to diff.go: doc-link icons must
// never be mistaken for user-drawn elements during reverse sync, or the
// model would accumulate phantom elements on every sync. Runs the icon-heavy
// combined model through two full Run() cycles and asserts the model's
// element/relationship counts never change from icons alone.
func TestSynchronizeDocLinkIcons_NotImportedByReverseSync(t *testing.T) {
	ts := minimalTemplates(t)
	m := &model.BausteinsichtModel{
		Specification: model.Specification{
			Decisions: []model.DecisionRecord{
				{ID: "ADR-001", Status: model.ADRActive, FilePath: "adr1.html"},
			},
		},
		Model: map[string]model.Element{
			"api": {
				Kind:      "container",
				Title:     "API",
				Link:      "architecture.adoc#sec-api",
				Decisions: []string{"ADR-001"},
				DocLinks:  []model.DocLink{{Type: "arc42", Href: "arc42.html"}},
			},
		},
	}
	doc := emptyDoc()

	r1 := Run(m, doc, emptyState(), ts, nil)
	if got := len(m.Model); got != 1 {
		t.Fatalf("round 1: expected model to still have 1 element, got %d: %v", got, m.Model)
	}
	if r1.Reverse.ElementsCreated != 0 {
		t.Errorf("round 1: expected 0 elements created by reverse sync, got %d", r1.Reverse.ElementsCreated)
	}

	// Persist round 1's state exactly like the CLI would, then re-sync — this
	// is the pass where a bug would surface: reverse sync re-scans the page
	// (now containing 3 icon <object> cells alongside the real element) and
	// must not mistake any of them for newly-drawn elements.
	state1 := emptyState()
	state1.Elements["api"] = ElementState{
		Title: "API", Kind: "container", Link: "architecture.adoc#sec-api",
	}
	r2 := Run(m, doc, state1, ts, nil)

	if got := len(m.Model); got != 1 {
		t.Errorf("round 2: expected model to still have exactly 1 element, got %d: %v", got, m.Model)
	}
	if _, ok := m.Model["api"]; !ok {
		t.Error("round 2: expected the original 'api' element to still be present")
	}
	if r2.Reverse.ElementsCreated != 0 {
		t.Errorf("round 2: expected 0 elements created by reverse sync (icons must not be imported), got %d", r2.Reverse.ElementsCreated)
	}
	if len(m.Relationships) != 0 {
		t.Errorf("round 2: expected 0 relationships (icons carry no edges), got %d: %v", len(m.Relationships), m.Relationships)
	}
	for _, w := range r2.Warnings {
		if strings.Contains(w, "doclink-") {
			t.Errorf("round 2: unexpected warning mentioning a doclink- icon id: %q", w)
		}
	}
}
