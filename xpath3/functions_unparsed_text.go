package xpath3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"

	iencoding "github.com/lestrrat-go/helium/internal/encoding"
)

func init() {
	registerFn("unparsed-text", 1, 2, fnUnparsedText)
	registerFn("unparsed-text-available", 1, 2, fnUnparsedTextAvailable)
	registerFn("unparsed-text-lines", 1, 2, fnUnparsedTextLines)
}

func fnUnparsedText(ctx context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 {
		return nil, nil
	}
	href := seqToString(args[0])
	encoding := ""
	if len(args) > 1 {
		encoding = seqToString(args[1])
	}

	text, err := loadUnparsedText(ctx, href, encoding)
	if err != nil {
		return nil, err
	}
	return SingleString(text), nil
}

func fnUnparsedTextAvailable(ctx context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 {
		return SingleBoolean(false), nil
	}
	href := seqToString(args[0])
	encoding := ""
	if len(args) > 1 {
		encoding = seqToString(args[1])
	}

	_, err := loadUnparsedText(ctx, href, encoding)
	return SingleBoolean(err == nil), nil
}

func fnUnparsedTextLines(ctx context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 {
		return nil, nil
	}
	href := seqToString(args[0])
	encoding := ""
	if len(args) > 1 {
		encoding = seqToString(args[1])
	}

	text, err := loadUnparsedText(ctx, href, encoding)
	if err != nil {
		return nil, err
	}

	lines := splitTextLines(text)
	result := make(Sequence, len(lines))
	for i, line := range lines {
		result[i] = AtomicValue{TypeName: TypeString, Value: line}
	}
	return result, nil
}

