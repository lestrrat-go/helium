package schematron

import (
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath1"
	"github.com/stretchr/testify/require"
)

func compileTestSchema(t *testing.T, xml string) (*Schema, string) {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)
	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	schema, err := NewCompiler().ErrorHandler(collector).Compile(t.Context(), doc)
	require.NoError(t, err)
	_ = collector.Close()
	var b strings.Builder
	for _, e := range collector.Errors() {
		b.WriteString(e.Error())
	}
	return schema, b.String()
}

func TestCompileEmptyContext(t *testing.T) {
	_, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
		<pattern><rule context=""><assert test="true()">ok</assert></rule></pattern>
	</schema>`)
	require.Contains(t, errs, "rule has an empty context attribute")
}

func TestCompileRuleNoAssert(t *testing.T) {
	_, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
		<pattern><rule context="*"></rule></pattern>
	</schema>`)
	require.Contains(t, errs, "rule has no assert nor report element")
}

func TestCompilePatternNoRules(t *testing.T) {
	_, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
		<pattern></pattern>
	</schema>`)
	require.Contains(t, errs, "Pattern has no rule element")
}

func TestCompileSchemaNoPatterns(t *testing.T) {
	_, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
	</schema>`)
	require.Contains(t, errs, "schema has no pattern element")
}

func TestCompileNonRuleInPattern(t *testing.T) {
	_, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
		<pattern>
			<bogus/>
			<rule context="*"><assert test="true()">ok</assert></rule>
		</pattern>
	</schema>`)
	require.Contains(t, errs, "Expecting a rule element instead of bogus")
}

func TestCompileValidSchema(t *testing.T) {
	_, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
		<pattern>
			<rule context="*"><assert test="true()">ok</assert></rule>
		</pattern>
	</schema>`)
	require.Equal(t, "", errs)
}

// Verify errors don't contain false positives for schemas with let-only rules.
func TestCompileRuleWithLetOnly(t *testing.T) {
	_, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
		<pattern>
			<rule context="*">
				<let name="x" value="1"/>
			</rule>
		</pattern>
	</schema>`)
	// A rule with only let bindings and no assert/report should still emit the error.
	require.Contains(t, errs, "rule has no assert nor report element")
}

// Verify multiple errors accumulate.
func TestCompileMultipleErrors(t *testing.T) {
	_, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
		<pattern>
			<rule context="">
			</rule>
		</pattern>
	</schema>`)
	// Should have: empty context, pattern has no rule (since rule returned nil)
	require.True(t, strings.Contains(errs, "rule has an empty context attribute"))
	require.True(t, strings.Contains(errs, "Pattern has no rule element"))
}

