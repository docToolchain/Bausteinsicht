//go:build !windows

package model

import (
	"errors"
	"os"
	"syscall"
)

// tryLockFile takes an exclusive flock on f without waiting and reports whether
// it got it.
func tryLockFile(f *os.File) (bool, error) {
	err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) // #nosec G115 -- file descriptor
	if errors.Is(err, syscall.EWOULDBLOCK) {
		return false, nil
	}
	return err == nil, err
}
