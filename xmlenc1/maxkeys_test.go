package xmlenc1_test

import (
	"context"
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

// malformedKeyCipherValueEncryptedData builds an EncryptedData whose single
// EncryptedKey carries cipherValue as raw CipherValue text. The text is
// written straight into the document rather than marshalled from an
// EncryptedKey, which is the only way to put base64 the decoder rejects in
// front of the byte budget.
func malformedKeyCipherValueEncryptedData(t *testing.T, cipherValue string) *helium.Element {
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
				elem := malformedKeyCipherValueEncryptedData(t, tc.cipherValue)
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
				elem := malformedKeyCipherValueEncryptedData(t, tc.cipherValue)
				_, err := xmlenc1.NewDecryptor().PrivateKey(generateRSAKey(t)).MaxEncryptedKeyBytes(1).Decrypt(t.Context(), elem)
				require.ErrorIs(t, err, xmlenc1.ErrEncryptedKeyBytesExceeded)
			})
		}
	})

	// A malformed CipherValue that fits the budget is still the decoder's to
	// reject, so the guard above does not turn small junk into a budget error.
	t.Run("malformed CipherValue within budget is rejected by the decode", func(t *testing.T) {
		elem := malformedKeyCipherValueEncryptedData(t, strings.Repeat("=", 64))
		_, err := xmlenc1.NewDecryptor().PrivateKey(generateRSAKey(t)).Decrypt(t.Context(), elem)
		require.Error(t, err)
		require.NotErrorIs(t, err, xmlenc1.ErrEncryptedKeyBytesExceeded)
	})

	// The EncryptedData's own CipherValue is the payload, not EncryptedKey
	// ciphertext, so a large one decrypts under a small budget.
	t.Run("payload ciphertext is not charged", func(t *testing.T) {
		sessionKey := randKey(t, 32)
		plaintext := `<x>` + strings.Repeat("secret", 40000) + `</x>`
		elem := manyKeySessionKeyEncryptedData(t, 1, rsaWrappedKeyBytes, sessionKey, plaintext)
		nodes, err := xmlenc1.NewDecryptor().SessionKey(sessionKey).MaxEncryptedKeyBytes(rsaWrappedKeyBytes).Decrypt(t.Context(), elem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
	})
}
