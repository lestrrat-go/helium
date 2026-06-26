package relaxng_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDataOrderingFacets covers the XSD <param> ordering facets on a RELAX NG
// <data> pattern (min/maxInclusive, min/maxExclusive). These were previously
// ignored (silently accepted), so an out-of-range value such as 5 against
// minInclusive=10 wrongly validated. They are now enforced through the shared
// XSD value engine.
func TestDataOrderingFacets(t *testing.T) {
	t.Parallel()

	const xsd = `datatypeLibrary="http://www.w3.org/2001/XMLSchema-datatypes"`

	mk := func(facet, bound string) string {
		return `<element name="r" xmlns="http://relaxng.org/ns/structure/1.0" ` + xsd + `>
  <data type="integer"><param name="` + facet + `">` + bound + `</param></data>
</element>`
	}

	t.Run("minInclusive rejects below bound", func(t *testing.T) {
		t.Parallel()
		err := validateWith(t, mk("minInclusive", "10"), `<r>5</r>`)
		require.Error(t, err, "5 must be rejected by minInclusive=10")
	})

	t.Run("minInclusive accepts bound", func(t *testing.T) {
		t.Parallel()
		err := validateWith(t, mk("minInclusive", "10"), `<r>10</r>`)
		require.NoError(t, err, "10 must satisfy minInclusive=10")
	})

	t.Run("minInclusive accepts above bound", func(t *testing.T) {
		t.Parallel()
		err := validateWith(t, mk("minInclusive", "10"), `<r>15</r>`)
		require.NoError(t, err, "15 must satisfy minInclusive=10")
	})

	t.Run("maxInclusive rejects above bound", func(t *testing.T) {
		t.Parallel()
		err := validateWith(t, mk("maxInclusive", "8"), `<r>9</r>`)
		require.Error(t, err, "9 must be rejected by maxInclusive=8")
	})

	t.Run("maxInclusive accepts bound", func(t *testing.T) {
		t.Parallel()
		err := validateWith(t, mk("maxInclusive", "8"), `<r>8</r>`)
		require.NoError(t, err, "8 must satisfy maxInclusive=8")
	})

	t.Run("minExclusive rejects bound", func(t *testing.T) {
		t.Parallel()
		err := validateWith(t, mk("minExclusive", "10"), `<r>10</r>`)
		require.Error(t, err, "10 must be rejected by minExclusive=10")
	})

	t.Run("minExclusive accepts above bound", func(t *testing.T) {
		t.Parallel()
		err := validateWith(t, mk("minExclusive", "10"), `<r>11</r>`)
		require.NoError(t, err, "11 must satisfy minExclusive=10")
	})

	t.Run("maxExclusive rejects bound", func(t *testing.T) {
		t.Parallel()
		err := validateWith(t, mk("maxExclusive", "8"), `<r>8</r>`)
		require.Error(t, err, "8 must be rejected by maxExclusive=8")
	})

	t.Run("maxExclusive accepts below bound", func(t *testing.T) {
		t.Parallel()
		err := validateWith(t, mk("maxExclusive", "8"), `<r>7</r>`)
		require.NoError(t, err, "7 must satisfy maxExclusive=8")
	})

	t.Run("combined range rejects out-of-range and accepts in-range", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="r" xmlns="http://relaxng.org/ns/structure/1.0" ` + xsd + `>
  <data type="integer">
    <param name="minInclusive">1</param>
    <param name="maxInclusive">8</param>
  </data>
</element>`
		require.Error(t, validateWith(t, schema, `<r>9</r>`), "9 must be out of [1,8]")
		require.Error(t, validateWith(t, schema, `<r>0</r>`), "0 must be out of [1,8]")
		require.NoError(t, validateWith(t, schema, `<r>4</r>`), "4 must be in [1,8]")
	})

	t.Run("ordering facet on string family is a compile error", func(t *testing.T) {
		t.Parallel()
		// Ordering facets are undefined for string-family types (their value space
		// is not ordered), so the facet is inapplicable and rejected at compile
		// time rather than silently accepted at validation.
		schema := `<element name="r" xmlns="http://relaxng.org/ns/structure/1.0" ` + xsd + `>
  <data type="string"><param name="minInclusive">b</param></data>
</element>`
		require.NotEmpty(t, compileErrorsFor(t, schema), "ordering facet on xs:string must be a compile error")
	})

	t.Run("ordering facet with invalid bound is a compile error", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="r" xmlns="http://relaxng.org/ns/structure/1.0" ` + xsd + `>
  <data type="integer"><param name="minInclusive">notanint</param></data>
</element>`
		require.NotEmpty(t, compileErrorsFor(t, schema), "invalid integer bound must be a compile error")
	})

	t.Run("ordering facet on bare built-in token is not a compile error", func(t *testing.T) {
		t.Parallel()
		// With NO datatypeLibrary declared, "token" is the EMPTY built-in RELAX NG
		// datatype: matchData resolves it first and accepts any text, applying no XSD
		// ordering facets. The compile-time facet check must therefore agree and not
		// raise a spurious "non-ordered datatype" error, even though XSD's xs:token
		// would reject an ordering facet. The facet is inert at runtime.
		schema := `<element name="r" xmlns="http://relaxng.org/ns/structure/1.0">
  <data type="token"><param name="minInclusive">1</param></data>
</element>`
		require.Empty(t, compileErrorsFor(t, schema),
			"ordering facet on bare built-in token must not be a compile error")
		require.NoError(t, validateWith(t, schema, `<r>anything</r>`),
			"bare built-in token accepts any text regardless of the inert facet")
	})

	t.Run("ordering facet on bare built-in string is not a compile error", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="r" xmlns="http://relaxng.org/ns/structure/1.0">
  <data type="string"><param name="minInclusive">b</param></data>
</element>`
		require.Empty(t, compileErrorsFor(t, schema),
			"ordering facet on bare built-in string must not be a compile error")
		require.NoError(t, validateWith(t, schema, `<r>anything</r>`),
			"bare built-in string accepts any text regardless of the inert facet")
	})
}

// TestDataOrderingFacetsFloatNaN covers that the bounding facets EXCLUDE
// incomparable xs:float/xs:double NaN values: a NaN instance value fails a finite
// bound, and a NaN bound fails a finite instance value. value.Compare reports NaN
// comparisons as indeterminate, which must reject (not satisfy) a range facet.
func TestDataOrderingFacetsFloatNaN(t *testing.T) {
	t.Parallel()

	const xsd = `datatypeLibrary="http://www.w3.org/2001/XMLSchema-datatypes"`

	mk := func(typ, facet, bound string) string {
		return `<element name="r" xmlns="http://relaxng.org/ns/structure/1.0" ` + xsd + `>
  <data type="` + typ + `"><param name="` + facet + `">` + bound + `</param></data>
</element>`
	}

	for _, typ := range []string{"float", "double"} {
		t.Run(typ+" NaN instance rejected by finite minInclusive", func(t *testing.T) {
			t.Parallel()
			err := validateWith(t, mk(typ, "minInclusive", "0"), `<r>NaN</r>`)
			require.Error(t, err, "NaN must be excluded by minInclusive=0")
		})

		t.Run(typ+" NaN instance rejected by finite maxInclusive", func(t *testing.T) {
			t.Parallel()
			err := validateWith(t, mk(typ, "maxInclusive", "0"), `<r>NaN</r>`)
			require.Error(t, err, "NaN must be excluded by maxInclusive=0")
		})

		t.Run(typ+" finite instance rejected by NaN minInclusive bound", func(t *testing.T) {
			t.Parallel()
			err := validateWith(t, mk(typ, "minInclusive", "NaN"), `<r>1.5</r>`)
			require.Error(t, err, "a NaN bound must not accept a finite value")
		})

		t.Run(typ+" finite instance rejected by whitespace-padded NaN bound", func(t *testing.T) {
			t.Parallel()
			err := validateWith(t, mk(typ, "minInclusive", " NaN "), `<r>1.5</r>`)
			require.Error(t, err, "a whitespace-padded NaN bound must not accept a finite value")
		})

		t.Run(typ+" finite instance accepted by finite bound", func(t *testing.T) {
			t.Parallel()
			err := validateWith(t, mk(typ, "minInclusive", "0"), `<r>1.5</r>`)
			require.NoError(t, err, "1.5 must satisfy minInclusive=0")
		})
	}
}

// TestDataDigitFacets covers the XSD totalDigits/fractionDigits <param> facets on
// a RELAX NG <data> pattern. These are valid facets for the xs:decimal family and
// constrain digit counts, so a value within the allowed digits is accepted and one
// exceeding it is rejected.
func TestDataDigitFacets(t *testing.T) {
	t.Parallel()

	const xsd = `datatypeLibrary="http://www.w3.org/2001/XMLSchema-datatypes"`

	t.Run("totalDigits accepts value within bound", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="r" xmlns="http://relaxng.org/ns/structure/1.0" ` + xsd + `>
  <data type="integer"><param name="totalDigits">3</param></data>
</element>`
		require.NoError(t, validateWith(t, schema, `<r>5</r>`), "5 has 1 digit, within totalDigits=3")
	})

	t.Run("totalDigits rejects value exceeding bound", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="r" xmlns="http://relaxng.org/ns/structure/1.0" ` + xsd + `>
  <data type="integer"><param name="totalDigits">3</param></data>
</element>`
		require.Error(t, validateWith(t, schema, `<r>5000</r>`), "5000 has 4 digits, exceeds totalDigits=3")
	})

	t.Run("fractionDigits accepts value within bound", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="r" xmlns="http://relaxng.org/ns/structure/1.0" ` + xsd + `>
  <data type="decimal"><param name="fractionDigits">2</param></data>
</element>`
		require.NoError(t, validateWith(t, schema, `<r>1.20</r>`), "1.20 has 1 significant fraction digit, within fractionDigits=2")
	})

	t.Run("fractionDigits rejects value exceeding bound", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="r" xmlns="http://relaxng.org/ns/structure/1.0" ` + xsd + `>
  <data type="decimal"><param name="fractionDigits">2</param></data>
</element>`
		require.Error(t, validateWith(t, schema, `<r>1.234</r>`), "1.234 has 3 fraction digits, exceeds fractionDigits=2")
	})

	t.Run("digit facet on non-decimal type is a compile error", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="r" xmlns="http://relaxng.org/ns/structure/1.0" ` + xsd + `>
  <data type="string"><param name="totalDigits">3</param></data>
</element>`
		require.NotEmpty(t, compileErrorsFor(t, schema), "totalDigits on xs:string must be a compile error")
	})
}

