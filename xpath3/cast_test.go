package xpath3_test

import (
	"errors"
	"math"
	"math/big"
	"testing"

	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

func TestCastFromString(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		targetType string
		check      func(t *testing.T, v xpath3.AtomicValue)
	}{
		{
			name:  "string to integer",
			input: "42", targetType: xpath3.TypeInteger,
			check: func(t *testing.T, v xpath3.AtomicValue) {
				t.Helper()
				require.Equal(t, int64(42), v.IntegerVal())
			},
		},
		{
			name:  "string to negative integer",
			input: "-7", targetType: xpath3.TypeInteger,
			check: func(t *testing.T, v xpath3.AtomicValue) {
				t.Helper()
				require.Equal(t, int64(-7), v.IntegerVal())
			},
		},
		{
			name:  "string to double",
			input: "3.14", targetType: xpath3.TypeDouble,
			check: func(t *testing.T, v xpath3.AtomicValue) {
				t.Helper()
				require.InDelta(t, 3.14, v.DoubleVal(), 0.001)
			},
		},
		{
			name:  "string to double INF",
			input: lexicon.FloatINF, targetType: xpath3.TypeDouble,
			check: func(t *testing.T, v xpath3.AtomicValue) {
				t.Helper()
				require.True(t, math.IsInf(v.DoubleVal(), 1))
			},
		},
		{
			name:  "string to double NaN",
			input: lexicon.FloatNaN, targetType: xpath3.TypeDouble,
			check: func(t *testing.T, v xpath3.AtomicValue) {
				t.Helper()
				require.True(t, math.IsNaN(v.DoubleVal()))
			},
		},
		{
			name:  "string to boolean true",
			input: lexicon.ValueTrue, targetType: xpath3.TypeBoolean,
			check: func(t *testing.T, v xpath3.AtomicValue) {
				t.Helper()
				require.True(t, v.BooleanVal())
			},
		},
		{
			name:  "string to boolean false",
			input: lexicon.ValueFalse, targetType: xpath3.TypeBoolean,
			check: func(t *testing.T, v xpath3.AtomicValue) {
				t.Helper()
				require.False(t, v.BooleanVal())
			},
		},
		{
			name:  "string to boolean 1",
			input: "1", targetType: xpath3.TypeBoolean,
			check: func(t *testing.T, v xpath3.AtomicValue) {
				t.Helper()
				require.True(t, v.BooleanVal())
			},
		},
		{
			name:  "string to decimal",
			input: "123.45", targetType: xpath3.TypeDecimal,
			check: func(t *testing.T, v xpath3.AtomicValue) {
				t.Helper()
				require.Equal(t, xpath3.TypeDecimal, v.TypeName)
			},
		},
		{
			name:  "string to date",
			input: "2024-01-15", targetType: xpath3.TypeDate,
			check: func(t *testing.T, v xpath3.AtomicValue) {
				t.Helper()
				tv := v.TimeVal()
				require.Equal(t, 2024, tv.Year())
				require.Equal(t, 1, int(tv.Month()))
				require.Equal(t, 15, tv.Day())
			},
		},
		{
			name:  "string to dateTime",
			input: "2024-01-15T10:30:00", targetType: xpath3.TypeDateTime,
			check: func(t *testing.T, v xpath3.AtomicValue) {
				t.Helper()
				tv := v.TimeVal()
				require.Equal(t, 2024, tv.Year())
				require.Equal(t, 10, tv.Hour())
				require.Equal(t, 30, tv.Minute())
			},
		},
		{
			// XSD 1.1: year 0000 is valid.
			name:  "string to date year zero",
			input: "0000-01-01", targetType: xpath3.TypeDate,
			check: func(t *testing.T, v xpath3.AtomicValue) {
				t.Helper()
				tv := v.TimeVal()
				require.Equal(t, 0, tv.Year())
				require.Equal(t, 1, int(tv.Month()))
				require.Equal(t, 1, tv.Day())
			},
		},
		{
			// XSD 1.1: year 0000 is a leap year.
			name:  "string to date year zero leap day",
			input: "0000-02-29", targetType: xpath3.TypeDate,
			check: func(t *testing.T, v xpath3.AtomicValue) {
				t.Helper()
				tv := v.TimeVal()
				require.Equal(t, 0, tv.Year())
				require.Equal(t, 2, int(tv.Month()))
				require.Equal(t, 29, tv.Day())
			},
		},
		{
			// XSD 1.1: year 0000 is valid.
			name:  "string to dateTime year zero",
			input: "0000-01-01T00:00:00", targetType: xpath3.TypeDateTime,
			check: func(t *testing.T, v xpath3.AtomicValue) {
				t.Helper()
				tv := v.TimeVal()
				require.Equal(t, 0, tv.Year())
				require.Equal(t, 1, int(tv.Month()))
				require.Equal(t, 1, tv.Day())
				require.Equal(t, 0, tv.Hour())
			},
		},
		{
			name:  "string to time",
			input: "10:30:00", targetType: xpath3.TypeTime,
			check: func(t *testing.T, v xpath3.AtomicValue) {
				t.Helper()
				tv := v.TimeVal()
				require.Equal(t, 10, tv.Hour())
				require.Equal(t, 30, tv.Minute())
			},
		},
		{
			name:  "string to duration",
			input: "P1Y2M3DT4H5M6S", targetType: xpath3.TypeDuration,
			check: func(t *testing.T, v xpath3.AtomicValue) {
				t.Helper()
				d := v.DurationVal()
				require.Equal(t, 14, d.Months) // 1*12 + 2
				require.InDelta(t, 3*86400+4*3600+5*60+6, d.Seconds, 0.001)
			},
		},
		{
			name:  "string to dayTimeDuration",
			input: "PT24H", targetType: xpath3.TypeDayTimeDuration,
			check: func(t *testing.T, v xpath3.AtomicValue) {
				t.Helper()
				d := v.DurationVal()
				require.Equal(t, 0, d.Months)
				require.InDelta(t, 86400.0, d.Seconds, 0.001)
			},
		},
		{
			name:  "string to anyURI",
			input: "http://example.com", targetType: xpath3.TypeAnyURI,
			check: func(t *testing.T, v xpath3.AtomicValue) {
				t.Helper()
				require.Equal(t, "http://example.com", v.StringVal())
			},
		},
		{
			name:  "string to hexBinary",
			input: "48656C6C6F", targetType: xpath3.TypeHexBinary,
			check: func(t *testing.T, v xpath3.AtomicValue) {
				t.Helper()
				require.Equal(t, []byte("Hello"), v.BytesVal())
			},
		},
		{
			name:  "whitespace trimmed",
			input: "  42  ", targetType: xpath3.TypeInteger,
			check: func(t *testing.T, v xpath3.AtomicValue) {
				t.Helper()
				require.Equal(t, int64(42), v.IntegerVal())
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v, err := xpath3.CastFromString(tc.input, tc.targetType)
			require.NoError(t, err)
			tc.check(t, v)
		})
	}
}

