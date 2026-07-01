package xsd

import (
	"context"
	"strings"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/xmlchar"
	"github.com/lestrrat-go/helium/internal/xsd/value"
)

// Conditional inclusion (XSD 1.1, version-control namespace
// http://www.w3.org/2007/XMLSchema-versioning) is a PRE-PASS over a parsed
// schema document, run before the element tree is interpreted
// (parseSchemaChildren). An element — together with its whole subtree — is
// removed from the tree unless EVERY vc: condition on it holds for the active
// processor version:
//
//   - vc:minVersion / vc:maxVersion (xs:decimal): keep iff
//     minVersion <= processorVersion < maxVersion.
//   - vc:typeAvailable     (QName list): keep iff EVERY listed type is available.
//   - vc:typeUnavailable   (QName list): keep iff at least one listed type is
//     UNavailable (equivalently: drop iff every listed type is available).
//   - vc:facetAvailable    (QName list): keep iff EVERY listed facet is available.
//   - vc:facetUnavailable  (QName list): keep iff at least one listed facet is
//     UNavailable (drop iff every listed facet is available).
//
// The empty-list edge cases fall out of the "all available" formulation: an
// empty *Available list is vacuously all-available (no effect — never drops),
// while an empty *Unavailable list is also vacuously all-available and so is an
// unconditional exclude. These match the W3C VC test suite (saxonData/VC).
//
// "Available" means a built-in known to the active processor version. A type
// QName is available iff it is in the XSD namespace and registered as a built-in
// for the version (so xs:integer is available in both 1.0 and 1.1, xs:error only
// in 1.1, and an unknown name like xs:bananaSkin or a non-XSD QName such as
// vc:list-of-QNames is unavailable). A facet QName is available iff it is in the
// XSD namespace and names a facet recognized by the version (the 1.1-only facets
// xs:assertion and xs:explicitTimezone are unavailable under 1.0).
//
// Conditional inclusion is the cross-version mechanism for schemas that carry
// both 1.0 and 1.1 formulations, so it is honored in BOTH version modes (a 1.0
// processor prunes 1.1-requiring elements; a 1.1 processor prunes 1.0
// fallbacks). Lexical/QName validity of vc: attribute values is, however, only
// ENFORCED as a schema error under 1.1 — under 1.0 a malformed vc: attribute is
// tolerated (its condition is simply skipped), matching the suite's vc902 (a
// schema with a bad vc:minVersion that is valid under 1.0 but invalid under 1.1).
//
// Misspelt or otherwise unrecognized attributes in the versioning namespace
// (e.g. vc:minversion, vc:what-on-earth) are foreign attributes with no effect:
// only the six attributes listed above are consulted.

const (
	vcMinVersion       = "minVersion"
	vcMaxVersion       = "maxVersion"
	vcTypeAvailable    = "typeAvailable"
	vcTypeUnavailable  = "typeUnavailable"
	vcFacetAvailable   = "facetAvailable"
	vcFacetUnavailable = "facetUnavailable"
)

// xsdFacetNames is the set of facet local names recognized by an XSD 1.0
// processor. The 1.1-only facets are tracked separately in xsdFacetNames11. The
// names are kept as one space-separated literal (split into a set at init) so the
// individual facet names are not scattered string literals across the package.
var xsdFacetNames = facetNameSet("length minLength maxLength pattern enumeration whiteSpace maxInclusive maxExclusive minInclusive minExclusive totalDigits fractionDigits")

// xsdFacetNames11 is the set of facet local names introduced by XSD 1.1.
var xsdFacetNames11 = facetNameSet("assertion explicitTimezone")

// facetNameSet builds a presence set from a space-separated list of facet names.
func facetNameSet(s string) map[string]struct{} {
	m := make(map[string]struct{})
	for _, n := range splitSpace(s) {
		m[n] = struct{}{}
	}
	return m
}

