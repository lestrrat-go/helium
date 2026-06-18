package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestValidateAllParticleQName verifies that an xs:all particle declared
// without a namespace (unqualified local form, elementFormDefault="unqualified")
// matches on the child element's full expanded name. A namespaced child must NOT
// be accepted by an unqualified <all> particle, while the correctly unqualified
// child is accepted.
func TestValidateAllParticleQName(t *testing.T) {
	t.Parallel()

	// targetNamespace with unqualified local elements: the wrapper is
	// {urn:right}root, but its <all> children "a"/"b" are unqualified ({}a, {}b).
	const schemaSrc = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="urn:right"
           xmlns="urn:right"
           elementFormDefault="unqualified">
  <xs:element name="root">
    <xs:complexType>
      <xs:all>
        <xs:element name="a" type="xs:string"/>
        <xs:element name="b" type="xs:string"/>
      </xs:all>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	compile := func(t *testing.T) *xsd.Schema {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaSrc))
		require.NoError(t, err)
		schema, err := xsd.NewCompiler().Compile(t.Context(), doc)
		require.NoError(t, err)
		return schema
	}

	t.Run("unqualified children accepted", func(t *testing.T) {
		t.Parallel()
		schema := compile(t)
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?>
<root xmlns="urn:right"><a xmlns="">x</a><b xmlns="">y</b></root>`))
		require.NoError(t, err)
		require.NoError(t, xsd.NewValidator(schema).Validate(t.Context(), doc))
	})

	t.Run("namespaced child rejected by unqualified particle", func(t *testing.T) {
		t.Parallel()
		schema := compile(t)
		// Here <a>/<b> inherit the urn:right default namespace, so they are
		// {urn:right}a / {urn:right}b — they must NOT match the unqualified
		// {}a / {}b particles in the <all> group.
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?>
<root xmlns="urn:right"><a>x</a><b>y</b></root>`))
		require.NoError(t, err)

		var errs string
		err = validateWithOutput(t, xsd.NewValidator(schema), doc, &errs)
		require.Error(t, err)
		require.Contains(t, errs, "not expected")
	})
}
