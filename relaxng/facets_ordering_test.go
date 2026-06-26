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

	t.Run("ordering facet on string family fails closed", func(t *testing.T) {
		t.Parallel()
		// Ordering facets are undefined for string-family types, so value.Compare
		// cannot order the operands; the value must be rejected, not accepted.
		schema := `<element name="r" xmlns="http://relaxng.org/ns/structure/1.0" ` + xsd + `>
  <data type="string"><param name="minInclusive">b</param></data>
</element>`
		require.Error(t, validateWith(t, schema, `<r>c</r>`), "ordering facet on xs:string must fail closed")
	})
}

// TestDataUnsupportedFacetFailsClosed covers that an unrecognized / unsupported
// <param> facet (e.g. totalDigits) is no longer silently accepted: a <data>
// carrying one rejects its instance rather than matching anything.
func TestDataUnsupportedFacetFailsClosed(t *testing.T) {
	t.Parallel()

	schema := `<element name="r" xmlns="http://relaxng.org/ns/structure/1.0" datatypeLibrary="http://www.w3.org/2001/XMLSchema-datatypes">
  <data type="integer"><param name="totalDigits">3</param></data>
</element>`
	err := validateWith(t, schema, `<r>5</r>`)
	require.Error(t, err, "unsupported facet totalDigits must fail closed")
}
