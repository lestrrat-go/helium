package xmlenc1_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"runtime"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmlenc1"
	"github.com/stretchr/testify/require"
)

// retrievalPlaintext is the element every RetrievalMethod document below
// encrypts, so a successful decrypt is recognizable by name alone.
const retrievalPlaintext = `<secret>hidden</secret>`

// wrappedKeyXML renders an xenc:EncryptedKey holding sessionKey under an
// AES-256 key wrap, carrying the given Id and the given extra markup inside
// the EncryptedKey (a ds:KeyInfo, say). The wrap is computed here, and never
// produced by an Encryptor, because these documents place the EncryptedKey
// somewhere no Encryptor writes one: outside the EncryptedData that names it.
func wrappedKeyXML(t *testing.T, id string, kek, sessionKey []byte, extra string) string {
	t.Helper()
	wrapped, err := xmlenc1.AESKeyWrapForTest(kek, sessionKey)
	require.NoError(t, err)
	idAttr := ""
	if id != "" {
		idAttr = ` Id="` + id + `"`
	}
	return `<xenc:EncryptedKey` + idAttr + `>` +
		`<xenc:EncryptionMethod Algorithm="` + xmlenc1.AES256KeyWrap + `"/>` +
		extra +
		`<xenc:CipherData><xenc:CipherValue>` + base64.StdEncoding.EncodeToString(wrapped) + `</xenc:CipherValue></xenc:CipherData>` +
		`</xenc:EncryptedKey>`
}

// retrievalDoc builds a document whose single EncryptedData carries keyInfo as
// the content of its ds:KeyInfo and is followed by trailing as a sibling, and
// returns that EncryptedData element. The payload is real AES-256-GCM
// ciphertext of retrievalPlaintext under sessionKey, so a decrypt that reaches
// the right key produces a <secret> element and one that does not fails.
//
// The markup carries no whitespace between elements, so the EncryptedData is
// the document element's first child whatever the case under test puts around
// it.
func retrievalDoc(t *testing.T, sessionKey []byte, keyInfo, trailing string) *helium.Element {
	t.Helper()
	cipher, err := xmlenc1.EncryptBytesForTest(xmlenc1.AES256GCM, sessionKey, []byte(retrievalPlaintext))
	require.NoError(t, err)
	doc := mustParseXML(t, `<root xmlns:xenc="`+xmlenc1.NamespaceXMLEnc+`" xmlns:ds="`+xmlenc1.NamespaceDSig+`">`+
		`<xenc:EncryptedData Type="`+xmlenc1.TypeElement+`">`+
		`<xenc:EncryptionMethod Algorithm="`+xmlenc1.AES256GCM+`"/>`+
		`<ds:KeyInfo>`+keyInfo+`</ds:KeyInfo>`+
		`<xenc:CipherData><xenc:CipherValue>`+base64.StdEncoding.EncodeToString(cipher)+`</xenc:CipherValue></xenc:CipherData>`+
		`</xenc:EncryptedData>`+
		trailing+
		`</root>`)
	elem := findEncryptedData(t, doc.DocumentElement())
	require.NotNil(t, elem)
	return elem
}

// retrievalMethodXML renders a ds:RetrievalMethod with the given Type (omitted
// when empty) and URI.
func retrievalMethodXML(typ, uri string) string {
	typAttr := ""
	if typ != "" {
		typAttr = ` Type="` + typ + `"`
	}
	return `<ds:RetrievalMethod` + typAttr + ` URI="` + uri + `"/>`
}

func retrievalMethodWithoutURIXML(typ string) string {
	typAttr := ""
	if typ != "" {
		typAttr = ` Type="` + typ + `"`
	}
	return `<ds:RetrievalMethod` + typAttr + `/>`
}

// requireSecret asserts that nodes is the single <secret> element
// retrievalPlaintext describes, i.e. that the decrypt found the session key
// the RetrievalMethod pointed at.
func requireSecret(t *testing.T, nodes []helium.Node) {
	t.Helper()
	require.Len(t, nodes, 1)
	elem, ok := nodes[0].(*helium.Element)
	require.True(t, ok)
	require.Equal(t, "secret", elem.LocalName())
}

