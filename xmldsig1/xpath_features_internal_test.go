package xmldsig1

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath1"
	"github.com/stretchr/testify/require"
)

// setElementText replaces elem's children with a single text node holding s.
func setElementText(t *testing.T, doc *helium.Document, elem *helium.Element, s string) {
	t.Helper()
	for c := elem.FirstChild(); c != nil; {
		next := c.NextSibling()
		if m, ok := c.(helium.MutableNode); ok {
			helium.UnlinkNode(m)
		}
		c = next
	}
	require.NoError(t, elem.AddChild(doc.CreateText([]byte(s))))
}

// TestGeneralXPointerVerifyRoundTrip proves the opt-in general XPointer resolves
// through the public Verifier.AllowXPointer path end to end: a hand-built,
// cryptographically valid signature whose single Reference selects an element via
// #xmlns()xpointer(//p:target) verifies, and the same document is rejected with a
// fail-closed ErrReferenceNotFound when AllowXPointer is left off.
func TestGeneralXPointerVerifyRoundTrip(t *testing.T) {
	const xpointerURI = "#xmlns(p=urn:t)xpointer(//p:target)"
	const xml = `<root xmlns:t="urn:t">` +
		`<t:target><t:v>SIGNED</t:v></t:target>` +
		`<ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#">` +
		`<ds:SignedInfo>` +
		`<ds:CanonicalizationMethod Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"/>` +
		`<ds:SignatureMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"/>` +
		`<ds:Reference URI="` + xpointerURI + `">` +
		`<ds:Transforms>` +
		`<ds:Transform Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"/>` +
		`</ds:Transforms>` +
		`<ds:DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"/>` +
		`<ds:DigestValue>AA==</ds:DigestValue>` +
		`</ds:Reference>` +
		`</ds:SignedInfo>` +
		`<ds:SignatureValue>AA==</ds:SignatureValue>` +
		`</ds:Signature>` +
		`</root>`

	doc := mustParse(t, xml)
	sig := findSig(doc.DocumentElement())
	require.NotNil(t, sig)

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	parsed, err := parseSignatureElement(t.Context(), newVerifyBudget(&verifierConfig{}), sig)
	require.NoError(t, err)
	require.Len(t, parsed.references, 1)

	// Compute the real Reference digest over the XPointer-selected subtree using
	// the same canonicalization path verification will use, then write it into the
	// DigestValue element.
	cfg := &verifierConfig{keySource: StaticKey(&key.PublicKey), allowXPointer: true}
	_, canonical, _, err := canonicalizeReference(t.Context(), cfg, doc, sig, parsed.references[0])
	require.NoError(t, err)
	digest, err := computeDigest(parsed.references[0].digestAlgorithm, canonical, false)
	require.NoError(t, err)
	digestValueElem := findLocal(sig, "DigestValue")
	require.NotNil(t, digestValueElem)
	setElementText(t, doc, digestValueElem, base64.StdEncoding.EncodeToString(digest))

	// Canonicalize the (now digest-populated) SignedInfo and sign it, writing the
	// SignatureValue.
	signedInfoCanon, err := canonicalizeSubtree(parsed.c14nMethod, parsed.signedInfoElem, parsed.c14nPrefixes)
	require.NoError(t, err)
	sigValue, err := signBytes(parsed.signatureAlg, key, signedInfoCanon, false)
	require.NoError(t, err)
	sigValueElem := findLocal(sig, "SignatureValue")
	require.NotNil(t, sigValueElem)
	setElementText(t, doc, sigValueElem, base64.StdEncoding.EncodeToString(sigValue))

	t.Run("verifies with AllowXPointer", func(t *testing.T) {
		result, err := NewVerifier(StaticKey(&key.PublicKey)).AllowXPointer(true).Verify(t.Context(), doc)
		require.NoError(t, err)
		require.Len(t, result.References, 1)
		require.Equal(t, xpointerURI, result.References[0].URI)
		require.Equal(t, findLocal(doc.DocumentElement(), "target"), result.References[0].Element)
	})

	t.Run("rejected without AllowXPointer", func(t *testing.T) {
		_, err := NewVerifier(StaticKey(&key.PublicKey)).Verify(t.Context(), doc)
		require.ErrorIs(t, err, ErrReferenceNotFound)
	})
}

