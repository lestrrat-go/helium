package xmldsig1

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/c14n"
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

		_, err = parseReferenceElement(context.Background(), testVerifyBudget(), ref)
		require.ErrorIs(t, err, ErrUnsupportedTransform)
		require.Contains(t, err.Error(), "Transform parameter")
	})

	// exc-c14n parsed is the positive control: an InclusiveNamespaces in the
	// exc-c14n namespace must still contribute its PrefixList.
	t.Run("exc-c14n parsed", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(context.Background(), []byte(`<root/>`))
		require.NoError(t, err)

		ref := buildExcC14NReference(t, doc, "ec", ExcC14N10, "a b c")

		parsed, err := parseReferenceElement(context.Background(), testVerifyBudget(), ref)
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
		out, err := canonicalizeSubtree(t.Context(), C14N10, target, nil)
		require.NoError(t, err)
		require.Contains(t, string(out), `xmlns:p="urn:p"`,
			"canonical subtree must carry the in-scope xmlns:p declaration")
		// The prefixed element name and its namespace decl must coexist.
		require.True(t, strings.HasPrefix(string(out), `<p:target xmlns:p="urn:p"`),
			"got: %s", out)
	})

	t.Run("exclusive C14N 1.0", func(t *testing.T) {
		out, err := canonicalizeSubtree(t.Context(), ExcC14N10, target, nil)
		require.NoError(t, err)
		require.Contains(t, string(out), `xmlns:p="urn:p"`,
			"exclusive canonical subtree must carry the visibly-utilized xmlns:p declaration")
	})

	t.Run("C14N 1.1", func(t *testing.T) {
		out, err := canonicalizeSubtree(t.Context(), C14N11URI, target, nil)
		require.NoError(t, err)
		require.Contains(t, string(out), `xmlns:p="urn:p"`,
			"C14N 1.1 canonical subtree must carry the in-scope xmlns:p declaration")
	})
}

