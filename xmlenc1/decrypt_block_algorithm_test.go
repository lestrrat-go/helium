package xmlenc1_test

import (
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmlenc1"
	"github.com/stretchr/testify/require"
)

// encryptedDataWithoutEncryptionMethod builds an EncryptedData element that
// carries no EncryptionMethod child, which W3C xmlenc-core1 §3.1 permits for a
// recipient that already knows the algorithm. algorithm is what the plaintext
// is actually encrypted under; nothing on the wire names it.
func encryptedDataWithoutEncryptionMethod(t *testing.T, algorithm string, sessionKey, plaintext []byte) *helium.Element {
	t.Helper()
	cipher, err := xmlenc1.EncryptBytesForTest(algorithm, sessionKey, plaintext)
	require.NoError(t, err)

	doc := mustParseXML(t, `<root/>`)
	elem, err := xmlenc1.MarshalEncryptedDataForTest(doc, &xmlenc1.EncryptedData{
		Type:        xmlenc1.TypeElement,
		CipherValue: cipher,
	})
	require.NoError(t, err)
	return elem
}

// An EncryptedData whose EncryptionMethod is absent is decryptable only when
// the caller states the block algorithm out of band, and then only when the
// document does not contradict it. W3C xmlenc-core1 §3.1/§3.2 make the element
// optional and require the recipient to know the algorithm already; §4.4 step 1
// admits obtaining it out of band.
func TestDecryptBlockAlgorithm(t *testing.T) {
	const plaintext = `<x>secret</x>`

	t.Run("absent EncryptionMethod decrypts with the setter", func(t *testing.T) {
		sessionKey := randKey(t, 32)
		elem := encryptedDataWithoutEncryptionMethod(t, xmlenc1.AES256GCM11, sessionKey, []byte(plaintext))

		nodes, err := xmlenc1.NewDecryptor().
			SessionKey(sessionKey).
			BlockAlgorithm(xmlenc1.AES256GCM11).
			Decrypt(t.Context(), elem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)

		s, err := helium.WriteString(nodes[0])
		require.NoError(t, err)
		require.Equal(t, plaintext, s)
	})

	t.Run("absent EncryptionMethod decrypts with the setter through DecryptBytes", func(t *testing.T) {
		sessionKey := randKey(t, 32)
		payload := []byte{0x00, 0x01, 0xff, 'n', 'o', 't', ' ', 'x', 'm', 'l'}
		elem := encryptedDataWithoutEncryptionMethod(t, xmlenc1.AES256GCM11, sessionKey, payload)

		got, err := xmlenc1.NewDecryptor().
			SessionKey(sessionKey).
			BlockAlgorithm(xmlenc1.AES256GCM11).
			DecryptBytes(t.Context(), elem)
		require.NoError(t, err)
		require.Equal(t, payload, got)
	})

	// Without the setter the document says nothing about how to decrypt it, so
	// both terminals must still refuse it and name the setter that would fix it.
	t.Run("absent EncryptionMethod without the setter fails", func(t *testing.T) {
		sessionKey := randKey(t, 32)
		elem := encryptedDataWithoutEncryptionMethod(t, xmlenc1.AES256GCM11, sessionKey, []byte(plaintext))
		decryptor := xmlenc1.NewDecryptor().SessionKey(sessionKey)

		_, err := decryptor.Decrypt(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
		require.Contains(t, err.Error(), "Decryptor.BlockAlgorithm")

		_, err = decryptor.DecryptBytes(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
		require.Contains(t, err.Error(), "Decryptor.BlockAlgorithm")
	})

	// The match is strict: a document that declares a different algorithm than
	// the caller stated out of band is refused rather than allowed to override
	// the caller, which would be an algorithm-confusion surface.
	t.Run("conflicting algorithms fail", func(t *testing.T) {
		sessionKey := randKey(t, 32)
		doc := mustParseXML(t, samlAssertion)
		elem, err := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES256GCM11).
			SessionKey(sessionKey).
			EncryptElement(t.Context(), doc.DocumentElement())
		require.NoError(t, err)

		decryptor := xmlenc1.NewDecryptor().
			SessionKey(sessionKey).
			BlockAlgorithm(xmlenc1.AES128GCM11)

		_, err = decryptor.Decrypt(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrConflictingBlockAlgorithm)
		require.Contains(t, err.Error(), xmlenc1.AES256GCM11)
		require.Contains(t, err.Error(), xmlenc1.AES128GCM11)

		_, err = decryptor.DecryptBytes(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrConflictingBlockAlgorithm)
		require.Contains(t, err.Error(), xmlenc1.AES256GCM11)
		require.Contains(t, err.Error(), xmlenc1.AES128GCM11)
	})

	// A matching pair is not a conflict: it resolves to the one algorithm both
	// sides name.
	t.Run("matching algorithms decrypt", func(t *testing.T) {
		sessionKey := randKey(t, 32)
		doc := mustParseXML(t, samlAssertion)
		elem, err := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES256GCM11).
			SessionKey(sessionKey).
			EncryptElement(t.Context(), doc.DocumentElement())
		require.NoError(t, err)

		nodes, err := xmlenc1.NewDecryptor().
			SessionKey(sessionKey).
			BlockAlgorithm(xmlenc1.AES256GCM11).
			Decrypt(t.Context(), elem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
	})

	// The CBC opt-in gate runs on the RESOLVED algorithm, so an out-of-band CBC
	// URI is gated exactly as a wire-declared one is.
	t.Run("CBC opt-in gate applies to the resolved algorithm", func(t *testing.T) {
		sessionKey := randKey(t, 32)
		elem := encryptedDataWithoutEncryptionMethod(t, xmlenc1.AES256CBC, sessionKey, []byte(plaintext))

		decryptor := xmlenc1.NewDecryptor().
			SessionKey(sessionKey).
			BlockAlgorithm(xmlenc1.AES256CBC)

		_, err := decryptor.Decrypt(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrCBCRequiresOptIn)

		_, err = decryptor.DecryptBytes(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrCBCRequiresOptIn)

		// The gate is the only barrier: with the opt-in the same document
		// decrypts, which is what makes the refusal above the gate's work and
		// not a side effect of the algorithm being absent from the wire.
		nodes, err := decryptor.AllowUnauthenticatedCBC(true).Decrypt(t.Context(), elem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
	})

	// Legacy-namespace GCM binds the algorithm URI into the AEAD additional
	// authenticated data. The resolved URI must feed that same slot, so a
	// legacy ciphertext opens under its own URI and fails under any other.
	t.Run("legacy GCM binds the resolved URI as additional data", func(t *testing.T) {
		sessionKey := randKey(t, 32)
		elem := encryptedDataWithoutEncryptionMethod(t, xmlenc1.AES256GCM, sessionKey, []byte(plaintext))

		nodes, err := xmlenc1.NewDecryptor().
			SessionKey(sessionKey).
			BlockAlgorithm(xmlenc1.AES256GCM).
			Decrypt(t.Context(), elem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)

		// The XML Encryption 1.1 URI names the same cipher and key length but
		// no additional authenticated data, so the tag no longer verifies.
		_, err = xmlenc1.NewDecryptor().
			SessionKey(sessionKey).
			BlockAlgorithm(xmlenc1.AES256GCM11).
			Decrypt(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrDecryptionFailed)
	})

	// The resolved URI is still bound to the real key length, so an out-of-band
	// URI of the wrong strength fails instead of running AES-128 under an
	// AES-256 label.
	t.Run("resolved URI is bound to the session-key length", func(t *testing.T) {
		sessionKey := randKey(t, 16)
		elem := encryptedDataWithoutEncryptionMethod(t, xmlenc1.AES128GCM11, sessionKey, []byte(plaintext))

		_, err := xmlenc1.NewDecryptor().
			SessionKey(sessionKey).
			BlockAlgorithm(xmlenc1.AES256GCM11).
			Decrypt(t.Context(), elem)
		var kse *xmlenc1.KeySizeError
		require.ErrorAs(t, err, &kse)
		require.Equal(t, xmlenc1.AES256GCM11, kse.Algorithm)
		require.Equal(t, 32, kse.Want)
		require.Equal(t, 16, kse.Got)
	})

	// An unset BlockAlgorithm leaves a wire-declared algorithm untouched, and
	// the setter is clone-on-write like every other builder method.
	t.Run("immutability", func(t *testing.T) {
		sessionKey := randKey(t, 32)
		base := xmlenc1.NewDecryptor().SessionKey(sessionKey)
		conflicting := base.BlockAlgorithm(xmlenc1.AES128GCM11)

		doc := mustParseXML(t, samlAssertion)
		elem, err := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES256GCM11).
			SessionKey(sessionKey).
			EncryptElement(t.Context(), doc.DocumentElement())
		require.NoError(t, err)

		nodes, err := base.Decrypt(t.Context(), elem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)

		_, err = conflicting.Decrypt(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrConflictingBlockAlgorithm)
	})
}

// TestDecryptRejectsKeySizeContradiction proves the xenc:KeySize consistency
// check (xmlenc-core1 §3.2, §5.3, §5.6.2.2) runs at parse, before any
// cryptography: an otherwise fully valid RSA-OAEP-protected EncryptedData
// whose own (block-algorithm) EncryptionMethod carries a KeySize
// contradicting its algorithm fails with ErrMalformedEncrypted even when the
// recipient's correct PrivateKey is configured, so the refusal cannot be a
// byproduct of a failed key transport or unwrap.
func TestDecryptRejectsKeySizeContradiction(t *testing.T) {
	key := generateRSAKey(t)
	doc := mustParseXML(t, samlAssertion)

	encryptor := xmlenc1.NewEncryptor().
		BlockAlgorithm(xmlenc1.AES256GCM).
		KeyTransportAlgorithm(xmlenc1.RSAOAEP11).
		OAEPDigest(xmlenc1.DigestSHA256).
		OAEPMGF(xmlenc1.MGFSHA256).
		RecipientPublicKey(&key.PublicKey)

	_, err := encryptor.EncryptElement(t.Context(), doc.DocumentElement())
	require.NoError(t, err)

	xml, err := helium.WriteString(doc)
	require.NoError(t, err)

	// The payload EncryptionMethod (aes256-gcm) is self-closing on the wire;
	// splice in a KeySize that contradicts its implied 256 bits.
	original := `<xenc:EncryptionMethod Algorithm="` + xmlenc1.AES256GCM + `"/>`
	tampered := `<xenc:EncryptionMethod Algorithm="` + xmlenc1.AES256GCM + `"><xenc:KeySize>128</xenc:KeySize></xenc:EncryptionMethod>`
	require.Contains(t, xml, original)
	xml = strings.Replace(xml, original, tampered, 1)

	tdoc := mustParseXML(t, xml)
	edElem := findEncryptedData(t, tdoc.DocumentElement())
	require.NotNil(t, edElem)

	_, err = xmlenc1.NewDecryptor().PrivateKey(key).Decrypt(t.Context(), edElem)
	require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)

	_, err = xmlenc1.NewDecryptor().PrivateKey(key).DecryptBytes(t.Context(), edElem)
	require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
}
