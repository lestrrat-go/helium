package xmldsig1_test

import (
	"context"
	"crypto/dsa" // verify-only legacy interop for the merlin DSA-SHA1 vector
	"os"
	"path/filepath"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmldsig1"
	"github.com/stretchr/testify/require"
)

// TestBase64TransformMerlinVector verifies the W3C/Baltimore merlin interop
// vector signature-enveloping-b64-dsa.xml end to end through the public API. The
// signature is an enveloping DSA-SHA1 signature whose single Reference
// (URI="#object") applies the base64 decode transform to a ds:Object holding
// base64 text: the Object's text decodes to "some text" and the SHA-1 digest of
// those decoded octets must match the recorded DigestValue. The DSA public key is
// carried inline as a DSAKeyValue and rebuilt by the KeySource, so the vector
// exercises the base64 transform, DSAKeyValue parsing, and the DSA-SHA1 verify
// path together. SHA-1 is opted in (AllowSHA1) because the vector predates the
// SHA-1 deprecation.
func TestBase64TransformMerlinVector(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "interop", "signature-enveloping-b64-dsa.xml"))
	require.NoError(t, err)
	doc, err := helium.NewParser().Parse(t.Context(), data)
	require.NoError(t, err)

	ks := xmldsig1.KeySourceFunc(func(_ context.Context, ki *xmldsig1.KeyInfoData, _ string) (any, error) {
		require.NotNil(t, ki.DSAKeyValue, "the inline DSAKeyValue must be parsed into KeyInfoData")
		return &dsa.PublicKey{
			Parameters: dsa.Parameters{P: ki.DSAKeyValue.P, Q: ki.DSAKeyValue.Q, G: ki.DSAKeyValue.G},
			Y:          ki.DSAKeyValue.Y,
		}, nil
	})

	res, err := xmldsig1.NewVerifier(ks).AllowSHA1(true).Verify(t.Context(), doc)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Len(t, res.References, 1)
	require.Equal(t, "#object", res.References[0].URI)
	require.Equal(t, "Object", res.References[0].Element.LocalName(),
		"the base64 Reference resolves to the ds:Object element")
}
