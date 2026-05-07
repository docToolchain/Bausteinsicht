package watcher

import (
	"os"
	"sync"
	"testing"
	"time"
)

func createTempFile(t *testing.T) string {
	t.Helper()
	tmp, err := os.CreateTemp("", "watcher-test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	name := tmp.Name()
	if _, err := tmp.WriteString("initial"); err != nil {
		t.Fatal(err)
	}
	if err := tmp.Close(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(name) })
	return name
}

func TestSingleFileChangeTriggers(t *testing.T) {
	path := createTempFile(t)

	var mu sync.Mutex
	var called int
	var lastFile string

	w, err := New([]string{path}, 100*time.Millisecond, func(changedFile string) {
		mu.Lock()
		defer mu.Unlock()
		called++
		lastFile = changedFile
	})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Stop()

	if err := w.Start(); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, []byte("changed"), 0644); err != nil {
		t.Fatal(err)
	}

	time.Sleep(300 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if called != 1 {
		t.Errorf("expected callback called once, got %d", called)
	}
	if lastFile != path {
		t.Errorf("expected file %s, got %s", path, lastFile)
	}
}

func TestRapidChangesDebounce(t *testing.T) {
	path := createTempFile(t)

	var mu sync.Mutex
	var called int

	w, err := New([]string{path}, 300*time.Millisecond, func(changedFile string) {
		mu.Lock()
		defer mu.Unlock()
		called++
	})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Stop()

	if err := w.Start(); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 5; i++ {
		if err := os.WriteFile(path, []byte("change"+string(rune('0'+i))), 0644); err != nil {
			t.Fatal(err)
		}
		time.Sleep(50 * time.Millisecond)
	}

	time.Sleep(600 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if called != 1 {
		t.Errorf("expected callback called once after debounce, got %d", called)
	}
}

func TestStopWorksCleanly(t *testing.T) {
	path := createTempFile(t)

	w, err := New([]string{path}, 100*time.Millisecond, func(changedFile string) {})
	if err != nil {
		t.Fatal(err)
	}

	if err := w.Start(); err != nil {
		t.Fatal(err)
	}

	w.Stop()

	// Writing after stop should not trigger callback or panic
	if err := os.WriteFile(path, []byte("after-stop"), 0644); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
}

func TestFileRedetectedAfterDeleteRecreate(t *testing.T) {
	path := createTempFile(t)

	var mu sync.Mutex
	var called int

	w, err := New([]string{path}, 100*time.Millisecond, func(changedFile string) {
		mu.Lock()
		defer mu.Unlock()
		called++
	})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Stop()

	if err := w.Start(); err != nil {
		t.Fatal(err)
	}

	// Delete the file
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)

	// Recreate the file
	if err := os.WriteFile(path, []byte("recreated"), 0644); err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)

	// Modify the recreated file
	if err := os.WriteFile(path, []byte("modified-after-recreate"), 0644); err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if called < 1 {
		t.Errorf("expected at least 1 callback after file delete+recreate, got %d", called)
	}
}

// TestFileRedetectedAfterSlowRecreate verifies that the watcher recovers
// when a file is deleted and recreated after more than 1 second. Regression
// test for #268: rewatch only polled 20x50ms=1s, then gave up.
func TestFileRedetectedAfterSlowRecreate(t *testing.T) {
	path := createTempFile(t)

	called := make(chan string, 5)

	w, err := New([]string{path}, 100*time.Millisecond, func(changedFile string) {
		select {
		case called <- changedFile:
		default:
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Stop()

	if err := w.Start(); err != nil {
		t.Fatal(err)
	}

	// Delete the file.
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}

	// Wait longer than the old 1s timeout.
	time.Sleep(1500 * time.Millisecond)

	// Recreate the file — watcher should still detect this.
	if err := os.WriteFile(path, []byte("late-recreate"), 0644); err != nil {
		t.Fatal(err)
	}

	select {
	case changedFile := <-called:
		if changedFile != path {
			t.Errorf("expected callback for %s, got %s", path, changedFile)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for callback after slow file recreate (>1s)")
	}
}

func TestAtomicRenameTriggersCallback(t *testing.T) {
	path := createTempFile(t)

	called := make(chan string, 1)

	w, err := New([]string{path}, 100*time.Millisecond, func(changedFile string) {
		select {
		case called <- changedFile:
		default:
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Stop()

	if err := w.Start(); err != nil {
		t.Fatal(err)
	}

	// Perform atomic rename: write new content to a temp file, then rename
	// over the watched file. This is what sed -i and vim do.
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, []byte("atomically-replaced"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(tmpPath) })

	if err := os.Rename(tmpPath, path); err != nil {
		t.Fatal(err)
	}

	select {
	case changedFile := <-called:
		if changedFile != path {
			t.Errorf("expected callback for %s, got %s", path, changedFile)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for callback after atomic rename")
	}
}

func TestSyncingFlagPreventsCallback(t *testing.T) {
	path := createTempFile(t)

	var mu sync.Mutex
	var called int

	w, err := New([]string{path}, 100*time.Millisecond, func(changedFile string) {
		mu.Lock()
		defer mu.Unlock()
		called++
	})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Stop()

	if err := w.Start(); err != nil {
		t.Fatal(err)
	}

	w.SetSyncing(true)

	if err := os.WriteFile(path, []byte("syncing-write"), 0644); err != nil {
		t.Fatal(err)
	}

	time.Sleep(300 * time.Millisecond)

	mu.Lock()
	c := called
	mu.Unlock()
	if c != 0 {
		t.Errorf("expected no callback while syncing, got %d", c)
	}

	w.SetSyncing(false)

	if err := os.WriteFile(path, []byte("after-sync"), 0644); err != nil {
		t.Fatal(err)
	}

	time.Sleep(300 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if called != 1 {
		t.Errorf("expected callback once after syncing disabled, got %d", called)
	}
}
