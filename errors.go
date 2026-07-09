package helium

import (
	"context"
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
	// ErrEntityNotWellBalanced is returned when an internal general entity's
	// replacement text is not well balanced with respect to element nesting —
	// e.g. it opens with an end-tag or closes an element opened outside the
	// entity (WFC: Parsed entities must be well-formed; XML §4.3.2). Referencing
	// such an entity in element content is a fatal well-formedness error.
	ErrEntityNotWellBalanced = errors.New("entity content is not well balanced")
	// ErrWalkCycle is returned by Walk when the traversal encounters a
	// child-pointer cycle — a node reachable from itself through child links.
	// A well-formed, parent-consistent tree never triggers it; it guards
	// hand-built or foreign-linked graphs (e.g. an entity reference whose
	// Entity child links back to the reference) from making Walk loop forever.
	// Match with errors.Is.
	ErrWalkCycle = errors.New("cycle detected during tree traversal")
	// ErrExternalDTDTooLarge is returned when an external DTD subset exceeds
	// the configured byte cap (set via Parser.MaxExternalDTDBytes), or
	// MaxExternalDTDSize when no cap is configured. The cap is enforced
	// against the actual number of bytes read, not any advisory Stat size.
	ErrExternalDTDTooLarge = errors.New("external DTD exceeds maximum allowed size")
	// ErrNodeContentTooLarge is returned when a single indivisible content run —
	// a CDATA section, comment body, processing-instruction body,
	// character-data run, or attribute value — exceeds the configured byte cap
	// (set via Parser.MaxNodeContentSize), or DefaultMaxNodeContentSize when no
	// cap is configured. These constructs map to a single SAX event / DOM node
	// (or attribute) and cannot be chunked, so an oversized one is a
	// memory-amplification vector on untrusted input. The same cap also bounds a
	// single contiguous run of XML whitespace (a blank skip): an
	// attacker-controlled unbounded whitespace run would otherwise grow the
	// cursor buffer without limit, so a blank run over the cap fails with this
	// error too. MaxNodeContentSize(-1) disables both the node-content and the
	// blank-run cap. The cap fires during accumulation, before the whole run is
	// buffered; match with errors.Is.
	ErrNodeContentTooLarge = errors.New("node content exceeds maximum allowed size")
	errParserStopped       = errors.New("parser stopped")
	errNoCursor            = errors.New("parser has no input")
)

// isParseAbort reports whether err signals that parsing must stop immediately
// rather than be treated as a recoverable parse error. This covers the
// internal stop sentinel (helium.StopParser) and context cancellation /
// deadline expiry. Such errors must never enter recovery, be rewrapped as a
// parse error, or fire SAX error handlers as if the document were malformed.
func isParseAbort(err error) bool {
	return errors.Is(err, errParserStopped) ||
		errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded)
}

// DTDDupTokenError is returned when a DTD attribute enumeration contains
// a duplicate token value.
type DTDDupTokenError struct {
	Name string
}

// AttrNotFoundError is returned when a referenced attribute token cannot
// be found in the DTD attribute declarations.
type AttrNotFoundError struct {
	Token string
}

func (e AttrNotFoundError) Error() string {
	return "attribute token '" + e.Token + "' not found"
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
	ErrAmpersandRequired             = errors.New("'&' was required here")
	ErrAttrListNotFinished           = errors.New("attrlist must finish with a ')'")
	ErrAttrListNotStarted            = errors.New("attrlist must start with a '('")
	ErrAttributeNameRequired         = errors.New("attribute name was required here (ATTLIST)")
	ErrByteCursorRequired            = errors.New("inconsistent state: required ByteCursor")
	ErrDocTypeNameRequired           = errors.New("doctype name required")
	ErrDocTypeNotFinished            = errors.New("doctype not finished")
	ErrDocumentEnd                   = errors.New("extra content at document end")
	ErrEOF                           = errors.New("end of file reached")
	ErrElementContentNotFinished     = errors.New("element content not finished")
	ErrEmptyDocument                 = errors.New("start tag expected, '<' not found")
	ErrEntityNotFound                = errors.New("entity not found")
	ErrEncodingBOMMismatch           = errors.New("declared encoding conflicts with the byte-order mark")
	ErrEqualSignRequired             = errors.New("'=' was required here")
	ErrGtRequired                    = errors.New("'>' was required here")
	ErrHyphenInComment               = errors.New("'--' not allowed in comment")
	ErrInvalidChar                   = errors.New("invalid char")
	ErrInvalidComment                = errors.New("invalid comment section")
	ErrInvalidCDSect                 = errors.New("invalid CDATA section")
	ErrInvalidDocument               = errors.New("invalid document")
	ErrInvalidDTD                    = errors.New("invalid DTD section")
	ErrInvalidElementDecl            = errors.New("invalid element declaration")
	ErrInvalidEncodingName           = errors.New("invalid encoding name")
	ErrInvalidName                   = errors.New("invalid xml name")
	ErrInvalidProcessingInstruction  = errors.New("invalid processing instruction")
	ErrInvalidVersionNum             = errors.New("invalid version")
	ErrInvalidXMLDecl                = errors.New("invalid XML declaration")
	ErrInvalidParserCtx              = errors.New("invalid parser context")
	ErrLtSlashRequired               = errors.New("'</' is required")
	ErrMisplacedCDATAEnd             = errors.New("misplaced CDATA end ']]>'")
	ErrNameTooLong                   = errors.New("name is too long")
	ErrNameRequired                  = errors.New("name is required")
	ErrNmtokenRequired               = errors.New("nmtoken is required")
	ErrNotationNameRequired          = errors.New("notation name expected in NOTATION declaration")
	ErrNotationExternalIDRequired    = errors.New("ExternalID or PublicID required in NOTATION declaration")
	ErrNotationNotFinished           = errors.New("notation must finish with a ')'")
	ErrNotationNotStarted            = errors.New("notation must start with a '('")
	ErrOpenParenRequired             = errors.New("'(' is required")
	ErrPCDATARequired                = errors.New("'#PCDATA' required")
	ErrPercentRequired               = errors.New("'%' is required")
	ErrPEReferenceInInternalSubset   = errors.New("PEReferences forbidden in internal subset")
	ErrPrematureEOF                  = errors.New("end of document reached")
	ErrNotStandalone                 = errors.New("document marked standalone but requires external subset")
	ErrUndeclaredEntity              = errors.New("undeclared entity")
	ErrSemicolonRequired             = errors.New("';' is required")
	ErrConditionalSectionNotFinished = errors.New("conditional section ']]>' expected")
	ErrConditionalSectionKeyword     = errors.New("INCLUDE or IGNORE keyword expected in conditional section")
	ErrSpaceRequired                 = errors.New("space required")
	ErrStartTagRequired              = errors.New("start tag expected, '<' not found")
	ErrValueRequired                 = errors.New("value required")
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
		col := max(e.Column-1, 0)
		for range col {
			b.WriteByte(' ')
		}
		b.WriteByte('^')
	}

	return b.String()
}

// Error returns the diagnostic message for this duplicate token error.
func (e DTDDupTokenError) Error() string {
	return "standalone: attribute enumeration value token " + e.Name + " duplicated"
}
