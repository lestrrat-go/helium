package node_test

import (
	"testing"

	"github.com/lestrrat-go/helium/node"
	"github.com/stretchr/testify/require"
)

func TestDocument(t *testing.T) {
	t.Run("NewDocument", func(t *testing.T) {
		doc := node.NewDocument()
		require.NotNil(t, doc)
		require.Equal(t, node.DocumentNodeType, doc.Type())
		require.Equal(t, "#document", doc.LocalName())
		require.Equal(t, "1.0", doc.Version())
		require.Equal(t, "utf-8", doc.Encoding())
	})

	t.Run("CreateElement", func(t *testing.T) {
		doc := node.NewDocument()
		elem := doc.CreateElement("test")
		require.NotNil(t, elem)
		require.Equal(t, "test", elem.LocalName())
		require.Equal(t, doc, elem.OwnerDocument())
	})

	t.Run("CreateText", func(t *testing.T) {
		doc := node.NewDocument()
		text := doc.CreateText([]byte("hello"))
		require.NotNil(t, text)
		require.Equal(t, node.TextNodeType, text.Type())
		require.Equal(t, doc, text.OwnerDocument())
	})

	t.Run("CreateComment", func(t *testing.T) {
		doc := node.NewDocument()
		comment := doc.CreateComment([]byte("test comment"))
		require.NotNil(t, comment)
		require.Equal(t, node.CommentNodeType, comment.Type())
		require.Equal(t, doc, comment.OwnerDocument())
	})

	t.Run("SetDocumentElement", func(t *testing.T) {
		doc := node.NewDocument()
		root := doc.CreateElement("root")

		err := doc.SetDocumentElement(root)
		require.NoError(t, err)
		require.Equal(t, root, doc.FirstChild())
		require.Equal(t, doc, root.Parent())
	})

}
