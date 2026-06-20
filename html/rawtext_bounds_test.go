package html_test

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/html"
	"github.com/stretchr/testify/require"
)

// tagPlaintext / tagTextarea are element names reused across these table-driven
// cases (factored out to satisfy goconst).
const (
	tagPlaintext = "plaintext"
	tagTextarea  = "textarea"
	tagTitle     = "title"

	// Subtest names for the comment-like constructs, shared across the
	// comment/bogus-comment/PI table tests.
	nameComment      = "comment"
	nameBogusComment = "bogus_comment"
)

// cancelAfterReader is an io.Reader that streams a fixed body and invokes a
// cancel func once a threshold number of bytes has been read. It lets a test
// cancel the context AFTER the parser has entered (and is actively scanning) a
// raw-text / comment section, rather than before parsing starts.
type cancelAfterReader struct {
	data   []byte
	pos    int
	after  int
	cancel context.CancelFunc
	fired  bool
}

func (r *cancelAfterReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	if !r.fired && r.pos >= r.after {
		r.fired = true
		r.cancel()
	}
	return n, nil
}

// TestRawTextContextCancellationAborts verifies that cancelling the context
// WHILE the parser is inside a raw-text / RCDATA / plaintext / comment section
// aborts the scan promptly with context.Canceled, instead of buffering the rest
// of the (possibly endless) section until EOF.
//
// The chunked sections (script/style/textarea/plaintext) emit content chunks
// from inside the scan loop, so a SAX callback cancels mid-scan on the first
// chunk. The comment section emits no mid-scan SAX event, so a controlled
// reader cancels after enough bytes have streamed in. Either way the scan loop
// observes ctx.Err() on its next iteration and unwinds. No large allocations.
func TestRawTextContextCancellationAborts(t *testing.T) {
	const limit = 8      // small cap → chunks flush almost immediately
	const reps = 1 << 16 // enough body that scanning is still in progress
	body := strings.Repeat("a", reps)

	// Sections that emit chunked SAX events: cancel from the callback.
	chunked := []struct {
		name  string
		input string
	}{
		{"script", "<script>" + body},
		{"style", "<style>" + body},
		{tagTextarea, "<textarea>" + body},
		{tagTitle, "<title>" + body},
		{tagPlaintext, "<plaintext>" + body},
	}
	for _, tc := range chunked {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			t.Cleanup(cancel)

			sax := &html.SAXCallbacks{}
			cancelOnChunk := html.CharactersFunc(func([]byte) error {
				cancel() // cancel AFTER the scan has begun emitting content
				return nil
			})
			sax.SetOnCharacters(cancelOnChunk)
			sax.SetOnCDataBlock(html.CDataBlockFunc(cancelOnChunk))

			done := make(chan error, 1)
			go func() {
				done <- html.NewParser().MaxContentSize(limit).
					ParseWithSAX(ctx, []byte(tc.input), sax)
			}()

			select {
			case err := <-done:
				require.ErrorIs(t, err, context.Canceled,
					"cancelled mid-scan parse should return context.Canceled")
			case <-time.After(10 * time.Second):
				t.Fatal("parse did not abort promptly on context cancellation")
			}
		})
	}

	// Comment: no mid-scan SAX event, so cancel via the reader after a few
	// bytes of the comment body have streamed in.
	t.Run(nameComment, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		t.Cleanup(cancel)

		input := []byte("<!--" + body)
		r := &cancelAfterReader{data: input, after: 16, cancel: cancel}

		done := make(chan error, 1)
		go func() {
			_, err := html.NewParser().ParseReader(ctx, r)
			done <- err
		}()

		select {
		case err := <-done:
			require.ErrorIs(t, err, context.Canceled,
				"cancelled mid-comment parse should return context.Canceled")
		case <-time.After(10 * time.Second):
			t.Fatal("parse did not abort promptly on context cancellation")
		}
	})
}

