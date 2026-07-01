package value

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/lestrrat-go/helium/internal/lexicon"
)

// xsdWhitespace is the set of XSD whitespace characters (#x20 space, #x9 tab,
// #xD carriage return, #xA newline). Unlike Go's unicode.IsSpace this excludes
// NBSP (U+00A0) and other Unicode whitespace, so an invalid value padded with
// such characters stays invalid under subsequent lexical validation.
const xsdWhitespace = " \t\r\n"

// builtinTime is the xs:time builtin local name, hoisted to a constant so the
// several switch/comparison sites that reference it share one literal.
const builtinTime = "time"

// trimXSDSpace trims only XSD whitespace from both ends of s, leaving any other
// Unicode whitespace (e.g. NBSP) in place so it is rejected by lexical
// validation rather than silently stripped.
func trimXSDSpace(s string) string {
	return strings.Trim(s, xsdWhitespace)
}

// xsdFields splits s on runs of XSD whitespace only (space, tab, CR, LF),
// unlike strings.Fields which also splits on NBSP and other Unicode whitespace.
// A token containing NBSP therefore stays a single token (and remains invalid
// under per-item lexical validation) instead of being silently split.
func xsdFields(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\r' || r == '\n'
	})
}

// XSDFields splits s into items on runs of XSD whitespace only (space, tab, CR,
// LF), the exported form of xsdFields. xs:list item separation is defined over
// XSD whitespace, so callers must use this rather than strings.Fields: a list
// item containing NBSP (or other Unicode whitespace) stays a single token and is
// then rejected by per-item lexical validation, instead of being silently split.
func XSDFields(s string) []string {
	return xsdFields(s)
}

// numericComparableTypes is the set of integer-family and decimal builtins that
// Compare and CanonicalKey treat as value-comparable via math/big.Rat after
// strict lexical validation (including range checks for the bounded subtypes).
var numericComparableTypes = map[string]struct{}{
	"decimal": {}, "integer": {},
	"nonPositiveInteger": {}, "negativeInteger": {}, "long": {}, "int": {},
	"short": {}, "byte": {},
	"nonNegativeInteger": {}, "unsignedLong": {}, "unsignedInt": {},
	"unsignedShort": {}, "unsignedByte": {},
	"positiveInteger": {},
}

// Compare dispatches to type-specific comparison.
// Returns (cmp, ok) where cmp is -1/0/+1 and ok is false when comparison
// is undefined (NaN, incomparable durations, parse failures, or — for every
// recognized value-comparable type — either operand failing the strict lexical
// space defined by ValidateBuiltin). Validation runs before any lenient parsing
// so Compare never accepts a value that the validation path would reject.
func Compare(a, b, builtinLocal string) (int, bool) {
	switch builtinLocal {
	case "boolean":
		return compareBoolean(a, b)
	case lexicon.TypeFloat:
		return compareFloat(a, b, true)
	case lexicon.TypeDouble:
		return compareFloat(a, b, false)
	case "dateTime", lexicon.TypeDateTimeStamp:
		// xs:dateTimeStamp is a subtype of xs:dateTime; compare in the dateTime
		// value space.
		return compareDateTime(a, b, "dateTime")
	case "date":
		return compareDate(a, b, builtinLocal)
	case builtinTime:
		return compareTime(a, b, builtinLocal)
	case "gYear":
		return compareGYear(a, b, builtinLocal)
	case "gYearMonth":
		return compareGYearMonth(a, b, builtinLocal)
	case "gMonth":
		return compareGMonth(a, b, builtinLocal)
	case "gDay":
		return compareGDay(a, b, builtinLocal)
	case "gMonthDay":
		return compareGMonthDay(a, b, builtinLocal)
	case "duration", lexicon.TypeDayTimeDuration, lexicon.TypeYearMonthDuration:
		// xs:dayTimeDuration and xs:yearMonthDuration are subtypes of xs:duration;
		// compare in the duration value space.
		return compareDuration(a, b)
	case "hexBinary":
		return compareHexBinary(a, b)
	case "base64Binary":
		return compareBase64Binary(a, b)
	default:
		if _, isNumeric := numericComparableTypes[builtinLocal]; !isNumeric {
			// Any type NOT in numericComparableTypes (string-family builtins and
			// unrecognized types) has no numeric value-space comparison defined here,
			// so comparison is indeterminate. Routing these through CompareDecimal
			// would wrongly treat e.g. "5.0"/"5" for xs:string as equal.
			return 0, false
		}
		// Validate both operands against the strict lexical space (including the
		// range checks for bounded integer subtypes) before comparing, so e.g.
		// "1.0"/integer, "2147483648"/int and "1/2"/decimal are indeterminate.
		if !validBuiltinOperands(a, b, builtinLocal) {
			return 0, false
		}
		cmp := CompareDecimal(trimXSDSpace(a), trimXSDSpace(b))
		if cmp == -2 {
			return 0, false
		}
		return cmp, true
	}
}

// validBuiltinOperands reports whether both lexicals pass the strict lexical
// space (ValidateBuiltin) for builtinLocal, after trimming XSD whitespace only.
//
// Comparison and canonicalization run on values the caller has already validated
// under the schema's actual XSD version, so the lexical guards here use Version11
// (the most permissive space). A value that is invalid under XSD 1.0 (e.g.
// "+INF") never reaches this path in 1.0 mode, so the permissive check cannot
// admit anything the version-specific validation already rejected. All the other
// ValidateBuiltin guards in this file pass Version11 for the same reason.
func validBuiltinOperands(a, b, builtinLocal string) bool {
	if ValidateBuiltin(trimXSDSpace(a), builtinLocal, Version11) != nil {
		return false
	}
	return ValidateBuiltin(trimXSDSpace(b), builtinLocal, Version11) == nil
}

