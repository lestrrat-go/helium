package xmldsig1_test

import (
	"context"
	"regexp"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmldsig1"
	"github.com/stretchr/testify/require"
)

// wrapElementBase64 line-wraps the text content of the named element at the
// given column using the full set of XML whitespace separators (CR, LF, tab,
// space), mimicking real-world signers that pretty-print/indent base64. XSD
// base64Binary permits interspersed XML whitespace, so a verifier must tolerate
// all of these. Go's base64 decoder skips CR/LF but rejects space and tab, so
// spaces/tabs are what actually exercise the strip-before-decode fix.
func wrapElementBase64(t *testing.T, xml, tag string, col int) string {
	t.Helper()
	re := regexp.MustCompile(`(<(?:\w+:)?` + tag + `[^>]*>)([^<]+)(</(?:\w+:)?` + tag + `>)`)
	loc := re.FindStringSubmatchIndex(xml)
	require.NotNil(t, loc, "expected to find <%s> element in serialized signature", tag)
	body := xml[loc[4]:loc[5]]
	require.NotContains(t, body, " ", "test setup: %s body should be single-line before wrapping", tag)

	// Rotate through all XML whitespace so the fix must handle every variant.
	seps := []string{"\n", "  \t", "\r\n\t", " "}
	var sb strings.Builder
	sepIdx := 0
	for i, r := range body {
		if i > 0 && i%col == 0 {
			sb.WriteString(seps[sepIdx%len(seps)])
			sepIdx++
		}
		sb.WriteRune(r)
	}
	wrapped := sb.String()
	require.Contains(t, wrapped, " ", "test setup: wrapping should introduce a space into %s", tag)
	return re.ReplaceAllString(xml, "${1}"+wrapped+"${3}")
}

// TestVerifyLineWrapped covers line-wrapped (whitespace-interspersed) base64 in
// various signature fields.
func TestVerifyLineWrapped(t *testing.T) {
	// SignatureValue: line-wrapped (valid xs:base64Binary with interspersed
	// whitespace) still verifies rather than failing to base64-decode.
	// SignatureValue lives outside SignedInfo, so wrapping it post-signing does
	// not disturb the signed bytes.
	t.Run("signature value", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, samlAssertion)

		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			Reference(xmldsig1.NewEnvelopedReference())

		err := signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key)
		require.NoError(t, err)

		xml, err := helium.WriteString(doc)
		require.NoError(t, err)

		xml = wrapElementBase64(t, xml, "SignatureValue", 64)

		doc2 := mustParseXML(t, xml)

		verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
		_, err = verifier.Verify(t.Context(), doc2)
		require.NoError(t, err)
	})

	// X509Certificate: a line-wrapped X509Certificate in KeyInfo still decodes
	// and resolves the key.
	t.Run("x509 certificate", func(t *testing.T) {
		key := generateRSAKey(t)
		cert := generateSelfSignedCert(t, key)
		doc := mustParseXML(t, samlAssertion)

		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			Reference(xmldsig1.NewEnvelopedReference()).
			KeyInfo(xmldsig1.X509DataKeyInfo(cert))

		err := signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key)
		require.NoError(t, err)

		xml, err := helium.WriteString(doc)
		require.NoError(t, err)

		xml = wrapElementBase64(t, xml, "X509Certificate", 64)

		doc2 := mustParseXML(t, xml)

		verifier := xmldsig1.NewVerifier(xmldsig1.X509CertKeySource(cert))
		_, err = verifier.Verify(t.Context(), doc2)
		require.NoError(t, err)
	})

	// RSAKeyValue: line-wrapped Modulus/Exponent in an RSAKeyValue still decode.
	t.Run("rsa key value", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, samlAssertion)

		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			Reference(xmldsig1.NewEnvelopedReference()).
			KeyInfo(xmldsig1.RSAKeyValueKeyInfo())

		err := signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key)
		require.NoError(t, err)

		xml, err := helium.WriteString(doc)
		require.NoError(t, err)

		xml = wrapElementBase64(t, xml, "Modulus", 64)

		doc2 := mustParseXML(t, xml)

		verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
		_, err = verifier.Verify(t.Context(), doc2)
		require.NoError(t, err)
	})
}

