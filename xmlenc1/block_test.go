package xmlenc1_test

import (
	"errors"
	"testing"

	"github.com/lestrrat-go/helium/xmlenc1"
	"github.com/stretchr/testify/require"
)

func TestKeySize(t *testing.T) {
	t.Run("encrypt block key mismatch", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			alg  string
			size int // wrong size
			cbc  bool
		}{
			{"aes256-gcm with 16-byte key", xmlenc1.AES256GCM, 16, false},
			{"aes128-gcm with 32-byte key", xmlenc1.AES128GCM, 32, false},
			{"aes256-cbc with 16-byte key", xmlenc1.AES256CBC, 16, true},
			{"aes128-cbc with 24-byte key", xmlenc1.AES128CBC, 24, true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				doc := mustParseXML(t, `<root><a>hi</a></root>`)
				enc := xmlenc1.NewEncryptor().
					BlockAlgorithm(tc.alg).
					SessionKey(randKey(t, tc.size))
				// Opt in to CBC so the failure exercises the key-size
				// binding rather than the CBC encryption opt-in gate.
				if tc.cbc {
					enc = enc.AllowLegacyCBC(true)
				}
				_, err := enc.EncryptElement(t.Context(), doc.DocumentElement())
				require.Error(t, err)
				var kse *xmlenc1.KeySizeError
				require.ErrorAs(t, err, &kse)
			})
		}
	})

	t.Run("decrypt session key mismatch", func(t *testing.T) {
		// Encrypt legitimately under AES-256-GCM.
		sessionKey := randKey(t, 32)
		doc := mustParseXML(t, samlAssertion)
		enc := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES256GCM).
			SessionKey(sessionKey)
		edElem, err := enc.EncryptElement(t.Context(), doc.DocumentElement())
		require.NoError(t, err)

		// Decrypt with a wrong-length session key.
		dec := xmlenc1.NewDecryptor().SessionKey(randKey(t, 16))
		_, err = dec.Decrypt(t.Context(), edElem)
		require.Error(t, err)
		var kse *xmlenc1.KeySizeError
		require.ErrorAs(t, err, &kse)
	})

	t.Run("decrypt post-unwrap session key mismatch", func(t *testing.T) {
		kek := randKey(t, 16) // valid AES-128 KEK
		// Wrap a 16-byte session key (valid for AES-128 algorithms) but
		// declare the data algorithm as AES-256-GCM.
		shortSessionKey := randKey(t, 16)

		wrapped, err := xmlenc1.AESKeyWrapForTest(kek, shortSessionKey)
		require.NoError(t, err)

		doc := mustParseXML(t, `<root/>`)
		// Plaintext bytes encrypted under the short key as AES-128-GCM.
		cipher, err := xmlenc1.EncryptBytesForTest(xmlenc1.AES128GCM, shortSessionKey, []byte("<x>secret</x>"))
		require.NoError(t, err)

		ed := &xmlenc1.EncryptedData{
			Type:             xmlenc1.TypeElement,
			EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES256GCM}, // declares 256
			EncryptedKeys: []*xmlenc1.EncryptedKey{
				{
					EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES128KeyWrap},
					CipherValue:      wrapped,
				},
			},
			CipherValue: cipher,
		}
		edElem, err := xmlenc1.MarshalEncryptedDataForTest(doc, ed)
		require.NoError(t, err)

		dec := xmlenc1.NewDecryptor().KeyEncryptionKey(kek)
		_, err = dec.Decrypt(t.Context(), edElem)
		require.Error(t, err)
		var kse *xmlenc1.KeySizeError
		require.ErrorAs(t, err, &kse)
	})

	t.Run("session key correct size round-trip", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			alg  string
			size int
			cbc  bool
		}{
			{"aes128-gcm", xmlenc1.AES128GCM, 16, false},
			{"aes256-gcm", xmlenc1.AES256GCM, 32, false},
			{"aes128-cbc", xmlenc1.AES128CBC, 16, true},
			{"aes256-cbc", xmlenc1.AES256CBC, 32, true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				key := randKey(t, tc.size)
				doc := mustParseXML(t, samlAssertion)
				enc := xmlenc1.NewEncryptor().
					BlockAlgorithm(tc.alg).
					SessionKey(key)
				if tc.cbc {
					enc = enc.AllowLegacyCBC(true)
				}
				edElem, err := enc.EncryptElement(t.Context(), doc.DocumentElement())
				require.NoError(t, err)

				dec := xmlenc1.NewDecryptor().SessionKey(key)
				if tc.cbc {
					dec = dec.AllowUnauthenticatedCBC(true)
				}
				nodes, err := dec.Decrypt(t.Context(), edElem)
				require.NoError(t, err)
				require.Len(t, nodes, 1)
			})
		}
	})
}

// TestKeySizeErrorMessage covers KeySizeError.Error(): a Decryptor handed a
// session key whose length contradicts the declared algorithm must surface a
// KeySizeError with the descriptive message.
func TestKeySizeErrorMessage(t *testing.T) {
	doc := mustParseXML(t, `<root/>`)
	// Build EncryptedData declaring AES-256-GCM but decrypt with a 16-byte
	// session key; validateKeySize rejects with KeySizeError before any
	// ciphertext is touched.
	ed := &xmlenc1.EncryptedData{
		Type:             xmlenc1.TypeElement,
		EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES256GCM},
		CipherValue:      make([]byte, 64),
	}
	elem, err := xmlenc1.MarshalEncryptedDataForTest(doc, ed)
	require.NoError(t, err)

	shortKey := make([]byte, 16)
	decryptor := xmlenc1.NewDecryptor().SessionKey(shortKey)
	_, err = decryptor.Decrypt(t.Context(), elem)
	require.Error(t, err)

	var kse *xmlenc1.KeySizeError
	require.True(t, errors.As(err, &kse))
	require.Equal(t, xmlenc1.AES256GCM, kse.Algorithm)
	require.Equal(t, 32, kse.Want)
	require.Equal(t, 16, kse.Got)
	require.Contains(t, kse.Error(), "requires a 32-byte key, got 16 bytes")
}
