package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestParallelProcessesLoseNoUpdate (#712): ten or more bausteinsicht processes
// that change the model at the same time (scripts, LLM agents, a watch plus a
// manual command) overwrote each other; every process exited with 0 and said
// "Added element", and only 7 of 10 elements were in the file.
func TestParallelProcessesLoseNoUpdate(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	const n = 20
	type result struct {
		out  []byte
		err  error
		args []string
	}
	done := make(chan result, n+5)
	run := func(args ...string) {
		cmd := exec.Command(bin, args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		done <- result{out, err, args}
	}
	for i := 0; i < n; i++ {
		go run("--model", "architecture.jsonc", "add", "element", "--id", fmt.Sprintf("q%d", i), "--kind", "actor", "--title", fmt.Sprintf("Q%d", i))
	}
	for i := 0; i < 5; i++ {
		go run("--model", "architecture.jsonc", "add", "view", fmt.Sprintf("pv%d", i), "--title", fmt.Sprintf("PV%d", i), "--include", "customer")
	}
	for i := 0; i < n+5; i++ {
		r := <-done
		if r.err != nil {
			t.Errorf("%v: %v\n%s", r.args, r.err, r.out)
		}
	}

	data, err := os.ReadFile(dir + "/architecture.jsonc")
	if err != nil {
		t.Fatal(err)
	}
	var missing []string
	for i := 0; i < n; i++ {
		if !strings.Contains(string(data), fmt.Sprintf(`"q%d"`, i)) {
			missing = append(missing, fmt.Sprintf("element q%d", i))
		}
	}
	for i := 0; i < 5; i++ {
		if !strings.Contains(string(data), fmt.Sprintf(`"pv%d"`, i)) {
			missing = append(missing, fmt.Sprintf("view pv%d", i))
		}
	}
	if len(missing) > 0 {
		t.Fatalf("%d changes are missing although every command exited with 0: %v", len(missing), missing)
	}
	runCLI(t, bin, dir, "validate")
}