// CanonicalKey maps a lexical value to a value-space canonical string for the
// given builtin type, so lexically-distinct but value-equal inputs (e.g. "5"
// and "+5" for xs:integer) produce the same key. It returns (key, true) when a
// canonical form is defined; otherwise (collapsed-whitespace value, false) for
// string-family types and anything unrecognized or unparsable.
func CanonicalKey(s, builtinLocal string) (string, bool) {
	switch builtinLocal {
	case "boolean":
		trimmed := trimXSDSpace(s)
		if ValidateBuiltin(trimmed, "boolean", Version11) != nil {
			return trimmed, false
		}
		if trimmed == "true" || trimmed == "1" {
			return "1", true
		}
		return "0", true
	case lexicon.TypeFloat:
		return canonicalFloatKey(s, 32) // xs:float is 32-bit IEEE-754
	case lexicon.TypeDouble:
		return canonicalFloatKey(s, 64)
	case "dateTime", "date", "time", "gYear", "gYearMonth", "gMonth", "gDay", "gMonthDay":
		return canonicalDateTimeKey(trimXSDSpace(s), builtinLocal)
	case lexicon.TypeDateTimeStamp:
		// xs:dateTimeStamp canonicalizes in the xs:dateTime value space.
		return canonicalDateTimeKey(trimXSDSpace(s), "dateTime")
	case "hexBinary":
		trimmed := trimXSDSpace(s)
		if ValidateBuiltin(trimmed, "hexBinary", Version11) != nil {
			return trimmed, false
		}
		decoded, err := hex.DecodeString(trimmed)
		if err != nil {
			return trimmed, false
		}
		// Stable byte key so case-distinct forms ("0A"/"0a") canonicalize equal.
		return hex.EncodeToString(decoded), true
	case "base64Binary":
		trimmed := trimXSDSpace(s)
		if ValidateBuiltin(trimmed, "base64Binary", Version11) != nil {
			return trimmed, false
		}
		decoded, ok := decodeBase64Binary(trimmed)
		if !ok {
			return trimmed, false
		}
		// Stable byte key so whitespace-distinct forms ("YWJj"/"YW Jj")
		// canonicalize equal; hex of the decoded octets is a canonical encoding.
		return hex.EncodeToString(decoded), true
	case "duration", lexicon.TypeDayTimeDuration, lexicon.TypeYearMonthDuration:
		// xs:dayTimeDuration / xs:yearMonthDuration canonicalize in the xs:duration
		// value space (they are valid xs:duration lexicals).
		trimmed := trimXSDSpace(s)
		// Validate against the strict xs:duration lexical space before the lenient
		// parseXSDDurationValue below, so CanonicalKey never canonicalizes a value
		// the validator rejects.
		if ValidateBuiltin(trimmed, "duration", Version11) != nil {
			return trimmed, false
		}
		d, ok := parseXSDDurationValue(trimmed)
		if !ok {
			return trimmed, false
		}
		// The xs:duration value space is the (months, seconds) pair; two lexicals
		// that compare equal via compareDuration must canonicalize equal. Apply the
		// sign to both components and emit a stable signed key consistent with
		// compareDuration, so e.g. "P1D" and "PT24H" (both 0 months, 86400 seconds)
		// collide while values with differing month/second components stay distinct.
		months, seconds := d.monVal(), d.secVal()
		if d.negative {
			months, seconds = new(big.Int).Neg(months), new(big.Rat).Neg(seconds)
		}
		return fmt.Sprintf("%s|%s", months.String(), seconds.RatString()), true
	case "decimal", "integer",
		"nonPositiveInteger", "negativeInteger", "long", "int", "short", "byte",
		"nonNegativeInteger", "unsignedLong", "unsignedInt", "unsignedShort", "unsignedByte",
		"positiveInteger":
		trimmed := trimXSDSpace(s)
		// Validate strictly (lexical space + range for bounded subtypes) before
		// canonicalizing, so e.g. "1.0"/integer, "2147483648"/int and "1/2"/
		// decimal yield ok=false rather than a spurious canonical key.
		if ValidateBuiltin(trimmed, builtinLocal, Version11) != nil {
			return trimmed, false
		}
		r, ok := new(big.Rat).SetString(trimmed)
		if !ok {
			return trimmed, false
		}
		return r.RatString(), true
	case "string":
		// whiteSpace=preserve: the value space is the exact lexical string, so
		// leading/trailing/internal whitespace is significant. Do not alter it.
		return s, false
	case "normalizedString":
		// whiteSpace=replace: tab/newline/carriage-return become spaces.
		return whitespaceReplace(s), false
	case "NMTOKENS", "IDREFS", "ENTITIES":
		// List types: collapse internal whitespace so token sequences that
		// differ only in separator whitespace are value-equal. Split on XSD
		// whitespace only so a token containing NBSP is preserved (and stays
		// invalid), rather than being silently split into two tokens.
		return strings.Join(xsdFields(s), " "), false
	default:
		// Remaining string-derived types (token, NMTOKEN, Name, NCName, ID,
		// IDREF, ENTITY, language, anyURI, …) have whiteSpace=collapse.
		return strings.Join(xsdFields(s), " "), false
	}
}

// canonicalFloatKey canonicalizes an xs:float/xs:double value at the given IEEE
// precision (bitSize 32 for xs:float, 64 for xs:double). Using the correct
// precision ensures values that are equal in xs:float's 32-bit value space (but
// distinct as 64-bit doubles) map to the same key, and vice versa.
func canonicalFloatKey(s string, bitSize int) (string, bool) {
	trimmed := trimXSDSpace(s)
	// Validate against the strict xs:float/xs:double lexical space first: the
	// lenient parseXSDFloat (and Go's strconv.ParseFloat) accept spellings such
	// as "Inf" that are not valid XSD lexical forms.
	if ValidateBuiltin(trimmed, lexicon.TypeDouble, Version11) != nil {
		return trimmed, false
	}
	f, ok := parseXSDFloat(trimmed)
	if !ok {
		return trimmed, false
	}
	if math.IsNaN(f) {
		return "NaN", true
	}
	if math.IsInf(f, 1) {
		return "INF", true
	}
	if math.IsInf(f, -1) {
		return "-INF", true
	}
	if bitSize == 32 {
		f = float64(float32(f)) // round to xs:float precision
	}
	// A finite lexical value can overflow to infinity at the target precision
	// (e.g. "1e40" rounds to +Inf in xs:float). Re-check after rounding so the
	// key matches the XSD-canonical "INF"/"-INF" form rather than Go's
	// strconv.FormatFloat output ("+Inf"/"-Inf"), keeping equal values' keys
	// consistent with the literal INF/-INF spellings above.
	if math.IsInf(f, 1) {
		return "INF", true
	}
	if math.IsInf(f, -1) {
		return "-INF", true
	}
	if f == 0 {
		f = 0 // normalize -0 to +0; they are equal in the value space
	}
	return strconv.FormatFloat(f, 'g', -1, bitSize), true
}

// whitespaceReplace applies the XSD whiteSpace="replace" normalization: each
// of the four XSD whitespace characters tab (#x9), newline (#xA), and carriage
// return (#xD) becomes a single space (#x20). Per the XSD datatype spec only
// those ASCII whitespace characters are affected; Unicode whitespace such as
// NBSP (U+00A0) is left untouched.
func whitespaceReplace(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\t' || r == '\n' || r == '\r' {
			return ' '
		}
		return r
	}, s)
}

// whitespaceCollapse applies the XSD whiteSpace="collapse" normalization:
// replace tab/newline/CR with space (ASCII-only, like whitespaceReplace), then
// collapse runs of spaces and trim leading/trailing spaces. Only the four XSD
// whitespace characters (#x20, #x9, #xD, #xA) are treated as whitespace; Unicode
// whitespace such as NBSP (U+00A0) is preserved, so an invalid value containing
// it remains invalid under subsequent lexical validation.
func whitespaceCollapse(s string) string {
	replaced := whitespaceReplace(s)
	var b strings.Builder
	b.Grow(len(replaced))
	inSpace := true // treat start as space to trim leading
	for i := range len(replaced) {
		if replaced[i] == ' ' {
			inSpace = true
			continue
		}
		if inSpace && b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteByte(replaced[i])
		inSpace = false
	}
	return b.String()
}

