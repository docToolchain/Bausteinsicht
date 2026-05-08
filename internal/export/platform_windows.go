//go:build windows

package export

import (
	"os"
	"os/user"
	"path/filepath"
)

// platformDrawioPaths returns platform-native draw.io install locations for Windows.
// Search order: Scoop → Chocolatey → Official Installer → Program Files
func platformDrawioPaths() []string {
	var paths []string

	// Scoop package manager (most common on Windows dev machines).
	// First, try SCOOP env var if set (allows override).
	// If not set, try default: C:\Users\<username>\scoop\
	if scoop := os.Getenv("SCOOP"); scoop != "" {
		paths = append(paths, filepath.Join(scoop, "apps", "drawio", "current", "draw.io.exe"))
		paths = append(paths, filepath.Join(scoop, "shims", "draw.io.exe"))
	} else {
		// Fallback: try default Scoop location if user home dir is available
		if currentUser, err := user.Current(); err == nil {
			scoopHome := filepath.Join(currentUser.HomeDir, "scoop")
			paths = append(paths, filepath.Join(scoopHome, "apps", "drawio", "current", "draw.io.exe"))
			paths = append(paths, filepath.Join(scoopHome, "shims", "draw.io.exe"))
		}
	}

	// Chocolatey package manager (fixed installation path).
	paths = append(paths, filepath.Join("C:\\ProgramData\\chocolatey\\bin", "draw.io.exe"))

	// Official installer (per-user install - LOCALAPPDATA).
	if localApp := os.Getenv("LOCALAPPDATA"); localApp != "" {
		paths = append(paths, filepath.Join(localApp, "Programs", "draw.io", "draw.io.exe"))
	}

	// System-wide install (Program Files).
	for _, prog := range []string{os.Getenv("PROGRAMFILES"), os.Getenv("PROGRAMFILES(X86)")} {
		if prog != "" {
			paths = append(paths, filepath.Join(prog, "draw.io", "draw.io.exe"))
		}
	}

	return paths
}
