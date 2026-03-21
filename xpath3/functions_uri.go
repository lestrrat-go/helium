package xpath3

import (
	"context"
	"fmt"
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
		return "", &XPathError{Code: errCodeXPTY0004, Message: "expected xs:string?, got sequence of length > 1"}
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

func fnResolveURI(ctx context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 {
		return nil, nil
	}
	relative, err := coerceArgToString(args[0])
	if err != nil {
		return nil, err
	}
	if relative == "" {
		if len(args) >= 2 {
			base, err := coerceArgToString(args[1])
			if err != nil {
				return nil, err
			}
			return SingleString(base), nil
		}
		// 1-arg with empty relative: return static base URI
		if cfg := getEvalConfig(ctx); cfg != nil && cfg.baseURI != "" {
			return SingleString(cfg.baseURI), nil
		}
		return SingleString(""), nil
	}
	base := ""
	if len(args) >= 2 {
		base, err = coerceArgToString(args[1])
		if err != nil {
			return nil, err
		}
	}
	// 1-arg form: use static base URI from context
	if base == "" {
		if cfg := getEvalConfig(ctx); cfg != nil {
			base = cfg.baseURI
		}
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

	// Convert absolute file paths to file: URIs so that resolve-uri works
	// correctly with file system paths (e.g. from static-base-uri()).
	// This applies to both context-derived bases and explicit arguments,
	// since an absolute file path is not a valid URI base without a scheme.
	if strings.HasPrefix(base, "/") && !strings.Contains(base, "://") {
		base = "file://" + base
	}
	parsedBase, err := parseURIReference(base)
	if err != nil {
		return nil, &XPathError{Code: "FORG0002", Message: "invalid base URI: " + base}
	}
	if parsedBase.Scheme == "" {
		return nil, &XPathError{Code: "FORG0002", Message: "base URI is not absolute: " + base}
	}
	if parsedBase.Fragment != "" {
		return nil, &XPathError{Code: "FORG0002", Message: "base URI must not contain a fragment: " + base}
	}
	_, err = parseURIReference(relative)
	if err != nil {
		return nil, &XPathError{Code: "FORG0002", Message: "invalid relative URI: " + relative}
	}
	result, err := resolveURIReference(base, relative)
	if err != nil {
		return nil, &XPathError{Code: "FORG0002", Message: "invalid relative URI: " + relative}
	}
	return SingleString(result), nil
}

// validateIRI checks that a string is a valid IRI reference.
// Rejects characters that are not allowed in IRIs (like raw spaces).
func validateIRI(s string) error {
	if err := validatePercentEncoding(s); err != nil {
		return err
	}
	// A URI reference may contain at most one '#' (fragment separator).
	if idx := strings.IndexByte(s, '#'); idx >= 0 {
		if strings.IndexByte(s[idx+1:], '#') >= 0 {
			return fmt.Errorf("invalid IRI: multiple '#' characters")
		}
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
