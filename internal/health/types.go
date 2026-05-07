package health

// ScoreCategory represents a dimension of architectural health.
type ScoreCategory string

const (
	CategoryCompleteness   ScoreCategory = "completeness"
	CategoryConformance    ScoreCategory = "conformance"
	CategoryComplexity     ScoreCategory = "complexity"
	CategoryDeprecation    ScoreCategory = "deprecation"
	CategoryDocumentation ScoreCategory = "documentation"
)

// Finding describes a single health issue or improvement area.
type Finding struct {
	Category ScoreCategory `json:"category"`
	Severity string        `json:"severity"` // "critical", "major", "minor", "info"
	Title    string        `json:"title"`
	Message  string        `json:"message"`
	Elements []string      `json:"elements,omitempty"` // affected element IDs
}

// CategoryScore represents the score for a single dimension.
type CategoryScore struct {
	Category  ScoreCategory `json:"category"`
	Score     float64       `json:"score"`      // 0-100
	Weight    float64       `json:"weight"`     // 0-1
	Findings  []Finding     `json:"findings"`
	Details   string        `json:"details,omitempty"`
}

// HealthScore is the overall architecture health assessment.
type HealthScore struct {
	Overall     float64          `json:"overall"`     // 0-100 weighted average
	Categories  []CategoryScore  `json:"categories"`
	Grade       string           `json:"grade"`       // A+, A, B+, B, C+, C, D, F
	Summary     string           `json:"summary"`
	Timestamp   string           `json:"timestamp"`   // ISO8601
	ElementCnt  int              `json:"elementCnt"`
	RelCnt      int              `json:"relCnt"`
	ViewCnt     int              `json:"viewCnt"`
}

// calculateGrade converts a numeric score to a letter grade.
func calculateGrade(score float64) string {
	switch {
	case score >= 97:
		return "A+"
	case score >= 93:
		return "A"
	case score >= 90:
		return "B+"
	case score >= 87:
		return "B"
	case score >= 80:
		return "C+"
	case score >= 70:
		return "C"
	case score >= 60:
		return "D"
	default:
		return "F"
	}
}
