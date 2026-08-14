package xmlenc1_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmlenc1"
	"github.com/stretchr/testify/require"
)

// cbcPaddingPayload is the plaintext every padding-policy case below encrypts.
// Its length is not a multiple of the AES block size, so it always produces a
// partial final block and therefore real padding to disagree about.
const cbcPaddingPayload = `<greeting xmlns="urn:example">hello</greeting>`

// arbitraryPaddedCipherValue builds the CipherValue a conforming third-party
// implementation emits: AES-CBC over a plaintext padded the way xmlenc-core1
// §5.2.1 allows, with the octets ahead of the final length octet holding filler
// chosen independently of the length. This package never writes that shape
// itself, so nothing built through Encryptor can stand in for it.
func arbitraryPaddedCipherValue(t *testing.T, key []byte) []byte {
	t.Helper()
	cipherValue, err := xmlenc1.EncryptCBCArbitraryPaddingForTest(key, []byte(cbcPaddingPayload), 0xAB)
	require.NoError(t, err)
	return cipherValue
}

// encryptedDataFor wraps a CipherValue in an AES-256-CBC EncryptedData element
// owned by doc.
func encryptedDataFor(t *testing.T, doc *helium.Document, cipherValue []byte) *helium.Element {
	t.Helper()
	edElem, err := xmlenc1.MarshalEncryptedDataForTest(doc, &xmlenc1.EncryptedData{
		Type:             xmlenc1.TypeElement,
		EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES256CBC},
		CipherValue:      cipherValue,
	})
	require.NoError(t, err)
	return edElem
}

