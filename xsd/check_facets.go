package xsd

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
)

// baseFacets returns the FacetSet from the nearest base type in the chain.
func baseFacets(td *TypeDef) *FacetSet {
	if td.BaseType == nil {
		return nil
	}
	for cur := td.BaseType; cur != nil; cur = cur.BaseType {
		if cur.Facets != nil {
			return cur.Facets
		}
	}
	return nil
}

// checkFacetConsistency validates facet constraints for all named types.
// It checks same-type mutual exclusion, same-type consistency, and
// base-type restriction narrowing rules.
func (c *compiler) checkFacetConsistency(ctx context.Context) {
	if c.filename == "" {
		return
	}

	// Collect and sort types by name for deterministic error ordering.
	type facetEntry struct {
		qn QName
		td *TypeDef
	}
	var entries []facetEntry
	for qn, td := range c.schema.types {
		if td.Facets == nil {
			continue
		}
		if qn.NS == lexicon.NamespaceXSD {
			continue
		}
		entries = append(entries, facetEntry{qn: qn, td: td})
	}
	sort.Slice(entries, func(i, j int) bool {
		si, oki := c.typeDefSources[entries[i].td]
		sj, okj := c.typeDefSources[entries[j].td]
		if oki && okj {
			return si.line < sj.line
		}
		if oki != okj {
			return oki
		}
		return entries[i].qn.Local < entries[j].qn.Local
	})

	for _, entry := range entries {
		td := entry.td
		fs := td.Facets

		src, hasSrc := c.typeDefSources[td]
		component := td.Name.Local
		if component == "" {
			component = "local simple type"
		}
		line := 0
		if hasSrc {
			line = src.line
		}

		c.checkFacetMutualExclusion(ctx, fs, line, component)
		c.checkFacetSameTypeConsistency(ctx, fs, line, component)
		c.checkFacetBaseRestriction(ctx, td, fs, line, component)
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
		return entries[i].td.Name.Local < entries[j].td.Name.Local
	})

	for _, e := range entries {
		td := e.td
		if td.Name.NS == lexicon.NamespaceXSD {
			continue
		}

		component := td.Name.Local
		if component == "" || e.src.isLocal {
			component = "local simple type"
		}

		variety := resolveVariety(td)

		// An un-enumerated xs:NOTATION use is not a permitted derivation. This is
		// checked recursively over the type's variety so a NOTATION carrier hidden
		// inside a list item type or a union member is caught too, not only a
		// direct atomic xs:NOTATION restriction base. A NOTATION carrier is allowed
		// only when it is itself enumeration-derived.
		if notationUsedWithoutEnumeration(td) {
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, e.src.line, "simpleType", component,
				"It is an error if the base type is the built-in 'NOTATION' and there is no 'enumeration' facet."), helium.ErrorLevelFatal))
			c.errorCount++
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
				c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, e.src.line, "simpleType", component, msg), helium.ErrorLevelFatal))
				c.errorCount++
			}
		}
	}
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
		for item := range strings.FieldsSeq(ev) {
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
			sub := &validationContext{errorHandler: helium.NilErrorHandler{}, suppressDepth: 1}
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
		_, err := resolveLexicalQName(ev, enumNS)
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
	if td == nil {
		return false
	}
	switch resolveVariety(td) {
	case TypeVarietyList:
		return notationCarrierNotEnumerated(resolveItemType(td))
	case TypeVarietyUnion:
		return slices.ContainsFunc(resolveUnionMembers(td), notationCarrierNotEnumerated)
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
	if td == nil {
		return false
	}
	switch resolveVariety(td) {
	case TypeVarietyList:
		return typeHasQNameNotationCarrier(resolveItemType(td))
	case TypeVarietyUnion:
		return slices.ContainsFunc(resolveUnionMembers(td), typeHasQNameNotationCarrier)
	default:
		bl := builtinBaseLocal(td)
		return bl == lexicon.TypeQName || bl == lexicon.TypeNotation
	}
}

// hasEffectiveEnumeration reports whether td or any of its base types along the
// restriction chain carries an enumeration facet.
func hasEffectiveEnumeration(td *TypeDef) bool {
	for cur := td; cur != nil; cur = cur.BaseType {
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
		c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserError(c.filename, e.src.line, e.src.elemName, elemElement,
			"It is an error if the type definition is the built-in 'NOTATION' and there is no 'enumeration' facet."), helium.ErrorLevelFatal))
		c.errorCount++
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
		c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserError(c.filename, a.src.line, a.src.local, elemAttribute,
			"It is an error if the type definition is the built-in 'NOTATION' and there is no 'enumeration' facet."), helium.ErrorLevelFatal))
		c.errorCount++
	}
}

