package xsd

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"path"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/iofs"
	"github.com/lestrrat-go/helium/internal/iolimit"
	"github.com/lestrrat-go/helium/internal/uripath"
	"github.com/lestrrat-go/helium/internal/xmlchar"
)

// errImportDepthExceeded signals that xs:import recursion reached the
// configured limit. processIncludes propagates this error rather than
// treating it as a warning the way it treats ordinary I/O failures.
var errImportDepthExceeded = errors.New("xsd: max import depth exceeded")

// errIncludeDepthExceeded signals that xs:include/xs:redefine nesting reached
// the configured limit. It is a secondary guard behind the includeVisited
// loaded-set; processNestedIncludes returns it as a fatal compilation error.
var errIncludeDepthExceeded = errors.New("xsd: max include depth exceeded")

// errSchemaPathEscape signals that a schemaLocation joined onto baseDir
// would escape upward via ".." segments. processIncludes surfaces this
// as a fatal error rather than swallowing it as a generic I/O warning,
// so the containment violation is visible to callers.
var errSchemaPathEscape = errors.New("xsd: schema location escapes base directory")

// errSchemaContentInvalid marks a CONTENT failure of a nested schema
// (xs:include/xs:import/xs:redefine target): the document was fetched and opened
// but is not well-formed XML, or its root element is not <xs:schema>. Unlike a
// fetch/resolution failure — which demotes to a warning because schemaLocation
// is only a hint (src-include.1 / src-import) — a content failure is a fatal
// schema error: the located document exists but is invalid. It is one of the
// two explicit tags in the three-way nested-load taxonomy (the other being
// [errSchemaFetchMiss]); the classifier ([nestedLoadFailureFatal]) is
// fail-closed, so this tag documents intent and is not strictly required to make
// a content failure fatal.
var errSchemaContentInvalid = errors.New("xsd: nested schema content is invalid")

// errSchemaFetchMiss is the POSITIVE tag applied — at exactly ONE place,
// [readNestedSchema]'s benign RESOLUTION-phase misses — to the only nested-load
// failures that may be demoted to a warning. [nestedFetchMiss] keys on this tag
// ALONE, never on a raw [fs.ErrNotExist]/[fs.ErrInvalid] errno. schemaLocation is
// only a hint (src-include.1 / src-import / src-redefine.1), so a target that
// cannot be RESOLVED — a FAILED Open reporting a benign missing/unresolvable
// errno, or an atomic fs.ReadFile fallback reporting a CANONICAL "file not
// found" — is skipped (matching libxml2); but the same errno arising in a
// LATER/unclassifiable phase — a post-Open read, a message-wrapped fs.ReadFile
// fallback error, content parsing, an external-entity read, a non-xs:schema
// root — carries no tag, so it stays FATAL. A SECURITY/POLICY denial (path
// escape, depth or byte-cap breach, a [FatalSchemaLoader] refusal such as
// xslt3's default-deny "no URIResolver configured" policy, a permission denial,
// or a refusal from the default deny-all FS) is likewise never tagged, so
// denials stay fatal. Making the demotable decision a single positive tag means
// no downstream error can masquerade as a resolution miss, however its errno
// chain reads. The wrapped error chain is preserved for errors.Is/errors.As.
// It is one of the tags in the nested-load taxonomy (the CONTENT-failure
// counterpart being [errSchemaContentInvalid]).
var errSchemaFetchMiss = errors.New("xsd: nested schema fetch miss")

// errSchemaTooLarge signals that a nested schema (xs:include/xs:import/
// xs:redefine target) exceeded [maxNestedSchemaSize] while being read. It is a
// resource-limit guard, so it is classified fatal by [IsFatalSchemaLoad] and
// must not be silently demoted to an I/O warning on the xs:import path: a
// hostile schemaLocation (e.g. /dev/zero) must abort compilation rather than
// be swallowed.
var errSchemaTooLarge = errors.New("xsd: schema resource exceeds size limit")

// errNestedSchemaReadAfterOpen tags an error that occurred while READING a
// nested schema document AFTER its [fs.File] was successfully opened (a
// post-open streaming Read failure), distinguishing it from an Open/resolution
// miss. schemaLocation is only a hint, so a genuine FETCH/RESOLUTION miss (the
// Open step failing with [fs.ErrNotExist]/[fs.ErrInvalid]) is demoted to a
// warning and skipped; but a location that RESOLVED and OPENED and then failed
// to read is a real I/O failure that must ABORT compilation (fail-closed).
// Wrapping the read error with this sentinel makes [IsFatalSchemaLoad] classify
// it fatal — so [nestedFetchMiss] never demotes it, even when the underlying
// read error is itself [fs.ErrInvalid]/[fs.ErrNotExist]. The wrapped error chain
// is preserved for errors.Is/errors.As.
var errNestedSchemaReadAfterOpen = errors.New("xsd: nested schema read failed after open")

// maxNestedSchemaSize bounds the number of bytes read from any single nested
// schema document loaded via xs:include/xs:import/xs:redefine, so an endless or
// oversized source cannot exhaust memory. It mirrors xinclude's per-resource
// cap (10 MiB).
const maxNestedSchemaSize = 10 << 20 // 10 MiB

// readNestedSchema reads path through the configured fs.FS under a strict
// [maxNestedSchemaSize] byte cap, so an endless or oversized source
// (xs:include/xs:import/xs:redefine target) cannot exhaust memory; it replaces
// the unbounded fs.ReadFile every nested-schema loader used to call. It prefers
// the streaming [fs.File] from Open so an endless device (e.g. /dev/zero) is
// bounded while reading (iolimit reads one extra byte so a source that
// under-reports its size is still caught).
//
// Demotion to a warning is by POSITIVE TAG. A benign miss ([fs.ErrNotExist] or
// [fs.ErrInvalid]) from a FAILED Open with NO ReadFileFS fallback is wrapped in
// [errSchemaFetchMiss], the sole tag [nestedFetchMiss] demotes — a FAILED Open is by
// definition the resolution phase. EVERY other Open error is FATAL and returned
// UNTAGGED: a resource/policy denial ([IsFatalSchemaLoad]), the default deny-all FS
// ([iofs.DenyAll]), a permission denial ([fs.ErrPermission]), an outside-root policy
// error, or any other/ambiguous errno.
//
// The [fs.ReadFileFS] fallback (retried when Open reports a benign miss) may SERVE
// THE BYTES (a ReadFileFS-only FS whose Open is unsupported). Because fs.ReadFile is
// ATOMIC (open+read+close in one call), its errors are mostly phase-UNCLASSIFIABLE —
// a post-resolution read failure is indistinguishable from a genuine miss — so a
// fallback error is DEMOTED only when it is a CANONICAL "file not found"
// ([isDirectNotExist]: the bare fs.ErrNotExist sentinel or a *fs.PathError with
// Op=="open" whose Err satisfies fs.ErrNotExist, the one shape that can never be a
// read failure). EVERY other fallback error — fs.ErrInvalid, a message-wrapped errno
// that is not such a PathError (even one wrapping fs.ErrNotExist), a non-"open" Op,
// anything else — is returned UNTAGGED and stays FATAL (fail-closed).
//
// A POST-OPEN read failure is wrapped in [errNestedSchemaReadAfterOpen] (fatal,
// untagged). The cap breach is [errSchemaTooLarge] (fatal). Because the tag is
// applied only at the FAILED-Open resolution phase, a downstream CONTENT/parse error
// whose chain happens to contain fs.ErrInvalid/fs.ErrNotExist is NOT tagged and stays
// fatal.
func (c *compiler) readNestedSchema(path string) ([]byte, error) {
	f, openErr := c.fsys.Open(path)
	if openErr != nil {
		// A FATAL open error aborts IMMEDIATELY and UNTAGGED: a resource/policy denial
		// ([IsFatalSchemaLoad]) or the default deny-all FS ([iofs.DenyAll], whose Open
		// returns a benign fs.ErrNotExist errno the FS type disambiguates).
		if _, denyAll := c.fsys.(iofs.DenyAll); denyAll || IsFatalSchemaLoad(openErr) {
			return nil, openErr //nolint:wrapcheck // callers/classifiers key on the original error
		}
		// A ReadFileFS may still SERVE THE BYTES even when its Open is UNSUPPORTED (a
		// ReadFileFS-only FS whose Open returns fs.ErrNotExist/fs.ErrInvalid), so
		// retry through the atomic fs.ReadFile for that FS kind when the Open error is
		// a benign miss OR an Open-unsupported signal ([isOpenMissOrUnsupported] — NOT
		// a permission/multi/fatal cause). fs.ReadFile is the AUTHORITATIVE path here,
		// so its OWN error is classified by [isDirectNotExist] (only a canonical
		// not-found demotes; fs.ErrInvalid and everything else stay fatal). A plain
		// fs.FS's fs.ReadFile would merely re-Open and reproduce the identical miss, so
		// the fallback is confined to a real ReadFileFS.
		if _, ok := c.fsys.(fs.ReadFileFS); ok && isOpenMissOrUnsupported(openErr) {
			data, rfErr := fs.ReadFile(c.fsys, path)
			if rfErr == nil {
				if int64(len(data)) > maxNestedSchemaSize {
					return nil, errSchemaTooLarge
				}
				return data, nil
			}
			// The fallback errored. fs.ReadFile is ATOMIC (open+read+close in one call),
			// so its errors are mostly phase-UNCLASSIFIABLE: a ReadFileFS whose ReadFile
			// RESOLVES the file but then fails mid-READ returns an errno indistinguishable
			// from a genuine miss. So the fallback demotes ONLY a CANONICAL filesystem
			// "file not found" at the RESOLUTION/OPEN phase ([isDirectNotExist]: the bare
			// fs.ErrNotExist sentinel or a *fs.PathError with Op=="open" whose Err satisfies
			// fs.ErrNotExist — including a real syscall.ENOENT from os.DirFS) — the one shape
			// that unambiguously denotes NON-EXISTENCE at resolution and can never be a
			// post-resolution read failure. EVERY other fallback error — fs.ErrInvalid, a
			// custom message wrap that is not such a PathError (even one wrapping
			// fs.ErrNotExist), a *fs.PathError with a non-open Op (e.g.
			// {Op:"read", Err: fs.ErrNotExist}, a post-open read failure), anything else —
			// is UNCLASSIFIABLE and returned UNTAGGED, so it stays FATAL (fail-closed).
			if !isDirectNotExist(rfErr) {
				return nil, rfErr //nolint:wrapcheck // unclassifiable fallback error; fail closed
			}
			return nil, fmt.Errorf("%w: %w", errSchemaFetchMiss, rfErr)
		}
		// Plain fs.FS (or a ReadFileFS whose Open error is not a benign/unsupported
		// miss): a failed Open IS the resolution answer.
		//
		// A non-file-scheme ABSOLUTE URI schemaLocation (http://, urn:, ...) is NOT a
		// local resource: a plain filesystem FS cannot resolve it and POSITIVELY signals
		// so with an FS-DEPENDENT errno — fs.ErrNotExist (os.Open / iofs.PermissiveRoot's
		// URI mapping) or fs.ErrInvalid ([os.DirFS], which rejects a non-fs.ValidPath
		// name). schemaLocation is only a hint (src-include.1 / src-import), so demote
		// the URI as a resolution MISS ONLY when the Open error is one of those two
		// local-cannot-resolve errnos AND not [notDemotable]. An OPAQUE Open error on a
		// non-file URI — a URI-AWARE Open-only fs.FS that actually FETCHED it and got an
		// "HTTP 500" / transport failure (neither fs.ErrInvalid nor fs.ErrNotExist) — is
		// a real fetch failure, NOT a local-resolution miss, and stays FATAL
		// (fail-closed). A URI-serving FS that CAN resolve the URI (xslt3's
		// schemaResolverFS, or a ReadFileFS handled above) serves the bytes, reports a
		// genuine miss as fs.ErrNotExist, or fails fatal via a FatalSchemaLoader, so it
		// never false-demotes here.
		if s := uriScheme(path); s != "" && s != "file" &&
			(errors.Is(openErr, fs.ErrInvalid) || errors.Is(openErr, fs.ErrNotExist)) &&
			!notDemotable(openErr) {
			return nil, fmt.Errorf("%w: %w", errSchemaFetchMiss, openErr)
		}
		// A genuinely-LOCAL path: demote ONLY a POSITIVE not-found
		// ([isBenignResolutionMiss] = fs.ErrNotExist); an fs.ErrPermission, an "outside
		// root" policy error, a MALFORMED-LOCAL fs.ErrInvalid, or any other/ambiguous
		// errno is NOT a confirmed absence and stays FATAL, returned verbatim and
		// UNTAGGED so nestedFetchMiss never demotes it (fail-closed).
		if !isBenignResolutionMiss(openErr) {
			return nil, openErr //nolint:wrapcheck // non-benign open error is fatal, not a miss
		}
		// A FAILED Open reporting a positive not-found IS the resolution phase: TAG it
		// so nestedFetchMiss, and nothing downstream, demotes it.
		return nil, fmt.Errorf("%w: %w", errSchemaFetchMiss, openErr)
	}
	data, exceeded, readErr := iolimit.ReadAll(f, maxNestedSchemaSize)
	_ = f.Close()
	if exceeded {
		return nil, errSchemaTooLarge
	}
	if readErr != nil {
		// Post-open read failure: the location RESOLVED and OPENED, so this is a
		// real I/O failure, NOT a benign schemaLocation fetch/resolution miss.
		// Tag it fatal so nestedFetchMiss cannot demote it to a warning even when
		// readErr is itself fs.ErrInvalid/fs.ErrNotExist (fail-closed).
		return nil, fmt.Errorf("%w: %w", errNestedSchemaReadAfterOpen, readErr)
	}
	return data, nil
}