// TestRawTextChunksAreValidUTF8 verifies that with a tiny MaxContentSize and
// multibyte content, every emitted raw-text / RCDATA / plaintext chunk is a
// whole-rune (valid UTF-8) slice. The cap-aware flush must never split a
// multi-byte UTF-8 sequence across two chunks: with MaxContentSize(1) the prior
// code emitted "\xc3" and "\xa9" of "é" as separate invalid chunks.
func TestRawTextChunksAreValidUTF8(t *testing.T) {
	// Mix of 1-, 2-, 3-, and 4-byte runes, repeated past several tiny caps.
	body := strings.Repeat("aé→𝄞z", 50)

	cases := []struct {
		name  string
		open  string
		close string
	}{
		{"script", "<script>", "</script>"},
		{"style", "<style>", "</style>"},
		{tagTextarea, "<textarea>", "</textarea>"}, // RCDATA
		{tagTitle, "<title>", "</title>"},          // RCDATA
		{tagPlaintext, "<plaintext>", ""},
	}

	// Exercise caps both larger and smaller than the widest rune (4 bytes).
	for _, limit := range []int{1, 2, 3, 4, 7} {
		for _, tc := range cases {
			t.Run(fmt.Sprintf("%s_cap%d", tc.name, limit), func(t *testing.T) {
				input := tc.open + body + tc.close

				var chunks [][]byte
				record := html.CharactersFunc(func(data []byte) error {
					chunks = append(chunks, append([]byte(nil), data...))
					return nil
				})
				sax := &html.SAXCallbacks{}
				sax.SetOnCharacters(record)
				sax.SetOnCDataBlock(html.CDataBlockFunc(record))

				err := html.NewParser().MaxContentSize(limit).
					ParseWithSAX(t.Context(), []byte(input), sax)
				require.NoError(t, err)

				var got strings.Builder
				for i, c := range chunks {
					require.True(t, utf8.Valid(c),
						"chunk %d must be valid UTF-8 (limit=%d): %q", i, limit, c)
					got.Write(c)
				}
				require.Equal(t, body, got.String(),
					"reassembled content must match input (limit=%d)", limit)
			})
		}
	}
}

// TestRawTextContentChunkedUnderCap verifies that an over-cap raw-text /
// plaintext / RCDATA section is delivered in bounded chunks rather than a
// single unbounded buffer, and that the full content is still produced.
func TestRawTextContentChunkedUnderCap(t *testing.T) {
	const limit = 1 << 10 // 1 KiB cap
	const total = 10 * limit

	cases := []struct {
		name  string
		open  string
		close string
	}{
		{"script", "<script>", "</script>"},
		{"style", "<style>", "</style>"},
		{tagTextarea, "<textarea>", "</textarea>"},
		{tagPlaintext, "<plaintext>", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := strings.Repeat("x", total)
			input := tc.open + body + tc.close

			var chunks [][]byte
			record := html.CharactersFunc(func(data []byte) error {
				chunks = append(chunks, append([]byte(nil), data...))
				return nil
			})
			sax := &html.SAXCallbacks{}
			sax.SetOnCharacters(record)
			sax.SetOnCDataBlock(html.CDataBlockFunc(record))

			p := html.NewParser().MaxContentSize(limit)
			err := p.ParseWithSAX(t.Context(), []byte(input), sax)
			require.NoError(t, err)

			// Full content must be preserved across the chunks.
			var got strings.Builder
			maxChunk := 0
			for _, c := range chunks {
				got.Write(c)
				if len(c) > maxChunk {
					maxChunk = len(c)
				}
			}
			require.Equal(t, body, got.String(), "reassembled content must match input")

			// Memory is bounded: no single chunk exceeds the cap by more than
			// a small terminator-handling slack.
			require.LessOrEqual(t, maxChunk, limit+16,
				"chunks must be bounded by the configured cap")
			require.Greater(t, len(chunks), 1,
				"over-cap content must be split into multiple chunks")
		})
	}
}

// TestCommentLikeOverCapHardErrors verifies that a comment, bogus comment, or
// processing instruction that exceeds MaxContentSize before its terminator
// fails the parse with ErrContentSizeExceeded instead of being chunked. These
// constructs map to a single indivisible SAX event / DOM node, so chunking
// would corrupt the document (the truncated tail would leak as stray text).
func TestCommentLikeOverCapHardErrors(t *testing.T) {
	const limit = 4
	body := strings.Repeat("a", 10*limit)

	cases := []struct {
		name  string
		input string
	}{
		{nameComment, "<!--" + body + "--><p>x</p>"},
		{nameBogusComment, "<!" + body + "><p>x</p>"},
		{"pi", "<?" + body + "><p>x</p>"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := html.NewParser().MaxContentSize(limit)
			_, err := p.Parse(t.Context(), []byte(tc.input))
			require.ErrorIs(t, err, html.ErrContentSizeExceeded,
				"over-cap %s must fail with ErrContentSizeExceeded", tc.name)
		})
	}
}

