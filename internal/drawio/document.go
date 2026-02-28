// Package drawio handles reading and writing draw.io XML files.
package drawio

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/beevik/etree"
)

// Document represents a draw.io file (mxfile).
type Document struct {
	tree *etree.Document
}

// Page represents a single page (diagram element) within a Document.
type Page struct {
	diagram *etree.Element
}

// LoadDocument parses a draw.io XML file from disk.
func LoadDocument(path string) (*Document, error) {
	tree := etree.NewDocument()
	if err := tree.ReadFromFile(path); err != nil {
		return nil, fmt.Errorf("LoadDocument %q: %w", path, err)
	}
	return &Document{tree: tree}, nil
}

// SaveDocument writes a Document to disk using an atomic temp-file + rename.
func SaveDocument(path string, doc *Document) error {
	doc.tree.Indent(2)

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".drawio-tmp-*")
	if err != nil {
		return fmt.Errorf("SaveDocument create temp: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := doc.tree.WriteTo(tmp); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("SaveDocument write: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("SaveDocument close: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("SaveDocument rename: %w", err)
	}
	return nil
}

// NewDocument creates an empty mxfile document.
func NewDocument() *Document {
	tree := etree.NewDocument()
	tree.CreateProcInst("xml", `version="1.0" encoding="UTF-8"`)
	mxfile := tree.CreateElement("mxfile")
	mxfile.CreateAttr("host", "bausteinsicht")
	mxfile.CreateAttr("compressed", "false")
	return &Document{tree: tree}
}

// Pages returns all pages in the document.
func (d *Document) Pages() []*Page {
	root := d.tree.Root()
	if root == nil {
		return nil
	}
	diagrams := root.SelectElements("diagram")
	pages := make([]*Page, len(diagrams))
	for i, el := range diagrams {
		pages[i] = &Page{diagram: el}
	}
	return pages
}

// GetPage returns the page with the given id, or nil if not found.
func (d *Document) GetPage(id string) *Page {
	root := d.tree.Root()
	if root == nil {
		return nil
	}
	for _, el := range root.SelectElements("diagram") {
		if el.SelectAttrValue("id", "") == id {
			return &Page{diagram: el}
		}
	}
	return nil
}

// AddPage adds a new page with the given id and name, initialised with base cells.
func (d *Document) AddPage(id, name string) *Page {
	root := d.tree.Root()
	if root == nil {
		root = d.tree.CreateElement("mxfile")
	}

	diagram := root.CreateElement("diagram")
	diagram.CreateAttr("id", id)
	diagram.CreateAttr("name", name)

	model := diagram.CreateElement("mxGraphModel")
	model.CreateAttr("dx", "1422")
	model.CreateAttr("dy", "794")
	model.CreateAttr("grid", "1")
	model.CreateAttr("gridSize", "10")
	model.CreateAttr("page", "1")
	model.CreateAttr("pageWidth", "1169")
	model.CreateAttr("pageHeight", "827")
	model.CreateAttr("background", "#ffffff")

	rootEl := model.CreateElement("root")
	cell0 := rootEl.CreateElement("mxCell")
	cell0.CreateAttr("id", "0")
	cell1 := rootEl.CreateElement("mxCell")
	cell1.CreateAttr("id", "1")
	cell1.CreateAttr("parent", "0")

	return &Page{diagram: diagram}
}

// Root returns the <root> element of the page for direct manipulation.
func (p *Page) Root() *etree.Element {
	model := p.diagram.FindElement("mxGraphModel")
	if model == nil {
		return nil
	}
	return model.FindElement("root")
}
