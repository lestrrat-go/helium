package xslt3

import (
	"strings"

	"github.com/lestrrat-go/helium/internal/lexicon"
)

// qnameKind identifies the syntactic position of a QName inside a sequence-type
// item test. The expansion rules for an unprefixed name differ by position:
//
//   - qnameElementName  — the name argument of element() / schema-element() and
//     of document-node(element(...)). An unprefixed name takes the default
//     element namespace (xpath-default-namespace).
//   - qnameAttributeName — the name argument of attribute() / schema-attribute().
//     An unprefixed name is in NO namespace; the default element namespace NEVER
//     applies to an attribute name.
//   - qnameTypeName — the type argument (2nd arg) of element(name, type) and
//     attribute(name, type). An unprefixed built-in type resolves to the XSD
//     namespace, any other unprefixed name is in no namespace, and xs:/xsd:
//     prefixes are aliases for the XSD namespace. The default element namespace
//     never applies.
//
// In every position the xml prefix is predeclared to the XML namespace; an
// explicit declaration-site binding for any prefix overrides predeclaration.
type qnameKind int

const (
	qnameElementName qnameKind = iota
	qnameAttributeName
	qnameTypeName
)

// nsResolver resolves a namespace prefix to its bound URI for the declaration
// site. It reports whether the prefix was bound. The xml prefix is handled by
// resolveSequenceTypeQName itself (predeclared), so a resolver need not special
// case it, though an explicit binding it returns takes precedence.
type nsResolver func(prefix string) (string, bool)

// nsResolverFromMap returns an nsResolver backed by a prefix→URI map.
func nsResolverFromMap(m map[string]string) nsResolver {
	return func(prefix string) (string, bool) {
		uri, ok := m[prefix]
		return uri, ok
	}
}

// resolveSequenceTypeQName resolves a QName appearing in a sequence-type item
// test to its (localName, namespaceURI) pair, applying the position-specific
// rules described on qnameKind. This is the single canonical resolution path
// used by both @as validation and the XTSE3087 canonical-expansion so that an
// element name, attribute name, or type name is interpreted identically
// everywhere.
//
// qname may be a prefixed QName (prefix:local), an unprefixed NCName, or an
// EQName in Q{uri}local form. A wildcard ("*") or empty string is returned
// unchanged with an empty namespace.
func resolveSequenceTypeQName(qname string, kind qnameKind, resolve nsResolver, defaultElemNS string, hasDefaultElemNS bool) (local, ns string) {
	qname = strings.TrimSpace(qname)
	if qname == "" || qname == "*" {
		return qname, ""
	}

	// EQName form: Q{uri}local — already fully expanded.
	if strings.HasPrefix(qname, "Q{") {
		if closeIdx := strings.IndexByte(qname, '}'); closeIdx > 0 {
			return qname[closeIdx+1:], qname[2:closeIdx]
		}
		return "", ""
	}

	prefix, loc, prefixed := strings.Cut(qname, ":")
	if prefixed {
		// xs:/xsd: are conventional aliases for the XSD namespace when naming a
		// type; for element/attribute names they are ordinary prefixes resolved
		// through the binding map (so xsd: works only if declared).
		if kind == qnameTypeName && (prefix == "xs" || prefix == "xsd") {
			return loc, lexicon.NamespaceXSD
		}
		// The xml prefix is predeclared everywhere; an explicit binding overrides.
		if uri, ok := resolvePrefixWithXML(prefix, resolve); ok {
			return loc, uri
		}
		return loc, ""
	}

	// Unprefixed name.
	switch kind {
	case qnameElementName:
		if hasDefaultElemNS {
			return qname, defaultElemNS
		}
		return qname, ""
	case qnameTypeName:
		// An unprefixed built-in type is in the XSD namespace; everything else
		// is in no namespace. The default element namespace is never applied.
		if isBuiltinXSDLocalName(qname) {
			return qname, lexicon.NamespaceXSD
		}
		return qname, ""
	default: // qnameAttributeName
		// An unprefixed attribute name is in no namespace.
		return qname, ""
	}
}

// resolvePrefixWithXML resolves a prefix through the given resolver, treating
// the xml prefix as predeclared to the XML namespace when the resolver has no
// explicit binding for it.
func resolvePrefixWithXML(prefix string, resolve nsResolver) (string, bool) {
	if resolve != nil {
		if uri, ok := resolve(prefix); ok {
			return uri, true
		}
	}
	if prefix == lexicon.PrefixXML {
		return lexicon.NamespaceXML, true
	}
	return "", false
}
