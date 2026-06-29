package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// validateOC compiles schemaXML (XSD 1.1) and validates instanceXML against it,
// returning the validation error (nil = valid). It fails the test if the schema
// itself does not compile.
func validateOC(t *testing.T, schemaXML, instanceXML string) error {
	t.Helper()
	schema, _, cerr := compileV11(t, schemaXML)
	require.NoError(t, cerr, "schema must compile")
	require.NotNil(t, schema)
	idoc, err := helium.NewParser().Parse(t.Context(), []byte(instanceXML))
	require.NoError(t, err)
	return xsd.NewValidator(schema).Validate(t.Context(), idoc)
}

// TestDefaultOpenContent_Apply covers the schema-level <xs:defaultOpenContent>:
// it applies to a complex type that has no explicit <xs:openContent>, honoring
// mode (interleave/suffix) and appliesToEmpty, and is suppressed by an explicit
// openContent (including mode="none").
func TestDefaultOpenContent_Apply(t *testing.T) {
	t.Parallel()
	const head = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">`

	t.Run("suffix applies to non-empty type", func(t *testing.T) {
		t.Parallel()
		schema := head + `
  <xs:defaultOpenContent mode="suffix">
    <xs:any namespace="http://open.com/" processContents="lax"/>
  </xs:defaultOpenContent>
  <xs:element name="doc">
    <xs:complexType><xs:sequence><xs:element name="a"/></xs:sequence></xs:complexType>
  </xs:element>
</xs:schema>`
		require.NoError(t, validateOC(t, schema,
			`<doc><a/><extra xmlns="http://open.com/"/></doc>`))
		// a declared element appearing after the suffix open content is misplaced.
		require.Error(t, validateOC(t, schema,
			`<doc><a/><extra xmlns="http://open.com/"/><a/></doc>`))
	})

	t.Run("interleave applies", func(t *testing.T) {
		t.Parallel()
		schema := head + `
  <xs:defaultOpenContent mode="interleave">
    <xs:any namespace="http://open.com/" processContents="lax"/>
  </xs:defaultOpenContent>
  <xs:element name="doc">
    <xs:complexType><xs:sequence><xs:element name="a"/><xs:element name="b"/></xs:sequence></xs:complexType>
  </xs:element>
</xs:schema>`
		require.NoError(t, validateOC(t, schema,
			`<doc><a/><extra xmlns="http://open.com/"/><b/></doc>`))
	})

	t.Run("appliesToEmpty governs empty content type", func(t *testing.T) {
		t.Parallel()
		mk := func(applies string) string {
			return head + `
  <xs:defaultOpenContent mode="suffix" appliesToEmpty="` + applies + `">
    <xs:any namespace="http://open.com/" processContents="lax"/>
  </xs:defaultOpenContent>
  <xs:element name="doc"><xs:complexType><xs:sequence/></xs:complexType></xs:element>
</xs:schema>`
		}
		require.NoError(t, validateOC(t, mk("true"),
			`<doc><extra xmlns="http://open.com/"/></doc>`))
		require.Error(t, validateOC(t, mk("false"),
			`<doc><extra xmlns="http://open.com/"/></doc>`))
	})

	t.Run("explicit mode=none suppresses default", func(t *testing.T) {
		t.Parallel()
		schema := head + `
  <xs:defaultOpenContent mode="suffix" appliesToEmpty="true">
    <xs:any namespace="http://open.com/" processContents="lax"/>
  </xs:defaultOpenContent>
  <xs:element name="doc">
    <xs:complexType>
      <xs:openContent mode="none"/>
      <xs:sequence><xs:element name="a"/></xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		require.Error(t, validateOC(t, schema,
			`<doc><a/><extra xmlns="http://open.com/"/></doc>`))
	})
}

// TestDefaultOpenContent_AllContentModel verifies default/explicit open content
// works inside an xs:all content model in both suffix and interleave modes.
func TestDefaultOpenContent_AllContentModel(t *testing.T) {
	t.Parallel()
	const head = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">`

	t.Run("suffix in all", func(t *testing.T) {
		t.Parallel()
		schema := head + `
  <xs:defaultOpenContent mode="suffix">
    <xs:any namespace="http://open.com/" processContents="lax"/>
  </xs:defaultOpenContent>
  <xs:element name="doc">
    <xs:complexType><xs:all>
      <xs:element name="a"/><xs:element name="b"/>
    </xs:all></xs:complexType>
  </xs:element>
</xs:schema>`
		require.NoError(t, validateOC(t, schema,
			`<doc><b/><a/><extra xmlns="http://open.com/"/></doc>`))
	})

	t.Run("interleave in all", func(t *testing.T) {
		t.Parallel()
		schema := head + `
  <xs:defaultOpenContent mode="interleave">
    <xs:any namespace="http://open.com/" processContents="lax"/>
  </xs:defaultOpenContent>
  <xs:element name="doc">
    <xs:complexType><xs:all>
      <xs:element name="a"/><xs:element name="b"/>
    </xs:all></xs:complexType>
  </xs:element>
</xs:schema>`
		require.NoError(t, validateOC(t, schema,
			`<doc><extra xmlns="http://open.com/"/><b/><a/></doc>`))
	})
}

// TestOpenContent_ExtensionInherits verifies an extension with no open content
// inherits the base type's open content, and a base interleave open content may
// not be relaxed to suffix by an extension (schema invalid).
func TestOpenContent_ExtensionInherits(t *testing.T) {
	t.Parallel()
	const head = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">`

	t.Run("derived inherits base open content", func(t *testing.T) {
		t.Parallel()
		schema := head + `
  <xs:complexType name="B">
    <xs:openContent mode="suffix"><xs:any namespace="http://open.com/" processContents="lax"/></xs:openContent>
    <xs:sequence><xs:element name="a"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="R">
    <xs:complexContent><xs:extension base="B">
      <xs:sequence><xs:element name="d" minOccurs="0"/></xs:sequence>
    </xs:extension></xs:complexContent>
  </xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		require.NoError(t, validateOC(t, schema,
			`<doc><a/><d/><extra xmlns="http://open.com/"/></doc>`))
	})

	t.Run("extension may not relax interleave to suffix", func(t *testing.T) {
		t.Parallel()
		schema := head + `
  <xs:complexType name="B">
    <xs:openContent mode="interleave"><xs:any namespace="http://open.com/" processContents="lax"/></xs:openContent>
    <xs:sequence><xs:element name="a"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="R">
    <xs:complexContent><xs:extension base="B">
      <xs:openContent mode="suffix"><xs:any namespace="http://open.com/" processContents="lax"/></xs:openContent>
      <xs:sequence><xs:element name="d" minOccurs="0"/></xs:sequence>
    </xs:extension></xs:complexContent>
  </xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.Error(t, cerr, "suffix extension of interleave base must be rejected")
	})
}

// TestOpenContent_RestrictionValidity covers §3.4.6.4 restriction-derivation
// checks: a restriction may not ADD open content the base lacks, but when the
// restriction's content model is empty the mode/wildcard comparison is waived.
func TestOpenContent_RestrictionValidity(t *testing.T) {
	t.Parallel()
	const head = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">`

	t.Run("adding open content to a closed base is invalid", func(t *testing.T) {
		t.Parallel()
		schema := head + `
  <xs:complexType name="B"><xs:sequence>
    <xs:element name="a"/><xs:element name="b" minOccurs="0"/>
  </xs:sequence></xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="B">
    <xs:openContent mode="suffix"><xs:any namespace="http://open.com/" processContents="lax"/></xs:openContent>
    <xs:sequence><xs:element name="a"/></xs:sequence>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.Error(t, cerr)
	})

	t.Run("interleave restricting suffix is valid when content empty", func(t *testing.T) {
		t.Parallel()
		schema := head + `
  <xs:complexType name="B" mixed="true">
    <xs:openContent mode="suffix"><xs:any namespace="http://open.com/" processContents="lax"/></xs:openContent>
    <xs:sequence><xs:element name="a" minOccurs="0" maxOccurs="unbounded"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent mixed="1"><xs:restriction base="B">
    <xs:openContent mode="interleave"><xs:any namespace="http://open.com/" processContents="lax"/></xs:openContent>
    <xs:sequence/>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.NoError(t, cerr)
		require.NoError(t, validateOC(t, schema,
			`<doc>text<extra xmlns="http://open.com/"/>more</doc>`))
	})
}

// TestOpenContent_SchemaRepresentationErrors covers the local schema-validity
// constraints on <xs:openContent>/<xs:defaultOpenContent>.
func TestOpenContent_SchemaRepresentationErrors(t *testing.T) {
	t.Parallel()
	const head = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">`

	cases := map[string]string{
		"mode none with wildcard": head + `
  <xs:element name="doc"><xs:complexType>
    <xs:openContent mode="none"><xs:any namespace="http://open.com/" processContents="lax"/></xs:openContent>
    <xs:sequence><xs:element name="a"/></xs:sequence>
  </xs:complexType></xs:element></xs:schema>`,
		"maxOccurs on open content any": head + `
  <xs:element name="doc"><xs:complexType>
    <xs:openContent><xs:any namespace="http://open.com/" processContents="lax" maxOccurs="unbounded"/></xs:openContent>
    <xs:sequence><xs:element name="a"/></xs:sequence>
  </xs:complexType></xs:element></xs:schema>`,
		"two annotations in open content": head + `
  <xs:element name="doc"><xs:complexType>
    <xs:openContent mode="suffix">
      <xs:annotation><xs:documentation>one</xs:documentation></xs:annotation>
      <xs:annotation><xs:documentation>two</xs:documentation></xs:annotation>
      <xs:any namespace="http://open.com/" processContents="lax"/>
    </xs:openContent>
    <xs:sequence><xs:element name="a"/></xs:sequence>
  </xs:complexType></xs:element></xs:schema>`,
		"default open content after a declaration": head + `
  <xs:element name="root" type="xs:string"/>
  <xs:defaultOpenContent><xs:any/></xs:defaultOpenContent>
</xs:schema>`,
	}
	for name, schema := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, _, cerr := compileV11(t, schema)
			require.Error(t, cerr, "schema must be rejected")
		})
	}
}

