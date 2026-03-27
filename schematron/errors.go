package schematron

import "fmt"

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

// schematronError formats a validation error in libxml2 format:
//
//	{file}:{line}: element {elemName}: schematron error : {nodePath} line {line}: {message}\n
func schematronError(file string, line int, elemName, nodePath, message string) string {
	return fmt.Sprintf("%s:%d: element %s: schematron error : %s line %d: %s\n", file, line, elemName, nodePath, line, message)
}
