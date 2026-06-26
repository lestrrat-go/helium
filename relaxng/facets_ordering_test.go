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
