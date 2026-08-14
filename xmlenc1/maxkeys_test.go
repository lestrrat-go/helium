package xmlenc1_test

import (
	"context"
	"encoding/base64"
	"errors"
	"runtime"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmlenc1"
	"github.com/stretchr/testify/require"
)

// rsaWrappedKeyBytes is the CipherValue length of a real RSA-2048 wrapped
// key, the size a test uses when the candidate's size is beside the point.
const rsaWrappedKeyBytes = 256

// junkRSAEncryptedKeys builds n syntactically valid RSA-OAEP EncryptedKey
// candidates of size bytes each, whose CipherValue is junk, so a test can
// pack a candidate list without any of them resolving to a session key.
// rsaWrappedKeyBytes is the size of a real one.
func junkRSAEncryptedKeys(n, size int) []*xmlenc1.EncryptedKey {
	keys := make([]*xmlenc1.EncryptedKey, 0, n)
	for range n {
		keys = append(keys, &xmlenc1.EncryptedKey{
			EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.RSAOAEP},
			CipherValue:      make([]byte, size),
		})
	}
	return keys
}

// manyKeyEncryptedData builds an EncryptedData element carrying n junk RSA
// EncryptedKey candidates of size bytes each, so a test can aim a document at
// either the candidate cap or the byte budget.
func manyKeyEncryptedData(t *testing.T, n, size int) *helium.Element {
	t.Helper()
	doc := mustParseXML(t, `<root/>`)
	ed := &xmlenc1.EncryptedData{
		Type:             xmlenc1.TypeElement,
		EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES256GCM},
		EncryptedKeys:    junkRSAEncryptedKeys(n, size),
		CipherValue:      make([]byte, 48),
	}
	elem, err := xmlenc1.MarshalEncryptedDataForTest(doc, ed)
	require.NoError(t, err)
	return elem
}

// manyKeySessionKeyEncryptedData builds an EncryptedData carrying real
// AES-256-GCM ciphertext under sessionKey plus n junk RSA EncryptedKey
// candidates. None of the candidates can resolve to a key, so a decrypt that
// succeeds here proves the session key was used without candidate selection,
// and a decrypt that fails on the cap proves the cap ran before that.
func manyKeySessionKeyEncryptedData(t *testing.T, n, size int, sessionKey []byte, plaintext string) *helium.Element {
	t.Helper()
	cipher, err := xmlenc1.EncryptBytesForTest(xmlenc1.AES256GCM, sessionKey, []byte(plaintext))
	require.NoError(t, err)
	doc := mustParseXML(t, `<root/>`)
	ed := &xmlenc1.EncryptedData{
		Type:             xmlenc1.TypeElement,
		EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES256GCM},
		EncryptedKeys:    junkRSAEncryptedKeys(n, size),
		CipherValue:      cipher,
	}
	elem, err := xmlenc1.MarshalEncryptedDataForTest(doc, ed)
	require.NoError(t, err)
	return elem
}

// rawKeyCipherValueEncryptedData builds an EncryptedData whose single
// EncryptedKey carries cipherValue as raw CipherValue text. The text is
// written straight into the document, and never marshalled from an
// EncryptedKey, which is the only way to put a chosen lexical form — base64
// the decoder rejects, interspersed whitespace, or CDATA sections — in front
// of the byte budget.
func rawKeyCipherValueEncryptedData(t *testing.T, cipherValue string) *helium.Element {
	t.Helper()
	doc := mustParseXML(t, `<xenc:EncryptedData xmlns:xenc="`+xmlenc1.NamespaceXMLEnc+`" xmlns:ds="`+xmlenc1.NamespaceDSig+`">`+
		`<xenc:EncryptionMethod Algorithm="`+xmlenc1.AES256GCM+`"/>`+
		`<ds:KeyInfo><xenc:EncryptedKey>`+
		`<xenc:EncryptionMethod Algorithm="`+xmlenc1.RSAOAEP11+`"/>`+
		`<xenc:CipherData><xenc:CipherValue>`+cipherValue+`</xenc:CipherValue></xenc:CipherData>`+
		`</xenc:EncryptedKey></ds:KeyInfo>`+
		`<xenc:CipherData><xenc:CipherValue>AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA</xenc:CipherValue></xenc:CipherData>`+
		`</xenc:EncryptedData>`)
	return doc.DocumentElement()
}

// rawPayloadCipherValueEncryptedData builds an EncryptedData whose payload
// CipherValue has caller-controlled base64 text. It reaches the payload budget
// before block-algorithm validation, key resolution, or block decryption.
func rawPayloadCipherValueEncryptedData(t *testing.T, cipherValue string) *helium.Element {
	t.Helper()
	doc := mustParseXML(t, `<xenc:EncryptedData xmlns:xenc="`+xmlenc1.NamespaceXMLEnc+`">`+
		`<xenc:CipherData><xenc:CipherValue>`+cipherValue+`</xenc:CipherValue></xenc:CipherData>`+
		`</xenc:EncryptedData>`)
	return doc.DocumentElement()
}

