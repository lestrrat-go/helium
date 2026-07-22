package xmldsig1_test

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"os"
	"testing"

	"github.com/lestrrat-go/helium/xmldsig1"
	"github.com/stretchr/testify/require"
)

// errStopResolve lets a capturing KeySource halt verification right after
// KeyInfo has been parsed, so a test can assert on the extracted KeyInfoData
// without needing real key material to complete a cryptographic verify.
var errStopResolve = errors.New("stop after keyinfo capture")

// TestRFC4050ECDSAKeyValueVerify verifies a real Apache Santuario interop
// vector whose KeyInfo carries an RFC 4050 ECDSAKeyValue (decimal X/Y in the
// http://www.w3.org/2001/04/xmldsig-more# namespace). The KeySource rebuilds
// the *ecdsa.PublicKey from the parsed KeyInfoData.ECKeyValue, and the existing
// ecdsa-sha256 verify path then checks the signature end-to-end.
func TestRFC4050ECDSAKeyValueVerify(t *testing.T) {
	raw, err := os.ReadFile("testdata/rfc4050_ecdsa_p256_sha256.xml")
	require.NoError(t, err)
	doc := mustParseXML(t, string(raw))

	ks := xmldsig1.KeySourceFunc(func(_ context.Context, ki *xmldsig1.KeyInfoData, _ string) (any, error) {
		require.NotNil(t, ki)
		require.NotNil(t, ki.ECKeyValue, "RFC 4050 ECDSAKeyValue must surface as ECKeyValue")
		require.NotNil(t, ki.ECKeyValue.Curve)
		require.NotNil(t, ki.ECKeyValue.X)
		require.NotNil(t, ki.ECKeyValue.Y)
		return &ecdsa.PublicKey{
			Curve: ki.ECKeyValue.Curve,
			X:     ki.ECKeyValue.X,
			Y:     ki.ECKeyValue.Y,
		}, nil
	})

	res, err := xmldsig1.NewVerifier(ks).Verify(t.Context(), doc)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Len(t, res.References, 1)
}

// TestRFC4050UnknownCurve confirms an ECDSAKeyValue naming a curve URN the
// library does not support is rejected with ErrInvalidKeyInfo during KeyInfo
// parsing, before any key resolution.
func TestRFC4050UnknownCurve(t *testing.T) {
	const src = `<Signature xmlns="http://www.w3.org/2000/09/xmldsig#">` +
		`<SignedInfo>` +
		`<CanonicalizationMethod Algorithm="http://www.w3.org/TR/2001/REC-xml-c14n-20010315"/>` +
		`<SignatureMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#ecdsa-sha256"/>` +
		`<Reference URI="#o"><DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"/><DigestValue>AAAA</DigestValue></Reference>` +
		`</SignedInfo>` +
		`<SignatureValue>AAAA</SignatureValue>` +
		`<KeyInfo><KeyValue>` +
		`<ECDSAKeyValue xmlns="http://www.w3.org/2001/04/xmldsig-more#">` +
		`<DomainParameters><NamedCurve URN="urn:oid:1.2.3.4.5"/></DomainParameters>` +
		`<PublicKey><X Value="1"/><Y Value="2"/></PublicKey>` +
		`</ECDSAKeyValue>` +
		`</KeyValue></KeyInfo>` +
		`<Object Id="o">x</Object>` +
		`</Signature>`
	doc := mustParseXML(t, src)

	ks := xmldsig1.StaticKey(nil)
	_, err := xmldsig1.NewVerifier(ks).Verify(t.Context(), doc)
	require.ErrorIs(t, err, xmldsig1.ErrInvalidKeyInfo)
}

