package helium

import (
	"errors"
	"fmt"
	"strings"
)

// ErrorLevel represents the severity of a parse error.
type ErrorLevel int

const (
	// ErrorLevelNone indicates no specific severity. When used with
	// [NewErrorCollector], it means all errors are collected regardless
	// of level.
	ErrorLevelNone ErrorLevel = iota
	// ErrorLevelWarning indicates a non-fatal condition that may still
	// produce correct output.
	ErrorLevelWarning
	// ErrorLevelError indicates a recoverable error in the input.
	ErrorLevelError
	// ErrorLevelFatal indicates an unrecoverable error that stops processing.
	ErrorLevelFatal
)

// ErrorDomain classifies the subsystem that produced an error.
type ErrorDomain int

const (
	// ErrorDomainParser indicates the error originated in the XML parser.
	ErrorDomainParser ErrorDomain = iota
	// ErrorDomainNamespace indicates the error originated in namespace
	// processing.
	ErrorDomainNamespace
)

// Sentinel errors for DOM operations.
var (
	ErrNilNode            = errors.New("nil node")
	ErrInvalidOperation   = errors.New("operation cannot be performed")
	ErrDuplicateAttribute = errors.New("duplicate attribute")
	ErrEntityBoundary     = errors.New("entity boundary violation")
	errParserStopped      = errors.New("parser stopped")
)

// ErrDTDDupToken is returned when a DTD attribute enumeration contains
// a duplicate token value.
type ErrDTDDupToken struct {
	Name string
}

// ErrAttrNotFound is returned when a referenced attribute token cannot
// be found in the DTD attribute declarations.
type ErrAttrNotFound struct {
	Token string
}

// ErrParseError is a structured parse error carrying the source location,
// context line, severity, and the underlying error. Use [ErrParseError.FormatError]
// to produce a libxml2-compatible multi-line diagnostic string. The error
// can be unwrapped via [ErrParseError.Unwrap] to access the underlying cause.
type ErrParseError struct {
	Column     int
	Domain     ErrorDomain
	Err        error
	File       string
	Level      ErrorLevel
	Line       string
	LineNumber int
}

// Sentinel errors for XML parse failures. Each corresponds to a specific
// syntactic violation detected by the parser. These errors typically appear
// as the Err field of an [ErrParseError].
var (
	ErrAmpersandRequired            = errors.New("'&' was required here")
	ErrAttrListNotFinished          = errors.New("attrlist must finish with a ')'")
	ErrAttrListNotStarted           = errors.New("attrlist must start with a '('")
	ErrAttributeNameRequired        = errors.New("attribute name was required here (ATTLIST)")
	ErrByteCursorRequired           = errors.New("inconsistent state: required ByteCursor")
	ErrDocTypeNameRequired          = errors.New("doctype name required")
	ErrDocTypeNotFinished           = errors.New("doctype not finished")
	ErrDocumentEnd                  = errors.New("extra content at document end")
	ErrEOF                          = errors.New("end of file reached")
	ErrElementContentNotFinished    = errors.New("element content not finished")
	ErrEmptyDocument                = errors.New("start tag expected, '<' not found")
	ErrEntityNotFound               = errors.New("entity not found")
	ErrEqualSignRequired            = errors.New("'=' was required here")
	ErrGtRequired                   = errors.New("'>' was required here")
	ErrHyphenInComment              = errors.New("'--' not allowed in comment")
	ErrInvalidChar                  = errors.New("invalid char")
	ErrInvalidComment               = errors.New("invalid comment section")
	ErrInvalidCDSect                = errors.New("invalid CDATA section")
	ErrInvalidDocument              = errors.New("invalid document")
	ErrInvalidDTD                   = errors.New("invalid DTD section")
	ErrInvalidElementDecl           = errors.New("invalid element declaration")
	ErrInvalidEncodingName          = errors.New("invalid encoding name")
	ErrInvalidName                  = errors.New("invalid xml name")
	ErrInvalidProcessingInstruction = errors.New("invalid processing instruction")
	ErrInvalidVersionNum            = errors.New("invalid version")
	ErrInvalidXMLDecl               = errors.New("invalid XML declaration")
	ErrInvalidParserCtx             = errors.New("invalid parser context")
	ErrLtSlashRequired              = errors.New("'</' is required")
	ErrMisplacedCDATAEnd            = errors.New("misplaced CDATA end ']]>'")
	ErrNameTooLong                  = errors.New("name is too long")
	ErrNameRequired                 = errors.New("name is required")
	ErrNmtokenRequired              = errors.New("nmtoken is required")
	ErrNotationNameRequired         = errors.New("notation name expected in NOTATION declaration")
	ErrNotationNotFinished          = errors.New("notation must finish with a ')'")
	ErrNotationNotStarted           = errors.New("notation must start with a '('")
	ErrOpenParenRequired            = errors.New("'(' is required")
	ErrPCDATARequired               = errors.New("'#PCDATA' required")
	ErrPercentRequired              = errors.New("'%' is required")
	ErrPrematureEOF                 = errors.New("end of document reached")
	ErrUndeclaredEntity             = errors.New("undeclared entity")
	ErrSemicolonRequired            = errors.New("';' is required")
	ErrConditionalSectionNotFinished = errors.New("conditional section ']]>' expected")
	ErrConditionalSectionKeyword    = errors.New("INCLUDE or IGNORE keyword expected in conditional section")
	ErrSpaceRequired                = errors.New("space required")
	ErrStartTagRequired             = errors.New("start tag expected, '<' not found")
	ErrValueRequired                = errors.New("value required")
)

