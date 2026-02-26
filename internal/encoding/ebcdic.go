package encoding

import (
	"unicode/utf8"

	enc "golang.org/x/text/encoding"
	"golang.org/x/text/transform"
)

// cp1141ToUnicode maps IBM Code Page 1141 (EBCDIC Austria/Germany + euro)
// byte values to Unicode code points. CP1141 is CP273 with the currency
// sign (U+00A4) at position 0x9F replaced by the euro sign (U+20AC).
//
// Source: ibm-1141_P100-1997.ucm from unicode-org/icu-data.
var cp1141ToUnicode = [256]rune{
	// 0x00-0x0F
	0x0000, 0x0001, 0x0002, 0x0003, 0x009C, 0x0009, 0x0086, 0x007F, 0x0097, 0x008D, 0x008E, 0x000B, 0x000C, 0x000D, 0x000E, 0x000F,
	// 0x10-0x1F
	0x0010, 0x0011, 0x0012, 0x0013, 0x009D, 0x0085, 0x0008, 0x0087, 0x0018, 0x0019, 0x0092, 0x008F, 0x001C, 0x001D, 0x001E, 0x001F,
	// 0x20-0x2F
	0x0080, 0x0081, 0x0082, 0x0083, 0x0084, 0x000A, 0x0017, 0x001B, 0x0088, 0x0089, 0x008A, 0x008B, 0x008C, 0x0005, 0x0006, 0x0007,
	// 0x30-0x3F
	0x0090, 0x0091, 0x0016, 0x0093, 0x0094, 0x0095, 0x0096, 0x0004, 0x0098, 0x0099, 0x009A, 0x009B, 0x0014, 0x0015, 0x009E, 0x001A,
	// 0x40-0x4F
	0x0020, 0x00A0, 0x00E2, 0x007B, 0x00E0, 0x00E1, 0x00E3, 0x00E5, 0x00E7, 0x00F1, 0x00C4, 0x002E, 0x003C, 0x0028, 0x002B, 0x0021,
	// 0x50-0x5F
	0x0026, 0x00E9, 0x00EA, 0x00EB, 0x00E8, 0x00ED, 0x00EE, 0x00EF, 0x00EC, 0x007E, 0x00DC, 0x0024, 0x002A, 0x0029, 0x003B, 0x005E,
	// 0x60-0x6F
	0x002D, 0x002F, 0x00C2, 0x005B, 0x00C0, 0x00C1, 0x00C3, 0x00C5, 0x00C7, 0x00D1, 0x00F6, 0x002C, 0x0025, 0x005F, 0x003E, 0x003F,
	// 0x70-0x7F
	0x00F8, 0x00C9, 0x00CA, 0x00CB, 0x00C8, 0x00CD, 0x00CE, 0x00CF, 0x00CC, 0x0060, 0x003A, 0x0023, 0x00A7, 0x0027, 0x003D, 0x0022,
	// 0x80-0x8F
	0x00D8, 0x0061, 0x0062, 0x0063, 0x0064, 0x0065, 0x0066, 0x0067, 0x0068, 0x0069, 0x00AB, 0x00BB, 0x00F0, 0x00FD, 0x00FE, 0x00B1,
	// 0x90-0x9F
	0x00B0, 0x006A, 0x006B, 0x006C, 0x006D, 0x006E, 0x006F, 0x0070, 0x0071, 0x0072, 0x00AA, 0x00BA, 0x00E6, 0x00B8, 0x00C6, 0x20AC,
	// 0xA0-0xAF
	0x00B5, 0x00DF, 0x0073, 0x0074, 0x0075, 0x0076, 0x0077, 0x0078, 0x0079, 0x007A, 0x00A1, 0x00BF, 0x00D0, 0x00DD, 0x00DE, 0x00AE,
	// 0xB0-0xBF
	0x00A2, 0x00A3, 0x00A5, 0x00B7, 0x00A9, 0x0040, 0x00B6, 0x00BC, 0x00BD, 0x00BE, 0x00AC, 0x007C, 0x00AF, 0x00A8, 0x00B4, 0x00D7,
	// 0xC0-0xCF
	0x00E4, 0x0041, 0x0042, 0x0043, 0x0044, 0x0045, 0x0046, 0x0047, 0x0048, 0x0049, 0x00AD, 0x00F4, 0x00A6, 0x00F2, 0x00F3, 0x00F5,
	// 0xD0-0xDF
	0x00FC, 0x004A, 0x004B, 0x004C, 0x004D, 0x004E, 0x004F, 0x0050, 0x0051, 0x0052, 0x00B9, 0x00FB, 0x007D, 0x00F9, 0x00FA, 0x00FF,
	// 0xE0-0xEF
	0x00D6, 0x00F7, 0x0053, 0x0054, 0x0055, 0x0056, 0x0057, 0x0058, 0x0059, 0x005A, 0x00B2, 0x00D4, 0x005C, 0x00D2, 0x00D3, 0x00D5,
	// 0xF0-0xFF
	0x0030, 0x0031, 0x0032, 0x0033, 0x0034, 0x0035, 0x0036, 0x0037, 0x0038, 0x0039, 0x00B3, 0x00DB, 0x005D, 0x00D9, 0x00DA, 0x009F,
}

