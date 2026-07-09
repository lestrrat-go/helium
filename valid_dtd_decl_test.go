package helium_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// parseValidatingDTD parses src as a validating processor with an internal
// subset, returning any collected validity errors and the parse error.
func parseValidatingDTD(t *testing.T, src string) ([]error, error) {
	t.Helper()
	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	p := helium.NewParser().
		ValidateDTD(true).
		DefaultDTDAttributes(true).
		ErrorHandler(collector)
	_, err := p.Parse(t.Context(), []byte(src))
	return collector.Errors(), err
}

// TestDTDDeclValidation exercises the DTD-declaration-consistency VCs added in
// validateDTDDeclarations. Each VC has a rejecting case and a valid near-miss to
// prove there is no over-rejection.
func TestDTDDeclValidation(t *testing.T) {
	t.Parallel()

	// No Duplicate Types (§3.2.2)
	t.Run("no duplicate types", func(t *testing.T) {
		t.Parallel()

		t.Run("duplicate rejected", func(t *testing.T) {
			t.Parallel()
			errs, err := parseValidatingDTD(t, `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root (#PCDATA|a|a)*>
  <!ELEMENT a (#PCDATA)>
]>
<root/>`)
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
			require.True(t, containsError(errs, "duplicate references of a"))
		})

		t.Run("distinct names accepted", func(t *testing.T) {
			t.Parallel()
			_, err := parseValidatingDTD(t, `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root (#PCDATA|a|b)*>
  <!ELEMENT a (#PCDATA)>
  <!ELEMENT b (#PCDATA)>
]>
<root/>`)
			require.NoError(t, err)
		})
	})

	// Notation Attributes — value among the enumerated notations (§3.3.1)
	t.Run("notation value in enum", func(t *testing.T) {
		t.Parallel()

		t.Run("declared notation outside the attr enum rejected", func(t *testing.T) {
			t.Parallel()
			errs, err := parseValidatingDTD(t, `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
  <!ATTLIST root type NOTATION (fruit|vegetable) #REQUIRED>
  <!NOTATION fruit SYSTEM "f">
  <!NOTATION vegetable SYSTEM "v">
  <!NOTATION candy SYSTEM "c">
]>
<root type="candy"/>`)
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
			require.True(t, containsError(errs, "not among the enumerated notations"))
		})

		t.Run("enumerated notation accepted", func(t *testing.T) {
			t.Parallel()
			// The element uses ANY content, not EMPTY: a NOTATION attribute on an
			// EMPTY element is itself invalid (No Notation on Empty Element VC), so
			// it would not be a genuine no-over-rejection witness.
			_, err := parseValidatingDTD(t, `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root ANY>
  <!ATTLIST root type NOTATION (fruit|vegetable) #REQUIRED>
  <!NOTATION fruit SYSTEM "f">
  <!NOTATION vegetable SYSTEM "v">
]>
<root type="fruit"/>`)
			require.NoError(t, err)
		})
	})

	// Attribute Default Legal (§3.3.2)
	t.Run("attribute default legal", func(t *testing.T) {
		t.Parallel()

		t.Run("enumeration default outside set rejected", func(t *testing.T) {
			t.Parallel()
			errs, err := parseValidatingDTD(t, `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
  <!ATTLIST root v (one|two|three) "four">
]>
<root v="one"/>`)
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
			require.True(t, containsError(errs, "not among the enumerated set"))
		})

		t.Run("notation default outside set rejected", func(t *testing.T) {
			t.Parallel()
			errs, err := parseValidatingDTD(t, `<?xml version="1.0"?>
<!DOCTYPE foo [
  <!ELEMENT foo ANY>
  <!ATTLIST foo a NOTATION (not) "not2">
  <!NOTATION not SYSTEM "n">
  <!NOTATION not2 SYSTEM "n2">
]>
<foo a="not"/>`)
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
			require.True(t, containsError(errs, "not among the enumerated set"))
		})

		t.Run("empty-string enumeration default rejected", func(t *testing.T) {
			t.Parallel()
			// A literal empty default is still a default value and must be one of
			// the enumerated tokens; the empty string never is.
			errs, err := parseValidatingDTD(t, `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
  <!ATTLIST root v (one|two) "">
]>
<root v="one"/>`)
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
			require.True(t, containsError(errs, "not among the enumerated set"))
		})

		t.Run("legal default accepted", func(t *testing.T) {
			t.Parallel()
			_, err := parseValidatingDTD(t, `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
  <!ATTLIST root v (one|two|three) "two">
]>
<root/>`)
			require.NoError(t, err)
		})
	})

	// ID Attribute Default + One ID per Element Type (§3.3.1)
	t.Run("id attribute rules", func(t *testing.T) {
		t.Parallel()

		t.Run("fixed ID default rejected", func(t *testing.T) {
			t.Parallel()
			errs, err := parseValidatingDTD(t, `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root ANY>
  <!ATTLIST root id ID #FIXED "x23">
]>
<root/>`)
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
			require.True(t, containsError(errs, "must be declared #IMPLIED or #REQUIRED"))
		})

		t.Run("literal ID default rejected", func(t *testing.T) {
			t.Parallel()
			errs, err := parseValidatingDTD(t, `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root ANY>
  <!ATTLIST root id ID "bogus">
]>
<root/>`)
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
			require.True(t, containsError(errs, "must be declared #IMPLIED or #REQUIRED"))
		})

		t.Run("two ID attributes rejected", func(t *testing.T) {
			t.Parallel()
			errs, err := parseValidatingDTD(t, `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root ANY>
  <!ELEMENT a EMPTY>
  <!ATTLIST a first ID #REQUIRED>
  <!ATTLIST a second ID #REQUIRED>
]>
<root><a first="x1" second="x2"/></root>`)
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
			require.True(t, containsError(errs, "more than one ID attribute"))
		})

		t.Run("single implied ID accepted", func(t *testing.T) {
			t.Parallel()
			_, err := parseValidatingDTD(t, `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
  <!ATTLIST root id ID #IMPLIED>
]>
<root id="x1"/>`)
			require.NoError(t, err)
		})
	})

	// Notation Declared (§4.7)
	t.Run("notation declared", func(t *testing.T) {
		t.Parallel()

		t.Run("undeclared NDATA notation rejected", func(t *testing.T) {
			t.Parallel()
			errs, err := parseValidatingDTD(t, `<?xml version="1.0"?>
<!DOCTYPE paper [
  <!ELEMENT paper EMPTY>
  <!ENTITY pic SYSTEM "pic.gif" NDATA gif>
]>
<paper/>`)
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
			require.True(t, containsError(errs, `NDATA notation "gif" is not declared`))
		})

		t.Run("undeclared notation in attr enum rejected", func(t *testing.T) {
			t.Parallel()
			errs, err := parseValidatingDTD(t, `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
  <!ATTLIST root type NOTATION (fruit|vegetable) #REQUIRED>
  <!NOTATION fruit SYSTEM "f">
]>
<root type="fruit"/>`)
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
			require.True(t, containsError(errs, `enumerates undeclared notation "vegetable"`))
		})

		t.Run("declared NDATA notation accepted", func(t *testing.T) {
			t.Parallel()
			_, err := parseValidatingDTD(t, `<?xml version="1.0"?>
<!DOCTYPE paper [
  <!ELEMENT paper EMPTY>
  <!NOTATION gif SYSTEM "gif">
  <!ENTITY pic SYSTEM "pic.gif" NDATA gif>
]>
<paper/>`)
			require.NoError(t, err)
		})
	})
}
