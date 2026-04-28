package model

// Config holds top-level configuration for diagram generation.
type Config struct {
	Metadata *bool  `json:"metadata,omitempty"`
	Legend   *bool  `json:"legend,omitempty"`
	Author string `json:"author,omitempty"`
	Repo   string `json:"repo,omitempty"`
}

// BausteinsichtModel is the top-level model file
type BausteinsichtModel struct {
	Schema        string             `json:"$schema,omitempty"`
	Config        Config             `json:"config,omitempty"`
	Specification Specification      `json:"specification"`
	Model         map[string]Element `json:"model"`
	Relationships []Relationship     `json:"relationships"`
	Views         map[string]View    `json:"views"`
	DynamicViews  []DynamicView      `json:"dynamicViews,omitempty"`

	// ElementOrder stores the definition order of element kinds from
	// specification.elements. Used by the layout engine for layer assignment.
	ElementOrder []string `json:"-"`
}

// StepType describes how a sequence step arrow is rendered.
type StepType string

const (
	StepSync   StepType = "sync"
	StepAsync  StepType = "async"
	StepReturn StepType = "return"
)

// SequenceStep is one message/call in a dynamic view.
type SequenceStep struct {
	Index int      `json:"index"`
	From  string   `json:"from"`
	To    string   `json:"to"`
	Label string   `json:"label"`
	Type  StepType `json:"type,omitempty"`
}

// DynamicView describes a sequence of interactions between elements.
type DynamicView struct {
	Key         string         `json:"key"`
	Title       string         `json:"title"`
	Description string         `json:"description,omitempty"`
	Steps       []SequenceStep `json:"steps"`
}

type Specification struct {
	Elements      map[string]ElementKind      `json:"elements"`
	Relationships map[string]RelationshipKind `json:"relationships,omitempty"`
}

type ElementKind struct {
	Notation    string `json:"notation"`
	Description string `json:"description,omitempty"`
	Container   bool   `json:"container,omitempty"`
}

type RelationshipKind struct {
	Notation string `json:"notation"`
	Dashed   bool   `json:"dashed,omitempty"`
}

type Element struct {
	Kind        string             `json:"kind"`
	Title       string             `json:"title"`
	Description string             `json:"description,omitempty"`
	Technology  string             `json:"technology,omitempty"`
	Tags        []string           `json:"tags,omitempty"`
	Children    map[string]Element `json:"children,omitempty"`
	Metadata    map[string]string  `json:"metadata,omitempty"`
}

type Relationship struct {
	From        string `json:"from"`
	To          string `json:"to"`
	Label       string `json:"label,omitempty"`
	Kind        string `json:"kind,omitempty"`
	Description string `json:"description,omitempty"`
}

type View struct {
	Title       string   `json:"title"`
	Scope       string   `json:"scope,omitempty"`
	Include     []string `json:"include,omitempty"`
	Exclude     []string `json:"exclude,omitempty"`
	Description string   `json:"description,omitempty"`
	Layout      string   `json:"layout,omitempty"`
}
