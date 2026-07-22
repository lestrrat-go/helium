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
each with an optional `#WithComments` variant), the XPath filter transform
(`http://www.w3.org/TR/1999/REC-xpath-19991116`), and the base64 decode transform
(`http://www.w3.org/2000/09/xmldsig#base64`). The XPath filter evaluates its
`ds:Transform/XPath` expression once per input node — with that node as the
context node, under the `XPath` element's in-scope namespace bindings — and keeps
each node whose result converts to boolean true (XPath 1.0 semantics: no default
element namespace). The XMLDSig `here()` function (core §6.6.3.1) is available
inside an XPath filter expression: it returns the `ds:XPath` element that bears
the expression, which is what the standard "enveloped signature via `here()`"
filter uses to omit the enclosing `ds:Signature`. Evaluation runs on a bounded
XPath 1.0 evaluator (an operation-count cap on top of the recursion and node-set
caps), so an attacker-supplied expression cannot stall verification. The base64 transform (**verify only**) takes the resolved
node-set's XPath 1.0 string-value — the referenced element's concatenated
descendant text, tags and comments stripped — base64-decodes it (whitespace in
the base64 text is ignored), and digests the decoded octets directly with no
canonicalization afterward; combining it with a preceding node-set transform is
not supported.

A node-set transform (enveloped-signature or XPath) may precede an octet-producing
transform (canonicalization or base64) or the octet-in→octet-out XSLT transform
(`http://www.w3.org/TR/1999/REC-xslt-19991116`), each of which ends the pipeline;
any transform ordered **after** an octet-ending transform — a second
canonicalization, a base64 after a canonicalization, an XSLT after a
canonicalization/base64, a second XSLT, or a canonicalization/base64 after an
XSLT — is rejected fail-closed with `ErrUnsupportedTransform`, as is any transform
URI the package does not implement. Signing does not support the base64 or XSLT
transforms: neither has a typed `Transform` constructor and the sign preflight
rejects both fail-closed.

### XSLT transform (opt-in, verify-only)

The XSLT transform is **off by default and verify-only**. XSLT is a powerful
language (`document()`, unbounded recursion and compute), and both the stylesheet
and its input are attacker-controlled on verification, so helium never runs XSLT
on its own: an XSLT transform fails closed with `ErrUnsupportedTransform` unless a
transformer is injected, mirroring the "no HTTP resolver shipped" stance for
external references.

To verify a signature whose Reference carries an XSLT transform, supply an
`XSLTTransformer`:

```go
type XSLTTransformer interface {
    TransformXSLT(ctx context.Context, stylesheet []byte, input []byte) ([]byte, error)
}
```

`Verifier.XSLTTransformer(t)` opts in. The single `ds:Transform/xsl:stylesheet`
(or `xsl:transform`) child is captured and serialized, and passed to `t` together
with the reference's pre-XSLT canonical octets; `t`'s output becomes the digest
input. The implementer **owns all resource and XXE policy** — compute/time/memory
limits and disabling `document()`/external access — because both inputs are
attacker-controlled. helium ships no transformer.

### General XPointer references (opt-in)

By default a `Reference` URI is resolved fail-closed to the four same-document
forms above. `Verifier.AllowXPointer(true)` additionally resolves a general
XPointer framework URI — zero or more `xmlns(prefix=uri)` scheme parts followed
by one `xpointer(<expr>)` part, for example
`#xmlns(a=urn:x)xpointer(//a:Target)`. It stays fail-closed by default: with
`AllowXPointer` off, a general XPointer URI is treated as an external reference
and, without a `ReferenceResolver`, rejected with `ErrReferenceNotFound`, so
default verification is unchanged.

When enabled, the `xpointer()` expression is evaluated on the same bounded XPath
1.0 evaluator (the document element's in-scope namespaces overlaid with the
`xmlns()` bindings), and its result **must identify a single element** — the XML
Signature Wrapping defense. An empty node-set is `ErrReferenceNotFound`; a
node-set selecting more than one element, or a non-element node, is
`ErrAmbiguousReference`. A literal `xpointer(id('X'))` keeps the same
duplicate-detecting id resolution the `#id` form uses (never a last-one-wins id
table). The `here()` function is not available inside a URI-borne XPointer.

### External references (opt-in)

By default a `Reference` URI that is not one of the four same-document forms — an
absolute URL, or a relative path pointing outside the document — is rejected with
`ErrReferenceNotFound`. This fail-closed default is unchanged: helium never
dereferences external content on its own.

