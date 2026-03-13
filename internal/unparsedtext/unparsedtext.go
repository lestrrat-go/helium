// Package unparsedtext implements the text resource loading pipeline for
// XPath 3.1 fn:unparsed-text, fn:unparsed-text-available, and
// fn:unparsed-text-lines. It handles URI resolution, content retrieval,
// BOM detection, encoding conversion, XML character validation, and
// line splitting.
package unparsedtext

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"

	iencoding "github.com/lestrrat-go/helium/internal/encoding"
)

// URIResolver resolves a URI to a readable stream.
type URIResolver interface {
	ResolveURI(uri string) (io.ReadCloser, error)
}

// Config holds the external dependencies needed by the loading pipeline.
type Config struct {
	BaseURI     string
	HTTPClient  *http.Client
	URIResolver URIResolver
}

// Error codes defined by the XPath 3.1 specification for unparsed-text functions.
const (
	// ErrCodeRetrieval is FOUT1170: the resource cannot be retrieved.
	ErrCodeRetrieval = "FOUT1170"
	// ErrCodeEncoding is FOUT1190: encoding mismatch or decode failure.
	ErrCodeEncoding = "FOUT1190"
)

// Error represents a typed error from the unparsed-text pipeline.
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// LoadText resolves href, retrieves its content, decodes it, validates
// XML characters, and returns the result as a string.
func LoadText(cfg *Config, href, encoding string) (string, error) {
	resolvedURI, err := ResolveURI(cfg, href)
	if err != nil {
		return "", err
	}

	data, err := ReadURI(cfg, resolvedURI)
	if err != nil {
		return "", &Error{Code: ErrCodeRetrieval, Message: fmt.Sprintf("cannot retrieve resource: %v", err)}
	}

	text, err := DecodeText(data, encoding)
	if err != nil {
		return "", err
	}

	if err := ValidateXMLChars(text); err != nil {
		return "", err
	}

	return text, nil
}

// LoadTextLines calls LoadText and splits the result into lines per
// the XPath spec line-ending rules.
func LoadTextLines(cfg *Config, href, encoding string) ([]string, error) {
	text, err := LoadText(cfg, href, encoding)
	if err != nil {
		return nil, err
	}
	return SplitLines(text), nil
}

// IsAvailable returns true if LoadText would succeed for the given href
// and encoding.
func IsAvailable(cfg *Config, href, encoding string) bool {
	_, err := LoadText(cfg, href, encoding)
	return err == nil
}

// ResolveURI validates and resolves an href against the base URI in cfg.
func ResolveURI(cfg *Config, href string) (string, error) {
	if strings.Contains(href, "#") {
		return "", &Error{Code: ErrCodeRetrieval, Message: "URI must not contain a fragment identifier"}
	}

	if href == "" {
		if cfg != nil && cfg.BaseURI != "" {
			return cfg.BaseURI, nil
		}
		return "", &Error{Code: ErrCodeRetrieval, Message: "empty href and no base URI available"}
	}

	parsed, err := parseURIReference(href)
	if err != nil {
		return "", &Error{Code: ErrCodeRetrieval, Message: fmt.Sprintf("invalid URI: %s", href)}
	}

	if parsed.Scheme == "" && strings.HasPrefix(href, ":/") {
		return "", &Error{Code: ErrCodeRetrieval, Message: fmt.Sprintf("invalid URI: %s", href)}
	}

	if isWindowsDriveScheme(parsed) {
		return "", &Error{Code: ErrCodeRetrieval, Message: fmt.Sprintf("unsupported URI scheme: %s", href)}
	}

	if parsed.Scheme != "" {
		if !isSupportedScheme(parsed.Scheme) {
			return "", &Error{Code: ErrCodeRetrieval, Message: fmt.Sprintf("unsupported URI scheme: %s", parsed.Scheme)}
		}
		return href, nil
	}

	if cfg != nil && cfg.BaseURI != "" {
		resolved, err := resolveReference(cfg.BaseURI, href)
		if err == nil {
			return resolved, nil
		}
	}

	return "", &Error{Code: ErrCodeRetrieval, Message: fmt.Sprintf("cannot resolve relative URI without base URI: %s", href)}
}

