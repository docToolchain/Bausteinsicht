//go:build darwin

package export

import "os"

// platformDrawioPaths returns platform-native draw.io install locations for macOS.
// Search order: Homebrew (Apple Silicon) → Homebrew (Intel) → App Store → Manual Install
func platformDrawioPaths() []string {
	var paths []string

	// Homebrew package manager (Apple Silicon - M1/M2/M3).
	// Homebrew installs to /opt/homebrew/ on Apple Silicon
	paths = append(paths,
		"/opt/homebrew/bin/draw.io",
		"/opt/homebrew/opt/drawio/bin/draw.io",
	)

	// Homebrew package manager (Intel).
	// Homebrew installs to /usr/local/ on Intel Macs
	paths = append(paths,
		"/usr/local/bin/draw.io",
		"/usr/local/opt/drawio/bin/draw.io",
	)

	// Official installer or App Store install (system-wide).
	paths = append(paths, "/Applications/draw.io.app/Contents/MacOS/draw.io")

	// User-level install (~/Applications).
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, home+"/Applications/draw.io.app/Contents/MacOS/draw.io")
	}

	return paths
}
