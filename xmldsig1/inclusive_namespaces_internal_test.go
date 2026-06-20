package xmldsig1

import (
	"context"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// buildExcC14NReference constructs a ds:Reference whose single Transform is
// Exclusive C14N carrying an InclusiveNamespaces child placed in namespace
// incNS (declared under prefix incPx) with the given PrefixList. The Reference
// (and its core children) live in the core XML-Signature namespace so that only
// the InclusiveNamespaces namespace varies between cases.
func buildExcC14NReference(t *testing.T, doc *helium.Document, incPx, incNS, prefixList string) *helium.Element {
	t.Helper()

	ref := doc.CreateElement("Reference")
	require.NoError(t, ref.DeclareNamespace(nsPrefix, NamespaceDSig))
	require.NoError(t, ref.SetActiveNamespace(nsPrefix, NamespaceDSig))

	transforms := doc.CreateElement("Transforms")
	require.NoError(t, transforms.SetActiveNamespace(nsPrefix, NamespaceDSig))
	require.NoError(t, ref.AddChild(transforms))

	transform := doc.CreateElement("Transform")
	require.NoError(t, transform.SetActiveNamespace(nsPrefix, NamespaceDSig))
	require.NoError(t, transform.SetLiteralAttribute("Algorithm", ExcC14N10))
	require.NoError(t, transforms.AddChild(transform))

	inc := doc.CreateElement("InclusiveNamespaces")
	require.NoError(t, inc.DeclareNamespace(incPx, incNS))
	require.NoError(t, inc.SetActiveNamespace(incPx, incNS))
	require.NoError(t, inc.SetLiteralAttribute("PrefixList", prefixList))
	require.NoError(t, transform.AddChild(inc))

	return ref
}

// TestInclusiveNamespacesForeignNamespaceIgnored guards against namespace
// confusion in the Exclusive C14N InclusiveNamespaces element. That element
// lives only in the exc-c14n namespace; matching on local name alone would let
// a foreign-namespace <evil:InclusiveNamespaces> inject a PrefixList and alter
// which namespaces are canonicalized. A foreign-namespace look-alike must
// therefore contribute no prefixes.
func TestInclusiveNamespacesForeignNamespaceIgnored(t *testing.T) {
	doc, err := helium.NewParser().Parse(context.Background(), []byte(`<root/>`))
	require.NoError(t, err)

	ref := buildExcC14NReference(t, doc, "evil", "urn:example:evil", "a b c")

	parsed, err := parseReferenceElement(ref)
	require.NoError(t, err)
	require.Len(t, parsed.transforms, 1)
	require.Empty(t, parsed.transforms[0].prefixes,
		"a foreign-namespace InclusiveNamespaces must contribute no prefixes")
}

// TestInclusiveNamespacesExcC14NParsed is the positive control: an
// InclusiveNamespaces in the exc-c14n namespace must still contribute its
// PrefixList.
func TestInclusiveNamespacesExcC14NParsed(t *testing.T) {
	doc, err := helium.NewParser().Parse(context.Background(), []byte(`<root/>`))
	require.NoError(t, err)

	ref := buildExcC14NReference(t, doc, "ec", ExcC14N10, "a b c")

	parsed, err := parseReferenceElement(ref)
	require.NoError(t, err)
	require.Len(t, parsed.transforms, 1)
	require.Equal(t, []string{"a", "b", "c"}, parsed.transforms[0].prefixes,
		"a correctly exc-c14n-namespaced InclusiveNamespaces must contribute its prefixes")
}
