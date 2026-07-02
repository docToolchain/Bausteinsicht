package e2e

// TestWatchMode (#492) verifies that `watch` detects a model file change and
// triggers an automatic sync. The test starts watch in the background, modifies
// the model, and asserts the draw.io reflects the change within a timeout.

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

func TestWatchMode(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	runCLI(t, bin, dir, "init")

	modelPath := filepath.Join(dir, "architecture.jsonc")
	drawioPath := filepath.Join(dir, "architecture.drawio")

	// Start watch in background with a short poll interval.
	watchCmd := exec.Command(bin, "watch",
		"--model", "architecture.jsonc",
	)
	watchCmd.Dir = dir
	if err := watchCmd.Start(); err != nil {
		t.Fatalf("start watch: %v", err)
	}
	t.Cleanup(func() { _ = watchCmd.Process.Kill() })

	// Give watch time to start and perform its initial sync.
	time.Sleep(500 * time.Millisecond)

	// Mutate the model: change customer.title.
	m, err := model.Load(modelPath)
	if err != nil {
		t.Fatalf("model.Load: %v", err)
	}
	cust := m.Model["customer"]
	cust.Title = "Watch Test Customer"
	m.Model["customer"] = cust
	if err := model.Save(modelPath, m); err != nil {
		t.Fatalf("model.Save: %v", err)
	}

	// Wait up to 5 seconds for watch to pick up the change and update draw.io.
	const timeout = 5 * time.Second
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		drawioBytes, err := os.ReadFile(drawioPath)
		if err == nil && strings.Contains(string(drawioBytes), "Watch Test Customer") {
			t.Logf("watch mode synced title change within %v", timeout)
			return
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Check if draw.io was updated at all (may have taken longer or used different title storage).
	drawioBytes, _ := os.ReadFile(drawioPath)
	if strings.Contains(string(drawioBytes), "Watch Test Customer") {
		return
	}
	t.Errorf("watch mode: draw.io did not reflect model change within %v", timeout)
}
