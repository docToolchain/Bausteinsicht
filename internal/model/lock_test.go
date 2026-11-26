package model

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func newLockTarget(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "architecture.jsonc")
	if err := os.WriteFile(p, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLockModel_CreatesAndRemovesTheLockFile(t *testing.T) {
	p := newLockTarget(t)
	unlock, err := LockModel(p)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(p + ".lock")
	if err != nil {
		t.Fatalf("the lock file must exist while the lock is held: %v", err)
	}
	if !strings.HasSuffix(string(data), "\n") || strings.TrimSpace(string(data)) == "" {
		t.Errorf("the lock file must name the owning process, got %q", data)
	}
	unlock()
	// The lock can be taken again after it was released.
	unlock2, err := LockModel(p)
	if err != nil {
		t.Fatal(err)
	}
	unlock2()
}

func TestLockModel_RelativeAndAbsolutePathsShareOneLock(t *testing.T) {
	p := newLockTarget(t)
	old, _ := os.Getwd()
	if err := os.Chdir(filepath.Dir(p)); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(old) }()

	unlock, err := LockModel("architecture.jsonc")
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	if _, err := os.Stat(p + ".lock"); err != nil {
		t.Errorf("a relative path must lock the same file as the absolute one: %v", err)
	}
}

func TestLockModel_SecondCallWaitsForTheFirst(t *testing.T) {
	p := newLockTarget(t)
	first, err := LockModel(p)
	if err != nil {
		t.Fatal(err)
	}

	acquired := make(chan struct{})
	go func() {
		unlock, err := LockModel(p)
		if err != nil {
			t.Error(err)
			return
		}
		close(acquired)
		unlock()
	}()

	select {
	case <-acquired:
		t.Fatal("the second call must wait while the first holds the lock")
	case <-time.After(100 * time.Millisecond):
	}
	first()
	select {
	case <-acquired:
	case <-time.After(5 * time.Second):
		t.Fatal("the second call did not get the lock after it was released")
	}
}

// Many goroutines read, change and write a counter file under the lock; without
// mutual exclusion updates are lost.
func TestLockModel_ProvidesMutualExclusion(t *testing.T) {
	p := newLockTarget(t)
	counter := filepath.Join(filepath.Dir(p), "counter")
	if err := os.WriteFile(counter, []byte("0"), 0o600); err != nil {
		t.Fatal(err)
	}
	const n = 40
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			unlock, err := LockModel(p)
			if err != nil {
				t.Error(err)
				return
			}
			defer unlock()
			data, _ := os.ReadFile(counter)
			v := 0
			for _, c := range string(data) {
				v = v*10 + int(c-'0')
			}
			time.Sleep(time.Millisecond) // widen the window
			_ = os.WriteFile(counter, []byte(strings.TrimSpace(itoa(v+1))), 0o600)
		}()
	}
	wg.Wait()
	if data, _ := os.ReadFile(counter); string(data) != itoa(n) {
		t.Errorf("the counter is %s, want %d: updates were lost", data, n)
	}
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var b []byte
	for v > 0 {
		b = append([]byte{byte('0' + v%10)}, b...)
		v /= 10
	}
	return string(b)
}

// A lock file is not proof of a dead owner: a process that is suspended for
// longer than any age limit still holds the lock, and taking it over would let
// two commands change the model at once (review of #712).
func TestLockModel_AnOldLockOfALiveOwnerIsNotTakenOver(t *testing.T) {
	oldTimeout := lockTimeout
	lockTimeout = 200 * time.Millisecond
	defer func() { lockTimeout = oldTimeout }()

	p := newLockTarget(t)
	holder, err := LockModel(p)
	if err != nil {
		t.Fatal(err)
	}
	defer holder()
	old := time.Now().Add(-24 * time.Hour)
	if err := os.Chtimes(p+".lock", old, old); err != nil {
		t.Fatal(err)
	}

	if unlock, err := LockModel(p); err == nil {
		unlock()
		t.Fatal("a lock that is still held must not be taken over because its file is old")
	}
}

