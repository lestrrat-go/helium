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
	if local == "foo" && ns == "" {
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
		require.Equal(t, "foo", attr.LocalName())
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
		require.Equal(t, "foo", attr.LocalName())
		require.Equal(t, "", attr.URI(),
			"schema-attribute(foo) must not match {urn:x}foo via the default element namespace")
	})
}
