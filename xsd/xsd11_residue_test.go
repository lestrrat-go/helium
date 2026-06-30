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
}

// TestSchemaComponentIDValidity covers saxon Open.testSet open038 / open039: a
// schema-component @id must be a valid xs:ID (NCName after whitespace collapse)
// and unique within the schema document. Enforced in XSD 1.1; XSD 1.0 keeps the
// historical lenient behavior (byte-identical goldens).
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

	// XSD 1.0 stays lenient (no @id enforcement) to preserve byte-identity.
	_, err = compileVer(t, dupID, xsd.Version10)
	require.NoError(t, err, "duplicate xs:ID is tolerated in 1.0")
	_, err = compileVer(t, badID, xsd.Version10)
	require.NoError(t, err, "invalid NCName id is tolerated in 1.0")
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
		require.NoError(t, compileFSVer(t, fsys, xsd.Version10), "1.0 stays lenient")
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
		require.NoError(t, compileFSVer(t, fsys, xsd.Version10), "1.0 stays lenient")
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
