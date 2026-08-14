package xmlenc1_test

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"runtime"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmlenc1"
	"github.com/stretchr/testify/require"
)

// base64Transform is the one ds:Transform algorithm a CipherReference may
// declare here. Every other algorithm is refused, which xmlenc-core1 §3.3.1
// permits: the Transform feature and the particular algorithms are OPTIONAL.
const base64Transform = xmlenc1.NamespaceDSig + "base64"

// cipherRefPlaintext is the element every CipherReference document below
// encrypts, so a successful decrypt is recognizable by name alone.
const cipherRefPlaintext = `<secret>hidden</secret>`

const (
	// externalCipherPath is the in-tree path the fs.FS resolvers below serve,
	// and externalCipherURI is the @URI naming it. It carries no fragment and
	// no scheme, so it is the one shape FSReferenceResolver accepts.
	externalCipherPath = "ct.bin"
	externalCipherURI  = ` URI="` + externalCipherPath + `"`
)

// cipherRefDoc builds a document whose single EncryptedData carries cipherData
// as the content of its xenc:CipherData and is followed by trailing as a
// sibling, and returns that EncryptedData element.
//
// The xenc and ds prefixes are declared on the EncryptedData, and not on the
// document element, so a trailing element the reference names inherits neither
// and its canonical form is exactly the markup written here.
func cipherRefDoc(t *testing.T, cipherData, trailing string) *helium.Element {
	t.Helper()
	doc := mustParseXML(t, `<root>`+
		`<xenc:EncryptedData xmlns:xenc="`+xmlenc1.NamespaceXMLEnc+`" xmlns:ds="`+xmlenc1.NamespaceDSig+`" Type="`+xmlenc1.TypeElement+`">`+
		`<xenc:EncryptionMethod Algorithm="`+xmlenc1.AES256GCM+`"/>`+
		`<xenc:CipherData>`+cipherData+`</xenc:CipherData>`+
		`</xenc:EncryptedData>`+
		trailing+
		`</root>`)
	elem := findEncryptedData(t, doc.DocumentElement())
	require.NotNil(t, elem)
	return elem
}

// cipherReferenceXML renders an xenc:CipherReference carrying uriAttr verbatim
// (so a case can omit @URI altogether) and inner as its content.
func cipherReferenceXML(uriAttr, inner string) string {
	return `<xenc:CipherReference` + uriAttr + `>` + inner + `</xenc:CipherReference>`
}

// transformsXML renders one xenc:Transforms wrapper holding a ds:Transform per
// algorithm. The wrapper is in the xenc namespace and its children are in the
// ds namespace, which is what the xenc schema declares.
func transformsXML(algorithms ...string) string {
	var inner strings.Builder
	for _, algorithm := range algorithms {
		inner.WriteString(`<ds:Transform Algorithm="` + algorithm + `"/>`)
	}
	return `<xenc:Transforms>` + inner.String() + `</xenc:Transforms>`
}

// cipherRefCiphertext is the AES-256-GCM ciphertext of cipherRefPlaintext under
// key, i.e. the octets a CipherReference has to yield for a decrypt to succeed.
func cipherRefCiphertext(t *testing.T, key []byte) []byte {
	t.Helper()
	cipher, err := xmlenc1.EncryptBytesForTest(xmlenc1.AES256GCM, key, []byte(cipherRefPlaintext))
	require.NoError(t, err)
	return cipher
}

// canonicalOctetLength recovers the exact number of octets the same-document
// reference uri yields against a document carrying trailing, by bisecting
// MaxCipherValueBytes: the budget refuses the resolution one byte short of its
// length and admits it at exactly that length. It measures the resolution
// without reading it, which is what makes a claim about WHICH nodes a form
// selects — comments among them — falsifiable from the outside.
func canonicalOctetLength(t *testing.T, uri, trailing string) int {
	t.Helper()
	const ceiling = 1 << 16
	require.True(t, cipherRefFitsBudget(t, uri, trailing, ceiling), "the resolution is longer than the bisection ceiling")
	lo, hi := 0, ceiling
	for lo < hi {
		mid := lo + (hi-lo)/2
		if cipherRefFitsBudget(t, uri, trailing, mid) {
			hi = mid
			continue
		}
		lo = mid + 1
	}
	return lo
}

// cipherRefFitsBudget reports whether a same-document resolution stays within a
// MaxCipherValueBytes of maxBytes. The decrypt itself always fails — the
// resolved octets are canonical XML, and no ciphertext at all — so the budget
// sentinel, not success, is what separates the two outcomes.
func cipherRefFitsBudget(t *testing.T, uri, trailing string, maxBytes int) bool {
	t.Helper()
	elem := cipherRefDoc(t, cipherReferenceXML(` URI="`+uri+`"`, ""), trailing)
	_, err := xmlenc1.NewDecryptor().
		SessionKey(randKey(t, 32)).
		MaxCipherValueBytes(maxBytes).
		Decrypt(t.Context(), elem)
	require.Error(t, err)
	return !errors.Is(err, xmlenc1.ErrCipherValueBytesExceeded)
}

// wideDoc builds a document whose CipherReference names a subtree of elemCount
// elements standing under a root that declares nsCount prefixes, and returns
// that EncryptedData with the source's own size.
//
// It is the shape that makes a node-set expensive: every one of those
// declarations is in scope on every one of those elements, so a resolution that
// materialized the in-scope namespace AXIS per element would hold
// elemCount × nsCount namespace nodes for a document of a few hundred KB.
func wideDoc(t *testing.T, elemCount, nsCount int) (*helium.Element, int) {
	t.Helper()
	var decls strings.Builder
	for i := range nsCount {
		fmt.Fprintf(&decls, ` xmlns:n%d="urn:n%d"`, i, i)
	}
	var body strings.Builder
	for range elemCount {
		body.WriteString(`<e/>`)
	}
	src := `<root` + decls.String() + `>` +
		`<xenc:EncryptedData xmlns:xenc="` + xmlenc1.NamespaceXMLEnc + `" xmlns:ds="` + xmlenc1.NamespaceDSig + `" Type="` + xmlenc1.TypeElement + `">` +
		`<xenc:EncryptionMethod Algorithm="` + xmlenc1.AES256GCM + `"/>` +
		`<xenc:CipherData>` + cipherReferenceXML(` URI="#t"`, "") + `</xenc:CipherData>` +
		`</xenc:EncryptedData>` +
		`<ct Id="t">` + body.String() + `</ct>` +
		`</root>`
	doc := mustParseXML(t, src)
	elem := findEncryptedData(t, doc.DocumentElement())
	require.NotNil(t, elem)
	return elem, len(src)
}

// wideWholeDoc is wideDoc's whole-document counterpart: it builds a document
// of the same shape, but the CipherReference's URI (uriAttr, verbatim as in
// cipherReferenceXML) names the document, so the canonicalization the
// EncryptedData carries has no node-set collector ahead of it — the whole-doc
// gate in canonicalizeCipherReferenceNodeSet skips straight to writing the
// canonical form. That makes canonicalizeCipherReferenceNodeSet's own
// context-aware write loop the ONLY code that can ever observe a caller's
// cancellation for this shape, which is what makes it the right fixture for
// proving that loop, and not some earlier stage, is what stops a decrypt.
func wideWholeDoc(t *testing.T, uriAttr string, elemCount, nsCount int) (*helium.Element, int) {
	t.Helper()
	var decls strings.Builder
	for i := range nsCount {
		fmt.Fprintf(&decls, ` xmlns:n%d="urn:n%d"`, i, i)
	}
	var body strings.Builder
	for range elemCount {
		body.WriteString(`<e/>`)
	}
	src := `<root` + decls.String() + `>` +
		`<xenc:EncryptedData xmlns:xenc="` + xmlenc1.NamespaceXMLEnc + `" xmlns:ds="` + xmlenc1.NamespaceDSig + `" Type="` + xmlenc1.TypeElement + `">` +
		`<xenc:EncryptionMethod Algorithm="` + xmlenc1.AES256GCM + `"/>` +
		`<xenc:CipherData>` + cipherReferenceXML(uriAttr, "") + `</xenc:CipherData>` +
		`</xenc:EncryptedData>` +
		body.String() +
		`</root>`
	doc := mustParseXML(t, src)
	elem := findEncryptedData(t, doc.DocumentElement())
	require.NotNil(t, elem)
	return elem, len(src)
}

