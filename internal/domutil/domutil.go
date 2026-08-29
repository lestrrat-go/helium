// Package domutil hosts small DOM/QName helpers shared by several helium
// processing packages (c14n, xmldsig1, xmlenc1, xpath1, xpath3, xsd, xslt3). It may
// import the helium root package; the root never imports it back, so there is
// no import cycle.
package domutil

import (
	"slices"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/nslookup"
	"github.com/lestrrat-go/helium/internal/xmlchar"
)

// LocalName returns the element's local name, i.e. the part after the first
// ':' in its tag name, or the whole name when there is no prefix.
func LocalName(e *helium.Element) string {
	name := e.Name()
	for i := range len(name) {
		if name[i] == ':' {
			return name[i+1:]
		}
	}
	return name
}

// TextContent returns the concatenated Content() of an element's direct
// children.
func TextContent(e *helium.Element) string {
	var sb []byte
	for child := e.FirstChild(); child != nil; child = child.NextSibling() {
		sb = append(sb, child.Content()...)
	}
	return string(sb)
}

// XMLLangMatches walks the ancestor chain from n looking for an xml:lang
// attribute. langArg must already be lower-cased by the caller. It returns
// (matched, found): found is true once an xml:lang attribute is encountered,
// at which point matched reports whether its lower-cased value equals langArg
// or begins with langArg+"-". When no xml:lang is found, both are false.
func XMLLangMatches(n helium.Node, langArg string) (matched, found bool) {
	for cur := n; cur != nil; cur = cur.Parent() {
		elem, ok := cur.(*helium.Element)
		if !ok {
			continue
		}
		for _, attr := range elem.Attributes() {
			if attr.LocalName() == "lang" && attr.URI() == lexicon.NamespaceXML {
				val := strings.ToLower(attr.Value())
				if val == langArg || strings.HasPrefix(val, langArg+"-") {
					return true, true
				}
				return false, true
			}
		}
	}
	return false, false
}

// InScopeNamespaces collects all in-scope namespace bindings for e by walking
// the ancestor chain from the document root down to e, so that closer (inner)
// declarations override outer ones. The result is keyed by prefix. When
// dropXML is true the implicit "xml" prefix binding is removed.
func InScopeNamespaces(e *helium.Element, dropXML bool) map[string]*helium.Namespace {
	byPrefix := make(map[string]*helium.Namespace)

	var chain []*helium.Element
	for n := helium.Node(e); n != nil; n = n.Parent() {
		if anc, ok := helium.AsNode[*helium.Element](n); ok {
			chain = append(chain, anc)
		}
	}

	// Outermost to innermost so the innermost binding wins.
	for _, anc := range slices.Backward(chain) {
		for _, ns := range anc.Namespaces() {
			byPrefix[ns.Prefix()] = ns
		}
		if ns := anc.Namespace(); ns != nil {
			if _, ok := byPrefix[ns.Prefix()]; !ok {
				byPrefix[ns.Prefix()] = ns
			}
		}
	}

	if dropXML {
		delete(byPrefix, lexicon.PrefixXML)
	}

	return byPrefix
}

// LookupNSPrefixURI walks start and its ancestors for a namespace declaration
// matching prefix, returning the bound URI. Unlike helium.LookupNSByPrefix it
// does NOT predeclare the "xml" prefix — it replicates the bare ancestor walk
// used by several call sites that intentionally omit that predeclaration. The
// bool reports whether a binding was found. start may be any node (or nil);
// non-element nodes in the chain are skipped.
func LookupNSPrefixURI(start helium.Node, prefix string) (string, bool) {
	return nslookup.PrefixURI(start, prefix)
}

// LookupNSURI walks start and its ancestors for the nearest namespace
// declaration matching uri. Unlike helium.LookupNSByHref it does not
// synthesize the implicit XML namespace declaration. The bool reports whether
// a declaration was found. start may be any node (or nil); non-element nodes
// in the chain are skipped.
func LookupNSURI(start helium.Node, uri string) (*helium.Namespace, bool) {
	raw, found := nslookup.ByURI(start, uri)
	if !found {
		return nil, false
	}
	ns, ok := raw.(*helium.Namespace)
	return ns, ok
}

// NamespaceDeclarationAt returns one namespace declaration from e by index.
// It reads the element's private declaration storage without cloning the whole
// slice, so callers with a resource cap can stop before visiting the remainder.
func NamespaceDeclarationAt(e *helium.Element, index int) (*helium.Namespace, bool) {
	raw, found := nslookup.DeclarationAt(e, index)
	if !found {
		return nil, false
	}
	ns, ok := raw.(*helium.Namespace)
	return ns, ok
}

// SplitLexicalQName trims surrounding whitespace from s, splits it at the first
// ':' into prefix and local, and validates each part as an NCName. It is
// error-code free: callers map a failure to whatever XPath error code
// (FORG0001, FOCA0002, ...) applies in their context. hadColon reports whether
// a ':' was present (so callers can flag a leading colon, where hadColon is
// true but prefix is empty). validNC reports whether prefix (when present) and
// local are both valid NCNames.
func SplitLexicalQName(s string) (prefix, local string, hadColon, validNC bool) {
	s = strings.TrimSpace(s)
	prefix = ""
	local = s
	if p, l, found := strings.Cut(s, ":"); found {
		prefix = p
		local = l
		hadColon = true
	}
	validNC = xmlchar.IsValidNCName(local) && (prefix == "" || xmlchar.IsValidNCName(prefix))
	return prefix, local, hadColon, validNC
}
