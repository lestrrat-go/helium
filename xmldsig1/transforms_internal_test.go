package xmldsig1

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/domutil"
	"github.com/stretchr/testify/require"
)

// buildExcC14NReference constructs a ds:Reference whose single Transform is
// Exclusive C14N carrying an InclusiveNamespaces child placed in namespace
// incNS (declared under prefix incPx) with the given PrefixList. The Reference
// (and its core children) live in the core XML-Signature namespace so that only
// the InclusiveNamespaces namespace varies between cases.
func buildExcC14NReference(t *testing.T, doc *helium.Document, incPx, incNS, prefixList string) *helium.Element {
	t.Helper()

	ref, err := doc.CreateElement("Reference")
	require.NoError(t, err)
	require.NoError(t, ref.DeclareNamespace(nsPrefix, NamespaceDSig))
	require.NoError(t, ref.SetActiveNamespace(nsPrefix, NamespaceDSig))

	transforms, err := doc.CreateElement("Transforms")
	require.NoError(t, err)
	require.NoError(t, transforms.SetActiveNamespace(nsPrefix, NamespaceDSig))
	require.NoError(t, ref.AddChild(transforms))

	transform, err := doc.CreateElement("Transform")
	require.NoError(t, err)
	require.NoError(t, transform.SetActiveNamespace(nsPrefix, NamespaceDSig))
	require.NoError(t, transform.SetAttribute("Algorithm", ExcC14N10))
	require.NoError(t, transforms.AddChild(transform))

	inc, err := doc.CreateElement("InclusiveNamespaces")
	require.NoError(t, err)
	require.NoError(t, inc.DeclareNamespace(incPx, incNS))
	require.NoError(t, inc.SetActiveNamespace(incPx, incNS))
	require.NoError(t, inc.SetAttribute("PrefixList", prefixList))
	require.NoError(t, transform.AddChild(inc))

	// DigestMethod/DigestValue are mandatory under Reference's content model, so
	// include them: parseReferenceElement now rejects a Reference missing either,
	// and this helper exercises the transform/InclusiveNamespaces parsing of an
	// otherwise complete Reference.
	digestMethod, err := doc.CreateElement("DigestMethod")
	require.NoError(t, err)
	require.NoError(t, digestMethod.SetActiveNamespace(nsPrefix, NamespaceDSig))
	require.NoError(t, digestMethod.SetAttribute("Algorithm", DigestSHA256))
	require.NoError(t, ref.AddChild(digestMethod))

	digestValue, err := doc.CreateElement("DigestValue")
	require.NoError(t, err)
	require.NoError(t, digestValue.SetActiveNamespace(nsPrefix, NamespaceDSig))
	require.NoError(t, digestValue.AddChild(doc.CreateText([]byte("AA=="))))
	require.NoError(t, ref.AddChild(digestValue))

	return ref
}

func findSig(root helium.Node) *helium.Element {
	return findLocal(root, "Signature")
}

func findLocal(root helium.Node, name string) *helium.Element {
	if root == nil {
		return nil
	}
	if e, ok := helium.AsNode[*helium.Element](root); ok && e.LocalName() == name {
		return e
	}
	for c := root.FirstChild(); c != nil; c = c.NextSibling() {
		if f := findLocal(c, name); f != nil {
			return f
		}
	}
	return nil
}

// findTransformByAlgorithm locates the ds:Transform element inside a Reference
// whose Algorithm attribute matches algURI, so a test can mutate that specific
// transform.
func findTransformByAlgorithm(t *testing.T, signedInfo *helium.Element, algURI string) *helium.Element {
	t.Helper()
	ref := findChild(t, signedInfo, "Reference")
	transforms := findChild(t, ref, "Transforms")
	for c := transforms.FirstChild(); c != nil; c = c.NextSibling() {
		e, ok := helium.AsNode[*helium.Element](c)
		if !ok {
			continue
		}
		if domutil.LocalName(e) != "Transform" {
			continue
		}
		alg, _ := e.GetAttribute("Algorithm")
		if alg == algURI {
			return e
		}
	}
	t.Fatalf("Transform with Algorithm %q not found", algURI)
	return nil
}

