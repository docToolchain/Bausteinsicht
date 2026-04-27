package sync

import (
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/drawio"
	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/docToolchain/Bausteinsicht/templates"
)

func loadTestTemplates(t *testing.T) *drawio.TemplateSet {
	t.Helper()
	ts, err := drawio.LoadTemplateFromBytes(templates.DefaultTemplate)
	if err != nil {
		t.Fatalf("failed to load templates: %v", err)
	}
	return ts
}

func buildLayoutTestModel() (*model.BausteinsichtModel, map[string]*model.Element) {
	m := &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"actor":     {Notation: "Actor"},
				"system":    {Notation: "System", Container: true},
				"container": {Notation: "Container"},
			},
		},
		Model: map[string]model.Element{
			"customer": {Kind: "actor", Title: "Customer"},
			"admin":    {Kind: "actor", Title: "Admin"},
			"shop":     {Kind: "system", Title: "Shop"},
			"payments": {Kind: "system", Title: "Payments"},
		},
		ElementOrder: []string{"actor", "system", "container"},
	}
	flat, _ := model.FlattenElements(m)
	return m, flat
}

func TestComputeLayout_Layered_GroupsByKind(t *testing.T) {
	_, flat := buildLayoutTestModel()
	ts := loadTestTemplates(t)
	elementOrder := []string{"actor", "system", "container"}

	ids := []string{"customer", "admin", "shop", "payments"}
	result := computeLayout(ids, flat, ts, elementOrder, "", "layered", nil)

	// Actors (tier 0) should be above systems (tier 1).
	customerY := result.Positions["customer"].Y
	adminY := result.Positions["admin"].Y
	shopY := result.Positions["shop"].Y
	paymentsY := result.Positions["payments"].Y

	if customerY != adminY {
		t.Errorf("actors should be on the same row: customer.Y=%v, admin.Y=%v", customerY, adminY)
	}
	if shopY != paymentsY {
		t.Errorf("systems should be on the same row: shop.Y=%v, payments.Y=%v", shopY, paymentsY)
	}
	if customerY >= shopY {
		t.Errorf("actors should be above systems: actor.Y=%v >= system.Y=%v", customerY, shopY)
	}
}

func TestComputeLayout_Layered_AlphabeticWithinTier(t *testing.T) {
	_, flat := buildLayoutTestModel()
	ts := loadTestTemplates(t)
	elementOrder := []string{"actor", "system"}

	ids := []string{"customer", "admin", "shop", "payments"}
	result := computeLayout(ids, flat, ts, elementOrder, "", "layered", nil)

	// Within actors tier: admin < customer alphabetically, so admin.X < customer.X
	if result.Positions["admin"].X >= result.Positions["customer"].X {
		t.Errorf("admin should be left of customer: admin.X=%v, customer.X=%v",
			result.Positions["admin"].X, result.Positions["customer"].X)
	}
	// Within systems tier: payments < shop
	if result.Positions["payments"].X >= result.Positions["shop"].X {
		t.Errorf("payments should be left of shop: payments.X=%v, shop.X=%v",
			result.Positions["payments"].X, result.Positions["shop"].X)
	}
}

func TestComputeLayout_Layered_Deterministic(t *testing.T) {
	_, flat := buildLayoutTestModel()
	ts := loadTestTemplates(t)
	elementOrder := []string{"actor", "system"}
	ids := []string{"customer", "admin", "shop", "payments"}

	result1 := computeLayout(ids, flat, ts, elementOrder, "", "layered", nil)
	result2 := computeLayout(ids, flat, ts, elementOrder, "", "layered", nil)

	for _, id := range ids {
		p1 := result1.Positions[id]
		p2 := result2.Positions[id]
		if p1.X != p2.X || p1.Y != p2.Y {
			t.Errorf("non-deterministic layout for %s: run1=%v, run2=%v", id, p1, p2)
		}
	}
}

