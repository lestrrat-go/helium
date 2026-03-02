package helium_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

func TestChildren(t *testing.T) {
	t.Run("direct children", func(t *testing.T) {
		doc, err := helium.Parse([]byte(`<root><a/><b/><c/></root>`))
		require.NoError(t, err)

		root := doc.DocumentElement()
		var names []string
		for child := range helium.Children(root) {
			names = append(names, child.Name())
		}
		require.Equal(t, []string{"a", "b", "c"}, names)
	})

	t.Run("empty", func(t *testing.T) {
		doc, err := helium.Parse([]byte(`<root/>`))
		require.NoError(t, err)

		var count int
		for range helium.Children(doc.DocumentElement()) {
			count++
		}
		require.Equal(t, 0, count)
	})

	t.Run("nil node", func(t *testing.T) {
		var count int
		for range helium.Children(nil) {
			count++
		}
		require.Equal(t, 0, count)
	})

	t.Run("break early", func(t *testing.T) {
		doc, err := helium.Parse([]byte(`<root><a/><b/><c/></root>`))
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
		doc, err := helium.Parse([]byte(`<root>text<a/><!--comment--></root>`))
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
	t.Run("flat children", func(t *testing.T) {
		doc, err := helium.Parse([]byte(`<root><a/><b/></root>`))
		require.NoError(t, err)

		var names []string
		for d := range helium.Descendants(doc.DocumentElement()) {
			names = append(names, d.Name())
		}
		require.Equal(t, []string{"a", "b"}, names)
	})

	t.Run("nested pre-order", func(t *testing.T) {
		doc, err := helium.Parse([]byte(`<root><a><b/></a><c/></root>`))
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
		doc, err := helium.Parse([]byte(`<root><a/></root>`))
		require.NoError(t, err)

		root := doc.DocumentElement()
		for d := range helium.Descendants(root) {
			require.NotEqual(t, root, d, "Descendants must not include the node itself")
		}
	})

	t.Run("nil node", func(t *testing.T) {
		var count int
		for range helium.Descendants(nil) {
			count++
		}
		require.Equal(t, 0, count)
	})

	t.Run("break early", func(t *testing.T) {
		doc, err := helium.Parse([]byte(`<root><a><b/></a><c/></root>`))
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
	t.Run("filters non-elements", func(t *testing.T) {
		doc, err := helium.Parse([]byte(`<root>text<a/><!--comment--><b/></root>`))
		require.NoError(t, err)

		var names []string
		for elem := range helium.ChildElements(doc.DocumentElement()) {
			names = append(names, elem.Name())
		}
		require.Equal(t, []string{"a", "b"}, names)
	})

	t.Run("empty when only text", func(t *testing.T) {
		doc, err := helium.Parse([]byte(`<root>hello</root>`))
		require.NoError(t, err)

		var count int
		for range helium.ChildElements(doc.DocumentElement()) {
			count++
		}
		require.Equal(t, 0, count)
	})

	t.Run("nil node", func(t *testing.T) {
		var count int
		for range helium.ChildElements(nil) {
			count++
		}
		require.Equal(t, 0, count)
	})
}
