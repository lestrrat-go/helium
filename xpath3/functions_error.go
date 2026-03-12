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
	code := errCodeFOER0000
	msg := "error() called"
	if len(args) > 0 && len(args[0]) > 0 {
		a, err := AtomizeItem(args[0][0])
		if err == nil {
			s, _ := atomicToString(a)
			if s != "" {
				code = s
			}
		}
	}
	if len(args) > 1 {
		var err error
		msg, err = coerceArgToStringRequired(args[1])
		if err != nil {
			return nil, err
		}
	}
	return nil, &XPathError{Code: code, Message: msg}
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