// ErrorLevel returns the severity of this parse error, satisfying the
// [ErrorLeveler] interface.
func (e ErrParseError) ErrorLevel() ErrorLevel {
	return e.Level
}

// Error returns a human-readable string including the file, line, column,
// and a context snippet showing approximately where the error occurred.
func (e ErrParseError) Error() string {
	if e.File != "" {
		return fmt.Sprintf(
			"%s: %s at line %d, column %d\n -> '%s' <-- around here",
			e.File,
			e.Err,
			e.LineNumber,
			e.Column,
			e.Line,
		)
	}
	return fmt.Sprintf(
		"%s at line %d, column %d\n -> '%s' <-- around here",
		e.Err,
		e.LineNumber,
		e.Column,
		e.Line,
	)
}

// Unwrap returns the underlying error, enabling use with [errors.Is] and
// [errors.As].
func (e ErrParseError) Unwrap() error {
	return e.Err
}

// FormatError returns a libxml2-compatible multi-line error string:
//
//	FILE:LINE: DOMAIN SEVERITY : MESSAGE
//	CONTEXT_LINE
//	     ^
func (e ErrParseError) FormatError() string {
	var domain string
	switch e.Domain {
	case ErrorDomainNamespace:
		domain = "namespace"
	default:
		domain = "parser"
	}

	var severity string
	switch e.Level {
	case ErrorLevelWarning:
		severity = "warning"
	default:
		severity = "error"
	}

	var b strings.Builder
	if e.File != "" {
		fmt.Fprintf(&b, "%s:%d: ", e.File, e.LineNumber)
	}
	fmt.Fprintf(&b, "%s %s : %s", domain, severity, e.Err)

	if e.Line != "" {
		b.WriteByte('\n')
		b.WriteString(e.Line)
		b.WriteByte('\n')
		col := e.Column - 1
		if col < 0 {
			col = 0
		}
		for i := 0; i < col; i++ {
			b.WriteByte(' ')
		}
		b.WriteByte('^')
	}

	return b.String()
}

// Error returns the diagnostic message for this duplicate token error.
func (e ErrDTDDupToken) Error() string {
	return "standalone: attribute enumeration value token " + e.Name + " duplicated"
}
