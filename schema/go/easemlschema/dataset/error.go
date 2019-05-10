package dataset

// Error is.
type Error interface {
	error
	Path() string
}

type datasetError struct {
	err  string
	path string
}

func (e *datasetError) Error() string {
	return e.err
}

func (e *datasetError) Path() string {
	return e.path
}
