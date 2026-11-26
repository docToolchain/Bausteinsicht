package model

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// tryLock is tryLockFile; a variable so that a test can make it fail.
var tryLock = tryLockFile

// lockSuffix is appended to the model path to name the lock file.
const lockSuffix = ".lock"

// lockTimeout is how long LockModel waits for another process to finish.
// Commands that change the model take milliseconds, so a longer wait means a
// hung process. (A variable so that tests can shorten it.)
var lockTimeout = 30 * time.Second

// LockModel takes an exclusive advisory lock for the model file at path and
// returns the function that releases it. A command that reads the model, changes
// it and writes it back must hold the lock from before the read until after the
// write; without it, two commands running at the same time overwrite each
// other's change and both report success (#712).
//
// The lock is an operating-system file lock (flock, LockFileEx) on the file
// path+".lock". The system releases it when the owner dies, however it dies,
// and keeps it while the owner is suspended, so neither a crash nor a long pause
// needs a time limit or a takeover. The lock file itself stays after the unlock
// (removing it would let a waiting process lock a file that no longer has the
// name); it is empty apart from the process ID of the last owner and can be
// ignored in version control. It is not reentrant: a process that holds the lock
// must not ask for it again. Waiting for a live owner gives up after 30 seconds.
func LockModel(path string) (unlock func(), err error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("locking model: %w", err)
	}
	lockPath := abs + lockSuffix
	// A symbolic link at the lock path would make the PID written below replace
	// the content of its target; refuse it before opening.
	if info, err := os.Lstat(lockPath); err == nil && !info.Mode().IsRegular() {
		return nil, fmt.Errorf("locking model: %s is a symbolic link or not a regular file; remove it", lockPath)
	}
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600) // #nosec G304 -- derived from the model path
	if err != nil {
		return nil, fmt.Errorf("locking model: %w", err)
	}
	// The path could have been swapped between the check and the open: the opened
	// file must be the regular file the path names.
	if err := mustBeTheRegularFile(f, lockPath); err != nil {
		_ = f.Close()
		return nil, err
	}

	deadline := time.Now().Add(lockTimeout)
	for {
		locked, err := tryLock(f)
		if err != nil {
			_ = f.Close()
			return nil, fmt.Errorf("locking model: %w", err)
		}
		if locked {
			break
		}
		if time.Now().After(deadline) {
			_ = f.Close()
			return nil, fmt.Errorf("timed out after %s waiting for the lock %s (%s); another bausteinsicht command is still working on the model",
				lockTimeout, lockPath, lockOwner(lockPath))
		}
		time.Sleep(time.Duration(5+rand.Intn(20)) * time.Millisecond) //nolint:gosec // jitter, not security relevant
	}

	// Best effort: name the owner for the message of a waiting command.
	if err := f.Truncate(0); err == nil {
		_, _ = f.WriteAt([]byte(strconv.Itoa(os.Getpid())+"\n"), 0)
	}
	return func() { _ = f.Close() }, nil // closing the file releases the lock
}

// mustBeTheRegularFile checks that f is a regular file and the one lockPath
// names now (not a symbolic link that was put there after the first check).
func mustBeTheRegularFile(f *os.File, lockPath string) error {
	opened, err := f.Stat()
	if err != nil {
		return fmt.Errorf("locking model: %w", err)
	}
	named, err := os.Lstat(lockPath)
	if err != nil || !opened.Mode().IsRegular() || !named.Mode().IsRegular() || !os.SameFile(opened, named) {
		return fmt.Errorf("locking model: %s is a symbolic link or not a regular file; remove it", lockPath)
	}
	return nil
}

// lockOwner describes the holder of a lock file for the error message.
func lockOwner(lockPath string) string {
	data, err := os.ReadFile(lockPath) // #nosec G304 -- derived from the model path
	if err != nil || strings.TrimSpace(string(data)) == "" {
		return "held by another process"
	}
	return "held by process " + strings.TrimSpace(string(data))
}
