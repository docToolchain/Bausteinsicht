package e2e

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// #722: copying an element in draw.io copies bausteinsicht_id with it. Sync
// reports the duplicate and the copy becomes an unmanaged shape.
func TestSyncReportsADuplicatedBausteinsichtID(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")
	runCLI(t, bin, dir, "sync")

	path := filepath.Join(dir, "architecture.drawio")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	xml := string(data)
	block := regexp.MustCompile(`(?s)<object [^>]*bausteinsicht_id="customer"[^>]*>.*?</object>`).FindString(xml)
	if block == "" {
		t.Fatalf("no object for \"customer\" in the draw.io file:\n%s", xml)
	}
	cell := regexp.MustCompile(`^<object [^>]*?\bid="([^"]+)"`).FindStringSubmatch(block)
	if cell == nil {
		t.Fatalf("object without id: %s", block)
	}
	cp := strings.Replace(block, `id="`+cell[1]+`"`, `id="`+cell[1]+`-copy"`, 1)
	xml = strings.Replace(xml, block, block+cp, 1)
	if err := os.WriteFile(path, []byte(xml), 0o644); err != nil {
		t.Fatal(err)
	}
	before := strings.Count(xml, `bausteinsicht_id="customer"`)

	out := runCLI(t, bin, dir, "sync")
	if !strings.Contains(out, `duplicate bausteinsicht_id "customer"`) {
		t.Errorf("sync does not report the duplicate:\n%s", out)
	}
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if after := strings.Count(string(data), `bausteinsicht_id="customer"`); after != before-1 {
		t.Errorf("%d cells with bausteinsicht_id \"customer\" after the sync, want %d", after, before-1)
	}

	// Nothing is left to report on the next run.
	if out := runCLI(t, bin, dir, "sync"); strings.Contains(out, "duplicate bausteinsicht_id") {
		t.Errorf("the duplicate is reported again:\n%s", out)
	}
}