// TestHereFunctionEnvelopedViaHere drives the classic "enveloped signature via
// here()" XPath filter transform (XMLDSig core §6.6.4 example) through the real
// verify canonicalization path. here() resolves to the ds:XPath element bearing
// the expression, so the standard filter keeps every node OUTSIDE the enclosing
// ds:Signature and drops the Signature's own subtree.
func TestHereFunctionEnvelopedViaHere(t *testing.T) {
	// The expression is true for a node whose ancestor-or-self Signature set is
	// smaller than the same set unioned with here()'s Signature — i.e. a node not
	// inside the Signature that here() belongs to.
	const xml = `<root xmlns:ds="http://www.w3.org/2000/09/xmldsig#">` +
		`<data>KEEPVAL</data>` +
		`<ds:Signature>` +
		`<ds:SignedInfo>` +
		`<ds:CanonicalizationMethod Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"/>` +
		`<ds:SignatureMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"/>` +
		`<ds:Reference URI="">` +
		`<ds:Transforms>` +
		`<ds:Transform Algorithm="http://www.w3.org/TR/1999/REC-xpath-19991116">` +
		`<ds:XPath>count(ancestor-or-self::ds:Signature | here()/ancestor::ds:Signature[1]) &gt; count(ancestor-or-self::ds:Signature)</ds:XPath>` +
		`</ds:Transform>` +
		`<ds:Transform Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"/>` +
		`</ds:Transforms>` +
		`<ds:DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"/>` +
		`<ds:DigestValue>AA==</ds:DigestValue>` +
		`</ds:Reference>` +
		`</ds:SignedInfo>` +
		`<ds:SignatureValue>AA==</ds:SignatureValue>` +
		`</ds:Signature>` +
		`</root>`
	doc := mustParse(t, xml)
	sig := findSig(doc.DocumentElement())
	require.NotNil(t, sig)

	parsed, err := parseSignatureElement(t.Context(), newVerifyBudget(&verifierConfig{}), sig)
	require.NoError(t, err)
	require.Len(t, parsed.references, 1)

	target, canonical, external, err := canonicalizeReference(t.Context(), &verifierConfig{}, doc, sig, parsed.references[0])
	require.NoError(t, err)
	require.False(t, external)
	require.Equal(t, doc.DocumentElement(), target)
	require.Contains(t, string(canonical), "KEEPVAL", "content outside the Signature must survive the here() filter")
	require.NotContains(t, string(canonical), "SignatureValue", "the Signature subtree must be dropped by the here() filter")
}

// TestHereFunctionFailsClosed locks the fail-closed edges of here(): a nil
// bearing node (the signing path, or any context with no ds:XPath element)
// yields the typed ErrHereUnavailable, and a here() call with any argument is
// rejected as an unsupported transform.
func TestHereFunctionFailsClosed(t *testing.T) {
	const xml = `<root xmlns:t="urn:t"><t:sec Id="x"><t:v>V</t:v></t:sec></root>`

	t.Run("nil bearing node", func(t *testing.T) {
		doc := mustParse(t, xml)
		ref := parsedReference{
			uri:             "#x",
			digestAlgorithm: DigestSHA256,
			transforms: []parsedTransform{
				// xpathHere left nil: here() has no bearing node and must fail closed.
				{algorithm: TransformXPath, xpathExpr: "here()", xpathNS: map[string]string{"t": "urn:t"}},
				{algorithm: C14N11URI},
			},
		}
		_, canon, _, err := canonicalizeReference(t.Context(), &verifierConfig{}, doc, nil, ref)
		require.ErrorIs(t, err, ErrHereUnavailable)
		require.Nil(t, canon)
	})

	t.Run("non-zero arguments", func(t *testing.T) {
		doc := mustParse(t, xml)
		ref := parsedReference{
			uri:             "#x",
			digestAlgorithm: DigestSHA256,
			transforms: []parsedTransform{
				{algorithm: TransformXPath, xpathExpr: "here(1)", xpathNS: map[string]string{"t": "urn:t"}, xpathHere: doc.DocumentElement()},
				{algorithm: C14N11URI},
			},
		}
		_, canon, _, err := canonicalizeReference(t.Context(), &verifierConfig{}, doc, nil, ref)
		require.ErrorIs(t, err, ErrUnsupportedTransform)
		require.Nil(t, canon)
	})
}

