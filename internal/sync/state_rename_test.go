package sync

import "testing"

func TestSyncState_RenameElement(t *testing.T) {
	s := &SyncState{
		Elements: map[string]ElementState{"shop": {Title: "S"}, "shop.api": {Title: "A"}, "shopper": {Title: "P"}},
		Relationships: []RelationshipState{
			{From: "shop.api", To: "shopper", Index: 0}, {From: "shopper", To: "shop", Index: 1},
		},
		RenderedElements: map[string]bool{"shop": true, "shop.api": true, "shopper": true},
	}
	s.RenameElement("shop", "store")

	for _, id := range []string{"store", "store.api", "shopper"} {
		if _, ok := s.Elements[id]; !ok {
			t.Errorf("element %q missing: %v", id, s.Elements)
		}
		if !s.RenderedElements[id] {
			t.Errorf("rendered %q missing: %v", id, s.RenderedElements)
		}
	}
	if _, ok := s.Elements["shop"]; ok || len(s.Elements) != 3 {
		t.Errorf("the old key is left: %v", s.Elements)
	}
	if s.Relationships[0].From != "store.api" || s.Relationships[0].To != "shopper" ||
		s.Relationships[1].From != "shopper" || s.Relationships[1].To != "store" {
		t.Errorf("relationships: %+v", s.Relationships)
	}
}
