package xpath3

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

func init() {
	registerFn("encode-for-uri", 1, 1, fnEncodeForURI)
	registerFn("iri-to-uri", 1, 1, fnIRIToURI)
	registerFn("escape-html-uri", 1, 1, fnEscapeHTMLURI)
	registerFn("resolve-uri", 1, 2, fnResolveURI)
}

func fnEncodeForURI(_ context.Context, args []Sequence) (Sequence, error) {
	s := seqToString(args[0])
	return SingleString(encodeForURI(s)), nil
}

// encodeForURI percent-encodes all characters except unreserved characters
// as defined by RFC 3986: A-Z a-z 0-9 - _ . ~
func encodeForURI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if isUnreservedChar(c) {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}

// isUnreservedChar returns true if the byte is an unreserved character per RFC 3986.
func isUnreservedChar(c byte) bool {
	return (c >= 'A' && c <= 'Z') ||
		(c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9') ||
		c == '-' || c == '_' || c == '.' || c == '~'
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

func fnResolveURI(_ context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 {
		return nil, nil
	}
	relative := seqToString(args[0])
	if relative == "" {
		if len(args) >= 2 {
			return SingleString(seqToString(args[1])), nil
		}
		return SingleString(""), nil
	}
	base := ""
	if len(args) >= 2 {
		base = seqToString(args[1])
	}
	if base == "" {
		// No base URI — just return the relative URI if it's absolute
		return SingleString(relative), nil
	}
	baseURL, err := url.Parse(base)
	if err != nil {
		return nil, &XPathError{Code: "FORG0002", Message: "invalid base URI: " + base}
	}
	relURL, err := url.Parse(relative)
	if err != nil {
		return nil, &XPathError{Code: "FORG0002", Message: "invalid relative URI: " + relative}
	}
	resolved := baseURL.ResolveReference(relURL)
	return SingleString(resolved.String()), nil
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
