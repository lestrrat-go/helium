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

		// An external UNPREFIXED tokenized declaration does not apply to a PREFIXED
		// instance attribute of the same local name, so the standalone
		// normalization check must NOT fire. The prefixed p:id is itself undeclared
		// (VC: Attribute Value Type — the id declaration is prefix-literal), so the
		// document is rejected for that reason, not for external normalization.
		t.Run("prefixed instance vs unprefixed external decl not normalized", func(t *testing.T) {
			t.Parallel()
			errs, err := parseStandalone(t, `<?xml version="1.0" standalone="yes"?>
<!DOCTYPE r SYSTEM "ext.dtd" [ <!ELEMENT r EMPTY> ]>
<r xmlns:p="urn:p" p:id="  spacedvalue  "/>`, `<!ATTLIST r id NMTOKEN #IMPLIED>`)
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
			require.True(t, containsError(errs, "no declaration for attribute p:id"))
			require.False(t, containsError(errs, "normalization"))
		})

		// The genuine case (unprefixed instance matching the unprefixed external
		// declaration) still fires.
		t.Run("unprefixed instance matching external decl rejected", func(t *testing.T) {
			t.Parallel()
			errs, err := parseStandalone(t, `<?xml version="1.0" standalone="yes"?>
<!DOCTYPE r SYSTEM "ext.dtd" [ <!ELEMENT r EMPTY> ]>
<r id="  spacedvalue  "/>`, `<!ATTLIST r id NMTOKEN #IMPLIED>`)
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
			require.True(t, containsError(errs, "normalization of attribute id"))
		})

		// A PREFIXED external declaration matched by a PREFIXED instance attribute of
		// the same QName normalizes and is reported under standalone="yes".
		t.Run("prefixed instance matching prefixed external decl rejected", func(t *testing.T) {
			t.Parallel()
			errs, err := parseStandalone(t, `<?xml version="1.0" standalone="yes"?>
<!DOCTYPE r SYSTEM "ext.dtd" [ <!ELEMENT r EMPTY> ]>
<r xmlns:p="urn:p" p:id="  spacedvalue  "/>`, `<!ATTLIST r p:id NMTOKEN #IMPLIED>`)
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
			require.True(t, containsError(errs, "normalization of attribute p:id"))
		})

		// The same prefixed match under standalone="no": the value is normalized (so
		// it is a valid NMTOKEN) and the document is accepted — no over-rejection.
		t.Run("prefixed instance matching prefixed external decl accepted when not standalone", func(t *testing.T) {
			t.Parallel()
			_, err := parseStandalone(t, `<?xml version="1.0" standalone="no"?>
<!DOCTYPE r SYSTEM "ext.dtd" [ <!ELEMENT r EMPTY> ]>
<r xmlns:p="urn:p" p:id="  spacedvalue  "/>`, `<!ATTLIST r p:id NMTOKEN #IMPLIED>`)
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

	// ATTLIST declarations are keyed by the declared element QName, so a declaration
	// for a PREFIXED element applies to that prefixed element and nothing else.
	t.Run("prefixed element", func(t *testing.T) {
		t.Parallel()

		extDTD := `<!ATTLIST p:r p:id NMTOKEN #IMPLIED>`

		// Prefixed element + prefixed attribute matching the prefixed external
		// declaration: the value is normalized and reported under standalone="yes".
		t.Run("prefixed element and attr rejected", func(t *testing.T) {
			t.Parallel()
			errs, err := parseStandalone(t, `<?xml version="1.0" standalone="yes"?>
<!DOCTYPE p:r SYSTEM "ext.dtd" [ <!ELEMENT p:r EMPTY> ]>
<p:r xmlns:p="urn:p" p:id="  spacedvalue  "/>`, extDTD)
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
			require.True(t, containsError(errs, "normalization of attribute p:id"))
		})

		// Same match under standalone="no": normalized to a valid NMTOKEN → accepted
		// (no over-rejection of the previously un-normalized value).
		t.Run("prefixed element and attr accepted when not standalone", func(t *testing.T) {
			t.Parallel()
			_, err := parseStandalone(t, `<?xml version="1.0" standalone="no"?>
<!DOCTYPE p:r SYSTEM "ext.dtd" [ <!ELEMENT p:r EMPTY> ]>
<p:r xmlns:p="urn:p" p:id="  spacedvalue  "/>`, extDTD)
			require.NoError(t, err)
		})

		// A declaration for the PREFIXED element `p:r` does not apply to an
		// unprefixed element `<r>`, so the external declaration never normalizes
		// `<r>`'s id. Because that declaration does not apply, id is undeclared on
		// `<r>` (VC: Attribute Value Type) — the document is rejected for the
		// undeclared attribute, not for external normalization.
		t.Run("prefixed decl vs unprefixed element not applied", func(t *testing.T) {
			t.Parallel()
			errs, err := parseStandalone(t, `<?xml version="1.0" standalone="yes"?>
<!DOCTYPE r SYSTEM "ext.dtd" [ <!ELEMENT r EMPTY> ]>
<r id="  spacedvalue  "/>`, extDTD)
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
			require.True(t, containsError(errs, "no declaration for attribute id"))
			require.False(t, containsError(errs, "normalization"))
		})

		// External default declared for a prefixed element applies to that element.
		t.Run("prefixed element external default rejected", func(t *testing.T) {
			t.Parallel()
			errs, err := parseStandalone(t, `<?xml version="1.0" standalone="yes"?>
<!DOCTYPE p:r SYSTEM "ext.dtd" [ <!ELEMENT p:r EMPTY> ]>
<p:r xmlns:p="urn:p"/>`, `<!ATTLIST p:r color CDATA #FIXED "yellow">`)
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
			require.True(t, containsError(errs, "defaulted from external subset"))
		})

		// The special-attribute key is a (element-QName, attribute-QName) struct, not
		// a `elem:attr` concatenation, so distinct QName pairs that would concatenate
		// identically do not cross-contaminate: a declaration for element `p:r`,
		// attribute `id` (concat "p:r:id") must NOT apply to element `p`, attribute
		// `r:id` (also concat "p:r:id"). If it did, `r:id` would be treated as an
		// external NMTOKEN, normalized, and wrongly reported. Because it does not
		// apply, `r:id` is undeclared on `<p>` (VC: Attribute Value Type): the absence
		// of a normalization report — with an undeclared-attribute rejection instead —
		// proves the keys are disambiguated.
		t.Run("ambiguous element/attr key does not cross-contaminate", func(t *testing.T) {
			t.Parallel()
			errs, err := parseStandalone(t, `<?xml version="1.0" standalone="yes"?>
<!DOCTYPE p SYSTEM "ext.dtd" [ <!ELEMENT p EMPTY> ]>
<p xmlns:r="urn:r" r:id="  spacedvalue  "/>`, `<!ATTLIST p:r id NMTOKEN #IMPLIED>`)
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
			require.True(t, containsError(errs, "no declaration for attribute r:id"))
			require.False(t, containsError(errs, "normalization"))
		})
	})
}

// TestDTDElementDeclQNameMatch verifies that element-declaration lookup during DTD
// validation is prefix-literal: a prefixed element requires an <!ELEMENT>
// declaration for the same QName, with no fallback to an unprefixed declaration.
func TestDTDElementDeclQNameMatch(t *testing.T) {
	t.Parallel()

	// A prefixed element with only an unprefixed declaration is undeclared.
	t.Run("prefixed element unprefixed decl rejected", func(t *testing.T) {
		t.Parallel()
		_, err := parseValidatingDTD(t, `<?xml version="1.0"?>
<!DOCTYPE p:r [ <!ELEMENT r EMPTY> ]>
<p:r xmlns:p="urn:p"/>`)
		require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
	})

	// A matching prefixed declaration validates.
	t.Run("prefixed element matching decl accepted", func(t *testing.T) {
		t.Parallel()
		_, err := parseValidatingDTD(t, `<?xml version="1.0"?>
<!DOCTYPE p:r [ <!ELEMENT p:r EMPTY> ]>
<p:r xmlns:p="urn:p"/>`)
		require.NoError(t, err)
	})
}

// TestStandaloneExternalElementContent verifies that DTD validation uses element
// declarations from the external subset even for a standalone="yes" document
// (docDTDs searches both subsets regardless of standalone), while the §2.9
// element-content-whitespace Standalone VC still rejects whitespace — text or
// CDATA — directly within an externally-declared element-content element.
func TestStandaloneExternalElementContent(t *testing.T) {
	t.Parallel()

	// root has element-only content declared ONLY in the external subset.
	extDTD := `<!ELEMENT root (child)*>
<!ELEMENT child EMPTY>`

	// standalone="yes" with the element declared externally and NO ignorable
	// whitespace: the external declaration is used for validation (before this
	// fix it was hidden, yielding a spurious "no declaration found").
	t.Run("external element decl found under standalone yes", func(t *testing.T) {
		t.Parallel()
		_, err := parseStandalone(t, `<?xml version="1.0" standalone="yes"?>
<!DOCTYPE root SYSTEM "ext.dtd">
<root><child/></root>`, extDTD)
		require.NoError(t, err)
	})

	// Whitespace text directly within the externally-declared element-content
	// element violates the standalone constraint.
	t.Run("whitespace text rejected", func(t *testing.T) {
		t.Parallel()
		errs, err := parseStandalone(t, `<?xml version="1.0" standalone="yes"?>
<!DOCTYPE root SYSTEM "ext.dtd">
<root>
  <child/>
</root>`, extDTD)
		require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
		require.True(t, containsError(errs, "white spaces nodes"))
	})

	// The same whitespace written as a CDATA section is equally a violation.
	t.Run("whitespace CDATA rejected", func(t *testing.T) {
		t.Parallel()
		errs, err := parseStandalone(t, `<?xml version="1.0" standalone="yes"?>
<!DOCTYPE root SYSTEM "ext.dtd">
<root><![CDATA[
  ]]><child/></root>`, extDTD)
		require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
		require.True(t, containsError(errs, "white spaces nodes"))
	})

	// Near-miss: standalone="no" with the same whitespace is accepted (the
	// document openly depends on the external subset).
	t.Run("whitespace under standalone no accepted", func(t *testing.T) {
		t.Parallel()
		_, err := parseStandalone(t, `<?xml version="1.0" standalone="no"?>
<!DOCTYPE root SYSTEM "ext.dtd">
<root>
  <child/>
</root>`, extDTD)
		require.NoError(t, err)
	})
}
