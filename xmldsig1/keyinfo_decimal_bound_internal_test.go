package xmldsig1

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// The two decimal integers a ds:KeyInfo carries as text rather than base64 — the
// ds:X509SerialNumber and the RFC 4050 PublicKey X/Y Value attributes — are
// converted to a big.Int, and that conversion is quadratic in the digit count.
// Both are read before the SignatureValue is checked, so both have a fixed digit
// ceiling and both must be refused BEFORE the conversion. The tests below pin
// each ceiling from both sides and then pin the cost, because an accept/refuse
// pair alone cannot see whether the refusal came before the work or after it.

// p256PointX and p256PointY are a real point on P-256, taken from the Apache
// Santuario RFC 4050 interop vector in testdata. A padded copy of a genuine
// point is what makes the at-ceiling cases below true accepts: the parse converts
// the value AND weighs it against the curve, so a made-up number would be refused
// for being off the curve and prove nothing about the ceiling.
const (
	p256PointX = "72346047708883099073857357917841715755940175004927717314128082527981683978864"
	p256PointY = "24418914917061776918936231657090344308413753520069738480182871474056860317726"
)

// maxConformingX509Serial is 2^160-1, the largest serial RFC 5280 §4.1.2.2
// admits: 20 octets, written out as 49 decimal digits.
const maxConformingX509Serial = "1461501637330902918203684832716283019655932542975"

// issuerSerialKeyInfo builds a ds:KeyInfo whose single X509IssuerSerial carries
// serial as its X509SerialNumber. The issuer name is present because
// parseX509IssuerSerial requires both children.
func issuerSerialKeyInfo(t *testing.T, serial string) *helium.Document {
	t.Helper()
	return mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`"><ds:X509Data><ds:X509IssuerSerial>`+
		`<ds:X509IssuerName>CN=Test CA,O=Example,C=US</ds:X509IssuerName>`+
		`<ds:X509SerialNumber>`+serial+`</ds:X509SerialNumber>`+
		`</ds:X509IssuerSerial></ds:X509Data></ds:KeyInfo>`)
}

// rfc4050KeyInfo builds a ds:KeyInfo carrying an RFC 4050 ECDSAKeyValue on P-256
// whose PublicKey X and Y Value attributes are x and y.
func rfc4050KeyInfo(t *testing.T, x, y string) *helium.Document {
	t.Helper()
	return mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`"><ds:KeyValue>`+
		`<ECDSAKeyValue xmlns="`+NamespaceDSigMore+`">`+
		`<DomainParameters><NamedCurve URN="urn:oid:1.2.840.10045.3.1.7"/></DomainParameters>`+
		`<PublicKey><X Value="`+x+`"/><Y Value="`+y+`"/></PublicKey>`+
		`</ECDSAKeyValue></ds:KeyValue></ds:KeyInfo>`)
}

// padDecimal returns value left-padded with zeros to exactly digits characters.
// Leading zeros do not change what a base-10 conversion yields, so a padded copy
// of a conforming value is the same value written at whatever length a case needs
// — which is how the at-ceiling cases stay real values rather than arbitrary
// digit strings.
func padDecimal(t *testing.T, value string, digits int) string {
	t.Helper()
	require.LessOrEqual(t, len(value), digits, "value is already longer than %d digits", digits)
	return strings.Repeat("0", digits-len(value)) + value
}

// parseKeyInfoDoc runs the KeyInfo parse over doc's document element under the
// default budget, which is the whole path a verification takes before the
// SignatureValue is checked.
func parseKeyInfoDoc(t *testing.T, doc *helium.Document) (*KeyInfoData, error) {
	t.Helper()
	return parseKeyInfo(t.Context(), newVerifyBudget(&verifierConfig{}), doc.DocumentElement())
}