// TestCommentLikeNULBypassHardErrors guards against a NUL-byte cap bypass: the
// comment / bogus-comment / PI scanners must distinguish a real U+0000 byte from
// end-of-input (PeekAt returns 0 for both) via HasByteAt, count the NUL as
// content, and still hard-fail when the run exceeds MaxContentSize before its
// terminator. A NUL placed before the terminator must not be mistaken for EOF
// and silently emit a (truncated) comment instead of erroring.
func TestCommentLikeNULBypassHardErrors(t *testing.T) {
	const limit = 4
	body := "\x00" + strings.Repeat("a", 10*limit)

	cases := []struct {
		name  string
		input string
	}{
		{nameComment, "<!--" + body + "--><p>x</p>"},
		{nameBogusComment, "<!" + body + "><p>x</p>"},
		{"pi", "<?" + body + "><p>x</p>"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := html.NewParser().MaxContentSize(limit)
			_, err := p.Parse(t.Context(), []byte(tc.input))
			require.ErrorIs(t, err, html.ErrContentSizeExceeded,
				"NUL-before-terminator %s must fail with ErrContentSizeExceeded", tc.name)
		})
	}
}

// TestCommentLikeUnderCapParses verifies that a comment / bogus comment / PI
// that fits within MaxContentSize parses correctly and is delivered as a single
// Comment SAX event.
func TestCommentLikeUnderCapParses(t *testing.T) {
	const limit = 1 << 10 // 1 KiB cap

	cases := []struct {
		name  string
		input string
		want  string
	}{
		{nameComment, "<!--hello world-->", "hello world"},
		{nameBogusComment, "<!bogus content>", "bogus content"},
		{"pi", "<?php echo 1 ?>", "?php echo 1 ?"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var comments [][]byte
			sax := &html.SAXCallbacks{}
			sax.SetOnComment(html.CommentFunc(func(value []byte) error {
				comments = append(comments, append([]byte(nil), value...))
				return nil
			}))

			p := html.NewParser().MaxContentSize(limit)
			err := p.ParseWithSAX(t.Context(), []byte(tc.input), sax)
			require.NoError(t, err, "under-cap %s must parse without error", tc.name)
			require.Len(t, comments, 1, "must be a single Comment event")
			require.Equal(t, tc.want, string(comments[0]))
		})
	}
}

// TestCommentLikeExactCapParses verifies the boundary contract: a comment /
// bogus comment / PI whose content is EXACTLY MaxContentSize bytes parses
// successfully as a single Comment event, while content of limit+1 bytes fails
// with ErrContentSizeExceeded. This pins the off-by-one fix where exact-limit
// content was previously rejected.
func TestCommentLikeExactCapParses(t *testing.T) {
	const limit = 4

	// PI content includes the leading '?' (libxml2 emits <?...> as a comment
	// without the surrounding '<' and '>'), so a PI with N body chars yields
	// N+1 content bytes. Account for that when building exact-limit input.
	cases := []struct {
		name    string
		atLimit string // content length == limit
		want    string
		over    string // content length == limit+1
	}{
		{nameComment, "<!--abcd-->", "abcd", "<!--abcde-->"},
		{nameBogusComment, "<!abcd>", "abcd", "<!abcde>"},
		{"pi", "<?abc>", "?abc", "<?abcd>"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var comments [][]byte
			sax := &html.SAXCallbacks{}
			sax.SetOnComment(html.CommentFunc(func(value []byte) error {
				comments = append(comments, append([]byte(nil), value...))
				return nil
			}))

			p := html.NewParser().MaxContentSize(limit)
			err := p.ParseWithSAX(t.Context(), []byte(tc.atLimit), sax)
			require.NoError(t, err,
				"exact-limit %s content must parse without error", tc.name)
			require.Len(t, comments, 1,
				"exact-limit %s must produce a single Comment event", tc.name)
			require.Equal(t, tc.want, string(comments[0]))

			_, err = html.NewParser().MaxContentSize(limit).
				Parse(t.Context(), []byte(tc.over))
			require.ErrorIs(t, err, html.ErrContentSizeExceeded,
				"limit+1 %s content must fail with ErrContentSizeExceeded", tc.name)
		})
	}
}