// notDemotable is the shared veto BOTH demotion predicates — [isBenignResolutionMiss]
// (the Open path) and [isDirectNotExist] (the fs.ReadFile fallback) — consult FIRST,
// so the "may this nested-load failure be demoted to a warning?" decision cannot
// diverge between the two sites. It reports whether err is DISQUALIFIED from any
// resolution-miss demotion regardless of its errno, because it:
//   - is (or wraps, anywhere in its LINEAR unwrap chain) a MULTI-ERROR
//     ([errors.Join] / any Unwrap() []error) — a single file-open never legitimately
//     produces one, and a join defeats errors.Is/errors.As first-match selection, so a
//     benign sibling could mask a fatal (permission/read) sibling ([containsMultiError]);
//   - satisfies fs.ErrPermission anywhere in its chain — a permission denial is not a
//     resolution miss; or
//   - is a fatal schema-load condition ([IsFatalSchemaLoad]).
//
// A predicate that passes this veto then applies its own PHASE-appropriate accept: the
// Open path admits the broad benign errno set (a FAILED Open IS the resolution phase),
// the atomic-fs.ReadFile fallback admits only the stricter canonical "file not found"
// shape. Because both route through this one guard, [errors.Join] of a permission and a
// benign miss — or any permission/fatal anywhere in the tree — is FATAL at EITHER site.
func notDemotable(err error) bool {
	return containsMultiError(err) || errors.Is(err, fs.ErrPermission) || IsFatalSchemaLoad(err)
}

// isBenignResolutionMiss reports whether err is a benign schemaLocation
// RESOLUTION miss — an error that POSITIVELY signals the resource is ABSENT
// ([fs.ErrNotExist]). A demotable miss must positively signal not-found: only
// [fs.ErrNotExist] qualifies. An [fs.ErrInvalid] (or any other) Open error is NOT
// a confirmed absence — a malformed/invalid LOCAL open must stay FATAL
// (fail-closed) — so it is not demoted here. A non-file-scheme absolute URI is
// already mapped to [fs.ErrNotExist] by [iofs.PermissiveRoot.Open] (it is not a
// local path, so it is genuinely absent), so the optional absolute-URI include
// still demotes. It gates the whitelist ONLY inside [readNestedSchema], where a
// benign miss is then wrapped in [errSchemaFetchMiss]; downstream classification
// keys on that tag, not on the errno, so this predicate is never consulted after
// the resolution phase.
//
// The shared [notDemotable] veto is applied FIRST — the SAME veto [isDirectNotExist]
// uses — so a MULTI-ERROR ([errors.Join]), an fs.ErrPermission anywhere in the chain,
// or a fatal schema-load condition is NEVER demoted even when it ALSO wraps a benign
// fs.ErrNotExist errno. A failed Open IS the resolution phase, so only after that
// veto is the fs.ErrNotExist accept applied.
func isBenignResolutionMiss(err error) bool {
	if notDemotable(err) {
		return false
	}
	return errors.Is(err, fs.ErrNotExist)
}

// isOpenMissOrUnsupported reports whether a FAILED Open error qualifies a
// [fs.ReadFileFS] for the atomic fs.ReadFile fallback: a benign resolution miss
// ([fs.ErrNotExist]) OR an Open-UNSUPPORTED signal ([fs.ErrInvalid]) that a
// ReadFileFS-only FS returns when Open is not implemented — but NOT a
// permission/multi-error/fatal cause ([notDemotable]). The fallback then serves
// the bytes or surfaces the AUTHORITATIVE fs.ReadFile error, which is classified
// by [isDirectNotExist] (fs.ErrNotExist only). This is DELIBERATELY broader than
// [isBenignResolutionMiss] (the final Open-path DEMOTION gate, fs.ErrNotExist
// only): fs.ErrInvalid may ROUTE a ReadFileFS to its fallback, but must never by
// itself DEMOTE an Open failure — a malformed-local fs.ErrInvalid on a plain
// fs.FS stays fatal.
func isOpenMissOrUnsupported(err error) bool {
	if notDemotable(err) {
		return false
	}
	return errors.Is(err, fs.ErrNotExist) || errors.Is(err, fs.ErrInvalid)
}

// isDirectNotExist reports whether err is a CANONICAL filesystem "file not found"
// at the RESOLUTION/OPEN phase — the one shape that unambiguously denotes
// non-existence and can never be a post-resolution read failure. It accepts ONLY a
// SINGLE-CHAIN error that is one of:
//   - the bare [fs.ErrNotExist] sentinel (err == fs.ErrNotExist), or
//   - a [*fs.PathError] (reachable through LINEAR wrapping via [errors.As]) whose Op
//     is exactly "open" — the resolution/open operation, NOT "read"/"stat"/any other
//     op, which are post-open failures — and whose OWN [fs.PathError.Err] cause
//     satisfies fs.ErrNotExist under [errors.Is].
//
// The shared [notDemotable] veto — a MULTI-ERROR ([errors.Join] / any error implementing
// Unwrap() []error, anywhere in the linear unwrap chain), an fs.ErrPermission anywhere in
// the chain, or a fatal schema-load condition — is applied FIRST, before any other test.
// A real filesystem never reports a single file-open through a joined error: a
// multi-error bundles multiple independent failures, which is inherently suspicious for a
// resolution miss AND defeats errors.As's first-match selection — errors.As would pick a
// benign Op=="open"/ErrNotExist sibling and ignore a fatal (read/permission) PathError
// joined alongside. Rejecting the whole multi-error class up front keeps every
// errors.Join shape FATAL and makes the subsequent errors.As on a single chain
// unambiguous (there is exactly one PathError to select). The same veto keeps a
// whole-chain fs.ErrPermission / [IsFatalSchemaLoad] cause FATAL. This is the SAME guard
// [isBenignResolutionMiss] consults, so the Open path and this fallback cannot diverge. A
// genuine [os.DirFS] / [fstest.MapFS] miss returns a SINGLE *fs.PathError{Op:"open",
// Err: syscall.ENOENT} — not a Join — so it is still accepted and WARNS.
//
// The Op=="open" guard is the load-bearing phase discriminator: a failed Open IS the
// resolution phase, so an "open" PathError denotes non-existence at resolution and can
// never be a post-resolution read failure. The membership test is on the SELECTED
// PathError's OWN cause (errors.Is(pe.Err, fs.ErrNotExist)), and is errors.Is (not an
// exact ==) so a real errno like syscall.ENOENT — what os.DirFS returns for a missing
// file, satisfying fs.ErrNotExist yet NOT the fs.ErrNotExist sentinel by == — is
// correctly accepted, while a *fs.PathError{Op:"read", Err: fs.ErrNotExist}/{Op:"stat",
// …} (a POST-resolution failure that merely reports the ErrNotExist errno) is still
// REJECTED, so it stays UNTAGGED and therefore FATAL (fail-closed).
//
// Everything else — a message-wrapped fs.ErrNotExist that is NOT a *fs.PathError
// (fmt.Errorf("...: %w", …)), a non-"open" Op, or an err whose selected PathError does
// not satisfy fs.ErrNotExist at all — is REJECTED. This is the last classifiable edge
// for the inherently-atomic fs.ReadFile fallback in [readNestedSchema].
func isDirectNotExist(err error) bool {
	if notDemotable(err) {
		return false
	}
	if err == fs.ErrNotExist {
		return true
	}
	var pe *fs.PathError
	if !errors.As(err, &pe) {
		return false
	}
	return pe.Op == "open" && errors.Is(pe.Err, fs.ErrNotExist)
}

// containsMultiError reports whether err or anything in its LINEAR unwrap chain
// (following single Unwrap() error links) implements Unwrap() []error — i.e. is an
// [errors.Join] / multi-error. A single file-open never legitimately produces one, so
// its presence anywhere in the chain disqualifies err from the resolution-miss
// classification in [isDirectNotExist].
func containsMultiError(err error) bool {
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

// FatalSchemaLoader is implemented by errors raised from a configured [fs.FS]
// (see [Compiler.FS]) that must abort compilation rather than be demoted to a
// warning when they occur while loading an xs:import target. The xsd compiler
// normally treats a failure to load an xs:import target as a non-fatal warning
// ("Failed to locate a schema ... Skipping the import."), matching libxml2.
// A resource-limit guard, however, must not be silently defeated by that
// demotion: an FS whose Open error wraps a value satisfying this interface (and
// returning true) is propagated as a fatal compilation error. The wrapped error
// chain is preserved, so callers can still errors.Is/errors.As the underlying
// cause (e.g. a resource-too-large sentinel).
type FatalSchemaLoader interface {
	FatalSchemaLoad() bool
}

// IsFatalSchemaLoad reports whether err (or anything in its chain) is a fatal
// schema-load condition that must ABORT compilation rather than be demoted to a
// warning or papered over by a fallback to a pre-compiled schema. It is the
// single source of truth for this classification, shared by the xsd import
// warn-and-continue paths and by xslt3's xsl:import-schema fallback guard so the
// two layers cannot drift apart.
//
// A condition is fatal when err (or anything it wraps) satisfies ANY of:
//
//   - the schemaLocation escaped its base directory via ".." ([errSchemaPathEscape]);
//   - xs:import recursion exceeded the configured depth ([errImportDepthExceeded]);
//   - xs:include/xs:redefine nesting exceeded the configured depth
//     ([errIncludeDepthExceeded]) — otherwise an over-deep include/redefine chain
//     inside an IMPORTED schema would be demoted to a warning and silently ignored
//     by loadImport's nested-processing fallback;
//   - a nested schema exceeded the byte cap while being read ([errSchemaTooLarge]);
//   - a nested schema failed to READ after its file was successfully opened
//     ([errNestedSchemaReadAfterOpen]) — a resolved-and-opened location that then
//     fails to read is a real I/O failure, not a benign schemaLocation miss;
//   - the configured [fs.FS] returned an error satisfying [FatalSchemaLoader]
//     (e.g. a resource-limit breach such as a too-large external resource).
//
// Everything else — a genuine "schema not found" miss or a not-applicable
// error — is non-fatal and may fall back / warn as before. All sentinels are
// matched via errors.Is / errors.As, so they remain unexported; this helper is
// the public surface.
func IsFatalSchemaLoad(err error) bool {
	if errors.Is(err, errSchemaPathEscape) || errors.Is(err, errImportDepthExceeded) || errors.Is(err, errIncludeDepthExceeded) || errors.Is(err, errSchemaTooLarge) || errors.Is(err, errNestedSchemaReadAfterOpen) {
		return true
	}
	var f FatalSchemaLoader
	return errors.As(err, &f) && f.FatalSchemaLoad()
}

// validateSchemaPath resolves an xs:include/xs:import/xs:redefine
// schemaLocation against baseDir and returns the name handed to the configured
// fs.FS. It is a thin wrapper over [ResolveSchemaURI], the single canonical
// URI-resolution helper shared with xslt3 (so the two layers cannot drift).
func validateSchemaPath(baseDir, location string) (string, error) {
	return ResolveSchemaURI(location, baseDir)
}

// schemaBaseDir returns the base used to resolve nested includes/imports of
// the schema located at loc. For a URI loc the base is the URI itself (RFC
// 3986 resolution replaces the last path segment), so it is returned verbatim;
// for a local filesystem path it is the containing directory.
func schemaBaseDir(loc string) string {
	if uriScheme(loc) != "" {
		return loc
	}
	// loc is an fs.FS key in forward-slash form (see ResolveSchemaURI); derive
	// its parent directory with path.Dir so the result stays slash-separated on
	// every OS rather than gaining backslashes via filepath.Dir on Windows.
	return path.Dir(uripath.ToSlash(loc))
}

// schemaDisplayLoc builds the human-readable location shown in import/include
// diagnostics: the raw schemaLocation resolved against the parent schema
// reference (filename). URI-aware so absolute URIs and URI bases are not
// collapsed by filepath.
func schemaDisplayLoc(filename, loc string) string {
	if uriScheme(loc) != "" {
		return loc
	}
	if scheme := uriScheme(filename); scheme != "" {
		resolved, err := resolveURIReference(filename, loc)
		if err == nil {
			return resolved
		}
	}
	// Join in forward-slash space so the diagnostic path is stable across OSes
	// (filepath.Join would emit '\' on Windows).
	return path.Join(path.Dir(uripath.ToSlash(filename)), uripath.ToSlash(loc))
}

// processIncludes handles xs:include and xs:import elements.
func (c *compiler) processIncludes(ctx context.Context, root *helium.Element) error {
	// Per-document set of xs:override target paths, so the same document overridden
	// twice within one schema document is rejected (XSD 1.1, W3C over022).
	overrideSeen := make(map[string]struct{})
	for child := range helium.Children(root) {
		if child.Type() != helium.ElementNode {
			continue
		}
		elem, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		switch {
		case isXSDElement(elem, elemInclude):
			// src-include.1 (§4.2.3 / the schema-for-schemas): @schemaLocation is
			// REQUIRED on xs:include. Its ABSENCE is a schema-representation error
			// (reject the schema), distinct from a present-but-unresolvable
			// schemaLocation hint (a warning, handled below). Version-independent.
			if !hasAttr(elem, attrSchemaLocation) {
				c.reportMissingSchemaLocation(ctx, elem, elemInclude)
				continue
			}
			loc := getAttr(elem, attrSchemaLocation)
			if loc == "" {
				continue
			}
			if err := c.loadInclude(ctx, loc, elem); err != nil {
				if c.nestedLoadFailureFatal(err) {
					return err
				}
				c.reportSchemaLoadWarning(ctx, elem, elemInclude, "include", loc)
			}
		case isXSDElement(elem, elemImport):
			if err := c.processImport(ctx, elem); err != nil {
				return err
			}
		case isXSDElement(elem, elemRedefine):
			// src-redefine.1 (§4.2.5 / the schema-for-schemas): @schemaLocation is
			// REQUIRED on xs:redefine — same representation rule as xs:include above.
			if !hasAttr(elem, attrSchemaLocation) {
				c.reportMissingSchemaLocation(ctx, elem, elemRedefine)
				continue
			}
			loc := getAttr(elem, attrSchemaLocation)
			if loc == "" {
				continue
			}
			if err := c.loadRedefine(ctx, loc, elem); err != nil {
				if c.nestedLoadFailureFatal(err) {
					return err
				}
				c.reportSchemaLoadWarning(ctx, elem, elemRedefine, "redefine", loc)
			}
		case c.version == Version11 && isXSDElement(elem, elemOverride):
			// xs:override is an XSD 1.1 construct. In 1.0 mode it is ignored
			// (skipped) so existing 1.0 behavior stays byte-identical.
			//
			// §4.2.4 (the schema-for-schemas): @schemaLocation is REQUIRED on
			// xs:override. Its ABSENCE is a schema-representation error (reject
			// the schema), matching the xs:include/xs:redefine handling above —
			// distinct from a present-but-unresolvable hint (demoted to a warning
			// by the load taxonomy).
			if !hasAttr(elem, attrSchemaLocation) {
				c.reportMissingSchemaLocation(ctx, elem, elemOverride)
				continue
			}
			loc := getAttr(elem, attrSchemaLocation)
			if loc == "" {
				continue
			}
			if !c.recordOverrideTarget(ctx, elem, loc, overrideSeen) {
				continue
			}
			if err := c.loadOverride(ctx, loc, elem); err != nil {
				// Same taxonomy as xs:include/xs:redefine above: a genuine fetch miss
				// of the override target (tagged errSchemaFetchMiss by the shared fetch
				// phase) is a schemaLocation-hint miss demoted to a warning; a content
				// or policy/security failure is fatal.
				if c.nestedLoadFailureFatal(err) {
					return err
				}
				c.reportSchemaLoadWarning(ctx, elem, elemOverride, "override", loc)
			}
		}
	}
	return nil
}

// nestedLoadFailureFatal reports whether an xs:include/xs:redefine (and, via
// [processImport], xs:import; and, via [loadOverride]/[overrideLoadTarget],
// xs:override) load error must abort compilation rather than be
// demoted to a warning. It is FAIL-CLOSED: the ONLY demotable condition is a
// CONFIRMED benign fetch/resolution miss ([nestedFetchMiss]) — schemaLocation is
// only a hint (src-include.1 / src-import), so a missing or unresolvable target
// is skipped (matching libxml2). Everything else — a security/policy denial, a
// CONTENT failure (malformed XML or a non-xs:schema root), or any other/ambiguous
// error — is fatal.
func (c *compiler) nestedLoadFailureFatal(err error) bool {
	return !c.nestedFetchMiss(err)
}

// nestedFetchMiss reports whether err is a CONFIRMED benign FETCH/RESOLUTION miss
// of a nested schemaLocation — the ONLY nested-load failure demoted to a warning.
// The demotable decision is a SINGLE POINT OF TRUTH: it keys ONLY on the POSITIVE
// [errSchemaFetchMiss] tag that [readNestedSchema] applies AT THE RESOLUTION PHASE,
// never on a raw [fs.ErrNotExist]/[fs.ErrInvalid] errno. So a benign missing/
// unresolvable target (the tag's only source) is skipped, while the SAME errno
// arising later — a post-open read failure, a CONTENT/parse error, an external-
// entity read, a non-xs:schema root, a security/policy denial, the deny-all FS —
// is UNTAGGED and stays fatal, however its errno chain reads. The
// [IsFatalSchemaLoad] guard is defensive: a tagged error can never also be a fatal
// sentinel (the tag wraps only benign resolution errnos), but keying "fatal wins"
// makes that invariant explicit and fail-closed.
func (c *compiler) nestedFetchMiss(err error) bool {
	return errors.Is(err, errSchemaFetchMiss) && !IsFatalSchemaLoad(err)
}

// reportMissingSchemaLocation reports the schema-representation error for an
// xs:include/xs:redefine that omits the REQUIRED @schemaLocation attribute
// (src-include.1 / src-redefine.1). Unlike a schemaLocation that is present but
// fails to resolve (a warning, so the composition element is skipped), a MISSING
// schemaLocation makes the schema invalid, so this is a fatal schema error.
// Version-independent. elemKind is the XSD element local name (include/redefine).
func (c *compiler) reportMissingSchemaLocation(ctx context.Context, elem *helium.Element, elemKind string) {
	c.schemaError(ctx, schemaParserErrorAttr(c.diagSource(), elem.Line(),
		elem.LocalName(), elemKind, attrSchemaLocation,
		"The attribute 'schemaLocation' is required but missing."))
}

// reportSchemaLoadWarning demotes a non-fatal xs:include/xs:import/xs:redefine
// load failure to the libxml2 "I/O warning" + "Failed to locate … Skipping the
// <verb>." warning pair, so an unresolvable schemaLocation hint skips that
// composition element rather than aborting compilation. libxml2 treats a missing
// include/import/redefine target as a warning; only a fatal load condition
// ([IsFatalSchemaLoad], e.g. a security-limit breach) aborts. elemKind is the XSD
// element local name and verb the word after "Skipping the".
func (c *compiler) reportSchemaLoadWarning(ctx context.Context, elem *helium.Element, elemKind, verb, loc string) {
	if c.filename == "" {
		return
	}
	displayLoc := schemaDisplayLoc(c.filename, loc)
	c.errorHandler.Handle(ctx, helium.NewLeveledError(fmt.Sprintf("I/O warning : failed to load \"%s\": %s\n", displayLoc, "No such file or directory"), helium.ErrorLevelWarning))
	c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserWarning(c.filename, elem.Line(),
		elem.LocalName(), elemKind,
		"Failed to locate a schema at location '"+displayLoc+"'. Skipping the "+verb+"."), helium.ErrorLevelWarning))
}

