package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// sharedBinaryPath is set by TestMain before any test runs and is valid for
// the entire test process lifetime.
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

	root, err := findModuleRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "findModuleRoot: %v\n", err)
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
