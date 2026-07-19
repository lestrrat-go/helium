package helium_test

import (
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// elem.AddChild(attr) must route the attribute into the element's property
// list, NOT the child list: it appears in Attributes()/GetAttribute, is absent
// from Children(), and serializes as an attribute rather than a child element.
func TestAddChildRoutesAttributeToProperties(t *testing.T) {
	doc := helium.NewDefaultDocument()
	elem := doc.CreateElement("root")

	attr, err := doc.CreateAttribute("orphan", "v", nil)
	require.NoError(t, err)

	require.NoError(t, elem.AddChild(attr))

	// Present as an attribute.
	got, ok := elem.GetAttribute("orphan")
	require.True(t, ok, "attribute must be reachable via GetAttribute")
	require.Equal(t, "v", got)

	attrs := elem.Attributes()
	require.Len(t, attrs, 1)
	require.Equal(t, "orphan", attrs[0].Name())

	// Absent from the child list.
	for child := range helium.Children(elem) {
		t.Fatalf("attribute must not appear in the child list, found %s", child.Type())
	}

	// Serializes as an attribute, not a child element.
	out, err := helium.WriteString(elem)
	require.NoError(t, err)
	require.Contains(t, out, `orphan="v"`)
	require.NotContains(t, out, "<orphan>")
}

// Routing an attribute through AddChild replaces an existing same-named
// attribute in place (libxml2 xmlAddChild parity via addProperty).
func TestAddChildAttributeReplacesSameName(t *testing.T) {
	doc := helium.NewDefaultDocument()
	elem := doc.CreateElement("root")

	err := elem.SetAttribute("id", "first")
	require.NoError(t, err)

	replacement, err := doc.CreateAttribute("id", "second", nil)
	require.NoError(t, err)
	require.NoError(t, elem.AddChild(replacement))

	got, ok := elem.GetAttribute("id")
	require.True(t, ok)
	require.Equal(t, "second", got)
	require.Len(t, elem.Attributes(), 1, "same-named attribute must be replaced, not duplicated")
}

// An attribute already parented on one element is detached from it before being
// spliced onto the new element.
func TestAddChildAttributeDetachesFromPreviousElement(t *testing.T) {
	doc := helium.NewDefaultDocument()
	src := doc.CreateElement("src")
	dst := doc.CreateElement("dst")

	err := src.SetAttribute("moved", "v")
	require.NoError(t, err)
	attr := src.Attributes()[0]

	require.NoError(t, dst.AddChild(attr))

	_, ok := src.GetAttribute("moved")
	require.False(t, ok, "attribute must be removed from its previous element")
	require.Empty(t, src.Attributes())

	got, ok := dst.GetAttribute("moved")
	require.True(t, ok)
	require.Equal(t, "v", got)
}

// A document accepts an element through AddChild (an element is a valid child of
// a document node), and the attribute-routing type switch must not block it.
func TestDocumentAddChildElement(t *testing.T) {
	doc := helium.NewDefaultDocument()
	root := doc.CreateElement("root")
	require.NoError(t, doc.AddChild(root))
	require.Equal(t, root, doc.DocumentElement())
}

// An attribute has no valid placement on a document and is rejected.
func TestDocumentAddChildRejectsAttribute(t *testing.T) {
	doc := helium.NewDefaultDocument()
	attr, err := doc.CreateAttribute("a", "v", nil)
	require.NoError(t, err)

	err = doc.AddChild(attr)
	require.Error(t, err)
	require.ErrorIs(t, err, helium.ErrInvalidOperation)
}

// An attribute has no valid placement on a non-element parent (Text) and is
// rejected with a descriptive %w-wrapped ErrInvalidOperation, not a bare
// sentinel.
func TestTextAddChildRejectsAttribute(t *testing.T) {
	doc := helium.NewDefaultDocument()
	text := doc.CreateText([]byte("hello"))
	attr, err := doc.CreateAttribute("a", "v", nil)
	require.NoError(t, err)

	err = text.AddChild(attr)
	require.Error(t, err)
	require.ErrorIs(t, err, helium.ErrInvalidOperation)
	// The message carries context beyond the bare sentinel text.
	require.NotEqual(t, helium.ErrInvalidOperation.Error(), err.Error())
	require.Contains(t, err.Error(), "cannot add")
}

// A Text node merges only another Text node; any other operand (here an Element)
// is rejected with a descriptive %w-wrapped ErrInvalidOperation.
func TestTextAddChildRejectsNonText(t *testing.T) {
	doc := helium.NewDefaultDocument()
	text := doc.CreateText([]byte("hello"))

	err := text.AddChild(doc.CreateElement("child"))
	require.Error(t, err)
	require.ErrorIs(t, err, helium.ErrInvalidOperation)
	require.NotEqual(t, helium.ErrInvalidOperation.Error(), err.Error())
	require.Contains(t, err.Error(), "cannot add")
}

// A Comment node merges only another Comment node; any other operand (here an
// Element) is rejected with a descriptive %w-wrapped ErrInvalidOperation.
func TestCommentAddChildRejectsNonComment(t *testing.T) {
	doc := helium.NewDefaultDocument()
	comment := doc.CreateComment([]byte("c"))

	err := comment.AddChild(doc.CreateElement("child"))
	require.Error(t, err)
	require.ErrorIs(t, err, helium.ErrInvalidOperation)
	require.NotEqual(t, helium.ErrInvalidOperation.Error(), err.Error())
	require.Contains(t, err.Error(), "cannot add")
}

// A CDATA section carries character data, not child nodes, so every operand is
// rejected with a descriptive %w-wrapped ErrInvalidOperation.
func TestCDATASectionAddChildRejectsChild(t *testing.T) {
	doc := helium.NewDefaultDocument()
	cdata := doc.CreateCDATASection([]byte("x"))

	err := cdata.AddChild(doc.CreateText([]byte("y")))
	require.Error(t, err)
	require.ErrorIs(t, err, helium.ErrInvalidOperation)
	require.NotEqual(t, helium.ErrInvalidOperation.Error(), err.Error())
	require.Contains(t, err.Error(), "cannot add")

	// A nil operand is rejected with ErrNilNode, not a panic.
	require.ErrorIs(t, cdata.AddChild(nil), helium.ErrNilNode)
}

// A ProcessingInstruction carries its content as a string, not as child nodes,
// so an attribute has no valid placement on it. Its AddChild override handles
// the operand itself (never reaching the shared addChild rejection), so it must
// reject an Attribute operand with a wrapped ErrInvalidOperation and leave the
// PI unchanged.
func TestPIAddChildRejectsAttribute(t *testing.T) {
	doc := helium.NewDefaultDocument()
	pi := doc.CreatePI("target", "data")

	attr, err := doc.CreateAttribute("a", "v", nil)
	require.NoError(t, err)

	err = pi.AddChild(attr)
	require.Error(t, err)
	require.ErrorIs(t, err, helium.ErrInvalidOperation)

	// The PI content is unchanged by the rejected operand.
	require.Equal(t, "data", string(pi.Content()))
}

// Regression guard: routing an attribute through AddChild must not leave a
// stray child element in the serialized output.
func TestAddChildAttributeSerializationShape(t *testing.T) {
	doc := helium.NewDefaultDocument()
	elem := doc.CreateElement("root")
	attr, err := doc.CreateAttribute("k", "v", nil)
	require.NoError(t, err)
	require.NoError(t, elem.AddChild(attr))

	out, err := helium.WriteString(elem)
	require.NoError(t, err)
	require.False(t, strings.Contains(out, "</root><"), "no sibling/child element must follow root")
	require.Contains(t, out, `<root k="v"`)
}
