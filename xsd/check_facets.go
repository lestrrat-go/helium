package xsd

import (
	"context"
	"fmt"
	"slices"
	"sort"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/xsd/value"
)

const (
	facetMinInclusive = "minInclusive"
	facetMaxInclusive = "maxInclusive"
	facetMinExclusive = "minExclusive"
	facetMaxExclusive = "maxExclusive"
)

// baseFacets returns the FacetSet from the nearest base type in the chain.
func baseFacets(td *TypeDef) *FacetSet {
	if td.BaseType == nil {
		return nil
	}
	for cur := range baseChain(td.BaseType) {
		if cur.Facets != nil {
			return cur.Facets
		}
	}
	return nil
}

// checkFacetConsistency validates facet constraints for every facet-bearing
// simple type — named globals AND inline/anonymous (local) simple types. It
// iterates c.typeDefSources rather than c.schema.types so that inline simple
// types on elements/attributes (which never enter the named-type table) are
// checked too; otherwise an invalid bound on an anonymous type would slip
// through checkFacetValueAgainstBase and become the very no-op this guards
// against. It checks the facet-value-against-base bound, same-type mutual
// exclusion, same-type consistency, and base-type restriction narrowing rules.
func (c *compiler) checkFacetConsistency(ctx context.Context) {
	if c.filename == "" {
		return
	}

	// Collect and sort facet-bearing simple types by source line (then local
	// name) for deterministic error ordering.
	type facetEntry struct {
		td  *TypeDef
		src typeDefSource
	}
	var entries []facetEntry
	for td, src := range c.typeDefSources {
		if td.Facets == nil {
			continue
		}
		if td.Name.NS == lexicon.NamespaceXSD {
			continue
		}
		entries = append(entries, facetEntry{td: td, src: src})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].src.line != entries[j].src.line {
			return entries[i].src.line < entries[j].src.line
		}
		if entries[i].td.Name.Local != entries[j].td.Name.Local {
			return entries[i].td.Name.Local < entries[j].td.Name.Local
		}
		// Final tie-breaker: anonymous types share an empty name (and may share a
		// line), so fall back to stable parse order for deterministic output.
		return entries[i].src.ordinal < entries[j].src.ordinal
	})

	for _, entry := range entries {
		td := entry.td
		fs := td.Facets

		component := td.Name.Local
		if component == "" || entry.src.isLocal {
			component = componentLocalSimpleType
		}
		line := entry.src.line

		// A restriction must only carry facets APPLICABLE to its variety/primitive
		// (XSD §4.1.5 — the {applicable facets} set). A range facet on a list/union
		// or on a non-ordered atomic primitive, or a digit facet outside the
		// xs:decimal family, is meaningless: its comparison is a no-op at validation
		// time, which would silently drop the constraint and let any instance
		// through. checkFacetApplicability reports those inapplicable facets AND, for
		// an inapplicable RANGE facet, returns false so the value-against-base bound
		// check is skipped — that check resolves a bound against the base value space
		// and would otherwise mis-accept a bound that merely validates as some member
		// of an unordered/list base instead of rejecting it outright.
		if c.checkFacetApplicability(ctx, td, fs, line) {
			c.checkFacetValueAgainstBase(ctx, td, fs, line, component)
		}
		c.checkEnumValueAgainstBase(ctx, td, fs, line, component)
		c.checkFacetMutualExclusion(ctx, fs, line, component)
		c.checkFacetSameTypeConsistency(ctx, td, fs, line, component)
		c.checkFacetBaseRestriction(ctx, td, fs, line, component)
	}
}

// facetVarietyComponent returns the component label used in a "facet not allowed"
// error for a list- or union-variety simple type, matching libxml2's phrasing:
// "union type 'name'" / "list type 'name'" for a named type and
// "local union type" / "local list type" for an anonymous one. Only the LOCAL
// name is used (libxml2 does not qualify it with the target namespace here).
func facetVarietyComponent(td *TypeDef, variety TypeVariety) string {
	kind := "union"
	if variety == TypeVarietyList {
		kind = "list"
	}
	if td.Name.Local == "" {
		return "local " + kind + " type"
	}
	return kind + " type '" + td.Name.Local + "'"
}

// stringDerivedTypes is the set of builtin base locals whose primitive ancestor
// is xs:string. anyURI is deliberately EXCLUDED: it is its own XSD primitive, so
// a "facet not allowed" message on an anyURI-derived type names xs:anyURI, not
// xs:string — matching libxml2.
var stringDerivedTypes = map[string]struct{}{
	lexicon.TypeNormalizedString: {}, "token": {}, "language": {},
	"Name": {}, "NCName": {}, "ID": {}, "IDREF": {}, "IDREFS": {},
	"ENTITY": {}, "ENTITIES": {}, "NMTOKEN": {}, "NMTOKENS": {},
}

// lengthApplicableTypes is the set of builtin base locals on whose atomic value
// space the length facets (length, minLength, maxLength) are applicable. Per XSD
// §3.16 length measures the number of characters (string family), octets (the
// binary types) or characters of the lexical/canonical form for anyURI, QName and
// NOTATION. It is INAPPLICABLE to the decimal/numeric family, boolean, float,
// double and the date/time/duration family — libxml2 rejects a length facet
// there. The string-derived family is enumerated explicitly (it mirrors
// stringDerivedTypes, plus "string" itself) so the gate does not depend on the
// primitive-collapsing in atomicPrimitiveLocal.
var lengthApplicableTypes = map[string]struct{}{
	// String and its derivations.
	lexicon.TypeString: {}, lexicon.TypeNormalizedString: {}, lexicon.TypeToken: {}, "language": {},
	"Name": {}, "NCName": {}, "ID": {}, lexicon.TypeIDREF: {}, "IDREFS": {},
	"ENTITY": {}, "ENTITIES": {}, "NMTOKEN": {}, "NMTOKENS": {},
	// anyURI, QName, NOTATION (own primitives) and the binary types.
	lexicon.TypeAnyURI: {}, lexicon.TypeQName: {}, lexicon.TypeNotation: {},
	"hexBinary": {}, "base64Binary": {},
}

// atomicPrimitiveLocal returns the local name of the XSD PRIMITIVE built-in that
// an atomic type's builtin base reduces to, used to name the offending primitive
// in a "facet not allowed on types derived from …" message (e.g. xs:token →
// "string"). xs:decimal and its integer derivations collapse to "decimal"; the
// xs:string-derived family collapses to "string"; anyURI and every other
// primitive (boolean, float, double, the date/time family, the binary types,
// QName, NOTATION) are their own primitive and pass through unchanged.
func atomicPrimitiveLocal(builtinLocal string) string {
	if value.IsDecimalFamily(builtinLocal) {
		return lexicon.TypeDecimal
	}
	if _, ok := stringDerivedTypes[builtinLocal]; ok {
		return lexicon.TypeString
	}
	return builtinLocal
}

// checkFacetApplicability rejects facets that are not applicable to a simple
// type's variety/primitive and reports whether the value-against-base bound check
// should still run for this type.
//
// Per XSD §4.1.5 the {applicable facets} of a simple type depend on its variety,
// and — for atomic types — on the {ordered}/numeric nature of its primitive:
//
//   - list: length, minLength, maxLength, enumeration, pattern, whiteSpace.
//   - union: enumeration, pattern.
//   - atomic: the range facets (min/maxInclusive, min/maxExclusive) apply ONLY to
//     an ordered primitive (numeric or the date/time/duration family); the digit
//     facets (totalDigits, fractionDigits) apply ONLY to the xs:decimal family.
//
// Any facet outside its variety's/primitive's applicable set — a range or digit
// facet on a list/union, the length family or whiteSpace on a union, a range
// facet on a non-ordered atomic primitive (string, boolean, anyURI, hexBinary,
// base64Binary, QName, NOTATION), or a digit facet on a non-decimal atomic
// primitive (float, double, date/time, …) — is an error, reported exactly as
// libxml2 does. The disallowed facets are emitted in a fixed canonical order so
// the output is deterministic.
//
// It returns true only when EVERY range facet present is applicable (an ordered
// atomic primitive), telling the caller to run checkFacetValueAgainstBase — which
// resolves each bound against the base value space and is meaningful only there.
// For a list/union, or an atomic whose range facet is inapplicable, it returns
// FALSE so the caller SKIPS that bound check: on a list/union or non-ordered base
// the bound comparison is a no-op at validation time, so the leftover
// value-against-base check would mis-accept a bound that merely validates as some
// member (e.g. minInclusive='abc' on union(xs:int xs:string)) instead of being
// rejected outright as it is here.
func (c *compiler) checkFacetApplicability(ctx context.Context, td *TypeDef, fs *FacetSet, line int) bool {
	variety := resolveVariety(td)
	if variety == TypeVarietyList || variety == TypeVarietyUnion {
		return c.checkListUnionFacetApplicability(ctx, td, fs, line, variety)
	}
	return c.checkAtomicFacetApplicability(ctx, td, fs, line)
}

