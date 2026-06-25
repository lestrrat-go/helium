package c14n

import (
	"net/url"
	"strings"
)

// joinURIReference combines an outer (ancestor) base xml:base value with an
// inner (descendant) reference xml:base value, following the modified RFC 3986
// "join-URI-References" procedure defined by Canonical XML 1.1 (W3C xml-c14n11
// §2.4) and implemented by libxml2's xmlC14NFixupBaseAttr/xmlBuildURI.
//
// Unlike plain RFC 3986 resolution, relative references stay relative and
// leading "../" segments are preserved (standard remove_dot_segments would
// discard them). The canonical form is the lexical join of the in-document
// xml:base attribute values — no external/retrieval base URI participates.
func joinURIReference(base, ref string) string {
	// libxml2 appends "/" when the base's second-to-last character is '.', i.e.
	// the base ends with ".." (or "x."), forcing the join to traverse upward.
	// Replicated verbatim from xmlC14NFixupBaseAttr for byte-for-byte compat.
	if len(base) > 1 && base[len(base)-2] == '.' {
		base += "/"
	}

	// An absolute reference (one carrying a scheme) wins outright.
	if refURL, err := url.Parse(ref); err == nil && refURL.IsAbs() {
		return ref
	}

	// An absolute base resolves the reference with full RFC 3986 semantics:
	// scheme and authority are carried and dot segments collapse normally.
	if baseURL, err := url.Parse(base); err == nil && baseURL.IsAbs() {
		if refURL, err := url.Parse(ref); err == nil {
			return baseURL.ResolveReference(refURL).String()
		}
	}

	// Both base and reference are relative: merge lexically, keeping leading
	// "../" segments.
	return mergeRelativeReference(base, ref)
}

// mergeRelativeReference merges a relative reference against a relative base
// path per RFC 3986 §5.2.3/§5.3, then removes dot segments while preserving
// leading "../" (the C14N 1.1 modification).
func mergeRelativeReference(base, ref string) string {
	// A reference beginning with "/" replaces the base path entirely. A relative
	// base carries no authority, so the absolute-path reference stands alone.
	if strings.HasPrefix(ref, "/") {
		return removeDotSegmentsKeepLeading(ref)
	}

	// Merge: everything up to and including the base's last "/" plus the
	// reference. A base with no "/" contributes no directory prefix.
	merged := ref
	if i := strings.LastIndex(base, "/"); i >= 0 {
		merged = base[:i+1] + ref
	}
	return removeDotSegmentsKeepLeading(merged)
}

// removeDotSegmentsKeepLeading applies RFC 3986 remove_dot_segments to a path
// but keeps leading ".." segments (which the standard algorithm discards) and
// collapses consecutive slashes, matching the C14N 1.1 join modification.
func removeDotSegmentsKeepLeading(path string) string {
	if path == "" {
		return ""
	}

	leadingSlash := strings.HasPrefix(path, "/")
	trailingSlash := strings.HasSuffix(path, "/")

	raw := strings.Split(strings.Trim(path, "/"), "/")
	out := make([]string, 0, len(raw))
	for i, seg := range raw {
		last := i == len(raw)-1
		switch seg {
		case "", ".":
			// "" collapses consecutive slashes; "." is a no-op. A trailing
			// "." still denotes the directory, so keep the trailing slash.
			if seg == "." && last {
				trailingSlash = true
			}
		case "..":
			n := len(out)
			if n > 0 && out[n-1] != ".." {
				out = out[:n-1]
				if last {
					trailingSlash = true
				}
			} else {
				// No parent to pop (or the parent is itself ".."): keep it so
				// leading "../" survives.
				out = append(out, "..")
			}
		default:
			out = append(out, seg)
		}
	}

	res := strings.Join(out, "/")
	if leadingSlash {
		res = "/" + res
	}
	if trailingSlash && res != "" && res != "/" && !strings.HasSuffix(res, "/") {
		res += "/"
	}
	if res == "" && (leadingSlash || trailingSlash) {
		res = "/"
	}
	return res
}

// reduceXMLBase folds a chain of xml:base values, ordered outermost to
// innermost, into a single canonical value. Per C14N 1.1 the innermost value is
// combined with the next-outer as base, and so on outward.
func reduceXMLBase(chain []string) string {
	res := chain[len(chain)-1]
	for i := len(chain) - 2; i >= 0; i-- {
		res = joinURIReference(chain[i], res)
	}
	return res
}