// applyConditionalInclusion prunes the elements of a parsed schema-document tree
// that are excluded by their version-control (vc:) attributes for the compiler's
// active XSD version. It runs before the tree is interpreted, so a removed
// element (and its subtree) is never compiled. Nodes to remove are collected
// first and unlinked afterward so the tree is not mutated mid-iteration.
//
// If the root <xs:schema> element itself is conditionally excluded, the whole
// document contributes nothing: all of its element children are dropped, leaving
// an empty (but valid) schema, and the function returns true so the caller can
// short-circuit BEFORE interpreting/validating the root's other (non-preserved)
// attributes (e.g. blockDefault/finalDefault) — an excluded root must not error
// on attributes it would never use.
// documentHasVCDirective reports whether elem or any descendant carries an
// attribute in the version-control namespace. It is a cheap pre-scan the
// TOP-LEVEL compile uses to decide whether the conditional-inclusion pre-pass
// could mutate the tree at all: when no vc attribute is present the pre-pass is a
// guaranteed no-op, so the caller's parsed document can be compiled in place
// without a defensive deep copy (the fast, no-allocation path that the vast
// majority of schemas take). Only when a vc directive IS present does the
// top-level compile clone the document so pruning never mutates the caller's DOM.
func documentHasVCDirective(elem *helium.Element) bool {
	if elem == nil {
		return false
	}
	for _, a := range elem.Attributes() {
		if a.URI() == lexicon.NamespaceXSDVersioning {
			return true
		}
	}
	for ch := range helium.Children(elem) {
		if ch.Type() != helium.ElementNode {
			continue
		}
		child, ok := helium.AsNode[*helium.Element](ch)
		if !ok {
			continue
		}
		if documentHasVCDirective(child) {
			return true
		}
	}
	return false
}

func (c *compiler) applyConditionalInclusion(ctx context.Context, root *helium.Element) bool {
	pv := c.processorVersionString()

	if c.vcExcluded(ctx, root, pv) {
		var kids []helium.Node
		for ch := range helium.Children(root) {
			kids = append(kids, ch)
		}
		for _, ch := range kids {
			if mn, ok := ch.(helium.MutableNode); ok {
				helium.UnlinkNode(mn)
			}
		}
		return true
	}

	var toRemove []*helium.Element
	c.collectConditionalExclusions(ctx, root, pv, &toRemove)
	for _, elem := range toRemove {
		helium.UnlinkNode(elem)
	}
	return false
}

// collectConditionalExclusions walks the element children of parent, recording
// every element excluded by its vc: attributes into out. A pruned element's
// subtree is not descended into (the whole subtree goes away with it).
func (c *compiler) collectConditionalExclusions(ctx context.Context, parent *helium.Element, pv string, out *[]*helium.Element) {
	for ch := range helium.Children(parent) {
		if ch.Type() != helium.ElementNode {
			continue
		}
		elem, ok := helium.AsNode[*helium.Element](ch)
		if !ok {
			continue
		}
		if c.vcExcluded(ctx, elem, pv) {
			*out = append(*out, elem)
			continue
		}
		c.collectConditionalExclusions(ctx, elem, pv, out)
	}
}

// processorVersionString returns the active processor version as the exact
// xs:decimal lexical string ("1.0" or "1.1") used to compare against
// vc:minVersion/vc:maxVersion. A string (not a float64) so the comparison is
// done with exact arbitrary-precision arithmetic (value.CompareDecimal /
// math/big.Rat) — xs:decimal is exact, and a float would mis-round a
// high-precision bound (e.g. "1.1000…001") or overflow a many-digit one.
func (c *compiler) processorVersionString() string {
	if c.version == Version11 {
		return "1.1"
	}
	return "1.0"
}