// checkListUnionFacetApplicability rejects facets inapplicable to a list- or
// union-variety type. It always returns false so the caller skips the
// value-against-base bound check (see checkFacetApplicability).
func (c *compiler) checkListUnionFacetApplicability(ctx context.Context, td *TypeDef, fs *FacetSet, line int, variety TypeVariety) bool {
	component := facetVarietyComponent(td, variety)

	// disallowed lists the facets inapplicable to this variety, in a fixed
	// canonical order. Range and digit facets are inapplicable to BOTH list and
	// union; the length family and whiteSpace are inapplicable to a union but
	// allowed on a list.
	type facetPresence struct {
		name    string
		present bool
	}
	disallowed := []facetPresence{
		{facetMinInclusive, fs.MinInclusive != nil},
		{facetMaxInclusive, fs.MaxInclusive != nil},
		{facetMinExclusive, fs.MinExclusive != nil},
		{facetMaxExclusive, fs.MaxExclusive != nil},
		{"totalDigits", fs.TotalDigits != nil},
		{"fractionDigits", fs.FractionDigits != nil},
		{"explicitTimezone", fs.ExplicitTimezone != nil},
	}
	if variety == TypeVarietyUnion {
		disallowed = append(disallowed,
			facetPresence{"length", fs.Length != nil},
			facetPresence{"minLength", fs.MinLength != nil},
			facetPresence{"maxLength", fs.MaxLength != nil},
			facetPresence{"whiteSpace", fs.WhiteSpace != nil},
		)
	}

	for _, fp := range disallowed {
		if !fp.present {
			continue
		}
		msg := fmt.Sprintf("The facet '%s' is not allowed.", fp.name)
		c.schemaError(ctx, schemaComponentError(c.filename, line, "simpleType", component, msg))
	}

	return false
}

// checkAtomicFacetApplicability rejects range facets on a non-ordered atomic
// primitive, digit facets on a non-decimal atomic primitive, and length-family
// facets (length, minLength, maxLength) on a primitive outside the
// length-applicable set (string-derived, the binary types, anyURI, QName,
// NOTATION). It returns false (so the caller skips the value-against-base bound
// check) only when an inapplicable RANGE facet was reported — on a non-ordered
// base that bound check is a no-op and must not run. Otherwise it returns true.
func (c *compiler) checkAtomicFacetApplicability(ctx context.Context, td *TypeDef, fs *FacetSet, line int) bool {
	builtinLocal := builtinBaseLocal(td)
	if builtinLocal == "" {
		// No resolvable builtin primitive (e.g. a type whose base chain has not
		// resolved). Leave the bound check to run; nothing to reject here.
		return true
	}

	component := "local atomic type"
	if td.Name.Local != "" {
		component = "atomic type '" + td.Name.Local + "'"
	}
	primitive := "xs:" + atomicPrimitiveLocal(builtinLocal)

	ordered := value.Orderable(builtinLocal)
	decimal := value.IsDecimalFamily(builtinLocal)
	_, lengthOK := lengthApplicableTypes[builtinLocal]
	explicitTimezoneOK := explicitTimezoneApplicable(builtinLocal)

	report := func(facet string) {
		msg := fmt.Sprintf("The facet '%s' is not allowed on types derived from the type %s.", facet, primitive)
		c.schemaError(ctx, schemaComponentError(c.filename, line, "simpleType", component, msg))
	}

	rangeRejected := false
	// Range facets, in canonical order. Inapplicable unless the primitive is
	// ordered; when rejected, the bound check must be skipped.
	if !ordered {
		for _, rf := range []struct {
			name    string
			present bool
		}{
			{facetMinInclusive, fs.MinInclusive != nil},
			{facetMaxInclusive, fs.MaxInclusive != nil},
			{facetMinExclusive, fs.MinExclusive != nil},
			{facetMaxExclusive, fs.MaxExclusive != nil},
		} {
			if !rf.present {
				continue
			}
			report(rf.name)
			rangeRejected = true
		}
	}

	// Digit facets are applicable only to the xs:decimal family. Their
	// rejection does not affect the range-bound check, so it never flips the
	// returned verdict.
	if !decimal {
		if fs.TotalDigits != nil {
			report("totalDigits")
		}
		if fs.FractionDigits != nil {
			report("fractionDigits")
		}
	}

	// Length facets are applicable only to the string-derived family, the binary
	// types, anyURI, QName and NOTATION (lengthApplicableTypes). On a numeric,
	// boolean, float/double or date/time/duration atomic they are meaningless, so
	// libxml2 rejects them. Their rejection is independent of the range-bound
	// check, so it never flips the returned verdict.
	if !lengthOK {
		if fs.Length != nil {
			report("length")
		}
		if fs.MinLength != nil {
			report("minLength")
		}
		if fs.MaxLength != nil {
			report("maxLength")
		}
	}
	if fs.ExplicitTimezone != nil && !explicitTimezoneOK {
		report("explicitTimezone")
	}

	return !rangeRejected
}

// checkFacetValueAgainstBase validates that each value-bearing range facet
// (min/maxInclusive, min/maxExclusive) is itself a valid instance of the type
// being restricted. Per XSD §3.16, a facet value must belong to the base type's
// value space; an invalid bound (e.g. <xs:minInclusive value="abc"/> on an
// xs:int base, or a value that overruns the base's value space) makes the schema
// in error. Without this check the bad bound silently fell through
// compareForRangeFacet's "can't compare" path at validation time, turning the
// constraint into a no-op and letting any instance value through.
func (c *compiler) checkFacetValueAgainstBase(ctx context.Context, td *TypeDef, fs *FacetSet, line int, component string) {
	base := td.BaseType
	if base == nil {
		return
	}
	builtinLocal := builtinBaseLocal(td)

	type rangeFacet struct {
		name              string
		value             *string
		ns                map[string]string
		sameExclusiveBase *string
	}
	for _, rf := range []rangeFacet{
		{facetMinInclusive, fs.MinInclusive, fs.MinInclusiveNS, nil},
		{facetMaxInclusive, fs.MaxInclusive, fs.MaxInclusiveNS, nil},
		{facetMinExclusive, fs.MinExclusive, fs.MinExclusiveNS, effectiveInheritedExclusiveRangeFacet(td, facetMinExclusive, builtinLocal)},
		{facetMaxExclusive, fs.MaxExclusive, fs.MaxExclusiveNS, effectiveInheritedExclusiveRangeFacet(td, facetMaxExclusive, builtinLocal)},
	} {
		if rf.value == nil {
			continue
		}
		if rf.sameExclusiveBase != nil && rangeFacetValueEqual(*rf.value, *rf.sameExclusiveBase, builtinLocal) {
			continue
		}
		// Validate the bound against the base type's value space with errors
		// suppressed; only the pass/fail verdict matters here. A non-nil result
		// means the bound is not a valid instance of the base type, so the
		// restriction is in error. Each bound is resolved with ITS OWN captured
		// namespace context so a prefixed bound (e.g. a QName-typed q:z) binds the
		// prefix declared at its own facet element, not a sibling's.
		sub := &validationContext{schema: c.schema, errorHandler: helium.NilErrorHandler{}, suppressDepth: 1, version: c.version}
		if validateValue(ctx, *rf.value, rf.ns, base, "", "", 0, sub) == nil {
			continue
		}
		msg := fmt.Sprintf("The value '%s' of the facet '%s' is not a valid value of the base type '%s'.",
			*rf.value, rf.name, typeDisplayName(base))
		c.schemaError(ctx, schemaComponentError(c.filename, line, "simpleType", component, msg))
	}
}

// checkEnumValueAgainstBase validates that each enumeration facet value is itself
// a valid instance of the base type's value space. Per XSD §3.16, an enumeration
// {value} must be datatype-valid against the {base type definition}; an invalid
// member (e.g. <xs:enumeration value="+NaN"/> on an xs:float base — signed NaN is
// not in the float/double lexical space) makes the schema in error and must be
// rejected at COMPILE time rather than silently compiling into an unsatisfiable
// enumeration that fails at instance-validation time.
//
// This applies to ALL varieties — atomic, list, and union. validateValue is
// variety-aware: an atomic literal is validated against the builtin base's value
// space, a list literal item-by-item against the item type (so a list
// itemType="xs:float" rejects a "+NaN" enumeration member), and a union literal
// against whichever member type accepts it (so a union with an xs:float member
// rejects "+NaN" when no member admits it).
//
// Suppression is PER LITERAL, not per type: only a literal that
// enumLiteralHasUnboundQName flags — a QName/NOTATION carrier, at any nesting
// depth within the variety structure, whose prefix is unbound — is skipped here,
// because checkEnumQNameAndNotation already reports that exact case with
// libxml2-matching phrasing; validating it here too would produce a duplicate /
// differently-phrased diagnostic. Every OTHER enumeration literal of a
// QName/NOTATION-carrying type is still validated against the base value space,
// so a QName base restricted with (e.g.) xs:length value="2" still rejects an
// out-of-space "abc" enumeration member.
func (c *compiler) checkEnumValueAgainstBase(ctx context.Context, td *TypeDef, fs *FacetSet, line int, component string) {
	base := td.BaseType
	if base == nil || len(fs.Enumeration) == 0 {
		return
	}
	variety := resolveVariety(td)

	for i, ev := range fs.Enumeration {
		var enumNS map[string]string
		if i < len(fs.EnumerationNS) {
			enumNS = fs.EnumerationNS[i]
		}
		// A QName/NOTATION literal whose prefix is unbound (at any nesting depth
		// within the variety structure) is already reported by
		// checkEnumQNameAndNotation; suppress the report HERE for just that literal
		// to avoid a duplicate / differently-phrased diagnostic. This is per-literal,
		// not a blanket skip of the whole type: a QName/NOTATION base still has its
		// other enumeration literals validated against the base value space, so e.g.
		// a QName base restricted with xs:length value="2" rejects an out-of-space
		// "abc" enumeration member rather than silently compiling it.
		if c.enumLiteralHasUnboundQName(ctx, ev, enumNS, td, variety) {
			continue
		}
		// Validate the member against the base type's value space with errors
		// suppressed; only the pass/fail verdict matters. A non-nil result means the
		// member is not a valid instance of the base type, so the enumeration facet
		// is in error.
		sub := &validationContext{schema: c.schema, errorHandler: helium.NilErrorHandler{}, suppressDepth: 1, version: c.version}
		if validateValue(ctx, ev, enumNS, base, "", "", 0, sub) == nil {
			continue
		}
		msg := fmt.Sprintf("The value '%s' of the facet 'enumeration' is not a valid value of the base type '%s'.",
			ev, typeDisplayName(base))
		c.schemaError(ctx, schemaComponentError(c.filename, line, "simpleType", component, msg))
	}
}

