package schema

// Error is.
type Error interface {
	error
	Path() string
}

type schemaError struct {
	err  string
	path string
}

func (e *schemaError) Error() string {
	return e.err
}

func (e *schemaError) Path() string {
	return e.path
}
