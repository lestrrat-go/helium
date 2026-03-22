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
