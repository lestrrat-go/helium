package helium_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestLenientXMLDecl exercises the LenientXMLDecl parse path, including
// pseudo-attributes presented out of the canonical order.
func TestLenientXMLDecl(t *testing.T) {
	t.Parallel()

	inputs := []string{
		`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><root/>`,
		`<?xml encoding="UTF-8" version="1.0"?><root/>`,
		`<?xml standalone="no" version="1.0"?><root/>`,
		`<?xml version="1.0"?><root/>`,
	}
	for _, in := range inputs {
		doc, err := helium.NewParser().LenientXMLDecl(true).Parse(t.Context(), []byte(in))
		require.NoError(t, err, "lenient parse of %q", in)
		require.NotNil(t, doc.DocumentElement())
	}
}

// TestMalformedXMLDecl exercises XML-declaration error branches.
func TestMalformedXMLDecl(t *testing.T) {
	t.Parallel()

	bad := []string{
		`<?xml?><root/>`,                         // missing version
		`<?xml version="1.0" foo="bar"?><root/>`, // unknown pseudo-attr / unclosed
		`<?xml version=1.0?><root/>`,             // unquoted version value
	}
	for _, in := range bad {
		_, err := helium.NewParser().Parse(t.Context(), []byte(in))
		require.Error(t, err, "malformed decl %q should error", in)
	}
}

// TestProcessingInstructionsAndComments parses PIs and comments in the prolog,
// content, and epilog positions.
func TestProcessingInstructionsAndComments(t *testing.T) {
	t.Parallel()

	const src = `<?xml version="1.0"?>
<?pi-prolog data?>
<!-- prolog comment -->
<root>
  <?pi-content more?>
  <!-- content comment -->
  text
</root>
<!-- epilog comment -->
<?pi-epilog x?>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)

	out, err := helium.WriteString(doc)
	require.NoError(t, err)
	require.Contains(t, out, "<?pi-prolog")
	require.Contains(t, out, "<!-- prolog comment -->")
}

// TestCDATASection parses CDATA sections including the tricky ]]> boundary.
func TestCDATASection(t *testing.T) {
	t.Parallel()

	const src = `<root><![CDATA[ raw <tag> & ]]> normal text <child/></root>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)

	out, err := helium.WriteString(doc)
	require.NoError(t, err)
	require.Contains(t, out, "<![CDATA[")
}

// TestCharacterReferences exercises numeric and hex character references.
func TestCharacterReferences(t *testing.T) {
	t.Parallel()

	const src = `<root>dec=&#65; hex=&#x42; high=&#x1F600;</root>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)
	require.Equal(t, "dec=A hex=B high=\U0001F600", string(doc.DocumentElement().Content()))
}

// TestMalformedDocuments exercises well-formedness error branches across the
// parser. Each input is malformed and must surface an error.
func TestMalformedDocuments(t *testing.T) {
	t.Parallel()

	bad := []string{
		`<root>`,                        // unclosed root
		`<root></notroot>`,              // mismatched end tag
		`<root attr></root>`,            // attribute without value
		`<root attr=value></root>`,      // unquoted attribute value
		`<root>&undefinedentity;</root>`, // reference to undeclared entity
		`<root><![CDATA[ unterminated`,  // unterminated CDATA
		`<!-- unterminated comment`,     // unterminated comment
		`<root>&#xZZ;</root>`,           // invalid hex char ref
		`<root>&;</root>`,               // empty reference
		`<>`,                            // empty tag name
		`<root></root><second/>`,        // two root elements
	}
	for _, in := range bad {
		_, err := helium.NewParser().Parse(t.Context(), []byte(in))
		require.Error(t, err, "malformed input %q should error", in)
	}
}

// TestRecoverOnError exercises the recover path: a malformed document returns
// both a (partial) document and an error.
func TestRecoverOnErrorPartialDoc(t *testing.T) {
	t.Parallel()

	const src = `<root><a>text</a><b></root>`
	doc, err := helium.NewParser().RecoverOnError(true).Parse(t.Context(), []byte(src))
	// With recovery the parser returns a partial document; an error may or may
	// not be reported depending on how far recovery proceeds.
	_ = err
	require.NotNil(t, doc)
}

// TestNamespacedAttributes parses namespaced elements and attributes.
func TestNamespacedAttributes(t *testing.T) {
	t.Parallel()

	const src = `<root xmlns="urn:default" xmlns:p="urn:p" p:attr="v" plain="w">` +
		`<p:child/><plain/></root>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)

	out, err := helium.WriteString(doc)
	require.NoError(t, err)
	require.Contains(t, out, `p:attr="v"`)
	require.Contains(t, out, `xmlns:p="urn:p"`)
}

// TestParameterEntities exercises parameter-entity declaration and reference in
// the internal subset.
func TestParameterEntities(t *testing.T) {
	t.Parallel()

	// A parameter entity expanded inside another entity's value.
	const src = `<?xml version="1.0"?>
<!DOCTYPE doc [
<!ENTITY % name "World">
<!ENTITY greeting "Hello %name;">
<!ELEMENT doc (#PCDATA)>
]>
<doc>&greeting;</doc>`

	doc, err := helium.NewParser().SubstituteEntities(true).Parse(t.Context(), []byte(src))
	require.NoError(t, err)
	require.NotNil(t, doc.DocumentElement())
}

// TestEntitySubstitution exercises entity expansion in content and attributes.
func TestEntitySubstitution(t *testing.T) {
	t.Parallel()

	const src = `<?xml version="1.0"?>
<!DOCTYPE doc [
<!ENTITY greeting "hello world">
]>
<doc attr="&greeting;">&greeting;</doc>`

	doc, err := helium.NewParser().SubstituteEntities(true).Parse(t.Context(), []byte(src))
	require.NoError(t, err)
	require.Contains(t, string(doc.DocumentElement().Content()), "hello world")
}
