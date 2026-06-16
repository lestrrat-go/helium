package encoding

// strict.go enforces that decoder-inserted U+FFFD replacement characters are
// treated as fatal decode errors, while genuinely-encoded U+FFFD characters
// pass through unchanged.
//
// golang.org/x/text decoders (UTF-16, UTF-32, UCS-4, ...) silently substitute
// U+FFFD for malformed input — e.g. an unpaired UTF-16 surrogate or a trailing
// odd byte — instead of returning an error. After transcoding into the internal
// UTF-8 buffer, that decoder-inserted U+FFFD is byte-identical to a real,
// legitimately-encoded U+FFFD. The XML parser's char-data scanner accepts a
// genuine U+FFFD (it is a valid XML character), so without this wrapper a
// UTF-16 document containing an unpaired surrogate would parse successfully
// even though it is malformed input, which XML requires to be a fatal error.
//
// This wrapper applies only to the fixed-width Unicode encodings (UTF-16 / 2
// bytes, UTF-32 & UCS-4 / 4 bytes, UCS-2 / 2 bytes). For those, a genuine
// U+FFFD is exactly the code unit whose bytes equal the U+FFFD encoding. The
// wrapper decodes normally, then compares the number of U+FFFD characters in
// the decoded output against the number of genuine U+FFFD code units present in
// the source. Any excess means the decoder substituted U+FFFD for malformed
// input, which is reported as a fatal decode error.

import (
	"bytes"
	"errors"

	enc "golang.org/x/text/encoding"
	"golang.org/x/text/transform"
)

// ErrInvalidEncodedChar is returned when a decoder substitutes U+FFFD for
// malformed input (for example an unpaired UTF-16 surrogate).
var ErrInvalidEncodedChar = errors.New("encoding: malformed input substituted with U+FFFD")

// utf8FFFD is the UTF-8 encoding of U+FFFD.
var utf8FFFD = []byte{0xEF, 0xBF, 0xBD}

// withStrictDecode wraps a fixed-width Unicode Encoding so that its decoder
// rejects malformed input that the base decoder would otherwise silently
// replace with U+FFFD. The encoder is left unchanged.
func withStrictDecode(e enc.Encoding) enc.Encoding {
	return &strictEncoding{Encoding: e}
}

type strictEncoding struct {
	enc.Encoding
}

func (e *strictEncoding) NewDecoder() *enc.Decoder {
	// unitFFFD holds the source-byte encoding of a genuine U+FFFD code unit for
	// this encoding (e.g. 0xFF 0xFD for UTF-16BE). Its length is the encoding's
	// fixed code-unit width. If the encoder cannot represent U+FFFD, every
	// emitted U+FFFD is treated as a substitution error.
	var unitFFFD []byte
	if b, err := e.Encoding.NewEncoder().Bytes(utf8FFFD); err == nil {
		unitFFFD = b
	}
	return &enc.Decoder{Transformer: &strictDecoderTransformer{
		base:     e.Encoding.NewDecoder().Transformer,
		unitFFFD: unitFFFD,
	}}
}

// strictDecoderTransformer wraps a fixed-width Unicode decoder, decoding via the
// base transformer and then rejecting any U+FFFD in the output that does not
// correspond to a genuine U+FFFD code unit in the source.
type strictDecoderTransformer struct {
	base     transform.Transformer
	unitFFFD []byte
}

func (t *strictDecoderTransformer) Reset() {
	t.base.Reset()
}

func (t *strictDecoderTransformer) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	nDst, nSrc, err = t.base.Transform(dst, src, atEOF)
	if err != nil && err != transform.ErrShortSrc {
		return nDst, nSrc, err
	}

	emitted := bytes.Count(dst[:nDst], utf8FFFD)
	if emitted == 0 {
		return nDst, nSrc, err
	}

	if t.countGenuineFFFDUnits(src[:nSrc]) < emitted {
		return nDst, nSrc, ErrInvalidEncodedChar
	}
	return nDst, nSrc, err
}

// countGenuineFFFDUnits counts the number of genuine U+FFFD code units in the
// consumed source. Units are scanned at the encoding's fixed code-unit width.
// A genuine U+FFFD never overlaps a multi-unit sequence (e.g. a UTF-16
// surrogate pair never contains the 0xFFFD code unit), so width-aligned
// scanning attributes every genuine U+FFFD exactly once.
func (t *strictDecoderTransformer) countGenuineFFFDUnits(src []byte) int {
	w := len(t.unitFFFD)
	if w == 0 {
		return 0
	}
	count := 0
	for off := 0; off+w <= len(src); off += w {
		if bytes.Equal(src[off:off+w], t.unitFFFD) {
			count++
		}
	}
	return count
}
