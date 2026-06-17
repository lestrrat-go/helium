package xslt3

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"net/url"
	"path/filepath"
	"time"
)

// schemaResolverFS adapts a byte-loader (backed by the XSLT engine's
// configured URIResolver / HTTPClient and its default-deny policy) into an
// [fs.FS] suitable for [xsd.Compiler.FS]. It is how nested xs:include /
// xs:import / xs:redefine references inside a resolver-loaded schema are
// routed through the SAME resolver rather than the xsd compiler's default
// os.Open. Without it, a schema fetched from an in-memory or HTTP resolver
// would have its nested references read off the local filesystem, bypassing
// the secure-by-default policy.
//
// The xsd compiler is seeded with the parent schema URI as its BaseDir and
// resolves nested schema-locations URI-aware (RFC 3986 for relative refs,
// pass-through for absolute-URI refs). The name reaching Open is therefore
// already the canonical nested URI — an absolute https/file URI, or a
// relative reference resolved against the base — so the adapter forwards it
// to the loader verbatim. No string repair of a filepath-collapsed name is
// needed (or attempted): that collapsing no longer happens.
type schemaResolverFS struct {
	ctx  context.Context //nolint:containedctx // loader needs the request context; FS has no per-Open ctx
	load func(ctx context.Context, uri string) ([]byte, error)
}

// Open implements [fs.FS]. It loads the named schema document through the
// configured byte-loader and returns it as an in-memory file. The name is the
// canonical nested-schema URI already resolved by the xsd compiler, so it is
// forwarded unchanged. Any loader error (including the default-deny "no
// URIResolver configured" case) is returned as a *fs.PathError so fs.ReadFile
// surfaces it.
func (s schemaResolverFS) Open(name string) (fs.File, error) {
	data, err := s.load(s.ctx, name)
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: err}
	}
	return &schemaResolverFile{name: name, r: bytes.NewReader(data), size: int64(len(data))}, nil
}

// uriScheme reports the scheme of s when s is an absolute URI reference (has a
// scheme per RFC 3986, e.g. "https://...", "file:/...", "mem:/...", "urn:..."),
// or "" otherwise. A bare local filesystem path — even an absolute one like
// "/tmp/x" — has no scheme; a single-letter scheme is rejected so a Windows
// drive letter ("C:\x") keeps its filepath handling. This mirrors the xsd
// package's own uriScheme detection so the two layers agree on what counts as
// an absolute URI.
func uriScheme(s string) string {
	u, err := url.Parse(s)
	if err != nil || len(u.Scheme) < 2 {
		return ""
	}
	return u.Scheme
}

// resolveSchemaURI resolves a schema-location reference against a base URI.
//
// An absolute-URI ref (it has a scheme — with or without a "//" authority, e.g.
// "https://other/x.xsd", "mem:/schemas/s.xsd", "urn:schemas:s", "file:/tmp/s")
// addresses its own location and is returned UNCHANGED. It must never be
// filepath.Join'ed onto a local base — doing so would produce a bogus path like
// "/work/mem:/schemas/s.xsd" and the resolver would be asked for the wrong URI.
//
// When the base is an absolute URL (has a scheme), a relative ref is resolved
// per RFC 3986 via net/url ResolveReference, so the result preserves the
// authority and applies "../"/subdir semantics correctly — and crucially is NOT
// collapsed by filepath. Otherwise (a local filesystem base, or an absolute
// local path ref), the historical filepath-based join is used so local schema
// resolution and the default-deny behavior are unchanged.
//
// Absolute-URI detection matches the xsd package's uriScheme semantics
// (url.Parse + multi-character scheme), keeping the two layers consistent.
func resolveSchemaURI(ref, baseURI string) string {
	if ref == "" || baseURI == "" {
		return ref
	}
	// Absolute-URI ref (any scheme, // or not): pass through verbatim.
	if uriScheme(ref) != "" {
		return ref
	}
	if filepath.IsAbs(ref) {
		return ref
	}
	if uriScheme(baseURI) == "" {
		// Local filesystem base: keep the historical filepath join.
		return filepath.Join(filepath.Dir(baseURI), ref)
	}
	base, err := url.Parse(baseURI)
	if err != nil {
		return filepath.Join(filepath.Dir(baseURI), ref)
	}
	refURL, err := url.Parse(ref)
	if err != nil {
		return filepath.Join(filepath.Dir(baseURI), ref)
	}
	return base.ResolveReference(refURL).String()
}

// schemaCompileBaseDir maps a base URI/path to the value passed to
// [xsd.Compiler.BaseDir] so the xsd compiler resolves nested
// xs:include/xs:import/xs:redefine references correctly.
//
// The xsd compiler is URI-aware: for a URI base it replaces the last path
// segment via RFC 3986 resolution, so it needs the FULL schema URI; for a
// local filesystem base it filepath.Join's, so it needs the containing
// DIRECTORY. (A single-letter "scheme" is treated as a Windows drive letter,
// not a URI, matching the xsd package's own detection.)
func schemaCompileBaseDir(base string) string {
	if uriScheme(base) != "" {
		return base
	}
	return filepath.Dir(base)
}

// schemaResolverFile is a minimal read-only [fs.File] backed by an in-memory
// byte slice.
type schemaResolverFile struct {
	name string
	r    *bytes.Reader
	size int64
}

func (f *schemaResolverFile) Stat() (fs.FileInfo, error) { return schemaResolverFileInfo{f}, nil }

func (f *schemaResolverFile) Read(p []byte) (int, error) {
	return f.r.Read(p) //nolint:wrapcheck // io.Reader passthrough
}

func (f *schemaResolverFile) Close() error { return nil }

type schemaResolverFileInfo struct {
	f *schemaResolverFile
}

func (i schemaResolverFileInfo) Name() string       { return i.f.name }
func (i schemaResolverFileInfo) Size() int64        { return i.f.size }
func (i schemaResolverFileInfo) Mode() fs.FileMode  { return 0o444 }
func (i schemaResolverFileInfo) ModTime() time.Time { return time.Time{} }
func (i schemaResolverFileInfo) IsDir() bool        { return false }
func (i schemaResolverFileInfo) Sys() any           { return nil }

// readCloserToBytes drains and closes a resolver-provided reader.
func readCloserToBytes(rc io.ReadCloser) ([]byte, error) {
	defer func() { _ = rc.Close() }()
	return io.ReadAll(rc) //nolint:wrapcheck // caller wraps with context
}
