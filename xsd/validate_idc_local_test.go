package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestIDCLocalKeyRefEvaluated is the regression test for the conformance gap
// where identity constraints declared on a LOCAL element declaration (buried
// inside a content model) were compile-checked but never EVALUATED at
// instance-validation time. Pass-2 resolved the host declaration via
// lookupElemDecl, which only finds GLOBAL declarations, so a key/unique/keyref
// on a local element was silently skipped: a dangling local keyref VALIDATED.
// xmllint rejects it. The fix records the actual *ElementDecl for each element
// instance during pass-1 and uses it in pass-2 before falling back to the
// global lookup.
func TestIDCLocalKeyRefEvaluated(t *testing.T) {
	t.Parallel()

	// Both the key (localItemKey) and the keyref (localRef) are declared on the
	// LOCAL element <items> nested inside <root>'s content model.
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="items">
          <xs:complexType>
            <xs:sequence>
              <xs:element name="item" maxOccurs="unbounded">
                <xs:complexType>
                  <xs:attribute name="id" type="xs:string"/>
                </xs:complexType>
              </xs:element>
              <xs:element name="ref" maxOccurs="unbounded">
                <xs:complexType>
                  <xs:attribute name="r" type="xs:string"/>
                </xs:complexType>
              </xs:element>
            </xs:sequence>
          </xs:complexType>
          <xs:key name="localItemKey">
            <xs:selector xpath="item"/>
            <xs:field xpath="@id"/>
          </xs:key>
          <xs:keyref name="localRef" refer="localItemKey">
            <xs:selector xpath="ref"/>
            <xs:field xpath="@r"/>
          </xs:keyref>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	v, compileErrs := compileXSD(t, schemaXML)
	require.Empty(t, compileErrs, "local IDC schema should compile clean")

	t.Run("dangling local keyref is rejected", func(t *testing.T) {
		t.Parallel()
		doc, err := helium.NewParser().Parse(t.Context(),
			[]byte(`<root><items><item id="a"/><ref r="missing"/></items></root>`))
		require.NoError(t, err)
		var errs string
		err = validateWithOutput(t, v, doc, &errs)
		require.Error(t, err, "a dangling keyref on a local element must be rejected, not silently skipped")
		require.Contains(t, errs, "No match found for key-sequence ['missing'] of keyref 'localRef'.")
	})

	t.Run("matching local keyref validates", func(t *testing.T) {
		t.Parallel()
		doc, err := helium.NewParser().Parse(t.Context(),
			[]byte(`<root><items><item id="a"/><ref r="a"/></items></root>`))
		require.NoError(t, err)
		var errs string
		err = validateWithOutput(t, v, doc, &errs)
		require.NoError(t, err, "expected valid, got: %s", errs)
	})
}

// TestIDCLocalKeyRefDescendantKey confirms XSD identity-constraint table
// propagation: a keyref on a LOCAL element referring to a key/unique declared on
// a DESCENDANT element is IN scope (descendant tables propagate up to the host),
// so a matching value validates and a dangling one is rejected. This mirrors the
// bug322411 golden case but with both constraints local. xmllint validates the
// matching instance and rejects the dangling one.
func TestIDCLocalKeyRefDescendantKey(t *testing.T) {
	t.Parallel()

	// keyref "ItemRef" is on the LOCAL <ELEMENT>; the unique "ItemUnique" it
	// refers to is on the GLOBAL <items>, a CHILD of <ELEMENT>.
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="items">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" maxOccurs="unbounded" type="xs:string"/>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="ItemUnique">
      <xs:selector xpath="item"/>
      <xs:field xpath="."/>
    </xs:unique>
  </xs:element>
  <xs:element name="ELEMENTS">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="ELEMENT">
          <xs:complexType>
            <xs:sequence>
              <xs:element ref="items"/>
              <xs:element name="ref" type="xs:string" maxOccurs="unbounded"/>
            </xs:sequence>
          </xs:complexType>
          <xs:keyref name="ItemRef" refer="ItemUnique">
            <xs:selector xpath="ref"/>
            <xs:field xpath="."/>
          </xs:keyref>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	v, compileErrs := compileXSD(t, schemaXML)
	require.Empty(t, compileErrs, "descendant-key keyref schema should compile clean")

	t.Run("descendant key satisfies keyref", func(t *testing.T) {
		t.Parallel()
		doc, err := helium.NewParser().Parse(t.Context(),
			[]byte(`<ELEMENTS><ELEMENT><items><item>a</item><item>b</item></items><ref>a</ref></ELEMENT></ELEMENTS>`))
		require.NoError(t, err)
		var errs string
		err = validateWithOutput(t, v, doc, &errs)
		require.NoError(t, err, "a key on a descendant element propagates up and satisfies the keyref; got: %s", errs)
	})

	t.Run("dangling keyref still rejected", func(t *testing.T) {
		t.Parallel()
		doc, err := helium.NewParser().Parse(t.Context(),
			[]byte(`<ELEMENTS><ELEMENT><items><item>a</item></items><ref>missing</ref></ELEMENT></ELEMENTS>`))
		require.NoError(t, err)
		var errs string
		err = validateWithOutput(t, v, doc, &errs)
		require.Error(t, err, "a dangling keyref must still be rejected even with descendant propagation")
		require.Contains(t, errs, "No match found for key-sequence ['missing'] of keyref 'ItemRef'.")
	})
}