// TestDataDigitFacetsLexical covers the XSD lexical validation of the digit-facet
// bounds: a bound padded with NBSP (not XSD whitespace) is a compile error, an
// out-of-value-space bound (totalDigits=0) is a compile error, and an arbitrarily
// large bound is honored faithfully via big.Int instead of overflowing int into a
// reject-all.
func TestDataDigitFacetsLexical(t *testing.T) {
	t.Parallel()

	const xsd = `datatypeLibrary="http://www.w3.org/2001/XMLSchema-datatypes"`

	// nbsp is U+00A0 — NOT XSD whitespace, so it does not collapse away and a bound
	// padded with it is not a valid integer lexical. Go's
	// strconv.Atoi(strings.TrimSpace(...)) treats it as Unicode space and would have
	// wrongly accepted it.
	nbsp := string(rune(0x00A0))

	mk := func(typ, facet, bound string) string {
		return `<element name="r" xmlns="http://relaxng.org/ns/structure/1.0" ` + xsd + `>
  <data type="` + typ + `"><param name="` + facet + `">` + bound + `</param></data>
</element>`
	}

	t.Run("NBSP-padded totalDigits is a compile error", func(t *testing.T) {
		t.Parallel()
		require.NotEmpty(t, compileErrorsFor(t, mk("integer", "totalDigits", nbsp+"3")),
			"NBSP-padded totalDigits must be a compile error")
	})

	t.Run("NBSP-padded fractionDigits is a compile error", func(t *testing.T) {
		t.Parallel()
		require.NotEmpty(t, compileErrorsFor(t, mk("decimal", "fractionDigits", nbsp+"2")),
			"NBSP-padded fractionDigits must be a compile error")
	})

	t.Run("totalDigits zero is a compile error", func(t *testing.T) {
		t.Parallel()
		// totalDigits is an xs:positiveInteger, so 0 is out of its value space.
		require.NotEmpty(t, compileErrorsFor(t, mk("integer", "totalDigits", "0")),
			"totalDigits=0 must be a compile error (xs:positiveInteger)")
	})

	t.Run("fractionDigits zero is allowed", func(t *testing.T) {
		t.Parallel()
		// fractionDigits is an xs:nonNegativeInteger, so 0 is valid.
		schema := mk("integer", "fractionDigits", "0")
		require.Empty(t, compileErrorsFor(t, schema), "fractionDigits=0 must compile")
		require.NoError(t, validateWith(t, schema, `<r>5</r>`), "5 has 0 fraction digits")
	})

	t.Run("very large totalDigits accepts any value", func(t *testing.T) {
		t.Parallel()
		// A bound far beyond int range must not overflow into a reject-all: every
		// finite value has fewer digits, so it is accepted.
		schema := mk("integer", "totalDigits", "99999999999999999999999999999999")
		require.Empty(t, compileErrorsFor(t, schema), "huge totalDigits must compile")
		require.NoError(t, validateWith(t, schema, `<r>123456789</r>`),
			"a huge totalDigits bound must accept normal values, not reject all")
	})

	t.Run("very large fractionDigits accepts any value", func(t *testing.T) {
		t.Parallel()
		schema := mk("decimal", "fractionDigits", "99999999999999999999999999999999")
		require.Empty(t, compileErrorsFor(t, schema), "huge fractionDigits must compile")
		require.NoError(t, validateWith(t, schema, `<r>1.234</r>`),
			"a huge fractionDigits bound must accept normal values, not reject all")
	})
}

// TestDataUnsupportedFacetFailsClosed covers that an unrecognized / unsupported
// <param> facet (a genuinely unknown name) is not silently accepted: a <data>
// carrying one rejects its instance rather than matching anything.
func TestDataUnsupportedFacetFailsClosed(t *testing.T) {
	t.Parallel()

	schema := `<element name="r" xmlns="http://relaxng.org/ns/structure/1.0" datatypeLibrary="http://www.w3.org/2001/XMLSchema-datatypes">
  <data type="integer"><param name="bogusFacet">3</param></data>
</element>`
	err := validateWith(t, schema, `<r>5</r>`)
	require.Error(t, err, "unknown facet bogusFacet must fail closed")
}