func TestRetrievalMethod(t *testing.T) {
	// Every case shares one session key and one key-encryption key: what
	// differs between them is where the EncryptedKey sits and how the
	// RetrievalMethod names it, never the cryptography.
	newKeys := func(t *testing.T) ([]byte, []byte) {
		t.Helper()
		return randKey(t, 32), randKey(t, 32)
	}

	t.Run("a missing URI is malformed before Type skipping or session-key use", func(t *testing.T) {
		for _, typ := range []string{xmlenc1.TypeEncryptedKey, xmlenc1.NamespaceDSig + "X509Data"} {
			t.Run(typ, func(t *testing.T) {
				sessionKey := randKey(t, 32)
				elem := retrievalDoc(t, sessionKey, retrievalMethodWithoutURIXML(typ), "")
				_, err := xmlenc1.NewDecryptor().SessionKey(sessionKey).Decrypt(t.Context(), elem)
				require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
				require.Contains(t, err.Error(), "ds:RetrievalMethod has no URI attribute")
				require.NotErrorIs(t, err, xmlenc1.ErrReferenceNotFound)
			})
		}
	})

	t.Run("a present empty URI remains valid", func(t *testing.T) {
		sessionKey := randKey(t, 32)
		elem := retrievalDoc(t, sessionKey, retrievalMethodXML("", ""), "")
		nodes, err := xmlenc1.NewDecryptor().SessionKey(sessionKey).Decrypt(t.Context(), elem)
		require.NoError(t, err)
		requireSecret(t, nodes)
	})

	t.Run("a same-document target supplies the session key", func(t *testing.T) {
		sessionKey, kek := newKeys(t)
		elem := retrievalDoc(t, sessionKey,
			retrievalMethodXML(xmlenc1.TypeEncryptedKey, "#k"),
			`<ds:KeyInfo>`+wrappedKeyXML(t, "k", kek, sessionKey, "")+`</ds:KeyInfo>`)
		nodes, err := xmlenc1.NewDecryptor().KeyEncryptionKey(kek).Decrypt(t.Context(), elem)
		require.NoError(t, err)
		requireSecret(t, nodes)
	})

	// The resolution root is the topmost ancestor of the EncryptedData, so a
	// target in a branch the EncryptedData is not part of resolves too.
	t.Run("a target outside the EncryptedData resolves", func(t *testing.T) {
		t.Run("in a sibling branch", func(t *testing.T) {
			sessionKey, kek := newKeys(t)
			elem := retrievalDoc(t, sessionKey,
				retrievalMethodXML(xmlenc1.TypeEncryptedKey, "#k"),
				`<keys><nested>`+wrappedKeyXML(t, "k", kek, sessionKey, "")+`</nested></keys>`)
			nodes, err := xmlenc1.NewDecryptor().KeyEncryptionKey(kek).Decrypt(t.Context(), elem)
			require.NoError(t, err)
			requireSecret(t, nodes)
		})

		// The EncryptedData is the document element here, so the walk up its
		// ancestors ends at the EncryptedData itself and the target must be
		// found inside it. This is the shape a caller decrypting an
		// EncryptedData it holds on its own presents.
		t.Run("with the EncryptedData as the resolution root", func(t *testing.T) {
			sessionKey, kek := newKeys(t)
			cipher, err := xmlenc1.EncryptBytesForTest(xmlenc1.AES256GCM, sessionKey, []byte(retrievalPlaintext))
			require.NoError(t, err)
			doc := mustParseXML(t, `<xenc:EncryptedData xmlns:xenc="`+xmlenc1.NamespaceXMLEnc+`" xmlns:ds="`+xmlenc1.NamespaceDSig+`" Type="`+xmlenc1.TypeElement+`">`+
				`<xenc:EncryptionMethod Algorithm="`+xmlenc1.AES256GCM+`"/>`+
				`<ds:KeyInfo>`+retrievalMethodXML(xmlenc1.TypeEncryptedKey, "#k")+`</ds:KeyInfo>`+
				`<xenc:EncryptionProperties>`+wrappedKeyXML(t, "k", kek, sessionKey, "")+`</xenc:EncryptionProperties>`+
				`<xenc:CipherData><xenc:CipherValue>`+base64.StdEncoding.EncodeToString(cipher)+`</xenc:CipherValue></xenc:CipherData>`+
				`</xenc:EncryptedData>`)
			nodes, err := xmlenc1.NewDecryptor().KeyEncryptionKey(kek).Decrypt(t.Context(), doc.DocumentElement())
			require.NoError(t, err)
			requireSecret(t, nodes)
		})
	})

	// A Type this package does not implement is skipped outright: no
	// resolution is attempted, so even a URI that could not resolve at all
	// costs nothing. What the decrypt then reports depends on whether the Type
	// was RECOGNIZED. A DerivedKey is a construct this package knows and does
	// not implement, so it reports the facility it lacks; a Type from another
	// specification says nothing about a key at all, so the document is simply
	// keyless. Neither URI is ever looked at, which the two NotErrorIs
	// assertions below pin for both.
	t.Run("a foreign Type is skipped and never resolved", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			typ  string
			want error
		}{
			{name: "DerivedKey", typ: xmlenc1.NamespaceXMLEnc + "DerivedKey", want: xmlenc1.ErrUnsupportedKeyDerivation},
			{name: "X509Data", typ: xmlenc1.NamespaceDSig + "X509Data", want: xmlenc1.ErrMissingKey},
		} {
			t.Run(tc.name, func(t *testing.T) {
				sessionKey, kek := newKeys(t)
				elem := retrievalDoc(t, sessionKey,
					retrievalMethodXML(tc.typ, "https://example.com/keys.xml#k"),
					`<ds:KeyInfo>`+wrappedKeyXML(t, "k", kek, sessionKey, "")+`</ds:KeyInfo>`)
				_, err := xmlenc1.NewDecryptor().KeyEncryptionKey(kek).Decrypt(t.Context(), elem)
				require.ErrorIs(t, err, tc.want)
				require.NotErrorIs(t, err, xmlenc1.ErrReferenceNotFound)
				require.NotErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
			})
		}
	})

	// An absent Type says nothing about what the URI names, so a target that
	// is not an EncryptedKey is passed over instead of failing the document.
	t.Run("an absent Type with a non-EncryptedKey target is skipped", func(t *testing.T) {
		sessionKey, kek := newKeys(t)
		elem := retrievalDoc(t, sessionKey,
			retrievalMethodXML("", "#k"),
			`<cert Id="k">not a key</cert>`)
		_, err := xmlenc1.NewDecryptor().KeyEncryptionKey(kek).Decrypt(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrMissingKey)
		require.NotErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
	})

	// A stated EncryptedKey Type that names something else is a contradiction
	// in the document, not a reference this package may pass over.
	t.Run("an EncryptedKey Type naming a non-EncryptedKey is malformed", func(t *testing.T) {
		sessionKey, kek := newKeys(t)
		elem := retrievalDoc(t, sessionKey,
			retrievalMethodXML(xmlenc1.TypeEncryptedKey, "#k"),
			`<cert Id="k">not a key</cert>`)
		_, err := xmlenc1.NewDecryptor().KeyEncryptionKey(kek).Decrypt(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
	})

	// Two elements answering to one id is XML Signature Wrapping applied to
	// encryption: an attacker who can inject a duplicate steers which key the
	// recipient uses, so the document is refused, and resolved to
	// neither candidate.
	t.Run("a duplicate id is ambiguous", func(t *testing.T) {
		sessionKey, kek := newKeys(t)
		elem := retrievalDoc(t, sessionKey,
			retrievalMethodXML(xmlenc1.TypeEncryptedKey, "#k"),
			`<ds:KeyInfo>`+wrappedKeyXML(t, "k", kek, sessionKey, "")+
				wrappedKeyXML(t, "k", kek, randKey(t, 32), "")+`</ds:KeyInfo>`)
		_, err := xmlenc1.NewDecryptor().KeyEncryptionKey(kek).Decrypt(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrAmbiguousReference)
	})

	t.Run("a missing target is not found", func(t *testing.T) {
		sessionKey, kek := newKeys(t)
		elem := retrievalDoc(t, sessionKey,
			retrievalMethodXML(xmlenc1.TypeEncryptedKey, "#absent"),
			`<ds:KeyInfo>`+wrappedKeyXML(t, "k", kek, sessionKey, "")+`</ds:KeyInfo>`)
		_, err := xmlenc1.NewDecryptor().KeyEncryptionKey(kek).Decrypt(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrReferenceNotFound)
	})

	// Only the same-document form is REQUIRED, and an external key location
	// steers which key material the recipient trial-decrypts, so every
	// non-same-document URI is refused with no opt-in to lift it.
	t.Run("an external URI is not found", func(t *testing.T) {
		for _, uri := range []string{
			"https://example.com/keys.xml#k",
			"keys.xml#k",
			"#xpointer(//EncryptedKey)",
			"#element(/1/2)",
		} {
			t.Run(uri, func(t *testing.T) {
				sessionKey, kek := newKeys(t)
				elem := retrievalDoc(t, sessionKey,
					retrievalMethodXML(xmlenc1.TypeEncryptedKey, uri),
					`<ds:KeyInfo>`+wrappedKeyXML(t, "k", kek, sessionKey, "")+`</ds:KeyInfo>`)
				_, err := xmlenc1.NewDecryptor().KeyEncryptionKey(kek).Decrypt(t.Context(), elem)
				require.ErrorIs(t, err, xmlenc1.ErrReferenceNotFound)
			})
		}
	})

	// A refused reference names the document it was refused in when the
	// document has a base URI, so a caller decrypting many of them can tell
	// which one carried the bad reference.
	t.Run("a refused reference names the document", func(t *testing.T) {
		sessionKey, kek := newKeys(t)
		cipher, err := xmlenc1.EncryptBytesForTest(xmlenc1.AES256GCM, sessionKey, []byte(retrievalPlaintext))
		require.NoError(t, err)
		doc := mustParseXML(t, `<root xmlns:xenc="`+xmlenc1.NamespaceXMLEnc+`" xmlns:ds="`+xmlenc1.NamespaceDSig+`" xml:base="https://example.com/orders/17.xml">`+
			`<xenc:EncryptedData Type="`+xmlenc1.TypeElement+`">`+
			`<xenc:EncryptionMethod Algorithm="`+xmlenc1.AES256GCM+`"/>`+
			`<ds:KeyInfo>`+retrievalMethodXML(xmlenc1.TypeEncryptedKey, "#absent")+`</ds:KeyInfo>`+
			`<xenc:CipherData><xenc:CipherValue>`+base64.StdEncoding.EncodeToString(cipher)+`</xenc:CipherValue></xenc:CipherData>`+
			`</xenc:EncryptedData></root>`)
		elem := findEncryptedData(t, doc.DocumentElement())
		require.NotNil(t, elem)

		_, err = xmlenc1.NewDecryptor().KeyEncryptionKey(kek).Decrypt(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrReferenceNotFound)
		require.Contains(t, err.Error(), "https://example.com/orders/17.xml")
	})

	// The two full-XPointer forms XMLDSig core names are same-document
	// references like the bare id, so they resolve, and never fail closed.
	t.Run("the XPointer id form resolves", func(t *testing.T) {
		sessionKey, kek := newKeys(t)
		elem := retrievalDoc(t, sessionKey,
			retrievalMethodXML(xmlenc1.TypeEncryptedKey, "#xpointer(id('k'))"),
			`<ds:KeyInfo>`+wrappedKeyXML(t, "k", kek, sessionKey, "")+`</ds:KeyInfo>`)
		nodes, err := xmlenc1.NewDecryptor().KeyEncryptionKey(kek).Decrypt(t.Context(), elem)
		require.NoError(t, err)
		requireSecret(t, nodes)
	})

	for _, tc := range []struct {
		name string
		ws   string
	}{
		{name: "space", ws: " "},
		{name: "tab", ws: "&#x9;"},
		{name: "carriage return", ws: "&#xD;"},
		{name: "line feed", ws: "&#xA;"},
	} {
		t.Run("XPath S accepts "+tc.name, func(t *testing.T) {
			sessionKey, kek := newKeys(t)
			uri := "#xpointer(" + tc.ws + "id(" + tc.ws + "'k'" + tc.ws + ")" + tc.ws + ")"
			elem := retrievalDoc(t, sessionKey,
				retrievalMethodXML(xmlenc1.TypeEncryptedKey, uri),
				`<ds:KeyInfo>`+wrappedKeyXML(t, "k", kek, sessionKey, "")+`</ds:KeyInfo>`)
			nodes, err := xmlenc1.NewDecryptor().KeyEncryptionKey(kek).Decrypt(t.Context(), elem)
			require.NoError(t, err)
			requireSecret(t, nodes)
		})
	}

	for _, tc := range []struct {
		name string
		ws   string
	}{
		{name: "no-break space", ws: "\u00a0"},
		{name: "next line", ws: "\u0085"},
		{name: "line separator", ws: "\u2028"},
		{name: "ideographic space", ws: "\u3000"},
	} {
		t.Run("rejects "+tc.name+" around the expression", func(t *testing.T) {
			sessionKey, kek := newKeys(t)
			uri := "#xpointer(" + tc.ws + "id('k')" + tc.ws + ")"
			elem := retrievalDoc(t, sessionKey,
				retrievalMethodXML(xmlenc1.TypeEncryptedKey, uri),
				`<ds:KeyInfo>`+wrappedKeyXML(t, "k", kek, sessionKey, "")+`</ds:KeyInfo>`)
			_, err := xmlenc1.NewDecryptor().KeyEncryptionKey(kek).Decrypt(t.Context(), elem)
			require.ErrorIs(t, err, xmlenc1.ErrReferenceNotFound)
		})
		t.Run("rejects "+tc.name+" around the id argument", func(t *testing.T) {
			sessionKey, kek := newKeys(t)
			uri := "#xpointer(id(" + tc.ws + "'k'" + tc.ws + "))"
			elem := retrievalDoc(t, sessionKey,
				retrievalMethodXML(xmlenc1.TypeEncryptedKey, uri),
				`<ds:KeyInfo>`+wrappedKeyXML(t, "k", kek, sessionKey, "")+`</ds:KeyInfo>`)
			_, err := xmlenc1.NewDecryptor().KeyEncryptionKey(kek).Decrypt(t.Context(), elem)
			require.ErrorIs(t, err, xmlenc1.ErrReferenceNotFound)
		})
	}

	t.Run("quoted id whitespace is preserved", func(t *testing.T) {
		sessionKey, kek := newKeys(t)
		elem := retrievalDoc(t, sessionKey,
			retrievalMethodXML(xmlenc1.TypeEncryptedKey, "#xpointer(id(' key '))"),
			`<ds:KeyInfo>`+wrappedKeyXML(t, "key", kek, sessionKey, "")+`</ds:KeyInfo>`)
		_, err := xmlenc1.NewDecryptor().KeyEncryptionKey(kek).Decrypt(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrReferenceNotFound)
	})

	// xmlenc-core1 §3.5.3 permits several RetrievalMethods, and two naming the
	// same EncryptedKey describe one key: it must be trial-decrypted once and
	// charged once, not once per reference.
	t.Run("two references to one EncryptedKey yield one candidate", func(t *testing.T) {
		sessionKey, kek := newKeys(t)
		build := func(t *testing.T) *helium.Element {
			t.Helper()
			return retrievalDoc(t, sessionKey,
				retrievalMethodXML(xmlenc1.TypeEncryptedKey, "#k")+
					retrievalMethodXML(xmlenc1.TypeEncryptedKey, "#xpointer(id('k'))"),
				`<ds:KeyInfo>`+wrappedKeyXML(t, "k", kek, sessionKey, "")+`</ds:KeyInfo>`)
		}

		ed, err := xmlenc1.ParseEncryptedDataForTest(build(t))
		require.NoError(t, err)
		require.Len(t, ed.EncryptedKeys, 1)

		// One count charge and one ciphertext charge: a cap of one and a
		// budget of exactly one key's worth both admit the document, which a
		// second charge of either would not.
		nodes, err := xmlenc1.NewDecryptor().
			KeyEncryptionKey(kek).
			MaxEncryptedKeys(1).
			MaxEncryptedKeyBytes(len(ed.EncryptedKeys[0].CipherValue)).
			Decrypt(t.Context(), build(t))
		require.NoError(t, err)
		requireSecret(t, nodes)
	})

	// An inline EncryptedKey marks itself taken, so a reference naming that
	// very element is a repeat like any other and adds no candidate.
	t.Run("a reference naming the inline EncryptedKey yields one candidate", func(t *testing.T) {
		sessionKey, kek := newKeys(t)
		build := func(t *testing.T) *helium.Element {
			t.Helper()
			return retrievalDoc(t, sessionKey,
				wrappedKeyXML(t, "k", kek, sessionKey, "")+
					retrievalMethodXML(xmlenc1.TypeEncryptedKey, "#k"),
				"")
		}

		ed, err := xmlenc1.ParseEncryptedDataForTest(build(t))
		require.NoError(t, err)
		require.Len(t, ed.EncryptedKeys, 1)

		nodes, err := xmlenc1.NewDecryptor().
			KeyEncryptionKey(kek).
			MaxEncryptedKeys(1).
			Decrypt(t.Context(), build(t))
		require.NoError(t, err)
		requireSecret(t, nodes)
	})

	// The cap counts candidates, not references examined, so a flood of
	// references naming one key costs one slot however long the flood is. The
	// byte budget is charged at the same point, so it too is charged once.
	t.Run("many references to one EncryptedKey yield one candidate", func(t *testing.T) {
		sessionKey, kek := newKeys(t)
		build := func(t *testing.T) *helium.Element {
			t.Helper()
			refs := strings.Repeat(retrievalMethodXML(xmlenc1.TypeEncryptedKey, "#k"), 50)
			return retrievalDoc(t, sessionKey, refs,
				`<ds:KeyInfo>`+wrappedKeyXML(t, "k", kek, sessionKey, "")+`</ds:KeyInfo>`)
		}

		ed, err := xmlenc1.ParseEncryptedDataForTest(build(t))
		require.NoError(t, err)
		require.Len(t, ed.EncryptedKeys, 1)

		nodes, err := xmlenc1.NewDecryptor().
			KeyEncryptionKey(kek).
			MaxEncryptedKeys(1).
			Decrypt(t.Context(), build(t))
		require.NoError(t, err)
		requireSecret(t, nodes)

		nodes, err = xmlenc1.NewDecryptor().
			KeyEncryptionKey(kek).
			MaxEncryptedKeyBytes(len(ed.EncryptedKeys[0].CipherValue)).
			Decrypt(t.Context(), build(t))
		require.NoError(t, err)
		requireSecret(t, nodes)
	})

	// A reference-supplied candidate lands where the document put the
	// reference, so a caller reading the candidate list sees document order,
	// and never inline candidates first.
	t.Run("a reference keeps document order alongside an inline EncryptedKey", func(t *testing.T) {
		sessionKey, kek := newKeys(t)
		inline := wrappedKeyXML(t, "", kek, randKey(t, 32), "")
		referenced := `<ds:KeyInfo>` + wrappedKeyXML(t, "k", kek, sessionKey, "") + `</ds:KeyInfo>`
		reference := retrievalMethodXML(xmlenc1.TypeEncryptedKey, "#k")

		cipherValues := func(t *testing.T, keyInfo string) [][]byte {
			t.Helper()
			ed, err := xmlenc1.ParseEncryptedDataForTest(retrievalDoc(t, sessionKey, keyInfo, referenced))
			require.NoError(t, err)
			values := make([][]byte, 0, len(ed.EncryptedKeys))
			for _, ek := range ed.EncryptedKeys {
				values = append(values, ek.CipherValue)
			}
			return values
		}

		first := cipherValues(t, inline+reference)
		require.Len(t, first, 2)
		second := cipherValues(t, reference+inline)
		require.Len(t, second, 2)
		require.Equal(t, first[0], second[1])
		require.Equal(t, first[1], second[0])

		// Whichever order the document states, the referenced key still
		// decrypts: the inline candidate wraps a different session key and
		// simply fails first.
		nodes, err := xmlenc1.NewDecryptor().KeyEncryptionKey(kek).Decrypt(t.Context(), retrievalDoc(t, sessionKey, inline+reference, referenced))
		require.NoError(t, err)
		requireSecret(t, nodes)
	})

	// parseEncryptedKey reads only EncryptionMethod, CipherData, and
	// ds:KeyInfo -> AgreementMethod, so a RetrievalMethod inside the target's
	// own ds:KeyInfo is never looked at. Resolution is one step deep by
	// construction: the self-reference below is inert, and the decrypt
	// terminates with the key the target holds.
	t.Run("a RetrievalMethod inside the target is not followed", func(t *testing.T) {
		sessionKey, kek := newKeys(t)
		nested := `<ds:KeyInfo>` + retrievalMethodXML(xmlenc1.TypeEncryptedKey, "#k") + `</ds:KeyInfo>`
		elem := retrievalDoc(t, sessionKey,
			retrievalMethodXML(xmlenc1.TypeEncryptedKey, "#k"),
			`<ds:KeyInfo>`+wrappedKeyXML(t, "k", kek, sessionKey, nested)+`</ds:KeyInfo>`)
		nodes, err := xmlenc1.NewDecryptor().KeyEncryptionKey(kek).Decrypt(t.Context(), elem)
		require.NoError(t, err)
		requireSecret(t, nodes)
	})

	// DecryptBytes builds the same resolution scope as Decrypt, so a
	// reference-supplied key is available on both terminals.
	t.Run("DecryptBytes resolves references too", func(t *testing.T) {
		sessionKey, kek := newKeys(t)
		elem := retrievalDoc(t, sessionKey,
			retrievalMethodXML(xmlenc1.TypeEncryptedKey, "#k"),
			`<ds:KeyInfo>`+wrappedKeyXML(t, "k", kek, sessionKey, "")+`</ds:KeyInfo>`)
		plaintext, err := xmlenc1.NewDecryptor().KeyEncryptionKey(kek).DecryptBytes(t.Context(), elem)
		require.NoError(t, err)
		require.Equal(t, retrievalPlaintext, string(plaintext))
	})

	// A reference is resolved while the document is read, which precedes the
	// pre-shared SessionKey early return, so a refused reference fails a
	// decrypt the caller could otherwise have completed on its own key. A
	// reference that is skipped, and never resolved, costs that caller
	// nothing.
	t.Run("a pre-shared SessionKey", func(t *testing.T) {
		t.Run("does not decrypt past a refused reference", func(t *testing.T) {
			sessionKey, _ := newKeys(t)
			elem := retrievalDoc(t, sessionKey, retrievalMethodXML(xmlenc1.TypeEncryptedKey, "#absent"), "")
			_, err := xmlenc1.NewDecryptor().SessionKey(sessionKey).Decrypt(t.Context(), elem)
			require.ErrorIs(t, err, xmlenc1.ErrReferenceNotFound)
		})

		t.Run("decrypts past a skipped foreign Type", func(t *testing.T) {
			sessionKey, _ := newKeys(t)
			elem := retrievalDoc(t, sessionKey,
				retrievalMethodXML(xmlenc1.NamespaceXMLEnc+"DerivedKey", "#absent"), "")
			nodes, err := xmlenc1.NewDecryptor().SessionKey(sessionKey).Decrypt(t.Context(), elem)
			require.NoError(t, err)
			requireSecret(t, nodes)
		})
	})
}