// checkFacetMutualExclusion checks that mutually exclusive facets are not
// both specified on the same type definition.
func (c *compiler) checkFacetMutualExclusion(ctx context.Context, fs *FacetSet, line int, component string) {
	if fs.Length != nil && (fs.MinLength != nil || fs.MaxLength != nil) {
		c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
			"It is an error for both 'length' and either of 'minLength' or 'maxLength' to be specified on the same type definition."), helium.ErrorLevelFatal))
		c.errorCount++
	}
	if fs.MaxInclusive != nil && fs.MaxExclusive != nil {
		c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
			"It is an error for both 'maxInclusive' and 'maxExclusive' to be specified."), helium.ErrorLevelFatal))
		c.errorCount++
	}
	if fs.MinInclusive != nil && fs.MinExclusive != nil {
		c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
			"It is an error for both 'minInclusive' and 'minExclusive' to be specified."), helium.ErrorLevelFatal))
		c.errorCount++
	}
}

// checkFacetSameTypeConsistency checks consistency of facets within the same type.
func (c *compiler) checkFacetSameTypeConsistency(ctx context.Context, fs *FacetSet, line int, component string) {
	if fs.MinLength != nil && fs.MaxLength != nil && *fs.MinLength > *fs.MaxLength {
		c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
			"It is an error for the value of 'minLength' to be greater than the value of 'maxLength'."), helium.ErrorLevelFatal))
		c.errorCount++
	}
	if fs.MinInclusive != nil && fs.MaxInclusive != nil {
		if compareDecimal(*fs.MinInclusive, *fs.MaxInclusive) > 0 {
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
				"It is an error for the value of 'minInclusive' to be greater than the value of 'maxInclusive'."), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}
	if fs.MinExclusive != nil && fs.MaxExclusive != nil {
		if compareDecimal(*fs.MinExclusive, *fs.MaxExclusive) >= 0 {
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
				"It is an error for the value of 'minExclusive' to be greater than or equal to the value of 'maxExclusive'."), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}
	if fs.FractionDigits != nil && fs.TotalDigits != nil && *fs.FractionDigits > *fs.TotalDigits {
		c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
			"It is an error for the value of 'fractionDigits' to be greater than the value of 'totalDigits'."), helium.ErrorLevelFatal))
		c.errorCount++
	}
	if fs.MinExclusive != nil && fs.MaxInclusive != nil {
		if compareDecimal(*fs.MinExclusive, *fs.MaxInclusive) >= 0 {
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
				"It is an error for the value of 'minExclusive' to be greater than or equal to the value of 'maxInclusive'."), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}
	if fs.MinInclusive != nil && fs.MaxExclusive != nil {
		if compareDecimal(*fs.MinInclusive, *fs.MaxExclusive) >= 0 {
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
				"It is an error for the value of 'minInclusive' to be greater than or equal to the value of 'maxExclusive'."), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}
}

