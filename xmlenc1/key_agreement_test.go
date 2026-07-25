package xmlenc1_test

import (
	"bytes"
	"testing"

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