func TestMaxEncryptedKeys(t *testing.T) {
	t.Run("over default cap fails fast", func(t *testing.T) {
		elem := manyKeyEncryptedData(t, xmlenc1.DefaultMaxEncryptedKeys+1, rsaWrappedKeyBytes)
		_, err := xmlenc1.NewDecryptor().PrivateKey(generateRSAKey(t)).Decrypt(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrTooManyEncryptedKeys)
	})

	t.Run("at default cap is not rejected by the cap", func(t *testing.T) {
		elem := manyKeyEncryptedData(t, xmlenc1.DefaultMaxEncryptedKeys, rsaWrappedKeyBytes)
		_, err := xmlenc1.NewDecryptor().PrivateKey(generateRSAKey(t)).Decrypt(t.Context(), elem)
		require.Error(t, err)
		require.NotErrorIs(t, err, xmlenc1.ErrTooManyEncryptedKeys)
	})

	t.Run("explicit cap rejects above it", func(t *testing.T) {
		elem := manyKeyEncryptedData(t, 3, rsaWrappedKeyBytes)
		_, err := xmlenc1.NewDecryptor().PrivateKey(generateRSAKey(t)).MaxEncryptedKeys(2).Decrypt(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrTooManyEncryptedKeys)
	})

	t.Run("cap stops before retaining the first excess candidate", func(t *testing.T) {
		doc := mustParseXML(t, `<xenc:EncryptedData xmlns:xenc="`+xmlenc1.NamespaceXMLEnc+`" xmlns:ds="`+xmlenc1.NamespaceDSig+`">`+
			`<ds:KeyInfo>`+
			`<xenc:EncryptedKey xmlns="`+xmlenc1.NamespaceXMLEnc+`"><xenc:CipherData><xenc:CipherValue>AA==</xenc:CipherValue></xenc:CipherData></xenc:EncryptedKey>`+
			`<xenc:EncryptedKey xmlns="`+xmlenc1.NamespaceXMLEnc+`"><xenc:CipherData><xenc:CipherValue>AA==</xenc:CipherValue></xenc:CipherData></xenc:EncryptedKey>`+
			`<xenc:EncryptedKey xmlns="`+xmlenc1.NamespaceXMLEnc+`"><xenc:CipherData><xenc:CipherValue>AA==</xenc:CipherValue></xenc:CipherData></xenc:EncryptedKey>`+
			`</ds:KeyInfo>`+
			`<xenc:CipherData><xenc:CipherValue>AA==</xenc:CipherValue></xenc:CipherData>`+
			`</xenc:EncryptedData>`)
		_, err := xmlenc1.NewDecryptor().MaxEncryptedKeys(2).Decrypt(t.Context(), doc.DocumentElement())
		require.ErrorIs(t, err, xmlenc1.ErrTooManyEncryptedKeys)
	})

	t.Run("candidate cap wins before byte budget", func(t *testing.T) {
		elem := manyKeyEncryptedData(t, 2, xmlenc1.DefaultMaxEncryptedKeyBytes)
		_, err := xmlenc1.NewDecryptor().PrivateKey(generateRSAKey(t)).
			MaxEncryptedKeys(1).
			MaxEncryptedKeyBytes(xmlenc1.DefaultMaxEncryptedKeyBytes).
			Decrypt(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrTooManyEncryptedKeys)
		require.NotErrorIs(t, err, xmlenc1.ErrEncryptedKeyBytesExceeded)
	})

	t.Run("negative cap removes the limit", func(t *testing.T) {
		elem := manyKeyEncryptedData(t, xmlenc1.DefaultMaxEncryptedKeys+5, rsaWrappedKeyBytes)
		_, err := xmlenc1.NewDecryptor().PrivateKey(generateRSAKey(t)).MaxEncryptedKeys(-1).Decrypt(t.Context(), elem)
		require.Error(t, err)
		require.NotErrorIs(t, err, xmlenc1.ErrTooManyEncryptedKeys)
	})

	t.Run("normal document still decrypts", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, samlAssertion)
		encryptor := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES256GCM).
			KeyTransportAlgorithm(xmlenc1.RSAOAEP11).
			OAEPDigest(xmlenc1.DigestSHA256).
			OAEPMGF(xmlenc1.MGFSHA256).
			RecipientPublicKey(&key.PublicKey)
		edElem, err := encryptor.EncryptElement(t.Context(), doc.DocumentElement())
		require.NoError(t, err)

		nodes, err := xmlenc1.NewDecryptor().PrivateKey(key).Decrypt(t.Context(), edElem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
	})

	// The cap's cost model is per-mechanism, not RSA-only: an AES key-wrap
	// candidate resolves through aesKeyUnwrap with no RSA key configured at
	// all, and a candidate whose algorithm needs a key the caller never
	// supplied stops at ErrMissingKey before doing any crypto.
	t.Run("candidate cost follows the declared algorithm", func(t *testing.T) {
		t.Run("AES key wrap needs no RSA key", func(t *testing.T) {
			kek := randKey(t, 32)
			doc := mustParseXML(t, `<root><a>secret</a></root>`)
			edElem, err := xmlenc1.NewEncryptor().
				BlockAlgorithm(xmlenc1.AES256GCM).
				KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
				KeyEncryptionKey(kek).
				EncryptElement(t.Context(), doc.DocumentElement())
			require.NoError(t, err)

			nodes, err := xmlenc1.NewDecryptor().KeyEncryptionKey(kek).Decrypt(t.Context(), edElem)
			require.NoError(t, err)
			require.Len(t, nodes, 1)
		})

		t.Run("unconfigured key yields ErrMissingKey", func(t *testing.T) {
			// RSA-OAEP candidates only; the Decryptor carries a KEK, which
			// no candidate declares, so none reaches a crypto operation.
			elem := manyKeyEncryptedData(t, 2, rsaWrappedKeyBytes)
			_, err := xmlenc1.NewDecryptor().KeyEncryptionKey(randKey(t, 32)).Decrypt(t.Context(), elem)
			require.ErrorIs(t, err, xmlenc1.ErrMissingKey)
		})
	})

	// The cap guards the candidate list itself, so it is applied before the
	// Decryptor's keys are consulted. A pre-shared SessionKey skips candidate
	// selection and key resolution, but not the cap.
	t.Run("cap applies with a pre-shared session key", func(t *testing.T) {
		t.Run("over default cap fails", func(t *testing.T) {
			sessionKey := randKey(t, 32)
			elem := manyKeySessionKeyEncryptedData(t, xmlenc1.DefaultMaxEncryptedKeys+1, rsaWrappedKeyBytes, sessionKey, `<x>secret</x>`)
			_, err := xmlenc1.NewDecryptor().SessionKey(sessionKey).Decrypt(t.Context(), elem)
			require.ErrorIs(t, err, xmlenc1.ErrTooManyEncryptedKeys)
		})

		t.Run("explicit cap rejects above it", func(t *testing.T) {
			sessionKey := randKey(t, 32)
			elem := manyKeySessionKeyEncryptedData(t, 2, rsaWrappedKeyBytes, sessionKey, `<x>secret</x>`)
			_, err := xmlenc1.NewDecryptor().SessionKey(sessionKey).MaxEncryptedKeys(1).Decrypt(t.Context(), elem)
			require.ErrorIs(t, err, xmlenc1.ErrTooManyEncryptedKeys)
		})

		t.Run("at default cap decrypts through the session key", func(t *testing.T) {
			sessionKey := randKey(t, 32)
			elem := manyKeySessionKeyEncryptedData(t, xmlenc1.DefaultMaxEncryptedKeys, rsaWrappedKeyBytes, sessionKey, `<x>secret</x>`)
			// The Decryptor holds no RSA key, and every candidate declares
			// RSA-OAEP with a junk CipherValue, so success means no candidate
			// was selected or resolved.
			nodes, err := xmlenc1.NewDecryptor().SessionKey(sessionKey).Decrypt(t.Context(), elem)
			require.NoError(t, err)
			require.Len(t, nodes, 1)
			s, err := helium.WriteString(nodes[0])
			require.NoError(t, err)
			require.Contains(t, s, "secret")
		})

		t.Run("negative cap removes the limit", func(t *testing.T) {
			sessionKey := randKey(t, 32)
			elem := manyKeySessionKeyEncryptedData(t, xmlenc1.DefaultMaxEncryptedKeys+5, rsaWrappedKeyBytes, sessionKey, `<x>secret</x>`)
			nodes, err := xmlenc1.NewDecryptor().SessionKey(sessionKey).MaxEncryptedKeys(-1).Decrypt(t.Context(), elem)
			require.NoError(t, err)
			require.Len(t, nodes, 1)
		})

		t.Run("DecryptBytes applies the same cap", func(t *testing.T) {
			sessionKey := randKey(t, 32)
			over := manyKeySessionKeyEncryptedData(t, xmlenc1.DefaultMaxEncryptedKeys+1, rsaWrappedKeyBytes, sessionKey, `payload`)
			_, err := xmlenc1.NewDecryptor().SessionKey(sessionKey).DecryptBytes(t.Context(), over)
			require.ErrorIs(t, err, xmlenc1.ErrTooManyEncryptedKeys)

			within := manyKeySessionKeyEncryptedData(t, xmlenc1.DefaultMaxEncryptedKeys, rsaWrappedKeyBytes, sessionKey, `payload`)
			plaintext, err := xmlenc1.NewDecryptor().SessionKey(sessionKey).DecryptBytes(t.Context(), within)
			require.NoError(t, err)
			require.Equal(t, []byte(`payload`), plaintext)
		})
	})

	// The two caps are independent: a document within the candidate count can
	// still blow the byte budget, and vice versa.
	t.Run("byte budget is not the candidate cap", func(t *testing.T) {
		elem := manyKeyEncryptedData(t, 2, xmlenc1.DefaultMaxEncryptedKeyBytes)
		_, err := xmlenc1.NewDecryptor().PrivateKey(generateRSAKey(t)).Decrypt(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrEncryptedKeyBytesExceeded)
		require.NotErrorIs(t, err, xmlenc1.ErrTooManyEncryptedKeys)
	})

	// A candidate a ds:RetrievalMethod supplies costs what an inline one
	// costs, so the cap counts both alike.
	t.Run("reference candidates count against the cap", func(t *testing.T) {
		sessionKey := randKey(t, 32)
		kek := randKey(t, 32)
		targets := `<ds:KeyInfo>` +
			wrappedKeyXML(t, "a", kek, randKey(t, 32), "") +
			wrappedKeyXML(t, "b", kek, randKey(t, 32), "") +
			wrappedKeyXML(t, "c", kek, sessionKey, "") +
			`</ds:KeyInfo>`
		refs := retrievalMethodXML(xmlenc1.TypeEncryptedKey, "#a") +
			retrievalMethodXML(xmlenc1.TypeEncryptedKey, "#b") +
			retrievalMethodXML(xmlenc1.TypeEncryptedKey, "#c")

		_, err := xmlenc1.NewDecryptor().KeyEncryptionKey(kek).MaxEncryptedKeys(2).
			Decrypt(t.Context(), retrievalDoc(t, sessionKey, refs, targets))
		require.ErrorIs(t, err, xmlenc1.ErrTooManyEncryptedKeys)

		nodes, err := xmlenc1.NewDecryptor().KeyEncryptionKey(kek).MaxEncryptedKeys(3).
			Decrypt(t.Context(), retrievalDoc(t, sessionKey, refs, targets))
		require.NoError(t, err)
		require.Len(t, nodes, 1)
	})

	// The slot is charged at the moment a candidate is RETAINED, which is
	// after the URI is looked up and after the visited-set dedup. A reference
	// that retains a candidate charges one slot; a reference that retains
	// nothing charges none, so its own lookup outcome is what a decrypt
	// reports.
	t.Run("the cap counts retained candidates, not references examined", func(t *testing.T) {
		// A reference naming a DISTINCT second EncryptedKey retains a second
		// candidate, so a cap of one refuses the document. That target's
		// CipherValue is far over the byte budget and the count is still what
		// answers, which pins that the count check precedes parseEncryptedKey.
		t.Run("a distinct second target charges a slot", func(t *testing.T) {
			sessionKey := randKey(t, 32)
			kek := randKey(t, 32)
			keyInfo := wrappedKeyXML(t, "", kek, sessionKey, "") +
				retrievalMethodXML(xmlenc1.TypeEncryptedKey, "#k")
			// 128 KiB of base64 decodes to 96 KiB, over the 64 KiB default.
			oversized := `<xenc:EncryptedKey Id="k">` +
				`<xenc:EncryptionMethod Algorithm="` + xmlenc1.RSAOAEP11 + `"/>` +
				`<xenc:CipherData><xenc:CipherValue>` + strings.Repeat("A", 128<<10) + `</xenc:CipherValue></xenc:CipherData>` +
				`</xenc:EncryptedKey>`

			_, err := xmlenc1.NewDecryptor().KeyEncryptionKey(kek).MaxEncryptedKeys(1).
				Decrypt(t.Context(), retrievalDoc(t, sessionKey, keyInfo, oversized))
			require.ErrorIs(t, err, xmlenc1.ErrTooManyEncryptedKeys)
			require.NotErrorIs(t, err, xmlenc1.ErrEncryptedKeyBytesExceeded)

			// Raising the cap lets the same document reach the ciphertext,
			// which is what makes the assertion above about ordering rather
			// than about a budget that was never going to be charged.
			_, err = xmlenc1.NewDecryptor().KeyEncryptionKey(kek).MaxEncryptedKeys(2).
				Decrypt(t.Context(), retrievalDoc(t, sessionKey, keyInfo, oversized))
			require.ErrorIs(t, err, xmlenc1.ErrEncryptedKeyBytesExceeded)
		})

		// Each reference below names something whose lookup has its own
		// distinctive error and retains nothing, so the lookup is what
		// answers even with the cap already filled by the inline key.
		t.Run("a reference that retains nothing is not charged", func(t *testing.T) {
			for _, tc := range []struct {
				name     string
				uri      string
				resolved error
				trailing string
			}{
				{
					name:     "against a duplicate id",
					uri:      "#k",
					resolved: xmlenc1.ErrAmbiguousReference,
					trailing: `<a Id="k"/><b Id="k"/>`,
				},
				{
					name:     "against a missing id",
					uri:      "#k",
					resolved: xmlenc1.ErrReferenceNotFound,
					trailing: "",
				},
				{
					name:     "against an external URI",
					uri:      "https://example.com/keys.xml#k",
					resolved: xmlenc1.ErrReferenceNotFound,
					trailing: "",
				},
			} {
				t.Run(tc.name, func(t *testing.T) {
					sessionKey := randKey(t, 32)
					kek := randKey(t, 32)
					keyInfo := wrappedKeyXML(t, "", kek, sessionKey, "") +
						retrievalMethodXML(xmlenc1.TypeEncryptedKey, tc.uri)
					elem := retrievalDoc(t, sessionKey, keyInfo, tc.trailing)

					_, err := xmlenc1.NewDecryptor().KeyEncryptionKey(kek).MaxEncryptedKeys(1).Decrypt(t.Context(), elem)
					require.ErrorIs(t, err, tc.resolved)
					require.NotErrorIs(t, err, xmlenc1.ErrTooManyEncryptedKeys)
				})
			}
		})
	})

	t.Run("cancelled context aborts the candidate loop", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, samlAssertion)
		encryptor := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES256GCM).
			KeyTransportAlgorithm(xmlenc1.RSAOAEP11).
			OAEPDigest(xmlenc1.DigestSHA256).
			OAEPMGF(xmlenc1.MGFSHA256).
			RecipientPublicKey(&key.PublicKey)
		edElem, err := encryptor.EncryptElement(t.Context(), doc.DocumentElement())
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		_, err = xmlenc1.NewDecryptor().PrivateKey(key).Decrypt(ctx, edElem)
		require.ErrorIs(t, err, context.Canceled)
	})
}

