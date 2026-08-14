package helium_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lestrrat-go/helium"
	heliumencoding "github.com/lestrrat-go/helium/internal/encoding"
	"github.com/lestrrat-go/helium/sax"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding/charmap"
)

// dataThenErrReader returns its payload together with a non-EOF error on the
// same Read (which io.Reader permits), then reports EOF. It models a reader
// that detects corruption/truncation only after emitting the final bytes,
// e.g. a checksumming or decompressing stream.
type dataThenErrReader struct {
	data []byte
	err  error
	done bool
}

// zeroProgressReader always returns (0, nil) for a non-empty request, never
// advancing and never erroring. A naive fill loop spins on it forever.
type zeroProgressReader struct{}

func (zeroProgressReader) Read(p []byte) (int, error) {
	return 0, nil
}

// blockingReader blocks forever inside Read until its done channel is closed,
// then returns io.EOF. It models a non-context-aware reader (e.g. a slow
// network stream) whose Read cannot be interrupted generically.
type blockingReader struct {
	done    chan struct{}
	entered chan struct{} // closed the first time Read is entered
	once    sync.Once
}

// stoppableEBCDICReader serves a finite EBCDIC head (whose invariant prefix
// triggers EBCDIC detection) on the first Reads, then an endless run of EBCDIC
// space bytes — an unterminated whitespace character-data run inside <root> —
// that never reaches EOF, until Stop is called (after which Read returns
// io.EOF). It models a hostile, never-ending stream. The parser must terminate
// it via its incremental per-node content cap WITHOUT buffering the tail; Stop
// plus a test timeout guarantee that even a regression cannot hang or OOM the
// test process.
type stoppableEBCDICReader struct {
	head    []byte
	pos     int
	stopped atomic.Bool
}

// infiniteBlankReader streams a fixed head followed by an unbounded run of
// ASCII spaces and never reaches EOF. It models a prolog / inter-root
// whitespace DoS: input that keeps the parser in skipBlanks forever. The blank
// run is OUTSIDE any element, so the char-data node-content cap never applies —
// only the blank-run guard in skipBlanks bounds it.
type infiniteBlankReader struct {
	head    []byte
	pos     int
	stopped atomic.Bool
}

// prefixThenZeroThenRestReader delivers a leading prefix, then a single
// transient (0, nil) read (which io.Reader explicitly permits while a slow
// producer waits for more input), then the remaining bytes, then io.EOF. It
// models a stream that splits the EBCDIC sniff prefix across a zero-progress
// read.
type prefixThenZeroThenRestReader struct {
	data        []byte
	prefixLen   int
	pos         int
	emittedZero bool
}

// ebcdicSlowTailReader returns the EBCDIC invariant prefix on its first Read (so
// detection succeeds), signals when its tail is first read, then drips the tail
// one byte at a time. The first tail byte is served immediately so the parser
// regains control (and re-checks ctx); every later tail Read blocks until the
// reader is cancelled, modeling a stalled tail whose remaining bytes never
// arrive. If ParseReader honored ctx between reads it returns before consuming
// the whole tail; if it drained eagerly it would block forever.
type ebcdicSlowTailReader struct {
	prefix    []byte
	gate      chan struct{} // closed to unblock later tail reads (test cleanup only)
	entered   chan struct{} // closed the first time a tail byte is served
	cancelled chan struct{} // closed by the test once ctx has been cancelled
	once      sync.Once
	served    int // number of tail bytes already delivered
}

// genCharDataReader lazily produces "<root>" + n copies of fill + "</root>"
// without ever holding the whole payload in memory, so the parser's own
// character-data buffering is the only large allocation under test. It records
// the cumulative number of payload bytes it has handed out in nread, which the
// memory-bounds test samples at the first SAX callback to prove the parser
// streams, draining no more than a bounded prefix before delivering anything.
type genCharDataReader struct {
	prefix []byte
	fill   byte
	remain int
	suffix []byte
	nread  int
}

// blockUntilCancelledBlankReader serves a fixed head, then (on the first read
// past the head) signals it has reached the trailing whitespace run and blocks
// until the test cancels the context, after which it streams ASCII spaces
// forever. It drives a cancellation first observable while the parser is
// scanning a whitespace run in the XML declaration or DTD subset.
type blockUntilCancelledBlankReader struct {
	head      []byte
	pos       int
	entered   chan struct{}
	cancelled chan struct{}
	once      sync.Once
}

// headThenReadErrReader serves head bytes once, then fails every subsequent
// Read with err. It models the push/streaming stream whose blocking Read
// returns context.Canceled when cancellation unblocks a pending wait: the
// ByteCursor records that as a sticky Err() while PeekAt reports 0, the same 0
// a genuine non-blank byte / clean EOF yields. ctx is never cancelled here, so
// the ONLY signal of the failure is the cursor's sticky read error.
type headThenReadErrReader struct {
	head []byte
	pos  int
	err  error
}

// largeDoc builds a document with many sibling elements so the content loop
// iterates enough times to observe a mid-parse cancellation.
func largeDoc() []byte {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0"?><root>`)
	for range 200000 {
		b.WriteString(`<a>x</a>`)
	}
	b.WriteString(`</root>`)
	return []byte(b.String())
}

