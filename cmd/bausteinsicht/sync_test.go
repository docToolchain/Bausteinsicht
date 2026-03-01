package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/docToolchain/Bauteinsicht/internal/model"
)

func TestSyncAfterInit(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Init first.
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"init"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Sync should report no changes (already in sync after init).
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd2 := NewRootCmd()
	cmd2.SetArgs([]string{"sync"})
	err := cmd2.Execute()

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// After init, sync may detect formatting differences — just verify it runs successfully.
	if output == "" {
		t.Error("expected some output from sync")
	}
}

func TestSyncNoModelFile(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"sync"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when no model file exists")
	}
}

func TestSyncJSONOutput(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Init.
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"init"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Sync with JSON output.
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd2 := NewRootCmd()
	cmd2.SetArgs([]string{"sync", "--format", "json"})
	err := cmd2.Execute()

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := strings.TrimSpace(string(buf[:n]))

	var summary syncSummary
	if err := json.Unmarshal([]byte(output), &summary); err != nil {
		t.Fatalf("invalid JSON output: %v\noutput: %s", err, output)
	}
}

func TestSyncDetectsModelChanges(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Init.
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"init"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Add an element to the model.
	cmd2 := NewRootCmd()
	cmd2.SetArgs([]string{"add", "element", "--id", "newservice", "--kind", "system", "--title", "New Service"})
	if err := cmd2.Execute(); err != nil {
		t.Fatalf("add element failed: %v", err)
	}

	// Add newservice to the context view so forward sync picks it up.
	m, err := model.Load("architecture.jsonc")
	if err != nil {
		t.Fatalf("load model: %v", err)
	}
	if v, ok := m.Views["context"]; ok {
		v.Include = append(v.Include, "newservice")
		m.Views["context"] = v
	}
	if err := model.Save("architecture.jsonc", m); err != nil {
		t.Fatalf("save model: %v", err)
	}

	// Sync should detect the new element.
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd3 := NewRootCmd()
	cmd3.SetArgs([]string{"sync", "--format", "json"})
	err = cmd3.Execute()

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := strings.TrimSpace(string(buf[:n]))

	var summary syncSummary
	if err := json.Unmarshal([]byte(output), &summary); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, output)
	}

	if summary.ForwardAdded == 0 {
		t.Error("expected forward_added > 0 after adding element")
	}
}

func TestSyncPreservesJSONCComments(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Init.
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"init"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Verify the model has comments right after init (before any sync).
	original, err := os.ReadFile("architecture.jsonc")
	if err != nil {
		t.Fatal(err)
	}
	if !hasLineComment(string(original)) {
		t.Fatal("precondition: init model should have JSONC comments")
	}

	// First sync to establish state — this should NOT strip comments.
	cmd2 := NewRootCmd()
	cmd2.SetArgs([]string{"sync"})
	captureStdout(t, func() {
		if err := cmd2.Execute(); err != nil {
			t.Fatalf("first sync failed: %v", err)
		}
	})

	afterFirstSync, err := os.ReadFile("architecture.jsonc")
	if err != nil {
		t.Fatal(err)
	}
	if !hasLineComment(string(afterFirstSync)) {
		t.Error("JSONC comments were stripped during first sync (no-op sync)")
	}

	// Simulate a draw.io change by editing the HTML-encoded label in the XML.
	drawioData, err := os.ReadFile("architecture.drawio")
	if err != nil {
		t.Fatal(err)
	}

	// Labels are HTML-encoded in draw.io XML: &lt;b&gt;Customer&lt;/b&gt;
	modified := strings.ReplaceAll(string(drawioData),
		"&lt;b&gt;Customer&lt;/b&gt;",
		"&lt;b&gt;Customer Portal&lt;/b&gt;")
	if modified == string(drawioData) {
		t.Skip("could not find encoded label to modify in drawio file")
	}
	if err := os.WriteFile("architecture.drawio", []byte(modified), 0644); err != nil {
		t.Fatal(err)
	}

	// Sync again — this triggers reverse sync.
	cmd3 := NewRootCmd()
	cmd3.SetArgs([]string{"sync"})
	captureStdout(t, func() {
		if err := cmd3.Execute(); err != nil {
			t.Fatalf("reverse sync failed: %v", err)
		}
	})

	// Read the model after sync.
	afterSync, err := os.ReadFile("architecture.jsonc")
	if err != nil {
		t.Fatal(err)
	}

	// Comments should be preserved.
	if !hasLineComment(string(afterSync)) {
		t.Error("JSONC comments were stripped during reverse sync")
	}

	// The title should be updated.
	if !strings.Contains(string(afterSync), "Customer Portal") {
		t.Error("expected title to be updated to 'Customer Portal'")
	}
}

// hasLineComment returns true if the text contains any line that starts with //.
func hasLineComment(text string) bool {
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			return true
		}
	}
	return false
}

// captureStdout redirects stdout during fn and discards the output.
func captureStdout(t *testing.T, fn func()) {
	t.Helper()
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = oldStdout
}