// TestX509SerialNumberDigitBound pins maxX509SerialNumberDigits from both sides
// and confirms the largest serial RFC 5280 admits is nowhere near it.
func TestX509SerialNumberDigitBound(t *testing.T) {
	overLimit := fmt.Sprintf("X509SerialNumber is over the %d digit limit", maxX509SerialNumberDigits)

	t.Run("a serial at the ceiling is accepted", func(t *testing.T) {
		serial := padDecimal(t, maxConformingX509Serial, maxX509SerialNumberDigits)
		data, err := parseKeyInfoDoc(t, issuerSerialKeyInfo(t, serial))
		require.NoError(t, err)
		require.Len(t, data.X509IssuerSerials, 1)
		require.Equal(t, maxConformingX509Serial, data.X509IssuerSerials[0].SerialNumber.String())
	})

	t.Run("a serial one digit over the ceiling is refused", func(t *testing.T) {
		serial := padDecimal(t, maxConformingX509Serial, maxX509SerialNumberDigits+1)
		_, err := parseKeyInfoDoc(t, issuerSerialKeyInfo(t, serial))
		require.ErrorIs(t, err, ErrInvalidKeyInfo)
		require.ErrorContains(t, err, overLimit)
	})

	// The ceiling has to be loose enough that no real issuer ever meets it. The
	// largest conforming serial is 49 digits, so this is the case that would fail
	// if the ceiling were ever tightened to something a certificate can reach.
	t.Run("the largest conforming serial is accepted", func(t *testing.T) {
		data, err := parseKeyInfoDoc(t, issuerSerialKeyInfo(t, maxConformingX509Serial))
		require.NoError(t, err)
		require.Len(t, data.X509IssuerSerials, 1)
		require.Equal(t, maxConformingX509Serial, data.X509IssuerSerials[0].SerialNumber.String())
	})
}

// TestRFC4050CoordinateDigitBound pins maxRFC4050CoordinateDigits from both sides
// at BOTH the X and the Y attribute. A ceiling applied to only one of them is not
// a ceiling: the other attribute is read the same way in the same loop and would
// be left as a way around it.
func TestRFC4050CoordinateDigitBound(t *testing.T) {
	atCeiling := struct{ x, y string }{
		x: padDecimal(t, p256PointX, maxRFC4050CoordinateDigits),
		y: padDecimal(t, p256PointY, maxRFC4050CoordinateDigits),
	}

	t.Run("both coordinates at the ceiling are accepted", func(t *testing.T) {
		data, err := parseKeyInfoDoc(t, rfc4050KeyInfo(t, atCeiling.x, atCeiling.y))
		require.NoError(t, err)
		require.NotNil(t, data.ECKeyValue)
		require.Equal(t, p256PointX, data.ECKeyValue.X.String())
		require.Equal(t, p256PointY, data.ECKeyValue.Y.String())
	})

	for _, tc := range []struct {
		name string
		x    string
		y    string
		// coordinate is the attribute the refusal must name, so a check that
		// fired for the other one cannot pass this case.
		coordinate string
	}{
		{
			name:       "an X one digit over the ceiling is refused",
			x:          padDecimal(t, p256PointX, maxRFC4050CoordinateDigits+1),
			y:          atCeiling.y,
			coordinate: "X",
		},
		{
			name:       "a Y one digit over the ceiling is refused",
			x:          atCeiling.x,
			y:          padDecimal(t, p256PointY, maxRFC4050CoordinateDigits+1),
			coordinate: "Y",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseKeyInfoDoc(t, rfc4050KeyInfo(t, tc.x, tc.y))
			require.ErrorIs(t, err, ErrInvalidKeyInfo)
			require.ErrorContains(t, err, fmt.Sprintf("RFC 4050 PublicKey %s Value is over the %d digit limit",
				tc.coordinate, maxRFC4050CoordinateDigits))
		})
	}
}

// decimalAllocDigits is how many digits the over-ceiling documents below write,
// and decimalAllocSlack is what each case allows on top of reading that text
// once.
//
// The slack is a CONSTANT, deliberately: a bound stated as a multiple of the
// attacker's own text can never fail on a cost that follows that text, which is
// the only cost being pinned here. It covers the rest of the KeyInfo, the error
// that is returned, and the runtime's own bookkeeping — including the race
// detector's, which is a megabyte of it when the suite runs under -race. All of
// those are fixed, and the whole of the slack still leaves more than an order of
// magnitude between it and what the conversion alone would cost.
const (
	decimalAllocDigits = 1 << 18
	decimalAllocSlack  = 1 << 22
)

