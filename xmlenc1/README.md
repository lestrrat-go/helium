# xmlenc1

> **EXPERIMENTAL** — This package is under active development. Its API may change without notice, and it may be moved to a separate repository in the future.

The `xmlenc1` package implements W3C XML Encryption 1.1 for helium documents.

Import path: `github.com/lestrrat-go/helium/xmlenc1`

## Security

- Secure by default. `Encryptor` defaults to authenticated AES-256-GCM under
  the XML Encryption 1.1 identifier `AES256GCM11` (`DefaultBlockAlgorithm`)
  when no `BlockAlgorithm` is set. W3C xmlenc-core1 §5.2.4 defines AES-GCM
  only in the XML Encryption 1.1 namespace
  `http://www.w3.org/2009/xmlenc11#`, so that is the namespace a peer can
  recognize it in; its §5.1.1 table then marks `aes128-gcm` REQUIRED and
  `aes256-gcm` OPTIONAL, so the default trades a guarantee of support for
  the longer key and `BlockAlgorithm(AES128GCM11)` is the identifier every
  conforming peer must accept. 1.1 GCM uses the specified IV, ciphertext,
  and authentication-tag encoding without additional authenticated data.
- The `AES128GCM` and `AES256GCM` identifiers, which put GCM in the 2001 XML
  Encryption namespace, are defined by no XML Security specification.
  `Encryptor.BlockAlgorithm` accepts them and `Decryptor` decrypts them, so
  every document this package has emitted keeps decrypting, but a conforming
  peer will not accept one. For those two identifiers the package binds the
  `EncryptionMethod/@Algorithm` URI into the AEAD additional authenticated
  data — this package's own measure against an on-the-wire algorithm
  substitution, not conformance with any specification, and a second reason a
  document carrying them does not interoperate.
- An `EncryptedData` that carries no `EncryptionMethod` is decryptable only as
  an opt-in. W3C xmlenc-core1 §3.1 and §3.2 leave the element optional and
  require the recipient to already know the algorithm, and §4.4 admits obtaining
  it out of band, so `Decryptor.BlockAlgorithm` supplies the block algorithm
  URI; without it such a document fails with `ErrMalformedEncrypted`. The match
  against the document is strict, so setting it can only narrow what a decrypt
  accepts: the URI set there is used when the document declares none, the
  document's is used when none is set, and a pair that disagrees fails with
  `ErrConflictingBlockAlgorithm`. A document can therefore never override a
  caller who stated the algorithm out of band. Whichever URI the resolution
  returns is what the CBC opt-in, the legacy GCM additional authenticated data,
  and every key-length binding act on.
- AES-CBC is unauthenticated and vulnerable to padding-oracle attacks
  (Jager/Somorovsky 2011).
  - **Encryption:** selecting a CBC `BlockAlgorithm` requires
    `Encryptor.AllowLegacyCBC(true)`; otherwise encryption returns
    `ErrCBCEncryptionRequiresOptIn`. Opt in only to produce ciphertext
    for a legacy recipient that cannot accept AES-GCM.
  - **Decryption:** `Decryptor` refuses CBC by default and returns
    `ErrCBCRequiresOptIn`. Pass `AllowUnauthenticatedCBC(true)` only if
    you must accept legacy CBC and you have verified that decryption
    errors are not exposed to remote attackers. Decryption is the
    exposed operation: the attack feeds modified ciphertext to something
    that decrypts it and reads the answer, so a decryptor that accepts
    CBC is the oracle, while emitting CBC only leaves ciphertext for
    some other recipient to accept. Hiding the errors narrows that
    oracle rather than closing it — xmlenc-core1 §6.1.1 notes the
    surrounding protocol can signal well-formedness by itself — so this
    package collapses nearly every CBC failure to one
    `ErrDecryptionFailed` value and message, and AES-GCM remains the
    only full answer. The one exception is a wrong-length pre-shared
    `SessionKey`: its length is caller-configured rather than
    attacker-influenced, so a mismatch is reported as a bare
    `KeySizeError` instead. The oracle is the outcome, not the error
    text: under CBC, `Decrypt`
    succeeds only when the recovered plaintext parses, so its success
    or failure is the well-formedness oracle itself. The exact
    predicate depends on `@Type`: a `Content` payload need only parse
    as a well-formed fragment, while an `Element` payload — the
    default when `@Type` is empty or absent — must in addition parse
    to exactly one node, and that node must be an element.
    `DecryptBytes` is stronger still — it returns the plaintext octets
    on valid padding alone, with no XML constraint at all.
