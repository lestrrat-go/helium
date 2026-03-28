package catalog

import "strings"

const urnPrefix = "urn:publicid:"

// UnwrapURN converts a urn:publicid: URN to a public identifier per RFC 3151.
// Returns "" if the input is not a urn:publicid: URN.
//
// Decoding rules:
//
//   - → ' ' (space)
//     :   → //
//     ;   → ::
//     %2B → +
//     %3A → :
//     %2F → /
//     %3B → ;
//     %27 → '
//     %3F → ?
//     %23 → #
//     %25 → %
func UnwrapURN(urn string) string {
	if !strings.HasPrefix(urn, urnPrefix) {
		return ""
	}
	rest := urn[len(urnPrefix):]

	var b strings.Builder
	b.Grow(len(rest))

	for i := 0; i < len(rest); i++ {
		c := rest[i]
		switch c {
		case '+':
			b.WriteByte(' ')
		case ':':
			b.WriteString("//")
		case ';':
			b.WriteString("::")
		case '%':
			if i+2 < len(rest) {
				esc := rest[i+1 : i+3]
				switch strings.ToUpper(esc) {
				case "2B":
					b.WriteByte('+')
				case "3A":
					b.WriteByte(':')
				case "2F":
					b.WriteByte('/')
				case "3B":
					b.WriteByte(';')
				case "27":
					b.WriteByte('\'')
				case "3F":
					b.WriteByte('?')
				case "23":
					b.WriteByte('#')
				case "25":
					b.WriteByte('%')
				default:
					b.WriteByte('%')
					continue
				}
				i += 2
			} else {
				b.WriteByte('%')
			}
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}