// TestIDCLocalKeyRefSiblingKeyOutOfScope confirms the scope boundary: a keyref
// on a LOCAL element referring to a key/unique declared on a SIBLING local
// element is out of scope (the key is not in the keyref host occurrence's
// subtree), so even a value that exists under the sibling key does NOT satisfy
// it. xmllint rejects this case; subtree-scoped key-table gathering must too.
func TestIDCLocalKeyRefSiblingKeyOutOfScope(t *testing.T) {
	t.Parallel()

	// localItemKey is on <items>; localRef is on the SIBLING <refs>.
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="items">
          <xs:complexType>
            <xs:sequence>
              <xs:element name="item" maxOccurs="unbounded">
                <xs:complexType><xs:attribute name="id" type="xs:string"/></xs:complexType>
              </xs:element>
            </xs:sequence>
          </xs:complexType>
          <xs:key name="localItemKey"><xs:selector xpath="item"/><xs:field xpath="@id"/></xs:key>
        </xs:element>
        <xs:element name="refs">
          <xs:complexType>
            <xs:sequence>
              <xs:element name="ref" maxOccurs="unbounded">
                <xs:complexType><xs:attribute name="r" type="xs:string"/></xs:complexType>
              </xs:element>
            </xs:sequence>
          </xs:complexType>
          <xs:keyref name="localRef" refer="localItemKey"><xs:selector xpath="ref"/><xs:field xpath="@r"/></xs:keyref>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	v, compileErrs := compileXSD(t, schemaXML)
	require.Empty(t, compileErrs, "sibling-key keyref schema should compile clean (refer resolves in the symbol space)")

	// "a" exists under the sibling localItemKey, but it is out of localRef's scope.
	doc, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<root><items><item id="a"/></items><refs><ref r="a"/></refs></root>`))
	require.NoError(t, err)
	var errs string
	err = validateWithOutput(t, v, doc, &errs)
	require.Error(t, err, "a key on a sibling local element is out of the keyref scope")
	require.Contains(t, errs, "No match found for key-sequence ['a'] of keyref 'localRef'.")
}

// TestIDCLocalUniqueEvaluated confirms an xs:unique declared on a LOCAL element
// is evaluated at validation time: a duplicate key-sequence is rejected.
func TestIDCLocalUniqueEvaluated(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="items">
          <xs:complexType>
            <xs:sequence>
              <xs:element name="item" maxOccurs="unbounded">
                <xs:complexType>
                  <xs:attribute name="id" type="xs:string"/>
                </xs:complexType>
              </xs:element>
            </xs:sequence>
          </xs:complexType>
          <xs:unique name="localItemUnique">
            <xs:selector xpath="item"/>
            <xs:field xpath="@id"/>
          </xs:unique>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	v, compileErrs := compileXSD(t, schemaXML)
	require.Empty(t, compileErrs, "local unique schema should compile clean")

	t.Run("duplicate rejected", func(t *testing.T) {
		t.Parallel()
		doc, err := helium.NewParser().Parse(t.Context(),
			[]byte(`<root><items><item id="a"/><item id="a"/></items></root>`))
		require.NoError(t, err)
		var errs string
		err = validateWithOutput(t, v, doc, &errs)
		require.Error(t, err, "a duplicate key-sequence under a local unique must be rejected")
		require.Contains(t, errs, "Duplicate key-sequence")
	})

	t.Run("distinct validates", func(t *testing.T) {
		t.Parallel()
		doc, err := helium.NewParser().Parse(t.Context(),
			[]byte(`<root><items><item id="a"/><item id="b"/></items></root>`))
		require.NoError(t, err)
		var errs string
		err = validateWithOutput(t, v, doc, &errs)
		require.NoError(t, err, "expected valid, got: %s", errs)
	})
}
