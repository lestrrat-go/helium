package parseopts

import "github.com/lestrrat-go/helium/internal/bitset"

// Option holds internal parser flags that are not part of the public API.
// These flags are shared across the module but not exported to consumers.
type Option int

const (
	// LenientXMLDecl allows the XML declaration pseudo-attributes
	// (version, encoding, standalone) to appear in any order.
	// The XML spec requires version first, then encoding, then standalone,
	// but some real-world documents violate this.
	LenientXMLDecl Option = 1 << iota
)

func (o *Option) Set(n Option) {
	bitset.Set(o, n)
}

func (o Option) IsSet(n Option) bool {
	return bitset.IsSet(o, n)
}
