package drawio_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/docToolchain/Bauteinsicht/internal/drawio"
)

func TestLoadDocument_Simple(t *testing.T) {
	doc, err := drawio.LoadDocument("testdata/simple-diagram.drawio")
	if err != nil {
		t.Fatalf("LoadDocument: %v", err)
	}
	pages := doc.Pages()
	if len(pages) != 1 {
		t.Errorf("expected 1 page, got %d", len(pages))
	}
}

func TestLoadDocument_MultiPage(t *testing.T) {
	doc, err := drawio.LoadDocument("testdata/multi-page.drawio")
	if err != nil {
		t.Fatalf("LoadDocument: %v", err)
	}
	pages := doc.Pages()
	if len(pages) != 2 {
		t.Errorf("expected 2 pages, got %d", len(pages))
	}
}

func TestGetPage(t *testing.T) {
	doc, err := drawio.LoadDocument("testdata/multi-page.drawio")
	if err != nil {
		t.Fatalf("LoadDocument: %v", err)
	}
	p := doc.GetPage("page2")
	if p == nil {
		t.Fatal("expected page2, got nil")
	}
	if doc.GetPage("nonexistent") != nil {
		t.Error("expected nil for nonexistent page")
	}
}

func TestSaveDocument_RoundTrip(t *testing.T) {
	doc, err := drawio.LoadDocument("testdata/simple-diagram.drawio")
	if err != nil {
		t.Fatalf("LoadDocument: %v", err)
	}

	tmp := filepath.Join(t.TempDir(), "out.drawio")
	if err := drawio.SaveDocument(tmp, doc); err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}

	doc2, err := drawio.LoadDocument(tmp)
	if err != nil {
		t.Fatalf("LoadDocument after save: %v", err)
	}
	if len(doc2.Pages()) != len(doc.Pages()) {
		t.Errorf("page count mismatch after round-trip: got %d, want %d", len(doc2.Pages()), len(doc.Pages()))
	}
}

func TestNewDocument(t *testing.T) {
	doc := drawio.NewDocument()
	if doc == nil {
		t.Fatal("NewDocument returned nil")
	}
	if len(doc.Pages()) != 0 {
		t.Errorf("expected 0 pages, got %d", len(doc.Pages()))
	}
}

func TestAddPage(t *testing.T) {
	doc := drawio.NewDocument()
	p := doc.AddPage("mypage", "My Page")
	if p == nil {
		t.Fatal("AddPage returned nil")
	}
	if len(doc.Pages()) != 1 {
		t.Errorf("expected 1 page after AddPage, got %d", len(doc.Pages()))
	}
	if doc.GetPage("mypage") == nil {
		t.Error("GetPage could not find the added page")
	}
}

func TestAddPage_BaseCells(t *testing.T) {
	doc := drawio.NewDocument()
	p := doc.AddPage("p1", "Page 1")
	root := p.Root()
	if root == nil {
		t.Fatal("Root() returned nil")
	}
	cells := root.SelectElements("mxCell")
	if len(cells) < 2 {
		t.Errorf("expected at least 2 base mxCells, got %d", len(cells))
	}
	if cells[0].SelectAttrValue("id", "") != "0" {
		t.Errorf("first base cell id should be '0', got '%s'", cells[0].SelectAttrValue("id", ""))
	}
	if cells[1].SelectAttrValue("id", "") != "1" {
		t.Errorf("second base cell id should be '1', got '%s'", cells[1].SelectAttrValue("id", ""))
	}
}

func TestLoadDocument_RejectsCorruptContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.drawio")
	if err := os.WriteFile(path, []byte("THIS IS NOT XML"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := drawio.LoadDocument(path)
	if err == nil {
		t.Fatal("expected error for corrupt drawio content")
	}
}

func TestLoadDocument_RejectsEmptyMxfile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.drawio")
	if err := os.WriteFile(path, []byte(`<?xml version="1.0"?><mxfile></mxfile>`), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := drawio.LoadDocument(path)
	if err == nil {
		t.Fatal("expected error for mxfile with no diagrams")
	}
}

func TestLoadDocument_RejectsMissingRoot(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.drawio")
	if err := os.WriteFile(path, []byte(`<?xml version="1.0"?><mxfile><diagram id="d1" name="Page"><mxGraphModel></mxGraphModel></diagram></mxfile>`), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := drawio.LoadDocument(path)
	if err == nil {
		t.Fatal("expected error for diagram missing root element")
	}
}

func TestLoadDocument_AcceptsValidDocument(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.drawio")
	xml := `<?xml version="1.0"?><mxfile><diagram id="d1" name="Page"><mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/></root></mxGraphModel></diagram></mxfile>`
	if err := os.WriteFile(path, []byte(xml), 0644); err != nil {
		t.Fatal(err)
	}
	doc, err := drawio.LoadDocument(path)
	if err != nil {
		t.Fatalf("expected valid document, got error: %v", err)
	}
	if len(doc.Pages()) != 1 {
		t.Errorf("expected 1 page, got %d", len(doc.Pages()))
	}
}

func TestSaveDocument_Atomic(t *testing.T) {
	doc := drawio.NewDocument()
	doc.AddPage("p1", "Page 1")

	dir := t.TempDir()
	path := filepath.Join(dir, "atomic.drawio")
	if err := drawio.SaveDocument(path, doc); err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Size() == 0 {
		t.Error("saved file is empty")
	}
}