// checkEnumQNameAndNotation runs post-resolve checks on QName/NOTATION-based
// atomic simple types:
//
//   - Each enumeration facet literal of a QName/NOTATION-restricted type is
//     resolved against the literal's captured in-scope namespaces. An unresolved
//     prefix makes the literal an invalid QName/NOTATION and is reported as a
//     schema error, rather than silently compiling into an unsatisfiable
//     enumeration.
//   - A simpleType whose base is (directly) xs:NOTATION with no enumeration facet
//     is rejected: per XSD, xs:NOTATION may only be used as a base for a
//     restriction that supplies an enumeration facet enumerating the permitted
//     notation names. Full xs:NOTATION declaration-table semantics (checking each
//     enumerated name against a declared <xs:notation>) is deferred (see memory).
func (c *compiler) checkEnumQNameAndNotation(ctx context.Context) {
	if c.filename == "" {
		return
	}

	type entry struct {
		td  *TypeDef
		src typeDefSource
	}
	var entries []entry
	for td, src := range c.typeDefSources {
		entries = append(entries, entry{td: td, src: src})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].src.line != entries[j].src.line {
			return entries[i].src.line < entries[j].src.line
		}
		if entries[i].td.Name.Local != entries[j].td.Name.Local {
			return entries[i].td.Name.Local < entries[j].td.Name.Local
		}
		return entries[i].src.ordinal < entries[j].src.ordinal
	})

	for _, e := range entries {
		td := e.td
		if td.Name.NS == lexicon.NamespaceXSD {
			continue
		}

		component := td.Name.Local
		if component == "" || e.src.isLocal {
			component = componentLocalSimpleType
		}

		variety := resolveVariety(td)

		// An un-enumerated xs:NOTATION use is not a permitted derivation. This is
		// checked recursively over the type's variety so a NOTATION carrier hidden
		// inside a list item type or a union member is caught too, not only a
		// direct atomic xs:NOTATION restriction base. A NOTATION carrier is allowed
		// only when it is itself enumeration-derived.
		if notationUsedWithoutEnumeration(td) {
			c.schemaError(ctx, schemaComponentError(c.filename, e.src.line, "simpleType", component,
				"It is an error if the base type is the built-in 'NOTATION' and there is no 'enumeration' facet."))
		}

		// Validate each enumeration literal's QName/NOTATION prefix binding,
		// variety-aware against the restriction base: an atomic QName/NOTATION
		// literal is validated directly, a list literal item-by-item against the
		// item type, and a union literal against whichever member type accepts it.
		if td.Facets == nil {
			continue
		}
		for i, ev := range td.Facets.Enumeration {
			var enumNS map[string]string
			if i < len(td.Facets.EnumerationNS) {
				enumNS = td.Facets.EnumerationNS[i]
			}
			if c.enumLiteralHasUnboundQName(ctx, ev, enumNS, td, variety) {
				msg := fmt.Sprintf("The value '%s' is not a valid value of the atomic type '%s'.", ev, typeDisplayName(td))
				c.schemaError(ctx, schemaComponentError(c.filename, e.src.line, "simpleType", component, msg))
				continue
			}
			// XSD 1.1: an xs:NOTATION restriction's enumeration values must name a
			// notation declared in the schema. (XSD 1.0 keeps the historical
			// behavior — declaration-table matching deferred — to stay byte-identical.)
			if c.version == Version11 && variety == TypeVarietyAtomic && builtinBaseLocal(td) == lexicon.TypeNotation {
				qn, qerr := resolveLexicalQName(normalizeWhiteSpace(ev, resolveWhiteSpace(td)), enumNS)
				if qerr != nil {
					continue
				}
				if _, ok := c.notations[qn]; !ok {
					msg := fmt.Sprintf("The enumeration value '%s' does not match a declared notation.", ev)
					c.schemaError(ctx, schemaComponentError(c.filename, e.src.line, "simpleType", component, msg))
				}
			}
		}
	}
}

// checkCircularSimpleTypes reports a schema error for any simple type that
// participates in a circular definition: a union whose memberTypes reference it
// (transitively), a list whose itemType reaches it, or a restriction whose base
// chain returns to it (XSD §3.16.6.3 / cos-no-circular-unions and the general
// "no circular type definitions" rule). Such a schema is invalid, and several
// variety-walking compile checks (and resolveVariety/resolveItemType base
// walks) would otherwise recurse forever on it; reporting it here surfaces the
// real error before those walks run. It returns true when at least one circular
// type was found, so the caller can stop before the (not fully cycle-guarded)
// downstream checks walk the broken type graph.
func (c *compiler) checkCircularSimpleTypes(ctx context.Context) bool {
	if c.filename == "" {
		return false
	}

	type entry struct {
		td  *TypeDef
		src typeDefSource
	}
	var entries []entry
	for td, src := range c.typeDefSources {
		if td.Name.NS == lexicon.NamespaceXSD {
			continue
		}
		entries = append(entries, entry{td: td, src: src})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].src.line != entries[j].src.line {
			return entries[i].src.line < entries[j].src.line
		}
		if entries[i].td.Name.Local != entries[j].td.Name.Local {
			return entries[i].td.Name.Local < entries[j].td.Name.Local
		}
		return entries[i].src.ordinal < entries[j].src.ordinal
	})

	found := false
	for _, e := range entries {
		if !simpleTypeReachesSelf(e.td) {
			continue
		}
		found = true
		component := e.td.Name.Local
		if component == "" || e.src.isLocal {
			component = componentLocalSimpleType
		}
		c.schemaError(ctx, schemaComponentError(c.filename, e.src.line, "simpleType", component,
			"Circular definition of the simple type; a type must not be a member, item, or base type of itself."))
	}
	return found
}

// simpleTypeReachesSelf reports whether start is reachable from itself by
// following simple-type definition edges — union member types, the list item
// type, and a non-builtin restriction base type. The visited set bounds the
// walk so it terminates on the cycle instead of recursing forever.
func simpleTypeReachesSelf(start *TypeDef) bool {
	visited := map[*TypeDef]struct{}{}
	var walk func(td *TypeDef) bool
	walk = func(td *TypeDef) bool {
		for _, n := range simpleTypeNeighbors(td) {
			if n == start {
				return true
			}
			if _, seen := visited[n]; seen {
				continue
			}
			visited[n] = struct{}{}
			if walk(n) {
				return true
			}
		}
		return false
	}
	return walk(start)
}

// simpleTypeNeighbors returns the non-builtin simple types that td directly
// depends on: its union member types, its list item type, and its restriction
// base type (only when that base is a user-defined, non-builtin type). Builtin
// XSD types are leaves and are never returned, so the walk stays within the
// user-declared simple-type graph.
func simpleTypeNeighbors(td *TypeDef) []*TypeDef {
	if td == nil {
		return nil
	}
	var out []*TypeDef
	for _, m := range td.MemberTypes {
		if m != nil && m.Name.NS != lexicon.NamespaceXSD {
			out = append(out, m)
		}
	}
	if td.ItemType != nil && td.ItemType.Name.NS != lexicon.NamespaceXSD {
		out = append(out, td.ItemType)
	}
	if td.BaseType != nil && td.BaseType.Name.NS != lexicon.NamespaceXSD {
		out = append(out, td.BaseType)
	}
	return out
}

