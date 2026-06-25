package c14n_test

import (
	"errors"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/c14n"
	"github.com/stretchr/testify/require"
)

// failWriter writes successfully for the first `limit` bytes, then returns an
// error. By varying limit, tests can trip individual io.WriteString / escape
// error-return branches in the canonicalizer that are otherwise unreachable
// through an in-memory bytes.Buffer.
type failWriter struct {
	limit   int
	written int
}

var errFailWriter = errors.New("failWriter: limit reached")

func (w *failWriter) Write(p []byte) (int, error) {
	remaining := w.limit - w.written
	if remaining <= 0 {
		return 0, errFailWriter
	}
	if len(p) <= remaining {
		w.written += len(p)
		return len(p), nil
	}
	w.written += remaining
	return remaining, errFailWriter
}

func parseDoc(t *testing.T, src string) *helium.Document {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err, "parse %q", src)
	return doc
}

// TestWriterErrors exercises writer-error propagation through the canonicalizer.
func TestWriterErrors(t *testing.T) {
	// propagate runs canonicalization against a writer that fails at every byte
	// boundary from 0 up to the full output length. Every failure must surface
	// as an error rather than a panic or silent success. This walks the
	// writer-error return branches throughout the canonicalizer.
	t.Run("propagate", func(t *testing.T) {
		docs := map[string]string{
			"element-attrs": `<?xml version="1.0"?>` +
				`<r xmlns="urn:d" xmlns:a="urn:a" a:x="1" b="two&amp;more"><c>te&lt;xt</c></r>`,
			"pi-comment-around-root": `<?xml version="1.0"?>` +
				`<?pi-before data?>` + "\n" +
				`<!-- comment before -->` + "\n" +
				`<root>body</root>` + "\n" +
				`<?pi-after more?>` + "\n" +
				`<!-- comment after -->`,
			"cdata":             `<r><![CDATA[a<b & c]]></r>`,
			"pi-comment-inline": `<r><?inline pidata?><!-- inline comment -->text</r>`,
			"pi-no-data":        `<r><?bare?></r>`,
		}

		for name, src := range docs {
			t.Run(name, func(t *testing.T) {
				doc := parseDoc(t, src)

				full, err := c14n.NewCanonicalizer(c14n.C14N11).Comments().CanonicalizeTo(doc)
				require.NoError(t, err, "baseline canonicalize")
				require.NotEmpty(t, full)

				for limit := range full {
					w := &failWriter{limit: limit}
					err := c14n.NewCanonicalizer(c14n.C14N11).Comments().Canonicalize(doc, w)
					require.Error(t, err, "limit=%d should error", limit)
				}

				// Full length must succeed.
				w := &failWriter{limit: len(full)}
				require.NoError(t, c14n.NewCanonicalizer(c14n.C14N11).Comments().Canonicalize(doc, w))
			})
		}
	})

	// exclusive exercises the exclusive-mode namespace rendering writer-error
	// branches.
	t.Run("exclusive", func(t *testing.T) {
		doc := parseDoc(t, `<r xmlns:a="urn:a" xmlns:b="urn:b" a:x="1"><a:c b:y="2">t</a:c></r>`)

		full, err := c14n.NewCanonicalizer(c14n.ExclusiveC14N10).
			InclusiveNamespaces([]string{"b"}).
			CanonicalizeTo(doc)
		require.NoError(t, err)
		require.NotEmpty(t, full)

		for limit := range full {
			w := &failWriter{limit: limit}
			err := c14n.NewCanonicalizer(c14n.ExclusiveC14N10).
				InclusiveNamespaces([]string{"b"}).
				Canonicalize(doc, w)
			require.Error(t, err, "limit=%d should error", limit)
		}
	})
}
