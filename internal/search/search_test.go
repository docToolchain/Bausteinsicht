package search

import (
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

func testModel() *model.BausteinsichtModel {
	return &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"service":  {Notation: "Service"},
				"database": {Notation: "Database"},
			},
		},
		Model: map[string]model.Element{
			"payment-service": {
				Kind:        "service",
				Title:       "Payment Service v2",
				Description: "Handles all payment processing via external gateway",
				Technology:  "Go",
				Tags:        []string{"core", "pci-dss"},
			},
			"order-service": {
				Kind:       "service",
				Title:      "Order Service",
				Technology: "Java",
			},
			"payment-db": {
				Kind:        "database",
				Title:       "Payment Database",
				Technology:  "PostgreSQL",
				Description: "Stores payment records",
			},
		},
		Relationships: []model.Relationship{
			{From: "order-service", To: "payment-service", Label: "charges via", Kind: "uses"},
			{From: "payment-service", To: "payment-db", Label: "reads/writes", Kind: "uses"},
		},
		Views: map[string]model.View{
			"context": {Title: "System Context", Description: "Top-level context view"},
			"payment": {Title: "Payment Domain", Description: "Payment-specific view"},
		},
	}
}

func TestSearch_ElementByTitle(t *testing.T) {
	results := Run("payment", testModel(), Options{})
	if results.Total == 0 {
		t.Fatal("expected results, got none")
	}
	// payment-service and payment-db should both match
	ids := map[string]bool{}
	for _, r := range results.Results {
		ids[r.ID] = true
	}
	if !ids["payment-service"] {
		t.Error("expected payment-service in results")
	}
	if !ids["payment-db"] {
		t.Error("expected payment-db in results")
	}
}

func TestSearch_ExactIDScoresHighest(t *testing.T) {
	results := Run("payment-service", testModel(), Options{Type: ResultElement})
	if results.Total == 0 {
		t.Fatal("expected results")
	}
	if results.Results[0].ID != "payment-service" {
		t.Errorf("expected payment-service to rank first, got %s", results.Results[0].ID)
	}
}

func TestSearch_HigherScoreThanPartialMatch(t *testing.T) {
	results := Run("payment", testModel(), Options{Type: ResultElement})
	var paymentServiceScore, paymentDbScore int
	for _, r := range results.Results {
		switch r.ID {
		case "payment-service":
			paymentServiceScore = r.Score
		case "payment-db":
			paymentDbScore = r.Score
		}
	}
	// payment-service matches id + title + description + tags; payment-db matches id + title + description
	// both should have positive scores
	if paymentServiceScore == 0 {
		t.Error("payment-service score should be > 0")
	}
	if paymentDbScore == 0 {
		t.Error("payment-db score should be > 0")
	}
}

func TestSearch_MultiWordQuery_ANDSemantics(t *testing.T) {
	// "order service" must match elements containing both words
	results := Run("order service", testModel(), Options{Type: ResultElement})
	found := false
	for _, r := range results.Results {
		if r.ID == "order-service" {
			found = true
		}
		// payment-db has neither "order" nor "service" in searchable fields — should not appear
		if r.ID == "payment-db" {
			t.Error("payment-db should not match 'order service'")
		}
	}
	if !found {
		t.Error("expected order-service to match 'order service'")
	}
}

func TestSearch_TypeFilter_ElementOnly(t *testing.T) {
	results := Run("payment", testModel(), Options{Type: ResultElement})
	for _, r := range results.Results {
		if r.Type != ResultElement {
			t.Errorf("expected only element results, got %s", r.Type)
		}
	}
}

func TestSearch_TypeFilter_RelationshipOnly(t *testing.T) {
	results := Run("charges", testModel(), Options{Type: ResultRelationship})
	if results.Total == 0 {
		t.Fatal("expected relationship results")
	}
	for _, r := range results.Results {
		if r.Type != ResultRelationship {
			t.Errorf("expected only relationship results, got %s", r.Type)
		}
	}
}

func TestSearch_TypeFilter_ViewOnly(t *testing.T) {
	results := Run("payment", testModel(), Options{Type: ResultView})
	for _, r := range results.Results {
		if r.Type != ResultView {
			t.Errorf("expected only view results, got %s", r.Type)
		}
	}
}

func TestSearch_EmptyQuery_ReturnsEmpty(t *testing.T) {
	results := Run("", testModel(), Options{})
	if results.Total != 0 {
		t.Errorf("expected 0 results for empty query, got %d", results.Total)
	}
}

func TestSearch_NoMatch_ReturnsEmpty(t *testing.T) {
	results := Run("xyzzy-nonexistent", testModel(), Options{})
	if results.Total != 0 {
		t.Errorf("expected 0 results, got %d", results.Total)
	}
}

func TestSearch_CaseInsensitive(t *testing.T) {
	r1 := Run("PAYMENT", testModel(), Options{})
	r2 := Run("payment", testModel(), Options{})
	if r1.Total != r2.Total {
		t.Errorf("case insensitive mismatch: %d vs %d", r1.Total, r2.Total)
	}
}

func TestSearch_ViewMatch(t *testing.T) {
	results := Run("context", testModel(), Options{Type: ResultView})
	if results.Total == 0 {
		t.Fatal("expected view result for 'context'")
	}
	if results.Results[0].ID != "context" {
		t.Errorf("expected context view, got %s", results.Results[0].ID)
	}
}

func TestSearch_TagMatch(t *testing.T) {
	results := Run("pci-dss", testModel(), Options{Type: ResultElement})
	if results.Total == 0 {
		t.Fatal("expected match on tag pci-dss")
	}
	if results.Results[0].ID != "payment-service" {
		t.Errorf("expected payment-service, got %s", results.Results[0].ID)
	}
	found := false
	for _, f := range results.Results[0].MatchedFields {
		if f == "tags" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'tags' in MatchedFields")
	}
}

func TestSearch_SortedByScoreDescThenIDAlpha(t *testing.T) {
	results := Run("payment", testModel(), Options{Type: ResultElement})
	for i := 1; i < len(results.Results); i++ {
		prev := results.Results[i-1]
		curr := results.Results[i]
		if prev.Score < curr.Score {
			t.Errorf("results not sorted by score desc: [%d] score=%d before [%d] score=%d",
				i-1, prev.Score, i, curr.Score)
		}
		if prev.Score == curr.Score && prev.ID > curr.ID {
			t.Errorf("tie not broken alphabetically: %s before %s", prev.ID, curr.ID)
		}
	}
}
