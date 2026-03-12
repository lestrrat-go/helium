package xpath3

import (
	"context"
	"fmt"
	"io"
	"os"
)

func init() {
	registerFn("error", 0, 3, fnError)
	registerFn("trace", 1, 2, fnTrace)
}

func fnError(_ context.Context, args []Sequence) (Sequence, error) {
	code := QNameValue{Prefix: "err", URI: NSErr, Local: errCodeFOER0000}
	msg := "error() called"
	if len(args) > 0 {
		qv, hasCode, err := coerceErrorCode(args[0])
		if err != nil {
			return nil, err
		}
		if hasCode {
			code = qv
		}
	}
	if len(args) > 1 {
		var err error
		msg, err = coerceArgToStringRequired(args[1])
		if err != nil {
			return nil, err
		}
	}
	return nil, &XPathError{Code: code.Local, Message: msg, codeQName: code}
}

func coerceErrorCode(seq Sequence) (QNameValue, bool, error) {
	switch len(seq) {
	case 0:
		return QNameValue{}, false, nil
	case 1:
	default:
		return QNameValue{}, false, &XPathError{Code: errCodeXPTY0004, Message: "fn:error code argument must be xs:QName?"}
	}

	a, err := AtomizeItem(seq[0])
	if err != nil {
		return QNameValue{}, false, err
	}
	if a.TypeName != TypeQName {
		return QNameValue{}, false, &XPathError{Code: errCodeXPTY0004, Message: "fn:error code argument must be xs:QName?"}
	}
	return a.QNameVal(), true, nil
}

// traceWriter is the destination for fn:trace output. Default is stderr.
// Can be overridden for testing.
var traceWriter io.Writer = os.Stderr

func fnTrace(_ context.Context, args []Sequence) (Sequence, error) {
	label := ""
	if len(args) > 1 {
		label = seqToString(args[1])
	}
	if label != "" {
		_, _ = fmt.Fprintf(traceWriter, "[trace] %s: ", label)
	} else {
		_, _ = fmt.Fprint(traceWriter, "[trace] ")
	}
	for i, item := range args[0] {
		if i > 0 {
			_, _ = fmt.Fprint(traceWriter, ", ")
		}
		a, err := AtomizeItem(item)
		if err != nil {
			_, _ = fmt.Fprintf(traceWriter, "<%T>", item)
		} else {
			s, _ := atomicToString(a)
			_, _ = fmt.Fprint(traceWriter, s)
		}
	}
	_, _ = fmt.Fprintln(traceWriter)
	return args[0], nil
}
