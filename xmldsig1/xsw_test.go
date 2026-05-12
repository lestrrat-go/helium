package xmldsig1_test

import (
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmldsig1"
	"github.com/stretchr/testify/require"
)

// findByLocalNameAndID walks the doc element subtree, finds an element
// whose name local part matches localName carrying an Id/ID attribute
// equal to id, and returns the *first* such element. Test helper only.
func findByLocalNameAndID(doc *helium.Document, localName, id string) *helium.Element {
	var found *helium.Element
	var walk func(n helium.Node)
	walk = func(n helium.Node) {
		if found != nil {
			return
		}
		elem, ok := helium.AsNode[*helium.Element](n)
		if !ok {
			return
		}
		name := elem.Name()
		l := name
		if _, after, ok := strings.Cut(name, ":"); ok {
			l = after
		}
		if l == localName {
			for _, a := range elem.Attributes() {
				if (a.Name() == "Id" || a.Name() == "ID") && a.Value() == id {
					found = elem
					return
				}
			}
		}
		for c := elem.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(doc.DocumentElement())
	return found
}

// TestXSW_DuplicateIDFailsVerify reproduces the classic XML Signature
// Wrapping shape: the document contains two elements with the same Id and
// the Reference URI matches both. Verify MUST refuse to resolve the
// reference rather than silently pick one.
func TestXSW_DuplicateIDFailsVerify(t *testing.T) {
	xml := `<root><payload Id="target"><val>good</val></payload></root>`
	key := generateRSAKey(t)
	doc := mustParseXML(t, xml)

	ref := xmldsig1.ReferenceConfig{
		URI:             "#target",
		DigestAlgorithm: xmldsig1.DigestSHA256,
		Transforms:      []xmldsig1.Transform{xmldsig1.ExcC14NTransform()},
	}

	sigElem, err := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(ref).
		SignDetached(t.Context(), doc, key)
	require.NoError(t, err)
	require.NoError(t, doc.DocumentElement().AddChild(sigElem))

	// Now mount the attack: serialize, inject a duplicate-ID payload, re-parse.
	signed, err := helium.WriteString(doc)
	require.NoError(t, err)

	// Inject an evil twin BEFORE the legitimate payload.
	evil := `<payload Id="target"><val>evil</val></payload>`
	tampered := strings.Replace(signed, `<payload Id="target">`, evil+`<payload Id="target">`, 1)
	require.NotEqual(t, signed, tampered)

	doc2 := mustParseXML(t, tampered)
	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
	_, err = verifier.Verify(t.Context(), doc2)
	require.ErrorIs(t, err, xmldsig1.ErrAmbiguousReference)
}

// TestXSW_DuplicateXMLIDFailsVerify is the xml:id variant of the
// duplicate-ID XSW shape. It guards against the failure mode where the
// document-level ID table (populated via xml:id during parse) silently
// overwrites duplicates and hands a single hit back to the reference
// resolver. The resolver MUST walk the tree and surface both matches.
func TestXSW_DuplicateXMLIDFailsVerify(t *testing.T) {
	xml := `<root xmlns:xml="http://www.w3.org/XML/1998/namespace"><payload xml:id="target"><val>good</val></payload></root>`
	key := generateRSAKey(t)
	doc := mustParseXML(t, xml)

	ref := xmldsig1.ReferenceConfig{
		URI:             "#target",
		DigestAlgorithm: xmldsig1.DigestSHA256,
		Transforms:      []xmldsig1.Transform{xmldsig1.ExcC14NTransform()},
	}

	sigElem, err := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(ref).
		SignDetached(t.Context(), doc, key)
	require.NoError(t, err)
	require.NoError(t, doc.DocumentElement().AddChild(sigElem))

	signed, err := helium.WriteString(doc)
	require.NoError(t, err)

	evil := `<payload xml:id="target"><val>evil</val></payload>`
	tampered := strings.Replace(signed, `<payload xml:id="target">`, evil+`<payload xml:id="target">`, 1)
	require.NotEqual(t, signed, tampered)

	doc2 := mustParseXML(t, tampered)
	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
	_, err = verifier.Verify(t.Context(), doc2)
	require.ErrorIs(t, err, xmldsig1.ErrAmbiguousReference)
}

// TestVerifyResult_ExposesSignedElement asserts that Verify returns the
// resolved element pointer for each Reference, so the caller can compare
// pointer equality against the element they are about to consume.
func TestVerifyResult_ExposesSignedElement(t *testing.T) {
	xml := `<root><data Id="payload">secret</data></root>`
	key := generateRSAKey(t)
	doc := mustParseXML(t, xml)

	target := findByLocalNameAndID(doc, "data", "payload")
	require.NotNil(t, target)

	ref := xmldsig1.ReferenceConfig{
		URI:             "#payload",
		DigestAlgorithm: xmldsig1.DigestSHA256,
		Transforms:      []xmldsig1.Transform{xmldsig1.ExcC14NTransform()},
	}

	sigElem, err := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(ref).
		SignDetached(t.Context(), doc, key)
	require.NoError(t, err)
	require.NoError(t, doc.DocumentElement().AddChild(sigElem))

	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
	res, err := verifier.Verify(t.Context(), doc)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Len(t, res.References, 1)
	require.Equal(t, "#payload", res.References[0].URI)
	require.Same(t, target, res.References[0].Element)
	require.Same(t, target, res.SignedElement("#payload"))
	require.True(t, res.Covers(target))
}

// TestVerifyPreservesSiblingPosition signs an enveloped signature and
// asserts that Verify is non-mutating — the serialized bytes of the
// document must be identical before and after Verify, even though Verify
// internally detaches the Signature element to canonicalize the enveloped
// content. The non-trailing variant below covers the case where naive
// AddChild reattachment would visibly relocate the Signature.
func TestVerifyPreservesSiblingPosition(t *testing.T) {
	key := generateRSAKey(t)
	doc := mustParseXML(t, samlAssertion)

	signer := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(xmldsig1.NewEnvelopedReference())
	require.NoError(t, signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key))

	before, err := helium.WriteString(doc)
	require.NoError(t, err)

	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
	_, err = verifier.Verify(t.Context(), doc)
	require.NoError(t, err)

	after, err := helium.WriteString(doc)
	require.NoError(t, err)
	require.Equal(t, before, after, "Verify must not restructure the document")
}