// TestVerifyRejectsInclusiveNamespacesOnNonExclusiveSignedInfoC14N is the
// end-to-end (public Verify) counterpart to the internal
// TestVerifyRejectsInclusiveNamespacesOnNonExclusiveC14N. It guards against the
// runtime path accepting an ec:InclusiveNamespaces PrefixList on a
// non-exclusive SignedInfo CanonicalizationMethod (here C14N 1.1). Because
// canonicalize() only honors that PrefixList for exclusive c14n, a
// non-exclusive method declaring it would otherwise canonicalize SignedInfo
// differently from what the signer declared, so it must be rejected fail-closed
// before any key resolution or signature check. This exercises the public
// Verifier.Verify entry point — not just the internal parseSignedInfo — so the
// gating is proven on the real code path.
func TestVerifyRejectsInclusiveNamespacesOnNonExclusiveSignedInfoC14N(t *testing.T) {
	key := generateRSAKey(t)
	for _, alg := range []string{
		xmldsig1.C14N10,
		xmldsig1.C14N10Comments,
		xmldsig1.C14N11URI,
		xmldsig1.C14N11Comments,
	} {
		t.Run(alg, func(t *testing.T) {
			sig := `<ds:Signature xmlns:ds="` + xmldsig1.NamespaceDSig + `">` +
				`<ds:SignedInfo>` +
				`<ds:CanonicalizationMethod Algorithm="` + alg + `">` +
				`<ec:InclusiveNamespaces xmlns:ec="` + xmldsig1.ExcC14N10 + `" PrefixList="extra"/>` +
				`</ds:CanonicalizationMethod>` +
				`<ds:SignatureMethod Algorithm="` + xmldsig1.AlgRSASHA256 + `"/>` +
				`<ds:Reference URI="">` +
				`<ds:DigestMethod Algorithm="` + xmldsig1.DigestSHA256 + `"/>` +
				`<ds:DigestValue>AA==</ds:DigestValue>` +
				`</ds:Reference>` +
				`</ds:SignedInfo>` +
				`<ds:SignatureValue>AA==</ds:SignatureValue>` +
				`</ds:Signature>`
			doc := mustParseXML(t, `<root>`+sig+`</root>`)
			verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
			_, err := verifier.Verify(t.Context(), doc)
			require.ErrorIs(t, err, xmldsig1.ErrUnsupportedTransform,
				"ec:InclusiveNamespaces on non-exclusive SignedInfo c14n must be rejected via public Verify")
			require.Contains(t, err.Error(), "ec:InclusiveNamespaces")
		})
	}
}

// findSignatureElement walks the tree and returns the first ds:Signature
// element, or nil if none is present.
func findSignatureElement(root helium.Node) *helium.Element {
	if root == nil {
		return nil
	}
	if e, ok := helium.AsNode[*helium.Element](root); ok {
		if e.LocalName() == "Signature" {
			return e
		}
	}
	for c := root.FirstChild(); c != nil; c = c.NextSibling() {
		if found := findSignatureElement(c); found != nil {
			return found
		}
	}
	return nil
}

func findElementByLocalName(root helium.Node, name string) *helium.Element {
	if root == nil {
		return nil
	}
	if e, ok := helium.AsNode[*helium.Element](root); ok {
		if e.LocalName() == name {
			return e
		}
	}
	for c := root.FirstChild(); c != nil; c = c.NextSibling() {
		if found := findElementByLocalName(c, name); found != nil {
			return found
		}
	}
	return nil
}

