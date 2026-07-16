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
	"golang.org/x/text/encoding/unicode/utf32"
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

// asciiEncodingNames is the set of normalized names (see normalizeEncodingName)
// that denote US-ASCII, covering every IANA-registered alias plus the common
// "ascii" synonym. It is the single source of truth for both Load (which returns
// the strict US-ASCII codec) and IsASCII.
var asciiEncodingNames = map[string]struct{}{
	"usascii":       {},
	"ascii":         {},
	"ansix341968":   {},
	"ansix341986":   {},
	"csascii":       {},
	"isoir6":        {},
	"iso646us":      {},
	"iso646irv1991": {},
	"us":            {},
	"ibm367":        {},
	"cp367":         {},
}

// asciiRawUTF8Names is the subset of US-ASCII aliases the no-override document
// serializer emits as raw UTF-8 (via a UTF-8 passthrough encoder) rather than as
// numeric character references. It is deliberately narrower than
// asciiEncodingNames (IsASCII): every other US-ASCII alias is
// character-referenced on the no-override path.
var asciiRawUTF8Names = map[string]struct{}{
	"ansix341968": {},
	"csascii":     {},
}

// IsASCIIRawUTF8Alias reports whether name is one of the two US-ASCII aliases the
// no-override document serializer emits as raw UTF-8 (see asciiRawUTF8Names).
func IsASCIIRawUTF8Alias(name string) bool {
	_, ok := asciiRawUTF8Names[normalizeEncodingName(name)]
	return ok
}

func Load(name string) enc.Encoding {
	norm := normalizeEncodingName(name)
	if _, ok := asciiEncodingNames[norm]; ok {
		return asciiEncoding{}
	}
	switch norm {
	case "utf8", "unicode11utf8", "unicode20utf8", "xunicode20utf8":
		return unicode.UTF8
	case "utf16le", "unicodefeff":
		return withStrictDecode(unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM), 2, orderLE2, false)
	case "utf16be", "unicodefffe":
		return withStrictDecode(unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM), 2, orderBE2, false)
	case "utf16", "unicode", "csunicode":
		// UseBOM with LittleEndian default: x/text's decoder falls back to the
		// configured default (LE) when no BOM is present, so the validator's
		// no-BOM order must be LE to match.
		return withStrictDecode(unicode.UTF16(unicode.LittleEndian, unicode.UseBOM), 2, orderLE2, true)
	case "eucjp", "xeucjp", "cseucpkdfmtjapanese":
		return japanese.EUCJP
	case "shiftjis", "cp932", "sjis", "ms932", "mskanji", "windows31j", "xsjis", "csshiftjis":
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
	case "iso88592", "iso885921987", "isoir101", "latin2", "l2", "csisolatin2", "isolatin2":
		return withC1Fallback(charmap.ISO8859_2)
	case "iso88593", "iso885931988", "isoir109", "latin3", "l3", "csisolatin3":
		return withC1Fallback(charmap.ISO8859_3)
	case "iso88594", "iso885941988", "isoir110", "latin4", "l4", "csisolatin4":
		return withC1Fallback(charmap.ISO8859_4)
	case "iso88595", "iso885951988", "isoir144", "cyrillic", "csisolatincyrillic":
		return withC1Fallback(charmap.ISO8859_5)
	case "iso88596", "iso885961987", "isoir127", "arabic", "ecma114", "asmo708", "csisolatinarabic", "csiso88596e", "csiso88596i":
		return withC1Fallback(charmap.ISO8859_6)
	case "iso88597", "iso885971987", "isoir126", "greek", "greek8", "elot928", "ecma118", "csisolatingreek", "suneugreek":
		return withC1Fallback(charmap.ISO8859_7)
	case "iso88598", "iso885981988", "isoir138", "hebrew", "csisolatinhebrew", "csiso88598e", "csiso88598i", "logical", "visual":
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
	case "iso88591", "iso885911987", "isoir100", "latin1", "l1", "ibm819", "cp819", "csisolatin1", "isolatin1":
		// True ISO-8859-1: bytes 0x80-0x9F map to the C1 controls
		// U+0080-U+009F, consistent with iso-8859-2..16 above. libxml2's XML
		// path uses the IANA/ISO codec here, not the Windows-1252 alias the
		// WHATWG HTML spec mandates (that leniency stays in the html/ package).
		return withC1Fallback(charmap.ISO8859_1)
	case "windows1252", "cp1252", "xcp1252":
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
	case "ucs4be", "utf32be", "iso10646ucs4":
		return withStrictDecode(utf32.UTF32(utf32.BigEndian, utf32.IgnoreBOM), 4, orderBE4, false)
	case "ucs4le", "utf32le":
		return withStrictDecode(utf32.UTF32(utf32.LittleEndian, utf32.IgnoreBOM), 4, orderLE4, false)
	case "ucs4", "utf32":
		// UseBOM with BigEndian default: x/text's decoder falls back to the
		// configured default (BE) when no BOM is present, so the validator's
		// no-BOM order must be BE to match.
		return withStrictDecode(utf32.UTF32(utf32.BigEndian, utf32.UseBOM), 4, orderBE4, true)
	case "ucs42143":
		return withStrictDecode(&ucs4SwapEncoding{swap: swap2143}, 4, order2143, false)
	case "ucs43412":
		return withStrictDecode(&ucs4SwapEncoding{swap: swap3412}, 4, order3412, false)
	case "ucs2", "iso10646ucs2":
		return withStrictDecode(unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM), 2, orderBE2, false)
	}
	return nil
}
