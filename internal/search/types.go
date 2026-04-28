package search

// ResultType identifies the kind of object a search result refers to.
type ResultType string

const (
	ResultElement      ResultType = "element"
	ResultRelationship ResultType = "relationship"
	ResultView         ResultType = "view"
)

// Options controls the search behaviour.
type Options struct {
	// Type restricts results to a specific object type. Empty string means all.
	Type ResultType
}

// Result represents a single match from a search query.
type Result struct {
	Type          ResultType `json:"type"`
	ID            string     `json:"id"`
	Title         string     `json:"title,omitempty"`
	Kind          string     `json:"kind,omitempty"`
	Score         int        `json:"score"`
	MatchedFields []string   `json:"matchedFields"`

	// Extra fields populated depending on type.
	Technology  string `json:"technology,omitempty"`
	Status      string `json:"status,omitempty"`
	From        string `json:"from,omitempty"`
	To          string `json:"to,omitempty"`
	Description string `json:"description,omitempty"`
}

// Response is the top-level JSON output of a search.
type Response struct {
	Query   string   `json:"query"`
	Results []Result `json:"results"`
	Total   int      `json:"total"`
}