func TestCastFromStringErrors(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		targetType string
	}{
		{"bad integer", "abc", xpath3.TypeInteger},
		{"bad double", "not-a-number", xpath3.TypeDouble},
		{"bad boolean", "yes", xpath3.TypeBoolean},
		{"bad date", "not-a-date", xpath3.TypeDate},
		{"bad leap date", "2002-02-29", xpath3.TypeDate},
		{"bad dateTime", "not-a-datetime", xpath3.TypeDateTime},
		{"bad leap dateTime", "2002-02-29T10:30:00", xpath3.TypeDateTime},
		{"bad time", "not-a-time", xpath3.TypeTime},
		{"bad duration", "bad", xpath3.TypeDuration},
		{"bad hexBinary", "GG", xpath3.TypeHexBinary},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := xpath3.CastFromString(tc.input, tc.targetType)
			require.Error(t, err)
		})
	}
}

func TestCastAtomic(t *testing.T) {
	t.Run("integer to double", func(t *testing.T) {
		v := xpath3.AtomicValue{TypeName: xpath3.TypeInteger, Value: big.NewInt(42)}
		result, err := xpath3.CastAtomic(v, xpath3.TypeDouble)
		require.NoError(t, err)
		require.Equal(t, xpath3.TypeDouble, result.TypeName)
		require.InDelta(t, float64(42), result.DoubleVal(), 0)
	})

	t.Run("double to integer", func(t *testing.T) {
		v := xpath3.AtomicValue{TypeName: xpath3.TypeDouble, Value: xpath3.NewDouble(3.7)}
		result, err := xpath3.CastAtomic(v, xpath3.TypeInteger)
		require.NoError(t, err)
		require.Equal(t, int64(3), result.IntegerVal())
	})

	t.Run("integer to string", func(t *testing.T) {
		v := xpath3.AtomicValue{TypeName: xpath3.TypeInteger, Value: big.NewInt(42)}
		result, err := xpath3.CastAtomic(v, xpath3.TypeString)
		require.NoError(t, err)
		require.Equal(t, "42", result.StringVal())
	})

	t.Run("boolean to integer", func(t *testing.T) {
		v := xpath3.AtomicValue{TypeName: xpath3.TypeBoolean, Value: true}
		result, err := xpath3.CastAtomic(v, xpath3.TypeInteger)
		require.NoError(t, err)
		require.Equal(t, int64(1), result.IntegerVal())
	})

	t.Run("boolean to double", func(t *testing.T) {
		v := xpath3.AtomicValue{TypeName: xpath3.TypeBoolean, Value: false}
		result, err := xpath3.CastAtomic(v, xpath3.TypeDouble)
		require.NoError(t, err)
		require.InDelta(t, float64(0), result.DoubleVal(), 0)
	})

	t.Run("same type is identity", func(t *testing.T) {
		v := xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: testHello}
		result, err := xpath3.CastAtomic(v, xpath3.TypeString)
		require.NoError(t, err)
		require.Equal(t, v, result)
	})

	t.Run("untypedAtomic to integer", func(t *testing.T) {
		v := xpath3.AtomicValue{TypeName: xpath3.TypeUntypedAtomic, Value: "123"}
		result, err := xpath3.CastAtomic(v, xpath3.TypeInteger)
		require.NoError(t, err)
		require.Equal(t, int64(123), result.IntegerVal())
	})

	t.Run("integer to boolean", func(t *testing.T) {
		v := xpath3.AtomicValue{TypeName: xpath3.TypeInteger, Value: big.NewInt(0)}
		result, err := xpath3.CastAtomic(v, xpath3.TypeBoolean)
		require.NoError(t, err)
		require.False(t, result.BooleanVal())
	})

	t.Run("NaN to integer fails", func(t *testing.T) {
		v := xpath3.AtomicValue{TypeName: xpath3.TypeDouble, Value: xpath3.NewDouble(math.NaN())}
		_, err := xpath3.CastAtomic(v, xpath3.TypeInteger)
		require.Error(t, err)
	})

	t.Run("INF to integer fails", func(t *testing.T) {
		v := xpath3.AtomicValue{TypeName: xpath3.TypeDouble, Value: xpath3.NewDouble(math.Inf(1))}
		_, err := xpath3.CastAtomic(v, xpath3.TypeInteger)
		require.Error(t, err)
	})

	t.Run("hexBinary to base64Binary", func(t *testing.T) {
		v := xpath3.AtomicValue{TypeName: xpath3.TypeHexBinary, Value: []byte("Hello")}
		result, err := xpath3.CastAtomic(v, xpath3.TypeBase64Binary)
		require.NoError(t, err)
		require.Equal(t, []byte("Hello"), result.BytesVal())
	})

	t.Run("dateTime to date", func(t *testing.T) {
		v := xpath3.AtomicValue{TypeName: xpath3.TypeDateTime, Value: mustParseDateTime("2024-03-15T10:30:00")}
		result, err := xpath3.CastAtomic(v, xpath3.TypeDate)
		require.NoError(t, err)
		require.Equal(t, xpath3.TypeDate, result.TypeName)
		require.Equal(t, 2024, result.TimeVal().Year())
		require.Equal(t, 15, result.TimeVal().Day())
	})

	t.Run("string to boolean invalid", func(t *testing.T) {
		v := xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "maybe"}
		_, err := xpath3.CastAtomic(v, xpath3.TypeBoolean)
		require.Error(t, err)
	})

	// AtomicValue is public and mutable, so a caller can retag a no-timezone
	// xs:dateTime as xs:dateTimeStamp. CastAtomic to xs:dateTimeStamp must
	// enforce the mandatory-timezone invariant even on the same-type fast path.
	t.Run("dateTimeStamp identity enforces timezone", func(t *testing.T) {
		t.Run("missing timezone rejected", func(t *testing.T) {
			v := xpath3.AtomicValue{TypeName: xpath3.TypeDateTimeStamp, Value: mustParseDateTime("2024-03-15T10:30:00")}
			_, err := xpath3.CastAtomic(v, xpath3.TypeDateTimeStamp)
			require.Error(t, err)
			require.Equal(t, "FORG0001", castErrorCode(t, err))
		})

		t.Run("with timezone accepted", func(t *testing.T) {
			v := xpath3.AtomicValue{TypeName: xpath3.TypeDateTimeStamp, Value: mustParseDateTime("2024-03-15T10:30:00Z")}
			result, err := xpath3.CastAtomic(v, xpath3.TypeDateTimeStamp)
			require.NoError(t, err)
			require.Equal(t, v, result)
		})

		t.Run("non-time value rejected", func(t *testing.T) {
			v := xpath3.AtomicValue{TypeName: xpath3.TypeDateTimeStamp, Value: "not-a-time"}
			_, err := xpath3.CastAtomic(v, xpath3.TypeDateTimeStamp)
			require.Error(t, err)
			require.Equal(t, "FORG0001", castErrorCode(t, err))
		})
	})
}

