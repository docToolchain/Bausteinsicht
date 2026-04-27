package sync

import (
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/drawio"
)

// requirePage retrieves a page by view key, failing the test if not found.
func requirePage(t *testing.T, doc *drawio.Document, viewKey string) *drawio.Page {
	t.Helper()
	page := doc.GetPage(viewKey)
	if page == nil {
		t.Fatalf("expected page %q to exist", viewKey)
	}
	return page
}

// requireFirstPage retrieves the first page, failing the test if none exist.
func requireFirstPage(t *testing.T, doc *drawio.Document) *drawio.Page {
	t.Helper()
	pages := doc.Pages()
	if len(pages) == 0 {
		t.Fatal("expected at least one page")
	}
	return pages[0]
}
