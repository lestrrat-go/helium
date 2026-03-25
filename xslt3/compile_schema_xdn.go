package xslt3

import (
	"strings"

	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/xpath3"
)

// resolveXSDTypeNameWithXDN is a convenience wrapper that resolves a type
// QName using the compiler's namespace bindings merged with xpath-default-namespace.
// The "" key convention in nsBindings carries the xpath-default-namespace so
// that resolveXSDTypeName can use it for bare (unprefixed) type names.
func (c *compiler) resolveXSDTypeNameWithXDN(qname string) string {
	if c.xpathDefaultNS == "" {
		return resolveXSDTypeName(qname, c.nsBindings)
	}
	merged := make(map[string]string, len(c.nsBindings)+1)
	for k, v := range c.nsBindings {
		merged[k] = v
	}
	merged[""] = c.xpathDefaultNS
	return resolveXSDTypeNameXDN(qname, merged)
}

// resolveXSDTypeNameXDN is like resolveXSDTypeName but uses the "" key in
// nsBindings as the xpath-default-namespace for bare (unprefixed) type names.
func resolveXSDTypeNameXDN(qname string, nsBindings map[string]string) string {
	qname = strings.TrimSpace(qname)
	if qname == "" {
		return ""
	}
	// Handle EQName Q{uri}local
	if strings.HasPrefix(qname, "Q{") {
		closeIdx := strings.IndexByte(qname, '}')
		if closeIdx > 0 {
			uri := qname[2:closeIdx]
			local := qname[closeIdx+1:]
			if uri == lexicon.NamespaceXSD {
				return "xs:" + local
			}
			return qname
		}
	}
	// Handle prefix:local
	if idx := strings.IndexByte(qname, ':'); idx >= 0 {
		prefix := qname[:idx]
		local := qname[idx+1:]
		if prefix == "xs" || prefix == "xsd" {
			return "xs:" + local
		}
		if uri, ok := nsBindings[prefix]; ok {
			if uri == lexicon.NamespaceXSD {
				return "xs:" + local
			}
			return xpath3.QAnnotation(uri, local)
		}
	}
	// Use xpath-default-namespace (passed as "" key) for bare names.
	if xdn := nsBindings[""]; xdn != "" {
		if xdn == lexicon.NamespaceXSD {
			return "xs:" + qname
		}
		return xpath3.QAnnotation(xdn, qname)
	}
	return "Q{}" + qname
}

// validateAsSequenceTypeWithXDN wraps validateAsSequenceType with
// xpath-default-namespace support for schema-element() and element(*, type)
// references in as= attributes. It is called by the compiler instead of
// validateAsSequenceType directly.
func (c *compiler) validateAsSequenceTypeWithXDN(as string, context string) error {
	if c.xpathDefaultNS == "" {
		return c.validateAsSequenceType(as, context)
	}
	// Temporarily inject xpathDefaultNS into nsBindings so that
	// resolveQNameToLocalNS (called inside validateAsSequenceType)
	// picks it up via the "" key convention.
	saved := c.nsBindings
	merged := make(map[string]string, len(c.nsBindings)+1)
	for k, v := range c.nsBindings {
		merged[k] = v
	}
	merged[""] = c.xpathDefaultNS
	c.nsBindings = merged
	err := c.validateAsSequenceType(as, context)
	c.nsBindings = saved
	return err
}
