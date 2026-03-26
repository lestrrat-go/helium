package schematron

import helium "github.com/lestrrat-go/helium"

type compileConfig struct {
	filename     string
	baseDir      string
	errorHandler helium.ErrorHandler
}

type validateConfig struct {
	filename     string
	quiet        bool
	errorHandler helium.ErrorHandler
}

// ValidationError represents a single schematron validation error.
// It implements the error interface so it can be passed to
// helium.ErrorHandler.Handle and extracted via errors.As.
type ValidationError struct {
	Filename string
	Line     int
	Element  string
	Path     string
	Message  string
}

// Error implements the error interface, producing libxml2-compatible output.
func (e *ValidationError) Error() string {
	return schematronError(e.Filename, e.Line, e.Element, e.Path, e.Message)
}