// wrappedKeyLen is the decoded CipherValue length wrappedKeyXML produces for
// the given keys taken together: RFC 3394 key wrap adds one 8-octet block to
// each key it wraps. The keys the matrix below wraps are all of DIFFERENT
// lengths, so a total identifies WHICH keys were charged, and not only how
// many.
func wrappedKeyLen(keys ...[]byte) int {
	total := 0
	for _, key := range keys {
		total += len(key) + 8
	}
	return total
}

// TestEncryptedKeyRetention drives every order in which one ds:KeyInfo can
// offer xenc:EncryptedKey elements — carried inline, and named by a
// ds:RetrievalMethod (xmlenc-core1 §3.5.3) — and holds each order to the same
// invariant: one element is one candidate however many times the document
// offers it, the peak candidate charge equals the final candidate count
// exactly, and each retained candidate's ciphertext is charged once.
//
// A cross-ds:KeyInfo order is absent because it is unreachable:
// parseEncryptedData refuses a second ds:KeyInfo as a duplicate.
func TestEncryptedKeyRetention(t *testing.T) {
	sessionKey := randKey(t, 32)
	kek := randKey(t, 32)
	// Two decoys, each a different length from the session key and from each
	// other, so a row's byte total names the keys it expects to be charged.
	// They wrap under the same KEK and unwrap to a length the AES-256-GCM
	// payload cannot use, so a candidate holding one simply fails and the
	// next is tried.
	shortDecoy := randKey(t, 16)
	longDecoy := randKey(t, 24)

	inlineSession := wrappedKeyXML(t, "k1", kek, sessionKey, "")
	inlineShort := wrappedKeyXML(t, "k2", kek, shortDecoy, "")
	anonymousInline := wrappedKeyXML(t, "", kek, longDecoy, "")
	// The session key nested inside another inline key's ds:KeyInfo, which
	// parseEncryptedKey reads only for an xenc:AgreementMethod: the nested
	// key is never walked inline, so only the reference reaches it.
	nestingInline := wrappedKeyXML(t, "k3", kek, longDecoy, `<ds:KeyInfo>`+inlineSession+`</ds:KeyInfo>`)

	refSession := retrievalMethodXML(xmlenc1.TypeEncryptedKey, "#k1")
	refShort := retrievalMethodXML(xmlenc1.TypeEncryptedKey, "#k2")

	for _, tc := range []struct {
		name     string
		keyInfo  string
		trailing string
		// keys are the DISTINCT EncryptedKey targets the ds:KeyInfo
		// retains, in the order the document offers them.
		keys [][]byte
	}{
		{
			name:    "a reference before the inline element it names",
			keyInfo: refSession + inlineSession,
			keys:    [][]byte{sessionKey},
		},
		{
			name:    "two references before the inline element they name",
			keyInfo: refSession + refSession + inlineSession,
			keys:    [][]byte{sessionKey},
		},
		{
			name:    "a reference on each side of the inline element they name",
			keyInfo: refSession + inlineSession + refSession,
			keys:    [][]byte{sessionKey},
		},
		{
			name:     "a reference to an outside key before a reference to the inline element",
			keyInfo:  refShort + refSession + inlineSession,
			trailing: `<keys>` + inlineShort + `</keys>`,
			keys:     [][]byte{shortDecoy, sessionKey},
		},
		{
			name:    "two inline elements both referenced first",
			keyInfo: refSession + refShort + inlineSession + inlineShort,
			keys:    [][]byte{sessionKey, shortDecoy},
		},
		{
			name:    "an anonymous inline element before a reference to a later inline element",
			keyInfo: anonymousInline + refSession + inlineSession,
			keys:    [][]byte{longDecoy, sessionKey},
		},
		{
			// A reference form carries no weight of its own: the
			// retention path is the same one every form reaches.
			name:    "a reference with no Type before the inline element it names",
			keyInfo: retrievalMethodXML("", "#k1") + inlineSession,
			keys:    [][]byte{sessionKey},
		},
		{
			name:    "an XPointer reference before the inline element it names",
			keyInfo: retrievalMethodXML(xmlenc1.TypeEncryptedKey, "#xpointer(id('k1'))") + inlineSession,
			keys:    [][]byte{sessionKey},
		},
		{
			name:    "an inline element alone",
			keyInfo: inlineSession,
			keys:    [][]byte{sessionKey},
		},
		{
			name:    "a reference after the inline element it names",
			keyInfo: inlineSession + refSession,
			keys:    [][]byte{sessionKey},
		},
		{
			name:    "two references after the inline element they name",
			keyInfo: inlineSession + refSession + refSession,
			keys:    [][]byte{sessionKey},
		},
		{
			name:     "a reference to a key outside the KeyInfo",
			keyInfo:  refSession,
			trailing: `<ds:KeyInfo>` + inlineSession + `</ds:KeyInfo>`,
			keys:     [][]byte{sessionKey},
		},
		{
			name:     "an inline element and a reference to a distinct outside key",
			keyInfo:  inlineShort + refSession,
			trailing: `<keys>` + inlineSession + `</keys>`,
			keys:     [][]byte{shortDecoy, sessionKey},
		},
		{
			name:    "two inline elements with references after both",
			keyInfo: inlineSession + inlineShort + refSession + refShort,
			keys:    [][]byte{sessionKey, shortDecoy},
		},
		{
			name:    "a reference to a key nested inside an inline key",
			keyInfo: nestingInline + refSession,
			keys:    [][]byte{longDecoy, sessionKey},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			elem := retrievalDoc(t, sessionKey, tc.keyInfo, tc.trailing)
			distinct := len(tc.keys)
			keyBytes := wrappedKeyLen(tc.keys...)

			// The candidate list itself. No budget states this: a
			// repeated candidate is a correctness bug with both caps
			// unlimited, because it is trial-decrypted twice.
			ed, err := xmlenc1.ParseEncryptedDataForTest(elem)
			require.NoError(t, err)
			require.Len(t, ed.EncryptedKeys, distinct)

			// The document decrypts at exactly the candidate count and
			// exactly the ciphertext total its distinct targets need, so
			// neither charge is made twice for one element.
			nodes, err := xmlenc1.NewDecryptor().
				KeyEncryptionKey(kek).
				MaxEncryptedKeys(distinct).
				MaxEncryptedKeyBytes(keyBytes).
				Decrypt(t.Context(), elem)
			require.NoError(t, err)
			requireSecret(t, nodes)

			// One byte less is refused, which is what says the total
			// covers every distinct target, and no subset of them.
			_, err = xmlenc1.NewDecryptor().
				KeyEncryptionKey(kek).
				MaxEncryptedKeys(-1).
				MaxEncryptedKeyBytes(keyBytes-1).
				Decrypt(t.Context(), elem)
			require.ErrorIs(t, err, xmlenc1.ErrEncryptedKeyBytesExceeded)

			// One candidate short of the distinct count is refused too,
			// so a retention that UNDER-charges fails here, and never
			// passes the two assertions above. A single-target row
			// cannot state this direction, because MaxEncryptedKeys(0)
			// selects the default cap, and no cap of none; its
			// byte assertion carries it instead.
			if distinct < 2 {
				return
			}
			_, err = xmlenc1.NewDecryptor().
				KeyEncryptionKey(kek).
				MaxEncryptedKeys(distinct-1).
				MaxEncryptedKeyBytes(-1).
				Decrypt(t.Context(), elem)
			require.ErrorIs(t, err, xmlenc1.ErrTooManyEncryptedKeys)
		})
	}
}

