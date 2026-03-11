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
	s, err := coerceArgToStringOpt(args[0])
	if err != nil {
		return nil, err
	}
	return SingleString(encodeForURI(s)), nil
}

// coerceArgToStringOpt extracts a string from xs:string? argument, validating
// both cardinality (0 or 1) and type (string-derived, anyURI, untypedAtomic).
func coerceArgToStringOpt(seq Sequence) (string, error) {
	if len(seq) == 0 {
		return "", nil
	}
	if len(seq) > 1 {
		return "", &XPathError{Code: "XPTY0004", Message: "expected xs:string?, got sequence of length > 1"}
	}
	return coerceArgToString(seq)
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
	s, err := coerceArgToStringOpt(args[0])
	if err != nil {
		return nil, err
	}
	// Per XPath F&O 3.0 §7.4.5: escape characters that are not allowed in URIs.
	// Keep: unreserved, reserved, and already percent-encoded sequences.
	// Escape: space, <, >, ", {, }, |, \, ^, `, non-ASCII (>0x7E), and control chars (<0x20).
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '%' && i+2 < len(s) && isHexDigit(s[i+1]) && isHexDigit(s[i+2]) {
			// Already percent-encoded: pass through
			b.WriteByte(c)
			continue
		}
		if c > 0x7E || c <= 0x20 || c == '"' || c == '<' || c == '>' ||
			c == '{' || c == '}' || c == '|' || c == '\\' || c == '^' || c == '`' {
			fmt.Fprintf(&b, "%%%02X", c)
		} else {
			b.WriteByte(c)
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
		return SingleString(relative), nil
	}
	if err := validateIRI(base); err != nil {
		return nil, &XPathError{Code: "FORG0002", Message: "invalid base URI: " + base}
	}
	if err := validateIRI(relative); err != nil {
		return nil, &XPathError{Code: "FORG0002", Message: "invalid relative URI: " + relative}
	}

	// Check if relative URI is already absolute
	if idx := strings.Index(relative, ":"); idx > 0 && !strings.ContainsAny(relative[:idx], "/?#") {
		return SingleString(relative), nil
	}

	// Resolve IRI-aware: encode non-ASCII for Go's url.Parse, then decode back.
	// Track whether inputs had non-ASCII chars so we know whether to decode back.
	encodedBase := iriToURI(base)
	encodedRel := iriToURI(relative)
	hadNonASCII := encodedBase != base || encodedRel != relative

	// Save original scheme case before Go lowercases it
	origScheme := ""
	if idx := strings.Index(encodedBase, ":"); idx > 0 {
		origScheme = encodedBase[:idx]
	}

	baseURL, err := url.Parse(encodedBase)
	if err != nil {
		return nil, &XPathError{Code: "FORG0002", Message: "invalid base URI: " + base}
	}
	if baseURL.Scheme == "" {
		return nil, &XPathError{Code: "FORG0002", Message: "base URI is not absolute: " + base}
	}
	if baseURL.Fragment != "" {
		return nil, &XPathError{Code: "FORG0002", Message: "base URI must not contain a fragment: " + base}
	}
	relURL, err := url.Parse(encodedRel)
	if err != nil {
		return nil, &XPathError{Code: "FORG0002", Message: "invalid relative URI: " + relative}
	}

	resolved := baseURL.ResolveReference(relURL)

	result := resolved.String()
	// Restore original scheme case (Go lowercases it)
	if origScheme != "" && resolved.Scheme != origScheme {
		result = origScheme + result[len(resolved.Scheme):]
	}
	// Decode percent-encoded non-ASCII back to IRI form only if we encoded them
	if hadNonASCII {
		result = uriToIRI(result)
	}

	return SingleString(result), nil
}

// validateIRI checks that a string is a valid IRI reference.
// Rejects characters that are not allowed in IRIs (like raw spaces).
func validateIRI(s string) error {
	if err := validatePercentEncoding(s); err != nil {
		return err
	}
	for _, r := range s {
		if r == ' ' || r < 0x20 || (r > 0x7E && r < 0xA0) {
			return fmt.Errorf("invalid IRI character U+%04X", r)
		}
	}
	return nil
}

// iriToURI percent-encodes non-ASCII characters for use with Go's url.Parse.
func iriToURI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c > 0x7E {
			fmt.Fprintf(&b, "%%%02X", c)
		} else {
			b.WriteByte(c)
		}
	}
	return b.String()
}

// uriToIRI decodes percent-encoded non-ASCII characters back to their IRI form.
func uriToIRI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && i+2 < len(s) {
			hi := unhexByte(s[i+1])
			lo := unhexByte(s[i+2])
			if hi >= 0 && lo >= 0 {
				c := byte(hi<<4 | lo)
				if c > 0x7E {
					b.WriteByte(c)
					i += 2
					continue
				}
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func unhexByte(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c - 'a' + 10)
	case c >= 'A' && c <= 'F':
		return int(c - 'A' + 10)
	default:
		return -1
	}
}


func fnEscapeHTMLURI(_ context.Context, args []Sequence) (Sequence, error) {
	s, err := coerceArgToStringOpt(args[0])
	if err != nil {
		return nil, err
	}
	// Per XPath F&O: escape non-ASCII and control characters, keep printable ASCII as-is
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c > 0x7E || c < 0x20 {
			fmt.Fprintf(&b, "%%%02X", c)
		} else {
			b.WriteByte(c)
		}
	}
	return SingleString(b.String()), nil
}
