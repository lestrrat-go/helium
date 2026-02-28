package schematron

import (
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath"
	"github.com/stretchr/testify/require"
)

func compileTestSchema(t *testing.T, xml string) *Schema {
	t.Helper()
	doc, err := helium.Parse([]byte(xml))
	require.NoError(t, err)
	schema, err := Compile(doc)
	require.NoError(t, err)
	return schema
}

func TestCompileEmptyContext(t *testing.T) {
	schema := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
		<pattern><rule context=""><assert test="true()">ok</assert></rule></pattern>
	</schema>`)
	require.Contains(t, schema.CompileErrors(), "rule has an empty context attribute")
}

func TestCompileRuleNoAssert(t *testing.T) {
	schema := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
		<pattern><rule context="*"></rule></pattern>
	</schema>`)
	require.Contains(t, schema.CompileErrors(), "rule has no assert nor report element")
}

func TestCompilePatternNoRules(t *testing.T) {
	schema := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
		<pattern></pattern>
	</schema>`)
	require.Contains(t, schema.CompileErrors(), "Pattern has no rule element")
}

func TestCompileSchemaNoPatterns(t *testing.T) {
	schema := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
	</schema>`)
	require.Contains(t, schema.CompileErrors(), "schema has no pattern element")
}

func TestCompileNonRuleInPattern(t *testing.T) {
	schema := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
		<pattern>
			<bogus/>
			<rule context="*"><assert test="true()">ok</assert></rule>
		</pattern>
	</schema>`)
	require.Contains(t, schema.CompileErrors(), "Expecting a rule element instead of bogus")
}

func TestCompileValidSchema(t *testing.T) {
	schema := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
		<pattern>
			<rule context="*"><assert test="true()">ok</assert></rule>
		</pattern>
	</schema>`)
	require.Equal(t, "", schema.CompileErrors())
}

// Verify errors don't contain false positives for schemas with let-only rules.
func TestCompileRuleWithLetOnly(t *testing.T) {
	schema := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
		<pattern>
			<rule context="*">
				<let name="x" value="1"/>
			</rule>
		</pattern>
	</schema>`)
	// A rule with only let bindings and no assert/report should still emit the error.
	require.Contains(t, schema.CompileErrors(), "rule has no assert nor report element")
}

// Verify multiple errors accumulate.
func TestCompileMultipleErrors(t *testing.T) {
	schema := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
		<pattern>
			<rule context="">
			</rule>
		</pattern>
	</schema>`)
	errs := schema.CompileErrors()
	// Should have: empty context, pattern has no rule (since rule returned nil)
	require.True(t, strings.Contains(errs, "rule has an empty context attribute"))
	require.True(t, strings.Contains(errs, "Pattern has no rule element"))
}

func TestXpathResultToStringBoolean(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		r := &xpath.Result{Type: xpath.BooleanResult, Boolean: true}
		require.Equal(t, "True", xpathResultToString(r))
	})
	t.Run("false", func(t *testing.T) {
		r := &xpath.Result{Type: xpath.BooleanResult, Boolean: false}
		require.Equal(t, "False", xpathResultToString(r))
	})
}

func TestXpathResultToStringNodeSet(t *testing.T) {
	doc, err := helium.Parse([]byte(`<root><a/><b/><c/></root>`))
	require.NoError(t, err)

	root := doc.DocumentElement()
	require.NotNil(t, root)

	// Collect child element nodes.
	var children []helium.Node
	for c := root.FirstChild(); c != nil; c = c.NextSibling() {
		if c.Type() == helium.ElementNode {
			children = append(children, c)
		}
	}
	require.Len(t, children, 3)

	t.Run("empty", func(t *testing.T) {
		r := &xpath.Result{Type: xpath.NodeSetResult, NodeSet: nil}
		require.Equal(t, "", xpathResultToString(r))
	})
	t.Run("single node", func(t *testing.T) {
		r := &xpath.Result{Type: xpath.NodeSetResult, NodeSet: children[:1]}
		require.Equal(t, "a", xpathResultToString(r))
	})
	t.Run("multiple nodes", func(t *testing.T) {
		r := &xpath.Result{Type: xpath.NodeSetResult, NodeSet: children}
		require.Equal(t, "a b c", xpathResultToString(r))
	})
}
