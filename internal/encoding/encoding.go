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

var encodingNameNormalizer = strings.NewReplacer(
	"-", "",
	"_", "",
	":", "",
	".", "",
	" ", "",
)

func normalizeEncodingName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ToLower(name)
	return encodingNameNormalizer.Replace(name)
}

func Load(name string) enc.Encoding {
	switch normalizeEncodingName(name) {
	case "utf8", "unicode11utf8", "unicode20utf8":
		return unicode.UTF8
	case "usascii", "ascii", "ansix341968", "csascii":
		return unicode.UTF8
	case "utf16le":
		return unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)
	case "utf16be":
		return unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM)
	case "utf16", "unicode", "csunicode":
		return unicode.UTF16(unicode.LittleEndian, unicode.UseBOM)
	case "eucjp", "xeucjp":
		return japanese.EUCJP
	case "shiftjis", "cp932", "sjis", "ms932", "mskanji", "windows31j", "xsjis":
		return japanese.ShiftJIS
	case "jis", "iso2022jp", "csiso2022jp":
		return japanese.ISO2022JP
	case "big5", "csbig5":
		return traditionalchinese.Big5
	case "euckr", "cseuckr", "ksc56011987", "csksc56011987":
		return korean.EUCKR
	case "hzgb2312":
		return simplifiedchinese.HZGB2312
	case "cp437", "ibm437", "cspc8codepage437":
		return charmap.CodePage437
	case "cp866", "ibm866", "csibm866":
		return charmap.CodePage866
	case "iso885910", "latin6", "l6", "isoir157", "csisolatin6":
		return withC1Fallback(charmap.ISO8859_10)
	case "iso885913":
		return withC1Fallback(charmap.ISO8859_13)
	case "iso885914", "latin8", "l8", "isoir199", "csisolatin8":
		return withC1Fallback(charmap.ISO8859_14)
	case "iso885915", "latin9", "l9", "csisolatin9":
		return withC1Fallback(charmap.ISO8859_15)
	case "iso885916", "latin10", "l10":
		return withC1Fallback(charmap.ISO8859_16)
	case "iso88592", "iso885921987", "isoir101", "latin2", "l2", "csisolatin2":
		return withC1Fallback(charmap.ISO8859_2)
	case "iso88593", "iso885931988", "isoir109", "latin3", "l3", "csisolatin3":
		return withC1Fallback(charmap.ISO8859_3)
	case "iso88594", "iso885941988", "isoir110", "latin4", "l4", "csisolatin4":
		return withC1Fallback(charmap.ISO8859_4)
	case "iso88595", "iso885951988", "isoir144", "cyrillic", "csisolatincyrillic":
		return withC1Fallback(charmap.ISO8859_5)
	case "iso88596", "iso885961987", "isoir127", "arabic", "ecma114", "asmo708", "csisolatinarabic":
		return withC1Fallback(charmap.ISO8859_6)
	case "iso88597", "iso885971987", "isoir126", "greek", "greek8", "elot928", "ecma118", "csisolatingreek":
		return withC1Fallback(charmap.ISO8859_7)
	case "iso88598", "iso885981988", "isoir138", "hebrew", "csisolatinhebrew":
		return withC1Fallback(charmap.ISO8859_8)
	case "iso88599", "iso885991989", "isoir148", "latin5", "l5", "csisolatin5":
		return withC1Fallback(charmap.ISO8859_9)
	case "iso885911", "tis620":
		// x/text/charmap does not expose ISO8859_11 directly; Windows-874 is
		// the closest Thai code page used in practice.
		return withC1Fallback(charmap.Windows874)
	case "koi8r", "cskoi8r":
		return charmap.KOI8R
	case "koi8u", "koir8u":
		return charmap.KOI8U
	case "macintosh", "mac", "csmacintosh":
		return charmap.Macintosh
	case "macintoshcyrillic", "xmaccyrillic", "maccyrillic":
		return charmap.MacintoshCyrillic
	case "windows1250", "cp1250":
		return withC1Fallback(charmap.Windows1250)
	case "windows1251", "cp1251":
		return withC1Fallback(charmap.Windows1251)
	case "iso88591", "iso885911987", "isoir100", "latin1", "l1", "ibm819", "cp819", "csisolatin1", "windows1252", "cp1252":
		return withC1Fallback(charmap.Windows1252)
	case "windows1253", "cp1253":
		return withC1Fallback(charmap.Windows1253)
	case "windows1254", "cp1254":
		return withC1Fallback(charmap.Windows1254)
	case "windows1255", "cp1255":
		return withC1Fallback(charmap.Windows1255)
	case "windows1256", "cp1256":
		return withC1Fallback(charmap.Windows1256)
	case "windows1257", "cp1257":
		return withC1Fallback(charmap.Windows1257)
	case "windows1258", "cp1258":
		return withC1Fallback(charmap.Windows1258)
	case "windows874", "cp874":
		return withC1Fallback(charmap.Windows874)
	case "xuserdefined":
		return charmap.XUserDefined
	case "ebcdic", "ibm037", "cp037", "ebcdicus", "ebcdiccpus", "csibm037":
		return charmap.CodePage037
	case "ibm1047", "cp1047":
		return charmap.CodePage1047
	case "ibm1140", "ibm01140", "cp1140", "ccsid01140":
		return charmap.CodePage1140
	case "ibm1141", "ibm01141", "cp1141", "ccsid01141":
		return codePage1141
	}
	return nil
}