// processImport handles a single xs:import element: it enforces the
// already-imported-namespace warning, loads the imported schema, and demotes a
// non-fatal load failure to the established I/O + "Failed to locate" warnings (so
// the import is skipped, matching libxml2). It returns a non-nil error ONLY for a
// fatal load condition ([IsFatalSchemaLoad]) that must abort compilation; every
// non-fatal path returns nil after warning. Shared by processIncludes and the
// xs:override nested processor so an import inside an overridden document gets the
// same diagnostics as a top-level import.
func (c *compiler) processImport(ctx context.Context, elem *helium.Element) error {
	ns := getAttr(elem, attrNamespace)

	// src-import (§4.2.6.1 Import Constraints and Semantics): the <xs:import>
	// @namespace representation constraints, version-independent. A violation is a
	// schema-representation error, so the invalid import declaration is reported and
	// skipped (never loaded). Only enforced when a filename is available (the same
	// gating the other schema-parser diagnostics use).
	if c.filename != "" {
		nsPresent := hasAttr(elem, attrNamespace)
		switch {
		case nsPresent && ns == "":
			// An empty @namespace is not a namespace name: the ABSENT namespace is
			// imported by OMITTING @namespace, never by namespace="".
			c.schemaError(ctx, schemaParserErrorAttr(c.filename, elem.Line(), elem.LocalName(), elemImport, attrNamespace,
				"The value '' is not valid; the namespace of an <import> must not be the empty string."))
			return nil
		case nsPresent && ns == c.schema.targetNamespace && c.schemaTargetNSSet:
			// src-import.1.1: @namespace must not match the enclosing schema's own
			// targetNamespace (a schema does not import its own namespace).
			c.schemaError(ctx, schemaParserErrorAttr(c.filename, elem.Line(), elem.LocalName(), elemImport, attrNamespace,
				"The namespace '"+ns+"' must not match the target namespace of the importing schema."))
			return nil
		case !nsPresent && !c.schemaTargetNSSet:
			// src-import.1.2: an <import> without @namespace requires the enclosing
			// schema to have a targetNamespace.
			c.schemaError(ctx, schemaParserError(c.filename, elem.Line(), elem.LocalName(), elemImport,
				"An <import> without a namespace attribute requires the enclosing schema to have a targetNamespace."))
			return nil
		}
	}

	// Record the namespace as import-declared BEFORE any early return, so a
	// location-less import (and, below, one whose load fails) still marks the
	// namespace referenceable — its declarations are simply unknown.
	c.importDeclaredNS[ns] = struct{}{}
	// Record the import against the DECLARING document (keyed by its filename), so
	// the non-imported-namespace reference check can tell a namespace a document
	// imported DIRECTLY from one merely reachable transitively through another
	// document's import. Includes share this compiler but carry the included
	// document's c.filename here, so an include's import is not attributed to the
	// including document.
	if c.filename != "" {
		if c.docImportedNS[c.filename] == nil {
			c.docImportedNS[c.filename] = make(map[string]struct{})
		}
		c.docImportedNS[c.filename][ns] = struct{}{}
	}

	loc := getAttr(elem, attrSchemaLocation)
	if loc == "" {
		return nil
	}

	// Check if this namespace was already imported.
	if prevLoc, ok := c.importedNS[ns]; ok && c.filename != "" {
		displayLoc := schemaDisplayLoc(c.filename, loc)
		displayPrevLoc := schemaDisplayLoc(c.filename, prevLoc)
		c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserWarning(c.filename, elem.Line(),
			elem.LocalName(), elemImport,
			"Skipping import of schema located at '"+displayLoc+"' for the namespace '"+ns+"', since this namespace was already imported with the schema located at '"+displayPrevLoc+"'."), helium.ErrorLevelWarning))
		return nil
	}

	if err := c.loadImport(ctx, loc, ns, elem); err != nil {
		// schemaLocation is only a hint (src-import), so a benign FETCH/RESOLUTION
		// miss (a missing file, or an http:// hint opened as a path yielding
		// fs.ErrInvalid) is demoted to a warning and the import skipped. The shared
		// fail-closed classifier keeps every other outcome FATAL — a
		// security/policy denial (the cap/deny-all cannot be silently defeated for
		// an xs:import target) and a CONTENT failure (malformed XML / non-xs:schema
		// root) — so the import path cannot diverge from the include/redefine path.
		if c.nestedLoadFailureFatal(err) {
			return err
		}
		// Import failure — report warning if we have a filename.
		c.reportSchemaLoadWarning(ctx, elem, elemImport, "import", loc)
		return nil
	}

	// Track the imported namespace.
	c.importedNS[ns] = loc
	return nil
}

// processNestedIncludes processes the xs:include/xs:import/xs:redefine declared
// by an already-parsed included or redefined schema (incRoot), so a transitive
// chain (main -> inc1 -> inc2) resolves rather than failing on declarations that
// only inc2 defines. The nested references in incRoot are relative to the
// included schema, so baseDir/filename are temporarily switched to it (path is
// its resolved fs key, location its raw schemaLocation). Recursion is bounded by
// the includeVisited loaded-set (registered by the caller) plus a depth cap.
func (c *compiler) processNestedIncludes(ctx context.Context, incRoot *helium.Element, path, location string) error {
	if c.includeDepth >= c.maxIncludeDepth {
		return fmt.Errorf("%w (limit=%d, location=%q)", errIncludeDepthExceeded, c.maxIncludeDepth, location)
	}
	savedBaseDir := c.baseDir
	savedFilename := c.filename
	c.baseDir = schemaBaseDir(path)
	if savedFilename != "" {
		c.filename = schemaDisplayLoc(savedFilename, location)
	}
	c.includeDepth++

	err := c.processIncludes(ctx, incRoot)

	c.includeDepth--
	c.baseDir = savedBaseDir
	c.filename = savedFilename
	return err
}