// TestMaxEncryptedKeyBytes covers the cumulative EncryptedKey ciphertext
// budget: it is charged while the document is read, so it holds ahead of the
// candidate loop and the pre-shared session-key early return alike.
func TestMaxEncryptedKeyBytes(t *testing.T) {
	t.Run("over default budget fails", func(t *testing.T) {
		elem := manyKeyEncryptedData(t, 1, xmlenc1.DefaultMaxEncryptedKeyBytes+1)
		_, err := xmlenc1.NewDecryptor().PrivateKey(generateRSAKey(t)).Decrypt(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrEncryptedKeyBytesExceeded)
	})

	t.Run("at default budget is not rejected by the budget", func(t *testing.T) {
		elem := manyKeyEncryptedData(t, 1, xmlenc1.DefaultMaxEncryptedKeyBytes)
		_, err := xmlenc1.NewDecryptor().PrivateKey(generateRSAKey(t)).Decrypt(t.Context(), elem)
		require.Error(t, err)
		require.NotErrorIs(t, err, xmlenc1.ErrEncryptedKeyBytesExceeded)
	})

	// The budget is the total across the document, so candidates that each
	// fit still fail together.
	t.Run("budget is cumulative across candidates", func(t *testing.T) {
		half := xmlenc1.DefaultMaxEncryptedKeyBytes/2 + 1
		elem := manyKeyEncryptedData(t, 2, half)
		_, err := xmlenc1.NewDecryptor().PrivateKey(generateRSAKey(t)).Decrypt(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrEncryptedKeyBytesExceeded)
	})

	t.Run("explicit budget rejects above it", func(t *testing.T) {
		elem := manyKeyEncryptedData(t, 2, 512)
		_, err := xmlenc1.NewDecryptor().PrivateKey(generateRSAKey(t)).MaxEncryptedKeyBytes(1023).Decrypt(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrEncryptedKeyBytesExceeded)
	})

	t.Run("explicit budget accepts at it", func(t *testing.T) {
		elem := manyKeyEncryptedData(t, 2, 512)
		_, err := xmlenc1.NewDecryptor().PrivateKey(generateRSAKey(t)).MaxEncryptedKeyBytes(1024).Decrypt(t.Context(), elem)
		require.Error(t, err)
		require.NotErrorIs(t, err, xmlenc1.ErrEncryptedKeyBytesExceeded)
	})

	t.Run("negative budget removes the limit", func(t *testing.T) {
		elem := manyKeyEncryptedData(t, 1, xmlenc1.DefaultMaxEncryptedKeyBytes+1)
		_, err := xmlenc1.NewDecryptor().PrivateKey(generateRSAKey(t)).MaxEncryptedKeyBytes(-1).Decrypt(t.Context(), elem)
		require.Error(t, err)
		require.NotErrorIs(t, err, xmlenc1.ErrEncryptedKeyBytesExceeded)
	})

	t.Run("zero is the default", func(t *testing.T) {
		// An explicit DefaultMaxEncryptedKeyBytes and an unset budget accept
		// and reject the same documents.
		within := manyKeyEncryptedData(t, 1, xmlenc1.DefaultMaxEncryptedKeyBytes)
		over := manyKeyEncryptedData(t, 1, xmlenc1.DefaultMaxEncryptedKeyBytes+1)
		for _, dec := range []xmlenc1.Decryptor{
			xmlenc1.NewDecryptor().PrivateKey(generateRSAKey(t)),
			xmlenc1.NewDecryptor().PrivateKey(generateRSAKey(t)).MaxEncryptedKeyBytes(0),
			xmlenc1.NewDecryptor().PrivateKey(generateRSAKey(t)).MaxEncryptedKeyBytes(xmlenc1.DefaultMaxEncryptedKeyBytes),
		} {
			_, err := dec.Decrypt(t.Context(), within)
			require.Error(t, err)
			require.NotErrorIs(t, err, xmlenc1.ErrEncryptedKeyBytesExceeded)

			_, err = dec.Decrypt(t.Context(), over)
			require.ErrorIs(t, err, xmlenc1.ErrEncryptedKeyBytesExceeded)
		}
	})

	t.Run("normal document still decrypts", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, samlAssertion)
		edElem, err := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES256GCM).
			KeyTransportAlgorithm(xmlenc1.RSAOAEP11).
			OAEPDigest(xmlenc1.DigestSHA256).
			OAEPMGF(xmlenc1.MGFSHA256).
			RecipientPublicKey(&key.PublicKey).
			EncryptElement(t.Context(), doc.DocumentElement())
		require.NoError(t, err)

		nodes, err := xmlenc1.NewDecryptor().PrivateKey(key).Decrypt(t.Context(), edElem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
	})

	// The budget is spent while the document is read, so it applies whatever
	// key the caller configured — including a pre-shared session key, which
	// returns before any candidate is selected.
	t.Run("budget applies with a pre-shared session key", func(t *testing.T) {
		t.Run("over default budget fails", func(t *testing.T) {
			sessionKey := randKey(t, 32)
			elem := manyKeySessionKeyEncryptedData(t, 1, xmlenc1.DefaultMaxEncryptedKeyBytes+1, sessionKey, `<x>secret</x>`)
			_, err := xmlenc1.NewDecryptor().SessionKey(sessionKey).Decrypt(t.Context(), elem)
			require.ErrorIs(t, err, xmlenc1.ErrEncryptedKeyBytesExceeded)
		})

		t.Run("within budget decrypts", func(t *testing.T) {
			sessionKey := randKey(t, 32)
			elem := manyKeySessionKeyEncryptedData(t, 1, xmlenc1.DefaultMaxEncryptedKeyBytes, sessionKey, `<x>secret</x>`)
			nodes, err := xmlenc1.NewDecryptor().SessionKey(sessionKey).Decrypt(t.Context(), elem)
			require.NoError(t, err)
			require.Len(t, nodes, 1)
			s, err := helium.WriteString(nodes[0])
			require.NoError(t, err)
			require.Contains(t, s, "secret")
		})
	})

	t.Run("DecryptBytes applies the same budget", func(t *testing.T) {
		sessionKey := randKey(t, 32)
		over := manyKeySessionKeyEncryptedData(t, 1, xmlenc1.DefaultMaxEncryptedKeyBytes+1, sessionKey, `payload`)
		_, err := xmlenc1.NewDecryptor().SessionKey(sessionKey).DecryptBytes(t.Context(), over)
		require.ErrorIs(t, err, xmlenc1.ErrEncryptedKeyBytesExceeded)

		within := manyKeySessionKeyEncryptedData(t, 1, xmlenc1.DefaultMaxEncryptedKeyBytes, sessionKey, `payload`)
		plaintext, err := xmlenc1.NewDecryptor().SessionKey(sessionKey).DecryptBytes(t.Context(), within)
		require.NoError(t, err)
		require.Equal(t, []byte(`payload`), plaintext)
	})

	// The budget must hold for malformed base64 too. The decoder sizes its
	// output buffer from the character count and allocates it before
	// validating anything, so a CipherValue whose padding cannot be trusted
	// has to be charged the full quantum count; charging a padding-adjusted
	// length would let an oversized value pass the budget and be allocated
	// anyway, only to fail as invalid base64.
	t.Run("malformed CipherValue is charged before the decode", func(t *testing.T) {
		// Each value is 128 KiB of characters, so the decoder would allocate
		// 96 KiB for it — over the 64 KiB default budget.
		const chars = 128 << 10
		for _, tc := range []struct {
			name        string
			cipherValue string
		}{
			// Deducting one byte per '=' would charge this 0.
			{name: "all padding", cipherValue: strings.Repeat("=", chars)},
			// Two of every four characters are padding, so deducting per '='
			// would charge 32 KiB — under budget, yet 96 KiB is allocated.
			{name: "padding in every quantum", cipherValue: strings.Repeat("AA==", chars/4)},
		} {
			t.Run(tc.name, func(t *testing.T) {
				elem := rawKeyCipherValueEncryptedData(t, tc.cipherValue)
				_, err := xmlenc1.NewDecryptor().PrivateKey(generateRSAKey(t)).Decrypt(t.Context(), elem)
				require.ErrorIs(t, err, xmlenc1.ErrEncryptedKeyBytesExceeded)
			})
		}
	})

	// Well-formed trailing padding on a body outside the base64 alphabet: the
	// padding alone says two bytes may be deducted, but the decoder refuses the
	// value and allocates the whole quantum, so the charge is the quantum. The
	// budget here is one byte, which the padding-adjusted count would fit.
	t.Run("junk CipherValue with well-formed padding is charged the quantum", func(t *testing.T) {
		for _, tc := range []struct {
			name        string
			cipherValue string
		}{
			{name: "junk body", cipherValue: `!!==`},
			{name: "part-alphabet body", cipherValue: `A!==`},
			{name: "junk body line-wrapped", cipherValue: "! \t!\r\n=="},
		} {
			t.Run(tc.name, func(t *testing.T) {
				elem := rawKeyCipherValueEncryptedData(t, tc.cipherValue)
				_, err := xmlenc1.NewDecryptor().PrivateKey(generateRSAKey(t)).MaxEncryptedKeyBytes(1).Decrypt(t.Context(), elem)
				require.ErrorIs(t, err, xmlenc1.ErrEncryptedKeyBytesExceeded)
			})
		}
	})

	// A malformed CipherValue that fits the budget is still the decoder's to
	// reject, so the guard above does not turn small junk into a budget error.
	t.Run("malformed CipherValue within budget is rejected by the decode", func(t *testing.T) {
		elem := rawKeyCipherValueEncryptedData(t, strings.Repeat("=", 64))
		_, err := xmlenc1.NewDecryptor().PrivateKey(generateRSAKey(t)).Decrypt(t.Context(), elem)
		require.Error(t, err)
		require.NotErrorIs(t, err, xmlenc1.ErrEncryptedKeyBytesExceeded)
	})

	// The EncryptedData payload uses its own budget, so it decrypts under a
	// small EncryptedKey budget while remaining below the payload default.
	t.Run("payload ciphertext is not charged to the EncryptedKey budget", func(t *testing.T) {
		sessionKey := randKey(t, 32)
		plaintext := `<x>` + strings.Repeat("secret", 40000) + `</x>`
		elem := manyKeySessionKeyEncryptedData(t, 1, rsaWrappedKeyBytes, sessionKey, plaintext)
		nodes, err := xmlenc1.NewDecryptor().SessionKey(sessionKey).MaxEncryptedKeyBytes(rsaWrappedKeyBytes).Decrypt(t.Context(), elem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
	})
}

