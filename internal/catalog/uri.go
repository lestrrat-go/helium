package catalog

import (
	"net/url"
	"path/filepath"
	"strings"
)

// ResolveURI resolves value against base. If value is already absolute
// (has a scheme) or base is empty, value is returned as-is.
// For file paths, filepath.Join is used.
func ResolveURI(base, value string) string {
	if value == "" {
		return ""
	}

	if HasScheme(value) {
		return value
	}

	if filepath.IsAbs(value) {
		return value
	}

	if base == "" {
		return value
	}

	if !HasScheme(base) {
		return filepath.Join(filepath.Dir(base), value)
	}

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
