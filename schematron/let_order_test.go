package schematron_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/schematron"
	"github.com/stretchr/testify/require"
)

// TestRuleLetDocumentOrder verifies that <let> bindings inside a rule are
// evaluated in document order, so a later let may depend on an earlier one.
func TestRuleLetDocumentOrder(t *testing.T) {
	const schemaSrc = `<?xml version="1.0" encoding="UTF-8"?>
<schema xmlns="http://purl.oclc.org/dsdl/schematron">
  <pattern>
    <rule context="/root">
      <let name="a" value="'ok'"/>
      <let name="b" value="$a"/>
      <assert test="$b='ok'">b must equal ok</assert>
    </rule>
  </pattern>
</schema>`

	const instanceSrc = `<?xml version="1.0" encoding="UTF-8"?><root/>`

	ctx := t.Context()

	schemaDoc, err := helium.NewParser().Parse(ctx, []byte(schemaSrc))
	require.NoError(t, err, "parse schema")

	schema, err := schematron.NewCompiler().Compile(ctx, schemaDoc)
	require.NoError(t, err, "compile schema")

	instDoc, err := helium.NewParser().Parse(ctx, []byte(instanceSrc))
	require.NoError(t, err, "parse instance")

	err = schematron.NewValidator(schema).Validate(ctx, instDoc)
	require.NoError(t, err, "dependent let b should resolve to a's value, so the assert must pass")
}
