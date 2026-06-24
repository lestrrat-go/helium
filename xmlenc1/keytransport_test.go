package xmlenc1_test

import (
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmlenc1"
	"github.com/stretchr/testify/require"
)

func TestRSAOAEP(t *testing.T) {
	t.Run("oaep11 sha256 round-trip", func(t *testing.T) {
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

		xml, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Contains(t, xml, xmlenc1.DigestSHA256)
		require.Contains(t, xml, xmlenc1.MGFSHA256)

		decryptor := xmlenc1.NewDecryptor().PrivateKey(key)
		nodes, err := decryptor.Decrypt(t.Context(), edElem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)

		s, err := helium.WriteString(nodes[0])
		require.NoError(t, err)
		require.Contains(t, s, "user@example.com")
	})

	t.Run("encrypt unsupported digest errors", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, samlAssertion)

		encryptor := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES256GCM).
			KeyTransportAlgorithm(xmlenc1.RSAOAEP11).
			OAEPDigest("http://www.w3.org/2001/04/xmlenc#sha512").
			RecipientPublicKey(&key.PublicKey)

		_, err := encryptor.EncryptElement(t.Context(), doc.DocumentElement())
		require.Error(t, err)
		var unsupp *xmlenc1.UnsupportedAlgorithmError
		require.ErrorAs(t, err, &unsupp)
	})

	t.Run("oaep11 mismatched MGF errors", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, samlAssertion)

		encryptor := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES256GCM).
			KeyTransportAlgorithm(xmlenc1.RSAOAEP11).
			OAEPDigest(xmlenc1.DigestSHA256).
			OAEPMGF(xmlenc1.MGFSHA1).
			RecipientPublicKey(&key.PublicKey)

		_, err := encryptor.EncryptElement(t.Context(), doc.DocumentElement())
		require.Error(t, err)
		require.ErrorIs(t, err, xmlenc1.ErrEncryptionFailed)
	})

	t.Run("mgf1p with sha256 digest errors", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, samlAssertion)

		encryptor := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES256GCM).
			KeyTransportAlgorithm(xmlenc1.RSAOAEP).
			OAEPDigest(xmlenc1.DigestSHA256).
			RecipientPublicKey(&key.PublicKey)

		_, err := encryptor.EncryptElement(t.Context(), doc.DocumentElement())
		require.Error(t, err)
		require.ErrorIs(t, err, xmlenc1.ErrEncryptionFailed)
	})

	t.Run("decrypt unsupported digest errors", func(t *testing.T) {
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

		// Tamper with the serialized DigestMethod to an unsupported URI, then
		// re-parse and attempt to decrypt: it must error, not fall back.
		xml, err := helium.WriteString(doc)
		require.NoError(t, err)
		tampered := strings.Replace(xml, xmlenc1.DigestSHA256, "http://www.w3.org/2001/04/xmlenc#sha512", 1)
		require.NotEqual(t, xml, tampered)

		_ = edElem
		tdoc := mustParseXML(t, tampered)
		edNodes := findEncryptedData(t, tdoc.DocumentElement())
		require.NotNil(t, edNodes)

		decryptor := xmlenc1.NewDecryptor().PrivateKey(key)
		_, err = decryptor.Decrypt(t.Context(), edNodes)
		require.Error(t, err)
		var unsupp *xmlenc1.UnsupportedAlgorithmError
		require.ErrorAs(t, err, &unsupp)
	})

	t.Run("mgf1p rejects any MGF", func(t *testing.T) {
		key := generateRSAKey(t)

		for _, mgf := range []string{xmlenc1.MGFSHA1, xmlenc1.MGFSHA256} {
			doc := mustParseXML(t, samlAssertion)
			encryptor := xmlenc1.NewEncryptor().
				BlockAlgorithm(xmlenc1.AES256GCM).
				KeyTransportAlgorithm(xmlenc1.RSAOAEP).
				OAEPMGF(mgf).
				RecipientPublicKey(&key.PublicKey)

			_, err := encryptor.EncryptElement(t.Context(), doc.DocumentElement())
			require.Error(t, err, "mgf %q must be rejected for rsa-oaep-mgf1p", mgf)
			require.ErrorIs(t, err, xmlenc1.ErrEncryptionFailed)

			// No partial EncryptedData/MGF element should have been serialized.
			xml, werr := helium.WriteString(doc)
			require.NoError(t, werr)
			require.NotContains(t, xml, "MGF")
		}
	})

	t.Run("decrypt param error is decryption failed", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, samlAssertion)

		// Produce a valid RSA-OAEP 1.1 SHA-256 EncryptedData, then tamper the
		// serialized MGF down to MGF1-SHA-1 so digest and MGF hashes mismatch.
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
		tampered := strings.Replace(xml, xmlenc1.MGFSHA256, xmlenc1.MGFSHA1, 1)
		require.NotEqual(t, xml, tampered)

		tdoc := mustParseXML(t, tampered)
		edNode := findEncryptedData(t, tdoc.DocumentElement())
		require.NotNil(t, edNode)

		decryptor := xmlenc1.NewDecryptor().PrivateKey(key)
		_, err = decryptor.Decrypt(t.Context(), edNode)
		require.Error(t, err)
		require.ErrorIs(t, err, xmlenc1.ErrDecryptionFailed)
		require.NotErrorIs(t, err, xmlenc1.ErrEncryptionFailed)
	})
}

