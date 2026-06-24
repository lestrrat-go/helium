package catalog

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/lestrrat-go/helium/internal/uripath"
)

// ResolveURI resolves value against base. If value already has a scheme, or
// base is empty, value is returned as-is once it parses as a valid URI. When
// base carries a URI scheme (e.g. "file:///..."), the reference is resolved in
// URI space via (*url.URL).ResolveReference, which handles path-absolute
// ("/abs/x"), relative ("x"), and absolute-URI references correctly. When base
// is a non-URI local path, OS-path semantics apply: an absolute value is
// returned unchanged and a relative value is joined against the base directory.
//
// A non-nil error is returned when base or value is a syntactically malformed
// URI that url.Parse rejects -- including an absolute value that already
// carries a scheme. In that case the returned string is empty so a malformed
// catalog entry is never silently stored as a usable mapping; the caller is
// expected to report the error and skip the entry.
func ResolveURI(base, value string) (string, error) {
	if value == "" {
		return "", nil
	}

	// An absolute value carrying a scheme must still be validated; a malformed
	// absolute URI (e.g. bad percent-encoding) must not bypass url.Parse and be
	// stored raw. Once validated, it stands on its own regardless of base.
	if HasScheme(value) {
		if _, err := url.Parse(value); err != nil {
			return "", fmt.Errorf("malformed URI %q: %w", value, err)
		}
		return value, nil
	}

	if base == "" {
		return value, nil
	}

	// When base has a URI scheme, resolve in URI space. A path-absolute
	// reference such as "/abs/asset.xml" must stay in the base's URI space
	// ("file:///abs/asset.xml"), not collapse to a bare local path.
	if HasScheme(base) {
		baseURL, err := url.Parse(base)
		if err != nil {
			return "", fmt.Errorf("malformed base URI %q: %w", base, err)
		}
		ref, err := url.Parse(value)
		if err != nil {
			return "", fmt.Errorf("malformed URI %q: %w", value, err)
		}
		return baseURL.ResolveReference(ref).String(), nil
	}

	// Non-URI local-path base: apply OS-path semantics. An absolute value is
	// returned unchanged; a relative one is joined against the base directory.
	// uripath.IsAbsolutePath recognizes both POSIX- and Windows-absolute shapes
	// regardless of the host OS, so a "/abs/x" reference is not mis-joined
	// against the base dir on Windows (where filepath.IsAbs("/abs/x") is false).
	if uripath.IsAbsolutePath(value) || filepath.IsAbs(value) {
		return value, nil
	}

	return filepath.Join(filepath.Dir(base), value), nil
}

// HasScheme checks if s looks like a URI with a scheme (e.g., "http://...").
// A single-letter "scheme" that is actually a Windows drive-letter prefix
// (e.g. "C:\\path" or "D:/path") is NOT treated as a URI scheme, so a native
// catalog base path is resolved with OS-path semantics rather than leaking the
// drive letter into URI space.
func HasScheme(s string) bool {
	if uripath.HasWindowsDrivePrefix(s) {
		return false
	}
	colon := strings.IndexByte(s, ':')
	if colon < 1 {
		return false
	}
	for i := range colon {
		c := s[i]
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') &&
			(i == 0 || (c < '0' || c > '9') && c != '+' && c != '-' && c != '.') {
			return false
		}
	}
	return true
}

// ParsePrefer converts a prefer attribute value to a Prefer constant.
func ParsePrefer(v string) Prefer {
	switch strings.ToLower(v) {
	case "system":
		return PreferSystem
	case "public":
		return PreferPublic
	default:
		return PreferPublic
	}
}

// HasNextCatalog checks if an identical nextCatalog entry already exists.
func HasNextCatalog(entries []Entry, url string) bool {
	for i := range entries {
		if entries[i].Type == EntryNextCatalog && entries[i].URL == url {
			return true
		}
	}
	return false
}
