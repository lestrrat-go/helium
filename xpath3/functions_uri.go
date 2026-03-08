package xpath3

import (
	"context"
	"net/url"
	"strings"
)

func init() {
	registerFn("encode-for-uri", 1, 1, fnEncodeForURI)
	registerFn("iri-to-uri", 1, 1, fnIRIToURI)
	registerFn("escape-html-uri", 1, 1, fnEscapeHTMLURI)
}

func fnEncodeForURI(_ context.Context, args []Sequence) (Sequence, error) {
	s := seqToString(args[0])
	return SingleString(url.PathEscape(s)), nil
}

func fnIRIToURI(_ context.Context, args []Sequence) (Sequence, error) {
	s := seqToString(args[0])
	// Only escape non-ASCII and disallowed characters
	var b strings.Builder
	for _, r := range s {
		if r > 0x7E || r < 0x20 {
			b.WriteString(url.PathEscape(string(r)))
		} else {
			b.WriteRune(r)
		}
	}
	return SingleString(b.String()), nil
}

func fnEscapeHTMLURI(_ context.Context, args []Sequence) (Sequence, error) {
	s := seqToString(args[0])
	// Only escape non-ASCII characters
	var b strings.Builder
	for _, r := range s {
		if r > 0x7E {
			b.WriteString(url.PathEscape(string(r)))
		} else {
			b.WriteRune(r)
		}
	}
	return SingleString(b.String()), nil
}
