package main

import (
	"fmt"
	"os"
	"strings"
	stdsync "sync"
	"testing"
	"time"
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

// Every command that changes the model takes the lock before it reads the model
// and reports a lock that cannot be taken (here: a symbolic link at the lock
// path, which is refused) as an error, without touching the model or the link's
// target.
func TestChangingCommandsReportAnUnusableLock(t *testing.T) {
	commands := map[string][]string{
		"add element":                {"add", "element", "--id", "z1", "--kind", "system", "--title", "Z"},
		"add relationship":           {"add", "relationship", "--from", "customer", "--to", "webshop", "--kind", "uses"},
		"add view":                   {"add", "view", "vz", "--title", "VZ", "--include", "customer"},
		"add specification element":  {"add", "specification", "element", "kz", "--notation", "KZ"},
		"add specification relation": {"add", "specification", "relationship", "rz", "--notation", "RZ"},
		"add-from-pattern":           {"add", "add-from-pattern", "svc", "--id", "z2"},
		"update":                     {"update", "element", "--id", "customer", "--title", "C2"},
		"rename":                     {"rename", "--id", "customer", "--to", "c2"},
		"remove":                     {"remove", "element", "--id", "customer"},
		"sync":                       {"sync"},
	}
	for name, args := range commands {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			modelPath := writeSampleModel(t, dir)
			before, _ := os.ReadFile(modelPath)
			victim := dir + "/victim.txt"
			if err := os.WriteFile(victim, []byte("keep\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(victim, modelPath+".lock"); err != nil {
				t.Skipf("symbolic links are not available: %v", err)
			}

			err := runInProcess(append(args, "--model", modelPath)...)

			if err == nil || !strings.Contains(err.Error(), "symbolic link") {
				t.Fatalf("the command must fail because the lock cannot be taken, got %v", err)
			}
			if after, _ := os.ReadFile(modelPath); string(after) != string(before) {
				t.Error("the model was changed although the lock could not be taken")
			}
			if data, _ := os.ReadFile(victim); string(data) != "keep\n" {
				t.Errorf("the target of the link was modified: %q", data)
			}
		})
	}
}

// A model that cannot be loaded must release the lock again (the helper of the
// two "add specification" commands took it before the load).
func TestAddSpecificationReleasesTheLockWhenTheModelCannotBeLoaded(t *testing.T) {
	dir := t.TempDir()
	modelPath := dir + "/architecture.jsonc"
	if err := os.WriteFile(modelPath, []byte("{ not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := runInProcess("add", "specification", "element", "kz", "--notation", "KZ", "--model", modelPath); err == nil {
		t.Fatal("a broken model must be reported")
	}
	// The lock must be free again: a second command gets it at once.
	done := make(chan error, 1)
	go func() {
		done <- runInProcess("add", "specification", "element", "kz", "--notation", "KZ", "--model", modelPath)
	}()
	select {
	case err := <-done:
		if err != nil && strings.Contains(err.Error(), "timed out") {
			t.Fatalf("the lock was not released: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the second command is blocked: the lock was not released")
	}
}
