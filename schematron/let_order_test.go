package schematron_test

import (
	"context"
	"strings"
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

// TestRuleLetEvalErrorFailsValidation verifies that when a matched rule's
// <let> expression cannot be evaluated, validation fails (returns
// ErrValidationFailed) even though the rule's assert would otherwise pass.
// A failing let must not silently let Validate report a "valid" result.
func TestRuleLetEvalErrorFailsValidation(t *testing.T) {
	const schemaSrc = `<?xml version="1.0" encoding="UTF-8"?>
<schema xmlns="http://purl.oclc.org/dsdl/schematron">
  <pattern>
    <rule context="/root">
      <let name="x" value="not-a-function()"/>
      <assert test="true()">always ok</assert>
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

	// With the default (nil) handler the validation must still fail: a broken
	// let must not produce a silent "valid" result.
	err = schematron.NewValidator(schema).Validate(ctx, instDoc)
	require.ErrorIs(t, err, schematron.ErrValidationFailed, "a matched rule whose let fails to evaluate must fail validation even with the default handler")

	// And the diagnostic must be surfaced to a handler when one is supplied.
	var captured []string
	handler := captureHandler{out: &captured}
	err = schematron.NewValidator(schema).ErrorHandler(handler).Validate(ctx, instDoc)
	require.ErrorIs(t, err, schematron.ErrValidationFailed, "validation must still fail with a handler attached")

	joined := strings.Join(captured, "\n")
	require.Contains(t, joined, "XPath error", "the let evaluation error must be surfaced to the handler")
}

// captureHandler records the string form of every error it receives.
type captureHandler struct {
	out *[]string
}

func (h captureHandler) Handle(_ context.Context, err error) {
	*h.out = append(*h.out, err.Error())
}