func TestExclusiveC14NEquivalentDocumentReferencesMatch(t *testing.T) {
	const src = `<r Id="r" xml:base="sub/" xml:lang="en" xml:space="preserve"><child/></r>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)

	cfg := &signerConfig{}
	reference := ReferenceConfig{
		DigestAlgorithm: DigestSHA256,
		Transforms:      []Transform{ExcC14NTransform()},
	}

	reference.URI = ""
	whole, err := signReferenceOctets(t.Context(), cfg, doc, nil, reference, nil)
	require.NoError(t, err)

	reference.URI = "#r"
	selected, err := signReferenceOctets(t.Context(), cfg, doc, nil, reference, nil)
	require.NoError(t, err)

	require.Equal(t, string(selected), string(whole))
	require.NotContains(t, string(whole), "xmlns:xml")

	wholeDigest, err := computeDigest(DigestSHA256, whole, false)
	require.NoError(t, err)
	selectedDigest, err := computeDigest(DigestSHA256, selected, false)
	require.NoError(t, err)
	require.Equal(t, selectedDigest, wholeDigest)
}

func TestResolveReferenceOverlappingRoots(t *testing.T) {
	doc := mustParse(t, `<root><branch><target Id=" target "/></branch></root>`)
	root := doc.DocumentElement()
	branch := findLocal(root, "branch")
	target := findLocal(root, "target")
	require.NotNil(t, branch)
	require.NotNil(t, target)

	t.Run("document element repeated as extra root", func(t *testing.T) {
		got, err := resolveReference(doc, "#target", root)
		require.NoError(t, err)
		require.Same(t, target, got)
	})

	t.Run("nested extra root overlaps document", func(t *testing.T) {
		got, err := resolveReference(doc, "#target", branch)
		require.NoError(t, err)
		require.Same(t, target, got)
	})

	t.Run("repeated extra roots count once", func(t *testing.T) {
		got, err := resolveReference(doc, "#target", branch, branch, target)
		require.NoError(t, err)
		require.Same(t, target, got)
	})

	t.Run("detached root remains searchable", func(t *testing.T) {
		detached, err := doc.CreateElement("detached")
		require.NoError(t, err)
		require.NoError(t, detached.SetAttribute("Id", "detached"))

		got, err := resolveReference(doc, "#detached", detached)
		require.NoError(t, err)
		require.Same(t, detached, got)
	})

	t.Run("distinct matches across domains remain ambiguous", func(t *testing.T) {
		detached, err := doc.CreateElement("duplicate")
		require.NoError(t, err)
		require.NoError(t, detached.SetAttribute("Id", "target"))

		_, err = resolveReference(doc, "#target", detached)
		require.ErrorIs(t, err, ErrAmbiguousReference)
		require.Contains(t, err.Error(), "matched 2 elements")
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

	nodes, err := collectSubtreeNodes(t.Context(), target)
	require.NoError(t, err)
	set := make(map[helium.Node]bool)
	for _, n := range nodes {
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

	out, err := canonicalizeSubtree(t.Context(), C14N10, target, nil)
	require.NoError(t, err)
	require.Equal(t, `<target Id="x"><child>v</child></target>`, string(out))
}

// TestCanonicalizeEnvelopedMatchesDetach is the byte-equivalence contract for
// the enveloped transform: the canonical bytes produced by skipping the
// Signature subtree during canonicalization MUST equal the bytes produced by
// physically detaching the Signature from the live tree. This guarantees no
// digest/signature value changes for valid documents while the live DOM is
// never mutated. It also asserts the live tree is byte-for-byte unchanged after
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
				want, err = canonicalizeSubtree(t.Context(), tc.method, target, nil)
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

			got, err := canonicalizeEnveloped(t.Context(), tc.method, doc, target, sig, wholeDoc, nil)
			require.NoError(t, err)

			require.Equal(t, string(want), string(got), "enveloped bytes must match the detach-based reference")

			liveAfter, err := helium.WriteString(doc)
			require.NoError(t, err)
			require.Equal(t, liveBefore, liveAfter, "canonicalizeEnveloped must not mutate the live document")
		})
	}
}

// envelopedEquivalenceMethods are the canonicalization methods the enveloped
// equivalence test runs. The exclusive ones also run with
// envelopedEquivalencePrefixes as their InclusiveNamespaces PrefixList.
var envelopedEquivalenceMethods = []string{C14N10, C14N10Comments, ExcC14N10, ExcC14N10Comments, C14N11URI, C14N11Comments}

var envelopedEquivalencePrefixes = []string{"#default", "a", "b", "ds", "x", "foo"}

// envelopedEquivalenceSynthetic are hand-written inputs for the shapes the
// testdata corpus covers thinly: a Signature deep in the tree under comments,
// entities, xmlns="" and inherited xml:base/xml:lang/xml:id; a Signature as the
// document element with top-level nodes on both sides; nested and sibling
// Signatures; a Signature inside an unexpanded entity; and relative namespace
// URIs inside and outside the Signature.
var envelopedEquivalenceSynthetic = []struct {
	name string
	xml  string
}{
	{
		name: "deep-signature",
		xml: `<?xml version="1.0"?>
<!DOCTYPE r [<!ENTITY e "<x:q xmlns:x='urn:x'>ent</x:q>"><!ENTITY t "text&#38;#38;more">]>
<?lead pi?>
<!-- top comment -->
<r xmlns="urn:d" xmlns:a="urn:a" xml:lang="en" xml:base="http://ex.com/b/" xml:space="preserve" xml:id="r1">
  <a:p xmlns:b="urn:b" xml:base="c/"><!-- c1 --><b:q a:at="1">t&amp;&lt;&t;</b:q>
    <w xmlns="" xml:lang="fr">
      <ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#" Id="S" xml:base="sig/"><ds:SignedInfo xmlns:z="urn:z"><!-- in sig --><ds:Reference URI=""/></ds:SignedInfo><ds:Object><a:o xmlns="urn:o">obj<inner xml:lang="de"/></a:o></ds:Object></ds:Signature>
      <after b:x="y">&e;</after>
    </w>
  </a:p>
  <tail><?pi data?></tail>
</r>
<?trailing pi?>
<!-- trailing comment -->`,
	},
	{
		name: "signature-document-element",
		xml:  `<?lead?><!--c--><ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#" xml:lang="en"><ds:SignedInfo/><ds:Object><o/></ds:Object></ds:Signature><?trail?><!--t-->`,
	},
	{
		name: "nested-signatures",
		xml:  `<r xmlns:ds="http://www.w3.org/2000/09/xmldsig#"><ds:Signature Id="outer"><ds:SignedInfo/><ds:Object><inner><ds:Signature Id="in"><ds:SignedInfo/></ds:Signature></inner></ds:Object></ds:Signature><s/>text<ds:Signature/>more</r>`,
	},
	{
		name: "signature-in-entity",
		xml:  `<!DOCTYPE r [<!ENTITY s "<ds:Signature xmlns:ds='http://www.w3.org/2000/09/xmldsig#'><ds:SignedInfo/></ds:Signature>">]><r><a/>&s;<b>&s;</b></r>`,
	},
	{
		name: "relative-namespace-inside-signature",
		xml:  `<r><ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#" xmlns:rel="rel/uri"><ds:SignedInfo/></ds:Signature><s><t/></s></r>`,
	},
	{
		name: "relative-namespace-outside-signature",
		xml:  `<r><x xmlns:rel="rel/uri"/><s><t/></s><ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#"><ds:SignedInfo/></ds:Signature></r>`,
	},
}