// loadInclude loads and merges an included schema file.
func (c *compiler) loadInclude(ctx context.Context, location string, includeElem *helium.Element) error {
	path, err := validateSchemaPath(c.baseDir, location)
	if err != nil {
		return fmt.Errorf("xsd: failed to load include %q: %w", location, err)
	}

	// include+override conflict (symmetric to overrideLoadTarget): the document was
	// already transformed by an xs:override, so pulling in its untransformed
	// originals via xs:include would collide. Report the fatal conflict instead of
	// silently skipping.
	if _, overridden := c.overridePaths[path]; overridden {
		c.reportOverrideIncludeConflict(ctx, includeElem, location, elemInclude)
		return nil
	}

	// Load each included document at most once: a transitive/diamond re-include
	// of an already-loaded document is skipped so its declarations are not
	// re-registered and a circular include cannot recurse forever.
	if _, seen := c.includeVisited[path]; seen {
		return nil
	}
	c.includeVisited[path] = struct{}{}

	data, err := c.readNestedSchema(path)
	if err != nil {
		// The read attempt failed, but the loaded-set marker was added above
		// (before the read, so a self-referential include cannot recurse).
		// Roll it back: a genuinely-missing target is demoted to a warning by
		// the caller, and a LATER xs:include/xs:redefine of the SAME location
		// must be retried and warned about again rather than treated as
		// "already loaded" and silently skipped (which, for xs:redefine, would
		// otherwise run its overrides against an empty Phase-A set and report a
		// spurious duplicate-redefine error). A fatal read error aborts
		// compilation regardless, so the rollback is harmless there.
		delete(c.includeVisited, path)
		return fmt.Errorf("xsd: failed to load include %q: %w", location, err)
	}

	doc, err := c.parse(ctx, data)
	if err != nil {
		return fmt.Errorf("xsd: failed to parse include %q: %w: %w", location, errSchemaContentInvalid, err)
	}

	incRoot := findDocumentElement(doc)
	if incRoot == nil || !isXSDElement(incRoot, elemSchema) {
		return fmt.Errorf("xsd: included document %q is not an xs:schema: %w", location, errSchemaContentInvalid)
	}

	// Conditional inclusion runs per schema document, BEFORE the targetNamespace
	// compatibility check and the default-attribute interpretation: a vc-excluded
	// root contributes an EMPTY schema and must not be rejected for a TNS mismatch.
	// (For a non-excluded root the same call prunes its vc:-excluded descendants.)
	// c.includeFile is set across the pre-pass so a malformed-vc diagnostic in the
	// included document is attributed to the included file, not the including one,
	// then restored on every path (the later block sets it again for parsing).
	savedIncludeFileVC := c.includeFile
	if c.filename != "" {
		c.includeFile = schemaDisplayLoc(c.filename, location)
	}
	rootExcluded := c.applyConditionalInclusion(ctx, incRoot)
	c.includeFile = savedIncludeFileVC
	if rootExcluded {
		return nil
	}

	// Check target namespace compatibility.
	incTargetNS := getAttr(incRoot, attrTargetNamespace)
	if incTargetNS != "" && incTargetNS != c.schema.targetNamespace {
		displayLoc := location
		if c.filename != "" {
			displayLoc = schemaDisplayLoc(c.filename, location)
		}
		c.schemaError(ctx, schemaParserError(c.filename, includeElem.Line(),
			includeElem.LocalName(), elemInclude,
			"The target namespace '"+incTargetNS+"' of the included/redefined schema '"+displayLoc+"' differs from '"+c.schema.targetNamespace+"' of the including/redefining schema."))
		return nil
	}

	// Chameleon include: if the included schema has no targetNamespace,
	// it adopts the including schema's targetNamespace.
	// The included schema's elementFormDefault/attributeFormDefault are
	// applied within the included declarations.

	// Save current form-qualified and default settings, then apply the included
	// schema's OWN settings. The elementFormDefault/attributeFormDefault/
	// blockDefault/finalDefault attributes are PER schema document, not inherited
	// from the including schema: a chain main -> inc1(elementFormDefault=
	// "qualified") -> inc2(omitted) must parse inc2's declarations as UNQUALIFIED
	// (the spec default), so each document is read against the spec defaults plus
	// only its own declared values. Reset to the spec defaults (unqualified / no
	// flags) before applying this document's attributes; the parent's defaults are
	// restored after processing so siblings are unaffected.
	savedElemForm := c.schema.elemFormQualified
	savedAttrForm := c.schema.attrFormQualified
	savedBlockDefault := c.schema.blockDefault
	savedFinalDefault := c.schema.finalDefault
	savedDefaultAttributes := c.schema.defaultAttributes
	savedDefaultAttrsSet := c.schema.defaultAttrsSet
	savedDefaultAttrsSrc := c.schema.defaultAttrsSrc
	savedIncludeFile := c.includeFile
	savedXPathDefaultNS := c.schemaXPathDefaultNS
	savedSchemaTargetNSSet := c.schemaTargetNSSet
	savedDefaultOpenContent := c.defaultOpenContent
	c.schemaTargetNSSet = c.schema.targetNamespace != ""
	if c.filename != "" {
		c.includeFile = schemaDisplayLoc(c.filename, location)
	}
	c.schema.elemFormQualified = getAttr(incRoot, attrElementFormDefault) == attrValQualified
	c.schema.attrFormQualified = getAttr(incRoot, attrAttributeFormDefault) == attrValQualified
	c.schema.blockDefault = parseBlockFlags(getAttr(incRoot, attrBlockDefault))
	c.schema.finalDefault = parseFinalFlags(getAttr(incRoot, attrFinalDefault))
	// schemaXPathDefaultNS is PER document (used by the SHARED resolveXPathDefaultNS
	// for the included schema's identity-constraint selector/field XPaths AND its
	// xs:assert/xs:assertion XPaths): an included root's @xpathDefaultNamespace must
	// govern its own IDCs/asserts, not inherit the including schema's. Reset to
	// spec-default (none) plus this document's value, RESOLVED against the included
	// root now (so an inherited ##defaultNamespace uses the included root's default
	// namespace).
	c.schemaXPathDefaultNS = ""
	// <xs:defaultOpenContent> is PER document: the included schema's complex types
	// use the included root's own default open content (or none), not the including
	// schema's. Reset and read this document's value before parsing its children.
	c.defaultOpenContent = c.readDefaultOpenContent(ctx, incRoot)
	if c.version == Version11 {
		c.schemaXPathDefaultNS = resolveXPathDefaultNSToken(incRoot, getAttr(incRoot, attrXPathDefaultNS), c.schema.targetNamespace)
		c.readSchemaDefaultAttributes(ctx, incRoot)
	}

	// The CTA static context (static base URI, xpathDefaultNamespace) is PER schema
	// document: an xs:alternative in the included file must see the included
	// document's base URI and its own xpathDefaultNamespace, not the including
	// schema's. Save/restore mirrors the form/default handling above.
	savedSchemaBaseURI := c.schemaBaseURI
	savedCTAXPathDefaultNSSet := c.xpathDefaultNSSet
	savedXPathDefaultNSToken := c.schemaXPathDefaultNSToken
	c.schemaBaseURI = path
	c.xpathDefaultNSSet = hasAttr(incRoot, attrXPathDefaultNamespace)
	c.schemaXPathDefaultNSToken = getAttr(incRoot, attrXPathDefaultNamespace)

	// Snapshot the component-name sets BEFORE parsing so the delta records which
	// components this included document contributes. A later xs:redefine of the
	// same (already-loaded) document needs that set to know which components it
	// may legally override.
	beforeTypes := snapshotKeys(c.schema.types)
	beforeGroups := snapshotKeys(c.schema.groups)
	beforeAttrGroups := snapshotKeys(c.schema.attrGroups)

	// Enforce xs:ID typing/uniqueness of schema-component @id attributes within
	// THIS included document (a fresh per-document scope; the same value may
	// recur across documents). Runs after conditional-inclusion pruning so
	// vc-excluded components' @ids don't count.
	c.checkSchemaComponentIDs(ctx, incRoot)
	c.checkSchemaNamespaceAttrs(ctx, incRoot)
	c.checkIDConstraintPlacement(ctx, incRoot)
	c.checkNotations(ctx, incRoot)
	c.checkAnnotations(ctx, incRoot)

	// Parse the included schema's declarations into the current compiler.
	// (Conditional inclusion already ran above, before the TNS check.)
	err = c.parseSchemaChildren(ctx, incRoot)

	// Process the included schema's OWN xs:include/xs:import/xs:redefine so a
	// transitive chain resolves (resolved relative to the included schema).
	if err == nil {
		err = c.processNestedIncludes(ctx, incRoot, path, location)
	}

	// Cache the redefinable component set this document contributed so a later
	// xs:redefine of the same already-loaded document can validate its overrides.
	if err == nil {
		c.loadedRedefinable[path] = &redefinableSet{
			keys:      c.computeRedefinableKeys(beforeTypes, beforeGroups, beforeAttrGroups),
			consumed:  make(map[redefineKind]map[QName]struct{}),
			groups:    c.computeRedefinableGroups(beforeGroups),
			chameleon: incTargetNS == "",
		}
	}

	// Restore form-qualified settings, defaults, CTA static context, and include file.
	c.schema.elemFormQualified = savedElemForm
	c.schema.attrFormQualified = savedAttrForm
	c.schema.blockDefault = savedBlockDefault
	c.schema.finalDefault = savedFinalDefault
	c.schema.defaultAttributes = savedDefaultAttributes
	c.schema.defaultAttrsSet = savedDefaultAttrsSet
	c.schema.defaultAttrsSrc = savedDefaultAttrsSrc
	c.schemaBaseURI = savedSchemaBaseURI
	c.xpathDefaultNSSet = savedCTAXPathDefaultNSSet
	c.schemaXPathDefaultNSToken = savedXPathDefaultNSToken
	c.includeFile = savedIncludeFile
	c.schemaXPathDefaultNS = savedXPathDefaultNS
	c.schemaTargetNSSet = savedSchemaTargetNSSet
	c.defaultOpenContent = savedDefaultOpenContent

	return err
}

// snapshotKeys captures the current key set of a component map so a later
// delta (newKeysSince) can isolate the keys added between two points.
func snapshotKeys[V any](m map[QName]V) map[QName]struct{} {
	keys := make(map[QName]struct{}, len(m))
	for qn := range m {
		keys[qn] = struct{}{}
	}
	return keys
}

// newKeysSince returns the keys present in m but absent from before, i.e. the
// components added since the snapshot was taken.
func newKeysSince[V any](m map[QName]V, before map[QName]struct{}) map[QName]struct{} {
	added := make(map[QName]struct{})
	for qn := range m {
		if _, existed := before[qn]; existed {
			continue
		}
		added[qn] = struct{}{}
	}
	return added
}

// computeRedefinableKeys builds the per-kind set of component names newly
// registered (afterKeys - beforeKeys) by loading a schema document, splitting
// the type names into simpleType/complexType via c.typeKinds. These are exactly
// the components an xs:redefine of that document may override. It is computed
// once when the document is first loaded (xs:include or xs:redefine) and cached
// in c.loadedRedefinable, since the delta cannot be reconstructed once the
// document's components are merged into the schema.
func (c *compiler) computeRedefinableKeys(beforeTypes, beforeGroups, beforeAttrGroups map[QName]struct{}) map[redefineKind]map[QName]struct{} {
	newTypes := newKeysSince(c.schema.types, beforeTypes)
	phaseASimpleTypes := make(map[QName]struct{})
	phaseAComplexTypes := make(map[QName]struct{})
	for qn := range newTypes {
		switch c.typeKinds[qn] {
		case redefineKindComplexType:
			phaseAComplexTypes[qn] = struct{}{}
		default:
			// simpleType (and builtin/anySimpleType fallbacks) are treated as
			// simple; only an xs:simpleType override may consume them.
			phaseASimpleTypes[qn] = struct{}{}
		}
	}
	return map[redefineKind]map[QName]struct{}{
		redefineKindSimpleType:  phaseASimpleTypes,
		redefineKindComplexType: phaseAComplexTypes,
		redefineKindGroup:       newKeysSince(c.schema.groups, beforeGroups),
		redefineKindAttrGroup:   newKeysSince(c.schema.attrGroups, beforeAttrGroups),
	}
}

func (c *compiler) computeRedefinableGroups(beforeGroups map[QName]struct{}) map[QName]*ModelGroup {
	groups := make(map[QName]*ModelGroup)
	for qn, group := range c.schema.groups {
		if _, existed := beforeGroups[qn]; existed {
			continue
		}
		groups[qn] = group
	}
	return groups
}

