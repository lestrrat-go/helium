package xmldsig1

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"path"
	"strings"

	helium "github.com/lestrrat-go/helium"
)

// ReferenceResolver supplies the octet stream for a Reference whose URI is NOT
// one of the four supported same-document forms (see [Verifier.Verify] for those
// forms). It is the opt-in seam for verifying detached signatures that reference
// content outside the signed document.
//
// A resolver is consulted ONLY for a non-same-document (external) Reference URI,
// after that URI has been joined against the document's base URI. Same-document
// references never reach it. When no resolver is configured an external
// reference stays fail-closed with [ErrReferenceNotFound], the default.
//
// The interface is public so callers can dereference references over any
// transport. helium ships only [FSReferenceResolver], a filesystem resolver with
// no network access. No HTTP resolver is provided: anyone implementing network
// dereferencing owns the resulting SSRF and availability risk (an attacker who
// controls a Reference URI could otherwise steer requests at internal hosts or
// stall verification), so that decision is left explicitly to the caller.
//
// ResolveReference must be safe to call from multiple goroutines, and should
// honor ctx cancellation. The returned octets are the resource's raw bytes; the
// package then applies every declared transform in order, parsing or
// canonicalizing only when the next transform requires the other value kind.
type ReferenceResolver interface {
	ResolveReference(ctx context.Context, uri string) ([]byte, error)
}

// maxReferenceBytes bounds the octets [FSReferenceResolver] reads for a single
// external Reference, so a large or attacker-supplied file cannot exhaust memory
// during verification. 64 MiB is far above any realistic signed external
// resource while still capping consumption; a file exceeding it fails with
// [ErrReferenceTooLarge].
const maxReferenceBytes = 64 << 20

// resolveReferenceOctets dereferences uri through resolver and enforces the
// maxReferenceBytes cap on the result. Every package-internal resolver call site
// — the external Reference digest paths (sign and verify) and RetrievalMethod
// resolution — goes through here so a caller-supplied ReferenceResolver, which
// owns its own read policy, still cannot force the package to buffer or parse a
// resource larger than 64 MiB. The shipped FSReferenceResolver already bounds its
// own read; this cap additionally guards a custom resolver. A result over the cap
// fails with ErrReferenceTooLarge before any parse or transform processing.
func resolveReferenceOctets(ctx context.Context, resolver ReferenceResolver, uri string) ([]byte, error) {
	octets, err := resolver.ResolveReference(ctx, uri)
	if err != nil {
		return nil, err
	}
	if len(octets) > maxReferenceBytes {
		return nil, fmt.Errorf("%w: external reference %q exceeds %d bytes", ErrReferenceTooLarge, uri, maxReferenceBytes)
	}
	return octets, nil
}

// fsReferenceResolver resolves external Reference URIs as slash-separated paths
// inside a fs.FS.
type fsReferenceResolver struct {
	fsys fs.FS
}

// FSReferenceResolver returns a [ReferenceResolver] that serves external
// references from fsys, treating the (already base-joined) Reference URI as a
// slash-separated path inside fsys. It performs NO network access.
//
// It is fail-closed on anything that is not a plain in-tree path:
//
//   - a URI carrying a scheme (http:, https:, file:, urn:, or any "scheme:" per
//     RFC 3986, including a Windows drive letter) is refused — the resolver never
//     interprets a scheme, so it cannot be steered into a fetch;
//   - a path escaping the root (an absolute path, or one with ".." segments that
//     leave the root after cleaning) is refused via an fs.ValidPath containment
//     check;
//   - a leftover fragment ("#...") is refused.
//
// Reads are bounded: a resource larger than 64 MiB fails with
// [ErrReferenceTooLarge] rather than being buffered in full. Every rejection
// wraps [ErrReferenceNotFound] (or [ErrReferenceTooLarge]) so callers can match
// it with errors.Is.
func FSReferenceResolver(fsys fs.FS) ReferenceResolver {
	return fsReferenceResolver{fsys: fsys}
}

