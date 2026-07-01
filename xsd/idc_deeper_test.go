package xsd_test

import (
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestIDOnRootDenotesNoElement covers the XSD 1.1 ID/IDREF rule that an xs:ID
// appearing as the ELEMENT CONTENT of the document root denotes no element (an
// element-content ID identifies its PARENT element, and the root has none). The
// ID is therefore never registered, so an xs:IDREF to it dangles and the instance
// is invalid (W3C ibmData idIDREF s3_3_4ii26/ii27). The same content wrapped in a
// parent element is valid, because then the ID identifies the wrapping parent.
func TestIDOnRootDenotesNoElement(t *testing.T) {
	compileValidate := func(t *testing.T, schemaXML, instanceXML string) error {
		t.Helper()
		sdoc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		schema, err := xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), sdoc)
		require.NoError(t, err)
		idoc, err := helium.NewParser().Parse(t.Context(), []byte(instanceXML))
		require.NoError(t, err)
		return xsd.NewValidator(schema).Validate(t.Context(), idoc)
	}

	// root: simpleContent that is a LIST of xs:ID plus an xs:IDREF attribute.
	const listSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:simpleContent>
        <xs:extension base="listOfIDs">
          <xs:attribute name="idref_attr" type="xs:IDREF"/>
        </xs:extension>
      </xs:simpleContent>
    </xs:complexType>
  </xs:element>
  <xs:simpleType name="listOfIDs"><xs:list itemType="xs:ID"/></xs:simpleType>
</xs:schema>`

	t.Run("list of ID on document root is invalid (ii26)", func(t *testing.T) {
		t.Parallel()
		// IDs b1/b2/b3 are root content but root has no parent, so they denote no
		// element; idref_attr="b2" then has no binding.
		require.Error(t, compileValidate(t, listSchema, `<root idref_attr="b2">b1 b2 b3</root>`),
			"an ID in the document root's content denotes no element, so the IDREF dangles")
	})

	// root: simpleContent that is a UNION involving xs:ID plus an xs:IDREF attribute.
	const unionSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:simpleContent>
        <xs:extension base="unionOfIDs">
          <xs:attribute name="idref_attr" type="xs:IDREF"/>
        </xs:extension>
      </xs:simpleContent>
    </xs:complexType>
  </xs:element>
  <xs:simpleType name="unionOfIDs"><xs:union memberTypes="xs:integer xs:boolean xs:ID"/></xs:simpleType>
</xs:schema>`

	t.Run("union of ID on document root is invalid (ii27)", func(t *testing.T) {
		t.Parallel()
		require.Error(t, compileValidate(t, unionSchema, `<root idref_attr="b2">b2</root>`),
			"a union ID in the document root's content denotes no element, so the IDREF dangles")
	})

	// Same content, but the ID-bearing element is now a child, so its content ID
	// identifies its PARENT and the IDREF on a sibling resolves.
	const wrappedSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="root">
          <xs:complexType>
            <xs:simpleContent>
              <xs:extension base="listOfIDs">
                <xs:attribute name="idref_attr" type="xs:IDREF"/>
              </xs:extension>
            </xs:simpleContent>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:simpleType name="listOfIDs"><xs:list itemType="xs:ID"/></xs:simpleType>
</xs:schema>`

	t.Run("list of ID on a non-root element is valid (denotes parent)", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, compileValidate(t, wrappedSchema,
			`<doc><root idref_attr="b2">b1 b2 b3</root></doc>`),
			"an element-content ID identifies its parent, so the IDREF on the same element resolves")
	})
}

// TestElementDefaultValidity covers the XSD 1.1 (Element Default Valid) check: an
// element declaration's default/fixed value must be valid against its declared
// simple type, enforced at compile time (W3C ibmData idIDREF s3_3_4si07/si08).
func TestElementDefaultValidity(t *testing.T) {
	compile := func(t *testing.T, schemaXML string) error {
		t.Helper()
		sdoc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		_, err = xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), sdoc)
		return err
	}
	compile10 := func(t *testing.T, schemaXML string) error {
		t.Helper()
		sdoc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		_, err = xsd.NewCompiler().Version(xsd.Version10).Compile(t.Context(), sdoc)
		return err
	}

	// si07: default "aka" on an element whose type is a list of (xs:ID restricted
	// to the enumeration "ala"). "aka" is not "ala", so the default is invalid.
	const listSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="list_of_ids" type="listOfIDs" default="DEFVAL"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:simpleType name="listOfIDs"><xs:list itemType="listTypeA"/></xs:simpleType>
  <xs:simpleType name="listTypeA">
    <xs:restriction base="xs:ID"><xs:enumeration value="ala"/></xs:restriction>
  </xs:simpleType>
</xs:schema>`

	t.Run("invalid element default on list of ID-enum is a schema error (si07)", func(t *testing.T) {
		t.Parallel()
		schema := strings.ReplaceAll(listSchema, "DEFVAL", "aka")
		require.ErrorIs(t, compile(t, schema), xsd.ErrCompilationFailed)
	})

	t.Run("valid element default on list of ID-enum compiles", func(t *testing.T) {
		t.Parallel()
		schema := strings.ReplaceAll(listSchema, "DEFVAL", "ala")
		require.NoError(t, compile(t, schema))
	})

	// si08: default "id_a1" on an element of union(xs:integer, xs:boolean, xs:ID
	// restricted to enumeration "id_a"). "id_a1" matches no member, so invalid.
	const unionSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="union_of_ids" type="unionOfIDs" default="DEFVAL"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:simpleType name="unionOfIDs"><xs:union memberTypes="xs:integer xs:boolean unionTypeA"/></xs:simpleType>
  <xs:simpleType name="unionTypeA">
    <xs:restriction base="xs:ID"><xs:enumeration value="id_a"/></xs:restriction>
  </xs:simpleType>
