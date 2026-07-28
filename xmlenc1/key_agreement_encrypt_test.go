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

// Params that omit DigestMethod are unspecified in full: the SHA-256 fallback
// must come with empty OtherInfo, not with the caller's OtherInfo attached to
// a digest the caller never chose.
func TestEncryptECDHESEmptyDigestDropsOtherInfo(t *testing.T) {
	key := generateECKey(t, elliptic.P256())
	doc := mustParseXML(t, samlAssertion)

	edElem, err := xmlenc1.NewEncryptor().
		KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
		RecipientECPublicKey(&key.PublicKey).
		KeyDerivationParams(&xmlenc1.ConcatKDFParams{
			AlgorithmID:  []byte{0x00, 0xaa, 0xbb},
			PartyUInfo:   []byte{0x01},
			PartyVInfo:   []byte{0x02},
			SuppPubInfo:  []byte{0x03},
			SuppPrivInfo: []byte{0x04},
		}).
		EncryptElement(t.Context(), doc.DocumentElement())
	require.NoError(t, err)

	// Nothing application-specific may reach the wire.
	encoded, err := helium.WriteString(edElem)
	require.NoError(t, err)
	require.NotContains(t, encoded, "AlgorithmID=")
	require.NotContains(t, encoded, "PartyUInfo=")
	require.NotContains(t, encoded, "PartyVInfo=")
	require.NotContains(t, encoded, "SuppPubInfo=")
	require.NotContains(t, encoded, "SuppPrivInfo=")

	ed, err := xmlenc1.ParseEncryptedDataForTest(edElem)
	require.NoError(t, err)
	kdf := ed.EncryptedKeys[0].AgreementMethod.KeyDerivationMethod.ConcatKDF
	require.Equal(t, xmlenc1.DigestSHA256, kdf.DigestMethod)
	require.Empty(t, kdf.AlgorithmID)
	require.Empty(t, kdf.PartyUInfo)
	require.Empty(t, kdf.PartyVInfo)
	require.Empty(t, kdf.SuppPubInfo)
	require.Empty(t, kdf.SuppPrivInfo)

	nodes, err := xmlenc1.NewDecryptor().ECPrivateKey(key).Decrypt(t.Context(), edElem)
	require.NoError(t, err)
	require.Len(t, nodes, 1)
}

// The Encryptor documents clone-on-write, so the OtherInfo arrays a caller
// hands over must be copied: they feed both the derived KEK and the emitted
// ConcatKDFParams, and a later mutation must not reach either.
func TestEncryptECDHESCopiesKDFParams(t *testing.T) {
	key := generateECKey(t, elliptic.P256())
	doc := mustParseXML(t, samlAssertion)

	algID := []byte{0x01, 0x02, 0x03}
	encryptor := xmlenc1.NewEncryptor().
		KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
		RecipientECPublicKey(&key.PublicKey).
		KeyDerivationParams(&xmlenc1.ConcatKDFParams{
			DigestMethod: xmlenc1.DigestSHA256,
			AlgorithmID:  algID,
		})
	algID[0] = 0xff

	edElem, err := encryptor.EncryptElement(t.Context(), doc.DocumentElement())
	require.NoError(t, err)

	ed, err := xmlenc1.ParseEncryptedDataForTest(edElem)
	require.NoError(t, err)
	kdf := ed.EncryptedKeys[0].AgreementMethod.KeyDerivationMethod.ConcatKDF
	require.Equal(t, []byte{0x01, 0x02, 0x03}, kdf.AlgorithmID)
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

	// P-224 has no crypto/ecdh form at all, so the recipient key is unusable
	// however much work is done first. Pairing it with a session key of the
	// wrong length pins WHERE the rejection happens: the session-key check
	// sits immediately before plaintext serialization and block encryption,
	// so a KeySizeError here would mean the curve was only noticed after all
	// that payload-proportional work had already run.
	t.Run("rejects an unusable curve before any payload work", func(t *testing.T) {
		p224 := generateECKey(t, elliptic.P224())
		doc := mustParseXML(t, samlAssertion)

		_, err := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES128GCM).
			SessionKey(make([]byte, 32)).
			KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
			RecipientECPublicKey(&p224.PublicKey).
			EncryptElement(t.Context(), doc.DocumentElement())
		require.ErrorIs(t, err, xmlenc1.ErrEncryptionFailed)

		var keySize *xmlenc1.KeySizeError
		require.NotErrorAs(t, err, &keySize)
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
