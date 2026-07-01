package xsd_test

import (
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// compileVer compiles schemaXML at the given version and returns the error.
func compileVer(t *testing.T, schemaXML string, v xsd.Version) (*xsd.Schema, error) {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	return xsd.NewCompiler().Version(v).Compile(t.Context(), doc)
}

// TestXSIAttributeReferenceRequired covers saxon Complex.testSet complex009 /
// complex010: an XSD 1.1 schema may reference an xsi: processor attribute as a
// (required) attribute use. A present xsi: attribute must satisfy that use
// instead of being skipped as a special attribute and reported missing.
func TestXSIAttributeReferenceRequired(t *testing.T) {
	t.Parallel()

	const xsiTypeSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <xs:element name="root" type="B"/>
  <xs:complexType name="B">
    <xs:sequence>
      <xs:element name="e" minOccurs="0" maxOccurs="5"/>
    </xs:sequence>
    <xs:attribute ref="xsi:type" use="required"/>
  </xs:complexType>
</xs:schema>`

	schema, cerr := compileVer(t, xsiTypeSchema, xsd.Version11)
	require.NoError(t, cerr, "schema must compile")

	validate := func(t *testing.T, instanceXML string) error {
		t.Helper()
		idoc, err := helium.NewParser().Parse(t.Context(), []byte(instanceXML))
		require.NoError(t, err)
		return xsd.NewValidator(schema).Validate(t.Context(), idoc)
	}

	// A present, valid xsi:type satisfies the required use.
	require.NoError(t, validate(t, `<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="B"><e/></root>`))
	// Missing xsi:type is rejected (the required use is unmet).
	require.Error(t, validate(t, `<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"><e/></root>`))
	// An EMPTY xsi:type is not a valid xs:QName, so it does not satisfy the
	// required use (false-accept guard).
	require.Error(t, validate(t, `<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type=""><e/></root>`))
}

// TestXSIAttributeReferenceValueValidation verifies that a DECLARED xsi:
// attribute use validates its value against the attribute's built-in type, so
// an empty/malformed value does not silently satisfy a required use.
func TestXSIAttributeReferenceValueValidation(t *testing.T) {
	t.Parallel()

	compileType := func(t *testing.T, ref string) *xsd.Schema {
		t.Helper()
		// root is nillable so a valid xsi:nil="true" is not rejected by cvc-elt.3.1
		// (this test isolates the declared-xsi VALUE validation).
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <xs:element name="root" type="B" nillable="true"/>
  <xs:complexType name="B">
    <xs:sequence/>
    <xs:attribute ref="` + ref + `" use="required"/>
  </xs:complexType>
</xs:schema>`
		s, cerr := compileVer(t, schema, xsd.Version11)
		require.NoError(t, cerr, "schema must compile")
		return s
	}
	validate := func(t *testing.T, s *xsd.Schema, instanceXML string) error {
		t.Helper()
		idoc, err := helium.NewParser().Parse(t.Context(), []byte(instanceXML))
		require.NoError(t, err)
		return xsd.NewValidator(s).Validate(t.Context(), idoc)
	}

	const xsiNS = `xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"`

	t.Run("xsi:type", func(t *testing.T) {
		s := compileType(t, "xsi:type")
		// Valid namespace-resolvable QName.
		require.NoError(t, validate(t, s, `<root `+xsiNS+` xsi:type="B"/>`))
		// Empty value → invalid QName → reject.
		require.Error(t, validate(t, s, `<root `+xsiNS+` xsi:type=""/>`))
		// Malformed QName → reject.
		require.Error(t, validate(t, s, `<root `+xsiNS+` xsi:type="a:b:c"/>`))
	})

	t.Run("xsi:nil", func(t *testing.T) {
		s := compileType(t, "xsi:nil")
		// Valid boolean.
		require.NoError(t, validate(t, s, `<root `+xsiNS+` xsi:nil="true"/>`))
		// Non-boolean → reject.
		require.Error(t, validate(t, s, `<root `+xsiNS+` xsi:nil="notbool"/>`))
		// Empty → reject.
		require.Error(t, validate(t, s, `<root `+xsiNS+` xsi:nil=""/>`))
	})

	t.Run("xsi:schemaLocation", func(t *testing.T) {
		s := compileType(t, "xsi:schemaLocation")
		// Valid: an even list of anyURI pairs, space-separated (XSD whitespace).
		require.NoError(t, validate(t, s, `<root `+xsiNS+` xsi:schemaLocation="urn:a loc.xsd"/>`))
		// NBSP (U+00A0) is NOT XSD list whitespace, so "urn:a loc.xsd" is ONE
		// token (odd count) → reject. strings.Fields would wrongly split it in two.
		require.Error(t, validate(t, s, "<root "+xsiNS+" xsi:schemaLocation=\"urn:a loc.xsd\"/>"))
		// Odd token count → reject.
		require.Error(t, validate(t, s, `<root `+xsiNS+` xsi:schemaLocation="urn:a"/>`))
	})

	// F3: a declared ref to a non-standard xsi: local name (xsi:foo) is NOT
	// specially accepted — it stays skipped as a special attribute, so a required
	// use of it is never satisfied and the instance is rejected.
	t.Run("xsi:foo not specially accepted", func(t *testing.T) {
		s := compileType(t, "xsi:foo")
		require.Error(t, validate(t, s, `<root `+xsiNS+` xsi:foo="x"/>`),
			"a required ref=xsi:foo use must not be satisfied by a present xsi:foo")
	})
}

// TestXSIAttributeReferenceFixedDefault verifies that a declared xsi: processor
// attribute use is associated with its built-in type so that fixed/default
// constraints are (1) validated for validity at compile time and (2) compared
// in VALUE space at runtime — not by raw string equality against a nil type.
func TestXSIAttributeReferenceFixedDefault(t *testing.T) {
	t.Parallel()

	validateSchema := func(t *testing.T, schema, instance string) (compileErr, validateErr error) {
		t.Helper()
		s, cerr := compileVer(t, schema, xsd.Version11)
		if cerr != nil {
			return cerr, nil
		}
		idoc, perr := helium.NewParser().Parse(t.Context(), []byte(instance))
		require.NoError(t, perr)
		return nil, xsd.NewValidator(s).Validate(t.Context(), idoc)
	}

	const xsiNS = `xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"`

	t.Run("xsi:nil fixed value-space match", func(t *testing.T) {
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <xs:element name="root" type="B" nillable="true"/>
  <xs:complexType name="B">
    <xs:sequence/>
    <xs:attribute ref="xsi:nil" fixed="true"/>
  </xs:complexType>
</xs:schema>`
		// xsi:nil="1" equals fixed "true" in the xs:boolean value space; without a
		// built-in type the raw-string compare "1" != "true" would false-reject.
		cerr, verr := validateSchema(t, schema, `<root `+xsiNS+` xsi:nil="1"/>`)
		require.NoError(t, cerr)
		require.NoError(t, verr, `xsi:nil="1" must satisfy fixed="true" by value space`)
	})

	t.Run("xsi:nil fixed invalid → schema error", func(t *testing.T) {
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <xs:element name="root" type="B"/>
  <xs:complexType name="B">
    <xs:sequence/>
    <xs:attribute ref="xsi:nil" fixed="notbool"/>
  </xs:complexType>
</xs:schema>`
		_, cerr := compileVer(t, schema, xsd.Version11)
		require.Error(t, cerr, `fixed="notbool" is not a valid xs:boolean`)
	})

	t.Run("xsi:nil default invalid → schema error", func(t *testing.T) {
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <xs:element name="root" type="B"/>
  <xs:complexType name="B">
    <xs:sequence/>
    <xs:attribute ref="xsi:nil" default="notbool"/>
  </xs:complexType>
</xs:schema>`
		_, cerr := compileVer(t, schema, xsd.Version11)
		require.Error(t, cerr, `default="notbool" is not a valid xs:boolean`)
	})

	t.Run("xsi:type fixed value-space match (different prefix, same ns)", func(t *testing.T) {
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xmlns:t="urn:t" targetNamespace="urn:t">
  <xs:element name="root" type="t:B"/>
  <xs:complexType name="B">
    <xs:sequence/>
    <xs:attribute ref="xsi:type" use="required" fixed="t:B"/>
  </xs:complexType>
</xs:schema>`
		// A different prefix u (same urn:t) → QName value-space equal to fixed t:B.
		cerr, verr := validateSchema(t, schema,
			`<t:root xmlns:t="urn:t" xmlns:u="urn:t" `+xsiNS+` xsi:type="u:B"/>`)
		require.NoError(t, cerr)
		require.NoError(t, verr, `xsi:type="u:B" must satisfy fixed="t:B" by QName value space`)
	})

	schemaLoc := func(constraint string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <xs:element name="root" type="B"/>
  <xs:complexType name="B">
    <xs:sequence/>
    <xs:attribute ref="xsi:schemaLocation" ` + constraint + `/>
  </xs:complexType>
</xs:schema>`
	}

	t.Run("xsi:schemaLocation default odd → schema error", func(t *testing.T) {
		// "urn:a" is one token (odd) — not an even (namespace, location) pair list.
		_, cerr := compileVer(t, schemaLoc(`default="urn:a"`), xsd.Version11)
		require.Error(t, cerr, `default="urn:a" is not a valid xsi:schemaLocation (odd token count)`)
	})

	t.Run("xsi:schemaLocation default empty → schema error", func(t *testing.T) {
		// An empty value is zero tokens — not a non-empty even pair list.
		_, cerr := compileVer(t, schemaLoc(`default=""`), xsd.Version11)
		require.Error(t, cerr, `default="" is not a valid xsi:schemaLocation (no tokens)`)
	})

	t.Run("xsi:schemaLocation default even valid → compiles", func(t *testing.T) {
		_, cerr := compileVer(t, schemaLoc(`default="urn:a loc.xsd"`), xsd.Version11)
		require.NoError(t, cerr, `default="urn:a loc.xsd" is a valid even anyURI pair list`)
	})

	t.Run("xsi:schemaLocation fixed value-space match", func(t *testing.T) {
		// The instance value differs only by whitespace runs; it is value-space
		// equal to the fixed value (same token list after collapse + tokenize).
		cerr, verr := validateSchema(t, schemaLoc(`fixed="urn:a loc.xsd"`),
			"<root "+xsiNS+" xsi:schemaLocation=\"  urn:a   loc.xsd  \"/>")
		require.NoError(t, cerr)
		require.NoError(t, verr, `whitespace-equivalent xsi:schemaLocation must satisfy the fixed value`)
	})
}

// TestSchemaComponentIDIgnoresAnnotationPayload verifies that the @id walk does
// NOT descend into xs:appinfo / xs:documentation payload (arbitrary content,
// not schema components), while the xs:annotation element's OWN @id still
// participates in xs:ID uniqueness.
func TestSchemaComponentIDIgnoresAnnotationPayload(t *testing.T) {
	t.Parallel()

	// A duplicate `id` on an element embedded INSIDE xs:appinfo must not reject
	// the schema (it collides with the real schema-component id "dup", but
	// annotation payload is not a schema component).
	const embeddedDup = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="a" id="dup">
    <xs:annotation>
      <xs:appinfo>
        <xs:element name="embedded" id="dup"/>
      </xs:appinfo>
      <xs:documentation>
        <thing id="dup"/>
      </xs:documentation>
    </xs:annotation>
  </xs:element>
</xs:schema>`
	_, err := compileVer(t, embeddedDup, xsd.Version11)
	require.NoError(t, err, "duplicate id inside annotation payload must NOT reject the schema")

	// The xs:annotation element's OWN @id still participates: two annotations
	// sharing an @id is a duplicate xs:ID.
	const annDup = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:annotation id="annA"/>
  <xs:annotation id="annA"/>
</xs:schema>`
	_, err = compileVer(t, annDup, xsd.Version11)
	require.Error(t, err, "duplicate @id on xs:annotation elements must reject")
}

// TestSchemaComponentIDValidity covers saxon Open.testSet open038 / open039: a
// schema-component @id must be a valid xs:ID (NCName after whitespace collapse)
// and unique within the schema document. This is a version-independent XSD rule,
// enforced in both 1.0 and 1.1.
func TestSchemaComponentIDValidity(t *testing.T) {
	t.Parallel()

	// open038: two ids that collapse to the same NCName ("open001").
	const dupID = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="a" id="open001"/>
  <xs:element name="b" id=" open001 "/>
</xs:schema>`

	// open039: an id that is not a valid NCName ("open001/2").
	const badID = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="a" id="open001"/>
  <xs:element name="b" id="open001/2"/>
</xs:schema>`

	_, err := compileVer(t, dupID, xsd.Version11)
	require.Error(t, err, "duplicate xs:ID must fail in 1.1")
	_, err = compileVer(t, badID, xsd.Version11)
	require.Error(t, err, "invalid NCName id must fail in 1.1")

	// The @id xs:ID validity/uniqueness rule is version-independent, so 1.0
	// rejects the same schemas.
	_, err = compileVer(t, dupID, xsd.Version10)
	require.Error(t, err, "duplicate xs:ID must fail in 1.0")
	_, err = compileVer(t, badID, xsd.Version10)
	require.Error(t, err, "invalid NCName id must fail in 1.0")
}

const (
	residueMainXSD = "main.xsd"
	residueIncXSD  = "inc.xsd"
	residueBXSD    = "b.xsd"
)

func residueMapFile(body string) *fstest.MapFile {
	return &fstest.MapFile{Data: []byte(body)}
}

// compileFSVer compiles residueMainXSD from fsys at the given version.
func compileFSVer(t *testing.T, fsys fstest.MapFS, v xsd.Version) error {
	t.Helper()
	data, err := fsys.ReadFile(residueMainXSD)
	require.NoError(t, err)
	doc, err := helium.NewParser().Parse(t.Context(), data)
	require.NoError(t, err)
	_, cerr := xsd.NewCompiler().Version(v).Label(residueMainXSD).FS(fsys).Compile(t.Context(), doc)
	return cerr
}

// TestSchemaComponentIDValidityNestedDocuments verifies that @id xs:ID
// uniqueness/NCName validity is enforced PER nested schema document
// (xs:include/xs:import), not only the entry document, and that the scope is
// per-document — the same @id value may recur ACROSS documents.
func TestSchemaComponentIDValidityNestedDocuments(t *testing.T) {
	t.Parallel()

	// Duplicate @id WITHIN an included document → schema error.
	t.Run("dup-in-include", func(t *testing.T) {
		fsys := fstest.MapFS{
			residueMainXSD: residueMapFile(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:include schemaLocation="inc.xsd"/>
  <xs:element name="root" type="xs:string"/>
</xs:schema>`),
			residueIncXSD: residueMapFile(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="a" id="dupe"/>
  <xs:element name="b" id="dupe"/>
</xs:schema>`),
		}
		require.Error(t, compileFSVer(t, fsys, xsd.Version11), "duplicate @id in included doc must fail in 1.1")
		require.Error(t, compileFSVer(t, fsys, xsd.Version10), "duplicate @id in included doc must fail in 1.0")
	})

	// Invalid (non-NCName) @id in an IMPORTED document → schema error.
	t.Run("bad-in-import", func(t *testing.T) {
		fsys := fstest.MapFS{
			residueMainXSD: residueMapFile(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:b="urn:b">
  <xs:import namespace="urn:b" schemaLocation="b.xsd"/>
  <xs:element name="root" type="xs:string"/>
</xs:schema>`),
			residueBXSD: residueMapFile(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:b">
  <xs:element name="x" id="bad/name"/>
</xs:schema>`),
		}
		require.Error(t, compileFSVer(t, fsys, xsd.Version11), "invalid @id in imported doc must fail in 1.1")
		require.Error(t, compileFSVer(t, fsys, xsd.Version10), "invalid @id in imported doc must fail in 1.0")
	})

	// The SAME @id value in two DIFFERENT documents is fine (per-document scope).
	t.Run("same-id-across-docs", func(t *testing.T) {
		fsys := fstest.MapFS{
			residueMainXSD: residueMapFile(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:include schemaLocation="inc.xsd"/>
  <xs:element name="root" type="xs:string" id="shared"/>
</xs:schema>`),
			residueIncXSD: residueMapFile(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="a" id="shared"/>
</xs:schema>`),
		}
		require.NoError(t, compileFSVer(t, fsys, xsd.Version11), "same @id across documents is per-document valid")
	})
}

// TestXSIAttributeReferenceProhibited verifies that a DECLARED xsi: attribute
// use with use="prohibited" participates in validation: a present xsi: attribute
// matching it is rejected (not skipped as a special attribute), while its absence
// is valid.
func TestXSIAttributeReferenceProhibited(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <xs:element name="root" type="B"/>
  <xs:complexType name="B">
    <xs:sequence>
      <xs:element name="e" minOccurs="0"/>
    </xs:sequence>
    <xs:attribute ref="xsi:type" use="prohibited"/>
  </xs:complexType>
</xs:schema>`

	compiled, cerr := compileVer(t, schema, xsd.Version11)
	require.NoError(t, cerr, "schema must compile")

	validate := func(t *testing.T, instanceXML string) error {
		t.Helper()
		idoc, err := helium.NewParser().Parse(t.Context(), []byte(instanceXML))
		require.NoError(t, err)
		return xsd.NewValidator(compiled).Validate(t.Context(), idoc)
	}

	// A present xsi:type is rejected by the prohibited use.
	require.Error(t, validate(t, `<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="B"><e/></root>`))
	// Absent xsi:type is valid.
	require.NoError(t, validate(t, `<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"><e/></root>`))
}
