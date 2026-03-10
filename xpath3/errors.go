package xpath3

import "errors"

// Sentinel errors for the xpath3 package.
var (
	ErrNotNodeSet          = errors.New("xpath3: result is not a node-set")
	ErrRecursionLimit      = errors.New("xpath3: recursion limit exceeded")
	ErrOpLimit             = errors.New("xpath3: operation limit exceeded")
	ErrUnknownFunction     = errors.New("xpath3: unknown function")
	ErrUnknownFunctionNS   = errors.New("xpath3: unknown function namespace prefix")
	ErrUnsupportedExpr     = errors.New("xpath3: unsupported expression type")
	ErrUndefinedVariable   = errors.New("xpath3: undefined variable")
	ErrTypeMismatch        = errors.New("xpath3: type mismatch")
	ErrArityMismatch       = errors.New("xpath3: arity mismatch")
	ErrUnexpectedToken     = errors.New("xpath3: unexpected token")
	ErrUnexpectedChar      = errors.New("xpath3: unexpected character")
	ErrUnterminatedString  = errors.New("xpath3: unterminated string")
	ErrUnknownAxis         = errors.New("xpath3: unknown axis")
	ErrExpectedToken       = errors.New("xpath3: expected token")
	ErrExprTooDeep         = errors.New("xpath3: expression nesting too deep")
	ErrUnionNotNodeSet     = errors.New("xpath3: union operands must be node-sets")
	ErrPathNotNodeSet      = errors.New("xpath3: path expression requires node-set")
	ErrUnsupportedBinaryOp = errors.New("xpath3: unsupported binary operator")
	ErrNodeSetLimit        = errors.New("xpath3: node-set length limit exceeded")
)

// XPathError is a structured error with an XPath error code.
// Codes are stored without namespace prefix (e.g. "XPTY0004", not "err:XPTY0004").
type XPathError struct {
	Code    string // e.g. "FOER0000", "XPTY0004"
	Message string
}

func (e *XPathError) Error() string {
	return e.Code + ": " + e.Message
}

// Is supports errors.Is matching by code.
func (e *XPathError) Is(target error) bool {
	if t, ok := target.(*XPathError); ok {
		return e.Code == t.Code
	}
	return false
}