// loadRedefine loads a schema via xs:redefine and processes override children.
// It works like xs:include (merging original declarations) but then applies
// redefinitions for complexType, simpleType, group, and attributeGroup children.
func (c *compiler) loadRedefine(ctx context.Context, location string, redefineElem *helium.Element) error {
	// Phase A: Load the redefined schema (same as include).
	path, err := validateSchemaPath(c.baseDir, location)
	if err != nil {
		return fmt.Errorf("xsd: failed to load redefine %q: %w", location, err)
	}

	// include+override conflict (symmetric to overrideLoadTarget): a document
	// already transformed by an xs:override cannot also be redefined.
	if _, overridden := c.overridePaths[path]; overridden {
		c.reportOverrideIncludeConflict(ctx, redefineElem, location, elemRedefine)
		return nil
	}

	// A redefine whose target document is ALREADY loaded — via a prior xs:include
	// or an earlier xs:redefine of the same schema — must not silently drop its
	// override children. Re-running Phase A would re-register the redefined
	// schema's declarations (duplicate components) or recurse forever, so loading
	// is correctly skipped; but XSD permits multiple xs:redefine of the same
	// document (redefining disjoint components, or a no-op repeat), so the
	// document path repeating is NOT itself an error. Process this redefine's
	// override children against the cached Phase-A component set instead. The
	// shared consumed set rejects only a TRUE duplicate — a component an earlier
	// xs:redefine of the same document already redefined.
	if _, seen := c.includeVisited[path]; seen {
		rs := c.loadedRedefinable[path]
		var phaseAKeys, consumed map[redefineKind]map[QName]struct{}
		var phaseAGroups map[QName]*ModelGroup
		if rs != nil {
			phaseAKeys = rs.keys
			consumed = rs.consumed
			phaseAGroups = rs.groups
		} else {
			// The document was registered without a recorded redefinable set
			// (e.g. the root schema seeded into includeVisited by CompileFile, or
			// an imported schema's own seed). Nothing from it is overridable, so
			// every override is reported as a duplicate by the override loop.
			phaseAKeys = map[redefineKind]map[QName]struct{}{
				redefineKindSimpleType:  {},
				redefineKindComplexType: {},
				redefineKindGroup:       {},
				redefineKindAttrGroup:   {},
			}
			phaseAGroups = map[QName]*ModelGroup{}
		}
		chameleon := rs != nil && rs.chameleon
		return c.processRedefineOverrides(ctx, redefineElem, phaseAKeys, consumed, chameleon, phaseAGroups)
	}
	c.includeVisited[path] = struct{}{}

	data, err := c.readNestedSchema(path)
	if err != nil {
		c.checkRedefineOverrideRepresentation(ctx, redefineElem)
		// Roll back the loaded-set marker added above (see loadInclude): a
		// missing redefine target demoted to a warning must not leave the path
		// marked, or a later xs:redefine of the same location would hit the
		// "seen" branch and run its overrides against a nil/empty Phase-A set,
		// reporting a spurious duplicate-redefine error instead of warning.
		delete(c.includeVisited, path)
		return fmt.Errorf("xsd: failed to load redefine %q: %w", location, err)
	}

	doc, err := c.parse(ctx, data)
	if err != nil {
		return fmt.Errorf("xsd: failed to parse redefine %q: %w: %w", location, errSchemaContentInvalid, err)
	}

	incRoot := findDocumentElement(doc)
	if incRoot == nil || !isXSDElement(incRoot, elemSchema) {
		return fmt.Errorf("xsd: redefined document %q is not an xs:schema: %w", location, errSchemaContentInvalid)
	}

	// Conditional inclusion runs per schema document, BEFORE the targetNamespace
	// check and default-attribute interpretation: a vc-excluded root contributes an
	// EMPTY schema (no Phase-A components) and must not be rejected for a TNS
	// mismatch. (For a non-excluded root the same call prunes its descendants.)
	// c.includeFile is set across the pre-pass so a malformed-vc diagnostic in the
	// redefined document is attributed to that file, then restored on every path.
	savedIncludeFileVC := c.includeFile
	if c.filename != "" {
		c.includeFile = schemaDisplayLoc(c.filename, location)
	}
	rootExcluded := c.applyConditionalInclusion(ctx, incRoot)
	c.includeFile = savedIncludeFileVC
	incTargetNS := getAttr(incRoot, attrTargetNamespace)
	if rootExcluded {
		// The redefined document's root is vc-excluded, so it contributes NO
		// Phase-A components. The <xs:redefine> override children (which live in the
		// REDEFINING schema, not the excluded document) must STILL be validated
		// against that empty target set: XSD rejects an override whose Phase-A
		// target does not exist, so an override of a now-absent component is an
		// error, not a silent no-op. (path is already in includeVisited from above;
		// c.includeFile/form-defaults are the redefining schema's, correct for
		// override-local diagnostics and declarations.)
		emptyKeys := map[redefineKind]map[QName]struct{}{
			redefineKindSimpleType:  {},
			redefineKindComplexType: {},
			redefineKindGroup:       {},
			redefineKindAttrGroup:   {},
		}
		rs := &redefinableSet{
			keys:      emptyKeys,
			consumed:  make(map[redefineKind]map[QName]struct{}),
			groups:    map[QName]*ModelGroup{},
			chameleon: incTargetNS == "",
		}
		c.loadedRedefinable[path] = rs
		return c.processRedefineOverrides(ctx, redefineElem, rs.keys, rs.consumed, rs.chameleon, rs.groups)
	}

	// Check target namespace compatibility (same rules as include).
	if incTargetNS != "" && incTargetNS != c.schema.targetNamespace {
		displayLoc := location
		if c.filename != "" {
			displayLoc = schemaDisplayLoc(c.filename, location)
		}
		c.schemaError(ctx, schemaParserError(c.filename, redefineElem.Line(),
			redefineElem.LocalName(), elemRedefine,
			"The target namespace '"+incTargetNS+"' of the included/redefined schema '"+displayLoc+"' differs from '"+c.schema.targetNamespace+"' of the including/redefining schema."))
		return nil
	}

	// Save/restore form-qualified settings and defaults (chameleon support).
	// As with xs:include, the elementFormDefault/attributeFormDefault/
	// blockDefault/finalDefault attributes are PER schema document and are NOT
	// inherited from the redefining schema: reset to the spec defaults
	// (unqualified / no flags) before applying this document's own declared
	// values, so a redefined schema that omits them parses its declarations
	// against the spec defaults rather than the parent's settings.
	savedElemForm := c.schema.elemFormQualified
	savedAttrForm := c.schema.attrFormQualified
	savedBlockDefault := c.schema.blockDefault
	savedFinalDefault := c.schema.finalDefault
	savedDefaultAttributes := c.schema.defaultAttributes
	savedDefaultAttrsSet := c.schema.defaultAttrsSet
	savedDefaultAttrsSrc := c.schema.defaultAttrsSrc
	savedIncludeFile := c.includeFile
	savedXPathDefaultNS := c.schemaXPathDefaultNS
	savedSchemaTargetNSSet := c.schemaTargetNSSet
	savedDefaultOpenContent := c.defaultOpenContent
	c.schemaTargetNSSet = c.schema.targetNamespace != ""
	if c.filename != "" {
		c.includeFile = schemaDisplayLoc(c.filename, location)
	}
	c.schema.elemFormQualified = getAttr(incRoot, attrElementFormDefault) == attrValQualified
	c.schema.attrFormQualified = getAttr(incRoot, attrAttributeFormDefault) == attrValQualified
	c.schema.blockDefault = parseBlockFlags(getAttr(incRoot, attrBlockDefault))
	c.schema.finalDefault = parseFinalFlags(getAttr(incRoot, attrFinalDefault))
	// schemaXPathDefaultNS is PER document, like the form/block/final defaults (see
	// loadInclude): the redefined root's @xpathDefaultNamespace governs its own
	// identity-constraint AND xs:assert/xs:assertion XPaths during Phase A, RESOLVED
	// against the redefined root now (so an inherited ##defaultNamespace uses that
	// root's default ns).
	c.schemaXPathDefaultNS = ""
	// Phase A parses the REDEFINED document's own declarations, so they use the
	// redefined root's default open content (the override children, parsed in Phase
	// B below, get the redefining schema's default restored before then).
	c.defaultOpenContent = c.readDefaultOpenContent(ctx, incRoot)
	if c.version == Version11 {
		c.schemaXPathDefaultNS = resolveXPathDefaultNSToken(incRoot, getAttr(incRoot, attrXPathDefaultNS), c.schema.targetNamespace)
		c.readSchemaDefaultAttributes(ctx, incRoot)
	}
	// The CTA static context (base URI + xpathDefaultNamespace) is per-document too:
	// Phase A parses the REDEFINED document's declarations, so set them here; the
	// override children (from the redefining schema) get the parent's values restored
	// before processRedefineOverrides, like the defaults above.
	savedSchemaBaseURI := c.schemaBaseURI
	savedCTAXPathDefaultNSSet := c.xpathDefaultNSSet
	savedXPathDefaultNSToken := c.schemaXPathDefaultNSToken
	c.schemaBaseURI = path
	c.xpathDefaultNSSet = hasAttr(incRoot, attrXPathDefaultNamespace)
	c.schemaXPathDefaultNSToken = getAttr(incRoot, attrXPathDefaultNamespace)
	// Snapshot the component-name sets per kind BEFORE Phase A. The including
	// (main) schema's root declarations are already registered at this point,
	// so taking the snapshot after Phase A would wrongly treat pre-existing
	// main-schema components as redefinable. Only names ACTUALLY loaded from the
	// redefined schema (afterKeys - beforeKeys) may be overridden.
	beforeTypes := snapshotKeys(c.schema.types)
	beforeGroups := snapshotKeys(c.schema.groups)
	beforeAttrGroups := snapshotKeys(c.schema.attrGroups)

	// Enforce xs:ID typing/uniqueness of schema-component @id attributes within
	// THIS redefined document (a fresh per-document scope). Runs after
	// conditional-inclusion pruning.
	c.checkSchemaComponentIDs(ctx, incRoot)
	c.checkSchemaNamespaceAttrs(ctx, incRoot)
	c.checkIDConstraintPlacement(ctx, incRoot)
	c.checkNotations(ctx, incRoot)
	c.checkAnnotations(ctx, incRoot)

	// Parse the included schema's declarations into the current compiler.
	// (Conditional inclusion already ran above, before the TNS check.)
	if err := c.parseSchemaChildren(ctx, incRoot); err != nil {
		c.schema.elemFormQualified = savedElemForm
		c.schema.attrFormQualified = savedAttrForm
		c.schema.blockDefault = savedBlockDefault
		c.schema.finalDefault = savedFinalDefault
		c.schema.defaultAttributes = savedDefaultAttributes
		c.schema.defaultAttrsSet = savedDefaultAttrsSet
		c.schema.defaultAttrsSrc = savedDefaultAttrsSrc
		c.schemaBaseURI = savedSchemaBaseURI
		c.xpathDefaultNSSet = savedCTAXPathDefaultNSSet
		c.schemaXPathDefaultNSToken = savedXPathDefaultNSToken
		c.includeFile = savedIncludeFile
		c.schemaXPathDefaultNS = savedXPathDefaultNS
		c.schemaTargetNSSet = savedSchemaTargetNSSet
		c.defaultOpenContent = savedDefaultOpenContent
		return err
	}

	// Process the redefined schema's OWN xs:include/xs:import/xs:redefine so a
	// transitive chain resolves (resolved relative to the redefined schema).
	if err := c.processNestedIncludes(ctx, incRoot, path, location); err != nil {
		c.schema.elemFormQualified = savedElemForm
		c.schema.attrFormQualified = savedAttrForm
		c.schema.blockDefault = savedBlockDefault
		c.schema.finalDefault = savedFinalDefault
		c.schema.defaultAttributes = savedDefaultAttributes
		c.schema.defaultAttrsSet = savedDefaultAttrsSet
		c.schema.defaultAttrsSrc = savedDefaultAttrsSrc
		c.schemaBaseURI = savedSchemaBaseURI
		c.xpathDefaultNSSet = savedCTAXPathDefaultNSSet
		c.schemaXPathDefaultNSToken = savedXPathDefaultNSToken
		c.includeFile = savedIncludeFile
		c.schemaXPathDefaultNS = savedXPathDefaultNS
		c.schemaTargetNSSet = savedSchemaTargetNSSet
		c.defaultOpenContent = savedDefaultOpenContent
		return err
	}

	// Phase B: compute the redefinable component set as (afterKeys - beforeKeys)
	// per kind — only the names ACTUALLY loaded by the redefined schema (not
	// pre-existing main-schema components) may be overridden. Cache it keyed by
	// the resolved document path so a later xs:redefine of this same document can
	// validate its overrides against it (the delta cannot be recomputed once the
	// components are merged), then process this redefine's overrides against it.
	phaseAKeys := c.computeRedefinableKeys(beforeTypes, beforeGroups, beforeAttrGroups)
	phaseAGroups := c.computeRedefinableGroups(beforeGroups)
	rs := &redefinableSet{
		keys:      phaseAKeys,
		consumed:  make(map[redefineKind]map[QName]struct{}),
		groups:    phaseAGroups,
		chameleon: incTargetNS == "",
	}
	c.loadedRedefinable[path] = rs

	// The override children come from the REDEFINING (main) schema, not the
	// redefined (base) schema loaded in Phase A. Restore c.includeFile to the
	// redefining file's label so duplicate-override diagnostics report the
	// correct source file and line; Phase A above needed the base label.
	// Likewise restore the per-document defaults to the REDEFINING schema's
	// values: override-local declarations must use the redefining schema's
	// elementFormDefault/attributeFormDefault/blockDefault/finalDefault (per
	// XSD), not the redefined document's values applied for Phase A above.
	c.schema.elemFormQualified = savedElemForm
	c.schema.attrFormQualified = savedAttrForm
	c.schema.blockDefault = savedBlockDefault
	c.schema.finalDefault = savedFinalDefault
	c.schema.defaultAttributes = savedDefaultAttributes
	c.schema.defaultAttrsSet = savedDefaultAttrsSet
	c.schema.defaultAttrsSrc = savedDefaultAttrsSrc
	c.schemaBaseURI = savedSchemaBaseURI
	c.xpathDefaultNSSet = savedCTAXPathDefaultNSSet
	c.schemaXPathDefaultNSToken = savedXPathDefaultNSToken
	c.includeFile = savedIncludeFile
	c.schemaXPathDefaultNS = savedXPathDefaultNS
	c.schemaTargetNSSet = savedSchemaTargetNSSet
	// Override children belong to the REDEFINING schema, so they use ITS default
	// open content (restored here), not the redefined document's Phase A value.
	c.defaultOpenContent = savedDefaultOpenContent

	return c.processRedefineOverrides(ctx, redefineElem, phaseAKeys, rs.consumed, rs.chameleon, phaseAGroups)
}

// checkRedefineSelfDerivation enforces src-redefine.5: a <simpleType> or
// <complexType> child of <xs:redefine> must have a <restriction> (or, for a
// complexType, <extension>) whose 'base' names the redefined type itself. The
// just-parsed replacement records its base ref in c.typeRefs; a self-derivation
// is present iff that ref equals the type's own name qn. When it is absent (a
// different base, an imported same-local base in another namespace, or no
// derivation at all) the redefine is invalid: a schema error is reported and
// false is returned so the caller skips the self-reference patch. The rule is
// version-independent — xs:redefine carries the same constraint in XSD 1.0 and
// 1.1.
func (c *compiler) checkRedefineSelfDerivation(ctx context.Context, elem *helium.Element, newType *TypeDef, qn QName, kind string) bool {
	if refQN, ok := c.typeRefs[newType]; ok && refQN == qn {
		return true
	}
	deriv := "restriction or extension"
	if kind == elemSimpleType {
		deriv = "restriction"
	}
	c.schemaError(ctx, schemaParserError(c.diagSource(), elem.Line(), elem.LocalName(), kind,
		fmt.Sprintf("src-redefine.5: The %s '%s' redefined inside <redefine> must have a %s whose 'base' names itself ('%s').", kind, qn.Local, deriv, qn.Local)))
	return false
}

// checkRedefineGroupRestriction enforces src-redefine.6.2 (§4.2.3): a
// redefining <group> with no self-reference must be a valid restriction of the
// original group. Redefine override processing runs before the normal group-ref
// resolver, so the original group is cloned with group-reference placeholders
// expanded just for this check. The real schema tree is left for the normal
// resolver so diagnostics and all-group reference checks stay centralized.
func (c *compiler) checkRedefineGroupRestriction(ctx context.Context, elem *helium.Element, qn QName, origGroup *ModelGroup, phaseAGroups map[QName]*ModelGroup) {
	redef := c.schema.groups[qn]
	if redef == nil || origGroup == nil {
		return
	}
	expandedOrig, ok := c.expandGroupRefsForRedefineRestriction(origGroup, phaseAGroups)
	if !ok {
		return
	}
	if redefineGroupValidRestriction(ctx, redef, expandedOrig, c.schema, c.version) {
		return
	}
	c.schemaError(ctx, schemaParserError(c.diagSource(), elem.Line(), elem.LocalName(), "group",
		fmt.Sprintf("src-redefine.6.2: The redefinition of group '%s' is not a valid restriction of the original group.", qn.Local)))
}

func (c *compiler) expandGroupRefsForRedefineRestriction(mg *ModelGroup, phaseAGroups map[QName]*ModelGroup) (*ModelGroup, bool) {
	return c.expandGroupRefsForRedefineRestrictionVisit(mg, phaseAGroups, make(map[*ModelGroup]*ModelGroup), make(map[QName]struct{}))
}

func (c *compiler) expandGroupRefsForRedefineRestrictionVisit(mg *ModelGroup, phaseAGroups map[QName]*ModelGroup, cloned map[*ModelGroup]*ModelGroup, resolving map[QName]struct{}) (*ModelGroup, bool) {
	if mg == nil {
		return nil, true
	}
	if qn, isRef := c.groupRefs[mg]; isRef {
		target, ok := c.lookupGroupForRef(qn, phaseAGroups)
		if !ok {
			return nil, false
		}
		if _, recursive := resolving[qn]; recursive {
			return nil, false
		}
		resolving[qn] = struct{}{}
		expanded, ok := c.expandGroupRefsForRedefineRestrictionVisit(target, phaseAGroups, cloned, resolving)
		delete(resolving, qn)
		if !ok || expanded == nil {
			return nil, false
		}
		return &ModelGroup{
			Compositor: expanded.Compositor,
			Particles:  expanded.Particles,
			MinOccurs:  mg.MinOccurs,
			MaxOccurs:  mg.MaxOccurs,
		}, true
	}
	if existing := cloned[mg]; existing != nil {
		return existing, true
	}
	out := &ModelGroup{
		Compositor: mg.Compositor,
		MinOccurs:  mg.MinOccurs,
		MaxOccurs:  mg.MaxOccurs,
	}
	cloned[mg] = out
	if len(mg.Particles) == 0 {
		return out, true
	}
	out.Particles = make([]*Particle, 0, len(mg.Particles))
	for _, p := range mg.Particles {
		if p == nil {
			continue
		}
		cp := *p
		if sub, ok := p.Term.(*ModelGroup); ok {
			expanded, ok := c.expandGroupRefsForRedefineRestrictionVisit(sub, phaseAGroups, cloned, resolving)
			if !ok {
				return nil, false
			}
			cp.Term = expanded
		}
		out.Particles = append(out.Particles, &cp)
	}
	return out, true
}

func (c *compiler) lookupGroupForRef(qn QName, groups map[QName]*ModelGroup) (*ModelGroup, bool) {
	grp, ok := groups[qn]
	if ok {
		return grp, true
	}
	if qn.NS != "" {
		grp, ok = groups[QName{Local: qn.Local}]
		if ok {
			return grp, true
		}
	}
	return c.lookupGroupForRefFallback(qn)
}

func (c *compiler) lookupGroupForRefFallback(qn QName) (*ModelGroup, bool) {
	grp, ok := c.schema.groups[qn]
	if ok {
		return grp, true
	}
	if qn.NS != "" {
		grp, ok = c.schema.groups[QName{Local: qn.Local}]
		return grp, ok
	}
	return nil, false
}