func TestUnsupportedKeyTransport(t *testing.T) {
	t.Run("encrypt errors", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, samlAssertion)

		encryptor := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES256GCM).
			KeyTransportAlgorithm("urn:bogus").
			RecipientPublicKey(&key.PublicKey)

		_, err := encryptor.EncryptElement(t.Context(), doc.DocumentElement())
		require.Error(t, err)
		require.ErrorIs(t, err, xmlenc1.ErrEncryptionFailed)
		var unsupp *xmlenc1.UnsupportedAlgorithmError
		require.ErrorAs(t, err, &unsupp)
		require.Equal(t, "urn:bogus", unsupp.Algorithm)

		// No EncryptedData (and certainly no bogus URI) should be serialized.
		xml, werr := helium.WriteString(doc)
		require.NoError(t, werr)
		require.NotContains(t, xml, "urn:bogus")
		require.NotContains(t, xml, elemEncryptedData)
	})

	t.Run("decrypt errors", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, samlAssertion)

		// Produce a valid RSA-OAEP 1.1 EncryptedData, then tamper the
		// serialized key-transport @Algorithm to an unsupported URI.
		encryptor := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES256GCM).
			KeyTransportAlgorithm(xmlenc1.RSAOAEP11).
			RecipientPublicKey(&key.PublicKey)

		_, err := encryptor.EncryptElement(t.Context(), doc.DocumentElement())
		require.NoError(t, err)

		xml, err := helium.WriteString(doc)
		require.NoError(t, err)
		tampered := strings.Replace(xml, xmlenc1.RSAOAEP11, "urn:bogus", 1)
		require.NotEqual(t, xml, tampered)

		tdoc := mustParseXML(t, tampered)
		edNode := findEncryptedData(t, tdoc.DocumentElement())
		require.NotNil(t, edNode)

		decryptor := xmlenc1.NewDecryptor().PrivateKey(key)
		_, err = decryptor.Decrypt(t.Context(), edNode)
		require.Error(t, err)
		require.ErrorIs(t, err, xmlenc1.ErrDecryptionFailed)
		require.NotErrorIs(t, err, xmlenc1.ErrEncryptionFailed)
		var unsupp *xmlenc1.UnsupportedAlgorithmError
		require.ErrorAs(t, err, &unsupp)
		require.Equal(t, "urn:bogus", unsupp.Algorithm)
	})
}

func TestResolveSessionKey(t *testing.T) {
	t.Run("missing key", func(t *testing.T) {
		t.Run("no key material at all", func(t *testing.T) {
			doc := mustParseXML(t, `<root/>`)
			ed := &xmlenc1.EncryptedData{
				Type:             xmlenc1.TypeElement,
				EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES256GCM},
				CipherValue:      make([]byte, 48),
			}
			elem, err := xmlenc1.MarshalEncryptedDataForTest(doc, ed)
			require.NoError(t, err)

			// A decryptor with no session key and no EncryptedKey present.
			_, err = xmlenc1.NewDecryptor().Decrypt(t.Context(), elem)
			require.ErrorIs(t, err, xmlenc1.ErrMissingKey)
		})

		t.Run("RSA EncryptedKey but no private key", func(t *testing.T) {
			doc := mustParseXML(t, `<root/>`)
			ed := &xmlenc1.EncryptedData{
				Type:             xmlenc1.TypeElement,
				EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES256GCM},
				EncryptedKey: &xmlenc1.EncryptedKey{
					EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.RSAOAEP},
					CipherValue:      make([]byte, 256),
				},
				CipherValue: make([]byte, 48),
			}
			elem, err := xmlenc1.MarshalEncryptedDataForTest(doc, ed)
			require.NoError(t, err)

			_, err = xmlenc1.NewDecryptor().Decrypt(t.Context(), elem)
			require.ErrorIs(t, err, xmlenc1.ErrMissingKey)
		})

		t.Run("AES key-wrap EncryptedKey but no KEK", func(t *testing.T) {
			doc := mustParseXML(t, `<root/>`)
			ed := &xmlenc1.EncryptedData{
				Type:             xmlenc1.TypeElement,
				EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES256GCM},
				EncryptedKey: &xmlenc1.EncryptedKey{
					EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES256KeyWrap},
					CipherValue:      make([]byte, 40),
				},
				CipherValue: make([]byte, 48),
			}
			elem, err := xmlenc1.MarshalEncryptedDataForTest(doc, ed)
			require.NoError(t, err)

			_, err = xmlenc1.NewDecryptor().Decrypt(t.Context(), elem)
			require.ErrorIs(t, err, xmlenc1.ErrMissingKey)
		})

		t.Run("EncryptedKey missing EncryptionMethod", func(t *testing.T) {
			doc := mustParseXML(t, `<root/>`)
			ed := &xmlenc1.EncryptedData{
				Type:             xmlenc1.TypeElement,
				EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES256GCM},
				EncryptedKey: &xmlenc1.EncryptedKey{
					CipherValue: make([]byte, 40),
				},
				CipherValue: make([]byte, 48),
			}
			elem, err := xmlenc1.MarshalEncryptedDataForTest(doc, ed)
			require.NoError(t, err)

			_, err = xmlenc1.NewDecryptor().PrivateKey(generateRSAKey(t)).Decrypt(t.Context(), elem)
			require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
		})
	})

	t.Run("unsupported key transport", func(t *testing.T) {
		doc := mustParseXML(t, `<root/>`)
		ed := &xmlenc1.EncryptedData{
			Type:             xmlenc1.TypeElement,
			EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES256GCM},
			EncryptedKey: &xmlenc1.EncryptedKey{
				EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: "urn:example:not-a-real-algorithm"},
				CipherValue:      make([]byte, 40),
			},
			CipherValue: make([]byte, 48),
		}
		elem, err := xmlenc1.MarshalEncryptedDataForTest(doc, ed)
		require.NoError(t, err)

		_, err = xmlenc1.NewDecryptor().PrivateKey(generateRSAKey(t)).Decrypt(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrDecryptionFailed)
		var unsupported *xmlenc1.UnsupportedAlgorithmError
		require.ErrorAs(t, err, &unsupported)
	})
}