// WhiteSpace returns the effective XSD whiteSpace facet ("preserve", "replace",
// or "collapse") for a builtin datatype's local name. xs:string preserves,
// xs:normalizedString replaces, and every other builtin (token, the integer and
// date/time families, boolean, the NCName/Name/NMTOKEN family, list types,
// anyURI, …) collapses, per the XSD 1.1 datatype spec. Unknown names default to
// "collapse" so callers normalize conservatively.
func WhiteSpace(builtinLocal string) string {
	switch builtinLocal {
	case "string":
		return "preserve"
	case "normalizedString":
		return "replace"
	default:
		return "collapse"
	}
}

// Normalize applies the XSD whiteSpace facet of the named builtin datatype to a
// lexical value, returning the whitespace-processed form that must be used
// before lexical validation (ValidateBuiltin) or value comparison. "preserve"
// leaves the value untouched, "replace" turns each tab/newline/CR into a space,
// and "collapse" additionally collapses runs of spaces and trims the ends.
func Normalize(s, builtinLocal string) string {
	switch WhiteSpace(builtinLocal) {
	case "preserve":
		return s
	case "replace":
		return whitespaceReplace(s)
	default: // collapse
		return whitespaceCollapse(s)
	}
}

// validDateTimeOperands reports whether both date/time/g* lexicals pass the
// strict lexical space (ValidateBuiltin) for builtinLocal. The parseXSD*
// parsers used by the compareX functions are deliberately lenient (they skip
// leap-year, month/day range and timezone-range checks and accept trailing
// junk), so the compareX functions call this first to guarantee Compare never
// accepts a value the validation path rejects.
func validDateTimeOperands(a, b, builtinLocal string) bool {
	if ValidateBuiltin(a, builtinLocal, Version11) != nil {
		return false
	}
	return ValidateBuiltin(b, builtinLocal, Version11) == nil
}

func canonicalDateTimeKey(s, builtinLocal string) (string, bool) {
	// Reject values the strict lexical validator rejects before the lenient
	// parseXSD* parsing below, so CanonicalKey never canonicalizes a date/time
	// value that ValidateBuiltin would reject (bad timezone, out-of-range
	// month/day, leap-day, trailing junk).
	if ValidateBuiltin(s, builtinLocal, Version11) != nil {
		return s, false
	}
	var dt xsdDateTime
	var ok bool
	switch builtinLocal {
	case "dateTime":
		dt, ok = parseXSDDateTime(s)
	case "date":
		dt, ok = parseXSDDate(s)
	case builtinTime:
		dt, ok = parseXSDTime(s)
	case "gYear":
		dt, ok = parseXSDGYear(s)
	case "gYearMonth":
		dt, ok = parseXSDGYearMonth(s)
	case "gMonth":
		dt, ok = parseXSDGMonth(s)
	case "gDay":
		dt, ok = parseXSDGDay(s)
	case "gMonthDay":
		dt, ok = parseXSDGMonthDay(s)
	}
	if !ok {
		return s, false
	}
	// Normalize 24:00:00 to 00:00:00 of the next day so its key matches the
	// equivalent start-of-day instant, then to UTC when timezoned.
	dt = dt.normalizeHour24()
	if builtinLocal == builtinTime {
		// xs:time carries no calendar date, so assign the SAME synthetic reference
		// date compareTime uses (2000-01-15) before UTC normalization. Otherwise a
		// timezoned value whose UTC offset crosses midnight (e.g. "11:30:00+01:00"
		// vs "10:30:00Z", both 10:30 UTC) would key off a zero date and differ in
		// the day/month/year fields, missing the equality compareTime reports.
		dt.year, dt.month, dt.day = big.NewInt(2000), 1, 15
	}
	if dt.hasTZ {
		dt = dt.normalizeToUTC()
	}
	return fmt.Sprintf("%s|%d|%d|%d|%d|%s|%t", dt.yearVal().String(), dt.month, dt.day, dt.hour, dt.min, dt.secVal().RatString(), dt.hasTZ), true
}

// CompareDecimal compares two decimal string values using math/big.Rat.
// Returns -1 if a < b, 0 if a == b, 1 if a > b, or -2 on parse error.
func CompareDecimal(a, b string) int {
	ra, ok1 := new(big.Rat).SetString(a)
	rb, ok2 := new(big.Rat).SetString(b)
	if !ok1 || !ok2 {
		return -2
	}
	return ra.Cmp(rb)
}

// parseXSDBoolean canonicalizes an xs:boolean lexical form. "true"/"1" map to
// true and "false"/"0" map to false. Any other input is not a valid boolean.
func parseXSDBoolean(s string) (bool, bool) {
	switch s {
	case "true", "1":
		return true, true
	case "false", "0":
		return false, true
	}
	return false, false
}

// compareBoolean compares two xs:boolean values in value space. xs:boolean has
// no order relation in XSD, so callers should rely only on equality (cmp == 0).
// For a total, deterministic result this orders false < true; equal values
// return 0.
func compareBoolean(a, b string) (int, bool) {
	if !validBuiltinOperands(a, b, "boolean") {
		return 0, false
	}
	ba, ok1 := parseXSDBoolean(trimXSDSpace(a))
	bb, ok2 := parseXSDBoolean(trimXSDSpace(b))
	if !ok1 || !ok2 {
		return 0, false
	}
	if ba == bb {
		return 0, true
	}
	if !ba {
		return -1, true
	}
	return 1, true
}

// compareHexBinary compares two xs:hexBinary values in value space (the decoded
// octet sequence), so lexically distinct forms that decode to the same bytes are
// equal (e.g. "0A" == "0a"). XSD does not order hexBinary, but a deterministic,
// antisymmetric total order (bytes.Compare of the decoded octets) is returned so
// the result is a well-behaved comparator; enumeration only relies on cmp == 0.
// Returns ok=false if either operand is not valid hexBinary.
func compareHexBinary(a, b string) (int, bool) {
	if !validBuiltinOperands(a, b, "hexBinary") {
		return 0, false
	}
	da, err1 := hex.DecodeString(trimXSDSpace(a))
	db, err2 := hex.DecodeString(trimXSDSpace(b))
	if err1 != nil || err2 != nil {
		return 0, false
	}
	return bytes.Compare(da, db), true
}

// compareBase64Binary compares two xs:base64Binary values in value space (the
// decoded octet sequence), ignoring the whitespace permitted in the lexical
// form. As with hexBinary, a deterministic bytes.Compare total order is returned
// rather than a bare equality flag. Returns ok=false if either operand is not
// valid base64Binary.
func compareBase64Binary(a, b string) (int, bool) {
	if !validBuiltinOperands(a, b, "base64Binary") {
		return 0, false
	}
	da, ok1 := decodeBase64Binary(trimXSDSpace(a))
	db, ok2 := decodeBase64Binary(trimXSDSpace(b))
	if !ok1 || !ok2 {
		return 0, false
	}
	return bytes.Compare(da, db), true
}

func decodeBase64Binary(s string) ([]byte, bool) {
	stripped := strings.Map(func(r rune) rune {
		if r == ' ' || r == '\n' || r == '\r' || r == '\t' {
			return -1
		}
		return r
	}, s)
	// Only correctly-padded base64 is a valid xs:base64Binary lexical form, so an
	// unpadded operand (e.g. "TQ") must fail to decode and yield ok=false rather
	// than comparing equal to its padded counterpart. Strict() additionally
	// rejects padded forms whose unused trailing bits are non-zero (e.g. "TR==").
	decoded, err := base64.StdEncoding.Strict().DecodeString(stripped)
	if err != nil {
		return nil, false
	}
	return decoded, true
}