// enumLiteralHasUnboundQName reports whether the enumeration literal ev has a
// QName/NOTATION component whose prefix is not bound in enumNS, dispatched on the
// restriction base's effective variety:
//
//   - atomic: the whole literal is a QName/NOTATION value (checked only when the
//     base resolves to builtin xs:QName/xs:NOTATION).
//   - list: each whitespace-separated item is validated against the item type.
//   - union: the literal must validate against some member type under enumNS; if
//     the only members that could carry it are QName/NOTATION and its prefix is
//     unbound, the literal is unsatisfiable and is flagged.
func (c *compiler) enumLiteralHasUnboundQName(ctx context.Context, ev string, enumNS map[string]string, td *TypeDef, variety TypeVariety) bool {
	switch variety {
	case TypeVarietyList:
		itemType := resolveItemType(td)
		if itemType == nil {
			return false
		}
		itemVariety := resolveVariety(itemType)
		for _, item := range value.XSDFields(ev) {
			if c.enumLiteralHasUnboundQName(ctx, item, enumNS, itemType, itemVariety) {
				return true
			}
		}
		return false
	case TypeVarietyUnion:
		members := resolveUnionMembers(td)
		hasQNameMember := false
		for _, member := range members {
			if member == nil {
				continue
			}
			// A QName/NOTATION carrier may sit inside a member that is itself a list
			// or a nested union, so detect it recursively rather than only on an
			// atomic QName/NOTATION member.
			if typeHasQNameNotationCarrier(member) {
				hasQNameMember = true
			}
			// The literal is satisfiable (and thus not flagged) as soon as some
			// member accepts it under the literal's own namespace bindings. A
			// QName/NOTATION carrier accepts it only with a bound prefix, so a
			// successful match means the prefix is bound.
			sub := &validationContext{schema: c.schema, errorHandler: helium.NilErrorHandler{}, suppressDepth: 1, version: c.version}
			if validateValue(ctx, ev, enumNS, member, "", "", 0, sub) == nil {
				return false
			}
		}
		// No member accepts the literal. Only treat this as a prefix-binding
		// failure when a QName/NOTATION carrier exists (otherwise the literal is
		// invalid for some other reason, not flagged by this check).
		return hasQNameMember
	default:
		if builtinBaseLocal(td) != lexicon.TypeQName && builtinBaseLocal(td) != lexicon.TypeNotation {
			return false
		}
		// The enumeration literal is a value in the constrained type's value space,
		// so apply the type's effective whiteSpace facet (QName/NOTATION collapse)
		// before resolving its prefix — otherwise a literal like " p:a " (with
		// surrounding spaces) would be reported as an invalid QName at compile time
		// even though its collapsed form "p:a" is a perfectly valid bound QName.
		_, err := resolveLexicalQName(normalizeWhiteSpace(ev, resolveWhiteSpace(td)), enumNS)
		return err != nil
	}
}

// notationUsedWithoutEnumeration reports whether td INTRODUCES an xs:NOTATION
// use that is not permitted because the NOTATION carrier is not
// enumeration-derived. Per XSD, xs:NOTATION may only appear in a derivation that
// supplies an enumeration of the permitted notation names. The check is keyed on
// the carrier declared DIRECTLY by td (so each type in a derivation chain is
// judged once, not once per ancestor step):
//
//   - Atomic: td restricts directly from the built-in xs:NOTATION and supplies no
//     enumeration facet.
//   - List: td declares an itemType whose item type is (recursively) a
//     NOTATION carrier that is not itself enumeration-derived.
//   - Union: td declares memberTypes and some member is (recursively) a NOTATION
//     carrier that is not itself enumeration-derived.
//
// A NOTATION carrier nested inside a list/union is permitted only when that
// item/member type is enumeration-derived (hasEffectiveEnumeration over its own
// chain), so an xs:list itemType="<enumerated NOTATION type>" compiles cleanly.
func notationUsedWithoutEnumeration(td *TypeDef) bool {
	if td == nil {
		return false
	}

	// Atomic: only the type that directly restricts xs:NOTATION is judged, exactly
	// as the original direct-base check did.
	if td.Derivation == DerivationRestriction && td.BaseType != nil &&
		td.BaseType.Name.NS == lexicon.NamespaceXSD && td.BaseType.Name.Local == lexicon.TypeNotation {
		return td.Facets == nil || len(td.Facets.Enumeration) == 0
	}

	// List: judged at the type that declares the itemType.
	if td.ItemType != nil {
		return notationCarrierNotEnumerated(td.ItemType)
	}

	// Union: judged at the type that declares the memberTypes.
	if len(td.MemberTypes) > 0 {
		return slices.ContainsFunc(td.MemberTypes, notationCarrierNotEnumerated)
	}

	return false
}

// notationCarrierNotEnumerated reports whether td is (recursively, through list
// item types and union members) a NOTATION carrier that is not enumeration-
// derived. A bare atomic xs:NOTATION carrier is permitted only when its own
// derivation chain supplies an enumeration; a list/union recurses into its
// item/member types.
func notationCarrierNotEnumerated(td *TypeDef) bool {
	return notationCarrierNotEnumeratedVisit(td, map[*TypeDef]struct{}{})
}

// notationCarrierNotEnumeratedVisit is the cycle-guarded recursion behind
// notationCarrierNotEnumerated. A cyclic / self-referential union member (a union
// whose memberTypes include itself, directly or transitively) would otherwise
// recurse forever; the visited set terminates the walk on a repeated type. A
// genuinely circular member type is an invalid schema reported by the regular
// compilation checks, so stopping here (treating the cyclic node as not a
// NOTATION carrier) lets that real error surface instead of crashing.
func notationCarrierNotEnumeratedVisit(td *TypeDef, visited map[*TypeDef]struct{}) bool {
	if td == nil {
		return false
	}
	if _, seen := visited[td]; seen {
		return false
	}
	visited[td] = struct{}{}
	switch resolveVariety(td) {
	case TypeVarietyList:
		return notationCarrierNotEnumeratedVisit(resolveItemType(td), visited)
	case TypeVarietyUnion:
		return slices.ContainsFunc(resolveUnionMembers(td), func(m *TypeDef) bool {
			return notationCarrierNotEnumeratedVisit(m, visited)
		})
	default:
		if builtinBaseLocal(td) != lexicon.TypeNotation {
			return false
		}
		return !hasEffectiveEnumeration(td)
	}
}

// typeHasQNameNotationCarrier reports whether td denotes — anywhere in its
// variety structure — a value of type xs:QName or xs:NOTATION. It walks
// recursively through list item types and nested union members, so a member like
// xs:list itemType="xs:QName" or a union nesting an xs:NOTATION member is
// recognized as carrying a QName/NOTATION value. Used by the enumeration-literal
// prefix-binding check (and the NOTATION-use check) so QName/NOTATION carriers
// hidden inside list/union members are not missed.
func typeHasQNameNotationCarrier(td *TypeDef) bool {
	return typeHasQNameNotationCarrierVisit(td, map[*TypeDef]struct{}{})
}

// typeHasQNameNotationCarrierVisit is the cycle-guarded recursion behind
// typeHasQNameNotationCarrier. As with notationCarrierNotEnumeratedVisit, a
// cyclic / self-referential union member would recurse forever; the visited set
// terminates the walk on a repeated type and reports no carrier for the cyclic
// node, leaving the real circular-type schema error to the regular checks.
func typeHasQNameNotationCarrierVisit(td *TypeDef, visited map[*TypeDef]struct{}) bool {
	if td == nil {
		return false
	}
	if _, seen := visited[td]; seen {
		return false
	}
	visited[td] = struct{}{}
	switch resolveVariety(td) {
	case TypeVarietyList:
		return typeHasQNameNotationCarrierVisit(resolveItemType(td), visited)
	case TypeVarietyUnion:
		return slices.ContainsFunc(resolveUnionMembers(td), func(m *TypeDef) bool {
			return typeHasQNameNotationCarrierVisit(m, visited)
		})
	default:
		bl := builtinBaseLocal(td)
		return bl == lexicon.TypeQName || bl == lexicon.TypeNotation
	}
}

// hasEffectiveEnumeration reports whether td or any of its base types along the
// restriction chain carries an enumeration facet.
func hasEffectiveEnumeration(td *TypeDef) bool {
	for cur := range baseChain(td) {
		if cur.Facets != nil && len(cur.Facets.Enumeration) > 0 {
			return true
		}
	}
	return false
}

