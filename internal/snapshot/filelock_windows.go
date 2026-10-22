//go:build windows

package snapshot

import (
	"os"

	"golang.org/x/sys/windows"
)

// lockFileExclusive takes a blocking, exclusive lock on the open file. Windows
// releases the lock automatically when the handle is closed or the process
// exits, so the lock cannot go stale.
func lockFileExclusive(f *os.File) error {
	return windows.LockFileEx(
		windows.Handle(f.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK,
		0,
		1,
		0,
		new(windows.Overlapped),
	)
}
