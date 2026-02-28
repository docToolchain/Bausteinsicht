package watcher

import (
	"os"
	"sync"
	"testing"
	"time"
)

func TestSingleFileChangeTriggers(t *testing.T) {
	tmp, err := os.CreateTemp("", "watcher-test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	tmp.WriteString("initial")
	tmp.Close()

	var mu sync.Mutex
	var called int
	var lastFile string

	w, err := New([]string{tmp.Name()}, 100*time.Millisecond, func(changedFile string) {
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

	// Trigger a write event
	if err := os.WriteFile(tmp.Name(), []byte("changed"), 0644); err != nil {
		t.Fatal(err)
	}

	// Wait for debounce + processing
	time.Sleep(300 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if called != 1 {
		t.Errorf("expected callback called once, got %d", called)
	}
	if lastFile != tmp.Name() {
		t.Errorf("expected file %s, got %s", tmp.Name(), lastFile)
	}
}

func TestRapidChangesDebounce(t *testing.T) {
	tmp, err := os.CreateTemp("", "watcher-test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	tmp.WriteString("initial")
	tmp.Close()

	var mu sync.Mutex
	var called int

	w, err := New([]string{tmp.Name()}, 200*time.Millisecond, func(changedFile string) {
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

	// Rapid writes within debounce window
	for i := 0; i < 5; i++ {
		if err := os.WriteFile(tmp.Name(), []byte("change"+string(rune('0'+i))), 0644); err != nil {
			t.Fatal(err)
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Wait for debounce to settle
	time.Sleep(400 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if called != 1 {
		t.Errorf("expected callback called once after debounce, got %d", called)
	}
}

func TestStopWorksCleanly(t *testing.T) {
	tmp, err := os.CreateTemp("", "watcher-test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	tmp.WriteString("initial")
	tmp.Close()

	w, err := New([]string{tmp.Name()}, 100*time.Millisecond, func(changedFile string) {})
	if err != nil {
		t.Fatal(err)
	}

	if err := w.Start(); err != nil {
		t.Fatal(err)
	}

	// Stop should not block or panic
	w.Stop()

	// Writing after stop should not trigger callback or panic
	os.WriteFile(tmp.Name(), []byte("after-stop"), 0644)
	time.Sleep(200 * time.Millisecond)
}

func TestSyncingFlagPreventsCallback(t *testing.T) {
	tmp, err := os.CreateTemp("", "watcher-test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	tmp.WriteString("initial")
	tmp.Close()

	var mu sync.Mutex
	var called int

	w, err := New([]string{tmp.Name()}, 100*time.Millisecond, func(changedFile string) {
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

	// Set syncing flag before writing
	w.SetSyncing(true)

	if err := os.WriteFile(tmp.Name(), []byte("syncing-write"), 0644); err != nil {
		t.Fatal(err)
	}

	// Wait for potential debounce
	time.Sleep(300 * time.Millisecond)

	mu.Lock()
	c := called
	mu.Unlock()
	if c != 0 {
		t.Errorf("expected no callback while syncing, got %d", c)
	}

	// Disable syncing, write again — should trigger
	w.SetSyncing(false)

	if err := os.WriteFile(tmp.Name(), []byte("after-sync"), 0644); err != nil {
		t.Fatal(err)
	}

	time.Sleep(300 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if called != 1 {
		t.Errorf("expected callback once after syncing disabled, got %d", called)
	}
}