// cancellingSAX embeds the default tree builder and cancels the parse context
// after a fixed number of StartElementNS callbacks. This makes cancellation
// deterministic: the parser is guaranteed to have entered the content loop and
// processed a known number of elements before the context is cancelled, so the
// next loop iteration observes the cancellation regardless of machine speed. It
// also records every SAX Error callback so a test can assert that a clean
// cancellation surfaces no error to the SAX handler.
type cancellingSAX struct {
	*helium.TreeBuilder
	cancel   context.CancelFunc
	cancelAt int
	mu       sync.Mutex
	starts   int
	errors   []error
}

func (s *cancellingSAX) StartElementNS(ctx context.Context, localname, prefix, uri string, namespaces []sax.Namespace, attrs []sax.Attribute) error {
	s.mu.Lock()
	s.starts++
	fire := s.cancel != nil && s.starts == s.cancelAt
	s.mu.Unlock()
	if fire {
		s.cancel()
	}
	return s.TreeBuilder.StartElementNS(ctx, localname, prefix, uri, namespaces, attrs)
}

func (s *cancellingSAX) Error(ctx context.Context, err error) error {
	s.mu.Lock()
	s.errors = append(s.errors, err)
	s.mu.Unlock()
	return s.TreeBuilder.Error(ctx, err)
}

func (s *cancellingSAX) recorded() []error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]error(nil), s.errors...)
}

// malformedRecoverableDoc builds a large document that becomes malformed right
// after the root start tag: the content is a huge run of plain text with no
// closing tag. With RecoverOnError the parser fails on the unterminated content
// and then sits in its skip-to-recover-point loop scanning the long tail. This
// keeps recovery active long enough for a concurrently-cancelled context to be
// observed inside the recovery/skip path.
func malformedRecoverableDoc() []byte {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0"?><root>`)
	// A stray '<' with no following name forces a parse error; the long run of
	// text after it must be skipped during recovery (skipToRecoverPoint), which
	// is where the cancellation needs to be observed.
	b.WriteString(`<`)
	b.WriteString(strings.Repeat("x", 10_000_000))
	return []byte(b.String())
}

func TestParseReader(t *testing.T) {
	t.Parallel()

	// is the convergence regression: a
	// reader that returns (n>0, non-EOF err) in a single Read must have its bytes
	// PARSED first and the error surfaced only AFTER they drain — on BOTH the
	// non-EBCDIC streaming path and the EBCDIC byte-slice path. Bytes returned
	// alongside a non-EOF error are never discarded.
	t.Run("bytes then a surfaced error", func(t *testing.T) {
		wantErr := errors.New("checksum mismatch")

		ebcdic := func(s string) []byte {
			b, err := charmap.CodePage037.NewEncoder().Bytes([]byte(s))
			require.NoError(t, err)
			require.Equal(t, []byte{0x4C, 0x6F, 0xA7, 0x94}, b[:4],
				"encoded bytes must start with the EBCDIC invariant prefix")
			return b
		}

		tests := []struct {
			name string
			data []byte
		}{
			{
				name: "non-EBCDIC",
				data: []byte(`<root><child/></root>`),
			},
			{
				name: "EBCDIC",
				data: ebcdic(`<?xml version="1.0" encoding="IBM037"?><root><child/></root>`),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var seen []string
				p := helium.NewParser().SAXHandler(startElementRecorder(&seen))

				_, err := p.ParseReader(t.Context(), &dataThenErrReader{
					data: tt.data,
					err:  wantErr,
				})
				require.ErrorIs(t, err, wantErr,
					"a non-EOF read error returned alongside the bytes must be surfaced")
				require.Equal(t, []string{"root", "child"}, seen,
					"the bytes returned alongside the error must be parsed before the error surfaces")
			})
		}
	})

	t.Run("an error returned with data", func(t *testing.T) {
		wantErr := errors.New("checksum mismatch")
		p := helium.NewParser()

		_, err := p.ParseReader(t.Context(), &dataThenErrReader{
			data: []byte("<root/>"),
			err:  wantErr,
		})
		require.ErrorIs(t, err, wantErr, "a reader error returned alongside the final bytes must not be swallowed")
	})

	// pins the cancellation
	// contract for a read failure that lands RIGHT AFTER "<?xml", before the
	// declaration's required trailing blank has been read. Because the sixth byte
	// is never delivered, looksLikeXMLDecl cannot confirm the declaration and the
	// parser falls through to treating "<?xml" as a processing instruction, whose
	// reserved "xml" target then synthesizes "XML declaration allowed only at the
	// start of the document". That synthesized syntax error must NOT mask the
	// underlying read failure: a parse whose stream fails (a push-stream Read
	// returning context.Canceled on cancellation) must surface that error, never a
	// malformed-document diagnostic, and must return no partial document.
	//
	// ctx is context.Background() (never cancelled) on purpose: the only signal of
	// the failure is the cursor's sticky Err(), exactly as in the push/streaming
	// path where the stream Read returns the context error.
	t.Run("a read error in the XML declaration is not masked", func(t *testing.T) {
		cases := map[string]string{
			// Read fails immediately after "<?xml" with no trailing blank read, so
			// looksLikeXMLDecl is false and "<?xml" is reparsed as a reserved PI.
			"no blank after <?xml": "<?xml",
			// A blank was read but the read fails before the version pseudo-attribute,
			// so the declaration parse begins and then stalls on the failed read.
			"blank then read error": "<?xml ",
		}

		for name, head := range cases {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				r := &headThenReadErrReader{head: []byte(head), err: context.Canceled}
				doc, err := helium.NewParser().ParseReader(context.Background(), r)
				require.ErrorIs(t, err, context.Canceled,
					"a read failure in the XML declaration (%s) must surface as context.Canceled, not a synthesized syntax error", name)
				require.Nil(t, doc, "a failed parse must not return a partial document")
			})
		}
	})

	t.Run("a zero-progress reader does not hang", func(t *testing.T) {
		p := helium.NewParser()

		done := make(chan error, 1)
		go func() {
			_, err := p.ParseReader(t.Context(), zeroProgressReader{})
			done <- err
		}()

		select {
		case err := <-done:
			require.ErrorIs(t, err, io.ErrNoProgress, "a zero-progress reader must fail with io.ErrNoProgress, not be accepted")
		case <-time.After(5 * time.Second):
			t.Fatal("ParseReader hung on a zero-progress reader instead of failing fast")
		}
	})

	// the prolog/inter-root
	// whitespace DoS that the per-node content cap does NOT cover: an unbounded run
	// of whitespace BEFORE the root element (or after it) is consumed by skipBlanks,
	// which formerly peeked an ever-growing offset and grew the cursor buffer
	// without bound. The blank-run guard now caps it and fails the parse instead of
	// hanging or exhausting memory. A goroutine + timeout + Stop guarantees a
	// regression cannot hang or OOM the test process.
	t.Run("unbounded prolog whitespace is bounded", func(t *testing.T) {
		cases := map[string]string{
			// Infinite whitespace between the XML declaration and the root element.
			"prolog before root": `<?xml version="1.0"?>`,
			// Infinite whitespace in the epilogue, after a complete root element.
			"epilogue after root": `<?xml version="1.0"?><root/>`,
		}

		for name, head := range cases {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				r := &infiniteBlankReader{head: []byte(head)}
				defer r.Stop()

				errCh := make(chan error, 1)
				go func() {
					// A small cap keeps the test fast: the blank-run guard must fire
					// well before the infinite tail could exhaust memory.
					_, perr := helium.NewParser().MaxNodeContentSize(4096).ParseReader(t.Context(), r)
					errCh <- perr
				}()

				select {
				case perr := <-errCh:
					require.ErrorIs(t, perr, helium.ErrNodeContentTooLarge,
						"unbounded %s whitespace must be bounded by the blank-run guard", name)
				case <-time.After(5 * time.Second):
					r.Stop() // unblock the parser so the leaked goroutine can exit
					t.Fatalf("ParseReader did not terminate unbounded %s whitespace within the timeout", name)
				}
			})
		}
	})
}

