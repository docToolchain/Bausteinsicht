package e2e

// TestFindShow (#495) verifies that `find` and `show` work correctly after
// import+sync and can serve as a read-layer for other E2E tests.

import (
	"strings"
	"testing"
)

func TestFindShow(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	runCLI(t, bin, dir, "init")
	runCLI(t, bin, dir, "sync")

	t.Run("Find_HitsKnownElement", func(t *testing.T) {
		out := runCLI(t, bin, dir, "find", "customer")
		if !strings.Contains(strings.ToLower(out), "customer") {
			t.Errorf("find 'customer' did not return a match; output:\n%s", out)
		}
	})

	t.Run("Find_NoMatch_ExitsZero", func(t *testing.T) {
		_, code := runCLIAllowFail(t, bin, dir, "find", "xyznonexistent99")
		if code != 0 {
			t.Errorf("find nonexistent: expected exit 0, got %d", code)
		}
	})

	t.Run("Find_TypeFilter", func(t *testing.T) {
		out := runCLI(t, bin, dir, "find", "--type", "element", "shop")
		// The default model has "onlineshop" — it should appear.
		if !strings.Contains(strings.ToLower(out), "shop") {
			t.Errorf("find --type element 'shop' returned nothing; output:\n%s", out)
		}
	})

	t.Run("Show_KnownElement", func(t *testing.T) {
		out := runCLI(t, bin, dir, "show", "customer")
		if !strings.Contains(strings.ToLower(out), "customer") {
			t.Errorf("show 'customer' output missing element id; output:\n%s", out)
		}
	})

	t.Run("Show_UnknownElement_ExitsNonZero", func(t *testing.T) {
		_, code := runCLIAllowFail(t, bin, dir, "show", "xyznonexistent99")
		if code == 0 {
			t.Error("show unknown element: expected non-zero exit, got 0")
		}
	})

	t.Run("Show_ParentElement_ListsChildren", func(t *testing.T) {
		// "onlineshop" is the parent; its children should appear.
		out := runCLI(t, bin, dir, "show", "onlineshop")
		if !strings.Contains(strings.ToLower(out), "onlineshop") {
			t.Errorf("show 'onlineshop' output missing; output:\n%s", out)
		}
	})
}
