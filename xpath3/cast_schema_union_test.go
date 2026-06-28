package xpath3_test

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

const (
	testBuiltinUnion = "Q{}BuiltinUnion"
	testSmallInt     = "Q{}SmallInt"
	testUserUnion    = "Q{}UserUnion"
)

type unionCastDecls struct{}

func (unionCastDecls) LookupSchemaElement(local, ns string) (string, bool)   { return "", false }
func (unionCastDecls) LookupSchemaAttribute(local, ns string) (string, bool) { return "", false }
func (unionCastDecls) LookupSchemaType(local, ns string) (string, bool) {
	if ns != "" {
		return "", false
	}
	switch local {
	case "BuiltinUnion":
		return testBuiltinUnion, true
	case "SmallInt":
		return xpath3.TypeInt, true
	case "UserUnion":
		return testUserUnion, true
	default:
		return "", false
	}
}
func (unionCastDecls) IsSubtypeOf(typeName, baseTypeName string) bool {
	return typeName == baseTypeName
}
func (unionCastDecls) ValidateCast(_ context.Context, value, typeName string) error {
	if typeName != testSmallInt {
		return nil
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	if n > 10 {
		return errors.New("value exceeds SmallInt max")
	}
	return nil
}
func (d unionCastDecls) ValidateCastWithNS(ctx context.Context, value, typeName string, nsMap map[string]string) error {
	return d.ValidateCast(ctx, value, typeName)
}
func (unionCastDecls) ListItemType(typeName string) (string, bool) { return "", false }
func (unionCastDecls) UnionMemberTypes(typeName string) []string {
	switch typeName {
	case testBuiltinUnion:
		return []string{xpath3.TypeInt, xpath3.TypeString}
	case testUserUnion:
		return []string{testSmallInt}
	default:
		return nil
	}
}

func TestSchemaAwareCastableUserUnion(t *testing.T) {
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		SchemaDeclarations(unionCastDecls{})

	res, err := eval.Evaluate(t.Context(), mustCompile(t, `'5' castable as BuiltinUnion`), nil)
	require.NoError(t, err)
	require.Equal(t, "true", res.StringValue())

	res, err = eval.Evaluate(t.Context(), mustCompile(t, `'5' castable as UserUnion`), nil)
	require.NoError(t, err)
	require.Equal(t, "true", res.StringValue())

	res, err = eval.Evaluate(t.Context(), mustCompile(t, `'12' castable as UserUnion`), nil)
	require.NoError(t, err)
	require.Equal(t, "false", res.StringValue())
}

func TestSchemaAwareCastUserUnionUsesUserMember(t *testing.T) {
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		SchemaDeclarations(unionCastDecls{})

	res, err := eval.Evaluate(t.Context(), mustCompile(t, `'5' cast as UserUnion`), nil)
	require.NoError(t, err)
	require.Equal(t, "5", res.StringValue())

	av, ok := res.Sequence().Get(0).(xpath3.AtomicValue)
	require.True(t, ok)
	require.Equal(t, testSmallInt, av.TypeName)

	_, err = eval.Evaluate(t.Context(), mustCompile(t, `'12' cast as UserUnion`), nil)
	require.Error(t, err)
}
