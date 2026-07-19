package helium

import (
	"bytes"
	"testing"

	"github.com/lestrrat-go/helium/enum"
	"github.com/stretchr/testify/require"
)

// TestAddElementDeclRejectsMalformedModel verifies that AddElementDecl rejects a
// structurally-incomplete content model (a sequence/choice node with nil
// children, as CreateElementContent alone produces) instead of storing it and
// letting serialization nil-dereference.
func TestAddElementDeclRejectsMalformedModel(t *testing.T) {
	doc := NewDocument("1.0", "UTF-8", StandaloneExplicitNo)
	dtd, err := doc.CreateInternalSubset("root", "", "")
	require.NoError(t, err)

	for _, etype := range []ElementContentType{ElementContentSeq, ElementContentOr} {
		content, err := doc.CreateElementContent("", etype)
		require.NoError(t, err)
		_, err = dtd.AddElementDecl("root", enum.ElementElementType, content)
		require.Error(t, err, "a seq/choice node with nil children must be rejected")

		// A rejected model must not have been stored, so serialization must not panic.
		var buf bytes.Buffer
		require.NotPanics(t, func() {
			_ = Write(&buf, doc)
		})
		require.NotContains(t, buf.String(), "<!ELEMENT root")
	}
}

// TestCreateElementContentSeqChoice verifies the safe composite constructors
// build a valid, serializable content model.
func TestCreateElementContentSeqChoice(t *testing.T) {
	doc := NewDocument("1.0", "UTF-8", StandaloneExplicitNo)
	dtd, err := doc.CreateInternalSubset("root", "", "")
	require.NoError(t, err)

	a, err := doc.CreateElementContent("a", ElementContentElement)
	require.NoError(t, err)
	b, err := doc.CreateElementContent("b", ElementContentElement)
	require.NoError(t, err)
	c, err := doc.CreateElementContent("c", ElementContentElement)
	require.NoError(t, err)

	// (b | c)
	choice, err := doc.CreateElementContentChoice(b, c, ElementContentOnce)
	require.NoError(t, err)
	// (a , (b | c)+)
	_, err = choice.SetOccurrence(ElementContentPlus)
	require.NoError(t, err)
	seq, err := doc.CreateElementContentSeq(a, choice, ElementContentOnce)
	require.NoError(t, err)

	_, err = dtd.AddElementDecl("root", enum.ElementElementType, seq)
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, Write(&buf, doc))
	require.Contains(t, buf.String(), "<!ELEMENT root (a , (b | c)+)>")
}

// TestCreateElementContentSeqRejectsNilChild verifies the composite constructors
// reject a nil or incomplete child.
func TestCreateElementContentSeqRejectsNilChild(t *testing.T) {
	doc := NewDocument("1.0", "UTF-8", StandaloneExplicitNo)
	a, err := doc.CreateElementContent("a", ElementContentElement)
	require.NoError(t, err)

	_, err = doc.CreateElementContentSeq(a, nil, ElementContentOnce)
	require.Error(t, err)

	_, err = doc.CreateElementContentChoice(nil, a, ElementContentOnce)
	require.Error(t, err)

	// An incomplete child (bare seq leaf with nil children) is also rejected.
	bad, err := doc.CreateElementContent("", ElementContentSeq)
	require.NoError(t, err)
	_, err = doc.CreateElementContentSeq(a, bad, ElementContentOnce)
	require.Error(t, err)
}

// TestRemoveElementUnlinks verifies RemoveElement drops the declaration from the
// serialized DTD (not just the lookup table) and returns it.
func TestRemoveElementUnlinks(t *testing.T) {
	doc := NewDocument("1.0", "UTF-8", StandaloneExplicitNo)
	dtd, err := doc.CreateInternalSubset("root", "", "")
	require.NoError(t, err)

	_, err = dtd.AddElementDecl("root", enum.EmptyElementType, nil)
	require.NoError(t, err)
	// A second decl keeps the elements table non-empty so the [...] block is
	// emitted; otherwise dumpDTD short-circuits and hides the leak.
	_, err = dtd.AddElementDecl("keep", enum.EmptyElementType, nil)
	require.NoError(t, err)

	removed := dtd.RemoveElement("root", "")
	require.NotNil(t, removed, "RemoveElement must return the removed declaration")
	require.Equal(t, "root", removed.name)

	_, ok := dtd.LookupElement("root", "")
	require.False(t, ok, "removed decl must be unmapped")

	var buf bytes.Buffer
	require.NoError(t, Write(&buf, doc))
	require.NotContains(t, buf.String(), "<!ELEMENT root", "removed decl must not be serialized")
	require.Contains(t, buf.String(), "<!ELEMENT keep EMPTY>")

	// Removing an absent key returns nil.
	require.Nil(t, dtd.RemoveElement("nope", ""))
}

