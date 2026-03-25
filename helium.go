// Package helium is a Go implementation of the libxml2 XML toolkit. It provides
// tree-based XML parsing, a DOM interface, SAX2 callbacks, namespace handling,
// DTD validation, and serialization (libxml2: libxml2 core).
package helium

import "github.com/lestrrat-go/helium/internal/lexicon"

const Version = `0.0.1`

const (
	XMLNamespace = lexicon.NamespaceXML
	XMLNsPrefix  = lexicon.PrefixXMLNS
	XMLPrefix    = lexicon.PrefixXML
	XMLTextNoEnc = "textnoenc"
)

// ClarkName returns the Clark notation "{uri}local" for a namespace URI and
// local name pair.
func ClarkName(uri, local string) string {
	return "{" + uri + "}" + local
}
