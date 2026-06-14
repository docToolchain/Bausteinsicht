package main

import (
	"os"
	"strings"
	"testing"
)

// TestSyncUsesLocalTemplateWithoutFlag verifies that `sync` auto-detects the
// scaffolded "template.drawio" next to the model and applies its custom styles,
// even when no --template flag is given. Regression test for #418.
func TestSyncUsesLocalTemplateWithoutFlag(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Init scaffolds architecture.jsonc + template.drawio + architecture.drawio.
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"init"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Customize the scaffolded template with a sentinel fill color.
	tmpl, err := os.ReadFile("template.drawio")
	if err != nil {
		t.Fatalf("reading template.drawio: %v", err)
	}
	const sentinel = "#ABCDEF"
	customized := strings.ReplaceAll(string(tmpl), "#1168BD", sentinel)
	if customized == string(tmpl) {
		t.Fatal("expected scaffolded template to contain #1168BD to replace")
	}
	if err := os.WriteFile("template.drawio", []byte(customized), 0o600); err != nil {
		t.Fatalf("writing customized template: %v", err)
	}

	// Force a full regeneration so element styles are re-rendered from template.
	if err := os.Remove("architecture.drawio"); err != nil {
		t.Fatalf("removing drawio: %v", err)
	}
	_ = os.Remove(".bausteinsicht-sync")

	// Sync WITHOUT --template — should pick up the local template.drawio.
	captureStdout(t, func() {
		cmd2 := NewRootCmd()
		cmd2.SetArgs([]string{"sync"})
		if err := cmd2.Execute(); err != nil {
			t.Fatalf("sync failed: %v", err)
		}
	})

	out, err := os.ReadFile("architecture.drawio")
	if err != nil {
		t.Fatalf("reading recreated drawio: %v", err)
	}
	if !strings.Contains(string(out), sentinel) {
		t.Errorf("recreated drawio did not use the local template's custom color %s; "+
			"the scaffolded template.drawio was ignored", sentinel)
	}
}

// TestSyncExplicitTemplateWinsOverLocal verifies that an explicit --template flag
// takes precedence over a local template.drawio. Regression test for #418.
func TestSyncExplicitTemplateWinsOverLocal(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"init"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	tmpl, err := os.ReadFile("template.drawio")
	if err != nil {
		t.Fatalf("reading template.drawio: %v", err)
	}

	// Local template gets a "local" sentinel; explicit template gets a different one.
	const localSentinel = "#AAAAAA"
	const explicitSentinel = "#BBBBBB"
	localTmpl := strings.ReplaceAll(string(tmpl), "#1168BD", localSentinel)
	if err := os.WriteFile("template.drawio", []byte(localTmpl), 0o600); err != nil {
		t.Fatalf("writing local template: %v", err)
	}
	explicitTmpl := strings.ReplaceAll(string(tmpl), "#1168BD", explicitSentinel)
	if err := os.WriteFile("custom.drawio", []byte(explicitTmpl), 0o600); err != nil {
		t.Fatalf("writing explicit template: %v", err)
	}

	if err := os.Remove("architecture.drawio"); err != nil {
		t.Fatalf("removing drawio: %v", err)
	}
	_ = os.Remove(".bausteinsicht-sync")

	captureStdout(t, func() {
		cmd2 := NewRootCmd()
		cmd2.SetArgs([]string{"sync", "--template", "custom.drawio"})
		if err := cmd2.Execute(); err != nil {
			t.Fatalf("sync failed: %v", err)
		}
	})

	out, err := os.ReadFile("architecture.drawio")
	if err != nil {
		t.Fatalf("reading recreated drawio: %v", err)
	}
	if !strings.Contains(string(out), explicitSentinel) {
		t.Errorf("explicit --template was not used; expected %s", explicitSentinel)
	}
	if strings.Contains(string(out), localSentinel) {
		t.Errorf("local template.drawio leaked despite explicit --template (%s present)", localSentinel)
	}
}