// checkNotationOnDeclarations rejects an element or attribute declaration whose
// effective type is the built-in xs:NOTATION (or NOTATION-derived) without an
// effective enumeration facet. Per XSD, xs:NOTATION may only be used in a
// derivation that supplies an enumeration of the permitted notation names; a
// declaration that types content directly as xs:NOTATION (e.g.
// type="xs:NOTATION") bypasses the simpleType-level restriction rule, so it is
// caught here after all type references are resolved. Full xs:NOTATION
// declaration-table semantics are deferred (see memory).
func (c *compiler) checkNotationOnDeclarations(ctx context.Context) {
	if c.filename == "" {
		return
	}

	// Elements: every element decl carrying a type= ref is tracked in
	// elemRefSources, which is exactly the type="xs:NOTATION" case.
	type elemEntry struct {
		decl *ElementDecl
		src  elemRefSource
	}
	var elemEntries []elemEntry
	for decl, src := range c.elemRefSources {
		elemEntries = append(elemEntries, elemEntry{decl: decl, src: src})
	}
	sort.Slice(elemEntries, func(i, j int) bool {
		if elemEntries[i].src.line != elemEntries[j].src.line {
			return elemEntries[i].src.line < elemEntries[j].src.line
		}
		return elemEntries[i].decl.Name.Local < elemEntries[j].decl.Name.Local
	})
	for _, e := range elemEntries {
		td := e.decl.Type
		if td == nil {
			continue
		}
		// Only the direct atomic type="xs:NOTATION" case is judged here; a list/union
		// (named or inline anonymous) whose item/member type is an un-enumerated
		// NOTATION carrier is already caught at the simpleType level by
		// checkEnumQNameAndNotation, so judging it again here would double-report.
		if builtinBaseLocal(td) != lexicon.TypeNotation {
			continue
		}
		if hasEffectiveEnumeration(td) {
			continue
		}
		c.schemaError(ctx, schemaParserError(c.filename, e.src.line, e.src.elemName, elemElement,
			"It is an error if the type definition is the built-in 'NOTATION' and there is no 'enumeration' facet."))
	}

	// Attributes: every attribute use is tracked in attrUseSources.
	type attrEntry struct {
		au  *AttrUse
		src attrConstraintSource
	}
	var attrEntries []attrEntry
	for au, src := range c.attrUseSources {
		attrEntries = append(attrEntries, attrEntry{au: au, src: src})
	}
	sort.Slice(attrEntries, func(i, j int) bool {
		if attrEntries[i].src.line != attrEntries[j].src.line {
			return attrEntries[i].src.line < attrEntries[j].src.line
		}
		return attrEntries[i].src.local < attrEntries[j].src.local
	})
	for _, a := range attrEntries {
		td := attrUseTypeDef(a.au, c.schema)
		if td == nil {
			continue
		}
		// Only the direct atomic type="xs:NOTATION" case is judged here; a list/union
		// (named or inline anonymous) whose item/member type is an un-enumerated
		// NOTATION carrier is already caught at the simpleType level by
		// checkEnumQNameAndNotation, so judging it again here would double-report.
		if builtinBaseLocal(td) != lexicon.TypeNotation {
			continue
		}
		if hasEffectiveEnumeration(td) {
			continue
		}
		c.schemaError(ctx, schemaParserError(c.filename, a.src.line, a.src.local, elemAttribute,
			"It is an error if the type definition is the built-in 'NOTATION' and there is no 'enumeration' facet."))
	}
}

// isAnyAtomicTypeDef reports whether td is the built-in xs:anyAtomicType.
func isAnyAtomicTypeDef(td *TypeDef) bool {
	return td != nil && td.Name.NS == lexicon.NamespaceXSD && td.Name.Local == lexicon.TypeAnyAtomicType
}

// checkAnyAtomicTypeUsage (XSD 1.1) rejects a user-defined simple type that uses
// xs:anyAtomicType as its restriction base, list item type, or union member type.
// xs:anyAtomicType is the abstract base of every atomic type and (per the
// resolution of W3C bug 11103) must not be named in a user derivation. It walks
// every parsed simple type, including inline anonymous ones (typeDefSources).
func (c *compiler) checkAnyAtomicTypeUsage(ctx context.Context) {
	if c.filename == "" {
		return
	}

	type entry struct {
		td  *TypeDef
		src typeDefSource
	}
	var entries []entry
	for td, src := range c.typeDefSources {
		if td.Name.NS == lexicon.NamespaceXSD {
			continue
		}
		entries = append(entries, entry{td: td, src: src})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].src.line != entries[j].src.line {
			return entries[i].src.line < entries[j].src.line
		}
		if entries[i].td.Name.Local != entries[j].td.Name.Local {
			return entries[i].td.Name.Local < entries[j].td.Name.Local
		}
		return entries[i].src.ordinal < entries[j].src.ordinal
	})

	for _, e := range entries {
		td := e.td
		component := td.Name.Local
		if component == "" || e.src.isLocal {
			component = componentLocalSimpleType
		}
		report := func(role string) {
			c.schemaError(ctx, schemaComponentError(c.filename, e.src.line, "simpleType", component,
				"The "+role+" must not be the built-in 'anyAtomicType'."))
		}
		switch {
		case td.Derivation == DerivationRestriction && isAnyAtomicTypeDef(td.BaseType):
			report("base type")
		case isAnyAtomicTypeDef(td.ItemType):
			report("item type")
		case slices.ContainsFunc(td.MemberTypes, isAnyAtomicTypeDef):
			report("member type")
		}
	}
}

// isAnySimpleTypeDef reports whether td is the built-in xs:anySimpleType.
func isAnySimpleTypeDef(td *TypeDef) bool {
	return td != nil && td.Name.NS == lexicon.NamespaceXSD && td.Name.Local == lexicon.TypeAnySimpleType
}

// checkSimpleTypeResolution enforces the type-resolution kind rules for simple
// type definitions (XSD Structures §3.14.6 / Part 2 §2.4), version-INDEPENDENT so
// it runs in BOTH XSD 1.0 and 1.1:
//
//   - a restriction's {base type definition} must be a SIMPLE type definition —
//     naming a complexType (including the ur-type xs:anyType) is a schema error
//     (cos-st-restricts.1 / st-props-correct);
//   - a list's {item type definition} must be a simple type whose variety is
//     atomic or union — naming a complexType, or another LIST type (a list of
//     lists is forbidden), is a schema error (cos-list-of-atomic);
//   - each union {member type definition} must be a simple type — naming a
//     complexType is a schema error.
//
// It walks every parsed simple type, including inline anonymous ones (recorded in
// typeDefSources); the built-in datatypes are registered separately and never
// appear there, so they are not re-examined. Complex type definitions are skipped
// entirely (td.IsComplex) — their derivation rules are enforced elsewhere. The
// base/item/member type pointers are already resolved by resolveRefs, and cyclic
// simple types have been rejected before this runs, so the resolveVariety base
// walk terminates.
func (c *compiler) checkSimpleTypeResolution(ctx context.Context) {
	if c.filename == "" {
		return
	}

	type entry struct {
		td  *TypeDef
		src typeDefSource
	}
	var entries []entry
	for td, src := range c.typeDefSources {
		if td.IsComplex || td.syntheticComplexBase || td.Name.NS == lexicon.NamespaceXSD {
			continue
		}
		entries = append(entries, entry{td: td, src: src})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].src.line != entries[j].src.line {
			return entries[i].src.line < entries[j].src.line
		}
		if entries[i].td.Name.Local != entries[j].td.Name.Local {
			return entries[i].td.Name.Local < entries[j].td.Name.Local
		}
		return entries[i].src.ordinal < entries[j].src.ordinal
	})

	qnameOf := func(td *TypeDef) string {
		return fmt.Sprintf("{%s}%s", td.Name.NS, td.Name.Local)
	}

	for _, e := range entries {
		td := e.td
		component := td.Name.Local
		if component == "" || e.src.isLocal {
			component = componentLocalSimpleType
		}
		report := func(msg string) {
			c.schemaError(ctx, schemaComponentError(c.diagSourceOrRecorded(e.src.source), e.src.line,
				elemSimpleType, component, msg))
		}

		switch {
		case td.Derivation == DerivationRestriction && td.BaseType != nil && td.BaseType.IsComplex:
			report(fmt.Sprintf("The base type '%s' of a simpleType restriction must be a simple type definition.", qnameOf(td.BaseType)))
		case td.ItemType != nil && td.ItemType.IsComplex:
			report(fmt.Sprintf("The item type '%s' of a list must be a simple type definition.", qnameOf(td.ItemType)))
		case td.ItemType != nil && resolveVariety(td.ItemType) == TypeVarietyList:
			report(fmt.Sprintf("The item type '%s' of a list must not itself be a list type.", qnameOf(td.ItemType)))
		default:
			if i := slices.IndexFunc(td.MemberTypes, func(m *TypeDef) bool { return m != nil && m.IsComplex }); i >= 0 {
				report(fmt.Sprintf("The member type '%s' of a union must be a simple type definition.", qnameOf(td.MemberTypes[i])))
			}
		}
	}
}

// checkAnySimpleTypeUsage (XSD 1.1) rejects user derivations that restrict the
// simple ur-type xs:anySimpleType. Per the note in XML Schema Part 2 §2.4.1 (and
// the resolution of W3C bug 14559) the simple ur-type must not be named as the
// {base type definition} of a restriction, the {item type definition} of a list,
// or a {member type definition} of a union — nor may a complexType with simple
// content restrict a base whose content type is xs:anySimpleType (which would
// derive a content simple type that restricts the ur-type). It stays valid as an
// element/attribute/xsi:type type and as the base of a simpleContent EXTENSION
// (e.g. the head of a substitution group). It walks every parsed type, including
// inline anonymous ones (typeDefSources).
func (c *compiler) checkAnySimpleTypeUsage(ctx context.Context) {
	if c.filename == "" {
		return
	}

	type entry struct {
		td  *TypeDef
		src typeDefSource
	}
	var entries []entry
	for td, src := range c.typeDefSources {
		if td.Name.NS == lexicon.NamespaceXSD {
			continue
		}
		entries = append(entries, entry{td: td, src: src})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].src.line != entries[j].src.line {
			return entries[i].src.line < entries[j].src.line
		}
		if entries[i].td.Name.Local != entries[j].td.Name.Local {
			return entries[i].td.Name.Local < entries[j].td.Name.Local
		}
		return entries[i].src.ordinal < entries[j].src.ordinal
	})

	for _, e := range entries {
		td := e.td
		elemKind := e.src.elemKind
		if elemKind == "" {
			elemKind = elemSimpleType
		}
		component := td.Name.Local
		if component == "" || e.src.isLocal {
			component = componentLocalSimpleType
			if td.IsComplex {
				component = componentLocalComplexType
			}
		}
		report := func(role string) {
			c.schemaError(ctx, schemaComponentError(c.filename, e.src.line, elemKind, component,
				"The "+role+" must not be the built-in 'anySimpleType'."))
		}

		// A complexType with simple content whose restriction leaves (or produces)
		// xs:anySimpleType as the content simple type is restricting the ur-type.
		if td.IsSimpleContent {
			if td.Derivation == DerivationRestriction && isAnySimpleTypeDef(effectiveContentSimpleType(td)) {
				report("base type")
			}
			continue
		}

		switch {
		case td.Derivation == DerivationRestriction && isAnySimpleTypeDef(td.BaseType):
			report("base type")
		case isAnySimpleTypeDef(td.ItemType):
			report("item type")
		case slices.ContainsFunc(td.MemberTypes, isAnySimpleTypeDef):
			report("member type")
		}
	}
}

