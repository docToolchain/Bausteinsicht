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
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600) // #nosec G304 -- derived from the model path
	if err != nil {
		return nil, fmt.Errorf("locking model: %w", err)
	}

	deadline := time.Now().Add(lockTimeout)
	for {
		locked, err := tryLockFile(f)
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

// lockOwner describes the holder of a lock file for the error message.
func lockOwner(lockPath string) string {
	data, err := os.ReadFile(lockPath) // #nosec G304 -- derived from the model path
	if err != nil || strings.TrimSpace(string(data)) == "" {
		return "held by another process"
	}
	return "held by process " + strings.TrimSpace(string(data))
}