// TestMaxCipherValueBytes covers the EncryptedData payload budget, which is
// independent from the cumulative budget for EncryptedKey ciphertext.
func TestMaxCipherValueBytes(t *testing.T) {
	// overDefault decodes to one byte more than DefaultMaxCipherValueBytes, so
	// the default budget refuses it and only an unlimited budget gets past it.
	// It is laid out as TWO CDATA sections, which is what makes the fixture
	// affordable: splitIntoCDATA takes a NODE COUNT, while splitIntoNodes takes
	// a character CHUNK SIZE, and a chunk of two characters would turn this
	// value into nearly seven million nodes. The many-node shape is covered by
	// TestPayloadCipherValueSplitAcrossNodes instead. Two sections still split
	// the value across children, and each is under the parser's 10 MiB per-node
	// content cap.
	overDefault := splitIntoCDATA(base64.StdEncoding.EncodeToString(make([]byte, xmlenc1.DefaultMaxCipherValueBytes+1)), 2)

	t.Run("over default budget fails", func(t *testing.T) {
		elem := rawPayloadCipherValueEncryptedData(t, overDefault)
		_, err := xmlenc1.NewDecryptor().SessionKey(randKey(t, 32)).DecryptBytes(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrCipherValueBytesExceeded)
	})

	t.Run("explicit budget applies to both terminals", func(t *testing.T) {
		for _, decrypt := range []func(*helium.Element) error{
			func(elem *helium.Element) error {
				_, err := xmlenc1.NewDecryptor().SessionKey(randKey(t, 32)).MaxCipherValueBytes(2).Decrypt(t.Context(), elem)
				return err
			},
			func(elem *helium.Element) error {
				_, err := xmlenc1.NewDecryptor().SessionKey(randKey(t, 32)).MaxCipherValueBytes(2).DecryptBytes(t.Context(), elem)
				return err
			},
		} {
			err := decrypt(rawPayloadCipherValueEncryptedData(t, `AAAA`))
			require.ErrorIs(t, err, xmlenc1.ErrCipherValueBytesExceeded)
		}
	})

	t.Run("exact budget decrypts", func(t *testing.T) {
		sessionKey := randKey(t, 32)
		elem := manyKeySessionKeyEncryptedData(t, 0, 0, sessionKey, `payload`)
		ed, err := xmlenc1.ParseEncryptedDataForTest(elem)
		require.NoError(t, err)

		plaintext, err := xmlenc1.NewDecryptor().SessionKey(sessionKey).MaxCipherValueBytes(len(ed.CipherValue)).DecryptBytes(t.Context(), elem)
		require.NoError(t, err)
		require.Equal(t, []byte(`payload`), plaintext)
	})

	// The two subtests below both assert on a value OVER the default budget,
	// which is what separates them. A value under it would be accepted by every
	// configuration, so zero and a negative value would be indistinguishable and
	// neither subtest could fail.
	t.Run("negative budget removes the limit", func(t *testing.T) {
		elem := rawPayloadCipherValueEncryptedData(t, overDefault)
		_, err := xmlenc1.NewDecryptor().SessionKey(randKey(t, 32)).MaxCipherValueBytes(-1).DecryptBytes(t.Context(), elem)
		// The value gets past the budget and fails afterwards for an unrelated
		// reason: this fixture declares no EncryptionMethod.
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
		require.NotErrorIs(t, err, xmlenc1.ErrCipherValueBytesExceeded)
	})

	t.Run("zero is the default", func(t *testing.T) {
		elem := rawPayloadCipherValueEncryptedData(t, overDefault)
		_, err := xmlenc1.NewDecryptor().SessionKey(randKey(t, 32)).MaxCipherValueBytes(0).DecryptBytes(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrCipherValueBytesExceeded)
	})
}

