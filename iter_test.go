package helium_test

import (
	"strconv"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

func TestIterators(t *testing.T) {
	t.Parallel()

	t.Run("children", func(t *testing.T) {
		t.Run("direct children", func(t *testing.T) {
			t.Parallel()

			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><a/><b/><c/></root>`))
			require.NoError(t, err)

			root := doc.DocumentElement()
			var names []string
			for child := range helium.Children(root) {
				names = append(names, child.Name())
			}
			require.Equal(t, []string{"a", "b", "c"}, names)
		})

		t.Run("empty", func(t *testing.T) {
			t.Parallel()

			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root/>`))
			require.NoError(t, err)

			var count int
			for range helium.Children(doc.DocumentElement()) {
				count++
			}
			require.Equal(t, 0, count)
		})

		t.Run("nil node", func(t *testing.T) {
			t.Parallel()

			var count int
			for range helium.Children(nil) {
				count++
			}
			require.Equal(t, 0, count)
		})

		t.Run("break early", func(t *testing.T) {
			t.Parallel()

			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><a/><b/><c/></root>`))
			require.NoError(t, err)

			var names []string
			for child := range helium.Children(doc.DocumentElement()) {
				names = append(names, child.Name())
				if child.Name() == "a" {
					break
				}
			}
			require.Equal(t, []string{"a"}, names)
		})

		t.Run("mixed node types", func(t *testing.T) {
			t.Parallel()

			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root>text<a/><!--comment--></root>`))
			require.NoError(t, err)

			var types []helium.ElementType
			for child := range helium.Children(doc.DocumentElement()) {
				types = append(types, child.Type())
			}
			require.Equal(t, []helium.ElementType{
				helium.TextNode,
				helium.ElementNode,
				helium.CommentNode,
			}, types)
		})
	})

	t.Run("child elements", func(t *testing.T) {
		t.Run("filters non-elements", func(t *testing.T) {
			t.Parallel()

			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root>text<a/><!--comment--><b/></root>`))
			require.NoError(t, err)

			var names []string
			for elem := range helium.ChildElements(doc.DocumentElement()) {
				names = append(names, elem.Name())
			}
			require.Equal(t, []string{"a", "b"}, names)
		})

		t.Run("empty when only text", func(t *testing.T) {
			t.Parallel()

			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root>hello</root>`))
			require.NoError(t, err)

			var count int
			for range helium.ChildElements(doc.DocumentElement()) {
				count++
			}
			require.Equal(t, 0, count)
		})

		t.Run("nil node", func(t *testing.T) {
			t.Parallel()

			var count int
			for range helium.ChildElements(nil) {
				count++
			}
			require.Equal(t, 0, count)
		})
	})

	t.Run("descendants", func(t *testing.T) {
		t.Run("flat children", func(t *testing.T) {
			t.Parallel()

			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><a/><b/></root>`))
			require.NoError(t, err)

			var names []string
			for d := range helium.Descendants(doc.DocumentElement()) {
				names = append(names, d.Name())
			}
			require.Equal(t, []string{"a", "b"}, names)
		})

		t.Run("nested pre-order", func(t *testing.T) {
			t.Parallel()

			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><a><b/></a><c/></root>`))
			require.NoError(t, err)

			var names []string
			for d := range helium.Descendants(doc.DocumentElement()) {
				if d.Type() == helium.ElementNode {
					names = append(names, d.Name())
				}
			}
			require.Equal(t, []string{"a", "b", "c"}, names)
		})

		t.Run("excludes self", func(t *testing.T) {
			t.Parallel()

			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><a/></root>`))
			require.NoError(t, err)

			root := doc.DocumentElement()
			for d := range helium.Descendants(root) {
				require.NotEqual(t, root, d, "Descendants must not include the node itself")
			}
		})

		t.Run("nil node", func(t *testing.T) {
			t.Parallel()

			var count int
			for range helium.Descendants(nil) {
				count++
			}
			require.Equal(t, 0, count)
		})

		t.Run("break early", func(t *testing.T) {
			t.Parallel()

			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><a><b/></a><c/></root>`))
			require.NoError(t, err)

			var names []string
			for d := range helium.Descendants(doc.DocumentElement()) {
				if d.Type() == helium.ElementNode {
					names = append(names, d.Name())
					if d.Name() == "b" {
						break
					}
				}
			}
			require.Equal(t, []string{"a", "b"}, names)
		})
	})

	// A sibling list corrupted into a ring through the Unsafe* link setters
	// terminates every iterator instead of looping forever. The guard is Brent's
	// algorithm, which stops within a bounded multiple of the cycle length rather
	// than at the exact first repeat, so the assertion is termination and a
	// bounded yield count, NOT an exact node sequence.
	t.Run("cyclic sibling list terminates", func(t *testing.T) {
		t.Parallel()

		for _, ring := range []int{1, 2, 3, 5, 8, 17, 64} {
			t.Run(strconv.Itoa(ring), func(t *testing.T) {
				t.Parallel()

				doc := helium.NewDefaultDocument()
				parent, err := doc.CreateElement("parent")
				require.NoError(t, err)
				var kids []*helium.Element
				for i := range ring {
					c, err := doc.CreateElement("c" + strconv.Itoa(i))
					require.NoError(t, err)
					require.NoError(t, parent.AddChild(c))
					kids = append(kids, c)
				}
				// Close the ring: the last child's next pointer is the first child.
				helium.UnsafeSetNextSiblingForTesting(kids[ring-1], kids[0])

				// A guard that never fires would spin forever, so cap every loop
				// well above the bound and fail rather than hang.
				limit := 16 * (ring + 1)

				children := 0
				for range helium.Children(parent) {
					children++
					require.Less(t, children, limit, "Children must terminate on a cyclic sibling list")
				}
				require.GreaterOrEqual(t, children, ring, "Children must yield the whole list before truncating")

				elements := 0
				for range helium.ChildElements(parent) {
					elements++
					require.Less(t, elements, limit, "ChildElements must terminate on a cyclic sibling list")
				}
				require.GreaterOrEqual(t, elements, ring, "ChildElements must yield the whole list before truncating")

				descendants := 0
				for range helium.Descendants(parent) {
					descendants++
					require.Less(t, descendants, limit, "Descendants must terminate on a cyclic sibling list")
				}
				require.GreaterOrEqual(t, descendants, ring, "Descendants must yield the whole list before truncating")
			})
		}
	})

	// The iter.go helpers together, including the element-only filter and
	// early termination.
	t.Run("combined", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
		root, err := doc.CreateElement("root")
		require.NoError(t, err)
		require.NoError(t, doc.AddChild(root))

		require.NoError(t, root.AppendText([]byte("text")))
		e1, err := doc.CreateElement("a")
		require.NoError(t, err)
		require.NoError(t, root.AddChild(e1))
		require.NoError(t, root.AddChild(doc.CreateComment([]byte("c"))))
		e2, err := doc.CreateElement("b")
		require.NoError(t, err)
		require.NoError(t, root.AddChild(e2))

		// ChildElements skips text/comment.
		var names []string
		for el := range helium.ChildElements(root) {
			names = append(names, el.Name())
		}
		require.Equal(t, []string{"a", "b"}, names)

		// Early break from ChildElements.
		count := 0
		for range helium.ChildElements(root) {
			count++
			break
		}
		require.Equal(t, 1, count)

		// Children yields all child nodes.
		all := 0
		for range helium.Children(root) {
			all++
		}
		require.Equal(t, 4, all)

		// Children/ChildElements/Descendants of nil yield nothing.
		for range helium.Children(nil) {
			t.Fatal("nil Children should yield nothing")
		}
		for range helium.ChildElements(nil) {
			t.Fatal("nil ChildElements should yield nothing")
		}
		for range helium.Descendants(nil) {
			t.Fatal("nil Descendants should yield nothing")
		}

		// Descendants does a depth-first walk; early break is honored.
		dcount := 0
		for range helium.Descendants(root) {
			dcount++
			break
		}
		require.Equal(t, 1, dcount)
	})
}