// TestCanonicalizeEnvelopedMatchesCopyAndUnlink checks canonicalizeEnveloped,
// which skips the Signature subtree while canonicalizing the live document,
// against a reference that deep-copies the document, unlinks the copied
// Signature and canonicalizes the copy. The inputs are the c14n golden inputs,
// the xmldsig1 testdata and envelopedEquivalenceSynthetic, each parsed with
// entities kept and with entities substituted. Every method runs with and
// without an InclusiveNamespaces PrefixList, for a whole-document reference and
// for a spread of #id targets, including the Signature, its parent and an
// element inside it. Bytes and failure must both match.
func TestCanonicalizeEnvelopedMatchesCopyAndUnlink(t *testing.T) {
	t.Parallel()

	paths, err := filepath.Glob("../testdata/libxml2-compat/c14n/*/test/*.xml")
	require.NoError(t, err)
	require.NotEmpty(t, paths)
	more, err := filepath.Glob("testdata/*.xml")
	require.NoError(t, err)
	paths = append(paths, more...)
	more, err = filepath.Glob("testdata/*/*.xml")
	require.NoError(t, err)
	paths = append(paths, more...)

	for _, path := range paths {
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		t.Run(strings.TrimPrefix(filepath.ToSlash(path), "../"), func(t *testing.T) {
			t.Parallel()
			checkEnvelopedEquivalence(t, path, data)
		})
	}
	for _, tc := range envelopedEquivalenceSynthetic {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			checkEnvelopedEquivalence(t, tc.name, []byte(tc.xml))
		})
	}
	for _, tc := range envelopedEquivalenceBuilt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			compareEnvelopedDocument(t, buildActiveNamespaceDocument(t, tc.rootPrefix, tc.rootURI, tc.childPrefix, tc.childURI, tc.inSignature))
		})
	}
}

