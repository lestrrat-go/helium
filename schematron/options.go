package schematron

import (
	"fmt"

	helium "github.com/lestrrat-go/helium"
)

type compileConfig struct {
	filename     string
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

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s:%d: element %s: Schematron error : %s", e.Filename, e.Line, e.Element, e.Message)
}
