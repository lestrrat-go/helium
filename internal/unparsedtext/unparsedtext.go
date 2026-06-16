// Package unparsedtext implements the text resource loading pipeline for
// XPath 3.1 fn:unparsed-text, fn:unparsed-text-available, fn:unparsed-text-lines,
// and the document retrieval that backs fn:doc / fn:json-doc.
//
// Resource loading is opt-in. With no URIResolver and no HTTPClient supplied,
// every retrieval attempt fails with ErrCodeRetrieval — there is no implicit
// network access and no implicit filesystem access. Callers who want network
// access must either pass an explicit *http.Client via Config.HTTPClient
// (caller owns the transport, timeouts, and redirect policy) or supply a
// URIResolver that accepts http(s) schemes. Callers who want filesystem
// access must supply a URIResolver such as FileURIResolver or one returned
// from NewFileResolver(fs.FS).
package unparsedtext

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"

	iencoding "github.com/lestrrat-go/helium/internal/encoding"
	"github.com/lestrrat-go/helium/internal/lexicon"
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
func LoadText(ctx context.Context, cfg *Config, href, encoding string) (string, error) {
	resolvedURI, err := ResolveURI(ctx, cfg, href)
	if err != nil {
		return "", err
	}

	data, httpEncoding, err := readURIWithEncoding(ctx, cfg, resolvedURI)
	if err != nil {
		return "", &Error{Code: ErrCodeRetrieval, Message: fmt.Sprintf("cannot retrieve resource: %v", err)}
	}

	text, err := DecodeText(data, encoding, httpEncoding)
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
func LoadTextLines(ctx context.Context, cfg *Config, href, encoding string) ([]string, error) {
	text, err := LoadText(ctx, cfg, href, encoding)
	if err != nil {
		return nil, err
	}
	return SplitLines(text), nil
}

// IsAvailable returns true if LoadText would succeed for the given href
// and encoding.
func IsAvailable(ctx context.Context, cfg *Config, href, encoding string) bool {
	_, err := LoadText(ctx, cfg, href, encoding)
	return err == nil
}

// ResolveURI validates and resolves an href against the base URI in cfg.
func ResolveURI(_ context.Context, cfg *Config, href string) (string, error) {
	if strings.Contains(href, "#") {
		return "", &Error{Code: ErrCodeRetrieval, Message: "URI must not contain a fragment identifier"}
	}

	if href == "" {
		if cfg != nil && cfg.BaseURI != "" {
			// An empty href resolves to the base URI verbatim; the base URI
			// must not carry a fragment identifier either.
			if strings.Contains(cfg.BaseURI, "#") {
				return "", &Error{Code: ErrCodeRetrieval, Message: "base URI must not contain a fragment identifier"}
			}
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

// readURIWithEncoding reads the content at the resolved URI and returns
// an optional encoding hint from HTTP Content-Type headers.
//
// Retrieval is opt-in: file/path URIs require an explicit URIResolver;
// http/https URIs require either an explicit HTTPClient or a URIResolver
// that accepts those schemes. There is no implicit os.ReadFile and no
// implicit http.DefaultClient — see the package-level documentation.
//
// When both HTTPClient and URIResolver are configured, HTTPClient wins
// for http/https so the Content-Type charset hint is preserved.
func readURIWithEncoding(ctx context.Context, cfg *Config, uri string) ([]byte, string, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, "", err
	}

	if parsed.Scheme == lexicon.SchemeHTTP || parsed.Scheme == lexicon.SchemeHTTPS {
		if cfg != nil && cfg.HTTPClient != nil {
			return doHTTPGet(ctx, cfg.HTTPClient, uri)
		}
		if cfg != nil && cfg.URIResolver != nil {
			return readViaResolver(cfg.URIResolver, uri)
		}
		return nil, "", fmt.Errorf("network retrieval requires an explicit HTTPClient or URIResolver: %s", uri)
	}

	if cfg != nil && cfg.URIResolver != nil {
		return readViaResolver(cfg.URIResolver, uri)
	}

	// file:// and bare paths require an explicit URIResolver.
	return nil, "", fmt.Errorf("retrieval of %q requires a URIResolver", uri)
}

func readViaResolver(r URIResolver, uri string) ([]byte, string, error) {
	rc, err := r.ResolveURI(uri)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = rc.Close() }()
	data, err := io.ReadAll(rc)
	return data, "", err
}

// doHTTPGet performs an HTTP GET using the caller-supplied client.
func doHTTPGet(ctx context.Context, client *http.Client, uri string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("HTTP %d for %s", resp.StatusCode, uri)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	enc := extractHTTPCharset(resp.Header.Get("Content-Type"))
	return data, enc, nil
}

// extractHTTPCharset parses the charset parameter from a Content-Type header value.
func extractHTTPCharset(contentType string) string {
	if contentType == "" {
		return ""
	}
	for part := range strings.SplitSeq(contentType, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(strings.ToLower(part), "charset=") {
			charset := part[len("charset="):]
			charset = strings.Trim(charset, "\"' ")
			return charset
		}
	}
	return ""
}

// ReadURI reads the content at the resolved URI.
//
// Retrieval is opt-in: file/path URIs require an explicit URIResolver;
// http/https URIs require either an explicit HTTPClient or a URIResolver.
// See the package-level documentation.
func ReadURI(ctx context.Context, cfg *Config, uri string) ([]byte, error) {
	data, _, err := readURIWithEncoding(ctx, cfg, uri)
	if err != nil {
		var e *Error
		if errors.As(err, &e) {
			return nil, err
		}
		return nil, &Error{Code: ErrCodeRetrieval, Message: fmt.Sprintf("cannot retrieve resource: %v", err)}
	}
	return data, nil
}

// DecodeText decodes raw bytes to a string, handling BOM detection,
// XML declaration sniffing, and encoding conversion. encoding is the
// user-specified encoding (from the function argument); transportHints
// are optional transport-level encoding hints (e.g. from HTTP Content-Type).
// BOM overrides transport hints but conflicts with explicit encoding.
// If all are empty, defaults to UTF-8.
func DecodeText(data []byte, encoding string, transportHints ...string) (string, error) {
	if encoding != "" {
		enc := iencoding.Load(encoding)
		if enc == nil {
			return "", &Error{Code: ErrCodeEncoding, Message: fmt.Sprintf("unsupported encoding: %s", encoding)}
		}
	}

	detectedEncoding, cleanData := detectBOM(data)

	// When no BOM is found and no explicit encoding is given, try to
	// detect encoding from an XML declaration (<?xml ... encoding="..."?>).
	if detectedEncoding == "" && encoding == "" {
		if xmlEnc := detectXMLDeclEncoding(data); xmlEnc != "" {
			detectedEncoding = xmlEnc
		}
	}

	// If we still have no detected encoding and no explicit encoding,
	// use the transport hint as a fallback.
	if detectedEncoding == "" && encoding == "" {
		for _, hint := range transportHints {
			if hint != "" {
				detectedEncoding = hint
				break
			}
		}
	}

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

// detectXMLDeclEncoding looks for an XML declaration at the beginning of
// data and extracts the encoding attribute value if present.
// It works on raw bytes assuming the declaration is in ASCII-compatible encoding.
func detectXMLDeclEncoding(data []byte) string {
	// XML declaration must start at the very beginning.
	if len(data) < 5 || string(data[:5]) != "<?xml" {
		return ""
	}
	// Find the end of the declaration.
	end := bytes.Index(data, []byte("?>"))
	if end < 0 || end > 200 {
		return ""
	}
	decl := string(data[:end])
	// Look for encoding="..." or encoding='...'.
	_, rest, found := strings.Cut(decl, "encoding")
	if !found {
		return ""
	}
	rest = strings.TrimLeft(rest, " \t\r\n")
	if len(rest) == 0 || rest[0] != '=' {
		return ""
	}
	rest = rest[1:]
	rest = strings.TrimLeft(rest, " \t\r\n")
	if len(rest) == 0 {
		return ""
	}
	quote := rest[0]
	if quote != '"' && quote != '\'' {
		return ""
	}
	encVal, _, found2 := strings.Cut(rest[1:], string(quote))
	if !found2 {
		return ""
	}
	return encVal
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
		return nil
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
// base directory. Resolution is confined to BaseDir: paths outside (via
// "../" traversal or absolute paths that don't share the BaseDir prefix)
// are refused.
type FileURIResolver struct {
	BaseDir string
}

// ResolveURI implements URIResolver for file-based resources.
func (r *FileURIResolver) ResolveURI(uri string) (io.ReadCloser, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	var target string
	switch parsed.Scheme {
	case lexicon.SchemeFile:
		target = parsed.Path
	case "":
		if filepath.IsAbs(uri) {
			target = uri
		} else {
			target = filepath.Join(r.BaseDir, uri)
		}
	default:
		return nil, fmt.Errorf("unsupported URI scheme: %s", parsed.Scheme)
	}

	if err := ensureWithin(r.BaseDir, target); err != nil {
		return nil, err
	}

	f, err := os.Open(target)
	if err != nil {
		return nil, err
	}
	return f, nil
}

// ensureWithin verifies that target is inside baseDir after symlink-free
// path cleaning. Both arguments are resolved to absolute, cleaned forms.
// Note that this is a path-level check; callers that must defend against
// symlink races should supply a URIResolver that uses io/fs primitives
// (see NewFileResolver).
func ensureWithin(baseDir, target string) error {
	if baseDir == "" {
		return fmt.Errorf("FileURIResolver: BaseDir is empty")
	}
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return err
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(absBase, absTarget)
	if err != nil {
		return fmt.Errorf("path %q is outside base %q", target, baseDir)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("path %q is outside base %q", target, baseDir)
	}
	return nil
}

// NewHTTPResolver returns a URIResolver that fetches http/https URIs using
// the supplied client. The caller owns the client's transport, timeouts,
// and redirect policy. Non-http(s) schemes are refused. The client must be
// non-nil; passing nil panics. There is intentionally no fallback to
// http.DefaultClient — that would reintroduce the unbounded-timeout
// behavior this package is designed to keep out of XPath evaluation.
func NewHTTPResolver(client *http.Client) URIResolver {
	if client == nil {
		panic("unparsedtext.NewHTTPResolver: client must not be nil")
	}
	return &httpResolver{client: client}
}

type httpResolver struct {
	client *http.Client
}

func (r *httpResolver) ResolveURI(uri string) (io.ReadCloser, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	if parsed.Scheme != lexicon.SchemeHTTP && parsed.Scheme != lexicon.SchemeHTTPS {
		return nil, fmt.Errorf("unsupported URI scheme for HTTP resolver: %s", parsed.Scheme)
	}
	// The URIResolver interface intentionally has no context parameter, so
	// callers cannot cancel an in-flight request via ctx. Cancellation and
	// deadlines must be enforced through the supplied http.Client's Timeout
	// and Transport settings.
	req, err := http.NewRequest(http.MethodGet, uri, nil) //nolint:noctx // URIResolver has no ctx; rely on client Timeout/Transport for cancellation
	if err != nil {
		return nil, err
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("HTTP %d for %s", resp.StatusCode, uri)
	}
	return resp.Body, nil
}

// NewFileResolver returns a URIResolver backed by an io/fs.FS. URIs are
// interpreted as fs paths (slash-separated, relative to the FS root).
// Bare relative paths and file: URIs without an authority are accepted;
// absolute paths (anything beginning with "/", including file:// URIs
// such as file:///etc/passwd) and "../" traversal are refused.
func NewFileResolver(fsys fs.FS) URIResolver {
	return &fsResolver{fsys: fsys}
}

type fsResolver struct {
	fsys fs.FS
}

func (r *fsResolver) ResolveURI(uri string) (io.ReadCloser, error) {
	if r.fsys == nil {
		return nil, fmt.Errorf("fsResolver: nil FS")
	}
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	var name string
	switch parsed.Scheme {
	case lexicon.SchemeFile:
		name = parsed.Path
	case "":
		name = uri
	default:
		return nil, fmt.Errorf("unsupported URI scheme: %s", parsed.Scheme)
	}

	if name == "" {
		return nil, fmt.Errorf("empty path")
	}
	if strings.HasPrefix(name, "/") {
		return nil, fmt.Errorf("absolute path %q is not allowed", name)
	}
	cleaned := path.Clean(name)
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return nil, fmt.Errorf("path %q escapes the FS root", name)
	}
	return r.fsys.Open(cleaned)
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
