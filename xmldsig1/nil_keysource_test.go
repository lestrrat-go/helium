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