// countingResolver records how many times a decrypt asked it for a resource
// and always refuses, so a test can assert that a document never reached the
// dereferencing stage at all.
type countingResolver struct {
	calls int
}

func (r *countingResolver) ResolveReference(_ context.Context, uri string) (io.ReadCloser, error) {
	r.calls++
	return nil, fmt.Errorf("%w: countingResolver serves nothing (%q)", xmlenc1.ErrReferenceNotFound, uri)
}

// streamResolver is a caller-written resolver: it satisfies the public
// interface and nothing more, which is what makes it the right subject for a
// claim about what the PACKAGE bounds. It hands out an endless stream and
// records that the stream was closed.
type streamResolver struct {
	closed chan struct{}
	once   sync.Once
}

func newStreamResolver() *streamResolver {
	return &streamResolver{closed: make(chan struct{})}
}

func (r *streamResolver) ResolveReference(context.Context, string) (io.ReadCloser, error) {
	return &endlessStream{resolver: r}, nil
}

// nilStreamResolver reports neither a stream nor an error, which is the one
// shape the interface cannot express and a resolver can still return.
type nilStreamResolver struct{}

func (nilStreamResolver) ResolveReference(context.Context, string) (io.ReadCloser, error) {
	//nolint:nilnil // the whole point of the case is the shape a linter forbids.
	return nil, nil
}

// errorStreamResolver returns a live stream ALONGSIDE its error, which is the
// one shape ReferenceResolver's godoc still owes a close: the resolver call
// itself failed, so nothing downstream ever reads the stream, and closing it
// is the only thing standing between that shape and a leak.
type errorStreamResolver struct {
	closed chan struct{}
	once   sync.Once
}

func newErrorStreamResolver() *errorStreamResolver {
	return &errorStreamResolver{closed: make(chan struct{})}
}

func (r *errorStreamResolver) ResolveReference(context.Context, string) (io.ReadCloser, error) {
	return &closeTrackingStream{resolver: r}, fmt.Errorf("%w: errorStreamResolver always fails", xmlenc1.ErrReferenceNotFound)
}

// closeTrackingStream is an empty stream that records its own Close against
// the errorStreamResolver that produced it.
type closeTrackingStream struct {
	resolver *errorStreamResolver
}

func (s *closeTrackingStream) Read([]byte) (int, error) {
	return 0, io.EOF
}

func (s *closeTrackingStream) Close() error {
	s.resolver.once.Do(func() { close(s.resolver.closed) })
	return nil
}

// endlessStream never reaches an end, so anything that finishes reading it did
// so because it stopped, not because the resource did.
type endlessStream struct {
	resolver *streamResolver
	read     int
}

func (s *endlessStream) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'x'
	}
	s.read += len(p)
	return len(p), nil
}

func (s *endlessStream) Close() error {
	s.resolver.once.Do(func() { close(s.resolver.closed) })
	return nil
}

// readResolved dereferences uri through resolver and returns the whole stream,
// closing it. A resolver hands back a stream, in place of octets, so a test
// asserting WHAT it served has to drain one.
func readResolved(t *testing.T, resolver xmlenc1.ReferenceResolver, uri string) string {
	t.Helper()
	rc, err := resolver.ResolveReference(t.Context(), uri)
	require.NoError(t, err)
	defer rc.Close()
	octets, err := io.ReadAll(rc)
	require.NoError(t, err)
	return string(octets)
}

// blockingFS serves one file whose Read blocks until the file is closed or the
// test releases it, which is the shape a hostile or wedged resource has: the
// resolver hands back a stream that never produces a byte.
//
// Close releases the blocked Read, so a package that closes the stream on
// cancellation is what lets the read goroutine finish, and never leak.
type blockingFS struct {
	released chan struct{}
	once     sync.Once
}

func newBlockingFS() *blockingFS {
	return &blockingFS{released: make(chan struct{})}
}

func (fsys *blockingFS) release() {
	fsys.once.Do(func() { close(fsys.released) })
}

func (fsys *blockingFS) Open(string) (fs.File, error) {
	return &blockingFile{fsys: fsys}, nil
}

type blockingFile struct {
	fsys *blockingFS
}

func (f *blockingFile) Stat() (fs.FileInfo, error) {
	return nil, fmt.Errorf("blockingFile has no stat")
}

func (f *blockingFile) Read([]byte) (int, error) {
	<-f.fsys.released
	return 0, io.EOF
}

func (f *blockingFile) Close() error {
	f.fsys.release()
	return nil
}

