package xsd_test

import (
	"strings"
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

// compileWarnings compiles a schema document and returns only the formatted
// non-fatal compile warnings.
func compileWarnings(t *testing.T, schemaXML string) string {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	_, _ = xsd.NewCompiler().Label("test.xsd").ErrorHandler(collector).Compile(t.Context(), doc)
	warnings, _ := partitionCompileErrors(collector.Errors())
	return warnings
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

	// The same "at most one content model particle" rule applies inside a
	// complexContent extension; a second model group must not silently
	// overwrite ContentModel.
	t.Run("rejects two model groups in extension", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base"><xs:sequence/></xs:complexType>
  <xs:complexType name="T">
    <xs:complexContent>
      <xs:extension base="Base">
        <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
        <xs:choice><xs:element name="b" type="xs:string"/></xs:choice>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="T"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), "more than one content model particle")
	})

	// And inside a complexContent restriction.
	t.Run("rejects two model groups in restriction", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base">
    <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="T">
    <xs:complexContent>
      <xs:restriction base="Base">
        <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
        <xs:choice><xs:element name="b" type="xs:string"/></xs:choice>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="T"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), "more than one content model particle")
	})

	t.Run("accepts single model group in extension", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base"><xs:sequence/></xs:complexType>
  <xs:complexType name="T">
    <xs:complexContent>
      <xs:extension base="Base">
        <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
        <xs:attribute name="x" type="xs:string"/>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="T"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})

	// An <xs:group ref> is a model-group particle too, so a group ref beside a
	// sequence inside an extension is two content-model particles and must be
	// rejected — not silently dropped.
	t.Run("rejects group and sequence in extension", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="g"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:group>
  <xs:complexType name="Base"><xs:sequence/></xs:complexType>
  <xs:complexType name="T">
    <xs:complexContent>
      <xs:extension base="Base">
        <xs:group ref="g"/>
        <xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="T"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), "more than one content model particle")
	})

	// Same for a complexContent restriction.
	t.Run("rejects group and sequence in restriction", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="g"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:group>
  <xs:complexType name="Base">
    <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="T">
    <xs:complexContent>
      <xs:restriction base="Base">
        <xs:group ref="g"/>
        <xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="T"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), "more than one content model particle")
	})

	// A single <xs:group ref> as the sole content model of an extension must
	// compile cleanly AND its content must be honored at validation time (not
	// silently dropped).
	t.Run("accepts and honors single group ref in extension", func(t *testing.T) {
		t.Parallel()
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

		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="g"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:group>
  <xs:complexType name="Base"><xs:sequence/></xs:complexType>
  <xs:complexType name="T">
    <xs:complexContent>
      <xs:extension base="Base">
        <xs:group ref="g"/>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="T"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
		require.NoError(t, validates(t, schema, `<root><a>hello</a></root>`),
			"a single group ref content model must accept its content")
		require.Error(t, validates(t, schema, `<root/>`),
			"the group ref content model must be honored (required element a missing)")
	})

	// A single <xs:group ref> content model on a restriction compiles cleanly
	// and is honored.
	t.Run("accepts single group ref in restriction", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="g"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:group>
  <xs:complexType name="Base">
    <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="T">
    <xs:complexContent>
      <xs:restriction base="Base">
        <xs:group ref="g"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="T"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})

	// A direct attribute sibling that follows a simpleContent/complexContent
	// wrapper is illegal: attributes must be declared inside the wrapper's
	// restriction/extension, not as direct complexType children. xmllint rejects
	// this; before the fix the stray sibling was silently accepted.
	t.Run("rejects attribute after simpleContent wrapper", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="T">
    <xs:simpleContent>
      <xs:extension base="xs:string"/>
    </xs:simpleContent>
    <xs:attribute name="x" type="xs:string"/>
  </xs:complexType>
  <xs:element name="root" type="T"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), "not allowed together")
	})

	t.Run("rejects attribute after complexContent wrapper", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base"><xs:sequence/></xs:complexType>
  <xs:complexType name="T">
    <xs:complexContent>
      <xs:extension base="Base"/>
    </xs:complexContent>
    <xs:attribute name="x" type="xs:string"/>
  </xs:complexType>
  <xs:element name="root" type="T"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), "not allowed together")
	})

	// The reverse order is equally illegal: a direct attribute sibling that
	// precedes the wrapper.
	t.Run("rejects simpleContent wrapper after attribute", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="T">
    <xs:attribute name="x" type="xs:string"/>
    <xs:simpleContent>
      <xs:extension base="xs:string"/>
    </xs:simpleContent>
  </xs:complexType>
  <xs:element name="root" type="T"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), "not allowed together")
	})

	// An attributeGroup or anyAttribute sibling alongside a wrapper is rejected
	// the same way.
	t.Run("rejects anyAttribute after complexContent wrapper", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base"><xs:sequence/></xs:complexType>
  <xs:complexType name="T">
    <xs:complexContent>
      <xs:extension base="Base"/>
    </xs:complexContent>
    <xs:anyAttribute/>
  </xs:complexType>
  <xs:element name="root" type="T"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), "not allowed together")
	})

	// A normal complex type with direct attributes and no wrapper stays valid.
	t.Run("accepts direct attributes without a wrapper", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="T">
    <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
    <xs:attribute name="x" type="xs:string"/>
    <xs:attribute name="y" type="xs:string"/>
    <xs:anyAttribute/>
  </xs:complexType>
  <xs:element name="root" type="T"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})

	// XSD 3.4.2 fixes the child ordering: an optional leading model-group
	// particle, then attribute uses, then an optional anyAttribute. A model
	// group that appears AFTER an attribute declaration is out of order and
	// must be rejected, not silently kept as the content model.
	t.Run("rejects sequence after attribute", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="T">
    <xs:attribute name="x" type="xs:string"/>
    <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
  </xs:complexType>
  <xs:element name="root" type="T"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), "must appear before the attribute declaration")
	})

	t.Run("rejects choice after attributeGroup", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="g"><xs:attribute name="x" type="xs:string"/></xs:attributeGroup>
  <xs:complexType name="T">
    <xs:attributeGroup ref="g"/>
    <xs:choice><xs:element name="a" type="xs:string"/></xs:choice>
  </xs:complexType>
  <xs:element name="root" type="T"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), "must appear before the attribute declaration")
	})

	t.Run("rejects sequence after attribute in extension", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base"><xs:sequence/></xs:complexType>
  <xs:complexType name="T">
    <xs:complexContent>
      <xs:extension base="Base">
        <xs:attribute name="x" type="xs:string"/>
        <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="T"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), "must appear before the attribute declaration")
	})

	t.Run("rejects sequence after attribute in restriction", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base">
    <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="T">
    <xs:complexContent>
      <xs:restriction base="Base">
        <xs:attribute name="x" type="xs:string"/>
        <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="T"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), "must appear before the attribute declaration")
	})

	// The correct order — model group first, then attributes, then anyAttribute
	// — compiles cleanly in a plain complex type and in both derivations.
	t.Run("accepts correct order in extension", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base"><xs:sequence/></xs:complexType>
  <xs:complexType name="T">
    <xs:complexContent>
      <xs:extension base="Base">
        <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
        <xs:attribute name="x" type="xs:string"/>
        <xs:anyAttribute/>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="T"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})

	t.Run("accepts correct order in restriction", func(t *testing.T) {
		t.Parallel()
		// The base supplies the attribute and wildcard; the restriction repeats
		// them in the correct order (model group, then attribute, then
		// anyAttribute) so the derivation is valid.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base">
    <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
    <xs:attribute name="x" type="xs:string"/>
    <xs:anyAttribute/>
  </xs:complexType>
  <xs:complexType name="T">
    <xs:complexContent>
      <xs:restriction base="Base">
        <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
        <xs:attribute name="x" type="xs:string"/>
        <xs:anyAttribute/>
      </xs:restriction>
    </xs:complexContent>
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

	// An attribute group with two attributes of the same expanded QName is
	// invalid (ag-props-correct.2) even when NO complex type references it. Such
	// a group is never merged into any type's attribute set, so the per-type
	// duplicate check never inspects it; the dedicated attribute-group check must
	// catch it. xmllint rejects this schema.
	t.Run("rejects duplicate in unreferenced attribute group", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="g">
    <xs:attribute name="x" type="xs:string"/>
    <xs:attribute name="x" type="xs:int"/>
  </xs:attributeGroup>
  <xs:element name="root" type="xs:string"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), dup)
	})

	// An attribute group with an internal duplicate is reported exactly once,
	// attributed to the attribute group, even when a complex type references it
	// (xmllint does not also report it against the referencing type).
	t.Run("reports internal group duplicate once when referenced", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="g">
    <xs:attribute name="x" type="xs:string"/>
    <xs:attribute name="x" type="xs:int"/>
  </xs:attributeGroup>
  <xs:complexType name="T">
    <xs:attributeGroup ref="g"/>
  </xs:complexType>
  <xs:element name="root" type="T"/>
</xs:schema>`
		errs := compileFatalErrors(t, schema)
		require.Equal(t, 1, strings.Count(errs, dup),
			"internal attribute-group duplicate must be reported once: %s", errs)
	})

	// Extension duplicate detection must key on the expanded QName, not the
	// local name alone: a base attribute {}a and an extension attribute
	// {urn:t}a share a local name but are distinct attribute uses.
	t.Run("accepts same local name in different namespaces across extension", func(t *testing.T) {
		t.Parallel()
		// Base has an unqualified local attribute {}a; the extension references a
		// global attribute {urn:t}a. Same local name, distinct namespaces => not
		// a duplicate.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:t="urn:t" targetNamespace="urn:t" attributeFormDefault="unqualified">
  <xs:complexType name="Base">
    <xs:attribute name="a" type="xs:string"/>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:extension base="t:Base">
        <xs:attribute ref="t:a"/>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:attribute name="a" type="xs:string"/>
  <xs:element name="root" type="t:Derived"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})

	// A prohibited attribute ref="..." use="prohibited" contributes no
	// attribute use, so it must not collide with a real use of the same
	// attribute. Before the fix the ref path never set Prohibited, so the
	// duplicate-use check rejected schemas libxml2 accepts.
	t.Run("accepts prohibited ref beside the same real use", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:t="urn:t" targetNamespace="urn:t" attributeFormDefault="unqualified">
  <xs:attribute name="a" type="xs:string"/>
  <xs:complexType name="T">
    <xs:attribute ref="t:a"/>
    <xs:attribute ref="t:a" use="prohibited"/>
  </xs:complexType>
  <xs:element name="root" type="t:T"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})

	t.Run("rejects same expanded QName across base and extension", func(t *testing.T) {
		t.Parallel()
		// Both the base and the extension reference the same global attribute
		// {urn:t}a, so the expanded QNames collide and it is a real duplicate.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:t="urn:t" targetNamespace="urn:t" attributeFormDefault="unqualified">
  <xs:complexType name="Base">
    <xs:attribute ref="t:a"/>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:extension base="t:Base">
        <xs:attribute ref="t:a"/>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:attribute name="a" type="xs:string"/>
  <xs:element name="root" type="t:Derived"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), dup)
	})

	// A simpleContent extension that redeclares an attribute its base already
	// declares is a duplicate use. Before the fix the simpleContent branch
	// inherited base attributes and returned BEFORE the base-vs-derived check,
	// so the redeclaration compiled clean.
	t.Run("rejects simpleContent extension redeclaring a base attribute", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base">
    <xs:simpleContent>
      <xs:extension base="xs:string">
        <xs:attribute name="a" type="xs:string"/>
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:simpleContent>
      <xs:extension base="Base">
        <xs:attribute name="a" type="xs:int"/>
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>
  <xs:element name="root" type="Derived"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), dup)
	})

	// A prohibited attribute use (here pulled in via an attribute group on the
	// base) contributes no attribute use, so the same attribute really used by
	// the extension must NOT false-trigger a duplicate. Before the fix the
	// base-vs-derived check counted prohibited uses (unlike checkDuplicateAttrUses).
	t.Run("accepts prohibited base attr via group beside extension use", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:t="urn:t" targetNamespace="urn:t" attributeFormDefault="unqualified">
  <xs:attribute name="a" type="xs:string"/>
  <xs:attributeGroup name="g">
    <xs:attribute ref="t:a" use="prohibited"/>
  </xs:attributeGroup>
  <xs:complexType name="Base">
    <xs:sequence/>
    <xs:attributeGroup ref="t:g"/>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:extension base="t:Base">
        <xs:sequence/>
        <xs:attribute ref="t:a"/>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:Derived"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})
}

// TestSimpleContentExtensionProhibitedAttr verifies that a simpleContent
// EXTENSION treats use="prohibited" as pointless (warn + skip) just like a
// complexContent extension, instead of propagating a prohibited attribute use
// that would wrongly block an attribute the base's wildcard admits.
func TestSimpleContentExtensionProhibitedAttr(t *testing.T) {
	t.Parallel()

	const warn = "Skipping attribute use prohibition, since it is pointless when extending a type."

	// Base is a complex type with simple content and an attribute wildcard, so a
	// foreign/extra attribute would normally be admitted via the wildcard. The
	// derived simpleContent extension carries a pointless use="prohibited"; that
	// prohibition must be warned+skipped, not turned into a real prohibited use
	// that blocks the attribute.
	schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:t="urn:t" targetNamespace="urn:t" attributeFormDefault="unqualified">
  <xs:attribute name="a" type="xs:string"/>
  <xs:complexType name="Base">
    <xs:simpleContent>
      <xs:extension base="xs:string">
        <xs:anyAttribute processContents="lax"/>
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:simpleContent>
      <xs:extension base="t:Base">
        <xs:attribute ref="t:a" use="prohibited"/>
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>
  <xs:element name="root" type="t:Derived"/>
</xs:schema>`

	t.Run("warns and compiles clean", func(t *testing.T) {
		t.Parallel()
		require.Empty(t, compileFatalErrors(t, schema))
		require.Contains(t, compileWarnings(t, schema), warn)
	})

	t.Run("attribute admitted by base wildcard", func(t *testing.T) {
		t.Parallel()
		sdoc, err := helium.NewParser().Parse(t.Context(), []byte(schema))
		require.NoError(t, err)
		s, err := xsd.NewCompiler().Compile(t.Context(), sdoc)
		require.NoError(t, err)
		doc, err := helium.NewParser().Parse(t.Context(),
			[]byte(`<root xmlns="urn:t" a="hi">text</root>`))
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		require.NoError(t, xsd.NewValidator(s).ErrorHandler(collector).Validate(t.Context(), doc),
			"the prohibited use must be skipped so the base wildcard admits @a")
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

	// The "##any" / "##other" keywords are EXACT singleton lexical forms: they
	// must not be padded with whitespace. Before the fix the value was
	// whitespace-collapsed before validation, so "  ##any  " wrongly compiled.
	// libxml2 rejects the padded keyword forms.
	t.Run("rejects padded ##any", func(t *testing.T) {
		t.Parallel()
		schema := wildcardSchema(`<xs:any namespace="  ##any  "/>`)
		require.Contains(t, compileFatalErrors(t, schema), "namespace constraint")
	})

	t.Run("rejects padded ##other", func(t *testing.T) {
		t.Parallel()
		schema := wildcardSchema(`<xs:any namespace="  ##other  "/>`)
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
			// True list forms tolerate surrounding/inner whitespace: only the
			// ##any / ##other keywords are exact singletons.
			`<xs:any namespace="  ##local   ##targetNamespace  " processContents="strict"/>`,
		} {
			t.Run(w, func(t *testing.T) {
				t.Parallel()
				require.Empty(t, compileFatalErrors(t, wildcardSchema(w)))
			})
		}
	})

	// A present-but-empty namespace="" is a (degenerate) namespace list that
	// matches nothing; it is distinct from an ABSENT namespace which defaults
	// to ##any. Before the fix readWildcard rewrote "" to ##any, so a skip
	// wildcard with namespace="" wrongly accepted any child.
	t.Run("empty namespace matches nothing", func(t *testing.T) {
		t.Parallel()
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

		emptyNS := wildcardSchema(`<xs:any namespace="" processContents="skip"/>`)
		require.Empty(t, compileFatalErrors(t, emptyNS))
		require.Error(t, validates(t, emptyNS, `<root><child/></root>`),
			`namespace="" must match no child`)

		absentNS := wildcardSchema(`<xs:any processContents="skip"/>`)
		require.Empty(t, compileFatalErrors(t, absentNS))
		require.NoError(t, validates(t, absentNS, `<root><child/></root>`),
			"absent namespace defaults to ##any and must accept any child")
	})
}

// TestPointlessAttributeProhibition verifies that a prohibited attribute use
// whose QName already names a real (non-prohibited) use in the same complex type
// is reported with the libxml2-compatible schema parser WARNING (the prohibition
// is pointless because the type itself declares the use). A prohibited use that
// does NOT duplicate a real use (the normal restriction case) stays silent.
func TestPointlessAttributeProhibition(t *testing.T) {
	t.Parallel()

	const warn = "Skipping pointless attribute use prohibition 'x', since a corresponding attribute use exists already in the type definition."

	t.Run("warns on prohibition duplicating a real use", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="T">
    <xs:attribute name="x" type="xs:string"/>
    <xs:attribute name="x" use="prohibited"/>
  </xs:complexType>
  <xs:element name="root" type="T"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema),
			"a pointless prohibition is a warning, not a fatal duplicate")
		require.Contains(t, compileWarnings(t, schema), warn)
	})

	// A use="prohibited" attribute declared directly inside an <xs:attributeGroup>
	// is pointless there: libxml2 warns and SKIPS it at the group, so it is never
	// propagated into a referencing type. The warning therefore cites the
	// <attributeGroup> context, and the type-level "corresponding use exists
	// already" warning does NOT fire (the prohibition never reaches the type).
	t.Run("skips prohibition declared inside an attribute group", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="g">
    <xs:attribute name="x" use="prohibited"/>
  </xs:attributeGroup>
  <xs:complexType name="T">
    <xs:attribute name="x" type="xs:string"/>
    <xs:attributeGroup ref="g"/>
  </xs:complexType>
  <xs:element name="root" type="T"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
		warnings := compileWarnings(t, schema)
		require.Contains(t, warnings,
			"Skipping attribute use prohibition, since it is pointless inside an <attributeGroup>.")
		require.NotContains(t, warnings, warn,
			"the prohibition is skipped at the group, so the type-level pointless warning must not fire")
	})

	t.Run("stays silent for a non-duplicating prohibition", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base">
    <xs:attribute name="x" type="xs:string"/>
  </xs:complexType>
  <xs:complexType name="T">
    <xs:complexContent>
      <xs:restriction base="Base">
        <xs:attribute name="x" use="prohibited"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="T"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
		require.NotContains(t, compileWarnings(t, schema), "pointless attribute use prohibition")
	})
}
