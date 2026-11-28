package model

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const readonlyModel = `{"specification":{"elements":{"system":{"notation":"S"}}},"model":{"a":{"kind":"system","title":"A"}}}`

func writeModelFile(t *testing.T, mode os.FileMode) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "m.jsonc")
	if err := os.WriteFile(path, []byte(readonlyModel), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
	return path
}

// A model file that is read-only is not replaced: every way of writing it says so
// and leaves it as it is (#724).
func TestWrite_RefusesAReadOnlyModelFile(t *testing.T) {
	writers := map[string]func(path string) error{
		"Save": func(path string) error {
			m, err := Load(path)
			if err != nil {
				return err
			}
			return Save(path, m)
		},
		"PatchSave": func(path string) error {
			return PatchSave(path, []PatchOp{{Path: []string{"model", "a", "title"}, Value: `"B"`}})
		},
		"PatchInsert": func(path string) error {
			return PatchInsert(path, func(b []byte) ([]byte, error) { return append(b, ' '), nil })
		},
	}
	for name, write := range writers {
		path := writeModelFile(t, 0o444)
		err := write(path)
		if err == nil || !strings.Contains(err.Error(), "read-only") || !strings.Contains(err.Error(), path) {
			t.Errorf("%s: want an error \"file is read-only\" naming the file, got %v", name, err)
		}
		if data, _ := os.ReadFile(path); string(data) != readonlyModel {
			t.Errorf("%s: the read-only file was changed:\n%s", name, data)
		}
		if info, _ := os.Stat(path); info.Mode().Perm() != 0o444 {
			t.Errorf("%s: the mode changed to %v", name, info.Mode().Perm())
		}
	}
}

// The file that replaces the model keeps the mode of the one it replaces (#724).
func TestWrite_KeepsTheModeOfTheModelFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows has no permission bits beyond read-only")
	}
	for name, write := range map[string]func(path string) error{
		"Save": func(path string) error {
			m, err := Load(path)
			if err != nil {
				return err
			}
			return Save(path, m)
		},
		"PatchSave": func(path string) error {
			return PatchSave(path, []PatchOp{{Path: []string{"model", "a", "title"}, Value: `"B"`}})
		},
		"PatchInsert": func(path string) error {
			return PatchInsert(path, func(b []byte) ([]byte, error) { return append(b, ' '), nil })
		},
	} {
		path := writeModelFile(t, 0o640)
		if err := write(path); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if info, _ := os.Stat(path); info.Mode().Perm() != 0o640 {
			t.Errorf("%s: want mode 0640 to be kept, got %v", name, info.Mode().Perm())
		}
	}
}
