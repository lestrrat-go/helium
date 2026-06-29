package xsd

import (
	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

// emptyCollectionResolver supplies an empty default collection so an XSD 1.1
// conditional-type-assignment / assertion test may call fn:collection() without
// triggering FODC0002 (there is no document collection during schema validation).
type emptyCollectionResolver struct{}

func (emptyCollectionResolver) ResolveCollection(string) (xpath3.Sequence, error) {
	return xpath3.EmptySequence(), nil
}

func (emptyCollectionResolver) ResolveURICollection(string) ([]string, error) {
	return []string{}, nil
}

// inheritedAttributes returns the XSD 1.1 inherited attributes in scope for elem:
// the attributes of its ancestors whose {inheritable} property is true, with a
// nearer ancestor's attribute shadowing a farther one of the same expanded name.
// A non-inheritable ancestor attribute is ignored entirely, so it does NOT mask
// an inheritable attribute of the same name on a higher ancestor. Inheritability
// is recorded per attribute node during attribute validation (vc.attrInheritable),
// which the top-down validation walk populates for every ancestor before a
// descendant's conditional type assignment runs.
func (vc *validationContext) inheritedAttributes(elem *helium.Element) []*helium.Attribute {
	if vc.version != Version11 || len(vc.attrInheritable) == 0 {
		return nil
	}
	var result []*helium.Attribute
	seen := make(map[QName]struct{})
	for anc := elem.Parent(); anc != nil; anc = anc.Parent() {
		ae, ok := helium.AsNode[*helium.Element](anc)
		if !ok {
			continue
		}
		for _, a := range ae.Attributes() {
			if _, marked := vc.attrInheritable[a]; !marked {
				continue
			}
			qn := QName{Local: a.LocalName(), NS: a.URI()}
			if _, dup := seen[qn]; dup {
				continue
			}
			seen[qn] = struct{}{}
			result = append(result, a)
		}
	}
	return result
}

// ctaContextNode builds the XSD 1.1 conditional-type-assignment / assertion XPath
// context node for elem: a detached element copy carrying the element's name,
// in-scope namespaces, own attributes and inherited attributes, but NO children
// and NO parent. Per XSD 1.1 §3.13 the test sees only attributes — empty(..),
// string()=” and `. is root()` therefore hold — while preserving the namespace
// context so resolve-QName(@x, .) and prefixed name tests resolve, and exposing
// inherited ancestor attributes so a test like @c:kind matches them.
func (vc *validationContext) ctaContextNode(elem *helium.Element) *helium.Element {
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneExplicitNo)
	// Carry the INSTANCE document's URI onto the synthetic document so fn:base-uri(.)
	// on the CTA context node resolves to the instance document (the element bears no
	// xml:base), matching the data model — a @test like ends-with(base-uri(.), '…')
	// must see the instance URI, not the empty string (cta0021). fn:static-base-uri()
	// stays the SCHEMA URI, set separately on the evaluator from alt.BaseURI.
	if owner := elem.OwnerDocument(); owner != nil && owner.URL() != "" {
		doc.SetURL(owner.URL())
	}
	// Use the LOCAL name: elem.Name() is the lexical QName (e.g. "p:root") for a
	// prefixed element, which CreateElement would store verbatim and SetActiveNamespace
	// would then leave with a wrong local part, so a name test in @test would miss.
	synth := doc.CreateElement(elem.LocalName())
	for prefix, uri := range collectNSContext(elem) {
		_ = synth.DeclareNamespace(prefix, uri)
	}
	// Link the element's own namespace so a self:: / name test in @test sees the
	// element in its real namespace (DeclareNamespace alone does not set it active).
	if elem.URI() != "" {
		_ = synth.SetActiveNamespace(elem.Prefix(), elem.URI())
	}
	seen := make(map[QName]struct{})
	for _, a := range elem.Attributes() {
		// The CTA context node keeps every real attribute (XDM excludes only
		// namespace declarations); unlike isSpecialAttr this RETAINS xml:* attributes
		// (xml:lang/xml:base/xml:space), so a test like @xml:lang='de' can drive CTA.
		if isNamespaceDeclAttr(a) {
			continue
		}
		seen[QName{Local: a.LocalName(), NS: a.URI()}] = struct{}{}
		addSynthAttr(synth, a)
	}
	for _, a := range vc.inheritedAttributes(elem) {
		qn := QName{Local: a.LocalName(), NS: a.URI()}
		if _, ok := seen[qn]; ok {
			continue
		}
		seen[qn] = struct{}{}
		addSynthAttr(synth, a)
	}
	return synth
}

// isNamespaceDeclAttr reports whether a is a namespace declaration (xmlns or
// xmlns:prefix). Unlike isSpecialAttr it does NOT treat xml:*/xsi:* attributes as
// special, so the CTA context node retains them.
func isNamespaceDeclAttr(a *helium.Attribute) bool {
	p := a.Prefix()
	return p == "xmlns" || (p == "" && a.LocalName() == "xmlns")
}

// addSynthAttr copies attribute a onto the synthetic context element synth,
// preserving its namespace. The value is set literally (entity references in the
// source were already resolved during parsing).
func addSynthAttr(synth *helium.Element, a *helium.Attribute) {
	if a.URI() == "" {
		_ = synth.SetLiteralAttribute(a.LocalName(), a.Value())
		return
	}
	_ = synth.SetLiteralAttributeNS(a.LocalName(), a.Value(), helium.NewNamespace(a.Prefix(), a.URI()))
}
