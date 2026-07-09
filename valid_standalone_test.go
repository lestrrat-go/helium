package helium_test

import (
	"testing"
	"testing/fstest"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// parseStandalone parses src as a validating processor with an external subset
// supplied via extDTD (loaded for the SYSTEM id "ext.dtd"), returning the
// collected validity errors and the parse/validation error. Default attributes
// are materialized.
func parseStandalone(t *testing.T, src, extDTD string) ([]error, error) {
	t.Helper()
	return parseStandaloneOpt(t, src, extDTD, true)
}

// parseStandaloneOpt is parseStandalone with control over whether DTD default
// attributes are materialized (DefaultDTDAttributes).
func parseStandaloneOpt(t *testing.T, src, extDTD string, defaultAttrs bool) ([]error, error) {
	t.Helper()
	const extDTDName = "ext.dtd"
	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	fsys := fstest.MapFS{extDTDName: {Data: []byte(extDTD)}}
	_, err := helium.NewParser().
		BaseURI("doc.xml").
		BlockXXE(false).
		LoadExternalDTD(true).
		DefaultDTDAttributes(defaultAttrs).
		SubstituteEntities(true).
		ValidateDTD(true).
		ErrorHandler(collector).
		FS(fsys).
		Parse(t.Context(), []byte(src))
	return collector.Errors(), err
}

// TestStandaloneExternalDecl exercises the VC: Standalone Document Declaration
// (XML §2.9) for external-subset markup effects: a standalone="yes" document may
// not rely on external default attributes or on attribute-value normalization
// driven by an external tokenized-type declaration. Each rule has a rejecting
// case plus near-misses (standalone="no"/absent and internal-only declarations)
// to prove there is no over-rejection.
func TestStandaloneExternalDecl(t *testing.T) {
	t.Parallel()

	// Rule: attribute defaulted from the external subset.
	t.Run("external default attribute", func(t *testing.T) {
		t.Parallel()

		extDTD := `<!ATTLIST animal color CDATA #FIXED "yellow">`

		t.Run("standalone yes rejected", func(t *testing.T) {
			t.Parallel()
			errs, err := parseStandalone(t, `<?xml version="1.0" standalone="yes"?>
<!DOCTYPE animal SYSTEM "ext.dtd" [ <!ELEMENT animal EMPTY> ]>
<animal/>`, extDTD)
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
			require.True(t, containsError(errs, "defaulted from external subset"))
		})

		t.Run("standalone no accepted", func(t *testing.T) {
			t.Parallel()
			_, err := parseStandalone(t, `<?xml version="1.0" standalone="no"?>
<!DOCTYPE animal SYSTEM "ext.dtd" [ <!ELEMENT animal EMPTY> ]>
<animal/>`, extDTD)
			require.NoError(t, err)
		})

		// The same default declared in the INTERNAL subset does not violate the
		// VC: no external markup is required to reproduce the document.
		t.Run("internal default accepted", func(t *testing.T) {
			t.Parallel()
			_, err := parseStandalone(t, `<?xml version="1.0" standalone="yes"?>
<!DOCTYPE animal SYSTEM "ext.dtd" [
  <!ELEMENT animal EMPTY>
  <!ATTLIST animal legs CDATA #FIXED "4">
]>
<animal/>`, `<!ELEMENT ignored EMPTY>`)
			require.NoError(t, err)
		})

		// The check is driven by the ATTLIST declaration, not a materialized default
		// node: ValidateDTD(true) does not imply DefaultDTDAttributes(true), so an
		// external default that is never materialized must still be reported.
		t.Run("standalone yes rejected without default materialization", func(t *testing.T) {
			t.Parallel()
			errs, err := parseStandaloneOpt(t, `<?xml version="1.0" standalone="yes"?>
<!DOCTYPE animal SYSTEM "ext.dtd" [ <!ELEMENT animal EMPTY> ]>
<animal/>`, extDTD, false)
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
			require.True(t, containsError(errs, "defaulted from external subset"))
		})

		// A specified (non-defaulted) attribute is fine even when the same external
		// ATTLIST provides its default.
		t.Run("explicitly specified accepted", func(t *testing.T) {
			t.Parallel()
			_, err := parseStandalone(t, `<?xml version="1.0" standalone="yes"?>
<!DOCTYPE animal SYSTEM "ext.dtd" [ <!ELEMENT animal EMPTY> ]>
<animal color="yellow"/>`, `<!ATTLIST animal color CDATA "yellow">`)
			require.NoError(t, err)
		})
	})

	// Rule: attribute value normalized by an external tokenized-type declaration.
	t.Run("external attribute normalization", func(t *testing.T) {
		t.Parallel()

		extDTD := `<!ATTLIST animal class NMTOKEN #IMPLIED>`

		t.Run("standalone yes rejected", func(t *testing.T) {
			t.Parallel()
			errs, err := parseStandalone(t, `<?xml version="1.0" standalone="yes"?>
<!DOCTYPE animal SYSTEM "ext.dtd" [ <!ELEMENT animal EMPTY> ]>
<animal class="  spacedvalue  "/>`, extDTD)
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
			require.True(t, containsError(errs, "normalization of attribute class"))
		})

		t.Run("standalone no accepted", func(t *testing.T) {
			t.Parallel()
			_, err := parseStandalone(t, `<?xml version="1.0" standalone="no"?>
<!DOCTYPE animal SYSTEM "ext.dtd" [ <!ELEMENT animal EMPTY> ]>
<animal class="  spacedvalue  "/>`, extDTD)
			require.NoError(t, err)
		})

		// An already-normalized value is unchanged, so there is no violation even
		// though the tokenized type is declared externally.
		t.Run("already normalized accepted", func(t *testing.T) {
			t.Parallel()
			_, err := parseStandalone(t, `<?xml version="1.0" standalone="yes"?>
<!DOCTYPE animal SYSTEM "ext.dtd" [ <!ELEMENT animal EMPTY> ]>
<animal class="normalized"/>`, extDTD)
			require.NoError(t, err)
		})

		// The same tokenized type declared in the INTERNAL subset normalizes the
		// value too, but that requires no external markup, so it is not a violation.
		t.Run("internal normalization accepted", func(t *testing.T) {
			t.Parallel()
			_, err := parseStandalone(t, `<?xml version="1.0" standalone="yes"?>
<!DOCTYPE animal SYSTEM "ext.dtd" [
  <!ELEMENT animal EMPTY>
  <!ATTLIST animal class NMTOKEN #IMPLIED>
]>
<animal class="  spacedvalue  "/>`, `<!ELEMENT ignored EMPTY>`)
			require.NoError(t, err)
		})
	})

	// A declaration pulled in by an EXTERNAL PARAMETER ENTITY referenced from the
	// INTERNAL subset is external markup for the VC (libxml2 PARSER_EXTERNAL), even
	// though it is registered in the internal-subset declaration table.
	t.Run("external parameter entity", func(t *testing.T) {
		t.Parallel()

		// External-PE-supplied ATTLIST default.
		t.Run("default rejected", func(t *testing.T) {
			t.Parallel()
			errs, err := parseStandalone(t, `<?xml version="1.0" standalone="yes"?>
<!DOCTYPE animal [
  <!ENTITY % ext SYSTEM "ext.dtd">
  %ext;
  <!ELEMENT animal EMPTY>
]>
<animal/>`, `<!ATTLIST animal color CDATA #FIXED "yellow">`)
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
			require.True(t, containsError(errs, "defaulted from external subset"))
		})

		// External-PE-supplied tokenized type driving attribute normalization.
		t.Run("normalization rejected", func(t *testing.T) {
			t.Parallel()
			errs, err := parseStandalone(t, `<?xml version="1.0" standalone="yes"?>
<!DOCTYPE animal [
  <!ENTITY % ext SYSTEM "ext.dtd">
  %ext;
  <!ELEMENT animal EMPTY>
]>
<animal class="  spacedvalue  "/>`, `<!ATTLIST animal class NMTOKEN #IMPLIED>`)
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
			require.True(t, containsError(errs, "normalization of attribute class"))
		})

		// A purely-internal parameter entity supplies no external markup, so its
		// defaulted attribute and normalization are not standalone violations.
		t.Run("internal PE accepted", func(t *testing.T) {
			t.Parallel()
			_, err := parseStandalone(t, `<?xml version="1.0" standalone="yes"?>
<!DOCTYPE animal [
  <!ENTITY % int "<!ATTLIST animal color CDATA #FIXED 'yellow'><!ATTLIST animal class NMTOKEN #IMPLIED>">
  %int;
  <!ELEMENT animal EMPTY>
]>
<animal class="  spacedvalue  "/>`, `<!ELEMENT ignored EMPTY>`)
			require.NoError(t, err)
		})
	})
}
