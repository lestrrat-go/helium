package helium_test

import (
	"testing"
	"testing/fstest"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// condSectExtName is the external-subset filename shared by the conditional-
// section tests.
const condSectExtName = "cond.dtd"

func condSectDoc() string {
	return `<?xml version="1.0"?>` + "\n" +
		`<!DOCTYPE doc SYSTEM "` + condSectExtName + `">` + "\n" +
		`<doc>&greeting;</doc>`
}

func condSectParse(t *testing.T, dtd string) (*helium.Document, error) {
	t.Helper()
	fsys := fstest.MapFS{condSectExtName: &fstest.MapFile{Data: []byte(dtd)}}
	return helium.NewParser().
		BlockXXE(false).
		LoadExternalDTD(true).
		SubstituteEntities(true).
		FS(fsys).
		Parse(t.Context(), []byte(condSectDoc()))
}

// A conditional section keyword is case-sensitive (XML §3.4 P62/P63): only the
// exact literals INCLUDE and IGNORE are permitted. A miscased keyword such as
// lowercase "include" is a fatal well-formedness error and must be reported even
// from the top level of the external subset, where truncation is tolerated.
func TestConditionalSectionLowercaseIncludeRejected(t *testing.T) {
	t.Parallel()

	const dtd = "<![ include [\n<!ELEMENT doc (#PCDATA)>\n]]>\n"
	_, err := condSectParse(t, dtd)
	require.Error(t, err, "lowercase 'include' keyword must be a fatal error")
	require.Contains(t, err.Error(), "INCLUDE or IGNORE keyword")
}

// Lowercase "ignore" is equally a fatal keyword error.
func TestConditionalSectionLowercaseIgnoreRejected(t *testing.T) {
	t.Parallel()

	const dtd = "<![ ignore [\n]]>\n<!ELEMENT doc (#PCDATA)>\n"
	_, err := condSectParse(t, dtd)
	require.Error(t, err, "lowercase 'ignore' keyword must be a fatal error")
	require.Contains(t, err.Error(), "INCLUDE or IGNORE keyword")
}

// A misspelled / non-keyword token where INCLUDE|IGNORE is required is fatal.
func TestConditionalSectionBogusKeywordRejected(t *testing.T) {
	t.Parallel()

	const dtd = "<![ CDATA [\n<!ELEMENT doc (#PCDATA)>\n]]>\n"
	_, err := condSectParse(t, dtd)
	require.Error(t, err, "a non-INCLUDE/IGNORE keyword must be a fatal error")
	require.Contains(t, err.Error(), "INCLUDE or IGNORE keyword")
}

// A missing '[' after a valid INCLUDE keyword is malformed and fatal
// (P62: '<![' S? 'INCLUDE' S? '[').
func TestConditionalSectionMissingBracketRejected(t *testing.T) {
	t.Parallel()

	const dtd = "<![INCLUDE\n<!ELEMENT doc (#PCDATA)>\n]]>\n"
	_, err := condSectParse(t, dtd)
	require.Error(t, err, "missing '[' after INCLUDE must be a fatal error")
	require.Contains(t, err.Error(), "INCLUDE or IGNORE keyword")
}

// A correctly-cased INCLUDE section parses cleanly and its declarations take
// effect (the general entity declared inside it resolves in the document).
func TestConditionalSectionIncludeAccepted(t *testing.T) {
	t.Parallel()

	const dtd = "<![INCLUDE[\n<!ELEMENT doc (#PCDATA)>\n<!ENTITY greeting \"hi from include\">\n]]>\n"
	doc, err := condSectParse(t, dtd)
	require.NoError(t, err, "a well-formed INCLUDE section must parse")
	require.NotNil(t, doc)
	require.Equal(t, "hi from include", string(doc.DocumentElement().Content()))
}

// A correctly-cased IGNORE section parses cleanly; its body is discarded, so a
// declaration OUTSIDE the section is the one that takes effect.
func TestConditionalSectionIgnoreAccepted(t *testing.T) {
	t.Parallel()

	const dtd = "<![IGNORE[\n<!ENTITY greeting \"ignored\">\n]]>\n" +
		"<!ELEMENT doc (#PCDATA)>\n<!ENTITY greeting \"kept\">\n"
	doc, err := condSectParse(t, dtd)
	require.NoError(t, err, "a well-formed IGNORE section must parse")
	require.NotNil(t, doc)
	require.Equal(t, "kept", string(doc.DocumentElement().Content()))
}

// The INCLUDE|IGNORE keyword may be supplied by a parameter entity. The keyword
// is validated AFTER PE expansion, so a PE resolving to INCLUDE keeps the
// section well-formed and must NOT be rejected.
func TestConditionalSectionPESuppliedIncludeAccepted(t *testing.T) {
	t.Parallel()

	const dtd = "<!ENTITY % inc \"INCLUDE\">\n<![ %inc; [\n" +
		"<!ELEMENT doc (#PCDATA)>\n<!ENTITY greeting \"pe include\">\n]]>\n"
	doc, err := condSectParse(t, dtd)
	require.NoError(t, err, "a PE-supplied INCLUDE keyword must be accepted")
	require.NotNil(t, doc)
	require.Equal(t, "pe include", string(doc.DocumentElement().Content()))
}

// A parameter entity supplying INCLUDE[ (keyword plus opening bracket) in one
// expansion is also well-formed and must be accepted.
func TestConditionalSectionPESuppliedIncludeBracketAccepted(t *testing.T) {
	t.Parallel()

	const dtd = "<!ENTITY % inc \"INCLUDE[\">\n<![ %inc;\n" +
		"<!ELEMENT doc (#PCDATA)>\n<!ENTITY greeting \"pe inc bracket\">\n]]>\n"
	doc, err := condSectParse(t, dtd)
	require.NoError(t, err, "a PE-supplied 'INCLUDE[' must be accepted")
	require.NotNil(t, doc)
	require.Equal(t, "pe inc bracket", string(doc.DocumentElement().Content()))
}
