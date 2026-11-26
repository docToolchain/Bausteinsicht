package model

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// lockSuffix is appended to the model path to name the lock file.
const lockSuffix = ".lock"

var (
	// lockTimeout is how long LockModel waits for another process to finish.
	// Commands that change the model take milliseconds, so a longer wait means a
	// leftover lock or a hung process. (A variable so that tests can shorten it.)
	lockTimeout = 30 * time.Second
	// lockStaleAfter is the age after which a lock file is taken to belong to a
	// process that died without removing it.
	lockStaleAfter = 2 * time.Minute
)

// LockModel takes an exclusive advisory lock for the model file at path and
// returns the function that releases it. A command that reads the model, changes
// it and writes it back must hold the lock from before the read until after the
// write; without it, two commands running at the same time overwrite each
// other's change and both report success (#712).
//
// The lock is the file path+".lock", created exclusively (O_EXCL), which works on
// every platform and file system the model can live on. It is not reentrant: a
// process that holds the lock must not ask for it again. A lock file older than
// two minutes is taken over, because its owner died; waiting for a live owner
// gives up after 30 seconds with an error that names the file.
func LockModel(path string) (unlock func(), err error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("locking model: %w", err)
	}
	lockPath := abs + lockSuffix
	deadline := time.Now().Add(lockTimeout)

	for {
		f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			_, _ = fmt.Fprintf(f, "%d\n", os.Getpid())
			if err := f.Close(); err != nil {
				_ = os.Remove(lockPath)
				return nil, fmt.Errorf("locking model: %w", err)
			}
			return func() { _ = os.Remove(lockPath) }, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("locking model: %w", err)
		}
		if takeOverStaleLock(lockPath) {
			continue
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timed out after %s waiting for the lock %s (%s); if no bausteinsicht process is running, delete the file",
				lockTimeout, lockPath, lockOwner(lockPath))
		}
		time.Sleep(time.Duration(5+rand.Intn(20)) * time.Millisecond) //nolint:gosec // jitter, not security relevant
	}
}

// takeOverStaleLock removes the lock file if it is older than lockStaleAfter
// and reports whether it did. The file is renamed first: of several processes
// that find the same stale lock, only one rename succeeds.
func takeOverStaleLock(lockPath string) bool {
	info, err := os.Stat(lockPath)
	if err != nil || time.Since(info.ModTime()) < lockStaleAfter {
		return errors.Is(err, os.ErrNotExist) // gone meanwhile: try to create it again
	}
	stale := lockPath + ".stale-" + strconv.Itoa(os.Getpid())
	if err := os.Rename(lockPath, stale); err != nil {
		return false
	}
	_ = os.Remove(stale)
	return true
}

// lockOwner describes the holder of a lock file for the error message.
func lockOwner(lockPath string) string {
	data, err := os.ReadFile(lockPath)
	if err != nil {
		return "held by another process"
	}
	return "held by process " + strings.TrimSpace(string(data))
}