// TestGeneralXPointerResolution exercises the opt-in general XPointer resolver
// through canonicalizeReference: a unique match resolves and canonicalizes its
// subtree, an empty match fails ErrReferenceNotFound, a duplicate match fails
// ErrAmbiguousReference (the XSW defense), and the whole feature stays disabled
// unless Verifier.AllowXPointer opted in.
func TestGeneralXPointerResolution(t *testing.T) {
	xpointerRef := func(uri string) parsedReference {
		return parsedReference{
			uri:             uri,
			digestAlgorithm: DigestSHA256,
			transforms:      []parsedTransform{{algorithm: C14N11URI}},
		}
	}

	t.Run("unique element resolves", func(t *testing.T) {
		const xml = `<root xmlns:t="urn:t"><t:sec><t:v>V</t:v></t:sec><t:other>O</t:other></root>`
		doc := mustParse(t, xml)
		ref := xpointerRef("#xmlns(p=urn:t)xpointer(//p:sec)")
		cfg := &verifierConfig{allowXPointer: true}
		target, canonical, external, err := canonicalizeReference(t.Context(), cfg, doc, nil, ref)
		require.NoError(t, err)
		require.False(t, external)
		require.Equal(t, findLocal(doc.DocumentElement(), "sec"), target)
		require.Contains(t, string(canonical), "V")
		require.NotContains(t, string(canonical), "O", "only the selected subtree is canonicalized")
	})

	t.Run("empty node-set is not found", func(t *testing.T) {
		const xml = `<root xmlns:t="urn:t"><t:sec/></root>`
		doc := mustParse(t, xml)
		ref := xpointerRef("#xmlns(p=urn:t)xpointer(//p:missing)")
		cfg := &verifierConfig{allowXPointer: true}
		_, _, _, err := canonicalizeReference(t.Context(), cfg, doc, nil, ref)
		require.ErrorIs(t, err, ErrReferenceNotFound)
	})

	t.Run("multiple matches are ambiguous", func(t *testing.T) {
		const xml = `<root xmlns:t="urn:t"><t:sec/><t:sec/></root>`
		doc := mustParse(t, xml)
		ref := xpointerRef("#xmlns(p=urn:t)xpointer(//p:sec)")
		cfg := &verifierConfig{allowXPointer: true}
		_, _, _, err := canonicalizeReference(t.Context(), cfg, doc, nil, ref)
		require.ErrorIs(t, err, ErrAmbiguousReference)
	})

	t.Run("non-element principal is ambiguous", func(t *testing.T) {
		const xml = `<root xmlns:t="urn:t"><t:sec>V</t:sec></root>`
		doc := mustParse(t, xml)
		ref := xpointerRef("#xmlns(p=urn:t)xpointer(//p:sec/text())")
		cfg := &verifierConfig{allowXPointer: true}
		_, _, _, err := canonicalizeReference(t.Context(), cfg, doc, nil, ref)
		require.ErrorIs(t, err, ErrAmbiguousReference)
	})

	t.Run("literal id() keeps duplicate detection", func(t *testing.T) {
		const xml = `<root><a Id="dup"/><b Id="dup"/></root>`
		doc := mustParse(t, xml)
		ref := xpointerRef("#xpointer(id('dup'))")
		cfg := &verifierConfig{allowXPointer: true}
		_, _, _, err := canonicalizeReference(t.Context(), cfg, doc, nil, ref)
		require.ErrorIs(t, err, ErrAmbiguousReference)
	})

	// A whitespace-spelled id() selector (id ('dup')) that the literal fast path
	// does NOT match must still route through the duplicate-detecting
	// findElementsByIDUnder, NOT xpath1's built-in id(). Under duplicate xml:id,
	// the built-in's Document.GetElementByID returns the LAST duplicate (a silent
	// XSW resolution to the wrong element); the resolver must instead surface
	// ErrAmbiguousReference.
	t.Run("whitespace id() selector keeps duplicate detection", func(t *testing.T) {
		const xml = `<root><a xml:id="dup">A</a><b xml:id="dup">B</b></root>`
		doc := mustParse(t, xml)
		ref := xpointerRef("#xpointer(id ('dup'))")
		cfg := &verifierConfig{allowXPointer: true}
		_, _, _, err := canonicalizeReference(t.Context(), cfg, doc, nil, ref)
		require.ErrorIs(t, err, ErrAmbiguousReference)
	})

	// Any id() use that is not the whole-expression selector (a wrapping paren, a
	// predicate) cannot be routed through findElementsByIDUnder, so it is rejected
	// fail-closed rather than handed to the built-in id() — under duplicate xml:id
	// the built-in would otherwise silently resolve to the last such element.
	for _, expr := range []string{"(id('dup'))", "//a[id('dup')]"} {
		t.Run("non-selector id() use is fail-closed: "+expr, func(t *testing.T) {
			const xml = `<root><a xml:id="dup">A</a><b xml:id="dup">B</b></root>`
			doc := mustParse(t, xml)
			ref := xpointerRef("#xpointer(" + expr + ")")
			cfg := &verifierConfig{allowXPointer: true}
			target, _, _, err := canonicalizeReference(t.Context(), cfg, doc, nil, ref)
			require.ErrorIs(t, err, ErrReferenceNotFound)
			require.Nil(t, target)
		})
	}

	t.Run("here() via general XPointer fails closed", func(t *testing.T) {
		const xml = `<root><a Id="x"/></root>`
		doc := mustParse(t, xml)
		// A URI-borne XPointer has no ds:XPath bearing node, so here() is
		// unavailable. The sentinel must survive the general-XPointer error path
		// as a matchable ErrHereUnavailable, not be flattened into
		// ErrReferenceNotFound.
		ref := xpointerRef("#xpointer(here())")
		cfg := &verifierConfig{allowXPointer: true}
		_, _, _, err := canonicalizeReference(t.Context(), cfg, doc, nil, ref)
		require.ErrorIs(t, err, ErrHereUnavailable)
	})

	t.Run("disabled by default", func(t *testing.T) {
		const xml = `<root xmlns:t="urn:t"><t:sec><t:v>V</t:v></t:sec></root>`
		doc := mustParse(t, xml)
		ref := xpointerRef("#xmlns(p=urn:t)xpointer(//p:sec)")
		// allowXPointer defaults false: the URI is treated as an unresolved
		// external reference and, with no resolver, fails closed.
		_, _, _, err := canonicalizeReference(t.Context(), &verifierConfig{}, doc, nil, ref)
		require.ErrorIs(t, err, ErrReferenceNotFound)
	})
}