// TestCastAtomicUserTypeNormalization covers the schema-derived USER type
// normalization in CastAtomic (a source whose TypeName is a non-XSD user type is
// normalized to its recorded builtin BaseType so it casts like its base) and its
// two guards.
func TestCastAtomicUserTypeNormalization(t *testing.T) {
	// A KNOWN XSD BaseType normalizes: a user xs:integer-derived atom casts to
	// xs:integer via the identity short-circuit, returning the value.
	t.Run("known XSD BaseType normalizes", func(t *testing.T) {
		v := xpath3.AtomicValue{TypeName: "Q{urn:x}MyInt", BaseType: xpath3.TypeInteger, Value: big.NewInt(5)}
		result, err := xpath3.CastAtomic(v, xpath3.TypeInteger)
		require.NoError(t, err)
		require.Equal(t, xpath3.TypeInteger, result.TypeName)
		require.Equal(t, int64(5), result.BigInt().Int64())
	})

	// PR859-CAST-EMPTY: an EMPTY TypeName is not a schema-derived USER type, so even a
	// KNOWN XSD BaseType must NOT trigger normalization — the atom stays opaque and
	// falls through to the normal XPTY0004 path on a non-string-like target.
	t.Run("empty TypeName with known BaseType stays opaque", func(t *testing.T) {
		v := xpath3.AtomicValue{BaseType: xpath3.TypeInteger, Value: big.NewInt(5)}
		_, err := xpath3.CastAtomic(v, xpath3.TypeInteger)
		require.Error(t, err)
		require.Equal(t, "XPTY0004", castErrorCode(t, err))
	})

	// PR859-CAST-NORM-002: a NON-XSD BaseType must NOT alter dispatch — it stays
	// opaque. Casting to the (also non-XSD) BaseType name must NOT become an identity
	// success via TypeName rewriting; it falls through to the normal XPTY0004 path.
	t.Run("non-XSD BaseType does not alter dispatch", func(t *testing.T) {
		v := xpath3.AtomicValue{TypeName: "Q{urn:x}Custom", BaseType: "Q{urn:x}Weird", Value: "x"}
		_, err := xpath3.CastAtomic(v, "Q{urn:x}Weird")
		require.Error(t, err)
		require.Equal(t, "XPTY0004", castErrorCode(t, err))
	})

	// PR859-CAST-DTS-001: normalization can surface xs:dateTimeStamp from a user
	// type; the post-normalization identity return must still enforce the
	// mandatory-timezone invariant, matching the built-in source path.
	t.Run("dateTimeStamp BaseType identity enforces timezone", func(t *testing.T) {
		t.Run("missing timezone rejected", func(t *testing.T) {
			v := xpath3.AtomicValue{TypeName: "Q{urn:x}MyDTS", BaseType: xpath3.TypeDateTimeStamp, Value: mustParseDateTime("2024-03-15T10:30:00")}
			_, err := xpath3.CastAtomic(v, xpath3.TypeDateTimeStamp)
			require.Error(t, err)
			require.Equal(t, "FORG0001", castErrorCode(t, err))
		})

		t.Run("with timezone accepted", func(t *testing.T) {
			v := xpath3.AtomicValue{TypeName: "Q{urn:x}MyDTS", BaseType: xpath3.TypeDateTimeStamp, Value: mustParseDateTime("2024-03-15T10:30:00Z")}
			result, err := xpath3.CastAtomic(v, xpath3.TypeDateTimeStamp)
			require.NoError(t, err)
			require.Equal(t, xpath3.TypeDateTimeStamp, result.TypeName)
		})
	})
}