- The inner parser used on the decrypted plaintext has DTD loading,
  external entity resolution, and network access all disabled. Decrypted
  bytes are attacker-controlled, so a relaxed parser would constitute an
  XXE oracle.
- XML Encryption 1.1 ECDH-ES works in both directions with P-256, P-384, or
  P-521 and ConcatKDF. `Decryptor.ECPrivateKey` decrypts;
  `Encryptor.RecipientECPublicKey` with a `KeyWrapAlgorithm` encrypts. The
  key-encryption key is derived, not supplied, so `KeyEncryptionKey` belongs
  to the separate AES key wrapping mechanism; a fresh ephemeral key pair is
  generated per encryption and only its public half travels, in the
  `xenc:AgreementMethod`. `KeyDerivationParams` sets the ConcatKDF parameters,
  which are written to the wire because both sides must derive with identical
  values.
- The five ConcatKDF OtherInfo fields are limited to 4096 bytes together,
  because they arrive in an attacker-supplied document and drive work
  proportional to their size. Real OtherInfo is identifiers and nonces, so
  the limit is far above any interoperable value; over it fails with
  `ErrMalformedEncrypted`, when parsing a document and when deriving from
  parameters a caller built directly. Parameters with an empty
  `DigestMethod` are the one set that is never measured: they fall back to
  SHA-256 with empty OtherInfo, which discards the caller's fields before
  any derivation. `ConcatKDFParams`' godoc owns this rule.
- An `xenc:OAEPparams` element is limited to 1 KiB decoded, on the
  `EncryptedData`'s own `EncryptionMethod` and on every `EncryptedKey`'s
  alike, because both are read before any key is resolved and before anything
  the document says has been authenticated. The element carries the RSA-OAEP
  label, which is hashed before use and is a handful of octets in practice;
  over the limit fails with `ErrMalformedEncrypted`. The same limit applies to
  `Encryptor.OAEPParams`, where a larger label fails the encryption with
  `ErrEncryptionFailed` before any payload work — and only when key transport is
  the mechanism in use, since that is the only one that writes a label — so a
  label this package writes is a label it reads back. The limit is a policy
  ceiling rather than a conformance boundary: the xenc schema puts no length
  facet on the element, so a larger label is valid, and this package
  intentionally refuses one in both directions, neither writing nor reading it.
  The value is weighed as it is read and never joined into one
  string, so what the parse keeps is sized by the limit no matter how much
  whitespace or how many CDATA sections a label is spread over. Only character
  data is read: a text or CDATA child, or an entity reference, which
  contributes its entity's declared replacement text and is not expanded any
  further. An element child is refused with the same error, because asking one
  for its content would pull in its whole subtree, and a comment or processing
  instruction is ignored rather than spliced into the base64. The one cost that
  still follows the document is the copy the DOM hands out per child, which the
  walk pays exactly once.
- A `dsig11:PublicKey` inside an ECDH-ES originator key is limited to 133 bytes
  decoded, because it is read while the document is parsed and the curve is
  otherwise the only thing that would refuse an oversized value — after the
  whole of it has been materialized. 133 is not a policy choice: it is the
  largest SEC1 uncompressed point the three supported curves encode (65 on
  P-256, 97 on P-384, 133 on P-521), and `crypto/ecdh` accepts nothing else, so
  a longer value is rejected either way. The limit is the maximum across all
  three rather than the selected curve's own size, because `dsig11:NamedCurve`
  may follow the `dsig11:PublicKey` or be absent altogether, so there may be no
  curve to size the value by when it is weighed. Over the limit fails with
  `ErrMalformedEncrypted`. The value is weighed as it is read and never joined
  into one string, so what the parse keeps is sized by the limit no matter how
  much whitespace or how many CDATA sections a point is spread over, and the
  same child rules as `xenc:OAEPparams` apply: only character data is read, an
  element child is refused, and a comment or processing instruction is ignored.
