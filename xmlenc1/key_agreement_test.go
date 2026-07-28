package xmlenc1_test

import (
	"bytes"
	"crypto/elliptic"
	"encoding/hex"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmlenc1"
	"github.com/stretchr/testify/require"
)

func TestDecryptBytes(t *testing.T) {
	key := randKey(t, 32)
	plaintext := []byte{0x00, 0x01, 0x02, 0xff, 0xfe}
	cipherValue, err := xmlenc1.EncryptBytesForTest(xmlenc1.AES256GCM11, key, plaintext)
	require.NoError(t, err)

	doc := mustParseXML(t, `<root/>`)
	ed := &xmlenc1.EncryptedData{
		Type:             xmlenc1.TypeElement,
		EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES256GCM11},
		CipherValue:      cipherValue,
	}
	edElem, err := xmlenc1.MarshalEncryptedDataForTest(doc, ed)
	require.NoError(t, err)

	got, err := xmlenc1.NewDecryptor().SessionKey(key).DecryptBytes(t.Context(), edElem)
	require.NoError(t, err)
	require.True(t, bytes.Equal(plaintext, got))
}

func TestParseECDHAgreement(t *testing.T) {
	const publicKey = "BGoi18CKzCb0K8RTLj0pEFAj2MWDDilMemPZWlOIxS1WbAiR7UrhIstegXjYfIBEYXmnXZFEx/1Ns+yakYTm/B8="
	xml := `<xenc:EncryptedData xmlns:xenc="` + xmlenc1.NamespaceXMLEnc + `" xmlns:ds="` + xmlenc1.NamespaceDSig + `" xmlns:dsig11="` + xmlenc1.NamespaceDSig11 + `" xmlns:xenc11="` + xmlenc1.NamespaceXMLEnc11 + `">` +
		`<xenc:EncryptionMethod Algorithm="` + xmlenc1.AES128GCM11 + `"/>` +
		`<ds:KeyInfo><xenc:EncryptedKey><xenc:EncryptionMethod Algorithm="` + xmlenc1.AES128KeyWrap + `"/>` +
		`<ds:KeyInfo><xenc:AgreementMethod Algorithm="` + xmlenc1.ECDHES + `">` +
		`<xenc11:KeyDerivationMethod Algorithm="` + xmlenc1.ConcatKDF + `"><xenc11:ConcatKDFParams AlgorithmID="" PartyUInfo="00b9e13a70c35edcb3b66fda86b4898942" PartyVInfo=""><ds:DigestMethod Algorithm="` + xmlenc1.DigestSHA256 + `"/></xenc11:ConcatKDFParams></xenc11:KeyDerivationMethod>` +
		`<xenc:OriginatorKeyInfo><ds:KeyValue><dsig11:ECKeyValue><dsig11:NamedCurve URI="urn:oid:1.2.840.10045.3.1.7"/><dsig11:PublicKey>` + publicKey + `</dsig11:PublicKey></dsig11:ECKeyValue></ds:KeyValue></xenc:OriginatorKeyInfo>` +
		`</xenc:AgreementMethod></ds:KeyInfo><xenc:CipherData><xenc:CipherValue>AAAA</xenc:CipherValue></xenc:CipherData></xenc:EncryptedKey></ds:KeyInfo>` +
		`<xenc:CipherData><xenc:CipherValue>AAAA</xenc:CipherValue></xenc:CipherData></xenc:EncryptedData>`
	doc := mustParseXML(t, xml)
	ed, err := xmlenc1.ParseEncryptedDataForTest(doc.DocumentElement())
	require.NoError(t, err)
	require.Len(t, ed.EncryptedKeys, 1)
	require.Equal(t, xmlenc1.ECDHES, ed.EncryptedKeys[0].AgreementMethod.Algorithm)
	require.Equal(t, xmlenc1.ConcatKDF, ed.EncryptedKeys[0].AgreementMethod.KeyDerivationMethod.Algorithm)
	require.Equal(t, []byte{0xb9, 0xe1, 0x3a, 0x70, 0xc3, 0x5e, 0xdc, 0xb3, 0xb6, 0x6f, 0xda, 0x86, 0xb4, 0x89, 0x89, 0x42}, ed.EncryptedKeys[0].AgreementMethod.KeyDerivationMethod.ConcatKDF.PartyUInfo)
}

