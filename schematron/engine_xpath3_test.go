package schematron_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/schematron"
	"github.com/stretchr/testify/require"
)

const xpath3Instance = `<?xml version="1.0" encoding="UTF-8"?>
<root code="AB-12">
  <item>one</item>
  <item>two</item>
  <item>three</item>
</root>`

// validateWithSchema compiles src, validates xpath3Instance against it, and
// returns everything the schema reported together with the validation error.
func validateWithSchema(t *testing.T, src string) ([]error, error) {
	t.Helper()

	schemaDoc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err, "parse schema")

	schema, err := schematron.NewCompiler().Compile(t.Context(), schemaDoc)
	require.NoError(t, err, "compile schema")

	instDoc, err := helium.NewParser().Parse(t.Context(), []byte(xpath3Instance))
	require.NoError(t, err, "parse instance")

	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	valErr := schematron.NewValidator(schema).
		Label("instance.xml").
		ErrorHandler(collector).
		Validate(t.Context(), instDoc)
	_ = collector.Close()

	return collector.Errors(), valErr
}

// schemaSrc wraps one rule body in a schema using the given binding.
func schemaSrc(binding, ruleBody string) string {
	attr := ""
	if binding != "" {
		attr = ` queryBinding="` + binding + `"`
	}
	return `<?xml version="1.0" encoding="UTF-8"?>
<schema xmlns="http://purl.oclc.org/dsdl/schematron"` + attr + `>
  <pattern>
    <rule context="root">
` + ruleBody + `
    </rule>
  </pattern>
</schema>`
}

// TestXPath3BindingFunctions checks that a function XPath 1.0 does not have
// resolves under the XPath 3.1 binding, and fails under the default one.
func TestXPath3BindingFunctions(t *testing.T) {
	const ruleBody = `      <assert test="matches(@code, '^[A-Z]{2}-[0-9]+$')">code must be well formed</assert>`

	t.Run("xslt3", func(t *testing.T) {
		reported, valErr := validateWithSchema(t, schemaSrc("xslt3", ruleBody))
		require.NoError(t, valErr, "fn:matches must resolve under XPath 3.1")
		require.Empty(t, reported, "no errors reported")
	})

	t.Run("default", func(t *testing.T) {
		reported, valErr := validateWithSchema(t, schemaSrc("", ruleBody))
		require.Error(t, valErr, "fn:matches does not exist in XPath 1.0")
		require.NotEmpty(t, reported, "the unregistered function is reported")
	})
}

// TestXPath3BindingValueOf checks the <value-of> conversion difference: XSLT
// 2.0 and later join every item with a space, XPath 1.0 keeps the first node.
func TestXPath3BindingValueOf(t *testing.T) {
	const ruleBody = `      <assert test="false()">items: <value-of select="item"/></assert>`

	t.Run("xslt3", func(t *testing.T) {
		reported, valErr := validateWithSchema(t, schemaSrc("xslt3", ruleBody))
		require.ErrorIs(t, valErr, schematron.ErrValidationFailed)
		require.Len(t, reported, 1)
		require.Contains(t, reported[0].Error(), "items: one two three")
	})

	t.Run("default", func(t *testing.T) {
		reported, valErr := validateWithSchema(t, schemaSrc("", ruleBody))
		require.ErrorIs(t, valErr, schematron.ErrValidationFailed)
		require.Len(t, reported, 1)
		require.Contains(t, reported[0].Error(), "items: one")
		require.NotContains(t, reported[0].Error(), "two")
	})
}

// TestXPath3BindingEffectiveBooleanValue checks that a test whose result has
// no effective boolean value is reported and treated as false.
func TestXPath3BindingEffectiveBooleanValue(t *testing.T) {
	const ruleBody = `      <assert test="string(item[1])">the first item has text</assert>`

	reported, valErr := validateWithSchema(t, schemaSrc("xslt3", `      <assert test="(1, 2)">two numbers have no boolean value</assert>`))
	require.ErrorIs(t, valErr, schematron.ErrValidationFailed, "the assert fires because the test is not true")
	require.NotEmpty(t, reported)
	require.Contains(t, reported[0].Error(), "FORG0006", "the XPath error code reaches the handler")

	// A single-item test still converts, so the sibling case passes.
	_, valErr = validateWithSchema(t, schemaSrc("xslt3", ruleBody))
	require.NoError(t, valErr, "a single string has an effective boolean value")
}

