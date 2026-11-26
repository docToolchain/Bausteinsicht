package main

import (
	"fmt"
	"os"
	"strings"
	stdsync "sync"
	"testing"
)

// runInProcess runs one command like the CLI would, in the current process.
func runInProcess(args ...string) error {
	cmd := NewRootCmd()
	cmd.SetOut(&strings.Builder{})
	cmd.SetErr(&strings.Builder{})
	cmd.SetArgs(args)
	return cmd.Execute()
}

// TestConcurrentWritesLoseNoUpdate (#712): commands that change the model read
// it, change it and write it back. Run at the same time, they overwrote each
// other: every command reported success and part of the changes was gone (7 of
// 10 elements after ten parallel "add element" runs).
func TestConcurrentWritesLoseNoUpdate(t *testing.T) {
	const n = 30
	dir := t.TempDir()
	modelPath := writeSampleModel(t, dir)

	var wg stdsync.WaitGroup
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = runInProcess("add", "element", "--model", modelPath,
				"--id", fmt.Sprintf("q%d", i), "--kind", "system", "--title", fmt.Sprintf("Q%d", i))
		}(i)
	}
	wg.Wait()

	data, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	missing := 0
	for i := 0; i < n; i++ {
		if errs[i] != nil {
			t.Errorf("command %d failed: %v", i, errs[i])
		}
		if !strings.Contains(string(data), fmt.Sprintf(`"q%d"`, i)) {
			missing++
		}
	}
	if missing > 0 {
		t.Fatalf("%d of %d elements are missing from the model although every command reported success", missing, n)
	}
}

// The other commands that change the model take part in the same race.
func TestConcurrentMixedWritesLoseNoUpdate(t *testing.T) {
	dir := t.TempDir()
	modelPath := writeSampleModel(t, dir)

	jobs := [][]string{}
	for i := 0; i < 10; i++ {
		id := fmt.Sprintf("n%d", i)
		jobs = append(jobs,
			[]string{"add", "element", "--model", modelPath, "--id", id, "--kind", "system", "--title", strings.ToUpper(id)},
			[]string{"add", "specification", "element", "kind" + id, "--model", modelPath, "--notation", "N" + id},
			[]string{"add", "view", "v" + id, "--model", modelPath, "--title", "V" + id, "--include", "customer"},
		)
	}
	var wg stdsync.WaitGroup
	errs := make([]error, len(jobs))
	for i, job := range jobs {
		wg.Add(1)
		go func(i int, job []string) {
			defer wg.Done()
			errs[i] = runInProcess(job...)
		}(i, job)
	}
	wg.Wait()

	data, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	for i, job := range jobs {
		if errs[i] != nil {
			t.Errorf("%v failed: %v", job[:3], errs[i])
			continue
		}
		var want string
		switch job[1] {
		case "element":
			want = `"` + job[5] + `"`
		case "specification":
			want = `"` + job[3] + `"`
		default:
			want = `"` + job[2] + `"`
		}
		if !strings.Contains(string(data), want) {
			t.Errorf("the change of %v is missing from the model (%s)", job[:3], want)
		}
	}
}
