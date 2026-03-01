package c14n

import (
	"cmp"
	"slices"

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
	slices.SortFunc(nss, func(a, b nsSortEntry) int {
		return cmp.Compare(a.prefix, b.prefix)
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
	slices.SortFunc(attrs, func(a, b attrSortEntry) int {
		// No-namespace attrs come first
		if a.nsURI == "" && b.nsURI != "" {
			return -1
		}
		if a.nsURI != "" && b.nsURI == "" {
			return 1
		}
		if a.nsURI == "" && b.nsURI == "" {
			return cmp.Compare(a.localName, b.localName)
		}
		// Both have namespaces: sort by URI then local name
		if c := cmp.Compare(a.nsURI, b.nsURI); c != 0 {
			return c
		}
		return cmp.Compare(a.localName, b.localName)
	})
}
