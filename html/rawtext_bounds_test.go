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
//
// maxRead caps the bytes returned per Read so a single large read can't gulp
// the whole input (and the entire target construct) before `after` is reached.
// ParseReader first drains a 1024-byte charset prescan; throttling per-read and
// placing `after` past that prescan AND inside the target construct ensures the
// cancellation lands while the parser is actively scanning the construct rather
// than during the prescan or before parsing starts.
type cancelAfterReader struct {
	data    []byte
	pos     int
	after   int
	maxRead int
	cancel  context.CancelFunc
	fired   bool
}

func (r *cancelAfterReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	if r.maxRead > 0 && len(p) > r.maxRead {
		p = p[:r.maxRead]
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	if !r.fired && r.pos >= r.after {
		r.fired = true
		r.cancel()
	}
	return n, nil
}

// metaUTF8 forces ParseReader onto the streaming sanitize path. Without a
// charset declaration ParseReader can buffer an all-valid-UTF-8 stream to EOF,
// which would consume the entire input (and target construct) before any
// cancellation threshold inside it is reached, masking the scanner path.
const metaUTF8 = `<meta charset="utf-8">`

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

	// Comment: no mid-scan SAX event, so cancel via the reader once the comment
	// body has streamed PAST the 1024-byte charset prescan (so the parser is
	// actively scanning the comment, not prescanning). A meta charset forces the
	// streaming path and throttled reads keep the cancel mid-comment.
	t.Run(nameComment, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		t.Cleanup(cancel)

		input := []byte(metaUTF8 + "<!--" + body)
		r := &cancelAfterReader{data: input, after: 1100, maxRead: 64, cancel: cancel}

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
// RCDATA char-ref bypass: with a tiny MaxContentSize, an UNRESOLVED named
// reference whose alphanumeric run runs far past the cap — e.g.
// `<title>&aaaa...(huge)...</title>` — must NOT be buffered whole. Entity
// resolution itself uses a FIXED maxEntityNameLen lookahead (a constant
// independent of MaxContentSize), so a name longer than the longest known
// entity (31 chars) that matches no legacy prefix can never resolve; its run is
// LITERAL text and is charged against MaxContentSize. Once the unresolved
// literal exceeds the cap before any terminator the parser fails with
// ErrContentSizeExceeded, keeping peak retained memory bounded.
//
// Legacy-prefix runs are NOT here: `&amp` + a long tail resolves the legacy
// "amp" prefix within the fixed lookahead and emits the tail as ordinary
// (chunked) text, never failing — see TestRCDATALegacyPrefixLongTailResolves.
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
					"over-cap unresolved %s reference in RCDATA must fail, not buffer the tail", tc.name)
				require.LessOrEqual(t, maxChunk, limit+16,
					"no single Characters chunk may exceed the cap before the abort")
			})
		}
	}
}

// TestRCDATAUnresolvedSemicolonCharged is the regression for the trailing-';'
// undercount: an unresolved short-name reference whose name fits the fixed
// lookahead emits the consumed ';' as part of the LITERAL run, so the cap check
// must charge it. With MaxContentSize(4), `&zzz;` emits 5 literal bytes
// (`&`,`z`,`z`,`z`,`;`) and must hard-fail (5 > 4); `&zz;` emits exactly 4 and
// is accepted (4 == 4), matching the strict '>' cap convention used elsewhere.
func TestRCDATAUnresolvedSemicolonCharged(t *testing.T) {
	const limit = 4

	cases := []struct {
		name    string
		body    string // RCDATA content
		wantErr bool
		want    string // expected literal echo when accepted
	}{
		{"over_cap_with_semicolon", "&zzz;", true, ""},
		{"at_cap_with_semicolon", "&zz;", false, "&zz;"},
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
				if tc.wantErr {
					require.ErrorIs(t, err, html.ErrContentSizeExceeded,
						"unresolved %s reference whose literal (incl. ';') exceeds the cap must fail", tc.name)
					return
				}
				require.NoError(t, err,
					"unresolved %s reference whose literal equals the cap must be accepted", tc.name)
				require.Equal(t, tc.want, got.String(),
					"accepted literal must echo the full unresolved run including ';'")
			})
		}
	}
}