// splitTextLines splits text into lines per the XPath spec.
// Line endings are normalized: CR+LF → LF, CR → LF.
// The final line ending (if any) is not included as an empty trailing line.
func splitTextLines(text string) []string {
	// First normalize line endings: CR+LF → LF, standalone CR → LF
	var b strings.Builder
	b.Grow(len(text))
	for i := 0; i < len(text); i++ {
		if text[i] == '\r' {
			b.WriteByte('\n')
			if i+1 < len(text) && text[i+1] == '\n' {
				i++ // skip the LF in CR+LF
			}
		} else {
			b.WriteByte(text[i])
		}
	}
	normalized := b.String()

	// Split on LF. Per spec, a trailing newline does not produce an extra empty line.
	if normalized == "" {
		return []string{""}
	}
	lines := strings.Split(normalized, "\n")
	// Remove trailing empty element if text ended with newline
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// loadUnparsedText resolves a URI and returns its text content.
func loadUnparsedText(ctx context.Context, href, encoding string) (string, error) {
	resolvedURI, err := resolveUnparsedTextURI(ctx, href)
	if err != nil {
		return "", err
	}

	data, err := readUnparsedTextURI(ctx, resolvedURI)
	if err != nil {
		return "", &XPathError{Code: "FOUT1170", Message: fmt.Sprintf("fn:unparsed-text: cannot retrieve resource: %v", err)}
	}

	text, err := decodeUnparsedText(data, encoding)
	if err != nil {
		return "", err
	}

	if err := validateXMLChars(text); err != nil {
		return "", err
	}

	return text, nil
}

// resolveUnparsedTextURI validates and resolves the href URI.
func resolveUnparsedTextURI(ctx context.Context, href string) (string, error) {
	// Reject URIs with fragments
	if strings.Contains(href, "#") {
		return "", &XPathError{Code: "FOUT1170", Message: "fn:unparsed-text: URI must not contain a fragment identifier"}
	}

	// Empty href → use static base URI
	if href == "" {
		ec := getFnContext(ctx)
		if ec != nil && ec.baseURI != "" {
			return ec.baseURI, nil
		}
		return "", &XPathError{Code: "FOUT1170", Message: "fn:unparsed-text: empty href and no base URI available"}
	}

	// Validate the URI
	parsed, err := url.Parse(href)
	if err != nil {
		return "", &XPathError{Code: "FOUT1170", Message: fmt.Sprintf("fn:unparsed-text: invalid URI: %s", href)}
	}

	// Reject invalid %-encoding
	if err := validatePercentEncoding(href); err != nil {
		return "", &XPathError{Code: "FOUT1170", Message: fmt.Sprintf("fn:unparsed-text: invalid URI: %s", err)}
	}

	// Reject URIs with only a scheme but no valid structure like ":/"
	if parsed.Scheme == "" && strings.HasPrefix(href, ":/") {
		return "", &XPathError{Code: "FOUT1170", Message: fmt.Sprintf("fn:unparsed-text: invalid URI: %s", href)}
	}

	// Reject Windows-style paths like "C:\..."
	if len(parsed.Scheme) == 1 && parsed.Scheme[0] >= 'A' && parsed.Scheme[0] <= 'Z' {
		return "", &XPathError{Code: "FOUT1170", Message: fmt.Sprintf("fn:unparsed-text: unsupported URI scheme: %s", href)}
	}

	// If it's an absolute URI with a scheme, use it directly
	if parsed.Scheme != "" {
		// Reject unknown/unsupported schemes
		switch parsed.Scheme {
		case "file", "http", "https":
			// supported
		default:
			return "", &XPathError{Code: "FOUT1170", Message: fmt.Sprintf("fn:unparsed-text: unsupported URI scheme: %s", parsed.Scheme)}
		}
		return href, nil
	}

	// Relative URI — resolve against base URI
	ec := getFnContext(ctx)
	if ec != nil && ec.baseURI != "" {
		baseURL, err := url.Parse(ec.baseURI)
		if err == nil {
			ref, err := url.Parse(href)
			if err == nil {
				resolved := baseURL.ResolveReference(ref)
				return resolved.String(), nil
			}
		}
	}

	return "", &XPathError{Code: "FOUT1170", Message: fmt.Sprintf("fn:unparsed-text: cannot resolve relative URI without base URI: %s", href)}
}

// validatePercentEncoding checks that %-encoded sequences are valid.
func validatePercentEncoding(uri string) error {
	for i := 0; i < len(uri); i++ {
		if uri[i] == '%' {
			if i+2 >= len(uri) {
				return fmt.Errorf("incomplete percent-encoding at position %d", i)
			}
			if !isHexDigit(uri[i+1]) || !isHexDigit(uri[i+2]) {
				return fmt.Errorf("invalid percent-encoding %%%c%c", uri[i+1], uri[i+2])
			}
			i += 2
		}
	}
	return nil
}

func isHexDigit(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}

// readUnparsedTextURI reads content from the resolved URI.
func readUnparsedTextURI(ctx context.Context, uri string) ([]byte, error) {
	ec := getFnContext(ctx)

	// Check for custom URI resolver first
	if ec != nil && ec.uriResolver != nil {
		rc, err := ec.uriResolver.ResolveURI(uri)
		if err != nil {
			return nil, err
		}
		defer func() { _ = rc.Close() }()
		return io.ReadAll(rc)
	}

	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	// HTTP/HTTPS: use the configured HTTP client
	if parsed.Scheme == "http" || parsed.Scheme == "https" {
		if ec == nil || ec.httpClient == nil {
			return nil, fmt.Errorf("no HTTP client configured for URI: %s", uri)
		}
		resp, err := ec.httpClient.Get(uri)
		if err != nil {
			return nil, err
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("HTTP %d for %s", resp.StatusCode, uri)
		}
		return io.ReadAll(resp.Body)
	}

	switch parsed.Scheme {
	case "file":
		return os.ReadFile(parsed.Path)
	case "":
		return os.ReadFile(uri)
	default:
		return nil, fmt.Errorf("unsupported URI scheme: %s", parsed.Scheme)
	}
}

// decodeUnparsedText decodes raw bytes to a string using BOM detection or
// the specified encoding.
func decodeUnparsedText(data []byte, encoding string) (string, error) {
	// Validate encoding name if provided
	if encoding != "" {
		enc := iencoding.Load(encoding)
		if enc == nil {
			return "", &XPathError{Code: "FOUT1190", Message: fmt.Sprintf("fn:unparsed-text: unsupported encoding: %s", encoding)}
		}
	}

	// BOM detection
	detectedEncoding := ""
	cleanData := data
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		detectedEncoding = "utf-8"
		cleanData = data[3:]
	} else if len(data) >= 2 && data[0] == 0xFF && data[1] == 0xFE {
		detectedEncoding = "utf-16le"
		cleanData = data[2:]
	} else if len(data) >= 2 && data[0] == 0xFE && data[1] == 0xFF {
		detectedEncoding = "utf-16be"
		cleanData = data[2:]
	}

	// Determine effective encoding
	effectiveEncoding := ""
	if encoding != "" {
		// User-specified encoding: verify it's compatible with BOM
		if detectedEncoding != "" {
			if !encodingsCompatible(encoding, detectedEncoding) {
				return "", &XPathError{Code: "FOUT1190", Message: fmt.Sprintf("fn:unparsed-text: specified encoding %s conflicts with BOM (%s)", encoding, detectedEncoding)}
			}
		}
		effectiveEncoding = encoding
	} else if detectedEncoding != "" {
		effectiveEncoding = detectedEncoding
	} else {
		// Default to UTF-8
		effectiveEncoding = "utf-8"
	}

	// Decode if not UTF-8
	switch strings.ToLower(strings.ReplaceAll(effectiveEncoding, "-", "")) {
	case "utf8", "":
		if !utf8.Valid(cleanData) {
			return "", &XPathError{Code: "FOUT1190", Message: "fn:unparsed-text: invalid UTF-8 data"}
		}
		return string(cleanData), nil
	case "utf16le":
		decoder := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder()
		decoded, err := io.ReadAll(transform.NewReader(bytes.NewReader(cleanData), decoder))
		if err != nil {
			return "", &XPathError{Code: "FOUT1190", Message: fmt.Sprintf("fn:unparsed-text: UTF-16LE decode error: %v", err)}
		}
		return string(decoded), nil
	case "utf16be":
		decoder := unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM).NewDecoder()
		decoded, err := io.ReadAll(transform.NewReader(bytes.NewReader(cleanData), decoder))
		if err != nil {
			return "", &XPathError{Code: "FOUT1190", Message: fmt.Sprintf("fn:unparsed-text: UTF-16BE decode error: %v", err)}
		}
		return string(decoded), nil
	default:
		enc := iencoding.Load(effectiveEncoding)
		if enc == nil {
			return "", &XPathError{Code: "FOUT1190", Message: fmt.Sprintf("fn:unparsed-text: unsupported encoding: %s", effectiveEncoding)}
		}
		decoder := enc.NewDecoder()
		decoded, err := io.ReadAll(transform.NewReader(bytes.NewReader(cleanData), decoder))
		if err != nil {
			return "", &XPathError{Code: "FOUT1190", Message: fmt.Sprintf("fn:unparsed-text: decode error: %v", err)}
		}
		return string(decoded), nil
	}
}