var codePage1141 enc.Encoding

func init() {
	// Build reverse mapping (rune → byte) for the encoder.
	reverse := make(map[rune]byte, 256)
	for i, r := range cp1141ToUnicode {
		reverse[r] = byte(i)
	}
	codePage1141 = &ebcdicEncoding{
		forward: cp1141ToUnicode,
		reverse: reverse,
	}
}

type ebcdicEncoding struct {
	forward [256]rune
	reverse map[rune]byte
}

func (e *ebcdicEncoding) NewDecoder() *enc.Decoder {
	return &enc.Decoder{Transformer: &ebcdicDecoder{table: &e.forward}}
}

func (e *ebcdicEncoding) NewEncoder() *enc.Encoder {
	return &enc.Encoder{Transformer: &ebcdicEncoder{table: e.reverse}}
}

type ebcdicDecoder struct {
	transform.NopResetter
	table *[256]rune
}

func (d *ebcdicDecoder) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	for nSrc < len(src) {
		r := d.table[src[nSrc]]
		size := utf8.RuneLen(r)
		if nDst+size > len(dst) {
			err = transform.ErrShortDst
			return
		}
		utf8.EncodeRune(dst[nDst:], r)
		nDst += size
		nSrc++
	}
	return
}

type ebcdicEncoder struct {
	transform.NopResetter
	table map[rune]byte
}

func (e *ebcdicEncoder) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	for nSrc < len(src) {
		r, size := utf8.DecodeRune(src[nSrc:])
		if r == utf8.RuneError && size <= 1 {
			if !atEOF && !utf8.FullRune(src[nSrc:]) {
				err = transform.ErrShortSrc
				return
			}
		}
		if nDst >= len(dst) {
			err = transform.ErrShortDst
			return
		}
		b, ok := e.table[r]
		if !ok {
			b = 0x3F // EBCDIC question mark (substitution character)
		}
		dst[nDst] = b
		nDst++
		nSrc += size
	}
	return
}