// rsaWrappedKeyXML renders an xenc:EncryptedKey carrying an RSA-OAEP
// ciphertext under pub, so resolving it costs one RSA private-key operation.
// The ciphertext is built with SHA-256 while the element declares RSA-OAEP's
// default SHA-1 label digest, so that operation runs in full and the padding
// check then fails: what the caller measures is the WORK one candidate costs,
// and every candidate is therefore reached.
func rsaWrappedKeyXML(t *testing.T, id string, pub *rsa.PublicKey) string {
	t.Helper()
	wrapped, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, randKey(t, 32), nil)
	require.NoError(t, err)
	return `<xenc:EncryptedKey Id="` + id + `">` +
		`<xenc:EncryptionMethod Algorithm="` + xmlenc1.RSAOAEP11 + `"/>` +
		`<xenc:CipherData><xenc:CipherValue>` + base64.StdEncoding.EncodeToString(wrapped) + `</xenc:CipherValue></xenc:CipherData>` +
		`</xenc:EncryptedKey>`
}

// decryptAllocBytes reports the bytes one failing Decrypt of elem allocates,
// which is how the RSA private-key operations it performs are counted: the
// error the terminal returns is the same whether a candidate was resolved once
// or twice.
func decryptAllocBytes(t *testing.T, decryptor xmlenc1.Decryptor, elem *helium.Element) uint64 {
	t.Helper()
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	_, err := decryptor.Decrypt(t.Context(), elem)
	runtime.ReadMemStats(&after)
	require.Error(t, err)
	return after.TotalAlloc - before.TotalAlloc
}

