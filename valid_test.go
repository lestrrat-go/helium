package helium_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

func containsError(errs []error, substr string) bool {
	for _, e := range errs {
		if strings.Contains(e.Error(), substr) {
			return true
		}
	}
	return false
}

// collectingErrorHandler records every validation error delivered during a
// validating parse so tests can assert on the failure surface.
type collectingErrorHandler struct {
	errs []error
}

func (h *collectingErrorHandler) Handle(_ context.Context, err error) {
	h.errs = append(h.errs, err)
}

// parseValidating parses src with DTD validation enabled, routing validation
// errors into a collector.
func parseValidating(t *testing.T, src string) []error {
	t.Helper()
	h := &collectingErrorHandler{}
	_, err := helium.NewParser().
		ValidateDTD(true).
		ErrorHandler(h).
		Parse(t.Context(), []byte(src))
	// A validation failure returns ErrDTDValidationFailed; the document is still
	// returned. Parser-level (well-formedness) errors are a different matter and
	// are not expected for these well-formed inputs.
	if err != nil {
		require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
	}
	return h.errs
}

func TestValidateDocument(t *testing.T) {
	t.Parallel()

	// requesting DTD validation on a document with
	// neither an internal nor an external subset is a validity error (libxml2
	// XML_DTD_NO_DTD "no DTD found!"), while the same document parsed without
	// ValidateDTD succeeds and a document carrying a DTD still validates.
	t.Run("no DTD", func(t *testing.T) {
		const noDTD = `<?xml version="1.0"?>
<root><child/></root>`

		t.Run("ValidateDTD(true) with no DTD is invalid", func(t *testing.T) {
			h := &collectingErrorHandler{}
			_, err := helium.NewParser().
				ValidateDTD(true).
				ErrorHandler(h).
				Parse(t.Context(), []byte(noDTD))
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
			require.True(t, containsError(h.errs, "no DTD found"),
				"expected a 'no DTD found' validity error, got %v", h.errs)
		})

		// The returned error must name the reason on its own. A caller that sets no
		// ErrorHandler (the default NilErrorHandler discards diagnostics) otherwise
		// sees only the bare "dtd: validation failed" sentinel and cannot tell a
		// document that violates its DTD from one that has no DTD to violate.
		t.Run("ValidateDTD(true) with no DTD reports the reason without a handler", func(t *testing.T) {
			_, err := helium.NewParser().
				ValidateDTD(true).
				Parse(t.Context(), []byte(noDTD))
			require.ErrorIs(t, err, helium.ErrNoDTDFound)
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
			require.Contains(t, err.Error(), "no DTD found")
		})

		// A document that HAS a DTD but violates it must stay distinguishable from
		// the no-DTD case, so a caller can branch on ErrNoDTDFound.
		t.Run("a document violating its DTD is not ErrNoDTDFound", func(t *testing.T) {
			const violatesDTD = `<?xml version="1.0"?>
<!DOCTYPE root [
<!ELEMENT root (child)>
<!ELEMENT child EMPTY>
]>
<root><wrong/></root>`
			_, err := helium.NewParser().
				ValidateDTD(true).
				Parse(t.Context(), []byte(violatesDTD))
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
			require.NotErrorIs(t, err, helium.ErrNoDTDFound)
		})

		t.Run("ValidateDTD(false) with no DTD succeeds", func(t *testing.T) {
			_, err := helium.NewParser().
				ValidateDTD(false).
				Parse(t.Context(), []byte(noDTD))
			require.NoError(t, err)
		})

		t.Run("ValidateDTD(true) with a DTD still validates", func(t *testing.T) {
			const withDTD = `<?xml version="1.0"?>
<!DOCTYPE root [
<!ELEMENT root (child)>
<!ELEMENT child EMPTY>
]>
<root><child/></root>`
			require.Empty(t, parseValidating(t, withDTD))
		})
	})

	// the root-name-vs-DTD-name check.
	t.Run("root name mismatch", func(t *testing.T) {
		const src = `<!DOCTYPE wrong [
<!ELEMENT root EMPTY>
]>
<root/>`

		errs := parseValidating(t, src)
		require.NotEmpty(t, errs, "root element not matching the DTD name is a validation error")
	})

	// the "no declaration found" branch.
	t.Run("undeclared element", func(t *testing.T) {
		const src = `<!DOCTYPE doc [
<!ELEMENT doc (a)>
<!ELEMENT a (#PCDATA)>
]>
<doc><a/><undeclared/></doc>`

		errs := parseValidating(t, src)
		require.NotEmpty(t, errs, "an undeclared element is a validation error")
	})

	// a DTD-validation diagnostic reports its
	// true severity so a level-filtered ErrorCollector receives it, and that the
	// structured DTDValidationError is recoverable via errors.As.
	t.Run("diagnostic error level", func(t *testing.T) {
		const src = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc EMPTY>
  <!ATTLIST doc id ID #REQUIRED>
]>
<doc/>`

		t.Run("collected at error level", func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			ec := helium.NewErrorCollector(ctx, helium.ErrorLevelError)
			_, err := helium.NewParser().
				ValidateDTD(true).
				ErrorHandler(ec).
				Parse(ctx, []byte(src))
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)

			errs := ec.Errors()
			require.Len(t, errs, 1, "the error-level collector must receive the DTD diagnostic")

			var dtdErr *helium.DTDValidationError
			require.ErrorAs(t, errs[0], &dtdErr)
			require.Equal(t, helium.ErrorLevelError, dtdErr.ErrorLevel())
			require.Equal(t, "element doc: attribute id is required", dtdErr.Message)
		})

		t.Run("filtered out of warning-level collector", func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			ec := helium.NewErrorCollector(ctx, helium.ErrorLevelWarning)
			_, err := helium.NewParser().
				ValidateDTD(true).
				ErrorHandler(ec).
				Parse(ctx, []byte(src))
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)

			require.Empty(t, ec.Errors(), "a genuine error must not be collected as a warning")
		})
	})
}

func TestValidateDTD(t *testing.T) {
	t.Parallel()

	t.Run("required attribute missing", func(t *testing.T) {
		t.Parallel()

		// #REQUIRED attribute missing -> validation error
		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc EMPTY>
  <!ATTLIST doc id ID #REQUIRED>
]>
<doc/>`

		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		p := helium.NewParser().ValidateDTD(true).ErrorHandler(collector)
		doc, err := p.Parse(t.Context(), []byte(input))

		require.Error(t, err, "missing #REQUIRED attribute should fail validation")
		require.NotNil(t, doc, "document should still be returned with validation error")
		require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
		require.True(t, containsError(collector.Errors(), "required"))
	})

	t.Run("required attribute present", func(t *testing.T) {
		t.Parallel()

		// #REQUIRED attribute present -> no error
		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc EMPTY>
  <!ATTLIST doc id ID #REQUIRED>
]>
<doc id="x1"/>`

		p := helium.NewParser().ValidateDTD(true)
		_, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err)
	})

	t.Run("fixed mismatch", func(t *testing.T) {
		t.Parallel()

		// #FIXED attribute with wrong value -> validation error
		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc EMPTY>
  <!ATTLIST doc version CDATA #FIXED "1.0">
]>
<doc version="2.0"/>`

		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		p := helium.NewParser().ValidateDTD(true).ErrorHandler(collector)
		_, err := p.Parse(t.Context(), []byte(input))

		require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
		require.True(t, containsError(collector.Errors(), "must be"))
	})

	t.Run("fixed correct", func(t *testing.T) {
		t.Parallel()

		// #FIXED attribute with correct value -> no error
		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc EMPTY>
  <!ATTLIST doc version CDATA #FIXED "1.0">
]>
<doc version="1.0"/>`

		p := helium.NewParser().ValidateDTD(true)
		_, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err)
	})

	t.Run("empty element with content", func(t *testing.T) {
		t.Parallel()

		// EMPTY element with content -> validation error
		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (child)>
  <!ELEMENT child EMPTY>
]>
<doc><child>text</child></doc>`

		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		p := helium.NewParser().ValidateDTD(true).ErrorHandler(collector)
		_, err := p.Parse(t.Context(), []byte(input))

		require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
		require.True(t, containsError(collector.Errors(), "EMPTY"))
	})

	t.Run("element content valid", func(t *testing.T) {
		t.Parallel()

		// Element content model (a, b) with correct content -> no error
		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (a, b)>
  <!ELEMENT a (#PCDATA)>
  <!ELEMENT b (#PCDATA)>
]>
<doc><a>hello</a><b>world</b></doc>`

		p := helium.NewParser().ValidateDTD(true)
		_, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err)
	})

	t.Run("element content mismatch", func(t *testing.T) {
		t.Parallel()

		// Element content model (a, b) with (b, a) -> validation error
		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (a, b)>
  <!ELEMENT a (#PCDATA)>
  <!ELEMENT b (#PCDATA)>
]>
<doc><b>world</b><a>hello</a></doc>`

		p := helium.NewParser().ValidateDTD(true)
		_, err := p.Parse(t.Context(), []byte(input))
		require.Error(t, err, "wrong element order should fail content model")
	})

	t.Run("mixed content valid", func(t *testing.T) {
		t.Parallel()

		// Mixed content (#PCDATA | a)* -- text and <a> are allowed
		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (#PCDATA | a)*>
  <!ELEMENT a (#PCDATA)>
]>
<doc>hello <a>world</a> end</doc>`

		p := helium.NewParser().ValidateDTD(true)
		_, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err)
	})

	t.Run("mixed content bad child", func(t *testing.T) {
		t.Parallel()

		// Mixed content (#PCDATA | a)* -- <b> is NOT allowed
		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (#PCDATA | a)*>
  <!ELEMENT a (#PCDATA)>
  <!ELEMENT b (#PCDATA)>
]>
<doc>hello <b>world</b></doc>`

		p := helium.NewParser().ValidateDTD(true)
		_, err := p.Parse(t.Context(), []byte(input))
		require.Error(t, err, "<b> not allowed in mixed content (a)")
	})

	t.Run("no flag skips validation", func(t *testing.T) {
		t.Parallel()

		// Same invalid document but WITHOUT ValidateDTD -> should pass
		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc EMPTY>
  <!ATTLIST doc id ID #REQUIRED>
]>
<doc/>`

		p := helium.NewParser()
		// Don't set ValidateDTD
		_, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err, "without ValidateDTD, validation should not run")
	})

	t.Run("choice content", func(t *testing.T) {
		t.Parallel()

		// Choice content model (a | b) with <a> -> valid
		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (a | b)>
  <!ELEMENT a (#PCDATA)>
  <!ELEMENT b (#PCDATA)>
]>
<doc><a>hello</a></doc>`

		p := helium.NewParser().ValidateDTD(true)
		_, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err)
	})

	t.Run("repeat content", func(t *testing.T) {
		t.Parallel()

		// Repetition content model (a)+ with multiple <a> -> valid
		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (a)+>
  <!ELEMENT a (#PCDATA)>
]>
<doc><a>1</a><a>2</a><a>3</a></doc>`

		p := helium.NewParser().ValidateDTD(true)
		_, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err)
	})

	t.Run("repeat content empty", func(t *testing.T) {
		t.Parallel()

		// Repetition content model (a)+ with zero <a> -> invalid
		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (a)+>
  <!ELEMENT a (#PCDATA)>
]>
<doc></doc>`

		p := helium.NewParser().ValidateDTD(true)
		_, err := p.Parse(t.Context(), []byte(input))
		require.Error(t, err, "(a)+ requires at least one <a>")
	})

	t.Run("ID unique", func(t *testing.T) {
		t.Parallel()

		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (item, item)>
  <!ELEMENT item EMPTY>
  <!ATTLIST item id ID #REQUIRED>
]>
<doc><item id="a"/><item id="b"/></doc>`

		p := helium.NewParser().ValidateDTD(true)
		_, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err)
	})

	t.Run("ID duplicate", func(t *testing.T) {
		t.Parallel()

		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (item, item)>
  <!ELEMENT item EMPTY>
  <!ATTLIST item id ID #REQUIRED>
]>
<doc><item id="a"/><item id="a"/></doc>`

		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		p := helium.NewParser().ValidateDTD(true).ErrorHandler(collector)
		_, err := p.Parse(t.Context(), []byte(input))

		require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
		require.True(t, containsError(collector.Errors(), "duplicate ID"))
	})

	t.Run("IDRef valid", func(t *testing.T) {
		t.Parallel()

		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (item, ref)>
  <!ELEMENT item EMPTY>
  <!ELEMENT ref EMPTY>
  <!ATTLIST item id ID #REQUIRED>
  <!ATTLIST ref target IDREF #REQUIRED>
]>
<doc><item id="x"/><ref target="x"/></doc>`

		p := helium.NewParser().ValidateDTD(true)
		_, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err)
	})

	t.Run("IDRef missing", func(t *testing.T) {
		t.Parallel()

		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (item, ref)>
  <!ELEMENT item EMPTY>
  <!ELEMENT ref EMPTY>
  <!ATTLIST item id ID #REQUIRED>
  <!ATTLIST ref target IDREF #REQUIRED>
]>
<doc><item id="x"/><ref target="y"/></doc>`

		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		p := helium.NewParser().ValidateDTD(true).ErrorHandler(collector)
		_, err := p.Parse(t.Context(), []byte(input))

		require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
		require.True(t, containsError(collector.Errors(), "unknown ID"))
	})

	t.Run("IDRefs valid", func(t *testing.T) {
		t.Parallel()

		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (item, item, refs)>
  <!ELEMENT item EMPTY>
  <!ELEMENT refs EMPTY>
  <!ATTLIST item id ID #REQUIRED>
  <!ATTLIST refs targets IDREFS #REQUIRED>
]>
<doc><item id="a"/><item id="b"/><refs targets="a b"/></doc>`

		p := helium.NewParser().ValidateDTD(true)
		_, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err)
	})

	t.Run("IDRefs missing", func(t *testing.T) {
		t.Parallel()

		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (item, refs)>
  <!ELEMENT item EMPTY>
  <!ELEMENT refs EMPTY>
  <!ATTLIST item id ID #REQUIRED>
  <!ATTLIST refs targets IDREFS #REQUIRED>
]>
<doc><item id="a"/><refs targets="a z"/></doc>`

		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		p := helium.NewParser().ValidateDTD(true).ErrorHandler(collector)
		_, err := p.Parse(t.Context(), []byte(input))

		require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
		require.True(t, containsError(collector.Errors(), "unknown ID"))
	})
}