// castErrorCode extracts the structured XPathError code from a CastAtomic error.
func castErrorCode(t *testing.T, err error) string {
	t.Helper()
	var xerr *xpath3.XPathError
	require.True(t, errors.As(err, &xerr), "error must be *xpath3.XPathError, got %T: %v", err, err)
	return xerr.Code
}

func TestAtomicToString(t *testing.T) {
	tests := []struct {
		name   string
		input  xpath3.AtomicValue
		expect string
	}{
		{"integer", xpath3.AtomicValue{TypeName: xpath3.TypeInteger, Value: big.NewInt(42)}, "42"},
		{"double", xpath3.AtomicValue{TypeName: xpath3.TypeDouble, Value: xpath3.NewDouble(3.14)}, "3.14"},
		{"double zero", xpath3.AtomicValue{TypeName: xpath3.TypeDouble, Value: xpath3.NewDouble(0.0)}, "0"},
		{"double INF", xpath3.AtomicValue{TypeName: xpath3.TypeDouble, Value: xpath3.NewDouble(math.Inf(1))}, lexicon.FloatINF},
		{"double NaN", xpath3.AtomicValue{TypeName: xpath3.TypeDouble, Value: xpath3.NewDouble(math.NaN())}, lexicon.FloatNaN},
		{lexicon.ValueTrue, xpath3.AtomicValue{TypeName: xpath3.TypeBoolean, Value: true}, lexicon.ValueTrue},
		{lexicon.ValueFalse, xpath3.AtomicValue{TypeName: xpath3.TypeBoolean, Value: false}, lexicon.ValueFalse},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := xpath3.CastAtomic(tc.input, xpath3.TypeString)
			require.NoError(t, err)
			require.Equal(t, tc.expect, result.StringVal())
		})
	}
}

