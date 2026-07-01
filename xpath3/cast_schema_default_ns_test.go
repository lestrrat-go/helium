package xpath3_test

import (
	"context"
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// nsFooTypeDecls is a minimal SchemaDeclarations that declares a single
// user-defined type {urn:t}foo whose base is xs:string. It declares nothing in
// no-namespace, so a cast fallback that mis-resolves the bare type name "foo"
// (used under an xpathDefaultNamespace) to no namespace would fail the lookup.
type nsFooTypeDecls struct{}

func (nsFooTypeDecls) LookupSchemaElement(local, ns string) (string, bool)   { return "", false }
func (nsFooTypeDecls) LookupSchemaAttribute(local, ns string) (string, bool) { return "", false }
func (nsFooTypeDecls) LookupSchemaType(local, ns string) (string, bool) {
	if local == testFoo && ns == "urn:t" {
		return xpath3.TypeString, true
	}
	return "", false
}
func (nsFooTypeDecls) IsSubtypeOf(typeName, baseTypeName string) bool {
	return typeName == baseTypeName
}
func (nsFooTypeDecls) ValidateCast(_ context.Context, value, typeName string) error { return nil }
func (nsFooTypeDecls) ValidateCastWithNS(_ context.Context, value, typeName string, nsMap map[string]string) error {
	return nil
}
func (nsFooTypeDecls) ListItemType(typeName string) (string, bool) { return "", false }
func (nsFooTypeDecls) UnionMemberTypes(typeName string) []string   { return nil }

// TestCastUnprefixedUserTypeUnderDefaultNS verifies that the cast schema fallback
// resolves an UNPREFIXED user-defined target type using the ALREADY-RESOLVED
// expanded name (the xpathDefaultNamespace applied), not by re-parsing the lexical
// prefix (PR859-CR15-02). With the default element namespace bound to urn:t, the
// bare name "foo" must resolve to {urn:t}foo and the cast must succeed.
func TestCastUnprefixedUserTypeUnderDefaultNS(t *testing.T) {
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Namespaces(map[string]string{"": "urn:t"}).
		SchemaDeclarations(nsFooTypeDecls{})

	res, err := eval.Evaluate(t.Context(), mustCompile(t, `string('x' cast as foo)`), nil)
	require.NoError(t, err)
	require.Equal(t, "x", res.StringValue())
}

// TestCastQNameValueNoDefaultNamespace verifies that under XSD value-space
// semantics (QNameValueNoDefaultNamespace), casting an UNPREFIXED string to
// xs:QName yields a QName with NO namespace, even when an XPath default element
// namespace is in scope (PR859-CR15-03). The default xpath3 behavior (no flag)
// still applies the default namespace, so this change is additive/opt-in.
func TestCastQNameValueNoDefaultNamespace(t *testing.T) {
	t.Run("opt-in: unprefixed value has no namespace", func(t *testing.T) {
		eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			Namespaces(map[string]string{"": "urn:def"}).
			QNameValueNoDefaultNamespace()

		res, err := eval.Evaluate(t.Context(), mustCompile(t, `namespace-uri-from-QName('bare' cast as xs:QName)`), nil)
		require.NoError(t, err)
		require.Equal(t, "", res.StringValue())
	})

	t.Run("default: unprefixed value adopts default namespace", func(t *testing.T) {
		eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			Namespaces(map[string]string{"": "urn:def"})

		res, err := eval.Evaluate(t.Context(), mustCompile(t, `namespace-uri-from-QName('bare' cast as xs:QName)`), nil)
		require.NoError(t, err)
		require.Equal(t, "urn:def", res.StringValue())
	})
}
