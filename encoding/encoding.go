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
	case "us-ascii", "ascii":
		return unicode.UTF8
	case "utf-16le", "utf16le":
		return unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)
	case "utf-16be", "utf16be":
		return unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM)
	case "utf-16", "utf16":
		return unicode.UTF16(unicode.LittleEndian, unicode.UseBOM)
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
		return withC1Fallback(charmap.ISO8859_10)
	case "iso-8859-13":
		return withC1Fallback(charmap.ISO8859_13)
	case "iso-8859-14":
		return withC1Fallback(charmap.ISO8859_14)
	case "iso-8859-15":
		return withC1Fallback(charmap.ISO8859_15)
	case "iso-8859-16":
		return withC1Fallback(charmap.ISO8859_16)
	case "iso-8859-2":
		return withC1Fallback(charmap.ISO8859_2)
	case "iso-8859-3":
		return withC1Fallback(charmap.ISO8859_3)
	case "iso-8859-4":
		return withC1Fallback(charmap.ISO8859_4)
	case "iso-8859-5":
		return withC1Fallback(charmap.ISO8859_5)
	case "iso-8859-6":
		return withC1Fallback(charmap.ISO8859_6)
	case "iso-8859-7":
		return withC1Fallback(charmap.ISO8859_7)
	case "iso-8859-8":
		return withC1Fallback(charmap.ISO8859_8)
	case "koi8r":
		return charmap.KOI8R
	case "koir8u":
		return charmap.KOI8U
	case "macintosh":
		return charmap.Macintosh
	case "macintoshcyrillic":
		return charmap.MacintoshCyrillic
	case "windows1250":
		return withC1Fallback(charmap.Windows1250)
	case "windows1251":
		return withC1Fallback(charmap.Windows1251)
	case "iso-8859-1", "windows1252":
		return withC1Fallback(charmap.Windows1252)
	case "windows1253":
		return withC1Fallback(charmap.Windows1253)
	case "windows1254":
		return withC1Fallback(charmap.Windows1254)
	case "windows1255":
		return withC1Fallback(charmap.Windows1255)
	case "windows1256":
		return withC1Fallback(charmap.Windows1256)
	case "windows1257":
		return withC1Fallback(charmap.Windows1257)
	case "windows1258":
		return withC1Fallback(charmap.Windows1258)
	case "windows874":
		return withC1Fallback(charmap.Windows874)
	case "xuserdefined":
		return charmap.XUserDefined
	}
	return nil
}
