package helium

import (
	"bytes"
	"strings"
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

func TestParseNCNameReportsInvalidStartRune(t *testing.T) {
	pctx := newNameTestParserCtx(t, []byte{'1', 'a'})

	_, err := pctx.parseNCName(t.Context())
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid name start char '1' (U+0031)")
	require.True(t, strings.Contains(err.Error(), "invalid name start char"), err.Error())
}