func TestComputeLayout_Grid(t *testing.T) {
	_, flat := buildLayoutTestModel()
	ts := loadTestTemplates(t)

	ids := []string{"customer", "admin", "shop", "payments"}
	result := computeLayout(ids, flat, ts, nil, "", "grid", nil)

	if len(result.Positions) != 4 {
		t.Fatalf("expected 4 positions, got %d", len(result.Positions))
	}

	// All positions should be positive.
	for id, pos := range result.Positions {
		if pos.X < 0 || pos.Y < 0 {
			t.Errorf("negative position for %s: %v", id, pos)
		}
	}
}

func TestComputeLayout_None(t *testing.T) {
	_, flat := buildLayoutTestModel()
	ts := loadTestTemplates(t)

	ids := []string{"customer", "admin", "shop"}
	result := computeLayout(ids, flat, ts, nil, "", "none", nil)

	// All elements should be on the same Y (horizontal row).
	y := result.Positions["admin"].Y
	for _, id := range ids {
		if result.Positions[id].Y != y {
			t.Errorf("none layout should place all on same row, %s.Y=%v != %v", id, result.Positions[id].Y, y)
		}
	}
}

func TestComputeLayout_Layered_ScopeAware(t *testing.T) {
	m := &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"actor":     {Notation: "Actor"},
				"system":    {Notation: "System", Container: true},
				"container": {Notation: "Container"},
			},
		},
		Model: map[string]model.Element{
			"customer": {Kind: "actor", Title: "Customer"},
			"shop": {Kind: "system", Title: "Shop", Children: map[string]model.Element{
				"api": {Kind: "container", Title: "API"},
				"db":  {Kind: "container", Title: "DB"},
			}},
		},
		ElementOrder: []string{"actor", "system", "container"},
	}
	flat, _ := model.FlattenElements(m)
	ts := loadTestTemplates(t)

	ids := []string{"customer", "shop.api", "shop.db"}
	result := computeLayout(ids, flat, ts, m.ElementOrder, "shop", "layered", nil)

	// Scoped children (shop.api, shop.db) should have positions relative to boundary.
	apiPos := result.Positions["shop.api"]
	dbPos := result.Positions["shop.db"]
	customerPos := result.Positions["customer"]

	if apiPos.X <= 0 || apiPos.Y <= 0 {
		t.Errorf("scope child shop.api should have positive coordinates: %v", apiPos)
	}
	if dbPos.X <= 0 || dbPos.Y <= 0 {
		t.Errorf("scope child shop.db should have positive coordinates: %v", dbPos)
	}

	// Boundary should have been sized.
	if result.BoundaryWidth < 400 {
		t.Errorf("boundary width should be at least 400, got %v", result.BoundaryWidth)
	}
	if result.BoundaryHeight < 300 {
		t.Errorf("boundary height should be at least 300, got %v", result.BoundaryHeight)
	}

	// Actor external (customer) should be ABOVE the boundary (actors always first row).
	// Boundary starts after actors, so customer.Y should be less than scope children Y.
	if customerPos.Y >= apiPos.Y {
		t.Errorf("actor external should be above scope children: customer.Y=%v, api.Y=%v",
			customerPos.Y, apiPos.Y)
	}
}

func TestComputeLayout_Layered_RowWrapping(t *testing.T) {
	// Create many elements that should wrap to the next row.
	flat := make(map[string]*model.Element)
	var ids []string
	for i := 0; i < 20; i++ {
		id := "elem" + string(rune('a'+i))
		flat[id] = &model.Element{Kind: "system", Title: "Elem"}
		ids = append(ids, id)
	}

	ts := loadTestTemplates(t)
	result := computeLayout(ids, flat, ts, []string{"system"}, "", "layered", nil)

	// With 20 elements and default page width, there should be multiple rows.
	yValues := make(map[float64]bool)
	for _, pos := range result.Positions {
		yValues[pos.Y] = true
	}
	if len(yValues) < 2 {
		t.Errorf("expected row wrapping with 20 elements, got %d unique Y values", len(yValues))
	}
}
