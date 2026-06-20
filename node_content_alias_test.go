package helium_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestContentDefensiveCopy verifies that the exported Content() on the leaf
// node types (Text, Comment, CDATASection) returns a defensive copy of the
// node's internal bytes. Mutating the returned slice must NOT corrupt the DOM,
// and a subsequent read must still return the original content.
func TestContentDefensiveCopy(t *testing.T) {
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneExplicitNo)

	const original = "hello world"

	makers := map[string]func() helium.Node{
		"Text": func() helium.Node {
			n := doc.CreateText([]byte(original))
			return n
		},
		"Comment": func() helium.Node {
			n := doc.CreateComment([]byte(original))
			return n
		},
		"CDATASection": func() helium.Node {
			n := doc.CreateCDATASection([]byte(original))
			return n
		},
	}

	for name, make := range makers {
		t.Run(name, func(t *testing.T) {
			n := make()
			require.Equal(t, original, string(n.Content()), "initial content")

			// Mutating the returned slice must not affect the node.
			got := n.Content()
			require.Len(t, got, len(original))
			for i := range got {
				got[i] = 'X'
			}

			// Re-read must return the untouched original.
			require.Equal(t, original, string(n.Content()), "content after caller mutation")

			// Two separate Content() calls must not alias each other either.
			a := n.Content()
			b := n.Content()
			if len(a) > 0 {
				a[0] = 'Z'
				require.NotEqual(t, a[0], b[0], "second Content() call must not alias the first")
			}
		})
	}
}
