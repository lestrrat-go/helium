package xsd

import (
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"github.com/lestrrat-go/helium/internal/uripath"
)

// URIScheme reports the scheme of s when s is an absolute URI reference (it has
// a scheme per RFC 3986, e.g. "https://...", "file:///...", "mem:/...",
// "urn:..."), or "" otherwise.
//
// A bare local filesystem path — even an absolute one like "/tmp/x" or a
// Windows "C:\x" — is NOT treated as a URI: url.Parse assigns no Scheme to
// "/tmp/x", and a single-letter scheme (a Windows drive letter) is rejected so
// OS paths keep their historical filepath handling. Only multi-character
// schemes count.
//
// This is the single canonical scheme-detector for schema-location resolution;
// both the xsd nested-include path and xslt3's schema loader use it so the two
// layers always agree on what counts as an absolute URI.
func URIScheme(s string) string {
	u, err := url.Parse(s)
	if err != nil || len(u.Scheme) < 2 {
		return ""
	}
	return u.Scheme
}

// uriScheme is the unexported spelling used throughout the xsd package.
func uriScheme(s string) string { return URIScheme(s) }

// schemaURIIsAbsolute reports whether s already addresses its own location and
// must NOT be re-resolved against a base directory: it has a URI scheme, or it
// is an absolute filesystem path. A relative reference (the common doc.URL()
// when a Compiler.BaseDir is configured) returns false so the caller resolves
// it against that base, matching the key a nested back-reference computes.
//
// Absolute-path detection uses [uripath.IsAbsolutePath] so a path that is
// absolute under EITHER POSIX or Windows conventions classifies the same way
// regardless of the host OS, matching the GOOS-independent slash math the rest
// of the package's resolution uses.
func schemaURIIsAbsolute(s string) bool {
	if uriScheme(s) != "" {
		return true
	}
	return uripath.IsAbsolutePath(s)
}

// rootSchemaKey computes the canonical fs.FS key of the TOP-LEVEL schema
// document, in EXACTLY the form a nested xs:include/xs:redefine pointing back at
// the root produces via [validateSchemaPath]/[ResolveSchemaURI]. Both the
// in-memory Compile path and CompileFile derive the circular-include guard's
// seed key here, so the seeded key and the key a back-reference computes cannot
// diverge (the bug that let a cycle back to the root re-parse it into spurious
// duplicate components).
//
// docURL is the root document's own location (CompileFile: the file path;
// Compile: doc.URL()). baseDir is the directory that a direct xs:include
// schemaLocation resolves against — by convention the root's own directory for a
// local filesystem, or the full root URI for a URI-addressed schema. Resolution
// branches on shape:
//
//   - An absolute/URI docURL already addresses its own location; it is resolved
//     with an empty base, so ResolveSchemaURI returns it unchanged.
//   - A relative docURL with an EMPTY baseDir is resolved by its FULL value
//     against "". An unset baseDir means nested includes resolve against ""
//     too (the directory embedded in docURL is NOT a resolution base), so a
//     back-reference to the root lands on the root's full relative URL — e.g.
//     a root URL "schemas/main.xsd" is re-included as "schemas/main.xsd", not
//     "main.xsd". Taking the basename here would seed "main.xsd", wrongly
//     skipping a real include of "main.xsd" AND missing the real back-reference
//     to "schemas/main.xsd".
//   - A relative docURL with a NON-EMPTY baseDir is joined onto baseDir, but
//     only its LAST segment is used. Because baseDir is the root's directory, a
//     back-reference lands on "<baseDir>/<root filename>". Using the basename
//     (not the full docURL) means a docURL that already carries the baseDir
//     prefix — an already resolved fs key like "schemas/main.xsd" under
//     BaseDir("schemas") — is not re-joined onto baseDir, which would otherwise
//     double the prefix to "schemas/schemas/main.xsd" and miss the cycle. This
//     mirrors CompileFile's historical filepath.Base(path) seeding.
//   - When docURL is empty, a URI-scheme baseDir IS the full root URI (the URI
//     convention treats the base as the schema's own location); it is the key
//     verbatim.
//
// In every branch the key is produced by the SAME ResolveSchemaURI call
// loadInclude/loadRedefine use for a back-reference href, so the seeded key and
// the computed key cannot diverge.
//
// The bool result is false when no key can be derived (no docURL and a
// non-URI/empty baseDir), in which case the caller leaves the guard unseeded.
func rootSchemaKey(docURL, baseDir string) (string, bool) {
	switch {
	case schemaURIIsAbsolute(docURL):
		key, err := ResolveSchemaURI(docURL, "")
		return key, err == nil
	case docURL != "" && baseDir == "":
		key, err := ResolveSchemaURI(uripath.ToSlash(docURL), "")
		return key, err == nil
	case docURL != "":
		key, err := ResolveSchemaURI(path.Base(uripath.ToSlash(docURL)), baseDir)
		return key, err == nil
	case uriScheme(baseDir) != "":
		return baseDir, true
	}
	return "", false
}

