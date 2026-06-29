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

	t.Run("oaep11 distinct digest and MGF hash round-trip", func(t *testing.T) {
		// XML Encryption 1.1 permits an RSA-OAEP DigestMethod that differs
		// from the MGF1 hash. crypto/rsa's option-bearing OAEP API can
		// represent the two distinctly, so this must round-trip rather than
		// be rejected.
		key := generateRSAKey(t)
		doc := mustParseXML(t, samlAssertion)

		encryptor := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES256GCM).
			KeyTransportAlgorithm(xmlenc1.RSAOAEP11).
			OAEPDigest(xmlenc1.DigestSHA256).
			OAEPMGF(xmlenc1.MGFSHA1).
			RecipientPublicKey(&key.PublicKey)

		edElem, err := encryptor.EncryptElement(t.Context(), doc.DocumentElement())
		require.NoError(t, err)

		xml, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Contains(t, xml, xmlenc1.DigestSHA256)
		require.Contains(t, xml, xmlenc1.MGFSHA1)

		decryptor := xmlenc1.NewDecryptor().PrivateKey(key)
		nodes, err := decryptor.Decrypt(t.Context(), edElem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)

		s, err := helium.WriteString(nodes[0])
		require.NoError(t, err)
		require.Contains(t, s, "user@example.com")
	})

	t.Run("mgf1p with sha256 digest round-trip", func(t *testing.T) {
		// rsa-oaep-mgf1p fixes MGF1 to SHA-1 but the label digest may still
		// be SHA-256; that distinct-hash combination must now round-trip.
		key := generateRSAKey(t)
		doc := mustParseXML(t, samlAssertion)

		encryptor := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES256GCM).
			KeyTransportAlgorithm(xmlenc1.RSAOAEP).
			OAEPDigest(xmlenc1.DigestSHA256).
			RecipientPublicKey(&key.PublicKey)

		edElem, err := encryptor.EncryptElement(t.Context(), doc.DocumentElement())
		require.NoError(t, err)

		decryptor := xmlenc1.NewDecryptor().PrivateKey(key)
		nodes, err := decryptor.Decrypt(t.Context(), edElem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
	})

	t.Run("oaep11 absent MGF defaults to SHA-1 not digest", func(t *testing.T) {
		// W3C xmlenc-core1 §5.5.2: an absent xenc11:MGF defaults to MGF1
		// with SHA-1, INDEPENDENT of DigestMethod. A no-MGF ciphertext must
		// therefore interoperate with explicit MGFSHA1 semantics even when
		// the label digest is SHA-256 (it must NOT use a SHA-256 MGF).
		key := generateRSAKey(t)
		doc := mustParseXML(t, samlAssertion)

		encryptor := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES256GCM).
			KeyTransportAlgorithm(xmlenc1.RSAOAEP11).
			OAEPDigest(xmlenc1.DigestSHA256).
			RecipientPublicKey(&key.PublicKey)

		_, err := encryptor.EncryptElement(t.Context(), doc.DocumentElement())
		require.NoError(t, err)

		// No MGF element must be serialized when none was requested.
		// Match the element start-tag markup (":MGF"), not the bare
		// substring "MGF": the latter can appear by chance inside the
		// randomized base64 CipherValue, but a colon never can.
		xml, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Contains(t, xml, xmlenc1.DigestSHA256)
		require.NotContains(t, xml, ":MGF")

		// Inject an explicit MGF1-SHA-1 element into the EncryptionMethod.
		// Because the absent-MGF default is SHA-1, decryption under explicit
		// MGFSHA1 must succeed. Had the default been the SHA-256 digest, the
		// MGF1 hashes would mismatch and decryption would fail.
		mgfElem := `<xenc11:MGF xmlns:xenc11="` + xmlenc1.NamespaceXMLEnc11 + `" Algorithm="` + xmlenc1.MGFSHA1 + `"/>`
		tampered := strings.Replace(xml, "</xenc:EncryptionMethod>", mgfElem+"</xenc:EncryptionMethod>", 1)
		require.NotEqual(t, xml, tampered)

		tdoc := mustParseXML(t, tampered)
		edNode := findEncryptedData(t, tdoc.DocumentElement())
		require.NotNil(t, edNode)

		decryptor := xmlenc1.NewDecryptor().PrivateKey(key)
		nodes, err := decryptor.Decrypt(t.Context(), edNode)
		require.NoError(t, err)
		require.Len(t, nodes, 1)

		s, err := helium.WriteString(nodes[0])
		require.NoError(t, err)
		require.Contains(t, s, "user@example.com")
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
			// Match the element start-tag markup (":MGF") rather than the bare
			// "MGF" substring, which a random base64 CipherValue could contain.
			xml, werr := helium.WriteString(doc)
			require.NoError(t, werr)
			require.NotContains(t, xml, ":MGF")
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
				EncryptedKeys: []*xmlenc1.EncryptedKey{
					{
						EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.RSAOAEP},
						CipherValue:      make([]byte, 256),
					},
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
				EncryptedKeys: []*xmlenc1.EncryptedKey{
					{
						EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES256KeyWrap},
						CipherValue:      make([]byte, 40),
					},
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
				EncryptedKeys: []*xmlenc1.EncryptedKey{
					{
						CipherValue: make([]byte, 40),
					},
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
			EncryptedKeys: []*xmlenc1.EncryptedKey{
				{
					EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: "urn:example:not-a-real-algorithm"},
					CipherValue:      make([]byte, 40),
				},
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
