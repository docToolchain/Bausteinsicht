package sync

import (
	"testing"

	"github.com/beevik/etree"
	"github.com/docToolchain/Bausteinsicht/internal/model"
)

func TestAddStatusBadge_CreatesChildCell(t *testing.T) {
	element := etree.NewElement("mxCell")
	element.CreateAttr("id", "test-element")
	element.CreateAttr("width", "100")
	element.CreateAttr("height", "60")

	AddStatusBadge(element, model.StatusDeployed)

	children := element.SelectElements("mxCell")
	if len(children) != 1 {
		t.Errorf("expected 1 badge child, got %d", len(children))
	}

	badge := children[0]
	if value := getAttr(badge, "value"); value != "deployed" {
		t.Errorf("expected value 'deployed', got %q", value)
	}
}

func TestAddStatusBadge_CorrectColor(t *testing.T) {
	tests := []struct {
		status       string
		expectedColor string
	}{
		{model.StatusProposed, "#fff2cc"},
		{model.StatusDesign, "#dae8fc"},
		{model.StatusImplementing, "#ffe6cc"},
		{model.StatusDeployed, "#d5e8d4"},
		{model.StatusDeprecated, "#f8cecc"},
		{model.StatusArchived, "#f5f5f5"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			element := etree.NewElement("mxCell")
			element.CreateAttr("id", "test")

			AddStatusBadge(element, tt.status)

			badge := element.SelectElements("mxCell")[0]
			style := getAttr(badge, "style")
			if !containsSubstring(style, tt.expectedColor) {
				t.Errorf("expected color %q in style, got %q", tt.expectedColor, style)
			}
		})
	}
}

func TestAddStatusBadge_NoopForEmptyStatus(t *testing.T) {
	element := etree.NewElement("mxCell")
	element.CreateAttr("id", "test")

	AddStatusBadge(element, "")

	children := element.SelectElements("mxCell")
	if len(children) > 0 {
		t.Error("expected no badge for empty status")
	}
}

func TestAddStatusBadge_BadgeHasGeometry(t *testing.T) {
	element := etree.NewElement("mxCell")
	element.CreateAttr("id", "test")

	AddStatusBadge(element, model.StatusDeployed)

	badge := element.SelectElements("mxCell")[0]
	geom := badge.SelectElement("mxGeometry")
	if geom == nil {
		t.Error("expected mxGeometry element in badge")
	}

	if x := getAttr(geom, "x"); x == "" {
		t.Error("expected x attribute in geometry")
	}
	if y := getAttr(geom, "y"); y == "" {
		t.Error("expected y attribute in geometry")
	}
}

func containsSubstring(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
