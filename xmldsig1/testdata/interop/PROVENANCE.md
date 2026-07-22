# Provenance

These files are verbatim vectors from the W3C "XML Signature 2nd Edition"
interoperability test suite (xmldsig2ed-tests, 2008), used here to lock
reference-processing behavior (same-document URI forms, comment-node node-set
semantics, and the XPath filter transform).

- `xpointer-{1..6}-SUN.xml` — W3C XML Signature interop, "xpointer" group.
  Exercise the four same-document Reference URI forms (`#xpointer(/)`,
  `#xpointer(id('X'))`, `""`, `#id`) crossed with C14N 1.1 WithComments.
- `defCan-{1,2,3}-signature.xml` — W3C XML Signature interop, "defCan" group.
  Exercise the `http://www.w3.org/TR/1999/REC-xpath-19991116` XPath filter
  transform over an external `xml-base-input.xml` reference.
- `xml-base-input.xml` — the external document referenced by the defCan vectors.
- `c14n11/xml-base-input.xml` — the same document at the RELATIVE path the
  defCan-1 Reference URI (`c14n11/xml-base-input.xml`) names, so an
  FSReferenceResolver rooted at this directory serves it. Used by the
  external-reference resolver tests to verify defCan-1 end-to-end (HMAC-SHA1 key
  `"secret"`, XPath filter + Canonical XML 1.1).
- `signature-external-dsa.xml` — from the 2002 Baltimore "merlin-xmldsig"
  collection (mirrored in Apache Santuario under Apache License 2.0). A detached
  DSA-SHA1 signature whose single Reference points at the ABSOLUTE URL
  `http://www.w3.org/TR/xml-stylesheet` with no transforms, digesting the raw
  resolved octets. The DSA key is inline (`KeyValue/DSAKeyValue`). Used to verify
  an external reference end-to-end via an in-test map resolver.
- `xml-stylesheet` — the document that `http://www.w3.org/TR/xml-stylesheet`
  resolves to (the W3C "Associating Style Sheets with XML documents" TR page),
  vendored from the Apache Santuario xmldsig11 test resources. Its raw-octet
  SHA-1 is the DigestValue in `signature-external-dsa.xml`.

Distributed by the W3C under the W3C Document License. Retained here only as
frozen conformance fixtures for helium's xmldsig1 tests.

## Baltimore merlin collection

- `signature-enveloping-b64-dsa.xml` — from the 2002 Baltimore "merlin-xmldsig"
  interoperability collection (merlin-xmldsig-twenty-three), mirrored in the
  Apache Santuario (xmlsec) test resources under Apache License 2.0. An
  enveloping DSA-SHA1 signature whose single Reference (`URI="#object"`) applies
  the `http://www.w3.org/2000/09/xmldsig#base64` decode transform to a
  `ds:Object` holding base64 text, digesting the decoded octets. Retained here
  to lock the base64 transform's verify path.
