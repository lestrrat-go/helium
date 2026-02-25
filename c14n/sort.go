package c14n

import (
	"sort"

	helium "github.com/lestrrat-go/helium"
)

// nsSortEntry holds a namespace for sorting purposes.
type nsSortEntry struct {
	prefix string
	uri    string
}

// sortNamespaces sorts namespace declarations per C14N rules:
// by prefix lexicographically, with empty prefix (default namespace) first.
func sortNamespaces(nss []nsSortEntry) {
	sort.Slice(nss, func(i, j int) bool {
		return nss[i].prefix < nss[j].prefix
	})
}

// attrSortEntry holds an attribute for sorting purposes.
type attrSortEntry struct {
	attr       *helium.Attribute
	nsURI      string
	localName  string
	fixupValue string // non-empty for synthetic xml:base fixup attrs (C14N 1.1)
}

// sortAttributes sorts attributes per C14N rules:
// no-namespace attributes first (sorted by name), then namespaced
// attributes sorted by (namespace URI, local name).
func sortAttributes(attrs []attrSortEntry) {
	sort.Slice(attrs, func(i, j int) bool {
		ai, aj := attrs[i], attrs[j]
		// No-namespace attrs come first
		if ai.nsURI == "" && aj.nsURI != "" {
			return true
		}
		if ai.nsURI != "" && aj.nsURI == "" {
			return false
		}
		if ai.nsURI == "" && aj.nsURI == "" {
			return ai.localName < aj.localName
		}
		// Both have namespaces: sort by URI then local name
		if ai.nsURI != aj.nsURI {
			return ai.nsURI < aj.nsURI
		}
		return ai.localName < aj.localName
	})
}
