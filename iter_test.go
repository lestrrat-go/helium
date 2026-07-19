package helium_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

func TestChildren(t *testing.T) {
	t.Parallel()

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
}

func TestDescendants(t *testing.T) {
	t.Parallel()

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
}

func TestChildElements(t *testing.T) {
	t.Parallel()

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
}

// TestChildElementsAndIterators covers the iter.go helpers including the
// element-only filter and early termination.
func TestChildElementsAndIterators(t *testing.T) {
	t.Parallel()
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
}