// TestCommentChunkingCorruptionRepro is the concrete regression repro from the
// review: with MaxContentSize(4), `<!--aaaaaaaaaa--><p>x</p>` previously parsed
// as a truncated `<!--aaaa-->` followed by stray text `aaaaaa--&gt;`, corrupting
// the document. It must now error instead of producing a corrupted document.
func TestCommentChunkingCorruptionRepro(t *testing.T) {
	const input = "<!--aaaaaaaaaa--><p>x</p>"

	doc, err := html.NewParser().MaxContentSize(4).Parse(t.Context(), []byte(input))
	require.ErrorIs(t, err, html.ErrContentSizeExceeded,
		"over-cap comment must error rather than corrupt the document")
	require.Nil(t, doc, "no document should be returned on a hard size error")

	// Sanity: the same input parses cleanly when the cap accommodates it, and
	// the comment text is preserved intact (not split or leaked as text).
	var comments [][]byte
	sax := &html.SAXCallbacks{}
	sax.SetOnComment(html.CommentFunc(func(value []byte) error {
		comments = append(comments, append([]byte(nil), value...))
		return nil
	}))
	err = html.NewParser().MaxContentSize(1<<10).
		ParseWithSAX(t.Context(), []byte(input), sax)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	require.Equal(t, "aaaaaaaaaa", string(comments[0]))
}

// TestRawTextChunkSlicesAreIndependent guards against a buffer-reuse aliasing
// bug in the chunk flush: the parser flushed content via bytes.Buffer.Bytes()
// and then called Reset(), which reuses the same backing array. A SAX handler
// that RETAINS the chunk slice (without copying) would then see earlier chunks
// overwritten by later content. This handler deliberately does NOT copy the
// slices, so the concatenation of retained slices must still equal the original
// content.
func TestRawTextChunkSlicesAreIndependent(t *testing.T) {
	const limit = 1 << 10 // 1 KiB cap
	const total = 10 * limit

	cases := []struct {
		name  string
		open  string
		close string
	}{
		{"script", "<script>", "</script>"},
		{"style", "<style>", "</style>"},
		{tagPlaintext, "<plaintext>", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Use a varying body so an aliased (overwritten) earlier chunk
			// would produce a detectable mismatch, not an accidental match.
			body := make([]byte, total)
			for i := range body {
				body[i] = byte('A' + (i % 26))
			}
			input := tc.open + string(body) + tc.close

			// Retain the slices WITHOUT copying them.
			var chunks [][]byte
			retain := html.CharactersFunc(func(data []byte) error {
				chunks = append(chunks, data)
				return nil
			})
			sax := &html.SAXCallbacks{}
			sax.SetOnCharacters(retain)
			sax.SetOnCDataBlock(html.CDataBlockFunc(retain))

			p := html.NewParser().MaxContentSize(limit)
			err := p.ParseWithSAX(t.Context(), []byte(input), sax)
			require.NoError(t, err)

			require.Greater(t, len(chunks), 1,
				"over-cap content must be split into multiple chunks")

			var got []byte
			for _, c := range chunks {
				got = append(got, c...)
			}
			require.Equal(t, string(body), string(got),
				"retained chunk slices must not be overwritten by later content")
		})
	}
}

