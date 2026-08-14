package helium_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// underReportingFS serves a single DTD file whose Stat under-reports the
// size while Read produces far more than MaxExternalDTDSize. It also counts
// the bytes pulled from the underlying reader so the test can assert the
// read is bounded.
type underReportingFS struct {
	read *int64
}

type underReportingFile struct {
	read *int64
}

// Stat lies: it claims a tiny size so the early precheck does not catch the
// oversized content.
func (f *underReportingFile) Stat() (fs.FileInfo, error) {
	return underReportingInfo{}, nil
}

func (f *underReportingFile) Read(p []byte) (int, error) {
	// Endless stream of spaces; the bounded read must stop it.
	for i := range p {
		p[i] = ' '
	}
	*f.read += int64(len(p))
	return len(p), nil
}

func (f *underReportingFile) Close() error { return nil }

type underReportingInfo struct{}

func (underReportingInfo) Name() string { return "huge.dtd" }

func (underReportingInfo) Size() int64 { return 1 }

func (underReportingInfo) Mode() fs.FileMode { return 0 }

func (underReportingInfo) ModTime() time.Time { return time.Time{} }

func (underReportingInfo) IsDir() bool { return false }

func (underReportingInfo) Sys() any { return nil }

// errReadingFS serves a DTD whose Read returns a full buffer of bytes (taking
// the running total past the configured cap) together with a NON-EOF error on
// the cap-crossing read. The cap must be enforced against the bytes that were
// returned, before the read error is inspected, so the size-cap error still
// fires. A small cap is used so the bounded read does not pull megabytes; the
// fake records whether the simulated read error was actually returned so the
// test can prove the error path was exercised.
type errReadingFS struct {
	cap        int
	hitReadErr *bool
}

type errReadingFile struct {
	cap        int
	read       int64
	hitReadErr *bool
}

func (f *errReadingFile) Stat() (fs.FileInfo, error) { return underReportingInfo{}, nil }

func (f *errReadingFile) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = ' '
	}
	f.read += int64(len(p))
	// Once enough bytes have been handed out to reach the cap, return the
	// filled buffer alongside a non-EOF error. A reader that handled the
	// error before checking the byte count would escape the cap.
	if f.read >= int64(f.cap) {
		*f.hitReadErr = true
		return len(p), errors.New("simulated transport failure")
	}
	return len(p), nil
}

func (f *errReadingFile) Close() error { return nil }

// overReportingFS serves a single small, valid DTD whose Stat over-reports the
// size as MaxExternalDTDSize+1. The actual content is well under the cap, so
// the parse must succeed: this proves Stat is advisory and never used to
// reject.
type overReportingFS struct {
	data []byte
}

type overReportingFile struct {
	data []byte
	off  int
}

// Stat lies the other way: it claims a size above the cap even though Read
// yields only a few valid bytes.
func (f *overReportingFile) Stat() (fs.FileInfo, error) {
	return overReportingInfo{}, nil
}

func (f *overReportingFile) Read(p []byte) (int, error) {
	if f.off >= len(f.data) {
		return 0, io.EOF
	}
	n := copy(p, f.data[f.off:])
	f.off += n
	return n, nil
}

func (f *overReportingFile) Close() error { return nil }

type overReportingInfo struct{}

func (overReportingInfo) Name() string { return "small.dtd" }

func (overReportingInfo) Size() int64 { return helium.MaxExternalDTDSize + 1 }

func (overReportingInfo) Mode() fs.FileMode { return 0 }

func (overReportingInfo) ModTime() time.Time { return time.Time{} }

func (overReportingInfo) IsDir() bool { return false }

func (overReportingInfo) Sys() any { return nil }

// partialReadFS serves a small DTD whose Read hands back a few valid bytes and
// then a NON-EOF error (a truncated/partial read) well under the size cap. A
// truncated external subset must surface the read error, and must never be
// silently treated as an absent DTD.
type partialReadFS struct {
	prefix []byte
}

type partialReadFile struct {
	prefix []byte
	done   bool
}

func (f *partialReadFile) Stat() (fs.FileInfo, error) { return underReportingInfo{}, nil }

func (f *partialReadFile) Read(p []byte) (int, error) {
	if f.done {
		return 0, io.ErrUnexpectedEOF
	}
	n := copy(p, f.prefix)
	f.done = true
	return n, io.ErrUnexpectedEOF
}

func (f *partialReadFile) Close() error { return nil }

// recordingFS wraps an fs.FS and records every path passed to Open, so a test
// can assert which resources a parse attempted to load.
type recordingFS struct {
	inner  fs.FS
	mu     sync.Mutex
	opened []string
}