func parseXSDFloat(s string) (float64, bool) {
	switch s {
	case "INF", "+INF":
		return math.Inf(1), true
	case "-INF":
		return math.Inf(-1), true
	// Per XSD the only valid lexical form for NaN is the bare "NaN"; +NaN and
	// -NaN are not valid (floatRegex rejects them). Reject them here too so the
	// value space stays consistent with the lexical space — otherwise an invalid
	// facet such as enumeration value="+NaN" would value-match instance "NaN".
	case "NaN":
		return math.NaN(), true
	}
	f, err := strconv.ParseFloat(s, 64)
	// XSD 1.1 maps a float/double lexical whose magnitude overflows the value
	// space to ±INF, and ValidateBuiltin accepts such lexicals (e.g. "1e400").
	// strconv.ParseFloat signals this with ErrRange and returns ±Inf as the
	// result, so honor that infinity rather than rejecting the value.
	if errors.Is(err, strconv.ErrRange) && math.IsInf(f, 0) {
		return f, true
	}
	if err != nil {
		return 0, false
	}
	// strconv.ParseFloat is lenient: it accepts non-XSD spellings such as "nan",
	// "NAN", "inf", and "Infinity". The only valid XSD lexicals denoting NaN/±INF
	// are the exact forms handled in the switch above, so any NaN/Inf reaching
	// here came from one of those lenient spellings and must be rejected.
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, false
	}
	return f, true
}

// IsFloatNaN reports whether s is a valid xs:float/xs:double lexical form that
// denotes NaN. Only the bare "NaN" qualifies; the sign-prefixed +NaN/-NaN are
// not valid XSD lexical forms and are rejected.
func IsFloatNaN(s string) bool {
	f, ok := parseXSDFloat(s)
	return ok && math.IsNaN(f)
}

// compareFloat compares two xs:float/xs:double lexical values in their numeric
// value space. When single is true the operands are xs:float, whose value space
// is IEEE-754 single precision: both are rounded to float32 first so values that
// round to the same single-precision number compare equal (e.g. "16777216" ==
// "16777217"). When single is false they are xs:double and compared as float64.
// Infinities round-trip identically through float32, so only the finite path
// changes precision.
func compareFloat(a, b string, single bool) (int, bool) {
	// xs:float and xs:double share one lexical validator; validate both operands
	// strictly (rejecting e.g. "Inf", the mixed-case spelling) before parsing.
	floatType := lexicon.TypeDouble
	if single {
		floatType = lexicon.TypeFloat
	}
	if !validBuiltinOperands(a, b, floatType) {
		return 0, false
	}
	fa, ok1 := parseXSDFloat(trimXSDSpace(a))
	fb, ok2 := parseXSDFloat(trimXSDSpace(b))
	if !ok1 || !ok2 {
		return 0, false
	}
	if math.IsNaN(fa) || math.IsNaN(fb) {
		return 0, false
	}
	if single {
		fa = float64(float32(fa))
		fb = float64(float32(fb))
	}
	if fa < fb {
		return -1, true
	}
	if fa > fb {
		return 1, true
	}
	return 0, true
}

// CompareFloatFacetBound compares two xs:float/xs:double facet BOUND lexicals for
// the purpose of the same-type range-facet consistency check (e.g. minInclusive
// vs maxInclusive). It differs from the ordinary value-space Compare in its
// treatment of NaN: where Compare returns incomparable (ok=false) for any NaN
// operand, this orders NaN as EQUAL to NaN and GREATER THAN every non-NaN value.
// That matches xmllint, which rejects a schema whose minInclusive="NaN" exceeds
// a finite maxInclusive while accepting minInclusive=finite with maxInclusive=NaN.
//
// builtinLocal must be xs:float or xs:double; for any other type it returns
// ok=false so the caller falls back to its normal comparator. Both operands are
// validated as valid float/double lexicals first: an invalid bound yields
// ok=false (no ordering decision), so the caller's invalid-bound check — not this
// consistency check — reports the error.
func CompareFloatFacetBound(a, b, builtinLocal string) (int, bool) {
	if builtinLocal != lexicon.TypeFloat && builtinLocal != lexicon.TypeDouble {
		return 0, false
	}
	if !validBuiltinOperands(a, b, builtinLocal) {
		return 0, false
	}
	fa, ok1 := parseXSDFloat(trimXSDSpace(a))
	fb, ok2 := parseXSDFloat(trimXSDSpace(b))
	if !ok1 || !ok2 {
		return 0, false
	}
	aNaN := math.IsNaN(fa)
	bNaN := math.IsNaN(fb)
	if aNaN || bNaN {
		switch {
		case aNaN && bNaN:
			return 0, true
		case aNaN:
			return 1, true
		default:
			return -1, true
		}
	}
	return compareFloat(a, b, builtinLocal == lexicon.TypeFloat)
}

// xsdDateTime is a deliberately separate date/time model from xpath3's
// time.Time-based date path (xpath3/cast_datetime.go). time.Time cannot hold the
// arbitrary expanded years or exact fractional seconds that XSD 1.1 date/time
// value-space comparison requires (see the year and sec fields below), so the
// two parsers are not consolidated despite accepting the same lexical forms.
type xsdDateTime struct {
	// year is held with arbitrary precision so that valid expanded years
	// (e.g. 999999999999999999999999) compare correctly rather than
	// overflowing a fixed-width int. A nil year is treated as 0 (used by the
	// year-agnostic gMonth/gDay/gMonthDay types).
	year       *big.Int
	month, day int
	hour, min  int
	// sec holds the seconds component (including any fractional digits) as an
	// EXACT rational rather than a float64, so that two distinct valid lexicals
	// that differ only in trailing fractional precision (e.g. "00.1" vs
	// "00.1000000000000000000000000000000000001") stay distinct in both Compare
	// and CanonicalKey instead of colliding through float rounding. A nil sec is
	// treated as 0 (the year-agnostic g-types never set it).
	sec   *big.Rat
	hasTZ bool
	tzMin int
}

// secVal returns the seconds component as a *big.Rat, substituting 0 for a nil
// sec so the date-only and year-agnostic g-types compare consistently.
func (dt xsdDateTime) secVal() *big.Rat {
	if dt.sec == nil {
		return new(big.Rat)
	}
	return dt.sec
}

// yearVal returns the year as a *big.Int, substituting 0 for a nil year so the
// year-agnostic g-types compare consistently.
func (dt xsdDateTime) yearVal() *big.Int {
	if dt.year == nil {
		return big.NewInt(0)
	}
	return dt.year
}

