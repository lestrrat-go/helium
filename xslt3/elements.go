package xslt3

import "github.com/lestrrat-go/helium/xslt3/internal/elements"

// elems is the package-level XSLT element registry, providing metadata
// about all recognized XSLT elements (version, context, allowed attrs, etc.).
var elems = elements.NewRegistry()