// TestResolveC14NMode covers the comment-variant and unsupported arms of
// resolveC14NMode.
func TestResolveC14NMode(t *testing.T) {
	// unsupported covers the default (error) arm.
	t.Run("unsupported", func(t *testing.T) {
		_, _, err := resolveC14NMode("urn:not-a-c14n-method")
		require.ErrorIs(t, err, ErrUnsupportedAlgorithm)
	})

	// comments covers the comment-variant arms.
	t.Run("comments", func(t *testing.T) {
		for _, m := range []string{C14N10Comments, ExcC14N10Comments, C14N11Comments} {
			_, comments, err := resolveC14NMode(m)
			require.NoError(t, err)
			require.True(t, comments)
		}
	})
}

// TestExcC14NTransformPrefixes covers excC14NTransform.Prefixes.
func TestExcC14NTransformPrefixes(t *testing.T) {
	tr := ExcC14NTransform("a", "b")
	exc, ok := tr.(excC14NTransform)
	require.True(t, ok)
	require.Equal(t, []string{"a", "b"}, exc.Prefixes())
}

// TestInclusiveNamespaces guards against namespace confusion in the Exclusive
// C14N InclusiveNamespaces element.
func TestInclusiveNamespaces(t *testing.T) {
	// foreign namespace rejected guards against namespace confusion in the
	// Exclusive C14N InclusiveNamespaces element. That element lives only in the
	// exc-c14n namespace; matching on local name alone would let a
	// foreign-namespace <evil:InclusiveNamespaces> inject a PrefixList and alter
	// which namespaces are canonicalized. A foreign-namespace look-alike is not a
	// recognized Transform parameter, so it must be rejected fail-closed rather
	// than silently ignored — digesting as if an unknown child were absent is
	// fail-open.
	t.Run("foreign namespace rejected", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(context.Background(), []byte(`<root/>`))
		require.NoError(t, err)

		ref := buildExcC14NReference(t, doc, "evil", "urn:example:evil", "a b c")

		_, err = parseReferenceElement(ref)
		require.ErrorIs(t, err, ErrUnsupportedTransform)
		require.Contains(t, err.Error(), "Transform parameter")
	})

	// exc-c14n parsed is the positive control: an InclusiveNamespaces in the
	// exc-c14n namespace must still contribute its PrefixList.
	t.Run("exc-c14n parsed", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(context.Background(), []byte(`<root/>`))
		require.NoError(t, err)

		ref := buildExcC14NReference(t, doc, "ec", ExcC14N10, "a b c")

		parsed, err := parseReferenceElement(ref)
		require.NoError(t, err)
		require.Len(t, parsed.transforms, 1)
		require.Equal(t, []string{"a", "b", "c"}, parsed.transforms[0].prefixes,
			"a correctly exc-c14n-namespaced InclusiveNamespaces must contribute its prefixes")
	})
}

