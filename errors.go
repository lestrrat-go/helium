package helium

import (
	"errors"
	"fmt"
)

// ErrorLevel represents the severity of a parse error.
type ErrorLevel int

const (
	ErrorLevelNone    ErrorLevel = iota
	ErrorLevelWarning
	ErrorLevelError
	ErrorLevelFatal
)

var (
	ErrNilNode            = errors.New("nil node")
	ErrInvalidOperation   = errors.New("operation cannot be performed")
	ErrDuplicateAttribute = errors.New("duplicate attribute")
	ErrEntityBoundary     = errors.New("entity boundary violation")
	errParserStopped      = errors.New("parser stopped")
)

type ErrUnimplemented struct {
	target string
}

type ErrDTDDupToken struct {
	Name string
}

type ErrAttrNotFound struct {
	Token string
}

type ErrParseError struct {
	Column     int
	Err        error
	File       string
	Level      ErrorLevel
	Line       string
	LineNumber int
}

var (
	ErrAmpersandRequired            = errors.New("'&' was required here")
	ErrAttrListNotFinished          = errors.New("attrlist must finish with a ')'")
	ErrAttrListNotStarted           = errors.New("attrlist must start with a '('")
	ErrAttributeNameRequired        = errors.New("attribute namewas required here (ATTLIST)")
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
	ErrInvalidXMLDecl               = errors.New("invalid XML declration")
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

func (e ErrParseError) Unwrap() error {
	return e.Err
}

func (e ErrUnimplemented) Error() string {
	return "unimplemented method: '" + e.target + "'"
}

func (e ErrDTDDupToken) Error() string {
	return "standalone: attribute enumeration value token " + e.Name + " duplicated"
}
