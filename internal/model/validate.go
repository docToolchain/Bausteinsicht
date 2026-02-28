package model

// ValidationError represents a single validation error.
type ValidationError struct {
	Path    string
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// Validate checks model consistency and returns a list of validation errors.
func Validate(m *BausteinsichtModel) []ValidationError {
	panic("not implemented")
}