// TestCanonicalizeSubtreeKeepsNamespaceDecls is the load-bearing regression
// guard. A namespace-qualified signed subtree must canonicalize WITH its
// in-scope xmlns declarations. c14n node-set mode only emits namespaces that
// are explicitly present in the node set, so collectSubtreeNodes must include
// the in-scope namespace axis for every element. Without it the prefixed
// element names are emitted WITHOUT their xmlns:p declaration, producing
// non-W3C canonical bytes that break cross-implementation signature interop.
func TestCanonicalizeSubtreeKeepsNamespaceDecls(t *testing.T) {
	const xml = `<doc xmlns:p="urn:p"><p:target Id="x"><p:child>v</p:child></p:target></doc>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)

	target, err := resolveReference(doc, "#x")
	require.NoError(t, err)

	t.Run("inclusive C14N 1.0", func(t *testing.T) {
		out, err := canonicalizeSubtree(C14N10, target, nil)
		require.NoError(t, err)
		require.Contains(t, string(out), `xmlns:p="urn:p"`,
			"canonical subtree must carry the in-scope xmlns:p declaration")
		// The prefixed element name and its namespace decl must coexist.
		require.True(t, strings.HasPrefix(string(out), `<p:target xmlns:p="urn:p"`),
			"got: %s", out)
	})

	t.Run("exclusive C14N 1.0", func(t *testing.T) {
		out, err := canonicalizeSubtree(ExcC14N10, target, nil)
		require.NoError(t, err)
		require.Contains(t, string(out), `xmlns:p="urn:p"`,
			"exclusive canonical subtree must carry the visibly-utilized xmlns:p declaration")
	})

	t.Run("C14N 1.1", func(t *testing.T) {
		out, err := canonicalizeSubtree(C14N11URI, target, nil)
		require.NoError(t, err)
		require.Contains(t, string(out), `xmlns:p="urn:p"`,
			"C14N 1.1 canonical subtree must carry the in-scope xmlns:p declaration")
	})
}

// dtdEntityDecl returns the DTD entity-declaration node named name, or nil.
func dtdEntityDecl(doc *helium.Document, name string) helium.Node {
	for c := range helium.Children(doc) {
		if c.Type() != helium.DTDNode && c.Type() != helium.DocumentTypeNode {
			continue
		}
		for d := range helium.Children(c) {
			if d.Type() == helium.EntityNode && d.Name() == name {
				return d
			}
		}
	}
	return nil
}

// TestCollectSubtreeNodesNoDTDSpill guards the owned-boundary child enumeration
// in collectSubtreeNodes. An EntityRefNode's child is the shared Entity node
// owned by the DTD, whose sibling pointers thread into the DTD's declaration
// list. A raw FirstChild / NextSibling recursion escapes into those sibling
// declarations and pulls foreign DTD-declaration nodes into the c14n node set.
// collectSubtreeNodes enumerates via helium.Children — the same primitive the
// c14n canonicalizer uses to walk element children and expand an entity
// reference — so the node set holds only the owned subtree.
//
// The signed subtree references &foo; (the FIRST declaration); under the buggy
// recursion the following sibling declaration &bar; leaked into the node set.
func TestCollectSubtreeNodesNoDTDSpill(t *testing.T) {
	const xml = "<?xml version=\"1.0\"?>\n" +
		"<!DOCTYPE doc [\n" +
		"<!ENTITY foo \"FOO\">\n" +
		"<!ENTITY bar \"BAR\">\n" +
		"]>\n" +
		"<doc><target Id=\"x\"><child>pre &foo; post</child></target></doc>"

	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)
	target, err := resolveReference(doc, "#x")
	require.NoError(t, err)

	set := make(map[helium.Node]bool)
	for _, n := range collectSubtreeNodes(target) {
		set[n] = true
	}

	// Sanity: the owned subtree element is collected.
	require.True(t, set[helium.Node(target)], "the target subtree element must be collected")

	// The un-referenced sibling declaration bar must NOT leak into the node set.
	barDecl := dtdEntityDecl(doc, "bar")
	require.NotNil(t, barDecl, "bar entity declaration should exist in the DTD")
	require.False(t, set[barDecl],
		"un-referenced sibling entity declaration bar must not spill into the c14n node set")
}

// TestCanonicalizeSubtreeEntityFreeUnchanged locks the byte-identical invariant:
// an entity-free signed subtree canonicalizes to the same W3C bytes as before
// the owned-boundary enumeration change.
func TestCanonicalizeSubtreeEntityFreeUnchanged(t *testing.T) {
	const xml = `<doc><target Id="x"><child>v</child></target></doc>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)
	target, err := resolveReference(doc, "#x")
	require.NoError(t, err)

	out, err := canonicalizeSubtree(C14N10, target, nil)
	require.NoError(t, err)
	require.Equal(t, `<target Id="x"><child>v</child></target>`, string(out))
}