// TestVerifyPreservesSiblingPosition_NonTrailing places the Signature at a
// non-trailing sibling position and asserts the same invariance. This
// catches the "AddChild reattaches to end" bug specifically.
func TestVerifyPreservesSiblingPosition_NonTrailing(t *testing.T) {
	// Build doc with Issuer, then Signature, then Subject siblings.
	src := `<saml:Assertion xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" ID="_abc123" IssueInstant="2024-01-01T00:00:00Z" Version="2.0"><saml:Issuer>https://idp.example.com</saml:Issuer><saml:Subject><saml:NameID>user@example.com</saml:NameID></saml:Subject></saml:Assertion>`
	key := generateRSAKey(t)
	doc := mustParseXML(t, src)

	signer := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(xmldsig1.NewEnvelopedReference())
	require.NoError(t, signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key))

	// Serialize+reparse so we can manipulate the Signature node position.
	signed, err := helium.WriteString(doc)
	require.NoError(t, err)

	// Manually relocate the Signature to between Issuer and Subject by
	// splicing the serialized form. Order in signed doc currently has
	// Signature at the end; move it.
	sigStart := strings.Index(signed, "<ds:Signature")
	require.GreaterOrEqual(t, sigStart, 0)
	sigEnd := strings.Index(signed, "</ds:Signature>") + len("</ds:Signature>")
	require.Greater(t, sigEnd, sigStart)
	sigBlock := signed[sigStart:sigEnd]
	// Remove the trailing Signature.
	without := signed[:sigStart] + signed[sigEnd:]
	// Insert between Issuer and Subject.
	insertAt := strings.Index(without, "<saml:Subject")
	require.GreaterOrEqual(t, insertAt, 0)
	relocated := without[:insertAt] + sigBlock + without[insertAt:]

	doc2 := mustParseXML(t, relocated)
	before, err := helium.WriteString(doc2)
	require.NoError(t, err)

	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
	_, err = verifier.Verify(t.Context(), doc2)
	require.NoError(t, err)

	after, err := helium.WriteString(doc2)
	require.NoError(t, err)
	require.Equal(t, before, after, "Verify must preserve Signature sibling position")
}

