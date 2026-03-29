package examples_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmldsig1"
)

func Example_xmldsig1_sign_enveloped() {
	// Parse an XML document that we want to sign. In SAML, this would
	// typically be an Assertion or Response element.
	const src = `<root Id="doc1"><data>Hello, World!</data></root>`

	doc, err := helium.NewParser().Parse(context.Background(), []byte(src))
	if err != nil {
		fmt.Printf("parse error: %s\n", err)
		return
	}

	// Generate an RSA key pair for signing. In production, load your
	// private key from a PEM file or key store.
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Printf("keygen error: %s\n", err)
		return
	}

	// Create a Signer configured for the most common SAML pattern:
	// RSA-SHA256 signature, enveloped signature transform + Exclusive
	// C14N, SHA-256 digest. NewEnvelopedReference() bundles these defaults.
	signer := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(xmldsig1.NewEnvelopedReference())

	// SignEnveloped inserts a <ds:Signature> element as a child of the
	// given parent element. The signature covers the entire document
	// (URI=""), excluding the Signature element itself.
	err = signer.SignEnveloped(context.Background(), doc, doc.DocumentElement(), key)
	if err != nil {
		fmt.Printf("sign error: %s\n", err)
		return
	}

	out, err := helium.WriteString(doc)
	if err != nil {
		fmt.Printf("write error: %s\n", err)
		return
	}

	// The output contains the original document with a ds:Signature child.
	fmt.Println(strings.Contains(out, "ds:Signature"))
	fmt.Println(strings.Contains(out, "ds:SignatureValue"))
	fmt.Println(strings.Contains(out, "Hello, World!"))
	// Output:
	// true
	// true
	// true
}
