package xsd

import (
	"fmt"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
)

// ValidationError represents a single XSD validation error with structured
// fields. It implements the error interface and can be extracted from the
// values passed to a [helium.ErrorHandler] via [errors.As].
//
// AttributeName is empty for element-level errors; non-empty when the error
// concerns a specific attribute on the element.
type ValidationError struct {
	Filename      string // source filename
	Line          int    // line number in the source document
	Element       string // element name
	AttributeName string // attribute name (optional)
	Message       string // human-readable error description
}

// Error implements the error interface and produces libxml2-compatible output.
func (e *ValidationError) Error() string {
	if e.AttributeName != "" {
		return validityErrorAttr(e.Filename, e.Line, e.Element, e.AttributeName, e.Message)
	}
	return validityError(e.Filename, e.Line, e.Element, e.Message)
}

// leveledValidationError wraps a *ValidationError with an ErrorLevel so the
// existing ErrorHandler pipeline can transport typed validation errors while
// preserving the libxml2-compatible formatted output that callers currently
// rely on.
type leveledValidationError struct {
	*ValidationError
	level helium.ErrorLevel
}

func (e *leveledValidationError) ErrorLevel() helium.ErrorLevel { return e.level }

func (e *leveledValidationError) Unwrap() error { return e.ValidationError }

// newLeveledValidationError pairs a *ValidationError with an ErrorLevel for
// transport through helium.ErrorHandler. The returned error unwraps to the
// embedded *ValidationError so callers can [errors.As] for it.
func newLeveledValidationError(ve *ValidationError, level helium.ErrorLevel) error {
	return &leveledValidationError{ValidationError: ve, level: level}
}

// validityError formats a validation error in libxml2 format:
//
//	./test/schemas/{file}.xml:{line}: Schemas validity error : Element '{name}': {msg}\n
func validityError(file string, line int, elemName, msg string) string {
	return fmt.Sprintf("%s:%d: Schemas validity error : Element '%s': %s\n", file, line, elemName, msg)
}

// validityErrorAttr formats a validation error for an attribute in libxml2 format:
//
//	./test/schemas/{file}.xml:{line}: Schemas validity error : Element '{elem}', attribute '{attr}': {msg}\n
func validityErrorAttr(file string, line int, elemName, attrName, msg string) string {
	return fmt.Sprintf("%s:%d: Schemas validity error : Element '%s', attribute '%s': %s\n", file, line, elemName, attrName, msg)
}

// schemaParserError formats a schema compilation error in libxml2 format:
//
//	{file}:{line}: element {elemLocal}: Schemas parser error : Element '{xsdNS}{xsdElem}': {msg}\n
func schemaParserError(file string, line int, elemLocal, xsdElem, msg string) string {
	return fmt.Sprintf("%s:%d: element %s: Schemas parser error : Element '{%s}%s': %s\n", file, line, elemLocal, lexicon.NamespaceXSD, xsdElem, msg)
}

// schemaParserErrorAttr formats a schema compilation error for a specific attribute:
//
//	{file}:{line}: element {elemLocal}: Schemas parser error : Element '{xsdNS}{xsdElem}', attribute '{attr}': {msg}\n
func schemaParserErrorAttr(file string, line int, elemLocal, xsdElem, attr, msg string) string {
	return fmt.Sprintf("%s:%d: element %s: Schemas parser error : Element '{%s}%s', attribute '%s': %s\n", file, line, elemLocal, lexicon.NamespaceXSD, xsdElem, attr, msg)
}

// schemaParserWarning formats a schema compilation warning in libxml2 format:
//
//	{file}:{line}: element {elemLocal}: Schemas parser warning : Element '{xsdNS}{xsdElem}': {msg}\n
func schemaParserWarning(file string, line int, elemLocal, xsdElem, msg string) string {
	return fmt.Sprintf("%s:%d: element %s: Schemas parser warning : Element '{%s}%s': %s\n", file, line, elemLocal, lexicon.NamespaceXSD, xsdElem, msg)
}

// schemaComponentError formats a schema compilation error for a component (e.g. "local complex type"):
//
//	{file}:{line}: element {elemLocal}: Schemas parser error : {component}: {msg}\n
func schemaComponentError(file string, line int, elemLocal, component, msg string) string {
	return fmt.Sprintf("%s:%d: element %s: Schemas parser error : %s: %s\n", file, line, elemLocal, component, msg)
}

// schemaElemDeclError formats a schema compilation error for an element declaration:
//
//	{file}:{line}: element element: Schemas parser error : element decl. '{name}': {msg}\n
func schemaElemDeclError(file string, line int, declName, msg string) string {
	return fmt.Sprintf("%s:%d: element element: Schemas parser error : element decl. '%s': %s\n", file, line, declName, msg)
}

// schemaElemDeclErrorAttr formats a schema compilation error for an element
// declaration's @type attribute (the only element-declaration attribute this
// diagnostic is emitted for):
//
//	{file}:{line}: element element: Schemas parser error : element decl. '{name}', attribute 'type': {msg}\n
func schemaElemDeclErrorAttr(file string, line int, declName, msg string) string {
	return fmt.Sprintf("%s:%d: element element: Schemas parser error : element decl. '%s', attribute '%s': %s\n", file, line, declName, attrType, msg)
}