// ecdhESDocument builds an ECDH-ES EncryptedData whose ConcatKDFParams carry
// the given hexBinary OtherInfo attributes. The wrapped key and ciphertext are
// junk: every case here is rejected before any crypto runs.
func ecdhESDocument(otherInfo map[string]string) string {
	const publicKey = "BGoi18CKzCb0K8RTLj0pEFAj2MWDDilMemPZWlOIxS1WbAiR7UrhIstegXjYfIBEYXmnXZFEx/1Ns+yakYTm/B8="
	var attrs strings.Builder
	for _, name := range []string{"AlgorithmID", "PartyUInfo", "PartyVInfo", "SuppPubInfo", "SuppPrivInfo"} {
		value, ok := otherInfo[name]
		if !ok {
			continue
		}
		attrs.WriteString(` ` + name + `="` + value + `"`)
	}
	return `<xenc:EncryptedData xmlns:xenc="` + xmlenc1.NamespaceXMLEnc + `" xmlns:ds="` + xmlenc1.NamespaceDSig + `" xmlns:dsig11="` + xmlenc1.NamespaceDSig11 + `" xmlns:xenc11="` + xmlenc1.NamespaceXMLEnc11 + `">` +
		`<xenc:EncryptionMethod Algorithm="` + xmlenc1.AES128GCM11 + `"/>` +
		`<ds:KeyInfo><xenc:EncryptedKey><xenc:EncryptionMethod Algorithm="` + xmlenc1.AES128KeyWrap + `"/>` +
		`<ds:KeyInfo><xenc:AgreementMethod Algorithm="` + xmlenc1.ECDHES + `">` +
		`<xenc11:KeyDerivationMethod Algorithm="` + xmlenc1.ConcatKDF + `"><xenc11:ConcatKDFParams` + attrs.String() + `><ds:DigestMethod Algorithm="` + xmlenc1.DigestSHA256 + `"/></xenc11:ConcatKDFParams></xenc11:KeyDerivationMethod>` +
		`<xenc:OriginatorKeyInfo><ds:KeyValue><dsig11:ECKeyValue><dsig11:NamedCurve URI="urn:oid:1.2.840.10045.3.1.7"/><dsig11:PublicKey>` + publicKey + `</dsig11:PublicKey></dsig11:ECKeyValue></ds:KeyValue></xenc:OriginatorKeyInfo>` +
		`</xenc:AgreementMethod></ds:KeyInfo><xenc:CipherData><xenc:CipherValue>AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA</xenc:CipherValue></xenc:CipherData></xenc:EncryptedKey></ds:KeyInfo>` +
		`<xenc:CipherData><xenc:CipherValue>AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA</xenc:CipherValue></xenc:CipherData></xenc:EncryptedData>`
}

// hexOtherInfoField renders n octets of OtherInfo in the on-the-wire form:
// a leading unused-bit octet followed by the field's own bytes.
func hexOtherInfoField(n int) string {
	return hex.EncodeToString(make([]byte, n+1))
}

// boundaryOtherInfo builds the two-attribute shape the budget-boundary cases
// share: an AlgorithmID and a PartyUInfo whose decoded lengths add up to the
// total under test, each short enough that only the cumulative check can
// refuse the set.
func boundaryOtherInfo(algorithmID, partyUInfo int) map[string]string {
	return map[string]string{
		"AlgorithmID": hexOtherInfoField(algorithmID),
		"PartyUInfo":  hexOtherInfoField(partyUInfo),
	}
}

// A remote peer supplies the EncryptedData, and nothing else in xmlenc1 bounds
// the ConcatKDF OtherInfo fields: the only ceiling is whatever the XML parser
// happened to allow for an attribute value, which a caller can raise, disable,
// or bypass entirely by handing xmlenc1 a DOM it built itself. The packing
// those fields feed costs one iteration per input BIT, so this has to be
// refused before any of it runs.
func TestDecryptRejectsOversizedConcatKDFOtherInfo(t *testing.T) {
	priv := generateECKey(t, elliptic.P256())

	tests := []struct {
		name      string
		otherInfo map[string]string
	}{
		{
			// The budget is on the five fields together, so no single
			// field has to be remarkable for the set to be refused.
			name: "cumulative over budget",
			otherInfo: map[string]string{
				"AlgorithmID":  hexOtherInfoField(1000),
				"PartyUInfo":   hexOtherInfoField(1000),
				"PartyVInfo":   hexOtherInfoField(1000),
				"SuppPubInfo":  hexOtherInfoField(1000),
				"SuppPrivInfo": hexOtherInfoField(1000),
			},
		},
		{
			name:      "one field over budget",
			otherInfo: map[string]string{"PartyUInfo": hexOtherInfoField(64 << 10)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := mustParseXML(t, ecdhESDocument(tt.otherInfo))
			_, err := xmlenc1.NewDecryptor().ECPrivateKey(priv).Decrypt(t.Context(), doc.DocumentElement())
			require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
		})
	}

	// A document whose OtherInfo lands exactly on the budget is still parsed
	// and still reaches the key agreement; it fails on the junk wrapped key,
	// not on the budget. 4095 + 1 is the largest legal total, so this pins
	// the accept side of the boundary at its strongest point.
	t.Run("at budget", func(t *testing.T) {
		doc := mustParseXML(t, ecdhESDocument(boundaryOtherInfo(4095, 1)))
		_, err := xmlenc1.NewDecryptor().ECPrivateKey(priv).Decrypt(t.Context(), doc.DocumentElement())
		require.Error(t, err)
		require.NotErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
		require.ErrorIs(t, err, xmlenc1.ErrDecryptionFailed)
	})

	// One byte past the budget is the reject side of the same boundary, and
	// it has to be the CUMULATIVE check that refuses it: neither attribute is
	// long enough for the per-attribute hex-length precheck to fire (4095
	// value octets plus the unused-bit octet encode to 8192 hex characters,
	// under that gate), so a budget that only rejected a single huge field
	// would accept this set.
	// Only the per-attribute gate says a field is over the limit "alone", so
	// its absence from the message is what shows the cumulative check fired.
	t.Run("one byte over budget", func(t *testing.T) {
		doc := mustParseXML(t, ecdhESDocument(boundaryOtherInfo(4095, 2)))
		_, err := xmlenc1.NewDecryptor().ECPrivateKey(priv).Decrypt(t.Context(), doc.DocumentElement())
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
		require.Contains(t, err.Error(), "OtherInfo")
		require.NotContains(t, err.Error(), "alone")
	})
}

