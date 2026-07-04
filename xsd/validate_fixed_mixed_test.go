package xsd_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestFixedValueMixedContent verifies cvc-elt.5.2.2 for an element whose content
// type is MIXED and whose declaration carries a fixed value constraint:
//   - an empty element is clause 5.1 (the fixed value is assigned) and is valid;
//   - matching character content with no element children is valid (5.2.2.2.2);
//   - non-matching character content is rejected (5.2.2.2.2);
//   - the presence of element children is rejected regardless of the character
//     content, even when the direct text matches the fixed value (5.2.2.1).
//
// This is a string comparison of the initial value (direct character data,
// element descendants removed), not a typed value-space comparison. The rule is
// version-independent (it applies in XSD 1.0 and 1.1).
func TestFixedValueMixedContent(t *testing.T) {
	// A mixed complex type with an optional element child, declared on a global
	// element carrying fixed="abc".
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="CT" fixed="abc"/>
  <xs:complexType name="CT" mixed="true">
    <xs:sequence>
      <xs:element name="a" type="xs:byte" minOccurs="0"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`

	cases := []struct {
		name       string
		instance   string
		wantReject bool
	}{
		{name: "empty element assigns fixed", instance: `<root></root>`},
		{name: "matching character content", instance: `<root>abc</root>`},
		{name: "non-matching character content", instance: `<root>def</root>`, wantReject: true},
		{name: "element child before matching text", instance: `<root>abc<a>1</a></root>`, wantReject: true},
		{name: "element child after matching text", instance: `<root><a>1</a>abc</root>`, wantReject: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			runFixedValueCase(t, schemaXML, tc.instance, tc.wantReject)
		})
	}
}

// TestFixedValueMixedEntityRef verifies the mixed-content fixed check treats a
// direct internal entity reference as character content: its replacement text
// contributes to the ·initial value·. With helium's default parser, an internal
// entity reference reaches validation as a direct EntityRefNode (not expanded
// into text), so the initial-value computation must include its content —
// otherwise the element would be mistaken for clause-5.1 empty and wrongly
// accepted against a non-matching fixed value.
func TestFixedValueMixedEntityRef(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="CT" fixed="abc"/>
  <xs:complexType name="CT" mixed="true">
    <xs:sequence>
      <xs:element name="a" type="xs:byte" minOccurs="0"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`

	t.Run("entity ref expands to non-matching value", func(t *testing.T) {
		t.Parallel()
		const instance = `<!DOCTYPE root [ <!ENTITY e "def"> ]>
<root>&e;</root>`
		runFixedValueCase(t, schemaXML, instance, true)
	})
	t.Run("entity ref expands to matching value", func(t *testing.T) {
		t.Parallel()
		const instance = `<!DOCTYPE root [ <!ENTITY e "abc"> ]>
<root>&e;</root>`
		runFixedValueCase(t, schemaXML, instance, false)
	})
	// An entity whose replacement text contains an ELEMENT hits clause
	// 5.2.2.1 (element children are forbidden when a fixed value is present),
	// even though the reference itself reaches validation as character content.
	// The expansion (EntityRefNode -> EntityNode -> ElementNode) must be walked
	// to see the element.
	t.Run("entity ref expands to element markup", func(t *testing.T) {
		t.Parallel()
		const instance = `<!DOCTYPE root [ <!ENTITY e "<a>1</a>"> ]>
<root>&e;</root>`
		runFixedValueCase(t, schemaXML, instance, true)
	})
	// An entity expanding to mixed text+element markup is likewise rejected by
	// clause 5.2.2.1 for the element it contributes.
	t.Run("entity ref expands to mixed markup", func(t *testing.T) {
		t.Parallel()
		const instance = `<!DOCTYPE root [ <!ENTITY e "abc<a>1</a>"> ]>
<root>&e;</root>`
		runFixedValueCase(t, schemaXML, instance, true)
	})
}

// mustCompileFixedMixedSchema parses and compiles a schema, failing the test on
// any error.
func mustCompileFixedMixedSchema(t *testing.T, schemaXML string) *xsd.Schema {
	t.Helper()
	schemaDOC, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	schema, err := xsd.NewCompiler().Compile(t.Context(), schemaDOC)
	require.NoError(t, err)
	return schema
}