// checkFacetBaseRestriction checks that facet values properly narrow (not widen)
// the base type's facets.
func (c *compiler) checkFacetBaseRestriction(ctx context.Context, td *TypeDef, fs *FacetSet, line int, component string) {
	base := baseFacets(td)
	if base == nil {
		return
	}

	// Length facets.
	if fs.MinLength != nil && base.MinLength != nil && *fs.MinLength < *base.MinLength {
		c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
			fmt.Sprintf("The 'minLength' value '%d' is less than the 'minLength' value of the base type '%d'.", *fs.MinLength, *base.MinLength)), helium.ErrorLevelFatal))
		c.errorCount++
	}
	if fs.MaxLength != nil && base.MaxLength != nil && *fs.MaxLength > *base.MaxLength {
		c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
			fmt.Sprintf("The 'maxLength' value '%d' is greater than the 'maxLength' value of the base type '%d'.", *fs.MaxLength, *base.MaxLength)), helium.ErrorLevelFatal))
		c.errorCount++
	}
	if fs.Length != nil && base.Length != nil && *fs.Length != *base.Length {
		c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
			fmt.Sprintf("The 'length' value '%d' does not match the 'length' value of the base type '%d'.", *fs.Length, *base.Length)), helium.ErrorLevelFatal))
		c.errorCount++
	}

	// Digit facets.
	if fs.TotalDigits != nil && base.TotalDigits != nil && *fs.TotalDigits > *base.TotalDigits {
		c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
			fmt.Sprintf("The 'totalDigits' value '%d' is greater than the 'totalDigits' value of the base type '%d'.", *fs.TotalDigits, *base.TotalDigits)), helium.ErrorLevelFatal))
		c.errorCount++
	}
	if fs.FractionDigits != nil && base.FractionDigits != nil && *fs.FractionDigits > *base.FractionDigits {
		c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
			fmt.Sprintf("The 'fractionDigits' value '%d' is greater than the 'fractionDigits' value of the base type '%d'.", *fs.FractionDigits, *base.FractionDigits)), helium.ErrorLevelFatal))
		c.errorCount++
	}

	// Inclusive/exclusive boundary facets vs base.
	if fs.MaxInclusive != nil && base.MaxInclusive != nil {
		if compareDecimal(*fs.MaxInclusive, *base.MaxInclusive) > 0 {
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'maxInclusive' value '%s' is greater than the 'maxInclusive' value of the base type '%s'.", *fs.MaxInclusive, *base.MaxInclusive)), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}
	if fs.MaxInclusive != nil && base.MaxExclusive != nil {
		if compareDecimal(*fs.MaxInclusive, *base.MaxExclusive) >= 0 {
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'maxInclusive' value '%s' must be less than the 'maxExclusive' value of the base type '%s'.", *fs.MaxInclusive, *base.MaxExclusive)), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}
	if fs.MaxInclusive != nil && base.MinInclusive != nil {
		if compareDecimal(*fs.MaxInclusive, *base.MinInclusive) < 0 {
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'maxInclusive' value '%s' is less than the 'minInclusive' value of the base type '%s'.", *fs.MaxInclusive, *base.MinInclusive)), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}
	if fs.MaxInclusive != nil && base.MinExclusive != nil {
		if compareDecimal(*fs.MaxInclusive, *base.MinExclusive) <= 0 {
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'maxInclusive' value '%s' must be greater than the 'minExclusive' value of the base type '%s'.", *fs.MaxInclusive, *base.MinExclusive)), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}
	if fs.MaxExclusive != nil && base.MaxExclusive != nil {
		if compareDecimal(*fs.MaxExclusive, *base.MaxExclusive) > 0 {
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'maxExclusive' value '%s' is greater than the 'maxExclusive' value of the base type '%s'.", *fs.MaxExclusive, *base.MaxExclusive)), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}
	if fs.MaxExclusive != nil && base.MaxInclusive != nil {
		if compareDecimal(*fs.MaxExclusive, *base.MaxInclusive) > 0 {
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'maxExclusive' value '%s' is greater than the 'maxInclusive' value of the base type '%s'.", *fs.MaxExclusive, *base.MaxInclusive)), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}
	if fs.MaxExclusive != nil && base.MinInclusive != nil {
		if compareDecimal(*fs.MaxExclusive, *base.MinInclusive) <= 0 {
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'maxExclusive' value '%s' must be greater than the 'minInclusive' value of the base type '%s'.", *fs.MaxExclusive, *base.MinInclusive)), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}
	if fs.MaxExclusive != nil && base.MinExclusive != nil {
		if compareDecimal(*fs.MaxExclusive, *base.MinExclusive) <= 0 {
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'maxExclusive' value '%s' must be greater than the 'minExclusive' value of the base type '%s'.", *fs.MaxExclusive, *base.MinExclusive)), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}
	if fs.MinInclusive != nil && base.MinInclusive != nil {
		if compareDecimal(*fs.MinInclusive, *base.MinInclusive) < 0 {
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'minInclusive' value '%s' is less than the 'minInclusive' value of the base type '%s'.", *fs.MinInclusive, *base.MinInclusive)), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}
	if fs.MinInclusive != nil && base.MinExclusive != nil {
		if compareDecimal(*fs.MinInclusive, *base.MinExclusive) <= 0 {
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'minInclusive' value '%s' must be greater than the 'minExclusive' value of the base type '%s'.", *fs.MinInclusive, *base.MinExclusive)), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}
	if fs.MinInclusive != nil && base.MaxInclusive != nil {
		if compareDecimal(*fs.MinInclusive, *base.MaxInclusive) > 0 {
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'minInclusive' value '%s' is greater than the 'maxInclusive' value of the base type '%s'.", *fs.MinInclusive, *base.MaxInclusive)), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}
	if fs.MinInclusive != nil && base.MaxExclusive != nil {
		if compareDecimal(*fs.MinInclusive, *base.MaxExclusive) >= 0 {
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'minInclusive' value '%s' must be less than the 'maxExclusive' value of the base type '%s'.", *fs.MinInclusive, *base.MaxExclusive)), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}
	if fs.MinExclusive != nil && base.MinExclusive != nil {
		if compareDecimal(*fs.MinExclusive, *base.MinExclusive) < 0 {
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'minExclusive' value '%s' is less than the 'minExclusive' value of the base type '%s'.", *fs.MinExclusive, *base.MinExclusive)), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}
	if fs.MinExclusive != nil && base.MinInclusive != nil {
		if compareDecimal(*fs.MinExclusive, *base.MinInclusive) < 0 {
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'minExclusive' value '%s' is less than the 'minInclusive' value of the base type '%s'.", *fs.MinExclusive, *base.MinInclusive)), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}
	if fs.MinExclusive != nil && base.MaxInclusive != nil {
		if compareDecimal(*fs.MinExclusive, *base.MaxInclusive) > 0 {
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'minExclusive' value '%s' is greater than the 'maxInclusive' value of the base type '%s'.", *fs.MinExclusive, *base.MaxInclusive)), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}
	if fs.MinExclusive != nil && base.MaxExclusive != nil {
		if compareDecimal(*fs.MinExclusive, *base.MaxExclusive) >= 0 {
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, line, "simpleType", component,
				fmt.Sprintf("The 'minExclusive' value '%s' must be less than the 'maxExclusive' value of the base type '%s'.", *fs.MinExclusive, *base.MaxExclusive)), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}
}