// TestVerifyEnveloped covers the non-mutation guarantees of enveloped-signature
// verification.
func TestVerifyEnveloped(t *testing.T) {
	// does not mutate DOM is the regression guard for D-SIG-001: the
	// enveloped-signature transform must NOT mutate the caller's live document.
	// The previous implementation unlinked the Signature element from the live
	// tree during canonicalization and reattached it afterward, which races with
	// concurrent readers and corrupts the document if the restore ever fails.
	// After Verify, the document must be byte-for-byte identical to before, and
	// the Signature element must remain linked at its original position with the
	// same node identity.
	t.Run("does not mutate dom", func(t *testing.T) {
		key := generateRSAKey(t)

		signDoc := mustParseXML(t, samlAssertion)
		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			Reference(xmldsig1.NewEnvelopedReference())
		require.NoError(t, signer.SignEnveloped(t.Context(), signDoc, signDoc.DocumentElement(), key))

		signed, err := helium.WriteString(signDoc)
		require.NoError(t, err)

		// Re-parse from the serialized form so the verifier operates on a fresh,
		// independently-owned tree.
		doc := mustParseXML(t, signed)
		before, err := helium.WriteString(doc)
		require.NoError(t, err)

		sigElem := findSignatureElement(doc.DocumentElement())
		require.NotNil(t, sigElem, "Signature must be present before verify")
		sigParentBefore := sigElem.Parent()

		verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
		_, err = verifier.Verify(t.Context(), doc)
		require.NoError(t, err)

		after, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Equal(t, before, after, "Verify must not mutate the caller's DOM")

		sigAfter := findSignatureElement(doc.DocumentElement())
		require.NotNil(t, sigAfter, "Signature must still be linked after verify")
		require.Same(t, sigElem, sigAfter, "Signature element identity must be preserved")
		require.Same(t, sigParentBefore, sigAfter.Parent(), "Signature must remain at its original parent")
	})

	// idempotent confirms that, because Verify no longer mutates the DOM,
	// repeated verifications of the same document all succeed.
	t.Run("idempotent", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, samlAssertion)

		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			Reference(xmldsig1.NewEnvelopedReference())
		require.NoError(t, signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key))

		verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
		for i := range 3 {
			_, err := verifier.Verify(t.Context(), doc)
			require.NoErrorf(t, err, "verify #%d", i)
		}
	})

	// fragment does not mutate DOM exercises the URI="#id" enveloped path: the
	// Signature is nested inside the signed element and must be omitted from the
	// subtree canonicalization without mutating the live tree.
	t.Run("fragment does not mutate dom", func(t *testing.T) {
		key := generateRSAKey(t)

		signDoc := mustParseXML(t, `<root><data Id="mydata"><v>hello</v></data></root>`)
		dataElem := findElementByLocalName(signDoc.DocumentElement(), "data")
		require.NotNil(t, dataElem)

		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			Reference(xmldsig1.ReferenceConfig{
				URI:             refURIMyData,
				DigestAlgorithm: xmldsig1.DigestSHA256,
				Transforms:      []xmldsig1.Transform{xmldsig1.Enveloped(), xmldsig1.ExcC14NTransform()},
			})
		// Place the Signature inside the referenced element so it is enveloped.
		require.NoError(t, signer.SignEnveloped(t.Context(), signDoc, dataElem, key))

		signed, err := helium.WriteString(signDoc)
		require.NoError(t, err)

		doc := mustParseXML(t, signed)
		before, err := helium.WriteString(doc)
		require.NoError(t, err)

		verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
		_, err = verifier.Verify(t.Context(), doc)
		require.NoError(t, err)

		after, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Equal(t, before, after, "Verify must not mutate the caller's DOM for fragment references")
	})

	// fragment signature before target is the regression guard for D-SIG-002:
	// when the ds:Signature is a SIBLING that precedes the referenced target
	// element, unlinking the cloned Signature shifts the target's child index.
	// The enveloped c14n must resolve the cloned target by its pre-unlink path
	// (before detaching the Signature), or it resolves the wrong/nil subtree and
	// verification fails with "reference target in canonicalization copy is not
	// an element". The live DOM must also remain unmutated.
	t.Run("fragment signature before target", func(t *testing.T) {
		key := generateRSAKey(t)

		signDoc := mustParseXML(t, `<root><x Id="x"><v>hello</v></x></root>`)
		root := signDoc.DocumentElement()
		xElem := findElementByLocalName(root, "x")
		require.NotNil(t, xElem)

		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			Reference(xmldsig1.ReferenceConfig{
				URI:             "#x",
				DigestAlgorithm: xmldsig1.DigestSHA256,
				Transforms:      []xmldsig1.Transform{xmldsig1.Enveloped(), xmldsig1.ExcC14NTransform()},
			})
		// Place the Signature as a child of root; it is appended AFTER <x>.
		require.NoError(t, signer.SignEnveloped(t.Context(), signDoc, root, key))

		// Reorder so the Signature precedes <x> as an earlier sibling. Moving <x>
		// past the Signature does not change <x>'s content or the enveloped omit
		// set, so the digest stays valid — but it forces the unlink-shifts-index
		// condition the fix guards against.
		helium.UnlinkNode(xElem)
		require.NoError(t, root.AddChild(xElem))

		sigElem := findSignatureElement(root)
		require.NotNil(t, sigElem)
		require.Same(t, sigElem, root.FirstChild(), "Signature must now be the first child")

		signed, err := helium.WriteString(signDoc)
		require.NoError(t, err)

		doc := mustParseXML(t, signed)
		before, err := helium.WriteString(doc)
		require.NoError(t, err)

		verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
		_, err = verifier.Verify(t.Context(), doc)
		require.NoError(t, err, "verify must resolve the shifted-index target")

		after, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Equal(t, before, after, "Verify must not mutate the caller's DOM")
	})
}

