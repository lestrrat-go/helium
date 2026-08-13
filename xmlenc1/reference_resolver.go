package xmlenc1

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"math"
	"path"
	"strings"

	helium "github.com/lestrrat-go/helium"
)

// ReferenceResolver supplies the octet stream an xenc:CipherReference names
// when its URI is NOT one of the four same-document forms ([Decryptor.Decrypt]
// states those forms). It is the opt-in seam for decrypting an EncryptedData
// whose cipher text lives outside the document carrying it.
//
// A resolver is consulted ONLY for a non-same-document (external) URI, after
// that URI has been joined against the document's base URI. Same-document
// references need no I/O and never reach it. When no resolver is configured an
// external reference stays fail-closed with [ErrReferenceNotFound], which is
// the default.
//
// That default is what the specification permits rather than a shortfall of it.
// W3C xmlenc-core1 §3.3.1 requires "the same URI encoding, dereferencing,
// scheme, and HTTP response codes as that of [XMLDSIG-CORE1]" and defines no
// dereferencing of its own; xmldsig-core1 §4.4.3.1 makes dereferencing URIs in
// the HTTP scheme RECOMMENDED, while §4.4.3.2 and §4.4.3.3 make the null URI,
// the shortname XPointer, and same-document dereferencing normative MUSTs. So
// the imported obligation covers URI="" and URI="#id", which this package
// implements unconditionally, and external fetching is a RECOMMENDED capability
// the caller opts into.
//
// The interface is public so a caller can dereference over any transport.
// helium ships only [FSReferenceResolver], a filesystem resolver with no
// network access. No HTTP resolver is provided: whoever implements network
// dereferencing owns the resulting SSRF and availability risk, because an
// attacker who controls a CipherReference URI could otherwise steer requests at
// internal hosts or stall a decrypt, so that decision is left explicitly to the
// caller.
//
// This interface and [github.com/lestrrat-go/helium/xmldsig1.ReferenceResolver]
// are structurally identical, so one value satisfies both and a caller that
// verifies signatures and decrypts can configure the same resolver on each.
// Neither package imports the other; the shared shape is the whole connection
// between them.
//
// ResolveReference must be safe to call from multiple goroutines, and should
// honor ctx cancellation. The returned octets are the resource's raw bytes: an
// external reference yields an octet stream, so no canonicalization applies to
// it and only a declared ds:Transform changes what the bytes mean.
type ReferenceResolver interface {
	ResolveReference(ctx context.Context, uri string) ([]byte, error)
}

// boundedReferenceResolver is the internal capability a resolver implements
// when it can apply the decrypt's remaining CipherValue allowance WHILE
// reading, instead of returning a whole resource the caller then weighs. The
// shipped FSReferenceResolver implements it, so an over-budget file is never
// buffered in full. A caller's own ReferenceResolver does not need to know
// about it: resolveExternalCipherReference falls back to charging the octets it
// returns, which bounds what this package holds but not what that resolver
// itself read.
//
// maxBytes is the budget's remaining allowance, and a negative value means
// unlimited. An implementation reads at most maxBytes+1 bytes, so the caller's
// charge of the returned length is what refuses an over-budget resource — there
// is no separate size sentinel to keep in step with the budgets.
type boundedReferenceResolver interface {
	resolveReferenceWithLimit(ctx context.Context, uri string, maxBytes int) ([]byte, error)
}

// FSReferenceResolver returns a [ReferenceResolver] that serves external
// CipherReference URIs from fsys, treating the (already base-joined) URI as a
// slash-separated path inside fsys. It performs NO network access.
//
// It is fail-closed on anything that is not a plain in-tree path:
//
//   - a URI carrying a scheme (http:, https:, file:, urn:, or any "scheme:" per
//     RFC 3986, including a Windows drive letter) is refused — the resolver
//     never interprets a scheme, so it cannot be steered into a fetch;
//   - a path escaping the root (an absolute path, or one with ".." segments
//     that leave the root after cleaning) is refused via an fs.ValidPath
//     containment check;
//   - a leftover fragment ("#...") is refused.
//
// Every rejection wraps [ErrReferenceNotFound], so a caller matches them all
// with errors.Is. Reads are bounded by the decrypt's own remaining CipherValue
// allowance ([Decryptor.MaxCipherValueBytes] for a payload reference,
// [Decryptor.MaxEncryptedKeyBytes] for a key reference): the resolver reads one
// byte past what is still allowed and the budget then refuses it, so an
// oversized resource is never buffered in full.
//
// Pass [helium.PermissiveFS] or an os.Root FS to widen what a caller is willing
// to serve.
func FSReferenceResolver(fsys fs.FS) ReferenceResolver {
	return fsReferenceResolver{fsys: fsys}
}

