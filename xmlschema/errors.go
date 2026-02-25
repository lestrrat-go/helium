package xmlschema

import (
	"fmt"
	"strings"
)

// SchemaError collects validation errors in libxml2-compatible format.
type SchemaError struct {
	errors []string
}

// Error returns all collected errors joined by newlines.
func (e *SchemaError) Error() string {
	return strings.Join(e.errors, "")
}

// Errors returns the individual error strings.
func (e *SchemaError) Errors() []string {
	return e.errors
}

// validityError formats a validation error in libxml2 format:
//
//	./test/schemas/{file}.xml:{line}: Schemas validity error : Element '{name}': {msg}\n
func validityError(file string, line int, elemName, msg string) string {
	return fmt.Sprintf("%s:%d: Schemas validity error : Element '%s': %s\n", file, line, elemName, msg)
}