// vcExcluded reports whether elem is conditionally excluded for processor
// version pv (an exact xs:decimal string). Every present vc: condition is
// evaluated (so malformed values are still diagnosed under 1.1); the element is
// excluded if ANY condition fails. vc:minVersion/vc:maxVersion are compared
// exactly via value.CompareDecimal: keep iff minVersion <= pv < maxVersion, so
// exclude iff pv < minVersion or pv >= maxVersion.
func (c *compiler) vcExcluded(ctx context.Context, elem *helium.Element, pv string) bool {
	excluded := false

	if v, ok := getVCAttr(elem, vcMinVersion); ok {
		if d, valid := c.vcDecimal(ctx, elem, vcMinVersion, v); valid && value.CompareDecimal(pv, d) < 0 {
			excluded = true
		}
	}
	if v, ok := getVCAttr(elem, vcMaxVersion); ok {
		if d, valid := c.vcDecimal(ctx, elem, vcMaxVersion, v); valid && value.CompareDecimal(pv, d) >= 0 {
			excluded = true
		}
	}
	if v, ok := getVCAttr(elem, vcTypeAvailable); ok {
		if all, valid := c.vcAllAvailable(ctx, elem, vcTypeAvailable, v, false); valid && !all {
			excluded = true
		}
	}
	if v, ok := getVCAttr(elem, vcTypeUnavailable); ok {
		if all, valid := c.vcAllAvailable(ctx, elem, vcTypeUnavailable, v, false); valid && all {
			excluded = true
		}
	}
	if v, ok := getVCAttr(elem, vcFacetAvailable); ok {
		if all, valid := c.vcAllAvailable(ctx, elem, vcFacetAvailable, v, true); valid && !all {
			excluded = true
		}
	}
	if v, ok := getVCAttr(elem, vcFacetUnavailable); ok {
		if all, valid := c.vcAllAvailable(ctx, elem, vcFacetUnavailable, v, true); valid && all {
			excluded = true
		}
	}

	return excluded
}

// vcDecimal validates a vc:minVersion/vc:maxVersion value as an xs:decimal and
// returns the whitespace-trimmed lexical string for exact comparison. "Malformed"
// means not a valid xs:decimal lexical form (a bad sign/dot/non-digit) — NOT a
// magnitude that a float could not hold; a many-digit or high-precision decimal
// is valid and compared exactly by the caller. A malformed value is a fatal
// schema error UNDER 1.1 only; under 1.0 it is tolerated (valid=false so the
// caller skips the condition).
func (c *compiler) vcDecimal(ctx context.Context, elem *helium.Element, attr, val string) (string, bool) {
	// Trim only ASCII XML whitespace (#x20/#x9/#xD/#xA), the whitespace facet's
	// scope for xs:decimal — NOT strings.TrimSpace, which also strips Unicode
	// whitespace like NBSP. So vc:minVersion="<NBSP>1.1" stays malformed (fatal
	// under 1.1) rather than being silently accepted as "1.1".
	s := strings.Trim(val, " \t\r\n")
	if isValidXSDDecimal(s) {
		return s, true
	}
	if c.version == Version11 && c.filename != "" {
		c.schemaError(ctx, schemaParserErrorAttr(c.diagSource(), elem.Line(), elem.LocalName(), elem.LocalName(),
			"vc:"+attr, "The value '"+val+"' is not a valid xs:decimal."))
	}
	return "", false
}

