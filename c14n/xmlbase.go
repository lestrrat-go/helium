package c14n

import (
	"net/url"
	"strings"
)

// joinURIReference combines an outer (ancestor) base xml:base value with an
// inner (descendant) reference xml:base value, following the modified RFC 3986
// "join-URI-References" procedure of Canonical XML 1.1 (W3C xml-c14n11 §2.4) as
// implemented by libxml2's xmlC14NFixupBaseAttr → xmlBuildURI.
//
// It is a port of xmlBuildURI's component algorithm (RFC 2396 §5.2): empty-path
// references keep the base path, network-path references keep their authority,
// absolute-path references replace the path, and a merged relative path is
// normalized by normalizeURIPath. Paths are handled decoded — as libxml2 does
// after unescaping on parse — and re-escaped on output, so an encoded delimiter
// like %2F acts as "/" while a space round-trips to %20.
//
// This matches libxml2 byte-for-byte for every well-formed xml:base value
// (relative/absolute paths, http(s)/file/urn URIs, and protocol-relative
// //host/path), validated by differential testing against libxml2 2.9.14's
// xmlBuildURI across thousands of cases.
//
// The returned bool reports whether the join was performed faithfully. It is
// false for inputs this code cannot reproduce libxml2 byte-for-byte — a
// malformed URI reference or a degenerate empty-authority form ("//", "///",
// "urn://") — which never occur in a well-formed xml:base. Callers in
// libxml2-compat mode ignore it (best-effort result); strict mode turns it into
// an operation failure.
//
// Component parsing uses net/url rather than a full port of libxml2's RFC 3986
// URI parser. This is a deliberate, accepted trade-off: net/url has a few
// micro-discrepancies from libxml2 on pathological URI forms that do not occur
// in real xml:base values, and we do not paper over them:
//   - IPvFuture authorities ("http://[v7.x]/…"): net/url rejects them, so the
//     join falls back to returning the reference unchanged. libxml2 accepts them.
//   - Strict-mode lexical validation (validURIReference) rejects "[" / "]"
//     outside an authority, so it also rejects a bracket inside a *fragment*
//     ("a#x[y]"), which libxml2 accepts (it still correctly rejects brackets in
//     a path or query, matching libxml2).
//
// Closing these would require porting libxml2's URI grammar wholesale for no
// real-world benefit, so we keep net/url and document the gaps instead.
func joinURIReference(base, ref string) (string, bool) {
	// libxml2 appends "/" when the base's second-to-last character is '.', i.e.
	// the base ends with ".." (or "x."), forcing the join to traverse upward.
	// Replicated verbatim from xmlC14NFixupBaseAttr for byte-for-byte compat.
	if len(base) > 1 && base[len(base)-2] == '.' {
		base += "/"
	}

	refURL, err := url.Parse(ref)
	if err != nil {
		return ref, false
	}
	// Step 3: an absolute reference (carrying a scheme) wins outright.
	if refURL.Scheme != "" {
		return ref, true
	}
	// Opaque references aren't hierarchical; emit as-is.
	if refURL.Opaque != "" {
		return ref, true
	}

	baseURL, err := url.Parse(base)
	if err != nil {
		return ref, false
	}
	basePath := decodedBasePath(baseURL)
	// url.Parse lowercases the scheme; libxml2 preserves its original case, so
	// inherit the raw lexical scheme rather than baseURL.Scheme.
	baseScheme := ""
	if baseURL.Scheme != "" {
		baseScheme = rawScheme(base)
	}

	// Work on the decoded path (like libxml2, which unescapes on parse): an
	// encoded delimiter such as %2F acts as "/", and %2E as ".". The result is
	// re-escaped on output by resolvedURI.String.
	refPath := refURL.Path
	refHasAuthority := uriHasAuthority(ref)
	refQuery, refHasQuery := refURL.RawQuery, refURL.RawQuery != "" || refURL.ForceQuery
	// The query is carried raw (libxml2 keeps query_raw verbatim), but the
	// fragment is decoded on parse and re-escaped on save, so derive it from the
	// decoded Fragment rather than EscapedFragment().
	refFrag, refHasFrag := escapeURIFragment(refURL.Fragment), strings.Contains(ref, "#")

	// A "//" authority marker with no host is degenerate. On the reference it is
	// always unfaithful; on the base it is unfaithful unless a scheme and a
	// non-empty path make it a valid empty-authority form like "file:///a".
	faithful := !emptyAuthority(ref, refURL) && !degenerateBaseAuthority(base, baseURL, basePath)

	var r resolvedURI
	switch {
	case refPath == "" && !refHasAuthority:
		// Step 2: empty path and no authority → reference to the base document.
		// Inherit the base scheme, authority and path; take the reference's
		// query/fragment when present, else the base's query.
		r.scheme = baseScheme
		r.hasAuthority = uriHasAuthority(base)
		r.authority = authorityOf(baseURL)
		r.path = basePath
		if refHasQuery {
			r.query, r.hasQuery = refQuery, true
		} else {
			r.query, r.hasQuery = baseURL.RawQuery, baseURL.RawQuery != "" || baseURL.ForceQuery
		}
		r.fragment, r.hasFragment = refFrag, refHasFrag

	case refHasAuthority:
		// Step 4: network-path reference → keep its authority and path verbatim.
		r.scheme = baseScheme
		r.hasAuthority = true
		r.authority = authorityOf(refURL)
		r.path = refPath
		r.query, r.hasQuery = refQuery, refHasQuery
		r.fragment, r.hasFragment = refFrag, refHasFrag

	case strings.HasPrefix(refPath, "/"):
		// Step 5: absolute-path reference → replace the path, keep base authority.
		r.scheme = baseScheme
		r.hasAuthority = uriHasAuthority(base)
		r.authority = authorityOf(baseURL)
		r.path = refPath
		r.query, r.hasQuery = refQuery, refHasQuery
		r.fragment, r.hasFragment = refFrag, refHasFrag

	default:
		// Step 6: relative-path reference → merge with the base path, then
		// normalize away dot segments.
		baseHasAuthority := uriHasAuthority(base)
		r.scheme = baseScheme
		r.hasAuthority = baseHasAuthority
		r.authority = authorityOf(baseURL)
		r.path = normalizeURIPath(mergePaths(basePath, refPath, baseHasAuthority))
		r.query, r.hasQuery = refQuery, refHasQuery
		r.fragment, r.hasFragment = refFrag, refHasFrag
	}

	return r.String(), faithful
}