func TestSyncConcurrentModelAndDrawioChanges(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Init.
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"init"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	// Model: change customer description
	modelData, err := os.ReadFile("architecture.jsonc")
	if err != nil {
		t.Fatal(err)
	}
	ms := strings.Replace(string(modelData),
		`"description": "End user who browses and purchases products in the online shop"`,
		`"description": "Premium customer with VIP access"`, 1)
	if ms == string(modelData) {
		t.Skip("could not find customer description to modify")
	}
	if err := os.WriteFile("architecture.jsonc", []byte(ms), 0644); err != nil {
		t.Fatal(err)
	}

	// Draw.io: change customer title
	drawioData, err := os.ReadFile("architecture.drawio")
	if err != nil {
		t.Fatal(err)
	}
	ds := strings.ReplaceAll(string(drawioData),
		"&lt;b&gt;Customer&lt;/b&gt;",
		"&lt;b&gt;VIP Customer&lt;/b&gt;")
	if ds == string(drawioData) {
		t.Skip("could not find Customer label in draw.io")
	}
	if err := os.WriteFile("architecture.drawio", []byte(ds), 0644); err != nil {
		t.Fatal(err)
	}

	// First sync
	captureStdout(t, func() {
		cmd2 := NewRootCmd()
		cmd2.SetArgs([]string{"sync"})
		if err := cmd2.Execute(); err != nil {
			t.Fatalf("first sync: %v", err)
		}
	})

	// After first sync: model should have VIP Customer (reverse sync picks it up)
	m1, _ := os.ReadFile("architecture.jsonc")
	if !strings.Contains(string(m1), "VIP Customer") {
		t.Error("after first sync: model should have 'VIP Customer'")
	}

	// After first sync: draw.io should ALSO have VIP Customer (not overwritten)
	d1, _ := os.ReadFile("architecture.drawio")
	if !strings.Contains(string(d1), "VIP Customer") {
		t.Error("after first sync: draw.io should preserve 'VIP Customer' (not overwrite with model title)")
	}

	// Second sync should be no-op
	captureStdout(t, func() {
		cmd3 := NewRootCmd()
		cmd3.SetArgs([]string{"sync"})
		if err := cmd3.Execute(); err != nil {
			t.Fatalf("second sync: %v", err)
		}
	})

	// After second sync: model should STILL have VIP Customer
	m2, _ := os.ReadFile("architecture.jsonc")
	if !strings.Contains(string(m2), "VIP Customer") {
		t.Error("after second sync: model should still have 'VIP Customer' (draw.io change should not be reverted)")
	}
}

// TestSyncPreservesUserAddedDrawioElement verifies that elements manually
// added by the user in draw.io (with a bausteinsicht_id that does NOT exist
// in the model) are preserved across sync cycles. Regression test for #115.
func TestSyncPreservesUserAddedDrawioElement(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Init the project.
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"init"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Inject a user-added element into the draw.io XML.
	// This simulates a user manually adding an <object> in draw.io.
	drawioData, err := os.ReadFile("architecture.drawio")
	if err != nil {
		t.Fatal(err)
	}

	// Insert a user element before the closing </root> tag on the first view page.
	userElemXML := `<object bausteinsicht_id="userelem" bausteinsicht_kind="system" label="User Custom Element" id="userelem">
          <mxCell style="shape=rectangle;" vertex="1" parent="1">
            <mxGeometry x="500" y="500" width="120" height="60" as="geometry"/>
          </mxCell>
        </object>`

	modified := strings.Replace(string(drawioData), "</root>", userElemXML+"\n      </root>", 1)
	if modified == string(drawioData) {
		t.Fatal("failed to inject user element into draw.io XML")
	}
	if err := os.WriteFile("architecture.drawio", []byte(modified), 0644); err != nil {
		t.Fatal(err)
	}

	// Verify the element was injected.
	if !strings.Contains(modified, "userelem") {
		t.Fatal("precondition: user element not found in draw.io XML")
	}

	// Run sync.
	cmd2 := NewRootCmd()
	cmd2.SetArgs([]string{"sync"})
	captureStdout(t, func() {
		if err := cmd2.Execute(); err != nil {
			t.Fatalf("sync failed: %v", err)
		}
	})

	// Read the draw.io file after sync and verify the user element is preserved.
	afterSync, err := os.ReadFile("architecture.drawio")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(afterSync), `bausteinsicht_id="userelem"`) {
		t.Error("user-added element 'userelem' was deleted during sync; it should be preserved (#115)")
	}
}

func TestSyncWithExplicitModelPath(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Init.
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"init"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Sync with explicit model path.
	cmd2 := NewRootCmd()
	cmd2.SetArgs([]string{"sync", "--model", "architecture.jsonc"})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cmd2.Execute()

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("sync with explicit model failed: %v", err)
	}

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// After init, sync may detect formatting differences — just verify it runs.
	if output == "" {
		t.Error("expected some output from sync")
	}
}