// TestFixedValueMixedEntityRefContentOnly verifies the mixed-content fixed check
// treats an entity whose replacement text is never materialized as a
// text/element subtree as character content. A DOM built via
// Document.CreateReference attaches the declared Entity node as the reference's
// child, but that Entity node itself has NO children (DTD.AddEntity stores the
// replacement text only as the entity's Content()), so the initial-value
// computation must fall back to the childless Entity node's Content();
// otherwise the element would be mistaken for clause-5.1 empty and wrongly
// accepted.
func TestFixedValueMixedEntityRefContentOnly(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="CT" fixed="abc"/>
  <xs:complexType name="CT" mixed="true">
    <xs:sequence>
      <xs:element name="a" type="xs:byte" minOccurs="0"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`

	schema := mustCompileFixedMixedSchema(t, schemaXML)

	build := func(t *testing.T, entityContent string) *helium.Document {
		t.Helper()
		doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
		dtd, err := doc.CreateInternalSubset("root", "", "")
		require.NoError(t, err)
		_, err = dtd.AddEntity("e", enum.InternalGeneralEntity, "", "", entityContent)
		require.NoError(t, err)
		root := doc.CreateElement("root")
		require.NoError(t, doc.SetDocumentElement(root))
		ref, err := doc.CreateReference("e")
		require.NoError(t, err)
		// Sanity: the replacement text lives only in Content() (the reference's
		// child is the childless Entity node — no text/element expansion).
		require.Equal(t, []byte(entityContent), ref.Content())
		require.NoError(t, root.AddChild(ref))
		return doc
	}

	t.Run("content-only entity expands to non-matching value", func(t *testing.T) {
		t.Parallel()
		doc := build(t, "def")
		var errs string
		err := validateWithOutput(t, xsd.NewValidator(schema), doc, &errs)
		require.Error(t, err)
		require.Contains(t, errs, "fixed value constraint")
	})
	t.Run("content-only entity expands to matching value", func(t *testing.T) {
		t.Parallel()
		doc := build(t, "abc")
		require.NoError(t, validateWithOutput(t, xsd.NewValidator(schema), doc, nil))
	})
}

// TestFixedValueMixedDuplicateEntityRef verifies that two references to the
// SAME entity within one expansion each contribute the replacement text to the
// initial value. The materialized expansion shares one Entity node across both
// references (Document.CreateReference links the declared Entity as each
// reference's child), so a naive walked-once cycle guard would count the shared
// node a single time and compute "x" instead of "xx"; the memoized scan must
// replay the cached contribution for the repeat reference.
func TestFixedValueMixedDuplicateEntityRef(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="CT" fixed="xx"/>
  <xs:complexType name="CT" mixed="true">
    <xs:sequence>
      <xs:element name="a" type="xs:byte" minOccurs="0"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`

	schema := mustCompileFixedMixedSchema(t, schemaXML)

	// build constructs <root>&outer;</root> where outer's materialized
	// expansion is two references to inner (replacement text innerContent).
	build := func(t *testing.T, innerContent string) *helium.Document {
		t.Helper()
		doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
		dtd, err := doc.CreateInternalSubset("root", "", "")
		require.NoError(t, err)
		_, err = dtd.AddEntity("inner", enum.InternalGeneralEntity, "", "", innerContent)
		require.NoError(t, err)
		outer, err := dtd.AddEntity("outer", enum.InternalGeneralEntity, "", "", "&inner;&inner;")
		require.NoError(t, err)

		refInner1, err := doc.CreateReference("inner")
		require.NoError(t, err)
		require.NoError(t, outer.AddChild(refInner1))
		refInner2, err := doc.CreateReference("inner")
		require.NoError(t, err)
		require.NoError(t, outer.AddChild(refInner2))

		root := doc.CreateElement("root")
		require.NoError(t, doc.SetDocumentElement(root))
		refOuter, err := doc.CreateReference("outer")
		require.NoError(t, err)
		require.NoError(t, root.AddChild(refOuter))
		return doc
	}

	t.Run("duplicate references concatenate to the fixed value", func(t *testing.T) {
		t.Parallel()
		doc := build(t, "x")
		require.NoError(t, validateWithOutput(t, xsd.NewValidator(schema), doc, nil))
	})
	t.Run("duplicate references concatenate to a non-matching value", func(t *testing.T) {
		t.Parallel()
		doc := build(t, "y")
		var errs string
		err := validateWithOutput(t, xsd.NewValidator(schema), doc, &errs)
		require.Error(t, err)
		require.Contains(t, errs, "fixed value constraint")
	})
}

