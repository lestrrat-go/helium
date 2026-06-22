package helium_test

import (
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestSerializeRichDTD parses a DTD that exercises every attribute type, every
// default kind, internal/external/parameter/unparsed entities, PUBLIC and
// SYSTEM notations, and EMPTY/ANY/mixed/element content models, then serializes
// it so the DTD-writer paths are covered. The serialized form round-trips
// through a second parse.
func TestSerializeRichDTD(t *testing.T) {
	t.Parallel()

	const src = `<?xml version="1.0"?>
<!DOCTYPE doc [
<!ELEMENT doc (a, b?, (c | d)*, e+)>
<!ELEMENT a EMPTY>
<!ELEMENT b ANY>
<!ELEMENT c (#PCDATA)>
<!ELEMENT d (#PCDATA|c)*>
<!ELEMENT e (#PCDATA)>
<!ATTLIST doc
  id    ID       #IMPLIED
  ref   IDREF    #IMPLIED
  refs  IDREFS   #IMPLIED
  ent   ENTITY   #IMPLIED
  ents  ENTITIES #IMPLIED
  tok   NMTOKEN  #IMPLIED
  toks  NMTOKENS #IMPLIED
  req   CDATA    #REQUIRED
  fix   CDATA    #FIXED "fixed"
  kind  (x|y|z)  "x"
  note  NOTATION (gif|png) #IMPLIED>
<!ENTITY internal "internal value">
<!ENTITY ext SYSTEM "ext.xml">
<!ENTITY pub PUBLIC "-//Example//Text//EN" "pub.xml">
<!ENTITY img SYSTEM "img.gif" NDATA gif>
<!ENTITY % pe "param value">
<!NOTATION gif SYSTEM "viewer.exe">
<!NOTATION png PUBLIC "-//Example//Notation//EN" "png.exe">
]>
<doc id="d1" req="x"><a/><c>c</c><e>e</e></doc>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err, "parse rich DTD")

	out, err := helium.WriteString(doc)
	require.NoError(t, err, "serialize rich DTD")

	// Spot-check the declarations made it into the serialized DTD.
	for _, want := range []string{
		"<!DOCTYPE doc",
		"<!ELEMENT a EMPTY>",
		"<!ELEMENT b ANY>",
		"ID",
		"IDREF",
		"IDREFS",
		"ENTITY",
		"ENTITIES",
		"NMTOKEN",
		"#REQUIRED",
		"#FIXED",
		"NOTATION",
		"<!ENTITY internal",
		"<!ENTITY ext SYSTEM",
		"<!ENTITY pub PUBLIC",
		"NDATA gif",
		"<!ENTITY % pe",
		"<!NOTATION gif SYSTEM",
		"<!NOTATION png PUBLIC",
	} {
		require.Contains(t, out, want, "serialized DTD should contain %q", want)
	}

	// The serialized output must itself re-parse cleanly.
	_, err = helium.NewParser().Parse(t.Context(), []byte(out))
	require.NoError(t, err, "re-parse serialized rich DTD")
}

// TestSerializeEscaping exercises the text/attribute escaping paths, including
// values that contain both single and double quotes and assorted control and
// markup characters.
func TestSerializeEscaping(t *testing.T) {
	t.Parallel()

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root := doc.CreateElement("root")
	require.NoError(t, doc.AddChild(root))

	// Attribute value containing both quote characters plus markup chars.
	// SetLiteralAttribute stores the value verbatim (no entity parsing) so the
	// serializer is the component responsible for escaping it.
	err := root.SetLiteralAttribute("attr", `he said "hi" & 'bye' <x>`)
	require.NoError(t, err)

	// Text content with markup, ampersand, tab and newline.
	require.NoError(t, root.AppendText([]byte("a < b & c > d\twith\ntabs")))

	out, err := helium.WriteString(doc)
	require.NoError(t, err)
	require.Contains(t, out, "&amp;")
	require.Contains(t, out, "&lt;")
	require.Contains(t, out, "&gt;")
	require.Contains(t, out, "&quot;")

	// Re-parse to confirm well-formedness of the escaped output.
	_, err = helium.NewParser().Parse(t.Context(), []byte(out))
	require.NoError(t, err)
}

// TestSerializeFormatting exercises the Format/IndentString writer options.
func TestSerializeFormatting(t *testing.T) {
	t.Parallel()

	const src = `<root><a><b>text</b></a><c/></root>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)

	var buf strings.Builder
	err = helium.NewWriter().
		Format(true).
		IndentString("    ").
		XMLDeclaration(false).
		WriteTo(&buf, doc)
	require.NoError(t, err)
	require.Contains(t, buf.String(), "\n    ")
	require.NotContains(t, buf.String(), "<?xml")
}

// TestSerializeSelfCloseToggle exercises the SelfCloseEmptyElements option.
func TestSerializeSelfCloseToggle(t *testing.T) {
	t.Parallel()

	const src = `<root><empty/></root>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)

	var buf strings.Builder
	err = helium.NewWriter().
		SelfCloseEmptyElements(false).
		XMLDeclaration(false).
		WriteTo(&buf, doc)
	require.NoError(t, err)
	require.Contains(t, buf.String(), "<empty></empty>")
}