func TestDurationParsing(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		months  int
		seconds float64
		neg     bool
	}{
		{"simple years", "P1Y", 12, 0, false},
		{"simple months", "P3M", 3, 0, false},
		{"simple days", "P5D", 0, 5 * 86400, false},
		{"simple hours", "PT2H", 0, 7200, false},
		{"simple minutes", "PT30M", 0, 1800, false},
		{"simple seconds", "PT45S", 0, 45, false},
		{"complex", "P1Y2M3DT4H5M6S", 14, 3*86400 + 4*3600 + 5*60 + 6, false},
		{"negative", "-P1Y", 12, 0, true},
		{"fractional seconds", "PT1.5S", 0, 1.5, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v, err := xpath3.CastFromString(tc.input, xpath3.TypeDuration)
			require.NoError(t, err)
			d := v.DurationVal()
			require.Equal(t, tc.months, d.Months)
			require.InDelta(t, tc.seconds, d.Seconds, 0.001)
			require.Equal(t, tc.neg, d.Negative)
		})
	}
}

// TestCastMalformedDateTimeStamp verifies that casting a malformed or retagged
// xs:dateTimeStamp (AtomicValue is public and mutable) to a date/time subtype
// reports a structured FORG0001 error instead of panicking. The subtype paths
// call TimeVal() unconditionally, so a non-time.Time or no-timezone value must
// be rejected up front.
func TestCastMalformedDateTimeStamp(t *testing.T) {
	// Source #1: TypeDateTimeStamp tag over a non-time.Time value.
	notTime := xpath3.AtomicValue{TypeName: xpath3.TypeDateTimeStamp, Value: "not a time"}

	// Source #2: TypeDateTimeStamp tag over a no-timezone time.Time, which
	// violates the mandatory-timezone invariant.
	noTZ, err := xpath3.CastFromString("2000-01-01T00:00:00", xpath3.TypeDateTime)
	require.NoError(t, err)
	noTZStamp := xpath3.AtomicValue{TypeName: xpath3.TypeDateTimeStamp, Value: noTZ.Value}

	targets := []string{
		xpath3.TypeDateTime,
		xpath3.TypeDate,
		xpath3.TypeTime,
		xpath3.TypeGYear,
		xpath3.TypeGYearMonth,
		xpath3.TypeGMonth,
		xpath3.TypeGMonthDay,
		xpath3.TypeGDay,
	}

	for _, src := range []struct {
		name string
		v    xpath3.AtomicValue
	}{
		{"non-time value", notTime},
		{"no-timezone value", noTZStamp},
	} {
		for _, target := range targets {
			t.Run(src.name+" to "+target, func(t *testing.T) {
				require.NotPanics(t, func() {
					_, err := xpath3.CastAtomic(src.v, target)
					require.Error(t, err)
					var xerr *xpath3.XPathError
					require.True(t, errors.As(err, &xerr), "error must be *xpath3.XPathError, got %T: %v", err, err)
					require.Equal(t, "FORG0001", xerr.Code)
				})
			})
		}
	}
}

