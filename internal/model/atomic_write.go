package model

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// WriteFileAtomic replaces the file at path with data: a temporary file next to
// it (named after tmpPattern, see os.CreateTemp) is written and renamed over it,
// so a reader never sees half a file. path is written as it is, a symbolic link
// has to be resolved first (ResolveWriteTarget). A file that is read-only is not
// replaced (`file is read-only`), and the replacement gets the mode of the file it
// replaces; a new file keeps the mode that CreateTemp gives it (#724).
func WriteFileAtomic(path, tmpPattern string, data []byte) error {
	mode, exists, err := writeTarget(path)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), tmpPattern)
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpName := tmp.Name()
	if err := fillTemp(tmp, mode, exists, data); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("renaming temp file: %w", err)
	}
	return nil
}

// fillTemp gives the temporary file the mode of the file it will replace (if
// there is one), writes data to it and closes it.
func fillTemp(tmp *os.File, mode os.FileMode, replaces bool, data []byte) error {
	if replaces {
		if err := tmp.Chmod(mode); err != nil {
			_ = tmp.Close()
			return fmt.Errorf("setting the mode of the temp file: %w", err)
		}
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}
	return nil
}

// writeTarget looks at the file that an atomic write is about to replace. A file
// that is not writable is refused (the replacement would ignore the read-only
// bit); for an existing file the mode is returned so that the replacement keeps
// it, exists is false for a file that is new.
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
