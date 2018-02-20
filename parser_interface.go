package helium

import (
	"errors"
	"io"

	"github.com/lestrrat-go/helium/sax"
)

type parserState int

const (
	psEOF parserState = iota - 1
	psStart
	psPI
	psContent
	psPrologue
	psEpilogue
	psCDATA
	psDTD
	psEntityDecl
	psAttributeValue
	psComment
	psStartTag
	psEndTag
	psSystemLiteral
	psPublicLiteral
	psEntityValue
	psIgnore
	psMisc
)

const MaxNameLength = 50000

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
	ErrSpaceRequired                = errors.New("space required")
	ErrStartTagRequired             = errors.New("start tag expected, '<' not found")
	ErrValueRequired                = errors.New("value required")
)

type ErrDTDDupToken struct {
	Name string
}

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
	sax sax.SAX2Handler
}

const (
	notInSubset = iota
	inInternalSubset
	inExternalSubset
)

type parserCtx struct {
	options ParseOption
	// ctx.encoding contains the explicit encoding. ctx.detectedEncoding
	// contains the encoding as detected by inspecting BOM, etc.
	// It is important to differentiate between the two, otherwise
	// we will not be able to reconstruct
	// <?xml version="1.0"?> vs <?xml version="1.0" encoding="utf-8"?>
	encoding          string
	detectedEncoding  string
	in                io.Reader
	nbread            int
	instate           parserState
	keepBlanks        bool
	remain            int
	replaceEntities   bool
	sax               sax.SAX2Handler
	space             int
	standalone        DocumentStandaloneType
	hasExternalSubset bool
	inSubset          int
	intSubName        string
	external          bool // true if parsing external DTDs
	extSubSystem      string
	extSubURI         string
	version           string
	attsSpecial       map[string]AttributeType
	attsDefault       map[string]map[string]*Attribute
	valid             bool
	hasPERefs         bool
	pedantic          bool
	wellFormed        bool
	depth             int
	loadsubset        LoadSubsetOption
	elem              *Element // current context element

	nsTab      nsStack
	doc        *Document
	userData   interface{}
	nodeTab    nodeStack
	elemidx    int
	nbentities int
	inputTab   inputStack
}

type SubstitutionType int

const (
	SubstituteNone SubstitutionType = iota
	SubstituteRef
	SubstitutePERef
	SubstituteBoth
)
