package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/docToolchain/Bausteinsicht/internal/snapshot"
)

func saveTestSnapshot(t *testing.T, dir string) {
	t.Helper()
	m := &model.BausteinsichtModel{Model: map[string]model.Element{"s": {Kind: "system", Title: "S"}}}
	if err := snapshot.NewManager(dir).Save(snapshot.NewSnapshot("a", m)); err != nil {
		t.Fatal(err)
	}
}

// snapshot list follows the global --format json; --output-format stays as the
// older spelling and wins when it is given (#693).
func TestSnapshotList_HonoursTheGlobalFormat(t *testing.T) {
	dir := t.TempDir()
	saveTestSnapshot(t, dir)
	t.Chdir(dir)

	for name, args := range map[string][]string{
		"global --format json": {"snapshot", "list", "--format", "json"},
		"--output-format json": {"snapshot", "list", "--output-format", "json"},
	} {
		out, err := executeImportCmd(args...)
		if err != nil {
			t.Fatalf("%s: %v\n%s", name, err, out)
		}
		var list []map[string]any
		if err := json.Unmarshal([]byte(out), &list); err != nil || len(list) != 1 {
			t.Errorf("%s: want a JSON list with one snapshot, got %q (%v)", name, out, err)
		}
	}

	out, err := executeImportCmd("snapshot", "list")
	if err != nil || !strings.Contains(out, "TIMESTAMP") {
		t.Errorf("the default is still the table, got %q (%v)", out, err)
	}
	out, err = executeImportCmd("snapshot", "list", "--format", "json", "--output-format", "table")
	if err != nil || !strings.Contains(out, "TIMESTAMP") {
		t.Errorf("an explicit --output-format table wins over --format json, got %q (%v)", out, err)
	}
}

// snapshot save uses the global --model like every other command and does not
// declare its own (#693).
func TestSnapshotSave_HasNoOwnModelFlag(t *testing.T) {
	if newSnapshotSaveCmd().Flags().Lookup("model") != nil {
		t.Error("snapshot save declares its own --model, which shadows the global flag")
	}
}

// index.json is not a snapshot: diff, restore and delete answer "snapshot not
// found" and leave the index alone (#693).
func TestSnapshotCommands_IndexIsNotASnapshot(t *testing.T) {
	dir := t.TempDir()
	saveTestSnapshot(t, dir)
	t.Chdir(dir)

	for _, args := range [][]string{
		{"snapshot", "diff", "index"},
		{"snapshot", "restore", "index", "out.jsonc"},
		{"snapshot", "delete", "index"},
	} {
		out, err := executeImportCmd(args...)
		if err == nil || !strings.Contains(err.Error(), "snapshot not found: index") {
			t.Errorf("%v: want \"snapshot not found: index\", got %v\n%s", args, err, out)
		}
	}
	if list, err := snapshot.NewManager(".").List(); err != nil || len(list) != 1 {
		t.Errorf("the index must be intact with one snapshot: %v %v", list, err)
	}
}
