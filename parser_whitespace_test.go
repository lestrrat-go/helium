package helium

import (
	"context"
	"testing"

	"github.com/lestrrat-go/helium/internal/strcursor"
	"github.com/stretchr/testify/require"
)

// blankThenReadErrReader yields blanks blank bytes and then fails every
// subsequent Read with err. It models the push parser's stream, whose blocking
// Read returns context.Canceled when cancellation unblocks a pending wait: the
// ByteCursor records that as a sticky Err() while PeekAt reports 0 (the same 0 a
// genuine non-blank byte / clean EOF yields).
type blankThenReadErrReader struct {
	blanks int
	served int
	err    error
}

func (r *blankThenReadErrReader) Read(p []byte) (int, error) {
	if r.served < r.blanks {
		n := min(len(p), r.blanks-r.served)
		for i := range n {
			p[i] = ' '
		}
		r.served += n
		return n, nil
	}
	return 0, r.err
}

// TestSkipBlankBytesSurfacesReadError pins the cancellation contract at the
// blank-scan layer: a sticky cursor read error (a push-stream Read returning
// context.Canceled) must be surfaced through pctx.blankRunErr so callers such as
// parseXMLDecl propagate context.Canceled instead of synthesizing a syntax error
// like "blank needed after '<?xml'".
//
// ctx is context.Background() (never cancelled) on purpose: the ONLY signal of
// the cancellation is the cursor's sticky Err(). Before the fix, skipBlankRun
// treated PeekAt==0 as "no blank / EOF" and returned a nil error, so blankRunErr
// was never set and the read failure was masked. ctx.Err() at the top of the
// scan loop cannot rescue this case here because ctx itself is not cancelled.
func TestSkipBlankBytesSurfacesReadError(t *testing.T) {
	cases := map[string]struct {
		blanks       int  // blanks buffered before the read error
		wantAdvanced bool // whether any whitespace was consumed first
	}{
		// First peek already hits the read error (no buffered blank): the
		// i==0 branch must consult the sticky Err() rather than report "no blank".
		"read error on first peek": {blanks: 0, wantAdvanced: false},
		// Some blanks are consumed, then the read error stops the run short of a
		// full chunk: the partial-chunk branch must surface the sticky Err() too.
		"read error after some blanks": {blanks: 3, wantAdvanced: true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			r := &blankThenReadErrReader{blanks: tc.blanks, err: context.Canceled}
			cur := strcursor.NewByteCursor(r)

			pctx := &parserCtx{}
			advanced := pctx.skipBlankBytes(context.Background(), cur)

			require.Equal(t, tc.wantAdvanced, advanced,
				"skipBlankBytes should report whether any whitespace was consumed")
			require.ErrorIs(t, pctx.blankRunErr, context.Canceled,
				"a sticky cursor read error must be surfaced through blankRunErr, not masked as no-blank")
		})
	}
}