// resolvedURI holds the components produced by joinURIReference. Presence flags
// distinguish an empty-but-present authority/query (e.g. "file:///") from an
// absent one.
type resolvedURI struct {
	scheme       string
	hasAuthority bool
	authority    string
	path         string
	hasQuery     bool
	query        string
	hasFragment  bool
	fragment     string
}

func (r resolvedURI) String() string {
	var b strings.Builder
	if r.scheme != "" {
		b.WriteString(r.scheme)
		b.WriteString(":")
	}
	if r.hasAuthority {
		b.WriteString("//")
		b.WriteString(r.authority)
	}
	b.WriteString(escapeURIPath(r.path))
	if r.hasQuery {
		b.WriteString("?")
		b.WriteString(r.query)
	}
	if r.hasFragment {
		b.WriteString("#")
		b.WriteString(r.fragment)
	}
	return b.String()
}

const uriHexUpper = "0123456789ABCDEF"

// escapeURIPath re-escapes a decoded path with the exact character set libxml2's
// xmlSaveUri keeps unescaped in a path: unreserved (alphanumerics and
// "-_.!~*'()"), plus "/;@&=+$,". Every other byte becomes %XX (uppercase). This
// mirrors libxml2's unescape-on-parse / escape-on-save round trip, so e.g. a
// space round-trips to %20 while "/" and "." stay literal.
func escapeURIPath(path string) string {
	if !strings.ContainsFunc(path, func(r rune) bool { return r > 127 || !isPathSafe(byte(r)) }) {
		return path
	}
	var b strings.Builder
	b.Grow(len(path))
	for i := range len(path) {
		c := path[i]
		if isPathSafe(c) {
			b.WriteByte(c)
			continue
		}
		b.WriteByte('%')
		b.WriteByte(uriHexUpper[c>>4])
		b.WriteByte(uriHexUpper[c&0x0f])
	}
	return b.String()
}

// escapeURIFragment re-escapes a decoded fragment with libxml2's xmlSaveUri
// fragment set: unreserved + reserved kept, everything else %XX. The reserved
// set is wider than a path's (it also keeps "?:[]"), e.g. "f/g" and "a/b?c" stay
// literal while a space becomes %20.
func escapeURIFragment(frag string) string {
	if !strings.ContainsFunc(frag, func(r rune) bool { return r > 127 || !isFragmentSafe(byte(r)) }) {
		return frag
	}
	var b strings.Builder
	b.Grow(len(frag))
	for i := range len(frag) {
		c := frag[i]
		if isFragmentSafe(c) {
			b.WriteByte(c)
			continue
		}
		b.WriteByte('%')
		b.WriteByte(uriHexUpper[c>>4])
		b.WriteByte(uriHexUpper[c&0x0f])
	}
	return b.String()
}

