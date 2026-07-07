package xpath3

import (
	"errors"

	"github.com/lestrrat-go/helium/internal/lexicon"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
)

const errCodeXPST0003 = "XPST0003"
const errCodeXPDY0002 = "XPDY0002"
const errCodeXPDY0050 = "XPDY0050"
const errCodeXPST0080 = "XPST0080"
const errCodeXPST0081 = "XPST0081"
const errCodeFOAR0001 = "FOAR0001"
const errCodeFOAR0002 = "FOAR0002"
const errCodeFOAP0001 = "FOAP0001"
const errCodeFOCA0002 = "FOCA0002"
const errCodeFOCH0002 = "FOCH0002"
const errCodeFOCH0003 = "FOCH0003"
const errCodeFODC0001 = "FODC0001"
const errCodeFODC0002 = "FODC0002"
const errCodeFODC0004 = "FODC0004"
const errCodeFODC0005 = "FODC0005"
const errCodeFODC0006 = "FODC0006"
const errCodeFODT0002 = "FODT0002"
const errCodeFODF1280 = "FODF1280"
const errCodeFODF1310 = "FODF1310"
const errCodeFOFD1340 = "FOFD1340"
const errCodeFOFD1350 = "FOFD1350"
const errCodeFOAY0001 = "FOAY0001"
const errCodeFOTY0012 = "FOTY0012"
const errCodeFOTY0013 = "FOTY0013"
const errCodeFONS0004 = "FONS0004"
const errCodeFORG0001 = "FORG0001"
const errCodeFORG0006 = "FORG0006"
const errCodeFOER0000 = "FOER0000"
const errCodeFORX0001 = "FORX0001"
const errCodeFORX0002 = "FORX0002"
const errCodeFORX0003 = "FORX0003"
const errCodeFORX0004 = "FORX0004"
const errCodeFOTY0014 = "FOTY0014"
const errCodeFOJS0001 = "FOJS0001"
const errCodeFOJS0003 = "FOJS0003"
const errCodeFOJS0005 = "FOJS0005"
const errCodeFOJS0006 = "FOJS0006"
const errCodeFOJS0007 = "FOJS0007"
const errCodeFORG0002 = "FORG0002"
const errCodeSENR0001 = "SENR0001"

// Error message constants reused across the package.
const (
	errMsgContextItemAbsent                = "context item is absent"
	errMsgParseXMLFragmentMalformedTextDec = "parse-xml-fragment: malformed text declaration"
)

// Sentinel errors for the xpath3 package.
var (
	ErrNotNodeSet               = errors.New("xpath3: result is not a node-set")
	ErrRecursionLimit           = errors.New("xpath3: recursion limit exceeded")
	ErrOpLimit                  = errors.New("xpath3: operation limit exceeded")
	ErrUnknownFunction          = errors.New("xpath3: unknown function")
	ErrUnknownFunctionNamespace = errors.New("xpath3: unknown function namespace prefix")
	ErrUnsupportedExpr          = errors.New("xpath3: unsupported expression type")
	ErrUndefinedVariable        = errors.New("xpath3: undefined variable")
	ErrTypeMismatch             = errors.New("xpath3: type mismatch")
	ErrArityMismatch            = errors.New("xpath3: arity mismatch")
	ErrUnexpectedToken          = errors.New("xpath3: unexpected token")
	ErrUnexpectedChar           = errors.New("xpath3: unexpected character")
	ErrUnterminatedString       = errors.New("xpath3: unterminated string")
	ErrUnknownAxis              = errors.New("xpath3: unknown axis")
	ErrExpectedToken            = errors.New("xpath3: expected token")
	ErrExprTooDeep              = errors.New("xpath3: expression nesting too deep")
	ErrUnionNotNodeSet          = errors.New("xpath3: union operands must be node-sets")
	ErrPathNotNodeSet           = errors.New("xpath3: path expression requires node-set")
	ErrUnsupportedBinaryOp      = errors.New("xpath3: unsupported binary operator")
	// ErrNodeSetLimit is returned when a node-set exceeds the maximum length.
	// Aliased from internal/xpath so errors.Is works end-to-end.
	ErrNodeSetLimit = ixpath.ErrNodeSetLimit
	// ErrRegexMatchLimit is returned by [Regex.EachSubmatchIndex] when a
	// leading-context pattern (a multi-line ^, \A, \b, ...) — which cannot be
	// streamed incrementally on the RE2 engine and must be materialized in one
	// pass — produces more matches than the safe full-context allocation ceiling
	// allows. It bounds the up-front materialization so a near-empty-matching
	// pattern over a large input cannot allocate a match record per input
	// position; callers should treat it as a resource-exhaustion condition.
	ErrRegexMatchLimit = errors.New("xpath3: regex full-context match count exceeds the safe allocation ceiling")
)

// XPathError is a structured error with an XPath error code.
// Codes are stored without namespace prefix (e.g. "XPTY0004", not "err:XPTY0004").
type XPathError struct {
	Code      string // e.g. "XPTY0004", "FOER0000" (without err: prefix)
	Message   string
	codeQName QNameValue
}

func (e *XPathError) Error() string {
	if e == nil {
		return "<nil XPathError>"
	}
	code := e.displayCode()
	if code == "" {
		return e.Message
	}
	return code + ": " + e.Message
}

// Is supports errors.Is matching by code.
func (e *XPathError) Is(target error) bool {
	if e == nil {
		return false
	}
	if t, ok := target.(*XPathError); ok {
		actual := e.qname()
		want := t.qname()
		if want.Local == "" {
			return false
		}
		if want.URI != "" {
			return actual.URI == want.URI && actual.Local == want.Local
		}
		return actual.Local == want.Local
	}
	return false
}

// CodeQName returns the error code as a QNameValue.
func (e *XPathError) CodeQName() QNameValue {
	return e.qname()
}

func (e *XPathError) qname() QNameValue {
	if e == nil {
		return QNameValue{}
	}
	if e.codeQName.Local != "" {
		return e.codeQName
	}
	if e.Code == "" {
		return QNameValue{}
	}
	return QNameValue{Prefix: lexicon.PrefixErr, URI: NSErr, Local: e.Code}
}

func (e *XPathError) displayCode() string {
	qv := e.qname()
	if qv.Local == "" {
		return ""
	}
	if qv.URI == "" || qv.URI == NSErr {
		return qv.Local
	}
	if qv.Prefix != "" {
		return qv.Prefix + ":" + qv.Local
	}
	return "Q{" + qv.URI + "}" + qv.Local
}