// parseYearBig parses a year digit-string with arbitrary precision so valid
// expanded years larger than an int compare correctly. neg negates the result.
func parseYearBig(s string, neg bool) (*big.Int, bool) {
	// big.Int.SetString accepts a leading sign, but the sign of an XSD year is
	// carried exclusively by neg. Reject anything that is not ASCII digits only
	// (including a leading '+' or '-') so malformed lexicals such as
	// "+2023-01-01" do not compare or canonicalize as valid.
	if s == "" {
		return nil, false
	}
	for i := range len(s) {
		if s[i] < '0' || s[i] > '9' {
			return nil, false
		}
	}
	n, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return nil, false
	}
	if neg {
		n.Neg(n)
	}
	return n, true
}

func parseTZ(s string) (bool, int) {
	if s == "" {
		return false, 0
	}
	if s[0] == 'Z' {
		return true, 0
	}
	if (s[0] == '+' || s[0] == '-') && len(s) >= 6 && s[3] == ':' {
		hh, err1 := strconv.Atoi(s[1:3])
		mm, err2 := strconv.Atoi(s[4:6])
		if err1 != nil || err2 != nil {
			return false, 0
		}
		offset := hh*60 + mm
		if s[0] == '-' {
			offset = -offset
		}
		return true, offset
	}
	return false, 0
}

func parseXSDDateTime(s string) (xsdDateTime, bool) {
	var dt xsdDateTime
	neg := false
	if len(s) > 0 && s[0] == '-' {
		neg = true
		s = s[1:]
	}
	// Find 'T' separator.
	datePart, timePart, found := strings.Cut(s, "T")
	if !found {
		return dt, false
	}

	// Parse date: YYYY-MM-DD
	dParts := strings.SplitN(datePart, "-", 3)
	if len(dParts) != 3 {
		return dt, false
	}
	year, ok := parseYearBig(dParts[0], neg)
	if !ok {
		return dt, false
	}
	month, err := strconv.Atoi(dParts[1])
	if err != nil {
		return dt, false
	}
	day, err := strconv.Atoi(dParts[2])
	if err != nil {
		return dt, false
	}
	dt.year = year
	dt.month = month
	dt.day = day

	// Parse time: HH:MM:SS[.frac][TZ]
	if !parseTimeInto(&dt, timePart) {
		return dt, false
	}
	return dt, true
}

func parseTimeFields(s string) (int, int, *big.Rat, string, bool) {
	if len(s) < 8 || s[2] != ':' || s[5] != ':' {
		return 0, 0, nil, "", false
	}
	hh, err1 := strconv.Atoi(s[0:2])
	mm, err2 := strconv.Atoi(s[3:5])
	if err1 != nil || err2 != nil {
		return 0, 0, nil, "", false
	}
	// Seconds may have fractional part.
	rest := s[6:]
	secEnd := 0
	for secEnd < len(rest) {
		c := rest[secEnd]
		if (c >= '0' && c <= '9') || c == '.' {
			secEnd++
		} else {
			break
		}
	}
	// big.Rat.SetString accepts a leading sign and various forms the seconds
	// field must not use, so the digit-run scan above (which only admits ASCII
	// digits and a single '.') has already restricted the repertoire. Parse the
	// remaining decimal as an exact rational to preserve full fractional
	// precision.
	sec, ok := new(big.Rat).SetString(rest[:secEnd])
	if !ok {
		return 0, 0, nil, "", false
	}
	return hh, mm, sec, rest[secEnd:], true
}

func parseTimeInto(dt *xsdDateTime, s string) bool {
	hh, mm, sec, rest, ok := parseTimeFields(s)
	if !ok {
		return false
	}
	dt.hour = hh
	dt.min = mm
	dt.sec = sec
	hasTZ, tzOff := parseTZ(rest)
	dt.hasTZ = hasTZ
	dt.tzMin = tzOff
	return true
}

func parseXSDDate(s string) (xsdDateTime, bool) {
	var dt xsdDateTime
	neg := false
	if len(s) > 0 && s[0] == '-' {
		neg = true
		s = s[1:]
	}
	// YYYY-MM-DD[TZ]; the year is at least 4 digits and may be an expanded
	// (arbitrarily long) year, so locate the first dash rather than assuming it
	// sits at offset 4.
	if len(s) < 10 {
		return dt, false
	}
	dashIdx := strings.IndexByte(s, '-')
	if dashIdx < 4 {
		return dt, false
	}
	year, ok := parseYearBig(s[:dashIdx], neg)
	if !ok {
		return dt, false
	}
	rest := s[dashIdx+1:]
	if len(rest) < 5 || rest[2] != '-' {
		return dt, false
	}
	month, err := strconv.Atoi(rest[0:2])
	if err != nil {
		return dt, false
	}
	day, err := strconv.Atoi(rest[3:5])
	if err != nil {
		return dt, false
	}
	dt.year = year
	dt.month = month
	dt.day = day
	hasTZ, tzOff := parseTZ(rest[5:])
	dt.hasTZ = hasTZ
	dt.tzMin = tzOff
	return dt, true
}

func parseXSDTime(s string) (xsdDateTime, bool) {
	var dt xsdDateTime
	if !parseTimeInto(&dt, s) {
		return dt, false
	}
	return dt, true
}

func parseXSDGYear(s string) (xsdDateTime, bool) {
	var dt xsdDateTime
	neg := false
	if len(s) > 0 && s[0] == '-' {
		neg = true
		s = s[1:]
	}
	// Find end of digit run.
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i < 4 {
		return dt, false
	}
	year, ok := parseYearBig(s[:i], neg)
	if !ok {
		return dt, false
	}
	dt.year = year
	hasTZ, tzOff := parseTZ(s[i:])
	dt.hasTZ = hasTZ
	dt.tzMin = tzOff
	return dt, true
}

func parseXSDGYearMonth(s string) (xsdDateTime, bool) {
	var dt xsdDateTime
	neg := false
	if len(s) > 0 && s[0] == '-' {
		neg = true
		s = s[1:]
	}
	// Find first dash after year digits.
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i < 4 || i >= len(s) || s[i] != '-' {
		return dt, false
	}
	year, ok := parseYearBig(s[:i], neg)
	if !ok {
		return dt, false
	}
	rest := s[i+1:]
	if len(rest) < 2 {
		return dt, false
	}
	month, err := strconv.Atoi(rest[:2])
	if err != nil {
		return dt, false
	}
	dt.year = year
	dt.month = month
	hasTZ, tzOff := parseTZ(rest[2:])
	dt.hasTZ = hasTZ
	dt.tzMin = tzOff
	return dt, true
}

func parseXSDGMonth(s string) (xsdDateTime, bool) {
	var dt xsdDateTime
	if len(s) < 4 || s[0] != '-' || s[1] != '-' {
		return dt, false
	}
	month, err := strconv.Atoi(s[2:4])
	if err != nil {
		return dt, false
	}
	dt.month = month
	hasTZ, tzOff := parseTZ(s[4:])
	dt.hasTZ = hasTZ
	dt.tzMin = tzOff
	return dt, true
}

func parseXSDGDay(s string) (xsdDateTime, bool) {
	var dt xsdDateTime
	if len(s) < 5 || s[0] != '-' || s[1] != '-' || s[2] != '-' {
		return dt, false
	}
	day, err := strconv.Atoi(s[3:5])
	if err != nil {
		return dt, false
	}
	dt.day = day
	hasTZ, tzOff := parseTZ(s[5:])
	dt.hasTZ = hasTZ
	dt.tzMin = tzOff
	return dt, true
}

