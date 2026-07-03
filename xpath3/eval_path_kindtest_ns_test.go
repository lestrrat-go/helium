package xpath3_test

import (
	"context"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// noNSFooAttrDecls is a minimal SchemaDeclarations that declares a single
// no-namespace attribute {}foo of type xs:string. It declares nothing in any
// other namespace, so a schema-attribute(foo) test that mis-resolves the bare
// name to a non-empty default element namespace would fail the lookup.
type noNSFooAttrDecls struct{}

func (noNSFooAttrDecls) LookupSchemaElement(local, ns string) (string, bool) { return "", false }
func (noNSFooAttrDecls) LookupSchemaAttribute(local, ns string) (string, bool) {
	if local == testFoo && ns == "" {
		return xpath3.TypeString, true
	}
	return "", false
}
func (noNSFooAttrDecls) LookupSchemaType(local, ns string) (string, bool) { return "", false }
func (noNSFooAttrDecls) IsSubtypeOf(typeName, baseTypeName string) bool {
	return typeName == baseTypeName
}
func (noNSFooAttrDecls) ValidateCast(_ context.Context, value, typeName string) error { return nil }
func (noNSFooAttrDecls) ValidateCastWithNS(_ context.Context, value, typeName string, nsMap map[string]string) error {
	return nil
}
func (noNSFooAttrDecls) ListItemType(typeName string) (string, bool) { return "", false }
func (noNSFooAttrDecls) UnionMemberTypes(typeName string) []string   { return nil }
func (noNSFooAttrDecls) IsSubstitutionGroupMember(memberLocal, memberNS, headLocal, headNS string) bool {
	return false
}

// predeclaredNSDecls is a minimal SchemaDeclarations that declares one element
// {NSXS}foo and one attribute {NSXS}foo of type xs:string, where NSXS is the
// XSD namespace bound to the predeclared XPath prefix "xs". It declares nothing
// else, so a schema-element(xs:foo)/schema-attribute(xs:foo) test that fails to
// fall back to the predeclared "xs" prefix (when "xs" is absent from the local
// bindings) would resolve to no namespace and fail the lookup.
type predeclaredNSDecls struct{}

func mustXSNamespace() string {
	ns, ok := xpath3.PredeclaredNamespace("xs")
	if !ok {
		panic("xs must be a predeclared XPath prefix")
	}
	return ns
}

func (predeclaredNSDecls) LookupSchemaElement(local, ns string) (string, bool) {
	if local == testFoo && ns == mustXSNamespace() {
		return xpath3.TypeString, true
	}
	return "", false
}
func (predeclaredNSDecls) LookupSchemaAttribute(local, ns string) (string, bool) {
	if local == testFoo && ns == mustXSNamespace() {
		return xpath3.TypeString, true
	}
	return "", false
}
func (predeclaredNSDecls) LookupSchemaType(local, ns string) (string, bool) { return "", false }
func (predeclaredNSDecls) IsSubtypeOf(typeName, baseTypeName string) bool {
	return typeName == baseTypeName
}
func (predeclaredNSDecls) ValidateCast(_ context.Context, value, typeName string) error { return nil }
func (predeclaredNSDecls) ValidateCastWithNS(_ context.Context, value, typeName string, nsMap map[string]string) error {
	return nil
}
func (predeclaredNSDecls) ListItemType(typeName string) (string, bool) { return "", false }
func (predeclaredNSDecls) UnionMemberTypes(typeName string) []string   { return nil }
func (predeclaredNSDecls) IsSubstitutionGroupMember(memberLocal, memberNS, headLocal, headNS string) bool {
	return false
}

// TestSchemaTestPredeclaredPrefix verifies that schema-element()/
// schema-attribute() step tests and the equivalent "instance of" tests resolve
// a PREDECLARED XPath prefix (here "xs") that is NOT present in the local
// namespace bindings, mirroring the xslt3 pattern resolver (resolvePatternPrefix:
// lexical bindings -> predeclared XPath prefixes -> xml). Before the fix, the
// step/instance-of path consulted only the local bindings and resolved "xs:foo"
// to no namespace, diverging from the pattern path. (codex finding 654-14)
func TestSchemaTestPredeclaredPrefix(t *testing.T) {
	t.Parallel()

	xsNS := mustXSNamespace()
	// An element and attribute in the XSD namespace, declared WITHOUT an "xs"
	// prefix in scope on the instance. The query relies on the predeclared "xs".
	doc := `<root xmlns:a="` + xsNS + `"><a:foo a:foo="v"/></root>`

	parse := func(t *testing.T) *helium.Document {
		t.Helper()
		parsed, err := helium.NewParser().Parse(t.Context(), []byte(doc))
		require.NoError(t, err)
		return parsed
	}

	// Annotate the {NSXS}foo element and attribute as xs:string.
	annotate := func(t *testing.T, parsed *helium.Document) map[helium.Node]string {
		t.Helper()
		annotations := map[helium.Node]string{}
		root := parsed.DocumentElement()
		for c := range helium.Children(root) {
			elem, ok := helium.AsNode[*helium.Element](c)
			if !ok {
				continue
			}
			annotations[elem] = xpath3.TypeString
			for _, a := range elem.Attributes() {
				annotations[a] = xpath3.TypeString
			}
		}
		require.Len(t, annotations, 2, "fixture must annotate the foo element and attribute")
		return annotations
	}

	t.Run("schema-element(xs:foo) step resolves predeclared prefix", func(t *testing.T) {
		t.Parallel()

		parsed := parse(t)
		annotations := annotate(t, parsed)
		compiled, err := xpath3.NewCompiler().Compile(`//*[self::schema-element(xs:foo)]`)
		require.NoError(t, err)
		r, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			TypeAnnotations(annotations).
			SchemaDeclarations(predeclaredNSDecls{}).
			Evaluate(t.Context(), compiled, parsed)
		require.NoError(t, err)
		nodes, err := r.Nodes()
		require.NoError(t, err)
		require.Len(t, nodes, 1,
			"schema-element(xs:foo) must resolve the predeclared xs prefix and match {NSXS}foo")
	})

	t.Run("schema-attribute(xs:foo) step resolves predeclared prefix", func(t *testing.T) {
		t.Parallel()

		parsed := parse(t)
		annotations := annotate(t, parsed)
		compiled, err := xpath3.NewCompiler().Compile(`//*/@*[self::schema-attribute(xs:foo)]`)
		require.NoError(t, err)
		r, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			TypeAnnotations(annotations).
			SchemaDeclarations(predeclaredNSDecls{}).
			Evaluate(t.Context(), compiled, parsed)
		require.NoError(t, err)
		nodes, err := r.Nodes()
		require.NoError(t, err)
		require.Len(t, nodes, 1,
			"schema-attribute(xs:foo) must resolve the predeclared xs prefix and match {NSXS}foo")
	})

	t.Run("instance of schema-element(xs:foo) resolves predeclared prefix", func(t *testing.T) {
		t.Parallel()

		parsed := parse(t)
		annotations := annotate(t, parsed)
		compiled, err := xpath3.NewCompiler().Compile(`//a:foo instance of schema-element(xs:foo)`)
		require.NoError(t, err)
		r, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			Namespaces(map[string]string{"a": xsNS}).
			TypeAnnotations(annotations).
			SchemaDeclarations(predeclaredNSDecls{}).
			Evaluate(t.Context(), compiled, parsed)
		require.NoError(t, err)
		b, ok := r.IsBoolean()
		require.True(t, ok, "instance of must produce a boolean result")
		require.True(t, b,
			"`instance of schema-element(xs:foo)` must resolve the predeclared xs prefix")
	})
}

// TestStepKindTestAttributeDefaultNS verifies that an attribute kind-test used
// as a self:: predicate / axis step resolves a bare (unprefixed) name to NO
// namespace, never to the default element namespace ("" binding). This must be
// symmetric with a NameTest and with the direct (top-level) pattern kind-test:
// xpath-default-namespace governs element names, not attribute names. The bug
// (codex 654-12) was that the self::/axis-step evaluation path applied the
// default element namespace to bare attribute kind-test names while the direct
// pattern path did not.
func TestStepKindTestAttributeDefaultNS(t *testing.T) {
	t.Parallel()

	// foo is a no-namespace attribute; p:foo is {urn:x}foo.
	const doc = `<root><e foo="v"/><e xmlns:p="urn:x" p:foo="w"/></root>`

	// The "" binding installs urn:x as the default element namespace. A bare
	// attribute kind-test must IGNORE this default.
	ns := map[string]string{"": "urn:x"}

	t.Run("attribute(foo) matches only no-namespace attribute", func(t *testing.T) {
		t.Parallel()

		parsed, err := helium.NewParser().Parse(t.Context(), []byte(doc))
		require.NoError(t, err)
		compiled, err := xpath3.NewCompiler().Compile(`//*/@*[self::attribute(foo)]`)
		require.NoError(t, err)
		r, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			Namespaces(ns).Evaluate(t.Context(), compiled, parsed)
		require.NoError(t, err)
		nodes, err := r.Nodes()
		require.NoError(t, err)

		require.Len(t, nodes, 1,
			"self::attribute(foo) under default element namespace must match exactly the no-namespace foo")
		attr, ok := helium.AsNode[*helium.Attribute](nodes[0])
		require.True(t, ok)
		require.Equal(t, testFoo, attr.LocalName())
		require.Equal(t, "", attr.URI(),
			"matched attribute must be in NO namespace, not the default element namespace")
	})

	t.Run("schema-attribute(foo) resolves bare name to no namespace", func(t *testing.T) {
		t.Parallel()

		parsed, err := helium.NewParser().Parse(t.Context(), []byte(doc))
		require.NoError(t, err)

		// Annotate both attributes as xs:string so the schema-attribute() type
		// check passes; the discriminator is the namespace resolution of the bare
		// name. The schema declares only {}foo, so a correct resolution matches the
		// no-namespace foo and rejects {urn:x}foo.
		root := parsed.DocumentElement()
		annotations := map[helium.Node]string{}
		for c := range helium.Children(root) {
			elem, ok := helium.AsNode[*helium.Element](c)
			if !ok {
				continue
			}
			for _, a := range elem.Attributes() {
				annotations[a] = xpath3.TypeString
			}
		}
		require.Len(t, annotations, 2, "test fixture must annotate both foo attributes")

		compiled, err := xpath3.NewCompiler().Compile(`//*/@*[self::schema-attribute(foo)]`)
		require.NoError(t, err)
		r, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			Namespaces(ns).
			TypeAnnotations(annotations).
			SchemaDeclarations(noNSFooAttrDecls{}).
			Evaluate(t.Context(), compiled, parsed)
		require.NoError(t, err)
		nodes, err := r.Nodes()
		require.NoError(t, err)

		require.Len(t, nodes, 1,
			"self::schema-attribute(foo) must resolve the bare name to NO namespace and match only the no-namespace foo")
		attr, ok := helium.AsNode[*helium.Attribute](nodes[0])
		require.True(t, ok)
		require.Equal(t, testFoo, attr.LocalName())
		require.Equal(t, "", attr.URI(),
			"schema-attribute(foo) must not match {urn:x}foo via the default element namespace")
	})
}

// Path steps with kind tests (processing-instruction(target), element(name),
// attribute(name), comment(), text(), node()) exercise matchNodeTest and
// matchElementOrAttributeName branches that bare name tests do not.
func TestNodeTestPaths(t *testing.T) {
	const xml = `<?xml version="1.0"?>` +
		`<root a="1" b="2">` +
		`<?proc-a data?><?proc-b more?>` +
		`text-before<child>inner</child>text-after` +
		`<!--a comment-->` +
		`</root>`

	doc := mustParseXML(t, xml)
	root := doc.DocumentElement()

	cases := []struct {
		expr   string
		expect int
	}{
		{`child::processing-instruction()`, 2},
		{`child::processing-instruction("proc-a")`, 1},
		{`child::processing-instruction("missing")`, 0},
		{`child::element()`, 1},
		{`child::element(child)`, 1},
		{`child::element(other)`, 0},
		{`attribute::attribute()`, 2},
		{`attribute::attribute(a)`, 1},
		{`attribute::attribute(missing)`, 0},
		{`child::text()`, 2},
		{`child::comment()`, 1},
		{`child::node()`, 6},
		{`descendant::text()`, 3},
		{`child::*`, 1},
		{`attribute::*`, 2},
	}
	for _, tc := range cases {
		t.Run(tc.expr, func(t *testing.T) {
			nodes, err := find(t.Context(), root, tc.expr)
			require.NoError(t, err, tc.expr)
			require.Len(t, nodes, tc.expect, tc.expr)
		})
	}

	// document-node(element(root)) matched from the document node.
	nodes, err := find(t.Context(), doc, `self::document-node(element(root))`)
	require.NoError(t, err)
	require.Len(t, nodes, 1)

	nodes, err = find(t.Context(), doc, `self::document-node(element(other))`)
	require.NoError(t, err)
	require.Empty(t, nodes)
}