// TestCanonicalizeEnvelopedMatchesDetach is the byte-equivalence contract for
// the clone-based enveloped transform: the canonical bytes produced by cloning
// the document and omitting the Signature from the copy MUST equal the bytes
// produced by physically detaching the Signature from the live tree (the
// previous, mutating implementation). This guarantees the fix does not change
// any digest/signature value for valid documents while eliminating the live
// DOM mutation. It also asserts the live tree is byte-for-byte unchanged after
// the call.
func TestCanonicalizeEnvelopedMatchesDetach(t *testing.T) {
	const sigXML = `<ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#"><ds:SignedInfo/></ds:Signature>`

	cases := []struct {
		name    string
		xml     string
		method  string
		wholeID string // "" => whole-document reference; else local-name of target element
	}{
		{
			name:   "whole-exc-c14n",
			xml:    "<a:Root xmlns:a=\"urn:a\" ID=\"r\">\n  <a:Child>x</a:Child>\n  " + sigXML + "\n  <a:Tail>y</a:Tail>\n</a:Root>",
			method: ExcC14N10,
		},
		{
			name:   "whole-c14n10",
			xml:    "<a:Root xmlns:a=\"urn:a\" ID=\"r\">\n  <a:Child>x</a:Child>\n  " + sigXML + "\n  <a:Tail>y</a:Tail>\n</a:Root>",
			method: C14N10,
		},
		{
			name:   "whole-c14n11",
			xml:    "<a:Root xmlns:a=\"urn:a\" ID=\"r\">\n  <a:Child>x</a:Child>\n  " + sigXML + "\n  <a:Tail>y</a:Tail>\n</a:Root>",
			method: C14N11URI,
		},
		{
			name:    "fragment-exc-c14n",
			xml:     "<root xmlns:p=\"urn:p\"><data ID=\"d\"><v>hi</v>" + sigXML + "</data></root>",
			method:  ExcC14N10,
			wholeID: "data",
		},
		{
			name:    "fragment-c14n10-inherited-ns",
			xml:     "<root xmlns:p=\"urn:p\"><data ID=\"d\"><v>hi</v>" + sigXML + "</data></root>",
			method:  C14N10,
			wholeID: "data",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.xml))
			require.NoError(t, err)
			root := doc.DocumentElement()

			sig := findSig(root)
			require.NotNil(t, sig)

			target := root
			wholeDoc := tc.wholeID == ""
			if !wholeDoc {
				target = findLocal(root, tc.wholeID)
				require.NotNil(t, target)
			}

			// Reference bytes via the old approach: physically detach the
			// Signature, canonicalize, then reattach.
			parent, ok := sig.Parent().(helium.MutableNode)
			require.True(t, ok)
			next := sig.NextSibling()
			helium.UnlinkNode(sig)
			var want []byte
			if wholeDoc {
				want, err = canonicalize(tc.method, doc, nil)
			} else {
				want, err = canonicalizeSubtree(tc.method, target, nil)
			}
			require.NoError(t, err)
			// Reattach so the live tree is restored for the comparison below.
			if next == nil {
				require.NoError(t, parent.AddChild(sig))
			} else if nm, ok := next.(helium.MutableNode); ok {
				require.NoError(t, nm.Replace(sig, next))
			}

			liveBefore, err := helium.WriteString(doc)
			require.NoError(t, err)

			got, err := canonicalizeEnveloped(tc.method, doc, target, sig, wholeDoc, nil)
			require.NoError(t, err)

			require.Equal(t, string(want), string(got), "clone-based enveloped bytes must match the detach-based reference")

			liveAfter, err := helium.WriteString(doc)
			require.NoError(t, err)
			require.Equal(t, liveBefore, liveAfter, "canonicalizeEnveloped must not mutate the live document")
		})
	}
}

