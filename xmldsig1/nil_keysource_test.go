package xmldsig1_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xmldsig1"
	"github.com/stretchr/testify/require"
)

// A nil KeySource must surface a typed error rather than panic on a nil
// pointer dereference when Verify reaches ResolveKey.
func TestNilKeySource(t *testing.T) {
	const sigDoc = `<doc xmlns:ds="http://www.w3.org/2000/09/xmldsig#">` +
		`<ds:Signature>` +
		`<ds:SignedInfo>` +
		`<ds:CanonicalizationMethod Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"/>` +
		`<ds:SignatureMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"/>` +
		`<ds:Reference URI="">` +
		`<ds:DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"/>` +
		`<ds:DigestValue>AAAA</ds:DigestValue>` +
		`</ds:Reference>` +
		`</ds:SignedInfo>` +
		`<ds:SignatureValue>AAAA</ds:SignatureValue>` +
		`</ds:Signature>` +
		`</doc>`

	doc := mustParseXML(t, sigDoc)

	require.NotPanics(t, func() {
		_, err := xmldsig1.NewVerifier(nil).Verify(t.Context(), doc)
		require.Error(t, err)
		require.ErrorIs(t, err, xmldsig1.ErrNoKeySource)
	})
}

// A typed-nil KeySourceFunc passes the interface!=nil check in verifySignature
// (the interface carries a concrete type), so ResolveKey must guard the nil
// func itself and return a typed error rather than panic on the nil call.
func TestTypedNilKeySourceFunc(t *testing.T) {
	doc := mustParseXML(t, nilKeySourceSigDoc)

	var ks xmldsig1.KeySourceFunc // typed nil

	require.NotPanics(t, func() {
		_, err := xmldsig1.NewVerifier(ks).Verify(t.Context(), doc)
		require.Error(t, err)
		require.ErrorIs(t, err, xmldsig1.ErrNoKeySource)
	})
}

// A zero-value Verifier{} constructed directly (bypassing NewVerifier) has a
// nil cfg. Verify/VerifyElement must surface a typed error rather than panic on
// the nil cfg dereference inside verifySignature.
func TestZeroValueVerifier(t *testing.T) {
	doc := mustParseXML(t, nilKeySourceSigDoc)

	require.NotPanics(t, func() {
		_, err := xmldsig1.Verifier{}.Verify(t.Context(), doc)
		require.Error(t, err)
		require.ErrorIs(t, err, xmldsig1.ErrNoKeySource)
	})
}

// nilKeySourceSigDoc is a minimal document carrying a single ds:Signature so
// the verify path reaches the key-resolution step.
const nilKeySourceSigDoc = `<doc xmlns:ds="http://www.w3.org/2000/09/xmldsig#">` +
	`<ds:Signature>` +
	`<ds:SignedInfo>` +
	`<ds:CanonicalizationMethod Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"/>` +
	`<ds:SignatureMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"/>` +
	`<ds:Reference URI="">` +
	`<ds:DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"/>` +
	`<ds:DigestValue>AAAA</ds:DigestValue>` +
	`</ds:Reference>` +
	`</ds:SignedInfo>` +
	`<ds:SignatureValue>AAAA</ds:SignatureValue>` +
	`</ds:Signature>` +
	`</doc>`
