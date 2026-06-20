package shim_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/lestrrat-go/helium/shim"
	"github.com/stretchr/testify/require"
)

// TestNilReaderReturnsError verifies NewDecoder with a nil io.Reader returns
// an error from Token rather than panicking on the nil reader.
func TestNilReaderReturnsError(t *testing.T) {
	require.NotPanics(t, func() {
		d := shim.NewDecoder(context.Background(), nil)
		_, err := d.Token()
		require.Error(t, err, "nil reader should yield an error")
	})
}

// TestNilTokenReaderReturnsError verifies NewTokenDecoder with a nil
// TokenReader returns an error from Token rather than panicking.
func TestNilTokenReaderReturnsError(t *testing.T) {
	require.NotPanics(t, func() {
		d := shim.NewTokenDecoder(context.Background(), nil)
		_, err := d.Token()
		require.Error(t, err, "nil TokenReader should yield an error")
	})
}

// nilNilTokenReader always returns (nil, nil), which previously caused an
// infinite spin in the decoder.
type nilNilTokenReader struct{}

func (nilNilTokenReader) Token() (shim.Token, error) { return nil, nil } //nolint:nilnil // exercises the (nil, nil) spin guard

// TestTokenReaderNilNilTerminates verifies a TokenReader that always returns
// (nil, nil) terminates with an error instead of spinning forever, once the
// bounded no-progress guard is exceeded.
func TestTokenReaderNilNilTerminates(t *testing.T) {
	done := make(chan error, 1)
	go func() {
		d := shim.NewTokenDecoder(context.Background(), nilNilTokenReader{})
		_, err := d.Token()
		done <- err
	}()
	select {
	case err := <-done:
		require.Error(t, err, "(nil, nil) TokenReader should yield an error")
	case <-time.After(5 * time.Second):
		t.Fatal("decoder spun forever on (nil, nil) TokenReader")
	}
}

// transientNilNilTokenReader returns (nil, nil) a few times before yielding a
// real result, exercising the documented "no token available yet" case.
type transientNilNilTokenReader struct {
	nilNils   int
	remaining int
}

func (r *transientNilNilTokenReader) Token() (shim.Token, error) { //nolint:nilnil // exercises transient (nil, nil) tolerance
	if r.remaining > 0 {
		r.remaining--
		return nil, nil
	}
	r.nilNils++
	if r.nilNils == 1 {
		return shim.CharData([]byte("ok")), nil
	}
	return nil, io.EOF
}

// TestTokenReaderTransientNilNil verifies a TokenReader returning (nil, nil) a
// few times (under the no-progress bound) before producing a real token does
// not error, preserving the encoding/xml.TokenReader transient-(nil,nil)
// contract.
func TestTokenReaderTransientNilNil(t *testing.T) {
	done := make(chan struct {
		tok shim.Token
		err error
	}, 1)
	go func() {
		d := shim.NewTokenDecoder(context.Background(), &transientNilNilTokenReader{remaining: 5})
		tok, err := d.Token()
		done <- struct {
			tok shim.Token
			err error
		}{tok, err}
	}()
	select {
	case res := <-done:
		require.NoError(t, res.err, "transient (nil, nil) should be retried, not errored")
		require.NotNil(t, res.tok, "expected a real token after transient (nil, nil)")
	case <-time.After(5 * time.Second):
		t.Fatal("decoder spun forever on transient (nil, nil) TokenReader")
	}
}
