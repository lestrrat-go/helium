package html_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/lestrrat-go/helium/html"
	"github.com/stretchr/testify/require"
)

// TestRawTextContextCancellationAborts verifies that cancelling the context
// aborts a long, unterminated raw-text/RCDATA/plaintext/comment section
// promptly instead of buffering the whole thing until EOF.
func TestRawTextContextCancellationAborts(t *testing.T) {
	// ~64 MiB of content per section: large enough that buffering it all
	// would be observable, small enough to keep the test fast.
	const size = 64 << 20
	body := strings.Repeat("a", size)

	cases := []struct {
		name  string
		input string
	}{
		{"script", "<script>" + body},
		{"style", "<style>" + body},
		{"textarea", "<textarea>" + body},
		{"title", "<title>" + body},
		{"plaintext", "<plaintext>" + body},
		{"comment", "<!--" + body},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			cancel() // already cancelled before parsing starts

			done := make(chan error, 1)
			go func() {
				_, err := html.NewParser().Parse(ctx, []byte(tc.input))
				done <- err
			}()

			select {
			case err := <-done:
				require.ErrorIs(t, err, context.Canceled,
					"cancelled parse should return context.Canceled")
			case <-time.After(10 * time.Second):
				t.Fatal("parse did not abort promptly on context cancellation")
			}
		})
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
		{"textarea", "<textarea>", "</textarea>"},
		{"plaintext", "<plaintext>", ""},
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
		{"comment", "<!--" + body + "--><p>x</p>"},
		{"bogus_comment", "<!" + body + "><p>x</p>"},
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
		{"comment", "<!--hello world-->", "hello world"},
		{"bogus_comment", "<!bogus content>", "bogus content"},
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
		{"comment", "<!--abcd-->", "abcd", "<!--abcde-->"},
		{"bogus_comment", "<!abcd>", "abcd", "<!abcde>"},
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
