package drawio

import (
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/chaos"
)

// TestLoadCorruptXML tests loading corrupted draw.io files.
func TestLoadCorruptXML(t *testing.T) {
	tc := chaos.NewTestChaos(t)

	testCases := map[string]string{
		"not-xml": "this is not xml at all",
		"incomplete-tag": "<mxfile><diagram",
		"invalid-nesting": "<mxfile></diagram></mxfile>",
		"empty-file": "",
	}

	for name, content := range testCases {
		path := tc.CreateFileWithContent(name+".drawio", content)
		_, err := LoadDocument(path)
		if err == nil {
			t.Fatalf("Should reject invalid XML (%s): %s", name, content)
		}
	}
}

// TestLoadMissingDrawioFile tests loading non-existent draw.io file.
func TestLoadMissingDrawioFile(t *testing.T) {
	_, err := LoadDocument("/nonexistent/diagram.drawio")
	if err == nil {
		t.Fatal("Should error on missing file")
	}
}

// TestLoadEmptyDrawioFile tests loading empty draw.io file.
func TestLoadEmptyDrawioFile(t *testing.T) {
	tc := chaos.NewTestChaos(t)
	path := tc.CreateEmptyFile("empty.drawio")
	_, err := LoadDocument(path)
	if err == nil {
		t.Fatal("Should reject empty file")
	}
}

// TestLoadDrawioReadOnlyFile tests loading from read-only file.
func TestLoadDrawioReadOnlyFile(t *testing.T) {
	tc := chaos.NewTestChaos(t)
	validXML := `<?xml version="1.0"?>
<mxfile>
  <diagram name="Page-1">
    <mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/></root></mxGraphModel>
  </diagram>
</mxfile>`

	path := tc.CreateFileWithContent("readonly.drawio", validXML)
	tc.MakeReadOnly(path)
	defer tc.MakeWritable(path)

	// Should still be readable despite read-only flag
	doc, err := LoadDocument(path)
	if err != nil {
		t.Fatalf("Should read read-only file: %v", err)
	}
	if doc == nil {
		t.Fatal("Document should not be nil")
	}
}

// TestLoadValidDrawioFile tests loading valid draw.io file.
func TestLoadValidDrawioFile(t *testing.T) {
	tc := chaos.NewTestChaos(t)
	validXML := `<?xml version="1.0"?>
<mxfile>
  <diagram id="diagram1" name="Page-1">
    <mxGraphModel><root>
      <mxCell id="0"/>
      <mxCell id="1" parent="0"/>
      <mxCell id="elem1" parent="1" value="Element 1"/>
    </root></mxGraphModel>
  </diagram>
</mxfile>`

	path := tc.CreateFileWithContent("valid.drawio", validXML)
	doc, err := LoadDocument(path)
	if err != nil {
		t.Fatalf("Load valid document: %v", err)
	}

	if doc == nil {
		t.Fatal("Document should not be nil")
	}

	// Verify document structure
	pages := doc.Pages()
	if len(pages) == 0 {
		t.Fatal("Should have at least one page")
	}
}

// TestLoadDrawioValidXMLButNoVersion tests XML without version attribute.
func TestLoadDrawioValidXMLButNoVersion(t *testing.T) {
	tc := chaos.NewTestChaos(t)
	xmlNoVersion := `<?xml version="1.0"?>
<mxfile>
  <diagram name="Page-1">
    <mxGraphModel><root><mxCell id="0"/></root></mxGraphModel>
  </diagram>
</mxfile>`

	path := tc.CreateFileWithContent("noversion.drawio", xmlNoVersion)
	_, err := LoadDocument(path)
	// Should either work (graceful) or fail with clear error
	if err != nil {
		t.Logf("Missing version handled: %v", err)
	}
}