func TestCipherReference(t *testing.T) {
	// A pre-shared session key is used throughout: a CipherReference decides
	// the CIPHERTEXT, and how the session key is protected is a separate axis
	// the RetrievalMethod suite already covers. Resolution runs while the
	// document is read, so it precedes that key's early return.
	newSessionKey := func(t *testing.T) []byte {
		t.Helper()
		return randKey(t, 32)
	}

	t.Run("a same-document reference with a base64 transform decrypts", func(t *testing.T) {
		for _, uri := range []string{"#t", "#xpointer(id('t'))"} {
			t.Run(uri, func(t *testing.T) {
				sessionKey := newSessionKey(t)
				encoded := base64.StdEncoding.EncodeToString(cipherRefCiphertext(t, sessionKey))
				elem := cipherRefDoc(t,
					cipherReferenceXML(` URI="`+uri+`"`, transformsXML(base64Transform)),
					`<ct Id="t">`+encoded+`</ct>`)
				nodes, err := xmlenc1.NewDecryptor().SessionKey(sessionKey).Decrypt(t.Context(), elem)
				require.NoError(t, err)
				requireSecret(t, nodes)
			})
		}
	})

	// A #base64 transform consumes the NODE-SET, and xmldsig-core1 §6.6.2 makes
	// its input the string-value of that set: it "strips away the start and end
	// tags of the identified element and any of its descendant elements", so the
	// markup between the characters is not part of the value and base64 written
	// anywhere under the target decodes.
	t.Run("a base64 transform takes the target's whole string-value", func(t *testing.T) {
		for _, tc := range []struct {
			name    string
			trailer func(encoded string) string
		}{
			{
				name:    "in a grandchild",
				trailer: func(encoded string) string { return `<ct Id="t"><wrap><part>` + encoded + `</part></wrap></ct>` },
			},
			{
				name: "split across an element boundary",
				trailer: func(encoded string) string {
					// The split falls inside a base64 quantum, so a walk that
					// read each child on its own would decode neither half.
					cut := len(encoded) / 2
					return `<ct Id="t">` + encoded[:cut] + `<part>` + encoded[cut:] + `</part></ct>`
				},
			},
			{
				name:    "through a CDATA section",
				trailer: func(encoded string) string { return `<ct Id="t"><part><![CDATA[` + encoded + `]]></part></ct>` },
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				sessionKey := newSessionKey(t)
				encoded := base64.StdEncoding.EncodeToString(cipherRefCiphertext(t, sessionKey))
				elem := cipherRefDoc(t,
					cipherReferenceXML(` URI="#t"`, transformsXML(base64Transform)),
					tc.trailer(encoded))
				nodes, err := xmlenc1.NewDecryptor().SessionKey(sessionKey).Decrypt(t.Context(), elem)
				require.NoError(t, err)
				requireSecret(t, nodes)
			})
		}

		// The whole-document forms name a node-set too, so a #base64 transform
		// takes the string-value of the WHOLE document. Every text node in the
		// document below is the encoded value, so that string-value is it.
		t.Run("for the whole-document forms", func(t *testing.T) {
			for _, uri := range []string{"", "#xpointer(/)"} {
				t.Run("URI="+uri, func(t *testing.T) {
					sessionKey := newSessionKey(t)
					encoded := base64.StdEncoding.EncodeToString(cipherRefCiphertext(t, sessionKey))
					elem := cipherRefDoc(t,
						cipherReferenceXML(` URI="`+uri+`"`, transformsXML(base64Transform)),
						`<ct Id="t">`+encoded+`</ct>`)
					nodes, err := xmlenc1.NewDecryptor().SessionKey(sessionKey).Decrypt(t.Context(), elem)
					require.NoError(t, err)
					requireSecret(t, nodes)
				})
			}
		})
	})

	// The transform list is a pipeline: the first transform consumes the
	// node-set and every one after it consumes the octets the one before
	// produced, so two #base64 transforms decode a doubly-encoded value.
	t.Run("a second base64 transform decodes the first one's octets", func(t *testing.T) {
		sessionKey := newSessionKey(t)
		cipher := cipherRefCiphertext(t, sessionKey)
		doubled := base64.StdEncoding.EncodeToString([]byte(base64.StdEncoding.EncodeToString(cipher)))
		elem := cipherRefDoc(t,
			cipherReferenceXML(` URI="#t"`, transformsXML(base64Transform, base64Transform)),
			`<ct Id="t">`+doubled+`</ct>`)
		nodes, err := xmlenc1.NewDecryptor().SessionKey(sessionKey).Decrypt(t.Context(), elem)
		require.NoError(t, err)
		requireSecret(t, nodes)
	})

	// With no transform the reference's value is a node-set, which XMLDSig core
	// §4.4.3.3 converts to octets by canonicalization. The target below inherits
	// no namespace declaration, so its canonical form is the markup verbatim.
	t.Run("no transform canonicalizes the named subtree", func(t *testing.T) {
		elem := cipherRefDoc(t,
			cipherReferenceXML(` URI="#t"`, ""),
			`<ct Id="t"><inner a="1">text</inner></ct>`)
		ed, err := xmlenc1.ParseEncryptedDataForTest(elem)
		require.NoError(t, err)
		require.Equal(t, `<ct Id="t"><inner a="1">text</inner></ct>`, string(ed.CipherValue))
	})

	// The two whole-document forms name the root, and no id, so their
	// value is the canonical form of the whole document — the EncryptedData
	// included, which is what makes them useless in practice and valid all the
	// same.
	t.Run("the whole-document forms resolve to the document", func(t *testing.T) {
		for _, uri := range []string{"", "#xpointer(/)"} {
			t.Run("URI="+uri, func(t *testing.T) {
				elem := cipherRefDoc(t, cipherReferenceXML(` URI="`+uri+`"`, ""), `<ct Id="t">x</ct>`)
				ed, err := xmlenc1.ParseEncryptedDataForTest(elem)
				require.NoError(t, err)
				require.True(t, strings.HasPrefix(string(ed.CipherValue), "<root>"))
				require.Contains(t, string(ed.CipherValue), `<ct Id="t">x</ct>`)
			})
		}
	})

	// A CipherReference carries no CanonicalizationMethod, so the only node-set
	// to octet conversion open to it is the default one — plain Canonical XML,
	// which omits comments — and all four forms are therefore comment-free, the
	// two XPointer ones included. The check is a measurement, and no mere
	// reading: the recovered octet count of each form is the same with a comment
	// in the target as without one, which a form that admitted comments could
	// not be.
	t.Run("every same-document form is comment-free", func(t *testing.T) {
		for _, tc := range []struct {
			uri  string
			want int
		}{
			{uri: "#t", want: 17},
			{uri: "#xpointer(id('t'))", want: 17},
			{uri: "", want: 402},
			{uri: "#xpointer(/)", want: 414},
		} {
			t.Run("URI="+tc.uri, func(t *testing.T) {
				plain := canonicalOctetLength(t, tc.uri, `<ct Id="t">x</ct>`)
				commented := canonicalOctetLength(t, tc.uri, `<ct Id="t">x<!-- a comment --></ct>`)
				require.Equal(t, tc.want, plain)
				require.Equal(t, plain, commented, "the comment reached the canonical octets")
			})
		}
	})

	// The xenc schema marks @URI required, so an ABSENT attribute is a
	// malformed document. A PRESENT and empty one is the valid null URI, which
	// the case above resolves; the two must not be conflated.
	t.Run("an absent URI is malformed", func(t *testing.T) {
		elem := cipherRefDoc(t, cipherReferenceXML("", ""), `<ct Id="t">x</ct>`)
		_, err := xmlenc1.NewDecryptor().SessionKey(newSessionKey(t)).Decrypt(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
		require.NotErrorIs(t, err, xmlenc1.ErrReferenceNotFound)
	})

	// Two elements answering to one id would let whoever injected the second
	// choose which octets the recipient decrypts, so the document is refused,
	// and resolved to neither.
	t.Run("a duplicate id is ambiguous", func(t *testing.T) {
		elem := cipherRefDoc(t,
			cipherReferenceXML(` URI="#t"`, transformsXML(base64Transform)),
			`<ct Id="t">AA==</ct><ct Id="t">AQ==</ct>`)
		_, err := xmlenc1.NewDecryptor().SessionKey(newSessionKey(t)).Decrypt(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrAmbiguousReference)
	})

	t.Run("a missing target is not found", func(t *testing.T) {
		elem := cipherRefDoc(t, cipherReferenceXML(` URI="#absent"`, ""), `<ct Id="t">x</ct>`)
		_, err := xmlenc1.NewDecryptor().SessionKey(newSessionKey(t)).Decrypt(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrReferenceNotFound)
	})

	// External dereferencing is RECOMMENDED, and never REQUIRED, by the
	// XMLDSig processing model xmlenc-core1 §3.3.1 imports, so it is off until
	// the caller supplies a resolver.
	t.Run("an external URI with no resolver is not found", func(t *testing.T) {
		for _, uri := range []string{"https://example.com/ct.bin", externalCipherPath, "#element(/1/2)"} {
			t.Run(uri, func(t *testing.T) {
				elem := cipherRefDoc(t, cipherReferenceXML(` URI="`+uri+`"`, ""), `<ct Id="t">x</ct>`)
				_, err := xmlenc1.NewDecryptor().SessionKey(newSessionKey(t)).Decrypt(t.Context(), elem)
				require.ErrorIs(t, err, xmlenc1.ErrReferenceNotFound)
			})
		}
	})

	// A resolver yields an octet stream, so no canonicalization applies and the
	// resource's bytes ARE the ciphertext.
	t.Run("an external URI resolves through FSReferenceResolver", func(t *testing.T) {
		sessionKey := newSessionKey(t)
		fsys := fstest.MapFS{externalCipherPath: &fstest.MapFile{Data: cipherRefCiphertext(t, sessionKey)}}
		elem := cipherRefDoc(t, cipherReferenceXML(externalCipherURI, ""), "")
		nodes, err := xmlenc1.NewDecryptor().
			SessionKey(sessionKey).
			CipherReferenceResolver(xmlenc1.FSReferenceResolver(fsys, "")).
			Decrypt(t.Context(), elem)
		require.NoError(t, err)
		requireSecret(t, nodes)
	})

	// A same-document reference never reaches the resolver, so configuring one
	// changes nothing about how those four forms resolve.
	t.Run("a resolver does not change same-document resolution", func(t *testing.T) {
		sessionKey := newSessionKey(t)
		encoded := base64.StdEncoding.EncodeToString(cipherRefCiphertext(t, sessionKey))
		elem := cipherRefDoc(t,
			cipherReferenceXML(` URI="#t"`, transformsXML(base64Transform)),
			`<ct Id="t">`+encoded+`</ct>`)
		nodes, err := xmlenc1.NewDecryptor().
			SessionKey(sessionKey).
			CipherReferenceResolver(xmlenc1.FSReferenceResolver(fstest.MapFS{}, "")).
			Decrypt(t.Context(), elem)
		require.NoError(t, err)
		requireSecret(t, nodes)
	})

	t.Run("the resolver refuses", func(t *testing.T) {
		fsys := fstest.MapFS{externalCipherPath: &fstest.MapFile{Data: []byte("payload")}}
		resolver := xmlenc1.FSReferenceResolver(fsys, "")
		for _, tc := range []struct {
			name string
			uri  string
		}{
			{name: "an http scheme", uri: "https://example.com/ct.bin"},
			{name: "a file scheme", uri: "file:///etc/passwd"},
			{name: "a Windows drive letter", uri: `C:\ct.bin`},
			{name: "a parent escape", uri: "../ct.bin"},
			{name: "a nested parent escape", uri: "sub/../../ct.bin"},
			{name: "an absolute path with no root declared", uri: "/ct.bin"},
			{name: "a fragment", uri: "ct.bin#frag"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				_, err := resolver.ResolveReference(t.Context(), tc.uri)
				require.ErrorIs(t, err, xmlenc1.ErrReferenceNotFound)
			})
		}

		// A plain in-tree path is what remains after those refusals.
		t.Run("but serves a plain path", func(t *testing.T) {
			require.Equal(t, "payload", readResolved(t, resolver, externalCipherPath))
		})
	})

	// An absolute URI is not refused for BEING absolute — it is refused for
	// lying outside the space the caller declared its fs.FS to stand for. A
	// document read from a file names its sibling cipher text through an
	// absolute path once the join is done, so refusing every absolute path
	// would refuse the feature's own motivating case.
	t.Run("a declared root decides which absolute paths resolve", func(t *testing.T) {
		fsys := fstest.MapFS{externalCipherPath: &fstest.MapFile{Data: []byte("payload")}}
		resolver := xmlenc1.FSReferenceResolver(fsys, "/srv/docs")

		t.Run("an in-root absolute path resolves", func(t *testing.T) {
			require.Equal(t, "payload", readResolved(t, resolver, "/srv/docs/"+externalCipherPath))
		})

		// The root is matched at a segment boundary, so a sibling directory
		// whose name merely starts with the root's is outside it.
		for _, tc := range []struct {
			name string
			uri  string
		}{
			{name: "above the root", uri: "/srv/" + externalCipherPath},
			{name: "in a sibling of the root", uri: "/srv/docs-public/" + externalCipherPath},
			{name: "elsewhere entirely", uri: "/etc/passwd"},
			{name: "escaping the root after stripping it", uri: "/srv/docs/../../etc/passwd"},
			{name: "the root itself", uri: "/srv/docs/"},
			{name: "a scheme URI outside the root", uri: "file:///srv/docs/" + externalCipherPath},
		} {
			t.Run(tc.name, func(t *testing.T) {
				_, err := resolver.ResolveReference(t.Context(), tc.uri)
				require.ErrorIs(t, err, xmlenc1.ErrReferenceNotFound)
			})
		}

		// Declaring a root only widens what resolves; a relative URI, which is
		// what a document parsed from memory produces, is served either way.
		t.Run("and still serves a relative path", func(t *testing.T) {
			require.Equal(t, "payload", readResolved(t, resolver, externalCipherPath))
		})
	})

	// Every declared algorithm is validated before any is executed, and only
	// #base64 is accepted. An XPath or XSLT transform evaluated over an
	// attacker-supplied document before anything is authenticated is unbounded
	// compute, and xmlenc-core1 §3.3.1 marks the whole Transform feature
	// OPTIONAL.
	t.Run("an unsupported transform is refused", func(t *testing.T) {
		for _, tc := range []struct {
			name      string
			algorithm string
		}{
			{name: "XPath", algorithm: "http://www.w3.org/TR/1999/REC-xpath-19991116"},
			{name: "XSLT", algorithm: "http://www.w3.org/TR/1999/REC-xslt-19991116"},
			{name: "C14N", algorithm: "http://www.w3.org/TR/2001/REC-xml-c14n-20010315"},
			{name: "an empty algorithm", algorithm: ""},
		} {
			t.Run(tc.name, func(t *testing.T) {
				elem := cipherRefDoc(t,
					cipherReferenceXML(` URI="#t"`, transformsXML(tc.algorithm)),
					`<ct Id="t">AA==</ct>`)
				_, err := xmlenc1.NewDecryptor().SessionKey(newSessionKey(t)).Decrypt(t.Context(), elem)
				require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
			})
		}

		// The refusal is decided over the WHOLE list, so a supported algorithm
		// standing ahead of an unsupported one does not get executed first.
		t.Run("behind a supported one", func(t *testing.T) {
			elem := cipherRefDoc(t,
				cipherReferenceXML(` URI="#t"`, transformsXML(base64Transform, "http://www.w3.org/TR/1999/REC-xpath-19991116")),
				`<ct Id="t">AA==</ct>`)
			_, err := xmlenc1.NewDecryptor().SessionKey(newSessionKey(t)).Decrypt(t.Context(), elem)
			require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
		})
	})

	// CipherReferenceType declares Transforms with maxOccurs 1, so a second
	// wrapper is schema-invalid and refused, and never silently merged.
	t.Run("two Transforms elements are refused", func(t *testing.T) {
		elem := cipherRefDoc(t,
			cipherReferenceXML(` URI="#t"`, transformsXML(base64Transform)+transformsXML(base64Transform)),
			`<ct Id="t">AA==</ct>`)
		_, err := xmlenc1.NewDecryptor().SessionKey(newSessionKey(t)).Decrypt(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
	})

	// Neither xenc:CipherReferenceType nor xenc:TransformsType carries a
	// wildcard, so an element the schema does not declare is refused outright,
	// and never stepped over. Skipping one would drop a transform declaration out of the
	// whitelist — and out of any policy or logging layer reading it — and
	// resolve the reference as though the document had asked for something else.
	t.Run("a foreign element in the transform list is refused", func(t *testing.T) {
		const evilNS = `xmlns:evil="urn:evil"`
		for _, tc := range []struct {
			name  string
			inner string
		}{
			{
				name:  "a foreign-namespace Transform child",
				inner: `<xenc:Transforms><ds:Transform Algorithm="` + base64Transform + `"/><evil:Transform ` + evilNS + ` Algorithm="urn:unsupported"/></xenc:Transforms>`,
			},
			{
				name:  "a foreign-namespace Transforms wrapper",
				inner: `<ds:Transforms><ds:Transform Algorithm="urn:unsupported"/></ds:Transforms>`,
			},
			{
				name:  "a foreign element beside the wrapper",
				inner: transformsXML(base64Transform) + `<evil:Extra ` + evilNS + `/>`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				elem := cipherRefDoc(t,
					cipherReferenceXML(` URI="#t"`, tc.inner),
					`<ct Id="t">AA==</ct>`)
				_, err := xmlenc1.NewDecryptor().SessionKey(newSessionKey(t)).Decrypt(t.Context(), elem)
				require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
			})
		}

		// Foreign children count against the cap as they are collected, so a
		// flood of them costs one refusal, well short of a walk of them all.
		t.Run("and counts against the transform cap", func(t *testing.T) {
			var inner strings.Builder
			for range 101 {
				inner.WriteString(`<evil:Transform xmlns:evil="urn:evil" Algorithm="urn:unsupported"/>`)
			}
			elem := cipherRefDoc(t,
				cipherReferenceXML(` URI="#t"`, `<xenc:Transforms>`+inner.String()+`</xenc:Transforms>`),
				`<ct Id="t">AA==</ct>`)
			_, err := xmlenc1.NewDecryptor().SessionKey(newSessionKey(t)).Decrypt(t.Context(), elem)
			require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
		})
	})

	t.Run("an over-cap transform list is refused", func(t *testing.T) {
		algorithms := make([]string, xmlenc1.MaxCipherReferenceTransformsForTest+1)
		for i := range algorithms {
			algorithms[i] = base64Transform
		}
		elem := cipherRefDoc(t,
			cipherReferenceXML(` URI="#t"`, transformsXML(algorithms...)),
			`<ct Id="t">AA==</ct>`)
		_, err := xmlenc1.NewDecryptor().SessionKey(newSessionKey(t)).Decrypt(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
	})

	// CipherData is a choice of exactly one member, and resolving the reference
	// does not change that: a document offering both is refused with the same
	// message it always was.
	t.Run("a CipherValue and a CipherReference together are refused", func(t *testing.T) {
		const want = "CipherData allows exactly one of CipherValue or CipherReference"
		for _, tc := range []struct {
			name       string
			cipherData string
		}{
			{
				name:       "value then reference",
				cipherData: `<xenc:CipherValue>AA==</xenc:CipherValue>` + cipherReferenceXML(` URI="#t"`, ""),
			},
			{
				name:       "reference then value",
				cipherData: cipherReferenceXML(` URI="#t"`, "") + `<xenc:CipherValue>AA==</xenc:CipherValue>`,
			},
			{
				name:       "two references",
				cipherData: cipherReferenceXML(` URI="#t"`, "") + cipherReferenceXML(` URI="#t"`, ""),
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				elem := cipherRefDoc(t, tc.cipherData, `<ct Id="t">AA==</ct>`)
				_, err := xmlenc1.NewDecryptor().SessionKey(newSessionKey(t)).Decrypt(t.Context(), elem)
				require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
				require.Contains(t, err.Error(), want)
			})
		}

		// The cardinality is settled before any member is read, so a
		// schema-invalid document never turns into a dereference — an
		// external URI standing first must not reach the caller's resolver.
		t.Run("without reaching the resolver", func(t *testing.T) {
			var resolver countingResolver
			elem := cipherRefDoc(t,
				cipherReferenceXML(externalCipherURI, "")+`<xenc:CipherValue>AA==</xenc:CipherValue>`,
				"")
			_, err := xmlenc1.NewDecryptor().
				SessionKey(newSessionKey(t)).
				CipherReferenceResolver(&resolver).
				Decrypt(t.Context(), elem)
			require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
			require.Contains(t, err.Error(), want)
			require.Zero(t, resolver.calls)
		})
	})

	// parseCipherData is shared by the EncryptedData payload and every
	// EncryptedKey, so an EncryptedKey may name its wrapped key by reference
	// too — and it is charged to the EncryptedKey budget, and never the
	// payload one.
	t.Run("a CipherReference supplies an EncryptedKey", func(t *testing.T) {
		sessionKey := newSessionKey(t)
		kek := randKey(t, 32)
		wrapped, err := xmlenc1.AESKeyWrapForTest(kek, sessionKey)
		require.NoError(t, err)
		keyInfo := `<ds:KeyInfo><xenc:EncryptedKey>` +
			`<xenc:EncryptionMethod Algorithm="` + xmlenc1.AES256KeyWrap + `"/>` +
			`<xenc:CipherData>` + cipherReferenceXML(` URI="#k"`, transformsXML(base64Transform)) + `</xenc:CipherData>` +
			`</xenc:EncryptedKey></ds:KeyInfo>`
		doc := mustParseXML(t, `<root>`+
			`<xenc:EncryptedData xmlns:xenc="`+xmlenc1.NamespaceXMLEnc+`" xmlns:ds="`+xmlenc1.NamespaceDSig+`" Type="`+xmlenc1.TypeElement+`">`+
			`<xenc:EncryptionMethod Algorithm="`+xmlenc1.AES256GCM+`"/>`+
			keyInfo+
			`<xenc:CipherData><xenc:CipherValue>`+base64.StdEncoding.EncodeToString(cipherRefCiphertext(t, sessionKey))+`</xenc:CipherValue></xenc:CipherData>`+
			`</xenc:EncryptedData>`+
			`<wk Id="k">`+base64.StdEncoding.EncodeToString(wrapped)+`</wk>`+
			`</root>`)
		elem := findEncryptedData(t, doc.DocumentElement())
		require.NotNil(t, elem)

		nodes, err := xmlenc1.NewDecryptor().KeyEncryptionKey(kek).Decrypt(t.Context(), elem)
		require.NoError(t, err)
		requireSecret(t, nodes)

		// The wrapped key is charged to MaxEncryptedKeyBytes, not to the
		// payload budget: one byte less is refused with the EncryptedKey
		// sentinel.
		_, err = xmlenc1.NewDecryptor().
			KeyEncryptionKey(kek).
			MaxEncryptedKeyBytes(len(wrapped)-1).
			Decrypt(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrEncryptedKeyBytesExceeded)
	})

	// A reference naming the EncryptedData that carries it is inert: the
	// resolved octets are canonicalized, never re-parsed as a document, so
	// there is nothing to recurse into and the decrypt terminates on the
	// ciphertext instead of hanging.
	t.Run("a self-referencing CipherReference terminates", func(t *testing.T) {
		for _, uri := range []string{"#self", ""} {
			t.Run("URI="+uri, func(t *testing.T) {
				sessionKey := newSessionKey(t)
				doc := mustParseXML(t, `<root>`+
					`<xenc:EncryptedData Id="self" xmlns:xenc="`+xmlenc1.NamespaceXMLEnc+`" xmlns:ds="`+xmlenc1.NamespaceDSig+`" Type="`+xmlenc1.TypeElement+`">`+
					`<xenc:EncryptionMethod Algorithm="`+xmlenc1.AES256GCM+`"/>`+
					`<xenc:CipherData>`+cipherReferenceXML(` URI="`+uri+`"`, "")+`</xenc:CipherData>`+
					`</xenc:EncryptedData></root>`)
				elem := findEncryptedData(t, doc.DocumentElement())
				require.NotNil(t, elem)
				_, err := xmlenc1.NewDecryptor().SessionKey(sessionKey).Decrypt(t.Context(), elem)
				require.ErrorIs(t, err, xmlenc1.ErrDecryptionFailed)
			})
		}
	})

	// parseEncryptedData uses a nil CipherValue as its missing-CipherData
	// sentinel, so a reference that legitimately yields zero octets must still
	// return a non-nil slice or an empty resource is reported as a malformed
	// document.
	t.Run("a zero-octet resolution is not missing CipherData", func(t *testing.T) {
		t.Run("through a base64 transform", func(t *testing.T) {
			elem := cipherRefDoc(t,
				cipherReferenceXML(` URI="#t"`, transformsXML(base64Transform)),
				`<ct Id="t"></ct>`)
			ed, err := xmlenc1.ParseEncryptedDataForTest(elem)
			require.NoError(t, err)
			require.NotNil(t, ed.CipherValue)
			require.Empty(t, ed.CipherValue)
		})

		t.Run("through a resolver", func(t *testing.T) {
			fsys := fstest.MapFS{"empty.bin": &fstest.MapFile{Data: []byte{}}}
			elem := cipherRefDoc(t, cipherReferenceXML(` URI="empty.bin"`, ""), "")
			_, err := xmlenc1.NewDecryptor().
				SessionKey(newSessionKey(t)).
				CipherReferenceResolver(xmlenc1.FSReferenceResolver(fsys, "")).
				Decrypt(t.Context(), elem)
			require.ErrorIs(t, err, xmlenc1.ErrDecryptionFailed)
			require.NotErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
		})
	})

	// The canonicalization writes through a budgetWriter, so an oversized
	// subtree is stopped at the limit, and never canonicalized in full and
	// rejected afterwards. This test must NOT run in parallel: TotalAlloc is
	// process-wide and a concurrent test would pollute the deltas.
	t.Run("an over-budget canonicalization stops at the limit", func(t *testing.T) {
		// no t.Parallel(): isolated so each delta reflects only its own Decrypt.
		big := strings.Repeat("a", 4<<20)
		cipherData := cipherReferenceXML(` URI="#t"`, "")
		trailing := `<ct Id="t">` + big + `</ct>`
		sessionKey := newSessionKey(t)

		spend := func(t *testing.T, maxBytes int) uint64 {
			t.Helper()
			elem := cipherRefDoc(t, cipherData, trailing)
			var before, after runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&before)
			_, err := xmlenc1.NewDecryptor().
				SessionKey(sessionKey).
				MaxCipherValueBytes(maxBytes).
				Decrypt(t.Context(), elem)
			runtime.ReadMemStats(&after)
			require.Error(t, err)
			if maxBytes >= 0 {
				require.ErrorIs(t, err, xmlenc1.ErrCipherValueBytesExceeded)
			}
			return after.TotalAlloc - before.TotalAlloc
		}

		// Both figures are weighed against the INPUT, and never against each
		// other. Comparing the two measurements would anchor the assertion to
		// the shared parse cost, which is most of the bounded figure and has
		// nothing to do with the property: any change adding to that cost would
		// flip the assertion while the property still held. Neither run can
		// avoid the one copy the DOM hands out per text node, which is this
		// package's documented floor for reading a value, so the bounded
		// multiple is what admits that floor and refuses a full canonical form.
		const boundedMultiple = 2
		unbounded := spend(t, -1)
		bounded := spend(t, 1024)
		t.Logf("canonicalizing a %d byte subtree allocated %d bytes unbounded and %d bytes under a 1024 byte budget", len(big), unbounded, bounded)
		require.Less(t, bounded, uint64(boundedMultiple*len(big)), "the canonicalization ran to completion before the budget refused it")
		require.Greater(t, unbounded, uint64(boundedMultiple*len(big)), "the unbounded run did not canonicalize the subtree it was given")
	})

	// The node-set a same-document reference names is bounded by the budget and
	// by the document, not by the product of the two. A document that declares
	// many prefixes high up and carries many elements below them is the shape
	// that separates the two: every declaration is in scope on every element, so
	// materializing the in-scope namespace axis per element would cost
	// elements × declarations nodes for a document of a few hundred KB — work no
	// output limit can see, since an inherited declaration is rendered once and
	// costs no output bytes at all.
	//
	// This test must NOT run in parallel: TotalAlloc is process-wide and a
	// concurrent test would pollute the deltas.
	t.Run("a wide node-set is bounded by the document", func(t *testing.T) {
		// no t.Parallel(): isolated so each delta reflects only its own Decrypt.
		const (
			elemCount = 50000
			nsCount   = 500
		)
		for _, tc := range []struct {
			name     string
			maxBytes int
			// spend bounds the allocation as a multiple of the source size.
			spend int
		}{
			// One byte of allowance cannot fit 50k elements, and knowing that
			// before the set is built is what stops the work: the refusal costs
			// a fraction of the document, and never a multiple of it.
			{name: "under a one byte budget", maxBytes: 1, spend: 8},
			// The DEFAULT budget admits this document, so nothing refuses it —
			// the resolution runs, and stays linear in the document.
			{name: "under the default budget", maxBytes: 0, spend: 200},
		} {
			t.Run(tc.name, func(t *testing.T) {
				elem, size := wideDoc(t, elemCount, nsCount)
				decryptor := xmlenc1.NewDecryptor().SessionKey(newSessionKey(t))
				if tc.maxBytes != 0 {
					decryptor = decryptor.MaxCipherValueBytes(tc.maxBytes)
				}
				var before, after runtime.MemStats
				runtime.GC()
				runtime.ReadMemStats(&before)
				_, err := decryptor.Decrypt(t.Context(), elem)
				runtime.ReadMemStats(&after)
				require.Error(t, err)
				if tc.maxBytes == 1 {
					require.ErrorIs(t, err, xmlenc1.ErrCipherValueBytesExceeded)
				}
				spent := after.TotalAlloc - before.TotalAlloc
				t.Logf("a %d byte document with %d elements under %d declarations allocated %d bytes", size, elemCount, nsCount, spent)
				require.Less(t, spent, uint64(tc.spend*size), "the resolution allocated a multiple of the document it was given")
			})
		}
	})

	// The same walk observes the context once per node, so a caller that
	// cancelled is answered while the walk is running, well before it has
	// stepped every node of a document the caller did not write.
	t.Run("a cancelled context stops the node-set walk", func(t *testing.T) {
		elem, _ := wideDoc(t, 50000, 500)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		_, err := xmlenc1.NewDecryptor().SessionKey(newSessionKey(t)).Decrypt(ctx, elem)
		require.ErrorIs(t, err, context.Canceled)

		// And a deadline is not out-waited: the same document under a deadline
		// of a tenth of a second returns in that order of time, and never in
		// tens of seconds, which is what a walk polling nothing would cost.
		deadline, stop := context.WithTimeout(t.Context(), 100*time.Millisecond)
		defer stop()
		start := time.Now()
		_, err = xmlenc1.NewDecryptor().SessionKey(newSessionKey(t)).Decrypt(deadline, elem)
		elapsed := time.Since(start)
		require.Error(t, err)
		t.Logf("a 100ms deadline over a 50000 element reference returned after %s", elapsed)
		require.Less(t, elapsed, 10*time.Second, "the resolution ran on after its caller's deadline passed")
	})

	// The two whole-document forms (URI="" and "#xpointer(/)") never build a
	// node-set: the gate in canonicalizeCipherReferenceNodeSet sends them
	// straight to c14n.Canonicalize, so the write loop feeding it is the ONLY
	// code in the whole resolution that is ever in a position to see a
	// caller's cancellation. wideWholeDoc's fixture keeps the structural
	// preamble (EncryptedData's own handful of children) fixed and tiny while
	// scaling namespace declarations on the root, which is what makes plain
	// document canonicalization — and only that, not the node-set path
	// wideDoc's named forms use — expensive: measured on this fixture, an
	// unbounded run takes over a second where every earlier stage this
	// decrypt passes through finishes in tens of milliseconds.
	//
	// This can't flake on timing: it never measures elapsed time. It races a
	// short, fixed deadline against a document sized so the write loop cannot
	// finish inside the watchdog if it ignores that deadline, and asserts only
	// on which of those two outcomes happened — the deadline's own error, or a
	// t.Fatal from a watchdog with wide margin on both sides (comfortably
	// above the deadline, comfortably below the fixture's measured unbounded
	// run), never a precise duration.
	for _, tc := range []struct {
		name string
		uri  string
	}{
		{name: `URI=""`, uri: ""},
		{name: `URI="#xpointer(/)"`, uri: "#xpointer(/)"},
	} {
		t.Run("a whole-document reference stops for "+tc.name, func(t *testing.T) {
			elem, _ := wideWholeDoc(t, ` URI="`+tc.uri+`"`, 10000, 1000)
			ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
			defer cancel()
			done := make(chan error, 1)
			go func() {
				_, err := xmlenc1.NewDecryptor().SessionKey(newSessionKey(t)).Decrypt(ctx, elem)
				done <- err
			}()
			select {
			case err := <-done:
				require.ErrorIs(t, err, context.DeadlineExceeded)
			case <-time.After(600 * time.Millisecond):
				t.Fatal("the whole-document canonicalization ran on after its caller's deadline passed")
			}
		})
	}

	// A named form ("#id") DOES build a node-set first, and that walk already
	// polls ctx. What it does not cover is a cancellation landing AFTER the
	// walk finishes and WHILE the resulting node-set is being canonicalized —
	// exactly the gap the whole-document tests above cannot exercise, since
	// they have no walk to finish first. Reaching that gap needs a deadline
	// that survives the walk and expires during canonicalization, which needs
	// the two phases to cost very differently: the walk here is linear in
	// element count, while c14n's OWN per-element namespace-visibility scan
	// (unrelated to xmlenc1's collector, which this package's other tests
	// document as linear) is what actually costs elements × in-scope
	// declarations, so a document with many elements AND many namespace
	// declarations makes canonicalization dominate the walk by a wide,
	// measured margin — on this fixture, well over an order of magnitude.
	//
	// Building this fixture is the expensive part of this test (parsing tens
	// of thousands of namespace declarations costs real time on its own), not
	// the decrypt this subtest races against a deadline. As above, nothing
	// here asserts on elapsed time; the watchdog only needs to sit inside the
	// measured gap between a cancelled decrypt's prompt return and an
	// uncancelled one's full run, and that gap is wide enough on this fixture
	// to leave comfortable margin on both sides.
	t.Run("a named reference stops during canonicalization, after its node-set is built", func(t *testing.T) {
		elem, _ := wideDoc(t, 60000, 60000)
		ctx, cancel := context.WithTimeout(t.Context(), 150*time.Millisecond)
		defer cancel()
		done := make(chan error, 1)
		go func() {
			_, err := xmlenc1.NewDecryptor().SessionKey(newSessionKey(t)).Decrypt(ctx, elem)
			done <- err
		}()
		select {
		case err := <-done:
			require.ErrorIs(t, err, context.DeadlineExceeded)
		case <-time.After(500 * time.Millisecond):
			t.Fatal("the canonicalization ran on after its caller's deadline passed, well past the node-set walk that already observes it")
		}
	})

	// The fix above changes nothing about what a non-cancelled decrypt
	// produces: the same same-document reference resolves to the identical
	// canonical octets whether or not the writer feeding c14n also polls a
	// context that never comes due. The two counts pinned here are
	// canonicalOctetLength's own bisection against this test's fixture, and
	// never change across a run: a budgetWriter that additionally checks a
	// live context still forwards every byte a within-limit write always did.
	t.Run("a live context does not change the canonical octets", func(t *testing.T) {
		require.Equal(t, 16, canonicalOctetLength(t, "#t", `<ct Id="t"></ct>`), "the named form's canonical octet count")
		require.Equal(t, 401, canonicalOctetLength(t, "", `<ct Id="t"></ct>`), "the whole-document form's canonical octet count")
	})

	// The bound is the INTERFACE's, not a privilege of the shipped resolver:
	// this resolver is written the way a caller writes one, its stream never
	// ends, and the decrypt still stops at the budget and closes the stream.
	// The package therefore holds no more than the allowance whatever the
	// resolver hands it.
	t.Run("a caller's endless stream is cut off at the budget", func(t *testing.T) {
		resolver := newStreamResolver()
		elem := cipherRefDoc(t, cipherReferenceXML(externalCipherURI, ""), "")
		done := make(chan error, 1)
		go func() {
			_, err := xmlenc1.NewDecryptor().
				SessionKey(newSessionKey(t)).
				MaxCipherValueBytes(1024).
				CipherReferenceResolver(resolver).
				Decrypt(t.Context(), elem)
			done <- err
		}()
		select {
		case err := <-done:
			require.ErrorIs(t, err, xmlenc1.ErrCipherValueBytesExceeded)
		case <-time.After(10 * time.Second):
			t.Fatal("the decrypt read the endless stream past its budget")
		}
		select {
		case <-resolver.closed:
		case <-time.After(10 * time.Second):
			t.Fatal("the decrypt left the resolver's stream open")
		}
	})

	// A resolver reporting neither a stream nor an error is a refusal rather
	// than a panic inside the decrypt of a document nobody authenticated.
	t.Run("a resolver returning no stream is not found", func(t *testing.T) {
		elem := cipherRefDoc(t, cipherReferenceXML(externalCipherURI, ""), "")
		_, err := xmlenc1.NewDecryptor().
			SessionKey(newSessionKey(t)).
			CipherReferenceResolver(nilStreamResolver{}).
			Decrypt(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrReferenceNotFound)
	})

	// A resolver reporting an error ALONGSIDE a live stream still handed over
	// a resource, and ReferenceResolver's godoc commits to closing a stream
	// "on every path" — this pins the one path that used to reach the caller
	// with the stream never closed, because the resolver's own error is
	// returned before readBoundedReference, and every close before this fix
	// lived only there.
	t.Run("a resolver returning a stream with its error still closes it", func(t *testing.T) {
		resolver := newErrorStreamResolver()
		elem := cipherRefDoc(t, cipherReferenceXML(externalCipherURI, ""), "")
		_, err := xmlenc1.NewDecryptor().
			SessionKey(newSessionKey(t)).
			CipherReferenceResolver(resolver).
			Decrypt(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrReferenceNotFound)
		select {
		case <-resolver.closed:
		default:
			t.Fatal("the decrypt left the resolver's stream open")
		}
	})

	// A resolver's own octets are bounded the same way: the shipped resolver
	// reads only as far as the budget still allows, plus the one byte that
	// proves the resource is over it.
	t.Run("an over-budget external resource fails", func(t *testing.T) {
		fsys := fstest.MapFS{externalCipherPath: &fstest.MapFile{Data: make([]byte, 4096)}}
		elem := cipherRefDoc(t, cipherReferenceXML(externalCipherURI, ""), "")
		_, err := xmlenc1.NewDecryptor().
			SessionKey(newSessionKey(t)).
			MaxCipherValueBytes(1024).
			CipherReferenceResolver(xmlenc1.FSReferenceResolver(fsys, "")).
			Decrypt(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrCipherValueBytesExceeded)
	})

	// math.MaxInt is the one budget value that must still let a valid
	// external resource resolve: reference_resolver.go builds the probe read
	// as int64(maxBytes)+1, and a maximum allowance must not look over
	// budget.
	t.Run("a maximum budget still resolves", func(t *testing.T) {
		sessionKey := newSessionKey(t)
		fsys := fstest.MapFS{externalCipherPath: &fstest.MapFile{Data: cipherRefCiphertext(t, sessionKey)}}
		elem := cipherRefDoc(t, cipherReferenceXML(externalCipherURI, ""), "")
		nodes, err := xmlenc1.NewDecryptor().
			SessionKey(sessionKey).
			MaxCipherValueBytes(math.MaxInt).
			CipherReferenceResolver(xmlenc1.FSReferenceResolver(fsys, "")).
			Decrypt(t.Context(), elem)
		require.NoError(t, err)
		requireSecret(t, nodes)
	})

	// The EncryptedKey budget reaches the same resolver code path for its own
	// external CipherReference, so the maximum allowance must resolve there
	// too.
	t.Run("a maximum EncryptedKey budget still resolves", func(t *testing.T) {
		sessionKey := newSessionKey(t)
		kek := randKey(t, 32)
		wrapped, err := xmlenc1.AESKeyWrapForTest(kek, sessionKey)
		require.NoError(t, err)
		keyInfo := `<ds:KeyInfo><xenc:EncryptedKey>` +
			`<xenc:EncryptionMethod Algorithm="` + xmlenc1.AES256KeyWrap + `"/>` +
			`<xenc:CipherData>` + cipherReferenceXML(externalCipherURI, "") + `</xenc:CipherData>` +
			`</xenc:EncryptedKey></ds:KeyInfo>`
		doc := mustParseXML(t, `<root>`+
			`<xenc:EncryptedData xmlns:xenc="`+xmlenc1.NamespaceXMLEnc+`" xmlns:ds="`+xmlenc1.NamespaceDSig+`" Type="`+xmlenc1.TypeElement+`">`+
			`<xenc:EncryptionMethod Algorithm="`+xmlenc1.AES256GCM+`"/>`+
			keyInfo+
			`<xenc:CipherData><xenc:CipherValue>`+base64.StdEncoding.EncodeToString(cipherRefCiphertext(t, sessionKey))+`</xenc:CipherValue></xenc:CipherData>`+
			`</xenc:EncryptedData>`+
			`</root>`)
		elem := findEncryptedData(t, doc.DocumentElement())
		require.NotNil(t, elem)

		fsys := fstest.MapFS{externalCipherPath: &fstest.MapFile{Data: wrapped}}
		nodes, err := xmlenc1.NewDecryptor().
			KeyEncryptionKey(kek).
			MaxEncryptedKeyBytes(math.MaxInt).
			CipherReferenceResolver(xmlenc1.FSReferenceResolver(fsys, "")).
			Decrypt(t.Context(), elem)
		require.NoError(t, err)
		requireSecret(t, nodes)
	})

	// A relative URI is joined against the document's base URI before it
	// reaches the resolver, exactly as any other relative reference is.
	t.Run("a relative URI joins the document base URI", func(t *testing.T) {
		sessionKey := newSessionKey(t)
		fsys := fstest.MapFS{"data/ct.bin": &fstest.MapFile{Data: cipherRefCiphertext(t, sessionKey)}}
		doc := mustParseXML(t, `<root xml:base="data/index.xml">`+
			`<xenc:EncryptedData xmlns:xenc="`+xmlenc1.NamespaceXMLEnc+`" xmlns:ds="`+xmlenc1.NamespaceDSig+`" Type="`+xmlenc1.TypeElement+`">`+
			`<xenc:EncryptionMethod Algorithm="`+xmlenc1.AES256GCM+`"/>`+
			`<xenc:CipherData>`+cipherReferenceXML(externalCipherURI, "")+`</xenc:CipherData>`+
			`</xenc:EncryptedData></root>`)
		elem := findEncryptedData(t, doc.DocumentElement())
		require.NotNil(t, elem)

		nodes, err := xmlenc1.NewDecryptor().
			SessionKey(sessionKey).
			CipherReferenceResolver(xmlenc1.FSReferenceResolver(fsys, "")).
			Decrypt(t.Context(), elem)
		require.NoError(t, err)
		requireSecret(t, nodes)
	})

	// The base a URI is joined against is the one in force at the element
	// CARRYING the URI, so an xml:base between the EncryptedData and the
	// CipherReference counts. Taking the base once at the EncryptedData would
	// join against a base the reference is not written in.
	t.Run("xml:base on the CipherReference is honored", func(t *testing.T) {
		sessionKey := newSessionKey(t)
		fsys := fstest.MapFS{"data/ct.bin": &fstest.MapFile{Data: cipherRefCiphertext(t, sessionKey)}}
		doc := mustParseXML(t, `<root>`+
			`<xenc:EncryptedData xmlns:xenc="`+xmlenc1.NamespaceXMLEnc+`" xmlns:ds="`+xmlenc1.NamespaceDSig+`" Type="`+xmlenc1.TypeElement+`">`+
			`<xenc:EncryptionMethod Algorithm="`+xmlenc1.AES256GCM+`"/>`+
			`<xenc:CipherData><xenc:CipherReference xml:base="data/"`+externalCipherURI+`/></xenc:CipherData>`+
			`</xenc:EncryptedData></root>`)
		elem := findEncryptedData(t, doc.DocumentElement())
		require.NotNil(t, elem)

		nodes, err := xmlenc1.NewDecryptor().
			SessionKey(sessionKey).
			CipherReferenceResolver(xmlenc1.FSReferenceResolver(fsys, "")).
			Decrypt(t.Context(), elem)
		require.NoError(t, err)
		requireSecret(t, nodes)
	})

	// A resource whose reads never return does not out-wait its caller: the
	// package selects on the context while it reads and closes the stream, which
	// is what releases the blocked read.
	t.Run("a blocking resource does not out-wait the deadline", func(t *testing.T) {
		fsys := newBlockingFS()
		t.Cleanup(fsys.release)
		elem := cipherRefDoc(t, cipherReferenceXML(externalCipherURI, ""), "")
		decryptor := xmlenc1.NewDecryptor().
			SessionKey(newSessionKey(t)).
			CipherReferenceResolver(xmlenc1.FSReferenceResolver(fsys, ""))

		ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
		defer cancel()
		done := make(chan error, 1)
		go func() {
			_, err := decryptor.Decrypt(ctx, elem)
			done <- err
		}()
		select {
		case err := <-done:
			require.ErrorIs(t, err, context.DeadlineExceeded)
		case <-time.After(10 * time.Second):
			t.Fatal("the decrypt ran on past its caller's deadline")
		}
	})

	// CipherReferenceResolver is clone-on-write like every other builder
	// method: configuring one leaves the Decryptor it was called on failing
	// closed.
	t.Run("CipherReferenceResolver does not mutate its receiver", func(t *testing.T) {
		sessionKey := newSessionKey(t)
		fsys := fstest.MapFS{externalCipherPath: &fstest.MapFile{Data: cipherRefCiphertext(t, sessionKey)}}
		base := xmlenc1.NewDecryptor().SessionKey(sessionKey)
		withResolver := base.CipherReferenceResolver(xmlenc1.FSReferenceResolver(fsys, ""))

		elem := cipherRefDoc(t, cipherReferenceXML(externalCipherURI, ""), "")
		_, err := base.Decrypt(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrReferenceNotFound)

		nodes, err := withResolver.Decrypt(t.Context(), cipherRefDoc(t, cipherReferenceXML(externalCipherURI, ""), ""))
		require.NoError(t, err)
		requireSecret(t, nodes)
	})
}