func TestParseReaderEBCDIC(t *testing.T) {
	t.Parallel()

	// confirms the streaming EBCDIC reader
	// path parses a normal, small EBCDIC document identically to Parse([]byte).
	t.Run("a small document under the cap", func(t *testing.T) {
		const xml = `<?xml version="1.0" encoding="IBM037"?><root><child>hi</child></root>`
		ebcdic, err := charmap.CodePage037.NewEncoder().Bytes([]byte(xml))
		require.NoError(t, err)
		require.Equal(t, []byte{0x4C, 0x6F, 0xA7, 0x94}, ebcdic[:4],
			"encoded bytes must start with the EBCDIC invariant prefix")

		serialize := func(doc *helium.Document) string {
			var buf bytes.Buffer
			require.NoError(t, helium.NewWriter().WriteTo(&buf, doc))
			return buf.String()
		}

		bytesDoc, err := helium.NewParser().Parse(t.Context(), ebcdic)
		require.NoError(t, err, "Parse([]byte) must handle EBCDIC")
		want := serialize(bytesDoc)

		readerDoc, err := helium.NewParser().ParseReader(t.Context(), bytes.NewReader(ebcdic))
		require.NoError(t, err, "a small EBCDIC doc must parse under the default ingestion cap")
		require.Equal(t, want, serialize(readerDoc),
			"ParseReader output must match Parse([]byte) for a small EBCDIC doc")
	})

	// the key property of the
	// streaming EBCDIC path: a finite document whose TOTAL size exceeds
	// MaxNodeContentSize but whose every individual node is well under the cap must
	// parse successfully and identically to Parse([]byte). A total-input cap (the
	// earlier, wrong approach) would have wrongly rejected this document and
	// diverged from Parse([]byte); the per-node cap accepts it.
	t.Run("a large finite document under the node cap", func(t *testing.T) {
		var sb strings.Builder
		sb.WriteString(`<?xml version="1.0" encoding="IBM037"?><root>`)
		for range 500 {
			sb.WriteString(`<c>x</c>`)
		}
		sb.WriteString(`</root>`)
		xml := sb.String()

		ebcdic, err := charmap.CodePage037.NewEncoder().Bytes([]byte(xml))
		require.NoError(t, err)
		require.Greater(t, len(ebcdic), 1024,
			"the document must be larger than the per-node cap used below")
		require.Equal(t, []byte{0x4C, 0x6F, 0xA7, 0x94}, ebcdic[:4],
			"encoded bytes must start with the EBCDIC invariant prefix")

		serialize := func(doc *helium.Document) string {
			var buf bytes.Buffer
			require.NoError(t, helium.NewWriter().WriteTo(&buf, doc))
			return buf.String()
		}

		// A cap smaller than the total document but larger than any single node.
		bytesDoc, err := helium.NewParser().MaxNodeContentSize(1024).Parse(t.Context(), ebcdic)
		require.NoError(t, err, "Parse([]byte) must accept a large finite EBCDIC doc with small nodes")
		want := serialize(bytesDoc)

		readerDoc, err := helium.NewParser().MaxNodeContentSize(1024).ParseReader(t.Context(), bytes.NewReader(ebcdic))
		require.NoError(t, err,
			"ParseReader must accept a large finite EBCDIC doc whose nodes are under the per-node cap")
		require.Equal(t, want, serialize(readerDoc),
			"ParseReader output must match Parse([]byte) for a large finite EBCDIC doc")
	})

	// the streaming
	// EBCDIC reader path against unbounded memory growth: EBCDIC now streams through
	// the normal cursor pipeline, so a hostile never-ending stream is bounded by the
	// parser's incremental per-node content cap (the single whitespace run inside
	// <root> exceeds MaxNodeContentSize) and fails with ErrNodeContentTooLarge —
	// never buffered whole into memory. The reader runs in a goroutine with a
	// timeout and a Stop so a regression that reintroduced whole-stream buffering
	// cannot hang or OOM the test.
	t.Run("an unbounded stream is bounded by the node cap", func(t *testing.T) {
		const decl = `<?xml version="1.0" encoding="IBM037"?><root>`
		head, err := charmap.CodePage037.NewEncoder().Bytes([]byte(decl))
		require.NoError(t, err)
		require.Equal(t, []byte{0x4C, 0x6F, 0xA7, 0x94}, head[:4],
			"encoded head must start with the EBCDIC invariant prefix")

		r := &stoppableEBCDICReader{head: head}
		defer r.Stop()

		errCh := make(chan error, 1)
		go func() {
			// A small cap keeps the test fast: the per-node cap must fire well before
			// the infinite tail could exhaust memory.
			_, perr := helium.NewParser().MaxNodeContentSize(4096).ParseReader(t.Context(), r)
			errCh <- perr
		}()

		select {
		case perr := <-errCh:
			require.ErrorIs(t, perr, helium.ErrNodeContentTooLarge,
				"an unbounded EBCDIC stream must be bounded by the per-node content cap")
		case <-time.After(5 * time.Second):
			r.Stop() // unblock the parser so the leaked goroutine can exit
			t.Fatal("ParseReader did not terminate an unbounded EBCDIC stream within the timeout")
		}
	})

	// the EBCDIC sniff against a
	// reader that returns its entire payload together with io.EOF in a single Read
	// (which io.Reader explicitly permits). EBCDIC decoding requires the full raw
	// input up front, so detection must happen even when the head read ends with
	// io.EOF; otherwise the streaming path resets the cursor from a nil rawInput and
	// loses the document.
	t.Run("data with EOF in the first read", func(t *testing.T) {
		const xml = `<?xml version="1.0" encoding="IBM037"?><root><child>hi</child></root>`
		ebcdic, err := charmap.CodePage037.NewEncoder().Bytes([]byte(xml))
		require.NoError(t, err)
		require.Equal(t, []byte{0x4C, 0x6F, 0xA7, 0x94}, ebcdic[:4],
			"encoded bytes must start with the EBCDIC invariant prefix")

		serialize := func(doc *helium.Document) string {
			var buf bytes.Buffer
			require.NoError(t, helium.NewWriter().WriteTo(&buf, doc))
			return buf.String()
		}

		bytesDoc, err := helium.NewParser().Parse(t.Context(), ebcdic)
		require.NoError(t, err, "Parse([]byte) must handle EBCDIC")
		want := serialize(bytesDoc)

		// dataThenErrReader with err == io.EOF returns all bytes plus io.EOF in the
		// FIRST Read, exactly the case that previously fell through to the streaming
		// path and produced a parse error.
		r := &dataThenErrReader{data: ebcdic, err: io.EOF}
		doc, err := helium.NewParser().ParseReader(t.Context(), r)
		require.NoError(t, err, "ParseReader must parse EBCDIC delivered with io.EOF in the first read")
		require.Equal(t, want, serialize(doc),
			"ParseReader output must match Parse([]byte) when EBCDIC arrives with io.EOF in one read")
	})

	// the EBCDIC
	// sniff-extension loop against a transient (0, nil) read that splits the sniff
	// prefix before the encoding declaration has been buffered. A non-IBM037 EBCDIC
	// variant (CP1141) is declared and its content uses byte 0x4A, which decodes to
	// U+00C4 (Ä) under CP1141 but to U+00A2 (¢) under CP037. If a (0, nil) read
	// truncated the sniff prefix, ExtractEBCDICEncoding would miss the declaration,
	// the parser would default to IBM-037, and the text would wrongly decode to ¢.
	// The bounded zero-progress retry must keep reading so CP1141 is detected and
	// the document parses identically to Parse([]byte).
	t.Run("a non-IBM037 codepage split by a zero-progress read", func(t *testing.T) {
		const xml = `<?xml version="1.0" encoding="IBM1141"?><root>Ä</root>`
		cp1141 := heliumencoding.Load("ibm1141")
		require.NotNil(t, cp1141, "internal encoding registry must know CP1141")
		ebcdic, err := cp1141.NewEncoder().Bytes([]byte(xml))
		require.NoError(t, err)
		require.Equal(t, []byte{0x4C, 0x6F, 0xA7, 0x94}, ebcdic[:4],
			"encoded bytes must start with the EBCDIC invariant prefix")

		serialize := func(doc *helium.Document) string {
			var buf bytes.Buffer
			require.NoError(t, helium.NewWriter().WriteTo(&buf, doc))
			return buf.String()
		}

		bytesDoc, err := helium.NewParser().Parse(t.Context(), ebcdic)
		require.NoError(t, err, "Parse([]byte) must handle CP1141 EBCDIC")
		want := serialize(bytesDoc)
		// The writer re-encodes to the document's declared EBCDIC encoding, so assert
		// the decoded DOM text directly: CP1141 byte 0x4A is Ä (U+00C4); the IBM-037
		// default would have decoded it to ¢ (U+00A2).
		require.Equal(t, "Ä", string(bytesDoc.DocumentElement().Content()),
			"CP1141 content byte 0x4A must decode to Ä, not the IBM-037 default (¢)")

		// Split the stream right after the invariant prefix so the (0, nil) read lands
		// inside the sniff-extension loop, before the encoding declaration is buffered.
		r := &prefixThenZeroThenRestReader{data: ebcdic, prefixLen: len(ebcdic[:4])}
		doc, err := helium.NewParser().ParseReader(t.Context(), r)
		require.NoError(t, err,
			"ParseReader must parse CP1141 EBCDIC when a transient (0, nil) read splits the sniff prefix")
		require.Equal(t, want, serialize(doc),
			"ParseReader must detect CP1141 (not default to IBM-037) across a zero-progress read")
	})

	// confirms the
	// inputSize-vs-consumed fix did NOT reopen the entity-amplification DoS over the
	// streaming EBCDIC reader path. The fix divides sizeentcopy by the real
	// consumed-byte count so a large entity referenced ONCE passes; an actual attack
	// (a modestly sized entity referenced MANY times) expands far beyond the
	// amplification factor of the real document size and must STILL be rejected,
	// exactly as Parse([]byte) rejects it. The per-node cap is disabled so the
	// failure is attributable to the amplification guard, not the node-content cap.
	t.Run("an amplification attack is still rejected", func(t *testing.T) {
		// ~250 KB entity referenced 30 times: the document is ~250 KB on disk but
		// expands to ~7.5 MB, well past 5x the real consumed size, so the
		// amplification-ratio guard must fire (and not the 1 GB hard ceiling).
		body := strings.Repeat("A", 250_000)
		refs := strings.Repeat("&a;", 30)
		xml := fmt.Sprintf(`<?xml version="1.0" encoding="IBM037"?><!DOCTYPE root [<!ENTITY a "%s">]><root>%s</root>`, body, refs)

		ebcdic, err := charmap.CodePage037.NewEncoder().Bytes([]byte(xml))
		require.NoError(t, err)
		require.Equal(t, []byte{0x4C, 0x6F, 0xA7, 0x94}, ebcdic[:4],
			"encoded bytes must start with the EBCDIC invariant prefix")

		// Parse([]byte) baseline: the attack must be rejected.
		_, err = helium.NewParser().SubstituteEntities(true).MaxNodeContentSize(-1).Parse(t.Context(), ebcdic)
		require.Error(t, err, "Parse([]byte) must reject an EBCDIC entity-amplification attack")
		require.Contains(t, err.Error(), "amplification",
			"Parse([]byte) must reject via the amplification guard")

		// ParseReader (unknown source size) must STILL reject it: the consumed-byte
		// divisor reflects the real (small) document, so the ratio guard fires.
		_, err = helium.NewParser().SubstituteEntities(true).MaxNodeContentSize(-1).ParseReader(t.Context(), bytes.NewReader(ebcdic))
		require.Error(t, err, "ParseReader must reject an EBCDIC entity-amplification attack")
		require.Contains(t, err.Error(), "amplification",
			"ParseReader must reject via the amplification guard, not silently accept the DoS")
	})

	// against a
	// regression where the streaming EBCDIC path left inputSize seeded from the
	// bounded sniff prefix (~256/512 bytes) in place of the real document size. A
	// large internal entity referenced exactly once is legitimate (no
	// amplification), and Parse([]byte) accepts it because inputSize is the full
	// slice length. With only the prefix length as the divisor, the
	// amplification-ratio guard would falsely reject the same document over
	// ParseReader. Tracking the bytes consumed from the stream fixes it.
	t.Run("a large entity is not falsely amplified", func(t *testing.T) {
		// Entity content just over the 1 MiB ratio-check baseline, referenced once.
		bigContent := strings.Repeat("A", 1_500_000)
		xml := fmt.Sprintf(`<?xml version="1.0" encoding="IBM037"?><!DOCTYPE root [<!ENTITY big "%s">]><root>&big;</root>`, bigContent)

		ebcdic, err := charmap.CodePage037.NewEncoder().Bytes([]byte(xml))
		require.NoError(t, err)
		require.Equal(t, []byte{0x4C, 0x6F, 0xA7, 0x94}, ebcdic[:4],
			"encoded bytes must start with the EBCDIC invariant prefix")

		serialize := func(doc *helium.Document) string {
			var buf bytes.Buffer
			require.NoError(t, helium.NewWriter().WriteTo(&buf, doc))
			return buf.String()
		}

		// Baseline: Parse([]byte) accepts it (inputSize == len(ebcdic)). Use a large
		// per-node cap so the 1.5 MiB text node is not what trips — this test is about
		// the amplification guard, not the node-content cap.
		bytesDoc, err := helium.NewParser().SubstituteEntities(true).MaxNodeContentSize(-1).Parse(t.Context(), ebcdic)
		require.NoError(t, err, "Parse([]byte) must accept a large EBCDIC entity referenced once")
		want := serialize(bytesDoc)

		// ParseReader (unknown source size) must match: the amplification guard must
		// use the real consumed-byte count, not the sniff-prefix length.
		readerDoc, err := helium.NewParser().SubstituteEntities(true).MaxNodeContentSize(-1).ParseReader(t.Context(), bytes.NewReader(ebcdic))
		require.NoError(t, err,
			"ParseReader must accept the same large-entity EBCDIC document as Parse([]byte)")
		require.Equal(t, want, serialize(readerDoc),
			"ParseReader output must match Parse([]byte) for a large-entity EBCDIC doc")
	})

	// the nested
	// entity sub-parse path. When a large entity is reached INDIRECTLY through
	// another entity's replacement text (&wrap; -> &big;), the inner &big; expansion
	// runs in a nested parser context. That nested context copied inputSize from the
	// parent (the bounded EBCDIC sniff prefix) but, before this fix, did NOT carry
	// the live ebcdicConsumed byte-counter, so the amplification-ratio guard divided
	// by the prefix length and falsely rejected a document Parse([]byte) accepts.
	// Propagating ebcdicConsumed through inheritNestedParserState fixes it.
	t.Run("a nested large entity is not falsely amplified", func(t *testing.T) {
		// Entity content just over the 1 MiB ratio-check baseline, referenced once
		// through a wrapping entity so the expansion happens inside a nested context.
		bigContent := strings.Repeat("A", 1_500_000)
		xml := fmt.Sprintf(`<?xml version="1.0" encoding="IBM037"?><!DOCTYPE root [<!ENTITY big "%s"><!ENTITY wrap "&big;">]><root>&wrap;</root>`, bigContent)

		ebcdic, err := charmap.CodePage037.NewEncoder().Bytes([]byte(xml))
		require.NoError(t, err)
		require.Equal(t, []byte{0x4C, 0x6F, 0xA7, 0x94}, ebcdic[:4],
			"encoded bytes must start with the EBCDIC invariant prefix")

		serialize := func(doc *helium.Document) string {
			var buf bytes.Buffer
			require.NoError(t, helium.NewWriter().WriteTo(&buf, doc))
			return buf.String()
		}

		// Baseline: Parse([]byte) accepts it (inputSize == len(ebcdic)). Disable the
		// per-node cap so the 1.5 MiB text node is not what trips — this test is about
		// the amplification guard inside the nested entity context.
		bytesDoc, err := helium.NewParser().SubstituteEntities(true).MaxNodeContentSize(-1).Parse(t.Context(), ebcdic)
		require.NoError(t, err, "Parse([]byte) must accept a nested large EBCDIC entity referenced once")
		want := serialize(bytesDoc)

		// ParseReader (unknown source size) must match: the nested context's
		// amplification guard must use the real consumed-byte count, not the
		// sniff-prefix length.
		readerDoc, err := helium.NewParser().SubstituteEntities(true).MaxNodeContentSize(-1).ParseReader(t.Context(), bytes.NewReader(ebcdic))
		require.NoError(t, err,
			"ParseReader must accept the same nested large-entity EBCDIC document as Parse([]byte)")
		require.Equal(t, want, serialize(readerDoc),
			"ParseReader output must match Parse([]byte) for a nested large-entity EBCDIC doc")
	})

	// EBCDIC encoding parity across entry
	// points: an EBCDIC-encoded document must parse identically whether read via
	// ParseFile, ParseReader, or Parse([]byte). EBCDIC detection/decode relies on
	// the original raw bytes, so the reader-based paths must buffer the input.
	t.Run("matches Parse for a file", func(t *testing.T) {
		const xml = `<?xml version="1.0" encoding="IBM037"?><root><child>hi</child></root>`
		ebcdic, err := charmap.CodePage037.NewEncoder().Bytes([]byte(xml))
		require.NoError(t, err)
		// Sanity: the encoded bytes must carry the EBCDIC invariant prefix that
		// triggers detection (otherwise the test would not exercise the path).
		require.Equal(t, []byte{0x4C, 0x6F, 0xA7, 0x94}, ebcdic[:4],
			"encoded bytes must start with the EBCDIC invariant prefix")

		serialize := func(doc *helium.Document) string {
			var buf bytes.Buffer
			require.NoError(t, helium.NewWriter().WriteTo(&buf, doc))
			return buf.String()
		}

		bytesDoc, err := helium.NewParser().Parse(t.Context(), ebcdic)
		require.NoError(t, err, "Parse([]byte) must handle EBCDIC")
		want := serialize(bytesDoc)

		readerDoc, err := helium.NewParser().ParseReader(t.Context(), bytes.NewReader(ebcdic))
		require.NoError(t, err, "ParseReader must handle EBCDIC")
		require.Equal(t, want, serialize(readerDoc),
			"ParseReader output must match Parse([]byte) for EBCDIC")

		dir := t.TempDir()
		path := filepath.Join(dir, "doc.xml")
		require.NoError(t, os.WriteFile(path, ebcdic, 0o600))
		fileDoc, err := helium.NewParser().ParseFile(t.Context(), path)
		require.NoError(t, err, "ParseFile must handle EBCDIC")
		require.Equal(t, want, serialize(fileDoc),
			"ParseFile output must match Parse([]byte) for EBCDIC")
	})
}