// TestCastToDateTimeStampMalformedSource verifies that casting a non-time.Time
// xs:dateTime or xs:date source (AtomicValue is public and mutable) TO
// xs:dateTimeStamp reports a structured FORG0001 error instead of panicking.
// The dateTimeStamp target path routes such sources through TimeVal()/recursive
// casts that assume a time.Time payload.
func TestCastToDateTimeStampMalformedSource(t *testing.T) {
	for _, src := range []struct {
		name string
		v    xpath3.AtomicValue
	}{
		{"non-time dateTime", xpath3.AtomicValue{TypeName: xpath3.TypeDateTime, Value: "not a time"}},
		{"non-time date", xpath3.AtomicValue{TypeName: xpath3.TypeDate, Value: "not a time"}},
	} {
		t.Run(src.name, func(t *testing.T) {
			require.NotPanics(t, func() {
				_, err := xpath3.CastAtomic(src.v, xpath3.TypeDateTimeStamp)
				require.Error(t, err)
				var xerr *xpath3.XPathError
				require.True(t, errors.As(err, &xerr), "error must be *xpath3.XPathError, got %T: %v", err, err)
				require.Equal(t, "FORG0001", xerr.Code)
			})
		})
	}
}

func mustParseDateTime(s string) any {
	v, err := xpath3.CastFromString(s, xpath3.TypeDateTime)
	if err != nil {
		panic(err)
	}
	return v.Value
}

