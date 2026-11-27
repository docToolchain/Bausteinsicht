//go:build !windows

package export

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// hangingDrawio is a stand-in for the draw.io wrapper of the issue: it prints
// a diagnostic, starts a child that outlives it (as xvfb-run/drawio do) and
// never produces the output file.
func hangingDrawio(t *testing.T) (binary, pidFile string) {
	t.Helper()
	dir := t.TempDir()
	pidFile = filepath.Join(dir, "child.pid")
	script := "#!/bin/sh\n" +
		"echo 'GPU process exited unexpectedly: exit_code=9' >&2\n" +
		"sleep 300 &\n" +
		"echo $! > '" + pidFile + "'\n" +
		"wait\n"
	binary = filepath.Join(dir, "drawio")
	if err := os.WriteFile(binary, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return binary, pidFile
}

// A draw.io process that never returns blocked `export` forever (#716).
func TestExportPage_TimesOutAndReportsTheDrawioOutput(t *testing.T) {
	bin, _ := hangingDrawio(t)
	done := make(chan error, 1)
	start := time.Now()
	go func() {
		done <- ExportPage(bin, ExportOptions{
			Format: "png", PageIndex: 1, InputFile: "in.drawio",
			OutputPath: filepath.Join(t.TempDir(), "out.png"),
			Timeout:    time.Second,
		})
	}()
	var err error
	select {
	case err = <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("ExportPage did not return after its timeout")
	}
	if err == nil {
		t.Fatal("expected a timeout error")
	}
	var te *TimeoutError
	if !errors.As(err, &te) {
		t.Fatalf("want a *TimeoutError, got %T: %v", err, err)
	}
	msg := err.Error()
	for _, want := range []string{"timed out after 1s", "GPU process exited unexpectedly", "issues/385"} {
		if !strings.Contains(msg, want) {
			t.Errorf("message lacks %q:\n%s", want, msg)
		}
	}
	if d := time.Since(start); d > 10*time.Second {
		t.Errorf("took %v", d)
	}
}

// The wrapper's children must not survive the timeout.
func TestExportPage_TimeoutKillsTheChildProcesses(t *testing.T) {
	bin, pidFile := hangingDrawio(t)
	_ = ExportPage(bin, ExportOptions{Format: "png", PageIndex: 1, InputFile: "in.drawio",
		OutputPath: filepath.Join(t.TempDir(), "out.png"), Timeout: time.Second})
	data, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("the stand-in did not start its child: %v", err)
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(data)))
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if err := syscall.Kill(pid, 0); err != nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Errorf("child process %d is still running after the timeout", pid)
	_ = syscall.Kill(pid, syscall.SIGKILL)
}

// Timeout 0 means no limit, as before.
func TestExportPage_ZeroTimeoutIsNoLimit(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.png")
	bin := filepath.Join(dir, "drawio")
	script := "#!/bin/sh\nsleep 1\necho x > '" + out + "'\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := ExportPage(bin, ExportOptions{Format: "png", PageIndex: 1, InputFile: "in.drawio", OutputPath: out}); err != nil {
		t.Fatal(err)
	}
}