// TestFixedValueMixedEntitySiblingIsolation verifies the scan attributes only
// the referenced entity's OWN expansion to the initial value, never the content
// of unrelated sibling entity declarations. A parsed entity reference's child is
// the shared Entity node, whose sibling pointers belong to the DTD's declaration
// list — so a naive NextSibling walk would spill the NEXT declared entity's
// replacement into the initial value and reject a matching document.
func TestFixedValueMixedEntitySiblingIsolation(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="CT" fixed="abc"/>
  <xs:complexType name="CT" mixed="true">
    <xs:sequence>
      <xs:element name="a" type="xs:byte" minOccurs="0"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`

	// &a; expands to "abc" (== fixed); entity b ("def") is a LATER sibling
	// declaration in the DTD and must not be pulled into the initial value.
	const instance = `<!DOCTYPE root [ <!ENTITY a "abc"> <!ENTITY b "def"> ]>
<root>&a;</root>`
	runFixedValueCase(t, schemaXML, instance, false)
}

// TestFixedValueMixedCommentOnlyEntity verifies that an entity whose materialized
// expansion is only a comment contributes no character content: the element is
// clause-5.1 empty (the fixed value is assigned) and valid. The entity carries a
// non-empty Content() (its raw replacement text) alongside the comment child, so
// the scan must recognize the materialized child subtree as authoritative rather
// than falling back to Content() and treating the comment markup as literal text.
func TestFixedValueMixedCommentOnlyEntity(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="CT" fixed="abc"/>
  <xs:complexType name="CT" mixed="true">
    <xs:sequence>
      <xs:element name="a" type="xs:byte" minOccurs="0"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`

	schema := mustCompileFixedMixedSchema(t, schemaXML)

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	dtd, err := doc.CreateInternalSubset("root", "", "")
	require.NoError(t, err)
	// Content() holds the raw replacement markup; the materialized expansion is a
	// comment child.
	ent, err := dtd.AddEntity("c", enum.InternalGeneralEntity, "", "", "<!-- x -->")
	require.NoError(t, err)
	require.NoError(t, ent.AddChild(doc.CreateComment([]byte(" x "))))

	root := doc.CreateElement("root")
	require.NoError(t, doc.SetDocumentElement(root))
	ref, err := doc.CreateReference("c")
	require.NoError(t, err)
	require.NoError(t, root.AddChild(ref))

	// The comment is neither character nor element content, so the element is
	// clause-5.1 empty and the fixed value is assigned — valid.
	require.NoError(t, validateWithOutput(t, xsd.NewValidator(schema), doc, nil))
}

// TestFixedValueMixedEmptyTextNode verifies that a present-but-empty character
// node (a zero-length text child) is NOT treated as clause-5.1 empty: it is
// character content, so it must match the fixed value. A zero-length text node
// is not produced by the parser, so the instance DOM is constructed directly.
func TestFixedValueMixedEmptyTextNode(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="CT" fixed="abc"/>
  <xs:complexType name="CT" mixed="true">
    <xs:sequence>
      <xs:element name="a" type="xs:byte" minOccurs="0"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`

	schemaDOC, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	schema, err := xsd.NewCompiler().Compile(t.Context(), schemaDOC)
	require.NoError(t, err)

	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root></root>`))
	require.NoError(t, err)
	root := doc.DocumentElement()
	require.NoError(t, root.AddChild(doc.CreateText([]byte{})))

	var errs string
	err = validateWithOutput(t, xsd.NewValidator(schema), doc, &errs)
	require.Error(t, err)
	require.Contains(t, errs, "fixed value constraint")
}

// TestFixedValueMixedCommentPIOnly verifies that an element whose only children
// are comments and/or processing instructions is clause-5.1 empty (comments and
// PIs are neither character nor element content), so the fixed value is assigned
// and the element is valid.
func TestFixedValueMixedCommentPIOnly(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="CT" fixed="abc"/>
  <xs:complexType name="CT" mixed="true">
    <xs:sequence>
      <xs:element name="a" type="xs:byte" minOccurs="0"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`

	t.Run("comment only", func(t *testing.T) {
		t.Parallel()
		runFixedValueCase(t, schemaXML, `<root><!-- c --></root>`, false)
	})
	t.Run("pi only", func(t *testing.T) {
		t.Parallel()
		runFixedValueCase(t, schemaXML, `<root><?pi data?></root>`, false)
	})
}

// TestFixedValueMixedAnyType verifies the mixed-content fixed check also fires
// when the governing type has no declared model group (an xs:anyType / empty
// mixed complex type): the initial value must equal the fixed value.
func TestFixedValueMixedAnyType(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" fixed="abc">
    <xs:complexType mixed="true"/>
  </xs:element>
</xs:schema>`

	t.Run("match", func(t *testing.T) {
		t.Parallel()
		runFixedValueCase(t, schemaXML, `<root>abc</root>`, false)
	})
	t.Run("mismatch", func(t *testing.T) {
		t.Parallel()
		runFixedValueCase(t, schemaXML, `<root>def</root>`, true)
	})
}