// Casting to integer-derived and string-derived XSD types exercises
// CastAtomic's integer-range checking (integerTypeRange / checkBigIntRange) and
// validateStringDerivedType, reached through the `cast as` operator.
func TestCast_IntegerSubtypeRange(t *testing.T) {
	ok := []string{
		`"100" cast as xs:byte`,
		`"100" cast as xs:short`,
		`"100" cast as xs:int`,
		`"100" cast as xs:long`,
		`"100" cast as xs:unsignedByte`,
		`"100" cast as xs:unsignedShort`,
		`"100" cast as xs:unsignedInt`,
		`"100" cast as xs:unsignedLong`,
		`"1" cast as xs:positiveInteger`,
		`"0" cast as xs:nonNegativeInteger`,
		`"0" cast as xs:nonPositiveInteger`,
		`"-1" cast as xs:negativeInteger`,
		// numeric sources, exercising the numeric->subtype path.
		`100 cast as xs:byte`,
		`100 cast as xs:int`,
	}
	for _, e := range ok {
		_, err := evaluate(t.Context(), nil, e)
		require.NoError(t, err, e)
	}

	outOfRange := []string{
		`"1000" cast as xs:byte`,       // > 127
		`"-1" cast as xs:unsignedByte`, // < 0
		`"0" cast as xs:positiveInteger`,
		`"1" cast as xs:negativeInteger`,
		`"1" cast as xs:nonPositiveInteger`,
		`"-1" cast as xs:nonNegativeInteger`,
		`"100000" cast as xs:short`,  // > 32767
		`"notanint" cast as xs:byte`, // unparseable
	}
	for _, e := range outOfRange {
		_, err := evaluate(t.Context(), nil, e)
		require.Error(t, err, e)
		var xpErr *xpath3.XPathError
		require.ErrorAs(t, err, &xpErr, e)
	}
}

func TestCast_StringDerivedTypes(t *testing.T) {
	ok := []string{
		`"abc" cast as xs:NCName`,
		`"abc" cast as xs:Name`,
		`"abc" cast as xs:NMTOKEN`,
		`"a b c" cast as xs:NMTOKENS`,
		`"abc" cast as xs:ID`,
		`"abc" cast as xs:IDREF`,
		`"a b" cast as xs:IDREFS`,
		`"abc" cast as xs:ENTITY`,
	}
	for _, e := range ok {
		_, err := evaluate(t.Context(), nil, e)
		require.NoError(t, err, e)
	}

	invalid := []string{
		`"a:b:c" cast as xs:NCName`, // colon not allowed in NCName
		`"" cast as xs:NMTOKENS`,    // empty list
		`"" cast as xs:IDREFS`,      // empty list
	}
	for _, e := range invalid {
		_, err := evaluate(t.Context(), nil, e)
		require.Error(t, err, e)
	}
}

