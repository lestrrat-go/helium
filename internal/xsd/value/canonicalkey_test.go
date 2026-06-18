package value_test

import (
	"testing"

	"github.com/lestrrat-go/helium/internal/xsd/value"
	"github.com/stretchr/testify/require"
)

// TestCanonicalKey covers value-space canonicalization used for identity-
// constraint keys, focusing on whiteSpace handling per type and float vs double
// precision.
func TestCanonicalKey(t *testing.T) {
	key := func(s, typ string) string {
		k, _ := value.CanonicalKey(s, typ)
		return k
	}

	// xs:string is whiteSpace=preserve: leading/trailing space is significant,
	// so distinct strings must NOT collide.
	require.NotEqual(t, key("a", "string"), key("a ", "string"), `"a" and "a " must differ for xs:string`)
	require.Equal(t, "a ", key("a ", "string"), "xs:string must be preserved verbatim")

	// Collapse types (token, etc.) and list types collapse internal whitespace,
	// so separator-only differences ARE value-equal.
	require.Equal(t, key("a b", "token"), key("a  b", "token"), "xs:token collapses internal whitespace")
	require.Equal(t, key("x y", "IDREFS"), key("x  y", "IDREFS"), "xs:IDREFS collapses internal whitespace")
	require.Equal(t, key("x y", "IDREFS"), key(" x y ", "IDREFS"), "xs:IDREFS trims leading/trailing")

	// xs:float uses 32-bit IEEE: 16777216 and 16777217 round to the same float32
	// (2^24 boundary) and must collide, while as xs:double they stay distinct.
	require.Equal(t, key("16777216", "float"), key("16777217", "float"), "values equal in float32 must collide for xs:float")
	require.NotEqual(t, key("16777216", "double"), key("16777217", "double"), "distinct doubles must not collide for xs:double")

	// Signed zero: -0 and 0 are value-equal, so they must produce the same key.
	require.Equal(t, key("0", "double"), key("-0", "double"), "-0 and 0 must collide for xs:double")
	require.Equal(t, key("0", "float"), key("-0", "float"), "-0 and 0 must collide for xs:float")
	require.Equal(t, key("0.0", "double"), key("-0.0", "double"), "-0.0 and 0.0 must collide")

	// Huge expanded years use arbitrary-precision year keys, so timezone-
	// equivalent forms canonicalize to the same key (used for enumeration and
	// fixed-value identity), while distinct huge years must not collide.
	require.Equal(t,
		key(hugeYearPlus1+"Z", "gYear"),
		key(hugeYearPlus1+"+00:00", "gYear"),
		"TZ-equivalent huge gYear values must canonicalize equal")
	require.NotEqual(t,
		key(hugeYear, "gYear"),
		key(hugeYearPlus1, "gYear"),
		"distinct huge gYear values must not collide")
}

// TestCanonicalKeySignedYearInvalid verifies that a leading '+' on the year is
// not accepted as a valid date/dateTime lexical form: it must NOT canonicalize
// as valid, and must NOT produce a key equal to the unsigned form.
func TestCanonicalKeySignedYearInvalid(t *testing.T) {
	_ = t.Context()

	plusDate, okPlusDate := value.CanonicalKey("+2023-01-01", "date")
	require.False(t, okPlusDate, `"+2023-01-01" must not canonicalize as a valid xs:date`)

	unsignedDate, okUnsignedDate := value.CanonicalKey("2023-01-01", "date")
	require.True(t, okUnsignedDate, `"2023-01-01" must canonicalize as a valid xs:date`)
	require.NotEqual(t, unsignedDate, plusDate, `"+2023-01-01" must not produce the same key as "2023-01-01"`)

	plusDT, okPlusDT := value.CanonicalKey("+2023-01-01T00:00:00", "dateTime")
	require.False(t, okPlusDT, `"+2023-01-01T00:00:00" must not canonicalize as a valid xs:dateTime`)

	unsignedDT, okUnsignedDT := value.CanonicalKey("2023-01-01T00:00:00", "dateTime")
	require.True(t, okUnsignedDT, `"2023-01-01T00:00:00" must canonicalize as a valid xs:dateTime`)
	require.NotEqual(t, unsignedDT, plusDT, `"+2023-01-01T00:00:00" must not produce the same key as "2023-01-01T00:00:00"`)
}

