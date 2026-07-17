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
	ErrNilNode          = errors.New("nil node")
	ErrInvalidOperation = errors.New("operation cannot be performed")
	ErrEntityBoundary   = errors.New("entity boundary violation")
	// ErrInvalidArgument is returned when a public builder is given an
	// out-of-range enum argument — e.g. an AddAttributeDecl attribute type or
	// default-declaration kind that is not a defined enum value. It is wrapped
	// (via %w) into a message describing the specific violation, so a caller can
	// branch on it with errors.Is while still surfacing the text.
	ErrInvalidArgument = errors.New("invalid argument")
	// ErrNoInternalSubset is returned by Document.InternalSubset when the
	// document has no internal DTD subset associated with it. Match with
	// errors.Is.
	ErrNoInternalSubset = errors.New("no internal subset is associated with this document")
	// ErrDuplicateDeclaration is returned when a DTD declaration collides with an
	// existing declaration of the same kind and name: a second AddElementDecl,
	// AddNotation, or AddAttributeDecl for an element, notation, or
	// (element, attribute) pair that is already declared in the DTD. It is
	// wrapped (via %w) into a message naming the kind and name, so a caller can
	// branch on the collision with errors.Is while still surfacing the
	// human-readable text.
	ErrDuplicateDeclaration = errors.New("duplicate DTD declaration")
	// ErrEntityVersionMismatch is returned when an external parsed entity (or
	// the external DTD subset) declares, in its TextDecl, an XML version later
	// than the referencing document's (XML §4.3.4). Helium targets XML 1.0, so a
	// TextDecl declaring anything other than "1.0" (e.g. "1.1") in a 1.0 document
	// is a fatal error (libxml2 XML_ERR_VERSION_MISMATCH).
	ErrEntityVersionMismatch = errors.New("version mismatch between document and entity")

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
	// ErrElementDeclNotFound is returned by Document.IsMixedElement when neither
	// the internal nor the external subset has an element declaration for the
	// given name (or the document has no internal subset at all). Match with
	// errors.Is.
	ErrElementDeclNotFound = errors.New("element declaration not found")
	// ErrUnsupportedOutputEncoding is returned by the writer for an effective
	// encoding it cannot faithfully emit. A malformed EncName label — whether
	// from an explicit OutputEncoding override OR a document's own encoding set
	// via Document.SetEncoding — is rejected before any output byte is written
	// (ahead of the transcoding encoder, whose deferred flush would emit a BOM).
	// The two emit-failure cases below are additionally scoped to an
	// explicit OutputEncoding override; a document's own PARSED encoding is
	// always a valid EncName and stays declaration-only, keeping default output
	// byte-identical:
	//   1. The encoding is neither UTF-8/US-ASCII nor a name the internal encoder
	//      table can load. Emitting UTF-8 octets under that declaration would make
	//      the XML declaration disagree with the bytes, so the writer fails.
	//   2. The encoding is US-ASCII (any alias) on the octet-producing WriteTo
	//      path and a non-ASCII character reaches the output where no character
	//      reference can represent it. Text and attribute values are always
	//      character-referenced to pure ASCII, so they never fail; ANY other
	//      non-ASCII byte does — a name, comment, CDATA, PI, namespace prefix, a
	//      character-map replacement, a DTD external/system/public-ID literal, an
	//      entity or notation value/name, or any future raw-write site. An
	//      exhaustive output-writer net rejects any surviving byte >= 0x80, with
	//      per-site guards giving earlier, labelled errors on the common paths.
	//      fn:serialize's declaration-only US-ASCII mode is excluded: it returns a
	//      UTF-8 string, so non-ASCII text/attr values still character-reference
	//      but reference-less content (and character-map replacements) stay raw.
	// Match with errors.Is.
	ErrUnsupportedOutputEncoding = errors.New("unsupported output encoding")
	// ErrInvalidOutputVersion is returned by the writer when the effective output
	// XML version (the OutputVersion override, or the document's own version) is
	// not a valid XML VersionNum production `'1.' [0-9]+` (XML §2.8). The version
	// is written raw between the double quotes of the XML declaration's version
	// pseudo-attribute, so a value carrying a quote or other illegal character
	// would break out of the pseudo-attribute and inject markup; a malformed value
	// would produce an unparseable declaration. The writer validates the version
	// before emitting any output byte and fails closed, emitting nothing.
	// Match with errors.Is.
	ErrInvalidOutputVersion = errors.New("invalid output XML version")
	// ErrWriterReservedElementName, ErrWriterReservedAttributeName, and
	// ErrWriterReservedNamespacePrefix flag a name reserved for namespace
	// declarations (the "xmlns" name or prefix); such declarations must go through
	// DeclareNamespace, not a literal name. These, and the sibling writer
	// structural sentinels below, flag a DOM node Writer.WriteTo cannot serialize
	// into well-formed XML. Each is wrapped (via %w) into a descriptive,
	// value-bearing message, so a caller can branch on the failure class with
	// errors.Is while still surfacing the human-readable text.
	ErrWriterReservedElementName     = errors.New("reserved element name")
	ErrWriterReservedAttributeName   = errors.New("reserved attribute name")
	ErrWriterReservedNamespacePrefix = errors.New("reserved namespace prefix")
	// ErrWriterInvalidElementName, ErrWriterInvalidAttributeName, and
	// ErrWriterInvalidNamespacePrefix flag a name that is not a valid XML
	// QName/NCName and would inject raw markup if emitted verbatim.
	ErrWriterInvalidElementName     = errors.New("invalid element name")
	ErrWriterInvalidAttributeName   = errors.New("invalid attribute name")
	ErrWriterInvalidNamespacePrefix = errors.New("invalid namespace prefix")
	// ErrWriterInvalidComment flags comment content that contains "--" or ends
	// with "-" (either would break the "-->" delimiter).
	ErrWriterInvalidComment = errors.New("invalid comment content")
	// ErrWriterInvalidPITarget flags a processing-instruction target that is not a
	// valid PITarget; ErrWriterInvalidPIContent flags PI content containing "?>".
	ErrWriterInvalidPITarget  = errors.New("invalid processing-instruction target")
	ErrWriterInvalidPIContent = errors.New("invalid processing-instruction content")
	// ErrWriterInvalidDTDNode flags a DTD node (element-content particle, entity,
	// element/attribute declaration) whose type/enum field holds an unrecognized
	// value, so it cannot be serialized.
	ErrWriterInvalidDTDNode = errors.New("invalid DTD node")
	// ErrUnsupportedNormalizationForm is returned by Writer.WriteTo when
	// Writer.Normalization was given a value outside the supported set
	// ("", "none", "NFC", "NFD", "NFKC", "NFKD"). The writer fails closed before
	// emitting any output byte instead of silently disabling normalization.
	// Match with errors.Is.
	ErrUnsupportedNormalizationForm = errors.New("unsupported normalization form")
	// ErrNetworkAccessForbidden is returned when an external resource (an
	// external DTD subset or external parsed entity) names a network URI —
	// an http, https, or ftp scheme — but the parser was configured to forbid
	// network access via Parser.AllowNetwork(false) (libxml2 XML_PARSE_NONET,
	// the default). helium has no dedicated network loader; every external load
	// goes through the configured fs.FS, so this guard refuses a network-scheme
	// name before it reaches a (possibly network-capable) caller-supplied FS.
	// Match with errors.Is.
	ErrNetworkAccessForbidden = errors.New("network access forbidden")
	errParserStopped          = errors.New("parser stopped")
	errNoCursor               = errors.New("parser has no input")
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
