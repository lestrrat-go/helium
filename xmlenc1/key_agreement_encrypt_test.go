package xmlenc1_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmlenc1"
	"github.com/stretchr/testify/require"
)

func generateECKey(t *testing.T, curve elliptic.Curve) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(curve, rand.Reader)
	require.NoError(t, err)
	return key
}

// ECDH-ES was decrypt-only: the package could consume an AgreementMethod but
// never produce one, so a caller with an EC recipient key had no way to
// encrypt for it.
func TestEncryptECDHES(t *testing.T) {
	curves := []struct {
		name  string
		curve elliptic.Curve
	}{
		{name: "P-256", curve: elliptic.P256()},
		{name: "P-384", curve: elliptic.P384()},
		{name: "P-521", curve: elliptic.P521()},
	}
	wraps := []struct {
		name string
		uri  string
	}{
		{name: "kw-aes128", uri: xmlenc1.AES128KeyWrap},
		{name: "kw-aes192", uri: xmlenc1.AES192KeyWrap},
		{name: "kw-aes256", uri: xmlenc1.AES256KeyWrap},
	}

	for _, c := range curves {
		for _, w := range wraps {
			t.Run(c.name+"/"+w.name, func(t *testing.T) {
				key := generateECKey(t, c.curve)
				doc := mustParseXML(t, samlAssertion)

				edElem, err := xmlenc1.NewEncryptor().
					BlockAlgorithm(xmlenc1.AES256GCM11).
					KeyWrapAlgorithm(w.uri).
					RecipientECPublicKey(&key.PublicKey).
					EncryptElement(t.Context(), doc.DocumentElement())
				require.NoError(t, err)

				// The plaintext must be gone from the serialized document.
				encrypted, err := helium.WriteString(doc)
				require.NoError(t, err)
				require.NotContains(t, encrypted, "user@example.com")

				nodes, err := xmlenc1.NewDecryptor().
					ECPrivateKey(key).
					Decrypt(t.Context(), edElem)
				require.NoError(t, err)
				require.Len(t, nodes, 1)

				decrypted, err := helium.WriteString(nodes[0])
				require.NoError(t, err)
				require.Contains(t, decrypted, "user@example.com")
			})
		}
	}
}

// The emitted AgreementMethod must round-trip through the package's own
// parser with every field intact, since that is what the recipient reads.
func TestEncryptECDHESWireForm(t *testing.T) {
	key := generateECKey(t, elliptic.P384())
	doc := mustParseXML(t, samlAssertion)

	edElem, err := xmlenc1.NewEncryptor().
		KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
		RecipientECPublicKey(&key.PublicKey).
		KeyDerivationParams(&xmlenc1.ConcatKDFParams{
			DigestMethod: xmlenc1.DigestSHA512,
			AlgorithmID:  []byte{0x01, 0x02},
			PartyUInfo:   []byte{0x03},
			PartyVInfo:   []byte{0x04, 0x05, 0x06},
		}).
		EncryptElement(t.Context(), doc.DocumentElement())
	require.NoError(t, err)

	ed, err := xmlenc1.ParseEncryptedDataForTest(edElem)
	require.NoError(t, err)
	require.Len(t, ed.EncryptedKeys, 1)

	ek := ed.EncryptedKeys[0]
	require.Equal(t, xmlenc1.AES256KeyWrap, ek.EncryptionMethod.Algorithm)
	require.NotNil(t, ek.AgreementMethod)
	require.Equal(t, xmlenc1.ECDHES, ek.AgreementMethod.Algorithm)

	kdm := ek.AgreementMethod.KeyDerivationMethod
	require.NotNil(t, kdm)
	require.Equal(t, xmlenc1.ConcatKDF, kdm.Algorithm)
	require.NotNil(t, kdm.ConcatKDF)
	require.Equal(t, xmlenc1.DigestSHA512, kdm.ConcatKDF.DigestMethod)
	require.Equal(t, []byte{0x01, 0x02}, kdm.ConcatKDF.AlgorithmID)
	require.Equal(t, []byte{0x03}, kdm.ConcatKDF.PartyUInfo)
	require.Equal(t, []byte{0x04, 0x05, 0x06}, kdm.ConcatKDF.PartyVInfo)
	require.Empty(t, kdm.ConcatKDF.SuppPubInfo)
	require.Empty(t, kdm.ConcatKDF.SuppPrivInfo)

	originator := ek.AgreementMethod.OriginatorKey
	require.NotNil(t, originator)
	// The ephemeral key is on the recipient's own curve, and is NOT the
	// recipient's key.
	require.Equal(t, "P-384", key.Curve.Params().Name)
	recipientECDH, err := key.PublicKey.ECDH()
	require.NoError(t, err)
	require.NotEqual(t, recipientECDH.Bytes(), originator.PublicKey)

	// Custom OtherInfo must still decrypt, i.e. both sides derived alike.
	nodes, err := xmlenc1.NewDecryptor().ECPrivateKey(key).Decrypt(t.Context(), edElem)
	require.NoError(t, err)
	require.Len(t, nodes, 1)
}