// dtdLoadDoc references an external subset by SYSTEM id. A valid content model
// for <doc> is declared in the external subset so a validating parse succeeds
// once the subset is actually loaded.
const dtdLoadDocName = "sub.dtd"

func dtdLoadDoc() string {
	return `<?xml version="1.0"?>` + "\n" +
		`<!DOCTYPE doc SYSTEM "` + dtdLoadDocName + `">` + "\n" +
		`<doc>hello</doc>`
}

func dtdLoadFS() fstest.MapFS {
	return fstest.MapFS{
		dtdLoadDocName: &fstest.MapFile{Data: []byte("<!ELEMENT doc (#PCDATA)>\n")},
	}
}

// warnCaptureSAX embeds the default TreeBuilder (so the full DOM is still
// built) and records every SAX Warning delivered during the parse. A wrapping
// SAX handler takes the generic SAX-dispatch path (pctx.treeBuilder stays nil),
// which is a supported configuration.
type warnCaptureSAX struct {
	*helium.TreeBuilder
	warnings []error
}

func (w *warnCaptureSAX) Warning(_ context.Context, err error) error {
	w.warnings = append(w.warnings, err)
	return nil
}

func TestParseExternalDTDLimits(t *testing.T) {
	t.Parallel()

	t.Run("size limit", func(t *testing.T) {
		const input = `<?xml version="1.0"?>
<!DOCTYPE r SYSTEM "huge.dtd">
<r/>`

		// An oversized external DTD must be rejected with a parse error rather
		// than being read whole into memory (potential OOM/hang).
		oversized := bytes.Repeat([]byte(" "), helium.MaxExternalDTDSize+1)
		fsys := fstest.MapFS{"huge.dtd": &fstest.MapFile{Data: oversized}}

		p := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).DefaultDTDAttributes(true).FS(fsys)
		_, err := p.Parse(t.Context(), []byte(input))
		require.Error(t, err, "oversized external DTD must produce a parse error")
	})

	t.Run("bounded read", func(t *testing.T) {
		const input = `<?xml version="1.0"?>
<!DOCTYPE r SYSTEM "huge.dtd">
<r/>`

		// A source whose Stat under-reports its size must still be rejected by
		// the bounded read, and that read must not consume more than
		// MaxExternalDTDSize+1 bytes from the underlying reader.
		var read int64
		fsys := underReportingFS{read: &read}

		p := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).DefaultDTDAttributes(true).FS(fsys)
		_, err := p.Parse(t.Context(), []byte(input))
		require.Error(t, err, "oversized external DTD must produce a parse error even when Stat under-reports size")
		require.ErrorIs(t, err, helium.ErrExternalDTDTooLarge, "rejection must come from the byte-count cap")
		// The bounded read must consume exactly MaxExternalDTDSize+1 bytes: enough
		// to prove the cap was exceeded, but no more. An implementation that
		// rejected before reading (e.g. trusting an advisory Stat) would leave
		// read==0 and fail the lower bound; one without a cap would overrun it.
		require.Equal(t, int64(helium.MaxExternalDTDSize)+1, read, "bounded read must consume exactly MaxExternalDTDSize+1 bytes")
	})

	t.Run("a read error is still capped", func(t *testing.T) {
		const input = `<?xml version="1.0"?>
<!DOCTYPE r SYSTEM "huge.dtd">
<r/>`

		// Use a small cap so the bounded read does not pull a 10 MiB stream. The
		// cap-crossing read returns n>0 plus a non-EOF error. The size cap must
		// still fire: the returned bytes already exceed the configured cap.
		const smallCap = 4096
		var hitReadErr bool
		fsys := errReadingFS{cap: smallCap, hitReadErr: &hitReadErr}

		p := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).DefaultDTDAttributes(true).
			MaxExternalDTDBytes(smallCap).FS(fsys)
		_, err := p.Parse(t.Context(), []byte(input))
		require.True(t, hitReadErr, "the simulated non-EOF read error must actually be returned by the fake")
		require.Error(t, err, "oversized external DTD must produce a parse error even when the read also errors")
		require.ErrorIs(t, err, helium.ErrExternalDTDTooLarge, "size cap must be enforced before the read error is handled")
	})

	t.Run("the stat size is advisory", func(t *testing.T) {
		const input = `<?xml version="1.0"?>
<!DOCTYPE r SYSTEM "small.dtd">
<r/>`

		// A small, valid DTD whose Stat over-reports its size must still load:
		// the cap is enforced against actual bytes read, not the advisory Stat.
		// The DTD defaults an attribute so the test observes that the external
		// subset was actually loaded and applied, not silently skipped.
		fsys := overReportingFS{data: []byte("<!ELEMENT r EMPTY>\n<!ATTLIST r x CDATA \"default\">")}

		p := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).DefaultDTDAttributes(true).FS(fsys)
		doc, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err, "small valid DTD must load even when Stat over-reports its size")

		root := doc.DocumentElement()
		require.NotNil(t, root, "root element should exist")
		x, ok := root.GetAttribute("x")
		require.True(t, ok, "external DTD ATTLIST default must be applied, proving the DTD was loaded")
		require.Equal(t, "default", x, "defaulted attribute value must come from the external DTD")
	})

	t.Run("a configurable limit", func(t *testing.T) {
		const input = `<?xml version="1.0"?>
<!DOCTYPE r SYSTEM "ext.dtd">
<r/>`

		t.Run("custom small limit rejects larger DTD", func(t *testing.T) {
			t.Parallel()

			// A 2 KiB DTD must be rejected when the configured cap is 1 KiB.
			oversized := bytes.Repeat([]byte(" "), 2<<10)
			fsys := fstest.MapFS{extDTDName: &fstest.MapFile{Data: oversized}}

			p := helium.NewParser().BlockXXE(false).
				LoadExternalDTD(true).
				DefaultDTDAttributes(true).
				MaxExternalDTDBytes(1 << 10).
				FS(fsys)
			_, err := p.Parse(t.Context(), []byte(input))
			require.Error(t, err, "DTD larger than the configured cap must be rejected")
			require.ErrorIs(t, err, helium.ErrExternalDTDTooLarge, "rejection must come from the byte-count cap")
		})

		t.Run("custom small limit allows smaller DTD", func(t *testing.T) {
			t.Parallel()

			// A DTD well under the 1 KiB cap must still load. It defaults an
			// attribute so the test observes that the DTD was actually applied,
			// not silently skipped.
			fsys := fstest.MapFS{extDTDName: &fstest.MapFile{Data: []byte("<!ELEMENT r EMPTY>\n<!ATTLIST r x CDATA \"default\">")}}

			p := helium.NewParser().BlockXXE(false).
				LoadExternalDTD(true).
				DefaultDTDAttributes(true).
				MaxExternalDTDBytes(1 << 10).
				FS(fsys)
			doc, err := p.Parse(t.Context(), []byte(input))
			require.NoError(t, err, "DTD under the configured cap must load")

			root := doc.DocumentElement()
			require.NotNil(t, root, "root element should exist")
			x, ok := root.GetAttribute("x")
			require.True(t, ok, "external DTD ATTLIST default must be applied, proving the DTD was loaded")
			require.Equal(t, "default", x, "defaulted attribute value must come from the external DTD")
		})

		t.Run("default cap allows a normal DTD over a small custom cap", func(t *testing.T) {
			t.Parallel()

			// Without configuring a custom cap, a DTD larger than 1 KiB (but well
			// under the 10 MiB default) must still load. It defaults an attribute
			// so the test observes that the DTD was actually applied.
			large := append([]byte("<!ELEMENT r EMPTY>\n<!ATTLIST r x CDATA \"default\">"), bytes.Repeat([]byte(" "), 4<<10)...)
			fsys := fstest.MapFS{extDTDName: &fstest.MapFile{Data: large}}

			p := helium.NewParser().BlockXXE(false).
				LoadExternalDTD(true).
				DefaultDTDAttributes(true).
				FS(fsys)
			doc, err := p.Parse(t.Context(), []byte(input))
			require.NoError(t, err, "DTD under the default cap must load")

			root := doc.DocumentElement()
			require.NotNil(t, root, "root element should exist")
			x, ok := root.GetAttribute("x")
			require.True(t, ok, "external DTD ATTLIST default must be applied, proving the DTD was loaded")
			require.Equal(t, "default", x, "defaulted attribute value must come from the external DTD")
		})

		// The sentinel convention matches every other Parser byte cap
		// (MaxNodeContentSize, MaxEntityAmplification, ...): 0 selects the built-in
		// default, and a negative value removes the cap. Build a DTD larger than the
		// 10 MiB default cap, but with no single whitespace run over the
		// node-content cap, so only the external-DTD byte cap decides the outcome.
		overDefault := []byte("<!ELEMENT r EMPTY>\n<!ATTLIST r x CDATA \"default\">\n")
		for range 12 {
			overDefault = append(overDefault, bytes.Repeat([]byte(" "), 1<<20)...)
			overDefault = append(overDefault, []byte("<!-- pad -->\n")...)
		}

		t.Run("zero selects the default cap and rejects a DTD over it", func(t *testing.T) {
			t.Parallel()

			fsys := fstest.MapFS{extDTDName: &fstest.MapFile{Data: overDefault}}

			p := helium.NewParser().BlockXXE(false).
				LoadExternalDTD(true).
				DefaultDTDAttributes(true).
				MaxExternalDTDBytes(0).
				FS(fsys)
			_, err := p.Parse(t.Context(), []byte(input))
			require.Error(t, err, "a DTD over the default cap must be rejected when the cap is the default (0)")
			require.ErrorIs(t, err, helium.ErrExternalDTDTooLarge, "rejection must come from the byte-count cap")
		})

		t.Run("negative removes the cap and loads a DTD over the default", func(t *testing.T) {
			t.Parallel()

			fsys := fstest.MapFS{extDTDName: &fstest.MapFile{Data: overDefault}}

			p := helium.NewParser().BlockXXE(false).
				LoadExternalDTD(true).
				DefaultDTDAttributes(true).
				MaxExternalDTDBytes(-1).
				FS(fsys)
			doc, err := p.Parse(t.Context(), []byte(input))
			require.NoError(t, err, "a negative cap must disable the limit, loading a DTD larger than the default")

			root := doc.DocumentElement()
			require.NotNil(t, root, "root element should exist")
			x, ok := root.GetAttribute("x")
			require.True(t, ok, "external DTD ATTLIST default must be applied, proving the oversized DTD was loaded")
			require.Equal(t, "default", x, "defaulted attribute value must come from the external DTD")
		})
	})

	t.Run("a partial-read error surfaces", func(t *testing.T) {
		const input = `<?xml version="1.0"?>
<!DOCTYPE r SYSTEM "trunc.dtd">
<r/>`

		// The DTD content is well under the cap but Read returns a non-EOF error,
		// modelling a truncated transport. The parse must fail, and must never silently
		// accept the document as if no external subset existed.
		fsys := partialReadFS{prefix: []byte("<!ELEMENT r EMPTY>")}

		p := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).DefaultDTDAttributes(true).FS(fsys)
		_, err := p.Parse(t.Context(), []byte(input))
		require.Error(t, err, "a truncated external DTD read must surface as a parse error")
	})
}