func parseXSDGMonthDay(s string) (xsdDateTime, bool) {
	var dt xsdDateTime
	if len(s) < 7 || s[0] != '-' || s[1] != '-' || s[4] != '-' {
		return dt, false
	}
	month, err := strconv.Atoi(s[2:4])
	if err != nil {
		return dt, false
	}
	day, err := strconv.Atoi(s[5:7])
	if err != nil {
		return dt, false
	}
	dt.month = month
	dt.day = day
	hasTZ, tzOff := parseTZ(s[7:])
	dt.hasTZ = hasTZ
	dt.tzMin = tzOff
	return dt, true
}

// daysInMonth returns the number of days in the given month/year. The year is
// taken with arbitrary precision so leap-year status is correct for valid
// expanded years that overflow a fixed-width int.
func daysInMonth(year *big.Int, month int) int {
	switch month {
	case 1, 3, 5, 7, 8, 10, 12:
		return 31
	case 4, 6, 9, 11:
		return 30
	case 2:
		if isLeapYearBig(year) {
			return 29
		}
		return 28
	}
	return 30
}

// isLeapYearBig reports whether the (possibly nil) proleptic Gregorian year is a
// leap year: divisible by 4 and (not divisible by 100, or divisible by 400).
func isLeapYearBig(year *big.Int) bool {
	if year == nil {
		year = big.NewInt(0)
	}
	mod := func(m int64) int64 {
		return new(big.Int).Mod(year, big.NewInt(m)).Int64()
	}
	if mod(4) != 0 {
		return false
	}
	if mod(100) != 0 {
		return true
	}
	return mod(400) == 0
}

func (dt xsdDateTime) normalizeToUTC() xsdDateTime {
	if !dt.hasTZ || dt.tzMin == 0 {
		return dt
	}
	r := dt
	// Copy the year so arithmetic below does not mutate the operand's *big.Int.
	r.year = new(big.Int).Set(dt.yearVal())
	r.min -= r.tzMin
	r.tzMin = 0

	// Propagate minutes overflow.
	for r.min < 0 {
		r.min += 60
		r.hour--
	}
	for r.min >= 60 {
		r.min -= 60
		r.hour++
	}

	// Propagate hours overflow.
	for r.hour < 0 {
		r.hour += 24
		r.day--
	}
	for r.hour >= 24 {
		r.hour -= 24
		r.day++
	}

	// Propagate day overflow.
	for r.day < 1 {
		r.month--
		if r.month < 1 {
			r.month = 12
			r.year.Sub(r.year, big.NewInt(1))
		}
		r.day += daysInMonth(r.year, r.month)
	}
	for r.month > 0 && r.day > daysInMonth(r.year, r.month) {
		r.day -= daysInMonth(r.year, r.month)
		r.month++
		if r.month > 12 {
			r.month = 1
			r.year.Add(r.year, big.NewInt(1))
		}
	}

	return r
}

// normalizeHour24 maps the XSD-valid end-of-day form 24:00:00 to 00:00:00 of
// the following day, so it lives in the same value space as the equivalent
// start-of-day instant. The lexical validator only accepts hour 24 with zero
// minutes/seconds, so no minute/second carry is needed; only the day (and any
// month/year it rolls into) advances. Types without a calendar date (xs:time,
// the year-agnostic g-types) simply reset the hour, matching how XSD treats
// 24:00:00 and 00:00:00 of the next day as the same xs:time value.
func (dt xsdDateTime) normalizeHour24() xsdDateTime {
	if dt.hour != 24 {
		return dt
	}
	r := dt
	r.hour = 0
	if dt.year != nil {
		r.year = new(big.Int).Set(dt.year)
	}
	// Only advance the calendar date when a full date is present; partial
	// gregorian types (year/month/day zero) carry no date to roll over.
	if r.day < 1 || r.month < 1 {
		return r
	}
	r.day++
	if r.day <= daysInMonth(r.yearVal(), r.month) {
		return r
	}
	r.day = 1
	r.month++
	if r.month <= 12 {
		return r
	}
	r.month = 1
	if r.year == nil {
		r.year = big.NewInt(0)
	}
	r.year.Add(r.year, big.NewInt(1))
	return r
}

func compareDateTimeFields(a, b xsdDateTime) int {
	if c := a.yearVal().Cmp(b.yearVal()); c != 0 {
		return c
	}
	if a.month != b.month {
		if a.month < b.month {
			return -1
		}
		return 1
	}
	if a.day != b.day {
		if a.day < b.day {
			return -1
		}
		return 1
	}
	if a.hour != b.hour {
		if a.hour < b.hour {
			return -1
		}
		return 1
	}
	if a.min != b.min {
		if a.min < b.min {
			return -1
		}
		return 1
	}
	return a.secVal().Cmp(b.secVal())
}

func compareDateTimeParsed(a, b xsdDateTime) (int, bool) {
	// Map the end-of-day 24:00:00 form onto 00:00:00 of the next day so it
	// compares in the correct value space before any timezone normalization.
	a = a.normalizeHour24()
	b = b.normalizeHour24()
	if a.hasTZ != b.hasTZ {
		return compareDateTimeMixedTZ(a, b)
	}
	if a.hasTZ {
		a = a.normalizeToUTC()
		b = b.normalizeToUTC()
	}
	return compareDateTimeFields(a, b), true
}

// compareDateTimeMixedTZ compares two date/time values when exactly one carries
// a timezone, applying the XSD 1.0 order relation (3.2.7.4). A non-timezoned
// value denotes the instant interval [v-14:00, v+14:00]; if that whole interval
// lies on one side of the timezoned operand the result is determinate. Only an
// overlapping interval is indeterminate.
func compareDateTimeMixedTZ(a, b xsdDateTime) (int, bool) {
	// The determinate rule normalizes a synthetic ±14:00 offset across day
	// boundaries, which requires a full calendar date (year, month, day). The
	// partial gregorian types leave some of those components zero — gYear
	// (month=0, day=0), gYearMonth (day=0), gMonth (year=0, day=0), gDay (year=0,
	// month=0), and gMonthDay (year=0). Applying the offset to a zero field makes
	// normalizeToUTC borrow into a neighbouring period and yield a determinately
	// wrong result (e.g. gYear "2020" rolling back to 2019), so those types stay
	// indeterminate, as they were before this rule existed. (xs:time is not in
	// this set: compareTime assigns a reference date before comparing, so it has
	// a full calendar date and flows through the determinate path correctly. A
	// present year equal to 0000 is a full XSD 1.1 calendar year, so only a nil
	// year means "missing year component" here.
	if a.year == nil || b.year == nil || a.month < 1 || b.month < 1 || a.day < 1 || b.day < 1 {
		return 0, false
	}

	// Orient so that `tz` is the timezoned operand and `plain` has no timezone.
	tz, plain := a, b
	swapped := false
	if !a.hasTZ {
		tz, plain = b, a
		swapped = true
	}

	tz = tz.normalizeToUTC()

	// Interpret the non-timezoned operand under its two extreme timezones.
	// +14:00 yields its earliest instant (largest subtraction from UTC),
	// -14:00 yields its latest instant. We compare plain against tz, so we
	// build the UTC-normalized plain value at each extreme.
	low := plain
	low.hasTZ = true
	low.tzMin = 14 * 60
	low = low.normalizeToUTC()

	high := plain
	high.hasTZ = true
	high.tzMin = -14 * 60
	high = high.normalizeToUTC()

	cmpLow := compareDateTimeFields(low, tz)
	cmpHigh := compareDateTimeFields(high, tz)

	// Both extremes on the same side → determinate result for `plain` vs `tz`.
	// orient converts that into the result for the original `a` vs `b` order:
	// when the operands were not swapped, `a` is `tz` and `b` is `plain`, so
	// the sign must be inverted.
	orient := func(cmp int) int {
		if swapped {
			return cmp
		}
		return -cmp
	}
	if cmpLow > 0 && cmpHigh > 0 {
		return orient(1), true
	}
	if cmpLow < 0 && cmpHigh < 0 {
		return orient(-1), true
	}
	return 0, false
}

