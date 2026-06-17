package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestValidateRootNamespace verifies that the root-element global-declaration
// lookup matches on the element's full expanded name. A no-target-namespace
// schema must NOT accept an instance root whose element carries a non-empty
// namespace, even when an unqualified declaration shares the local name.
// libxml2 (xmllint --schema) rejects {urn:wrong}foo against a no-namespace
// schema declaring {}foo.
func TestValidateRootNamespace(t *testing.T) {
	t.Parallel()

	const noNSSchema = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="foo" type="xs:string"/>
</xs:schema>`

	const targetNSSchema = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="urn:right"
           xmlns="urn:right"
           elementFormDefault="qualified">
  <xs:element name="foo" type="xs:string"/>
</xs:schema>`

	compile := func(t *testing.T, src string) *xsd.Schema {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, err)
		schema, err := xsd.NewCompiler().Compile(t.Context(), doc)
		require.NoError(t, err)
		return schema
	}

	t.Run("no-namespace instance against no-namespace schema validates", func(t *testing.T) {
		t.Parallel()
		schema := compile(t, noNSSchema)
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?>
<foo>x</foo>`))
		require.NoError(t, err)
		require.NoError(t, xsd.NewValidator(schema).Validate(t.Context(), doc))
	})

	t.Run("wrong-namespace instance against no-namespace schema fails", func(t *testing.T) {
		t.Parallel()
		schema := compile(t, noNSSchema)
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?>
<foo xmlns="urn:wrong">x</foo>`))
		require.NoError(t, err)

		var errs string
		err = validateWithOutput(t, xsd.NewValidator(schema), doc, &errs)
		require.Error(t, err)
		require.Contains(t, errs, "No matching global declaration")
	})

	t.Run("correct-namespace instance against target-namespace schema validates", func(t *testing.T) {
		t.Parallel()
		schema := compile(t, targetNSSchema)
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?>
<foo xmlns="urn:right">x</foo>`))
		require.NoError(t, err)
		require.NoError(t, xsd.NewValidator(schema).Validate(t.Context(), doc))
	})
}
