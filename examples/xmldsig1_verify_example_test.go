package examples_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmldsig1"
)

func Example_xmldsig1_verify() {
	// First, create a signed document to demonstrate verification.
	const src = `<root><message>Verified content</message></root>`
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

	err = xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(xmldsig1.NewEnvelopedReference()).
		SignEnveloped(context.Background(), doc, doc.DocumentElement(), key)
	if err != nil {
		fmt.Printf("sign error: %s\n", err)
		return
	}

	// To verify, create a Verifier with a KeySource that provides the
	// public key. StaticKey always returns the same key; for SAML you
	// would typically use X509CertKeySource with the IdP's certificate.
	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))

	// Verify checks the first ds:Signature element in the document.
	// It validates both the SignatureValue (cryptographic signature over
	// the canonical SignedInfo) and each Reference digest.
	err = verifier.Verify(context.Background(), doc)
	if err != nil {
		fmt.Printf("verification failed: %s\n", err)
		return
	}

	fmt.Println("signature valid")
	// Output:
	// signature valid
}
