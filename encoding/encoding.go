// Package encoding wraps around the various encoding stuff in
// golang.org/x/text/encoding. Part of the reason this exists is that
// the package names such as "unicode" clash with the stdlib, and
// it's rather easier if we just hide it from helium
package encoding

import (
	"strings"

	enc "golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/encoding/unicode"
)

func Load(name string) enc.Encoding {
	switch strings.ToLower(name) {
	case "utf8", "utf-8":
		return unicode.UTF8
	case "euc-jp":
		return japanese.EUCJP
	case "shift_jis", "shift-jis", "shiftjis", "cp932":
		return japanese.ShiftJIS
	case "jis", "iso-2022-jp":
		return japanese.ISO2022JP
	case "big5":
		return traditionalchinese.Big5
	case "euc-kr":
		return korean.EUCKR
	case "hz-gb2312":
		return simplifiedchinese.HZGB2312
	case "cp437":
		return charmap.CodePage437
	case "cp866":
		return charmap.CodePage866
	case "iso-8859-10":
		return charmap.ISO8859_10
	case "iso-8859-13":
		return charmap.ISO8859_13
	case "iso-8859-14":
		return charmap.ISO8859_14
	case "iso-8859-15":
		return charmap.ISO8859_15
	case "iso-8859-16":
		return charmap.ISO8859_16
	case "iso-8859-2":
		return charmap.ISO8859_2
	case "iso-8859-3":
		return charmap.ISO8859_3
	case "iso-8859-4":
		return charmap.ISO8859_4
	case "iso-8859-5":
		return charmap.ISO8859_5
	case "iso-8859-6":
		return charmap.ISO8859_6
	case "iso-8859-7":
		return charmap.ISO8859_7
	case "iso-8859-8":
		return charmap.ISO8859_8
	case "koi8r":
		return charmap.KOI8R
	case "koir8u":
		return charmap.KOI8U
	case "macintosh":
		return charmap.Macintosh
	case "macintoshcyrillic":
		return charmap.MacintoshCyrillic
	case "windows1250":
		return charmap.Windows1250
	case "windows1251":
		return charmap.Windows1251
	case "iso-8859-1", "windows1252":
		return charmap.Windows1252
	case "windows1253":
		return charmap.Windows1253
	case "windows1254":
		return charmap.Windows1254
	case "windows1255":
		return charmap.Windows1255
	case "windows1256":
		return charmap.Windows1256
	case "windows1257":
		return charmap.Windows1257
	case "windows1258":
		return charmap.Windows1258
	case "windows874":
		return charmap.Windows874
	case "xuserdefined":
		return charmap.XUserDefined
	}
	return nil
}
