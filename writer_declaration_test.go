package helium_test

import (
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

const (
	ver10 = "1.0"
	ver11 = "1.1"
)

// newDocWithRoot builds a minimal <root/> document for exercising the XML
// declaration writer.
func newDocWithRoot(t *testing.T, version, encoding string) *helium.Document {
	t.Helper()
	doc := helium.NewDocument(version, encoding, helium.StandaloneImplicitNo)
	root := doc.CreateElement("root")
	require.NoError(t, doc.AddChild(root))
	return doc
}

// TestDeclarationVersionInjectionRejected asserts that an effective XML version
// carrying a quote (or other non-VersionNum character) cannot break out of the
// version pseudo-attribute of the XML declaration. The writer must fail closed
// with ErrInvalidOutputVersion and emit no declaration bytes.
func TestDeclarationVersionInjectionRejected(t *testing.T) {
	t.Parallel()

	t.Run("via SetVersion", func(t *testing.T) {
		t.Parallel()
		doc := newDocWithRoot(t, `1.0" evil="x`, "UTF-8")
		out, err := helium.WriteString(doc)
		require.ErrorIs(t, err, helium.ErrInvalidOutputVersion)
		require.NotContains(t, out, "evil=")
		require.NotContains(t, out, "<?xml")
	})

	t.Run("via OutputVersion", func(t *testing.T) {
		t.Parallel()
		doc := newDocWithRoot(t, ver10, "UTF-8")
		var buf strings.Builder
		err := helium.NewWriter().OutputVersion(`1.0" evil="x`).WriteTo(&buf, doc)
		require.ErrorIs(t, err, helium.ErrInvalidOutputVersion)
		require.NotContains(t, buf.String(), "evil=")
		require.NotContains(t, buf.String(), "<?xml")
	})

	t.Run("non-1.x version rejected", func(t *testing.T) {
		t.Parallel()
		doc := newDocWithRoot(t, "2.0", "UTF-8")
		out, err := helium.WriteString(doc)
		require.ErrorIs(t, err, helium.ErrInvalidOutputVersion)
		require.NotContains(t, out, "<?xml")
	})
}

// TestDeclarationEncodingInjectionRejected asserts that a malformed effective
// encoding label (one carrying a quote or a space) cannot inject markup into the
// encoding pseudo-attribute. The writer must fail closed with
// ErrUnsupportedOutputEncoding and emit no declaration bytes.
func TestDeclarationEncodingInjectionRejected(t *testing.T) {
	t.Parallel()

	t.Run("via OutputEncoding quote injection", func(t *testing.T) {
		t.Parallel()
		doc := newDocWithRoot(t, ver10, "UTF-8")
		var buf strings.Builder
		err := helium.NewWriter().OutputEncoding(`UTF-8" x="y`).WriteTo(&buf, doc)
		require.ErrorIs(t, err, helium.ErrUnsupportedOutputEncoding)
		require.NotContains(t, buf.String(), `x="y`)
		require.NotContains(t, buf.String(), "<?xml")
	})

	t.Run("via SetEncoding malformed label", func(t *testing.T) {
		t.Parallel()
		doc := newDocWithRoot(t, ver10, "bad enc")
		out, err := helium.WriteString(doc)
		require.ErrorIs(t, err, helium.ErrUnsupportedOutputEncoding)
		require.NotContains(t, out, "<?xml")
	})
}

// TestDeclarationValidValuesByteIdentical asserts that every legal version and
// encoding combination serializes with the expected declaration and no error —
// the validation must not perturb valid output.
func TestDeclarationValidValuesByteIdentical(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		version  string
		encoding string
		wantDecl string
	}{
		{"1.0 UTF-8", ver10, "UTF-8", `<?xml version="1.0" encoding="UTF-8"?>`},
		{"1.1 UTF-8", ver11, "UTF-8", `<?xml version="1.1" encoding="UTF-8"?>`},
		{"1.0 ISO-8859-1", ver10, "ISO-8859-1", `<?xml version="1.0" encoding="ISO-8859-1"?>`},
		{"1.0 US-ASCII", ver10, "US-ASCII", `<?xml version="1.0" encoding="US-ASCII"?>`},
		{"1.0 Shift_JIS", ver10, "Shift_JIS", `<?xml version="1.0" encoding="Shift_JIS"?>`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			doc := newDocWithRoot(t, tc.version, tc.encoding)
			out, err := helium.WriteString(doc)
			require.NoError(t, err)
			require.True(t, strings.HasPrefix(out, tc.wantDecl),
				"got %q, want prefix %q", out, tc.wantDecl)
		})
	}
}

// TestDeclarationEmptyEncodingOmitsPseudoAttr asserts that an empty effective
// encoding still omits the encoding pseudo-attribute (unchanged behavior) rather
// than being rejected.
func TestDeclarationEmptyEncodingOmitsPseudoAttr(t *testing.T) {
	t.Parallel()

	doc := newDocWithRoot(t, ver10, "")
	out, err := helium.WriteString(doc)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(out, `<?xml version="1.0"?>`),
		"got %q", out)
	require.NotContains(t, out, "encoding=")
}