// TestDecimalBoundAllocation pins what an over-ceiling decimal value may
// ALLOCATE, which the refusals above cannot see. A ceiling checked AFTER the
// conversion returns exactly the same error for exactly the same document, so
// the assertions above would pass unchanged while the quadratic cost was paid in
// full — 262144 digits alone convert to a big.Int in about 150 MB of scratch,
// six hundred times the text that asked for it. This is the test that says the
// refusal comes first, and it is what a future rewrite that validates after
// parsing would have to break.
//
// Each case reads the process-wide TotalAlloc delta across one KeyInfo parse, so
// these subtests must NOT run in parallel: a concurrent test's allocations would
// pollute the delta.
func TestDecimalBoundAllocation(t *testing.T) {
	// no t.Parallel(): isolated so each delta reflects only its own parse.

	// Reading the value once is the floor a bound has to clear, and only the
	// serial pays it: helium builds an element's text content by growing a buffer
	// to the text's length and copying it into a string, about three copies, so
	// four is the headroom. An attribute value is handed over as the string the
	// parser already holds, so the coordinate cases carry no lexical term at all.
	const (
		oneTextRead  = decimalAllocDigits*4 + decimalAllocSlack
		noTextCopied = decimalAllocSlack
	)

	overCeiling := strings.Repeat("9", decimalAllocDigits)

	for _, tc := range []struct {
		name string
		// base is a document whose values are all conforming, and over is the
		// same document with one value written at decimalAllocDigits digits. The
		// bound is on the difference, so everything the two share drops out.
		base     *helium.Document
		over     *helium.Document
		maxAlloc uint64
	}{
		{
			name:     "over-ceiling X509SerialNumber",
			base:     issuerSerialKeyInfo(t, maxConformingX509Serial),
			over:     issuerSerialKeyInfo(t, overCeiling),
			maxAlloc: oneTextRead,
		},
		{
			name:     "over-ceiling RFC 4050 X coordinate",
			base:     rfc4050KeyInfo(t, p256PointX, p256PointY),
			over:     rfc4050KeyInfo(t, overCeiling, p256PointY),
			maxAlloc: noTextCopied,
		},
		{
			// Y is charged separately: a document whose X is conforming reaches
			// the Y attribute, so an unbounded Y is spent even when X is not.
			name:     "over-ceiling RFC 4050 Y coordinate",
			base:     rfc4050KeyInfo(t, p256PointX, p256PointY),
			over:     rfc4050KeyInfo(t, p256PointX, overCeiling),
			maxAlloc: noTextCopied,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			baseAlloc := parseKeyInfoAlloc(t, tc.base, false)
			overAlloc := parseKeyInfoAlloc(t, tc.over, true)

			require.Less(t, overAlloc-baseAlloc, tc.maxAlloc,
				"refusing %d digits allocated %d bytes over the %d the conforming document cost",
				decimalAllocDigits, overAlloc-baseAlloc, baseAlloc)
		})
	}
}

// parseKeyInfoAlloc reports the TotalAlloc delta across one parse of doc's
// KeyInfo. The document is built by the caller, outside the measurement, so the
// delta is the parse alone. wantErr says whether that parse must fail, checked
// after the measurement is taken so the assertion is not part of it.
func parseKeyInfoAlloc(t *testing.T, doc *helium.Document, wantErr bool) uint64 {
	t.Helper()

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	_, err := parseKeyInfo(t.Context(), newVerifyBudget(&verifierConfig{}), doc.DocumentElement())
	runtime.ReadMemStats(&after)

	if wantErr {
		require.ErrorIs(t, err, ErrInvalidKeyInfo)
	}
	if !wantErr {
		require.NoError(t, err)
	}
	return after.TotalAlloc - before.TotalAlloc
}