// TestParseGeneralXPointer covers the XPointer framework URI recognition: which
// shapes match, the xmlns() overrides, and the circumflex-escape unescaping.
func TestParseGeneralXPointer(t *testing.T) {
	t.Run("xpointer only", func(t *testing.T) {
		overrides, expr, ok := parseGeneralXPointer("#xpointer(//foo)")
		require.True(t, ok)
		require.Equal(t, "//foo", expr)
		require.Empty(t, overrides)
	})

	t.Run("xmlns then xpointer", func(t *testing.T) {
		overrides, expr, ok := parseGeneralXPointer("#xmlns(a=urn:x) xmlns(b=urn:y)xpointer(//a:foo/b:bar)")
		require.True(t, ok)
		require.Equal(t, "//a:foo/b:bar", expr)
		require.Equal(t, map[string]string{"a": "urn:x", "b": "urn:y"}, overrides)
	})

	t.Run("circumflex escape is unescaped", func(t *testing.T) {
		_, expr, ok := parseGeneralXPointer("#xpointer(//a[contains(.,'^(x^)')])")
		require.True(t, ok)
		require.Equal(t, "//a[contains(.,'(x)')]", expr)
	})

	t.Run("xmlns after xpointer does not match", func(t *testing.T) {
		// The XPointer framework grammar requires every xmlns() part to precede
		// the xpointer() part. A trailing xmlns() must NOT be bound out of order;
		// the URI fails to match and stays fail-closed.
		_, _, ok := parseGeneralXPointer("#xpointer(//a:foo)xmlns(a=urn:x)")
		require.False(t, ok)
	})

	t.Run("unsupported scheme does not match", func(t *testing.T) {
		_, _, ok := parseGeneralXPointer("#element(/1/2)")
		require.False(t, ok)
	})

	t.Run("bare fragment does not match", func(t *testing.T) {
		_, _, ok := parseGeneralXPointer("#someid")
		require.False(t, ok)
	})

	t.Run("malformed xmlns does not match", func(t *testing.T) {
		_, _, ok := parseGeneralXPointer("#xmlns(nope)xpointer(//foo)")
		require.False(t, ok)
	})

	t.Run("unbalanced parens do not match", func(t *testing.T) {
		_, _, ok := parseGeneralXPointer("#xpointer(//foo")
		require.False(t, ok)
	})

	t.Run("external URL does not match", func(t *testing.T) {
		_, _, ok := parseGeneralXPointer("http://example.com/doc.xml")
		require.False(t, ok)
	})
}

// TestDSigXPathEvaluatorOpLimit confirms the shared evaluator enforces its
// operation-count bound: a heavy expression under a low OpLimit trips
// xpath1.ErrOpLimit instead of running to completion.
func TestDSigXPathEvaluatorOpLimit(t *testing.T) {
	const xml = `<root><a><b/><b/></a><a><b/><b/></a><a><b/><b/></a></root>`
	doc := mustParse(t, xml)

	expr, err := xpath1.Compile("count(//node()//node())")
	require.NoError(t, err)

	eval := newDSigXPathEvaluator(nil, nil, 3)
	_, err = eval.Evaluate(t.Context(), expr, doc.DocumentElement())
	require.ErrorIs(t, err, xpath1.ErrOpLimit)
}
