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
// normalized by normalizeURIPath. All work is done on the raw (percent-encoded)
// path so encoding is preserved byte-for-byte.
func joinURIReference(base, ref string) string {
	// libxml2 appends "/" when the base's second-to-last character is '.', i.e.
	// the base ends with ".." (or "x."), forcing the join to traverse upward.
	// Replicated verbatim from xmlC14NFixupBaseAttr for byte-for-byte compat.
	if len(base) > 1 && base[len(base)-2] == '.' {
		base += "/"
	}

	refURL, err := url.Parse(ref)
	if err != nil {
		return ref
	}
	// Step 3: an absolute reference (carrying a scheme) wins outright.
	if refURL.Scheme != "" {
		return ref
	}
	// Opaque references aren't hierarchical; emit as-is.
	if refURL.Opaque != "" {
		return ref
	}

	baseURL, err := url.Parse(base)
	if err != nil {
		return ref
	}

	refPath := refURL.EscapedPath()
	refHasAuthority := uriHasAuthority(ref)
	refQuery, refHasQuery := refURL.RawQuery, refURL.RawQuery != "" || refURL.ForceQuery
	refFrag, refHasFrag := refURL.EscapedFragment(), strings.Contains(ref, "#")

	var r resolvedURI
	switch {
	case refPath == "" && !refHasAuthority:
		// Step 2: empty path and no authority → reference to the base document.
		// Inherit the base scheme, authority and path; take the reference's
		// query/fragment when present, else the base's query.
		r.scheme = baseURL.Scheme
		r.hasAuthority = uriHasAuthority(base)
		r.authority = authorityOf(baseURL)
		r.path = baseURL.EscapedPath()
		if refHasQuery {
			r.query, r.hasQuery = refQuery, true
		} else {
			r.query, r.hasQuery = baseURL.RawQuery, baseURL.RawQuery != "" || baseURL.ForceQuery
		}
		r.fragment, r.hasFragment = refFrag, refHasFrag

	case refHasAuthority:
		// Step 4: network-path reference → keep its authority and path verbatim.
		r.scheme = baseURL.Scheme
		r.hasAuthority = true
		r.authority = authorityOf(refURL)
		r.path = refPath
		r.query, r.hasQuery = refQuery, refHasQuery
		r.fragment, r.hasFragment = refFrag, refHasFrag

	case strings.HasPrefix(refPath, "/"):
		// Step 5: absolute-path reference → replace the path, keep base authority.
		r.scheme = baseURL.Scheme
		r.hasAuthority = uriHasAuthority(base)
		r.authority = authorityOf(baseURL)
		r.path = refPath
		r.query, r.hasQuery = refQuery, refHasQuery
		r.fragment, r.hasFragment = refFrag, refHasFrag

	default:
		// Step 6: relative-path reference → merge with the base path, then
		// normalize away dot segments.
		baseHasAuthority := uriHasAuthority(base)
		r.scheme = baseURL.Scheme
		r.hasAuthority = baseHasAuthority
		r.authority = authorityOf(baseURL)
		r.path = normalizeURIPath(mergePaths(baseURL.EscapedPath(), refPath, baseHasAuthority))
		r.query, r.hasQuery = refQuery, refHasQuery
		r.fragment, r.hasFragment = refFrag, refHasFrag
	}

	return r.String()
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
	b.WriteString(r.path)
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

// uriHasAuthority reports whether a URI reference carries an authority ("//…"),
// looking past an optional scheme. An empty authority ("file:///x") still
// counts.
func uriHasAuthority(raw string) bool {
	if i := strings.IndexByte(raw, ':'); i >= 0 && !strings.ContainsAny(raw[:i], "/?#") {
		raw = raw[i+1:]
	}
	return strings.HasPrefix(raw, "//")
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
// combined with the next-outer as base, and so on outward.
func reduceXMLBase(chain []string) string {
	res := chain[len(chain)-1]
	for i := len(chain) - 2; i >= 0; i-- {
		res = joinURIReference(chain[i], res)
	}
	return res
}
