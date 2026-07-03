package e2e

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
//
// If BAUSTEINSICHT_E2E_BIN is set, that pre-built binary is used as-is and no
// build step occurs. This enables binary-level coverage measurement: build the
// binary with `go build -cover`, set GOCOVERDIR, and the instrumented binary
// writes coverage data for each CLI invocation during the E2E run.
func TestMain(m *testing.M) {
	if bin := os.Getenv("BAUSTEINSICHT_E2E_BIN"); bin != "" {
		sharedBinaryPath = bin
		os.Exit(m.Run())
	}

	root, err := findModuleRootPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "findModuleRootPath: %v\n", err)
		os.Exit(1)
	}

	bin := filepath.Join(root, "bausteinsicht-e2e-test")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
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

// ─── File helpers ─────────────────────────────────────────────────────────────

// findModuleRoot returns the module root directory, failing the test on error.
func findModuleRoot(t *testing.T) string {
	t.Helper()
	root, err := findModuleRootPath()
	if err != nil {
		t.Fatalf("findModuleRoot: %v", err)
	}
	return root
}

// copyTestFile copies src to dst, failing the test on error.
func copyTestFile(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("copyTestFile read %s: %v", src, err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("copyTestFile write %s: %v", dst, err)
	}
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
		// exec itself failed (e.g., binary not found) — fatal to avoid misleading downstream errors.
		t.Fatalf("exec %s: %v", bin, err)
	}
	code := 0
	if cmd.ProcessState != nil {
		code = cmd.ProcessState.ExitCode()
	}
	return string(out), code
}

// runCLISplit runs the binary with stdout and stderr captured separately
// (unlike runCLI/runCLIAllowFail, which merge them) and returns both plus
// the exit code, without failing the test on a non-zero exit.
func runCLISplit(t *testing.T, bin, dir string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	if err != nil && cmd.ProcessState == nil {
		t.Fatalf("exec %s: %v", bin, err)
	}
	if cmd.ProcessState != nil {
		code = cmd.ProcessState.ExitCode()
	}
	return outBuf.String(), errBuf.String(), code
}
