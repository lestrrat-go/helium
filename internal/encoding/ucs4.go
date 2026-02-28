package encoding

import (
	enc "golang.org/x/text/encoding"
	"golang.org/x/text/encoding/unicode/utf32"
	"golang.org/x/text/transform"
)

// ucs4SwapEncoding handles the unusual UCS-4 byte orders (2143 and 3412)
// by byte-swapping to/from standard UTF-32BE.
type ucs4SwapEncoding struct {
	swap func(dst, src []byte)
}

func (e *ucs4SwapEncoding) NewDecoder() *enc.Decoder {
	return &enc.Decoder{Transformer: transform.Chain(
		&ucs4SwapTransformer{swap: e.swap},
		utf32.UTF32(utf32.BigEndian, utf32.IgnoreBOM).NewDecoder().Transformer,
	)}
}

func (e *ucs4SwapEncoding) NewEncoder() *enc.Encoder {
	return &enc.Encoder{Transformer: transform.Chain(
		utf32.UTF32(utf32.BigEndian, utf32.IgnoreBOM).NewEncoder().Transformer,
		&ucs4SwapTransformer{swap: e.swap},
	)}
}

// ucs4SwapTransformer reorders bytes within each 4-byte group.
type ucs4SwapTransformer struct {
	swap func(dst, src []byte)
}

func (t *ucs4SwapTransformer) Reset() {}

func (t *ucs4SwapTransformer) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	for nSrc+4 <= len(src) {
		if nDst+4 > len(dst) {
			err = transform.ErrShortDst
			return
		}
		t.swap(dst[nDst:], src[nSrc:])
		nDst += 4
		nSrc += 4
	}
	if nSrc < len(src) && !atEOF {
		err = transform.ErrShortSrc
	}
	return
}

// swap2143 converts between UCS-4 byte order 2143 and big-endian (1234).
// This swap is self-inverse: applying it twice yields the original bytes.
func swap2143(dst, src []byte) {
	dst[0] = src[1]
	dst[1] = src[0]
	dst[2] = src[3]
	dst[3] = src[2]
}

// swap3412 converts between UCS-4 byte order 3412 and big-endian (1234).
// This swap is self-inverse: applying it twice yields the original bytes.
func swap3412(dst, src []byte) {
	dst[0] = src[2]
	dst[1] = src[3]
	dst[2] = src[0]
	dst[3] = src[1]
}