// fsReferenceResolver resolves external CipherReference URIs as
// slash-separated paths inside a fs.FS.
type fsReferenceResolver struct {
	fsys fs.FS
}

// ResolveReference implements ReferenceResolver. It reads without a limit,
// which is only reached when the decrypt's budget is itself unlimited or when a
// caller calls the resolver directly; every in-package call goes through
// resolveReferenceWithLimit with the budget's remaining allowance.
func (r fsReferenceResolver) ResolveReference(ctx context.Context, uri string) ([]byte, error) {
	return r.resolveReferenceWithLimit(ctx, uri, -1)
}

func (r fsReferenceResolver) resolveReferenceWithLimit(ctx context.Context, uri string, maxBytes int) ([]byte, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	name, err := fsNameFromURI(uri)
	if err != nil {
		return nil, err
	}
	if r.fsys == nil {
		return nil, fmt.Errorf("%w: FSReferenceResolver has no filesystem to resolve %q against", ErrReferenceNotFound, uri)
	}
	f, err := r.fsys.Open(name)
	if err != nil {
		return nil, fmt.Errorf("%w: cannot open external reference %q: %v", ErrReferenceNotFound, uri, err)
	}
	defer f.Close()

	// Read at most maxBytes+1, so an over-budget resource is recognized by the
	// extra byte alone rather than by buffering the whole of it. The caller
	// charges the returned length against the same budget, which is what turns
	// that extra byte into the budget's own over-limit error.
	var src io.Reader = f
	if maxBytes >= 0 {
		// int64(maxBytes)+1 wraps when maxBytes is math.MaxInt on a 64-bit
		// platform, so only add the probe byte when there is room for it. At
		// that allowance no resource can exceed the budget anyway.
		limit := int64(maxBytes)
		if limit < math.MaxInt64 {
			limit++
		}
		src = io.LimitReader(f, limit)
	}
	data, err := io.ReadAll(src)
	if err != nil {
		return nil, fmt.Errorf("%w: reading external reference %q: %v", ErrReferenceNotFound, uri, err)
	}
	return data, nil
}

// fsNameFromURI converts an external CipherReference URI into a validated fs.FS
// path, fail-closed. It refuses a scheme URI, a leftover fragment, and any path
// that does not stay inside the root.
func fsNameFromURI(uri string) (string, error) {
	if strings.IndexByte(uri, '#') >= 0 {
		return "", fmt.Errorf("%w: reference URI %q carries a fragment", ErrReferenceNotFound, uri)
	}
	if uriHasScheme(uri) {
		return "", fmt.Errorf("%w: FSReferenceResolver refuses scheme URI %q", ErrReferenceNotFound, uri)
	}
	// path.Clean collapses "." and ".." segments; fs.ValidPath then rejects an
	// absolute path or one that still escapes the root ("..", leading "/"),
	// which is this repository's established containment idiom.
	name := path.Clean(uri)
	if !fs.ValidPath(name) {
		return "", fmt.Errorf("%w: reference URI %q escapes the resolver root", ErrReferenceNotFound, uri)
	}
	return name, nil
}

// uriHasScheme reports whether uri carries an RFC 3986 scheme, i.e. a ":"
// appears before the first "/", "?", or "#". This catches every "scheme:" form
// — http://, https://, file:///, urn:..., a single-letter scheme, and a Windows
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

// joinReferenceURI joins an external CipherReference URI against the document's
// base URI. With no base the URI is passed through unchanged. The join reuses
// helium.ResolveURI — the root package's byte-faithful libxml2 xmlBuildURI
// port, in conventional (base, reference) order — so a relative URI resolves
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