// TestGetElementDescKey verifies that an element declaration registered via
// AddElementDecl can be retrieved through GetElementDesc using the same DTD Name.
// Registration splits the raw spelling for lookup; GetElementDesc must compose the
// same key instead of treating the DTD Name as a namespace-aware QName.
func TestGetElementDescKey(t *testing.T) {
	t.Run("unprefixed", func(t *testing.T) {
		dtd := newDTD()
		content, err := newElementContent("", ElementContentPCDATA)
		require.NoError(t, err)
		_, err = dtd.AddElementDecl("r", enum.MixedElementType, content)
		require.NoError(t, err)

		decl, ok := dtd.GetElementDesc("r")
		require.True(t, ok, "GetElementDesc must find the registered decl")
		require.Equal(t, enum.MixedElementType, decl.decltype)
	})
	t.Run("prefixed", func(t *testing.T) {
		dtd := newDTD()
		_, err := dtd.AddElementDecl("foo:bar", enum.EmptyElementType, nil)
		require.NoError(t, err)

		decl, ok := dtd.GetElementDesc("foo:bar")
		require.True(t, ok, "GetElementDesc must find the prefixed decl by QName")
		require.Equal(t, enum.EmptyElementType, decl.decltype)
	})
	t.Run("leading colon distinct from unprefixed", func(t *testing.T) {
		// A leading colon is NOT a prefix separator (libxml2 xmlSplitQName3): ":r"
		// is a distinct element name from the unprefixed "r" and must not be
		// reported as a redefinition of it (XML 1.0 5th-edition Name; eduni
		// ibm04v01).
		dtd := newDTD()
		_, err := dtd.AddElementDecl("r", enum.EmptyElementType, nil)
		require.NoError(t, err)
		_, err = dtd.AddElementDecl(":r", enum.AnyElementType, nil)
		require.NoError(t, err, "leading-colon name must not collide with the unprefixed name")

		decl, ok := dtd.GetElementDesc(":r")
		require.True(t, ok, "GetElementDesc must find the leading-colon decl")
		require.Equal(t, enum.AnyElementType, decl.decltype)

		decl, ok = dtd.GetElementDesc("r")
		require.True(t, ok, "GetElementDesc must still find the unprefixed decl")
		require.Equal(t, enum.EmptyElementType, decl.decltype)
	})
}

// TestWriterPreservesColonBearingDTDElementNames verifies that DTD element and
// content-model names retain the XML Name spelling supplied by the caller.
func TestWriterPreservesColonBearingDTDElementNames(t *testing.T) {
	for _, tc := range []struct {
		name string
	}{
		{name: "a:"},
		{name: ":r"},
		{name: ":"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc := NewDocument("1.0", "UTF-8", StandaloneExplicitNo)
			dtd, err := doc.CreateInternalSubset("root", "", "")
			require.NoError(t, err)
			_, err = dtd.AddElementDecl(tc.name, enum.AnyElementType, nil)
			require.NoError(t, err)
			root, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.AddChild(root))

			var buf bytes.Buffer
			require.NoError(t, Write(&buf, doc))
			require.Contains(t, buf.String(), "<!ELEMENT "+tc.name+" ANY>")
			_, err = NewParser().Parse(t.Context(), buf.Bytes())
			require.NoError(t, err)
		})
	}

	for _, tc := range []struct {
		name string
	}{
		{name: "a:"},
		{name: ":r"},
		{name: ":"},
	} {
		t.Run("content-"+tc.name, func(t *testing.T) {
			doc := NewDocument("1.0", "UTF-8", StandaloneExplicitNo)
			dtd, err := doc.CreateInternalSubset("root", "", "")
			require.NoError(t, err)
			pcdata, err := doc.CreateElementContent("", ElementContentPCDATA)
			require.NoError(t, err)
			leaf, err := doc.CreateElementContent(tc.name, ElementContentElement)
			require.NoError(t, err)
			model, err := doc.CreateElementContentChoice(pcdata, leaf, ElementContentMult)
			require.NoError(t, err)
			_, err = dtd.AddElementDecl("model", enum.MixedElementType, model)
			require.NoError(t, err)
			root, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.AddChild(root))

			var buf bytes.Buffer
			require.NoError(t, Write(&buf, doc))
			require.Contains(t, buf.String(), "#PCDATA | "+tc.name)
			_, err = NewParser().Parse(t.Context(), buf.Bytes())
			require.NoError(t, err)
		})
	}
}

// TestIsMixedElementWhitespace checks that Document.IsMixedElement reports an
// element declared with mixed (#PCDATA) content in the internal subset as mixed
// — the declared-content-model property the whitespace-significance classification
// depends on. The test does not parse whitespace itself; the end-to-end whitespace
// path (which consults elementDeclType across both subsets) is covered by
// TestStripBlanksExternalSubsetContentModel.
func TestIsMixedElementWhitespace(t *testing.T) {
	doc := NewDocument("1.0", "UTF-8", StandaloneExplicitNo)
	dtd := newDTD()
	dtd.doc = doc
	doc.intSubset = dtd

	content, err := newElementContent("", ElementContentPCDATA)
	require.NoError(t, err)
	_, err = dtd.AddElementDecl("r", enum.MixedElementType, content)
	require.NoError(t, err)

	mixed, err := doc.IsMixedElement("r")
	require.NoError(t, err)
	require.True(t, mixed, "mixed-content element must be reported as mixed")
}