func compareDateTime(a, b, builtinLocal string) (int, bool) {
	// Collapse XSD whitespace before both validation and parsing, matching the
	// numeric/binary paths: date/time lexicals have whiteSpace="collapse", so a
	// value padded with XSD whitespace is valid and must compare equal to its
	// trimmed form. NBSP and other non-XSD whitespace stays in place and is
	// rejected by ValidateBuiltin.
	a, b = trimXSDSpace(a), trimXSDSpace(b)
	if !validDateTimeOperands(a, b, builtinLocal) {
		return 0, false
	}
	da, ok1 := parseXSDDateTime(a)
	db, ok2 := parseXSDDateTime(b)
	if !ok1 || !ok2 {
		return 0, false
	}
	return compareDateTimeParsed(da, db)
}

func compareDate(a, b, builtinLocal string) (int, bool) {
	a, b = trimXSDSpace(a), trimXSDSpace(b)
	if !validDateTimeOperands(a, b, builtinLocal) {
		return 0, false
	}
	da, ok1 := parseXSDDate(a)
	db, ok2 := parseXSDDate(b)
	if !ok1 || !ok2 {
		return 0, false
	}
	return compareDateTimeParsed(da, db)
}

func compareTime(a, b, builtinLocal string) (int, bool) {
	a, b = trimXSDSpace(a), trimXSDSpace(b)
	if !validDateTimeOperands(a, b, builtinLocal) {
		return 0, false
	}
	da, ok1 := parseXSDTime(a)
	db, ok2 := parseXSDTime(b)
	if !ok1 || !ok2 {
		return 0, false
	}
	// Normalize the end-of-day 24:00:00 form to 00:00:00 before assigning a
	// reference date: xs:time carries no date, so 24:00:00 and 00:00:00 are the
	// same value and must not roll the synthetic reference date forward a day.
	da = da.normalizeHour24()
	db = db.normalizeHour24()
	// Set a reference date so TZ normalization day overflow works.
	da.year, da.month, da.day = big.NewInt(2000), 1, 15
	db.year, db.month, db.day = big.NewInt(2000), 1, 15
	return compareDateTimeParsed(da, db)
}

func compareGYear(a, b, builtinLocal string) (int, bool) {
	a, b = trimXSDSpace(a), trimXSDSpace(b)
	if !validDateTimeOperands(a, b, builtinLocal) {
		return 0, false
	}
	da, ok1 := parseXSDGYear(a)
	db, ok2 := parseXSDGYear(b)
	if !ok1 || !ok2 {
		return 0, false
	}
	return compareDateTimeParsed(da, db)
}

func compareGYearMonth(a, b, builtinLocal string) (int, bool) {
	a, b = trimXSDSpace(a), trimXSDSpace(b)
	if !validDateTimeOperands(a, b, builtinLocal) {
		return 0, false
	}
	da, ok1 := parseXSDGYearMonth(a)
	db, ok2 := parseXSDGYearMonth(b)
	if !ok1 || !ok2 {
		return 0, false
	}
	return compareDateTimeParsed(da, db)
}

func compareGMonth(a, b, builtinLocal string) (int, bool) {
	a, b = trimXSDSpace(a), trimXSDSpace(b)
	if !validDateTimeOperands(a, b, builtinLocal) {
		return 0, false
	}
	da, ok1 := parseXSDGMonth(a)
	db, ok2 := parseXSDGMonth(b)
	if !ok1 || !ok2 {
		return 0, false
	}
	return compareDateTimeParsed(da, db)
}

func compareGDay(a, b, builtinLocal string) (int, bool) {
	a, b = trimXSDSpace(a), trimXSDSpace(b)
	if !validDateTimeOperands(a, b, builtinLocal) {
		return 0, false
	}
	da, ok1 := parseXSDGDay(a)
	db, ok2 := parseXSDGDay(b)
	if !ok1 || !ok2 {
		return 0, false
	}
	return compareDateTimeParsed(da, db)
}

func compareGMonthDay(a, b, builtinLocal string) (int, bool) {
	a, b = trimXSDSpace(a), trimXSDSpace(b)
	if !validDateTimeOperands(a, b, builtinLocal) {
		return 0, false
	}
	da, ok1 := parseXSDGMonthDay(a)
	db, ok2 := parseXSDGMonthDay(b)
	if !ok1 || !ok2 {
		return 0, false
	}
	return compareDateTimeParsed(da, db)
}

type xsdDuration struct {
	negative bool
	// months is the accumulated months component (years*12 + months) held as a
	// *big.Int rather than an int so a valid lexical with a huge year/month
	// component (e.g. "P999999999999999999999999Y") that passes ValidateBuiltin
	// also compares and canonicalizes without overflow. A nil months is 0.
	months *big.Int
	// seconds is the accumulated seconds component (days/hours/minutes/seconds,
	// including fractional seconds) held as an EXACT rational rather than a
	// float64. This keeps two durations that differ only in trailing fractional
	// precision (e.g. "PT0.1S" vs "PT0.1000000000000000000000000000000000001S")
	// distinct in both Compare and CanonicalKey instead of colliding through
	// float rounding. A nil seconds is treated as 0.
	seconds *big.Rat
}

// secVal returns the seconds component as a *big.Rat, substituting 0 for nil.
func (d xsdDuration) secVal() *big.Rat {
	if d.seconds == nil {
		return new(big.Rat)
	}
	return d.seconds
}

// monVal returns the months component as a *big.Int, substituting 0 for nil.
func (d xsdDuration) monVal() *big.Int {
	if d.months == nil {
		return new(big.Int)
	}
	return d.months
}