func isFragmentSafe(c byte) bool {
	switch {
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		return true
	}
	switch c {
	case '-', '_', '.', '!', '~', '*', '\'', '(', ')': // unreserved marks
		return true
	case ';', '/', '?', ':', '@', '&', '=', '+', '$', ',', '[', ']': // reserved
		return true
	}
	return false
}

func isPathSafe(c byte) bool {
	switch {
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		return true
	}
	switch c {
	case '-', '_', '.', '!', '~', '*', '\'', '(', ')': // unreserved marks
		return true
	case '/', ';', '@', '&', '=', '+', '$', ',': // path-allowed reserved
		return true
	}
	return false
}

// uriHasAuthority reports whether a URI reference carries an authority ("//…"),
// looking past an optional scheme. An empty authority ("file:///x") still
// counts.
func uriHasAuthority(raw string) bool {
	if i := strings.IndexByte(raw, ':'); i >= 0 && !strings.ContainsAny(raw[:i], "/?#") {
		raw = raw[i+1:]
	}
	return strings.HasPrefix(raw, "//")
}

// rawScheme returns the scheme of a URI reference with its original case (url.Parse
// lowercases it), or "" when there is no scheme.
func rawScheme(s string) string {
	i := strings.IndexByte(s, ':')
	if i <= 0 || strings.ContainsAny(s[:i], "/?#") {
		return ""
	}
	return s[:i]
}

// decodedBasePath returns a parsed base's path in decoded form. Go parses a
// scheme-only base like "urn:base" with its payload in Opaque rather than Path;
// libxml2 treats that payload as the (decoded) path, so fold it back, decoding
// it the way url.Parse already decodes Path.
func decodedBasePath(u *url.URL) string {
	if u.Opaque == "" {
		return u.Path
	}
	if dec, err := url.PathUnescape(u.Opaque); err == nil {
		return dec
	}
	return u.Opaque
}

// faithfulXMLBaseValue reports whether a standalone xml:base value can be
// canonicalized faithfully: it must parse as a URI reference and not be a
// degenerate empty-authority form ("//", "///", "urn://", …). An empty value is
// fine — it is dropped rather than emitted. Strict mode rejects a chain term for
// which this is false, even a lone term that never participates in a join.
func faithfulXMLBaseValue(v string) bool {
	if v == "" {
		return true
	}
	// url.Parse tolerates raw characters a URI reference may not contain (spaces,
	// the "unwise" set, bad %-encoding) that libxml2 rejects; validate the lexical
	// form first, then check for a degenerate empty authority.
	if !validURIReference(v) {
		return false
	}
	u, err := url.Parse(v)
	if err != nil {
		return false
	}
	return !degenerateBaseAuthority(v, u, decodedBasePath(u))
}

// validURIReference reports whether s is lexically a URI reference as libxml2's
// parser accepts it: every byte is unreserved, reserved, "#", or part of a
// well-formed "%"HEXHEX escape. This rejects raw spaces, control bytes, truncated
// escapes, and the "unwise" set ("{}|\\^`") — values libxml2 canonicalization
// fails on. Square brackets are allowed only as an IPv6 authority literal
// (e.g. "http://[::1]/"), matching libxml2 which accepts that but rejects "a[b]".
func validURIReference(s string) bool {
	brackets := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '%' {
			if i+2 >= len(s) || !isHexDigit(s[i+1]) || !isHexDigit(s[i+2]) {
				return false
			}
			i += 2
			continue
		}
		if c == '[' || c == ']' {
			brackets++
			continue
		}
		if !validURIChar(c) {
			return false
		}
	}
	if brackets == 0 {
		return true
	}
	// The only legal brackets are the single pair delimiting an IPv6 host, which
	// url.Parse surfaces in Host (possibly with a ":port" suffix).
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	return brackets == 2 && strings.HasPrefix(u.Host, "[") &&
		strings.Count(u.Host, "[") == 1 && strings.Count(u.Host, "]") == 1
}

func validURIChar(c byte) bool {
	switch {
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		return true
	}
	switch c {
	case '-', '_', '.', '!', '~', '*', '\'', '(', ')': // unreserved marks
		return true
	case ';', '/', '?', ':', '@', '&', '=', '+', '$', ',', '#': // reserved + fragment
		return true
	}
	return false
}

func isHexDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

// emptyAuthority reports whether a URI reference carries a "//" authority marker
// with no host or userinfo (e.g. "//", "///", "urn://", "//?q").
func emptyAuthority(raw string, u *url.URL) bool {
	return uriHasAuthority(raw) && u.Host == "" && u.User == nil
}

