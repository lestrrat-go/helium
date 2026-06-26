package shim

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

// TestScanPrologSizeBound verifies that a prolog larger than maxPrologSize
// (here, an oversized comment ahead of the root element) is rejected with
// errPrologTooLarge instead of being buffered unboundedly.
func TestScanPrologSizeBound(t *testing.T) {
	t.Run("oversized prolog comment fails", func(t *testing.T) {
		var b bytes.Buffer
		b.WriteString("<!--")
		b.Write(bytes.Repeat([]byte("a"), maxPrologSize+16))
		b.WriteString("--><root/>")

		_, _, _, err := scanProlog(bytes.NewReader(b.Bytes()))
		if !errors.Is(err, errPrologTooLarge) {
			t.Fatalf("expected errPrologTooLarge, got %v", err)
		}
	})

	t.Run("small prolog succeeds", func(t *testing.T) {
		in := "<!-- hello --><root/>"
		_, _, _, err := scanProlog(strings.NewReader(in))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("decoder surfaces size error", func(t *testing.T) {
		var b bytes.Buffer
		b.WriteString("<!--")
		b.Write(bytes.Repeat([]byte("a"), maxPrologSize+16))
		b.WriteString("--><root/>")

		dec := NewDecoder(context.Background(), bytes.NewReader(b.Bytes()))
		var lastErr error
		for {
			_, err := dec.Token()
			if err != nil {
				lastErr = err
				break
			}
		}
		if !errors.Is(lastErr, errPrologTooLarge) {
			t.Fatalf("expected errPrologTooLarge from Decoder, got %v", lastErr)
		}
		if lastErr == io.EOF {
			t.Fatal("decoder reached EOF without reporting size error")
		}
	})
}
