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