// TestCanonicalKeyStrictDateValidation verifies that CanonicalKey validates the
// value against the strict lexical space before canonicalizing, so malformed
// date/time inputs (bad timezone, trailing junk, out-of-range fields) yield
// ok=false rather than being silently canonicalized by the lenient internal
// parsers.
func TestCanonicalKeyStrictDateValidation(t *testing.T) {
	_ = t.Context()

	_, okBadTZ := value.CanonicalKey("2023-01-01+99:99", "date")
	require.False(t, okBadTZ, `"2023-01-01+99:99" has an out-of-range timezone and must not canonicalize`)

	_, okJunkTZ := value.CanonicalKey("2023-01-01Zjunk", "date")
	require.False(t, okJunkTZ, `"2023-01-01Zjunk" has trailing junk after Z and must not canonicalize`)

	_, okLeap := value.CanonicalKey("2023-02-29", "date")
	require.False(t, okLeap, `"2023-02-29" is not a leap year and must not canonicalize`)

	_, okMonthDay := value.CanonicalKey("--02-30", "gMonthDay")
	require.False(t, okMonthDay, `"--02-30" is not a valid gMonthDay and must not canonicalize`)

	// A valid huge-year date (leap year, Feb 29) still canonicalizes correctly
	// and is not regressed by the strict pre-validation.
	_, okHuge := value.CanonicalKey("999999999999999999999996-02-29", "date")
	require.True(t, okHuge, `valid huge-year leap date must still canonicalize`)
}

// TestCanonicalKeyStrictNumericValidation verifies CanonicalKey gates the
// numeric, float and boolean value-comparable types on the strict lexical space
// (including range checks for bounded integer subtypes) before producing a key,
// matching the comparison path.
func TestCanonicalKeyStrictNumericValidation(t *testing.T) {
	_ = t.Context()

	_, okIntFrac := value.CanonicalKey("1.0", "integer")
	require.False(t, okIntFrac, `"1.0" is not a valid xs:integer lexical form`)

	_, okIntRange := value.CanonicalKey("2147483648", "int")
	require.False(t, okIntRange, `"2147483648" is out of range for xs:int`)

	_, okUByte := value.CanonicalKey("-1", "unsignedByte")
	require.False(t, okUByte, `"-1" is out of range for xs:unsignedByte`)

	_, okDecimal := value.CanonicalKey("1/2", "decimal")
	require.False(t, okDecimal, `"1/2" is not a valid xs:decimal lexical form`)

	_, okFloat := value.CanonicalKey("Inf", "float")
	require.False(t, okFloat, `"Inf" is not a valid xs:float lexical form`)

	// NBSP padding is not XSD whitespace, so it is not trimmed and the value
	// stays invalid (Go's strings.TrimSpace would have wrongly accepted it).
	_, okNBSP := value.CanonicalKey(" 1 ", "integer")
	require.False(t, okNBSP, `NBSP-padded integer must not canonicalize`)

	// A valid integer with XSD-whitespace padding still canonicalizes.
	plus5, okPlus5 := value.CanonicalKey(" +5 ", "integer")
	require.True(t, okPlus5, `XSD-whitespace-padded "+5" must canonicalize`)
	five, _ := value.CanonicalKey("5", "integer")
	require.Equal(t, five, plus5, `"+5" and "5" must canonicalize equal`)
}