func TestPayloadCipherValueSplitAcrossNodes(t *testing.T) {
	const (
		splitNodes   = 1000
		splitDecoded = splitNodes * 3 / 4 * 3
	)

	payload := strings.Repeat(`<![CDATA[AAA]]>`, splitNodes)
	elem := rawPayloadCipherValueEncryptedData(t, payload)
	_, err := xmlenc1.NewDecryptor().SessionKey(randKey(t, 32)).MaxCipherValueBytes(splitDecoded-1).Decrypt(t.Context(), elem)
	require.ErrorIs(t, err, xmlenc1.ErrCipherValueBytesExceeded)
}

// TestPayloadCipherValueElementChild pins that an element child of the PAYLOAD
// CipherValue is refused before the payload budget is charged, and before
// anything underneath it is read.
//
// [helium.Element] answers Content by concatenating its whole descendant
// subtree into a fresh buffer, so a counting loop that asked every child for
// its content would spend the entire subtree before a budget measured in
// DECODED octets could refuse it — and since a subtree of whitespace decodes to
// nothing, the budget would not fire on it at all. The lexical size of that
// subtree is the document author's to choose, so the cost would follow the
// markup, and never the budget.
//
// The error assertions alone cannot see this: the value is refused either way,
// and only what was allocated getting there tells the two apart. This reads the
// process-wide TotalAlloc delta across DecryptBytes, so it must NOT run in
// parallel with anything — a concurrent test's allocations would pollute it.
func TestPayloadCipherValueElementChild(t *testing.T) {
	// no t.Parallel(): isolated so the delta reflects only its own DecryptBytes.

	// The element wraps 800,000 characters of otherwise valid base64. Reading
	// it would cost at least one copy of that, so a bound an order of magnitude
	// below it separates "refused the element" from "read the subtree first".
	const (
		leaves   = 200000
		maxAlloc = 64 << 10
	)

	elem := rawPayloadCipherValueEncryptedData(t, `<junk>`+strings.Repeat(`<![CDATA[AAAA]]>`, leaves)+`</junk>`)
	decryptor := xmlenc1.NewDecryptor().SessionKey(randKey(t, 32)).MaxCipherValueBytes(1)

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	_, err := decryptor.DecryptBytes(t.Context(), elem)
	runtime.ReadMemStats(&after)

	// The classifier's refusal, not the budget's: an element child makes the
	// content invalid xs:base64Binary whatever the budget is, and the budget
	// never got the chance to speak.
	require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
	require.ErrorContains(t, err, "CipherValue holds a child of type")
	require.NotErrorIs(t, err, xmlenc1.ErrCipherValueBytesExceeded)

	allocated := after.TotalAlloc - before.TotalAlloc
	require.Less(t, allocated, uint64(maxAlloc), "refusing an element child of a %d-leaf subtree allocated %d bytes", leaves, allocated)
}

