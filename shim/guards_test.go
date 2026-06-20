package shim_test

import (
	"context"
	stdxml "encoding/xml"
	"io"
	"testing"
	"time"

	"github.com/lestrrat-go/helium/shim"
	"github.com/stretchr/testify/require"
)

// nilNilTokenReader always returns (nil, nil).
type nilNilTokenReader struct{}

func (nilNilTokenReader) Token() (shim.Token, error) { return nil, nil } //nolint:nilnil // exercises (nil, nil) stdlib pass-through

// TestTokenReaderNilNilPassThrough verifies a single Decoder.Token() call on a
// (nil, nil) TokenReader returns (nil, nil) verbatim, matching encoding/xml
// exactly. Per the TokenReader contract, (nil, nil) means "nothing happened"
// (not EOF, not an error); stdlib's Decoder.Token() passes it straight to the
// caller, and so must the shim to stay drop-in compatible.
func TestTokenReaderNilNilPassThrough(t *testing.T) {
	d := shim.NewTokenDecoder(context.Background(), nilNilTokenReader{})
	tok, err := d.Token()
	require.NoError(t, err, "(nil, nil) is not an error")
	require.Nil(t, tok, "(nil, nil) token should pass straight through")
}

// TestStdlibNilNilPassThroughParity pins the shim's (nil, nil) behavior to
// encoding/xml's: a single Token() call returns (nil, nil) verbatim.
func TestStdlibNilNilPassThroughParity(t *testing.T) {
	// shim
	shimTok, shimErr := shim.NewTokenDecoder(context.Background(), nilNilTokenReader{}).Token()

	// stdlib
	stdTok, stdErr := stdxml.NewTokenDecoder(stdNilNilReader{}).Token()

	require.Equal(t, stdTok, shimTok, "token should match stdlib")
	require.Equal(t, stdErr, shimErr, "error should match stdlib")
}

// stdNilNilReader is an encoding/xml.TokenReader that always returns (nil, nil).
type stdNilNilReader struct{}

func (stdNilNilReader) Token() (stdxml.Token, error) { return nil, nil } //nolint:nilnil // stdlib parity comparison

// TestInternalLoopNoProgressTerminates verifies that an internal driving loop
// (here via Decode, which loops calling Token() until a StartElement) does NOT
// hang on a TokenReader that always returns (nil, nil): the bounded no-progress
// guard surfaces io.ErrNoProgress instead.
func TestInternalLoopNoProgressTerminates(t *testing.T) {
	done := make(chan error, 1)
	go func() {
		d := shim.NewTokenDecoder(context.Background(), nilNilTokenReader{})
		var v struct{}
		done <- d.Decode(&v)
	}()
	select {
	case err := <-done:
		require.ErrorIs(t, err, io.ErrNoProgress, "internal loop should fail with io.ErrNoProgress, not hang")
	case <-time.After(5 * time.Second):
		t.Fatal("internal driving loop spun forever on (nil, nil) TokenReader")
	}
}

// TestTokenReaderTransientNilNilDrivenLoop verifies that an internal driving
// loop tolerates transient (nil, nil) reads ("no token available yet") and
// proceeds once a real token is produced, rather than treating (nil, nil) as
// EOF or error. Decode drives Token() in a loop, so this exercises the
// driveToken no-progress retry on the transient case.
func TestTokenReaderTransientNilNilDrivenLoop(t *testing.T) {
	done := make(chan error, 1)
	go func() {
		// remaining transient (nil,nil) reads, then a StartElement so Decode
		// can make progress.
		d := shim.NewTokenDecoder(context.Background(), &transientStartTokenReader{remaining: 5})
		var v struct {
			XMLName shim.Name `xml:"ok"`
		}
		done <- d.Decode(&v)
	}()
	select {
	case err := <-done:
		require.NoError(t, err, "transient (nil, nil) should be retried by the driving loop")
	case <-time.After(5 * time.Second):
		t.Fatal("driving loop spun forever on transient (nil, nil) TokenReader")
	}
}

// transientStartTokenReader emits (nil, nil) a few times, then a complete
// <ok></ok> element so Decode can succeed.
type transientStartTokenReader struct {
	remaining int
	step      int
}

func (r *transientStartTokenReader) Token() (shim.Token, error) {
	if r.remaining > 0 {
		r.remaining--
		return nil, nil //nolint:nilnil // exercises transient (nil, nil) tolerance
	}
	switch r.step {
	case 0:
		r.step++
		return shim.StartElement{Name: shim.Name{Local: "ok"}}, nil
	case 1:
		r.step++
		return shim.EndElement{Name: shim.Name{Local: "ok"}}, nil
	default:
		return nil, io.EOF
	}
}
