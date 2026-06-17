package xslt3

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"net/url"
	"path/filepath"
	"strings"
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
// The xsd compiler forms the name handed to Open via
// filepath.Join(baseDir, schemaLocation), where baseDir is seeded from
// filepath.Dir(parentSchemaURI). When the parent schema URI is an absolute
// URL (e.g. https://example.com/s/main.xsd), filepath collapses the
// "scheme://" authority separator down to "scheme:/", and the name reaching
// Open is the lossy "https:/example.com/s/part.xsd". The collapsed string
// alone is ambiguous — from "scheme:/X" you cannot tell whether the first
// segment of X is a URI authority/host or a path component.
//
// To avoid that ambiguity, the adapter keeps the ORIGINAL (un-collapsed)
// base URI and reconstructs the nested reference from it: it recovers the
// relative schema-location as filepath.Rel(filepath.Dir(baseURI), name) —
// undoing the join the xsd compiler performed — and resolves it against the
// base URI with net/url ResolveReference, applying standard RFC 3986 URI
// resolution (correct for http/https/file/ftp and for relative "../"/subdir
// references alike). When the base URI carries no URL scheme — i.e. it is a
// genuine local filesystem path — the name is forwarded verbatim, preserving
// the existing local-path and default-deny behavior.
type schemaResolverFS struct {
	ctx     context.Context //nolint:containedctx // loader needs the request context; FS has no per-Open ctx
	load    func(ctx context.Context, uri string) ([]byte, error)
	baseURI string
}

// Open implements [fs.FS]. It loads the named schema document through the
// configured byte-loader and returns it as an in-memory file. Any loader
// error (including the default-deny "no URIResolver configured" case) is
// returned as a *fs.PathError so fs.ReadFile surfaces it.
func (s schemaResolverFS) Open(name string) (fs.File, error) {
	uri := s.resolveName(name)
	data, err := s.load(s.ctx, uri)
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: uri, Err: err}
	}
	return &schemaResolverFile{name: uri, r: bytes.NewReader(data), size: int64(len(data))}, nil
}

// resolveName recovers the canonical URI for a nested schema reference whose
// authority separator the xsd compiler's filepath.Join collapsed.
//
// name is filepath.Join(filepath.Dir(baseURI), schemaLocation). Recovering the
// original relative schema-location is therefore filepath.Rel against
// filepath.Dir(baseURI); resolving it against the original base URI via
// net/url then yields the correct absolute URI under standard URI rules. This
// works for arbitrarily nested includes because every name the xsd compiler
// produces is a filepath.Join under the (collapsed) tree rooted at the
// original base directory, so the recovered Rel is always the correct relative
// reference from that base.
//
// If baseURI has no URL scheme (a genuine local filesystem path) or the
// reconstruction cannot be performed, name is returned unchanged.
func (s schemaResolverFS) resolveName(name string) string {
	base, err := url.Parse(s.baseURI)
	if err != nil || base.Scheme == "" {
		// No base URI, or a local filesystem path: forward verbatim.
		return name
	}
	ref, err := filepath.Rel(filepath.Dir(s.baseURI), name)
	if err != nil {
		return name
	}
	ref = filepath.ToSlash(ref)
	refURL, err := url.Parse(ref)
	if err != nil {
		return name
	}
	return base.ResolveReference(refURL).String()
}

// resolveSchemaURI resolves a schema-location reference against a base URI.
//
// When the base is an absolute URL (has a scheme), resolution follows RFC 3986
// via net/url ResolveReference, so the result preserves the authority and
// applies "../"/subdir semantics correctly — and crucially is NOT collapsed by
// filepath. When the base is a local filesystem path (no scheme), or either
// side is empty/absolute, the existing filepath-based join is used so local
// schema resolution and the default-deny behavior are unchanged.
func resolveSchemaURI(ref, baseURI string) string {
	if ref == "" || baseURI == "" {
		return ref
	}
	if strings.Contains(ref, "://") || filepath.IsAbs(ref) {
		return ref
	}
	base, err := url.Parse(baseURI)
	if err != nil || base.Scheme == "" {
		// Local filesystem path: keep the historical filepath join.
		return filepath.Join(filepath.Dir(baseURI), ref)
	}
	refURL, err := url.Parse(ref)
	if err != nil {
		return filepath.Join(filepath.Dir(baseURI), ref)
	}
	return base.ResolveReference(refURL).String()
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
