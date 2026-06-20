package examples_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmldsig1"
)

// Example_xmldsig1_sha1_optin demonstrates the explicit opt-in required to
// produce and verify legacy SHA-1 signatures. SHA-1 (rsa-sha1, hmac-sha1, and
// the sha1 digest) is rejected by default with ErrWeakAlgorithm; call
// AllowSHA1(true) on both the Signer and the Verifier only when you must
// interoperate with a legacy system that cannot be upgraded.
func Example_xmldsig1_sha1_optin() {
	const src = `<root Id="doc1"><data>Hello, World!</data></root>`

	doc, err := helium.NewParser().Parse(context.Background(), []byte(src))
	if err != nil {
		fmt.Printf("parse error: %s\n", err)
		return
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Printf("keygen error: %s\n", err)
		return
	}

	// Produce a legacy SHA-1 signature (discouraged). AllowSHA1(true) is
	// required; without it SignEnveloped returns ErrWeakAlgorithm.
	signer := xmldsig1.NewSigner().
		AllowSHA1(true).
		SignatureAlgorithm(xmldsig1.AlgRSASHA1).
		Reference(xmldsig1.ReferenceConfig{
			URI:             "",
			DigestAlgorithm: xmldsig1.DigestSHA1,
			Transforms:      []xmldsig1.Transform{xmldsig1.Enveloped(), xmldsig1.ExcC14NTransform()},
		})

	if err := signer.SignEnveloped(context.Background(), doc, doc.DocumentElement(), key); err != nil {
		fmt.Printf("sign error: %s\n", err)
		return
	}

	// Verify the legacy SHA-1 signature. The default verifier rejects SHA-1,
	// so AllowSHA1(true) is required here as well.
	_, err = xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey)).
		AllowSHA1(true).
		Verify(context.Background(), doc)
	if err != nil {
		fmt.Printf("verification failed: %s\n", err)
		return
	}

	fmt.Println("legacy SHA-1 signature valid")
	// Output:
	// legacy SHA-1 signature valid
}