// TestCanonicalKeyBinary verifies hexBinary and base64Binary canonicalize to a
// stable byte key so value-equal but lexically distinct forms collide, while
// invalid forms yield ok=false.
func TestCanonicalKeyBinary(t *testing.T) {
	_ = t.Context()

	hexUpper, okHU := value.CanonicalKey("0A", "hexBinary")
	require.True(t, okHU)
	hexLower, okHL := value.CanonicalKey("0a", "hexBinary")
	require.True(t, okHL)
	require.Equal(t, hexUpper, hexLower, `"0A" and "0a" must canonicalize equal`)

	b64Plain, okBP := value.CanonicalKey("YWJj", "base64Binary")
	require.True(t, okBP)
	b64Spaced, okBS := value.CanonicalKey("YW Jj", "base64Binary")
	require.True(t, okBS)
	require.Equal(t, b64Plain, b64Spaced, `"YWJj" and "YW Jj" must canonicalize equal`)

	hex0B, ok0B := value.CanonicalKey("0B", "hexBinary")
	require.True(t, ok0B)
	require.NotEqual(t, hexUpper, hex0B, `distinct byte values must not collide`)

	_, okBadHex := value.CanonicalKey("0G", "hexBinary")
	require.False(t, okBadHex, `"0G" is not valid hexBinary`)

	_, okBadB64 := value.CanonicalKey("@@@@", "base64Binary")
	require.False(t, okBadB64, `"@@@@" is not valid base64Binary`)
}

// TestCanonicalKeyHour24 verifies the end-of-day 24:00:00 form canonicalizes to
// the same key as 00:00:00 of the next day for xs:dateTime, and to start-of-day
// for xs:time.
func TestCanonicalKeyHour24(t *testing.T) {
	_ = t.Context()

	dt24, ok24 := value.CanonicalKey("2024-01-01T24:00:00", "dateTime")
	require.True(t, ok24)
	dt00, ok00 := value.CanonicalKey("2024-01-02T00:00:00", "dateTime")
	require.True(t, ok00)
	require.Equal(t, dt00, dt24, `"...T24:00:00" must canonicalize to next-day 00:00:00`)

	t24, okT24 := value.CanonicalKey("24:00:00", "time")
	require.True(t, okT24)
	t00, okT00 := value.CanonicalKey("00:00:00", "time")
	require.True(t, okT00)
	require.Equal(t, t00, t24, `xs:time "24:00:00" must canonicalize to "00:00:00"`)
}

// TestCanonicalKeyDuration verifies that xs:duration values equal in the value
// space (months, seconds) canonicalize to the same key, so IDC unique/key fields
// over xs:duration detect duplicates that differ only lexically.
func TestCanonicalKeyDuration(t *testing.T) {
	_ = t.Context()

	day, okDay := value.CanonicalKey("P1D", "duration")
	require.True(t, okDay, `"P1D" must canonicalize as a valid xs:duration`)
	hours, okHours := value.CanonicalKey("PT24H", "duration")
	require.True(t, okHours, `"PT24H" must canonicalize as a valid xs:duration`)
	require.Equal(t, day, hours, `"P1D" and "PT24H" are value-equal and must canonicalize equal`)

	// A leading-XSD-whitespace form collapses to the same key.
	spaced, okSpaced := value.CanonicalKey("  P1D  ", "duration")
	require.True(t, okSpaced)
	require.Equal(t, day, spaced, "XSD-whitespace padding must not change the duration key")

	// Distinct durations must not collide.
	year, okYear := value.CanonicalKey("P1Y", "duration")
	require.True(t, okYear)
	require.NotEqual(t, day, year, `"P1Y" (12 months) and "P1D" must not collide`)

	// Huge month/day components (far beyond int64 range) are parsed as big.Int /
	// big.Rat, so a valid huge-component duration canonicalizes and equal values
	// collide rather than failing to parse.
	huge, okHuge := value.CanonicalKey("P999999999999999999999999Y", "duration")
	require.True(t, okHuge, "huge-year duration must canonicalize as valid")
	hugeAlt, okHugeAlt := value.CanonicalKey("P999999999999999999999998Y12M", "duration")
	require.True(t, okHugeAlt)
	require.Equal(t, huge, hugeAlt, `"…998Y12M" equals "…999Y" in months and must collide`)
	require.NotEqual(t, year, huge, "distinct huge duration must not collide with P1Y")

	// An invalid duration must not canonicalize as valid.
	_, okBad := value.CanonicalKey("P", "duration")
	require.False(t, okBad, `"P" is not a valid xs:duration`)
	// NBSP padding is not XSD whitespace, so it stays invalid.
	_, okNBSP := value.CanonicalKey(" P1D", "duration")
	require.False(t, okNBSP, "NBSP-padded duration must not canonicalize as valid")
}

