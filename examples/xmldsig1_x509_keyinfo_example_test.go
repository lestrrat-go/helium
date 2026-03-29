package examples_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmldsig1"
)

func Example_xmldsig1_x509_keyinfo() {
	// When signing for SAML, include the X.509 certificate in KeyInfo
	// so the verifier can identify which key was used.
	const src = `<saml:Assertion xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" ID="_a1" Version="2.0" IssueInstant="2024-01-01T00:00:00Z"><saml:Issuer>https://idp.example.com</saml:Issuer></saml:Assertion>`

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

	// Create a self-signed certificate. In production, use a certificate
	// issued by your organization's CA.
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "idp.example.com"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		fmt.Printf("cert error: %s\n", err)
		return
	}
	cert, _ := x509.ParseCertificate(certDER)

	// X509DataKeyInfo embeds the certificate in the signature so the
	// verifier can extract it. The verifier still needs a trusted copy
	// of the certificate to prevent key substitution attacks.
	err = xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(xmldsig1.NewEnvelopedReference()).
		KeyInfo(xmldsig1.X509DataKeyInfo(cert)).
		SignEnveloped(context.Background(), doc, doc.DocumentElement(), key)
	if err != nil {
		fmt.Printf("sign error: %s\n", err)
		return
	}

	out, _ := helium.WriteString(doc)
	fmt.Println(strings.Contains(out, "ds:X509Certificate"))

	// Verify using the trusted certificate.
	err = xmldsig1.NewVerifier(xmldsig1.X509CertKeySource(cert)).
		Verify(context.Background(), doc)
	if err != nil {
		fmt.Printf("verify error: %s\n", err)
		return
	}
	fmt.Println("signature valid")
	// Output:
	// true
	// signature valid
}
