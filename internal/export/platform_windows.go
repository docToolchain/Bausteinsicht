//go:build windows

package export

import "os"

// platformDrawioPaths returns platform-native draw.io install locations for Windows.
func platformDrawioPaths() []string {
	var paths []string
	// Per-user install (most common via the official installer).
	if localApp := os.Getenv("LOCALAPPDATA"); localApp != "" {
		paths = append(paths, localApp+`\Programs\draw.io\draw.io.exe`)
	}
	// System-wide install.
	for _, prog := range []string{os.Getenv("PROGRAMFILES"), os.Getenv("PROGRAMFILES(X86)")} {
		if prog != "" {
			paths = append(paths, prog+`\draw.io\draw.io.exe`)
		}
	}
	return paths
}
