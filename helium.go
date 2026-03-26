package helium

import "github.com/lestrrat-go/helium/internal/lexicon"

const Version = `0.0.1`

const (
	XMLNamespace = lexicon.NamespaceXML
	XMLNsPrefix  = lexicon.PrefixXMLNS
	XMLPrefix    = lexicon.PrefixXML
	xmlTextNoEnc = "textnoenc"
)

// ClarkName returns the Clark notation "{uri}local" for a namespace URI and
// local name pair.
func ClarkName(uri, local string) string {
	return "{" + uri + "}" + local
}
