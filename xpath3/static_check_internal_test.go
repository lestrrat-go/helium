package xpath3

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompileBuildsPrefixValidationPlan(t *testing.T) {
	expr, err := Compile(`p:noop()`)
	require.NoError(t, err)
	require.NotEmpty(t, expr.prefixPlan.prefixes)
	require.NoError(t, expr.prefixPlan.Validate(map[string]string{
		"p": "urn:test",
	}, false, nil))
}

type noNamespaceSchemaDecls struct{}

func (noNamespaceSchemaDecls) LookupSchemaElement(local, ns string) (string, bool) { return "", false }
func (noNamespaceSchemaDecls) LookupSchemaAttribute(local, ns string) (string, bool) {
	return "", false
}
func (noNamespaceSchemaDecls) LookupSchemaType(local, ns string) (string, bool) {
	return "xs:NOTATION", local == "nota" && ns == ""
}
func (noNamespaceSchemaDecls) IsSubtypeOf(typeName, baseTypeName string) bool { return false }
func (noNamespaceSchemaDecls) ValidateCast(value, typeName string) error      { return nil }
func (noNamespaceSchemaDecls) ListItemType(typeName string) (string, bool)    { return "", false }
func (noNamespaceSchemaDecls) UnionMemberTypes(typeName string) []string      { return nil }

func TestPrefixValidationAllowsImportedNoNamespaceSchemaType(t *testing.T) {
	expr, err := Compile(`$v instance of nota`)
	require.NoError(t, err)
	require.NoError(t, expr.prefixPlan.Validate(nil, true, noNamespaceSchemaDecls{}))
}
