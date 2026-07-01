package helium_test

import (
	"bytes"
	"io"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding/charmap"
)

// TestMaxNodeContentSizeEntityValueLiteral guards the internal-DTD entity-value
// literal scanner (parseEntityValueInternal): a giant entity value is an
// indivisible content run that must be bounded by the per-node content cap, both
// when the literal is properly terminated and when it runs unterminated toward
// EOF. Before the fix this scanner had no cap and peeked an ever-growing offset,
// so an internal DTD (which parses by default) could grow memory without bound.
func TestMaxNodeContentSizeEntityValueLiteral(t *testing.T) {
	t.Parallel()

	const limit = 64

	t.Run("terminated over-cap entity value fails closed", func(t *testing.T) {
		t.Parallel()
		body := strings.Repeat("a", 200)
		doc := `<!DOCTYPE r [<!ENTITY e "` + body + `">]><r/>`
		_, err := helium.NewParser().
			MaxNodeContentSize(limit).
			Parse(t.Context(), []byte(doc))
		require.ErrorIs(t, err, helium.ErrNodeContentTooLarge)
	})

	t.Run("unterminated over-cap entity value fails closed (Parse)", func(t *testing.T) {
		t.Parallel()
		// No closing quote: the scanner runs to EOF. The cap must trip before the
		// whole run is buffered, rather than growing unbounded.
		body := strings.Repeat("a", 200)
		doc := `<!DOCTYPE r [<!ENTITY e "` + body
		_, err := helium.NewParser().
			MaxNodeContentSize(limit).
			Parse(t.Context(), []byte(doc))
		require.ErrorIs(t, err, helium.ErrNodeContentTooLarge)
	})

	t.Run("unterminated over-cap entity value fails closed (ParseReader)", func(t *testing.T) {
		t.Parallel()
		body := strings.Repeat("a", 200)
		doc := `<!DOCTYPE r [<!ENTITY e "` + body
		_, err := helium.NewParser().
			MaxNodeContentSize(limit).
			ParseReader(t.Context(), strings.NewReader(doc))
		require.ErrorIs(t, err, helium.ErrNodeContentTooLarge)
	})

	t.Run("within-cap entity value parses fine", func(t *testing.T) {
		t.Parallel()
		doc := `<!DOCTYPE r [<!ENTITY e "` + strings.Repeat("a", 32) + `">]><r>&e;</r>`
		d, err := helium.NewParser().
			MaxNodeContentSize(limit).
			Parse(t.Context(), []byte(doc))
		require.NoError(t, err)
		require.NotNil(t, d)
	})
}

// TestMaxNodeContentSizeExternalIDLiteral guards the SYSTEM/PUBLIC literal
// scanners (parseSystemLiteral/parsePubidLiteral) reached through the DOCTYPE
// external ID. A giant system or public literal must fail closed with the
// per-node content cap rather than buffering unbounded; the generic "URI
// required" message must not mask the resource-limit error.
func TestMaxNodeContentSizeExternalIDLiteral(t *testing.T) {
	t.Parallel()

	const limit = 64

	t.Run("over-cap system literal fails closed", func(t *testing.T) {
		t.Parallel()
		body := strings.Repeat("a", 200)
		doc := `<!DOCTYPE r SYSTEM "` + body + `"><r/>`
		_, err := helium.NewParser().
			MaxNodeContentSize(limit).
			Parse(t.Context(), []byte(doc))
		require.ErrorIs(t, err, helium.ErrNodeContentTooLarge)
	})

	t.Run("over-cap public literal fails closed", func(t *testing.T) {
		t.Parallel()
		body := strings.Repeat("a", 200)
		doc := `<!DOCTYPE r PUBLIC "` + body + `" "sys"><r/>`
		_, err := helium.NewParser().
			MaxNodeContentSize(limit).
			Parse(t.Context(), []byte(doc))
		require.ErrorIs(t, err, helium.ErrNodeContentTooLarge)
	})

	t.Run("within-cap external id parses fine", func(t *testing.T) {
		t.Parallel()
		doc := `<!DOCTYPE r PUBLIC "` + strings.Repeat("a", 16) + `" "` + strings.Repeat("b", 16) + `"><r/>`
		d, err := helium.NewParser().
			MaxNodeContentSize(limit).
			Parse(t.Context(), []byte(doc))
		require.NoError(t, err)
		require.NotNil(t, d)
	})
}

