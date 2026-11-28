package model

import (
	"errors"
	"fmt"
	"io/fs"
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
