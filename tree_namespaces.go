package helium

import "github.com/lestrrat-go/helium/internal/lexicon"

// LookupNSByPrefix walks the element and its ancestors to find a namespace
// declaration matching the given prefix. The "xml" prefix is always
// implicitly bound to the XML namespace.
func LookupNSByPrefix(e *Element, prefix string) *Namespace {
	var node Node = e
	for node != nil {
		if el, ok := node.(*Element); ok {
			for _, ns := range el.Namespaces() {
				if ns.Prefix() == prefix {
					return ns
				}
			}
		}
		node = node.Parent()
	}
	if prefix == "xml" {
		return NewNamespace("xml", lexicon.NamespaceXML)
	}
	return nil
}

// lookupNSByPrefix is the unexported alias for internal callers.
func lookupNSByPrefix(e *Element, prefix string) *Namespace {
	return LookupNSByPrefix(e, prefix)
}

// LookupNSByHref walks the element and its ancestors to find a namespace
// declaration matching the given URI.
func LookupNSByHref(e *Element, href string) *Namespace {
	if href == lexicon.NamespaceXML {
		return NewNamespace("xml", lexicon.NamespaceXML)
	}
	var node Node = e
	for node != nil {
		if el, ok := node.(*Element); ok {
			for _, ns := range el.Namespaces() {
				if ns.URI() == href {
					return ns
				}
			}
		}
		node = node.Parent()
	}
	return nil
}
