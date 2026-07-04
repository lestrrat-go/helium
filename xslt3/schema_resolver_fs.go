package xslt3

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"path/filepath"
	"time"

	"github.com/lestrrat-go/helium/internal/uripath"
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
		// The xsd compiler demotes an ordinary xs:import load failure to a
		// non-fatal warning ("Failed to locate a schema ... Skipping the
		// import."). A resource-limit breach must NOT be silently demoted, or
		// the cap is defeated for a nested xs:import target. Mark it so the xsd
		// import path (which checks xsd.FatalSchemaLoader) treats it as fatal,
		// while keeping ErrResourceTooLarge in the chain so callers at the
		// xslt3 boundary can still errors.Is it.
		//
		// A default-deny POLICY denial ([errSchemaResolverDenied] — no
		// URIResolver configured, filesystem access is opt-in) must ALSO stay
		// fatal: unlike a configured resolver that merely lacks the target (a
		// fetch miss the xsd compiler may demote to a warning, since
		// schemaLocation is only a hint), a policy denial silently continuing
		// would hand back a half-assembled schema and mask that host access was
		// refused. Marking it here routes it through xsd.IsFatalSchemaLoad while
		// preserving the "no URIResolver configured" message for callers.
		if errors.Is(err, ErrResourceTooLarge) || errors.Is(err, errSchemaResolverDenied) {
			err = fatalSchemaLoadError{err}
		}
		return nil, &fs.PathError{Op: "open", Path: name, Err: err}
	}
	return &schemaResolverFile{name: name, r: bytes.NewReader(data), size: int64(len(data))}, nil
}

// errSchemaResolverDenied marks a nested-schema load refused by the compile-time
// default-deny policy (no URIResolver configured — filesystem access is opt-in).
// It distinguishes a POLICY DENIAL from a resolver fetch MISS (a configured
// resolver that simply lacks the target): the former must stay fatal, the latter
// may be demoted to a warning by the xsd compiler (schemaLocation is only a
// hint). [schemaResolverFS.Open] tags an error carrying it [fatalSchemaLoadError]
// so [xsd.IsFatalSchemaLoad] recognizes it while the "no URIResolver configured"
// message is preserved for callers.
var errSchemaResolverDenied = errors.New("xslt3: nested-schema load denied by default-deny policy")

// errSchemaContentInvalid marks a schema-load failure that occurred AFTER the
// bytes were successfully fetched — a malformed XML parse error, or an invalid
// XSD / schema-construction failure reported by the xsd compiler. It is the
// phase boundary of a schema load: the fetch phase ([compiler.loadSchemaBytes])
// returns its errors UNTAGGED (a genuine fetch miss the top-level import-schema
// path may fall back on), while every post-fetch return from
// [compiler.compileSchemaFromURI] carries this sentinel. The top-level fallback
// consults [isSchemaContentError] so a fetched-but-invalid schema-location is
// fatal (not masked by a matching precompiled ImportSchemas entry) exactly like
// the nested path — one consistent fetch-miss / content / denial taxonomy.
var errSchemaContentInvalid = errors.New("xslt3: schema content invalid")

// isSchemaContentError reports whether err (or anything in its chain) is a
// post-fetch content error tagged with [errSchemaContentInvalid] — the bytes
// loaded but were unusable (malformed XML or invalid XSD). Such an error must
// never be papered over by the precompiled-schema fallback: the schema-location
// was reachable, so masking it would silently swap an authoritative-but-broken
// schema for a precompiled one.
func isSchemaContentError(err error) bool {
	return errors.Is(err, errSchemaContentInvalid)
}

// fatalSchemaLoadError marks a schema-load failure that the xsd compiler must
// treat as fatal rather than demoting to a warning. It satisfies
// [xsd.FatalSchemaLoader] and unwraps to the underlying error so the original
// cause (e.g. [ErrResourceTooLarge]) remains discoverable via errors.Is /
// errors.As across the xsd→xslt3 boundary.
type fatalSchemaLoadError struct{ err error }

func (e fatalSchemaLoadError) Error() string { return e.err.Error() }

func (e fatalSchemaLoadError) Unwrap() error { return e.err }

func (e fatalSchemaLoadError) FatalSchemaLoad() bool { return true }

// isFatalSchemaLoadError reports whether err (or anything in its chain) is a
// fatal schema-load condition that must never be demoted to a warning or papered
// over by a fallback to a pre-compiled schema. It is the SINGLE classifier on
// the xslt3 side: it delegates the cross-package conditions (path escape,
// import-depth overflow, and any [xsd.FatalSchemaLoader]) to [xsd.IsFatalSchemaLoad]
// — the one source of truth shared with the xsd compiler — and additionally
// recognizes two xslt3-package sentinels the TOP-LEVEL schema-location load
// returns directly (unwrapped, so they are not yet a FatalSchemaLoader at that
// point): [ErrResourceTooLarge], the per-resource cap sentinel, and
// [errSchemaResolverDenied], the default-deny policy denial ("no URIResolver
// configured"). A policy denial on the top-level import-schema path must stay
// fatal — falling through to the precompiled fallback would let a
// no-resolver-configured schema-location silently compile via a registered
// ImportSchemas entry, bypassing the secure-by-default policy.
func isFatalSchemaLoadError(err error) bool {
	return errors.Is(err, ErrResourceTooLarge) ||
		errors.Is(err, errSchemaResolverDenied) ||
		xsd.IsFatalSchemaLoad(err)
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
// directory-like path from xml:base processing (e.g. "/a/b"); uripath.LocalBaseDir
// distinguishes the two so a directory base is not truncated.
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

	// Local filesystem base (a FILE path): resolve with forward-slash
	// (path) semantics so the result uses '/' on every OS. uripath.IsAbsolutePath
	// recognizes both POSIX- and Windows-absolute shapes regardless of GOOS.
	if uripath.IsAbsolutePath(ref) {
		return ref, nil
	}
	return uripath.JoinLocalBaseDir(uripath.LocalBaseDir(baseURI), ref), nil
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

// readCloserToBytes drains and closes a resolver-provided reader, bounding the
// read at limit (0 = MaxResourceBytes default, <0 = unbounded).
func readCloserToBytes(rc io.ReadCloser, limit int64) ([]byte, error) {
	defer func() { _ = rc.Close() }()
	return readResourceBounded(rc, limit)
}
