package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// compileFatalErrors compiles a schema document and returns only the formatted
// fatal compile errors (warnings stripped) via the shared partition helper.
func compileFatalErrors(t *testing.T, schemaXML string) string {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	_, _ = xsd.NewCompiler().Label("test.xsd").ErrorHandler(collector).Compile(t.Context(), doc)
	_, errors := partitionCompileErrors(collector.Errors())
	return errors
}

// TestSchemaBooleanAttr (C-008) verifies that boolean schema attributes
// (abstract, mixed) are parsed with the full xs:boolean lexical space
// (true/false/1/0) and that invalid lexical forms are diagnosed instead of
// silently treated as false.
func TestSchemaBooleanAttr(t *testing.T) {
	t.Parallel()

	const invalidBool = "is not a valid value of the atomic type 'xs:boolean'"

	// validates returns the error from validating instanceXML against schema.
	validates := func(t *testing.T, schema, instanceXML string) error {
		t.Helper()
		sdoc, err := helium.NewParser().Parse(t.Context(), []byte(schema))
		require.NoError(t, err)
		s, err := xsd.NewCompiler().Compile(t.Context(), sdoc)
		require.NoError(t, err)
		doc, err := helium.NewParser().Parse(t.Context(), []byte(instanceXML))
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		return xsd.NewValidator(s).ErrorHandler(collector).Validate(t.Context(), doc)
	}

	t.Run("abstract=1 honored on complexType", func(t *testing.T) {
		t.Parallel()
		// abstract="1" must set the abstract flag (XSD boolean lexical form "1"),
		// so a direct (non-xsi:type) instance of the abstract type is rejected.
		// Before the fix == "true" left abstract silently false and this validated.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base" abstract="1">
    <xs:sequence><xs:element name="value" type="xs:string"/></xs:sequence>
  </xs:complexType>
  <xs:element name="root" type="Base"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
		require.Error(t, validates(t, schema, `<root><value>hello</value></root>`),
			"abstract=\"1\" complex type must reject a direct instance")
	})

	t.Run("abstract=0 leaves type concrete", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base" abstract="0">
    <xs:sequence><xs:element name="value" type="xs:string"/></xs:sequence>
  </xs:complexType>
  <xs:element name="root" type="Base"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
		require.NoError(t, validates(t, schema, `<root><value>hello</value></root>`),
			"abstract=\"0\" must leave the type concrete")
	})

	t.Run("rejects invalid boolean lexical forms", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name   string
			schema string
		}{
			{
				name: "element abstract yes",
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="xs:string" abstract="yes"/>
</xs:schema>`,
			},
			{
				name: "complexType abstract TRUE",
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base" abstract="TRUE"><xs:sequence/></xs:complexType>
  <xs:element name="root" type="Base"/>
</xs:schema>`,
			},
			{
				name: "complexType mixed maybe",
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base" mixed="maybe"><xs:sequence/></xs:complexType>
  <xs:element name="root" type="Base"/>
</xs:schema>`,
			},
			{
				name: "element nillable 2",
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="xs:string" nillable="2"/>
</xs:schema>`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				require.Contains(t, compileFatalErrors(t, tc.schema), invalidBool)
			})
		}
	})

	t.Run("accepts all valid boolean lexical forms", func(t *testing.T) {
		t.Parallel()
		for _, v := range []string{"true", "false", "1", "0"} {
			t.Run(v, func(t *testing.T) {
				t.Parallel()
				schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base" mixed="` + v + `"><xs:sequence/></xs:complexType>
  <xs:element name="root" type="Base" abstract="` + v + `" nillable="` + v + `"/>
</xs:schema>`
				require.Empty(t, compileFatalErrors(t, schema))
			})
		}
	})
}

// TestComplexTypeContentModelExclusivity (C-009) verifies that a complex type
// definition rejects more than one content-model particle and rejects mixing a
// content-model particle with a simpleContent/complexContent wrapper, instead
// of silently keeping the last child seen.
func TestComplexTypeContentModelExclusivity(t *testing.T) {
	t.Parallel()

	t.Run("rejects two model groups", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="T">
    <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
    <xs:choice><xs:element name="b" type="xs:string"/></xs:choice>
  </xs:complexType>
  <xs:element name="root" type="T"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), "more than one content model particle")
	})

	t.Run("rejects sequence then all", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="T">
    <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
    <xs:all><xs:element name="b" type="xs:string"/></xs:all>
  </xs:complexType>
  <xs:element name="root" type="T"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), "more than one content model particle")
	})

	t.Run("rejects model group with complexContent", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base"><xs:sequence/></xs:complexType>
  <xs:complexType name="T">
    <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
    <xs:complexContent>
      <xs:extension base="Base"/>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="T"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), "not allowed together")
	})

	t.Run("accepts single model group with attributes", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="T">
    <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
    <xs:attribute name="x" type="xs:string"/>
  </xs:complexType>
  <xs:element name="root" type="T"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})
}

// TestDuplicateAttributeUse (C-010) verifies that two attribute uses with the
// same expanded name within a single type are rejected, including the case
// where the collision arises from attribute-group expansion. Before the fix the
// validation-time map silently coalesced them.
func TestDuplicateAttributeUse(t *testing.T) {
	t.Parallel()

	const dup = "Duplicate attribute use"

	t.Run("rejects two direct attributes same name", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="T">
    <xs:attribute name="x" type="xs:string"/>
    <xs:attribute name="x" type="xs:int"/>
  </xs:complexType>
  <xs:element name="root" type="T"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), dup)
	})

	t.Run("rejects attribute colliding with attribute group", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="g">
    <xs:attribute name="x" type="xs:string"/>
  </xs:attributeGroup>
  <xs:complexType name="T">
    <xs:attribute name="x" type="xs:int"/>
    <xs:attributeGroup ref="g"/>
  </xs:complexType>
  <xs:element name="root" type="T"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), dup)
	})

	t.Run("rejects two attribute groups with same name", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="g1"><xs:attribute name="x" type="xs:string"/></xs:attributeGroup>
  <xs:attributeGroup name="g2"><xs:attribute name="x" type="xs:int"/></xs:attributeGroup>
  <xs:complexType name="T">
    <xs:attributeGroup ref="g1"/>
    <xs:attributeGroup ref="g2"/>
  </xs:complexType>
  <xs:element name="root" type="T"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), dup)
	})

	t.Run("accepts distinct attribute names", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="g"><xs:attribute name="y" type="xs:string"/></xs:attributeGroup>
  <xs:complexType name="T">
    <xs:attribute name="x" type="xs:int"/>
    <xs:attributeGroup ref="g"/>
  </xs:complexType>
  <xs:element name="root" type="T"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})
}

// TestWildcardConstraintValidation (C-011) verifies that a wildcard's
// processContents and namespace attributes are validated at compile time.
func TestWildcardConstraintValidation(t *testing.T) {
	t.Parallel()

	wildcardSchema := func(wildcard string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="T">
    <xs:sequence>
      ` + wildcard + `
    </xs:sequence>
  </xs:complexType>
  <xs:element name="root" type="T"/>
</xs:schema>`
	}

	t.Run("rejects bad processContents", func(t *testing.T) {
		t.Parallel()
		schema := wildcardSchema(`<xs:any processContents="bogus"/>`)
		require.Contains(t, compileFatalErrors(t, schema), "processContents")
	})

	t.Run("rejects bad namespace token", func(t *testing.T) {
		t.Parallel()
		schema := wildcardSchema(`<xs:any namespace="##bogus"/>`)
		require.Contains(t, compileFatalErrors(t, schema), "namespace constraint")
	})

	t.Run("rejects ##any combined with uri", func(t *testing.T) {
		t.Parallel()
		schema := wildcardSchema(`<xs:any namespace="##any http://example.com"/>`)
		require.Contains(t, compileFatalErrors(t, schema), "namespace constraint")
	})

	t.Run("rejects ##other combined with ##targetNamespace", func(t *testing.T) {
		t.Parallel()
		schema := wildcardSchema(`<xs:any namespace="##other ##targetNamespace"/>`)
		require.Contains(t, compileFatalErrors(t, schema), "namespace constraint")
	})

	t.Run("rejects bad processContents on anyAttribute", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="T">
    <xs:anyAttribute processContents="bogus"/>
  </xs:complexType>
  <xs:element name="root" type="T"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), "processContents")
	})

	t.Run("accepts valid wildcard constraints", func(t *testing.T) {
		t.Parallel()
		for _, w := range []string{
			`<xs:any/>`,
			`<xs:any namespace="##any" processContents="lax"/>`,
			`<xs:any namespace="##other" processContents="skip"/>`,
			`<xs:any namespace="##local ##targetNamespace" processContents="strict"/>`,
			`<xs:any namespace="http://example.com ##local"/>`,
		} {
			t.Run(w, func(t *testing.T) {
				t.Parallel()
				require.Empty(t, compileFatalErrors(t, wildcardSchema(w)))
			})
		}
	})
}
