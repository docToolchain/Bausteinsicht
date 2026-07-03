package e2eplan

import (
	"regexp"
	"testing"
)

var planIDPattern = regexp.MustCompile(`^\d+[a-z]?\.\d+$`)

func TestRegistry_KeysAreValidPlanIDs(t *testing.T) {
	for id := range Registry {
		if !planIDPattern.MatchString(id) {
			t.Errorf("registry key %q does not look like a plan ID (e.g. \"5.1\", \"8b.4\")", id)
		}
	}
}

func TestRegistry_ValuesAreQualifiedTestNames(t *testing.T) {
	testFuncPattern := regexp.MustCompile(`^Test[A-Za-z0-9_]+(/[A-Za-z0-9_.]+)?$`)
	for id, names := range Registry {
		if len(names) == 0 {
			t.Errorf("plan ID %q has an empty test-name list", id)
		}
		for _, name := range names {
			if !testFuncPattern.MatchString(name) {
				t.Errorf("plan ID %q: %q doesn't look like a qualified go-test name (\"TestFunc\" or \"TestFunc/Subtest\")", id, name)
			}
		}
	}
}

func TestRegistry_NotEmpty(t *testing.T) {
	if len(Registry) == 0 {
		t.Error("Registry is empty — expected exception entries for the pre-convention #519 test files")
	}
}