func TestExtSubsetLookup(t *testing.T) {
	t.Parallel()

	t.Run("element in the external subset", func(t *testing.T) {
		dir := t.TempDir()
		dtdPath := filepath.Join(dir, "ext.dtd")
		require.NoError(t, os.WriteFile(dtdPath, []byte(`<!ELEMENT root (child)>
<!ELEMENT child EMPTY>
<!ATTLIST child role CDATA #REQUIRED>`), 0600))

		xml := `<?xml version="1.0"?>
<!DOCTYPE root SYSTEM "` + dtdPath + `">
<root><child role="main"/></root>`

		p := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).ValidateDTD(true).FS(helium.PermissiveFS())
		_, err := p.Parse(t.Context(), []byte(xml))
		require.NoError(t, err, "validation should pass when declarations are in extSubset")
	})

	t.Run("entity in the external subset", func(t *testing.T) {
		dir := t.TempDir()
		dtdPath := filepath.Join(dir, "ext.dtd")
		require.NoError(t, os.WriteFile(dtdPath, []byte(`<!ELEMENT root (#PCDATA)>
<!ENTITY extEnt "hello">`), 0600))

		xml := `<?xml version="1.0"?>
<!DOCTYPE root SYSTEM "` + dtdPath + `">
<root/>`

		p := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).FS(helium.PermissiveFS())
		doc, err := p.Parse(t.Context(), []byte(xml))
		require.NoError(t, err)

		ent, found := doc.GetEntity("extEnt")
		require.True(t, found, "entity in extSubset should be found")
		require.Equal(t, "hello", string(ent.Content()))
	})

	t.Run("attribute in the external subset", func(t *testing.T) {
		dir := t.TempDir()
		dtdPath := filepath.Join(dir, "ext.dtd")
		require.NoError(t, os.WriteFile(dtdPath, []byte(`<!ELEMENT root (child)>
<!ELEMENT child EMPTY>
<!ATTLIST child role CDATA #REQUIRED>`), 0600))

		xml := `<?xml version="1.0"?>
<!DOCTYPE root SYSTEM "` + dtdPath + `">
<root><child/></root>`

		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		p := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).ValidateDTD(true).ErrorHandler(collector).FS(helium.PermissiveFS())
		_, err := p.Parse(t.Context(), []byte(xml))

		require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
		require.True(t, containsError(collector.Errors(), "attribute role is required"))
	})

	t.Run("standalone yes prevents external-subset lookup", func(t *testing.T) {
		dir := t.TempDir()
		dtdPath := filepath.Join(dir, "ext.dtd")
		require.NoError(t, os.WriteFile(dtdPath, []byte(`<!ELEMENT root (child)>
<!ELEMENT child EMPTY>
<!ENTITY extEnt "hello">`), 0600))

		xml := `<?xml version="1.0" standalone="yes"?>
<!DOCTYPE root SYSTEM "` + dtdPath + `">
<root><child/></root>`

		p := helium.NewParser().LoadExternalDTD(true)
		doc, err := p.Parse(t.Context(), []byte(xml))
		require.NoError(t, err)

		_, found := doc.GetEntity("extEnt")
		require.False(t, found, "standalone=yes should prevent extSubset entity lookup")
	})
}