// encodingsCompatible checks if a user-specified encoding is compatible with
// the BOM-detected encoding.
func encodingsCompatible(specified, detected string) bool {
	norm := func(s string) string {
		return strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(s, "-", ""), "_", ""))
	}
	s := norm(specified)
	d := norm(detected)

	if s == d {
		return true
	}
	// utf-8 compatible with itself
	if s == "utf8" && d == "utf8" {
		return true
	}
	// utf-16 (generic) is compatible with utf-16le or utf-16be
	if s == "utf16" && (d == "utf16le" || d == "utf16be") {
		return true
	}
	return false
}

// validateXMLChars checks that the string contains only valid XML characters.
// Per XML 1.0 §2.2: #x9 | #xA | #xD | [#x20-#xD7FF] | [#xE000-#xFFFD] | [#x10000-#x10FFFF]
func validateXMLChars(s string) error {
	for i, r := range s {
		if !isValidXMLChar(r) {
			return &XPathError{Code: "FOUT1190", Message: fmt.Sprintf("fn:unparsed-text: non-XML character U+%04X at position %d", r, i)}
		}
	}
	return nil
}

func isValidXMLChar(r rune) bool {
	if r == 0x9 || r == 0xA || r == 0xD {
		return true
	}
	if r >= 0x20 && r <= 0xD7FF {
		return true
	}
	if r >= 0xE000 && r <= 0xFFFD {
		return true
	}
	if r >= 0x10000 && r <= 0x10FFFF {
		return true
	}
	return false
}

// FileURIResolver resolves file:// URIs and relative file paths against a base directory.
type FileURIResolver struct {
	BaseDir string
}

// ResolveURI implements URIResolver for file-based resources.
func (r *FileURIResolver) ResolveURI(uri string) (io.ReadCloser, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	var path string
	switch parsed.Scheme {
	case "file":
		path = parsed.Path
	case "":
		if filepath.IsAbs(uri) {
			path = uri
		} else {
			path = filepath.Join(r.BaseDir, uri)
		}
	default:
		return nil, fmt.Errorf("unsupported URI scheme: %s", parsed.Scheme)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return f, nil
}
