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

## Reference processing

### Same-document URI forms

A `Reference` URI is dereferenced to a node-set fail-closed: only same-document
forms are supported, and every other URI (an external reference, or an
unrecognized XPointer scheme) is rejected with `ErrReferenceNotFound`. The
supported forms and their comment-node semantics (XMLDSig core §4.3.3.2-3) are:

| URI | Node-set | Comment nodes |
|-----|----------|---------------|
| `""` | whole document | **excluded** |
| `"#id"` | element with that id | **excluded** |
| `"#xpointer(/)"` | whole document | **included** |
| `"#xpointer(id('id'))"` | element with that id | **included** |

Comment membership is a property of the reference form, not of the
canonicalization method. A C14N `#WithComments` method only emits comment nodes
that are part of the node-set, so a bare `"#id"` or `""` reference never emits
comments even under a `#WithComments` canonicalization — the two `#xpointer`
forms are the only ones that carry comments through. An `"#id"` that matches more
than one element (across the document and any enveloping `Object` content) is
rejected with `ErrAmbiguousReference`, defending against XML Signature Wrapping.

### Transforms

The supported transforms are the enveloped-signature transform, the
canonicalization transforms (Canonical XML 1.0 / 1.1 and Exclusive C14N 1.0,
each with an optional `#WithComments` variant), and the XPath filter transform
(`http://www.w3.org/TR/1999/REC-xpath-19991116`). The XPath filter evaluates its
`ds:Transform/XPath` expression once per input node — with that node as the
context node, under the `XPath` element's in-scope namespace bindings — and keeps
each node whose result converts to boolean true (XPath 1.0 semantics: no default
element namespace). A node-set transform (enveloped-signature or XPath) may
precede the canonicalization transform; any transform ordered **after** the
octet-producing canonicalization — including an octet-consuming transform such as
XSLT or base64 — is rejected fail-closed with `ErrUnsupportedTransform`, as is
any transform URI the package does not implement.

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

## Legacy and interop KeyInfo (verification)

For interoperating with older producers, verification-side `KeyInfo` parsing
recognizes several legacy constructs and surfaces them through `KeyInfoData` so
a `KeySource` can build the verification key. Parsing is namespace-strict and
fails closed (`ErrInvalidKeyInfo`) on unknown or partial key material.

- **RFC 4050 `ECDSAKeyValue`** (namespace `http://www.w3.org/2001/04/xmldsig-more#`):
  `DomainParameters/NamedCurve@URN` selects the curve (P-256/P-384/P-521) and
  `PublicKey/X`,`/Y` carry the point as decimal integer `Value` attributes. It is
  surfaced through the same `KeyInfoData.ECKeyValue` as a 1.1 `ECKeyValue`, so a
  `KeySource` builds an `*ecdsa.PublicKey` from `ECKeyValue.Curve`/`X`/`Y`.
  Emitting RFC 4050 on the signing side is not supported.
- **`X509IssuerSerial` and `X509SubjectName`** inside `X509Data`: the issuer DN +
  serial number (`KeyInfoData.X509IssuerSerials`) and subject DN
  (`KeyInfoData.X509SubjectNames`) are extracted verbatim — the library does no
  DName canonicalization or matching — so a `KeySource` can select the right
  certificate out of band.
- **`DSAKeyValue`**: `P`/`Q`/`G`/`Y` are parsed into `KeyInfoData.DSAKeyValue`;
  a `KeySource` builds a `*dsa.PublicKey` from them.

### DSA-SHA1 (verify-only)

DSA-SHA1 (`xmldsig#dsa-sha1`) is supported for **verification only**, as legacy
interop. It sits behind the same SHA-1 weak gate as `rsa-sha1`: verification
requires `Verifier.AllowSHA1(true)`, otherwise it fails with `ErrWeakAlgorithm`.
The `SignatureValue` is the XML-DSig fixed-width `r||s` concatenation. A DSA key
may come from a parsed `DSAKeyValue` or from an X.509 certificate (which
`crypto/x509` parses into a `*dsa.PublicKey`). **Signing with DSA is not
supported** — a signing attempt with the DSA URI fails with a clear
`ErrUnsupportedAlgorithm` ("DSA signing is not supported").

## W3C interop conformance

The package is measured against two W3C XML Signature interop suites through
the [helium-w3c-tests](https://github.com/lestrrat-go/helium-w3c-tests)
harness (`xmldsig2ed` and `xmldsig11` suites). Committed point-in-time
evidence:

- [Test Cases for C14N 1.1 and XMLDSig Interoperability](summary-xmldsig2ed.md)
  (W3C Note, 2008) — canonicalization node-set cases plus signature
  verification (C14N 1.1, XPointer references, X.509 Distinguished Name
  KeyInfo).
- [XML Signature 1.1 interop vectors](summary-xmldsig11.md) — enveloping
  signatures covering ECDSA P-256/P-384/P-521, RSA and HMAC with the SHA-2
  family, and RFC 4050 ECDSAKeyValue KeyInfo.

The remaining expected failures are deliberate fail-closed design choices
(no external-reference dereferencing, no XSLT transform), each documented
with its reason in the harness expectations.
