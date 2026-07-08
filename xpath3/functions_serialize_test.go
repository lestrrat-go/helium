package xpath3_test

import (
	"errors"
	"testing"

	"github.com/lestrrat-go/helium"
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
// not raising XPTY0004; W3C serialize-xml-035b, serialize-xml-036b). An invalid
// value stays XPTY0004. Because undeclarations require XML/XHTML 1.1, a "yes"
// value is honored only when version="1.1"; at the default 1.0 it is SEPM0010.
func TestSerialize_OptionsNodeUndeclarePrefixes(t *testing.T) {
	build := func(value, version string) xpath3.Sequence {
		t.Helper()
		xml := `<output:serialization-parameters xmlns:output="http://www.w3.org/2010/xslt-xquery-serialization">` +
			`<output:method value="xml"/>` +
			`<output:version value="` + version + `"/>` +
			`<output:undeclare-prefixes value="` + value + `"/>` +
			`</output:serialization-parameters>`
		doc := mustParseXML(t, xml)
		nodes, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			Evaluate(t.Context(), mustCompile(t, `.`), doc.DocumentElement())
		require.NoError(t, err)
		return nodes.Sequence()
	}

	run := func(value, version string) error {
		t.Helper()
		doc := mustParseXML(t, `<root/>`)
		eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Variables(map[string]xpath3.Sequence{
			paramsVar: build(value, version),
		})
		_, err := eval.Evaluate(t.Context(), mustCompile(t, `serialize(., $params)`), doc)
		return err
	}

	// A valid "yes" at version 1.1 is accepted.
	require.NoError(t, run("yes", "1.1"))

	var xpErr *xpath3.XPathError

	// A valid "yes" at version 1.0 is the SEPM0010 static error.
	err := run("yes", "1.0")
	require.Error(t, err)
	require.ErrorAs(t, err, &xpErr)
	require.Equal(t, "SEPM0010", xpErr.Code)

	// An invalid boolean value stays XPTY0004 (parse-time, before the version check).
	err = run("maybe", "1.1")
	require.Error(t, err)
	require.ErrorAs(t, err, &xpErr)
	require.Equal(t, "XPTY0004", xpErr.Code)
}

func mustCompile(t *testing.T, expr string) *xpath3.Expression {
	t.Helper()
	c, err := xpath3.NewCompiler().Compile(expr)
	require.NoError(t, err)
	return c
}

// Sequence normalization (Serialization 3.1 §2, step 3): with an absent
// item-separator a single space separates only adjacent atomic-value strings,
// never adjacent nodes nor a node-and-atomic pair. A specified item-separator is
// inserted between every adjacent pair of items regardless of kind.
func TestSerialize_ItemSeparatorNormalization(t *testing.T) {
	doc := mustParseXML(t, `<r><a/><b/></r>`)

	run := func(t *testing.T, ctx helium.Node, expr string) string {
		t.Helper()
		res, err := evaluate(t.Context(), ctx, expr)
		require.NoError(t, err)
		return res.StringValue()
	}

	t.Run("adjacent nodes get no separator (xml)", func(t *testing.T) {
		require.Equal(t, "<a/><b/>", run(t, doc, `serialize(/r/*, map{"method":"xml"})`))
	})
	t.Run("adjacent nodes get no separator (default method)", func(t *testing.T) {
		require.Equal(t, "<a/><b/>", run(t, doc, `serialize(/r/*)`))
	})
	t.Run("adjacent atomics keep the space", func(t *testing.T) {
		require.Equal(t, "1 2 3", run(t, doc, `serialize((1,2,3), map{"method":"xml"})`))
	})
	t.Run("node adjacent to atomic gets no separator", func(t *testing.T) {
		require.Equal(t, "<a/>1", run(t, doc, `serialize((/r/a, 1), map{"method":"xml"})`))
	})
	t.Run("array flattens to nodes with no separator", func(t *testing.T) {
		require.Equal(t, "<a/><b/>", run(t, doc, `serialize([/r/a, /r/b], map{"method":"xml"})`))
	})
	t.Run("specified item-separator joins all items", func(t *testing.T) {
		require.Equal(t, "<a/>,<b/>", run(t, doc, `serialize(/r/*, map{"method":"xml","item-separator":","})`))
	})
	t.Run("specified item-separator joins atomics", func(t *testing.T) {
		require.Equal(t, "1|2|3|4", run(t, doc, `serialize(1 to 4, map{"method":"xml","item-separator":"|"})`))
	})
	t.Run("html adjacent nodes get no separator", func(t *testing.T) {
		require.Equal(t, "<a></a><b></b>", run(t, doc, `serialize(/r/*, map{"method":"html"})`))
	})
}

// The DTD / internal subset of a source document node is not part of the XDM
// data model, so fn:serialize (xml method) must never reproduce it. With no
// doctype-system parameter, serializing a document that has a DOCTYPE / internal
// subset emits no <!DOCTYPE> (Serialization 3.1 §5.1.7; W3C parse-xml-006/008,
// fn-doc-25/26/29).
func TestSerialize_NoSourceDoctypeLeak(t *testing.T) {
	run := func(t *testing.T, xml string) string {
		t.Helper()
		doc := mustParseXML(t, xml)
		res, err := evaluate(t.Context(), doc, `serialize(.)`)
		require.NoError(t, err)
		return res.StringValue()
	}

	t.Run("external subset (SYSTEM)", func(t *testing.T) {
		out := run(t, `<!DOCTYPE root SYSTEM "x.dtd">`+"\n"+`<root><child/></root>`)
		require.NotContains(t, out, "DOCTYPE")
		require.Contains(t, out, "<root><child/></root>")
	})
	t.Run("internal subset", func(t *testing.T) {
		out := run(t, `<!DOCTYPE a [<!ELEMENT a (#PCDATA)>]><a>foo</a>`)
		require.NotContains(t, out, "DOCTYPE")
		require.Contains(t, out, "<a>foo</a>")
	})

	// With doctype-system specified, the DOCTYPE IS emitted (from the parameter,
	// not the source tree), naming the document element with an empty internal
	// subset.
	t.Run("doctype-system emits DOCTYPE", func(t *testing.T) {
		doc := mustParseXML(t, `<!DOCTYPE a [<!ELEMENT a (#PCDATA)>]><a>foo</a>`)
		res, err := evaluate(t.Context(), doc, `serialize(., map{"doctype-system":"here.dtd"})`)
		require.NoError(t, err)
		out := res.StringValue()
		require.Contains(t, out, `<!DOCTYPE a SYSTEM "here.dtd">`)
		require.Contains(t, out, "<a>foo</a>")
		require.NotContains(t, out, "ELEMENT")
	})
}

// fn:serialize serializes an element as if it were the root of a tree, so an
// element selected from a larger document must carry the namespace declarations
// for every namespace in scope on it — including those inherited from ancestors
// outside the serialized subtree (XDM: an element node's namespace nodes are ALL
// its in-scope namespaces; W3C fn-union-node-args-015/016/017,
// fn-intersect-node-args-015/016, XQueryComment002).
func TestSerialize_IsolatedElementInScopeNamespaces(t *testing.T) {
	t.Run("used inherited prefix is declared", func(t *testing.T) {
		doc := mustParseXML(t, `<a xmlns:p="urn:P"><p:b/></a>`)
		res, err := evaluate(t.Context(), doc, `serialize(/a/*[1])`)
		require.NoError(t, err)
		require.Equal(t, `<p:b xmlns:p="urn:P"/>`, res.StringValue())
	})

	// All in-scope namespaces are declared, including ones inherited but not used
	// by the element itself (matching Saxon / the XDM data model).
	t.Run("unused inherited prefixes are also declared", func(t *testing.T) {
		doc := mustParseXML(t,
			`<atomic:root xmlns:atomic="urn:A" xmlns:foo="urn:F" xmlns:xsi="urn:X">`+
				`<atomic:integer>12</atomic:integer></atomic:root>`)
		res, err := evaluate(t.Context(), doc, `serialize(/*/*[1])`)
		require.NoError(t, err)
		out := res.StringValue()
		require.Contains(t, out, `xmlns:atomic="urn:A"`)
		require.Contains(t, out, `xmlns:foo="urn:F"`)
		require.Contains(t, out, `xmlns:xsi="urn:X"`)
		require.Contains(t, out, `>12</atomic:integer>`)
	})

	// A document root (no element ancestor) is serialized byte-identically — no
	// spurious extra declarations.
	t.Run("document root is unchanged", func(t *testing.T) {
		doc := mustParseXML(t, `<a xmlns:p="urn:P"><p:b/></a>`)
		res, err := evaluate(t.Context(), doc, `serialize(.)`)
		require.NoError(t, err)
		require.Equal(t, "<?xml version=\"1.0\"?>\n"+`<a xmlns:p="urn:P"><p:b/></a>`, res.StringValue())
	})

	// The document root ELEMENT (not the document node) also has no element
	// ancestor, so it too is serialized without spurious extra declarations.
	t.Run("root element is unchanged", func(t *testing.T) {
		doc := mustParseXML(t, `<a xmlns:p="urn:P"><p:b/></a>`)
		res, err := evaluate(t.Context(), doc, `serialize(/a)`)
		require.NoError(t, err)
		require.Equal(t, `<a xmlns:p="urn:P"><p:b/></a>`, res.StringValue())
	})
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

// TestSerialize_OptionElementOnlyRaisesFOTY0012 verifies that a string-valued
// fn:serialize option (method / item-separator / encoding / standalone) whose
// map value is an element-only-typed node surfaces err:FOTY0012 — the node has
// no typed value, so option (function) conversion cannot atomize it to a string.
// This guards that option-map string extraction threads the ctx-aware
// typed-value atomization rather than plain AtomizeSequence.
func TestSerialize_OptionElementOnlyRaisesFOTY0012(t *testing.T) {
	doc := mustParseXML(t, `<root><child>hi</child></root>`)
	root := doc.DocumentElement()

	decls := contentKindDecls{kinds: map[string]xpath3.ContentTypeKind{
		xpath3.QAnnotation("urn:t", "rootType"): xpath3.ContentTypeElementOnly,
	}}
	newEval := func() xpath3.Evaluator {
		return xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			TypeAnnotations(map[helium.Node]string{
				root: xpath3.QAnnotation("urn:t", "rootType"),
			}).
			SchemaDeclarations(decls)
	}

	requireFOTY0012 := func(t *testing.T, expr string) {
		t.Helper()
		compiled, err := xpath3.NewCompiler().Compile(expr)
		require.NoError(t, err)
		_, err = newEval().Evaluate(t.Context(), compiled, doc)
		require.Error(t, err)
		var xerr *xpath3.XPathError
		require.True(t, errors.As(err, &xerr), "want *xpath3.XPathError, got %T: %v", err, err)
		require.Equal(t, "FOTY0012", xerr.Code)
	}

	t.Run("method element-only raises FOTY0012", func(t *testing.T) {
		requireFOTY0012(t, `serialize("x", map{"method": /*})`)
	})
	t.Run("item-separator element-only raises FOTY0012", func(t *testing.T) {
		requireFOTY0012(t, `serialize("x", map{"item-separator": /*})`)
	})
	t.Run("encoding element-only raises FOTY0012", func(t *testing.T) {
		requireFOTY0012(t, `serialize("x", map{"encoding": /*})`)
	})
	t.Run("standalone element-only raises FOTY0012", func(t *testing.T) {
		requireFOTY0012(t, `serialize("x", map{"standalone": /*})`)
	})
}

// TestOptionMapElementOnlyRaisesFOTY0012 locks in the proactive sweep: every
// string-valued OPTION-MAP extractor that atomizes its value must surface
// err:FOTY0012 for an element-only-typed node (no typed value) rather than
// masking it as the function's own bad-option error (FOJS0005 / XPTY0004).
// Covers map:merge, fn:parse-json, and fn:json-to-xml "duplicates" options.
func TestOptionMapElementOnlyRaisesFOTY0012(t *testing.T) {
	doc := mustParseXML(t, `<root><child>hi</child></root>`)
	root := doc.DocumentElement()

	decls := contentKindDecls{kinds: map[string]xpath3.ContentTypeKind{
		xpath3.QAnnotation("urn:t", "rootType"): xpath3.ContentTypeElementOnly,
	}}
	newEval := func() xpath3.Evaluator {
		return xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			TypeAnnotations(map[helium.Node]string{
				root: xpath3.QAnnotation("urn:t", "rootType"),
			}).
			SchemaDeclarations(decls)
	}

	requireFOTY0012 := func(t *testing.T, expr string) {
		t.Helper()
		compiled, err := xpath3.NewCompiler().Compile(expr)
		require.NoError(t, err)
		_, err = newEval().Evaluate(t.Context(), compiled, doc)
		require.Error(t, err)
		var xerr *xpath3.XPathError
		require.True(t, errors.As(err, &xerr), "want *xpath3.XPathError, got %T: %v", err, err)
		require.Equal(t, "FOTY0012", xerr.Code)
	}

	t.Run("map:merge duplicates element-only", func(t *testing.T) {
		requireFOTY0012(t, `map:merge((map{"a": 1}), map{"duplicates": /*})`)
	})
	t.Run("parse-json duplicates element-only", func(t *testing.T) {
		requireFOTY0012(t, `parse-json('{}', map{"duplicates": /*})`)
	})
	t.Run("json-to-xml duplicates element-only", func(t *testing.T) {
		requireFOTY0012(t, `json-to-xml('{}', map{"duplicates": /*})`)
	})
}

// TestOptionMapFunctionValueRaisesFOTY0013 verifies the companion propagation:
// a function/map item used as a string-valued option value has no atomizable
// typed value, so atomizing it raises err:FOTY0013 — the extractors must let it
// surface rather than masking it as their own bad-option error (FOJS0005 /
// XPTY0004 / SEPM0016). No schema wiring is needed since the value is a function
// item regardless of content-kind.
func TestOptionMapFunctionValueRaisesFOTY0013(t *testing.T) {
	doc := mustParseXML(t, `<root/>`)

	requireFOTY0013 := func(t *testing.T, expr string) {
		t.Helper()
		compiled, err := xpath3.NewCompiler().Compile(expr)
		require.NoError(t, err)
		_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(t.Context(), compiled, doc)
		require.Error(t, err)
		var xerr *xpath3.XPathError
		require.True(t, errors.As(err, &xerr), "want *xpath3.XPathError, got %T: %v", err, err)
		require.Equal(t, "FOTY0013", xerr.Code)
	}

	t.Run("map:merge duplicates function item", func(t *testing.T) {
		requireFOTY0013(t, `map:merge((map{"a": 1}), map{"duplicates": true#0})`)
	})
	t.Run("parse-json duplicates function item", func(t *testing.T) {
		requireFOTY0013(t, `parse-json('{}', map{"duplicates": true#0})`)
	})
	t.Run("json-to-xml duplicates function item", func(t *testing.T) {
		requireFOTY0013(t, `json-to-xml('{}', map{"duplicates": true#0})`)
	})
	t.Run("serialize standalone function item", func(t *testing.T) {
		requireFOTY0013(t, `serialize(/*, map{"standalone": true#0})`)
	})
}
