package helium

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestOutputEncodingUSASCIIRejectsTextNoEnc asserts that under an explicit
// US-ASCII OutputEncoding a text node carrying the xmlTextNoEnc marker (emitted
// verbatim with no escaping) whose content is non-ASCII fails with
// ErrUnsupportedOutputEncoding via the ASCII-reject net rather than leaking raw
// UTF-8 under the US-ASCII declaration. It is a white-box test because the marker
// has no public setter (U2).
func TestOutputEncodingUSASCIIRejectsTextNoEnc(t *testing.T) {
	t.Parallel()

	doc := NewDefaultDocument()
	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.SetDocumentElement(root))
	txt := doc.CreateText([]byte("café"))
	txt.name = xmlTextNoEnc // mark as pre-encoded, emitted without escaping
	require.NoError(t, root.AddChild(txt))

	var buf bytes.Buffer
	err = NewWriter().OutputEncoding("US-ASCII").WriteTo(&buf, doc)
	require.ErrorIs(t, err, ErrUnsupportedOutputEncoding)
	for i := range buf.Len() {
		require.Less(t, buf.Bytes()[i], byte(0x80), "leaked non-ASCII octet 0x%X at %d", buf.Bytes()[i], i)
	}
}