// TestRCDATASmallCapKnownEntityResolves pins the convergent invariant that
// entity resolution uses a FIXED maxEntityNameLen lookahead, NOT MaxContentSize:
// a known named reference whose resolved value is tiny (a single '&', '<', …)
// resolves even when MaxContentSize is smaller than the entity NAME itself. With
// MaxContentSize(2), `<title>&amp;</title>` must resolve to "&" rather than
// being rejected because the 3-char name "amp" exceeds the cap.
func TestRCDATASmallCapKnownEntityResolves(t *testing.T) {
	const limit = 2 // smaller than the entity names below

	cases := []struct {
		name string
		body string
		want string
	}{
		{"amp_semicolon", "&amp;", "&"},
		{"lt_semicolon", "&lt;", "<"},
		{"gt_semicolon", "&gt;", ">"},
		{"quot_semicolon", "&quot;", "\""},
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
					"a known entity must resolve even when its name exceeds MaxContentSize")
				require.Equal(t, tc.want, got.String(),
					"resolved value must match regardless of cap")
			})
		}
	}
}

// TestRCDATAOverCapLegacyPrefixLongTailFails pins the convergent BOUNDED-SPOOL
// contract: a legacy-prefix reference with a long no-semicolon tail
// (`&amp` + many chars) is AMBIGUOUS until the run ends — a trailing ';' would
// make it an over-long unknown literal, its absence legacy-resolves "amp". The
// decision can only be made at the run's end, so settling it without an over-cap
// spool is impossible once the run exceeds MaxContentSize. The parser therefore
// HARD-FAILS with ErrContentSizeExceeded and emits NOTHING, rather than streaming
// or buffering the unbounded tail. (A within-cap run still resolves — see
// TestRCDATAWithinCapSaturatedLegacyResolves.)
func TestRCDATAOverCapLegacyPrefixLongTailFails(t *testing.T) {
	const limit = 4
	const runLen = 4096

	for _, elem := range []string{tagTitle, tagTextarea} {
		t.Run(elem, func(t *testing.T) {
			tail := strings.Repeat("x", runLen)
			input := "<" + elem + ">&amp" + tail + "</" + elem + ">"

			var events [][]byte
			maxChunk := 0
			record := html.CharactersFunc(func(data []byte) error {
				cp := make([]byte, len(data))
				copy(cp, data)
				events = append(events, cp)
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
				"an over-cap ambiguous legacy-prefix run must hard-fail, not stream the tail")
			require.Empty(t, events,
				"no Characters callback may be delivered before the ErrContentSizeExceeded; got %q", events)
			require.LessOrEqual(t, maxChunk, limit+16,
				"no over-cap chunk may be emitted before the abort")
		})
	}
}

// TestRCDATAWithinCapSaturatedLegacyResolves pins that a SATURATED legacy-prefix
// run (its alphanumeric run overflows the fixed maxEntityNameLen lookahead) whose
// whole would-be literal still fits MaxContentSize is resolved through the
// bounded spool exactly like the unbounded parseCharRef path: `&amp` + a tail
// that overflows the 32-byte lookahead but fits a generous cap legacy-resolves
// "amp" to "&" and echoes the tail. This proves the hard-fail is scoped to the
// over-cap case and the spool still resolves a within-cap saturated run.
func TestRCDATAWithinCapSaturatedLegacyResolves(t *testing.T) {
	const tailLen = 40 // overflows maxEntityNameLen (32) so the scan saturates

	for _, elem := range []string{tagTitle, tagTextarea} {
		t.Run(elem, func(t *testing.T) {
			tail := strings.Repeat("x", tailLen)
			input := "<" + elem + ">&amp" + tail + "</" + elem + ">"

			var got strings.Builder
			record := html.CharactersFunc(func(data []byte) error {
				got.Write(data)
				return nil
			})
			sax := &html.SAXCallbacks{}
			sax.SetOnCharacters(record)
			sax.SetOnCDataBlock(html.CDataBlockFunc(record))

			err := html.NewParser().MaxContentSize(1<<10).
				ParseWithSAX(t.Context(), []byte(input), sax)
			require.NoError(t, err,
				"a within-cap saturated legacy-prefix run must resolve, not fail")
			require.Equal(t, "&"+tail, got.String(),
				"legacy 'amp' prefix resolves to '&' and the tail is echoed literally")
		})
	}
}

// TestRCDATALongSemicolonNameNotLegacyResolved is the convergence regression for
// the bounded-vs-unbounded char-ref decision: a long ';'-terminated name that
// merely BEGINS with a legacy prefix (`&amp` + a long alphanumeric tail + `;`)
// must NOT be legacy-resolved — it is an over-long unknown name. parseCharRef
// scans the whole run before deciding and never legacy-resolves a
// ';'-terminated name (its prefix loop is gated on the no-semicolon case), so it
// emits the WHOLE run literally. The bounded scanner must make the identical
// decision: the run is literal text, charged against MaxContentSize, and it
// hard-fails once the literal exceeds the cap before the terminator. The earlier
// bug resolved the legacy "amp" prefix from the truncated 32-byte lookahead
// (before seeing the eventual ';'), dropping "amp" and emitting "&xxxx...;".
//
// The no-semicolon counterpart (`&amp` + long tail, no ';') is the OPPOSITE
// decision and DOES legacy-resolve when its run fits the cap — see
// TestRCDATAWithinCapSaturatedLegacyResolves; this test re-pins that resolve half
// (under a within-cap limit) alongside the literal halves to lock them together.
// (When such a no-';' run is OVER the cap it hard-fails as an ambiguous run —
// TestRCDATAOverCapLegacyPrefixLongTailFails.)
func TestRCDATALongSemicolonNameNotLegacyResolved(t *testing.T) {
	const tailLen = 40 // run far exceeds maxEntityNameLen (32) → lookahead saturates

	for _, elem := range []string{tagTitle, tagTextarea} {
		// Over-cap, ';'-terminated, begins with the legacy "amp" prefix: must be
		// treated as an unresolved literal and hard-fail (never legacy-resolved).
		t.Run(elem+"_over_cap_semicolon_literal", func(t *testing.T) {
			body := "&amp" + strings.Repeat("x", tailLen) + ";"
			input := "<" + elem + ">" + body + "</" + elem + ">"

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

			err := html.NewParser().MaxContentSize(4).
				ParseWithSAX(t.Context(), []byte(input), sax)
			require.ErrorIs(t, err, html.ErrContentSizeExceeded,
				"a long ';'-terminated name beginning with a legacy prefix must be literal and hard-fail, not legacy-resolved")
			require.LessOrEqual(t, maxChunk, 4+16,
				"no over-cap literal may be emitted before the abort")
		})

		// Within-cap, ';'-terminated: same run but a cap large enough to hold the
		// literal. It must be echoed VERBATIM (still no legacy resolution),
		// exactly as the unbounded parseCharRef would emit it.
		t.Run(elem+"_within_cap_semicolon_literal", func(t *testing.T) {
			body := "&amp" + strings.Repeat("x", tailLen) + ";"
			input := "<" + elem + ">" + body + "</" + elem + ">"

			var got strings.Builder
			record := html.CharactersFunc(func(data []byte) error {
				got.Write(data)
				return nil
			})
			sax := &html.SAXCallbacks{}
			sax.SetOnCharacters(record)
			sax.SetOnCDataBlock(html.CDataBlockFunc(record))

			err := html.NewParser().MaxContentSize(1<<10).
				ParseWithSAX(t.Context(), []byte(input), sax)
			require.NoError(t, err,
				"a within-cap long ';'-terminated unknown name must be echoed, not rejected")
			require.Equal(t, body, got.String(),
				"the whole run (incl. 'amp' and ';') must be echoed literally — no legacy resolution")
		})

		// The no-semicolon sibling under a WITHIN-cap limit legacy-resolves the
		// "amp" prefix and echoes the tail — the opposite decision, locked in
		// alongside the literal cases above. (Over the cap it would hard-fail as an
		// ambiguous run — see TestRCDATAOverCapLegacyPrefixLongTailFails.)
		t.Run(elem+"_no_semicolon_legacy_resolves", func(t *testing.T) {
			tail := strings.Repeat("x", tailLen)
			input := "<" + elem + ">&amp" + tail + "</" + elem + ">"

			var got strings.Builder
			record := html.CharactersFunc(func(data []byte) error {
				got.Write(data)
				return nil
			})
			sax := &html.SAXCallbacks{}
			sax.SetOnCharacters(record)
			sax.SetOnCDataBlock(html.CDataBlockFunc(record))

			err := html.NewParser().MaxContentSize(1<<10).
				ParseWithSAX(t.Context(), []byte(input), sax)
			require.NoError(t, err,
				"a within-cap no-semicolon legacy-prefix reference must resolve, not fail")
			require.Equal(t, "&"+tail, got.String(),
				"legacy 'amp' prefix resolves to '&' and the tail is echoed literally")
		})
	}
}

// TestRCDATASaturatedLegacyPrefixNoPartialEmit is the regression for the
// partial-emit-before-error bug: a saturated char-ref run that BEGINS with a
// legacy prefix and is ';'-terminated over the cap (`&amp` + a long tail + `;`)
// must NOT deliver ANY Characters callback before it hard-fails. The earlier
// code optimistically emitted the legacy resolution ('&') and the tail chunk by
// chunk while draining the run, then discovered the trailing ';' — which makes
// the whole run an over-cap unresolved literal — and returned
// ErrContentSizeExceeded. That left a partial emission ('&' plus a truncated,
// 'amp'-dropped tail) sitting ahead of the error, corrupting downstream output.
//
// The fix settles the ';' decision only AFTER consuming the run into a
// cap-bounded spool, so the error path delivers nothing. Under the convergent
// BOUNDED-SPOOL contract the over-cap NO-';' sibling ALSO hard-fails (ambiguous
// until the run ends, which can't be reached within cap) and likewise emits
// nothing; a within-cap no-';' run still legacy-resolves and emits correctly.
func TestRCDATASaturatedLegacyPrefixNoPartialEmit(t *testing.T) {
	const limit = 4
	const tailLen = 40 // run far exceeds maxEntityNameLen (32) so the scan saturates

	for _, elem := range []string{tagTitle, tagTextarea} {
		// ';'-terminated over-cap legacy-prefix run: hard-fail with NOTHING
		// emitted to the SAX Characters handler before the error.
		t.Run(elem+"_semicolon_emits_nothing_before_error", func(t *testing.T) {
			body := "&amp" + strings.Repeat("x", tailLen) + ";"
			input := "<" + elem + ">" + body + "</" + elem + ">"

			var events [][]byte
			record := html.CharactersFunc(func(data []byte) error {
				cp := make([]byte, len(data))
				copy(cp, data)
				events = append(events, cp)
				return nil
			})
			sax := &html.SAXCallbacks{}
			sax.SetOnCharacters(record)
			sax.SetOnCDataBlock(html.CDataBlockFunc(record))

			err := html.NewParser().MaxContentSize(limit).
				ParseWithSAX(t.Context(), []byte(input), sax)
			require.ErrorIs(t, err, html.ErrContentSizeExceeded,
				"a ';'-terminated over-cap legacy-prefix run must hard-fail")
			require.Empty(t, events,
				"no Characters callback may be delivered before the ErrContentSizeExceeded; got %q", events)
		})

		// The over-cap NO-';' sibling under the same tiny cap is ambiguous until
		// the run ends (which lies past the cap), so it ALSO hard-fails — and like
		// the ';' case must emit NOTHING before the error.
		t.Run(elem+"_no_semicolon_over_cap_emits_nothing_before_error", func(t *testing.T) {
			tail := strings.Repeat("x", tailLen)
			input := "<" + elem + ">&amp" + tail + "</" + elem + ">"

			var events [][]byte
			record := html.CharactersFunc(func(data []byte) error {
				cp := make([]byte, len(data))
				copy(cp, data)
				events = append(events, cp)
				return nil
			})
			sax := &html.SAXCallbacks{}
			sax.SetOnCharacters(record)
			sax.SetOnCDataBlock(html.CDataBlockFunc(record))

			err := html.NewParser().MaxContentSize(limit).
				ParseWithSAX(t.Context(), []byte(input), sax)
			require.ErrorIs(t, err, html.ErrContentSizeExceeded,
				"an over-cap no-';' ambiguous legacy-prefix run must hard-fail")
			require.Empty(t, events,
				"no Characters callback may be delivered before the ErrContentSizeExceeded; got %q", events)
		})

		// The genuine WITHIN-cap no-';' legacy-resolve sibling still emits the
		// resolved '&' and the echoed tail — proving the spool did not regress the
		// resolve path.
		t.Run(elem+"_no_semicolon_within_cap_still_resolves", func(t *testing.T) {
			tail := strings.Repeat("x", tailLen)
			input := "<" + elem + ">&amp" + tail + "</" + elem + ">"

			var got strings.Builder
			record := html.CharactersFunc(func(data []byte) error {
				got.Write(data)
				return nil
			})
			sax := &html.SAXCallbacks{}
			sax.SetOnCharacters(record)
			sax.SetOnCDataBlock(html.CDataBlockFunc(record))

			err := html.NewParser().MaxContentSize(1<<10).
				ParseWithSAX(t.Context(), []byte(input), sax)
			require.NoError(t, err,
				"a within-cap no-';' legacy-prefix run must still resolve, not fail")
			require.Equal(t, "&"+tail, got.String(),
				"legacy 'amp' resolves to '&' and the tail is echoed literally")
		})
	}
}

// TestRCDATANumericRefContextCancellation verifies Finding 2: a context
// cancelled WHILE the parser drains a long numeric character reference inside
// RCDATA (e.g. <title>&#9999...) aborts promptly with context.Canceled instead
// of consuming the entire digit run first. The digit run runs far past the
// 1024-byte charset prescan; a meta charset forces the streaming path and the
// reader cancels once the run has streamed past the prescan and is being
// drained. The bounded numeric scanner observes ctx.Err() between chunks and
// unwinds without emitting a partial entity.
func TestRCDATANumericRefContextCancellation(t *testing.T) {
	const reps = 1 << 16 // long digit run so the scan is still in progress

	for _, elem := range []string{tagTitle, tagTextarea} {
		t.Run(elem, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			t.Cleanup(cancel)

			// Unterminated numeric reference: '&#' followed by a huge digit run.
			input := []byte(metaUTF8 + "<" + elem + ">&#" + strings.Repeat("9", reps))
			r := &cancelAfterReader{data: input, after: 1100, maxRead: 64, cancel: cancel}

			done := make(chan error, 1)
			go func() {
				_, err := html.NewParser().MaxContentSize(8).ParseReader(ctx, r)
				done <- err
			}()

			select {
			case err := <-done:
				require.ErrorIs(t, err, context.Canceled,
					"cancelled mid-numeric-ref parse should return context.Canceled")
			case <-time.After(10 * time.Second):
				t.Fatal("parse did not abort promptly on numeric-ref cancellation")
			}
		})
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

// TestRCDATALongWithinCapNamedEntityPreserved pins the convergent bound: the
// memory limit tracks the user's MaxContentSize, NOT the fixed 32-byte
// maxEntityNameLen. A named-entity alphanumeric run that is longer than every
// known entity (so it can never resolve) but still fits within MaxContentSize
// must be PRESERVED literally — identical to the normal-text path — instead of
// being rejected. Only a run that genuinely exceeds the cap hard-fails.
func TestRCDATALongWithinCapNamedEntityPreserved(t *testing.T) {
	const limit = 100

	for _, elem := range []string{tagTitle, tagTextarea} {
		// A 40-char unknown name: longer than maxEntityNameLen (32) yet well
		// within MaxContentSize(100). Must be echoed verbatim, no error.
		t.Run(elem+"_within_cap_preserved", func(t *testing.T) {
			run := strings.Repeat("a", 40)
			body := "&" + run
			input := "<" + elem + ">" + body + "</" + elem + ">"

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
				"a within-cap named-entity run must be preserved, not rejected")
			require.Equal(t, body, got.String(),
				"within-cap unresolved run must be echoed literally like normal text")
		})

		// The same construction but with a run that genuinely exceeds the cap
		// still hard-fails — the failure is scoped to over-cap runs only.
		t.Run(elem+"_over_cap_fails", func(t *testing.T) {
			run := strings.Repeat("a", limit+50)
			body := "&" + run
			input := "<" + elem + ">" + body + "</" + elem + ">"

			_, err := html.NewParser().MaxContentSize(limit).
				Parse(t.Context(), []byte(input))
			require.ErrorIs(t, err, html.ErrContentSizeExceeded,
				"a named-entity run exceeding the cap must still hard-fail")
		})
	}
}

// TestRCDATAShortNameOverCapFails pins that an UNRESOLVED named reference whose
// alphanumeric name fits inside the fixed maxEntityNameLen lookahead — so it
// takes the within-cap fallback path rather than the long-run branch — is STILL
// charged against MaxContentSize. The literal run it produces is "&" plus the
// name; if that alone exceeds the cap the parse must hard-fail with
// ErrContentSizeExceeded, exactly like the long-run path. This guards the bug
// where a short unknown name (e.g. `&zzzzz;`, 7 literal bytes) under
// MaxContentSize(4) was silently emitted instead of erroring, because only runs
// continuing past the fixed lookahead reached the size check.
func TestRCDATAShortNameOverCapFails(t *testing.T) {
	const limit = 4

	// name length 5 → "&zzzzz" is 6 literal bytes > limit (4); both the
	// semicolon-terminated and no-semicolon forms must fail.
	overCap := []struct {
		name string
		body string
	}{
		{"no_semicolon", "&zzzzz"},
		{"semicolon", "&zzzzz;"},
	}

	for _, elem := range []string{tagTitle, tagTextarea} {
		for _, tc := range overCap {
			t.Run(elem+"_overcap_"+tc.name, func(t *testing.T) {
				input := "<" + elem + ">" + tc.body + "</" + elem + ">"

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
					"a short unknown name whose literal exceeds the cap must hard-fail")
				require.LessOrEqual(t, maxChunk, limit+16,
					"no over-cap literal may be emitted before the abort")
			})
		}
	}

	// Within-cap counterparts: "&zz" is 3 literal bytes <= limit (4), so it is
	// echoed verbatim (with any trailing ';') and never errors.
	withinCap := []struct {
		name string
		body string
		want string
	}{
		{"no_semicolon", "&zz", "&zz"},
		{"semicolon", "&zz;", "&zz;"},
	}

	for _, elem := range []string{tagTitle, tagTextarea} {
		for _, tc := range withinCap {
			t.Run(elem+"_withincap_"+tc.name, func(t *testing.T) {
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
					"a short unknown name within the cap must be echoed, not rejected")
				require.Equal(t, tc.want, got.String(),
					"within-cap unknown name must be echoed verbatim")
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
		{nameComment, metaUTF8 + "<!--" + body},                // parseComment
		{nameBogusComment, metaUTF8 + "<!" + body},             // parseBogusComment
		{"processing_instruction", metaUTF8 + "<?php " + body}, // parsePI
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			t.Cleanup(cancel)

			// Cancel once the (still-unterminated) indivisible node has streamed
			// PAST the 1024-byte charset prescan, so the scan loop observes
			// ctx.Err() mid-construct rather than during the prescan. A meta
			// charset forces the streaming path; throttled reads keep the cancel
			// landing inside the construct.
			r := &cancelAfterReader{data: []byte(tc.input), after: 1100, maxRead: 64, cancel: cancel}

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

// countingReader streams a fixed body and records how many bytes have actually
// been read from it. It lets a test assert that the parser ABORTED an over-cap
// run without first draining the whole (possibly unbounded) tail — bounding WORK,
// not just retained memory.
type countingReader struct {
	data []byte
	pos  int
}

func (r *countingReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

// TestRCDATAOverCapNonLegacyAbortsWithoutDrainingTail is the WORK-bound
// regression for parseSaturatedCharRefLiteral: a very long UNRESOLVED named
// reference whose run neither matches a known entity nor begins with a legacy
// prefix (so it can never resolve) must hard-fail with ErrContentSizeExceeded as
// soon as the run exceeds MaxContentSize — WITHOUT reading the rest of the run.
// Over a streaming reader the tail can be arbitrarily long; the prior code kept
// draining the entire alphanumeric tail to reach the trailing-';' check even
// though the literal had already blown the cap, bounding retained memory but not
// READ/WORK. The byte-counting reader proves only a small bounded prefix of the
// multi-megabyte run is consumed before the abort.
func TestRCDATAOverCapNonLegacyAbortsWithoutDrainingTail(t *testing.T) {
	const limit = 8
	const runLen = 4 << 20 // 4 MiB run: "zzzz..." matches no entity / legacy prefix

	for _, elem := range []string{tagTitle, tagTextarea} {
		t.Run(elem, func(t *testing.T) {
			// 'z' begins no legacy prefix and no known entity, so the whole run is
			// an unresolved literal. The run far exceeds maxEntityNameLen so the
			// saturated-literal path is taken. A <meta charset="utf-8"> declaration
			// selects the STREAMING sanitize reader so reads stay bounded — the
			// default deferred-Latin-1 path must buffer the whole stream to settle
			// the encoding, which would mask the parser-side WORK bound under test.
			input := []byte(metaUTF8 + "<" + elem + ">&" +
				strings.Repeat("z", runLen) + "</" + elem + ">")
			r := &countingReader{data: input}

			_, err := html.NewParser().MaxContentSize(limit).ParseReader(t.Context(), r)
			require.ErrorIs(t, err, html.ErrContentSizeExceeded,
				"over-cap non-legacy run must hard-fail with ErrContentSizeExceeded")

			// Only a bounded prefix may have been read: the meta + open tag, '&', a
			// small multiple of the cap, plus reader/decoder buffering slack — NOT
			// the whole multi-megabyte run.
			require.Less(t, r.pos, runLen/2,
				"abort must not drain the whole tail (read %d of %d run bytes)", r.pos, runLen)
		})
	}
}

// TestRCDATAOverCapLegacyPrefixAbortsWithoutDrainingTail is the convergent
// memory+work-bound regression named by the round-19/21 finding: an AMBIGUOUS
// legacy-prefix reference (`&amp` + a multi-megabyte alphanumeric tail) over a
// streaming reader must NOT buffer or drain the whole tail to decide. Resolving
// it would require reaching the run's end (to rule out a trailing ';'), which an
// unbounded non-consuming lookahead would buffer whole — re-violating the
// MaxContentSize streaming bound. The bounded spool instead hard-fails with
// ErrContentSizeExceeded as soon as the run exceeds the cap, reading only a small
// bounded prefix — proved by the counting reader. (No-partial-emit for the same
// legacy case is pinned in-memory by TestRCDATAOverCapLegacyPrefixLongTailFails.)
func TestRCDATAOverCapLegacyPrefixAbortsWithoutDrainingTail(t *testing.T) {
	const limit = 8
	const runLen = 4 << 20 // 4 MiB tail after the "amp" legacy prefix

	for _, elem := range []string{tagTitle, tagTextarea} {
		t.Run(elem, func(t *testing.T) {
			// <meta charset="utf-8"> selects the STREAMING sanitize reader so reads
			// stay bounded (the default deferred-Latin-1 path would buffer the whole
			// stream to settle the encoding, masking the parser-side work bound).
			input := []byte(metaUTF8 + "<" + elem + ">&amp" +
				strings.Repeat("x", runLen) + "</" + elem + ">")
			r := &countingReader{data: input}

			_, err := html.NewParser().MaxContentSize(limit).ParseReader(t.Context(), r)
			require.ErrorIs(t, err, html.ErrContentSizeExceeded,
				"over-cap ambiguous legacy-prefix run must hard-fail, not stream/drain the tail")
			require.Less(t, r.pos, runLen/2,
				"abort must not drain the whole tail (read %d of %d run bytes)", r.pos, runLen)
		})
	}
}

// TestRCDATASaturatedRefContextCancellation verifies that a context cancelled
// WHILE parseSaturatedCharRefLiteral spools a long alphanumeric run (the
// legacy-prefix path, which must reach the run's end to settle a possible ';')
// aborts promptly with context.Canceled instead of consuming the entire run. A
// generous MaxContentSize keeps the run WITHIN cap so the spool loop keeps
// draining (an over-cap run would hard-fail before cancellation could fire); the
// reader cancels a few bytes into the run and the loop observes ctx.Err() between
// bounded chunks and unwinds.
func TestRCDATASaturatedRefContextCancellation(t *testing.T) {
	const reps = 1 << 16    // long run so the drain is still in progress
	const sizeCap = 1 << 30 // far above reps so the within-cap spool keeps draining

	for _, elem := range []string{tagTitle, tagTextarea} {
		t.Run(elem, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			t.Cleanup(cancel)

			// "&amp" + a long no-';' tail: the saturated path spools the tail
			// (legacy "amp" resolves) and must keep draining to learn whether a ';'
			// terminates it — exactly the loop that must honor cancellation. A meta
			// charset forces the streaming path; the cancel fires once the tail has
			// streamed past the 1024-byte charset prescan and the spool is draining,
			// and throttled reads keep it landing mid-run.
			input := []byte(metaUTF8 + "<" + elem + ">&amp" + strings.Repeat("x", reps))
			r := &cancelAfterReader{data: input, after: 1100, maxRead: 64, cancel: cancel}

			done := make(chan error, 1)
			go func() {
				_, err := html.NewParser().MaxContentSize(sizeCap).ParseReader(ctx, r)
				done <- err
			}()

			select {
			case err := <-done:
				require.ErrorIs(t, err, context.Canceled,
					"cancelled mid-saturated-run parse should return context.Canceled")
			case <-time.After(10 * time.Second):
				t.Fatal("parse did not abort promptly on saturated-run cancellation")
			}
		})
	}
}

// TestRCDATACharRefEmitPathsCapEnforced is the convergent, cross-path regression
// for the legacy-prefix cap bypass (codex 615-27) and a sweep over every
// char-ref emit path in parseCharRefBounded. The documented contract
// (html.go / sax.go) is that a NO-';' legacy or legacy-PREFIX reference is
// exempt from MaxContentSize ONLY when its whole consumed run ("&" + name) fits
// the cap; over a tiny cap it must hard-fail with ErrContentSizeExceeded and
// emit NOTHING — never a partial resolution. Each subtest pins a different emit
// path under a tiny cap and asserts the SAME behavior, so a future tiny-cap
// case in any of them is already covered:
//
//   - legacy_prefix_short: the cited `&ampZ` under MaxContentSize(2) — the
//     SHORT (within-lookahead) successful resolveNamedEntity legacy-PREFIX path.
//     5-byte run > 2 → fail, no "&Z" emitted.
//   - legacy_full_short: `&amp` (no ';') under MaxContentSize(2) — the SHORT
//     successful full-legacy-entity resolve path. 4-byte run > 2 → fail.
//   - unresolved_short: `&zzz` under MaxContentSize(2) — the unresolved literal
//     path. 4-byte run > 2 → fail.
//   - legacy_prefix_saturated: `&amp` + long tail (no ';') under a tiny cap —
//     the saturated/over-cap ambiguous legacy-prefix path. Hard-fail.
//
// The within-cap sibling for each resolving path (legacy resolves and emits when
// the run fits) is already pinned by TestRCDATAWithinCapNamedEntity and
// TestRCDATAWithinCapSaturatedLegacyResolves; this test locks the over-cap halves
// together so the cap is enforced uniformly BEFORE any emit.
func TestRCDATACharRefEmitPathsCapEnforced(t *testing.T) {
	const limit = 2

	cases := []struct {
		name string
		body string // RCDATA content under the tiny cap
	}{
		{"legacy_prefix_short", "&ampZ"},                              // resolveNamedEntity legacy-prefix success path
		{"legacy_full_short", "&amp"},                                 // resolveNamedEntity full-legacy success path
		{"unresolved_short", "&zzz"},                                  // unresolved literal path
		{"legacy_prefix_saturated", "&amp" + strings.Repeat("x", 40)}, // saturated ambiguous legacy-prefix path
	}

	for _, elem := range []string{tagTitle, tagTextarea} {
		for _, tc := range cases {
			t.Run(elem+"_"+tc.name, func(t *testing.T) {
				input := "<" + elem + ">" + tc.body + "</" + elem + ">"

				var events [][]byte
				record := html.CharactersFunc(func(data []byte) error {
					cp := make([]byte, len(data))
					copy(cp, data)
					events = append(events, cp)
					return nil
				})
				sax := &html.SAXCallbacks{}
				sax.SetOnCharacters(record)
				sax.SetOnCDataBlock(html.CDataBlockFunc(record))

				err := html.NewParser().MaxContentSize(limit).
					ParseWithSAX(t.Context(), []byte(input), sax)
				require.ErrorIs(t, err, html.ErrContentSizeExceeded,
					"%s: over-cap char-ref run must hard-fail with ErrContentSizeExceeded", tc.name)
				require.Empty(t, events,
					"%s: no Characters callback may be delivered before the error (no partial emit); got %q", tc.name, events)
			})
		}
	}
}

// TestRCDATANumericRefExemptFromCap pins that a numeric character reference is a
// resolved character reference (one rune, always emitted intact) and therefore
// EXEMPT from MaxContentSize even when the resolved rune is larger than the cap —
// matching the documented "a single rune larger than the cap is emitted whole"
// rule. `&#128512;` resolves to a 4-byte UTF-8 emoji under MaxContentSize(2); it
// must succeed and emit the whole rune, NOT hard-fail. This locks the numeric
// emit path on the exempt side of the unified cap behavior so it is not
// over-eagerly charged when the legacy/literal paths are tightened.
func TestRCDATANumericRefExemptFromCap(t *testing.T) {
	const limit = 2

	cases := []struct {
		name string
		body string
		want string
	}{
		{"decimal_emoji", "&#128512;", "\U0001F600"}, // 4-byte rune > cap, exempt
		{"hex_emoji", "&#x1F600;", "\U0001F600"},
		{"ascii_A", "&#65;", "A"},
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
					"%s: a resolved numeric reference must be exempt from the cap", tc.name)
				require.Equal(t, tc.want, got.String(),
					"%s: the resolved rune must be emitted whole", tc.name)
			})
		}
	}
}