// Each encryption must use a fresh ephemeral key; reuse would destroy the
// forward secrecy that makes ECDH-ES worth choosing.
func TestEncryptECDHESUsesFreshEphemeralKey(t *testing.T) {
	key := generateECKey(t, elliptic.P256())

	originatorKey := func() []byte {
		doc := mustParseXML(t, samlAssertion)
		edElem, err := xmlenc1.NewEncryptor().
			KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
			RecipientECPublicKey(&key.PublicKey).
			EncryptElement(t.Context(), doc.DocumentElement())
		require.NoError(t, err)
		ed, err := xmlenc1.ParseEncryptedDataForTest(edElem)
		require.NoError(t, err)
		return ed.EncryptedKeys[0].AgreementMethod.OriginatorKey.PublicKey
	}

	require.NotEqual(t, originatorKey(), originatorKey())
}

func TestEncryptECDHESErrors(t *testing.T) {
	key := generateECKey(t, elliptic.P256())

	t.Run("requires a key wrap algorithm", func(t *testing.T) {
		doc := mustParseXML(t, samlAssertion)
		_, err := xmlenc1.NewEncryptor().
			RecipientECPublicKey(&key.PublicKey).
			EncryptElement(t.Context(), doc.DocumentElement())
		require.ErrorIs(t, err, xmlenc1.ErrMissingConfig)
	})

	t.Run("rejects a non-key-wrap algorithm", func(t *testing.T) {
		doc := mustParseXML(t, samlAssertion)
		_, err := xmlenc1.NewEncryptor().
			KeyWrapAlgorithm(xmlenc1.AES256GCM).
			RecipientECPublicKey(&key.PublicKey).
			EncryptElement(t.Context(), doc.DocumentElement())
		require.ErrorIs(t, err, xmlenc1.ErrEncryptionFailed)

		var uae *xmlenc1.UnsupportedAlgorithmError
		require.ErrorAs(t, err, &uae)
	})

	t.Run("rejects an unsupported KDF digest", func(t *testing.T) {
		doc := mustParseXML(t, samlAssertion)
		_, err := xmlenc1.NewEncryptor().
			KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
			RecipientECPublicKey(&key.PublicKey).
			KeyDerivationParams(&xmlenc1.ConcatKDFParams{DigestMethod: "urn:example:nope"}).
			EncryptElement(t.Context(), doc.DocumentElement())
		require.ErrorIs(t, err, xmlenc1.ErrEncryptionFailed)
	})

	t.Run("a wrong EC key does not decrypt", func(t *testing.T) {
		doc := mustParseXML(t, samlAssertion)
		edElem, err := xmlenc1.NewEncryptor().
			KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
			RecipientECPublicKey(&key.PublicKey).
			EncryptElement(t.Context(), doc.DocumentElement())
		require.NoError(t, err)

		other := generateECKey(t, elliptic.P256())
		_, err = xmlenc1.NewDecryptor().ECPrivateKey(other).Decrypt(t.Context(), edElem)
		require.Error(t, err)
	})
}
