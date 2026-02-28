package model

// BausteinsichtModel is the top-level model file
type BausteinsichtModel struct {
	Schema        string             `json:"$schema,omitempty"`
	Specification Specification      `json:"specification"`
	Model         map[string]Element `json:"model"`
	Relationships []Relationship     `json:"relationships"`
	Views         map[string]View    `json:"views"`
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
	Kind        string            `json:"kind"`
	Title       string            `json:"title"`
	Description string            `json:"description,omitempty"`
	Technology  string            `json:"technology,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Children    map[string]Element `json:"children,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
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
}