// When the owner dies, the lock is gone with it, whatever the file looks like.
// Releasing the lock without removing the file is what a dead process leaves.
func TestLockModel_ADeadOwnerDoesNotBlock(t *testing.T) {
	p := newLockTarget(t)
	unlock, err := LockModel(p)
	if err != nil {
		t.Fatal(err)
	}
	unlock()
	if err := os.WriteFile(p+".lock", []byte("99999\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-24 * time.Hour)
	if err := os.Chtimes(p+".lock", old, old); err != nil {
		t.Fatal(err)
	}

	again, err := LockModel(p)
	if err != nil {
		t.Fatalf("a leftover lock file without an owner must not block: %v", err)
	}
	again()
}

func TestLockModel_TimesOutWithAMessageThatNamesTheFile(t *testing.T) {
	oldTimeout := lockTimeout
	lockTimeout = 150 * time.Millisecond
	defer func() { lockTimeout = oldTimeout }()

	p := newLockTarget(t)
	holder, err := LockModel(p)
	if err != nil {
		t.Fatal(err)
	}
	defer holder()

	start := time.Now()
	_, err = LockModel(p)
	if err == nil {
		t.Fatal("a held lock must make the second call fail after the timeout")
	}
	if time.Since(start) < 100*time.Millisecond {
		t.Errorf("it gave up after %s, before the timeout", time.Since(start))
	}
	for _, want := range []string{p + ".lock", "held by process", "still working"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error must contain %q: %v", want, err)
		}
	}
}

func TestLockModel_ReportsErrorsThatAreNotContention(t *testing.T) {
	if _, err := LockModel(filepath.Join(t.TempDir(), "missing", "architecture.jsonc")); err == nil {
		t.Error("a model in a directory that does not exist cannot be locked")
	}
}

func TestLockOwner_WithoutAReadableFile(t *testing.T) {
	if got := lockOwner(filepath.Join(t.TempDir(), "none.lock")); !strings.Contains(got, "another process") {
		t.Errorf("lockOwner = %q", got)
	}
}

// A checkout can contain <model>.lock as a symbolic link. LockModel writes the
// process ID into the lock file, so following the link would replace the content
// of whatever file the link points at (review of #712).
func TestLockModel_RefusesASymbolicLinkAndLeavesItsTargetAlone(t *testing.T) {
	p := newLockTarget(t)
	victim := filepath.Join(filepath.Dir(p), "victim.txt")
	if err := os.WriteFile(victim, []byte("precious content\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(victim, p+".lock"); err != nil {
		t.Skipf("symbolic links are not available: %v", err)
	}

	unlock, err := LockModel(p)
	if err == nil {
		unlock()
		t.Fatal("a symbolic link at the lock path must be refused")
	}
	if data, _ := os.ReadFile(victim); string(data) != "precious content\n" {
		t.Errorf("the target of the link was modified: %q", data)
	}
	if !strings.Contains(err.Error(), "symbolic link") {
		t.Errorf("the error must say why: %v", err)
	}
}

func TestMustBeTheRegularFile(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a"), filepath.Join(dir, "b")
	for _, p := range []string{a, b} {
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	fa, err := os.Open(a)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = fa.Close() }()

	if err := mustBeTheRegularFile(fa, a); err != nil {
		t.Errorf("the file and its own path must pass: %v", err)
	}
	if err := mustBeTheRegularFile(fa, b); err == nil {
		t.Error("a path that names another file must be refused (swapped after the check)")
	}
	if err := mustBeTheRegularFile(fa, filepath.Join(dir, "missing")); err == nil {
		t.Error("a path that no longer exists must be refused")
	}
	d, err := os.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = d.Close() }()
	if err := mustBeTheRegularFile(d, dir); err == nil {
		t.Error("a directory is not a lock file")
	}
	closed, _ := os.Open(a)
	_ = closed.Close()
	if err := mustBeTheRegularFile(closed, a); err == nil {
		t.Error("a file that cannot be examined must be refused")
	}
}

func TestLockModel_ReportsAnErrorOfTheSystemLock(t *testing.T) {
	old := tryLock
	tryLock = func(*os.File) (bool, error) { return false, os.ErrPermission }
	defer func() { tryLock = old }()

	p := newLockTarget(t)
	if unlock, err := LockModel(p); err == nil {
		unlock()
		t.Fatal("an error of the file lock must be reported")
	}
}

func TestLockModel_RefusesAnythingButARegularFile(t *testing.T) {
	p := newLockTarget(t)
	if err := os.Mkdir(p+".lock", 0o700); err != nil {
		t.Fatal(err)
	}
	if unlock, err := LockModel(p); err == nil {
		unlock()
		t.Fatal("a directory at the lock path must be refused")
	}
}