// TestEncryptedKeyRetentionTrialDecryptions pins the security consequence the
// candidate caps exist to bound: a document that offers ONE EncryptedKey
// element both by ds:RetrievalMethod and inline must cost ONE trial
// decryption, not two. A repeat candidate is trial-decrypted like any other,
// so a document could otherwise double the private-key work it buys.
//
// The count is read from what a decrypt ALLOCATES, following
// TestEncryptedKeyBytesAllocation's precedent for measuring what an error
// cannot report. The cost of one extra candidate is measured in the same
// suite, and never assumed, so the bound holds whatever an RSA operation
// happens to allocate. This test must NOT run in parallel: TotalAlloc is
// process-wide and a concurrent test would pollute the delta.
func TestEncryptedKeyRetentionTrialDecryptions(t *testing.T) {
	// no t.Parallel(): isolated so each delta reflects only its own Decrypt.
	key := generateRSAKey(t)
	sessionKey := randKey(t, 32)
	inline := rsaWrappedKeyXML(t, "k1", &key.PublicKey)
	otherInline := rsaWrappedKeyXML(t, "k2", &key.PublicKey)
	reference := retrievalMethodXML(xmlenc1.TypeEncryptedKey, "#k1")
	// A reference that resolves and retains NOTHING, so the document it
	// sits in pays everything the measured one pays — an extra ds:KeyInfo
	// child, and the id index resolveSameDocument builds — except the
	// candidate itself. It is the baseline the measurement is read against,
	// so what remains between the two is one trial decryption and nothing
	// else.
	inertReference := retrievalMethodXML("", "#cert")

	decryptor := xmlenc1.NewDecryptor().PrivateKey(key)
	one := decryptAllocBytes(t, decryptor, retrievalDoc(t, sessionKey, inline, ""))
	two := decryptAllocBytes(t, decryptor, retrievalDoc(t, sessionKey, inline+otherInline, ""))
	baseline := decryptAllocBytes(t, decryptor, retrievalDoc(t, sessionKey, inertReference+inline, `<cert Id="cert">not a key</cert>`))
	refThenInline := decryptAllocBytes(t, decryptor, retrievalDoc(t, sessionKey, reference+inline, ""))
	t.Logf("one candidate: %d bytes; two candidates: %d bytes; one candidate behind a reference that retains nothing: %d bytes; a reference then the inline element it names: %d bytes", one, two, baseline, refThenInline)

	// The cost of one more trial decryption, measured in this same suite
	// and never assumed, so the bound holds whatever an RSA operation of
	// the day allocates.
	perCandidate := two - one
	require.Greater(t, two, one, "a second candidate must cost measurably more than one for this bound to mean anything")
	require.Less(t, refThenInline, baseline+perCandidate/2, "a reference naming the inline element performed a second trial decryption")
}