// TestXPath3BindingLet checks that a <let> binds a whole sequence, which
// XPath 1.0 cannot express.
func TestXPath3BindingLet(t *testing.T) {
	const ruleBody = `      <let name="items" value="item"/>
      <let name="total" value="count($items)"/>
      <assert test="$total = 3">expected three items</assert>
      <assert test="string-join($items, '|') = 'one|two|three'">items must join in order</assert>`

	reported, valErr := validateWithSchema(t, schemaSrc("xpath3", ruleBody))
	require.NoError(t, valErr, "sequence-valued let bindings must work")
	require.Empty(t, reported)
}

// TestXPath3BindingRuleContext checks that rule contexts keep working under
// the XPath 3.1 binding, including the union and attribute forms.
func TestXPath3BindingRuleContext(t *testing.T) {
	const src = `<?xml version="1.0" encoding="UTF-8"?>
<schema xmlns="http://purl.oclc.org/dsdl/schematron" queryBinding="xslt3">
  <pattern>
    <rule context="item">
      <assert test="normalize-space(.) != ''">item must have text</assert>
    </rule>
  </pattern>
  <pattern>
    <rule context="@code">
      <assert test="string-length(.) = 5">code must be five characters</assert>
    </rule>
  </pattern>
</schema>`

	reported, valErr := validateWithSchema(t, src)
	require.NoError(t, valErr, "element and attribute contexts must both match")
	require.Empty(t, reported)
}

// TestXPath3BindingResourceAccessDenied checks that a schema cannot read a
// document through fn:doc: retrieval needs a URI resolver, and the engine
// supplies none.
func TestXPath3BindingResourceAccessDenied(t *testing.T) {
	const ruleBody = `      <assert test="doc('file:///etc/hostname')">no document access</assert>`

	reported, valErr := validateWithSchema(t, schemaSrc("xslt3", ruleBody))
	require.ErrorIs(t, valErr, schematron.ErrValidationFailed)
	require.NotEmpty(t, reported)
	require.Contains(t, reported[0].Error(), "FODC0002", "retrieval must fail")
}

// TestXPath3BindingNamespaces checks that <ns> prefixes are in scope for
// XPath 3.1 expressions.
func TestXPath3BindingNamespaces(t *testing.T) {
	const src = `<?xml version="1.0" encoding="UTF-8"?>
<schema xmlns="http://purl.oclc.org/dsdl/schematron" queryBinding="xslt3">
  <ns prefix="e" uri="http://example.com/ns"/>
  <pattern>
    <rule context="e:root">
      <assert test="count(e:item) = 2">expected two namespaced items</assert>
    </rule>
  </pattern>
</schema>`

	const instance = `<?xml version="1.0" encoding="UTF-8"?>
<root xmlns="http://example.com/ns"><item/><item/></root>`

	schemaDoc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err, "parse schema")

	schema, err := schematron.NewCompiler().Compile(t.Context(), schemaDoc)
	require.NoError(t, err, "compile schema")
	require.Equal(t, schematron.QueryBindingXPath3, schema.QueryBinding())

	instDoc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
	require.NoError(t, err, "parse instance")

	require.NoError(t, schematron.NewValidator(schema).Validate(t.Context(), instDoc))
}

// TestXPath3BindingCompileError checks that an expression the XPath 3.1
// engine cannot parse fails compilation the same way an XPath 1.0 one does.
func TestXPath3BindingCompileError(t *testing.T) {
	src := schemaSrc("xslt3", `      <assert test="count(">broken</assert>`)

	schemaDoc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err, "parse schema")

	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	schema, err := schematron.NewCompiler().ErrorHandler(collector).Compile(t.Context(), schemaDoc)
	_ = collector.Close()

	require.ErrorIs(t, err, schematron.ErrCompileFailed)
	require.Nil(t, schema)
	require.NotEmpty(t, collector.Errors())
	require.Contains(t, collector.Errors()[0].Error(), "Failed to compile test expression")
}
