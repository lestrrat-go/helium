package helium

import (
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/nslookup"
)

func init() {
	nslookup.PrefixURI = namespacePrefixURI
	nslookup.ByURI = namespaceByURIHook
}

// namespaceByPrefix walks raw namespace declarations without cloning each
// element's declaration slice. The returned Namespace remains read-only to
// sibling packages because its fields are unexported.
func namespaceByPrefix(start Node, prefix string) (*Namespace, bool) {
	for node := start; !isNilNode(node); node = node.Parent() {
		el, ok := node.(*Element)
		if !ok {
			continue
		}
		for _, ns := range el.nsDefs {
			if ns != nil && ns.Prefix() == prefix {
				return ns, true
			}
		}
	}
	return nil, false
}

// namespaceByURI is the URI counterpart to namespaceByPrefix. A nearer
// declaration or undeclaration hides every ancestor binding for the same
// prefix, even when the nearer URI does not match.
func namespaceByURI(start Node, uri string) (*Namespace, bool) {
	for node := start; !isNilNode(node); node = node.Parent() {
		el, ok := node.(*Element)
		if !ok {
			continue
		}
		for _, ns := range el.nsDefs {
			if ns == nil {
				continue
			}
			if ns.URI() != uri {
				continue
			}
			active, found := namespaceByPrefix(start, ns.Prefix())
			if found && active == ns {
				return ns, true
			}
		}
	}
	return nil, false
}

// namespacePrefixURI adapts namespaceByPrefix to the internal hook without
// exposing the Namespace object or its containing slice.
func namespacePrefixURI(start any, prefix string) (string, bool) {
	node, ok := start.(Node)
	if !ok {
		return "", false
	}
	ns, found := namespaceByPrefix(node, prefix)
	if !found {
		return "", false
	}
	return ns.URI(), true
}

// namespaceByURIHook adapts namespaceByURI to the untyped internal hook.
func namespaceByURIHook(start any, uri string) (any, bool) {
	node, ok := start.(Node)
	if !ok {
		return nil, false
	}
	ns, found := namespaceByURI(node, uri)
	if !found {
		return nil, false
	}
	return ns, true
}

// LookupNSByPrefix walks the element and its ancestors to find a namespace
// declaration matching the given prefix. The "xml" prefix is always
// implicitly bound to the XML namespace.
func LookupNSByPrefix(e *Element, prefix string) *Namespace {
	if ns, found := namespaceByPrefix(e, prefix); found {
		return ns
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
	if ns, found := namespaceByURI(e, href); found {
		return ns
	}
	return nil
}