// envelopedEquivalenceBuilt are documents built through the DOM API whose
// child element gets an active namespace without a declaration. A parser never
// produces this shape: when the prefix is already bound to another URI, c14n
// and helium.CopyDoc disagree on what the element declares, and
// canonicalizeEnveloped takes the copy path.
var envelopedEquivalenceBuilt = []struct {
	name                  string
	rootPrefix, rootURI   string
	childPrefix, childURI string
	inSignature           bool
}{
	{name: "built-conflicting-default", rootPrefix: "", rootURI: "urn:p", childPrefix: "", childURI: "urn:c"},
	{name: "built-conflicting-prefixed", rootPrefix: "p", rootURI: "urn:1", childPrefix: "p", childURI: "urn:2"},
	{name: "built-unbound-prefix", rootPrefix: "p", rootURI: "urn:1", childPrefix: "q", childURI: "urn:q"},
	{name: "built-conflict-inside-signature", rootPrefix: "p", rootURI: "urn:1", childPrefix: "p", childURI: "urn:2", inSignature: true},
}

// buildActiveNamespaceDocument builds <r> declaring rootPrefix=rootURI with an
// enveloped ds:Signature and a <c> whose active namespace is
// childPrefix=childURI with no declaration. inSignature places <c> inside the
// Signature instead of beside it.
func buildActiveNamespaceDocument(t *testing.T, rootPrefix, rootURI, childPrefix, childURI string, inSignature bool) *helium.Document {
	t.Helper()
	doc := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
	root, err := doc.CreateElement("r")
	require.NoError(t, err)
	require.NoError(t, root.DeclareNamespace(rootPrefix, rootURI))
	require.NoError(t, doc.SetDocumentElement(root))

	sig, err := doc.CreateElement("Signature")
	require.NoError(t, err)
	require.NoError(t, sig.DeclareNamespace(nsPrefix, NamespaceDSig))
	require.NoError(t, sig.SetActiveNamespace(nsPrefix, NamespaceDSig))
	require.NoError(t, root.AddChild(sig))

	child, err := doc.CreateElement("c")
	require.NoError(t, err)
	require.NoError(t, child.SetActiveNamespace(childPrefix, childURI))
	require.NoError(t, child.AddChild(doc.CreateText([]byte("v"))))
	parent := root
	if inSignature {
		parent = sig
	}
	require.NoError(t, parent.AddChild(child))
	return doc
}

// checkEnvelopedEquivalence parses data with entities kept and with entities
// substituted, and compares canonicalizeEnveloped against the copy-and-unlink
// reference on every document that parses.
func checkEnvelopedEquivalence(t *testing.T, baseURI string, data []byte) {
	t.Helper()
	parsers := []helium.Parser{
		helium.NewParser().BaseURI(baseURI),
		helium.NewParser().BlockXXE(false).SubstituteEntities(true).LoadExternalDTD(true).DefaultDTDAttributes(true).BaseURI(baseURI).FS(helium.PermissiveFS()),
	}
	parsed := 0
	for _, p := range parsers {
		doc, err := p.Parse(t.Context(), data)
		if err != nil {
			continue
		}
		parsed++
		before, err := helium.WriteString(doc)
		require.NoError(t, err)
		compareEnvelopedDocument(t, doc)
		after, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Equal(t, before, after, "canonicalizeEnveloped must not mutate the live document")
	}
	require.NotZero(t, parsed, "no parser accepted the input")
}

// compareEnvelopedDocument runs the comparison for every Signature in doc, or,
// in a document without one, for a few elements spread across it standing in
// for the Signature.
func compareEnvelopedDocument(t *testing.T, doc *helium.Document) {
	t.Helper()
	var elems []*helium.Element
	collectEquivalenceElements(doc, &elems)
	if len(elems) == 0 {
		return
	}
	var sigs []*helium.Element
	for _, e := range elems {
		if e.LocalName() == "Signature" {
			sigs = append(sigs, e)
		}
	}
	if len(sigs) == 0 {
		sigs = spreadElements(elems, 4)
	}
	for _, sig := range sigs {
		targets := spreadElements(elems, 12)
		targets = append(targets, sig)
		if parent, ok := helium.AsNode[*helium.Element](sig.Parent()); ok {
			targets = append(targets, parent)
		}
		if inner := firstChildElement(sig); inner != nil {
			targets = append(targets, inner)
		}
		compareEnvelopedSignature(t, doc, sig, targets)
	}
}

