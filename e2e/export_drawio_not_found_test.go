package e2e

// TestExportDrawioNotFound verifies the draw.io-binary-resolution error path
// in `export` (ResolveDrawioBinary/DetectDrawioBinary/buildDrawioNotFoundError),
// deterministically and without needing a real draw.io CLI or a headless
// display — the existing SVGExportAndLinks subtest in bigbank_arc42_test.go
// only covers the happy path and skips entirely when draw.io is absent, so
// this failure path had no E2E coverage on any environment.

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/export"
)

func TestExportDrawioNotFound(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	runCLI(t, bin, dir, "init")

	t.Run("ExplicitPathMissing", func(t *testing.T) {
		// --drawio-path pointing at a nonexistent file: verifyExplicitBinary's
		// error path, distinct from the auto-detect path below.
		out, code := runCLIAllowFail(t, bin, dir, "export",
			"--model", "architecture.jsonc",
			"--drawio-path", "/nonexistent/path/to/drawio",
		)
		if code == 0 {
			t.Fatal("export --drawio-path <missing>: expected non-zero exit")
		}
		if !strings.Contains(out, "not found") || !strings.Contains(out, "--drawio-path") {
			t.Errorf("expected an error naming --drawio-path and \"not found\", got: %s", out)
		}
	})

	t.Run("EnvVarPathMissing", func(t *testing.T) {
		// BAUSTEINSICHT_DRAWIO_PATH set (no --drawio-path flag) to a
		// nonexistent file: exercises ResolveDrawioBinary's second precedence
		// tier (env var over auto-detection) and its own distinct error
		// message naming the env var instead of the flag. Deterministic and
		// environment-independent, unlike forcing full auto-detection
		// failure — this sandbox has a real draw.io install reachable via
		// platform-specific absolute paths (checked directly with os.Stat,
		// not just $PATH lookup), so clearing $PATH alone doesn't force
		// DetectDrawioBinary to fail here.
		cmd := exec.Command(bin, "export", "--model", "architecture.jsonc")
		cmd.Dir = dir
		cmd.Env = append(cmd.Environ(), export.DrawioPathEnvVar+"=/nonexistent/path/to/drawio")
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatal("export with BAUSTEINSICHT_DRAWIO_PATH=<missing>: expected non-zero exit")
		}
		if !strings.Contains(string(out), "not found") || !strings.Contains(string(out), export.DrawioPathEnvVar) {
			t.Errorf("expected an error naming %s and \"not found\", got: %s", export.DrawioPathEnvVar, out)
		}
	})
}
