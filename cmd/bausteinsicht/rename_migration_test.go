package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
	bsync "github.com/docToolchain/Bausteinsicht/internal/sync"
)

const renameDrawio = `<mxfile><diagram id="view-context" name="Context"><mxGraphModel><root>
<mxCell id="0"/><mxCell id="1" parent="0"/>
<object id="context--shop" bausteinsicht_id="shop"><mxCell vertex="1" parent="1"><mxGeometry x="777" as="geometry"/></mxCell></object>
<object id="context--payment" bausteinsicht_id="payment"><mxCell vertex="1" parent="1"/></object>
</root></mxGraphModel></diagram></mxfile>`

// renameDir writes the edit model next to a draw.io file and a sync state that
// know the element shop.
func renameDir(t *testing.T) (modelPath, drawioPath, statePath string) {
	t.Helper()
	modelPath = writeEditCmdModel(t)
	dir := filepath.Dir(modelPath)
	drawioPath = filepath.Join(dir, "architecture.drawio")
	statePath = filepath.Join(dir, ".bausteinsicht-sync")
	if err := os.WriteFile(drawioPath, []byte(renameDrawio), 0o644); err != nil {
		t.Fatal(err)
	}
	state := &bsync.SyncState{
		Elements:         map[string]bsync.ElementState{"shop": {Title: "Shop"}, "payment": {Title: "Payment"}},
		Relationships:    []bsync.RelationshipState{{From: "shop", To: "payment"}},
		RenderedElements: map[string]bool{"shop": true},
	}
	if err := bsync.SaveState(statePath, state); err != nil {
		t.Fatal(err)
	}
	return
}

// rename carries the new ID into the draw.io file and the sync state (#686).
func TestRenameCmd_MigratesTheDrawioFileAndTheSyncState(t *testing.T) {
	modelPath, drawioPath, statePath := renameDir(t)
	out, err := executeRootCmd("rename", "--model", modelPath, "--id", "shop", "--to", "store")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if !strings.Contains(out, "renamed to store") {
		t.Errorf("no note about the migrated cells:\n%s", out)
	}
	data, _ := os.ReadFile(drawioPath)
	if !strings.Contains(string(data), `bausteinsicht_id="store"`) || strings.Contains(string(data), `"shop"`) || !strings.Contains(string(data), `x="777"`) {
		t.Errorf("draw.io file not migrated or geometry lost:\n%s", data)
	}
	state, err := bsync.LoadState(statePath)
	if err != nil {
		t.Fatalf("the migrated state does not load (checksum?): %v", err)
	}
	if _, ok := state.Elements["store"]; !ok || state.Relationships[0].From != "store" || !state.RenderedElements["store"] {
		t.Errorf("sync state not migrated: %+v", state)
	}
	if state.ModelHash == "" || state.DrawioHash == "" {
		t.Errorf("the file hashes were not refreshed: %+v", state)
	}
}

// --dry-run changes no file.
func TestRenameCmd_DryRunLeavesTheDrawioFileAlone(t *testing.T) {
	modelPath, drawioPath, statePath := renameDir(t)
	before, _ := os.ReadFile(drawioPath)
	stateBefore, _ := os.ReadFile(statePath)
	if out, err := executeRootCmd("rename", "--model", modelPath, "--id", "shop", "--to", "store", "--dry-run"); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	after, _ := os.ReadFile(drawioPath)
	stateAfter, _ := os.ReadFile(statePath)
	if string(before) != string(after) || string(stateBefore) != string(stateAfter) {
		t.Error("--dry-run changed the draw.io file or the sync state")
	}
}

// Without a draw.io file there is nothing to migrate and nothing to report.
func TestRenameCmd_WithoutADrawioFile(t *testing.T) {
	modelPath := writeEditCmdModel(t)
	out, err := executeRootCmd("rename", "--model", modelPath, "--id", "shop", "--to", "store")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if strings.Contains(out, "note:") {
		t.Errorf("unexpected note:\n%s", out)
	}
	if _, statErr := os.Stat(filepath.Join(filepath.Dir(modelPath), "architecture.drawio")); statErr == nil {
		t.Error("rename created a draw.io file")
	}
}

// A draw.io file that cannot be read does not undo the rename of the model.
func TestRenameCmd_UnreadableDrawioFileKeepsTheModelRename(t *testing.T) {
	modelPath, drawioPath, _ := renameDir(t)
	if err := os.WriteFile(drawioPath, []byte("<mxfile><diagram"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := executeRootCmd("rename", "--model", modelPath, "--id", "shop", "--to", "store")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if !strings.Contains(out, "the draw.io file was not migrated") {
		t.Errorf("no note about the failed migration:\n%s", out)
	}
	m, loadErr := model.Load(modelPath)
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	if _, ok := m.Model["store"]; !ok {
		t.Error("the model was not renamed")
	}
}

// A corrupted sync state is reported; the draw.io cells are already migrated.
func TestRenameCmd_CorruptedSyncStateIsReported(t *testing.T) {
	modelPath, _, statePath := renameDir(t)
	if err := os.WriteFile(statePath, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := executeRootCmd("rename", "--model", modelPath, "--id", "shop", "--to", "store")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if !strings.Contains(out, "the draw.io file was not migrated") {
		t.Errorf("no note about the corrupted state:\n%s", out)
	}
}

// A draw.io file without the element is saved unchanged and reports no cells.
func TestRenameCmd_NoCellsOfTheElement(t *testing.T) {
	modelPath, drawioPath, _ := renameDir(t)
	if err := os.WriteFile(drawioPath, []byte(strings.ReplaceAll(renameDrawio, `bausteinsicht_id="shop"`, `bausteinsicht_id="other"`)), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := executeRootCmd("rename", "--model", modelPath, "--id", "shop", "--to", "store")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if strings.Contains(out, "renamed to") {
		t.Errorf("reported renamed cells although there were none:\n%s", out)
	}
}
