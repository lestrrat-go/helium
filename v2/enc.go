package helium

const (
	encNone     = ""
	encUCS4BE   = "ucs4be"
	encUCS4LE   = "ucs4le"
	encUCS42143 = "ucs4_2143"
	encUCS43412 = "ucs4_3412"
	encEBCDIC   = "ebcdic"
	encUTF8     = "utf8"
	encUTF16LE  = "utf16le"
	encUTF16BE  = "utf16be"
)

// BOM patterns
var (
	patUCS4BE       = []byte{0x00, 0x00, 0x00, 0x3C}
	patUCS4LE       = []byte{0x3C, 0x00, 0x00, 0x00}
	patUCS42143     = []byte{0x00, 0x00, 0x3C, 0x00}
	patUCS43412     = []byte{0x00, 0x3C, 0x00, 0x00}
	patEBCDIC       = []byte{0x4C, 0x6F, 0xA7, 0x94}
	patUTF16LE4B    = []byte{0x3C, 0x00, 0x3F, 0x00}
	patUTF16BE4B    = []byte{0x00, 0x3C, 0x00, 0x3F}
	patUTF8         = []byte{0xEF, 0xBB, 0xBF}
	patUTF16LE2B    = []byte{0xFF, 0xFE}
	patUTF16BE2B    = []byte{0xFE, 0xFF}
	patMaybeXMLDecl = []byte{0x3C, 0x3F, 0x78, 0x6D}
)

