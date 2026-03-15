// Package xslt3 implements an XSLT 3.0 processor.
package xslt3

import (
	"errors"
	"fmt"
)

// XSLT error codes per the W3C XSLT 3.0 specification.
const (
	errCodeXTSE0010 = "XTSE0010" // generic static error
	errCodeXTSE0020 = "XTSE0020" // reserved namespace
	errCodeXTSE0080 = "XTSE0080" // duplicate template name
	errCodeXTSE0090 = "XTSE0090" // unknown XSLT element
	errCodeXTSE0110 = "XTSE0110" // required attribute missing
	errCodeXTSE0120 = "XTSE0120" // unsupported version
	errCodeXTSE0130 = "XTSE0130" // duplicate global variable
	errCodeXTSE0165 = "XTSE0165" // invalid xpath expression
	errCodeXTSE0210 = "XTSE0210" // circular import
	errCodeXTSE0280 = "XTSE0280" // element not allowed in context
	errCodeXTSE0340 = "XTSE0340" // unknown attribute on XSLT element
	errCodeXTSE0350 = "XTSE0350" // value-of with content and select
	errCodeXTSE0500 = "XTSE0500" // invalid pattern
	errCodeXTSE0580 = "XTSE0580" // invalid attribute value template
	errCodeXTSE0620 = "XTSE0620" // cannot write to output
	errCodeXTSE0680 = "XTSE0680" // invalid xsl:number
	errCodeXTRE0540 = "XTRE0540" // attribute after child content
	errCodeXTDE0820 = "XTDE0820" // dynamic type error in template match
	errCodeXTDE0830 = "XTDE0830" // no matching template rule
	errCodeXTDE0835 = "XTDE0835" // xsl:message terminate=yes
	errCodeXTDE0045 = "XTDE0045" // evaluation error in AVT
	errCodeXTDE0060 = "XTDE0060" // invalid call-template target
	errCodeXTDE0410 = "XTDE0410" // duplicate parameter
	errCodeXTDE0160 = "XTDE0160" // multiple output documents same URI
	errCodeXTDE0430 = "XTDE0430" // variable type error
	errCodeXTDE0560 = "XTDE0560" // copy-of type error
	errCodeXTDE0700 = "XTDE0700" // sort key error
	errCodeXTDE1110 = "XTDE1110" // circular variable reference
	errCodeXTDE1140 = "XTDE1140" // sort key comparison error
	errCodeXTDE1170 = "XTDE1170" // key function lookup error
	errCodeXTSE3430 = "XTSE3430" // non-streamable expression in streaming context
	errCodeXTSE3441 = "XTSE3441" // streamability error in accumulator
	errCodeXTDE3630 = "XTDE3630" // source-document runtime error
	errCodeXTDE3635 = "XTDE3635" // source-document URI resolution error
)

// Sentinel errors for the xslt3 package.
var (
	ErrStaticError   = errors.New("xslt3: static error")
	ErrDynamicError  = errors.New("xslt3: dynamic error")
	ErrCircularRef   = errors.New("xslt3: circular reference")
	ErrNoTemplate    = errors.New("xslt3: no matching template")
	ErrTerminated    = errors.New("xslt3: terminated by xsl:message")
	ErrInvalidOutput = errors.New("xslt3: invalid output specification")
)

// XSLTError is a structured error with an XSLT error code.
type XSLTError struct {
	Code    string
	Message string
	Cause   error
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
	}
}

func dynamicError(code, format string, args ...any) *XSLTError {
	return &XSLTError{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}
