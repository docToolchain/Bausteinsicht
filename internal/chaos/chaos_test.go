package chaos

import (
	"os"
	"runtime"
	"testing"
)

// TestMakeReadOnly_NotWorldAccessible is a regression test for SonarCloud
// go:S2612 ("File permissions should not be set to world-accessible
// values"): MakeReadOnly used to chmod to 0444, which — while it does block
// the owner from writing — also grants read access to group and other. The
// chaos helper only needs to affect the owning test process, so group/other
// bits must be cleared entirely (#585).
//
// The group/other-bits assertions are POSIX-only: os.Chmod on Windows only
// ever toggles the read-only attribute via the 0200 bit, and os.Stat
// synthesizes a FileMode that mirrors that single attribute across all
// three permission classes (0444 read-only / 0666 writable) — there is no
// Windows equivalent of "owner-only" access to assert on.
func TestMakeReadOnly_NotWorldAccessible(t *testing.T) {
	tc := NewTestChaos(t)
	path := tc.CreateEmptyFile("readonly.txt")

	tc.MakeReadOnly(path)

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	perm := info.Mode().Perm()
	if runtime.GOOS != "windows" {
		if perm&0077 != 0 {
			t.Errorf("MakeReadOnly left group/other-accessible bits set: mode=%#o", perm)
		}
		if perm&0200 != 0 {
			t.Errorf("MakeReadOnly left the file owner-writable: mode=%#o", perm)
		}
	}

	// The permission change must still do its actual job: writes fail.
	if err := os.WriteFile(path, []byte("x"), 0644); err == nil {
		t.Error("expected write to a read-only file to fail")
	}
}

// TestMakeWritable_NotWorldAccessible is the MakeWritable counterpart of
// TestMakeReadOnly_NotWorldAccessible (#585, go:S2612). See that test's
// comment for why the group/other-bits assertions are POSIX-only.
func TestMakeWritable_NotWorldAccessible(t *testing.T) {
	tc := NewTestChaos(t)
	path := tc.CreateEmptyFile("writable.txt")
	tc.MakeReadOnly(path)

	tc.MakeWritable(path)

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	perm := info.Mode().Perm()
	if runtime.GOOS != "windows" {
		if perm&0077 != 0 {
			t.Errorf("MakeWritable left group/other-accessible bits set: mode=%#o", perm)
		}
		if perm&0200 == 0 {
			t.Errorf("MakeWritable left the file owner-unwritable: mode=%#o", perm)
		}
	}

	// The permission change must still do its actual job: writes succeed.
	if err := os.WriteFile(path, []byte("x"), 0644); err != nil {
		t.Errorf("expected write to a writable file to succeed, got: %v", err)
	}
}
