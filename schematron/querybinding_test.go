package schematron_test

import (
	"fmt"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/schematron"
	"github.com/stretchr/testify/require"
)

// schemaWithBinding builds a minimal schema whose queryBinding attribute is
// set to the given value. An empty value omits the attribute entirely.
func schemaWithBinding(binding string) string {
	attr := ""
	if binding != "" {
		attr = fmt.Sprintf(" queryBinding=%q", binding)
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<schema xmlns="http://purl.oclc.org/dsdl/schematron"%s>
  <pattern>
    <rule context="root">
      <assert test="child">child is required</assert>
    </rule>
  </pattern>
</schema>`, attr)
}

func compileBindingSchema(t *testing.T, c schematron.Compiler, binding string) (*schematron.Schema, error) {
	t.Helper()

	doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaWithBinding(binding)))
	require.NoError(t, err, "parse schema")

	return c.Compile(t.Context(), doc)
}

func TestQueryBindingResolution(t *testing.T) {
	t.Run("accepted attribute values", func(t *testing.T) {
		for _, binding := range []string{"", "xslt", "xslt1", "xpath", "xpath1", "XSLT", " xpath "} {
			t.Run(fmt.Sprintf("%q", binding), func(t *testing.T) {
				schema, err := compileBindingSchema(t, schematron.NewCompiler(), binding)
				require.NoError(t, err, "compile schema")
				require.Equal(t, schematron.QueryBindingXPath1, schema.QueryBinding())
			})
		}
	})

	t.Run("rejected attribute values", func(t *testing.T) {
		for _, binding := range []string{"xslt2", "xpath2", "xquery", "stx", "exslt", "nonsense"} {
			t.Run(binding, func(t *testing.T) {
				collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
				c := schematron.NewCompiler().ErrorHandler(collector)

				schema, err := compileBindingSchema(t, c, binding)
				require.ErrorIs(t, err, schematron.ErrUnsupportedQueryBinding, "compile must reject the binding")
				require.Nil(t, schema, "no schema on failure")

				_ = collector.Close()
				require.Len(t, collector.Errors(), 1, "the failure reaches the error handler")
				require.Contains(t, collector.Errors()[0].Error(), binding)
			})
		}
	})

	t.Run("forced binding wins over an unsupported attribute", func(t *testing.T) {
		c := schematron.NewCompiler().QueryBinding(schematron.QueryBindingXPath1)

		schema, err := compileBindingSchema(t, c, "xquery")
		require.NoError(t, err, "a forced binding skips the attribute")
		require.Equal(t, schematron.QueryBindingXPath1, schema.QueryBinding())
	})

	t.Run("default binding applies only when the attribute is absent", func(t *testing.T) {
		c := schematron.NewCompiler().DefaultQueryBinding(schematron.QueryBindingXPath1)

		schema, err := compileBindingSchema(t, c, "")
		require.NoError(t, err, "compile schema")
		require.Equal(t, schematron.QueryBindingXPath1, schema.QueryBinding())

		_, err = compileBindingSchema(t, c, "xslt2")
		require.ErrorIs(t, err, schematron.ErrUnsupportedQueryBinding, "the attribute still wins over the default")
	})

	t.Run("nil schema reports no binding", func(t *testing.T) {
		var schema *schematron.Schema
		require.Equal(t, schematron.QueryBindingUnspecified, schema.QueryBinding())
	})
}

func TestParseQueryBinding(t *testing.T) {
	for _, tc := range []struct {
		input  string
		expect schematron.QueryBinding
	}{
		{input: "", expect: schematron.QueryBindingUnspecified},
		{input: "xslt", expect: schematron.QueryBindingXPath1},
		{input: "XPath1", expect: schematron.QueryBindingXPath1},
	} {
		t.Run(fmt.Sprintf("%q", tc.input), func(t *testing.T) {
			b, err := schematron.ParseQueryBinding(tc.input)
			require.NoError(t, err, "parse binding")
			require.Equal(t, tc.expect, b)
		})
	}

	t.Run("unsupported", func(t *testing.T) {
		b, err := schematron.ParseQueryBinding("xslt2")
		require.ErrorIs(t, err, schematron.ErrUnsupportedQueryBinding)
		require.Equal(t, schematron.QueryBindingUnspecified, b)
	})
}

func TestQueryBindingString(t *testing.T) {
	require.Equal(t, "unspecified", schematron.QueryBindingUnspecified.String())
	require.Equal(t, "xpath1", schematron.QueryBindingXPath1.String())
}

// TestQueryBindingCompiledSchemaEvaluates checks that a schema naming an
// explicit XPath 1.0 binding validates exactly like one that names none.
func TestQueryBindingCompiledSchemaEvaluates(t *testing.T) {
	const instanceSrc = `<?xml version="1.0" encoding="UTF-8"?><root/>`

	instDoc, err := helium.NewParser().Parse(t.Context(), []byte(instanceSrc))
	require.NoError(t, err, "parse instance")

	for _, binding := range []string{"", "xslt"} {
		t.Run(fmt.Sprintf("%q", binding), func(t *testing.T) {
			schema, err := compileBindingSchema(t, schematron.NewCompiler(), binding)
			require.NoError(t, err, "compile schema")

			err = schematron.NewValidator(schema).Validate(t.Context(), instDoc)
			require.ErrorIs(t, err, schematron.ErrValidationFailed, "the missing child must fail the assert")
		})
	}
}