// TestRCDATAOverCapNamedEntityFails is the core-invariant regression for the
// RCDATA char-ref bypass: with a tiny MaxContentSize, a NAMED entity reference
// whose name runs far past the cap — e.g. `<title>&aaaa...(huge)...</title>` —
// must NOT be buffered whole. A name longer than the longest known entity (31
// chars) can never resolve to a full entity; deciding whether it resolves a
// legacy prefix requires knowing whether a ';' eventually terminates the run,
// and with a forward-only cursor finding that terminator while preserving SAX
// source order would mean retaining the entire tail — exactly the unbounded
// growth MaxContentSize exists to prevent. So the parser rejects the over-cap
// named reference with ErrContentSizeExceeded instead, keeping peak retained
// memory bounded by the cap. Covers both RCDATA elements (title, textarea),
// with and without a trailing ';', and legacy-prefix runs (which only resolve
// without ';' and so are equally unbounded to decide).
//
// Numeric references are unaffected (see TestRCDATANumericEntityNormalized):
// they accumulate a saturating value without retaining the digit run, so an
// overlong numeric reference still resolves rather than failing.
func TestRCDATAOverCapNamedEntityFails(t *testing.T) {
	const limit = 4
	const runLen = 4096 // far larger than any valid entity name or legacy prefix

	cases := []struct {
		name string
		body string // RCDATA content
	}{
		{"unknown", "&" + strings.Repeat("a", runLen)},
		{"unknown_semicolon", "&" + strings.Repeat("a", runLen) + ";"},
		{"legacy_prefix", "&amp" + strings.Repeat("x", runLen)},
		{"legacy_prefix_semicolon", "&amp" + strings.Repeat("x", runLen) + ";"},
	}

	for _, elem := range []string{tagTitle, tagTextarea} {
		for _, tc := range cases {
			t.Run(elem+"_"+tc.name, func(t *testing.T) {
				input := "<" + elem + ">" + tc.body + "</" + elem + ">"

				// Track the largest single Characters chunk and total retained
				// bytes: nothing of the over-cap run may be buffered before the
				// parse aborts.
				maxChunk := 0
				record := html.CharactersFunc(func(data []byte) error {
					if len(data) > maxChunk {
						maxChunk = len(data)
					}
					return nil
				})
				sax := &html.SAXCallbacks{}
				sax.SetOnCharacters(record)
				sax.SetOnCDataBlock(html.CDataBlockFunc(record))

				err := html.NewParser().MaxContentSize(limit).
					ParseWithSAX(t.Context(), []byte(input), sax)
				require.ErrorIs(t, err, html.ErrContentSizeExceeded,
					"over-cap named %s reference in RCDATA must fail, not buffer the tail", tc.name)
				require.LessOrEqual(t, maxChunk, limit+16,
					"no single Characters chunk may exceed the cap before the abort")
			})
		}
	}
}

// TestRCDATAWithinCapNamedEntity verifies that a within-cap named reference in
// RCDATA still resolves exactly like the normal-text (parseCharRef) path under
// a small MaxContentSize: a known entity with ';' resolves, a legacy (HTML4)
// entity resolves WITHOUT ';' and emits its tail literally, and an unknown but
// within-cap name is echoed verbatim. Only names longer than the cap (which can
// never be a known entity) are rejected — this pins that the hard-fail is
// scoped to the over-cap case and does not break ordinary references.
func TestRCDATAWithinCapNamedEntity(t *testing.T) {
	const limit = 8 // larger than every name below but far smaller than 16 MiB

	cases := []struct {
		name string
		body string // RCDATA source
		want string // expected concatenated Characters output
	}{
		{"amp_semicolon", "&amp;", "&"},
		{"lt_semicolon", "&lt;", "<"},
		{"amp_no_semicolon_tail", "&ampZ", "&Z"}, // legacy "amp" resolves; "Z" literal
		{"unknown_semicolon", "&zzz;", "&zzz;"},  // unknown name echoed verbatim
	}

	for _, elem := range []string{tagTitle, tagTextarea} {
		for _, tc := range cases {
			t.Run(elem+"_"+tc.name, func(t *testing.T) {
				input := "<" + elem + ">" + tc.body + "</" + elem + ">"

				var got strings.Builder
				record := html.CharactersFunc(func(data []byte) error {
					got.Write(data)
					return nil
				})
				sax := &html.SAXCallbacks{}
				sax.SetOnCharacters(record)
				sax.SetOnCDataBlock(html.CDataBlockFunc(record))

				err := html.NewParser().MaxContentSize(limit).
					ParseWithSAX(t.Context(), []byte(input), sax)
				require.NoError(t, err,
					"within-cap named reference in RCDATA must parse")
				require.Equal(t, tc.want, got.String(),
					"within-cap named reference must resolve like the normal-text path")
			})
		}
	}
}