func redefineGroupValidRestriction(ctx context.Context, redef, origGroup *ModelGroup, schema *Schema, version Version) bool {
	derivedP := &Particle{MinOccurs: redef.MinOccurs, MaxOccurs: redef.MaxOccurs, Term: redef}
	baseP := &Particle{MinOccurs: origGroup.MinOccurs, MaxOccurs: origGroup.MaxOccurs, Term: origGroup}

	// XSD 1.0 applies the intensional Particle Valid (Restriction) rules. For
	// all->all redefine groups that means member mapping is source-order
	// preserving; the XSD 1.1 language-subset relaxation below is what makes the
	// same all-member set in a different declaration order valid.
	if version != Version11 && redef.Compositor == CompositorAll && origGroup.Compositor == CompositorAll {
		if !occurrenceEmptiableRestriction(derivedP, baseP, version) {
			return false
		}
		return recurseOrdered(ctx, redef.Particles, origGroup.Particles, schema, version)
	}

	if particleValidRestriction(ctx, derivedP, baseP, schema, version) {
		return true
	}
	if version == Version11 && particleLanguageSubset(ctx, derivedP, baseP, schema, version) {
		return true
	}
	return false
}

func (c *compiler) checkRedefineOverrideRepresentation(ctx context.Context, redefineElem *helium.Element) {
	for child := range helium.Children(redefineElem) {
		if child.Type() != helium.ElementNode {
			continue
		}
		elem, ok := helium.AsNode[*helium.Element](child)
		if !ok || !isXSDElement(elem, elemGroup) {
			continue
		}
		c.checkRedefineGroupDefinitionRepresentation(ctx, elem)
	}
}

func (c *compiler) checkRedefineGroupDefinitionRepresentation(ctx context.Context, elem *helium.Element) {
	if c.filename == "" {
		return
	}
	if hasAttr(elem, attrRef) {
		c.schemaError(ctx, schemaParserErrorAttr(c.diagSource(), elem.Line(), elem.LocalName(), "group", attrRef,
			"The attribute 'ref' is not allowed on a model group definition."))
	}
	for child := range helium.Children(elem) {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce, ok := helium.AsNode[*helium.Element](child)
		if !ok || !isXSDElement(ce, elemAll) {
			continue
		}
		c.validateAllOccurs(ctx, ce)
		for grand := range helium.Children(ce) {
			if grand.Type() != helium.ElementNode {
				continue
			}
			ge, ok := helium.AsNode[*helium.Element](grand)
			if ok && isXSDElement(ge, elemElement) {
				c.checkAllElementParticleOccurs(ctx, ge)
			}
		}
	}
}

func redefineGroupHasUnprefixedSelfRef(elem *helium.Element, local string) bool {
	for child := range helium.Children(elem) {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		if isXSDElement(ce, elemGroup) && hasAttr(ce, attrRef) {
			ref := normalizeWhiteSpace(getAttr(ce, attrRef), "collapse")
			if ref == local && refChameleonEligible(ce, ref) {
				return true
			}
		}
		if redefineGroupHasUnprefixedSelfRef(ce, local) {
			return true
		}
	}
	return false
}

// processRedefineOverrides applies the override children of an xs:redefine
// element against phaseAKeys, the component set loaded from the redefined
// document. consumed, when non-nil, is the cross-redefine consumption set shared
// with the document's redefinableSet cache, so a component already redefined by
// an EARLIER xs:redefine of the same document is rejected as a duplicate. Each
// accepted override replaces its same-named Phase-A component exactly once; an
// override targeting a name absent from Phase A, repeated within this element,
// or already consumed by an earlier redefine is reported as a duplicate. The
// override children belong to the REDEFINING schema, so the caller must have
// restored that schema's per-document defaults and include-file label first.
// validateComponentChildName validates the @name of a NAMED top-level component
// child (complexType/simpleType/element/attribute/group/attributeGroup/notation)
// that a name-keyed component-dispatch loop — xs:redefine (processRedefineOverrides)
// or xs:override (collectOverrideChildren) — uses as its dispatch/match KEY. Because
// the child is keyed by name BEFORE its named parser runs, a malformed @name would
// otherwise be silently dropped (matching no target) and never validated. This ONE
// shared gate closes that hole for every such loop:
//   - @name ABSENT (hasAttr false): returns ("", false) with NO error — the caller
//     keeps its existing behavior (the child is not a valid component key).
//   - @name PRESENT but not a valid xs:NCName after collapse (empty "", whitespace-
//     only "   ", or malformed like "a b"/"1x"/"a:b"): reports the invalid-NCName
//     schema-representation error (with the child's own component label, matching its
//     named parser) and returns (collapsedName, false).
//   - @name PRESENT and a valid NCName: returns (collapsedName, true).
func (c *compiler) validateComponentChildName(ctx context.Context, elem *helium.Element) (string, bool) {
	name := collapsedAttr(elem, attrName)
	if !hasAttr(elem, attrName) {
		return name, false
	}
	if xmlchar.IsValidNCName(name) {
		return name, true
	}
	if c.filename != "" {
		msg := "The value '" + name + "' of attribute 'name' is not a valid 'xs:NCName'."
		switch {
		case isXSDElement(elem, elemComplexType):
			c.schemaError(ctx, schemaComponentError(c.diagSource(), elem.Line(), elem.LocalName(), componentLocalComplexType, msg))
		case isXSDElement(elem, elemSimpleType):
			c.schemaError(ctx, schemaComponentError(c.diagSource(), elem.Line(), elem.LocalName(), componentLocalSimpleType, msg))
		default:
			// group/attributeGroup/element/attribute/notation: elem.LocalName() is the
			// schema-for-schemas element name (e.g. "group"), matching each kind's own
			// named-parser diagnostic.
			c.schemaError(ctx, schemaParserError(c.diagSource(), elem.Line(), elem.LocalName(), elem.LocalName(), msg))
		}
	}
	return name, false
}

func (c *compiler) processRedefineOverrides(ctx context.Context, redefineElem *helium.Element, phaseAKeys, consumed map[redefineKind]map[QName]struct{}, chameleon bool, phaseAGroups map[QName]*ModelGroup) error {
	savedElemForm := c.schema.elemFormQualified
	savedAttrForm := c.schema.attrFormQualified
	savedBlockDefault := c.schema.blockDefault
	savedFinalDefault := c.schema.finalDefault
	savedIncludeFile := c.includeFile
	if phaseAGroups == nil {
		phaseAGroups = map[QName]*ModelGroup{}
	}
	c.redefine = &redefineState{
		phaseAKeys: phaseAKeys,
		seen:       make(map[redefineKind]map[QName]struct{}),
		consumed:   consumed,
	}
	defer func() { c.redefine = nil }()
	for child := range helium.Children(redefineElem) {
		if child.Type() != helium.ElementNode {
			continue
		}
		elem, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		switch {
		case isXSDElement(elem, elemAnnotation):
			// skip
		case isXSDElement(elem, elemComplexType):
			name, ok := c.validateComponentChildName(ctx, elem)
			if !ok {
				// Malformed @name (empty/whitespace-only/"a b") reported; absent @name
				// silently skipped — the name-keyed dispatch matches no target.
				continue
			}
			qn := QName{Local: name, NS: c.schema.targetNamespace}
			// Validate and consume the override target before any parse side
			// effects: it must name a type loaded from the redefined schema
			// (Phase A) and may be overridden only once.
			if !c.consumeRedefineTarget(ctx, elem, redefineKindComplexType, qn, "complexType", "A global type definition") {
				continue
			}
			origType := c.schema.types[qn]
			if err := c.parseNamedComplexType(ctx, elem); err != nil {
				c.schema.elemFormQualified = savedElemForm
				c.schema.attrFormQualified = savedAttrForm
				c.schema.blockDefault = savedBlockDefault
				c.schema.finalDefault = savedFinalDefault
				c.includeFile = savedIncludeFile
				return err
			}
			// Patch self-reference: redirect the typeRef to a temporary key
			// holding the original type, so resolveRefs handles extension
			// merge (content model + attribute inheritance) naturally.
			newType := c.schema.types[qn]
			if !c.checkRedefineSelfDerivation(ctx, elem, newType, qn, elemComplexType) {
				continue
			}
			if origType != nil {
				origKey := QName{Local: "\x00redefine:" + name, NS: qn.NS}
				c.schema.types[origKey] = origType
				c.typeRefs[newType] = origKey
			}
		case isXSDElement(elem, elemSimpleType):
			name, ok := c.validateComponentChildName(ctx, elem)
			if !ok {
				continue
			}
			qn := QName{Local: name, NS: c.schema.targetNamespace}
			if !c.consumeRedefineTarget(ctx, elem, redefineKindSimpleType, qn, "simpleType", "A global type definition") {
				continue
			}
			origType := c.schema.types[qn]
			if err := c.parseNamedSimpleType(ctx, elem); err != nil {
				c.schema.elemFormQualified = savedElemForm
				c.schema.attrFormQualified = savedAttrForm
				c.schema.blockDefault = savedBlockDefault
				c.schema.finalDefault = savedFinalDefault
				c.includeFile = savedIncludeFile
				return err
			}
			newType := c.schema.types[qn]
			if !c.checkRedefineSelfDerivation(ctx, elem, newType, qn, elemSimpleType) {
				continue
			}
			if origType != nil {
				origKey := QName{Local: "\x00redefine:" + name, NS: qn.NS}
				c.schema.types[origKey] = origType
				c.typeRefs[newType] = origKey
			}
		case isXSDElement(elem, elemGroup):
			if hasAttr(elem, attrRef) {
				c.schemaError(ctx, schemaParserErrorAttr(c.diagSource(), elem.Line(), elem.LocalName(), "group", attrRef,
					"The attribute 'ref' is not allowed on a model group definition."))
				continue
			}
			name, ok := c.validateComponentChildName(ctx, elem)
			if !ok {
				continue
			}
			qn := QName{Local: name, NS: c.schema.targetNamespace}
			if !c.consumeRedefineTarget(ctx, elem, redefineKindGroup, qn, "group", "A global model group definition") {
				continue
			}
			origGroup := c.schema.groups[qn]
			// Snapshot existing groupRefs keys.
			existingRefs := make(map[*ModelGroup]bool, len(c.groupRefs))
			for mg := range c.groupRefs {
				existingRefs[mg] = true
			}
			if err := c.parseNamedGroup(ctx, elem); err != nil {
				c.schema.elemFormQualified = savedElemForm
				c.schema.attrFormQualified = savedAttrForm
				c.schema.blockDefault = savedBlockDefault
				c.schema.finalDefault = savedFinalDefault
				c.includeFile = savedIncludeFile
				return err
			}
			chameleonUnprefixedSelfRef := chameleon && qn.NS != "" && redefineGroupHasUnprefixedSelfRef(elem, qn.Local)
			if chameleonUnprefixedSelfRef {
				c.schemaError(ctx, schemaParserError(c.diagSource(), elem.Line(), elem.LocalName(), "group",
					fmt.Sprintf("src-redefine.6.1.1: The self-reference to chameleon group '%s' inside <redefine> must use a qualified name in the redefining namespace.", qn.Local)))
			}
			// Patch self-reference: find newly-added groupRefs entries referencing qn.
			if origGroup != nil {
				// Collect the newly-parsed group references in the redefine child. A
				// group reference inside a redefining group must name the group being
				// redefined; non-self refs are invalid and cannot be left for the normal
				// resolver because this path runs before reference resolution.
				var selfRefs []*ModelGroup
				var nonSelfRef QName
				var nonSelfRefSeen bool
				for mg, refQN := range c.groupRefs {
					if existingRefs[mg] {
						continue
					}
					if refQN == qn {
						selfRefs = append(selfRefs, mg)
						continue
					}
					if !nonSelfRefSeen {
						nonSelfRef = refQN
						nonSelfRefSeen = true
					}
				}
				if nonSelfRefSeen {
					c.schemaError(ctx, schemaParserError(c.diagSource(), elem.Line(), elem.LocalName(), "group",
						fmt.Sprintf("src-redefine.6.1.1: The group '%s' redefined inside <redefine> must not reference a different group ('%s').", qn.Local, nonSelfRef.Local)))
				}
				// src-redefine.6.1.1/6.1.2 (§4.2.3): a <group> child of <redefine>
				// that references itself must do so exactly ONCE and with
				// minOccurs = maxOccurs = 1. A group without a self-reference is
				// governed by clause 6.2 (valid restriction of the original) and is
				// not constrained here. Version-independent, so enforced in XSD 1.0.
				if len(selfRefs) > 1 {
					c.schemaError(ctx, schemaParserError(c.diagSource(), elem.Line(), elem.LocalName(), "group",
						fmt.Sprintf("src-redefine.6.1.1: The group '%s' redefined inside <redefine> must not reference itself more than once.", qn.Local)))
				}
				for _, mg := range selfRefs {
					if mg.MinOccurs != 1 || mg.MaxOccurs != 1 {
						c.schemaError(ctx, schemaParserError(c.diagSource(), elem.Line(), elem.LocalName(), "group",
							fmt.Sprintf("src-redefine.6.1.2: The self-reference to group '%s' inside <redefine> must have minOccurs = maxOccurs = 1.", qn.Local)))
					}
					// The self-reference resolves to the original group's content.
					// resolveRefs deletes this entry from groupRefs before it can
					// run checkAllGroupRef, so the all-group placement rule
					// (cos-all-limited) would be bypassed for a redefine that nests
					// an all-group self-reference inside a sequence/choice. Enforce
					// it here, while the source record is still available, before the
					// entry is removed.
					if origGroup.Compositor == CompositorAll {
						c.checkAllGroupRef(ctx, mg)
					}
					mg.Compositor = origGroup.Compositor
					mg.Particles = origGroup.Particles
					delete(c.groupRefs, mg)
				}
				// src-redefine.6.2 (§4.2.3): a redefining <group> with NO
				// self-reference must be a ·valid restriction· of the original
				// group (Particle Valid (Restriction) §3.9.6). Enforce the
				// provably-sound core of that rule here.
				if len(selfRefs) == 0 && !nonSelfRefSeen {
					c.checkRedefineGroupRestriction(ctx, elem, qn, origGroup, phaseAGroups)
				}
			}
		case isXSDElement(elem, elemAttributeGroup):
			name, ok := c.validateComponentChildName(ctx, elem)
			if !ok {
				continue
			}
			qn := QName{Local: name, NS: c.schema.targetNamespace}
			// This case writes c.schema.attrGroups directly (bypassing
			// parseNamedAttributeGroup), so enforce the redefine duplicate rule
			// here: the override must target a Phase-A attribute group and may
			// consume it only once. A target absent from Phase A or repeated is
			// reported and skipped.
			if !c.consumeRedefineTarget(ctx, elem, redefineKindAttrGroup, qn, "attributeGroup", "A global attribute group definition") {
				continue
			}
			origAttrs := c.schema.attrGroups[qn]
			origEffectiveAttrs := c.expandAttrGroupUses(qn, map[QName]struct{}{})
			origEffectiveWildcard := c.attrGroupCompleteWildcard(qn, map[QName]struct{}{})
			// The override REPLACES the Phase-A attribute group, so the nested
			// attribute-group ref set must be rebuilt from the redefining group's
			// children. Snapshot the Phase-A refs first (for self-reference
			// expansion), then clear the slot so stale Phase-A refs cannot leak and
			// the override's own non-self refs are recorded below. Without this,
			// checkAttrGroupDuplicates would flatten the wrong reference set (old
			// refs leak, new refs are ignored).
			origRefChildren := c.attrGroupRefChildren[qn]
			origRefSources := c.attrGroupRefSources[qn]
			delete(c.attrGroupRefChildren, qn)
			delete(c.attrGroupRefSources, qn)
			// XSD 1.1: the override REPLACES the Phase-A group's xs:anyAttribute too.
			// Snapshot the original group wildcard (a self-reference re-contributes
			// it) and clear the slot so a stale base wildcard cannot leak.
			origWildcard := c.attrGroupWildcards[qn]
			delete(c.attrGroupWildcards, qn)
			var ownWildcard, selfRefWildcard *Wildcard
			var selfRefSeen bool
			var anyAttributeSeen bool
			reportAfterWildcard := func(gce *helium.Element) {
				c.schemaError(ctx, schemaParserError(c.diagSource(), gce.Line(), gce.LocalName(), "attributeGroup",
					fmt.Sprintf("The attribute declaration '%s' must appear before the attribute wildcard 'anyAttribute'.", gce.LocalName())))
			}
			// Build the new attribute list manually, expanding self-references
			// inline. parseNamedAttributeGroup only collects xs:attribute children
			// and doesn't handle xs:attributeGroup ref children within a definition.
			var attrs []*AttrUse
			for gc := range helium.Children(elem) {
				if gc.Type() != helium.ElementNode {
					continue
				}
				gce, ok := helium.AsNode[*helium.Element](gc)
				if !ok {
					continue
				}
				switch {
				case isXSDElement(gce, elemAttribute):
					if c.version == Version11 && anyAttributeSeen {
						reportAfterWildcard(gce)
						continue
					}
					// A use="prohibited" attribute is pointless inside an
					// <xs:attributeGroup> (including a redefine override): libxml2
					// warns and skips it so a referencing wildcard still admits the
					// attribute. Mirror parseNamedAttributeGroup here.
					if getAttr(gce, attrUse) == attrValProhibited {
						if c.filename != "" {
							c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserWarning(c.diagSource(), gce.Line(), gce.LocalName(), "attribute",
								"Skipping attribute use prohibition, since it is pointless inside an <attributeGroup>."), helium.ErrorLevelWarning))
						}
						continue
					}
					au := c.parseAttributeUse(ctx, gce)
					attrs = append(attrs, au)
				case isXSDElement(gce, elemAnyAttribute):
					if anyAttributeSeen {
						if c.version == Version11 {
							c.schemaError(ctx, schemaParserError(c.diagSource(), gce.Line(), gce.LocalName(), "attributeGroup",
								fmt.Sprintf("An attribute group definition must not have more than one attribute wildcard (found a second '%s').", gce.LocalName())))
						}
						continue
					}
					anyAttributeSeen = true
					if c.version == Version11 {
						ownWildcard = c.parseAnyAttribute(ctx, gce)
					} else {
						ns := WildcardNSAny
						if hasAttr(gce, attrNamespace) {
							ns = getAttr(gce, attrNamespace)
						}
						ownWildcard = &Wildcard{
							Namespace:       ns,
							ProcessContents: quietProcessContents(gce),
							TargetNS:        c.schema.targetNamespace,
						}
					}
				case isXSDElement(gce, elemAttributeGroup):
					if c.version == Version11 && anyAttributeSeen {
						reportAfterWildcard(gce)
						continue
					}
					// Dispatch on PRESENCE via resolveQNameRef: a PRESENT-but-empty
					// ref="" is an invalid (empty) QName, reported once, not silently
					// dropped. The invalidQName sentinel it yields never equals the
					// redefined group's name, so it routes to the non-self default
					// branch and is skipped by checkAttrGroupRefsResolve's sentinel guard.
					if refQN, ok := c.resolveQNameRef(ctx, gce, attrRef); ok {
						switch refQN {
						case qn:
							// A self-reference resolves to the Phase-A group content,
							// including its xs:anyAttribute wildcard.
							if origAttrs != nil {
								attrs = append(attrs, origAttrs...)
							}
							selfRefSeen = true
							selfRefWildcard = origWildcard
							if len(origRefChildren) > 0 {
								c.attrGroupRefChildren[qn] = append(c.attrGroupRefChildren[qn], origRefChildren...)
								c.attrGroupRefSources[qn] = append(c.attrGroupRefSources[qn], origRefSources...)
							}
						default:
							// A non-self nested ref in the override is recorded so
							// checkAttrGroupDuplicates flattens the redefining ref set.
							c.attrGroupRefChildren[qn] = append(c.attrGroupRefChildren[qn], refQN)
							c.attrGroupRefSources[qn] = append(c.attrGroupRefSources[qn], attrGroupSource{line: gce.Line(), source: c.diagSource()})
						}
					}
				}
			}
			c.schema.attrGroups[qn] = attrs
			// Store the override's effective group wildcard: the group's own
			// xs:anyAttribute INTERSECTED with the original wildcard a self-reference
			// re-contributes. The type's "complete wildcard" further intersects the
			// non-self refs recorded above at link time.
			if w := combineGroupWildcards(c.version, ownWildcard, selfRefWildcard); w != nil {
				c.attrGroupWildcards[qn] = w
			}
			if !selfRefSeen {
				derivedAttrs := c.expandAttrGroupUses(qn, map[QName]struct{}{})
				derivedWildcard := c.attrGroupCompleteWildcard(qn, map[QName]struct{}{})
				c.checkRedefineAttrGroupRestriction(ctx, elem, qn, derivedAttrs, derivedWildcard, origEffectiveAttrs, origEffectiveWildcard)
			}
			// The override REPLACES the Phase-A attribute group, so re-record its
			// source to the redefining file/line (c.includeFile is the redefining
			// label here). Without this the duplicate-attribute-use diagnostic over
			// this group would keep the stale Phase-A source from parseNamedAttribute
			// Group and cite the redefined (base) file instead of the redefine.
			c.attrGroupSources[qn] = attrGroupSource{line: elem.Line(), source: c.diagSource()}
		}
	}

	// Restore form-qualified settings, defaults, and include file.
	c.schema.elemFormQualified = savedElemForm
	c.schema.attrFormQualified = savedAttrForm
	c.schema.blockDefault = savedBlockDefault
	c.schema.finalDefault = savedFinalDefault
	c.includeFile = savedIncludeFile

	return nil
}

