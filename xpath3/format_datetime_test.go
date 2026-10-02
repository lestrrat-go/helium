package xpath3_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// formatDateTimeWidthValue is the dateTime every width test formats:
// year 2012, month 5, day 18, hour 13, minute 7, second 9, and a
// fractional second of .5.
const formatDateTimeWidthValue = `xs:dateTime('2012-05-18T13:07:09.5')`

// formatDateTimeMaxMinWidth mirrors the documented limit on a picture's
// minimum width (format_datetime.go maxPictureMinWidth).
const formatDateTimeMaxMinWidth = 1000

// formatDateTimeWidths covers the widths around the int64 year modulus
// (19 digits), the old 10^64 wrap to zero, the minimum-width limit, and a
// value past 2^32 that a 32-bit int saturates.
var formatDateTimeWidths = []int64{18, 19, 20, 63, 64, 65, 1000, 1001, 4294967298}

// formatDateTimePadded returns the decimal component value zero-padded to
// width digits, which is what a minimum width asks for.
func formatDateTimePadded(comp byte, width int64) string {
	switch comp {
	case 'Y':
		return strings.Repeat("0", int(width)-4) + "2012"
	case 'M':
		return strings.Repeat("0", int(width)-1) + "5"
	case 'D':
		return strings.Repeat("0", int(width)-2) + "18"
	case 'H':
		return strings.Repeat("0", int(width)-2) + "13"
	case 'm':
		return strings.Repeat("0", int(width)-1) + "7"
	case 's':
		return strings.Repeat("0", int(width)-1) + "9"
	case 'f':
		// fractional seconds never exceed nanosecond precision
		return "500000000"
	}
	panic("unexpected component " + string(comp))
}

// formatDateTimeUnpadded returns the component value under a minimum width
// of 1, which a large maximum width must leave untouched.
func formatDateTimeUnpadded(comp byte) string {
	switch comp {
	case 'Y':
		return "2012"
	case 'M':
		return "5"
	case 'D':
		return "18"
	case 'H':
		return "13"
	case 'm':
		return "7"
	case 's':
		return "9"
	case 'f':
		return "5"
	}
	panic("unexpected component " + string(comp))
}

// TestFormatDateTimeWidthModifier verifies that a width modifier of any
// size formats without a panic on every platform: a minimum width up to the
// documented limit pads to exactly that many digits, a minimum width past it
// raises FOFD1340, and a maximum width of any size is honored exactly.
func TestFormatDateTimeWidthModifier(t *testing.T) {
	t.Parallel()

	for _, comp := range []byte("YMDHmsf") {
		for _, width := range formatDateTimeWidths {
			w := strconv.FormatInt(width, 10)
			minOnly := `format-dateTime(` + formatDateTimeWidthValue + `, '[` + string(comp) + `,` + w + `]')`
			minMax := `format-dateTime(` + formatDateTimeWidthValue + `, '[` + string(comp) + `,` + w + `-` + w + `]')`
			maxOnly := `format-dateTime(` + formatDateTimeWidthValue + `, '[` + string(comp) + `,1-` + w + `]')`

			if width > formatDateTimeMaxMinWidth {
				_, err := evaluate(t.Context(), nil, minOnly)
				requireErrorCode(t, err, "FOFD1340")
				_, err = evaluate(t.Context(), nil, minMax)
				requireErrorCode(t, err, "FOFD1340")
			} else {
				want := formatDateTimePadded(comp, width)
				result, err := evaluate(t.Context(), nil, minOnly)
				require.NoError(t, err, minOnly)
				require.Equal(t, want, result.StringValue(), minOnly)
				result, err = evaluate(t.Context(), nil, minMax)
				require.NoError(t, err, minMax)
				require.Equal(t, want, result.StringValue(), minMax)
			}

			result, err := evaluate(t.Context(), nil, maxOnly)
			require.NoError(t, err, maxOnly)
			require.Equal(t, formatDateTimeUnpadded(comp), result.StringValue(), maxOnly)
		}
	}
}

// TestFormatDateTimeWidthOutputBound verifies that padding stays linear in
// the requested width: the largest allowed minimum width yields exactly that
// many characters per component, for decimal and Roman presentations, and a
// picture repeating the component grows the output only proportionally.
func TestFormatDateTimeWidthOutputBound(t *testing.T) {
	t.Parallel()

	limit := strconv.Itoa(formatDateTimeMaxMinWidth)

	result, err := evaluate(t.Context(), nil, `format-date(xs:date('2012-05-18'), '[Y,`+limit+`]')`)
	require.NoError(t, err)
	require.Len(t, result.StringValue(), formatDateTimeMaxMinWidth)

	result, err = evaluate(t.Context(), nil, `format-date(xs:date('2012-05-18'), '[YI,`+limit+`]')`)
	require.NoError(t, err)
	require.Equal(t, "MMXII"+strings.Repeat(" ", formatDateTimeMaxMinWidth-5), result.StringValue())

	picture := strings.Repeat(`[D,`+limit+`]`, 50)
	result, err = evaluate(t.Context(), nil, `format-date(xs:date('2012-05-18'), '`+picture+`')`)
	require.NoError(t, err)
	require.Len(t, result.StringValue(), 50*formatDateTimeMaxMinWidth)

	_, err = evaluate(t.Context(), nil, `format-date(xs:date('2012-05-18'), '[YI,4294967298]')`)
	requireErrorCode(t, err, "FOFD1340")
}

