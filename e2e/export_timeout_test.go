//go:build !windows

package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestExportTimesOutWhenDrawioHangs (#716): `export` waited forever for a draw.io
// wrapper that never returned and printed nothing. With --timeout it stops the
// process, names the view and shows what draw.io printed.
func TestExportTimesOutWhenDrawioHangs(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")
	runCLI(t, bin, dir, "sync")

	fake := filepath.Join(dir, "fake-drawio")
	script := "#!/bin/sh\necho 'GPU process exited unexpectedly: exit_code=9' >&2\nsleep 300 &\nwait\n"
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	out, code := runCLIAllowFail(t, bin, dir, "export", "--drawio-path", fake, "--view", "context", "--timeout", "1s")
	if d := time.Since(start); d > 30*time.Second {
		t.Errorf("export took %v with --timeout 1s", d)
	}
	if code == 0 {
		t.Errorf("export reported success although draw.io never finished:\n%s", out)
	}
	for _, want := range []string{`view "context"`, "timed out after 1s", "GPU process exited unexpectedly"} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
}
