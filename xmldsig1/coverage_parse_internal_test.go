package xmldsig1

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDigestEqual(t *testing.T) {
	require.True(t, digestEqual([]byte{1, 2, 3}, []byte{1, 2, 3}))
	require.False(t, digestEqual([]byte{1, 2, 3}, []byte{1, 2}))    // length mismatch
	require.False(t, digestEqual([]byte{1, 2, 3}, []byte{1, 2, 4})) // content mismatch
}

const dsigNS = NamespaceDSig

func TestParseSignatureElementMissingSignedInfo(t *testing.T) {
	doc := mustParse(t, `<ds:Signature xmlns:ds="`+dsigNS+`"><ds:SignatureValue xmlns:ds="`+dsigNS+`">AA==</ds:SignatureValue></ds:Signature>`)
	_, err := parseSignatureElement(doc.DocumentElement())
	require.ErrorIs(t, err, ErrInvalidSignature)
	require.Contains(t, err.Error(), "missing SignedInfo")
}

func TestParseSignatureElementMissingSignatureValue(t *testing.T) {
	si := `<ds:SignedInfo xmlns:ds="` + dsigNS + `">` +
		`<ds:CanonicalizationMethod xmlns:ds="` + dsigNS + `" Algorithm="` + ExcC14N10 + `"/>` +
		`<ds:SignatureMethod xmlns:ds="` + dsigNS + `" Algorithm="` + AlgRSASHA256 + `"/>` +
		`<ds:Reference xmlns:ds="` + dsigNS + `" URI="">` +
		`<ds:DigestMethod xmlns:ds="` + dsigNS + `" Algorithm="` + DigestSHA256 + `"/>` +
		`<ds:DigestValue xmlns:ds="` + dsigNS + `">AA==</ds:DigestValue>` +
		`</ds:Reference></ds:SignedInfo>`
	doc := mustParse(t, `<ds:Signature xmlns:ds="`+dsigNS+`">`+si+`</ds:Signature>`)
	_, err := parseSignatureElement(doc.DocumentElement())
	require.ErrorIs(t, err, ErrInvalidSignature)
	require.Contains(t, err.Error(), "missing SignatureValue")
}

func TestParseSignatureElementBadSignatureValueBase64(t *testing.T) {
	si := `<ds:SignedInfo xmlns:ds="` + dsigNS + `">` +
		`<ds:Reference xmlns:ds="` + dsigNS + `" URI="">` +
		`<ds:DigestMethod xmlns:ds="` + dsigNS + `" Algorithm="` + DigestSHA256 + `"/>` +
		`<ds:DigestValue xmlns:ds="` + dsigNS + `">AA==</ds:DigestValue>` +
		`</ds:Reference></ds:SignedInfo>`
	doc := mustParse(t, `<ds:Signature xmlns:ds="`+dsigNS+`">`+si+
		`<ds:SignatureValue xmlns:ds="`+dsigNS+`">!!!bad</ds:SignatureValue></ds:Signature>`)
	_, err := parseSignatureElement(doc.DocumentElement())
	require.ErrorIs(t, err, ErrInvalidSignature)
}

func TestParseSignedInfoMissingC14NAlgorithm(t *testing.T) {
	si := `<ds:SignedInfo xmlns:ds="` + dsigNS + `">` +
		`<ds:CanonicalizationMethod xmlns:ds="` + dsigNS + `"/>` +
		`</ds:SignedInfo>`
	doc := mustParse(t, si)
	var parsed parsedSignature
	err := parseSignedInfo(doc.DocumentElement(), &parsed)
	require.ErrorIs(t, err, ErrInvalidSignature)
	require.Contains(t, err.Error(), "CanonicalizationMethod missing Algorithm")
}

func TestParseSignedInfoMissingSignatureMethodAlgorithm(t *testing.T) {
	si := `<ds:SignedInfo xmlns:ds="` + dsigNS + `">` +
		`<ds:SignatureMethod xmlns:ds="` + dsigNS + `"/>` +
		`</ds:SignedInfo>`
	doc := mustParse(t, si)
	var parsed parsedSignature
	err := parseSignedInfo(doc.DocumentElement(), &parsed)
	require.ErrorIs(t, err, ErrInvalidSignature)
	require.Contains(t, err.Error(), "SignatureMethod missing Algorithm")
}

func TestParseSignedInfoNoReference(t *testing.T) {
	si := `<ds:SignedInfo xmlns:ds="` + dsigNS + `">` +
		`<ds:CanonicalizationMethod xmlns:ds="` + dsigNS + `" Algorithm="` + ExcC14N10 + `"/>` +
		`<ds:SignatureMethod xmlns:ds="` + dsigNS + `" Algorithm="` + AlgRSASHA256 + `"/>` +
		`</ds:SignedInfo>`
	doc := mustParse(t, si)
	var parsed parsedSignature
	err := parseSignedInfo(doc.DocumentElement(), &parsed)
	require.ErrorIs(t, err, ErrInvalidSignature)
	require.Contains(t, err.Error(), "no Reference")
}

func TestParseReferenceElementMissingDigestMethodAlgorithm(t *testing.T) {
	r := `<ds:Reference xmlns:ds="` + dsigNS + `" URI="">` +
		`<ds:DigestMethod xmlns:ds="` + dsigNS + `"/>` +
		`</ds:Reference>`
	doc := mustParse(t, r)
	_, err := parseReferenceElement(doc.DocumentElement())
	require.ErrorIs(t, err, ErrInvalidSignature)
	require.Contains(t, err.Error(), "DigestMethod missing Algorithm")
}

func TestParseReferenceElementBadDigestValueBase64(t *testing.T) {
	r := `<ds:Reference xmlns:ds="` + dsigNS + `" URI="">` +
		`<ds:DigestMethod xmlns:ds="` + dsigNS + `" Algorithm="` + DigestSHA256 + `"/>` +
		`<ds:DigestValue xmlns:ds="` + dsigNS + `">!!!bad</ds:DigestValue>` +
		`</ds:Reference>`
	doc := mustParse(t, r)
	_, err := parseReferenceElement(doc.DocumentElement())
	require.ErrorIs(t, err, ErrInvalidSignature)
}

func TestParseReferenceElementWithInclusiveNamespaces(t *testing.T) {
	const exc = ExcC14N10
	r := `<ds:Reference xmlns:ds="` + dsigNS + `" URI="">` +
		`<ds:Transforms xmlns:ds="` + dsigNS + `">` +
		`<ds:Transform xmlns:ds="` + dsigNS + `" Algorithm="` + exc + `">` +
		`<ec:InclusiveNamespaces xmlns:ec="` + exc + `" PrefixList="a b"/>` +
		`</ds:Transform></ds:Transforms>` +
		`<ds:DigestMethod xmlns:ds="` + dsigNS + `" Algorithm="` + DigestSHA256 + `"/>` +
		`<ds:DigestValue xmlns:ds="` + dsigNS + `">AA==</ds:DigestValue>` +
		`</ds:Reference>`
	doc := mustParse(t, r)
	ref, err := parseReferenceElement(doc.DocumentElement())
	require.NoError(t, err)
	require.Len(t, ref.transforms, 1)
	require.Equal(t, []string{"a", "b"}, ref.transforms[0].prefixes)
}