// An over-budget parameter set has to be refused for its size even when the
// same set names a digest this package does not implement. The size is the
// reason the set is unusable at all, and a caller that fixes the digest would
// otherwise meet the budget error only on the next attempt.
func TestEncryptRejectsOversizedOtherInfoWhateverTheDigest(t *testing.T) {
	key := generateECKey(t, elliptic.P256())

	tests := []struct {
		name   string
		digest string
	}{
		{name: "supported digest", digest: xmlenc1.DigestSHA256},
		{name: "unsupported digest", digest: "http://example.com/bogus-digest"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := mustParseXML(t, samlAssertion)
			_, err := xmlenc1.NewEncryptor().
				BlockAlgorithm(xmlenc1.AES256GCM11).
				KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
				RecipientECPublicKey(&key.PublicKey).
				KeyDerivationParams(&xmlenc1.ConcatKDFParams{
					DigestMethod: tt.digest,
					AlgorithmID:  make([]byte, 5000),
				}).
				EncryptElement(t.Context(), doc.DocumentElement())
			require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
		})
	}
}

// The OtherInfo budget and the octet-oriented packing must leave an ordinary
// ECDH-ES exchange exactly as it was, including one carrying the kind of
// AlgorithmID/PartyInfo values a real deployment sets.
func TestEncryptECDHESRoundTripWithOtherInfo(t *testing.T) {
	key := generateECKey(t, elliptic.P256())
	doc := mustParseXML(t, samlAssertion)

	params := &xmlenc1.ConcatKDFParams{
		DigestMethod: xmlenc1.DigestSHA256,
		AlgorithmID:  []byte(xmlenc1.AES256GCM11),
		PartyUInfo:   []byte("urn:example:originator"),
		PartyVInfo:   []byte("urn:example:recipient"),
		SuppPubInfo:  []byte{0x00, 0x00, 0x01, 0x00},
	}

	edElem, err := xmlenc1.NewEncryptor().
		BlockAlgorithm(xmlenc1.AES256GCM11).
		KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
		RecipientECPublicKey(&key.PublicKey).
		KeyDerivationParams(params).
		EncryptElement(t.Context(), doc.DocumentElement())
	require.NoError(t, err)

	nodes, err := xmlenc1.NewDecryptor().ECPrivateKey(key).Decrypt(t.Context(), edElem)
	require.NoError(t, err)
	require.Len(t, nodes, 1)

	decrypted, err := helium.WriteString(nodes[0])
	require.NoError(t, err)
	require.Contains(t, decrypted, "user@example.com")

	// The OtherInfo travelled on the wire and parsed back unchanged, which is
	// what lets a different implementation reach the same KEK.
	parsed, err := xmlenc1.ParseEncryptedDataForTest(edElem)
	require.NoError(t, err)
	require.Len(t, parsed.EncryptedKeys, 1)
	wire := parsed.EncryptedKeys[0].AgreementMethod.KeyDerivationMethod.ConcatKDF
	require.Equal(t, params.AlgorithmID, wire.AlgorithmID)
	require.Equal(t, params.PartyUInfo, wire.PartyUInfo)
	require.Equal(t, params.PartyVInfo, wire.PartyVInfo)
	require.Equal(t, params.SuppPubInfo, wire.SuppPubInfo)
	require.Equal(t, params.DigestMethod, wire.DigestMethod)
}
