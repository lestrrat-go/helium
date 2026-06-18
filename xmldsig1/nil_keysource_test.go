package xmldsig1_test

import (
	"context"
	"testing"

	"github.com/lestrrat-go/helium/xmldsig1"
	"github.com/stretchr/testify/require"
)

// pointerKeySource is a pointer-receiver KeySource used to construct a typed-nil
// pointer value. A typed-nil pointer carries a concrete type, so the interface
// is non-nil and survives a plain == nil check; ResolveKey would then be called
// on a nil receiver.
type pointerKeySource struct{}

func (*pointerKeySource) ResolveKey(_ context.Context, _ *xmldsig1.KeyInfoData, _ string) (any, error) {
	return nil, nil
}

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

// A typed-nil POINTER KeySource (e.g. var ks *pointerKeySource;
// NewVerifier(ks)) yields a non-nil interface whose underlying value is nil, so
// a plain cfg.keySource == nil check misses it and ResolveKey would panic on
// the nil receiver. verifySignature must detect this via isNilKeySource and
// return a typed error instead.
func TestTypedNilPointerKeySource(t *testing.T) {
	doc := mustParseXML(t, nilKeySourceSigDoc)

	var ks *pointerKeySource // typed-nil pointer

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