func TestParseExternalDTDMalformed(t *testing.T) {
	t.Parallel()

	t.Run("a malformed declaration terminates", func(t *testing.T) {
		const input = `<?xml version="1.0"?>
<!DOCTYPE r SYSTEM "bogus.dtd">
<r/>`

		// "<!BOGUS" is not a valid markup declaration and may neither advance the
		// cursor nor return an error. The progress guard must turn that into a
		// terminating error instead of an infinite loop.
		fsys := fstest.MapFS{"bogus.dtd": &fstest.MapFile{Data: []byte("<!BOGUS")}}

		p := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).DefaultDTDAttributes(true).FS(fsys)

		done := make(chan struct{})
		var err error
		go func() {
			defer close(done)
			_, err = p.Parse(t.Context(), []byte(input))
		}()

		select {
		case <-done:
			require.Error(t, err, "a malformed external DTD declaration must produce a parse error")
		case <-time.After(10 * time.Second):
			t.Fatal("parsing a malformed external DTD did not terminate (no cursor-progress guard)")
		}
	})

	t.Run("a malformed declaration reports its location", func(t *testing.T) {
		const input = `<?xml version="1.0"?>
<!DOCTYPE r SYSTEM "bogus.dtd">
<r/>`

		// A malformed declaration in the external subset must report the external
		// DTD's location, not the main document's doctype line. The progress-guard
		// error must be raised while the external DTD cursor and baseURI are still
		// active so the reported File carries the DTD path.
		fsys := fstest.MapFS{"bogus.dtd": &fstest.MapFile{Data: []byte("<!BOGUS")}}

		p := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).DefaultDTDAttributes(true).FS(fsys)
		_, err := p.Parse(t.Context(), []byte(input))
		require.Error(t, err, "a malformed external DTD declaration must produce a parse error")

		var pe helium.ErrParseError
		require.ErrorAs(t, err, &pe, "error must be a structured parse error")
		require.Equal(t, "bogus.dtd", pe.File, "error must reference the external DTD, not the main document")
	})

	t.Run("a malformed declaration inside INCLUDE surfaces", func(t *testing.T) {
		const input = `<?xml version="1.0"?>
<!DOCTYPE r SYSTEM "inc.dtd">
<r/>`

		// A malformed declaration ("<!BOGUS") inside a WELL-FORMED, properly
		// terminated top-level <![INCLUDE[ ... ]]> section must surface as a parse
		// error. Previously the top-level external-subset loop swallowed EVERY error
		// from parseConditionalSections, silently accepting the bogus declaration.
		// Now conditional-section errors propagate: a missing/malformed keyword and
		// an unterminated "]]>" section are both fatal, and an actual declaration
		// parse error inside the INCLUDE body propagates.
		const dtd = `<![INCLUDE[ <!BOGUS ]]>`
		fsys := fstest.MapFS{"inc.dtd": &fstest.MapFile{Data: []byte(dtd)}}

		p := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).DefaultDTDAttributes(true).FS(fsys)
		_, err := p.Parse(t.Context(), []byte(input))
		require.Error(t, err, "a malformed declaration inside a top-level INCLUDE section must surface as a parse error")
	})

	t.Run("an unterminated INCLUDE does not hang", func(t *testing.T) {
		const input = `<?xml version="1.0"?>
<!DOCTYPE r SYSTEM "inc.dtd">
<r/>`

		// An external DTD whose <![INCLUDE[ ... ]]> section reaches EOF before its
		// "]]>" terminator must report an error PROMPTLY. The INCLUDE body loop reads
		// the section through the shared declaration step; when that step signals stop
		// (the section's own cursor is exhausted), getCursor() would auto-pop the
		// spent section cursor up to the main document cursor (which is not Done),
		// defeating the EOF check. Honoring the stop signal — and inspecting the floor
		// cursor directly — turns the former infinite loop into a prompt error.
		const dtd = `<![INCLUDE[
<!ELEMENT r EMPTY>`
		fsys := fstest.MapFS{"inc.dtd": &fstest.MapFile{Data: []byte(dtd)}}

		// Guard against a regression manifesting as a hang: run the parse on a
		// goroutine with a deadline so a re-introduced infinite loop fails the test
		// instead of hanging the whole suite. The requirement here is PROMPT
		// completion (whether or not the parse surfaces a conditional-section error),
		// not a hang.
		ctx := t.Context()
		done := make(chan struct{})
		go func() {
			defer close(done)
			p := helium.NewParser().LoadExternalDTD(true).DefaultDTDAttributes(true).FS(fsys)
			_, _ = p.Parse(ctx, []byte(input))
		}()

		select {
		case <-done:
			// Completed promptly: the unterminated INCLUDE section did not loop.
		case <-time.After(5 * time.Second):
			t.Fatal("parsing an unterminated INCLUDE section hung (infinite loop regression)")
		}
	})
}

