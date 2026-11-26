package model

import "path/filepath"

// ResolveWriteTarget returns the file that a write to path must replace. A model,
// diagram or state file can be a symbolic link (monorepo, dotfiles); writing a
// temp file next to the link and renaming it over the link would replace the
// link by a regular file and leave its target untouched (#723), so atomic writes
// resolve the link first and replace the target. A path that is not a link, or
// does not exist yet, is returned as it is.
func ResolveWriteTarget(path string) string {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved
	}
	return path
}