// TestComplexContent_MixedHandling covers <xs:complexContent mixed="..."> and the
// extension mixed/element-only consistency rule (XSD 1.1).
func TestComplexContent_MixedHandling(t *testing.T) {
	t.Parallel()
	const head = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="a" xmlns:a="a">`

	t.Run("complexType mixed conflicts with complexContent mixed", func(t *testing.T) {
		t.Parallel()
		schema := head + `
  <xs:element name="root"><xs:complexType mixed="true">
    <xs:complexContent mixed="false"><xs:extension base="a:bele">
      <xs:sequence><xs:element name="e2" minOccurs="0"/></xs:sequence>
    </xs:extension></xs:complexContent>
  </xs:complexType></xs:element>
  <xs:complexType name="bele" mixed="true"><xs:sequence><xs:element name="e1" minOccurs="0"/></xs:sequence></xs:complexType>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.Error(t, cerr)
	})

	t.Run("mixed extension of element-only base is invalid", func(t *testing.T) {
		t.Parallel()
		schema := head + `
  <xs:element name="root"><xs:complexType mixed="true">
    <xs:complexContent><xs:extension base="a:bele">
      <xs:sequence><xs:element name="e2" minOccurs="0"/></xs:sequence>
    </xs:extension></xs:complexContent>
  </xs:complexType></xs:element>
  <xs:complexType name="bele"><xs:sequence><xs:element name="e1" minOccurs="0"/></xs:sequence></xs:complexType>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.Error(t, cerr)
	})
}

// TestDefaultOpenContent_XSD10Ignored confirms <xs:defaultOpenContent> has no
// effect under the default XSD 1.0 semantics: a non-declared child is rejected.
func TestDefaultOpenContent_XSD10Ignored(t *testing.T) {
	t.Parallel()
	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:defaultOpenContent mode="suffix">
    <xs:any namespace="http://open.com/" processContents="lax"/>
  </xs:defaultOpenContent>
  <xs:element name="doc">
    <xs:complexType><xs:sequence><xs:element name="a"/></xs:sequence></xs:complexType>
  </xs:element>
</xs:schema>`
	s10, err := compileV10(t, schema)
	require.NoError(t, err)
	require.NotNil(t, s10)
	idoc, perr := helium.NewParser().Parse(t.Context(),
		[]byte(`<doc><a/><extra xmlns="http://open.com/"/></doc>`))
	require.NoError(t, perr)
	require.Error(t, xsd.NewValidator(s10).Validate(t.Context(), idoc),
		"XSD 1.0 must ignore defaultOpenContent and reject the extra child")
}
