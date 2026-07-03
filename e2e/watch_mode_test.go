package e2e

// TestWatchMode (#492) verifies that `watch` detects a model file change and
// triggers an automatic sync. The test starts watch in the background, modifies
// the model, and asserts the draw.io reflects the change within a timeout.

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
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
	stdout, err := watchCmd.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe: %v", err)
	}
	if err := watchCmd.Start(); err != nil {
		t.Fatalf("start watch: %v", err)
	}
	t.Cleanup(func() {
		// watch handles SIGINT/SIGTERM for graceful shutdown (cmd/bausteinsicht/watch.go).
		// A hard Process.Kill() (SIGKILL on Unix) cannot be intercepted, so the
		// -cover instrumented binary never gets to flush its coverage counters
		// to GOCOVERDIR on exit — this silently zeroed out internal/watcher's
		// E2E coverage despite this test existing. SIGTERM + Wait() lets it
		// exit normally instead; Kill() remains as a fallback if it hangs.
		//
		// On Windows, os/exec's Process.Signal only supports os.Kill — any
		// other signal, including SIGTERM, is a silent no-op returning
		// syscall.EWINDOWS without touching the process (see os/exec_windows.go
		// in the Go stdlib). Sending SIGTERM there would just waste the full
		// 2s timeout below before falling through to the same Kill() this is
		// trying to avoid. So on Windows we skip straight to Kill(): the
		// coverage-flush goal of this cleanup is unreachable on that platform
		// with this mechanism, not silently degraded — see the "Watch Mode"
		// row in E2E-Test-Plan.adoc's Automation Status table.
		done := make(chan struct{})
		go func() { _ = watchCmd.Wait(); close(done) }()
		if runtime.GOOS == "windows" {
			_ = watchCmd.Process.Kill()
			<-done
			return
		}
		_ = watchCmd.Process.Signal(syscall.SIGTERM)
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			_ = watchCmd.Process.Kill()
			<-done
		}
	})

	// Wait for watch's startup log line before mutating the model, instead of
	// a fixed sleep — reduces (but does not fully eliminate; the watcher is
	// registered slightly after this line is printed) the race between the
	// model write below and fsnotify's watch registration.
	readyCh := make(chan struct{})
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), "Watching") {
				close(readyCh)
				return
			}
		}
	}()
	select {
	case <-readyCh:
	case <-time.After(3 * time.Second):
		t.Fatal("watch did not print its startup message within 3s")
	}
	time.Sleep(200 * time.Millisecond)

	// Mutate the model: change customer.title.
	m, loadErr := model.Load(modelPath)
	if loadErr != nil {
		t.Fatalf("model.Load: %v", loadErr)
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
