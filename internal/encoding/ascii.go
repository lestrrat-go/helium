package encoding

// ascii.go provides a strict US-ASCII decoder. Per the XML specification
// US-ASCII is a 7-bit encoding: every byte must be <= 0x7F. The base
// golang.org/x/text library has no dedicated US-ASCII codec, so US-ASCII used
// to be aliased to UTF-8, which silently accepted multibyte sequences (any
// byte >= 0x80) in a document declaring US-ASCII. This wrapper rejects any
// byte >= 0x80 as a fatal decode error instead.

import (
	"errors"

	enc "golang.org/x/text/encoding"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// ErrInvalidASCII is returned when a document declared as US-ASCII contains a
// byte outside the 7-bit range (>= 0x80).
var ErrInvalidASCII = errors.New("encoding: byte outside US-ASCII range (>= 0x80)")

// asciiEncoding is a strict US-ASCII encoding. Its decoder rejects any byte
// >= 0x80; its encoder delegates to UTF-8 (US-ASCII output is never exercised
// on the parse path and the serializer handles ASCII output separately).
type asciiEncoding struct{}

func (asciiEncoding) NewDecoder() *enc.Decoder {
	return &enc.Decoder{Transformer: asciiDecoder{}}
}

func (asciiEncoding) NewEncoder() *enc.Encoder {
	return unicode.UTF8.NewEncoder()
}

// asciiDecoder copies 7-bit bytes through unchanged and returns
// ErrInvalidASCII on the first byte >= 0x80.
type asciiDecoder struct{}

func (asciiDecoder) Reset() {}

func (asciiDecoder) Transform(dst, src []byte, _ bool) (nDst, nSrc int, err error) {
	for nSrc < len(src) {
		b := src[nSrc]
		if b >= 0x80 {
			return nDst, nSrc, ErrInvalidASCII
		}
		if nDst >= len(dst) {
			return nDst, nSrc, transform.ErrShortDst
		}
		dst[nDst] = b
		nDst++
		nSrc++
	}
	return nDst, nSrc, nil
}
