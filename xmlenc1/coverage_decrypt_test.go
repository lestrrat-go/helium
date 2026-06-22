package xmlenc1_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmlenc1"
	"github.com/stretchr/testify/require"
)

// TestDecryptMissingEncryptionMethod covers the decryptElement branch that
// rejects an EncryptedData carrying CipherData but no EncryptionMethod.
func TestDecryptMissingEncryptionMethod(t *testing.T) {
	doc := mustParseXML(t, `<root/>`)
	ed := &xmlenc1.EncryptedData{
		Type:        xmlenc1.TypeElement,
		CipherValue: make([]byte, 48),
	}
	elem, err := xmlenc1.MarshalEncryptedDataForTest(doc, ed)
	require.NoError(t, err)

	decryptor := xmlenc1.NewDecryptor().SessionKey(make([]byte, 32))
	_, err = decryptor.Decrypt(t.Context(), elem)
	require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
}

// TestResolveSessionKeyMissingKey covers the no-key-available branches of
// resolveSessionKey: no session key, no private key, no KEK.
func TestResolveSessionKeyMissingKey(t *testing.T) {
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
}

// TestResolveSessionKeyUnsupportedKeyTransport covers the default branch of
// resolveSessionKey: an EncryptedKey declaring an unknown key-transport URI.
func TestResolveSessionKeyUnsupportedKeyTransport(t *testing.T) {
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

// TestDecryptElementYieldsNonElement covers the decryptElement branch that
// rejects a Type=Element payload whose decrypted plaintext is not a single
// element (here: plain text content).
func TestDecryptElementYieldsNonElement(t *testing.T) {
	sessionKey := make([]byte, 32)
	algorithm := xmlenc1.AES256GCM

	// Plaintext that is bare text, not an element. For Type=Element this
	// must be rejected as ErrDecryptionFailed.
	cipher, err := xmlenc1.EncryptBytesForTest(algorithm, sessionKey, []byte("just text, no element"))
	require.NoError(t, err)

	doc := mustParseXML(t, `<root/>`)
	ed := &xmlenc1.EncryptedData{
		Type:             xmlenc1.TypeElement,
		EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: algorithm},
		CipherValue:      cipher,
	}
	elem, err := xmlenc1.MarshalEncryptedDataForTest(doc, ed)
	require.NoError(t, err)

	_, err = xmlenc1.NewDecryptor().SessionKey(sessionKey).Decrypt(t.Context(), elem)
	require.ErrorIs(t, err, xmlenc1.ErrDecryptionFailed)
}

// TestDecryptContentMultipleNodes covers the Type=Content branch of
// decryptElement that collects multiple decrypted child nodes.
func TestDecryptContentMultipleNodes(t *testing.T) {
	key := generateRSAKey(t)
	doc := mustParseXML(t, `<root><a>1</a><b>2</b>text</root>`)

	encryptor := xmlenc1.NewEncryptor().
		BlockAlgorithm(xmlenc1.AES256GCM).
		KeyTransportAlgorithm(xmlenc1.RSAOAEP11).
		RecipientPublicKey(&key.PublicKey)

	elem, err := encryptor.EncryptContent(t.Context(), doc.DocumentElement())
	require.NoError(t, err)

	decryptor := xmlenc1.NewDecryptor().PrivateKey(key)
	nodes, err := decryptor.Decrypt(t.Context(), elem)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(nodes), 2)

	// Sanity: the recovered content includes the original element children.
	var names []string
	for _, n := range nodes {
		if e, ok := helium.AsNode[*helium.Element](n); ok {
			names = append(names, e.Name())
		}
	}
	require.Contains(t, names, "a")
	require.Contains(t, names, "b")
}