// ebcdicInvariantToASCII maps EBCDIC byte positions to ASCII equivalents
// for the characters that share the same position across all EBCDIC
// single-byte Latin code pages. 0x00 means "no invariant mapping".
// This is used to extract the encoding declaration from raw EBCDIC bytes
// without knowing the specific code page.
var ebcdicInvariantToASCII = [256]byte{
	// 0x00-0x3F: control characters → 0
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	// 0x40-0x4F: SP                                          .     <     (     +
	0x20, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x2E, 0x3C, 0x28, 0x2B, 0,
	// 0x50-0x5F: &                                     *     )     ;
	0x26, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x2A, 0x29, 0x3B, 0,
	// 0x60-0x6F: -     /                               ,     %     _     >     ?
	0x2D, 0x2F, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x2C, 0x25, 0x5F, 0x3E, 0x3F,
	// 0x70-0x7F:                                       :           '     =     "
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x3A, 0, 0, 0x27, 0x3D, 0x22,
	// 0x80-0x8F:       a     b     c     d     e     f     g     h     i
	0, 0x61, 0x62, 0x63, 0x64, 0x65, 0x66, 0x67, 0x68, 0x69, 0, 0, 0, 0, 0, 0,
	// 0x90-0x9F:       j     k     l     m     n     o     p     q     r
	0, 0x6A, 0x6B, 0x6C, 0x6D, 0x6E, 0x6F, 0x70, 0x71, 0x72, 0, 0, 0, 0, 0, 0,
	// 0xA0-0xAF:             s     t     u     v     w     x     y     z
	0, 0, 0x73, 0x74, 0x75, 0x76, 0x77, 0x78, 0x79, 0x7A, 0, 0, 0, 0, 0, 0,
	// 0xB0-0xBF
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	// 0xC0-0xCF:       A     B     C     D     E     F     G     H     I
	0, 0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47, 0x48, 0x49, 0, 0, 0, 0, 0, 0,
	// 0xD0-0xDF:       J     K     L     M     N     O     P     Q     R
	0, 0x4A, 0x4B, 0x4C, 0x4D, 0x4E, 0x4F, 0x50, 0x51, 0x52, 0, 0, 0, 0, 0, 0,
	// 0xE0-0xEF:             S     T     U     V     W     X     Y     Z
	0, 0, 0x53, 0x54, 0x55, 0x56, 0x57, 0x58, 0x59, 0x5A, 0, 0, 0, 0, 0, 0,
	// 0xF0-0xFF: 0     1     2     3     4     5     6     7     8     9
	0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39, 0, 0, 0, 0, 0, 0,
}

// ExtractEBCDICEncoding scans raw EBCDIC bytes for an XML encoding
// declaration and returns the encoding name. It uses the EBCDIC invariant
// character set (shared across all single-byte EBCDIC Latin code pages)
// so it works regardless of the specific EBCDIC variant.
// Returns "" if no encoding declaration is found.
func ExtractEBCDICEncoding(raw []byte) string {
	// Translate to ASCII using invariant positions.
	limit := len(raw)
	if limit > 200 {
		limit = 200
	}
	ascii := make([]byte, limit)
	for i := 0; i < limit; i++ {
		ascii[i] = ebcdicInvariantToASCII[raw[i]]
	}

	// Find "encoding" in the ASCII-translated bytes.
	target := []byte("encoding")
	pos := -1
	for i := 0; i+len(target) <= limit; i++ {
		if ascii[i] == '>' {
			break // past the XML declaration
		}
		match := true
		for j, b := range target {
			if ascii[i+j] != b {
				match = false
				break
			}
		}
		if match {
			pos = i + len(target)
			break
		}
	}
	if pos < 0 {
		return ""
	}

	// Skip whitespace and '='
	for pos < limit && (ascii[pos] == ' ' || ascii[pos] == 0) {
		pos++
	}
	if pos >= limit || ascii[pos] != '=' {
		return ""
	}
	pos++
	for pos < limit && (ascii[pos] == ' ' || ascii[pos] == 0) {
		pos++
	}

	// Read quoted value
	if pos >= limit {
		return ""
	}
	quote := ascii[pos]
	if quote != '"' && quote != '\'' {
		return ""
	}
	pos++
	start := pos
	for pos < limit && ascii[pos] != quote && ascii[pos] != 0 {
		pos++
	}
	if pos >= limit || ascii[pos] != quote {
		return ""
	}
	return string(ascii[start:pos])
}