// TestEncryptedKeyCipherValueSplitAcrossNodes covers a CipherValue whose
// characters arrive as several text and CDATA children, which is a shape the
// document author chooses freely. The charge is defined on the whole value, so
// counting each child on its own and adding the results is not an
// approximation of it but a way around it: three characters are not a whole
// base64 quantum, so every "AAA" child counts zero however many there are.
func TestEncryptedKeyCipherValueSplitAcrossNodes(t *testing.T) {
	// 1000 children of "AAA" are 3000 characters, which decode to 2250 bytes.
	const (
		splitNodes   = 1000
		splitDecoded = splitNodes * 3 / 4 * 3
	)
	splitValue := strings.Repeat(`<![CDATA[AAA]]>`, splitNodes)

	t.Run("sub-quantum children are charged their concatenation", func(t *testing.T) {
		elem := rawKeyCipherValueEncryptedData(t, splitValue)
		_, err := xmlenc1.NewDecryptor().PrivateKey(generateRSAKey(t)).MaxEncryptedKeyBytes(splitDecoded-1).Decrypt(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrEncryptedKeyBytesExceeded)
	})

	// The same value one byte of budget higher is not over budget, so the
	// charge is the exact count and not a conservative over-charge.
	t.Run("the charge is exact", func(t *testing.T) {
		elem := rawKeyCipherValueEncryptedData(t, splitValue)
		_, err := xmlenc1.NewDecryptor().PrivateKey(generateRSAKey(t)).MaxEncryptedKeyBytes(splitDecoded).Decrypt(t.Context(), elem)
		require.Error(t, err)
		require.NotErrorIs(t, err, xmlenc1.ErrEncryptedKeyBytesExceeded)
	})

	// And the value the split children spell still decrypts: a quantum, and
	// the padding that ends it, may be cut anywhere.
	t.Run("split value decrypts", func(t *testing.T) {
		kek := randKey(t, 32)
		sessionKey := randKey(t, 32)
		wrapped, err := xmlenc1.AESKeyWrapForTest(kek, sessionKey)
		require.NoError(t, err)

		for _, tc := range []struct {
			name  string
			chunk int
		}{
			// One character per child cuts every quantum three times.
			{name: "one character per child", chunk: 1},
			// Two characters per child cuts the trailing padding in half.
			{name: "two characters per child", chunk: 2},
			{name: "three characters per child", chunk: 3},
			{name: "five characters per child", chunk: 5},
		} {
			t.Run(tc.name, func(t *testing.T) {
				elem := splitKeyCipherValueEncryptedData(t, splitIntoNodes(base64.StdEncoding.EncodeToString(wrapped), tc.chunk), sessionKey, `<x>secret</x>`)
				nodes, err := xmlenc1.NewDecryptor().KeyEncryptionKey(kek).Decrypt(t.Context(), elem)
				require.NoError(t, err)
				require.Len(t, nodes, 1)
			})
		}
	})
}