// TestVerifyDuplicateRejected covers rejection of structurally-ambiguous
// signatures carrying duplicated SignedInfo / SignatureValue children.
func TestVerifyDuplicateRejected(t *testing.T) {
	// duplicate SignedInfo mounts the duplicate-SignedInfo attack: the genuine,
	// signed SignedInfo is the LAST child of the Signature (so it is the one
	// canonicalized and checked against SignatureValue), while an attacker
	// prepends a second, UNSIGNED SignedInfo carrying a Reference to
	// attacker-controlled content with a self-consistent DigestValue. Verify must
	// reject the document rather than report the injected reference as covered by
	// the signature.
	t.Run("signed info", func(t *testing.T) {
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

		// Extract the genuine SignedInfo block from the signed output.
		siStart := strings.Index(signed, "<ds:SignedInfo")
		require.GreaterOrEqual(t, siStart, 0)
		siEnd := strings.Index(signed, "</ds:SignedInfo>") + len("</ds:SignedInfo>")
		require.Greater(t, siEnd, siStart)
		genuineSI := signed[siStart:siEnd]

		// Build the injected, unsigned SignedInfo by computing a correct digest
		// for an evil payload. We obtain that digest by signing a throwaway doc
		// containing the evil element and lifting its SignedInfo verbatim — the
		// attacker controls the document, so this digest is self-consistent.
		evilDoc := mustParseXML(t, `<root><data Id="evil">attacker-controlled</data></root>`)
		evilSig, err := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			Reference(xmldsig1.ReferenceConfig{
				URI:             "#evil",
				DigestAlgorithm: xmldsig1.DigestSHA256,
				Transforms:      []xmldsig1.Transform{xmldsig1.ExcC14NTransform()},
			}).
			SignDetached(t.Context(), evilDoc, generateRSAKey(t))
		require.NoError(t, err)
		require.NoError(t, evilDoc.DocumentElement().AddChild(evilSig))
		evilSigned, err := helium.WriteString(evilDoc)
		require.NoError(t, err)
		eStart := strings.Index(evilSigned, "<ds:SignedInfo")
		eEnd := strings.Index(evilSigned, "</ds:SignedInfo>") + len("</ds:SignedInfo>")
		injectedSI := evilSigned[eStart:eEnd]

		// Inject: prepend the unsigned (evil) SignedInfo before the genuine one,
		// and inject the evil element into the document body. The genuine
		// SignedInfo remains the last SignedInfo child of the Signature, so it is
		// still the one whose canonical form matches SignatureValue.
		require.NotEqual(t, genuineSI, injectedSI)
		tampered := strings.Replace(signed, genuineSI, injectedSI+genuineSI, 1)
		tampered = strings.Replace(tampered, `<data Id="payload">secret</data>`,
			`<data Id="evil">attacker-controlled</data><data Id="payload">secret</data>`, 1)
		require.NotEqual(t, signed, tampered)

		doc2 := mustParseXML(t, tampered)
		verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
		res, err := verifier.Verify(t.Context(), doc2)

		// The document is structurally invalid (two SignedInfo elements in one
		// Signature). Verify must reject it outright.
		require.Error(t, err, "duplicate SignedInfo must be rejected")
		if res != nil {
			for _, r := range res.References {
				require.NotEqual(t, "#evil", r.URI,
					"injected, unsigned reference must never be reported as verified")
			}
		}
	})

	// signature value guards against a Signature element carrying more than one
	// SignatureValue child.
	t.Run("signature value", func(t *testing.T) {
		xml := `<root><data Id="payload">secret</data></root>`
		key := generateRSAKey(t)
		doc := mustParseXML(t, xml)

		sigElem, err := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			Reference(xmldsig1.ReferenceConfig{
				URI:             "#payload",
				DigestAlgorithm: xmldsig1.DigestSHA256,
				Transforms:      []xmldsig1.Transform{xmldsig1.ExcC14NTransform()},
			}).
			SignDetached(t.Context(), doc, key)
		require.NoError(t, err)
		require.NoError(t, doc.DocumentElement().AddChild(sigElem))

		signed, err := helium.WriteString(doc)
		require.NoError(t, err)

		svStart := strings.Index(signed, "<ds:SignatureValue")
		require.GreaterOrEqual(t, svStart, 0)
		svEnd := strings.Index(signed, "</ds:SignatureValue>") + len("</ds:SignatureValue>")
		svBlock := signed[svStart:svEnd]
		tampered := strings.Replace(signed, svBlock, svBlock+svBlock, 1)
		require.NotEqual(t, signed, tampered)

		doc2 := mustParseXML(t, tampered)
		verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
		_, err = verifier.Verify(t.Context(), doc2)
		require.Error(t, err, "duplicate SignatureValue must be rejected")
	})
}

