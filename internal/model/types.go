package model

// Element lifecycle status values
const (
	StatusProposed      = "proposed"
	StatusDesign        = "design"
	StatusImplementing  = "implementation"
	StatusDeployed      = "deployed"
	StatusDeprecated    = "deprecated"
	StatusArchived      = "archived"
)

var ValidStatuses = []string{
	StatusProposed,
	StatusDesign,
	StatusImplementing,
	StatusDeployed,
	StatusDeprecated,
	StatusArchived,
}

// StatusColor returns the draw.io badge color for a given status
func StatusColor(status string) string {
	switch status {
	case StatusProposed:
		return "#fff2cc" // yellow
	case StatusDesign:
		return "#dae8fc" // blue
	case StatusImplementing:
		return "#ffe6cc" // orange
	case StatusDeployed:
		return "#d5e8d4" // green
	case StatusDeprecated:
		return "#f8cecc" // red
	case StatusArchived:
		return "#f5f5f5" // grey
	default:
		return "#ffffff" // white
	}
}

// DecisionBadgeColor returns the draw.io badge color for a given ADR status
func DecisionBadgeColor(status ADRStatus) string {
	switch status {
	case ADRActive:
		return "#0066cc" // blue
	case ADRSuperseded, ADRDeprecated:
		return "#999999" // grey
	case ADRProposed:
		return "#ffcc00" // yellow
	default:
		return "#ffffff" // white
	}
}

// Config holds top-level configuration for diagram generation.
type Config struct {
	Metadata *bool  `json:"metadata,omitempty"`
	Legend   *bool  `json:"legend,omitempty"`
	Author   string `json:"author,omitempty"`
	Repo     string `json:"repo,omitempty"`
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
	Constraints   []Constraint       `json:"constraints,omitempty"`

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

// Constraint defines an architectural rule that can be enforced via `bausteinsicht lint`.
type Constraint struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Rule        string `json:"rule"`

	// no-relationship / allowed-relationship
	FromKind  string   `json:"from-kind,omitempty"`
	ToKind    string   `json:"to-kind,omitempty"`
	FromKinds []string `json:"from-kinds,omitempty"`

	// required-field
	ElementKind string `json:"element-kind,omitempty"`
	Field       string `json:"field,omitempty"`

	// max-depth
	Max int `json:"max,omitempty"`

	// technology-allowed
	Technologies []string `json:"technologies,omitempty"`
}

// ADRStatus describes the status of an architecture decision record
type ADRStatus string

const (
	ADRProposed    ADRStatus = "proposed"
	ADRActive      ADRStatus = "active"
	ADRDeprecated  ADRStatus = "deprecated"
	ADRSuperseded  ADRStatus = "superseded"
)

// DecisionRecord represents an architecture decision record (ADR)
type DecisionRecord struct {
	ID      string    `json:"id"`
	Title   string    `json:"title"`
	Status  ADRStatus `json:"status"`
	Date    string    `json:"date,omitempty"`
	FilePath string   `json:"file,omitempty"`
}

type Specification struct {
	Elements      map[string]ElementKind      `json:"elements"`
	Relationships map[string]RelationshipKind `json:"relationships,omitempty"`
	Decisions     []DecisionRecord            `json:"decisions,omitempty"`
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
	Status      string             `json:"status,omitempty"`
	Decisions   []string           `json:"decisions,omitempty"`
	Children    map[string]Element `json:"children,omitempty"`
	Metadata    map[string]string  `json:"metadata,omitempty"`
}

type Relationship struct {
	From        string   `json:"from"`
	To          string   `json:"to"`
	Label       string   `json:"label,omitempty"`
	Kind        string   `json:"kind,omitempty"`
	Description string   `json:"description,omitempty"`
	Decisions   []string `json:"decisions,omitempty"`
}

type View struct {
	Title       string   `json:"title"`
	Scope       string   `json:"scope,omitempty"`
	Include     []string `json:"include,omitempty"`
	Exclude     []string `json:"exclude,omitempty"`
	Description string   `json:"description,omitempty"`
	Layout      string   `json:"layout,omitempty"`
}