// TestRCDATANumericEntityNormalized verifies that the bounded RCDATA char-ref
// scanner makes the SAME entity-resolution decision as the normal-text scanner
// for numeric references, even with a tiny MaxContentSize: an overlong numeric
// reference resolves (to U+FFFD on overflow) rather than being emitted as
// literal text, and a long leading-zero reference still resolves to its value.
func TestRCDATANumericEntityNormalized(t *testing.T) {
	const limit = 4
	const runLen = 4096

	cases := []struct {
		name string
		body string // RCDATA content
		want string // expected concatenated Characters output
	}{
		// Overlong decimal/hex runs overflow U+10FFFF and normalize to U+FFFD,
		// matching parseNumericCharRef in the unbounded path.
		{"overflow_dec", "&#" + strings.Repeat("9", runLen) + ";", "�"},
		{"overflow_hex", "&#x" + strings.Repeat("f", runLen) + ";", "�"},
		{"overflow_dec_no_semi", "&#" + strings.Repeat("9", runLen), "�"},
		// Long leading-zero runs are valid: a zero-padded reference resolves to
		// its actual code point (U+0041 'A' here) instead of being treated as
		// unresolved literal text.
		{"leading_zero_dec", "&#" + strings.Repeat("0", runLen) + "65;", "A"},
		{"leading_zero_hex", "&#x" + strings.Repeat("0", runLen) + "41;", "A"},
	}

	for _, elem := range []string{tagTitle, tagTextarea} {
		for _, tc := range cases {
			t.Run(elem+"_"+tc.name, func(t *testing.T) {
				input := "<" + elem + ">" + tc.body + "</" + elem + ">"

				var got strings.Builder
				record := html.CharactersFunc(func(data []byte) error {
					got.Write(data)
					return nil
				})
				sax := &html.SAXCallbacks{}
				sax.SetOnCharacters(record)
				sax.SetOnCDataBlock(html.CDataBlockFunc(record))

				err := html.NewParser().MaxContentSize(limit).
					ParseWithSAX(t.Context(), []byte(input), sax)
				require.NoError(t, err,
					"over-cap numeric reference in RCDATA must parse, not error")
				require.Equal(t, tc.want, got.String(),
					"bounded numeric char-ref must normalize like the unbounded path")
			})
		}
	}
}

// TestIndivisibleNodeCancellationNoPartialEmit verifies that cancelling the
// context WHILE the parser is scanning an indivisible node (comment, bogus
// comment, or processing instruction) aborts WITHOUT emitting a truncated node.
// These constructs map to a single SAX Comment event / DOM node; emitting the
// bytes scanned so far would publish a partial comment whose remainder leaks as
// stray text. The parse must return context.Canceled and no Comment node may
// land in the resulting tree.
func TestIndivisibleNodeCancellationNoPartialEmit(t *testing.T) {
	const reps = 1 << 16 // long, unterminated body so the scan is still running
	body := strings.Repeat("a", reps)

	cases := []struct {
		name  string
		input string
	}{
		{nameComment, "<!--" + body},                // parseComment
		{nameBogusComment, "<!" + body},             // parseBogusComment
		{"processing_instruction", "<?php " + body}, // parsePI
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			t.Cleanup(cancel)

			// Cancel a few bytes into the (still-unterminated) indivisible node so
			// the scan loop observes ctx.Err() mid-construct.
			r := &cancelAfterReader{data: []byte(tc.input), after: 32, cancel: cancel}

			done := make(chan struct {
				doc *helium.Document
				err error
			}, 1)
			go func() {
				doc, err := html.NewParser().ParseReader(ctx, r)
				done <- struct {
					doc *helium.Document
					err error
				}{doc, err}
			}()

			select {
			case res := <-done:
				require.ErrorIs(t, res.err, context.Canceled,
					"mid-scan cancellation should return context.Canceled")
				if res.doc == nil {
					return // no tree at all → trivially no partial comment node
				}
				_ = helium.Walk(res.doc, helium.NodeWalkerFunc(func(n helium.Node) error {
					require.NotEqual(t, helium.CommentNode, n.Type(),
						"no partial comment/PI node after mid-scan cancellation")
					return nil
				}))
			case <-time.After(10 * time.Second):
				t.Fatal("parse did not abort promptly on context cancellation")
			}
		})
	}
}