// TestTransformNamespace guards against namespace confusion in Transform
// elements.
func TestTransformNamespace(t *testing.T) {
	// foreign namespace rejected guards against a namespace-confusion bypass where
	// an attacker rewrites a core ds:Transform into a foreign namespace. A
	// Transform element is itself in the XML-Signature namespace, so
	// parseReferenceElement must honor only ds:Transform elements; matching on
	// local name alone would let an <evil:Transform Algorithm="...enveloped...">
	// drive a privileged transform.
	//
	// The attack: an enveloped signature relies on the enveloped-signature
	// Transform to detach the Signature element before digesting. If a foreign
	// <evil:Transform> carrying the enveloped-signature algorithm is honored, the
	// Signature is detached and the recomputed digest matches. Once the guard
	// ignores the foreign transform, the Signature element is no longer detached,
	// so the canonical bytes include it and the digest no longer matches — the
	// forgery is rejected. Full key control is assumed (worst case): recompute a
	// valid SignatureValue over the mutated SignedInfo.
	t.Run("foreign namespace rejected", func(t *testing.T) {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		doc, err := helium.NewParser().Parse(context.Background(), []byte(`<root><data>secret</data></root>`))
		require.NoError(t, err)

		signer := NewSigner().
			SignatureAlgorithm(AlgRSASHA256).
			Reference(ReferenceConfig{
				URI:             "",
				DigestAlgorithm: DigestSHA256,
				Transforms:      []Transform{Enveloped(), ExcC14NTransform()},
			})
		require.NoError(t, signer.SignEnveloped(context.Background(), doc, doc.DocumentElement(), key))

		sigElem := findChild(t, doc.DocumentElement(), "Signature")
		signedInfo := findChild(t, sigElem, "SignedInfo")

		// Move the enveloped-signature Transform into a foreign namespace so that,
		// by local name alone, it still looks like a Transform while it is no
		// longer a genuine ds:Transform.
		envTransform := findTransformByAlgorithm(t, signedInfo, TransformEnvelopedSignature)
		const evilNS = "urn:example:evil"
		require.NoError(t, envTransform.DeclareNamespace("evil", evilNS))
		require.NoError(t, envTransform.SetActiveNamespace("evil", evilNS))
		require.Equal(t, evilNS, elementNamespaceURI(envTransform))

		// Recompute a valid SignatureValue over the mutated SignedInfo so the only
		// thing standing between this document and a false "verified" result is the
		// namespace check on the Transform element.
		canonical, err := canonicalizeSubtree(ExcC14N10, signedInfo, nil)
		require.NoError(t, err)
		sigBytes, err := signBytes(AlgRSASHA256, key, canonical, false)
		require.NoError(t, err)

		sigValueElem := findChild(t, sigElem, "SignatureValue")
		for c := sigValueElem.FirstChild(); c != nil; c = sigValueElem.FirstChild() {
			mc, ok := c.(helium.MutableNode)
			require.True(t, ok)
			helium.UnlinkNode(mc)
		}
		require.NoError(t, sigValueElem.AddChild(
			doc.CreateText([]byte(base64.StdEncoding.EncodeToString(sigBytes)))))

		verifier := NewVerifier(StaticKey(&key.PublicKey))
		_, err = verifier.Verify(context.Background(), doc)
		require.Error(t, err, "signature whose enveloped Transform is in a foreign namespace must be rejected")
		require.ErrorIs(t, err, ErrDigestMismatch)
	})

	// genuine transforms still verify is the positive control for the Transform
	// namespace guard. A genuine enveloped + Exclusive C14N signature, including an
	// InclusiveNamespaces child (which lives in the xml-exc-c14n namespace, NOT the
	// XML-Signature namespace), must continue to verify — proving the guard rejects
	// only foreign-namespace Transform elements and does not reach the exc-c14n
	// InclusiveNamespaces child.
	t.Run("genuine transforms still verify", func(t *testing.T) {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		doc, err := helium.NewParser().Parse(context.Background(),
			[]byte(`<root xmlns:p="urn:example:p"><p:data>secret</p:data></root>`))
		require.NoError(t, err)

		signer := NewSigner().
			SignatureAlgorithm(AlgRSASHA256).
			Reference(ReferenceConfig{
				URI:             "",
				DigestAlgorithm: DigestSHA256,
				// ExcC14NTransform with a prefix list emits an InclusiveNamespaces
				// child in the xml-exc-c14n namespace, exercising the exc-c14n
				// child path the guard must leave untouched.
				Transforms: []Transform{Enveloped(), ExcC14NTransform("p")},
			})
		require.NoError(t, signer.SignEnveloped(context.Background(), doc, doc.DocumentElement(), key))

		// Confirm the InclusiveNamespaces child really is in the exc-c14n namespace,
		// not the DSig namespace, so the positive control is meaningful.
		sigElem := findChild(t, doc.DocumentElement(), "Signature")
		signedInfo := findChild(t, sigElem, "SignedInfo")
		excTransform := findTransformByAlgorithm(t, signedInfo, ExcC14N10)
		incNS := findChild(t, excTransform, "InclusiveNamespaces")
		require.Equal(t, "http://www.w3.org/2001/10/xml-exc-c14n#", elementNamespaceURI(incNS))
		require.False(t, isDSigCoreNS(incNS), "InclusiveNamespaces must not be in the DSig namespace")

		verifier := NewVerifier(StaticKey(&key.PublicKey))
		_, err = verifier.Verify(context.Background(), doc)
		require.NoError(t, err, "a genuine enveloped + exc-c14n signature with InclusiveNamespaces must still verify")
	})
}