// checkFacetMutualExclusion checks that mutually exclusive facets are not
// both specified on the same type definition.
func (c *compiler) checkFacetMutualExclusion(ctx context.Context, fs *FacetSet, line int, component string) {
	if fs.Length != nil && (fs.MinLength != nil || fs.MaxLength != nil) {
		c.schemaError(ctx, schemaComponentError(c.filename, line, "simpleType", component,
			"It is an error for both 'length' and either of 'minLength' or 'maxLength' to be specified on the same type definition."))
	}
	if fs.MaxInclusive != nil && fs.MaxExclusive != nil {
		c.schemaError(ctx, schemaComponentError(c.filename, line, "simpleType", component,
			"It is an error for both 'maxInclusive' and 'maxExclusive' to be specified."))
	}
	if fs.MinInclusive != nil && fs.MinExclusive != nil {
		c.schemaError(ctx, schemaComponentError(c.filename, line, "simpleType", component,
			"It is an error for both 'minInclusive' and 'minExclusive' to be specified."))
	}
}

// checkFacetSameTypeConsistency checks consistency of facets within the same type.
//
// Each consistency comparison (length, digit, range) is gated to the facet
// family's APPLICABLE type/variety. When a facet family is inapplicable to the
// type, checkFacetApplicability already emits the canonical "facet not allowed"
// error; running the consistency comparison there too would add a SPURIOUS extra
// error (e.g. minLength>maxLength on an xs:int, or fractionDigits>totalDigits on
// an xs:double) that xmllint never reports. So each block runs ONLY where its
// facet family is applicable, mirroring the applicability gate.
func (c *compiler) checkFacetSameTypeConsistency(ctx context.Context, td *TypeDef, fs *FacetSet, line int, component string) {
	variety := resolveVariety(td)
	builtinLocal := builtinBaseLocal(td)
	_, lengthApplicable := lengthApplicableTypes[builtinLocal]
	decimalFamily := value.IsDecimalFamily(builtinLocal)

	// Length consistency (minLength > maxLength). Length facets are applicable to
	// a list variety (measured as item count) and to atomic primitives in the
	// length-applicable set (string-derived, the binary types, anyURI, QName,
	// NOTATION). On any other type the length facets are inapplicable and already
	// rejected as "not allowed", so this check must not run there.
	lengthFacetsApplicable := variety == TypeVarietyList || (variety == TypeVarietyAtomic && lengthApplicable)
	if lengthFacetsApplicable && fs.MinLength != nil && fs.MaxLength != nil && *fs.MinLength > *fs.MaxLength {
		c.schemaError(ctx, schemaComponentError(c.filename, line, "simpleType", component,
			"It is an error for the value of 'minLength' to be greater than the value of 'maxLength'."))
	}

	// Digit consistency (fractionDigits > totalDigits). The digit facets are
	// applicable only to the xs:decimal-family atomic types. On float/double or any
	// non-decimal primitive they are inapplicable and already rejected as "not
	// allowed", so this check must not run there (an xs:double carrying both
	// totalDigits and fractionDigits must report ONLY the two applicability errors).
	digitFacetsApplicable := variety == TypeVarietyAtomic && decimalFamily
	if digitFacetsApplicable && fs.FractionDigits != nil && fs.TotalDigits != nil && *fs.FractionDigits > *fs.TotalDigits {
		c.schemaError(ctx, schemaComponentError(c.filename, line, "simpleType", component,
			"It is an error for the value of 'fractionDigits' to be greater than the value of 'totalDigits'."))
	}

	// Range-bound ORDERING consistency (min/max{Inclusive,Exclusive}) is only
	// meaningful — and only checked — when the range facets are APPLICABLE to the
	// type: an ORDERED ATOMIC primitive (numeric, float/double, date/time/duration).
	//
	// For a list- or union-variety type, or an atomic type whose primitive is not
	// ordered, the range facets are inapplicable and already rejected as "not
	// allowed" by checkFacetApplicability. Running the ordering check there too
	// would emit a SPURIOUS extra min>max error that xmllint never reports (xmllint
	// emits only the "facet not allowed" applicability error). So we gate the whole
	// ordering block on an ordered-atomic builtin: list/union and non-ordered atomic
	// primitives skip it entirely (no comparison, no decimal fallback).
	//
	// When the gate passes, the bounds are compared in the type's ORDERED VALUE
	// SPACE, not lexically: for a non-decimal ordered atomic (date/time/duration,
	// float, double) compareDecimal would treat the bounds as incomparable and let
	// an inconsistent pair (e.g. minInclusive 2021-01-01 > maxInclusive 2020-01-01)
	// compile. cmp resolves the builtin primitive and uses the same value-space
	// comparison the instance validator uses.
	//
	// compareForRangeFacet ok=false means one of the bounds is not a valid value of
	// that type's value space (e.g. xs:int with minInclusive="1.5"). That invalid
	// bound is already reported by the bound-value validation, so we treat the pair
	// as incomparable and SKIP the ordering check (no spurious extra min>max error).
	//
	// float/double get a dedicated bound comparator first: compareForRangeFacet
	// reports NaN as incomparable (the right answer for INSTANCE ordering, where
	// NaN is unordered), but for THIS facet-consistency check xmllint orders NaN
	// as equal to NaN and above every finite/infinite bound. So minInclusive="NaN"
	// with a non-NaN maxInclusive is min>max (rejected) while min=0,max=NaN is
	// 0<NaN (accepted). value.CompareFloatFacetBound encodes that ordering and
	// still returns ok=false for an invalid float bound, leaving the invalid-bound
	// error to the dedicated bound-value check (no spurious extra ordering error).
	orderedAtomic := value.Orderable(builtinLocal)
	if variety != TypeVarietyAtomic || !orderedAtomic {
		return
	}
	cmp := func(a, b string) (int, bool) {
		if v, ok := value.CompareFloatFacetBound(a, b, builtinLocal); ok {
			return v, true
		}
		if v, ok := compareForRangeFacet(a, b, builtinLocal); ok {
			return v, true
		}
		return 0, false
	}

	if fs.MinInclusive != nil && fs.MaxInclusive != nil {
		if v, ok := cmp(*fs.MinInclusive, *fs.MaxInclusive); ok && v > 0 {
			c.schemaError(ctx, schemaComponentError(c.filename, line, "simpleType", component,
				"It is an error for the value of 'minInclusive' to be greater than the value of 'maxInclusive'."))
		}
	}
	if fs.MinExclusive != nil && fs.MaxExclusive != nil {
		if v, ok := cmp(*fs.MinExclusive, *fs.MaxExclusive); ok && v >= 0 {
			c.schemaError(ctx, schemaComponentError(c.filename, line, "simpleType", component,
				"It is an error for the value of 'minExclusive' to be greater than or equal to the value of 'maxExclusive'."))
		}
	}
	if fs.MinExclusive != nil && fs.MaxInclusive != nil {
		if v, ok := cmp(*fs.MinExclusive, *fs.MaxInclusive); ok && v >= 0 {
			c.schemaError(ctx, schemaComponentError(c.filename, line, "simpleType", component,
				"It is an error for the value of 'minExclusive' to be greater than or equal to the value of 'maxInclusive'."))
		}
	}
	if fs.MinInclusive != nil && fs.MaxExclusive != nil {
		if v, ok := cmp(*fs.MinInclusive, *fs.MaxExclusive); ok && v >= 0 {
			c.schemaError(ctx, schemaComponentError(c.filename, line, "simpleType", component,
				"It is an error for the value of 'minInclusive' to be greater than or equal to the value of 'maxExclusive'."))
		}
	}
}