// ResolveSchemaURI resolves a schema-location reference ref against a base
// (the FULL location of the schema that contains the reference) and returns
// the canonical name to hand to the configured loader.
//
// It is the single canonical URI-resolution helper shared by the xsd nested
// xs:include/xs:import/xs:redefine path and by xslt3's schema loader, so the
// two layers cannot drift apart again. Resolution branches on ref/base type:
//
//   - An ABSOLUTE-URI ref (it has its own scheme — with or without a "//"
//     authority, e.g. "https://cdn/part.xsd", "mem:/schemas/s.xsd",
//     "urn:schemas:s", "file:/tmp/s") addresses its own location and is
//     returned UNCHANGED, regardless of base. It must never be filepath.Join'ed
//     onto a base — doing so collapses the "//" authority separator (dropping
//     the host) or produces a bogus "/work/mem:/schemas/s".
//
//   - A relative ref against a URI base (the base has a scheme) is resolved
//     with RFC 3986 [url.URL.ResolveReference] semantics. The base's last path
//     segment is treated as the document and replaced, so a sibling "part.xsd"
//     against "mem:/schemas/main.xsd" resolves to "mem:/schemas/part.xsd". The
//     base authority is preserved, so a root-relative "/p" keeps scheme+host
//     and "../" applies dot-segment rules — never filepath collapsing. The
//     base's OmitHost flag is re-applied when the base had no authority (e.g.
//     "mem:/..." stays "mem:/...", never "mem:///...") while canonical
//     empty-authority bases like "file:///..." keep their "///".
//
//   - Otherwise (a genuine local filesystem base and ref) the historical
//     [filepath] join + ".."-escape guard is used unchanged: ref is joined onto
//     base and a result that ascends above base via ".." is rejected with
//     [errSchemaPathEscape]. The base here is a DIRECTORY (the containing dir of
//     the schema), matching the xsd compiler's BaseDir convention.
//
// The escape guard mirrors the defense-in-depth path normalization in xinclude
// (#420/#425); the configured loader may further constrain, but catching the
// escape here gives consistent behavior across loaders.
func ResolveSchemaURI(ref, base string) (string, error) {
	// Absolute-URI ref: address its own location verbatim.
	if uriScheme(ref) != "" {
		return ref, nil
	}
	// Relative ref against a URI base: resolve per RFC 3986.
	if uriScheme(base) != "" {
		return resolveURIReference(base, ref)
	}
	// Local filesystem base + ref. Resolve in FORWARD-SLASH space so the
	// returned name — used as a key into the configured fs.FS, whose contract
	// mandates '/' (io/fs.ValidPath) — never gains backslashes on Windows, where
	// filepath.Join/Rel/Separator would otherwise produce "schemas\\x" and miss
	// every MapFS / os.DirFS key. The shape-based detection and slash math are
	// GOOS-independent, so the escape guard is exercised identically on POSIX.
	slashRef := uripath.ToSlash(ref)
	if base == "" {
		return path.Clean(slashRef), nil
	}
	slashBase := uripath.ToSlash(base)
	p := path.Join(slashBase, slashRef)
	rel, err := filepath.Rel(slashBase, p)
	if err != nil {
		// Rel only fails when one is absolute and the other isn't;
		// nothing actionable here — accept and let the loader decide.
		return p, nil //nolint:nilerr
	}
	rel = uripath.ToSlash(rel)
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return "", fmt.Errorf("%w: %q", errSchemaPathEscape, ref)
	}
	return p, nil
}

// resolveURIReference resolves a relative reference against a URI base using
// RFC 3986 semantics. The base is the FULL schema URI (e.g.
// "https://example.com/s/main.xsd"); url.URL.ResolveReference treats its last
// path segment as the document and replaces it, so a sibling "part.xsd"
// resolves to "https://example.com/s/part.xsd".
//
// It re-applies the base's OmitHost flag when the base had no authority and the
// resolved reference introduced none: net/url's ResolveReference returns a
// fresh URL that does NOT carry OmitHost, so a no-authority base like
// "mem:/schemas/main.xsd" would otherwise emit an empty "//" authority
// ("mem:///schemas/part.xsd") and miss an exact-match loader keyed on
// "mem:/...". Canonical empty-authority bases like "file:///..." (OmitHost
// false) keep their "///".
func resolveURIReference(baseURI, ref string) (string, error) {
	base, err := url.Parse(baseURI)
	if err != nil {
		return "", fmt.Errorf("%w: invalid base URI %q", errSchemaPathEscape, baseURI)
	}
	refURL, err := url.Parse(ref)
	if err != nil {
		return "", fmt.Errorf("%w: invalid reference %q", errSchemaPathEscape, ref)
	}
	resolved := base.ResolveReference(refURL)
	if base.OmitHost && resolved.Host == "" && resolved.User == nil {
		resolved.OmitHost = true
	}
	return resolved.String(), nil
}