func TestParseReaderCancel(t *testing.T) {
	t.Parallel()

	// the cancellation contract
	// in the XML DECLARATION and DTD whitespace positions (not only prolog /
	// epilogue): a context cancelled while the parser is scanning whitespace there
	// must surface as context.Canceled with no partial document, never as a syntax
	// error. The blank-skip helpers only return a bool, so without the central
	// preference for the sticky blank-run error in errorAtLevel a follow-on syntax
	// error (e.g. "blank needed after '<?xml'") would mask the cancellation.
	t.Run("in declaration and DTD whitespace", func(t *testing.T) {
		cases := map[string]string{
			"xml declaration whitespace": "<?xml",
			"dtd subset whitespace":      `<?xml version="1.0"?><!DOCTYPE root [`,
		}

		for name, head := range cases {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				r := &blockUntilCancelledBlankReader{
					head:      []byte(head),
					entered:   make(chan struct{}),
					cancelled: make(chan struct{}),
				}

				ctx, cancel := context.WithCancel(context.Background())

				type result struct {
					doc *helium.Document
					err error
				}
				resCh := make(chan result, 1)
				go func() {
					// A generous cap so the over-cap guard does not fire first; the
					// cancellation must win.
					doc, err := helium.NewParser().MaxNodeContentSize(1<<20).ParseReader(ctx, r)
					resCh <- result{doc, err}
				}()

				select {
				case <-r.entered:
				case <-time.After(2 * time.Second):
					close(r.cancelled) // unblock the reader so the goroutine can exit
					t.Fatalf("parser did not reach the %s run", name)
				}
				cancel()
				close(r.cancelled)

				select {
				case res := <-resCh:
					require.ErrorIs(t, res.err, context.Canceled,
						"cancellation while scanning %s must surface as context.Canceled", name)
					require.Nil(t, res.doc, "a cancelled parse must not return a partial document")
				case <-time.After(2 * time.Second):
					t.Fatalf("ParseReader did not return after cancellation in %s", name)
				}
			})
		}
	})

	// the "cancelled before any
	// blocking read" contract: when the context is already cancelled, ParseReader
	// must return the context error promptly WITHOUT ever entering the underlying
	// reader's Read (the EBCDIC sniff must check ctx first).
	t.Run("cancelled up front does not block", func(t *testing.T) {
		r := &blockingReader{done: make(chan struct{}), entered: make(chan struct{})}
		// Never unblock the reader: if ParseReader touches it, the test deadlocks
		// and is caught by the timeout below.
		defer close(r.done)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // already cancelled before the call

		type result struct {
			doc *helium.Document
			err error
		}
		resCh := make(chan result, 1)
		go func() {
			doc, err := helium.NewParser().ParseReader(ctx, r)
			resCh <- result{doc, err}
		}()

		select {
		case res := <-resCh:
			require.ErrorIs(t, res.err, context.Canceled,
				"a context cancelled before any read must surface as context.Canceled")
			require.Nil(t, res.doc, "a cancelled parse must not return a document")
		case <-time.After(2 * time.Second):
			t.Fatal("ParseReader blocked on a non-context-aware reader despite an already-cancelled context")
		}

		select {
		case <-r.entered:
			t.Fatal("ParseReader read from the underlying reader despite an already-cancelled context")
		default:
		}
	})

	// the ctx-cancellation
	// contract on the EBCDIC tail-drain path: once the EBCDIC prefix is detected,
	// the remainder of the stream must be read through a loop that re-checks ctx
	// BEFORE each Read. When the context is cancelled after the prefix read and the
	// tail stalls, ParseReader must return context.Canceled promptly WITHOUT
	// blocking on the unread tail.
	t.Run("a cancelled EBCDIC tail is not drained", func(t *testing.T) {
		r := &ebcdicSlowTailReader{
			prefix:    []byte{0x4C, 0x6F, 0xA7, 0x94},
			gate:      make(chan struct{}),
			entered:   make(chan struct{}),
			cancelled: make(chan struct{}),
		}
		// Never unblock the stalled tail: if ParseReader drains past the ctx check,
		// the test deadlocks and is caught by the timeout below.
		defer close(r.gate)

		ctx, cancel := context.WithCancel(context.Background())

		type result struct {
			doc *helium.Document
			err error
		}
		resCh := make(chan result, 1)
		go func() {
			doc, err := helium.NewParser().ParseReader(ctx, r)
			resCh <- result{doc, err}
		}()

		// Wait until the parser has entered the tail-drain loop (prefix consumed),
		// then cancel: this exercises a cancellation observed after the prefix read.
		select {
		case <-r.entered:
		case <-time.After(2 * time.Second):
			t.Fatal("ParseReader did not begin draining the EBCDIC tail")
		}
		cancel()
		close(r.cancelled)

		select {
		case res := <-resCh:
			require.ErrorIs(t, res.err, context.Canceled,
				"a context cancelled while draining the EBCDIC tail must surface as context.Canceled")
			require.Nil(t, res.doc, "a cancelled parse must not return a document")
		case <-time.After(2 * time.Second):
			t.Fatal("ParseReader blocked on the EBCDIC tail despite a cancelled context")
		}
	})
}