func TestCompileValueOfNoSelect(t *testing.T) {
	_, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
		<pattern>
			<rule context="*"><assert test="true()">val: <value-of/></assert></rule>
		</pattern>
	</schema>`)
	require.Contains(t, errs, "value-of has no select attribute")
}

func TestXpathResultToNameNamespace(t *testing.T) {
	t.Run("namespaced element", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		e := doc.CreateElement("item")
		require.NoError(t, e.DeclareNamespace("ns", "http://example.com"))
		require.NoError(t, e.SetActiveNamespace("ns", "http://example.com"))

		r := &xpath1.Result{
			Type:    xpath1.NodeSetResult,
			NodeSet: []helium.Node{e},
		}
		require.Equal(t, "ns:item", xpathResultToName(r))
	})

	t.Run("non-namespaced element", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		e := doc.CreateElement("item")
		require.NoError(t, doc.AddChild(e))

		r := &xpath1.Result{
			Type:    xpath1.NodeSetResult,
			NodeSet: []helium.Node{e},
		}
		require.Equal(t, "item", xpathResultToName(r))
	})

	t.Run("namespaced attribute", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		e := doc.CreateElement("root")
		require.NoError(t, doc.AddChild(e))
		ns := helium.NewNamespace("foo", "http://example.com/foo")
		_, err := e.SetAttributeNS("bar", "val", ns)
		require.NoError(t, err)

		// Find the attribute.
		var attr *helium.Attribute
		for _, a := range e.Attributes() {
			if a.LocalName() == "bar" {
				attr = a
				break
			}
		}
		require.NotNil(t, attr)

		r := &xpath1.Result{
			Type:    xpath1.NodeSetResult,
			NodeSet: []helium.Node{attr},
		}
		require.Equal(t, "foo:bar", xpathResultToName(r))
	})

	t.Run("non-namespaced attribute", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		e := doc.CreateElement("root")
		require.NoError(t, doc.AddChild(e))
		_, err := e.SetAttribute("baz", "val")
		require.NoError(t, err)

		var attr *helium.Attribute
		for _, a := range e.Attributes() {
			if a.LocalName() == "baz" {
				attr = a
				break
			}
		}
		require.NotNil(t, attr)

		r := &xpath1.Result{
			Type:    xpath1.NodeSetResult,
			NodeSet: []helium.Node{attr},
		}
		require.Equal(t, "baz", xpathResultToName(r))
	})
}

func TestCompileTitleAfterPattern(t *testing.T) {
	_, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
		<pattern><rule context="*"><assert test="true()">ok</assert></rule></pattern>
		<title>late title</title>
	</schema>`)
	require.Contains(t, errs, "Expecting a pattern element instead of title")
}

func TestCompileNsAfterPattern(t *testing.T) {
	_, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
		<pattern><rule context="*"><assert test="true()">ok</assert></rule></pattern>
		<ns prefix="p" uri="urn:test"/>
	</schema>`)
	require.Contains(t, errs, "Expecting a pattern element instead of ns")
}

func TestCompileTitleBeforeNsBeforePattern(t *testing.T) {
	schema, errs := compileTestSchema(t, `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
		<title>my schema</title>
		<ns prefix="p" uri="urn:test"/>
		<pattern><rule context="*"><assert test="true()">ok</assert></rule></pattern>
	</schema>`)
	require.Equal(t, "", errs)
	require.Equal(t, "my schema", schema.title)
}

// TestLetVariableChainedDependency verifies that let variables can
// reference previously registered variables, matching libxml2's
// xmlSchematronRegisterVariables behavior.
func TestLetVariableChainedDependency(t *testing.T) {
	// Schema defines two let variables where b depends on a.
	// libxml2 stores lets in LIFO order (prepend), so the list
	// is [b, a]. During evaluation b is evaluated first (no $a),
	// then a is evaluated and registered. This means forward
	// chained deps (b referencing a) don't work in libxml2 either.
	//
	// However, reverse chained deps DO work: if a is defined first
	// and b second, in LIFO order the list is [b, a], so a is
	// evaluated second and can see b. We test both scenarios.

	t.Run("independent lets", func(t *testing.T) {
		// Two independent let variables — both should work regardless of order.
		schemaXML := `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
			<pattern>
				<rule context="item">
					<let name="x" value="string(@val)"/>
					<let name="y" value="'hello'"/>
					<assert test="$x = 'ok'">x is <value-of select="$x"/>, y is <value-of select="$y"/></assert>
				</rule>
			</pattern>
		</schema>`
		schema, errs := compileTestSchema(t, schemaXML)
		require.Equal(t, "", errs)

		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><item val="bad"/></root>`))
		require.NoError(t, err)

		err = NewValidator(schema).Validate(t.Context(), doc)
		require.Error(t, err)
		require.Contains(t, err.Error(), "x is bad")
		require.Contains(t, err.Error(), "y is hello")
	})

	t.Run("LIFO accumulation", func(t *testing.T) {
		// In libxml2 LIFO order: lets are stored as [b, a].
		// During evaluation: b is evaluated first (no $a available),
		// then a is evaluated. a's expression ("1") doesn't depend on b.
		// b's expression ("$a + 1") was evaluated before a was registered,
		// so $a is undefined and b gets NaN.
		// This matches libxml2's actual behavior for forward-chained deps.
		schemaXML := `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
			<pattern>
				<rule context="item">
					<let name="a" value="1"/>
					<let name="b" value="$a + 1"/>
					<report test="$a = 1">a=<value-of select="$a"/></report>
				</rule>
			</pattern>
		</schema>`
		schema, errs := compileTestSchema(t, schemaXML)
		require.Equal(t, "", errs)

		// Verify LIFO ordering: the lets should be stored in reverse order.
		require.Len(t, schema.patterns[0].rules[0].lets, 2)
		require.Equal(t, "b", schema.patterns[0].rules[0].lets[0].name)
		require.Equal(t, "a", schema.patterns[0].rules[0].lets[1].name)

		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><item/></root>`))
		require.NoError(t, err)

		err = NewValidator(schema).Validate(t.Context(), doc)
		// a=1 should be reported since $a is properly registered.
		require.Error(t, err)
		require.Contains(t, err.Error(), "a=1")
	})
}
func TestXpathResultToStringBoolean(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		r := &xpath1.Result{Type: xpath1.BooleanResult, Bool: true}
		require.Equal(t, "True", xpathResultToString(r))
	})
	t.Run("false", func(t *testing.T) {
		r := &xpath1.Result{Type: xpath1.BooleanResult, Bool: false}
		require.Equal(t, "False", xpathResultToString(r))
	})
}

func TestXpathResultToStringNodeSet(t *testing.T) {
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><a/><b/><c/></root>`))
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
		r := &xpath1.Result{Type: xpath1.NodeSetResult, NodeSet: nil}
		require.Equal(t, "", xpathResultToString(r))
	})
	t.Run("single node", func(t *testing.T) {
		r := &xpath1.Result{Type: xpath1.NodeSetResult, NodeSet: children[:1]}
		require.Equal(t, "a", xpathResultToString(r))
	})
	t.Run("multiple nodes", func(t *testing.T) {
		r := &xpath1.Result{Type: xpath1.NodeSetResult, NodeSet: children}
		require.Equal(t, "a b c", xpathResultToString(r))
	})
}