// TestCanonicalKeyTimeTimezone verifies that timezoned xs:time values equal in
// the value space canonicalize to the same key. CanonicalKey must apply the same
// synthetic reference date compareTime uses so an offset crossing midnight does
// not shift the date fields of the key.
func TestCanonicalKeyTimeTimezone(t *testing.T) {
	_ = t.Context()

	plus, okPlus := value.CanonicalKey("11:30:00+01:00", "time")
	require.True(t, okPlus)
	utc, okUTC := value.CanonicalKey("10:30:00Z", "time")
	require.True(t, okUTC)
	require.Equal(t, utc, plus, `"11:30:00+01:00" and "10:30:00Z" are the same xs:time instant`)
}

// TestCanonicalKeyFractionalPrecision verifies that fractional seconds are
// preserved EXACTLY: two distinct valid lexicals differing only in trailing
// fractional precision must NOT collide. Holding seconds as a float64 rounded
// both to the same bits and produced identical keys; an exact rational keeps
// them distinct.
func TestCanonicalKeyFractionalPrecision(t *testing.T) {
	_ = t.Context()

	const lo = "0.1"
	const hi = "0.1000000000000000000000000000000000001"

	durLo, okDurLo := value.CanonicalKey("PT"+lo+"S", "duration")
	require.True(t, okDurLo)
	durHi, okDurHi := value.CanonicalKey("PT"+hi+"S", "duration")
	require.True(t, okDurHi)
	require.NotEqual(t, durLo, durHi, "durations differing in trailing fractional seconds must not collide")

	timeLo, okTimeLo := value.CanonicalKey("10:30:0"+lo, "time")
	require.True(t, okTimeLo)
	timeHi, okTimeHi := value.CanonicalKey("10:30:0"+hi, "time")
	require.True(t, okTimeHi)
	require.NotEqual(t, timeLo, timeHi, "xs:time values differing in trailing fractional seconds must not collide")

	dtLo, okDTLo := value.CanonicalKey("2023-01-15T10:30:0"+lo, "dateTime")
	require.True(t, okDTLo)
	dtHi, okDTHi := value.CanonicalKey("2023-01-15T10:30:0"+hi, "dateTime")
	require.True(t, okDTHi)
	require.NotEqual(t, dtLo, dtHi, "xs:dateTime values differing in trailing fractional seconds must not collide")

	// Equal-but-differently-spelled fractions must still collide (value identity).
	dtEqA, _ := value.CanonicalKey("2023-01-15T10:30:00.10", "dateTime")
	dtEqB, _ := value.CanonicalKey("2023-01-15T10:30:00.1", "dateTime")
	require.Equal(t, dtEqA, dtEqB, "0.10 and 0.1 fractional seconds are the same value and must collide")
}

// TestCompareFractionalPrecision verifies the same exact-precision guarantee
// through the public Compare API for durations and fractional-second date/time
// types: distinct values must compare non-equal.
func TestCompareFractionalPrecision(t *testing.T) {
	_ = t.Context()

	const lo = "0.1"
	const hi = "0.1000000000000000000000000000000000001"

	durCmp, okDur := value.Compare("PT"+lo+"S", "PT"+hi+"S", "duration")
	require.True(t, okDur)
	require.NotEqual(t, 0, durCmp, "durations differing in trailing fractional seconds must not compare equal")

	timeCmp, okTime := value.Compare("10:30:0"+lo, "10:30:0"+hi, "time")
	require.True(t, okTime)
	require.Equal(t, -1, timeCmp, "lower fractional second must compare less for xs:time")

	dtCmp, okDT := value.Compare("2023-01-15T10:30:0"+lo, "2023-01-15T10:30:0"+hi, "dateTime")
	require.True(t, okDT)
	require.Equal(t, -1, dtCmp, "lower fractional second must compare less for xs:dateTime")
}
