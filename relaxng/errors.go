package relaxng

import (
	"fmt"
	"strings"

	helium "github.com/lestrrat-go/helium"
)

// ValidationError represents a single validation error with structured fields.
type ValidationError struct {
	Filename string // source filename
	Line     int    // line number in the source document
	Element  string // element name
	Message  string // human-readable error description
}

func (e *ValidationError) Error() string {
	if e.Filename == "" && e.Line == 0 && e.Element == "" {
		return bareValidityError(e.Message)
	}
	return validityError(e.Filename, e.Line, e.Element, e.Message)
}

// validityError formats a validation error in libxml2 format:
//
//	{file}:{line}: element {name}: Relax-NG validity error : {msg}\n
func validityError(file string, line int, elemName, msg string) string {
	return fmt.Sprintf("%s:%d: element %s: Relax-NG validity error : %s\n", file, line, elemName, msg)
}

// bareValidityError formats a validation error without file/line/element prefix:
//
//	Relax-NG validity error : {msg}\n
func bareValidityError(msg string) string {
	return fmt.Sprintf("Relax-NG validity error : %s\n", msg)
}

// rngParserError formats a schema compilation error in libxml2 format:
//
//	Relax-NG parser error : {msg}\n
func rngParserError(msg string) string {
	return fmt.Sprintf("Relax-NG parser error : %s\n", msg)
}

// formatXMLParseError formats a helium XML parse error in libxml2-compatible style:
//
//	{file}:{line}: parser error : {msg}
//	{source_line}
//	{caret}
func formatXMLParseError(filename string, pe helium.ErrParseError) string {
	line := strings.TrimRight(pe.Line, "\n")
	caretPos := pe.Column - 1
	if caretPos < 0 {
		caretPos = 0
	}
	return fmt.Sprintf("%s:%d: parser error : %s\n%s\n%s^\n",
		filename, pe.LineNumber, pe.Err.Error(), line, strings.Repeat(" ", caretPos))
}

// rngParserErrorAt formats a schema compilation error with file/line context:
//
//	{file}:{line}: element {elemName}: Relax-NG parser error : {msg}\n
func rngParserErrorAt(file string, line int, elemName, msg string) string {
	return fmt.Sprintf("%s:%d: element %s: Relax-NG parser error : %s\n", file, line, elemName, msg)
}