// cast as xs:numeric (union) and xs:QName exercise the special-cased target
// branches in evalCastExpr.
func TestCast_UnionAndQName(t *testing.T) {
	// numeric: already-numeric returns as-is.
	r, err := evaluate(t.Context(), nil, `1 cast as xs:numeric instance of xs:integer`)
	require.NoError(t, err)
	b, ok := r.IsBoolean()
	require.True(t, ok)
	require.True(t, b)

	// numeric: string casts to double.
	r, err = evaluate(t.Context(), nil, `"2.5" cast as xs:numeric instance of xs:double`)
	require.NoError(t, err)
	b, ok = r.IsBoolean()
	require.True(t, ok)
	require.True(t, b)

	// QName cast from string.
	r, err = evaluate(t.Context(), nil, `string(fn:local-name-from-QName("foo" cast as xs:QName))`)
	require.NoError(t, err)
	require.Equal(t, "foo", r.StringValue())

	// Empty-sequence with ? cast modifier yields empty.
	r, err = evaluate(t.Context(), nil, `() cast as xs:integer?`)
	require.NoError(t, err)
	require.Equal(t, "", r.StringValue())

	// Multi-item cast -> XPTY0004.
	_, err = evaluate(t.Context(), nil, `(1, 2) cast as xs:integer`)
	require.Error(t, err)
}

// fn:string-to-codepoints / xs:double special values exercise atomicToString and
// formatting branches through the public string conversion surface.
func TestStringValueConversions(t *testing.T) {
	cases := map[string]string{
		`string(1)`:                 "1",
		`string(1.5)`:               want1Dot5,
		`string(true())`:            wantTrue,
		`string(xs:double("INF"))`:  wantINF,
		`string(xs:double("-INF"))`: "-INF",
		`string(xs:double("NaN"))`:  wantNaN,
	}
	for expr, want := range cases {
		r, err := evaluate(t.Context(), nil, expr)
		require.NoError(t, err, expr)
		require.Equal(t, want, r.StringValue(), expr)
	}
}

func TestCastableExprForms(t *testing.T) {
	cases := []struct {
		expr   string
		expect bool
	}{
		{`"1" castable as xs:integer`, true},
		{`"x" castable as xs:integer`, false},
		{`"1.5" castable as xs:decimal`, true},
		{`"2023-06-22" castable as xs:date`, true},
		{`"notadate" castable as xs:date`, false},
		{`"true" castable as xs:boolean`, true},
		{`() castable as xs:integer?`, true},
		{`() castable as xs:integer`, false},
		{`"abc" castable as xs:NCName`, true},
		{`"a:b:c" castable as xs:NCName`, false},
		// multi-item sequence is never castable to a single-item type.
		{`(1, 2) castable as xs:integer`, false},
		// numeric union type.
		{`1 castable as xs:numeric`, true},
		{`"1.5" castable as xs:numeric`, true},
		{`"x" castable as xs:numeric`, false},
		// list types: whitespace-separated members.
		{`"a b c" castable as xs:NMTOKENS`, true},
		{`"a b" castable as xs:IDREFS`, true},
		{`"" castable as xs:NMTOKENS`, false},
	}
	for _, tc := range cases {
		r, err := evaluate(t.Context(), nil, tc.expr)
		require.NoError(t, err, tc.expr)
		b, ok := r.IsBoolean()
		require.True(t, ok, tc.expr)
		require.Equal(t, tc.expect, b, tc.expr)
	}
}

func TestCastableExprAtomizationError(t *testing.T) {
	// Atomizing a non-atomizable operand (map/array/function) is a type error
	// (FOTY0013) that must PROPAGATE from `castable as`, not be swallowed into a
	// false result — W3C QT3 CastableAs666/668.
	for _, expr := range []string{
		`map{} castable as xs:integer`,
		// A nested array that flattens to reach a map still atomizes to FOTY0013.
		`[[], (), [[3, map{}]]] castable as xs:integer`,
	} {
		_, err := evaluate(t.Context(), nil, expr)
		require.Error(t, err, expr)
		var xerr *xpath3.XPathError
		require.True(t, errors.As(err, &xerr), "%s: error must be *xpath3.XPathError, got %T: %v", expr, err, err)
		require.Equal(t, "FOTY0013", xerr.Code, expr)
	}
}
