package xmlenc1_test

import (
	"context"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmlenc1"
	"github.com/stretchr/testify/require"
)

// junkRSAEncryptedKeys builds n syntactically valid RSA-OAEP EncryptedKey
// candidates whose CipherValue is junk, so a test can pack a candidate list
// without any of them resolving to a session key.
func junkRSAEncryptedKeys(n int) []*xmlenc1.EncryptedKey {
	keys := make([]*xmlenc1.EncryptedKey, 0, n)
	for range n {
		keys = append(keys, &xmlenc1.EncryptedKey{
			EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.RSAOAEP},
			CipherValue:      make([]byte, 256),
		})
	}
	return keys
}

// manyKeyEncryptedData builds an EncryptedData element carrying n junk RSA
// EncryptedKey candidates, used to exercise the trial-decrypt cap.
func manyKeyEncryptedData(t *testing.T, n int) *helium.Element {
	t.Helper()
	doc := mustParseXML(t, `<root/>`)
	ed := &xmlenc1.EncryptedData{
		Type:             xmlenc1.TypeElement,
		EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES256GCM},
		EncryptedKeys:    junkRSAEncryptedKeys(n),
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
func manyKeySessionKeyEncryptedData(t *testing.T, n int, sessionKey []byte, plaintext string) *helium.Element {
	t.Helper()
	cipher, err := xmlenc1.EncryptBytesForTest(xmlenc1.AES256GCM, sessionKey, []byte(plaintext))
	require.NoError(t, err)
	doc := mustParseXML(t, `<root/>`)
	ed := &xmlenc1.EncryptedData{
		Type:             xmlenc1.TypeElement,
		EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES256GCM},
		EncryptedKeys:    junkRSAEncryptedKeys(n),
		CipherValue:      cipher,
	}
	elem, err := xmlenc1.MarshalEncryptedDataForTest(doc, ed)
	require.NoError(t, err)
	return elem
}

func TestMaxEncryptedKeys(t *testing.T) {
	t.Run("over default cap fails fast", func(t *testing.T) {
		elem := manyKeyEncryptedData(t, xmlenc1.DefaultMaxEncryptedKeys+1)
		_, err := xmlenc1.NewDecryptor().PrivateKey(generateRSAKey(t)).Decrypt(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrTooManyEncryptedKeys)
	})

	t.Run("at default cap is not rejected by the cap", func(t *testing.T) {
		elem := manyKeyEncryptedData(t, xmlenc1.DefaultMaxEncryptedKeys)
		_, err := xmlenc1.NewDecryptor().PrivateKey(generateRSAKey(t)).Decrypt(t.Context(), elem)
		require.Error(t, err)
		require.NotErrorIs(t, err, xmlenc1.ErrTooManyEncryptedKeys)
	})

	t.Run("explicit cap rejects above it", func(t *testing.T) {
		elem := manyKeyEncryptedData(t, 3)
		_, err := xmlenc1.NewDecryptor().PrivateKey(generateRSAKey(t)).MaxEncryptedKeys(2).Decrypt(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrTooManyEncryptedKeys)
	})

	t.Run("negative cap removes the limit", func(t *testing.T) {
		elem := manyKeyEncryptedData(t, xmlenc1.DefaultMaxEncryptedKeys+5)
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
			elem := manyKeyEncryptedData(t, 2)
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
			elem := manyKeySessionKeyEncryptedData(t, xmlenc1.DefaultMaxEncryptedKeys+1, sessionKey, `<x>secret</x>`)
			_, err := xmlenc1.NewDecryptor().SessionKey(sessionKey).Decrypt(t.Context(), elem)
			require.ErrorIs(t, err, xmlenc1.ErrTooManyEncryptedKeys)
		})

		t.Run("explicit cap rejects above it", func(t *testing.T) {
			sessionKey := randKey(t, 32)
			elem := manyKeySessionKeyEncryptedData(t, 2, sessionKey, `<x>secret</x>`)
			_, err := xmlenc1.NewDecryptor().SessionKey(sessionKey).MaxEncryptedKeys(1).Decrypt(t.Context(), elem)
			require.ErrorIs(t, err, xmlenc1.ErrTooManyEncryptedKeys)
		})

		t.Run("at default cap decrypts through the session key", func(t *testing.T) {
			sessionKey := randKey(t, 32)
			elem := manyKeySessionKeyEncryptedData(t, xmlenc1.DefaultMaxEncryptedKeys, sessionKey, `<x>secret</x>`)
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
			elem := manyKeySessionKeyEncryptedData(t, xmlenc1.DefaultMaxEncryptedKeys+5, sessionKey, `<x>secret</x>`)
			nodes, err := xmlenc1.NewDecryptor().SessionKey(sessionKey).MaxEncryptedKeys(-1).Decrypt(t.Context(), elem)
			require.NoError(t, err)
			require.Len(t, nodes, 1)
		})

		t.Run("DecryptBytes applies the same cap", func(t *testing.T) {
			sessionKey := randKey(t, 32)
			over := manyKeySessionKeyEncryptedData(t, xmlenc1.DefaultMaxEncryptedKeys+1, sessionKey, `payload`)
			_, err := xmlenc1.NewDecryptor().SessionKey(sessionKey).DecryptBytes(t.Context(), over)
			require.ErrorIs(t, err, xmlenc1.ErrTooManyEncryptedKeys)

			within := manyKeySessionKeyEncryptedData(t, xmlenc1.DefaultMaxEncryptedKeys, sessionKey, `payload`)
			plaintext, err := xmlenc1.NewDecryptor().SessionKey(sessionKey).DecryptBytes(t.Context(), within)
			require.NoError(t, err)
			require.Equal(t, []byte(`payload`), plaintext)
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
