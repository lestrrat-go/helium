package helium

import "errors"

type parserState int

const (
	psEOF parserState = iota - 1
	psStart
	psPI
	psContent
	psEpilogue
	psCDATA
)

const MaxNameLength = 50000

var (
	ErrEqualSignRequired   = errors.New("'=' was required here")
	ErrHyphenInComment     = errors.New("'--' not allowed in comment")
	ErrInvalidChar         = errors.New("invalid char")
	ErrInvalidEncodingName = errors.New("invalid encoding name")
	ErrInvalidName         = errors.New("invalid xml name")
	ErrInvalidVersionNum   = errors.New("invalid version")
	ErrLtSlashRequired     = errors.New("'</' is required")
	ErrNameTooLong         = errors.New("name is too long")
	ErrPrematureEOF        = errors.New("end of document reached")
	ErrSpaceRequired       = errors.New("space required")
	ErrStartTagRequired    = errors.New("start tag expected, '<' not found")
)

type ErrAttrNotFound struct {
	Token string
}

type ErrParseError struct {
	Err      error
	Location int
	Line     int
	Column   int
}

type SAXHandler interface {
	StartDocument(interface{}) error
	EndDocument(interface{}) error
	ProcessingInstruction(interface{}, string, string) error
	StartElement(interface{}, *ParsedElement) error
	EndElement(interface{}, *ParsedElement) error
	Characters(interface{}, []byte) error
	CDATABlock(interface{}, []byte) error
	Comment(interface{}, []byte) error
}

type Parser struct{}

type ParsedElement struct {
	local      string
	prefix     string
	value      string
	attributes []ParsedAttribute
	next       *ParsedElement
}

type ParsedAttribute struct {
	local  string
	prefix string
	value  string
}

type parserCtx struct {
	options    int
	encoding   string
	idx        int
	input      []byte
	inputsz    int
	instate    parserState
	lineno     int
	remain     int
	sax        SAXHandler
	standalone bool
	version    string

	doc      *Document
	userData interface{}
	element  *ParsedElement
	elemidx  int
}
