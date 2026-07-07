package xpath3_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// fn:serialize with an output:serialization-parameters element as the second
// argument exercises parseSerializeOptionsNode (the element-options path),
// distinct from the map-options path covered elsewhere.
func TestSerialize_OptionsNode(t *testing.T) {
	const paramsXML = `<output:serialization-parameters xmlns:output="http://www.w3.org/2010/xslt-xquery-serialization">` +
		`<output:method value="xml"/>` +
		`<output:indent value="no"/>` +
		`<output:omit-xml-declaration value="yes"/>` +
		`</output:serialization-parameters>`

	doc := mustParseXML(t, paramsXML)
	paramsElem := doc.DocumentElement()

	nodes, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Evaluate(t.Context(), mustCompile(t, `.`), paramsElem)
	require.NoError(t, err)

	const paramsVar = "params"
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Variables(map[string]xpath3.Sequence{
		paramsVar: nodes.Sequence(),
	})
	compiled := mustCompile(t, `serialize("hello", $params)`)
	res, err := eval.Evaluate(t.Context(), compiled, doc)
	require.NoError(t, err)
	require.Contains(t, res.StringValue(), "hello")
}

// undeclare-prefixes is a valid yes/no boolean serialization parameter
// (Serialization 3.1 §2); the element-options form must accept it (validated,
// not necessarily honored) rather than raising XPTY0004 (W3C serialize-xml-035b,
// serialize-xml-036b). An invalid value stays XPTY0004.
func TestSerialize_OptionsNodeUndeclarePrefixes(t *testing.T) {
	build := func(value string) xpath3.Sequence {
		t.Helper()
		xml := `<output:serialization-parameters xmlns:output="http://www.w3.org/2010/xslt-xquery-serialization">` +
			`<output:method value="xml"/>` +
			`<output:undeclare-prefixes value="` + value + `"/>` +
			`</output:serialization-parameters>`
		doc := mustParseXML(t, xml)
		nodes, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			Evaluate(t.Context(), mustCompile(t, `.`), doc.DocumentElement())
		require.NoError(t, err)
		return nodes.Sequence()
	}

	run := func(value string) error {
		t.Helper()
		doc := mustParseXML(t, `<root/>`)
		eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Variables(map[string]xpath3.Sequence{
			"params": build(value),
		})
		_, err := eval.Evaluate(t.Context(), mustCompile(t, `serialize(., $params)`), doc)
		return err
	}

	require.NoError(t, run("yes"))

	err := run("maybe")
	require.Error(t, err)
	var xpErr *xpath3.XPathError
	require.ErrorAs(t, err, &xpErr)
	require.Equal(t, "XPTY0004", xpErr.Code)
}

func mustCompile(t *testing.T, expr string) *xpath3.Expression {
	t.Helper()
	c, err := xpath3.NewCompiler().Compile(expr)
	require.NoError(t, err)
	return c
}

// fn:serialize with no serialization parameters uses the xml output method
// (adaptive is opt-in), under which serializing an attribute or namespace node
// is err:SENR0001 (W3C serialize-xml-002/011/012).
func TestSerialize_AttributeNodeSENR0001(t *testing.T) {
	doc := mustParseXML(t, `<root attr="value"/>`)

	_, err := evaluate(t.Context(), doc, `serialize((//@*)[1])`)
	require.Error(t, err)
	var xpErr *xpath3.XPathError
	require.ErrorAs(t, err, &xpErr)
	require.Equal(t, "SENR0001", xpErr.Code)
}

func TestSerialize_NamespaceNodeSENR0001(t *testing.T) {
	doc := mustParseXML(t, `<root xmlns:p="urn:example"/>`)

	_, err := evaluate(t.Context(), doc, `serialize((//namespace::*)[1])`)
	require.Error(t, err)
	var xpErr *xpath3.XPathError
	require.ErrorAs(t, err, &xpErr)
	require.Equal(t, "SENR0001", xpErr.Code)
}

// An explicit method="adaptive" still serializes an attribute as name="value".
func TestSerialize_AttributeNodeAdaptiveOK(t *testing.T) {
	doc := mustParseXML(t, `<root attr="value"/>`)

	res, err := evaluate(t.Context(), doc, `serialize((//@*)[1], map{"method":"adaptive"})`)
	require.NoError(t, err)
	require.Contains(t, res.StringValue(), `attr="value"`)
}

// The standalone map option value space is union(xs:boolean, "omit"); a bad
// string value (" omit " with surrounding spaces) is err:XPTY0004
// (W3C serialize-xml-131a).
func TestSerialize_StandaloneMapBadValueXPTY0004(t *testing.T) {
	doc := mustParseXML(t, `<root/>`)

	_, err := evaluate(t.Context(), doc,
		`serialize(., map{"omit-xml-declaration": false(), "standalone": " omit "})`)
	require.Error(t, err)
	var xpErr *xpath3.XPathError
	require.ErrorAs(t, err, &xpErr)
	require.Equal(t, "XPTY0004", xpErr.Code)

	// A typed xs:string is not an xs:boolean member of union(xs:boolean,"omit"),
	// and "yes"/"no" are not xs:boolean lexicals — both stay XPTY0004 (matching
	// the readBool value space used by the other boolean serialize options).
	for _, bad := range []string{"yes", "true"} {
		_, err := evaluate(t.Context(), doc,
			`serialize(., map{"standalone": "`+bad+`"})`)
		require.Error(t, err, bad)
		require.ErrorAs(t, err, &xpErr)
		require.Equal(t, "XPTY0004", xpErr.Code, bad)
	}
}

// A valid standalone map value ("omit") is accepted.
func TestSerialize_StandaloneMapValidOK(t *testing.T) {
	doc := mustParseXML(t, `<root/>`)

	_, err := evaluate(t.Context(), doc,
		`serialize(., map{"omit-xml-declaration": true(), "standalone": "omit"})`)
	require.NoError(t, err)

	// An empty-sequence value selects the default and must NOT error (QT3
	// serialize-xml-131).
	_, err = evaluate(t.Context(), doc,
		`serialize(., map{"omit-xml-declaration": true(), "standalone": ()})`)
	require.NoError(t, err)
}

// xml-to-json with an options map exercises parseXMLToJSONOptions.
func TestXMLToJSON_OptionsMap(t *testing.T) {
	r, err := evaluate(t.Context(), nil,
		`xml-to-json(json-to-xml('{"a": 1}'), map { "indent": true() })`)
	require.NoError(t, err)
	require.Contains(t, r.StringValue(), "\"a\"")

	// Invalid 'indent' option type -> XPTY0004.
	_, err = evaluate(t.Context(), nil,
		`xml-to-json(json-to-xml('{"a": 1}'), map { "indent": "notbool" })`)
	require.Error(t, err)
	var xpErr *xpath3.XPathError
	require.ErrorAs(t, err, &xpErr)
}