func TestParseContextCancel(t *testing.T) {
	t.Parallel()

	// a context cancelled before
	// Parse runs aborts immediately with the context error instead of doing work.
	t.Run("cancelled up front", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		_, err := helium.NewParser().Parse(ctx, []byte(`<?xml version="1.0"?><root><a/></root>`))
		require.ErrorIs(t, err, context.Canceled, "Parse must return the context error")
	})

	// cancelling the context
	// while Parse is running aborts promptly with the context error, well before
	// it would run to completion. Cancellation is triggered deterministically from a SAX
	// handler after a known number of elements have been parsed.
	t.Run("cancelled during the parse", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		handler := &cancellingSAX{TreeBuilder: helium.NewTreeBuilder(), cancel: cancel, cancelAt: 100}

		done := make(chan error, 1)
		go func() {
			_, err := helium.NewParser().SAXHandler(handler).Parse(ctx, largeDoc())
			done <- err
		}()

		select {
		case err := <-done:
			require.ErrorIs(t, err, context.Canceled, "Parse must return the context error")
		case <-time.After(10 * time.Second):
			t.Fatal("Parse did not return promptly after context cancellation")
		}
	})

	// a mid-parse
	// cancellation is not treated as a recoverable parse error: even with
	// RecoverOnError(true) enabled, Parse must return the context error AND a nil
	// document, never a partial tree.
	t.Run("with RecoverOnError", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		handler := &cancellingSAX{TreeBuilder: helium.NewTreeBuilder(), cancel: cancel, cancelAt: 100}

		done := make(chan struct {
			doc *helium.Document
			err error
		}, 1)
		go func() {
			doc, err := helium.NewParser().RecoverOnError(true).SAXHandler(handler).Parse(ctx, largeDoc())
			done <- struct {
				doc *helium.Document
				err error
			}{doc, err}
		}()

		select {
		case res := <-done:
			require.ErrorIs(t, res.err, context.Canceled, "Parse must return the context error even with RecoverOnError")
			require.Nil(t, res.doc, "cancelled parse must not return a partial document")
		case <-time.After(10 * time.Second):
			t.Fatal("Parse did not return promptly after context cancellation")
		}
	})

	// cancellation observed
	// while the parser is in its recovery / skip-to-recover-point path returns
	// promptly with the context error and a nil document, and never blocks. The
	// input is malformed so recovery is active and RecoverOnError is enabled so the
	// parser would otherwise keep scanning the long tail to the end of input.
	t.Run("during recovery", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())

		done := make(chan struct {
			doc *helium.Document
			err error
		}, 1)
		go func() {
			doc, err := helium.NewParser().RecoverOnError(true).Parse(ctx, malformedRecoverableDoc())
			done <- struct {
				doc *helium.Document
				err error
			}{doc, err}
		}()

		// Give the parser a moment to enter the recovery/skip loop, then cancel.
		time.AfterFunc(20*time.Millisecond, cancel)

		select {
		case res := <-done:
			require.ErrorIs(t, res.err, context.Canceled, "Parse must return the context error when cancelled during recovery")
			require.Nil(t, res.doc, "cancelled parse must not return a partial document")
		case <-time.After(10 * time.Second):
			t.Fatal("Parse did not return promptly after cancellation during recovery")
		}
	})

	// when a context is
	// cancelled mid-parse, Parse returns the context error and the SAX Error
	// handler is NOT invoked at all: a clean cancellation must not look like a
	// malformed document to the handler.
	t.Run("does not fire the SAX error callback", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		handler := &cancellingSAX{TreeBuilder: helium.NewTreeBuilder(), cancel: cancel, cancelAt: 100}

		done := make(chan error, 1)
		go func() {
			_, err := helium.NewParser().SAXHandler(handler).Parse(ctx, largeDoc())
			done <- err
		}()

		var err error
		select {
		case err = <-done:
		case <-time.After(10 * time.Second):
			t.Fatal("Parse did not return promptly after context cancellation")
		}

		require.ErrorIs(t, err, context.Canceled, "Parse must return the context error")
		require.Empty(t, handler.recorded(), "SAX Error handler must not be invoked on a clean cancellation")
	})
}
