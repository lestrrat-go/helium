package xmlenc1

import (
	"context"
	"errors"
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
// A resolver is consulted ONLY for a non-same-document (external) URI. Same-document
// references need no I/O and never reach it. When no resolver is configured an
// external reference stays fail-closed with [ErrReferenceNotFound], which is
// the default.
//
// That default is what the specification permits, and no shortfall of it.
// W3C xmlenc-core1 §3.3.1 requires "the same URI encoding, dereferencing,
// scheme, and HTTP response codes as that of [XMLDSIG-CORE1]" and defines no
// dereferencing of its own; xmldsig-core1 §4.4.3.1 makes dereferencing URIs in
// the HTTP scheme RECOMMENDED, while §4.4.3.2 and §4.4.3.3 make the null URI,
// the shortname XPointer, and same-document dereferencing normative MUSTs. So
// the imported obligation covers URI="" and URI="#id", which this package
// implements unconditionally, and external fetching is a RECOMMENDED capability
// the caller opts into.
//
// # The address space
//
// uri is ALWAYS the JOINED, DOCUMENT-SPACE URI: the @URI the document wrote,
// resolved against the base URI in force at the xenc:CipherReference element
// itself (the document's own URL, narrowed by every xml:base on the way down to
// that element). A resolver therefore never sees the raw attribute and never
// performs the join. A document parsed with [helium.Parser.ParseFile] carries
// the file's absolute path as its URL, so a relative @URI arrives here as an
// absolute path in that document's space — which is why a filesystem resolver
// has to be told which prefix of that space its fs.FS stands for. See
// [FSReferenceResolver].
//
// # Who bounds and who cancels
//
// The returned stream is the resource's raw bytes: an external reference yields
// an octet stream, so no canonicalization applies to it and only a declared
// ds:Transform changes what the bytes mean.
//
// This package reads that stream itself, under the decrypt's remaining
// CipherValue allowance ([Decryptor.MaxCipherValueBytes] for a payload
// reference, [Decryptor.MaxEncryptedKeyBytes] for a key reference) and under
// the caller's context. It reads one byte past what the allowance still admits
// and no further, so it never materializes an oversized resource; it abandons
// the read the moment ctx is done, and it CLOSES the stream on every path —
// completion, over-budget, and cancellation alike, and equally when
// ResolveReference itself returns a stream alongside an error. A resolver
// hands over a stream and owes it nothing more.
//
// Be precise about what that does and does not buy. It removes THIS PACKAGE's
// complicity: whatever a URI names, the package holds no more than the budget
// admits and its own reading stops when the caller says stop. It does NOT stop
// a resolver from buffering a whole resource internally, or from blocking
// inside ResolveReference before it returns a stream at all — nothing outside a
// resolver can constrain what happens inside it. That division is the right one
// because the two sides are trusted differently: the resolver is code the
// caller chose, while the URI is chosen by an unauthenticated document, and it
// is the URI the package must be immune to.
//
// ResolveReference must be safe to call from multiple goroutines. Returning a
// nil stream with a nil error is refused as [ErrReferenceNotFound], and never
// dereferenced.
type ReferenceResolver interface {
	ResolveReference(ctx context.Context, uri string) (io.ReadCloser, error)
}

// FSReferenceResolver returns a [ReferenceResolver] that serves external
// CipherReference URIs from fsys. It performs NO network access.
//
// root declares the document-space prefix fsys stands for, which is what lets
// the resolver map the joined URI it is handed ([ReferenceResolver] states that
// the URI is always the joined one) onto a path fsys knows. A URI lying under
// root is served as the remainder of the path below it; every other URI is
// taken as a plain relative path in fsys. An empty root serves relative URIs
// only, which is what a document parsed from memory produces.
//
// So for a document read with [helium.Parser.ParseFile] from
// /srv/docs/index.xml, whose sibling cipher text the document names as
// URI="ct.bin", the joined URI is /srv/docs/ct.bin, and
// FSReferenceResolver(os.DirFS("/srv/docs"), "/srv/docs") serves it as "ct.bin".
//
// Whatever root admits, the resolved path is still fail-closed on anything that
// is not a plain in-tree path:
//
//   - a URI carrying a scheme (http:, https:, file:, urn:, or any "scheme:" per
//     RFC 3986, including a Windows drive letter) is refused — the resolver
//     never interprets a scheme, so it cannot be steered into a fetch;
//   - a path escaping the root (an absolute path outside the declared root, or
//     one with ".." segments that leave the root after cleaning) is refused via
//     an fs.ValidPath containment check;
//   - a leftover fragment ("#...") is refused.
//
// Every rejection wraps [ErrReferenceNotFound], so a caller matches them all
// with errors.Is. This resolver opens the file and hands the stream over; the
// decrypt's own allowance bounds how much of it is read, and the package closes
// it — [ReferenceResolver] owns that division.
//
// Pass [helium.PermissiveFS] or an os.Root FS to widen what a caller is willing
// to serve.
func FSReferenceResolver(fsys fs.FS, root string) ReferenceResolver {
	return fsReferenceResolver{fsys: fsys, root: root}
}

// fsReferenceResolver resolves external CipherReference URIs as
// slash-separated paths inside a fs.FS, below a declared document-space root.
type fsReferenceResolver struct {
	fsys fs.FS
	root string
}

// ResolveReference implements ReferenceResolver. It opens the named file and
// returns it; reading is the package's job, so nothing here is bounded by a
// budget this resolver cannot see.
func (r fsReferenceResolver) ResolveReference(ctx context.Context, uri string) (io.ReadCloser, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	name, err := fsNameFromURI(relativizeToRoot(r.root, uri))
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
	return f, nil
}

// relativizeToRoot turns a document-space URI into the path an fs.FS rooted at
// root knows it by, by stripping that prefix.
//
// A URI that does not lie under root is returned UNCHANGED, and never refused
// here, so it goes on to face exactly the checks an empty root applies: a
// relative path is served, and an absolute path or a scheme URI outside the
// declared root is refused by fsNameFromURI. Declaring a root therefore only
// ever widens what resolves, and only within the space the caller named.
//
// The prefix is compared with a trailing slash so it can only match at a
// segment boundary: a root of "/srv/docs" does not admit "/srv/docs-public/x".
// A URI equal to the root itself names a directory, and no resource, so
// it strips to nothing and is left to be refused.
func relativizeToRoot(root, uri string) string {
	if root == "" {
		return uri
	}
	if !strings.HasSuffix(root, "/") {
		root += "/"
	}
	rest, ok := strings.CutPrefix(uri, root)
	if !ok || rest == "" {
		return uri
	}
	return rest
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

// referenceRead is one completed read of a resolver's stream, carried back from
// the goroutine that performed it.
type referenceRead struct {
	octets []byte
	err    error
}

// readBoundedReference drains a resolver's stream into memory under the
// decrypt's remaining allowance and the caller's context, and closes rc on
// every path out.
//
// maxBytes is that remaining allowance, and a negative value means unlimited.
// At most maxBytes+1 bytes are read, so an over-budget resource is recognized
// by the one extra byte instead of by materializing the whole of it; the caller
// charges the returned length against the same budget, which is what turns that
// extra byte into the budget's own over-limit error. There is no separate size
// sentinel to keep in step with the budgets.
//
// The read runs on its own goroutine because a Read that never returns cannot
// be polled out of: this function leaves as soon as ctx is done, and CLOSING rc
// is what releases the blocked read so that goroutine finishes, and never
// strands. The channel is buffered, so the goroutine's send never blocks even
// when nothing is waiting for it any more. Reading and closing therefore
// overlap on the cancellation path, which is the same contract every stream
// handed across a package boundary is closed under (an os.File, an HTTP
// response body).
//
// The goroutine polls ctx once per chunk as well, so a stream that keeps
// yielding bytes stops being read too, and not merely stops being waited
// for.
func readBoundedReference(ctx context.Context, rc io.ReadCloser, uri string, maxBytes int) ([]byte, error) {
	if err := contextErr(ctx); err != nil {
		rc.Close()
		return nil, err
	}
	done := make(chan referenceRead, 1)
	go readReferenceOctets(ctx, rc, maxBytes, done)
	select {
	case <-contextDone(ctx):
		rc.Close()
		return nil, ctx.Err()
	case read := <-done:
		rc.Close()
		if read.err != nil {
			return nil, fmt.Errorf("%w: reading external reference %q: %v", ErrReferenceNotFound, uri, read.err)
		}
		return read.octets, nil
	}
}

// referenceReadChunk is how much of a resolver's stream one Read asks for. It
// is the size io.Copy and bufio use, and it is also the granularity of the
// context poll below: a resource yielding bytes steadily is answered once per
// chunk, and never once at its end.
const referenceReadChunk = 32 * 1024

// readReferenceOctets performs the bounded read readBoundedReference waits on,
// and reports the result exactly once.
//
// The copy is written out, and never handed to io.ReadAll, so the context poll
// sits inside the loop: a stream that keeps yielding bytes — an endless one
// above all — is abandoned at the caller's cancellation instead of at an end it
// may never reach.
func readReferenceOctets(ctx context.Context, r io.Reader, maxBytes int, done chan<- referenceRead) {
	src := r
	size := referenceReadChunk
	if maxBytes >= 0 {
		// int64(maxBytes)+1 wraps when maxBytes is math.MaxInt on a 64-bit
		// platform, so only add the probe byte when there is room for it. At
		// that allowance no resource can exceed the budget anyway.
		limit := int64(maxBytes)
		if limit < math.MaxInt64 {
			limit++
		}
		src = io.LimitReader(r, limit)
		if maxBytes < referenceReadChunk {
			size = maxBytes + 1
		}
	}

	var octets []byte
	buf := make([]byte, size)
	for {
		if err := contextErr(ctx); err != nil {
			done <- referenceRead{err: err}
			return
		}
		n, err := src.Read(buf)
		octets = append(octets, buf[:n]...)
		if errors.Is(err, io.EOF) {
			done <- referenceRead{octets: octets}
			return
		}
		if err != nil {
			done <- referenceRead{err: err}
			return
		}
	}
}

// contextDone returns the channel a select waits on for ctx's cancellation. A
// nil context yields a nil channel, which blocks forever — the same "never
// cancelled" answer contextErr gives one.
func contextDone(ctx context.Context) <-chan struct{} {
	if ctx == nil {
		return nil
	}
	return ctx.Done()
}

// joinReferenceURI joins an external CipherReference URI against the base URI
// in force at the element carrying it. With no base the URI is passed through
// unchanged. The join reuses helium.ResolveURI — the root package's
// byte-faithful libxml2 xmlBuildURI port, in conventional (base, reference)
// order — so a relative URI resolves exactly as the parser resolves any other
// relative reference.
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
