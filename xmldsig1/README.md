# xmldsig1

> **EXPERIMENTAL** — This package is under active development. Its API may change without notice, and it may be moved to a separate repository in the future.

The `xmldsig1` package implements W3C XML Digital Signatures 1.1 for helium documents.

Import path: `github.com/lestrrat-go/helium/xmldsig1`

<!-- INCLUDE(examples/xmldsig1_sign_verify_example_test.go) -->
```go
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

func Example_xmldsig1_sign_verify() {
  // Parse an XML document to sign. In SAML, this is typically an
  // Assertion or Response element.
  const src = `<root Id="doc1"><data>Hello, World!</data></root>`

  doc, err := helium.NewParser().Parse(context.Background(), []byte(src))
  if err != nil {
    fmt.Printf("parse error: %s\n", err)
    return
  }

  // Generate an RSA key pair. In production, load your private key
  // from a PEM file or key store.
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

  out, _ := helium.WriteString(doc)
  fmt.Println(strings.Contains(out, "ds:Signature"))

  // To verify, create a Verifier with a KeySource that provides the
  // public key. StaticKey always returns the same key; for SAML you
  // would typically use X509CertKeySource with the IdP's certificate.
  //
  // Verify requires the document to contain exactly one ds:Signature
  // element; it returns ErrAmbiguousSignature when more than one is
  // present (use VerifyElement to disambiguate in that case). It
  // validates both the SignatureValue (cryptographic signature over the
  // canonical SignedInfo) and each Reference digest.
  _, err = xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey)).
    Verify(context.Background(), doc)
  if err != nil {
    fmt.Printf("verification failed: %s\n", err)
    return
  }

  fmt.Println("signature valid")
  // Output:
  // true
  // signature valid
}
```
source: [examples/xmldsig1_sign_verify_example_test.go](https://github.com/lestrrat-go/helium/blob/main/examples/xmldsig1_sign_verify_example_test.go)
<!-- END INCLUDE -->

## Security: SHA-1 rejected by default

SHA-1-based algorithms (`rsa-sha1`, `ecdsa-sha1`, `hmac-sha1`, and the `sha1`
digest) are **rejected by default** for both signing and verification. SHA-1 is
cryptographically weak; accepting it silently exposes callers to algorithm
downgrade and collision attacks. When a SHA-1 algorithm is encountered without
an explicit opt-in, the operation fails with `ErrWeakAlgorithm`.

If you must interoperate with a legacy system that cannot be upgraded, opt in
explicitly by calling `AllowSHA1(true)` on both the `Signer` and the
`Verifier`, as shown in the example below:

<!-- INCLUDE(examples/xmldsig1_sha1_optin_example_test.go) -->
```go
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
```
source: [examples/xmldsig1_sha1_optin_example_test.go](https://github.com/lestrrat-go/helium/blob/main/examples/xmldsig1_sha1_optin_example_test.go)
<!-- END INCLUDE -->

> **Note (breaking default change):** earlier versions accepted SHA-1
> signatures and digests without any opt-in. Code that relied on verifying
> SHA-1 signatures must now call `Verifier.AllowSHA1(true)`; code that produced
> SHA-1 signatures must call `Signer.AllowSHA1(true)`. SHA-256 and stronger
> algorithms are unaffected.