// degenerateBaseAuthority reports a base with an empty authority that cannot be
// joined faithfully — i.e. one that is not a valid empty-authority form, which
// requires both a scheme and a non-empty path (e.g. "file:///a").
func degenerateBaseAuthority(base string, u *url.URL, path string) bool {
	return emptyAuthority(base, u) && (u.Scheme == "" || path == "")
}

// authorityOf returns the userinfo+host authority string of a parsed URL.
func authorityOf(u *url.URL) string {
	if u.User != nil {
		return u.User.String() + "@" + u.Host
	}
	return u.Host
}

// mergePaths merges a relative reference path onto a base path per RFC 3986
// §5.3: all of the base up to and including its last "/" plus the reference.
// When the base has an authority but an empty path, the result is rooted at "/".
func mergePaths(basePath, refPath string, baseHasAuthority bool) string {
	if i := strings.LastIndex(basePath, "/"); i >= 0 {
		return basePath[:i+1] + refPath
	}
	if baseHasAuthority && basePath == "" {
		return "/" + refPath
	}
	return refPath
}

// normalizeURIPath removes "." and ".." path segments, a faithful port of
// libxml2's xmlNormalizeURIPath (RFC 2396 §5.2 step 6 c–g). Of note: a leading
// path segment is never consumed by a trailing ".." (so "a/b/../.." → "a/.."),
// leading ".." segments survive on a relative path, "//" collapses to "/", and
// leading "/../" is discarded only on an absolute path.
func normalizeURIPath(path string) string {
	b := []byte(path)
	n := len(b)

	first := 0
	for first < n && b[first] == '/' {
		first++
	}
	if first >= n {
		return path
	}

	// Pass 1: (c) drop "./"; (d) drop a trailing "."; collapse "//".
	out, cur := first, first
	for cur < n {
		if b[cur] == '.' && cur+1 < n && b[cur+1] == '/' {
			cur += 2
			for cur < n && b[cur] == '/' {
				cur++
			}
			continue
		}
		if b[cur] == '.' && cur+1 == n {
			break
		}
		for cur < n && b[cur] != '/' {
			b[out] = b[cur]
			out++
			cur++
		}
		if cur >= n {
			break
		}
		for cur+1 < n && b[cur+1] == '/' {
			cur++
		}
		b[out] = b[cur]
		out++
		cur++
	}
	n = out

	// Pass 2: (e)(f) remove "<segment>/.." where <segment> != "..".
	start := 0
	for start < n && b[start] == '/' {
		start++
	}
	cur = start
	for cur < n {
		segp := cur
		for segp < n && b[segp] != '/' {
			segp++
		}
		if segp >= n {
			break
		}
		segp++ // past '/'
		curIsDotDot := b[cur] == '.' && cur+1 < n && b[cur+1] == '.' && segp == cur+3
		nextIsDotDot := segp+1 < n && b[segp] == '.' && b[segp+1] == '.' && (segp+2 >= n || b[segp+2] == '/')
		if curIsDotDot || !nextIsDotDot {
			cur = segp
			continue
		}
		if segp+2 >= n {
			n = cur // the trailing ".." ends the buffer
			break
		}
		copy(b[cur:], b[segp+3:n])
		n -= segp + 3 - cur
		// Back up to the previous segment; if it is the first one, stop here.
		sp := cur
		for sp > 0 {
			sp--
			if b[sp] != '/' {
				break
			}
		}
		if sp == 0 {
			continue
		}
		cur = sp
		for cur > 0 && b[cur-1] != '/' {
			cur--
		}
	}
	b = b[:n]

	// Pass 3: (g) discard leading "/../" segments — absolute paths only.
	if len(b) > 0 && b[0] == '/' {
		c := 0
		for c+2 < len(b) && b[c] == '/' && b[c+1] == '.' && b[c+2] == '.' && (c+3 >= len(b) || b[c+3] == '/') {
			c += 3
		}
		if c != 0 {
			b = b[c:]
		}
	}

	return string(b)
}

// reduceXMLBase folds a chain of xml:base values, ordered outermost to
// innermost, into a single canonical value. Per C14N 1.1 the innermost value is
// combined with the next-outer as base, and so on outward. The bool is false if
// any chain term is itself un-canonicalizable, or any join could not be
// performed faithfully — so a lone term (no join) is still validated.
func reduceXMLBase(chain []string) (string, bool) {
	res := chain[len(chain)-1]
	faithful := faithfulXMLBaseValue(res)
	for i := len(chain) - 2; i >= 0; i-- {
		faithful = faithful && faithfulXMLBaseValue(chain[i])
		joined, ok := joinURIReference(chain[i], res)
		res, faithful = joined, faithful && ok
	}
	return res, faithful
}