</xs:schema>`

	t.Run("invalid element default on union with ID is a schema error (si08)", func(t *testing.T) {
		t.Parallel()
		schema := strings.ReplaceAll(unionSchema, "DEFVAL", "id_a1")
		require.ErrorIs(t, compile(t, schema), xsd.ErrCompilationFailed)
	})

	t.Run("valid element default on union with ID compiles", func(t *testing.T) {
		t.Parallel()
		schema := strings.ReplaceAll(unionSchema, "DEFVAL", "id_a")
		require.NoError(t, compile(t, schema))
	})

	// A plain invalid default against a builtin type is also rejected at compile.
	const intSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="n" type="xs:int" default="DEFVAL"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	t.Run("invalid element default on xs:int is a schema error", func(t *testing.T) {
		t.Parallel()
		require.ErrorIs(t, compile(t, strings.ReplaceAll(intSchema, "DEFVAL", "notint")), xsd.ErrCompilationFailed)
	})

	t.Run("valid element default on xs:int compiles", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, compile(t, strings.ReplaceAll(intSchema, "DEFVAL", "42")))
	})

	t.Run("XSD 1.0 also enforces element default validity", func(t *testing.T) {
		t.Parallel()
		// Element Default Valid (Immediate) (§3.3.6) is a version-independent XSD rule,
		// so an invalid element default is a schema error in 1.0 as well as 1.1.
		require.ErrorIs(t, compile10(t, strings.ReplaceAll(intSchema, "DEFVAL", "notint")), xsd.ErrCompilationFailed)
	})

	// PR885-EDV-SG-INHERIT: a no-type global substitution-group member inherits its
	// head's type, so its default/fixed must be validated against the INHERITED
	// (effective) type, not the member's own nil Type.
	const sgSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="head" type="xs:int"/>
  <xs:element name="member" substitutionGroup="head" default="DEFVAL"/>
</xs:schema>`

	t.Run("invalid default on no-type substitution-group member is a schema error", func(t *testing.T) {
		t.Parallel()
		require.ErrorIs(t, compile(t, strings.ReplaceAll(sgSchema, "DEFVAL", "notint")), xsd.ErrCompilationFailed)
	})

	t.Run("valid default on no-type substitution-group member compiles", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, compile(t, strings.ReplaceAll(sgSchema, "DEFVAL", "42")))
	})

	t.Run("invalid fixed on no-type substitution-group member is a schema error", func(t *testing.T) {
		t.Parallel()
		s := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="head" type="xs:int"/>
  <xs:element name="member" substitutionGroup="head" fixed="notint"/>
</xs:schema>`
		require.ErrorIs(t, compile(t, s), xsd.ErrCompilationFailed)
	})

	// PR885-EDV-SC-BASE: a simpleContent element default must satisfy the INHERITED
	// base content facets, not only the nested <xs:simpleType>'s own facets. Base
	// content is xs:string maxLength=2; the derived nested type only adds minLength=1,
	// so a default that the nested type accepts but the base maxLength rejects must
	// still be a schema error. (Same schema shape as the runtime
	// TestVersion11SimpleContentNestedTypeKeepsBaseFacets.)
	const scBaseSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:simpleContent>
      <xs:restriction base="xs:anySimpleType">
        <xs:simpleType><xs:restriction base="xs:string"><xs:maxLength value="2"/></xs:restriction></xs:simpleType>
      </xs:restriction>
    </xs:simpleContent>
  </xs:complexType>
  <xs:complexType name="mid">
    <xs:simpleContent>
      <xs:restriction base="base">
        <xs:simpleType><xs:restriction base="xs:string"><xs:minLength value="1"/></xs:restriction></xs:simpleType>
      </xs:restriction>
    </xs:simpleContent>
  </xs:complexType>
  <xs:element name="e" default="DEFVAL">
    <xs:complexType><xs:simpleContent><xs:extension base="mid"/></xs:simpleContent></xs:complexType>
  </xs:element>
</xs:schema>`

	t.Run("simpleContent default violating inherited base facet is a schema error", func(t *testing.T) {
		t.Parallel()
		// "abc" satisfies the nested minLength=1 but violates the inherited maxLength=2.
		require.ErrorIs(t, compile(t, strings.ReplaceAll(scBaseSchema, "DEFVAL", "abc")), xsd.ErrCompilationFailed)
	})

	t.Run("simpleContent default satisfying all chain facets compiles", func(t *testing.T) {
		t.Parallel()
		// "ab" satisfies both the nested minLength=1 and the inherited maxLength=2.
		require.NoError(t, compile(t, strings.ReplaceAll(scBaseSchema, "DEFVAL", "ab")))
	})
}

// TestElementDefaultValidityXSD10 exercises Element Default Valid (Immediate)
// (§3.3.6) under the DEFAULT (XSD 1.0) compiler. The rule is version-independent:
// an element declaration's explicit default/fixed value must be valid against its
// declared simple (content) type in 1.0 exactly as in 1.1. These cases mirror the
// sun sunMeta/ElemDecl.testSet valueConstraint invalid schemas (a decimal "XII",
// a boolean "Yes", a float "1.0F-2", a pattern-violating restriction, a
// simpleContent extension of xs:boolean).
func TestElementDefaultValidityXSD10(t *testing.T) {
	t.Parallel()

	compile := func(t *testing.T, schemaXML string) error {
		t.Helper()
		sdoc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		_, err = xsd.NewCompiler().Compile(t.Context(), sdoc)
		return err
	}

	for _, tc := range []struct {
		name       string
		decl       string
		wantReject bool
	}{
		{"decimal default XII invalid", `<xs:element name="root" type="xs:decimal" default="XII"/>`, true},
		{"boolean default Yes invalid", `<xs:element name="E" type="xs:boolean" default="Yes"/>`, true},
		{"boolean fixed Yes invalid", `<xs:element name="E" type="xs:boolean" fixed="Yes"/>`, true},
		{"float fixed 1.0F-2 invalid", `<xs:element name="root" type="xs:float" fixed="1.0F-2"/>`, true},
		{"float default 1.0F-2 invalid", `<xs:element name="root" type="xs:float" default="1.0F-2"/>`, true},
		{"decimal default 12 valid", `<xs:element name="root" type="xs:decimal" default="12"/>`, false},
		{"boolean default true valid", `<xs:element name="E" type="xs:boolean" default="true"/>`, false},
		{"float fixed 1.0 valid", `<xs:element name="root" type="xs:float" fixed="1.0"/>`, false},
		{"ur-type default alpha valid", `<xs:element name="root" default="alpha"/>`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">` + tc.decl + `</xs:schema>`
			err := compile(t, s)
			if tc.wantReject {
				require.ErrorIs(t, err, xsd.ErrCompilationFailed)
				return
			}
			require.NoError(t, err)
		})
	}

	// A restriction with a pattern the default violates is rejected, and a
	// simpleContent extension of xs:boolean validates its default against the base
	// boolean type — both under the default 1.0 compiler.
	t.Run("restriction pattern violated by default invalid", func(t *testing.T) {
		t.Parallel()
		s := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="E" type="answer" default="false"/>
  <xs:simpleType name="answer">
    <xs:restriction base="xs:boolean"><xs:pattern value="true"/></xs:restriction>
  </xs:simpleType>
</xs:schema>`
		require.ErrorIs(t, compile(t, s), xsd.ErrCompilationFailed)
	})

	t.Run("simpleContent extension of boolean with invalid default invalid", func(t *testing.T) {
		t.Parallel()
		s := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="E" type="answer" default="Yes"/>
  <xs:complexType name="answer">
    <xs:simpleContent>
      <xs:extension base="xs:boolean"><xs:attribute name="certainty"/></xs:extension>
    </xs:simpleContent>
  </xs:complexType>
</xs:schema>`
		require.ErrorIs(t, compile(t, s), xsd.ErrCompilationFailed)
	})
}
