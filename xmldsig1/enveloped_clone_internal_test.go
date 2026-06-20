package xmldsig1

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

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