To verify a detached signature whose References point outside the document,
supply a `ReferenceResolver`:

```go
type ReferenceResolver interface {
    ResolveReference(ctx context.Context, uri string) ([]byte, error)
}
```

`Verifier.ReferenceResolver(r)` opts in. An external Reference URI is joined
against the document's base URI (the `BaseURI` the document was parsed with, via
the same libxml2 URI-resolution helium uses elsewhere) and passed to the resolver;
the resolved octets are then run through the Reference's transform pipeline before
digesting:

- an empty transform chain, or a base64 decode transform, digests the resolved
  octets directly (after base64 decoding when present);
- a chain with a canonicalization or XPath filter transform needs a node-set, so
  the octets are first parsed into XML by `Verifier.ReferenceParser` — a
  **locked-down parser by default** (`helium.NewParser()`: XXE blocked, no
  filesystem, no network) — then filtered and canonicalized;
- an **enveloped-signature transform on an external reference is rejected**
  fail-closed (`ErrUnsupportedTransform`): removing the Signature's own subtree is
  meaningless on a resource that does not contain the Signature;
- an **XSLT transform on an external reference** applies the same off-by-default,
  verify-only rule as a same-document reference: its pre-XSLT octets are handed to
  the injected `XSLTTransformer` and its output digested, and with no (or a
  typed-nil) transformer configured it fails closed with `ErrUnsupportedTransform`.

A Reference satisfied through the resolver is marked `External` in the result. An
external reference covers bytes outside the document, not an element, so
`VerifyResult.Covers` and `VerifyResult.SignedElement` never attribute
in-document coverage to it — confirming a specific `*Element` was signed still
requires a same-document reference.

helium ships one resolver, `FSReferenceResolver(fsys fs.FS)`, which serves the
(base-joined) URI as a slash path inside `fsys` with **no network access**. It is
fail-closed on anything that is not a plain in-tree path: a URI carrying a scheme
(`http:`, `https:`, `file:`, `urn:`, any `scheme:` per RFC 3986, or a Windows
drive letter) is refused; a path escaping the root (absolute, or `..` past the
root) is refused; a leftover fragment is refused. Reads are bounded — a resource
larger than 64 MiB fails with `ErrReferenceTooLarge` rather than being buffered in
full.

**No HTTP resolver is provided.** The interface is public so callers can
dereference over any transport, but anyone implementing network dereferencing
owns the resulting SSRF and availability risk (an attacker who controls a
Reference URI could otherwise steer requests at internal hosts or stall
verification), so that decision is left explicitly to the caller.

`Signer.ReferenceResolver` / `Signer.ReferenceParser` are the symmetric signing
side, letting a detached signature cover external content. The sign and verify
paths funnel through the same octet-to-digest logic, so the signed digest is
byte-identical to what verification recomputes for the same input.

## Manifest inner-reference validation (opt-in)

A `ds:Reference` whose `Type` is
`http://www.w3.org/2000/09/xmldsig#Manifest` points at a `ds:Manifest`, which
holds its own list of `ds:Reference` elements (XMLDSig core §5.1). The signature
commits to the Manifest's own bytes — the top-level Manifest reference's digest
over the `ds:Manifest` subtree is checked exactly like any other reference — but
by design it says nothing about whether the Manifest's **inner** references still
match their targets. Per §5.1 that is left to the application.

`Verifier.ValidateManifests(true)` opts in to walking those inner references.
When enabled, after a top-level Manifest-typed reference has itself verified,
each inner `ds:Reference` is resolved, run through its transform pipeline, and
digested through the **same fail-closed path** as a top-level reference, and the
per-reference outcome is reported in `VerifyResult.Manifests`:

```go
type ManifestResult struct {
    Reference  *VerifiedReference // the top-level Manifest reference
    Element    *helium.Element    // the ds:Manifest element
    References []ManifestReference
}
type ManifestReference struct {
    URI, DigestAlgorithm string
    Element              *helium.Element
    Valid                bool
    Err                  error
}
```

Inner-reference results are **advisory**. A failed inner digest, an unsupported
inner transform, or an unresolved external inner reference is recorded as that
`ManifestReference`'s `Valid:false` / `Err` — it does **not** fail `Verify`, and
it never contributes to `VerifyResult.Covers` or `SignedElement`. Coverage is
never attributed through a Manifest, preserving the XML Signature Wrapping
guarantee: confirming a specific `*Element` was signed still requires a
top-level same-document reference. Only one level is walked — a Manifest nested
inside a Manifest is digested but not recursively expanded, which bounds the
work.

