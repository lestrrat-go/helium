package xslt3

import (
	"errors"
	"fmt"
)

// Sentinel errors for the xslt3 package.
var (
	ErrStaticError   = errors.New("xslt3: static error")
	ErrDynamicError  = errors.New("xslt3: dynamic error")
	ErrCircularRef   = errors.New("xslt3: circular reference")
	ErrNoTemplate    = errors.New("xslt3: no matching template")
	ErrTerminated    = errors.New("xslt3: terminated by xsl:message")
	ErrInvalidOutput = errors.New("xslt3: invalid output specification")

	errNilStylesheet  = errors.New("xslt3: nil stylesheet")
	errNilDocument    = errors.New("xslt3: nil document")
	errZeroInvocation = errors.New("xslt3: unconfigured invocation")
)

// XSLTError is a structured error with an XSLT error code.
type XSLTError struct {
	Code    string
	Message string
	Cause   error
	// Value holds the sequence associated with the error. For xsl:message
	// terminate="yes", this carries the message body content (nodes/atomics)
	// so that $err:value in xsl:catch can expose it.
	Value any
}

func (e *XSLTError) Error() string {
	if e.Code != "" {
		return e.Code + ": " + e.Message
	}
	return e.Message
}

func (e *XSLTError) Unwrap() error {
	return e.Cause
}

func staticError(code, format string, args ...any) *XSLTError {
	return &XSLTError{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
		Cause:   ErrStaticError,
	}
}

// staticErrorCause builds a static error whose Cause preserves both
// [ErrStaticError] and the underlying cause, so that callers can match the
// specific cause (e.g. [ErrResourceTooLarge]) via errors.Is while
// errors.Is(err, ErrStaticError) continues to hold. Use this at compile-time
// sites where a distinguishable sentinel (documented in the public API) must
// remain observable through the XSLTError wrapper.
func staticErrorCause(code string, cause error, format string, args ...any) *XSLTError {
	return &XSLTError{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
		Cause:   errors.Join(ErrStaticError, cause),
	}
}

func dynamicError(code, format string, args ...any) *XSLTError {
	return &XSLTError{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
		Cause:   ErrDynamicError,
	}
}

// dynamicErrorCause builds a dynamic error whose Cause preserves both
// [ErrDynamicError] and the underlying cause, so that callers can match the
// specific cause (e.g. [ErrResourceTooLarge]) via errors.Is while
// errors.Is(err, ErrDynamicError) continues to hold. Use this at sites where a
// distinguishable sentinel (documented in the public API) must remain
// observable through the XSLTError wrapper.
func dynamicErrorCause(code string, cause error, format string, args ...any) *XSLTError {
	return &XSLTError{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
		Cause:   errors.Join(ErrDynamicError, cause),
	}
}

// isXSLTError returns true if err is an XSLTError with the given code.
func isXSLTError(err error, code string) bool {
	if xe, ok := errors.AsType[*XSLTError](err); ok {
		return xe.Code == code
	}
	return false
}
