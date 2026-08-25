package helium_test

import (
	"regexp"
	"strings"
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
			// ANY (not EMPTY) content isolates this to the enumerated-value VC.
			errs, err := parseValidatingDTD(t, `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root ANY>
  <!ATTLIST root type NOTATION (fruit|vegetable) #REQUIRED>
  <!NOTATION fruit SYSTEM "f">
  <!NOTATION vegetable SYSTEM "v">
  <!NOTATION candy SYSTEM "c">
]>
<root type="candy"/>`)
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
			require.True(t, containsError(errs, "not among the enumerated notations"))
		})

		t.Run("notation attribute on EMPTY element rejected", func(t *testing.T) {
			t.Parallel()
			errs, err := parseValidatingDTD(t, `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
  <!ATTLIST root type NOTATION (fruit|vegetable) #IMPLIED>
  <!NOTATION fruit SYSTEM "f">
  <!NOTATION vegetable SYSTEM "v">
]>
<root/>`)
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
			require.True(t, containsError(errs, "not allowed on an EMPTY element"))
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

		t.Run("empty IDREF default rejected", func(t *testing.T) {
			t.Parallel()
			// A literal empty default must still satisfy the declared type; an empty
			// string is not a valid Name for an IDREF.
			errs, err := parseValidatingDTD(t, `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root ANY>
  <!ELEMENT unused EMPTY>
  <!ATTLIST unused ref IDREF "">
]>
<root/>`)
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
			require.True(t, containsError(errs, "is not valid"))
		})

		t.Run("empty NMTOKEN default rejected", func(t *testing.T) {
			t.Parallel()
			errs, err := parseValidatingDTD(t, `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root ANY>
  <!ELEMENT unused EMPTY>
  <!ATTLIST unused tok NMTOKEN "">
]>
<root/>`)
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
			require.True(t, containsError(errs, "is not valid"))
		})

		t.Run("valid NMTOKEN default accepted", func(t *testing.T) {
			t.Parallel()
			_, err := parseValidatingDTD(t, `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root ANY>
  <!ATTLIST root tok NMTOKEN "abc">
]>
<root/>`)
			require.NoError(t, err)
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
			// ANY (not EMPTY) content isolates this to the Notation Declared VC.
			errs, err := parseValidatingDTD(t, `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root ANY>
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

// enumDefaultAttrPattern captures the attribute name out of an Attribute
// Default Legal diagnostic.
var enumDefaultAttrPattern = regexp.MustCompile(`for attribute (\S+) is not among`)

// oneIDElemPattern captures the element name out of a One ID per Element Type
// diagnostic.
var oneIDElemPattern = regexp.MustCompile(`element (\S+) has more than one ID attribute`)

// matchedNames returns the first capture group of pat for every error in errs it
// matches, in emission order, joined with commas so an ordering mismatch reads
// directly in the failure output. Errors that do not match are ignored.
func matchedNames(errs []error, pat *regexp.Regexp) string {
	var names []string
	for _, e := range errs {
		m := pat.FindStringSubmatch(e.Error())
		if m == nil {
			continue
		}
		names = append(names, m[1])
	}
	return strings.Join(names, ",")
}

// The declaration-level validity checks (validateDTDDeclarations,
// validateOneIDPerElement) iterate the subset's registration-order sequence of
// attribute declarations, not the attributes map, so a document reports the same
// diagnostics in the same order on every run. Ranging a Go map instead would
// rotate the sequence randomly per run, making a golden or a diff over
// validation output useless.
//
// Each case repeats the parse many times: a random order agrees with declaration
// order once in n! draws, so a handful of iterations already makes a map-order
// implementation fail with near-certainty.
func TestDTDDeclDiagnosticOrder(t *testing.T) {
	t.Parallel()

	const repeats = 50

	// Eight enumerated attributes, each with a default outside its own token set,
	// so every declaration yields exactly one Attribute Default Legal diagnostic.
	t.Run("attribute declaration diagnostics", func(t *testing.T) {
		t.Parallel()

		const src = `<?xml version="1.0"?>
<!DOCTYPE root [
<!ELEMENT root EMPTY>
<!ATTLIST root a1 (p|q) "z">
<!ATTLIST root a2 (p|q) "z">
<!ATTLIST root a3 (p|q) "z">
<!ATTLIST root a4 (p|q) "z">
<!ATTLIST root a5 (p|q) "z">
<!ATTLIST root a6 (p|q) "z">
<!ATTLIST root a7 (p|q) "z">
<!ATTLIST root a8 (p|q) "z">
]>
<root/>`

		const want = "a1,a2,a3,a4,a5,a6,a7,a8"
		for i := range repeats {
			errs, err := parseValidatingDTD(t, src)
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
			require.Equal(t, want, matchedNames(errs, enumDefaultAttrPattern),
				"attribute declaration diagnostics must follow declaration order (iteration %d); errs=%v", i, errs)
		}
	})

	// Four element types, each over-declaring ID attributes, are reported in the
	// order of their first ID declaration.
	t.Run("one ID per element diagnostics", func(t *testing.T) {
		t.Parallel()

		const src = `<?xml version="1.0"?>
<!DOCTYPE root [
<!ELEMENT root (alpha,bravo,charlie,delta)>
<!ELEMENT alpha EMPTY>
<!ELEMENT bravo EMPTY>
<!ELEMENT charlie EMPTY>
<!ELEMENT delta EMPTY>
<!ATTLIST alpha i1 ID #IMPLIED i2 ID #IMPLIED>
<!ATTLIST bravo i1 ID #IMPLIED i2 ID #IMPLIED>
<!ATTLIST charlie i1 ID #IMPLIED i2 ID #IMPLIED>
<!ATTLIST delta i1 ID #IMPLIED i2 ID #IMPLIED>
]>
<root><alpha/><bravo/><charlie/><delta/></root>`

		const want = "alpha,bravo,charlie,delta"
		for i := range repeats {
			errs, err := parseValidatingDTD(t, src)
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
			require.Equal(t, want, matchedNames(errs, oneIDElemPattern),
				"one-ID-per-element diagnostics must follow declaration order (iteration %d); errs=%v", i, errs)
		}
	})
}