// TestX509SubjectNameExtraction reads a real W3C dname interop vector and
// confirms the X509SubjectName distinguished-name string is extracted verbatim
// into KeyInfoData for out-of-band certificate selection.
func TestX509SubjectNameExtraction(t *testing.T) {
	raw, err := os.ReadFile("testdata/dname_dsa_sha1_subjectname.xml")
	require.NoError(t, err)
	doc := mustParseXML(t, string(raw))

	var captured *xmldsig1.KeyInfoData
	ks := xmldsig1.KeySourceFunc(func(_ context.Context, ki *xmldsig1.KeyInfoData, _ string) (any, error) {
		captured = ki
		return nil, errStopResolve
	})

	// AllowSHA1(true) so the dsa-sha1 weak gate does not short-circuit before
	// KeyInfo is parsed and handed to the KeySource.
	_, err = xmldsig1.NewVerifier(ks).AllowSHA1(true).Verify(t.Context(), doc)
	require.ErrorIs(t, err, errStopResolve)
	require.NotNil(t, captured)
	require.Equal(t, []string{"CN=John,C=US"}, captured.X509SubjectNames)
}

// TestX509IssuerSerialExtraction confirms X509IssuerSerial (issuer DN + serial)
// is extracted into KeyInfoData for out-of-band certificate selection.
func TestX509IssuerSerialExtraction(t *testing.T) {
	const src = `<Signature xmlns="http://www.w3.org/2000/09/xmldsig#">` +
		`<SignedInfo>` +
		`<CanonicalizationMethod Algorithm="http://www.w3.org/TR/2001/REC-xml-c14n-20010315"/>` +
		`<SignatureMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#ecdsa-sha256"/>` +
		`<Reference URI="#o"><DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"/><DigestValue>AAAA</DigestValue></Reference>` +
		`</SignedInfo>` +
		`<SignatureValue>AAAA</SignatureValue>` +
		`<KeyInfo><X509Data>` +
		`<X509IssuerSerial>` +
		`<X509IssuerName>CN=Test CA,O=Example,C=US</X509IssuerName>` +
		`<X509SerialNumber>1234567890</X509SerialNumber>` +
		`</X509IssuerSerial>` +
		`</X509Data></KeyInfo>` +
		`<Object Id="o">x</Object>` +
		`</Signature>`
	doc := mustParseXML(t, src)

	var captured *xmldsig1.KeyInfoData
	ks := xmldsig1.KeySourceFunc(func(_ context.Context, ki *xmldsig1.KeyInfoData, _ string) (any, error) {
		captured = ki
		return nil, errStopResolve
	})

	_, err := xmldsig1.NewVerifier(ks).Verify(t.Context(), doc)
	require.ErrorIs(t, err, errStopResolve)
	require.NotNil(t, captured)
	require.Len(t, captured.X509IssuerSerials, 1)
	require.Equal(t, "CN=Test CA,O=Example,C=US", captured.X509IssuerSerials[0].IssuerName)
	require.NotNil(t, captured.X509IssuerSerials[0].SerialNumber)
	require.Equal(t, "1234567890", captured.X509IssuerSerials[0].SerialNumber.String())
}

// TestDSASignUnsupported confirms a signing attempt with the DSA-SHA1 URI fails
// with a clear "DSA signing is not supported" error rather than a generic
// key-type mismatch or map miss. AllowSHA1(true) is set so the weak-algorithm
// gate does not mask the DSA-specific rejection.
func TestDSASignUnsupported(t *testing.T) {
	key := generateRSAKey(t)
	doc := mustParseXML(t, samlAssertion)

	signer := xmldsig1.NewSigner().
		AllowSHA1(true).
		SignatureAlgorithm(xmldsig1.AlgDSASHA1).
		Reference(xmldsig1.NewEnvelopedReference())
	err := signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key)
	require.ErrorIs(t, err, xmldsig1.ErrUnsupportedAlgorithm)
	require.Contains(t, err.Error(), "DSA signing")
}

// TestDSAVerifyWeakGated confirms DSA-SHA1 verification is rejected with
// ErrWeakAlgorithm unless the caller opts in via Verifier.AllowSHA1(true),
// exactly like rsa-sha1.
func TestDSAVerifyWeakGated(t *testing.T) {
	raw, err := os.ReadFile("testdata/dname_dsa_sha1_subjectname.xml")
	require.NoError(t, err)
	doc := mustParseXML(t, string(raw))

	_, err = xmldsig1.NewVerifier(xmldsig1.StaticKey(nil)).Verify(t.Context(), doc)
	require.ErrorIs(t, err, xmldsig1.ErrWeakAlgorithm)
}
