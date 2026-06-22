package xmldsig1

import (
	"errors"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestVerificationErrorFormat covers both branches of VerificationError.Error
// (signature-value failure with Reference < 0, and a per-Reference failure).
func TestVerificationErrorFormat(t *testing.T) {
	cause := errors.New("boom")

	sigErr := &VerificationError{Reference: -1, Err: cause}
	require.Contains(t, sigErr.Error(), "signature value verification failed")
	require.ErrorIs(t, sigErr, cause)

	refErr := &VerificationError{Reference: 2, URI: "#x", Err: cause}
	msg := refErr.Error()
	require.Contains(t, msg, "reference 2")
	require.Contains(t, msg, "#x")
	require.ErrorIs(t, refErr, cause)
}

// TestExcC14NTransformPrefixes covers excC14NTransform.Prefixes.
func TestExcC14NTransformPrefixes(t *testing.T) {
	tr := ExcC14NTransform("a", "b")
	exc, ok := tr.(excC14NTransform)
	require.True(t, ok)
	require.Equal(t, []string{"a", "b"}, exc.Prefixes())
}

func mustParse(t *testing.T, xml string) *helium.Document {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)
	return doc
}

func ecElem(t *testing.T, doc *helium.Document, inner string) *helium.Element {
	t.Helper()
	full := `<dsig11:ECKeyValue xmlns:dsig11="` + NamespaceDSig11 + `">` + inner + `</dsig11:ECKeyValue>`
	d := mustParse(t, full)
	return d.DocumentElement()
}

// TestParseECKeyValueUnsupportedCurve covers the unsupported-curve branch.
func TestParseECKeyValueUnsupportedCurve(t *testing.T) {
	doc := mustParse(t, "<root/>")
	elem := ecElem(t, doc, `<dsig11:NamedCurve xmlns:dsig11="`+NamespaceDSig11+`" URI="urn:oid:bogus"/>`)
	var data KeyInfoData
	err := parseECKeyValue(elem, &data)
	require.ErrorIs(t, err, ErrInvalidKeyInfo)
	require.Contains(t, err.Error(), "unsupported EC curve")
}

// TestParseECKeyValueMissingCurve covers the PublicKey-before-NamedCurve branch.
func TestParseECKeyValueMissingCurve(t *testing.T) {
	elem := ecElem(t, nil, `<dsig11:PublicKey xmlns:dsig11="`+NamespaceDSig11+`">BBBB</dsig11:PublicKey>`)
	var data KeyInfoData
	err := parseECKeyValue(elem, &data)
	require.ErrorIs(t, err, ErrInvalidKeyInfo)
	require.Contains(t, err.Error(), "missing NamedCurve")
}

// TestParseECKeyValueInvalidPoint covers the invalid-point branch: a valid
// curve but a PublicKey blob that does not unmarshal to a point.
func TestParseECKeyValueInvalidPoint(t *testing.T) {
	inner := `<dsig11:NamedCurve xmlns:dsig11="` + NamespaceDSig11 + `" URI="urn:oid:1.2.840.10045.3.1.7"/>` +
		`<dsig11:PublicKey xmlns:dsig11="` + NamespaceDSig11 + `">AAAA</dsig11:PublicKey>`
	elem := ecElem(t, nil, inner)
	var data KeyInfoData
	err := parseECKeyValue(elem, &data)
	require.ErrorIs(t, err, ErrInvalidKeyInfo)
	require.Contains(t, err.Error(), "invalid EC public key point")
}

// TestParseECKeyValueBadBase64 covers the base64-decode error branch.
func TestParseECKeyValueBadBase64(t *testing.T) {
	inner := `<dsig11:NamedCurve xmlns:dsig11="` + NamespaceDSig11 + `" URI="urn:oid:1.2.840.10045.3.1.7"/>` +
		`<dsig11:PublicKey xmlns:dsig11="` + NamespaceDSig11 + `">!!!!notbase64</dsig11:PublicKey>`
	elem := ecElem(t, nil, inner)
	var data KeyInfoData
	err := parseECKeyValue(elem, &data)
	require.ErrorIs(t, err, ErrInvalidKeyInfo)
}

// TestParseRSAKeyValueExponentOutOfRange covers the exponent-range guard.
func TestParseRSAKeyValueExponentOutOfRange(t *testing.T) {
	// Exponent of 0 (base64 "AA==" decodes to a single 0 byte -> Sign() == 0).
	inner := `<ds:Modulus xmlns:ds="` + NamespaceDSig + `">AQAB</ds:Modulus>` +
		`<ds:Exponent xmlns:ds="` + NamespaceDSig + `">AA==</ds:Exponent>`
	d := mustParse(t, `<ds:RSAKeyValue xmlns:ds="`+NamespaceDSig+`">`+inner+`</ds:RSAKeyValue>`)
	var data KeyInfoData
	err := parseRSAKeyValue(d.DocumentElement(), &data)
	require.ErrorIs(t, err, ErrInvalidKeyInfo)
	require.Contains(t, err.Error(), "out of range")
}

// TestParseRSAKeyValueBadBase64 covers the base64 error branch.
func TestParseRSAKeyValueBadBase64(t *testing.T) {
	d := mustParse(t, `<ds:RSAKeyValue xmlns:ds="`+NamespaceDSig+`"><ds:Modulus xmlns:ds="`+NamespaceDSig+`">!!!</ds:Modulus></ds:RSAKeyValue>`)
	var data KeyInfoData
	err := parseRSAKeyValue(d.DocumentElement(), &data)
	require.ErrorIs(t, err, ErrInvalidKeyInfo)
}

// TestResolveC14NModeUnsupported covers the default (error) arm.
func TestResolveC14NModeUnsupported(t *testing.T) {
	_, _, err := resolveC14NMode("urn:not-a-c14n-method")
	require.ErrorIs(t, err, ErrUnsupportedAlgorithm)
}

// TestResolveC14NModeComments covers the comment-variant arms.
func TestResolveC14NModeComments(t *testing.T) {
	for _, m := range []string{C14N10Comments, ExcC14N10Comments, C14N11Comments} {
		_, comments, err := resolveC14NMode(m)
		require.NoError(t, err)
		require.True(t, comments)
	}
}

// TestKeySourceFuncNil covers the typed-nil guard in KeySourceFunc.ResolveKey.
func TestKeySourceFuncNil(t *testing.T) {
	var f KeySourceFunc
	_, err := f.ResolveKey(t.Context(), nil, "")
	require.ErrorIs(t, err, ErrNoKeySource)
}

// TestParseX509DataBadCert covers the invalid-base64 and invalid-cert branches.
func TestParseX509DataBadCert(t *testing.T) {
	// well-formed base64 but not a certificate.
	d := mustParse(t, `<ds:X509Data xmlns:ds="`+NamespaceDSig+`"><ds:X509Certificate xmlns:ds="`+NamespaceDSig+`">aGVsbG8=</ds:X509Certificate></ds:X509Data>`)
	var data KeyInfoData
	err := parseX509Data(d.DocumentElement(), &data)
	require.ErrorIs(t, err, ErrInvalidKeyInfo)

	d2 := mustParse(t, `<ds:X509Data xmlns:ds="`+NamespaceDSig+`"><ds:X509Certificate xmlns:ds="`+NamespaceDSig+`">!!!bad</ds:X509Certificate></ds:X509Data>`)
	var data2 KeyInfoData
	err = parseX509Data(d2.DocumentElement(), &data2)
	require.ErrorIs(t, err, ErrInvalidKeyInfo)
	require.True(t, strings.Contains(err.Error(), "base64"))
}