// TestEncryptedKeyBytesAllocation pins what an EncryptedKey CipherValue may
// ALLOCATE, which the error assertions above cannot see. The budget governs
// decoded bytes, but the lexical text an attacker wraps around them is
// unbounded: xs:base64Binary permits XML whitespace between characters, and
// the value may be spread over as many text and CDATA children as the document
// likes. Joining that text into one string before the budget is charged makes
// the budget an accounting formality — the memory is allocated by the time the
// error is returned — and for a value the budget ACCEPTS it is never refused
// at all, so a test that only checks for the rejection would miss half of it.
//
// Each case reads the process-wide TotalAlloc delta across Decrypt, so these
// subtests must NOT run in parallel: a concurrent test's allocations would
// pollute the delta.
func TestEncryptedKeyBytesAllocation(t *testing.T) {
	// no t.Parallel(): isolated so each delta reflects only its own Decrypt.

	// whitespace is the padding the attacker writes around the value, and
	// every bound below is a multiple of it, because the defect being pinned
	// is exactly a cost that follows the lexical length. Only space and tab
	// are used: an XML parser folds CRLF to LF, which would make the text the
	// DOM holds shorter than the text written here and every multiple below
	// harder to read.
	const whitespace = 4 << 20
	padding := strings.Repeat(" \t", whitespace/2)

	// Reading each child's content is the floor a bound has to clear: a DOM
	// hands out a copy per node, and no value can be counted without looking
	// at it. A rejected value is looked at once, and an accepted one twice —
	// once to count it and once to build the characters the count approved.
	const (
		countOnly     = whitespace * 3 / 2
		countAndBuild = whitespace * 5 / 2
	)

	// overBudget decodes to one byte more than the default budget allows, so
	// the value is refused however much whitespace surrounds it.
	overBudget := base64.StdEncoding.EncodeToString(make([]byte, xmlenc1.DefaultMaxEncryptedKeyBytes+1))

	for _, tc := range []struct {
		name        string
		cipherValue string
		rejected    bool
		maxAlloc    uint64
	}{
		{
			name:        "whitespace in one text node",
			cipherValue: overBudget + padding,
			rejected:    true,
			maxAlloc:    countOnly,
		},
		{
			// CDATA is how the whitespace evades a per-node content cap: the
			// cap bounds one indivisible run, and every section is its own.
			name:        "whitespace split across CDATA sections",
			cipherValue: overBudget + splitIntoCDATA(padding, 16),
			rejected:    true,
			maxAlloc:    countOnly,
		},
		{
			// Nothing here is ever refused: one quantum of payload is far
			// under budget, and the whitespace after it is not charged at
			// all. Allocating for it would be unbounded amplification with no
			// error anywhere to notice it, which a rejection-only test would
			// never see.
			name:        "under-budget value with trailing whitespace",
			cipherValue: "AA==" + padding,
			rejected:    false,
			maxAlloc:    countAndBuild,
		},
		{
			// The same accepted value spread over CDATA sections, where
			// joining the children costs the most.
			name:        "under-budget value with whitespace split across CDATA sections",
			cipherValue: "AA==" + splitIntoCDATA(padding, 16),
			rejected:    false,
			maxAlloc:    countAndBuild,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			elem := rawKeyCipherValueEncryptedData(t, tc.cipherValue)
			decryptor := xmlenc1.NewDecryptor().PrivateKey(generateRSAKey(t))

			var before, after runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&before)
			_, err := decryptor.Decrypt(t.Context(), elem)
			runtime.ReadMemStats(&after)

			require.Error(t, err)
			require.Equal(t, tc.rejected, errors.Is(err, xmlenc1.ErrEncryptedKeyBytesExceeded), "err=%v", err)

			allocated := after.TotalAlloc - before.TotalAlloc
			require.Less(t, allocated, tc.maxAlloc, "decrypting %d lexical bytes allocated %d bytes", len(tc.cipherValue), allocated)
		})
	}
}