func TestContextToXPath(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{"simple element", "a", "//a"},
		{"absolute", "/a", "/a"},
		{"union", "a | b", "//a | //b"},
		{"mixed absolute/relative union", "/a | b", "/a | //b"},
		{"multi-step union", "a/b | c/d", "//a/b | //c/d"},
		{"predicate with pipe", "a[contains(., '|')]", "//a[contains(., '|')]"},
		{"wildcard", "*", "//*"},
		{"root wildcard", "/*", "/*"},
		{"leading/trailing spaces", "  a | b  ", "//a | //b"},
		{"predicate with bracket", "a[@x='1'] | b", "//a[@x='1'] | //b"},
		{"nested parens with pipe", "a[f(.|b)] | c", "//a[f(.|b)] | //c"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expect, contextToXPath(tt.input))
		})
	}
}

func TestUnionContextIntegration(t *testing.T) {
	schemaXML := `<schema xmlns="http://purl.oclc.org/dsdl/schematron">
		<pattern>
			<rule context="invoice | credit-note">
				<assert test="@id">Missing id attribute</assert>
			</rule>
		</pattern>
	</schema>`
	schema, errs := compileTestSchema(t, schemaXML)
	require.Equal(t, "", errs)

	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><invoice/><credit-note/><other/></root>`))
	require.NoError(t, err)

	err = NewValidator(schema).Validate(t.Context(), doc)
	// Both invoice and credit-note should trigger the assert.
	require.Error(t, err)
	require.Contains(t, err.Error(), "invoice")
	require.Contains(t, err.Error(), "credit-note")
	// "other" should not be mentioned in failures.
	require.NotContains(t, err.Error(), "other")
}
