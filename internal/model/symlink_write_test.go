package model

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// linkedFile creates target with content and a symbolic link to it in another
// directory, the way a monorepo or a dotfile setup uses a model; it skips the
// test where symbolic links are not available.
func linkedFile(t *testing.T, content string) (link, target string) {
	t.Helper()
	root := t.TempDir()
	target = filepath.Join(root, "real", "architecture.jsonc")
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	link = filepath.Join(root, "link", "architecture.jsonc")
	if err := os.MkdirAll(filepath.Dir(link), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symbolic links are not available: %v", err)
	}
	return link, target
}

func assertStillALink(t *testing.T, link, target, wantInTarget string) {
	t.Helper()
	info, err := os.Lstat(link)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("the symbolic link was replaced by a regular file (err %v)", err)
	}
	data, err := os.ReadFile(target)
	if err != nil || !strings.Contains(string(data), wantInTarget) {
		t.Errorf("the target of the link was not updated (err %v):\n%s", err, data)
	}
}

// Writing through a symbolic link must update the file it points to and keep the
// link; the temp file + rename pattern replaced the link itself (#723).
func TestSave_KeepsASymbolicLink(t *testing.T) {
	link, target := linkedFile(t, `{"specification":{"elements":{"a":{"notation":"A"}}},"model":{"x":{"kind":"a","title":"X"}}}`)
	m, err := Load(link)
	if err != nil {
		t.Fatal(err)
	}
	m.Model["y"] = Element{Kind: "a", Title: "Y"}
	if err := Save(link, m); err != nil {
		t.Fatal(err)
	}
	assertStillALink(t, link, target, `"y"`)
}

func TestPatchSave_KeepsASymbolicLink(t *testing.T) {
	link, target := linkedFile(t, "{\n  \"model\": { \"x\": { \"title\": \"X\" } }\n}\n")
	if err := PatchSave(link, []PatchOp{{Path: []string{"model", "x", "title"}, Value: `"Z"`}}); err != nil {
		t.Fatal(err)
	}
	assertStillALink(t, link, target, `"Z"`)
}

func TestPatchInsert_KeepsASymbolicLink(t *testing.T) {
	link, target := linkedFile(t, "{\n  \"model\": { \"x\": { \"title\": \"X\" } }\n}\n")
	err := PatchInsert(link, func(data []byte) ([]byte, error) {
		return InsertObjectEntry(data, []string{"model"}, "y", `{ "title": "Y" }`)
	})
	if err != nil {
		t.Fatal(err)
	}
	assertStillALink(t, link, target, `"y"`)
}

func TestResolveWriteTarget(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real.jsonc")
	if err := os.WriteFile(real, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	canonical, err := filepath.EvalSymlinks(real)
	if err != nil {
		t.Fatal(err)
	}

	if got, err := ResolveWriteTarget(real); err != nil || got != canonical {
		t.Errorf("a regular file resolves to itself: %q, %v, want %q", got, err, canonical)
	}
	missing := filepath.Join(dir, "new.jsonc")
	if got, err := ResolveWriteTarget(missing); err != nil || got != missing {
		t.Errorf("a file that does not exist yet is returned as it is: %q, %v", got, err)
	}

	one := filepath.Join(dir, "one")
	two := filepath.Join(dir, "two")
	if err := os.Symlink(real, one); err != nil {
		t.Skipf("symbolic links are not available: %v", err)
	}
	if err := os.Symlink(one, two); err != nil {
		t.Fatal(err)
	}
	if got, err := ResolveWriteTarget(two); err != nil || got != canonical {
		t.Errorf("a chain of links resolves to the final file: %q, %v, want %q", got, err, canonical)
	}

	dangling := filepath.Join(dir, "dangling")
	if err := os.Symlink(filepath.Join(dir, "gone"), dangling); err != nil {
		t.Fatal(err)
	}
	if got, err := ResolveWriteTarget(dangling); err != nil || got != dangling {
		t.Errorf("a link to nothing is returned as it is: %q, %v", got, err)
	}
}

// If the link cannot be resolved for another reason than "does not exist" (no
// permission to enter the directory of the target), the write must fail instead
// of replacing the link by a regular file and reporting success (review of #723).
func TestSave_FailsWhenTheLinkCannotBeResolved(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permissions do not restrict root")
	}
	link, target := linkedFile(t, `{"specification":{"elements":{"a":{"notation":"A"}}},"model":{"x":{"kind":"a","title":"X"}}}`)
	m, err := Load(link)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Dir(target)
	if err := os.Chmod(dir, 0o000); err != nil {
		t.Skipf("cannot restrict the directory: %v", err)
	}
	defer func() { _ = os.Chmod(dir, 0o700) }()

	m.Model["y"] = Element{Kind: "a", Title: "Y"}
	if err := Save(link, m); err == nil {
		t.Error("Save must fail when the link cannot be resolved")
	}
	_ = os.Chmod(dir, 0o700)
	if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("the symbolic link was replaced by a regular file (err %v)", err)
	}
	if data, _ := os.ReadFile(target); strings.Contains(string(data), `"y"`) {
		t.Error("the target must not have been changed")
	}
}
