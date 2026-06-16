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
// bytes, UTF-32 & UCS-4 / 4 bytes, UCS-2 / 2 bytes). Because the encoding's
// identity, code-unit width, and resolved byte order are all known at Load()
// time, the wrapper validates the SOURCE bytes deterministically rather than
// counting U+FFFD substitutions in the decoded output. It walks the source one
// fixed-width code unit at a time in the resolved byte order, computes each
// unit's scalar value, and rejects any unit that is malformed:
//
//   - UTF-16: a high surrogate (D800–DBFF) must be immediately followed by a
//     low surrogate (DC00–DFFF); a lone high or lone low surrogate is malformed.
//     A unit equal to 0xFFFD is a genuine U+FFFD and is accepted.
//   - UTF-32 / UCS-4: the scalar must be a valid code point (≤ 0x10FFFF and not
//     a surrogate). 0xFFFD is a genuine U+FFFD and is accepted; anything out of
//     range is malformed.
//   - A trailing incomplete unit (leftover bytes that do not fill a full code
//     unit) is malformed.
//
// For BOM-sensitive encodings (UTF-16/UTF-32 with UseBOM) the leading BOM is
// inspected to fix big- vs little-endian, then skipped from validation. Because
// the decision is made on the source bytes in the resolved byte order, it is
// independent of how many U+FFFD the base decoder emitted, and independent of
// how the decoder chunked its destination buffer (the round-4 ErrShortDst gap).

import (
	"errors"

	enc "golang.org/x/text/encoding"
	"golang.org/x/text/transform"
)

// ErrInvalidEncodedChar is returned when the source contains malformed input
// that the base decoder would otherwise silently replace with U+FFFD (for
// example an unpaired UTF-16 surrogate or an out-of-range UTF-32 scalar).
var ErrInvalidEncodedChar = errors.New("encoding: malformed input substituted with U+FFFD")

// byteOrder describes how to read a fixed-width code unit's scalar value. perm
// maps each destination byte position (most- to least-significant, big-endian)
// to the source byte index within the unit. For example UTF-16LE uses {1, 0}:
// the high-order scalar byte comes from source byte 1.
type byteOrder struct {
	perm []int
}

var (
	orderBE2  = byteOrder{perm: []int{0, 1}}       // UTF-16BE
	orderLE2  = byteOrder{perm: []int{1, 0}}       // UTF-16LE
	orderBE4  = byteOrder{perm: []int{0, 1, 2, 3}} // UTF-32BE / UCS-4 1234
	orderLE4  = byteOrder{perm: []int{3, 2, 1, 0}} // UTF-32LE / UCS-4 4321
	order2143 = byteOrder{perm: []int{1, 0, 3, 2}} // UCS-4 2143
	order3412 = byteOrder{perm: []int{2, 3, 0, 1}} // UCS-4 3412
)

// scalar reads the scalar value of a width-sized unit using this byte order.
// The unit must be at least len(perm) bytes long.
func (o byteOrder) scalar(unit []byte) uint32 {
	var v uint32
	for _, idx := range o.perm {
		v = v<<8 | uint32(unit[idx])
	}
	return v
}

// withStrictDecode wraps a fixed-width Unicode Encoding so that its decoder
// rejects malformed input that the base decoder would otherwise silently
// replace with U+FFFD. width is the encoding's fixed code-unit width in bytes
// (2 for UTF-16 / UCS-2, 4 for UTF-32 / UCS-4). order is the resolved byte
// order for fixed-endianness encodings; for UseBOM encodings it is ignored and
// useBOM must be true so the order is resolved from the leading BOM. The
// encoder is left unchanged.
func withStrictDecode(e enc.Encoding, width int, order byteOrder, useBOM bool) enc.Encoding {
	return &strictEncoding{Encoding: e, width: width, order: order, useBOM: useBOM}
}

type strictEncoding struct {
	enc.Encoding
	width  int
	order  byteOrder
	useBOM bool
}

func (e *strictEncoding) NewDecoder() *enc.Decoder {
	return &enc.Decoder{Transformer: &strictDecoderTransformer{
		base:   e.Encoding.NewDecoder().Transformer,
		width:  e.width,
		order:  e.order,
		useBOM: e.useBOM,
	}}
}

// strictDecoderTransformer wraps a fixed-width Unicode decoder. It decodes via
// the base transformer but, before trusting the output, validates the consumed
// source bytes deterministically in the resolved byte order and rejects any
// malformed unit (which the base decoder would have substituted with U+FFFD).
type strictDecoderTransformer struct {
	base   transform.Transformer
	width  int
	order  byteOrder
	useBOM bool

	// bomResolved records whether the leading BOM has been inspected (UseBOM
	// encodings only). order is fixed once resolved. carry holds source bytes
	// that span Transform calls (a unit straddling a chunk boundary, or a BOM
	// not yet fully seen).
	bomResolved bool
	resolved    byteOrder
	carry       []byte
}