// ReadURI reads the content at the resolved URI.
func ReadURI(cfg *Config, uri string) ([]byte, error) {
	if cfg != nil && cfg.URIResolver != nil {
		rc, err := cfg.URIResolver.ResolveURI(uri)
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

	if parsed.Scheme == "http" || parsed.Scheme == "https" {
		client := http.DefaultClient
		if cfg != nil && cfg.HTTPClient != nil {
			client = cfg.HTTPClient
		}
		resp, err := client.Get(uri)
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

// DecodeText decodes raw bytes to a string, handling BOM detection and
// encoding conversion. If encoding is empty, UTF-8 is assumed unless a
// BOM indicates otherwise.
func DecodeText(data []byte, encoding string) (string, error) {
	if encoding != "" {
		enc := iencoding.Load(encoding)
		if enc == nil {
			return "", &Error{Code: ErrCodeEncoding, Message: fmt.Sprintf("unsupported encoding: %s", encoding)}
		}
	}

	detectedEncoding, cleanData := detectBOM(data)

	effectiveEncoding := resolveEncoding(encoding, detectedEncoding)
	if effectiveEncoding == "" {
		return "", &Error{Code: ErrCodeEncoding, Message: fmt.Sprintf("specified encoding %s conflicts with BOM (%s)", encoding, detectedEncoding)}
	}

	return decodeBytes(cleanData, effectiveEncoding)
}

// detectBOM checks for a byte-order mark and returns the detected encoding
// name and data with the BOM stripped.
func detectBOM(data []byte) (string, []byte) {
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return "utf-8", data[3:]
	}
	if len(data) >= 2 && data[0] == 0xFF && data[1] == 0xFE {
		return "utf-16le", data[2:]
	}
	if len(data) >= 2 && data[0] == 0xFE && data[1] == 0xFF {
		return "utf-16be", data[2:]
	}
	return "", data
}

// resolveEncoding determines the effective encoding from the user-specified
// encoding and BOM-detected encoding. Returns "" if they conflict.
func resolveEncoding(specified, detected string) string {
	if specified != "" {
		if detected != "" && !EncodingsCompatible(specified, detected) {
			return ""
		}
		return specified
	}
	if detected != "" {
		return detected
	}
	return "utf-8"
}

func decodeBytes(data []byte, encoding string) (string, error) {
	switch normalizeEncodingName(encoding) {
	case "utf8", "":
		if !utf8.Valid(data) {
			return "", &Error{Code: ErrCodeEncoding, Message: "invalid UTF-8 data"}
		}
		return string(data), nil
	case "utf16le":
		decoder := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder()
		decoded, err := io.ReadAll(transform.NewReader(bytes.NewReader(data), decoder))
		if err != nil {
			return "", &Error{Code: ErrCodeEncoding, Message: fmt.Sprintf("UTF-16LE decode error: %v", err)}
		}
		return string(decoded), nil
	case "utf16be":
		decoder := unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM).NewDecoder()
		decoded, err := io.ReadAll(transform.NewReader(bytes.NewReader(data), decoder))
		if err != nil {
			return "", &Error{Code: ErrCodeEncoding, Message: fmt.Sprintf("UTF-16BE decode error: %v", err)}
		}
		return string(decoded), nil
	default:
		enc := iencoding.Load(encoding)
		if enc == nil {
			return "", &Error{Code: ErrCodeEncoding, Message: fmt.Sprintf("unsupported encoding: %s", encoding)}
		}
		decoder := enc.NewDecoder()
		decoded, err := io.ReadAll(transform.NewReader(bytes.NewReader(data), decoder))
		if err != nil {
			return "", &Error{Code: ErrCodeEncoding, Message: fmt.Sprintf("decode error: %v", err)}
		}
		return string(decoded), nil
	}
}

// SplitLines splits text into lines per the XPath spec. Line endings are
// normalized: CR+LF -> LF, CR -> LF. A trailing newline does not produce
// an extra empty line.
func SplitLines(text string) []string {
	var b strings.Builder
	b.Grow(len(text))
	for i := 0; i < len(text); i++ {
		if text[i] == '\r' {
			b.WriteByte('\n')
			if i+1 < len(text) && text[i+1] == '\n' {
				i++
			}
		} else {
			b.WriteByte(text[i])
		}
	}
	normalized := b.String()

	if normalized == "" {
		return []string{""}
	}
	lines := strings.Split(normalized, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// ValidateXMLChars checks that the string contains only valid XML 1.0
// characters: #x9 | #xA | #xD | [#x20-#xD7FF] | [#xE000-#xFFFD] | [#x10000-#x10FFFF].
func ValidateXMLChars(s string) error {
	for i, r := range s {
		if !IsValidXMLChar(r) {
			return &Error{Code: ErrCodeEncoding, Message: fmt.Sprintf("non-XML character U+%04X at position %d", r, i)}
		}
	}
	return nil
}

// IsValidXMLChar reports whether r is a valid XML 1.0 character.
func IsValidXMLChar(r rune) bool {
	if r == 0x9 || r == 0xA || r == 0xD {
		return true
	}
	if r >= 0x20 && r <= 0xD7FF {
		return true
	}
	if r >= 0xE000 && r <= 0xFFFD {
		return true
	}
	return r >= 0x10000 && r <= 0x10FFFF
}

// EncodingsCompatible checks if a user-specified encoding is compatible
// with a BOM-detected encoding.
func EncodingsCompatible(specified, detected string) bool {
	s := normalizeEncodingName(specified)
	d := normalizeEncodingName(detected)

	if s == d {
		return true
	}
	if s == "utf16" && (d == "utf16le" || d == "utf16be") {
		return true
	}
	return false
}

// FileURIResolver resolves file:// URIs and relative file paths against a
// base directory.
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

func normalizeEncodingName(s string) string {
	return strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(s, "-", ""), "_", ""))
}

func parseURIReference(raw string) (*url.URL, error) {
	if err := validatePercentEncoding(raw); err != nil {
		return nil, err
	}
	return url.Parse(raw)
}

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

func isWindowsDriveScheme(parsed *url.URL) bool {
	if len(parsed.Scheme) != 1 {
		return false
	}
	b := parsed.Scheme[0]
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}

func isSupportedScheme(scheme string) bool {
	switch scheme {
	case "file", "http", "https":
		return true
	default:
		return false
	}
}

func resolveReference(base, ref string) (string, error) {
	baseURL, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	refURL, err := url.Parse(ref)
	if err != nil {
		return "", err
	}
	return baseURL.ResolveReference(refURL).String(), nil
}