// splitIntoNodes lays out s as CDATA sections of chunk characters each, so the
// value reaches the parser as that many separate children.
func splitIntoNodes(s string, chunk int) string {
	var b strings.Builder
	for i := 0; i < len(s); i += chunk {
		b.WriteString(`<![CDATA[` + s[i:min(i+chunk, len(s))] + `]]>`)
	}
	return b.String()
}

// splitIntoCDATA lays out s as parts CDATA sections of roughly equal size.
func splitIntoCDATA(s string, parts int) string {
	return splitIntoNodes(s, (len(s)+parts-1)/parts)
}

// splitKeyCipherValueEncryptedData builds an EncryptedData whose EncryptedKey
// carries keyCipherValue as raw CipherValue markup — text, CDATA sections, or
// any mix — over sessionKey wrapped with AES-256 key wrap and a real
// AES-256-GCM payload. A decrypt that returns plaintext therefore proves the
// split value was assembled into the wrapped key byte for byte.
func splitKeyCipherValueEncryptedData(t *testing.T, keyCipherValue string, sessionKey []byte, plaintext string) *helium.Element {
	t.Helper()
	cipher, err := xmlenc1.EncryptBytesForTest(xmlenc1.AES256GCM, sessionKey, []byte(plaintext))
	require.NoError(t, err)
	doc := mustParseXML(t, `<xenc:EncryptedData xmlns:xenc="`+xmlenc1.NamespaceXMLEnc+`" xmlns:ds="`+xmlenc1.NamespaceDSig+`" Type="`+xmlenc1.TypeElement+`">`+
		`<xenc:EncryptionMethod Algorithm="`+xmlenc1.AES256GCM+`"/>`+
		`<ds:KeyInfo><xenc:EncryptedKey>`+
		`<xenc:EncryptionMethod Algorithm="`+xmlenc1.AES256KeyWrap+`"/>`+
		`<xenc:CipherData><xenc:CipherValue>`+keyCipherValue+`</xenc:CipherValue></xenc:CipherData>`+
		`</xenc:EncryptedKey></ds:KeyInfo>`+
		`<xenc:CipherData><xenc:CipherValue>`+base64.StdEncoding.EncodeToString(cipher)+`</xenc:CipherValue></xenc:CipherData>`+
		`</xenc:EncryptedData>`)
	return doc.DocumentElement()
}
