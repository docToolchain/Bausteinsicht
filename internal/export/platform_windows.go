//go:build windows

package export

import "os"

// platformDrawioPaths returns platform-native draw.io install locations for Windows.
// Search order: Scoop → Chocolatey → Official Installer → Program Files
func platformDrawioPaths() []string {
	var paths []string

	// Scoop package manager (most common on Windows dev machines).
	// Scoop installs to C:\Users\<username>\scoop\apps\drawio\current\
	if scoop := os.Getenv("SCOOP"); scoop != "" {
		paths = append(paths, scoop+`\apps\drawio\current\draw.io.exe`)
		paths = append(paths, scoop+`\shims\draw.io.exe`)
	}

	// Chocolatey package manager.
	// Chocolatey installs to C:\ProgramData\chocolatey\bin\
	paths = append(paths, `C:\ProgramData\chocolatey\bin\draw.io.exe`)

	// Official installer (per-user install - most common).
	if localApp := os.Getenv("LOCALAPPDATA"); localApp != "" {
		paths = append(paths, localApp+`\Programs\draw.io\draw.io.exe`)
	}

	// System-wide install (Program Files).
	for _, prog := range []string{os.Getenv("PROGRAMFILES"), os.Getenv("PROGRAMFILES(X86)")} {
		if prog != "" {
			paths = append(paths, prog+`\draw.io\draw.io.exe`)
		}
	}

	return paths
}
