package sync

// Field-name constants shared across diff.go, forward.go, and metadata.go —
// used both as ElementChange/RelationshipChange.Field values and as draw.io
// element attribute names, which happen to coincide. Defined once to avoid
// the same literal being repeated throughout the package (SonarCloud go:S1192).
const (
	fieldTitle       = "title"
	fieldDescription = "description"
	fieldTechnology  = "technology"
	fieldLabel       = "label"

	attrBausteinsichtID = "bausteinsicht_id"
)
