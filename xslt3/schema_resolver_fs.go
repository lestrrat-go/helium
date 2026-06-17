package xslt3

import (
	"bytes"
	"context"
	"io"
	"io/fs"
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
// The names handed to Open are produced by the xsd compiler via
// filepath.Join(baseDir, schemaLocation), where baseDir is the directory of
// the parent schema's URI. When the parent schema URI is an absolute URL
// (e.g. https://example.com/s/main.xsd), filepath.Dir/Join collapse the
// "scheme://" authority separator down to "scheme:/" — so the name reaching
// Open is the malformed "https:/example.com/s/part.xsd". canonicalizeName
// restores the dropped slash before the loader (and thus the resolver) sees
// it, so nested loads target the correct canonical URI. Genuine local paths
// (no URL scheme) are forwarded verbatim.
type schemaResolverFS struct {
	ctx  context.Context //nolint:containedctx // loader needs the request context; FS has no per-Open ctx
	load func(ctx context.Context, uri string) ([]byte, error)
}

// Open implements [fs.FS]. It loads the named schema document through the
// configured byte-loader and returns it as an in-memory file. Any loader
// error (including the default-deny "no URIResolver configured" case) is
// returned as a *fs.PathError so fs.ReadFile surfaces it.
func (s schemaResolverFS) Open(name string) (fs.File, error) {
	name = canonicalizeName(name)
	data, err := s.load(s.ctx, name)
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: err}
	}
	return &schemaResolverFile{name: name, r: bytes.NewReader(data), size: int64(len(data))}, nil
}

// canonicalizeName repairs a name whose "scheme://authority" was collapsed to
// "scheme:/authority" by filepath.Dir/Join (which treat the URI as a local
// path and squash the doubled slash). It restores the missing slash so the
// loader receives the canonical absolute URI. Names without a URL scheme — i.e.
// genuine local filesystem paths — are returned unchanged, so local-schema
// resolution and the default-deny tests are unaffected.
func canonicalizeName(name string) string {
	scheme, rest, ok := splitURLScheme(name)
	if !ok {
		return name
	}
	// A canonical URL already has "scheme://"; only repair when filepath
	// collapsed it to "scheme:/" with a single leading slash on the rest.
	if strings.HasPrefix(rest, "//") {
		return name
	}
	if !strings.HasPrefix(rest, "/") {
		// Opaque or relative remainder (e.g. "mailto:x") — leave as-is.
		return name
	}
	return scheme + "://" + rest[1:]
}

// splitURLScheme splits name into its URL scheme and the remainder after the
// "scheme:" prefix. It returns ok=false when name does not begin with a valid
// RFC 3986 scheme (ALPHA *( ALPHA / DIGIT / "+" / "-" / "." ) ":"), which
// covers all local filesystem paths.
func splitURLScheme(name string) (string, string, bool) {
	colon := strings.IndexByte(name, ':')
	if colon <= 0 {
		return "", "", false
	}
	scheme := name[:colon]
	// Require at least two characters so a Windows drive letter ("C:\path")
	// is never mistaken for a URL scheme.
	if len(scheme) < 2 || !isASCIILetter(scheme[0]) {
		return "", "", false
	}
	for i := 1; i < len(scheme); i++ {
		c := scheme[i]
		if isASCIILetter(c) || (c >= '0' && c <= '9') || c == '+' || c == '-' || c == '.' {
			continue
		}
		return "", "", false
	}
	return scheme, name[colon+1:], true
}

func isASCIILetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
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
