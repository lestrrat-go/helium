package xmlenc1_test

import (
	"errors"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmlenc1"
	"github.com/stretchr/testify/require"
)

// TestMarshalParseRoundTripAllFields exercises the serialize and parse paths
// for every optional field: EncryptedData ID/Type, an EncryptedKey carrying
// its own ID/Recipient/CarriedKeyName and an EncryptionMethod with
// DigestMethod, MGFAlgorithm and OAEPParams. The marshaled element is
// serialized to bytes, reparsed through the public XML parser, and the
// resulting DOM is fed back through the internal EncryptedData parser so
// both directions are covered honestly via a real round-trip.
func TestMarshalParseRoundTripAllFields(t *testing.T) {
	doc := mustParseXML(t, `<root/>`)

	ed := &xmlenc1.EncryptedData{
		ID:   "ED-1",
		Type: xmlenc1.TypeElement,
		EncryptionMethod: &xmlenc1.EncryptionMethod{
			Algorithm:    xmlenc1.AES256GCM,
			DigestMethod: xmlenc1.DigestSHA256,
			MGFAlgorithm: xmlenc1.MGFSHA256,
			OAEPParams:   []byte("params-bytes"),
		},
		EncryptedKey: &xmlenc1.EncryptedKey{
			ID: "EK-1",
			EncryptionMethod: &xmlenc1.EncryptionMethod{
				Algorithm:    xmlenc1.RSAOAEP11,
				DigestMethod: xmlenc1.DigestSHA256,
				MGFAlgorithm: xmlenc1.MGFSHA256,
			},
			CipherValue: []byte("wrapped-key-bytes"),
		},
		CipherValue: []byte("cipher-bytes"),
	}

	elem, err := xmlenc1.MarshalEncryptedDataForTest(doc, ed)
	require.NoError(t, err)

	// Parse the marshaled DOM back through the internal EncryptedData
	// parser. The marshaler sets active namespaces on each element, so the
	// namespace-aware matcher resolves the xenc/ds/xenc11 URIs directly.
	parsed, err := xmlenc1.ParseEncryptedDataForTest(elem)
	require.NoError(t, err)

	require.Equal(t, "ED-1", parsed.ID)
	require.Equal(t, xmlenc1.TypeElement, parsed.Type)
	require.NotNil(t, parsed.EncryptionMethod)
	require.Equal(t, xmlenc1.AES256GCM, parsed.EncryptionMethod.Algorithm)
	require.Equal(t, xmlenc1.DigestSHA256, parsed.EncryptionMethod.DigestMethod)
	require.Equal(t, xmlenc1.MGFSHA256, parsed.EncryptionMethod.MGFAlgorithm)
	require.Equal(t, []byte("params-bytes"), parsed.EncryptionMethod.OAEPParams)

	require.NotNil(t, parsed.EncryptedKey)
	require.Equal(t, "EK-1", parsed.EncryptedKey.ID)
	require.NotNil(t, parsed.EncryptedKey.EncryptionMethod)
	require.Equal(t, xmlenc1.RSAOAEP11, parsed.EncryptedKey.EncryptionMethod.Algorithm)
	require.Equal(t, []byte("wrapped-key-bytes"), parsed.EncryptedKey.CipherValue)
	require.Equal(t, []byte("cipher-bytes"), parsed.CipherValue)
}

// TestParseEncryptedDataErrors covers the early-rejection branches of the
// internal parser.
func TestParseEncryptedDataErrors(t *testing.T) {
	t.Run("nil element", func(t *testing.T) {
		_, err := xmlenc1.ParseEncryptedDataForTest(nil)
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
	})

	t.Run("wrong element name/namespace", func(t *testing.T) {
		doc := mustParseXML(t, `<root xmlns="http://www.w3.org/2001/04/xmlenc#"><NotEncryptedData/></root>`)
		child, ok := helium.AsNode[*helium.Element](doc.DocumentElement().FirstChild())
		require.True(t, ok)
		_, err := xmlenc1.ParseEncryptedDataForTest(child)
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
	})

	t.Run("missing CipherData", func(t *testing.T) {
		// An EncryptedData with no CipherData/CipherValue must be rejected.
		doc := mustParseXML(t, `<xenc:EncryptedData xmlns:xenc="http://www.w3.org/2001/04/xmlenc#"><xenc:EncryptionMethod Algorithm="`+xmlenc1.AES256GCM+`"/></xenc:EncryptedData>`)
		elem, ok := helium.AsNode[*helium.Element](doc.DocumentElement())
		require.True(t, ok)
		_, err := xmlenc1.ParseEncryptedDataForTest(elem)
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
	})
}

// TestParseEncryptionMethodMissingAlgorithm covers the EncryptionMethod
// missing-@Algorithm rejection inside parseEncryptionMethod, reached through
// the EncryptedData parser.
func TestParseEncryptionMethodMissingAlgorithm(t *testing.T) {
	doc := mustParseXML(t, `<xenc:EncryptedData xmlns:xenc="http://www.w3.org/2001/04/xmlenc#"><xenc:EncryptionMethod/><xenc:CipherData><xenc:CipherValue>AAAA</xenc:CipherValue></xenc:CipherData></xenc:EncryptedData>`)
	elem, ok := helium.AsNode[*helium.Element](doc.DocumentElement())
	require.True(t, ok)
	_, err := xmlenc1.ParseEncryptedDataForTest(elem)
	require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
}

// TestParseInvalidBase64 covers the base64-decode error branches in
// parseCipherData and parseEncryptionMethod (OAEPparams).
func TestParseInvalidBase64(t *testing.T) {
	t.Run("CipherValue", func(t *testing.T) {
		doc := mustParseXML(t, `<xenc:EncryptedData xmlns:xenc="http://www.w3.org/2001/04/xmlenc#"><xenc:CipherData><xenc:CipherValue>!!!not-base64!!!</xenc:CipherValue></xenc:CipherData></xenc:EncryptedData>`)
		elem, ok := helium.AsNode[*helium.Element](doc.DocumentElement())
		require.True(t, ok)
		_, err := xmlenc1.ParseEncryptedDataForTest(elem)
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
	})

	t.Run("OAEPparams", func(t *testing.T) {
		doc := mustParseXML(t, `<xenc:EncryptedData xmlns:xenc="http://www.w3.org/2001/04/xmlenc#"><xenc:EncryptionMethod Algorithm="`+xmlenc1.RSAOAEP11+`"><xenc:OAEPparams>!!!bad!!!</xenc:OAEPparams></xenc:EncryptionMethod><xenc:CipherData><xenc:CipherValue>AAAA</xenc:CipherValue></xenc:CipherData></xenc:EncryptedData>`)
		elem, ok := helium.AsNode[*helium.Element](doc.DocumentElement())
		require.True(t, ok)
		_, err := xmlenc1.ParseEncryptedDataForTest(elem)
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
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
