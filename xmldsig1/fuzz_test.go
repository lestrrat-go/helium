package xmldsig1

import (
	"context"
	"testing"
	"time"

	helium "github.com/lestrrat-go/helium"
)

// FuzzVerify drives malformed Signature, SignedInfo, Reference, and KeyInfo
// trees through the public verifier. The verifier has no resolver or XSLT
// transformer here, so fuzz inputs cannot cause filesystem, network, or XSLT
// work.
func FuzzVerify(f *testing.F) {
	seeds := []string{
		`<root Id="doc"><ds:Signature xmlns:ds="` + NamespaceDSig + `"><ds:SignedInfo><ds:CanonicalizationMethod Algorithm="` + C14N10 + `"/><ds:SignatureMethod Algorithm="` + AlgHMACSHA256 + `"/><ds:Reference URI="#doc"><ds:Transforms><ds:Transform Algorithm="` + TransformEnvelopedSignature + `"/></ds:Transforms><ds:DigestMethod Algorithm="` + DigestSHA256 + `"/><ds:DigestValue>AA==</ds:DigestValue></ds:Reference></ds:SignedInfo><ds:SignatureValue>AA==</ds:SignatureValue><ds:KeyInfo><ds:KeyName>fuzz</ds:KeyName><ds:X509Data><ds:X509SKI>AA==</ds:X509SKI></ds:X509Data></ds:KeyInfo></ds:Signature></root>`,
		`<root><ds:Signature xmlns:ds="` + NamespaceDSig + `"><ds:SignedInfo><ds:CanonicalizationMethod Algorithm="` + C14N11URI + `"/><ds:SignatureMethod Algorithm="` + AlgRSASHA256 + `"/><ds:Reference URI="external.xml"><ds:DigestMethod Algorithm="` + DigestSHA256 + `"/><ds:DigestValue>AA==</ds:DigestValue></ds:Reference></ds:SignedInfo><ds:SignatureValue>AA==</ds:SignatureValue><ds:KeyInfo><ds:RetrievalMethod URI="key.xml"/></ds:KeyInfo></ds:Signature></root>`,
		`<root><ds:Signature xmlns:ds="` + NamespaceDSig + `"><ds:SignedInfo/><ds:SignatureValue>not-base64</ds:SignatureValue><ds:KeyInfo><ds:X509Data><ds:X509Certificate>not-a-certificate</ds:X509Certificate></ds:X509Data></ds:KeyInfo></ds:Signature></root>`,
	}
	for _, seed := range seeds {
		f.Add([]byte(seed))
	}

	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > 1<<20 {
			return
		}
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()

		doc, err := helium.NewParser().Parse(ctx, input)
		if err != nil {
			return
		}

		verifier := NewVerifier(StaticKey([]byte("xmldsig1-fuzz-key"))).
			MaxReferences(32).
			MaxKeyInfoEntries(32).
			MaxDecodedBytes(1 << 20)
		_, _ = verifier.Verify(ctx, doc)
	})
}