// checkFacetBaseRestriction checks that facet values properly narrow (not widen)
// the base type's facets.
func (c *compiler) checkFacetBaseRestriction(ctx context.Context, td *TypeDef, fs *FacetSet, line int, component string) {
	c.checkBuiltinFixedFacetRestriction(ctx, td, fs, line, component)

	base := baseFacets(td)
	if base == nil {
		return
	}

	// Length facets.
	if fs.MinLength != nil && base.MinLength != nil && *fs.MinLength < *base.MinLength {
		c.schemaError(ctx, schemaComponentError(c.filename, line, "simpleType", component,
			fmt.Sprintf("The 'minLength' value '%d' is less than the 'minLength' value of the base type '%d'.", *fs.MinLength, *base.MinLength)))
	}
	if fs.MaxLength != nil && base.MaxLength != nil && *fs.MaxLength > *base.MaxLength {
		c.schemaError(ctx, schemaComponentError(c.filename, line, "simpleType", component,
			fmt.Sprintf("The 'maxLength' value '%d' is greater than the 'maxLength' value of the base type '%d'.", *fs.MaxLength, *base.MaxLength)))
	}
	if fs.Length != nil && base.Length != nil && *fs.Length != *base.Length {
		c.schemaError(ctx, schemaComponentError(c.filename, line, "simpleType", component,
			fmt.Sprintf("The 'length' value '%d' does not match the 'length' value of the base type '%d'.", *fs.Length, *base.Length)))
	}

	// Digit facets.
	if fs.TotalDigits != nil && base.TotalDigits != nil && *fs.TotalDigits > *base.TotalDigits {
		c.schemaError(ctx, schemaComponentError(c.filename, line, "simpleType", component,
			fmt.Sprintf("The 'totalDigits' value '%d' is greater than the 'totalDigits' value of the base type '%d'.", *fs.TotalDigits, *base.TotalDigits)))
	}
	if fs.FractionDigits != nil && base.FractionDigits != nil && *fs.FractionDigits > *base.FractionDigits {
		c.schemaError(ctx, schemaComponentError(c.filename, line, "simpleType", component,
			fmt.Sprintf("The 'fractionDigits' value '%d' is greater than the 'fractionDigits' value of the base type '%d'.", *fs.FractionDigits, *base.FractionDigits)))
	}
	c.checkFixedRangeFacetRestriction(ctx, td, fs, line, component)

	// Inclusive/exclusive boundary facets vs base. These compare a derived bound
	// against a base bound in the type's ORDERED VALUE SPACE — exactly as the
	// same-type consistency check does — so a non-decimal ordered atomic
	// (date/time/duration, float, double) is compared correctly instead of being
	// treated as incomparable by compareDecimal (which would FALSE-REJECT a valid
	// xs:date restriction such as base minInclusive=2021-01-01, derived
	// maxInclusive=2022-01-01).
	//
	// rangeCmp returns (cmp, true) when the two bounds are comparable in this
	// type's value space, and (0, false) when they are not. For a resolved ORDERED
	// builtin it uses the value-space comparator (float/double NaN handled via
	// value.CompareFloatFacetBound) and SKIPS the ordering check on an indeterminate
	// result (e.g. an invalid bound, already reported by the bound-value check) —
	// never falling back to compareDecimal for a resolved ordered type. compareDecimal
	// is used ONLY for the unresolved/no-builtin case, preserving prior behavior for
	// a base chain whose primitive has not resolved.
	builtinLocal := builtinBaseLocal(td)
	orderedAtomic := value.Orderable(builtinLocal)
	rangeCmp := func(a, b string) (int, bool) {
		if orderedAtomic {
			if v, ok := value.CompareFloatFacetBound(a, b, builtinLocal); ok {
				return v, true
			}
			return compareForRangeFacet(a, b, builtinLocal)
		}
		if builtinLocal != "" {
			// A resolved but NON-ordered builtin (string/boolean/binary/anyURI/QName/
			// NOTATION). A range facet is inapplicable there and already rejected as
			// "not allowed"; do not compare.
			return 0, false
		}
		// Unresolved primitive: fall back to the legacy decimal comparison.
		return compareDecimal(a, b), true
	}

	lower, upper := effectiveInheritedRangeBounds(td, rangeCmp)
	reportRangeBase := func(name string, value string, base inheritedRangeBound, relation string) {
		if base.immediate {
			c.schemaError(ctx, schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The '%s' value '%s' %s the '%s' value of the base type '%s'.",
					name, value, relation, base.name, *base.value)))
			return
		}
		c.schemaError(ctx, schemaComponentError(c.filename, line, "simpleType", component,
			fmt.Sprintf("The '%s' value '%s' is not a valid restriction of the effective inherited '%s' value '%s' of the base type.",
				name, value, base.name, *base.value)))
	}
	checkUpper := func(name string, value *string, inclusive bool) {
		if value == nil {
			return
		}
		if upper.value != nil {
			if v, ok := rangeCmp(*value, *upper.value); ok {
				switch {
				case v > 0:
					reportRangeBase(name, *value, upper, "is greater than")
				case v == 0 && inclusive && upper.exclusive:
					reportRangeBase(name, *value, upper, "must be less than")
				}
			}
		}
		if lower.value != nil {
			if v, ok := rangeCmp(*value, *lower.value); ok {
				switch {
				case v < 0:
					reportRangeBase(name, *value, lower, "is less than")
				case v == 0 && (lower.exclusive || !inclusive):
					reportRangeBase(name, *value, lower, "must be greater than")
				}
			}
		}
	}
	checkLower := func(name string, value *string, inclusive bool) {
		if value == nil {
			return
		}
		if lower.value != nil {
			if v, ok := rangeCmp(*value, *lower.value); ok {
				switch {
				case v < 0:
					reportRangeBase(name, *value, lower, "is less than")
				case v == 0 && inclusive && lower.exclusive:
					reportRangeBase(name, *value, lower, "must be greater than")
				}
			}
		}
		if upper.value != nil {
			if v, ok := rangeCmp(*value, *upper.value); ok {
				switch {
				case v > 0:
					reportRangeBase(name, *value, upper, "is greater than")
				case v == 0 && (!inclusive || upper.exclusive):
					reportRangeBase(name, *value, upper, "must be less than")
				}
			}
		}
	}
	checkUpper(facetMaxInclusive, fs.MaxInclusive, true)
	checkUpper(facetMaxExclusive, fs.MaxExclusive, false)
	checkLower(facetMinInclusive, fs.MinInclusive, true)
	checkLower(facetMinExclusive, fs.MinExclusive, false)
}

func (c *compiler) checkFixedRangeFacetRestriction(ctx context.Context, td *TypeDef, fs *FacetSet, line int, component string) {
	builtinLocal := builtinBaseLocal(td)
	check := func(name string, value, baseValue *string) {
		if value == nil || baseValue == nil {
			return
		}
		if rangeFacetValueEqual(*value, *baseValue, builtinLocal) {
			return
		}
		c.schemaError(ctx, schemaComponentError(c.filename, line, "simpleType", component,
			fmt.Sprintf("The value '%s' of the facet '%s' does not match the fixed value '%s' of the base type.",
				*value, name, *baseValue)))
	}
	check(facetMinInclusive, fs.MinInclusive, inheritedFixedRangeFacet(td, facetMinInclusive))
	check(facetMaxInclusive, fs.MaxInclusive, inheritedFixedRangeFacet(td, facetMaxInclusive))
	check(facetMinExclusive, fs.MinExclusive, inheritedFixedRangeFacet(td, facetMinExclusive))
	check(facetMaxExclusive, fs.MaxExclusive, inheritedFixedRangeFacet(td, facetMaxExclusive))
}

func inheritedFixedRangeFacet(td *TypeDef, name string) *string {
	if td == nil || td.BaseType == nil {
		return nil
	}
	for cur := range baseChain(td.BaseType) {
		if cur.Facets == nil {
			continue
		}
		switch name {
		case facetMinInclusive:
			if cur.Facets.MinInclusive != nil && cur.Facets.MinInclusiveFixed {
				return cur.Facets.MinInclusive
			}
		case facetMaxInclusive:
			if cur.Facets.MaxInclusive != nil && cur.Facets.MaxInclusiveFixed {
				return cur.Facets.MaxInclusive
			}
		case facetMinExclusive:
			if cur.Facets.MinExclusive != nil && cur.Facets.MinExclusiveFixed {
				return cur.Facets.MinExclusive
			}
		case facetMaxExclusive:
			if cur.Facets.MaxExclusive != nil && cur.Facets.MaxExclusiveFixed {
				return cur.Facets.MaxExclusive
			}
		}
	}
	return nil
}

type inheritedRangeBound struct {
	name      string
	value     *string
	exclusive bool
	immediate bool
}

func effectiveInheritedExclusiveRangeFacet(td *TypeDef, name, builtinLocal string) *string {
	lower, upper := effectiveInheritedRangeBounds(td, func(a, b string) (int, bool) {
		return rangeFacetCmp(a, b, builtinLocal)
	})
	switch name {
	case facetMinExclusive:
		if lower.exclusive {
			return lower.value
		}
	case facetMaxExclusive:
		if upper.exclusive {
			return upper.value
		}
	}
	return nil
}

