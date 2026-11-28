package main

import (
	"strings"
	"testing"
)

// changelog resolves a snapshot ID as --since and --until, so that two snapshots
// can be compared without git (#728).
func TestChangelogCmd_ResolvesSnapshotIDs(t *testing.T) {
	t.Chdir(t.TempDir())
	if out, err := executeRootCmd("init"); err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}
	before := savedSnapshotID(t, "")
	if out, err := executeRootCmd("add", "element", "--id", "newone", "--kind", "actor", "--title", "Brand New"); err != nil {
		t.Fatalf("add element: %v\n%s", err, out)
	}
	after := savedSnapshotID(t, "")

	out, err := executeRootCmd("changelog", "--since", before, "--until", after)
	if err != nil {
		t.Fatalf("changelog between two snapshots: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Brand New") && !strings.Contains(out, "newone") {
		t.Errorf("the added element must be in the changelog:\n%s", out)
	}
	if !strings.Contains(out, before) {
		t.Errorf("the changelog names the snapshot it starts at (%s):\n%s", before, out)
	}

	out, err = executeRootCmd("changelog", "--since", after, "--until", before)
	if err != nil || (!strings.Contains(out, "Brand New") && !strings.Contains(out, "newone")) {
		t.Errorf("the other direction shows the element as removed, got %q (%v)", out, err)
	}
}

// A ref that is neither a snapshot nor a git ref is still an error that says so.
func TestChangelogCmd_UnknownRefIsStillAnError(t *testing.T) {
	t.Chdir(t.TempDir())
	if _, err := executeRootCmd("init"); err != nil {
		t.Fatal(err)
	}
	_, err := executeRootCmd("changelog", "--since", "snapshot-1999-01-01T00-00-00.000000000Z", "--until", "snapshot-1999-01-02T00-00-00.000000000Z")
	if err == nil || !strings.Contains(err.Error(), "snapshot-1999-01-01") {
		t.Errorf("want an error that names the unknown ref, got %v", err)
	}
}
