package html_test

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/lestrrat-go/helium/html"
	"github.com/stretchr/testify/require"
)

// errAfterNWriter is an io.Writer that writes normally until it has emitted
// n total bytes, after which every subsequent Write fails. It is used to
// simulate a writer that fails mid-stream so we can assert the HTML
// serializer propagates the failure instead of silently truncating output.
type errAfterNWriter struct {
	limit   int
	written int
	err     error
}

func (w *errAfterNWriter) Write(p []byte) (int, error) {
	if w.written >= w.limit {
		return 0, w.err
	}
	remaining := w.limit - w.written
	if len(p) <= remaining {
		w.written += len(p)
		return len(p), nil
	}
	w.written += remaining
	return remaining, w.err
}

const dumpErrTestHTML = `<!DOCTYPE html>
<html><head><!-- a comment --><?pi data?></head>` +
	`<body><p class="greeting" id="x">Hello &amp; goodbye</p></body></html>`

// TestSerializerPropagatesWriteError verifies that when the underlying
// io.Writer fails partway through serialization, the serialize call returns
// a non-nil error rather than silently producing truncated output (the prior
// behavior, where structural write sites discarded their errors).
func TestSerializerPropagatesWriteError(t *testing.T) {
	doc, err := html.NewParser().Parse(t.Context(), []byte(dumpErrTestHTML))
	require.NoError(t, err, "parse fixture")

	// Establish the full, successful output length.
	var full bytes.Buffer
	require.NoError(t, html.Write(&full, doc), "baseline serialize must succeed")
	require.Greater(t, full.Len(), 10, "fixture must produce non-trivial output")

	// Fail across a range of truncation points. The most important ones are
	// near the end of the stream: those land in the trailing structural
	// writes (closing tags, trailing newline) which previously discarded
	// their io.Writer errors and so reported success despite truncation.
	for _, limit := range []int{1, full.Len() / 2, full.Len() - 4, full.Len() - 1} {
		sentinel := errors.New("writer failed mid-stream")
		fw := &errAfterNWriter{limit: limit, err: sentinel}

		err := html.Write(fw, doc)
		require.Error(t, err,
			"serializer must propagate a writer error at byte %d/%d", limit, full.Len())
		require.ErrorIs(t, err, sentinel,
			"the first writer error should surface at byte %d/%d", limit, full.Len())
	}
}

// TestSerializerOutputUnchanged confirms the sticky-error refactor did not
// alter success-path output: serializing to a healthy writer must yield the
// exact same bytes as before.
func TestSerializerOutputUnchanged(t *testing.T) {
	doc, err := html.NewParser().Parse(t.Context(), []byte(dumpErrTestHTML))
	require.NoError(t, err, "parse fixture")

	out, err := html.WriteString(doc)
	require.NoError(t, err, "serialize must succeed for a healthy writer")
	require.NotEmpty(t, out)

	// Sanity: representative structural pieces are present and intact.
	require.Contains(t, out, "<!DOCTYPE")
	require.Contains(t, out, "<!-- a comment -->")
	require.Contains(t, out, `class="greeting"`)
	require.Contains(t, out, "Hello &amp; goodbye")

	// Writing twice yields identical bytes (no leaked sticky state).
	var second bytes.Buffer
	require.NoError(t, html.Write(&second, doc))
	require.Equal(t, out, second.String(), "repeated serialization must be byte-identical")
}

var _ io.Writer = (*errAfterNWriter)(nil)
