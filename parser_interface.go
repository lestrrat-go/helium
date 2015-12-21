package helium

import (
	"errors"

	"github.com/lestrrat/go-strcursor"
	"github.com/lestrrat/helium/sax"
)

type parserState int

const (
	psEOF parserState = iota - 1
	psStart
	psPI
	psContent
	psEpilogue
	psCDATA
	psDTD
)

const MaxNameLength = 50000

var (
	ErrAmpersandRequired            = errors.New("'&' was required here")
	ErrDocTypeNameRequired          = errors.New("doctype name required")
	ErrDocTypeNotFinished           = errors.New("doctype not finished")
	ErrEOF                          = errors.New("end of file reached")
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
	ErrNameTooLong                  = errors.New("name is too long")
	ErrNameRequired                 = errors.New("name is required")
	ErrOpenParenRequired            = errors.New("'(' is required")
	ErrPCDATARequired               = errors.New("'#PCDATA' required")
	ErrPercentRequired              = errors.New("'%' is required")
	ErrPrematureEOF                 = errors.New("end of document reached")
	ErrSemicolonRequired            = errors.New("';' is required")
	ErrSpaceRequired                = errors.New("space required")
	ErrStartTagRequired             = errors.New("start tag expected, '<' not found")
)

type ErrAttrNotFound struct {
	Token string
}

type ErrParseError struct {
	Column     int
	Err        error
	Location   int
	Line       string
	LineNumber int
}

type Parser struct {
	sax sax.Handler
}

type ParsedElement struct {
	local      string
	prefix     string
	uri        string
	value      string
	attributes []sax.ParsedAttribute
	next       *ParsedElement
}

type ParsedAttribute struct {
	local  string
	prefix string
	value  string
}

const (
	inSubsetNo = iota
	inInternalSubset
	inExternalSubset
)

type parserCtx struct {
	options    int
	encoding   string
	cursor     *strcursor.Cursor
	nbread     int
	instate    parserState
	lineno     int
	remain     int
	sax        sax.Handler
	standalone DocumentStandaloneType
	inSubset   int
	version    string

	doc        *Document
	userData   interface{}
	element    *ParsedElement
	elemidx    int
	nbentities int
}
