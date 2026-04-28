//go:build darwin

package export

import "os"

// platformDrawioPaths returns platform-native draw.io install locations for macOS.
func platformDrawioPaths() []string {
	paths := []string{
		"/Applications/draw.io.app/Contents/MacOS/draw.io",
	}
	// Also check user-level install.
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, home+"/Applications/draw.io.app/Contents/MacOS/draw.io")
	}
	return paths
}
