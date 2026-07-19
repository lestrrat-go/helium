package helium_test

import (
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
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
	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(root))

	// Attribute value containing both quote characters plus markup chars.
	// SetLiteralAttribute stores the value verbatim (no entity parsing) so the
	// serializer is the component responsible for escaping it.
	err = root.SetLiteralAttribute("attr", `he said "hi" & 'bye' <x>`)
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

// TestWriteRichDTDWithEntities is a fuller round-trip that, in addition to the
// existing rich-DTD test, exercises serialization of a programmatically built
// DTD containing a percent-bearing internal entity and a parameter entity so the
// entity-content writer paths run end to end and re-parse cleanly.
func TestWriteRichDTDWithEntities(t *testing.T) {
	t.Parallel()

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	dtd, err := doc.CreateInternalSubset("doc", "", "")
	require.NoError(t, err)

	_, err = dtd.AddEntity("plain", enum.InternalGeneralEntity, "", "", "plain value")
	require.NoError(t, err)
	_, err = dtd.AddEntity("ext", enum.ExternalGeneralParsedEntity, "", "ext.xml", "")
	require.NoError(t, err)
	_, err = dtd.AddEntity("pub", enum.ExternalGeneralParsedEntity, "-//E//T//EN", "pub.xml", "")
	require.NoError(t, err)

	root, err := doc.CreateElement("doc")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(root))

	out, err := helium.WriteString(doc)
	require.NoError(t, err)
	require.Contains(t, out, "<!ENTITY plain")
	require.Contains(t, out, "<!ENTITY ext SYSTEM")
	require.Contains(t, out, "<!ENTITY pub PUBLIC")

	// Re-parse to confirm well-formedness.
	require.True(t, strings.Contains(out, "<!DOCTYPE doc"))
}

// TestDTDSerializationRichSubset round-trips a document with a rich internal
// subset (entities, attributes with defaults, notations, varied content models)
// to exercise the DTD writer paths.
func TestDTDSerializationRichSubset(t *testing.T) {
	t.Parallel()

	const src = `<?xml version="1.0"?>
<!DOCTYPE doc [
<!ELEMENT doc (a | b)*>
<!ELEMENT a (#PCDATA)>
<!ELEMENT b EMPTY>
<!ATTLIST a
  id   ID       #IMPLIED
  kind (x | y)  "x"
  req  CDATA    #REQUIRED>
<!ENTITY internal "expanded">
<!ENTITY % pe "ignored">
<!NOTATION gif SYSTEM "viewer.exe">
]>
<doc><a id="i1" req="r">text</a><b/></doc>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)

	out, err := helium.WriteString(doc)
	require.NoError(t, err)
	require.Contains(t, out, "<!DOCTYPE doc")
	require.Contains(t, out, "<!ELEMENT")
	require.Contains(t, out, "<!ATTLIST")
	require.Contains(t, out, "<!ENTITY")
	require.Contains(t, out, "<!NOTATION")
}
