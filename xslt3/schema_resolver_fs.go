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

// Open implements [fs.FS] and is the xslt3→xsd BOUNDARY for NESTED schema loads
// (xs:include/xs:import/xs:redefine reached while an xsl:import-schema or
// source-document schema compiles). The name is the canonical nested-schema URI
// already resolved by the xsd compiler, so it is forwarded to the byte-loader
// unchanged. The loader's error is TRANSLATED into XSD's fs.FS classification
// vocabulary so the xsd nested-load classifier ([xsd.readNestedSchema]) demotes
// or keeps-fatal CONSISTENTLY with the xslt3 side — the boundary is total:
//   - a CONFIRMED benign resolution miss ([isDemotableSchemaMiss] — the positively
//     tagged errSchemaResolutionMiss / not-found, e.g. a resolver returning a bare
//     errors.New("not found") that does NOT itself satisfy fs.ErrNotExist) is
//     re-expressed as a *fs.PathError{Op:"open", Err: fs.ErrNotExist} so XSD's
//     Open-path isBenignResolutionMiss demotes the OPTIONAL nested include to a
//     warning-and-skip instead of over-rejecting it (schemaLocation is a hint);
//   - EVERYTHING ELSE (a post-open read failure, a permission denial, a
//     resource-cap breach, a policy denial, or any other/ambiguous error) is
//     wrapped [fatalSchemaLoadError] so xsd.IsFatalSchemaLoad (and XSD's
//     notDemotable veto) keeps it FATAL regardless of the inner errno — a
//     non-miss error must never be demoted through the boundary. The original
//     chain is preserved (Unwrap), so errors.Is(fs.ErrPermission)/ErrResourceTooLarge
//     still hold for callers at the xslt3 boundary.
func (s schemaResolverFS) Open(name string) (fs.File, error) {
	data, err := s.load(s.ctx, name)
	if err == nil {
		return &schemaResolverFile{name: name, r: bytes.NewReader(data), size: int64(len(data))}, nil
	}
	if isDemotableSchemaMiss(err) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: schemaMissNotExistError{err}}
	}
	return nil, &fs.PathError{Op: "open", Path: name, Err: fatalSchemaLoadError{err}}
}

// schemaMissNotExistError adapts a CONFIRMED xslt3 resolution miss into XSD's fs.FS
// vocabulary: it satisfies errors.Is(_, fs.ErrNotExist) — so xsd's Open-path
// isBenignResolutionMiss demotes the optional nested load — while preserving the
// original loader error's message and unwrap chain. It reports fs.ErrNotExist via
// Is rather than embedding the sentinel so the original chain (carrying the
// resolver's own message and any errSchemaResolutionMiss tag) stays intact.
type schemaMissNotExistError struct{ cause error }

func (e schemaMissNotExistError) Error() string { return e.cause.Error() }

func (e schemaMissNotExistError) Unwrap() error { return e.cause }

func (schemaMissNotExistError) Is(target error) bool { return target == fs.ErrNotExist }

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
// tags ONLY a confirmed resolution miss [errSchemaResolutionMiss] (the sole
// demotable outcome), while every post-fetch return from
// [compiler.compileSchemaFromURI] carries this sentinel. A content-tagged error
// is NOT a resolution miss, so the positive-tag partition ([isDemotableSchemaMiss]
// is false) keeps it fatal — a fetched-but-invalid schema-location is never
// masked by a matching precompiled ImportSchemas entry, exactly like the nested
// path.
var errSchemaContentInvalid = errors.New("xslt3: schema content invalid")

// errSchemaResolutionMiss is the POSITIVE tag the xslt3 schema loaders apply at
// the ONE place a fetch outcome is a CONFIRMED benign RESOLUTION miss: the
// resolver could not produce a reader for the target ([compiler.loadSchemaBytes]'s
// Resolve failure / [fetchViaResolver]'s ResolveURI failure), or an HTTP fetch
// returned a definite NOT-FOUND status (404/410). It mirrors the xsd side's
// errSchemaFetchMiss: schemaLocation is only a hint, so ONLY a confirmed
// resolution miss may be demoted (a source-document schema skipped under lax, or
// a top-level import-schema falling back to a precompiled entry). Every OTHER
// fetch failure — a reader obtained then failing during Read (post-open), an
// HTTP 401/403/5xx/other-non-2xx, a connection/transport error, or any
// untagged/ambiguous error — is NOT tagged and is therefore FATAL, mirroring the
// xsd notDemotable/errNestedSchemaReadAfterOpen discipline (fail-closed).
var errSchemaResolutionMiss = errors.New("xslt3: schema resolution miss")

