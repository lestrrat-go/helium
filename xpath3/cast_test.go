package xpath3_test

import (
	"math"
	"math/big"
	"testing"

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
			input: "INF", targetType: xpath3.TypeDouble,
			check: func(t *testing.T, v xpath3.AtomicValue) {
				t.Helper()
				require.True(t, math.IsInf(v.DoubleVal(), 1))
			},
		},
		{
			name:  "string to double NaN",
			input: "NaN", targetType: xpath3.TypeDouble,
			check: func(t *testing.T, v xpath3.AtomicValue) {
				t.Helper()
				require.True(t, math.IsNaN(v.DoubleVal()))
			},
		},
		{
			name:  "string to boolean true",
			input: "true", targetType: xpath3.TypeBoolean,
			check: func(t *testing.T, v xpath3.AtomicValue) {
				t.Helper()
				require.True(t, v.BooleanVal())
			},
		},
		{
			name:  "string to boolean false",
			input: "false", targetType: xpath3.TypeBoolean,
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
		{"bad dateTime", "not-a-datetime", xpath3.TypeDateTime},
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
		v := xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "hello"}
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
		{"double INF", xpath3.AtomicValue{TypeName: xpath3.TypeDouble, Value: xpath3.NewDouble(math.Inf(1))}, "INF"},
		{"double NaN", xpath3.AtomicValue{TypeName: xpath3.TypeDouble, Value: xpath3.NewDouble(math.NaN())}, "NaN"},
		{"true", xpath3.AtomicValue{TypeName: xpath3.TypeBoolean, Value: true}, "true"},
		{"false", xpath3.AtomicValue{TypeName: xpath3.TypeBoolean, Value: false}, "false"},
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

func mustParseDateTime(s string) any {
	v, err := xpath3.CastFromString(s, xpath3.TypeDateTime)
	if err != nil {
		panic(err)
	}
	return v.Value
}