// TestVerifyMultipleSignatures rejects ambiguous documents containing
// more than one Signature. The caller must use VerifyElement to choose.
// We use detached signatures so we can guarantee that adding a duplicate
// Signature element does not perturb the canonical bytes of the signed
// fragment.
func TestVerifyMultipleSignatures(t *testing.T) {
	xml := `<root><data Id="payload">secret</data></root>`
	key := generateRSAKey(t)
	doc := mustParseXML(t, xml)

	ref := xmldsig1.ReferenceConfig{
		URI:             "#payload",
		DigestAlgorithm: xmldsig1.DigestSHA256,
		Transforms:      []xmldsig1.Transform{xmldsig1.ExcC14NTransform()},
	}
	sigElem, err := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(ref).
		SignDetached(t.Context(), doc, key)
	require.NoError(t, err)
	require.NoError(t, doc.DocumentElement().AddChild(sigElem))

	signed, err := helium.WriteString(doc)
	require.NoError(t, err)

	// Inject a second copy of the (legitimate) Signature element. From
	// Verify's perspective the document is now ambiguous.
	sigStart := strings.Index(signed, "<ds:Signature")
	require.GreaterOrEqual(t, sigStart, 0, "signed output must contain a ds:Signature open tag")
	sigEnd := strings.Index(signed, "</ds:Signature>")
	require.Greater(t, sigEnd, sigStart, "signed output must contain a matching ds:Signature close tag")
	sigEnd += len("</ds:Signature>")
	sigBlock := signed[sigStart:sigEnd]
	doubled := signed[:sigEnd] + sigBlock + signed[sigEnd:]

	doc2 := mustParseXML(t, doubled)
	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
	_, err = verifier.Verify(t.Context(), doc2)
	require.ErrorIs(t, err, xmldsig1.ErrAmbiguousSignature)

	// But VerifyElement on a specific Signature element succeeds — detached
	// signatures don't depend on document-level structure beyond the
	// referenced fragment.
	var sig1 *helium.Element
	var walk func(n helium.Node)
	walk = func(n helium.Node) {
		if sig1 != nil {
			return
		}
		elem, ok := helium.AsNode[*helium.Element](n)
		if !ok {
			return
		}
		nm := elem.Name()
		if nm == "Signature" || strings.HasSuffix(nm, ":Signature") {
			sig1 = elem
			return
		}
		for c := elem.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(doc2.DocumentElement())
	require.NotNil(t, sig1)
	_, err = verifier.VerifyElement(t.Context(), doc2, sig1)
	require.NoError(t, err)
}

// TestSignEnvelopedSiblingPositionStable locks in the current behavior
// (Signature appended at end of parent) and ensures any future change is
// deliberate. The signed document's canonical form is byte-stable across
// Verify.
func TestSignEnvelopedSiblingPositionStable(t *testing.T) {
	key := generateRSAKey(t)
	doc := mustParseXML(t, samlAssertion)

	signer := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(xmldsig1.NewEnvelopedReference())
	require.NoError(t, signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key))

	// Verify twice — should be idempotent.
	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
	_, err := verifier.Verify(t.Context(), doc)
	require.NoError(t, err)
	out1, err := helium.WriteString(doc)
	require.NoError(t, err)

	_, err = verifier.Verify(t.Context(), doc)
	require.NoError(t, err)
	out2, err := helium.WriteString(doc)
	require.NoError(t, err)
	require.Equal(t, out1, out2)
}