func effectiveInheritedRangeBounds(td *TypeDef, cmp func(string, string) (int, bool)) (inheritedRangeBound, inheritedRangeBound) {
	var lower, upper inheritedRangeBound
	if td == nil || td.BaseType == nil {
		return lower, upper
	}
	immediate := true
	for cur := range baseChain(td.BaseType) {
		if cur.Facets == nil {
			immediate = false
			continue
		}
		mergeInheritedLowerBound(&lower, facetMinInclusive, cur.Facets.MinInclusive, false, immediate, cmp)
		mergeInheritedLowerBound(&lower, facetMinExclusive, cur.Facets.MinExclusive, true, immediate, cmp)
		mergeInheritedUpperBound(&upper, facetMaxInclusive, cur.Facets.MaxInclusive, false, immediate, cmp)
		mergeInheritedUpperBound(&upper, facetMaxExclusive, cur.Facets.MaxExclusive, true, immediate, cmp)
		immediate = false
	}
	return lower, upper
}

func mergeInheritedLowerBound(bound *inheritedRangeBound, name string, candidate *string, exclusive, immediate bool, cmp func(string, string) (int, bool)) {
	if candidate == nil {
		return
	}
	if bound.value == nil {
		bound.name = name
		bound.value = candidate
		bound.exclusive = exclusive
		bound.immediate = immediate
		return
	}
	if v, ok := cmp(*candidate, *bound.value); ok {
		if v > 0 || (v == 0 && exclusive && !bound.exclusive) {
			bound.name = name
			bound.value = candidate
			bound.exclusive = exclusive
			bound.immediate = immediate
		}
	}
}

func mergeInheritedUpperBound(bound *inheritedRangeBound, name string, candidate *string, exclusive, immediate bool, cmp func(string, string) (int, bool)) {
	if candidate == nil {
		return
	}
	if bound.value == nil {
		bound.name = name
		bound.value = candidate
		bound.exclusive = exclusive
		bound.immediate = immediate
		return
	}
	if v, ok := cmp(*candidate, *bound.value); ok {
		if v < 0 || (v == 0 && exclusive && !bound.exclusive) {
			bound.name = name
			bound.value = candidate
			bound.exclusive = exclusive
			bound.immediate = immediate
		}
	}
}

func rangeFacetCmp(a, b, builtinLocal string) (int, bool) {
	if cmp, ok := value.CompareFloatFacetBound(a, b, builtinLocal); ok {
		return cmp, true
	}
	if cmp, ok := compareForRangeFacet(a, b, builtinLocal); ok {
		return cmp, true
	}
	return 0, false
}

func rangeFacetValueEqual(a, b, builtinLocal string) bool {
	if cmp, ok := rangeFacetCmp(a, b, builtinLocal); ok {
		return cmp == 0
	}
	return a == b
}

func (c *compiler) checkBuiltinFixedFacetRestriction(ctx context.Context, td *TypeDef, fs *FacetSet, line int, component string) {
	builtinLocal := builtinBaseLocal(td)

	// xs:integer and every type derived from it carry the built-in FIXED facet
	// fractionDigits=0 (§3.3.13). A restriction may not change a fixed facet, so a
	// non-zero fractionDigits on an integer-family type is a schema error (an
	// explicit fractionDigits="0" is permitted — it equals the fixed value). This
	// is version-independent; the fixed facet holds identically in XSD 1.0 and 1.1.
	if fs.FractionDigits != nil && *fs.FractionDigits != 0 && value.IsIntegerFamily(builtinLocal) {
		c.schemaError(ctx, schemaComponentError(c.filename, line, "simpleType", component,
			fmt.Sprintf("The value '%d' of the facet 'fractionDigits' does not match the fixed value '0' of the base type '%s'.",
				*fs.FractionDigits, typeDisplayName(td.BaseType))))
	}

	if fs.WhiteSpace != nil {
		if fixed, ok := fixedBuiltinWhiteSpace(builtinLocal); ok {
			if *fs.WhiteSpace != fixed {
				c.schemaError(ctx, schemaComponentError(c.filename, line, "simpleType", component,
					fmt.Sprintf("The value '%s' of the facet 'whiteSpace' does not match the fixed value '%s' of the base type '%s'.",
						*fs.WhiteSpace, fixed, typeDisplayName(td.BaseType))))
			}
		} else if td.BaseType != nil {
			// whiteSpace Valid Restriction (§4.3.6): the derived value must not be
			// LESS restrictive than the base type's effective whiteSpace. The
			// ordering is preserve < replace < collapse, so a restriction of a
			// `replace` base (e.g. xs:normalizedString) to `preserve`, or of a
			// `collapse` base (e.g. xs:token) to `preserve`/`replace`, is a schema
			// error. Version-independent. (The fixed-builtin case above already
			// covers the date/time family, which fixes whiteSpace=collapse.)
			inherited := resolveWhiteSpace(td.BaseType)
			if whiteSpaceRank(*fs.WhiteSpace) < whiteSpaceRank(inherited) {
				c.schemaError(ctx, schemaComponentError(c.filename, line, "simpleType", component,
					fmt.Sprintf("The value '%s' of the facet 'whiteSpace' is less restrictive than the 'whiteSpace' value '%s' of the base type '%s'.",
						*fs.WhiteSpace, inherited, typeDisplayName(td.BaseType))))
			}
		}
	}

	if fs.ExplicitTimezone == nil {
		return
	}
	if fixedValue := inheritedFixedExplicitTimezone(td); fixedValue != "" && *fs.ExplicitTimezone != fixedValue {
		c.schemaError(ctx, schemaComponentError(c.filename, line, "simpleType", component,
			fmt.Sprintf("The value '%s' of the facet 'explicitTimezone' does not match the fixed value '%s' of the base type '%s'.",
				*fs.ExplicitTimezone, fixedValue, typeDisplayName(td.BaseType))))
		return
	}
	baseValue := baseExplicitTimezone(td)
	switch baseValue {
	case attrValRequired:
		if *fs.ExplicitTimezone != attrValRequired {
			c.schemaError(ctx, schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The value '%s' of the facet 'explicitTimezone' does not match the fixed value '%s' of the base type '%s'.",
					*fs.ExplicitTimezone, attrValRequired, typeDisplayName(td.BaseType))))
		}
	case attrValProhibited:
		if *fs.ExplicitTimezone != attrValProhibited {
			c.schemaError(ctx, schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The value '%s' of the facet 'explicitTimezone' does not match the fixed value '%s' of the base type '%s'.",
					*fs.ExplicitTimezone, attrValProhibited, typeDisplayName(td.BaseType))))
		}
	}
}

func explicitTimezoneApplicable(builtinLocal string) bool {
	switch builtinLocal {
	case lexicon.TypeDateTime, lexicon.TypeDateTimeStamp, lexicon.TypeDate, lexicon.TypeTime,
		lexicon.TypeGYear, lexicon.TypeGYearMonth, lexicon.TypeGMonth, lexicon.TypeGDay, lexicon.TypeGMonthDay:
		return true
	default:
		return false
	}
}

// whiteSpaceRank orders the three whiteSpace facet values by restrictiveness so a
// derived value can be compared against the inherited base value (§4.3.6):
// preserve (0) < replace (1) < collapse (2). An unrecognized value ranks as the
// least restrictive so it never spuriously rejects a restriction.
func whiteSpaceRank(v string) int {
	switch v {
	case "replace":
		return 1
	case "collapse":
		return 2
	default: // "preserve" and any unrecognized value
		return 0
	}
}

func fixedBuiltinWhiteSpace(builtinLocal string) (string, bool) {
	switch builtinLocal {
	case lexicon.TypeDateTime, lexicon.TypeDateTimeStamp, lexicon.TypeDate, lexicon.TypeTime,
		lexicon.TypeDuration, lexicon.TypeDayTimeDuration, lexicon.TypeYearMonthDuration,
		lexicon.TypeGYear, lexicon.TypeGYearMonth, lexicon.TypeGMonth, lexicon.TypeGDay, lexicon.TypeGMonthDay:
		return "collapse", true
	default:
		return "", false
	}
}

func baseExplicitTimezone(td *TypeDef) string {
	if td == nil || td.BaseType == nil {
		return ""
	}
	for cur := range baseChain(td.BaseType) {
		if cur.Facets != nil && cur.Facets.ExplicitTimezone != nil {
			return *cur.Facets.ExplicitTimezone
		}
		if cur.Name.NS != lexicon.NamespaceXSD {
			continue
		}
		switch cur.Name.Local {
		case lexicon.TypeDateTimeStamp:
			return attrValRequired
		case lexicon.TypeDateTime, lexicon.TypeDate, lexicon.TypeTime,
			lexicon.TypeGYear, lexicon.TypeGYearMonth, lexicon.TypeGMonth, lexicon.TypeGDay, lexicon.TypeGMonthDay:
			return attrValOptional
		default:
			return ""
		}
	}
	return ""
}

func inheritedFixedExplicitTimezone(td *TypeDef) string {
	if td == nil || td.BaseType == nil {
		return ""
	}
	for cur := range baseChain(td.BaseType) {
		if cur.Facets != nil && cur.Facets.ExplicitTimezone != nil && cur.Facets.ExplicitTimezoneFixed {
			return *cur.Facets.ExplicitTimezone
		}
		if cur.Name.NS == lexicon.NamespaceXSD && cur.Name.Local == lexicon.TypeDateTimeStamp {
			return attrValRequired
		}
	}
	return ""
}
