# xmlenc1

> **EXPERIMENTAL** — This package is under active development. Its API may change without notice, and it may be moved to a separate repository in the future.

The `xmlenc1` package implements W3C XML Encryption 1.1 for helium documents.

Import path: `github.com/lestrrat-go/helium/xmlenc1`

## Security

- Secure by default. `Encryptor` defaults to authenticated AES-256-GCM
  (`DefaultBlockAlgorithm`) when no `BlockAlgorithm` is set. The package
  binds the `EncryptionMethod/@Algorithm` URI into the AEAD
  additional-authenticated-data for the legacy XML Encryption GCM
  identifiers. XML Encryption 1.1 GCM uses its standard IV, ciphertext, and
  authentication-tag encoding without additional authenticated data.
- AES-CBC is unauthenticated and vulnerable to padding-oracle attacks
  (Jager/Somorovsky 2011).
  - **Encryption:** selecting a CBC `BlockAlgorithm` requires
    `Encryptor.AllowLegacyCBC(true)`; otherwise encryption returns
    `ErrCBCEncryptionRequiresOptIn`. Opt in only to produce ciphertext
    for a legacy recipient that cannot accept AES-GCM.
  - **Decryption:** `Decryptor` refuses CBC by default and returns
    `ErrCBCRequiresOptIn`. Pass `AllowUnauthenticatedCBC(true)` only if
    you must accept legacy CBC and you have verified that decryption
    errors are not exposed to remote attackers. Decrypting existing CBC
    ciphertext is not the vulnerability; emitting new CBC is.
- The inner parser used on the decrypted plaintext has DTD loading,
  external entity resolution, and network access all disabled. Decrypted
  bytes are attacker-controlled, so a relaxed parser would constitute an
  XXE oracle.
- XML Encryption 1.1 ECDH-ES works in both directions with P-256, P-384, or
  P-521 and ConcatKDF. `Decryptor.ECPrivateKey` decrypts;
  `Encryptor.RecipientECPublicKey` with a `KeyWrapAlgorithm` encrypts. The
  key-encryption key is derived, not supplied, so `KeyEncryptionKey` plays no
  part; a fresh ephemeral key pair is generated per encryption and only its
  public half travels, in the `xenc:AgreementMethod`. `KeyDerivationParams`
  sets the ConcatKDF parameters, which are written to the wire because both
  sides must derive with identical values.
- `Encryptor.EncryptBytes` and `Decryptor.DecryptBytes` handle payloads that
  are neither an element nor element content. `EncryptBytes` returns a
  detached `EncryptedData` with no `Type` attribute (W3C xmlenc-core1 §3.1)
  and does not modify the tree; `DecryptBytes` returns the plaintext octets
  without parsing them as XML.