// vcAllAvailable evaluates a vc:typeAvailable/typeUnavailable (facet=false) or
// vc:facetAvailable/facetUnavailable (facet=true) QName-list value. It returns
// (allAvailable, valid): allAvailable is whether EVERY QName in the list names a
// component available to the active version (vacuously true for an empty list);
// valid is false (with a fatal schema error under 1.1) when any list item is not
// a valid QName or carries an unbound prefix.
func (c *compiler) vcAllAvailable(ctx context.Context, elem *helium.Element, attr, value string, facet bool) (bool, bool) {
	all := true
	for _, tok := range splitSpace(value) {
		if !xmlchar.IsValidQName(tok) {
			if c.version == Version11 && c.filename != "" {
				c.schemaError(ctx, schemaParserErrorAttr(c.diagSource(), elem.Line(), elem.LocalName(), elem.LocalName(),
					"vc:"+attr, "The value '"+value+"' is not a valid list of QNames."))
			}
			return false, false
		}
		ns, local, ok := c.resolveVCQName(elem, tok)
		if !ok {
			if c.version == Version11 && c.filename != "" {
				c.schemaError(ctx, schemaParserErrorAttr(c.diagSource(), elem.Line(), elem.LocalName(), elem.LocalName(),
					"vc:"+attr, "The QName '"+tok+"' has an unbound namespace prefix."))
			}
			return false, false
		}
		var available bool
		if facet {
			available = c.facetAvailable(ns, local)
		} else {
			available = c.typeAvailable(ns, local)
		}
		if !available {
			all = false
		}
	}
	return all, true
}

// resolveVCQName resolves a vc: QName-list item to its (namespace, local)
// expansion against elem's in-scope namespaces. ok is false when the QName
// carries a prefix that is not bound in scope.
func (c *compiler) resolveVCQName(elem *helium.Element, qname string) (string, string, bool) {
	prefix, local, found := strings.Cut(qname, ":")
	if !found {
		// Unprefixed: bind to the in-scope default namespace (possibly empty).
		return lookupNS(elem, ""), qname, true
	}
	ns := lookupNS(elem, prefix)
	if ns == "" {
		return "", "", false
	}
	return ns, local, true
}

// typeAvailable reports whether the type named {ns}local is a built-in known to
// the active processor version. Only XSD-namespace built-ins count, checked
// against the IMMUTABLE per-version capability set (builtinTypeAvailable) — NOT
// c.schema.types, which can already hold user/included declarations: a 1.0 schema
// that declares a type literally named {XSD}error must not make
// vc:typeAvailable="xs:error" true (capability detection, not "is it declared").
// A non-XSD or unknown type is unavailable.
func (c *compiler) typeAvailable(ns, local string) bool {
	if ns != lexicon.NamespaceXSD {
		return false
	}
	return builtinTypeAvailable(local, c.version)
}

// facetAvailable reports whether the facet named {ns}local is recognized by the
// active processor version. The 1.1-only facets (assertion, explicitTimezone)
// are available only in 1.1 mode.
func (c *compiler) facetAvailable(ns, local string) bool {
	if ns != lexicon.NamespaceXSD {
		return false
	}
	if _, ok := xsdFacetNames[local]; ok {
		return true
	}
	if c.version == Version11 {
		_, ok := xsdFacetNames11[local]
		return ok
	}
	return false
}

// getVCAttr reads a version-control (vc:) attribute by local name, reporting
// whether it is present (distinct from an empty value, which carries meaning for
// vc: list attributes).
func getVCAttr(elem *helium.Element, name string) (string, bool) {
	attr, ok := elem.FindAttribute(helium.NSPredicate{Local: name, NamespaceURI: lexicon.NamespaceXSDVersioning})
	if !ok {
		return "", false
	}
	return attr.Value(), true
}

// isValidXSDDecimal reports whether s is a valid xs:decimal LEXICAL form:
//
//	(+|-)? ( digits ('.' digits?)? | '.' digits )
//
// It validates form only — magnitude is irrelevant (xs:decimal is unbounded), so
// an arbitrary-precision value passes and is compared exactly by the caller via
// value.CompareDecimal (math/big.Rat). Scientific notation, INF, and NaN are not
// valid decimals.
func isValidXSDDecimal(s string) bool {
	if s == "" {
		return false
	}
	body := s
	if body[0] == '+' || body[0] == '-' {
		body = body[1:]
	}
	if body == "" {
		return false
	}
	digits := false
	dot := false
	for i := range len(body) {
		ch := body[i]
		switch {
		case ch >= '0' && ch <= '9':
			digits = true
		case ch == '.':
			if dot {
				return false
			}
			dot = true
		default:
			return false
		}
	}
	return digits
}
