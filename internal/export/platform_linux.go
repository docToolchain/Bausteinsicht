//go:build linux

package export

// platformDrawioPaths returns platform-native draw.io install locations for Linux.
// On Linux, draw.io is typically on PATH already; this is a last-resort fallback.
func platformDrawioPaths() []string {
	return []string{
		"/opt/drawio/drawio",
		"/usr/lib/drawio/drawio",
	}
}
