package workspace

import "github.com/docToolchain/Bausteinsicht/internal/model"

// Config defines a multi-model workspace configuration
type Config struct {
	Workspace WorkspaceMetadata        `json:"workspace"`
	Models    []ModelRef               `json:"models"`
	Views     map[string]WorkspaceView `json:"views,omitempty"`
	CrossRels []CrossModelRelationship `json:"crossModelRelationships,omitempty"`
}

// WorkspaceMetadata contains workspace identification and metadata
type WorkspaceMetadata struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// ModelRef references a team model with ID and path
type ModelRef struct {
	ID     string `json:"id"`
	Path   string `json:"path"`
	Prefix string `json:"prefix,omitempty"`
	// Elements selects top-level elements of the model to import (each with its
	// children, cut off at Depth). Empty means all top-level elements.
	Elements []string `json:"elements,omitempty"`
	// Depth limits how many levels of the selected elements are imported:
	// 1 is the element alone (a black box), 2 adds its children, and so on.
	// 0 means no limit. A model with Elements or Depth set is imported
	// partially: its views, dynamic views and constraints are not imported.
	Depth int `json:"depth,omitempty"`
}

// CrossModelRelationship connects elements across different models
type CrossModelRelationship struct {
	ID          string `json:"id"`
	From        string `json:"from"`
	To          string `json:"to"`
	Label       string `json:"label,omitempty"`
	Kind        string `json:"kind,omitempty"`
	Description string `json:"description,omitempty"`
}

// WorkspaceView filters and displays elements from multiple models
type WorkspaceView struct {
	Title        string   `json:"title"`
	IncludeFrom  []string `json:"include-from,omitempty"`
	IncludeKinds []string `json:"include-kinds,omitempty"`
	ExcludeKinds []string `json:"exclude-kinds,omitempty"`
	Description  string   `json:"description,omitempty"`
	Layout       string   `json:"layout,omitempty"`
}

// LoadedModel holds a loaded model with its reference metadata
type LoadedModel struct {
	Ref   ModelRef
	Model *model.BausteinsichtModel
}