// pointerKeySource is a pointer-receiver KeySource used to construct a typed-nil
// pointer value. A typed-nil pointer carries a concrete type, so the interface
// is non-nil and survives a plain == nil check; ResolveKey would then be called
// on a nil receiver.
type pointerKeySource struct{}

func (*pointerKeySource) ResolveKey(_ context.Context, _ *xmldsig1.KeyInfoData, _ string) (any, error) {
	// Must be unreachable in TestTypedNilPointerKeySource: a typed-nil
	// *pointerKeySource is detected by isNilKeySource before ResolveKey is ever
	// called. Panic here so the test fails loudly if the isNilKeySource guard is
	// removed and ResolveKey is invoked on the nil receiver.
	panic("typed-nil *pointerKeySource ResolveKey should not be called")
}

// nilKeySourceSigDoc is a minimal document carrying a single ds:Signature so
// the verify path reaches the key-resolution step.
const nilKeySourceSigDoc = `<doc xmlns:ds="http://www.w3.org/2000/09/xmldsig#">` +
	`<ds:Signature>` +
	`<ds:SignedInfo>` +
	`<ds:CanonicalizationMethod Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"/>` +
	`<ds:SignatureMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"/>` +
	`<ds:Reference URI="">` +
	`<ds:DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"/>` +
	`<ds:DigestValue>AAAA</ds:DigestValue>` +
	`</ds:Reference>` +
	`</ds:SignedInfo>` +
	`<ds:SignatureValue>AAAA</ds:SignatureValue>` +
	`</ds:Signature>` +
	`</doc>`

