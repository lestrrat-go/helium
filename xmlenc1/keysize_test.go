package xmlenc1_test

import (
	"crypto/rand"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmlenc1"
	"github.com/stretchr/testify/require"
)

func randKey(t *testing.T, n int) []byte {
	t.Helper()
	k := make([]byte, n)
	_, err := rand.Read(k)
	require.NoError(t, err)
	return k
}

// The declared block-algorithm URI must be bound to the session-key
// length. Supplying a 16-byte key while declaring an AES-256 algorithm
// would otherwise emit an AES-256 URI but encrypt with AES-128.
func TestEncrypt_BlockKeySizeMismatch(t *testing.T) {
	for _, tc := range []struct {
		name string
		alg  string
		size int // wrong size
	}{
		{"aes256-gcm with 16-byte key", xmlenc1.AES256GCM, 16},
		{"aes128-gcm with 32-byte key", xmlenc1.AES128GCM, 32},
		{"aes256-cbc with 16-byte key", xmlenc1.AES256CBC, 16},
		{"aes128-cbc with 24-byte key", xmlenc1.AES128CBC, 24},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc := mustParseXML(t, `<root><a>hi</a></root>`)
			enc := xmlenc1.NewEncryptor().
				BlockAlgorithm(tc.alg).
				SessionKey(randKey(t, tc.size))
			_, err := enc.EncryptElement(t.Context(), doc.DocumentElement())
			require.Error(t, err)
			var kse *xmlenc1.KeySizeError
			require.ErrorAs(t, err, &kse)
		})
	}
}

// The declared key-wrap URI must be bound to the KEK length on encrypt.
func TestEncrypt_KeyWrapKEKSizeMismatch(t *testing.T) {
	doc := mustParseXML(t, `<root><a>hi</a></root>`)
	enc := xmlenc1.NewEncryptor().
		BlockAlgorithm(xmlenc1.AES256GCM).
		KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
		KeyEncryptionKey(randKey(t, 16)) // AES-128 KEK declared as kw-aes256
	_, err := enc.EncryptElement(t.Context(), doc.DocumentElement())
	require.Error(t, err)
	var kse *xmlenc1.KeySizeError
	require.ErrorAs(t, err, &kse)
}

// Correct sizes must still round-trip via key wrap (GCM, no opt-in needed).
func TestKeyWrap_CorrectSizeRoundTrip(t *testing.T) {
	kek := randKey(t, 32)
	doc := mustParseXML(t, samlAssertion)
	enc := xmlenc1.NewEncryptor().
		BlockAlgorithm(xmlenc1.AES256GCM).
		KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
		KeyEncryptionKey(kek)
	edElem, err := enc.EncryptElement(t.Context(), doc.DocumentElement())
	require.NoError(t, err)

	dec := xmlenc1.NewDecryptor().KeyEncryptionKey(kek)
	nodes, err := dec.Decrypt(t.Context(), edElem)
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	s, err := helium.WriteString(nodes[0])
	require.NoError(t, err)
	require.Contains(t, s, "user@example.com")
}

// Decrypt path: a direct SessionKey whose length does not match the
// declared block algorithm must be rejected.
func TestDecrypt_SessionKeySizeMismatch(t *testing.T) {
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
}

// Decrypt path: a wrong-length KEK against the declared key-wrap URI must
// be rejected before unwrap.
func TestDecrypt_KeyWrapKEKSizeMismatch(t *testing.T) {
	kek := randKey(t, 32)
	doc := mustParseXML(t, samlAssertion)
	enc := xmlenc1.NewEncryptor().
		BlockAlgorithm(xmlenc1.AES256GCM).
		KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
		KeyEncryptionKey(kek)
	edElem, err := enc.EncryptElement(t.Context(), doc.DocumentElement())
	require.NoError(t, err)

	// kw-aes256 was declared on the wire; supply a 16-byte KEK.
	dec := xmlenc1.NewDecryptor().KeyEncryptionKey(randKey(t, 16))
	_, err = dec.Decrypt(t.Context(), edElem)
	require.Error(t, err)
	var kse *xmlenc1.KeySizeError
	require.ErrorAs(t, err, &kse)
}

// Decrypt path: an unwrapped session key whose length does not match the
// data-encryption algorithm must be rejected after unwrap. We craft an
// EncryptedData that declares AES-256-GCM but whose EncryptedKey wraps a
// 16-byte session key under a correctly-sized kw-aes128 KEK.
func TestDecrypt_PostUnwrapSessionKeySizeMismatch(t *testing.T) {
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
		EncryptedKey: &xmlenc1.EncryptedKey{
			EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES128KeyWrap},
			CipherValue:      wrapped,
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
}

// Correct sizes must still round-trip via a pre-shared session key for
// every supported block algorithm (CBC requires opt-in on decrypt).
func TestSessionKey_CorrectSizeRoundTrip(t *testing.T) {
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
}
