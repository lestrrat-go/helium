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

func TestEncryptECDHESSHA224(t *testing.T) {
	key := generateECKey(t, elliptic.P256())
	doc := mustParseXML(t, samlAssertion)

	edElem, err := xmlenc1.NewEncryptor().
		BlockAlgorithm(xmlenc1.AES256GCM11).
		KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
		RecipientECPublicKey(&key.PublicKey).
		KeyDerivationParams(&xmlenc1.ConcatKDFParams{
			DigestMethod: xmlenc1.DigestSHA224,
			PartyUInfo:   []byte{0x01, 0x02},
		}).
		EncryptElement(t.Context(), doc.DocumentElement())
	require.NoError(t, err)

	ed, err := xmlenc1.ParseEncryptedDataForTest(edElem)
	require.NoError(t, err)
	require.Equal(t, xmlenc1.DigestSHA224, ed.EncryptedKeys[0].AgreementMethod.KeyDerivationMethod.ConcatKDF.DigestMethod)

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

// The OtherInfo budget governs what is derived from and written, so a set that
// is discarded before either is never weighed against it: params with an empty
// DigestMethod are replaced wholesale by the SHA-256 default carrying no
// OtherInfo, so even fields far over the budget encrypt. Every point that
// applies the budget sits behind that substitution, and this pins that they all
// do.
func TestEncryptECDHESEmptyDigestDropsOversizedOtherInfo(t *testing.T) {
	key := generateECKey(t, elliptic.P256())
	doc := mustParseXML(t, samlAssertion)

	edElem, err := xmlenc1.NewEncryptor().
		KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
		RecipientECPublicKey(&key.PublicKey).
		KeyDerivationParams(&xmlenc1.ConcatKDFParams{
			AlgorithmID:  make([]byte, xmlenc1.MaxConcatKDFOtherInfoBytesForTest),
			PartyUInfo:   make([]byte, xmlenc1.MaxConcatKDFOtherInfoBytesForTest),
			PartyVInfo:   make([]byte, xmlenc1.MaxConcatKDFOtherInfoBytesForTest),
			SuppPubInfo:  make([]byte, xmlenc1.MaxConcatKDFOtherInfoBytesForTest),
			SuppPrivInfo: make([]byte, xmlenc1.MaxConcatKDFOtherInfoBytesForTest),
		}).
		EncryptElement(t.Context(), doc.DocumentElement())
	require.NoError(t, err)

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

	// Every caller setting key agreement reads is decided before any payload
	// work, and pairing each with a session key of the wrong length pins that:
	// the session-key binding is the last check before block encryption, so a
	// KeySizeError would mean the setting that can never work was only noticed
	// after the payload had been encrypted, and the caller would hear about the
	// session key instead.
	t.Run("reports a bad setting ahead of the session key", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			// enc carries the bad setting; every case adds the same
			// wrong-length session key below.
			enc xmlenc1.Encryptor
			// errFragment is what the setting's own rejection says.
			errFragment string
		}{
			{
				name:        "a non-key-wrap algorithm",
				enc:         xmlenc1.NewEncryptor().KeyWrapAlgorithm(xmlenc1.AES256GCM),
				errFragment: `unsupported key wrap algorithm`,
			},
			{
				name: "oversized OtherInfo",
				enc: xmlenc1.NewEncryptor().
					KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
					KeyDerivationParams(&xmlenc1.ConcatKDFParams{
						DigestMethod: xmlenc1.DigestSHA256,
						AlgorithmID:  make([]byte, xmlenc1.MaxConcatKDFOtherInfoBytesForTest+1),
					}),
				errFragment: "ConcatKDF OtherInfo is over the",
			},
			{
				name: "an unsupported KDF digest",
				enc: xmlenc1.NewEncryptor().
					KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
					KeyDerivationParams(&xmlenc1.ConcatKDFParams{DigestMethod: "urn:example:nope"}),
				errFragment: `unsupported ConcatKDF digest algorithm`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				doc := mustParseXML(t, samlAssertion)
				_, err := tc.enc.
					BlockAlgorithm(xmlenc1.AES128GCM).
					SessionKey(make([]byte, 32)).
					RecipientECPublicKey(&key.PublicKey).
					EncryptElement(t.Context(), doc.DocumentElement())
				require.ErrorIs(t, err, xmlenc1.ErrEncryptionFailed)
				require.Contains(t, err.Error(), tc.errFragment)

				var keySize *xmlenc1.KeySizeError
				require.NotErrorAs(t, err, &keySize)

				// And the caller's tree still holds the plaintext.
				xml, err := helium.WriteString(doc)
				require.NoError(t, err)
				require.Contains(t, xml, "user@example.com")
			})
		}
	})

	// The OtherInfo budget is shared with the parse side, so an Encryptor
	// refused by it carries that sentinel too. A caller matching on either one
	// must see the same verdict.
	t.Run("oversized OtherInfo carries both sentinels", func(t *testing.T) {
		doc := mustParseXML(t, samlAssertion)
		_, err := xmlenc1.NewEncryptor().
			KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
			RecipientECPublicKey(&key.PublicKey).
			KeyDerivationParams(&xmlenc1.ConcatKDFParams{
				DigestMethod: xmlenc1.DigestSHA256,
				AlgorithmID:  make([]byte, xmlenc1.MaxConcatKDFOtherInfoBytesForTest+1),
			}).
			EncryptElement(t.Context(), doc.DocumentElement())
		require.ErrorIs(t, err, xmlenc1.ErrEncryptionFailed)
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
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

	// A recipient key with unset coordinates is caller-constructed garbage,
	// but ecdhRecipientKey is the one gate that decides whether a recipient
	// key is usable at all, and crypto/ecdsa reads the coordinates before it
	// validates them. Without the gate the public API panics where its
	// documented contract is an error.
	t.Run("rejects a recipient key with no curve point", func(t *testing.T) {
		p256 := generateECKey(t, elliptic.P256())

		for _, tc := range []struct {
			name string
			pub  *ecdsa.PublicKey
		}{
			{name: "no coordinates", pub: &ecdsa.PublicKey{Curve: elliptic.P256()}},
			{name: "no Y", pub: &ecdsa.PublicKey{Curve: elliptic.P256(), X: p256.X}},
			{name: "no X", pub: &ecdsa.PublicKey{Curve: elliptic.P256(), Y: p256.Y}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				doc := mustParseXML(t, samlAssertion)
				_, err := xmlenc1.NewEncryptor().
					KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
					RecipientECPublicKey(tc.pub).
					EncryptElement(t.Context(), doc.DocumentElement())
				require.ErrorIs(t, err, xmlenc1.ErrEncryptionFailed)
				require.Contains(t, err.Error(), "recipient EC public key")
			})
		}
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

// ecElementRejectedBeforeSerialization returns an element that cannot be
// serialized: Document.CreateElement only rejects a colon in the name, while
// the writer rejects the whole name at emit time. Serializing it is the first
// payload-proportional step of EncryptElement, so its failure marks where
// serialization happened.
func ecElementRejectedBeforeSerialization(t *testing.T) *helium.Element {
	t.Helper()
	doc := helium.NewDefaultDocument()
	elem, err := doc.CreateElement(`root injected="1"`)
	require.NoError(t, err)
	require.NoError(t, doc.SetDocumentElement(elem))
	return elem
}

// An unusable recipient curve must be noticed before the plaintext is
// serialized, not after. Pinning it on an element that itself fails to
// serialize says WHERE the rejection happens without timing anything: a
// serialization error would mean the curve was noticed too late.
func TestEncryptECDHESRejectsCurveBeforeSerialization(t *testing.T) {
	p224 := generateECKey(t, elliptic.P224())

	_, err := xmlenc1.NewEncryptor().
		KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
		RecipientECPublicKey(&p224.PublicKey).
		EncryptElement(t.Context(), ecElementRejectedBeforeSerialization(t))
	require.ErrorIs(t, err, xmlenc1.ErrEncryptionFailed)
	require.Contains(t, err.Error(), "recipient EC public key")
	require.NotContains(t, err.Error(), "invalid element name")
}

// RSA key transport and ECDH-ES key agreement both protect the same session
// key, and only one EncryptedKey is emitted. Accepting both would drop the EC
// recipient, so a recipient holding only the EC private key would fail to
// decrypt with an error that points nowhere near the real mistake.
func TestEncryptConflictingTransportAndAgreement(t *testing.T) {
	rsaKey := generateRSAKey(t)
	ecKey := generateECKey(t, elliptic.P256())

	newEncryptor := func() xmlenc1.Encryptor {
		return xmlenc1.NewEncryptor().
			KeyTransportAlgorithm(xmlenc1.RSAOAEP).
			RecipientPublicKey(&rsaKey.PublicKey).
			KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
			RecipientECPublicKey(&ecKey.PublicKey)
	}

	t.Run("EncryptElement", func(t *testing.T) {
		doc := mustParseXML(t, samlAssertion)
		_, err := newEncryptor().EncryptElement(t.Context(), doc.DocumentElement())
		require.ErrorIs(t, err, xmlenc1.ErrConflictingKeyConfig)
		require.Contains(t, err.Error(), xmlenc1.RSAOAEP)
		require.Contains(t, err.Error(), xmlenc1.AES256KeyWrap)
	})

	t.Run("EncryptContent", func(t *testing.T) {
		doc := mustParseXML(t, samlAssertion)
		_, err := newEncryptor().EncryptContent(t.Context(), doc.DocumentElement())
		require.ErrorIs(t, err, xmlenc1.ErrConflictingKeyConfig)
	})

	t.Run("EncryptBytes", func(t *testing.T) {
		_, err := newEncryptor().EncryptBytes(t.Context(), helium.NewDefaultDocument(), []byte("payload"))
		require.ErrorIs(t, err, xmlenc1.ErrConflictingKeyConfig)
	})

	t.Run("key transport alone round-trips", func(t *testing.T) {
		doc := mustParseXML(t, samlAssertion)
		edElem, err := xmlenc1.NewEncryptor().
			KeyTransportAlgorithm(xmlenc1.RSAOAEP).
			RecipientPublicKey(&rsaKey.PublicKey).
			EncryptElement(t.Context(), doc.DocumentElement())
		require.NoError(t, err)

		nodes, err := xmlenc1.NewDecryptor().PrivateKey(rsaKey).Decrypt(t.Context(), edElem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
	})

	t.Run("key agreement alone round-trips", func(t *testing.T) {
		doc := mustParseXML(t, samlAssertion)
		edElem, err := xmlenc1.NewEncryptor().
			KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
			RecipientECPublicKey(&ecKey.PublicKey).
			EncryptElement(t.Context(), doc.DocumentElement())
		require.NoError(t, err)

		nodes, err := xmlenc1.NewDecryptor().ECPrivateKey(ecKey).Decrypt(t.Context(), edElem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
	})

	// An EC public key with no KeyWrapAlgorithm does not select key
	// agreement, so it is not a conflict and its curve is never resolved: an
	// unusable curve must not fail an encryption that would not have touched
	// that key.
	t.Run("unused EC key does not block key transport", func(t *testing.T) {
		p224 := generateECKey(t, elliptic.P224())
		doc := mustParseXML(t, samlAssertion)

		edElem, err := xmlenc1.NewEncryptor().
			KeyTransportAlgorithm(xmlenc1.RSAOAEP).
			RecipientPublicKey(&rsaKey.PublicKey).
			RecipientECPublicKey(&p224.PublicKey).
			EncryptElement(t.Context(), doc.DocumentElement())
		require.NoError(t, err)

		nodes, err := xmlenc1.NewDecryptor().PrivateKey(rsaKey).Decrypt(t.Context(), edElem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
	})
}

// agreementAndKeyWrapEncryptor configures the two mechanisms that share one
// KeyWrapAlgorithm URI: ECDH-ES key agreement, which derives the
// key-encryption key, and AES key wrap, which takes the supplied one.
func agreementAndKeyWrapEncryptor(ecPub *ecdsa.PublicKey, kek []byte) xmlenc1.Encryptor {
	return xmlenc1.NewEncryptor().
		KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
		RecipientECPublicKey(ecPub).
		KeyEncryptionKey(kek)
}

// ECDH-ES key agreement and AES key wrap protect the same session key under
// the same KeyWrapAlgorithm URI, and only one EncryptedKey is emitted.
// Accepting both would derive a key-encryption key and discard the configured
// one, so a recipient holding only the KEK would fail to decrypt with an
// error that points nowhere near the real mistake.
func TestEncryptConflictingAgreementAndKeyWrap(t *testing.T) {
	ecKey := generateECKey(t, elliptic.P256())
	kek := randKey(t, 32)

	t.Run("EncryptElement", func(t *testing.T) {
		doc := mustParseXML(t, samlAssertion)
		_, err := agreementAndKeyWrapEncryptor(&ecKey.PublicKey, kek).
			EncryptElement(t.Context(), doc.DocumentElement())
		require.ErrorIs(t, err, xmlenc1.ErrConflictingKeyConfig)

		// Both mechanisms declare the same key wrap URI, so the message must
		// name the two setters: naming algorithms would not say which two
		// things to choose between.
		require.Contains(t, err.Error(), "RecipientECPublicKey")
		require.Contains(t, err.Error(), "KeyEncryptionKey")
		require.Contains(t, err.Error(), xmlenc1.AES256KeyWrap)
	})

	t.Run("EncryptContent", func(t *testing.T) {
		doc := mustParseXML(t, samlAssertion)
		_, err := agreementAndKeyWrapEncryptor(&ecKey.PublicKey, kek).
			EncryptContent(t.Context(), doc.DocumentElement())
		require.ErrorIs(t, err, xmlenc1.ErrConflictingKeyConfig)
	})

	t.Run("EncryptBytes", func(t *testing.T) {
		_, err := agreementAndKeyWrapEncryptor(&ecKey.PublicKey, kek).
			EncryptBytes(t.Context(), helium.NewDefaultDocument(), []byte("payload"))
		require.ErrorIs(t, err, xmlenc1.ErrConflictingKeyConfig)
	})

	t.Run("key wrap alone round-trips", func(t *testing.T) {
		doc := mustParseXML(t, samlAssertion)
		edElem, err := xmlenc1.NewEncryptor().
			KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
			KeyEncryptionKey(kek).
			EncryptElement(t.Context(), doc.DocumentElement())
		require.NoError(t, err)

		nodes, err := xmlenc1.NewDecryptor().KeyEncryptionKey(kek).Decrypt(t.Context(), edElem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
	})

	// A session key alongside key agreement is not a conflict: it supplies
	// the key that the agreed key-encryption key then wraps.
	t.Run("session key with key agreement is allowed", func(t *testing.T) {
		doc := mustParseXML(t, samlAssertion)
		edElem, err := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES256GCM).
			SessionKey(randKey(t, 32)).
			KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
			RecipientECPublicKey(&ecKey.PublicKey).
			EncryptElement(t.Context(), doc.DocumentElement())
		require.NoError(t, err)

		nodes, err := xmlenc1.NewDecryptor().ECPrivateKey(ecKey).Decrypt(t.Context(), edElem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
	})
}
