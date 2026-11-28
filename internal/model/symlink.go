package model

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// ResolveWriteTarget returns the file that a write to path must replace. A model,
// diagram or state file can be a symbolic link (monorepo, dotfiles); writing a
// temp file next to the link and renaming it over the link would replace the
// link by a regular file and leave its target untouched (#723), so atomic writes
// resolve the link first and replace the target. A path that is not a link, does
// not exist yet, or is a link to nothing is returned as it is. Any other failure
// (no permission to enter the directory of the target) is an error: falling back
// to the link would replace it and report success.
func ResolveWriteTarget(path string) (string, error) {
	resolved, err := filepath.EvalSymlinks(path)
	switch {
	case err == nil:
		return resolved, nil
	case errors.Is(err, fs.ErrNotExist):
		return path, nil
	default:
		return "", fmt.Errorf("resolving %s: %w", path, err)
	}
}

// writeTarget looks at the file that an atomic write is about to replace. A file
// that is not writable is refused (the replacement would ignore the read-only
// bit); for an existing file the mode is returned so that the replacement keeps
// it, exists is false for a file that is new (#724).
func writeTarget(path string) (mode os.FileMode, exists bool, err error) {
	info, err := os.Stat(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return 0, false, nil
	case err != nil:
		return 0, false, fmt.Errorf("checking %s: %w", path, err)
	}
	if info.Mode().Perm()&0o200 == 0 {
		return 0, true, fmt.Errorf("file is read-only: %s (make it writable to change it)", path)
	}
	return info.Mode().Perm(), true, nil
}

// keepMode gives the temporary file the mode of the file that it replaces.
func keepMode(tmp *os.File, mode os.FileMode, exists bool) error {
	if !exists {
		return nil
	}
	if err := tmp.Chmod(mode); err != nil {
		return fmt.Errorf("setting the mode of the temp file: %w", err)
	}
	return nil
}
