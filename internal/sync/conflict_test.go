package sync

import (
	"strings"
	"testing"
)

func TestModelWinsResolver_NoConflicts(t *testing.T) {
	r := NewModelWinsResolver()
	result := r.Resolve([]Conflict{})
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d items", len(result))
	}
}

func TestModelWinsResolver_SingleConflict(t *testing.T) {
	r := NewModelWinsResolver()
	conflicts := []Conflict{
		{
			ElementID:     "paymentService",
			Field:         "title",
			ModelValue:    "Payment Service",
			DrawioValue:   "Pay Svc",
			LastSyncValue: "Payment Service",
		},
	}
	result := r.Resolve(conflicts)
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	rc := result[0]
	if rc.Winner != "model" {
		t.Errorf("expected winner 'model', got %q", rc.Winner)
	}
	if rc.ElementID != "paymentService" {
		t.Errorf("expected ElementID 'paymentService', got %q", rc.ElementID)
	}
}

func TestModelWinsResolver_MultipleConflicts(t *testing.T) {
	r := NewModelWinsResolver()
	conflicts := []Conflict{
		{ElementID: "elem1", Field: "title", ModelValue: "A", DrawioValue: "B", LastSyncValue: "A"},
		{ElementID: "elem2", Field: "description", ModelValue: "Desc", DrawioValue: "D", LastSyncValue: "Desc"},
		{ElementID: "elem3", Field: "technology", ModelValue: "Go", DrawioValue: "Java", LastSyncValue: "Go"},
	}
	result := r.Resolve(conflicts)
	if len(result) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result))
	}
	for i, rc := range result {
		if rc.Winner != "model" {
			t.Errorf("result[%d]: expected winner 'model', got %q", i, rc.Winner)
		}
	}
}

func TestModelWinsResolver_WarningFormat(t *testing.T) {
	r := NewModelWinsResolver()
	conflicts := []Conflict{
		{
			ElementID:     "userService",
			Field:         "technology",
			ModelValue:    "Go",
			DrawioValue:   "Python",
			LastSyncValue: "Go",
		},
	}
	result := r.Resolve(conflicts)
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	w := result[0].Warning
	checks := []string{
		"userService",
		"technology",
		"Go",
		"Python",
		"Keeping model value",
	}
	for _, check := range checks {
		if !strings.Contains(w, check) {
			t.Errorf("warning missing %q:\n%s", check, w)
		}
	}
}

func TestConflictResolverInterface(t *testing.T) {
	// Ensure ModelWinsResolver satisfies the ConflictResolver interface.
	var _ ConflictResolver = NewModelWinsResolver()
}
