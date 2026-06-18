package helium_test

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lestrrat-go/helium"

	"github.com/lestrrat-go/helium/enum"
	"github.com/stretchr/testify/require"
)

// infiniteFile is an fs.File whose Read never returns io.EOF, simulating an
// unbounded source such as /dev/zero. It records whether Close was called.
type infiniteFile struct {
	closed *atomic.Bool
}

func (f *infiniteFile) Stat() (fs.FileInfo, error) { return nil, fs.ErrInvalid }

func (f *infiniteFile) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'A'
	}
	return len(p), nil
}

func (f *infiniteFile) Close() error {
	f.closed.Store(true)
	return nil
}

// infiniteFS hands out a single infiniteFile for any path, recording closure.
type infiniteFS struct {
	closed *atomic.Bool
}

func (s infiniteFS) Open(string) (fs.File, error) {
	return &infiniteFile{closed: s.closed}, nil
}

// TestExternalEntitySizeCap ensures that an external parsed entity backed by an
// unbounded source is rejected (rather than read via io.ReadAll, which would
// OOM) and that the resolved input is closed.
func TestExternalEntitySizeCap(t *testing.T) {
	t.Parallel()

	const input = `<?xml version="1.0"?>
<!DOCTYPE r [<!ENTITY x SYSTEM "zero">]><r>&x;</r>`

	var closed atomic.Bool
	p := helium.NewParser().SubstituteEntities(true).FS(infiniteFS{closed: &closed})

	done := make(chan struct{})
	var parseErr error
	go func() {
		defer close(done)
		_, parseErr = p.Parse(t.Context(), []byte(input))
	}()

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("parsing an unbounded external entity did not terminate (no size cap)")
	}

	require.Error(t, parseErr, "unbounded external entity must be rejected")
	require.True(t, closed.Load(), "resolved external entity input must be closed")
}

// readCloserFile wraps a string and records Close, used to verify external
// entity inputs are closed even on the success path.
type readCloserFile struct {
	r      *io.SectionReader
	closed *atomic.Bool
}

func (f *readCloserFile) Stat() (fs.FileInfo, error) { return nil, fs.ErrInvalid }
func (f *readCloserFile) Read(p []byte) (int, error) { return f.r.Read(p) }
func (f *readCloserFile) Close() error {
	f.closed.Store(true)
	return nil
}

type closingFS struct {
	data   string
	closed *atomic.Bool
}

func (s closingFS) Open(string) (fs.File, error) {
	return &readCloserFile{
		r:      io.NewSectionReader(strings.NewReader(s.data), 0, int64(len(s.data))),
		closed: s.closed,
	}, nil
}

func TestExternalEntityInputClosed(t *testing.T) {
	t.Parallel()

	const input = `<?xml version="1.0"?>
<!DOCTYPE r [<!ENTITY x SYSTEM "ext">]><r>&x;</r>`

	var closed atomic.Bool
	p := helium.NewParser().SubstituteEntities(true).FS(closingFS{data: "<e>ok</e>", closed: &closed})
	_, err := p.Parse(t.Context(), []byte(input))
	require.NoError(t, err)
	require.True(t, closed.Load(), "resolved external entity input must be closed on success")
}

// TestEntityValueMalformedGeneralRef ensures that a general reference inside an
// EntityValue is syntax-checked: a missing semicolon must be rejected even
// though the general reference itself is not expanded.
func TestEntityValueMalformedGeneralRef(t *testing.T) {
	t.Parallel()

	const input = `<!DOCTYPE r [<!ENTITY e "&broken">]><r/>`

	p := helium.NewParser()
	_, err := p.Parse(t.Context(), []byte(input))
	require.Error(t, err, "malformed general reference in entity value must be rejected")
}

// TestEntityValueValidGeneralRefLiteral ensures that a well-formed general
// reference in an EntityValue is accepted and stored literally (not expanded).
func TestEntityValueValidGeneralRefLiteral(t *testing.T) {
	t.Parallel()

	const input = `<!DOCTYPE r [<!ENTITY e "&amp; &good;">]><r/>`

	p := helium.NewParser()
	_, err := p.Parse(t.Context(), []byte(input))
	require.NoError(t, err, "well-formed general references in entity value must be accepted")
}

