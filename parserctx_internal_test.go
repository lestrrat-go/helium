package helium

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func newNameTestParserCtx(t *testing.T, input []byte) *parserCtx {
	t.Helper()

	pctx := &parserCtx{rawInput: input}
	require.NoError(t, pctx.init(nil, bytes.NewReader(input)))
	t.Cleanup(func() {
		require.NoError(t, pctx.release())
	})
	return pctx
}

func TestParseNameRejectsInvalidUTF8InContinuation(t *testing.T) {
	pctx := newNameTestParserCtx(t, []byte{'r', 'o', 0xff, 'o', 't'})

	_, err := pctx.parseName(t.Context())
	require.ErrorIs(t, err, errInvalidUTF8Name)
}

func TestParseNCNameRejectsInvalidUTF8InContinuation(t *testing.T) {
	pctx := newNameTestParserCtx(t, []byte{'a', 't', 0xff, 'r'})

	_, err := pctx.parseNCName(t.Context())
	require.ErrorIs(t, err, errInvalidUTF8Name)
}