func TestParseExternalDTDParameterEntity(t *testing.T) {
	t.Parallel()

	t.Run("expands", func(t *testing.T) {
		const input = `<?xml version="1.0"?>
<!DOCTYPE r SYSTEM "pe.dtd">
<r/>`

		// The external subset declares a parameter entity whose replacement text is
		// a markup declaration, then references it. The reference must be expanded
		// (not merely validated and skipped) so the <!ATTLIST> takes effect and the
		// default attribute is applied to <r/>. A trailing newline after the
		// reference exercises the progress guard: it must not misfire on valid
		// PE-expanding input.
		const dtd = `<!ELEMENT r EMPTY>
<!ENTITY % defaults "<!ATTLIST r x CDATA 'default'>">
%defaults;
`
		fsys := fstest.MapFS{"pe.dtd": &fstest.MapFile{Data: []byte(dtd)}}

		p := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).DefaultDTDAttributes(true).FS(fsys)
		doc, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err, "valid external-subset parameter-entity reference must parse")
		require.NotNil(t, doc, "document must be returned")

		root := doc.DocumentElement()
		require.NotNil(t, root, "root element must be available")

		val, ok := root.GetAttribute("x")
		require.True(t, ok, "default attribute from expanded PE must be present")
		require.Equal(t, "default", val, "expanded PE must apply the default attribute value")
	})

	t.Run("a conditional section followed by a declaration", func(t *testing.T) {
		const input = `<?xml version="1.0"?>
<!DOCTYPE r SYSTEM "cs.dtd">
<r/>`

		// The external subset declares a parameter entity whose replacement text is
		// a conditional section, references it, and THEN declares another markup
		// declaration. After the PE expands to the conditional section and is
		// exhausted, the loop must resume in the parent DTD and apply the trailing
		// <!ATTLIST>. Before the fix, the conditional-section path "continue"-d past
		// the cursor-cleanup/progress guard, leaving the spent PE cursor on the
		// stack so the next iteration broke the loop and the trailing declaration
		// was silently skipped.
		const dtd = `<!ELEMENT r EMPTY>
<!ENTITY % cs "<![INCLUDE[ <!ELEMENT a EMPTY> ]]>">
%cs;
<!ATTLIST r x CDATA 'd'>
`
		fsys := fstest.MapFS{"cs.dtd": &fstest.MapFile{Data: []byte(dtd)}}

		p := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).DefaultDTDAttributes(true).FS(fsys)
		doc, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err, "PE expanding to a conditional section must parse")
		require.NotNil(t, doc, "document must be returned")

		root := doc.DocumentElement()
		require.NotNil(t, root, "root element must be available")

		val, ok := root.GetAttribute("x")
		require.True(t, ok, "declaration following the PE conditional section must be applied")
		require.Equal(t, "d", val, "trailing <!ATTLIST> must not be silently skipped")
	})

	t.Run("whitespace followed by a declaration", func(t *testing.T) {
		const input = `<?xml version="1.0"?>
<!DOCTYPE r SYSTEM "ws.dtd">
<r/>`

		// The external subset declares a parameter entity whose replacement text is
		// ONLY whitespace, references it, and THEN declares another markup
		// declaration. The blank skip consumes the PE's entire replacement text, so
		// the PE cursor is exhausted by the skip itself. The loop must pop the spent
		// PE cursor and resume in the parent DTD to apply the trailing <!ATTLIST>.
		// Before the fix, the blank-skip's Done()-cursor break exited the loop and
		// the deferred cleanup popped the parent DTD cursor too, silently skipping
		// the trailing declaration.
		const dtd = `<!ELEMENT r EMPTY>
<!ENTITY % ws "   ">
%ws;
<!ATTLIST r x CDATA 'd'>
`
		fsys := fstest.MapFS{"ws.dtd": &fstest.MapFile{Data: []byte(dtd)}}

		p := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).DefaultDTDAttributes(true).FS(fsys)
		doc, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err, "PE expanding to only whitespace must parse")
		require.NotNil(t, doc, "document must be returned")

		root := doc.DocumentElement()
		require.NotNil(t, root, "root element must be available")

		val, ok := root.GetAttribute("x")
		require.True(t, ok, "declaration following the whitespace-only PE must be applied")
		require.Equal(t, "d", val, "trailing <!ATTLIST> must not be silently skipped")
	})

	t.Run("expands inside an INCLUDE section", func(t *testing.T) {
		const input = `<?xml version="1.0"?>
<!DOCTYPE r SYSTEM "inc.dtd">
<r/>`

		// The external subset wraps its declarations in an <![INCLUDE[ ... ]]>
		// conditional section. Inside that section it declares a parameter entity
		// whose replacement text is an <!ATTLIST> and then references it. The
		// reference must be expanded (not merely validated and skipped by the
		// blank-skip's handlePEReference) so the default attribute is applied to
		// <r/>. Before the fix, the INCLUDE loop's skipBlanks consumed "%attrs;"
		// without pushing its replacement text, silently dropping the declaration.
		const dtd = `<![INCLUDE[
<!ELEMENT r EMPTY>
<!ENTITY % attrs "<!ATTLIST r x CDATA 'inc'>">
%attrs;
]]>`
		fsys := fstest.MapFS{"inc.dtd": &fstest.MapFile{Data: []byte(dtd)}}

		p := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).DefaultDTDAttributes(true).FS(fsys)
		doc, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err, "PE reference inside an INCLUDE section must parse")
		require.NotNil(t, doc, "document must be returned")

		root := doc.DocumentElement()
		require.NotNil(t, root, "root element must be available")

		val, ok := root.GetAttribute("x")
		require.True(t, ok, "default attribute from PE expanded inside INCLUDE must be present")
		require.Equal(t, "inc", val, "PE inside INCLUDE must apply the default attribute value")
	})

	// the property
	// the blank-run cap MUST NOT break: the external-subset declaration step uses a
	// blank-ONLY skip (skipBlankRun) precisely so a "%pe;" reference that follows
	// whitespace is left for parsePEReference to expand. Consuming it
	// through skipBlanks/handlePEReference would push no replacement text. With a
	// non-trivial (but under-cap) whitespace run before "%pe;", the PE must still
	// expand and its declarations apply — here a general entity declared inside the
	// PE is registered.
	t.Run("expands after whitespace in the external subset", func(t *testing.T) {
		ws := strings.Repeat(" ", 2048)
		fsys := fstest.MapFS{
			dtdSystemID: {Data: []byte(
				`<!ENTITY ctrl "control">` + "\n" +
					`<!ENTITY % pe SYSTEM "pe.ent">` + ws + `%pe;`)},
			peSystemID: {Data: []byte(`<!ENTITY fromPE "loaded-from-external-pe">`)},
		}
		const input = `<?xml version="1.0"?>` + "\n" +
			`<!DOCTYPE r SYSTEM "d.dtd"><r/>`

		doc, err := helium.NewParser().
			BlockXXE(false).
			LoadExternalDTD(true).
			MaxNodeContentSize(4096).
			FS(fsys).
			Parse(t.Context(), []byte(input))
		require.NoError(t, err, "under-cap whitespace before %%pe; must not break parsing")
		require.NotNil(t, doc)

		ent, ok := doc.GetEntity("fromPE")
		require.True(t, ok, "the PE following whitespace must still expand and register its declarations")
		require.Equal(t, "loaded-from-external-pe", string(ent.Content()))
	})

	t.Run("trailing whitespace preserves post-DOCTYPE misc", func(t *testing.T) {
		const input = `<?xml version="1.0"?>
<!DOCTYPE r SYSTEM "d.dtd"><!--after--><?pi go?><r/>`

		// The external DTD ends with trailing whitespace AFTER its last declaration.
		// The shared declaration step's blank-only skip consumes that whitespace and
		// reaches EOF on the pushed external-DTD (floor) cursor. getCursor() would
		// then auto-pop the exhausted floor cursor and return the cursor BELOW it —
		// the MAIN DOCUMENT cursor positioned right after the DOCTYPE — which is not
		// Done. The step would parse the document's post-DOCTYPE "<!--after-->"
		// comment and "<?pi go?>" PI as if they were external-subset markup, dropping
		// them from the parsed document. Inspecting the floor cursor directly (rather
		// than via getCursor) stops at the floor instead, so the misc nodes survive.
		const dtd = "<!ELEMENT r EMPTY>\n"
		fsys := fstest.MapFS{dtdSystemID: &fstest.MapFile{Data: []byte(dtd)}}

		p := helium.NewParser().LoadExternalDTD(true).DefaultDTDAttributes(true).FS(fsys)
		doc, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err, "external DTD with trailing whitespace must parse")
		require.NotNil(t, doc, "document must be returned")

		root := doc.DocumentElement()
		require.NotNil(t, root, "root element must be available")

		var sawComment, sawPI bool
		for n := doc.FirstChild(); n != nil; n = n.NextSibling() {
			switch n.Type() {
			case helium.CommentNode:
				require.Equal(t, "after", string(n.Content()), "post-DOCTYPE comment content must be preserved")
				sawComment = true
			case helium.ProcessingInstructionNode:
				sawPI = true
			}
		}
		require.True(t, sawComment, "post-DOCTYPE comment must not be consumed as external-subset markup")
		require.True(t, sawPI, "post-DOCTYPE PI must not be consumed as external-subset markup")
	})
}