func TestEntityAmplification(t *testing.T) {
	t.Parallel()

	t.Run("billion laughs", func(t *testing.T) {
		t.Parallel()
		// Classic billion-laughs: 10 nested entities, each referencing 10 copies
		// of the previous. Total expansion: 10^10 bytes.
		xml := `<?xml version="1.0"?>
<!DOCTYPE lolz [
  <!ENTITY lol "lol">
  <!ENTITY lol2 "&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;">
  <!ENTITY lol3 "&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;">
  <!ENTITY lol4 "&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;">
  <!ENTITY lol5 "&lol4;&lol4;&lol4;&lol4;&lol4;&lol4;&lol4;&lol4;&lol4;&lol4;">
  <!ENTITY lol6 "&lol5;&lol5;&lol5;&lol5;&lol5;&lol5;&lol5;&lol5;&lol5;&lol5;">
  <!ENTITY lol7 "&lol6;&lol6;&lol6;&lol6;&lol6;&lol6;&lol6;&lol6;&lol6;&lol6;">
  <!ENTITY lol8 "&lol7;&lol7;&lol7;&lol7;&lol7;&lol7;&lol7;&lol7;&lol7;&lol7;">
  <!ENTITY lol9 "&lol8;&lol8;&lol8;&lol8;&lol8;&lol8;&lol8;&lol8;&lol8;&lol8;">
]>
<root>&lol9;</root>`

		p := helium.NewParser().SubstituteEntities(true)
		_, err := p.Parse(t.Context(), []byte(xml))
		require.Error(t, err, "billion laughs should be rejected")
		require.Contains(t, err.Error(), "amplification")
	})

	t.Run("quadratic blowup", func(t *testing.T) {
		t.Parallel()
		// Large entity repeated many times: quadratic blowup.
		// helium.Entity content is 100KB, referenced 100 times → 10MB expansion from ~110KB input.
		bigContent := strings.Repeat("A", 100_000)
		refs := strings.Repeat("&big;", 100)
		xml := fmt.Sprintf(`<?xml version="1.0"?>
<!DOCTYPE root [
  <!ENTITY big "%s">
]>
<root>%s</root>`, bigContent, refs)

		p := helium.NewParser().SubstituteEntities(true)
		_, err := p.Parse(t.Context(), []byte(xml))
		require.Error(t, err, "quadratic blowup should be rejected")
		require.Contains(t, err.Error(), "amplification")
	})

	t.Run("normal entities", func(t *testing.T) {
		t.Parallel()
		// Small expansion well within limits — must succeed.
		xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ENTITY greeting "Hello, World!">
]>
<root>&greeting;</root>`

		p := helium.NewParser().SubstituteEntities(true)
		doc, err := p.Parse(t.Context(), []byte(xml))
		require.NoError(t, err)
		require.NotNil(t, doc)
	})

	t.Run("RelaxLimits disables guard", func(t *testing.T) {
		t.Parallel()
		// With RelaxLimits, billion laughs should be allowed (guard disabled).
		// Use a smaller version to avoid actual memory exhaustion.
		xml := `<?xml version="1.0"?>
<!DOCTYPE lolz [
  <!ENTITY lol "lol">
  <!ENTITY lol2 "&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;">
  <!ENTITY lol3 "&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;">
  <!ENTITY lol4 "&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;">
  <!ENTITY lol5 "&lol4;&lol4;&lol4;&lol4;&lol4;&lol4;&lol4;&lol4;&lol4;&lol4;">
]>
<root>&lol5;</root>`

		p := helium.NewParser().SubstituteEntities(true).RelaxLimits(true)
		doc, err := p.Parse(t.Context(), []byte(xml))
		require.NoError(t, err)
		require.NotNil(t, doc)
	})

	t.Run("RelaxLimits still capped by absolute ceiling", func(t *testing.T) {
		// Intentionally NOT t.Parallel: this subtest drives expansion up
		// to entityHardCeiling (1 GiB). Running it alongside the parallel
		// subtests above amplified peak memory under loaded CI runners.
		// The ceiling does eventually trip, but the parser still
		// materializes nontrivial intermediate state, so we serialize it.
		// A bigger billion-laughs that would expand to many GB even with
		// the ratio check disabled. The absolute ceiling (entityHardCeiling
		// in parserctx.go) must still trip and abort the parse.
		xml := `<?xml version="1.0"?>
<!DOCTYPE lolz [
  <!ENTITY lol "lol">
  <!ENTITY lol2 "&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;">
  <!ENTITY lol3 "&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;">
  <!ENTITY lol4 "&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;">
  <!ENTITY lol5 "&lol4;&lol4;&lol4;&lol4;&lol4;&lol4;&lol4;&lol4;&lol4;&lol4;">
  <!ENTITY lol6 "&lol5;&lol5;&lol5;&lol5;&lol5;&lol5;&lol5;&lol5;&lol5;&lol5;">
  <!ENTITY lol7 "&lol6;&lol6;&lol6;&lol6;&lol6;&lol6;&lol6;&lol6;&lol6;&lol6;">
  <!ENTITY lol8 "&lol7;&lol7;&lol7;&lol7;&lol7;&lol7;&lol7;&lol7;&lol7;&lol7;">
  <!ENTITY lol9 "&lol8;&lol8;&lol8;&lol8;&lol8;&lol8;&lol8;&lol8;&lol8;&lol8;">
]>
<root>&lol9;</root>`

		p := helium.NewParser().SubstituteEntities(true).RelaxLimits(true)
		_, err := p.Parse(t.Context(), []byte(xml))
		require.Error(t, err, "absolute ceiling must trip even with RelaxLimits")
		require.Contains(t, err.Error(), "maximum entity expansion size",
			"error must explain the ceiling, got: %v", err)
		require.Regexp(t, `\(\d+ > \d+\)`, err.Error(),
			"error must include observed and configured sizes for diagnosis, got: %v", err)
	})
}

func TestPredefinedEntities(t *testing.T) {
	// Predefined entities (&lt; &gt; &amp; &apos; &quot;) must never trigger the guard.
	xml := `<?xml version="1.0"?>
<root>&lt;&gt;&amp;&apos;&quot;</root>`

	p := helium.NewParser()
	doc, err := p.Parse(t.Context(), []byte(xml))
	require.NoError(t, err)
	require.NotNil(t, doc)
}

func TestPredefinedEntityRedeclaration(t *testing.T) {
	t.Run("valid redeclaration accepted", func(t *testing.T) {
		// §4.6: redeclaring lt with content "<" (via &#60;) is allowed.
		xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ENTITY lt "&#60;">
]>
<root>&lt;</root>`
		_, err := helium.NewParser().Parse(t.Context(), []byte(xml))
		require.NoError(t, err)
	})

	t.Run("invalid redeclaration rejected", func(t *testing.T) {
		// §4.6: redeclaring lt with wrong content is a hard error.
		xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ENTITY lt "X">
]>
<root>&lt;</root>`
		_, err := helium.NewParser().Parse(t.Context(), []byte(xml))
		require.Error(t, err)
		require.Contains(t, err.Error(), "redeclared")
	})

	t.Run("valid redeclaration with char ref accepted", func(t *testing.T) {
		// Content is &#60; (char ref for <), which resolves to <
		xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ENTITY lt "&#60;">
  <!ENTITY gt "&#62;">
  <!ENTITY amp "&#38;">
  <!ENTITY apos "&#39;">
  <!ENTITY quot "&#34;">
]>
<root/>`
		_, err := helium.NewParser().Parse(t.Context(), []byte(xml))
		require.NoError(t, err)
	})

	t.Run("DTD.AddEntity rejects wrong content", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
		dtd, err := doc.CreateDTD()
		require.NoError(t, err)
		_, err = dtd.AddEntity("amp", enum.InternalGeneralEntity, "", "", "wrong")
		require.Error(t, err)
		require.Contains(t, err.Error(), "redeclared")
	})

	t.Run("DTD.AddEntity accepts correct content", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
		dtd, err := doc.CreateDTD()
		require.NoError(t, err)
		_, err = dtd.AddEntity("amp", enum.InternalGeneralEntity, "", "", "&")
		require.NoError(t, err)
	})

	t.Run("DTD.AddEntity accepts char ref content", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
		dtd, err := doc.CreateDTD()
		require.NoError(t, err)
		// &#60; resolves to <
		_, err = dtd.AddEntity("lt", enum.InternalGeneralEntity, "", "", "&#60;")
		require.NoError(t, err)
	})
}

