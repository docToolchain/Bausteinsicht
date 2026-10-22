//go:build !windows

package snapshot

import (
	"os"

	"golang.org/x/sys/unix"
)

// lockFileExclusive takes a blocking, exclusive advisory lock on the open file.
// The lock is tied to the open file description, so closing the file (or the
// process exiting) releases it automatically — it cannot go stale.
func lockFileExclusive(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_EX)
}
