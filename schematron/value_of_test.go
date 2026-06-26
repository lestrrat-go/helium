package schematron_test

import (
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/schematron"
	"github.com/stretchr/testify/require"
)

// TestValueOfNodeSetStringValue verifies that a <value-of> over a node-set
// emits the XPath 1.0 string-value (the text content of the first node in
// document order) rather than the node's element name.
func TestValueOfNodeSetStringValue(t *testing.T) {
	const schemaSrc = `<?xml version="1.0" encoding="UTF-8"?>
<schema xmlns="http://purl.oclc.org/dsdl/schematron">
  <pattern>
    <rule context="/root">
      <report test="child">found <value-of select="child"/></report>
    </rule>
  </pattern>
</schema>`

	const instanceSrc = `<?xml version="1.0" encoding="UTF-8"?><root><child>TEXT</child></root>`

	ctx := t.Context()

	schemaDoc, err := helium.NewParser().Parse(ctx, []byte(schemaSrc))
	require.NoError(t, err, "parse schema")

	schema, err := schematron.NewCompiler().Compile(ctx, schemaDoc)
	require.NoError(t, err, "compile schema")

	instDoc, err := helium.NewParser().Parse(ctx, []byte(instanceSrc))
	require.NoError(t, err, "parse instance")

	var captured []string
	handler := captureHandler{out: &captured}
	_ = schematron.NewValidator(schema).ErrorHandler(handler).Validate(ctx, instDoc)

	joined := strings.Join(captured, "\n")
	require.Contains(t, joined, "found TEXT", "value-of must emit the node string-value (TEXT), not the element name (child)")
	require.NotContains(t, joined, "found child", "value-of must not emit the element name")
}