// formatDateTimeWidthCase is one picture and the output it must produce.
type formatDateTimeWidthCase struct {
	picture string
	expect  string
}

// requireFormatDateTimeWidth evaluates fn(value, picture) for every case and
// requires the expected output.
func requireFormatDateTimeWidth(t *testing.T, fn, value string, cases []formatDateTimeWidthCase) {
	t.Helper()

	for _, tc := range cases {
		expr := fn + `(` + value + `, '` + tc.picture + `')`
		result, err := evaluate(t.Context(), nil, expr)
		require.NoError(t, err, expr)
		require.Equal(t, tc.expect, result.StringValue(), expr)
	}
}

// TestFormatDateYearMaximumWidth verifies the year modulus of F&O 3.1
// §9.8.4.4. A finite maximum width keeps that many low-order digits. Without
// one, a decimal digit pattern of two or more digit signs sets the modulus,
// and otherwise the year is output in full. An omitted maximum width means
// `*` (§9.8.4.2), so `[Y,2]` keeps every digit. A maximum width of 19 or more
// digits keeps every digit of any representable year.
func TestFormatDateYearMaximumWidth(t *testing.T) {
	t.Parallel()

	requireFormatDateTimeWidth(t, "format-date", `xs:date('2012-05-18')`, []formatDateTimeWidthCase{
		{`[Y,2]`, "2012"},
		{`[Y,2-2]`, "12"},
		{`[Y,2-*]`, "2012"},
		{`[Y01]`, "12"},
		{`[Y,4]`, "2012"},
		{`[Y,1-2]`, "12"},
		{`[Y,3-4]`, "2012"},
		{`[Y01,4]`, "0012"},
		{`[Y9999,25]`, "0000000000000000000002012"},
		{`[Y,1-18]`, "2012"},
		{`[Y,1-19]`, "2012"},
		{`[Y,1-64]`, "2012"},
		{`[Y,1-*]`, "2012"},
	})
	requireFormatDateTimeWidth(t, "format-date", `xs:date('0005-05-18')`, []formatDateTimeWidthCase{
		{`[Y,2]`, "05"},
		{`[Y,2-2]`, "05"},
		{`[Y01]`, "05"},
		{`[Y,4]`, "0005"},
		{`[Y,1-2]`, "5"},
	})
}

// TestFormatDateTimeMaximumWidth verifies how a maximum width treats the
// components other than the year. F&O 3.1 §9.8.4.3 ignores the maximum width
// of a decimal component, so it never drops digits. §9.8.4.5 extends the
// fractional seconds picture to the maximum width and truncates past it. A
// name longer than the maximum width is abbreviated, and a name shorter than
// the minimum width is padded with spaces. §9.8.4.6 never shortens a timezone.
func TestFormatDateTimeMaximumWidth(t *testing.T) {
	t.Parallel()

	requireFormatDateTimeWidth(t, "format-dateTime", `xs:dateTime('2012-11-25T13:47:38.123456+05:00')`,
		[]formatDateTimeWidthCase{
			{`[D,1-1]`, "25"},
			{`[D,2]`, "25"},
			{`[D,3]`, "025"},
			{`[M,1-1]`, "11"},
			{`[d,1-2]`, "330"},
			{`[H,1-1]`, "13"},
			{`[m,1-1]`, "47"},
			{`[s,1-1]`, "38"},
			{`[W,1-1]`, "47"},
			{`[f,1-1]`, "1"},
			{`[f,1-3]`, "123"},
			{`[f,*-2]`, "12"},
			{`[f,3]`, "123"},
			{`[f,1-*]`, "123456"},
			{`[f001,1-1]`, "123"},
			{`[MNn,3-3]`, "Nov"},
			{`[MNn,3]`, "November"},
			{`[MNn,3-10]`, "November"},
			{`[MNn,*-3]`, "Nov"},
			{`[MNn,1-3]`, "N"},
			{`[MN,3-3]`, "NOV"},
			{`[MNn,10]`, "November  "},
			{`[FNn,3-3]`, "Sun"},
			{`[FNn,3]`, "Sunday"},
			{`[FNn,8-8]`, "Sunday  "},
			{`[Z,2]`, "+05:00"},
			{`[Z,6-6]`, "+05:00"},
		})
	requireFormatDateTimeWidth(t, "format-date", `xs:date('2012-05-18')`, []formatDateTimeWidthCase{
		{`[MNn,5-5]`, "May  "},
		{`[MNn,1-3]`, "May"},
	})
}