// TestCBCPaddingPolicy covers the padding rule a decrypt applies, which
// Decryptor.StrictPKCS7Padding selects.
//
// The default is the XML Encryption rule (xmlenc-core1 §5.2.1), under which the
// octets ahead of the final length octet are arbitrary and unread. That is what
// makes ciphertext from other implementations decryptable. The opt-in narrows
// it to PKCS#7, which fixes every one of those octets to the length.
func TestCBCPaddingPolicy(t *testing.T) {
	// A conforming peer's ciphertext decrypts by default, which is the whole
	// point of the default: refusing it would refuse a valid document.
	t.Run("arbitrary padding decrypts by default", func(t *testing.T) {
		key := randKey(t, 32)
		doc := mustParseXML(t, cbcPaddingPayload)
		edElem := encryptedDataFor(t, doc, arbitraryPaddedCipherValue(t, key))

		nodes, err := xmlenc1.NewDecryptor().
			SessionKey(key).
			AllowUnauthenticatedCBC(true).
			Decrypt(t.Context(), edElem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)

		s, err := helium.WriteString(nodes[0])
		require.NoError(t, err)
		require.Contains(t, s, "hello")
	})

	// The same ciphertext under the opt-in, which is the behavior a caller
	// asks for when it controls both ends.
	t.Run("arbitrary padding is refused under the opt-in", func(t *testing.T) {
		key := randKey(t, 32)
		doc := mustParseXML(t, cbcPaddingPayload)
		edElem := encryptedDataFor(t, doc, arbitraryPaddedCipherValue(t, key))

		_, err := xmlenc1.NewDecryptor().
			SessionKey(key).
			AllowUnauthenticatedCBC(true).
			StrictPKCS7Padding(true).
			Decrypt(t.Context(), edElem)
		require.Error(t, err)
		require.ErrorIs(t, err, xmlenc1.ErrDecryptionFailed)
	})

	// Turning the opt-in explicitly off is the same as leaving it unset, so a
	// caller threading a configuration flag through gets exactly the
	// documented default.
	t.Run("the opt-in set to false matches the default", func(t *testing.T) {
		key := randKey(t, 32)
		doc := mustParseXML(t, cbcPaddingPayload)
		edElem := encryptedDataFor(t, doc, arbitraryPaddedCipherValue(t, key))

		nodes, err := xmlenc1.NewDecryptor().
			SessionKey(key).
			AllowUnauthenticatedCBC(true).
			StrictPKCS7Padding(false).
			Decrypt(t.Context(), edElem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
	})

	// This package pads PKCS#7-style, which both rules accept, so the opt-in
	// never rejects a document it produced. A caller can turn it on without
	// breaking its own round trip.
	t.Run("this package round-trips under both rules", func(t *testing.T) {
		for _, strict := range []bool{false, true} {
			key := randKey(t, 32)
			doc := mustParseXML(t, cbcPaddingPayload)

			edElem, err := xmlenc1.NewEncryptor().
				BlockAlgorithm(xmlenc1.AES256CBC).
				AllowLegacyCBC(true).
				SessionKey(key).
				EncryptElement(t.Context(), doc.DocumentElement())
			require.NoError(t, err)

			nodes, err := xmlenc1.NewDecryptor().
				SessionKey(key).
				AllowUnauthenticatedCBC(true).
				StrictPKCS7Padding(strict).
				Decrypt(t.Context(), edElem)
			require.NoError(t, err, "strict=%v must accept this package's own padding", strict)
			require.Len(t, nodes, 1)
		}
	})

	// The padding rule governs CBC alone. AES-GCM does not pad, so the opt-in
	// must not change anything about a GCM decrypt.
	t.Run("the opt-in does not affect GCM", func(t *testing.T) {
		key := randKey(t, 32)
		doc := mustParseXML(t, cbcPaddingPayload)

		edElem, err := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES256GCM11).
			SessionKey(key).
			EncryptElement(t.Context(), doc.DocumentElement())
		require.NoError(t, err)

		nodes, err := xmlenc1.NewDecryptor().
			SessionKey(key).
			StrictPKCS7Padding(true).
			Decrypt(t.Context(), edElem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
	})

	// DecryptBytes shares the block decryption, so the rule it applies must be
	// the one Decrypt applies.
	t.Run("DecryptBytes applies the same rule", func(t *testing.T) {
		const opaque = "not xml at all"

		key := randKey(t, 32)
		doc := mustParseXML(t, cbcPaddingPayload)
		cipherValue, err := xmlenc1.EncryptCBCArbitraryPaddingForTest(key, []byte(opaque), 0xAB)
		require.NoError(t, err)

		edElem, err := xmlenc1.MarshalEncryptedDataForTest(doc, &xmlenc1.EncryptedData{
			EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES256CBC},
			CipherValue:      cipherValue,
		})
		require.NoError(t, err)

		got, err := xmlenc1.NewDecryptor().
			SessionKey(key).
			AllowUnauthenticatedCBC(true).
			DecryptBytes(t.Context(), edElem)
		require.NoError(t, err)
		require.Equal(t, opaque, string(got))

		_, err = xmlenc1.NewDecryptor().
			SessionKey(key).
			AllowUnauthenticatedCBC(true).
			StrictPKCS7Padding(true).
			DecryptBytes(t.Context(), edElem)
		require.Error(t, err)
		require.ErrorIs(t, err, xmlenc1.ErrDecryptionFailed)
	})

	// A padding failure stays indistinguishable from every other CBC failure.
	// Widening the accepted padding must not have introduced an error a
	// padding oracle could read.
	t.Run("a refused padding reports the generic error", func(t *testing.T) {
		key := randKey(t, 32)
		doc := mustParseXML(t, cbcPaddingPayload)
		edElem := encryptedDataFor(t, doc, arbitraryPaddedCipherValue(t, key))

		_, strictErr := xmlenc1.NewDecryptor().
			SessionKey(key).
			AllowUnauthenticatedCBC(true).
			StrictPKCS7Padding(true).
			Decrypt(t.Context(), edElem)
		require.Error(t, strictErr)

		// A wrong key fails for an entirely different reason and must read
		// identically.
		wrongElem := encryptedDataFor(t, doc, arbitraryPaddedCipherValue(t, key))
		_, wrongKeyErr := xmlenc1.NewDecryptor().
			SessionKey(randKey(t, 32)).
			AllowUnauthenticatedCBC(true).
			Decrypt(t.Context(), wrongElem)
		require.Error(t, wrongKeyErr)

		require.Equal(t, wrongKeyErr.Error(), strictErr.Error())
		require.NotErrorIs(t, strictErr, xmlenc1.ErrInvalidPadding)
	})
}