// TestVerifyNilKeySource covers nil/typed-nil/zero-value key source guards.
func TestVerifyNilKeySource(t *testing.T) {
	// nil key source: a nil KeySource must surface a typed error rather than
	// panic on a nil pointer dereference when Verify reaches ResolveKey.
	t.Run("nil", func(t *testing.T) {
		const sigDoc = `<doc xmlns:ds="http://www.w3.org/2000/09/xmldsig#">` +
			`<ds:Signature>` +
			`<ds:SignedInfo>` +
			`<ds:CanonicalizationMethod Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"/>` +
			`<ds:SignatureMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"/>` +
			`<ds:Reference URI="">` +
			`<ds:DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"/>` +
			`<ds:DigestValue>AAAA</ds:DigestValue>` +
			`</ds:Reference>` +
			`</ds:SignedInfo>` +
			`<ds:SignatureValue>AAAA</ds:SignatureValue>` +
			`</ds:Signature>` +
			`</doc>`

		doc := mustParseXML(t, sigDoc)

		require.NotPanics(t, func() {
			_, err := xmldsig1.NewVerifier(nil).Verify(t.Context(), doc)
			require.Error(t, err)
			require.ErrorIs(t, err, xmldsig1.ErrNoKeySource)
		})
	})

	// typed-nil func: a typed-nil KeySourceFunc passes the interface!=nil check in
	// verifySignature (the interface carries a concrete type), so ResolveKey must
	// guard the nil func itself and return a typed error rather than panic on the
	// nil call.
	t.Run("typed nil func", func(t *testing.T) {
		doc := mustParseXML(t, nilKeySourceSigDoc)

		var ks xmldsig1.KeySourceFunc // typed nil

		require.NotPanics(t, func() {
			_, err := xmldsig1.NewVerifier(ks).Verify(t.Context(), doc)
			require.Error(t, err)
			require.ErrorIs(t, err, xmldsig1.ErrNoKeySource)
		})
	})

	// typed-nil pointer: a typed-nil POINTER KeySource (e.g. var ks
	// *pointerKeySource; NewVerifier(ks)) yields a non-nil interface whose
	// underlying value is nil, so a plain cfg.keySource == nil check misses it and
	// ResolveKey would panic on the nil receiver. verifySignature must detect this
	// via isNilKeySource and return a typed error instead.
	t.Run("typed nil pointer", func(t *testing.T) {
		doc := mustParseXML(t, nilKeySourceSigDoc)

		var ks *pointerKeySource // typed-nil pointer

		require.NotPanics(t, func() {
			_, err := xmldsig1.NewVerifier(ks).Verify(t.Context(), doc)
			require.Error(t, err)
			require.ErrorIs(t, err, xmldsig1.ErrNoKeySource)
		})
	})

	// zero-value verifier: a zero-value Verifier{} constructed directly
	// (bypassing NewVerifier) has a nil cfg. Verify/VerifyElement must surface a
	// typed error rather than panic on the nil cfg dereference inside
	// verifySignature.
	t.Run("zero value verifier", func(t *testing.T) {
		doc := mustParseXML(t, nilKeySourceSigDoc)

		require.NotPanics(t, func() {
			_, err := xmldsig1.Verifier{}.Verify(t.Context(), doc)
			require.Error(t, err)
			require.ErrorIs(t, err, xmldsig1.ErrNoKeySource)
		})
	})
}

// TestVerifyHonorsContextCancellation ensures the per-Reference verification
// loop checks ctx between references: a document with several References must
// not be digested to completion once the caller's context is cancelled. The
// StaticKey source ignores ctx and verifyBytes does not check it, so a
// pre-cancelled context reaches the Reference loop, where cancellation must be
// observed and surfaced as the context error instead of a full verify.
func TestVerifyHonorsContextCancellation(t *testing.T) {
	xml := `<root><a Id="one">first</a><b Id="two">second</b></root>`
	key := generateRSAKey(t)
	doc := mustParseXML(t, xml)

	mkRef := func(uri string) xmldsig1.ReferenceConfig {
		return xmldsig1.ReferenceConfig{
			URI:             uri,
			DigestAlgorithm: xmldsig1.DigestSHA256,
			Transforms:      []xmldsig1.Transform{xmldsig1.ExcC14NTransform()},
		}
	}

	sigElem, err := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(mkRef("#one")).
		Reference(mkRef("#two")).
		SignDetached(t.Context(), doc, key)
	require.NoError(t, err)
	require.NoError(t, doc.DocumentElement().AddChild(sigElem))

	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))

	// Sanity: verifies under a live context.
	_, err = verifier.Verify(t.Context(), doc)
	require.NoError(t, err)

	// A context cancelled before verification must abort the Reference loop and
	// surface the context error rather than completing a full verification.
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = verifier.Verify(ctx, doc)
	require.ErrorIs(t, err, context.Canceled)
}
