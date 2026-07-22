# xmldsig1 test vectors

Small, self-contained interop vectors copied verbatim so the `xmldsig1` tests do
not depend on an external checkout. State only; regenerate from the upstream
sources below if they need refreshing.

## rfc4050_ecdsa_p256_sha256.xml

RFC 4050 `ECDSAKeyValue` (namespace `http://www.w3.org/2001/04/xmldsig-more#`)
enveloping signature, ECDSA-SHA256 over an inclusive Canonical XML 1.0 reference.
The public-key point is carried as decimal `X`/`Y` `Value` attributes.

- Source: Apache Santuario (santuario) XML Security interop oracle vectors,
  `signature-enveloping-p256_sha256_4050.xml`.
- License: Apache License 2.0.

## dname_dsa_sha1_subjectname.xml

W3C XML Signature 2nd Edition interop `dname` vector: a DSA-SHA1 enveloped-object
signature whose `KeyInfo` carries only an `X509SubjectName` (`CN=John,C=US`). It
exercises `X509SubjectName` extraction and the DSA-SHA1 verify-path structure.

The vector does NOT carry the DSA public key (no certificate, no `DSAKeyValue`),
and the upstream suite ships no matching certificate, so its `SignatureValue`
cannot be verified end-to-end from these fixtures alone; it is used for KeyInfo
parsing only.

- Source: W3C `xmldsig2ed-tests`, `xmldsig/dname/diffRFCs-1-SUN.xml`.
- License: W3C Document License.
