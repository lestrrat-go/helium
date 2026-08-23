package helium

import (
	"bytes"
	"testing"

	"github.com/lestrrat-go/helium/enum"
	"github.com/stretchr/testify/require"
)

// A DTD Name's storage split does not impose QName rules on its serialized
// spelling: element, attribute-declaration and NOTATION-enumeration names keep
// the XML Name the caller supplied, while a malformed in-memory notation node is
// still refused at the writer's defensive NCName boundary.
func TestWriterDTDNames(t *testing.T) {
	t.Run("attribute declaration names keep their colons", func(t *testing.T) {
		for _, name := range []string{"a:", ":a", ":"} {
			t.Run(name, func(t *testing.T) {
				doc := NewDocument("1.0", "UTF-8", StandaloneExplicitNo)
				dtd, err := doc.CreateInternalSubset(attrDeclElem, "", "")
				require.NoError(t, err)
				_, err = dtd.AddAttributeDecl(attrDeclElem, name, enum.AttrCDATA, enum.AttrDefaultImplied, "", nil)
				require.NoError(t, err)
				root, err := doc.CreateElement(attrDeclElem)
				require.NoError(t, err)
				require.NoError(t, doc.AddChild(root))

				var buf bytes.Buffer
				require.NoError(t, Write(&buf, doc))
				require.Contains(t, buf.String(), "<!ATTLIST item "+name+" CDATA #IMPLIED>")
			})
		}
	})

	// DTD element and content-model names retain the XML Name spelling supplied by
	// the caller.
	t.Run("element declaration names keep their colons", func(t *testing.T) {
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
	})

	// NOTATION lists use the full DTD Name grammar, not an NCName/QName grammar.
	t.Run("notation enumeration accepts colons", func(t *testing.T) {
		doc := NewDocument("1.0", "UTF-8", StandaloneExplicitNo)
		dtd, err := doc.CreateInternalSubset(attrDeclElem, "", "")
		require.NoError(t, err)
		_, err = dtd.AddAttributeDecl(attrDeclElem, "note", enum.AttrNotation, enum.AttrDefaultImplied, "", Enumeration{"x:a", ":", "a:"})
		require.NoError(t, err)
		root, err := doc.CreateElement(attrDeclElem)
		require.NoError(t, err)
		require.NoError(t, doc.AddChild(root))

		var buf bytes.Buffer
		require.NoError(t, Write(&buf, doc))
		require.Contains(t, buf.String(), "<!ATTLIST item note NOTATION (x:a | : | a:) #IMPLIED>")
		_, err = NewParser().Parse(t.Context(), buf.Bytes())
		require.NoError(t, err)
	})

	// The writer's defensive NCName boundary for a malformed in-memory notation
	// node.
	t.Run("notation declaration rejects colons", func(t *testing.T) {
		doc := NewDocument("1.0", "UTF-8", StandaloneExplicitNo)
		dtd, err := doc.CreateInternalSubset(attrDeclElem, "", "")
		require.NoError(t, err)
		notation, err := dtd.AddNotation("note", "", "note.sys")
		require.NoError(t, err)
		notation.name = "p:note"
		root, err := doc.CreateElement(attrDeclElem)
		require.NoError(t, err)
		require.NoError(t, doc.AddChild(root))

		var buf bytes.Buffer
		require.ErrorIs(t, Write(&buf, doc), ErrWriterInvalidName)
		require.NotContains(t, buf.String(), "<!NOTATION p:note")
	})
}

// Emission checks the exact grammar selected by the attribute type before any
// token is written.
func TestWriterAttributeEnumeration(t *testing.T) {
	tests := []struct {
		name    string
		atype   enum.AttributeType
		values  Enumeration
		replace bool
	}{
		{name: "ordinary-empty", atype: enum.AttrEnumeration},
		{name: "ordinary-empty-token", atype: enum.AttrEnumeration, values: Enumeration{""}},
		{name: "ordinary-whitespace", atype: enum.AttrEnumeration, values: Enumeration{"bad token"}},
		{name: "ordinary-pipe", atype: enum.AttrEnumeration, values: Enumeration{"bad|token"}},
		{name: "ordinary-duplicate", atype: enum.AttrEnumeration, values: Enumeration{"same", "same"}},
		{name: "ordinary-invalid-char-replace", atype: enum.AttrEnumeration, values: Enumeration{"bad\x01"}, replace: true},
		{name: "notation-empty", atype: enum.AttrNotation},
		{name: "notation-leading-digit", atype: enum.AttrNotation, values: Enumeration{"9bad"}},
		{name: "notation-whitespace", atype: enum.AttrNotation, values: Enumeration{"bad token"}},
		{name: "notation-pipe", atype: enum.AttrNotation, values: Enumeration{"bad|token"}},
		{name: "notation-duplicate", atype: enum.AttrNotation, values: Enumeration{"same", "same"}},
		{name: "notation-invalid-char-replace", atype: enum.AttrNotation, values: Enumeration{"bad\x01"}, replace: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc := NewDocument("1.0", "UTF-8", StandaloneExplicitNo)
			dtd, err := doc.CreateInternalSubset(attrDeclElem, "", "")
			require.NoError(t, err)
			_, err = dtd.AddAttributeDecl(attrDeclElem, "kind", tc.atype, enum.AttrDefaultImplied, "", tc.values)
			require.NoError(t, err)
			root, err := doc.CreateElement(attrDeclElem)
			require.NoError(t, err)
			require.NoError(t, doc.AddChild(root))

			writer := NewWriter()
			if tc.replace {
				writer = writer.RejectInvalidChars(false)
			}
			var buf bytes.Buffer
			require.ErrorIs(t, writer.WriteTo(&buf, doc), ErrWriterInvalidName)
			for _, value := range tc.values {
				if value == "" {
					continue
				}
				require.NotContains(t, buf.String(), value)
			}
		})
	}
}

// A text node carrying the xmlTextNoEnc marker is emitted verbatim with no
// escaping. These are white-box tests because the marker has no public setter
// (U2).
func TestWriteTextNoEnc(t *testing.T) {
	t.Parallel()

	// Non-ASCII marked content must fail with ErrUnsupportedOutputEncoding via
	// the ASCII-reject net, leaking no raw UTF-8 under the US-ASCII declaration.
	t.Run("US-ASCII rejects non-ASCII content", func(t *testing.T) {
		doc := NewDefaultDocument()
		root, err := doc.CreateElement("root")
		require.NoError(t, err)
		require.NoError(t, doc.SetDocumentElement(root))
		txt := doc.CreateText([]byte("café"))
		txt.name = xmlTextNoEnc // mark as pre-encoded, emitted without escaping
		require.NoError(t, root.AddChild(txt))

		var buf bytes.Buffer
		err = NewWriter().OutputEncoding("US-ASCII").WriteTo(&buf, doc)
		require.ErrorIs(t, err, ErrUnsupportedOutputEncoding)
		for i := range buf.Len() {
			require.Less(t, buf.Bytes()[i], byte(0x80), "leaked non-ASCII octet 0x%X at %d", buf.Bytes()[i], i)
		}
	})
}
