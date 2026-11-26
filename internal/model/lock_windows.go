//go:build windows

package model

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

// tryLockFile takes an exclusive LockFileEx lock on f without waiting and
// reports whether it got it. The locked byte lies far behind the process ID
// that the owner writes at the start of the file, so a waiting process can still
// read the ID (Windows locks are mandatory for the locked range).
func tryLockFile(f *os.File) (bool, error) {
	ol := &windows.Overlapped{Offset: 0x80000000}
	err := windows.LockFileEx(windows.Handle(f.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, ol)
	if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
		return false, nil
	}
	return err == nil, err
}
