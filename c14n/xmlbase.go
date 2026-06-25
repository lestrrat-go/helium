package c14n

import (
	"net/url"
	"strings"
)

// joinURIReference combines an outer (ancestor) base xml:base value with an
// inner (descendant) reference xml:base value, following the modified RFC 3986
// "join-URI-References" procedure defined by Canonical XML 1.1 (W3C xml-c14n11
// §2.4) and implemented by libxml2's xmlC14NFixupBaseAttr → xmlBuildURI.
//
// It is a faithful port of libxml2's xmlBuildURI component algorithm: the
// reference is resolved against the base by RFC 3986 §5.3 rules (empty-path
// references keep the base path, network-path references keep their authority,
// absolute-path references replace the path), and the merged relative path is
// normalized by normalizeURIPath. Relative references stay relative, leading
// "../" segments survive, and consecutive slashes collapse.
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

	baseURL, err := url.Parse(base)
	if err != nil {
		return ref
	}

	refHasAuthority := refURL.Host != "" || refURL.User != nil || strings.HasPrefix(ref, "//")
	refHasQuery := refURL.RawQuery != "" || refURL.ForceQuery

	res := &url.URL{}

	switch {
	case refURL.Opaque != "":
		// Opaque references aren't hierarchical; emit as-is.
		return ref

	case refURL.Path == "" && !refHasAuthority:
		// Step 2: empty path and no authority → reference to the base document.
		// Inherit the base scheme, authority and path; take the reference's
		// query/fragment when present, else the base's query.
		res.Scheme = baseURL.Scheme
		res.Host = baseURL.Host
		res.User = baseURL.User
		res.Path = baseURL.Path
		if refHasQuery {
			res.RawQuery = refURL.RawQuery
			res.ForceQuery = refURL.ForceQuery
		} else {
			res.RawQuery = baseURL.RawQuery
			res.ForceQuery = baseURL.ForceQuery
		}
		res.Fragment = refURL.Fragment

	case refHasAuthority:
		// Step 4: network-path reference → keep its authority and path verbatim.
		res.Scheme = baseURL.Scheme
		res.Host = refURL.Host
		res.User = refURL.User
		res.Path = refURL.Path
		res.RawQuery = refURL.RawQuery
		res.ForceQuery = refURL.ForceQuery
		res.Fragment = refURL.Fragment

	case strings.HasPrefix(refURL.Path, "/"):
		// Step 5: absolute-path reference → replace the path, keep base authority.
		res.Scheme = baseURL.Scheme
		res.Host = baseURL.Host
		res.User = baseURL.User
		res.Path = refURL.Path
		res.RawQuery = refURL.RawQuery
		res.ForceQuery = refURL.ForceQuery
		res.Fragment = refURL.Fragment

	default:
		// Step 6: relative-path reference → merge with the base path, then
		// normalize away dot segments (keeping leading "..").
		res.Scheme = baseURL.Scheme
		res.Host = baseURL.Host
		res.User = baseURL.User
		res.Path = normalizeURIPath(mergePaths(baseURL.Path, refURL.Path, baseURL.Host != ""))
		res.RawQuery = refURL.RawQuery
		res.ForceQuery = refURL.ForceQuery
		res.Fragment = refURL.Fragment
	}

	return recombineURI(res)
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

// normalizeURIPath removes "." and ".." path segments, faithfully matching
// libxml2's xmlNormalizeURIPath (RFC 2396 §5.2 step 6 c–f): leading ".."
// segments are kept, consecutive slashes collapse to one, and a relative path
// that fully cancels becomes "" (not "/").
func normalizeURIPath(path string) string {
	if path == "" {
		return ""
	}

	leading := strings.HasPrefix(path, "/")
	segs := strings.Split(path, "/")
	out := make([]string, 0, len(segs))
	trailingDir := false
	for i, seg := range segs {
		last := i == len(segs)-1
		switch seg {
		case "":
			// Empty segment: a leading, trailing or doubled slash. Only a
			// trailing one denotes a directory; interior ones collapse away.
			trailingDir = last
		case ".":
			// "." is dropped; as the final segment it still denotes a directory.
			trailingDir = last
		case "..":
			if n := len(out); n > 0 && out[n-1] != ".." {
				out = out[:n-1]
			} else {
				// No parent to pop (or it is itself ".."): keep so leading
				// "../" survives.
				out = append(out, "..")
			}
			trailingDir = true
		default:
			out = append(out, seg)
			trailingDir = false
		}
	}

	res := strings.Join(out, "/")
	if leading {
		res = "/" + res
	}
	if trailingDir && res != "" && !strings.HasSuffix(res, "/") {
		res += "/"
	}
	if res == "" && leading {
		res = "/"
	}
	return res
}

// recombineURI reassembles a resolved URL into its string form, mirroring
// libxml2's xmlSaveUri for the components C14N xml:base fixup produces.
func recombineURI(u *url.URL) string {
	var b strings.Builder
	if u.Scheme != "" {
		b.WriteString(u.Scheme)
		b.WriteString(":")
	}
	if u.Host != "" || u.User != nil {
		b.WriteString("//")
		if u.User != nil {
			b.WriteString(u.User.String())
			b.WriteString("@")
		}
		b.WriteString(u.Host)
	}
	b.WriteString(u.Path)
	if u.RawQuery != "" || u.ForceQuery {
		b.WriteString("?")
		b.WriteString(u.RawQuery)
	}
	if u.Fragment != "" {
		b.WriteString("#")
		b.WriteString(u.EscapedFragment())
	}
	return b.String()
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
