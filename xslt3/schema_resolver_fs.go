package xslt3

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"path/filepath"
	"time"

	"github.com/lestrrat-go/helium/xsd"
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

// resolveSchemaURI resolves a schema-location reference against a base URI.
//
// The URI cases (absolute-URI ref pass-through, and RFC 3986 resolution against
// a URI base with OmitHost preservation) are delegated to [xsd.ResolveSchemaURI]
// — the single canonical helper shared with the xsd nested-include path, so the
// two layers cannot drift apart again. Its error (e.g. an invalid URI reference
// such as "%zz.xsd") is PROPAGATED: an unresolvable reference is rejected, never
// silently filepath-collapsed into a corrupted name (which would drop the
// authority, turning "https://host/%zz.xsd" into "https:/host/%zz.xsd").
//
// Only the LOCAL filesystem base case is handled here, because xslt3's base is
// the full FILE path of the referencing stylesheet or document. An absolute
// local ref is returned as-is; otherwise the ref is joined onto the base's
// directory. The base may be a full file path (e.g. "/a/b/style.xsl") or a
// directory-like path from xml:base processing (e.g. "/a/b"); baseURIDir
// distinguishes the two so a directory base is not truncated by filepath.Dir.
func resolveSchemaURI(ref, baseURI string) (string, error) {
	if ref == "" || baseURI == "" {
		return ref, nil
	}

	// Absolute-URI ref, or any ref against a URI base: defer to the shared
	// canonical resolver (RFC 3986 + OmitHost preservation). Propagate any
	// error instead of falling back to a host-dropping filepath join.
	if xsd.URIScheme(ref) != "" || xsd.URIScheme(baseURI) != "" {
		return xsd.ResolveSchemaURI(ref, baseURI)
	}

	// Local filesystem base (a FILE path): keep historical filepath semantics.
	if filepath.IsAbs(ref) {
		return ref, nil
	}
	return filepath.Join(baseURIDir(baseURI), ref), nil
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
	if xsd.URIScheme(base) != "" {
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
	return readResourceBounded(rc)
}
