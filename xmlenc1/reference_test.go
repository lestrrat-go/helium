package xmlenc1_test

import (
	"encoding/base64"
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
// the EncryptedKey (a ds:KeyInfo, say). The wrap is computed here rather than
// produced by an Encryptor because these documents place the EncryptedKey
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
	// costs nothing and the decrypt fails only for want of a key.
	t.Run("a foreign Type is skipped rather than resolved", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			typ  string
		}{
			{name: "DerivedKey", typ: xmlenc1.NamespaceXMLEnc + "DerivedKey"},
			{name: "X509Data", typ: xmlenc1.NamespaceDSig + "X509Data"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				sessionKey, kek := newKeys(t)
				elem := retrievalDoc(t, sessionKey,
					retrievalMethodXML(tc.typ, "https://example.com/keys.xml#k"),
					`<ds:KeyInfo>`+wrappedKeyXML(t, "k", kek, sessionKey, "")+`</ds:KeyInfo>`)
				_, err := xmlenc1.NewDecryptor().KeyEncryptionKey(kek).Decrypt(t.Context(), elem)
				require.ErrorIs(t, err, xmlenc1.ErrMissingKey)
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
	// recipient uses, so the document is refused rather than resolved to
	// either candidate.
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
	// references like the bare id, so they resolve rather than fail closed.
	t.Run("the XPointer id form resolves", func(t *testing.T) {
		sessionKey, kek := newKeys(t)
		elem := retrievalDoc(t, sessionKey,
			retrievalMethodXML(xmlenc1.TypeEncryptedKey, "#xpointer(id('k'))"),
			`<ds:KeyInfo>`+wrappedKeyXML(t, "k", kek, sessionKey, "")+`</ds:KeyInfo>`)
		nodes, err := xmlenc1.NewDecryptor().KeyEncryptionKey(kek).Decrypt(t.Context(), elem)
		require.NoError(t, err)
		requireSecret(t, nodes)
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

		// One ciphertext charge: a budget of exactly one key's worth admits
		// the document, which a second charge would not. The candidate cap is
		// checked before a reference is resolved, so the repeat reference
		// still occupies a check even though it adds no candidate; a cap of
		// two is what that ordering costs.
		nodes, err := xmlenc1.NewDecryptor().
			KeyEncryptionKey(kek).
			MaxEncryptedKeys(2).
			MaxEncryptedKeyBytes(len(ed.EncryptedKeys[0].CipherValue)).
			Decrypt(t.Context(), build(t))
		require.NoError(t, err)
		requireSecret(t, nodes)
	})

	// A reference-supplied candidate lands where the document put the
	// reference, so a caller reading the candidate list sees document order
	// rather than inline candidates first.
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
	// reference that is skipped rather than resolved costs that caller
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