// TestMaxNodeContentSizeExternalEntityDeclLiteral guards the SYSTEM/PUBLIC
// literal scanners reached through an external ENTITY declaration in the
// internal subset (parseEntityDecl -> parseExternalID). A giant system or public
// literal in an external general or parameter entity declaration must fail closed
// with the per-node content cap rather than buffering unbounded; the generic
// "value required" message must not mask the resource-limit error.
func TestMaxNodeContentSizeExternalEntityDeclLiteral(t *testing.T) {
	t.Parallel()

	const limit = 64
	body := strings.Repeat("a", 200)

	cases := []struct {
		name string
		doc  string
	}{
		{
			name: "external general entity SYSTEM over-cap",
			doc:  `<!DOCTYPE r [<!ENTITY e SYSTEM "` + body + `">]><r/>`,
		},
		{
			name: "external parameter entity SYSTEM over-cap",
			doc:  `<!DOCTYPE r [<!ENTITY % e SYSTEM "` + body + `">]><r/>`,
		},
		{
			name: "external parameter entity PUBLIC over-cap system literal",
			doc:  `<!DOCTYPE r [<!ENTITY % e PUBLIC "pub" "` + body + `">]><r/>`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name+" (Parse)", func(t *testing.T) {
			t.Parallel()
			_, err := helium.NewParser().
				MaxNodeContentSize(limit).
				Parse(t.Context(), []byte(tc.doc))
			require.ErrorIs(t, err, helium.ErrNodeContentTooLarge)
		})

		t.Run(tc.name+" (ParseReader)", func(t *testing.T) {
			t.Parallel()
			_, err := helium.NewParser().
				MaxNodeContentSize(limit).
				ParseReader(t.Context(), strings.NewReader(tc.doc))
			require.ErrorIs(t, err, helium.ErrNodeContentTooLarge)
		})
	}
}

// stoppableEBCDICEntityReader serves a finite EBCDIC head ending in an open
// entity-value literal (<!ENTITY e "), then an endless run of an EBCDIC filler
// byte that never reaches EOF, modeling a hostile never-ending internal-DTD
// entity value over the streaming EBCDIC path. The parser must terminate it via
// the per-node content cap WITHOUT buffering the tail.
type stoppableEBCDICEntityReader struct {
	head    []byte
	fill    byte
	pos     int
	stopped atomic.Bool
}

func (r *stoppableEBCDICEntityReader) Stop() { r.stopped.Store(true) }

func (r *stoppableEBCDICEntityReader) Read(p []byte) (int, error) {
	if r.stopped.Load() {
		return 0, io.EOF
	}
	if r.pos < len(r.head) {
		n := copy(p, r.head[r.pos:])
		r.pos += n
		return n, nil
	}
	for i := range p {
		p[i] = r.fill
	}
	return len(p), nil
}

// TestParseReaderEBCDICUnboundedEntityValueBoundedByNodeCap proves the parser.go
// "unbounded EBCDIC streams are bounded by parser caps" claim holds for the
// internal-DTD entity-value literal scanner: an EBCDIC stream whose entity value
// never terminates is bounded by MaxNodeContentSize and fails with
// ErrNodeContentTooLarge, never buffered whole into memory.
func TestParseReaderEBCDICUnboundedEntityValueBoundedByNodeCap(t *testing.T) {
	t.Parallel()

	const decl = `<?xml version="1.0" encoding="IBM037"?><!DOCTYPE r [<!ENTITY e "`
	head, err := charmap.CodePage037.NewEncoder().Bytes([]byte(decl))
	require.NoError(t, err)
	require.Equal(t, []byte{0x4C, 0x6F, 0xA7, 0x94}, head[:4],
		"encoded head must start with the EBCDIC invariant prefix")

	fill, err := charmap.CodePage037.NewEncoder().Bytes([]byte("a"))
	require.NoError(t, err)

	r := &stoppableEBCDICEntityReader{head: head, fill: fill[0]}
	defer r.Stop()

	errCh := make(chan error, 1)
	go func() {
		_, perr := helium.NewParser().MaxNodeContentSize(4096).ParseReader(t.Context(), r)
		errCh <- perr
	}()

	select {
	case perr := <-errCh:
		require.ErrorIs(t, perr, helium.ErrNodeContentTooLarge,
			"an unbounded EBCDIC entity value must be bounded by the per-node content cap")
	case <-time.After(5 * time.Second):
		r.Stop()
		t.Fatal("ParseReader did not terminate an unbounded EBCDIC entity value within the timeout")
	}
}

// TestParseReaderEBCDICEntityValueMatchesParse guards that a finite EBCDIC
// document with a (within-cap) entity value parses identically via ParseReader
// and Parse([]byte), so the bounded literal scanner did not change normal output.
func TestParseReaderEBCDICEntityValueMatchesParse(t *testing.T) {
	t.Parallel()

	const xml = `<?xml version="1.0" encoding="IBM037"?><!DOCTYPE r [<!ENTITY e "hello">]><r>&e;</r>`
	ebcdic, err := charmap.CodePage037.NewEncoder().Bytes([]byte(xml))
	require.NoError(t, err)

	serialize := func(doc *helium.Document) string {
		var buf bytes.Buffer
		require.NoError(t, helium.NewWriter().WriteTo(&buf, doc))
		return buf.String()
	}

	bytesDoc, err := helium.NewParser().Parse(t.Context(), ebcdic)
	require.NoError(t, err)
	want := serialize(bytesDoc)

	readerDoc, err := helium.NewParser().ParseReader(t.Context(), bytes.NewReader(ebcdic))
	require.NoError(t, err)
	require.Equal(t, want, serialize(readerDoc))
}