func (r fsReferenceResolver) ResolveReference(ctx context.Context, uri string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	name, err := fsNameFromURI(uri)
	if err != nil {
		return nil, err
	}
	f, err := r.fsys.Open(name)
	if err != nil {
		return nil, fmt.Errorf("%w: cannot open external reference %q: %v", ErrReferenceNotFound, uri, err)
	}
	defer f.Close()

	// Read at most maxReferenceBytes+1 so an over-cap file is detected by the
	// extra byte without buffering the whole resource.
	data, err := io.ReadAll(io.LimitReader(f, maxReferenceBytes+1))
	if err != nil {
		return nil, fmt.Errorf("%w: reading external reference %q: %v", ErrReferenceNotFound, uri, err)
	}
	if len(data) > maxReferenceBytes {
		return nil, fmt.Errorf("%w: external reference %q exceeds %d bytes", ErrReferenceTooLarge, uri, maxReferenceBytes)
	}
	return data, nil
}

// fsNameFromURI converts an external Reference URI into a validated fs.FS path,
// fail-closed. It refuses a scheme URI, a leftover fragment, and any path that
// does not stay inside the root.
func fsNameFromURI(uri string) (string, error) {
	if strings.IndexByte(uri, '#') >= 0 {
		return "", fmt.Errorf("%w: reference URI %q carries a fragment", ErrReferenceNotFound, uri)
	}
	if uriHasScheme(uri) {
		return "", fmt.Errorf("%w: FSReferenceResolver refuses scheme URI %q", ErrReferenceNotFound, uri)
	}
	// path.Clean collapses "." and ".." segments; fs.ValidPath then rejects an
	// absolute path or one that still escapes the root ("..", leading "/"), the
	// repository's established containment idiom (see baseRelativeFSName).
	name := path.Clean(uri)
	if !fs.ValidPath(name) {
		return "", fmt.Errorf("%w: reference URI %q escapes the resolver root", ErrReferenceNotFound, uri)
	}
	return name, nil
}

// uriHasScheme reports whether uri carries an RFC 3986 scheme, i.e. a ":" appears
// before the first "/", "?", or "#". This catches every "scheme:" form —
// http://, https://, file:///, urn:..., a single-letter scheme, and a Windows
// drive letter ("C:\\...") — so [FSReferenceResolver] never mistakes a
// scheme-bearing URI for an in-tree path. A relative reference (RFC 3986 §4.2)
// has no ":" in its first path segment, so it is correctly not a scheme.
func uriHasScheme(uri string) bool {
	for i := range len(uri) {
		switch uri[i] {
		case '/', '?', '#':
			return false
		case ':':
			return true
		}
	}
	return false
}

// joinReferenceURI joins an external Reference URI against the document's base
// URI. With no base the URI is passed through unchanged. The join reuses
// helium.ResolveURI — the root package's byte-faithful libxml2 xmlBuildURI port,
// in conventional (base, reference) order — so a relative Reference URI resolves
// exactly as the parser resolves any other relative reference.
func joinReferenceURI(base, uri string) (string, error) {
	if base == "" {
		return uri, nil
	}
	joined, err := helium.ResolveURI(base, uri)
	if err != nil {
		return "", fmt.Errorf("%w: cannot resolve reference URI %q against base %q: %v", ErrReferenceNotFound, uri, base, err)
	}
	return joined, nil
}

// externalReferenceDigestInput is the octet-valued entry adapter shared by
// signing, verification, Manifest validation, and RetrievalMethod processing.
// The ordered executor decides every later parse or canonicalization from the
// current value kind and the next transform's contract.
func externalReferenceDigestInput(ctx context.Context, octets []byte, steps []transformStep, runtime transformRuntime) ([]byte, error) {
	runtime.external = true
	return executeTransformPipeline(ctx, runtime, newOctetTransformValue(octets), steps)
}
