package examples_test

import (
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium"
)

type valuePrefixPredicate string

// Match makes valuePrefixPredicate satisfy helium.AttributePredicate.
// Custom predicates let callers express lookups that are not covered by the
// built-in QName/local-name/namespace helpers.
func (p valuePrefixPredicate) Match(a *helium.Attribute) bool {
	return strings.HasPrefix(a.Value(), string(p))
}

func Example_helium_attribute_predicates() {
	// Build a single element with both plain and namespaced attributes so the
	// example can show how each predicate chooses a match.
	doc := helium.NewDefaultDocument()
	item, err := doc.CreateElement("item")
	if err != nil {
		fmt.Printf("failed to create element: %s\n", err)
		return
	}

	// The plain "id" attribute and the namespaced "cfg:id" attribute share the
	// same local name. This makes the difference between QName, local-name, and
	// namespace-aware matching visible in the output.
	ns := helium.NewNamespace("cfg", "http://example.com/cfg")
	if err := item.SetAttribute("id", "42"); err != nil {
		fmt.Printf("failed to set attribute: %s\n", err)
		return
	}
	if err := item.SetAttributeNS("id", "cfg-42", ns); err != nil {
		fmt.Printf("failed to set namespaced attribute: %s\n", err)
		return
	}
	if err := item.SetAttribute("role", "admin"); err != nil {
		fmt.Printf("failed to set attribute: %s\n", err)
		return
	}

	// show runs FindAttribute with the supplied predicate and prints the matched
	// attribute. A real program would typically inspect attr.Value(), attr.Name(),
	// attr.LocalName(), or attr.URI() depending on what it needs next.
	show := func(label string, pred helium.AttributePredicate) {
		attr, ok := item.FindAttribute(pred)
		if !ok {
			fmt.Printf("%s: not found\n", label)
			return
		}
		fmt.Printf("%s: %s=%s\n", label, attr.Name(), attr.Value())
	}

	// QNamePredicate matches the exact attribute name as returned by attr.Name().
	// Use this when you already know the lexical QName, including any prefix.
	show("QNamePredicate", helium.QNamePredicate("cfg:id"))

	// LocalNamePredicate ignores the namespace and matches by local name only.
	// Because FindAttribute returns the first match in attribute order, this finds
	// the plain "id" attribute that was added before "cfg:id".
	show("LocalNamePredicate", helium.LocalNamePredicate("id"))

	// NSPredicate matches by local name plus namespace URI. This is the safest
	// choice when prefixes may vary but the namespace identity matters.
	show("NSPredicate", helium.NSPredicate{Local: "id", NamespaceURI: ns.URI()})

	// Any type with a Match(*helium.Attribute) bool method can be used as a
	// custom predicate. Here we match the first attribute whose value starts with
	// "cfg-".
	show("custom", valuePrefixPredicate("cfg-"))
	// Output:
	// QNamePredicate: cfg:id=cfg-42
	// LocalNamePredicate: id=42
	// NSPredicate: cfg:id=cfg-42
	// custom: cfg:id=cfg-42
}
