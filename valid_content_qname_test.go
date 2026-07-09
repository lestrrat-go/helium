package helium_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestContentModelQNameMatching verifies that DTD content-model validation
// compares raw qualified names (prefix + local, as written), not local names
// only. DTD validation is not namespace-aware: a declared prefix is an opaque
// part of the element name and must match the instance tag literally.
func TestContentModelQNameMatching(t *testing.T) {
	t.Parallel()

	parse := func(t *testing.T, xml string) error {
		t.Helper()
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		p := helium.NewParser().ValidateDTD(true).ErrorHandler(collector)
		_, err := p.Parse(t.Context(), []byte(xml))
		return err
	}

	t.Run("element content (p:a) rejects unprefixed <a/>", func(t *testing.T) {
		t.Parallel()
		xml := `<?xml version="1.0"?>
<!DOCTYPE r [
  <!ELEMENT r (p:a)>
  <!ELEMENT p:a EMPTY>
  <!ELEMENT a EMPTY>
]>
<r xmlns:p="u"><a/></r>`
		require.ErrorIs(t, parse(t, xml), helium.ErrDTDValidationFailed)
	})

	t.Run("element content (p:a) accepts <p:a/>", func(t *testing.T) {
		t.Parallel()
		xml := `<?xml version="1.0"?>
<!DOCTYPE r [
  <!ELEMENT r (p:a)>
  <!ELEMENT p:a EMPTY>
  <!ELEMENT a EMPTY>
]>
<r xmlns:p="u"><p:a/></r>`
		require.NoError(t, parse(t, xml))
	})

	t.Run("element content (a) rejects prefixed <p:a/>", func(t *testing.T) {
		t.Parallel()
		xml := `<?xml version="1.0"?>
<!DOCTYPE r [
  <!ELEMENT r (a)>
  <!ELEMENT p:a EMPTY>
  <!ELEMENT a EMPTY>
]>
<r xmlns:p="u"><p:a/></r>`
		require.ErrorIs(t, parse(t, xml), helium.ErrDTDValidationFailed)
	})

	t.Run("mixed content (#PCDATA|p:a)* rejects unprefixed <a/>", func(t *testing.T) {
		t.Parallel()
		xml := `<?xml version="1.0"?>
<!DOCTYPE r [
  <!ELEMENT r (#PCDATA|p:a)*>
  <!ELEMENT p:a EMPTY>
  <!ELEMENT a EMPTY>
]>
<r xmlns:p="u">text<a/></r>`
		require.ErrorIs(t, parse(t, xml), helium.ErrDTDValidationFailed)
	})

	t.Run("mixed content (#PCDATA|p:a)* accepts <p:a/>", func(t *testing.T) {
		t.Parallel()
		xml := `<?xml version="1.0"?>
<!DOCTYPE r [
  <!ELEMENT r (#PCDATA|p:a)*>
  <!ELEMENT p:a EMPTY>
  <!ELEMENT a EMPTY>
]>
<r xmlns:p="u">text<p:a/></r>`
		require.NoError(t, parse(t, xml))
	})

	t.Run("unprefixed element content unchanged", func(t *testing.T) {
		t.Parallel()
		xml := `<?xml version="1.0"?>
<!DOCTYPE r [
  <!ELEMENT r (a, b)>
  <!ELEMENT a EMPTY>
  <!ELEMENT b EMPTY>
]>
<r><a/><b/></r>`
		require.NoError(t, parse(t, xml))
	})

	t.Run("unprefixed mixed content unchanged", func(t *testing.T) {
		t.Parallel()
		xml := `<?xml version="1.0"?>
<!DOCTYPE r [
  <!ELEMENT r (#PCDATA|a)*>
  <!ELEMENT a EMPTY>
]>
<r>text<a/>more</r>`
		require.NoError(t, parse(t, xml))
	})
}
