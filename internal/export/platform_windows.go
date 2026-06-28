//go:build windows

package export

import (
	"os"
	"os/user"
	"path/filepath"
)

const drawioExe = "draw.io.exe"
const drawioAppDir = "draw.io"

// platformDrawioPaths returns platform-native draw.io install locations for Windows.
// Search order: Scoop → Chocolatey → Official Installer → Program Files
func platformDrawioPaths() []string {
	var paths []string

	// Scoop package manager (most common on Windows dev machines).
	// First, try SCOOP env var if set (allows override).
	// If not set, try default: C:\Users\<username>\scoop\
	if scoop := os.Getenv("SCOOP"); scoop != "" {
		// Scoop uses "draw.io" (with dot) as the app directory; keep "drawio" as fallback.
		paths = append(paths, filepath.Join(scoop, "apps", drawioAppDir, "current", drawioExe))
		paths = append(paths, filepath.Join(scoop, "apps", "drawio", "current", drawioExe))
		paths = append(paths, filepath.Join(scoop, "shims", drawioExe))
	} else {
		// Fallback: try default Scoop location if user home dir is available
		if currentUser, err := user.Current(); err == nil {
			scoopHome := filepath.Join(currentUser.HomeDir, "scoop")
			paths = append(paths, filepath.Join(scoopHome, "apps", drawioAppDir, "current", drawioExe))
			paths = append(paths, filepath.Join(scoopHome, "apps", "drawio", "current", drawioExe))
			paths = append(paths, filepath.Join(scoopHome, "shims", drawioExe))
		}
	}

	// Chocolatey package manager (uses %PROGRAMDATA%, defaults to C:\ProgramData).
	progData := os.Getenv("PROGRAMDATA")
	if progData == "" {
		progData = `C:\ProgramData`
	}
	paths = append(paths, filepath.Join(progData, "chocolatey", "bin", drawioExe))

	// Official installer (per-user install - LOCALAPPDATA).
	if localApp := os.Getenv("LOCALAPPDATA"); localApp != "" {
		paths = append(paths, filepath.Join(localApp, "Programs", drawioAppDir, drawioExe))
	}

	// System-wide install (Program Files).
	for _, prog := range []string{os.Getenv("PROGRAMFILES"), os.Getenv("PROGRAMFILES(X86)")} {
		if prog != "" {
			paths = append(paths, filepath.Join(prog, drawioAppDir, drawioExe))
		}
	}

	return paths
}
