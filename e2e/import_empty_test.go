package e2e

import (
	"os"
	"strings"
	"testing"
)

// TestImportWithTheWrongFormatKeepsTheModel (#705): importing a file in another
// format wrote an empty model over the existing one and exited with 0.
func TestImportWithTheWrongFormatKeepsTheModel(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	existing := `{"specification":{"elements":{"system":{"notation":"System"}}},"model":{"keep":{"kind":"system","title":"Keep"}},"relationships":[],"views":{}}`
	writeFile(t, dir+"/architecture.jsonc", existing)
	writeFile(t, dir+"/g.txt", "garbage text\n")

	out, code := runCLIAllowFail(t, bin, dir, "import", "g.txt", "--from", "structurizr", "--force")
	if code != 1 || !strings.Contains(out, "no elements found") {
		t.Errorf("exit %d, want 1 with 'no elements found':\n%s", code, out)
	}
	if data, _ := os.ReadFile(dir + "/architecture.jsonc"); string(data) != existing {
		t.Errorf("the model must not be overwritten:\n%s", data)
	}
	runCLI(t, bin, dir, "validate")
}
