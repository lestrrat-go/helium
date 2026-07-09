package helium_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// utf16be encodes an ASCII string as UTF-16BE bytes (each ASCII code point
// becomes 0x00 followed by the byte), the form used for the fixed-width test
// documents below.
func utf16be(s string) []byte {
	out := make([]byte, 0, len(s)*2)
	for i := range len(s) {
		out = append(out, 0x00, s[i])
	}
	return out
}

// utf16le encodes an ASCII string as UTF-16LE bytes (byte then 0x00).
func utf16le(s string) []byte {
	out := make([]byte, 0, len(s)*2)
	for i := range len(s) {
		out = append(out, s[i], 0x00)
	}
	return out
}

// TestBOMEncodingConflict covers XML §4.3.3: a byte-order mark asserts the
// entity's encoding, so a declaration naming a contradicting encoding is a
// fatal well-formedness error (W3C xml suite hst-lhs-007, hst-lhs-008). Each
// contradicting document must be fatal, while the well-formed near-misses (a
// matching declaration, a BOM alias, and — crucially — a BOM-less document
// declaring a single-byte encoding) must still parse, guarding against
// over-rejection.
func TestBOMEncodingConflict(t *testing.T) {
	t.Parallel()

	bomUTF8 := []byte{0xEF, 0xBB, 0xBF}
	bomUTF16BE := []byte{0xFE, 0xFF}
	bomUTF16LE := []byte{0xFF, 0xFE}

	t.Run("rejected", func(t *testing.T) {
		t.Parallel()

		t.Run("utf-8 BOM with iso-8859-1 declaration", func(t *testing.T) {
			t.Parallel()
			src := append(append([]byte{}, bomUTF8...),
				[]byte(`<?xml version='1.0' encoding='iso-8859-1'?><x/>`)...)
			_, err := helium.NewParser().Parse(t.Context(), src)
			require.ErrorIs(t, err, helium.ErrEncodingBOMMismatch)
		})

		t.Run("utf-16be BOM with utf-8 declaration", func(t *testing.T) {
			t.Parallel()
			src := append(append([]byte{}, bomUTF16BE...),
				utf16be(`<?xml version='1.0' encoding='utf-8'?><x/>`)...)
			_, err := helium.NewParser().Parse(t.Context(), src)
			require.ErrorIs(t, err, helium.ErrEncodingBOMMismatch)
		})

		t.Run("utf-16le BOM with utf-8 declaration", func(t *testing.T) {
			t.Parallel()
			src := append(append([]byte{}, bomUTF16LE...),
				utf16le(`<?xml version='1.0' encoding='utf-8'?><x/>`)...)
			_, err := helium.NewParser().Parse(t.Context(), src)
			require.ErrorIs(t, err, helium.ErrEncodingBOMMismatch)
		})
	})

	t.Run("accepted", func(t *testing.T) {
		t.Parallel()

		t.Run("utf-8 BOM with matching declaration", func(t *testing.T) {
			t.Parallel()
			src := append(append([]byte{}, bomUTF8...),
				[]byte(`<?xml version='1.0' encoding='UTF-8'?><x/>`)...)
			_, err := helium.NewParser().Parse(t.Context(), src)
			require.NoError(t, err)
		})

		t.Run("utf-8 BOM with no encoding declaration", func(t *testing.T) {
			t.Parallel()
			src := append(append([]byte{}, bomUTF8...),
				[]byte(`<?xml version='1.0'?><x/>`)...)
			_, err := helium.NewParser().Parse(t.Context(), src)
			require.NoError(t, err)
		})

		t.Run("utf-16be BOM with utf-16 alias", func(t *testing.T) {
			t.Parallel()
			src := append(append([]byte{}, bomUTF16BE...),
				utf16be(`<?xml version='1.0' encoding='UTF-16'?><x/>`)...)
			_, err := helium.NewParser().Parse(t.Context(), src)
			require.NoError(t, err)
		})

		t.Run("utf-16be BOM with utf-16be declaration", func(t *testing.T) {
			t.Parallel()
			src := append(append([]byte{}, bomUTF16BE...),
				utf16be(`<?xml version='1.0' encoding='UTF-16BE'?><x/>`)...)
			_, err := helium.NewParser().Parse(t.Context(), src)
			require.NoError(t, err)
		})

		// The key over-rejection guard: a BOM-less, ASCII-compatible document
		// that declares a single-byte encoding must NOT be treated as a
		// conflict — no BOM was consumed, so autoEncoding is empty.
		t.Run("no BOM with iso-8859-1 declaration", func(t *testing.T) {
			t.Parallel()
			_, err := helium.NewParser().Parse(t.Context(),
				[]byte(`<?xml version='1.0' encoding='iso-8859-1'?><x/>`))
			require.NoError(t, err)
		})
	})

	// IgnoreEncoding(true) suppresses the decoder switch but must NOT suppress
	// the BOM/encoding-mismatch well-formedness check — the declared encoding is
	// still recorded for the check even though ctx.encoding is erased.
	t.Run("ignore-encoding", func(t *testing.T) {
		t.Parallel()

		t.Run("utf-8 BOM with iso-8859-1 declaration still fatal", func(t *testing.T) {
			t.Parallel()
			src := append(append([]byte{}, bomUTF8...),
				[]byte(`<?xml version='1.0' encoding='iso-8859-1'?><x/>`)...)
			_, err := helium.NewParser().IgnoreEncoding(true).Parse(t.Context(), src)
			require.ErrorIs(t, err, helium.ErrEncodingBOMMismatch)
		})

		t.Run("utf-8 BOM with matching declaration parses", func(t *testing.T) {
			t.Parallel()
			src := append(append([]byte{}, bomUTF8...),
				[]byte(`<?xml version='1.0' encoding='UTF-8'?><x/>`)...)
			_, err := helium.NewParser().IgnoreEncoding(true).Parse(t.Context(), src)
			require.NoError(t, err)
		})

		t.Run("no BOM with iso-8859-1 declaration parses", func(t *testing.T) {
			t.Parallel()
			_, err := helium.NewParser().IgnoreEncoding(true).Parse(t.Context(),
				[]byte(`<?xml version='1.0' encoding='iso-8859-1'?><x/>`))
			require.NoError(t, err)
		})
	})

	// LenientXMLDecl(true) relaxes declaration parsing but must NOT suppress the
	// BOM/encoding-mismatch check. The declared EncName is recorded at the leaf
	// EncName parser, so it is authoritative on the lenient path too. The check
	// must also hold when LenientXMLDecl and IgnoreEncoding are combined.
	t.Run("lenient-decl", func(t *testing.T) {
		t.Parallel()

		t.Run("utf-8 BOM with iso-8859-1 declaration still fatal", func(t *testing.T) {
			t.Parallel()
			src := append(append([]byte{}, bomUTF8...),
				[]byte(`<?xml version='1.0' encoding='iso-8859-1'?><x/>`)...)
			_, err := helium.NewParser().LenientXMLDecl(true).Parse(t.Context(), src)
			require.ErrorIs(t, err, helium.ErrEncodingBOMMismatch)
		})

		t.Run("utf-16be BOM with utf-8 declaration still fatal", func(t *testing.T) {
			t.Parallel()
			src := append(append([]byte{}, bomUTF16BE...),
				utf16be(`<?xml version='1.0' encoding='utf-8'?><x/>`)...)
			_, err := helium.NewParser().LenientXMLDecl(true).Parse(t.Context(), src)
			require.ErrorIs(t, err, helium.ErrEncodingBOMMismatch)
		})

		t.Run("lenient plus ignore-encoding still fatal", func(t *testing.T) {
			t.Parallel()
			src := append(append([]byte{}, bomUTF8...),
				[]byte(`<?xml version='1.0' encoding='iso-8859-1'?><x/>`)...)
			_, err := helium.NewParser().
				LenientXMLDecl(true).IgnoreEncoding(true).Parse(t.Context(), src)
			require.ErrorIs(t, err, helium.ErrEncodingBOMMismatch)
		})

		// The combined knobs must not over-reject: a matching BOM+declaration
		// pair and a BOM-less document still parse under LenientXMLDecl+IgnoreEncoding.
		t.Run("lenient plus ignore-encoding matching declaration parses", func(t *testing.T) {
			t.Parallel()
			src := append(append([]byte{}, bomUTF8...),
				[]byte(`<?xml version='1.0' encoding='UTF-8'?><x/>`)...)
			_, err := helium.NewParser().
				LenientXMLDecl(true).IgnoreEncoding(true).Parse(t.Context(), src)
			require.NoError(t, err)
		})

		t.Run("lenient plus ignore-encoding no BOM parses", func(t *testing.T) {
			t.Parallel()
			_, err := helium.NewParser().
				LenientXMLDecl(true).IgnoreEncoding(true).Parse(t.Context(),
				[]byte(`<?xml version='1.0' encoding='iso-8859-1'?><x/>`))
			require.NoError(t, err)
		})

		t.Run("utf-8 BOM with matching declaration parses", func(t *testing.T) {
			t.Parallel()
			src := append(append([]byte{}, bomUTF8...),
				[]byte(`<?xml version='1.0' encoding='UTF-8'?><x/>`)...)
			_, err := helium.NewParser().LenientXMLDecl(true).Parse(t.Context(), src)
			require.NoError(t, err)
		})

		t.Run("no BOM with iso-8859-1 declaration parses", func(t *testing.T) {
			t.Parallel()
			_, err := helium.NewParser().LenientXMLDecl(true).Parse(t.Context(),
				[]byte(`<?xml version='1.0' encoding='iso-8859-1'?><x/>`))
			require.NoError(t, err)
		})
	})
}
