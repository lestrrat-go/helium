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
			ctx, cancel := context.WithCancel(context.Background())
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
			err := p.ParseWithSAX(context.Background(), []byte(input), sax)
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
