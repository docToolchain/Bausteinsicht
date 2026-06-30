package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// sharedBinaryPath is set by TestMain before any test runs and is valid for
// the entire test process lifetime. Never use sync.Once + t.Cleanup — the
// cleanup would delete the binary after the first test finishes, breaking all
// subsequent tests in the same run.
var sharedBinaryPath string

// TestMain builds the bausteinsicht binary once before all tests and removes
// it after the run. This is the correct place for process-lifetime setup that
// must outlive individual tests.
func TestMain(m *testing.M) {
	root, err := findModuleRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "findModuleRoot: %v\n", err)
		os.Exit(1)
	}

	bin := filepath.Join(root, "bausteinsicht-e2e-test")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/bausteinsicht")
	cmd.Dir = root
	if out, buildErr := cmd.CombinedOutput(); buildErr != nil {
		fmt.Fprintf(os.Stderr, "build bausteinsicht: %v\n%s", buildErr, out)
		os.Exit(1)
	}
	sharedBinaryPath = bin

	code := m.Run()
	_ = os.Remove(bin)
	os.Exit(code)
}

// buildBinary returns the path to the shared pre-built bausteinsicht binary.
func buildBinary(t *testing.T) string {
	t.Helper()
	if sharedBinaryPath == "" {
		t.Fatal("binary not built: TestMain did not set sharedBinaryPath")
	}
	return sharedBinaryPath
}

// ─── CLI execution helpers ────────────────────────────────────────────────────

// runCLI runs the binary with the given args in dir, returns combined stdout+stderr,
// and fails the test if the command exits with a non-zero code.
func runCLI(t *testing.T, bin, dir string, args ...string) string {
	t.Helper()
	out, code := runCLIAllowFail(t, bin, dir, args...)
	if code != 0 {
		t.Fatalf("bausteinsicht %s: exit %d\n%s", strings.Join(args, " "), code, out)
	}
	return out
}

// runCLIAllowFail runs the binary and returns output + exit code without failing.
func runCLIAllowFail(t *testing.T, bin, dir string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil && cmd.ProcessState == nil {
		// exec itself failed (e.g., binary not found) — surface the error.
		t.Logf("exec %s: %v", bin, err)
	}
	code := 0
	if cmd.ProcessState != nil {
		code = cmd.ProcessState.ExitCode()
	}
	return string(out), code
}

// ─── File helpers ─────────────────────────────────────────────────────────────

// copyFile copies src to dst.
func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("copyFile read %s: %v", src, err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("copyFile write %s: %v", dst, err)
	}
}
