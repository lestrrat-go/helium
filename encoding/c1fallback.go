package encoding

// c1fallback.go provides wrappers for single-byte charmap encodings where
// Go's x/text library (following the WHATWG spec) maps bytes 0x80-0x9F to
// U+FFFD instead of C1 control characters U+0080-U+009F.
//
// libxml2 (which uses iconv) follows the IANA/ISO standard where these bytes
// map to the corresponding C1 control code points. Since the RuneCursor in
// github.com/lestrrat-go/strcursor treats U+FFFD (== utf8.RuneError) as a
// decode failure, we need to avoid emitting U+FFFD for these bytes.
//
// This file provides:
// - c1Encoding: wraps an enc.Encoding to add C1 fallback support
// - c1DecoderTransformer: wraps a decoder Transformer, post-processing its
//   output to replace U+FFFD with the C1 character corresponding to the
//   original byte
// - c1EncoderTransformer: wraps an encoder Transformer, pre-processing its
//   input to pass C1 characters (U+0080-U+009F) directly as bytes 0x80-0x9F

import (
	"unicode/utf8"

	enc "golang.org/x/text/encoding"
	"golang.org/x/text/transform"
)

// c1Encoding wraps a charmap Encoding to provide C1 control character support.
// Bytes 0x80-0x9F are mapped to/from U+0080-U+009F instead of U+FFFD.
type c1Encoding struct {
	enc.Encoding
}

func withC1Fallback(e enc.Encoding) enc.Encoding {
	return &c1Encoding{Encoding: e}
}

func (e *c1Encoding) NewDecoder() *enc.Decoder {
	return &enc.Decoder{Transformer: &c1DecoderTransformer{
		base: e.Encoding.NewDecoder().Transformer,
	}}
}

func (e *c1Encoding) NewEncoder() *enc.Encoder {
	return &enc.Encoder{Transformer: &c1EncoderTransformer{
		base: e.Encoding.NewEncoder().Transformer,
	}}
}

// c1DecoderTransformer wraps a decoder transformer. After the base decoder
// produces output, it scans for U+FFFD sequences and replaces them with the
// UTF-8 encoding of the original byte value (treating it as a C1 code point).
type c1DecoderTransformer struct {
	base transform.Transformer
}

func (t *c1DecoderTransformer) Reset() {
	t.base.Reset()
}

func (t *c1DecoderTransformer) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	// For single-byte charmap encodings, each source byte produces one
	// Unicode character in UTF-8. We process one byte at a time to maintain
	// the mapping between source bytes and output characters.
	for nSrc < len(src) {
		b := src[nSrc]

		// ASCII bytes pass through unchanged
		if b < 0x80 {
			if nDst >= len(dst) {
				err = transform.ErrShortDst
				return
			}
			dst[nDst] = b
			nDst++
			nSrc++
			continue
		}

		// For bytes 0x80-0x9F, check if the base decoder maps them to U+FFFD.
		// If so, use the byte value as the Unicode code point instead.
		if b <= 0x9F {
			// Decode single byte through the base decoder
			var tmpDst [4]byte
			nd, ns, _ := t.base.Transform(tmpDst[:], src[nSrc:nSrc+1], atEOF && nSrc+1 == len(src))
			if ns == 0 {
				// Base decoder couldn't consume the byte, treat as C1
				r := rune(b)
				n := utf8.EncodeRune(tmpDst[:], r)
				if nDst+n > len(dst) {
					err = transform.ErrShortDst
					return
				}
				copy(dst[nDst:], tmpDst[:n])
				nDst += n
				nSrc++
				continue
			}

			// Check if the decoder produced U+FFFD
			if nd == 3 && tmpDst[0] == 0xef && tmpDst[1] == 0xbf && tmpDst[2] == 0xbd {
				// Replace with the C1 code point (same as the byte value)
				r := rune(b)
				n := utf8.EncodeRune(tmpDst[:], r)
				if nDst+n > len(dst) {
					err = transform.ErrShortDst
					return
				}
				copy(dst[nDst:], tmpDst[:n])
				nDst += n
				nSrc += ns
				continue
			}

			// The base decoder produced a valid (non-FFFD) character
			if nDst+nd > len(dst) {
				err = transform.ErrShortDst
				return
			}
			copy(dst[nDst:], tmpDst[:nd])
			nDst += nd
			nSrc += ns
			continue
		}

		// For bytes 0xA0-0xFF, use the base decoder
		var tmpDst [4]byte
		nd, ns, derr := t.base.Transform(tmpDst[:], src[nSrc:nSrc+1], atEOF && nSrc+1 == len(src))
		if ns == 0 {
			if derr == transform.ErrShortDst || derr == transform.ErrShortSrc {
				err = derr
			} else {
				// Fallback: use byte value as code point
				r := rune(b)
				n := utf8.EncodeRune(tmpDst[:], r)
				if nDst+n > len(dst) {
					err = transform.ErrShortDst
					return
				}
				copy(dst[nDst:], tmpDst[:n])
				nDst += n
				nSrc++
				continue
			}
			return
		}
		if nDst+nd > len(dst) {
			err = transform.ErrShortDst
			return
		}
		copy(dst[nDst:], tmpDst[:nd])
		nDst += nd
		nSrc += ns
	}
	return
}

// c1EncoderTransformer wraps an encoder transformer. It handles C1 control
// characters (U+0080-U+009F) by writing them directly as bytes 0x80-0x9F
// instead of passing them through the base encoder (which would error).
type c1EncoderTransformer struct {
	base transform.Transformer
}

func (t *c1EncoderTransformer) Reset() {
	t.base.Reset()
}

func (t *c1EncoderTransformer) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	for nSrc < len(src) {
		// Decode the next rune from the UTF-8 source
		if src[nSrc] < utf8.RuneSelf {
			// ASCII byte - pass through to base encoder
			if nDst >= len(dst) {
				err = transform.ErrShortDst
				return
			}
			dst[nDst] = src[nSrc]
			nDst++
			nSrc++
			continue
		}

		r, size := utf8.DecodeRune(src[nSrc:])
		if r == utf8.RuneError && size <= 1 {
			if !atEOF && !utf8.FullRune(src[nSrc:]) {
				err = transform.ErrShortSrc
				return
			}
			// Invalid UTF-8, pass through to base encoder to handle
			nd, ns, eerr := t.base.Transform(dst[nDst:], src[nSrc:nSrc+1], atEOF && nSrc+1 == len(src))
			nDst += nd
			nSrc += ns
			if eerr != nil {
				err = eerr
				return
			}
			continue
		}

		// C1 control character? Write directly as the byte value.
		if r >= 0x80 && r <= 0x9F {
			if nDst >= len(dst) {
				err = transform.ErrShortDst
				return
			}
			dst[nDst] = byte(r)
			nDst++
			nSrc += size
			continue
		}

		// For all other runes, use the base encoder
		nd, ns, eerr := t.base.Transform(dst[nDst:], src[nSrc:nSrc+size], atEOF && nSrc+size >= len(src))
		nDst += nd
		nSrc += ns
		if eerr != nil {
			err = eerr
			return
		}
	}
	return
}