func TestUndeclaredEntityFatal(t *testing.T) {
	t.Parallel()

	// An undeclared general entity reference, with no DTD/external subset
	// and no parameter-entity references, is a fatal well-formedness error.
	xml := `<r>&bogus;</r>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.Error(t, err, "undeclared entity with no DTD must be fatal")
	require.Nil(t, doc, "no document should be returned on a fatal error")
	require.ErrorIs(t, err, helium.ErrUndeclaredEntity, "error must be the undeclared-entity sentinel")

	var pe helium.ErrParseError
	require.True(t, errors.As(err, &pe), "error must be an ErrParseError")
	require.Equal(t, helium.ErrorLevelFatal, pe.Level, "undeclared entity must be a fatal-level error")
}

func TestEntityDepthLimit(t *testing.T) {
	// Build deeply nested entity references (depth > 40).
	var dtd strings.Builder
	dtd.WriteString(`<?xml version="1.0"?>` + "\n" + `<!DOCTYPE root [` + "\n")
	dtd.WriteString(`  <!ENTITY e0 "x">` + "\n")
	for i := 1; i <= 45; i++ {
		fmt.Fprintf(&dtd, "  <!ENTITY e%d \"&e%d;\">\n", i, i-1)
	}
	dtd.WriteString("]>\n")
	dtd.WriteString("<root>&e45;</root>")

	p := helium.NewParser().SubstituteEntities(true).RelaxLimits(true) // disable amplification guard to test depth only
	_, err := p.Parse(t.Context(), []byte(dtd.String()))
	require.Error(t, err, "depth > 40 should still error")
	require.Contains(t, err.Error(), "entity loop")
}
