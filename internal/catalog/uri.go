package catalog

import (
	"net/url"
	"path/filepath"
	"strings"
)

// ResolveURI resolves value against base. If value already has a scheme, or
// base is empty, value is returned as-is. When base carries a URI scheme (e.g.
// "file:///..."), the reference is resolved in URI space via
// (*url.URL).ResolveReference, which handles path-absolute ("/abs/x"),
// relative ("x"), and absolute-URI references correctly. When base is a
// non-URI local path, OS-path semantics apply: an absolute value is returned
// unchanged and a relative value is joined against the base directory.
func ResolveURI(base, value string) string {
	if value == "" {
		return ""
	}

	if HasScheme(value) {
		return value
	}

	if base == "" {
		return value
	}

	// When base has a URI scheme, resolve in URI space. A path-absolute
	// reference such as "/abs/asset.xml" must stay in the base's URI space
	// ("file:///abs/asset.xml"), not collapse to a bare local path.
	if HasScheme(base) {
		baseURL, err := url.Parse(base)
		if err != nil {
			return value
		}
		ref, err := url.Parse(value)
		if err != nil {
			return value
		}
		return baseURL.ResolveReference(ref).String()
	}

	// Non-URI local-path base: apply OS-path semantics. An absolute value is
	// returned unchanged; a relative one is joined against the base directory.
	if filepath.IsAbs(value) {
		return value
	}

	return filepath.Join(filepath.Dir(base), value)
}

// HasScheme checks if s looks like a URI with a scheme (e.g., "http://...").
func HasScheme(s string) bool {
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
