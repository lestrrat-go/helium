package shim_test

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/shim"
)

// byteOnlyReader implements io.Reader and io.ByteReader. When passed to
// ensureReader via CharsetReader, the decoder wraps it in a byteReaderWrapper
// that drives ReadByte, exercising byteReaderWrapper.Read.
type byteOnlyReader struct {
	r *bytes.Reader
}

func (b *byteOnlyReader) Read(p []byte) (int, error) { return b.r.Read(p) }
func (b *byteOnlyReader) ReadByte() (byte, error)    { return b.r.ReadByte() }

func TestDecoder(t *testing.T) {
	t.Run("charset-reader-byte-reader", func(t *testing.T) {
		input := `<?xml version="1.0" encoding="latin1"?><root>hi</root>`
		dec := shim.NewDecoder(context.Background(), strings.NewReader(input))
		dec.CharsetReader = func(charset string, in io.Reader) (io.Reader, error) {
			if charset != "latin1" {
				t.Errorf("unexpected charset %q", charset)
			}
			// Drain the converted input and return a ByteReader-only wrapper so
			// the decoder must adapt it via byteReaderWrapper.
			data, err := io.ReadAll(in)
			if err != nil {
				return nil, err
			}
			return &byteOnlyReader{r: bytes.NewReader(data)}, nil
		}

		var sawRoot bool
		for {
			tok, err := dec.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("Token error: %v", err)
			}
			if se, ok := tok.(shim.StartElement); ok && se.Name.Local == "root" {
				sawRoot = true
			}
		}
		if !sawRoot {
			t.Fatal("did not see <root> element")
		}
	})

	t.Run("close", func(t *testing.T) {
		dec := shim.NewDecoder(context.Background(), strings.NewReader(`<root><a/><b/></root>`))
		// Read one token to start the SAX goroutine, then Close to cancel it.
		if _, err := dec.Token(); err != nil {
			t.Fatalf("Token error: %v", err)
		}
		dec.Close()
		// Closing again must be safe (cancel is idempotent).
		dec.Close()
	})

	t.Run("close-before-read", func(t *testing.T) {
		dec := shim.NewDecoder(context.Background(), strings.NewReader(`<root/>`))
		// Close before any read: cancel is non-nil but goroutine never started.
		dec.Close()
	})

	t.Run("input-pos", func(t *testing.T) {
		dec := shim.NewDecoder(context.Background(), strings.NewReader(`<root>text</root>`))
		// Before reading, line is 1 -> returns (1,1).
		if l, c := dec.InputPos(); l != 1 || c != 1 {
			t.Fatalf("expected (1,1) before read, got (%d,%d)", l, c)
		}
		for {
			if _, err := dec.Token(); err != nil {
				break
			}
		}
		// After reading content, positions should be reported (line >= 1).
		if l, _ := dec.InputPos(); l < 1 {
			t.Fatalf("expected line >= 1, got %d", l)
		}
	})
}
