package helium_test

import (
	"bytes"
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

// TestOutputVersionValidatedForFragment asserts that a non-empty OutputVersion
// override is validated on the bare-element/fragment path exactly as on the
// Document path: a malformed override fails with ErrInvalidOutputVersion and
// writes nothing, while a valid override (and no override) serializes normally.
func TestOutputVersionValidatedForFragment(t *testing.T) {
	t.Parallel()

	t.Run("element rejects malformed override", func(t *testing.T) {
		t.Parallel()
		doc := newDocWithRoot(t, ver10, "UTF-8")
		elem := doc.DocumentElement()
		var buf bytes.Buffer
		err := helium.NewWriter().OutputVersion("garbage").WriteTo(&buf, elem)
		require.ErrorIs(t, err, helium.ErrInvalidOutputVersion)
		require.Zero(t, buf.Len())
	})

	t.Run("element rejects injection override", func(t *testing.T) {
		t.Parallel()
		doc := newDocWithRoot(t, ver10, "UTF-8")
		elem := doc.DocumentElement()
		var buf bytes.Buffer
		err := helium.NewWriter().OutputVersion(`1.0" evil="x`).WriteTo(&buf, elem)
		require.ErrorIs(t, err, helium.ErrInvalidOutputVersion)
		require.Zero(t, buf.Len())
	})

	t.Run("document still rejects malformed override", func(t *testing.T) {
		t.Parallel()
		doc := newDocWithRoot(t, ver10, "UTF-8")
		var buf bytes.Buffer
		err := helium.NewWriter().OutputVersion("garbage").WriteTo(&buf, doc)
		require.ErrorIs(t, err, helium.ErrInvalidOutputVersion)
		require.Zero(t, buf.Len())
	})

	t.Run("element with valid override serializes", func(t *testing.T) {
		t.Parallel()
		doc := newDocWithRoot(t, ver10, "UTF-8")
		elem := doc.DocumentElement()
		var buf bytes.Buffer
		require.NoError(t, helium.NewWriter().OutputVersion(ver11).WriteTo(&buf, elem))
		require.Equal(t, "<root/>", buf.String())
	})

	t.Run("element with no override serializes", func(t *testing.T) {
		t.Parallel()
		doc := newDocWithRoot(t, ver10, "UTF-8")
		elem := doc.DocumentElement()
		var buf bytes.Buffer
		require.NoError(t, helium.NewWriter().WriteTo(&buf, elem))
		require.Equal(t, "<root/>", buf.String())
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

// TestDeclarationInvalidValueEmptyOutput asserts that an invalid effective
// version or encoding aborts serialization with ZERO bytes written to the
// caller's io.Writer. The validation runs at the writeDoc entry, ahead of the
// transcoding-encoder setup, so no encoder is installed and no BOM is flushed —
// a stronger guarantee than "no <?xml": a UTF-16/UTF-32 BOM contains no "<?xml"
// yet is still a leaked byte. The malformed labels below (UTF:16, UTF 16, UCS:4)
// are ones the internal encoder table would normalize and load — installing that
// encoder previously flushed its BOM before the EncName check rejected the label.
func TestDeclarationInvalidValueEmptyOutput(t *testing.T) {
	t.Parallel()

	t.Run("encoding UTF:16 leaks no BOM", func(t *testing.T) {
		t.Parallel()
		doc := newDocWithRoot(t, ver10, "UTF-8")
		var buf bytes.Buffer
		err := helium.NewWriter().OutputEncoding("UTF:16").WriteTo(&buf, doc)
		require.ErrorIs(t, err, helium.ErrUnsupportedOutputEncoding)
		require.Zero(t, buf.Len())
	})

	t.Run("encoding UTF 16 leaks no BOM", func(t *testing.T) {
		t.Parallel()
		doc := newDocWithRoot(t, ver10, "UTF-8")
		var buf bytes.Buffer
		err := helium.NewWriter().OutputEncoding("UTF 16").WriteTo(&buf, doc)
		require.ErrorIs(t, err, helium.ErrUnsupportedOutputEncoding)
		require.Zero(t, buf.Len())
	})

	t.Run("encoding UCS:4 leaks no BOM", func(t *testing.T) {
		t.Parallel()
		doc := newDocWithRoot(t, ver10, "UTF-8")
		var buf bytes.Buffer
		err := helium.NewWriter().OutputEncoding("UCS:4").WriteTo(&buf, doc)
		require.ErrorIs(t, err, helium.ErrUnsupportedOutputEncoding)
		require.Zero(t, buf.Len())
	})

	t.Run("invalid version writes nothing", func(t *testing.T) {
		t.Parallel()
		doc := newDocWithRoot(t, ver10, "UTF-8")
		var buf bytes.Buffer
		err := helium.NewWriter().OutputVersion("2.0").WriteTo(&buf, doc)
		require.ErrorIs(t, err, helium.ErrInvalidOutputVersion)
		require.Zero(t, buf.Len())
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
