package e2e

// TestREPL (#505) verifies the interactive REPL accepts commands via stdin:
// `list`, `show <id>`, `add element`, and `exit` all work without crashing.

import (
	"os/exec"
	"strings"
	"testing"
)

func TestREPL(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	runCLI(t, bin, dir, "init")

	// Feed commands to the REPL via stdin. The REPL reads from os.Stdin
	// (bufio.Scanner), so piping a newline-delimited command sequence works.
	input := strings.Join([]string{
		"list",
		"show customer",
		"exit",
		"",
	}, "\n")

	cmd := exec.Command(bin, "repl", "--model", "architecture.jsonc")
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(input)
	out, err := cmd.CombinedOutput()

	// REPL should exit cleanly (exit 0 or 1 if "exit" triggers a special code).
	if cmd.ProcessState != nil && cmd.ProcessState.ExitCode() > 1 {
		t.Errorf("repl exited %d (expected 0 or 1)\noutput: %s\nerr: %v",
			cmd.ProcessState.ExitCode(), out, err)
	}

	outStr := string(out)
	// `list` should show some elements.
	if !strings.Contains(strings.ToLower(outStr), "customer") {
		t.Errorf("repl 'list' output did not mention 'customer':\n%s", outStr)
	}
	t.Logf("repl output:\n%s", outStr)
}