- `Encryptor.EncryptBytes` and `Decryptor.DecryptBytes` handle payloads that
  are neither an element nor element content. `EncryptBytes` returns a
  detached `EncryptedData` with no `Type` attribute and does not modify the
  tree; because `Decrypt` treats an empty or absent `Type` as `TypeElement`, recover
  this payload with `DecryptBytes`, which returns the plaintext octets without
  parsing them as XML. `Decrypt` accepts only an empty or absent `Type`, `TypeElement`,
  or `TypeContent`; use `DecryptBytes` for opaque or application-defined
  payloads and for any other non-empty `Type`.
- `Decryptor.MaxEncryptedKeys` caps how many `<EncryptedKey>` candidates are
  trial-decrypted (default 100, negative for unlimited), because an unbounded
  count is a CPU amplification vector; over the cap fails while parsing,
  before the excess candidate is parsed, retained, or reaches candidate
  crypto. A candidate a
  [`ds:RetrievalMethod`](#conformance-scope) supplies costs a slot exactly as
  an inline `<EncryptedKey>` does, and the slot is charged when the candidate
  is retained, so several references naming one `<EncryptedKey>` cost one slot
  between them and a reference that supplies no candidate costs none — only a
  probe of the id index the decrypt builds once. Its godoc owns the
  per-candidate branch dispatch:
  which key a candidate uses and what it costs. The cap is applied before the key
  configuration is consulted, so it also bounds a decrypt driven by a
  pre-shared
  [`Decryptor.SessionKey`](#decrypting-with-a-pre-shared-session-key).
- `Decryptor.MaxEncryptedKeyBytes` caps the total `<EncryptedKey>` ciphertext
  those candidates may carry together (default 64 KiB, negative for
  unlimited), because a count alone does not bound their size; over the budget
  fails with `ErrEncryptedKeyBytesExceeded`. Its godoc owns what the budget
  covers and when it is charged. The budget is charged while the document is
  read, so it too bounds a decrypt driven by a pre-shared `SessionKey`.
- `Decryptor.MaxCipherValueBytes` caps the decoded `<EncryptedData>`
  payload one decrypt will hold (default 10 MiB, negative for
  unlimited), because the payload is the one value a document may make
  arbitrarily large; over the budget fails with
  `ErrCipherValueBytesExceeded` before the value is assembled or
  decoded, and so before block decryption or any plaintext parse. Its
  godoc owns what the budget covers and when it is charged. The default
  matches helium's own per-node content limit, which a payload spread
  over several text or CDATA nodes would otherwise slip past; this
  budget measures decoded octets, so neither that splitting nor the
  whitespace `xs:base64Binary` permits changes what it charges. It is
  charged while the document is read, so it too bounds a decrypt driven
  by a pre-shared `SessionKey`, and it is separate from
  `MaxEncryptedKeyBytes`, which bounds only the wrapped session-key
  candidates.
- An [`xenc:CipherReference`](#conformance-scope) naming a resource OUTSIDE the
  document is denied by default. The four same-document forms need no I/O and
  always resolve; every other URI fails with `ErrReferenceNotFound` until the
  caller sets `Decryptor.CipherReferenceResolver`, and configuring one adds the
  external form without changing how any same-document form resolves. helium
  ships one implementation, `FSReferenceResolver(fsys)`, which performs no
  network access and is fail-closed on anything that is not a plain in-tree
  path: a URI carrying an RFC 3986 scheme (a Windows drive letter included), a
  leftover fragment, and a path escaping the root after cleaning are all
  refused. No HTTP resolver ships, because an attacker who controls a
  `CipherReference` URI would otherwise steer requests at internal hosts or
  stall a decrypt, so whoever wants network dereferencing owns that SSRF and
  availability risk. A resolved resource is charged against the same budget its
  `CipherData` would have been — `MaxCipherValueBytes` for a payload reference,
  `MaxEncryptedKeyBytes` for a key one — and the shipped resolver reads only one
  byte past what the budget still allows, so an oversized resource is refused
  rather than buffered. A same-document reference is bounded by the same
  budgets: its canonical form is written through a limit-aware writer and stops
  at the first byte past the allowance, instead of canonicalizing an
  attacker-chosen subtree in full and discarding the result afterwards. The
  node-set feeding that writer is bounded too — it is linear in the subtree
  rather than in the product of its elements and the namespace declarations in
  scope on them, and a selection whose element count alone cannot fit the
  allowance is refused before it is built — and the walk observes the caller's
  context once per node.
- A `CipherReference` may declare transforms, and only the XMLDSig
  `#base64` transform is accepted. Every declared algorithm is validated before
  any of them runs, so a supported one standing ahead of an unsupported one is
  not executed first; a list longer than four, or a second `xenc:Transforms`,
  is refused unread. Neither `xenc:CipherReferenceType` nor
  `xenc:TransformsType` carries a wildcard, so an element the schema does not
  declare — a `Transform` in a foreign namespace, a `Transforms` wrapper in the
  wrong one — is refused rather than stepped over, and counts against that cap
  as it is read: a namespace-shifted transform cannot hide from the whitelist,
  or from a policy layer reading the list. Refusing is conforming — xmlenc-core1 §3.3.1 marks both
  the `Transform` feature and the particular algorithms OPTIONAL — and XPath and
  XSLT are the reason the rule exists: either one evaluates an expression the
  document chose over a document nothing has authenticated yet, which is
  unbounded compute bought with a few bytes of markup. An unsupported algorithm
  fails with `ErrMalformedEncrypted` wrapping an `*UnsupportedAlgorithmError`
  that names the refused URI.
- An AES key-wrap `<EncryptedKey>` must carry exactly the session-key length
  the declared block algorithm requires, plus RFC 3394's 8-byte integrity
  block. Any other length fails with `ErrKeyUnwrapFailed` before the unwrap
  rounds run, so a document cannot spend AES work on a ciphertext that is
  provably not a wrap of the key it declares. The length check belongs to
  the unwrap, so it is one of the steps a pre-shared
  [`Decryptor.SessionKey`](#decrypting-with-a-pre-shared-session-key)
  returns ahead of: that candidate is never resolved, and a document
  carrying one still decrypts.

## Conformance scope

This package implements W3C xmlenc-core1, and where it deliberately departs
from the specification it names the departure and the reason for it. One
construct the specification marks REQUIRED is absent, and the omission is
deliberate rather than pending:

- **Triple DES** — `#tripledes-cbc` (§5.2.2, REQUIRED) and `#kw-tripledes`
  (§5.7.1, REQUIRED) — refused deliberately, and this package will not
  implement them. Triple DES is a 64-bit block cipher, so Sweet32
  (CVE-2016-2183) applies, and NIST SP 800-131A Rev. 2 disallows TDEA
  encryption after 2023. The specification marks both REQUIRED; this is the
  place where following it no longer makes sense. Block encryption and key
  wrapping are AES only.
  `#tripledes-cbc` names the block cipher and fails with an error matching
  the relevant operation sentinel while preserving
  `*UnsupportedAlgorithmError`, in every key configuration, a pre-shared
  `SessionKey` included; the CBC opt-in gate names only the two AES-CBC
  URIs, so `AllowUnauthenticatedCBC(true)` is neither required for that
  error nor changes it. The one case that reports something else is an
  `EncryptedData` carrying no `<EncryptedKey>` and no `SessionKey`: the
  missing key is checked first, so it fails with `ErrMissingKey`.
  `#kw-tripledes` names an `<EncryptedKey>`'s wrapping and fails the same
  way only when that key must be resolved, so a pre-shared `SessionKey`
  decrypts past it.

`xenc:CipherReference` (§3.3.1, REQUIRED) is implemented, on an `EncryptedData`
payload and on every `EncryptedKey` alike. `CipherData` carries either a
`CipherValue` holding the cipher text inline or a `CipherReference` naming it by
URI, and the two are read into the same octets. `Encryptor` writes no
`CipherReference`: §3.3.1 requires support for reading one, not for writing one.

§3.3.1 defines no dereferencing of its own — it requires "the same URI encoding,
dereferencing, scheme, and HTTP response codes as that of [XMLDSIG-CORE1]" — so
what is REQUIRED is what that model makes required. There, dereferencing URIs in
the HTTP scheme is RECOMMENDED (xmldsig-core1 §4.4.3.1) while the null URI, the
shortname XPointer, and same-document dereferencing are MUSTs (§4.4.3.2,
§4.4.3.3). So the same-document forms are implemented unconditionally, with no
setting to turn them off, and external URIs are
[default-denied behind an opt-in resolver](#security).

The same four forms `ds:RetrievalMethod` recognizes resolve here: `URI=""`,
`URI="#id"`, `URI="#xpointer(/)"`, and `URI="#xpointer(id('id'))"`. A **present
but empty** `@URI` is the null URI naming the whole document; an **absent**
`@URI` is a different thing entirely and fails with `ErrMalformedEncrypted`,
because the xenc schema marks the attribute required. A URI matching more than
one element fails with `ErrAmbiguousReference` and one matching none with
`ErrReferenceNotFound`, for the same reasons those two refusals exist for
`ds:RetrievalMethod`.

What a same-document reference names is a node-set, and how it becomes octets
depends on the transforms:

- with **no transform**, the node-set is converted by Canonical XML 1.0 without
  comments, which is what §4.4.3.3 requires of a node-set that has to become an
  octet stream. A whole-document form naming the document element canonicalizes
  the document, so the top-level processing instructions outside that element
  are included; every other form canonicalizes the named element's subtree.
- with a **`#base64` transform** first, that transform consumes the node-set
  directly and decodes its string-value: the text nodes of the selection in
  document order, concatenated. xmldsig-core1 §6.6.2 "strips away the start and
  end tags of the identified element and any of its descendant elements", so
  base64 written in a descendant, or split across an element boundary, decodes
  as the same characters written directly under the target would. The value is
  counted before it is built, through the same bounded walk an inline
  `CipherValue` goes through.

An external URI is joined against the document's base URI and handed to
`Decryptor.CipherReferenceResolver`. Its result is an octet stream, so no
canonicalization applies to it. With no resolver configured it fails with
`ErrReferenceNotFound`.

A transform list may declare up to four transforms and every one of them must be
`#base64`; each one past the first decodes the octets the one before it
produced. [Security](#security) owns why nothing else is accepted there.

The resolved octets are cipher text and go straight to the block decryption;
they are never re-parsed as a document. A `CipherReference` naming the
`EncryptedData` that carries it, or naming itself, is therefore inert rather
than recursive — it terminates with whatever those octets decrypt to — and there
is no recursion depth to configure. Every one of these outcomes is decided while
the document is read, so all of them precede the block-algorithm resolution, the
AES-CBC opt-in gate, and a pre-shared
[`Decryptor.SessionKey`](#decrypting-with-a-pre-shared-session-key)'s early
return: that caller does not decrypt past a reference this package refused, and
a document this package will later refuse for its block algorithm still has its
reference resolved first.

Same-document `ds:RetrievalMethod` (§3.5, REQUIRED) is implemented and always
on, with no setting to turn it off. Inside a `ds:KeyInfo`, a
`<ds:RetrievalMethod Type=".../EncryptedKey" URI="#id"/>` names the
`xenc:EncryptedKey` holding the session key, wherever in the same document that
key sits, and §3.5.3 permits several of them. The candidate it supplies is
tried at the position the reference occupies, so a `ds:KeyInfo` mixing inline
`xenc:EncryptedKey` children with references tries them in document order. Two
references naming one `EncryptedKey` yield one candidate, decrypted once and
charged once against both [`MaxEncryptedKeys`](#bounds-and-limits) and
[`MaxEncryptedKeyBytes`](#bounds-and-limits). Only the
same-document form is REQUIRED and no external form is mandated anywhere, so a
URI naming another resource is refused with `ErrReferenceNotFound` whatever it
names — an external key location decides which key material the recipient
trial-decrypts, which is not a decision a document gets to make for a caller.
The four recognized forms are the ones XMLDSig core defines: `URI=""`,
`URI="#id"`, `URI="#xpointer(/)"`, and `URI="#xpointer(id('id'))"`.

Two refusals are worth naming. A URI matching MORE than one element fails with
`ErrAmbiguousReference` rather than resolving to either: an attacker who can
inject an element carrying an `Id` already in use would otherwise choose which
key the recipient unwraps, which is XML Signature Wrapping applied to
encryption. A URI matching none fails with `ErrReferenceNotFound`. Both are
decided while the document is read, so they precede a pre-shared
[`Decryptor.SessionKey`](#decrypting-with-a-pre-shared-session-key)'s early
return: that caller does not decrypt past a reference this package refused.

A `ds:RetrievalMethod` whose `Type` this package does not implement — a
`#DerivedKey` (§3.5.2), or any type from another specification — is stepped
over before its URI is looked at, so it costs nothing, cannot fail a decrypt,
and a pre-shared `SessionKey` decrypts past it. A `Type` of
`#EncryptedKey` naming something that is not an `xenc:EncryptedKey` is a
contradiction inside the document and fails with `ErrMalformedEncrypted`; a
reference with no `Type` at all is resolved, and its target used only if it is
an `xenc:EncryptedKey`. `Encryptor` writes no `ds:RetrievalMethod`: §3.5
requires support for reading one, not for writing one.

The other `ds:KeyInfo` children this package does not read are `ds:KeyValue`
(§3.5, OPTIONAL), `ds:KeyName` (§3.5, RECOMMENDED), and `xenc11:DerivedKey`
(§3.5.2). A `ds:KeyValue` inside an `xenc:OriginatorKeyInfo` is a different
position and is read, since that is where ECDH-ES carries the sender's
ephemeral key.

An `xenc:KeySize` child of `EncryptionMethod` is read and checked, but it is
never used as a key length: every algorithm URI this package implements
already fixes its own. A `KeySize` under a URI that implies a length must
state exactly that length — in bits, per §5.6.2.2 — or the document is
refused with `ErrMalformedEncrypted` while it is read, ahead of a pre-shared
[`Decryptor.SessionKey`](#decrypting-with-a-pre-shared-session-key)'s early
return, so a contradicting value cannot be bypassed that way; an
`EncryptedData` carrying no `EncryptionMethod` at all has no `KeySize` to
check either way. A `KeySize` under a URI that implies no length — RSA key
transport above all — is accepted and ignored. What remains absent is
`KeySize` as the length SOURCE, used when the URI alone cannot supply one;
that case arises only for stream ciphers and key agreements naming such an
algorithm, none of which this package implements. `Encryptor` writes no
`KeySize`.

Section 3 carries a blanket "features described in this section MUST be
implemented", so four more constructs are unimplemented, and only one of them
can fail a decrypt: `xenc:ReferenceList` (§3.6), which points from a key to the
items it encrypted and so only matters for a detached key this package cannot
follow anyway; `xenc11:DerivedKey` (§3.5.2), which may appear in an
`EncryptedData`'s own `ds:KeyInfo` and tells the recipient to derive the
content key from master key material it already holds — the parse ignores it,
so such an `EncryptedData` fails with `ErrMissingKey` instead of deriving the
key. That `ErrMissingKey` is what the caller sees only once the block algorithm
has resolved: resolution and the AES-CBC opt-in both run ahead of the
missing-key check, so an `EncryptedData` with no `EncryptionMethod` and no
`Decryptor.BlockAlgorithm` fails with `ErrMalformedEncrypted`, and an AES-CBC
one without `AllowUnauthenticatedCBC(true)` with `ErrCBCRequiresOptIn`,
whatever its `ds:KeyInfo` holds. It does not arise at all when the caller
supplies that key as a pre-shared `SessionKey`, whose early return precedes key
resolution;
`xenc:CarriedKeyName` (§3.5.1), which the parse steps over rather than
reads; and `xenc:EncryptionProperties` (§3.7), which is advisory metadata.

## Choosing how the session key is protected

The content is always encrypted under a symmetric session key. What differs
is how the recipient obtains that key:

| Configuration | Wire result |
|---|---|
| `KeyTransportAlgorithm` + `RecipientPublicKey` | `<EncryptedKey>` holding the session key under RSA-OAEP |
| `KeyWrapAlgorithm` + `RecipientECPublicKey` | `<EncryptedKey>` holding the session key under AES Key Wrap, with the wrapping key derived by ECDH-ES |
| `KeyWrapAlgorithm` + `KeyEncryptionKey` | `<EncryptedKey>` holding the session key under AES Key Wrap (RFC 3394) |
| non-empty `SessionKey` alone | no `<EncryptedKey>`; the recipient must already hold the key |
| none of the above | `ErrMissingConfig` — nothing can protect the session key |

The first three rows are mechanisms, and configuring two of them fails with
`ErrConflictingKeyConfig`; its godoc owns that rule and why a single
`<EncryptedKey>` makes it necessary. A `SessionKey` alongside one mechanism is
allowed — it is the key that mechanism protects.

A non-empty `SessionKey` must match the block algorithm's key length exactly,
else `KeySizeError`. An empty or nil `SessionKey` counts as not set:
encryption generates a random key of the right length instead, so it never
hits the length check.

## Decrypting with a pre-shared session key

A non-empty `Decryptor.SessionKey` is not a preference among keys; it is an
early return. `Decrypt` and `DecryptBytes` take it as the session key and
return before candidate selection, per-candidate validation, and per-candidate
key resolution. Its godoc owns the account of what that skips. It returns
early from the key handling alone, not from the decrypt, so everything ahead
of that point still holds: the `MaxEncryptedKeys` count, the
`MaxEncryptedKeyBytes` budget, and the `MaxCipherValueBytes` payload budget,
all charged while the document is read; `Decrypt`'s `@Type` check
(`DecryptBytes` does not read `@Type`, so it has no such gate); the
block-algorithm resolution `ErrMalformedEncrypted` and
`ErrConflictingBlockAlgorithm` come out of; the AES-CBC opt-in gate; and the
check binding the supplied key's length to the resolved algorithm
(`KeySizeError`). A decrypt that fails any of those reports that failure,
whatever key the caller holds.

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
    BlockAlgorithm(xmlenc1.AES128GCM11).
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
passing, none skipped and none failing.

Those ten vectors exercise key protection: six ECDH-ES ConcatKDF cases on
EC-P256, EC-P384, and EC-P521, and four rsa-oaep cases on RSA-2048, RSA-3072,
and RSA-4096. Every one of them uses AES-GCM block encryption. So the snapshot
is evidence about how the session key is protected and about AES-GCM, and it is
evidence about nothing else: no CBC block algorithm appears in it, the Triple
DES algorithms the [conformance scope](#conformance-scope) above refuses are not
covered by it, and it is not the merlin interop corpus. The ten are the
suite in full, and the 1.1 interop corpus holds no Triple DES vector, so the
zero skips is not a vector being passed over.

No interop vector covers `xenc:CipherReference`, in this suite or in any other
corpus the harness fetches. The one `CipherReference` vector in the Apache
Santuario corpus needs BOTH an XPath transform and `aes192-cbc`, and this
package implements neither, so there is nothing runnable to add and no skip
entry to write — a skip would imply a vector is being passed over when none
exists. The feature's evidence is this package's own tests.

The suite is available through the manual Conformance workflow and is not part
of the release gate.

Run it from `../helium-w3c-tests`:

```sh
go run ./cmd/w3cgen fetch xmlenc11
go run ./cmd/w3ctest xmlenc11
```