// compareEnvelopedSignature builds the copy-and-unlink reference for sig once
// and compares every method, PrefixList and target against it.
func compareEnvelopedSignature(t *testing.T, doc *helium.Document, sig *helium.Element, targets []*helium.Element) {
	t.Helper()
	ctx := t.Context()
	clone, cloneTargets := copyAndUnlinkSignature(t, doc, sig, targets)
	defer clone.Free()

	for _, method := range envelopedEquivalenceMethods {
		prefixLists := [][]string{nil}
		if mode, _, _ := resolveC14NMode(method); mode == c14n.ExclusiveC14N10 {
			prefixLists = append(prefixLists, envelopedEquivalencePrefixes)
		}
		for _, prefixes := range prefixLists {
			want, wantErr := referenceEnvelopedBytes(ctx, method, clone, nil, prefixes)
			got, gotErr := canonicalizeEnveloped(ctx, method, doc, nil, sig, true, prefixes)
			requireSameCanonicalResultf(t, want, wantErr, got, gotErr, "whole document, method %s, prefixes %v", method, prefixes)

			for i, target := range targets {
				want, wantErr := referenceEnvelopedBytes(ctx, method, clone, cloneTargets[i], prefixes)
				got, gotErr := canonicalizeEnveloped(ctx, method, doc, target, sig, false, prefixes)
				requireSameCanonicalResultf(t, want, wantErr, got, gotErr, "target %s, method %s, prefixes %v", target.Name(), method, prefixes)
			}
		}
	}
}

func requireSameCanonicalResultf(t *testing.T, want []byte, wantErr error, got []byte, gotErr error, format string, args ...any) {
	t.Helper()
	msg := fmt.Sprintf(format, args...)
	require.Equal(t, wantErr != nil, gotErr != nil, "%s: reference error %v, got error %v", msg, wantErr, gotErr)
	require.Equal(t, string(want), string(got), msg)
}

// copyAndUnlinkSignature deep-copies doc, finds the copies of sig and of each
// target by their child-index paths, and unlinks the copied Signature. The
// paths are resolved before the unlink, which would shift the indexes of the
// Signature's following siblings.
func copyAndUnlinkSignature(t *testing.T, doc *helium.Document, sig *helium.Element, targets []*helium.Element) (*helium.Document, []*helium.Element) {
	t.Helper()
	clone, err := helium.CopyDoc(doc)
	require.NoError(t, err)

	cloneSig, ok := helium.AsNode[*helium.Element](nodeAtChildIndexPath(clone, childIndexPathOf(sig)))
	require.True(t, ok, "Signature copy not found")
	cloneTargets := make([]*helium.Element, len(targets))
	for i, target := range targets {
		cloneTargets[i], ok = helium.AsNode[*helium.Element](nodeAtChildIndexPath(clone, childIndexPathOf(target)))
		require.True(t, ok, "target copy not found")
	}
	helium.UnlinkNode(cloneSig)
	return clone, cloneTargets
}

// referenceEnvelopedBytes canonicalizes the copy with the Signature unlinked:
// the whole document when target is nil, else target's node set.
func referenceEnvelopedBytes(ctx context.Context, method string, clone *helium.Document, target *helium.Element, prefixes []string) ([]byte, error) {
	mode, comments, err := resolveC14NMode(method)
	if err != nil {
		return nil, err
	}
	canon := c14n.NewCanonicalizer(mode)
	if comments {
		canon = canon.Comments()
	}
	if mode == c14n.ExclusiveC14N10 && len(prefixes) > 0 {
		canon = canon.InclusiveNamespaces(prefixes)
	}
	if target == nil {
		return canon.CanonicalizeTo(clone)
	}
	nodes, err := collectCanonicalizationNodes(ctx, target, mode)
	if err != nil {
		return nil, err
	}
	return canon.NodeSet(nodes).CanonicalizeTo(clone)
}

