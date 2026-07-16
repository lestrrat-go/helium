package helium_test

import (
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestSerializeXHTML parses an XHTML 1.0 document (recognized via its PUBLIC
// identifier) and serializes it, exercising the XHTML-specific writer paths:
// void elements, the html xmlns injection, head content-type meta injection, and
// boolean attribute handling.
func TestSerializeXHTML(t *testing.T) {
	t.Parallel()

	const src = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Strict//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>t</title></head>
<body>
<p>para<br/>after break</p>
<img src="x.png" alt="x"/>
<form action="/go"><input type="checkbox" checked="checked"/></form>
<hr/>
</body>
</html>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err, "parse XHTML document")

	out, err := helium.WriteString(doc)
	require.NoError(t, err, "serialize XHTML document")

	// Void elements use the " />" form in XHTML output.
	require.Contains(t, out, "<br />")
	require.Contains(t, out, "<hr />")
	require.True(t, strings.Contains(out, "<img"), "img element serialized")

	// Re-parse the serialized output to confirm well-formedness.
	_, err = helium.NewParser().Parse(t.Context(), []byte(out))
	require.NoError(t, err, "re-parse serialized XHTML")
}

// TestSerializeXHTMLFormatted serializes XHTML with formatting enabled to drive
// the indentation branches of the XHTML writer.
func TestSerializeXHTMLFormatted(t *testing.T) {
	t.Parallel()

	const src = `<?xml version="1.0"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>t</title></head>
<body><div><p>x</p></div></body>
</html>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)

	var buf strings.Builder
	err = helium.NewWriter().Format(true).WriteTo(&buf, doc)
	require.NoError(t, err)
	require.Contains(t, buf.String(), "<html")
}

// TestSerializeXHTMLMixedContent verifies the XHTML serializer honors the
// mixed-content rule: an element with any text-like child (mixed content) is
// serialized inline with no injected indentation whitespace, and the suppression
// propagates across the whole subtree — matching the regular writeNode path and
// libxml2's xhtmlNodeDumpOutput. A second parse+format pass must be byte-identical
// (idempotent); injected whitespace would corrupt the text content on each pass.
func TestSerializeXHTMLMixedContent(t *testing.T) {
	t.Parallel()

	const src = `<?xml version="1.0"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
<body><p>left<b><i/></b>right</p></body>
</html>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err, "parse XHTML document")

	var buf strings.Builder
	err = helium.NewWriter().Format(true).WriteTo(&buf, doc)
	require.NoError(t, err, "serialize XHTML with formatting")
	out := buf.String()

	// The mixed-content <p> (and the nested <b>) serialize inline: the text runs
	// abut their sibling elements with no injected newline/indent.
	require.Contains(t, out, "<p>left<b><i></i></b>right</p>", "output:\n%s", out)

	// A second parse+format pass must reproduce the first byte-for-byte.
	doc2, err := helium.NewParser().Parse(t.Context(), []byte(out))
	require.NoError(t, err, "re-parse serialized XHTML")
	var buf2 strings.Builder
	err = helium.NewWriter().Format(true).WriteTo(&buf2, doc2)
	require.NoError(t, err, "re-serialize XHTML with formatting")
	require.Equal(t, out, buf2.String(), "XHTML mixed-content formatting must be idempotent")
}

// TestSerializeXHTMLCharacterMap verifies Writer.CharacterMap applies to XHTML
// attribute values (including the synthesized id-from-name attribute) and text
// content in the XHTML serialization path (Serialization 3.1 §6). This XHTML path
// performs no URI percent-encoding, so character maps apply to every attribute.
func TestSerializeXHTMLCharacterMap(t *testing.T) {
	t.Parallel()

	const src = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Strict//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>t</title></head>
<body><a name="foo" title="foo">foo</a></body>
</html>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err, "parse XHTML document")

	var buf strings.Builder
	err = helium.NewWriter().CharacterMap(map[rune]string{'o': "0"}).WriteTo(&buf, doc)
	require.NoError(t, err, "serialize XHTML with character map")
	out := buf.String()

	// The character map ('o' -> "0") applies to the source attribute values, the
	// synthesized id (derived from @name on <a>), and text content.
	require.Contains(t, out, `name="f00"`, "output:\n%s", out)
	require.Contains(t, out, `title="f00"`, "output:\n%s", out)
	require.Contains(t, out, `id="f00"`, "output:\n%s", out)
	require.Contains(t, out, `>f00<`, "output:\n%s", out)
}
