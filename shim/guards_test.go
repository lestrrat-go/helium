package shim_test

import (
	"context"
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

// TestTokenReaderNilNilTerminates verifies a TokenReader returning (nil, nil)
// terminates with an error instead of spinning forever.
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
