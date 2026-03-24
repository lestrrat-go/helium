package xsd

import (
	"strconv"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
)

func findDocumentElement(doc *helium.Document) *helium.Element {
	return doc.DocumentElement()
}

// collectNSContext collects namespace declarations from a schema element and its ancestors.
func collectNSContext(elem *helium.Element) map[string]string {
	nsMap := make(map[string]string)
	var node helium.Node = elem
	for node != nil {
		if e, ok := node.(*helium.Element); ok {
			for _, ns := range e.Namespaces() {
				prefix := ns.Prefix()
				if _, exists := nsMap[prefix]; !exists {
					nsMap[prefix] = ns.URI()
				}
			}
		}
		node = node.Parent()
	}
	return nsMap
}

func isXSDElement(elem *helium.Element, localName string) bool {
	return elem.LocalName() == localName && elem.URI() == lexicon.NamespaceXSD
}

func getAttr(elem *helium.Element, name string) string {
	attr, ok := elem.FindAttribute(helium.LocalNamePredicate(name))
	if !ok {
		return ""
	}
	return attr.Value()
}

// parseBlockFlags parses a block attribute value into BlockFlags.
func parseBlockFlags(v string) BlockFlags {
	if v == "#all" {
		return BlockExtension | BlockRestriction | BlockSubstitution
	}
	var f BlockFlags
	for _, part := range splitSpace(v) {
		switch part {
		case attrValExtension:
			f |= BlockExtension
		case attrValRestriction:
			f |= BlockRestriction
		case attrValSubstitution:
			f |= BlockSubstitution
		}
	}
	return f
}

// parseFinalFlags parses a finalDefault or simpleType final attribute value into FinalFlags.
func parseFinalFlags(v string) FinalFlags {
	if v == "#all" {
		return FinalExtension | FinalRestriction | FinalList | FinalUnion
	}
	var f FinalFlags
	for _, part := range splitSpace(v) {
		switch part {
		case attrValExtension:
			f |= FinalExtension
		case attrValRestriction:
			f |= FinalRestriction
		case attrValList:
			f |= FinalList
		case attrValUnion:
			f |= FinalUnion
		}
	}
	return f
}

// parseElemFinalFlags parses a final attribute value for elements/complexTypes
// (only extension/restriction are valid).
func parseElemFinalFlags(v string) FinalFlags {
	if v == "#all" {
		return FinalExtension | FinalRestriction
	}
	var f FinalFlags
	for _, part := range splitSpace(v) {
		switch part {
		case attrValExtension:
			f |= FinalExtension
		case attrValRestriction:
			f |= FinalRestriction
		}
	}
	return f
}

func lookupNS(elem *helium.Element, prefix string) string {
	// Walk up the tree looking for namespace declarations.
	var node helium.Node = elem
	for node != nil {
		if e, ok := node.(*helium.Element); ok {
			for _, ns := range e.Namespaces() {
				if ns.Prefix() == prefix {
					return ns.URI()
				}
			}
			// Also check the element's own namespace.
			if e.Prefix() == prefix {
				return e.URI()
			}
		}
		node = node.Parent()
	}
	return ""
}

func parseOccurs(s string, defaultVal int) int {
	if s == "unbounded" {
		return Unbounded
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return n
}

func registerBuiltinTypes(s *Schema) {
	builtins := []string{
		"string", "boolean", "decimal", "float", "double",
		"integer", "nonPositiveInteger", "negativeInteger",
		"long", "int", "short", "byte",
		"nonNegativeInteger", "unsignedLong", "unsignedInt", "unsignedShort", "unsignedByte",
		"positiveInteger",
		"normalizedString", "token", "language", "Name", "NCName",
		"ID", "IDREF", "IDREFS", "ENTITY", "ENTITIES", "NMTOKEN", "NMTOKENS",
		"date", "dateTime", "time", "duration",
		"gYearMonth", "gYear", "gMonthDay", "gDay", "gMonth",
		"hexBinary", "base64Binary",
		"anyURI", "QName", "NOTATION",
		"anyType", "anySimpleType",
	}
	for _, name := range builtins {
		qn := QName{Local: name, NS: lexicon.NamespaceXSD}
		ct := ContentTypeSimple
		td := &TypeDef{
			Name:        qn,
			ContentType: ct,
		}
		if name == "anyType" {
			td.ContentType = ContentTypeMixed
			td.AnyAttribute = &Wildcard{
				Namespace:       WildcardNSAny,
				ProcessContents: ProcessLax,
			}
		}
		s.types[qn] = td
	}
}