The toggle defaults to **false**: `VerifyResult.Manifests` is nil and no inner
references are walked, byte-identical to a Verifier without it. It is opt-in
because inner references may pull in transforms or external URIs the top-level
policy did not intend. The top-level `VerifiedReference.Type` is reported in the
result regardless of the toggle.

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

## Verification resource limits

An attacker-controlled, unsigned document can force verification to do
substantial decode/parse work *before* the `SignatureValue` is ever checked:
many or large `DigestValue`/`SignatureValue`/`X509Certificate` values to
base64-decode, and one `x509.ParseCertificate` per embedded certificate. To
bound that work the `Verifier` enforces three parse-time caps, each with a
conservative default that sits well above any legitimate signature so existing
documents verify unchanged:

| Builder | Bounds | Default |
|---------|--------|---------|
| `Verifier.MaxReferences(n)` | number of `ds:Reference` elements | 1024 |
| `Verifier.MaxKeyInfoEntries(n)` | `KeyInfo` children + `X509Data` children | 256 |
| `Verifier.MaxDecodedBytes(n)` | running total of base64-decoded bytes | 10 MiB |

Exceeding a cap fails with `ErrResourceLimitExceeded` before any Reference is
digested or the signature is checked. For each builder, `n == 0` selects the
default and a negative `n` disables that cap. Verification also polls the
context inside the KeyInfo and Reference parse loops, so a cancelled context or
passed deadline stops the work promptly rather than only at loop boundaries —
pass a `ctx` with a deadline to bound the per-Reference canonicalization of a
SignedInfo that declares many References.

## Detached signature placement (inclusive C14N)

`SignDetached` and `SignEnveloping` return a detached `ds:Signature` for the
caller to place. `SignedInfo` (and any in-Object Reference for `SignEnveloping`)
is canonicalized under a proxy carrying the **signing document element's**
inherited canonicalization context. If `SignedInfo`'s `CanonicalizationMethod`
— or an in-Object Reference — uses **inclusive** Canonical XML (`C14N10` /
`C14N11`), the caller MUST place the returned Signature directly under the
document element, or under an element with the same in-scope namespaces and
inherited `xml:*` attributes. Placing it under an element that contributes extra
in-scope namespace declarations or `xml:*` attributes changes the bytes
inclusive C14N canonicalizes, so verification recomputes a different canonical
form and fails. **Exclusive** Canonical XML (the `NewSigner` default,
`ExcC14NTransform`) inherits no namespaces or `xml:*` and is unaffected by
placement.

## Legacy and interop KeyInfo (verification)

For interoperating with older producers, verification-side `KeyInfo` parsing
recognizes several legacy constructs and surfaces them through `KeyInfoData` so
a `KeySource` can build the verification key. Parsing is namespace-strict and
fails closed (`ErrInvalidKeyInfo`) on unknown or partial key material.

**Security: `KeyInfoData` is untrusted.** A `KeySource` receives the parsed
`KeyInfoData` *before* the signature is verified, so every value in it —
embedded `X509Certificate`s, `RSAKeyValue`/`ECKeyValue`/`DSAKeyValue`,
issuer/serial and subject-name selectors — is attacker-controlled and NOT
authenticated by the signature. A `KeySource.ResolveKey` implementation MUST
decide trust itself: match the `KeyInfoData` against a trust store, a pinned
key, or a validated certificate chain, and return a key the caller already
trusts. It MUST NOT blindly return an embedded certificate's public key or a
`KeyValue` as the verification key — that lets an attacker sign with their own
key and have it verify. `KeyInfoData` is a *selector* into trusted key material,
never the key material itself. `StaticKey` and `X509CertKeySource` ignore
`KeyInfoData` entirely and return a pre-trusted key, which is the safe default; a
custom `KeySource` that consults `KeyInfoData` owns the trust decision.

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
- [merlin-xmldsig-twenty-three baseline](summary-merlinxmldsig.md) (2002
  IETF/W3C interop collection) — DSA/RSA/HMAC-SHA1 signatures, the base64
  transform, the truncated-HMAC must-reject case, and the X.509 KeyInfo
  variants.

The remaining expected failures are deliberate fail-closed design choices
(no external-reference dereferencing without a resolver, no XSLT transform
without an injected transformer), each documented with its reason in the harness
expectations.