// collectEquivalenceElements appends every element under n in document order,
// including the replacement content of entity declarations in the DTD.
func collectEquivalenceElements(n helium.Node, out *[]*helium.Element) {
	for c := range helium.Children(n) {
		if e, ok := helium.AsNode[*helium.Element](c); ok {
			*out = append(*out, e)
		}
		collectEquivalenceElements(c, out)
	}
}

// spreadElements returns up to n elements evenly spaced across elems, first
// one included.
func spreadElements(elems []*helium.Element, n int) []*helium.Element {
	if len(elems) <= n {
		return slices.Clone(elems)
	}
	out := make([]*helium.Element, 0, n)
	for i := range n {
		out = append(out, elems[i*len(elems)/n])
	}
	return out
}

func firstChildElement(e *helium.Element) *helium.Element {
	for c := range helium.Children(e) {
		if child, ok := helium.AsNode[*helium.Element](c); ok {
			return child
		}
	}
	return nil
}

// childIndexPathOf returns the child indexes that lead from n's document down
// to n.
func childIndexPathOf(n helium.Node) []int {
	var rev []int
	for cur := n; cur.Type() != helium.DocumentNode; cur = cur.Parent() {
		idx := 0
		for c := cur.Parent().FirstChild(); c != cur; c = c.NextSibling() {
			idx++
		}
		rev = append(rev, idx)
	}
	slices.Reverse(rev)
	return rev
}

// nodeAtChildIndexPath follows a childIndexPathOf path down from doc.
func nodeAtChildIndexPath(doc *helium.Document, path []int) helium.Node {
	var cur helium.Node = doc
	for _, idx := range path {
		cur = cur.FirstChild()
		for range idx {
			cur = cur.NextSibling()
		}
	}
	return cur
}

// TestCanonicalizeEnvelopedDetachedInputs covers the two inputs that are not
// attached to the document: a detached Signature omits nothing, and a detached
// #id target is an error.
func TestCanonicalizeEnvelopedDetachedInputs(t *testing.T) {
	t.Parallel()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<r xmlns:a="urn:a"><a:x>1</a:x></r>`))
	require.NoError(t, err)
	sig, err := doc.CreateElement("Signature")
	require.NoError(t, err)

	got, err := canonicalizeEnveloped(t.Context(), ExcC14N10, doc, nil, sig, true, nil)
	require.NoError(t, err)
	require.Equal(t, `<r><a:x xmlns:a="urn:a">1</a:x></r>`, string(got))

	_, err = canonicalizeEnveloped(t.Context(), ExcC14N10, doc, sig, sig, false, nil)
	require.Error(t, err)
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
		canonical, err := canonicalizeSubtree(t.Context(), ExcC14N10, signedInfo, nil)
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
	// rejected before digesting, and never silently ignored and verified against
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
	// Signature element, leaving it attached).
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

// xpathTransformStub is a Transform whose URI is the XPath filter transform.
// There is no exported constructor because the signing API cannot carry or emit
// the required ds:XPath child.
type xpathTransformStub struct{}

func (xpathTransformStub) URI() string { return TransformXPath }

// TestXPathSignPreflightRejected proves the shared capability validator rejects
// a sign-side XPath step before DOM mutation because processReference cannot emit
// its required <XPath> child.
func TestXPathSignPreflightRejected(t *testing.T) {
	cfg := &signerConfig{
		references: []ReferenceConfig{{
			URI:             objectFragment,
			DigestAlgorithm: DigestSHA1,
			Transforms:      []Transform{xpathTransformStub{}},
		}},
	}
	err := preflightSignerTransforms(cfg)
	require.ErrorIs(t, err, ErrUnsupportedTransform)
	var refErr *ReferenceError
	require.ErrorAs(t, err, &refErr, "the failure must name the offending Reference")
	require.Equal(t, 0, refErr.Reference)
}