// schemaResolutionMissError wraps a benign resolution-miss cause so it satisfies
// errors.Is(_, errSchemaResolutionMiss) WITHOUT forming a multi-error: it keeps a
// SINGLE Unwrap() chain to the cause (so errors.Is(_, fs.ErrNotExist) etc. still
// traverse and the message is preserved) and reports itself as the tag via Is.
// A [fmt.Errorf] "%w: %w" (two verbs) would instead produce an Unwrap() []error
// that [isFatalSchemaLoadError]'s multi-error guard treats as fatal — defeating
// the demotion — so the tag is applied through this single-chain wrapper.
type schemaResolutionMissError struct{ cause error }

func (e schemaResolutionMissError) Error() string { return e.cause.Error() }

func (e schemaResolutionMissError) Unwrap() error { return e.cause }

func (schemaResolutionMissError) Is(target error) bool { return target == errSchemaResolutionMiss }

// markResolutionMiss tags err as a confirmed benign resolution miss (see
// [errSchemaResolutionMiss]) while preserving its message and unwrap chain. A nil
// err is returned unchanged.
func markResolutionMiss(err error) error {
	if err == nil {
		return nil
	}
	return schemaResolutionMissError{err}
}

// isDemotableSchemaMiss reports whether err is a CONFIRMED benign schema
// resolution miss that may be demoted — the SOLE demotable classification on the
// xslt3 side, the positive-tag counterpart of the xsd compiler's nestedFetchMiss.
// It requires the POSITIVE [errSchemaResolutionMiss] tag AND that err is not
// otherwise fatal ([isFatalSchemaLoadError] — a permission/multi-error/cap/policy
// cause overrides the tag). Together with [isFatalSchemaLoadError] it PARTITIONS
// the space: anything not positively demotable (a post-open read failure, an
// HTTP 401/403/5xx, an untagged/ambiguous error, a content error) is fatal.
func isDemotableSchemaMiss(err error) bool {
	return errors.Is(err, errSchemaResolutionMiss) && !isFatalSchemaLoadError(err)
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
// the xslt3 side and MIRRORS the xsd compiler's nested-load demotion veto
// (`notDemotable`), so the top-level import-schema and source-document loaders
// agree with the nested include/import/redefine path on what may be demoted.
//
// It delegates the cross-package sentinel conditions (path escape, import-depth
// overflow, and any [xsd.FatalSchemaLoader]) to [xsd.IsFatalSchemaLoad] — the
// one source of truth shared with the xsd compiler — and additionally treats as
// fatal:
//
//   - two xslt3-package sentinels the TOP-LEVEL schema-location load returns
//     directly (unwrapped, so they are not yet a FatalSchemaLoader at that
//     point): [ErrResourceTooLarge], the per-resource cap sentinel, and
//     [errSchemaResolverDenied], the default-deny policy denial ("no URIResolver
//     configured") — falling through to the precompiled fallback would let a
//     no-resolver-configured schema-location silently compile via a registered
//     ImportSchemas entry, bypassing the secure-by-default policy;
//   - a PERMISSION denial ([fs.ErrPermission]) anywhere in the chain — a policy
//     denial from a configured resolver/FS is NOT a benign fetch miss, so it must
//     never fall back to a precompiled schema;
//   - a MULTI-ERROR ([errors.Join] / any Unwrap() []error) — a single
//     schema-location fetch never legitimately produces one, and a join defeats
//     errors.Is first-match selection, so a benign miss could mask a fatal
//     sibling; rejecting the whole class is fail-closed.
func isFatalSchemaLoadError(err error) bool {
	return errors.Is(err, ErrResourceTooLarge) ||
		errors.Is(err, errSchemaResolverDenied) ||
		errors.Is(err, fs.ErrPermission) ||
		schemaLoadHasMultiError(err) ||
		xsd.IsFatalSchemaLoad(err)
}

// schemaLoadHasMultiError reports whether err or anything in its LINEAR unwrap
// chain (following single Unwrap() error links) implements Unwrap() []error —
// i.e. is an [errors.Join] / multi-error. It mirrors the xsd compiler's
// containsMultiError guard so a schema-load failure carrying a joined error tree
// (where a benign sibling could mask a fatal one under errors.Is) is never
// demoted to a fetch miss.
func schemaLoadHasMultiError(err error) bool {
	for e := err; e != nil; {
		if _, ok := e.(interface{ Unwrap() []error }); ok {
			return true
		}
		u, ok := e.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		e = u.Unwrap()
	}
	return false
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