// TestOAEPNameHelperBranches drives the digest/MGF-name formatting helpers
// (oaepDigestName / oaepMGFName) through their non-default and default
// branches. They are only reached when oaepHashFunc rejects a digest/MGF
// hash mismatch, so each case sets up an RSA-OAEP 1.1 configuration whose
// declared digest and MGF hashes disagree and asserts the encrypt path
// surfaces ErrEncryptionFailed.
func TestOAEPNameHelperBranches(t *testing.T) {
	key := generateRSAKey(t)

	t.Run("default digest vs explicit SHA256 MGF", func(t *testing.T) {
		// digest unset -> SHA1 (default); MGF SHA256 -> mismatch.
		// Exercises oaepDigestName("") default branch and the
		// oaepMGFName non-empty branch.
		doc := mustParseXML(t, samlAssertion)
		encryptor := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES256GCM).
			KeyTransportAlgorithm(xmlenc1.RSAOAEP11).
			OAEPMGF(xmlenc1.MGFSHA256).
			RecipientPublicKey(&key.PublicKey)

		_, err := encryptor.EncryptElement(t.Context(), doc.DocumentElement())
		require.ErrorIs(t, err, xmlenc1.ErrEncryptionFailed)
	})

	t.Run("explicit SHA256 digest vs default MGF", func(t *testing.T) {
		// digest SHA256; MGF unset -> SHA1 (default) -> mismatch.
		// Exercises the oaepMGFName default branch for RSAOAEP11.
		doc := mustParseXML(t, samlAssertion)
		encryptor := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES256GCM).
			KeyTransportAlgorithm(xmlenc1.RSAOAEP11).
			OAEPDigest(xmlenc1.DigestSHA256).
			RecipientPublicKey(&key.PublicKey)

		_, err := encryptor.EncryptElement(t.Context(), doc.DocumentElement())
		require.ErrorIs(t, err, xmlenc1.ErrEncryptionFailed)
	})

	t.Run("legacy mgf1p with SHA256 digest", func(t *testing.T) {
		// rsa-oaep-mgf1p fixes MGF to SHA1; a SHA256 digest mismatches.
		// Exercises oaepMGFName's "implied by rsa-oaep-mgf1p" branch.
		doc := mustParseXML(t, samlAssertion)
		encryptor := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES256GCM).
			KeyTransportAlgorithm(xmlenc1.RSAOAEP).
			OAEPDigest(xmlenc1.DigestSHA256).
			RecipientPublicKey(&key.PublicKey)

		_, err := encryptor.EncryptElement(t.Context(), doc.DocumentElement())
		require.ErrorIs(t, err, xmlenc1.ErrEncryptionFailed)
	})
}

// TestEncryptorOAEPParamsRoundTrip covers the Encryptor.OAEPParams setter and
// the OAEPParams encrypt/decrypt path through RSA-OAEP 1.1.
func TestEncryptorOAEPParamsRoundTrip(t *testing.T) {
	key := generateRSAKey(t)
	doc := mustParseXML(t, samlAssertion)

	encryptor := xmlenc1.NewEncryptor().
		BlockAlgorithm(xmlenc1.AES256GCM).
		KeyTransportAlgorithm(xmlenc1.RSAOAEP11).
		OAEPDigest(xmlenc1.DigestSHA256).
		OAEPMGF(xmlenc1.MGFSHA256).
		OAEPParams([]byte("label-params")).
		RecipientPublicKey(&key.PublicKey)

	elem, err := encryptor.EncryptElement(t.Context(), doc.DocumentElement())
	require.NoError(t, err)

	decryptor := xmlenc1.NewDecryptor().PrivateKey(key)
	nodes, err := decryptor.Decrypt(t.Context(), elem)
	require.NoError(t, err)
	require.Len(t, nodes, 1)
}
