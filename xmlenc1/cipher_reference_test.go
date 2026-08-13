package xmlenc1_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"math"
	"runtime"
	"strings"
	"testing"
	"testing/fstest"

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
// The xenc and ds prefixes are declared on the EncryptedData rather than on the
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

// countingResolver records how many times a decrypt asked it for a resource
// and always refuses, so a test can assert that a document never reached the
// dereferencing stage at all.
type countingResolver struct {
	calls int
}

func (r *countingResolver) ResolveReference(_ context.Context, uri string) ([]byte, error) {
	r.calls++
	return nil, fmt.Errorf("%w: countingResolver serves nothing (%q)", xmlenc1.ErrReferenceNotFound, uri)
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

	// The two whole-document forms name the root rather than an id, so their
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
	// choose which octets the recipient decrypts, so the document is refused
	// rather than resolved to either.
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

	// External dereferencing is RECOMMENDED rather than REQUIRED by the
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
			CipherReferenceResolver(xmlenc1.FSReferenceResolver(fsys)).
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
			CipherReferenceResolver(xmlenc1.FSReferenceResolver(fstest.MapFS{})).
			Decrypt(t.Context(), elem)
		require.NoError(t, err)
		requireSecret(t, nodes)
	})

	t.Run("the resolver refuses", func(t *testing.T) {
		fsys := fstest.MapFS{externalCipherPath: &fstest.MapFile{Data: []byte("payload")}}
		resolver := xmlenc1.FSReferenceResolver(fsys)
		for _, tc := range []struct {
			name string
			uri  string
		}{
			{name: "an http scheme", uri: "https://example.com/ct.bin"},
			{name: "a file scheme", uri: "file:///etc/passwd"},
			{name: "a Windows drive letter", uri: `C:\ct.bin`},
			{name: "a parent escape", uri: "../ct.bin"},
			{name: "a nested parent escape", uri: "sub/../../ct.bin"},
			{name: "an absolute path", uri: "/ct.bin"},
			{name: "a fragment", uri: "ct.bin#frag"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				_, err := resolver.ResolveReference(t.Context(), tc.uri)
				require.ErrorIs(t, err, xmlenc1.ErrReferenceNotFound)
			})
		}

		// A plain in-tree path is what remains after those refusals.
		t.Run("but serves a plain path", func(t *testing.T) {
			octets, err := resolver.ResolveReference(t.Context(), externalCipherPath)
			require.NoError(t, err)
			require.Equal(t, "payload", string(octets))
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
	// wrapper is schema-invalid and refused rather than silently merged.
	t.Run("two Transforms elements are refused", func(t *testing.T) {
		elem := cipherRefDoc(t,
			cipherReferenceXML(` URI="#t"`, transformsXML(base64Transform)+transformsXML(base64Transform)),
			`<ct Id="t">AA==</ct>`)
		_, err := xmlenc1.NewDecryptor().SessionKey(newSessionKey(t)).Decrypt(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
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
	// too — and it is charged to the EncryptedKey budget rather than the
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
				CipherReferenceResolver(xmlenc1.FSReferenceResolver(fsys)).
				Decrypt(t.Context(), elem)
			require.ErrorIs(t, err, xmlenc1.ErrDecryptionFailed)
			require.NotErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
		})
	})

	// The canonicalization writes through a budgetWriter, so an oversized
	// subtree is stopped at the limit rather than canonicalized in full and
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

		// A negative budget removes the limit, so the same document
		// canonicalizes in full: that is what the bounded run must not pay.
		// Neither run can avoid the one copy the DOM hands out per text node,
		// which is this package's documented floor for reading a value.
		unbounded := spend(t, -1)
		bounded := spend(t, 1024)
		t.Logf("canonicalizing a %d byte subtree allocated %d bytes unbounded and %d bytes under a 1024 byte budget", len(big), unbounded, bounded)
		require.Less(t, bounded, unbounded/2, "the canonicalization ran to completion before the budget refused it")
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
			CipherReferenceResolver(xmlenc1.FSReferenceResolver(fsys)).
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
			CipherReferenceResolver(xmlenc1.FSReferenceResolver(fsys)).
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
			CipherReferenceResolver(xmlenc1.FSReferenceResolver(fsys)).
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
			CipherReferenceResolver(xmlenc1.FSReferenceResolver(fsys)).
			Decrypt(t.Context(), elem)
		require.NoError(t, err)
		requireSecret(t, nodes)
	})

	// CipherReferenceResolver is clone-on-write like every other builder
	// method: configuring one leaves the Decryptor it was called on failing
	// closed.
	t.Run("CipherReferenceResolver does not mutate its receiver", func(t *testing.T) {
		sessionKey := newSessionKey(t)
		fsys := fstest.MapFS{externalCipherPath: &fstest.MapFile{Data: cipherRefCiphertext(t, sessionKey)}}
		base := xmlenc1.NewDecryptor().SessionKey(sessionKey)
		withResolver := base.CipherReferenceResolver(xmlenc1.FSReferenceResolver(fsys))

		elem := cipherRefDoc(t, cipherReferenceXML(externalCipherURI, ""), "")
		_, err := base.Decrypt(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrReferenceNotFound)

		nodes, err := withResolver.Decrypt(t.Context(), cipherRefDoc(t, cipherReferenceXML(externalCipherURI, ""), ""))
		require.NoError(t, err)
		requireSecret(t, nodes)
	})
}
