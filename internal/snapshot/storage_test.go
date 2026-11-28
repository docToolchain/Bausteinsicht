package snapshot

import (
	"os"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// The directory holds index.json next to the snapshots; it is not a snapshot,
// so "index" is an unknown ID for every operation, and deleting it must not
// remove the index (#693).
func TestManager_IndexIsNotASnapshot(t *testing.T) {
	m := NewManager(t.TempDir())
	snap := NewSnapshot("a", &model.BausteinsichtModel{})
	if err := m.Save(snap); err != nil {
		t.Fatal(err)
	}
	if !m.Exists(snap.ID) {
		t.Fatal("the saved snapshot must exist")
	}
	if m.Exists("index") {
		t.Error("index.json is not a snapshot")
	}
	if _, err := m.Load("index"); err == nil || !strings.Contains(err.Error(), "snapshot not found") {
		t.Errorf("Load(index): want a not-found error, got %v", err)
	}
	if err := m.Delete("index"); err == nil || !strings.Contains(err.Error(), "snapshot not found") {
		t.Errorf("Delete(index): want a not-found error, got %v", err)
	}
	if _, err := os.Stat(m.indexPath()); err != nil {
		t.Errorf("the index was removed: %v", err)
	}
	if list, err := m.List(); err != nil || len(list) != 1 {
		t.Errorf("the saved snapshot must still be listed: %v %v", list, err)
	}
}