func (t *strictDecoderTransformer) Reset() {
	t.base.Reset()
	t.bomResolved = false
	t.resolved = byteOrder{}
	t.carry = t.carry[:0]
}

func (t *strictDecoderTransformer) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	nDst, nSrc, err = t.base.Transform(dst, src, atEOF)
	if err != nil && err != transform.ErrShortSrc && err != transform.ErrShortDst {
		return nDst, nSrc, err
	}

	// Validate exactly the bytes the base decoder consumed this call. Carry over
	// any partial trailing unit so it is validated once the rest arrives.
	if verr := t.validate(src[:nSrc], atEOF); verr != nil {
		return nDst, nSrc, verr
	}
	return nDst, nSrc, err
}

// validate walks the freshly-consumed source bytes (combined with any carry
// from a previous call) one code unit at a time in the resolved byte order and
// returns ErrInvalidEncodedChar on the first malformed unit. Bytes that do not
// yet form a complete unit are buffered in carry; at EOF an incomplete trailing
// unit is malformed.
func (t *strictDecoderTransformer) validate(src []byte, atEOF bool) error {
	buf := src
	if len(t.carry) > 0 {
		buf = append(append([]byte{}, t.carry...), src...)
	}
	t.carry = t.carry[:0]

	order := t.order
	off := 0

	if t.useBOM {
		// Resolve big- vs little-endian from the leading BOM, then skip it.
		if !t.bomResolved {
			if len(buf) < t.width {
				if atEOF {
					// Fewer than a full unit before EOF: malformed trailing bytes
					// (an empty stream is fine — no bytes to validate).
					if len(buf) > 0 {
						return ErrInvalidEncodedChar
					}
					return nil
				}
				t.carry = append(t.carry, buf...)
				return nil
			}
			t.resolved = t.resolveBOM(buf[:t.width])
			t.bomResolved = true
			off = t.width // skip the BOM unit
		}
		order = t.resolved
	}

	for off+t.width <= len(buf) {
		next, status := t.unitOK(buf, off, order)
		switch status {
		case unitBad:
			return ErrInvalidEncodedChar
		case unitIncomplete:
			// A high surrogate whose low half has not arrived yet. Before EOF,
			// buffer the surrogate (and any trailing partial unit) for the next
			// call; at EOF it is an unpaired surrogate and therefore malformed.
			if atEOF {
				return ErrInvalidEncodedChar
			}
			t.carry = append(t.carry, buf[off:]...)
			return nil
		}
		off = next
	}

	// Leftover bytes that do not fill a unit.
	if off < len(buf) {
		if atEOF {
			return ErrInvalidEncodedChar
		}
		t.carry = append(t.carry, buf[off:]...)
	}
	return nil
}

// resolveBOM determines the byte order from the leading BOM unit. A little-
// endian BOM has its low-order byte first; anything else (including a big-
// endian BOM or a missing BOM) is treated as big-endian, matching x/text's
// UseBOM default.
func (t *strictDecoderTransformer) resolveBOM(bom []byte) byteOrder {
	if t.width == 2 {
		// LE BOM: FF FE. BE BOM: FE FF.
		if bom[0] == 0xFF && bom[1] == 0xFE {
			return orderLE2
		}
		return orderBE2
	}
	// width 4. LE BOM: FF FE 00 00. BE BOM: 00 00 FE FF.
	if bom[0] == 0xFF && bom[1] == 0xFE && bom[2] == 0x00 && bom[3] == 0x00 {
		return orderLE4
	}
	return orderBE4
}

type unitStatus int

const (
	unitGood       unitStatus = iota // valid; next offset is returned
	unitBad                          // malformed
	unitIncomplete                   // needs more bytes (high surrogate, pair not yet present)
)

// unitOK validates the code unit at off and returns the offset just past the
// last byte it consumed (a surrogate pair consumes two units) together with a
// status. unitIncomplete is returned for a high surrogate whose low half has
// not arrived yet so the caller can decide based on EOF.
func (t *strictDecoderTransformer) unitOK(buf []byte, off int, order byteOrder) (int, unitStatus) {
	v := order.scalar(buf[off : off+t.width])

	if t.width == 4 {
		if v == 0xFFFD {
			return off + 4, unitGood
		}
		if v > 0x10FFFF || (v >= 0xD800 && v <= 0xDFFF) {
			return off, unitBad
		}
		return off + 4, unitGood
	}

	// width == 2 (UTF-16 / UCS-2).
	switch {
	case v >= 0xD800 && v <= 0xDBFF:
		// High surrogate: must be followed by a low surrogate.
		if off+2*t.width > len(buf) {
			return off, unitIncomplete
		}
		lo := order.scalar(buf[off+t.width : off+2*t.width])
		if lo < 0xDC00 || lo > 0xDFFF {
			return off, unitBad
		}
		return off + 2*t.width, unitGood
	case v >= 0xDC00 && v <= 0xDFFF:
		// Lone low surrogate.
		return off, unitBad
	default:
		// BMP scalar (includes a genuine 0xFFFD).
		return off + t.width, unitGood
	}
}
