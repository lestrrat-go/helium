package xslt3

import (
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// derivedAtomic builds a synthetic schema-derived atomic value: a value cast to a
// built-in primitive type whose TypeName is then re-tagged with a user-defined
// name and whose BaseType records the built-in primitive. This mirrors what node
// atomization produces for a type derived from a built-in (see AtomizeItem).
//
// It exists because the xsd schema compiler does not register
// xs:dayTimeDuration / xs:yearMonthDuration as derivable base types, so a derived
// duration cannot be materialized through the full import-schema pipeline; the
// conversion helpers are exercised directly instead.
func derivedAtomic(t *testing.T, lexical, builtin, derivedName string) xpath3.AtomicValue {
	t.Helper()
	av, err := xpath3.CastFromString(lexical, builtin)
	require.NoError(t, err)
	av.TypeName = derivedName
	av.BaseType = builtin
	return av
}

// XSLT3-102 r7: the sort comparison-value conversion must be BaseType-aware so a
// schema-derived date/time/duration sorts by its typed value, matching what the
// XTDE1030 validation gate accepts (it promotes via
// xpath3.ValueCompare/PromoteSchemaType). Pre-r7 atomicToNumericSortValue
// switched on the EXACT TypeName and returned ok=false for any derived type, so
// such a key sorted as text or NaN instead of by value.
func TestAtomicToNumericSortValueDerivedTypes(t *testing.T) {
	t.Run("dayTimeDuration", func(t *testing.T) {
		mk := func(lex string) float64 {
			sv, ok := atomicToNumericSortValue(
				derivedAtomic(t, lex, xpath3.TypeDayTimeDuration, "Q{urn:my}myDur"), nil)
			require.True(t, ok, "derived dayTimeDuration must convert by value")
			require.Equal(t, sortValueNumber, sv.kind)
			return sv.num
		}
		// By value PT1H < PT3H < PT20H (text order would be PT1H < PT20H < PT3H).
		require.Less(t, mk("PT1H"), mk("PT3H"))
		require.Less(t, mk("PT3H"), mk("PT20H"))
	})

	t.Run("yearMonthDuration", func(t *testing.T) {
		mk := func(lex string) float64 {
			sv, ok := atomicToNumericSortValue(
				derivedAtomic(t, lex, xpath3.TypeYearMonthDuration, "Q{urn:my}myYM"), nil)
			require.True(t, ok, "derived yearMonthDuration must convert by value")
			require.Equal(t, sortValueNumber, sv.kind)
			return sv.num
		}
		// By value P1Y (12mo) < P18M (18mo) < P2Y (24mo).
		require.Less(t, mk("P1Y"), mk("P18M"))
		require.Less(t, mk("P18M"), mk("P2Y"))
	})

	t.Run("date", func(t *testing.T) {
		mk := func(lex string) float64 {
			sv, ok := atomicToNumericSortValue(
				derivedAtomic(t, lex, xpath3.TypeDate, "Q{urn:my}myDate"), nil)
			require.True(t, ok, "derived date must convert by value")
			require.Equal(t, sortValueNumber, sv.kind)
			return sv.num
		}
		// Chronological value order reverses the BCE text order.
		require.Less(t, mk("-0100-01-01"), mk("-0044-01-01"))
		require.Less(t, mk("-0044-01-01"), mk("0050-01-01"))
	})

	t.Run("dateTime", func(t *testing.T) {
		mk := func(lex string) float64 {
			sv, ok := atomicToNumericSortValue(
				derivedAtomic(t, lex, xpath3.TypeDateTime, "Q{urn:my}myDT"), nil)
			require.True(t, ok, "derived dateTime must convert by value")
			require.Equal(t, sortValueNumber, sv.kind)
			return sv.num
		}
		require.Less(t, mk("2019-01-01T00:00:00"), mk("2020-01-01T00:00:00"))
	})

	t.Run("non-orderable derived string stays unconverted", func(t *testing.T) {
		_, ok := atomicToNumericSortValue(
			derivedAtomic(t, "abc", xpath3.TypeString, "Q{urn:my}myStr"), nil)
		require.False(t, ok)
	})
}

// XSLT3-102 r7: applyAutoSortPromotion must also be BaseType-aware. A default
// (auto) data-type level whose first key is a schema-derived date/duration must
// flip to number-auto and rewrite the sort value numerically, just as it does for
// the built-in types. Pre-r7 the exact-TypeName switches missed the derived type,
// leaving the level in text mode.
func TestApplyAutoSortPromotionDerivedTypes(t *testing.T) {
	cases := []struct {
		name    string
		lexical string
		builtin string
		derived string
	}{
		{"date", "2020-01-01", xpath3.TypeDate, "Q{urn:my}myDate"},
		{"dayTimeDuration", "PT3H", xpath3.TypeDayTimeDuration, "Q{urn:my}myDur"},
		{"yearMonthDuration", "P1Y", xpath3.TypeYearMonthDuration, "Q{urn:my}myYM"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			av := derivedAtomic(t, tc.lexical, tc.builtin, tc.derived)
			sv := sortValue{kind: sortValueText, str: tc.lexical}
			mode := dataTypeAuto
			applyAutoSortPromotion(&sv, av, &mode, nil)
			require.Equal(t, dataTypeNumberAuto, mode,
				"derived %s must flip the level to number-auto", tc.name)
			require.Equal(t, sortValueNumber, sv.kind,
				"derived %s must rewrite the sort value numerically", tc.name)
			// The original derived TypeName must be preserved for the gate.
			require.Equal(t, tc.derived, sv.typeName)
		})
	}
}