func parseXSDDurationValue(s string) (xsdDuration, bool) {
	var d xsdDuration
	if len(s) == 0 {
		return d, false
	}
	if s[0] == '-' {
		d.negative = true
		s = s[1:]
	}
	if len(s) == 0 || s[0] != 'P' {
		return d, false
	}
	s = s[1:]
	if s == "" || s == "T" {
		return d, false
	}

	inTime := false
	for len(s) > 0 {
		if s[0] == 'T' {
			inTime = true
			s = s[1:]
			continue
		}
		// Read number (may have fractional part for seconds).
		numEnd := 0
		for numEnd < len(s) && ((s[numEnd] >= '0' && s[numEnd] <= '9') || s[numEnd] == '.') {
			numEnd++
		}
		if numEnd == 0 || numEnd >= len(s) {
			return d, false
		}
		numStr := s[:numEnd]
		designator := s[numEnd]
		s = s[numEnd+1:]

		if d.seconds == nil {
			d.seconds = new(big.Rat)
		}
		if d.months == nil {
			d.months = new(big.Int)
		}
		if !inTime {
			// numStr for non-second designators is a plain non-negative integer
			// (the scan admits digits and at most one '.', but '.' is only valid
			// for seconds and is rejected here). Parse as *big.Int so huge
			// year/month components do not overflow.
			n, ok := new(big.Int).SetString(numStr, 10)
			if !ok {
				return d, false
			}
			switch designator {
			case 'Y':
				d.months.Add(d.months, new(big.Int).Mul(n, big.NewInt(12)))
			case 'M':
				d.months.Add(d.months, n)
			case 'D':
				d.seconds.Add(d.seconds, new(big.Rat).Mul(new(big.Rat).SetInt(n), big.NewRat(86400, 1)))
			default:
				return d, false
			}
		} else {
			switch designator {
			case 'H':
				n, ok := new(big.Int).SetString(numStr, 10)
				if !ok {
					return d, false
				}
				d.seconds.Add(d.seconds, new(big.Rat).Mul(new(big.Rat).SetInt(n), big.NewRat(3600, 1)))
			case 'M':
				n, ok := new(big.Int).SetString(numStr, 10)
				if !ok {
					return d, false
				}
				d.seconds.Add(d.seconds, new(big.Rat).Mul(new(big.Rat).SetInt(n), big.NewRat(60, 1)))
			case 'S':
				// big.Rat.SetString accepts forms the seconds field must not use,
				// but the digit-run scan above admits only ASCII digits and a single
				// '.', so numStr is a plain non-negative decimal. Parse it exactly to
				// preserve full fractional precision.
				f, ok := new(big.Rat).SetString(numStr)
				if !ok {
					return d, false
				}
				d.seconds.Add(d.seconds, f)
			default:
				return d, false
			}
		}
	}
	return d, true
}

func compareDuration(a, b string) (int, bool) {
	if !validBuiltinOperands(a, b, "duration") {
		return 0, false
	}
	da, ok1 := parseXSDDurationValue(trimXSDSpace(a))
	db, ok2 := parseXSDDurationValue(trimXSDSpace(b))
	if !ok1 || !ok2 {
		return 0, false
	}

	am, as := signedDurationComponents(da)
	bm, bs := signedDurationComponents(db)
	return compareDurationComponents(am, as, bm, bs)
}

type durationReferenceDate struct {
	year       int64
	month, day int
}

var durationReferenceDates = [...]durationReferenceDate{
	{1696, 9, 1},
	{1697, 2, 1},
	{1903, 3, 1},
	{1903, 7, 1},
}

type durationTimelinePoint struct {
	day *big.Int
	sec *big.Rat
}

func signedDurationComponents(d xsdDuration) (*big.Int, *big.Rat) {
	months := new(big.Int).Set(d.monVal())
	seconds := new(big.Rat).Set(d.secVal())
	if d.negative {
		months.Neg(months)
		seconds.Neg(seconds)
	}
	return months, seconds
}

func compareDurationComponents(am *big.Int, as *big.Rat, bm *big.Int, bs *big.Rat) (int, bool) {
	result := 0
	for i, ref := range durationReferenceDates {
		ap := durationPointAtReference(ref, am, as)
		bp := durationPointAtReference(ref, bm, bs)
		cmp := compareDurationTimelinePoint(ap, bp)
		if i == 0 {
			result = cmp
			continue
		}
		if cmp != result {
			return 0, false
		}
	}
	return result, true
}

func durationPointAtReference(ref durationReferenceDate, months *big.Int, seconds *big.Rat) durationTimelinePoint {
	year, month, day := addMonthsToDate(big.NewInt(ref.year), ref.month, ref.day, months)
	ordinal := daysBeforeDate(year, month, day)
	dayOffset, secondOfDay := splitSecondsInDay(seconds)
	ordinal.Add(ordinal, dayOffset)
	return durationTimelinePoint{day: ordinal, sec: secondOfDay}
}

func compareDurationTimelinePoint(a, b durationTimelinePoint) int {
	if cmp := a.day.Cmp(b.day); cmp != 0 {
		return cmp
	}
	return a.sec.Cmp(b.sec)
}

func addMonthsToDate(year *big.Int, month, day int, delta *big.Int) (*big.Int, int, int) {
	totalMonths := new(big.Int).Mul(year, big.NewInt(12))
	totalMonths.Add(totalMonths, big.NewInt(int64(month-1)))
	totalMonths.Add(totalMonths, delta)

	newYear, monthRemainder := new(big.Int), new(big.Int)
	newYear.DivMod(totalMonths, big.NewInt(12), monthRemainder)
	newMonth := int(monthRemainder.Int64()) + 1
	if maxDay := daysInMonth(newYear, newMonth); day > maxDay {
		day = maxDay
	}
	return newYear, newMonth, day
}

func daysBeforeDate(year *big.Int, month, day int) *big.Int {
	days := daysBeforeYear(year)
	days.Add(days, big.NewInt(int64(daysBeforeMonth(year, month)+day-1)))
	return days
}

func daysBeforeYear(year *big.Int) *big.Int {
	days := new(big.Int).Mul(year, big.NewInt(365))
	days.Add(days, floorDivInt(new(big.Int).Add(year, big.NewInt(3)), 4))
	days.Sub(days, floorDivInt(new(big.Int).Add(year, big.NewInt(99)), 100))
	days.Add(days, floorDivInt(new(big.Int).Add(year, big.NewInt(399)), 400))
	return days
}

func daysBeforeMonth(year *big.Int, month int) int {
	common := [...]int{0, 31, 59, 90, 120, 151, 181, 212, 243, 273, 304, 334}
	leap := [...]int{0, 31, 60, 91, 121, 152, 182, 213, 244, 274, 305, 335}
	if month < 1 {
		return 0
	}
	if month > 12 {
		month = 12
	}
	if isLeapYearBig(year) {
		return leap[month-1]
	}
	return common[month-1]
}

func floorDivInt(n *big.Int, d int64) *big.Int {
	return new(big.Int).Div(n, big.NewInt(d))
}

func splitSecondsInDay(seconds *big.Rat) (*big.Int, *big.Rat) {
	numerator := seconds.Num()
	divisor := new(big.Int).Mul(seconds.Denom(), big.NewInt(86400))
	dayOffset, remainder := new(big.Int), new(big.Int)
	dayOffset.DivMod(numerator, divisor, remainder)
	secondOfDay := new(big.Rat).SetFrac(remainder, seconds.Denom())
	return dayOffset, secondOfDay
}