func TestExternalDTDLoad(t *testing.T) {
	t.Parallel()

	// The external subset is loaded when load, default-attributes, or validation is
	// requested; the three intents are independent and the load decision does not
	// depend on the order the setters were called. ValidateDTD(true) must cause the
	// external subset to load even when LoadExternalDTD(false) is also set, and the
	// result must be identical regardless of call order.
	t.Run("load order is independent", func(t *testing.T) {
		validateThenNoLoad := helium.NewParser().
			BlockXXE(false).
			FS(dtdLoadFS()).
			ValidateDTD(true).
			LoadExternalDTD(false)

		noLoadThenValidate := helium.NewParser().
			BlockXXE(false).
			FS(dtdLoadFS()).
			LoadExternalDTD(false).
			ValidateDTD(true)

		doc1, err1 := validateThenNoLoad.Parse(t.Context(), []byte(dtdLoadDoc()))
		require.NoError(t, err1, "validation should succeed once the external subset is loaded")
		require.NotNil(t, doc1)

		doc2, err2 := noLoadThenValidate.Parse(t.Context(), []byte(dtdLoadDoc()))
		require.NoError(t, err2)
		require.NotNil(t, doc2)

		// Both orders resolve the load decision the same way: validation causes the
		// external subset to load, LoadExternalDTD(false) notwithstanding.
		require.NotNil(t, doc1.ExtSubset(), "validation must load the external subset regardless of LoadExternalDTD(false)")
		require.NotNil(t, doc2.ExtSubset(), "call order must not change the load decision")
	})

	// A requested external-subset load that fails to open must surface a non-fatal
	// warning instead of being silently swallowed. The parse stays lenient (no
	// fatal error, document still returned), matching libxml2.
	t.Run("a failed load warns", func(t *testing.T) {
		capture := &warnCaptureSAX{TreeBuilder: helium.NewTreeBuilder()}

		// Empty filesystem: the SYSTEM "sub.dtd" open fails.
		doc, err := helium.NewParser().
			BlockXXE(false).
			FS(fstest.MapFS{}).
			SAXHandler(capture).
			LoadExternalDTD(true).
			Parse(t.Context(), []byte(dtdLoadDoc()))

		require.NoError(t, err, "a failed external-subset load stays non-fatal")
		require.NotNil(t, doc)
		require.NotEmpty(t, capture.warnings, "a requested-but-failed external-subset load must emit a warning")

		var lvl helium.ErrorLeveler
		require.ErrorAs(t, capture.warnings[0], &lvl)
		require.Equal(t, helium.ErrorLevelWarning, lvl.ErrorLevel())
		require.Contains(t, capture.warnings[0].Error(), dtdLoadDocName,
			"the warning must name the external DTD that failed to load")
	})

	// Turning DefaultDTDAttributes back off must clear the load intent it set. A
	// DefaultDTDAttributes(true) followed by DefaultDTDAttributes(false) must leave
	// the parser NOT loading the external subset — the load bit does not get stuck
	// on. DefaultDTDAttributes(true) on its own does load.
	t.Run("toggling default attributes clears the load", func(t *testing.T) {
		toggledOff := helium.NewParser().
			BlockXXE(false).
			FS(dtdLoadFS()).
			DefaultDTDAttributes(true).
			DefaultDTDAttributes(false)

		doc, err := toggledOff.Parse(t.Context(), []byte(dtdLoadDoc()))
		require.NoError(t, err)
		require.NotNil(t, doc)
		require.Nil(t, doc.ExtSubset(), "toggling DefaultDTDAttributes off must not leave the load bit stuck on")

		on := helium.NewParser().
			BlockXXE(false).
			FS(dtdLoadFS()).
			DefaultDTDAttributes(true)

		doc2, err2 := on.Parse(t.Context(), []byte(dtdLoadDoc()))
		require.NoError(t, err2)
		require.NotNil(t, doc2)
		require.NotNil(t, doc2.ExtSubset(), "DefaultDTDAttributes(true) must load the external subset")
	})

	// the not-found branch of external DTD
	// resolution: a SYSTEM id pointing at a non-existent file must not crash.
	t.Run("a missing file", func(t *testing.T) {
		xml := `<?xml version="1.0"?>
<!DOCTYPE root SYSTEM "/nonexistent/path/to.dtd">
<root>content</root>`

		// The document body is still well-formed; the external DTD simply cannot be
		// loaded. Parsing should not panic.
		doc, _ := helium.NewParser().LoadExternalDTD(true).Parse(t.Context(), []byte(xml))
		if doc != nil {
			require.NotNil(t, doc.DocumentElement())
		}
	})

	// a DOCTYPE that declares a PUBLIC
	// external identifier (with both public and system IDs).
	t.Run("a public identifier", func(t *testing.T) {
		const dtd = `<!ELEMENT root (#PCDATA)>
<!ENTITY who "world">`
		path := writeDTD(t, dtd)

		xml := `<?xml version="1.0"?>
<!DOCTYPE root PUBLIC "-//Example//DTD root//EN" "` + path + `">
<root/>`

		doc, err := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).FS(helium.PermissiveFS()).Parse(t.Context(), []byte(xml))
		require.NoError(t, err)
		_, found := doc.GetEntity("who")
		require.True(t, found)
	})

	// notation and external entity
	// declarations resolved from the external subset.
	t.Run("notations and entities", func(t *testing.T) {
		const dtd = `<!ELEMENT root (#PCDATA)>
<!NOTATION gif SYSTEM "viewer.exe">
<!NOTATION png PUBLIC "-//N//EN" "png.exe">
<!ENTITY img SYSTEM "img.gif" NDATA gif>
<!ENTITY ext SYSTEM "data.xml">
<!ENTITY pub PUBLIC "-//P//EN" "pub.xml">`
		path := writeDTD(t, dtd)

		xml := `<?xml version="1.0"?>
<!DOCTYPE root SYSTEM "` + path + `">
<root/>`

		doc, err := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).FS(helium.PermissiveFS()).Parse(t.Context(), []byte(xml))
		require.NoError(t, err)

		require.NotNil(t, doc.ExtSubset(), "external subset must be present")
		// The external general entities are resolvable from the document.
		_, ok := doc.GetEntity("ext")
		require.True(t, ok, "external SYSTEM entity declared in ext subset")
		_, ok = doc.GetEntity("pub")
		require.True(t, ok, "external PUBLIC entity declared in ext subset")
	})

	// INCLUDE/IGNORE conditional
	// sections, which only appear in the external subset.
	t.Run("conditional sections", func(t *testing.T) {
		const dtd = `<!ELEMENT root (#PCDATA)>
<![INCLUDE[
<!ENTITY included "in">
]]>
<![IGNORE[
<!ENTITY ignored "out">
]]>`
		path := writeDTD(t, dtd)

		xml := `<?xml version="1.0"?>
<!DOCTYPE root SYSTEM "` + path + `">
<root/>`

		doc, err := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).FS(helium.PermissiveFS()).Parse(t.Context(), []byte(xml))
		require.NoError(t, err)

		_, found := doc.GetEntity("included")
		require.True(t, found, "entity inside INCLUDE section must be declared")

		_, found = doc.GetEntity("ignored")
		require.False(t, found, "entity inside IGNORE section must be skipped")
	})
}