// loadImport loads an imported schema and merges its declarations. The ns
// argument is the namespace declared on the <xs:import> element; the imported
// schema's targetNamespace must match it (XSD src-import / libxml2 semantics):
// when ns is present the imported schema must declare that targetNamespace, and
// when ns is absent the imported schema must have no targetNamespace. A
// mismatch is a fatal schema error, not an I/O warning, so importElem carries
// the source line for the diagnostic.
func (c *compiler) loadImport(ctx context.Context, location, ns string, importElem *helium.Element) error {
	// Bound the import recursion. Each sub-compiler inherits this limit
	// and tracks its own depth so namespace-cycling chains (A → B → C → A …)
	// cannot exhaust memory / stack even when every link uses a distinct
	// namespace URI.
	if c.importDepth+1 > c.maxImportDepth {
		return fmt.Errorf("%w (limit=%d, location=%q)", errImportDepthExceeded, c.maxImportDepth, location)
	}

	path, err := validateSchemaPath(c.baseDir, location)
	if err != nil {
		return fmt.Errorf("xsd: failed to load import %q: %w", location, err)
	}

	// Circular-import guard: if this document is already on the active import
	// ancestry it is mid-parse in an ancestor loader, so a mutual (A ↔ B) or self
	// import would recurse forever and can contribute no new components. Short-
	// circuit the RELOAD — the ancestor provides the document's components and
	// cross-namespace references resolve at the top level after all sub-compilers
	// merge up. Pushed before descending and popped on unwind (via the shared map)
	// so a diamond import of the same document on two disjoint branches still loads
	// independently. src-import validity is NOT skipped: the back-edge's requested
	// namespace must still match the namespace recorded for the mid-parse target (the
	// target's targetNamespace, validated on its first non-cycle load), else it is a
	// genuine src-import mismatch — including the namespace-absent case (ns=""), which
	// must import a no-targetNamespace schema — that the acyclic path would report.
	if expectedNS, active := c.importActive[path]; active {
		if ns != expectedNS {
			displayLoc := location
			if c.filename != "" {
				displayLoc = schemaDisplayLoc(c.filename, location)
			}
			c.schemaError(ctx, schemaParserError(c.filename, importElem.Line(),
				importElem.LocalName(), elemImport,
				"The namespace '"+expectedNS+"' of the imported schema '"+displayLoc+"' differs from the requested namespace '"+ns+"'."))
		}
		return nil
	}
	c.importActive[path] = ns
	defer delete(c.importActive, path)

	data, err := c.readNestedSchema(path)
	if err != nil {
		return fmt.Errorf("xsd: failed to load import %q: %w", location, err)
	}

	doc, err := c.parse(ctx, data)
	if err != nil {
		return fmt.Errorf("xsd: failed to parse import %q: %w: %w", location, errSchemaContentInvalid, err)
	}

	impRoot := findDocumentElement(doc)
	if impRoot == nil || !isXSDElement(impRoot, elemSchema) {
		return fmt.Errorf("xsd: imported document %q is not an xs:schema: %w", location, errSchemaContentInvalid)
	}

	// Compute display filename for the imported schema (for error messages).
	var impFilename string
	if c.filename != "" {
		impFilename = schemaDisplayLoc(c.filename, location)
	}

	// Create a temporary compiler for the imported schema. The imported schema is
	// compiled under the importing schema's effective XSD version.
	impC := &compiler{
		schema: &Schema{
			version:     c.version,
			elements:    make(map[QName]*ElementDecl),
			types:       make(map[QName]*TypeDef),
			groups:      make(map[QName]*ModelGroup),
			attrGroups:  make(map[QName][]*AttrUse),
			globalAttrs: make(map[QName]*AttrUse),
			substGroups: make(map[QName][]*ElementDecl),
		},
		version:                   c.version,
		baseDir:                   schemaBaseDir(path),
		fsys:                      c.fsys,
		parser:                    c.parser,
		typeRefs:                  make(map[*TypeDef]QName),
		elemRefs:                  make(map[*ElementDecl]QName),
		elemRefSources:            make(map[*ElementDecl]elemRefSource),
		groupRefs:                 make(map[*ModelGroup]QName),
		groupRefSources:           make(map[*ModelGroup]groupRefSource),
		groupSources:              make(map[QName]groupSource),
		attrGroupSources:          make(map[QName]attrGroupSource),
		attrGroupRefs:             make(map[*TypeDef][]QName),
		attrGroupRefUseSources:    make(map[*TypeDef][]attrGroupRefUseSource),
		defaultAttrUses:           make(map[*TypeDef]map[QName]*AttrUse),
		attrGroupRefChildren:      make(map[QName][]QName),
		attrGroupRefSources:       make(map[QName][]attrGroupSource),
		attrGroupWildcards:        make(map[QName]*Wildcard),
		globalElemSources:         make(map[*ElementDecl]elemRefSource),
		typeDefSources:            make(map[*TypeDef]typeDefSource),
		typeKinds:                 make(map[QName]redefineKind),
		itemTypeRefs:              make(map[*TypeDef]QName),
		chameleonEligible:         make(map[any]struct{}),
		attrRefs:                  make(map[*AttrUse]QName),
		attrRefSources:            make(map[*AttrUse]attrConstraintSource),
		attrUseConstraintSources:  make(map[*AttrUse]attrConstraintSource),
		attrUseSources:            make(map[*AttrUse]attrConstraintSource),
		elemDeclConstraintSources: make(map[*ElementDecl]attrConstraintSource),
		filename:                  impFilename,
		importedNS:                make(map[string]string),
		importDeclaredNS:          make(map[string]struct{}),
		docImportedNS:             make(map[string]map[string]struct{}),
		importDepth:               c.importDepth + 1,
		maxImportDepth:            c.maxImportDepth,
		// Share the active import-ancestry set by pointer so the whole import tree
		// sees a consistent load stack and any cycle (through this sub-compiler's own
		// nested imports/includes) is detected and cut.
		importActive:      c.importActive,
		includeVisited:    make(map[string]struct{}),
		maxIncludeDepth:   c.maxIncludeDepth,
		loadedRedefinable: make(map[string]*redefinableSet),
		notations:         make(map[QName]struct{}),
	}

	// Seed the imported sub-compiler's circular-include guard with the imported
	// schema's own resolved key, mirroring CompileFile's seeding of the top-level
	// root. Without this, an imported schema that circularly includes back to its
	// own root (import imp.xsd -> include inc.xsd -> include imp.xsd) re-parses
	// imp.xsd and emits spurious duplicate-component errors.
	impC.includeVisited[path] = struct{}{}
	// The imported schema is this sub-compiler's root: record it so an xs:override
	// cascade inside the imported schema that points back at its own root
	// terminates without re-loading it (mirrors the top-level rootKey seeding).
	impC.rootKey = path

	// Sub-compiler collects errors into its own collector so we can
	// conditionally forward them. This matches libxml2's behavior of
	// stopping error reporting after the first import failure.
	subCollector := helium.NewErrorCollector(ctx, helium.ErrorLevelNone)
	// Guarantee the sub-collector's backing sink is drained on every exit path,
	// including the fatal early returns below (parseSchemaChildren failure and the
	// include-depth/path-escape/resource-limit fatal nested-load). Close is
	// idempotent, so the explicit Close before the Errors() read below still runs
	// and the collected diagnostics remain available for forwarding.
	defer func() { _ = subCollector.Close() }()
	impC.errorHandler = subCollector

	// propagateImpErrors drains the import sub-compiler's collected diagnostics and
	// folds its error count into the parent. Preserving libxml2's "stop after the
	// first import failure" rule, the diagnostic TEXT is forwarded only when the
	// parent has no prior errors, but impC.errorCount is ALWAYS added so an
	// imported-schema failure still fails the compile. It is IDEMPOTENT (a
	// `propagated` guard makes the second and later calls no-ops) and is `defer`red
	// immediately below so EVERY exit path after the sub-collector is installed —
	// including the parseSchemaChildren-error and fatal nested-load early returns —
	// flushes the sub-compiler's diagnostics. The explicit calls that remain only
	// fix ORDERING (forward while the parent is still error-free, BEFORE a TNS error
	// is reported; skip the declaration merge when the import failed). Close is
	// idempotent, so the separate Close defer above stays harmless.
	propagated := false
	propagateImpErrors := func() {
		if propagated {
			return
		}
		propagated = true
		_ = subCollector.Close()
		if impC.errorCount == 0 {
			return
		}
		if c.errorCount == 0 {
			for _, e := range subCollector.Errors() {
				c.errorHandler.Handle(ctx, e)
			}
		}
		c.errorCount += impC.errorCount
	}
	// Guaranteed flush on every remaining exit path (idempotent; explicit calls
	// below run first where ordering matters and turn this into a no-op).
	defer propagateImpErrors()

	impC.schema.targetNamespace = getAttr(impRoot, attrTargetNamespace)
	impC.schemaTargetNSSet = impC.schema.targetNamespace != ""
	impC.schema.elemFormQualified = getAttr(impRoot, attrElementFormDefault) == attrValQualified
	impC.schema.attrFormQualified = getAttr(impRoot, attrAttributeFormDefault) == attrValQualified
	if v := getAttr(impRoot, attrBlockDefault); v != "" {
		impC.schema.blockDefault = parseBlockFlags(v)
	}
	if v := getAttr(impRoot, attrFinalDefault); v != "" {
		impC.schema.finalDefault = parseFinalFlags(v)
	}
	// The imported root's @xpathDefaultNamespace governs its own
	// identity-constraint selector/field XPaths (resolveXPathDefaultNS reads the
	// sub-compiler's schemaXPathDefaultNS); without this an imported IDC selector
	// like xpath="emp" would not inherit the imported root's default namespace.
	// Resolved against the imported root now (so an inherited ##defaultNamespace
	// uses the imported root's default namespace, not a selector/field's).
	if impC.version == Version11 {
		impC.schemaXPathDefaultNS = resolveXPathDefaultNSToken(impRoot, getAttr(impRoot, attrXPathDefaultNS), impC.schema.targetNamespace)
	}

	// Seed the imported sub-compiler's CTA static context from the IMPORTED document
	// so an xs:alternative parsed there sees its own schema's static base URI
	// (fn:static-base-uri) and xpathDefaultNamespace, not the importing schema's.
	impC.schemaBaseURI = path
	if hasAttr(impRoot, attrXPathDefaultNamespace) {
		impC.xpathDefaultNSSet = true
	}
	impC.schemaXPathDefaultNSToken = getAttr(impRoot, attrXPathDefaultNamespace)

	registerBuiltinTypes(impC.schema, impC.version)

	// Conditional inclusion runs per schema document, BEFORE the src-import
	// targetNamespace check: a vc-excluded imported root contributes an EMPTY
	// schema and must not be rejected for a namespace mismatch. A malformed 1.1 vc
	// value on the (excluded or non-excluded) root is still a schema error, so its
	// diagnostics are propagated on every exit path below.
	if impC.applyConditionalInclusion(ctx, impRoot) {
		propagateImpErrors()
		return nil
	}

	// The targetNamespace of the located schema must match the namespace
	// declared on <xs:import> (XSD src-import; libxml2 rejects the mismatch
	// rather than merging declarations from the wrong namespace). A present
	// namespace requires that exact targetNamespace; an absent namespace
	// requires the imported schema to have no targetNamespace. This is a
	// fatal schema error, not an I/O warning, so it is emitted directly here
	// and reported via a nil return (mirroring the xs:include check above).
	// Pre-pass diagnostics collected in impC are flushed FIRST (while the parent
	// is still error-free) so they are not dropped by this early return.
	impTargetNS := getAttr(impRoot, attrTargetNamespace)
	if impTargetNS != ns {
		propagateImpErrors()
		displayLoc := location
		if c.filename != "" {
			displayLoc = schemaDisplayLoc(c.filename, location)
		}
		c.schemaError(ctx, schemaParserError(c.filename, importElem.Line(),
			importElem.LocalName(), elemImport,
			"The namespace '"+impTargetNS+"' of the imported schema '"+displayLoc+"' differs from the requested namespace '"+ns+"'."))
		return nil
	}

	if impC.version == Version11 {
		impC.readSchemaDefaultAttributes(ctx, impRoot)
		// The imported document's own <xs:defaultOpenContent> applies to its complex
		// types (it is per-document and does not cross the import boundary). Read it
		// AFTER applyConditionalInclusion (above), matching xs:include/xs:redefine, so
		// a vc-excluded <xs:defaultOpenContent> is not captured and applied.
		impC.defaultOpenContent = impC.readDefaultOpenContent(ctx, impRoot)
	}
	// The @id xs:ID validity/uniqueness and identity-constraint placement/content
	// rules are version-independent, so enforce them on the imported document in
	// 1.0 and 1.1 alike.
	impC.checkSchemaComponentIDs(ctx, impRoot)
	impC.checkSchemaNamespaceAttrs(ctx, impRoot)
	impC.checkIDConstraintPlacement(ctx, impRoot)
	impC.checkNotations(ctx, impRoot)
	impC.checkAnnotations(ctx, impRoot)

	if err := impC.parseSchemaChildren(ctx, impRoot); err != nil {
		return err
	}

	// Process includes/imports in the imported schema (but skip back-references).
	// impC.processIncludes has ALREADY demoted every benign nested fetch miss to
	// a warning internally (via its own nestedLoadFailureFatal classifier) and
	// returns non-nil ONLY for a fatal condition — a security/policy denial, a
	// content failure (malformed XML / non-xs:schema root), or a post-fetch
	// declaration/redefine/nested-processing failure of a nested target (e.g. a
	// fetched included schema whose top-level complexType is missing its @name).
	// So the error is propagated unconditionally; downgrading it here would let a
	// fetched-but-invalid nested schema pass as a warning.
	if err := impC.processIncludes(ctx, impRoot); err != nil {
		return err
	}

	// Propagate sub-compiler diagnostics (same rule as the early-return paths).
	propagateImpErrors()
	if impC.errorCount > 0 {
		return nil
	}

	// Merge the imported schema's declarations into the main schema.
	for qn, edecl := range impC.schema.elements {
		if _, exists := c.schema.elements[qn]; !exists {
			c.schema.elements[qn] = edecl
		}
	}
	for qn, td := range impC.schema.types {
		if _, exists := c.schema.types[qn]; !exists {
			c.schema.types[qn] = td
		}
	}
	for qn, mg := range impC.schema.groups {
		if _, exists := c.schema.groups[qn]; !exists {
			c.schema.groups[qn] = mg
		}
	}
	for qn, attrs := range impC.schema.attrGroups {
		if _, exists := c.schema.attrGroups[qn]; !exists {
			c.schema.attrGroups[qn] = attrs
		}
	}
	for qn, au := range impC.schema.globalAttrs {
		if _, exists := c.schema.globalAttrs[qn]; !exists {
			c.schema.globalAttrs[qn] = au
		}
	}

	// Merge ref maps from the sub-compiler into the parent compiler.
	// This defers resolution to the parent's resolveRefs(), which has
	// access to all merged declarations (handles circular imports).
	maps.Copy(c.elemRefs, impC.elemRefs)
	maps.Copy(c.elemRefSources, impC.elemRefSources)
	maps.Copy(c.typeRefs, impC.typeRefs)
	// A namespace imported anywhere in the assembly is referenceable everywhere the
	// merged refs are resolved (the ref check runs on the parent), so union the
	// import-declared namespaces up.
	maps.Copy(c.importDeclaredNS, impC.importDeclaredNS)
	// Offset each imported type's parse-order ordinal past the parent's counter
	// so ordinals remain globally unique across the merged compilers; otherwise
	// a parent type and an imported type sharing a source line and an empty name
	// could collide on the diagnostic tie-breaker.
	base := c.nextTypeDefOrdinal
	for td, src := range impC.typeDefSources {
		src.ordinal += base
		// Preserve the originating file for imported types. A type parsed
		// directly in the imported document (not via a nested include) has an
		// empty source; attribute it to the imported file so diagnostics cite
		// the file whose line number they carry, not the importing schema.
		if src.source == "" {
			src.source = impC.filename
		}
		c.typeDefSources[td] = src
	}
	c.nextTypeDefOrdinal = base + impC.nextTypeDefOrdinal
	maps.Copy(c.groupRefs, impC.groupRefs)
	maps.Copy(c.groupRefSources, impC.groupRefSources)
	// Merge named-group source info, but only for groups the parent does not
	// already define (mirroring the schema.groups merge above): a group present
	// in both keeps the parent's declaration and source.
	for qn, src := range impC.groupSources {
		if _, exists := c.groupSources[qn]; !exists {
			c.groupSources[qn] = src
		}
	}
	// Merge attribute-group source info (mirroring the schema.attrGroups merge
	// above): a group present in both keeps the parent's declaration and source.
	for qn, src := range impC.attrGroupSources {
		if _, exists := c.attrGroupSources[qn]; !exists {
			// An attribute group parsed directly in the imported document (not via
			// a nested include) has an empty source; attribute it to the imported
			// file so its duplicate-attribute-use diagnostic cites the file whose
			// line number it carries, not the importing schema.
			if src.source == "" {
				src.source = impC.filename
			}
			c.attrGroupSources[qn] = src
		}
	}
	maps.Copy(c.attrGroupRefs, impC.attrGroupRefs)
	for _, ref := range impC.schemaDefaultAttrRefs {
		if ref.src.source == "" {
			ref.src.source = impC.filename
		}
		c.schemaDefaultAttrRefs = append(c.schemaDefaultAttrRefs, ref)
	}
	for td, srcs := range impC.attrGroupRefUseSources {
		merged := make([]attrGroupRefUseSource, len(srcs))
		for i, src := range srcs {
			if src.source == "" {
				src.source = impC.filename
			}
			merged[i] = src
		}
		c.attrGroupRefUseSources[td] = merged
	}
	for qn, refs := range impC.attrGroupRefChildren {
		if _, exists := c.attrGroupRefChildren[qn]; !exists {
			c.attrGroupRefChildren[qn] = refs
		}
	}
	// Merge attribute-group wildcards so a type in the importing schema that
	// references an imported group sees the group's xs:anyAttribute.
	for qn, wc := range impC.attrGroupWildcards {
		if _, exists := c.attrGroupWildcards[qn]; !exists {
			c.attrGroupWildcards[qn] = wc
		}
	}
	// Merge the per-edge ref sources alongside attrGroupRefChildren. A ref edge
	// parsed directly in the imported document (not via a nested include) has an
	// empty source; attribute it to the imported file so an indirect-cycle
	// diagnostic cites the file whose ref line number it carries.
	for qn, srcs := range impC.attrGroupRefSources {
		if _, exists := c.attrGroupRefSources[qn]; exists {
			continue
		}
		merged := make([]attrGroupSource, len(srcs))
		for i, src := range srcs {
			if src.source == "" {
				src.source = impC.filename
			}
			merged[i] = src
		}
		c.attrGroupRefSources[qn] = merged
	}
	maps.Copy(c.globalElemSources, impC.globalElemSources)
	maps.Copy(c.itemTypeRefs, impC.itemTypeRefs)
	maps.Copy(c.chameleonEligible, impC.chameleonEligible)
	c.unionMemberRefs = append(c.unionMemberRefs, impC.unionMemberRefs...)
	maps.Copy(c.attrRefs, impC.attrRefs)
	// Merge attribute-ref sources, preserving the originating file so the
	// ref-kind diagnostic (checkAttributeResolution) cites the file whose line it
	// carries, not the importing schema.
	for au, src := range impC.attrRefSources {
		if src.source == "" {
			src.source = impC.filename
		}
		c.attrRefSources[au] = src
	}
	maps.Copy(c.notations, impC.notations)
	// Merge attribute-use default/fixed constraint sources, preserving the
	// originating file. An attribute use parsed directly in the imported document
	// (not via a nested include) has an empty source; attribute it to the
	// imported file so the invalid-default/fixed diagnostic cites the file whose
	// line number it carries, not the importing schema.
	for au, src := range impC.attrUseConstraintSources {
		if src.source == "" {
			src.source = impC.filename
		}
		c.attrUseConstraintSources[au] = src
	}
	// Merge element-declaration default/fixed constraint sources, mirroring the
	// attribute-use merge above so an invalid element default/fixed in an imported
	// document is still checked and the diagnostic cites the imported file.
	for decl, src := range impC.elemDeclConstraintSources {
		if src.source == "" {
			src.source = impC.filename
		}
		c.elemDeclConstraintSources[decl] = src
	}
	// Merge prohibited/ref'd attribute-use sources, preserving the originating
	// file. An attribute use parsed directly in the imported document (not via a
	// nested include) has an empty source; attribute it to the imported file so
	// warnPointlessProhibition cites the file whose line it carries, not the
	// importing schema.
	for au, src := range impC.attrUseSources {
		if src.source == "" {
			src.source = impC.filename
		}
		c.attrUseSources[au] = src
	}

	// Carry the imported CTA bookkeeping into the parent so imported alternatives
	// participate in the parent's deferred resolution and checks: altTypeRefs are
	// resolved by the parent's resolveAltTypeRefs (against the merged type table),
	// and ctaElems are checked by the parent's checkAltSubstitutability.
	c.altTypeRefs = append(c.altTypeRefs, impC.altTypeRefs...)
	c.ctaElems = append(c.ctaElems, impC.ctaElems...)

	return nil
}
