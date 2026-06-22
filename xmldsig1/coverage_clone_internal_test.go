package xmldsig1

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSignerCloneNilConfig covers the s.cfg == nil branch of Signer.clone by
// invoking a builder method on a zero-value Signer.
func TestSignerCloneNilConfig(t *testing.T) {
	var s Signer // zero value: cfg is nil
	s2 := s.SignatureAlgorithm(AlgRSASHA256)
	require.NotNil(t, s2.cfg)
	require.Equal(t, AlgRSASHA256, s2.cfg.signatureAlgorithm)
	require.Equal(t, ExcC14N10, s2.cfg.c14nMethod)
}

// TestVerifierCloneNilConfig covers the v.cfg == nil branch of Verifier.clone.
func TestVerifierCloneNilConfig(t *testing.T) {
	var v Verifier // zero value: cfg is nil
	v2 := v.AllowSHA1(true)
	require.NotNil(t, v2.cfg)
	require.True(t, v2.cfg.allowSHA1)
}

// TestParseKeyValueForeignRSA covers parseKeyValue's continue branch: an
// RSAKeyValue look-alike in a foreign namespace must be skipped, leaving no
// RSAKeyValue parsed.
func TestParseKeyValueForeignRSA(t *testing.T) {
	doc := mustParse(t, `<ds:KeyValue xmlns:ds="`+NamespaceDSig+`"><evil:RSAKeyValue xmlns:evil="urn:evil"/></ds:KeyValue>`)
	var data KeyInfoData
	require.NoError(t, parseKeyValue(doc.DocumentElement(), &data))
	require.Nil(t, data.RSAKeyValue)
}

// TestParseKeyValueForeignEC covers parseKeyValue's ECKeyValue continue branch:
// an ECKeyValue look-alike in a non-dsig11 namespace must be skipped.
func TestParseKeyValueForeignEC(t *testing.T) {
	doc := mustParse(t, `<ds:KeyValue xmlns:ds="`+NamespaceDSig+`"><evil:ECKeyValue xmlns:evil="urn:evil"/></ds:KeyValue>`)
	var data KeyInfoData
	require.NoError(t, parseKeyValue(doc.DocumentElement(), &data))
	require.Nil(t, data.ECKeyValue)
}

// TestParseRSAKeyValueForeignChild covers parseRSAKeyValue's foreign-namespace
// child continue branch.
func TestParseRSAKeyValueForeignChild(t *testing.T) {
	doc := mustParse(t, `<ds:RSAKeyValue xmlns:ds="`+NamespaceDSig+`"><evil:Modulus xmlns:evil="urn:evil">AQAB</evil:Modulus></ds:RSAKeyValue>`)
	var data KeyInfoData
	require.NoError(t, parseRSAKeyValue(doc.DocumentElement(), &data))
	require.NotNil(t, data.RSAKeyValue)
	require.Nil(t, data.RSAKeyValue.Modulus) // foreign Modulus was skipped
}

// TestParseKeyInfoForeignChild covers parseKeyInfo's continue branch for a
// foreign-namespace X509Data look-alike.
func TestParseKeyInfoForeignChild(t *testing.T) {
	doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`"><evil:X509Data xmlns:evil="urn:evil"/></ds:KeyInfo>`)
	data, err := parseKeyInfo(doc.DocumentElement())
	require.NoError(t, err)
	require.Empty(t, data.X509Certificates)
}

// TestParseX509DataForeignCert covers parseX509Data's foreign-namespace
// look-alike continue branch.
func TestParseX509DataForeignCert(t *testing.T) {
	doc := mustParse(t, `<ds:X509Data xmlns:ds="`+NamespaceDSig+`"><evil:X509Certificate xmlns:evil="urn:evil">AA==</evil:X509Certificate></ds:X509Data>`)
	var data KeyInfoData
	require.NoError(t, parseX509Data(doc.DocumentElement(), &data))
	require.Empty(t, data.X509Certificates)
}