// TestVerifyReferenceRejectsTransform guards against signature-coverage
// fail-open for references declaring transforms the verifier cannot apply.
func TestVerifyReferenceRejectsTransform(t *testing.T) {
	// unsupported transform guards against signature-coverage fail-open: a
	// Reference that declares a transform the verifier cannot apply must be
	// rejected before digesting, rather than silently ignored and verified against
	// the untransformed canonical bytes.
	//
	// This exercises verifyReference directly because SignedInfo (which contains
	// the Transforms list) is itself protected by the signature value, so the
	// unsupported transform must be caught at the per-reference stage.
	t.Run("unsupported transform", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><data>hello</data></root>`))
		require.NoError(t, err)

		// A whole-document reference whose only transform is an unsupported URI.
		// digestValue is irrelevant: rejection must happen before the digest is
		// even computed.
		ref := parsedReference{
			uri:             "",
			digestAlgorithm: DigestSHA256,
			transforms: []parsedTransform{
				{algorithm: "urn:bogus:transform"},
			},
		}

		_, _, err = verifyReference(t.Context(), &verifierConfig{}, doc, nil, ref)
		require.ErrorIs(t, err, ErrUnsupportedTransform)
	})

	// unsupported transform with enveloped ensures the enveloped detach/restore
	// path also rejects an unsupported sibling transform (and restores the
	// Signature element rather than leaving it detached).
	t.Run("unsupported transform with enveloped", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(),
			[]byte(`<root><ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#"/></root>`))
		require.NoError(t, err)

		root := doc.DocumentElement()
		sigElem, ok := helium.AsNode[*helium.Element](root.FirstChild())
		require.True(t, ok)

		ref := parsedReference{
			uri:             "",
			digestAlgorithm: DigestSHA256,
			transforms: []parsedTransform{
				{algorithm: TransformEnvelopedSignature},
				{algorithm: "urn:bogus:transform"},
			},
		}

		_, _, err = verifyReference(t.Context(), &verifierConfig{}, doc, sigElem, ref)
		require.ErrorIs(t, err, ErrUnsupportedTransform)

		// The Signature element must have been reattached, not left detached.
		require.Same(t, sigElem, root.FirstChild(), "signature element must be restored after rejection")
	})
}

func TestFindElementsByID(t *testing.T) {
	// findElementsByID recognizes the "id" attribute token in the casings
	// Id/ID/id plus xml:id and DTD/schema-declared ID typing. Distinct
	// convention tokens (e.g. wsu:Id) are NOT recognized by name.
	const wantID = "foo"
	testcases := []struct {
		name  string
		xml   string
		id    string
		count int // number of matching elements expected
	}{
		{
			name:  "capitalized Id",
			xml:   `<root><target Id="foo"/></root>`,
			id:    wantID,
			count: 1,
		},
		{
			name:  "uppercase ID",
			xml:   `<root><target ID="foo"/></root>`,
			id:    wantID,
			count: 1,
		},
		{
			name:  "lowercase id",
			xml:   `<root><target id="foo"/></root>`,
			id:    wantID,
			count: 1,
		},
		{
			name:  "xml:id",
			xml:   `<root><target xml:id="foo"/></root>`,
			id:    wantID,
			count: 1,
		},
		{
			name:  "no match",
			xml:   `<root><target id="bar"/></root>`,
			id:    wantID,
			count: 0,
		},
		{
			name:  "distinct token not recognized",
			xml:   `<root xmlns:wsu="http://x"><target wsu:Id="foo"/></root>`,
			id:    wantID,
			count: 0,
		},
		{
			name:  "duplicate id is ambiguous",
			xml:   `<root><a Id="foo"/><b id="foo"/></root>`,
			id:    wantID,
			count: 2,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.xml))
			require.NoError(t, err)

			matches := findElementsByIDUnder(doc.DocumentElement(), tc.id)
			require.Len(t, matches, tc.count)
		})
	}
}