- `Decryptor.MaxEncryptedKeys` caps how many `<EncryptedKey>` candidates are
  trial-decrypted (default 100, negative for unlimited), because an unbounded
  count is a CPU amplification vector; over the cap fails with
  `ErrTooManyEncryptedKeys` before any candidate crypto runs. Its godoc owns
  the per-candidate branch dispatch: which key a candidate uses and what it
  costs. The cap belongs to the `<EncryptedKey>` stage, which a non-empty
  `Decryptor.SessionKey` returns before — see
  [Decrypting with a pre-shared session key](#decrypting-with-a-pre-shared-session-key).

## Choosing how the session key is protected

The content is always encrypted under a symmetric session key. What differs
is how the recipient obtains that key:

| Configuration | Wire result |
|---|---|
| `KeyTransportAlgorithm` + `RecipientPublicKey` | `<EncryptedKey>` holding the session key under RSA-OAEP |
| `KeyWrapAlgorithm` + `RecipientECPublicKey` | `<EncryptedKey>` holding the session key under AES Key Wrap, with the wrapping key derived by ECDH-ES; a `KeyEncryptionKey` set alongside is unused |
| `KeyWrapAlgorithm` + `KeyEncryptionKey` | `<EncryptedKey>` holding the session key under AES Key Wrap (RFC 3394) |
| non-empty `SessionKey` alone | no `<EncryptedKey>`; the recipient must already hold the key |
| none of the above | `ErrMissingConfig` — nothing can protect the session key |

A non-empty `SessionKey` must match the block algorithm's key length exactly,
else `KeySizeError`. An empty or nil `SessionKey` counts as not set:
encryption generates a random key of the right length instead, so it never
hits the length check.

## Decrypting with a pre-shared session key

A non-empty `Decryptor.SessionKey` is not a preference among keys; it is an
early return. `Decrypt` and `DecryptBytes` take it as the session key and
return before the whole `<EncryptedKey>` stage. Its godoc owns the account of
what that stage skips.

## Decryption does not modify the tree

`EncryptElement` and `EncryptContent` splice `<EncryptedData>` into the
document. `Decrypt` is deliberately not their mirror image: it leaves
`<EncryptedData>` where it is and returns the decrypted nodes detached, so
the caller decides whether to restore them, inspect them, or discard the
document. Reinsert with `elem.Replace(nodes[0])` for a `Type="...#Element"`
payload.

<!-- INCLUDE(examples/xmlenc1_encrypt_decrypt_example_test.go) -->
```go
package examples_test

import (
  "context"
  "crypto/rand"
  "crypto/rsa"
  "fmt"
  "strings"

  "github.com/lestrrat-go/helium"
  "github.com/lestrrat-go/helium/xmlenc1"
)

func Example_xmlenc1_encrypt_decrypt() {
  // Parse a document containing sensitive data. In SAML, this would
  // be an Assertion element inside a Response.
  const src = `<Response><Assertion>sensitive user data</Assertion></Response>`

  doc, err := helium.NewParser().Parse(context.Background(), []byte(src))
  if err != nil {
    fmt.Printf("parse error: %s\n", err)
    return
  }

  // Generate an RSA key pair. In production, use the recipient's
  // public key (e.g., the SP's certificate in SAML).
  key, err := rsa.GenerateKey(rand.Reader, 2048)
  if err != nil {
    fmt.Printf("keygen error: %s\n", err)
    return
  }

  // Encrypt the Assertion element. The Encryptor:
  // 1. Generates a random AES session key
  // 2. Encrypts the serialized element with AES-128-GCM
  // 3. Wraps the session key with RSA-OAEP
  // 4. Replaces the element in the tree with <EncryptedData>
  assertion, ok := helium.AsNode[*helium.Element](doc.DocumentElement().FirstChild())
  if !ok {
    fmt.Println("assertion not found")
    return
  }

  edElem, err := xmlenc1.NewEncryptor().
    BlockAlgorithm(xmlenc1.AES128GCM).
    KeyTransportAlgorithm(xmlenc1.RSAOAEP).
    RecipientPublicKey(&key.PublicKey).
    EncryptElement(context.Background(), assertion)
  if err != nil {
    fmt.Printf("encrypt error: %s\n", err)
    return
  }

  encrypted, _ := helium.WriteString(doc)
  fmt.Println(strings.Contains(encrypted, "sensitive user data"))
  fmt.Println(strings.Contains(encrypted, "EncryptedData"))

  // Decrypt returns the original node(s). The caller decides whether
  // to re-insert them into the tree or process them standalone.
  nodes, err := xmlenc1.NewDecryptor().PrivateKey(key).
    Decrypt(context.Background(), edElem)
  if err != nil {
    fmt.Printf("decrypt error: %s\n", err)
    return
  }

  decrypted, _ := helium.WriteString(nodes[0])
  fmt.Println(strings.Contains(decrypted, "sensitive user data"))
  // Output:
  // false
  // true
  // true
}
```
source: [examples/xmlenc1_encrypt_decrypt_example_test.go](https://github.com/lestrrat-go/helium/blob/main/examples/xmlenc1_encrypt_decrypt_example_test.go)
<!-- END INCLUDE -->

## W3C interop conformance

The sibling [`helium-w3c-tests`](https://github.com/lestrrat-go/helium-w3c-tests)
module runs the XML Encryption 1.1 core vectors with the `xmlenc11` suite. The
current [conformance summary](summary-xmlenc11.md) records all ten vectors as
passing, including the six ECDH-ES cases and four RSA cases. The suite is
available through the manual Conformance workflow and is not part of the
release gate.

Run it from `../helium-w3c-tests`:

```sh
go run ./cmd/w3cgen fetch xmlenc11
go run ./cmd/w3ctest xmlenc11
```
