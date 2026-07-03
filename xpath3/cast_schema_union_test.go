package xpath3_test

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

const (
	testBuiltinUnion = "Q{}BuiltinUnion"
	testDateOrString = "Q{}DateOrString"
	testMyDate       = "Q{}MyDate"
	testMyTime       = "Q{}MyTime"
	testRejectUnion  = "Q{}RejectUnion"
	testSmallInt     = "Q{}SmallInt"
	testTwoDigit     = "Q{}TwoDigit"
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
	case "DateOrString":
		return testDateOrString, true
	case "MyDate":
		return xpath3.TypeDate, true
	case "MyTime":
		return xpath3.TypeTime, true
	case "RejectUnion":
		return testRejectUnion, true
	case "SmallInt":
		return xpath3.TypeInt, true
	case "TwoDigit":
		return xpath3.TypeInteger, true
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
	if typeName == testRejectUnion && value == "7" {
		return errors.New("RejectUnion rejects 7")
	}
	if typeName == testTwoDigit {
		if len(value) != 2 {
			return errors.New("TwoDigit requires exactly two lexical digits")
		}
		for _, r := range value {
			if r < '0' || r > '9' {
				return errors.New("TwoDigit requires decimal digits")
			}
		}
		return nil
	}
	if typeName == testMyDate {
		if strings.Contains(value, "T") {
			return errors.New("MyDate requires date lexical form")
		}
		_, err := xpath3.CastFromString(value, xpath3.TypeDate)
		return err
	}
	if typeName == testDateOrString {
		if strings.Contains(value, "T") {
			return errors.New("DateOrString rejects dateTime lexical forms")
		}
		return nil
	}
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
func (unionCastDecls) IsSubstitutionGroupMember(memberLocal, memberNS, headLocal, headNS string) bool {
	return false
}
func (unionCastDecls) UnionMemberTypes(typeName string) []string {
	switch typeName {
	case testBuiltinUnion:
		return []string{xpath3.TypeInt, xpath3.TypeString}
	case testDateOrString:
		return []string{testMyDate, xpath3.TypeString}
	case testRejectUnion:
		return []string{testSmallInt}
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

	res, err = eval.Evaluate(t.Context(), mustCompile(t, `'7' castable as RejectUnion`), nil)
	require.NoError(t, err)
	require.Equal(t, "false", res.StringValue())
}

func TestSchemaAwareCastUserUnionValidatesTypedMemberResult(t *testing.T) {
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		SchemaDeclarations(unionCastDecls{})

	res, err := eval.Evaluate(t.Context(), mustCompile(t, `xs:dateTime('2020-01-02T03:04:05Z') castable as DateOrString`), nil)
	require.NoError(t, err)
	require.Equal(t, "true", res.StringValue())

	res, err = eval.Evaluate(t.Context(), mustCompile(t, `xs:dateTime('2020-01-02T03:04:05Z') cast as DateOrString`), nil)
	require.NoError(t, err)
	require.NotContains(t, res.StringValue(), "T")

	av, ok := res.Sequence().Get(0).(xpath3.AtomicValue)
	require.True(t, ok)
	require.Equal(t, testMyDate, av.TypeName)
	require.Equal(t, xpath3.TypeDate, av.BaseType)

	res, err = eval.Evaluate(t.Context(), mustCompile(t, `'2020-01-02T03:04:05Z' castable as DateOrString`), nil)
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

	_, err = eval.Evaluate(t.Context(), mustCompile(t, `'7' cast as RejectUnion`), nil)
	require.Error(t, err)
}

func TestSchemaAwareCastPreservesBuiltinBase(t *testing.T) {
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		SchemaDeclarations(unionCastDecls{})

	res, err := eval.Evaluate(t.Context(), mustCompile(t, `'10:00:00' cast as MyTime`), nil)
	require.NoError(t, err)

	av, ok := res.Sequence().Get(0).(xpath3.AtomicValue)
	require.True(t, ok)
	require.Equal(t, testMyTime, av.TypeName)
	require.Equal(t, xpath3.TypeTime, av.BaseType)

	res, err = eval.Evaluate(t.Context(), mustCompile(t, `('10:00:00' cast as MyTime) = xs:time('10:00:00')`), nil)
	require.NoError(t, err)
	require.Equal(t, "true", res.StringValue())
}

func TestSchemaAwareCastUserTypeValidatesTypedCastResult(t *testing.T) {
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		SchemaDeclarations(unionCastDecls{})

	res, err := eval.Evaluate(t.Context(), mustCompile(t, `xs:dateTime('2020-01-02T03:04:05Z') castable as MyDate`), nil)
	require.NoError(t, err)
	require.Equal(t, "true", res.StringValue())

	res, err = eval.Evaluate(t.Context(), mustCompile(t, `xs:dateTime('2020-01-02T03:04:05Z') cast as MyDate`), nil)
	require.NoError(t, err)
	require.NotContains(t, res.StringValue(), "T")

	av, ok := res.Sequence().Get(0).(xpath3.AtomicValue)
	require.True(t, ok)
	require.Equal(t, testMyDate, av.TypeName)
	require.Equal(t, xpath3.TypeDate, av.BaseType)
}

func TestSchemaAwareCastUserTypeValidatesSourceLexical(t *testing.T) {
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		SchemaDeclarations(unionCastDecls{})

	res, err := eval.Evaluate(t.Context(), mustCompile(t, `'05' castable as TwoDigit`), nil)
	require.NoError(t, err)
	require.Equal(t, "true", res.StringValue())

	res, err = eval.Evaluate(t.Context(), mustCompile(t, `'5' castable as TwoDigit`), nil)
	require.NoError(t, err)
	require.Equal(t, "false", res.StringValue())

	res, err = eval.Evaluate(t.Context(), mustCompile(t, `'05' cast as TwoDigit`), nil)
	require.NoError(t, err)
	require.Equal(t, "5", res.StringValue())

	av, ok := res.Sequence().Get(0).(xpath3.AtomicValue)
	require.True(t, ok)
	require.Equal(t, testTwoDigit, av.TypeName)
	require.Equal(t, xpath3.TypeInteger, av.BaseType)
}

func TestSchemaAwareGeneralComparisonCastsUntypedToUserType(t *testing.T) {
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		SchemaDeclarations(unionCastDecls{})

	res, err := eval.Evaluate(t.Context(), mustCompile(t, `('5' cast as SmallInt) = xs:untypedAtomic('5')`), nil)
	require.NoError(t, err)
	require.Equal(t, "true", res.StringValue())

	_, err = eval.Evaluate(t.Context(), mustCompile(t, `('5' cast as SmallInt) = xs:untypedAtomic('12')`), nil)
	require.Error(t, err)
}
