package helium

import (
	"io"

	"github.com/lestrrat-go/helium/internal/strcursor"
	"golang.org/x/text/transform"
)

// normalizeXML11Input installs input normalization after the declaration and
// encoding are known, before parsing markup. XML 1.1 section 2.11 forbids NEL
// and LS inside the declaration itself. Internal entity replacement text is
// already processed and must not pass through this reader again: a character
// reference in an entity value can deliberately contribute NEL or LS.
func (pctx *parserCtx) normalizeXML11Input() error {
	if !pctx.isXML11() {
		return nil
	}
	cur := pctx.getCursor()
	if cur == nil {
		return errNoCursor
	}
	next := strcursor.NewUTF8Cursor(xml11LineEndingReader(cur.Unused()), pctx.inputBufSize())
	next.StartAt(strcursor.PositionOf(cur))
	pctx.popInput()
	pctx.pushInput(next)
	return nil
}

func xml11LineEndingReader(r io.Reader) io.Reader {
	return transform.NewReader(r, xml11LineEndings{})
}

// xml11LineEndings processes UTF-8 input without decoding or replacing other
// bytes. Incomplete line endings wait for the next read; malformed UTF-8 stays
// intact so the parser can report it through its normal character checks.
type xml11LineEndings struct{}

func (xml11LineEndings) Reset() {}

func (xml11LineEndings) Transform(dst, src []byte, atEOF bool) (int, int, error) {
	nDst, nSrc := 0, 0
	for nSrc < len(src) {
		if nDst == len(dst) {
			return nDst, nSrc, transform.ErrShortDst
		}
		remaining := src[nSrc:]
		b, width := remaining[0], 1
		switch b {
		case '\r':
			if len(remaining) == 1 && !atEOF {
				return nDst, nSrc, transform.ErrShortSrc
			}
			b = '\n'
			if len(remaining) > 1 {
				switch remaining[1] {
				case '\n':
					width = 2
				case 0xc2:
					if len(remaining) == 2 && !atEOF {
						return nDst, nSrc, transform.ErrShortSrc
					}
					if len(remaining) > 2 && remaining[2] == 0x85 {
						width = 3
					}
				}
			}
		case 0xc2:
			if len(remaining) == 1 && !atEOF {
				return nDst, nSrc, transform.ErrShortSrc
			}
			if len(remaining) > 1 && remaining[1] == 0x85 {
				b, width = '\n', 2
			}
		case 0xe2:
			if len(remaining) < 3 && !atEOF && (len(remaining) == 1 || remaining[1] == 0x80) {
				return nDst, nSrc, transform.ErrShortSrc
			}
			if len(remaining) > 2 && remaining[1] == 0x80 && remaining[2] == 0xa8 {
				b, width = '\n', 3
			}
		}
		dst[nDst] = b
		nDst++
		nSrc += width
	}
	return nDst, nSrc, nil
}
