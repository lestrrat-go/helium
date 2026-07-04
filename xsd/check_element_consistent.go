package xsd

import (
	"context"
	"fmt"
	"sort"

	"github.com/lestrrat-go/helium/internal/lexicon"
)

// checkElementConsistent enforces the XSD "Element Declarations Consistent"
// constraint (XSD 1.1 Part 1, §3.8.6.3 cos-element-consistent): if the
// {particles} of a content model contain two or more element declarations with
// the same {name} and {target namespace}, then all their {type definition}s
// must be the same. Two same-named element declarations with incompatible types
// in one effective content model are a fatal schema error.
//
// libxml2 does not implement this constraint (it is an "URGENT TODO" in
// xmlschemas.c), so there is no libxml2 wording to mirror; the diagnostic is
// emitted in the existing component-error style.
//
// This runs after reference resolution and substitution-group construction (so
// every element particle's Type, every group reference's Particles, and
// schema.substGroups are populated) and only when no prior schema error has been
// reported, matching the gating of checkUPA.
//
// Coverage: a content model's same-name element declarations are compared via
// their substitution-group-aware {type definition}s (see resolveDeclaredType).
// In addition, for each element term the constraint folds in the term's
// transitive substitution-group MEMBERS as implicitly-containable declarations
// (a head's particle stands in for any of its members, and substitution chains
// transitively), so a head reference colliding by name with a different-typed
// local element is rejected. The check also runs
// over standalone named xs:group definitions, so a named group never referenced
// by a complex type is still checked.
//
// Conservatism: the check is only ever under-strict (it may fail to reject a
// genuinely inconsistent schema) and never false-rejects a valid one — a missed
// violation is safe, a false reject breaks valid schemas.
func (c *compiler) checkElementConsistent(ctx context.Context) {
	if c.filename == "" || c.errorCount != 0 {
		return
	}

	// Order complex-type content models by source line then ordinal so
	// diagnostics are deterministic regardless of Go map iteration order.
	type checkedType struct {
		td  *TypeDef
		src typeDefSource
	}
	checked := make([]checkedType, 0, len(c.typeDefSources))
	for td, src := range c.typeDefSources {
		if td.ContentModel == nil {
			continue
		}
		checked = append(checked, checkedType{td: td, src: src})
	}
	sort.Slice(checked, func(i, j int) bool {
		if checked[i].src.line != checked[j].src.line {
			return checked[i].src.line < checked[j].src.line
		}
		return checked[i].src.ordinal < checked[j].src.ordinal
	})

	for _, ct := range checked {
		// Attribute the diagnostic to the file the type was declared in (an
		// include/import file when set) so the cited line number matches that
		// file, not the top-level schema label. Types declared directly in the
		// top-level schema have an empty source and fall back to c.filename.
		source := ct.src.source
		if source == "" {
			source = c.filename
		}
		c.checkContentModelConsistent(ctx, ct.td.ContentModel, source, ct.src.line, c.complexTypeComponent(ct.td, ct.src))
		// The type's EFFECTIVE open-content wildcard can also resolve a same-named
		// declared local element to a global (interleave moves an extra occurrence to
		// the open partition; suffix matches a trailing one), so it participates in the
		// type-table EDC exactly like a content-model wildcard.
		c.checkWildcardElementConsistent(ctx, ct.td.ContentModel, ct.td.OpenContent, source, ct.src.line, "complexType", c.complexTypeComponent(ct.td, ct.src))
	}

	// Run the same consistency check over standalone named model group
	// definitions (xs:group name="..."). A named group that no complex type
	// references is otherwise never reached through a type's content model, so
	// its own same-name inconsistencies would be missed.
	type checkedGroup struct {
		mg  *ModelGroup
		src groupSource
		qn  QName
	}
	groups := make([]checkedGroup, 0, len(c.schema.groups))
	for qn, mg := range c.schema.groups {
		if mg == nil {
			continue
		}
		src, ok := c.groupSources[qn]
		if !ok {
			continue
		}
		groups = append(groups, checkedGroup{mg: mg, src: src, qn: qn})
	}
	// Stable source order: declaring file, then line, then QName.
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].src.source != groups[j].src.source {
			return groups[i].src.source < groups[j].src.source
		}
		if groups[i].src.line != groups[j].src.line {
			return groups[i].src.line < groups[j].src.line
		}
		if groups[i].qn.NS != groups[j].qn.NS {
			return groups[i].qn.NS < groups[j].qn.NS
		}
		return groups[i].qn.Local < groups[j].qn.Local
	})
	for _, g := range groups {
		component := "model group '" + g.qn.Local + "'"
		c.checkNamedGroupConsistent(ctx, g.mg, g.src, component)
		// A named group has no open content (it is a complexType-level construct).
		c.checkWildcardElementConsistent(ctx, g.mg, nil, g.src.source, g.src.line, "group", component)
	}
}

// checkWildcardElementConsistent enforces the XSD 1.1 addition to "Element
// Declarations Consistent" (cos-element-consistent) that governs the interaction
// between a wildcard particle and a same-named GLOBAL element declaration: if a
// content model contains an element declaration particle E with expanded name N
// AND a wildcard particle W whose namespace constraint ·allows· N, and there is a
// top-level (global) element declaration G with expanded name N that the wildcard
// could resolve to (lax/strict), then E's {type table} must be the same as G's
// {type table}. A strict/lax wildcard match of <N> resolves to G, so if E and G
// have inconsistent conditional-type-assignment tables the same element could be
// governed by different type tables depending on which particle matched.
//
// Only the {type table} (xs:alternative list) is compared, NOT the {type
// definition}: a wildcard intentionally admits elements of differing types, so a
// type-definition difference between E and G is permitted (e.g. a local element
// of one simple type plus a wildcard matching a differently-typed global of the
// same name is valid). The comparison is deliberately under-strict: a mismatch is
// flagged only when EXACTLY ONE of E/G carries a non-empty type table (the
// observed conformance violations), so two distinct-but-present type tables are
// not false-rejected.
//
// The type's EFFECTIVE open-content wildcard (oc) is folded into the wildcard set:
// interleave open content can move an extra same-name declared child into the open
// partition, and suffix can match a trailing one, where validateWildcardChild governs
// it via the GLOBAL declaration's CTA type table — so an inconsistent type table is
// reachable through it exactly as through a content-model wildcard. A skip
// open-content wildcard imposes no constraint (anyWildcardAllows skips it), like a
// content-model skip wildcard. oc is nil for a named group (no open content).
//
// Gated on XSD 1.1: a wildcard never resolves to a same-named global in 1.0 mode
// for this constraint, and 1.0 stays byte-identical.
func (c *compiler) checkWildcardElementConsistent(ctx context.Context, mg *ModelGroup, oc *OpenContent, source string, line int, kind, component string) {
	if c.version != Version11 || mg == nil {
		return
	}
	elems, wildcards := c.collectModelGroupParticles(mg)
	if oc != nil && oc.Wildcard != nil {
		wildcards = append(wildcards, oc.Wildcard)
	}
	if len(wildcards) == 0 || len(elems) == 0 {
		return
	}
	for _, qn := range sortedNames(elems) {
		global, ok := c.schema.elements[qn]
		if !ok {
			continue
		}
		for _, decl := range elems[qn] {
			// A reference to the global declaration itself (or the global decl) is
			// the same component as G — no inconsistency is possible.
			if decl == global || decl.IsRef {
				continue
			}
			if typeTablesConsistent(decl, global) {
				continue
			}
			if !c.anyWildcardAllows(wildcards, qn) {
				continue
			}
			msg := fmt.Sprintf("The wildcard matches the global element declaration '%s', whose type table is inconsistent with the like-named local element declaration's type table.", qn.Local)
			c.schemaError(ctx, schemaComponentError(source, line, kind, component, msg))
			break
		}
	}
}

// anyWildcardAllows reports whether any of the wildcards admits the expanded name
// via a lax/strict match (skip wildcards never resolve to a global declaration,
// so they impose no EDC type-table constraint).
func (c *compiler) anyWildcardAllows(wildcards []*Wildcard, qn QName) bool {
	for _, wc := range wildcards {
		if wc.ProcessContents == ProcessSkip {
			continue
		}
		if wildcardAllowsExpandedName(wc, qn.Local, qn.NS, c.schema, false) {
			return true
		}
	}
	return false
}

// typeTablesConsistent reports whether two element declarations have consistent
// {type table}s for the wildcard EDC check. Two tables are consistent when both
// are absent OR both present AND EQUIVALENT; the asymmetric case (exactly one
// present) and two present-but-NON-equivalent tables are inconsistent — matching
// the same-name EDC (typeTablesEquivalent), so the same element cannot get
// different governing type tables depending on whether it is reached via the local
// particle or via a lax wildcard to the global declaration. Differing type
// DEFINITIONS remain permitted (#886); only the TYPE TABLE is compared here.
func typeTablesConsistent(a, b *ElementDecl) bool {
	aHas := len(a.Alternatives) > 0
	bHas := len(b.Alternatives) > 0
	if aHas != bHas {
		return false
	}
	if !aHas {
		return true
	}
	return typeTablesEquivalent(a.Alternatives, b.Alternatives)
}

// collectModelGroupParticles walks a content model and returns its LOCAL element
// declaration particles keyed by expanded name (no substitution-group folding;
// raw particle declarations) together with every wildcard particle reachable in
// the model. Prohibited particles (maxOccurs=0) are excluded.
func (c *compiler) collectModelGroupParticles(mg *ModelGroup) (map[QName][]*ElementDecl, []*Wildcard) {
	elems := make(map[QName][]*ElementDecl)
	var wildcards []*Wildcard
	visited := make(map[*ModelGroup]struct{})
	var walk func(*ModelGroup)
	walk = func(g *ModelGroup) {
		if g == nil || g.MaxOccurs == 0 {
			return
		}
		if _, seen := visited[g]; seen {
			return
		}
		visited[g] = struct{}{}
		for _, p := range g.Particles {
			if p.MaxOccurs == 0 {
				continue
			}
			switch term := p.Term.(type) {
			case *ElementDecl:
				elems[term.Name] = append(elems[term.Name], term)
			case *Wildcard:
				wildcards = append(wildcards, term)
			case *ModelGroup:
				walk(term)
			}
		}
	}
	walk(mg)
	return elems, wildcards
}

// complexTypeComponent returns the diagnostic component label for a complex
// type, matching the wording used by the other cos/derivation checks.
func (c *compiler) complexTypeComponent(td *TypeDef, src typeDefSource) string {
	if src.isLocal {
		return componentLocalComplexType
	}
	return td.Name.Local
}

// checkContentModelConsistent collects every element declaration reachable in mg
// (including substitution-group members implied by each element term), grouped
// by expanded name, and reports the first inconsistent pair found per name. The
// diagnostic is attributed to a complexType component.
func (c *compiler) checkContentModelConsistent(ctx context.Context, mg *ModelGroup, source string, line int, component string) {
	byName := c.collectModelGroupElements(mg)
	for _, qn := range sortedNames(byName) {
		decls := byName[qn]
		if len(decls) < 2 {
			continue
		}
		if !c.declsConsistent(decls) {
			msg := fmt.Sprintf("Two elements with the same name '%s' and namespace '%s', but different type definitions, appear in the content model.", qn.Local, qn.NS)
			c.schemaError(ctx, schemaComponentError(source, line, "complexType", component, msg))
		}
	}
}

// checkNamedGroupConsistent runs the consistency check over a standalone named
// model group definition, reporting against the declaring file with a group
// component label.
func (c *compiler) checkNamedGroupConsistent(ctx context.Context, mg *ModelGroup, src groupSource, component string) {
	byName := c.collectModelGroupElements(mg)
	for _, qn := range sortedNames(byName) {
		decls := byName[qn]
		if len(decls) < 2 {
			continue
		}
		if !c.declsConsistent(decls) {
			msg := fmt.Sprintf("Two elements with the same name '%s' and namespace '%s', but different type definitions, appear in the content model.", qn.Local, qn.NS)
			c.schemaError(ctx, schemaComponentError(src.source, src.line, "group", component, msg))
		}
	}
}

// collectModelGroupElements returns every element declaration reachable in mg
// keyed by expanded name, folding in each element term's substitution-group
// members as implicitly-containable declarations.
func (c *compiler) collectModelGroupElements(mg *ModelGroup) map[QName][]*ElementDecl {
	byName := make(map[QName][]*ElementDecl)
	c.collectContentModelElements(mg, byName, make(map[*ModelGroup]struct{}))
	return byName
}

// declsConsistent reports whether all declarations sharing an expanded name have
// the same substitution-group-aware {type definition}. It compares each later
// declaration against the first; only a genuinely different type is inconsistent.
func (c *compiler) declsConsistent(decls []*ElementDecl) bool {
	firstType := c.resolveDeclaredType(decls[0])
	for _, other := range decls[1:] {
		if !elementTypesConsistent(firstType, c.resolveDeclaredType(other)) {
			return false
		}
		// XSD 1.1 extends cos-element-consistent to require the same {type table}
		// (conditional type assignment) on same-named element declarations, so a
		// content model with two same-named elements carrying DIFFERENT type tables
		// (or one with a table and one without) is inconsistent (cta9009err/cta9010err).
		if c.version == Version11 && !c.typeTablesEDCConsistent(decls[0], other) {
			return false
		}
	}
	return true
}

// typeTablesEDCConsistent reports whether two same-named element declarations have
// equivalent {type table}s for the XSD 1.1 Element Declarations Consistent
// constraint (both must be absent or equivalent). It resolves each declaration's
// effective alternatives (own or, for a ref, the global's) and defers the
// structural comparison to the shared typeTablesEquivalent.
func (c *compiler) typeTablesEDCConsistent(a, b *ElementDecl) bool {
	return typeTablesEquivalent(elementAlternatives(a, c.schema), elementAlternatives(b, c.schema))
}

// sortedNames returns the keys of byName in a deterministic (namespace, local)
// order so reported errors are stable.
func sortedNames(byName map[QName][]*ElementDecl) []QName {
	names := make([]QName, 0, len(byName))
	for qn := range byName {
		names = append(names, qn)
	}
	sort.Slice(names, func(i, j int) bool {
		if names[i].NS != names[j].NS {
			return names[i].NS < names[j].NS
		}
		return names[i].Local < names[j].Local
	})
	return names
}

// collectContentModelElements walks a model group recursively, appending every
// element declaration term to byName keyed by its expanded name. For each
// element term it also folds in the term's transitive substitution-group
// MEMBERS that can actually substitute for the head (see foldSubstitutionMembers),
// keyed by each member's own name, so a head reference colliding by name with a
// different-typed same-named element elsewhere in the model is detected. The
// visited set bounds shared/recursive model-group structures (e.g. a group
// referenced from multiple places resolves to a shared *ModelGroup).
func (c *compiler) collectContentModelElements(mg *ModelGroup, byName map[QName][]*ElementDecl, visited map[*ModelGroup]struct{}) {
	if mg == nil {
		return
	}
	// A model group with maxOccurs="0" is a prohibited particle and maps to no
	// particle at all, so it contributes no element declarations to the
	// effective content model and must not be collected.
	if mg.MaxOccurs == 0 {
		return
	}
	if _, seen := visited[mg]; seen {
		return
	}
	visited[mg] = struct{}{}
	for _, p := range mg.Particles {
		// A particle with maxOccurs="0" is prohibited (it maps to NO particle),
		// so it is not part of the effective content model and must not be
		// collected or recursed into for cos-element-consistent.
		if p.MaxOccurs == 0 {
			continue
		}
		switch term := p.Term.(type) {
		case *ElementDecl:
			byName[term.Name] = append(byName[term.Name], term)
			// Fold in the term's transitive substitution-group members: a head's
			// particle implicitly contains its members (and their members), so each
			// is effectively present in this content model under its own name. This
			// applies only when the particle IS the global head declaration or a ref
			// to it — a distinct LOCAL element particle merely sharing a head's QName
			// admits no substitution members.
			if term.IsRef || c.schema.elements[term.Name] == term {
				c.foldSubstitutionMembers(term, byName)
			}
		case *ModelGroup:
			c.collectContentModelElements(term, byName, visited)
		}
	}
}

// foldSubstitutionMembers folds the transitive substitution-group closure of
// the head element term into byName. A head's particle implicitly stands in for
// any member that can actually substitute for it, and substitution chains
// transitively: if head admits mid and mid admits leaf, then leaf can also
// appear where head's particle is, so leaf must be folded in too. Walking only
// the DIRECT members of the head would miss such transitive members, letting a
// nested chain (head -> mid -> leaf) collide undetected with a different-typed
// same-named local element.
//
// Per §3.3.6.3 (Substitution Group OK, Transitive) a member can substitute for
// the ORIGINAL head only if it is substitutable at every step AND its derivation
// chain back to the original head's type does not use a method the original head
// disallows. So eligibility is checked against BOTH the immediate intermediate
// head 'cur' (block="substitution" and derivation-blocking for that step) AND the
// original head: a member blocked by the original head's disallowed substitutions
// cannot substitute for it and must not be folded in, even though it is reachable
// through an unblocked intermediate. Folding it would wrongly treat it as
// implicitly contained and could trigger a false same-name collision.
//
// The visited set is keyed by member name and guards against substitution-group
// cycles (rejected elsewhere, but defended against here so the walk terminates).
func (c *compiler) foldSubstitutionMembers(head *ElementDecl, byName map[QName][]*ElementDecl) {
	headType := c.resolveDeclaredType(head)
	// The original head's block="substitution" blocks every member outright.
	if head.Block&BlockSubstitution != 0 {
		return
	}
	visited := map[QName]struct{}{head.Name: {}}
	// queue holds heads whose direct members are not yet expanded.
	queue := []*ElementDecl{head}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		// A head with block="substitution" admits no members, so its subtree
		// contributes nothing further.
		if cur.Block&BlockSubstitution != 0 {
			continue
		}
		curType := c.resolveDeclaredType(cur)
		for _, member := range c.schema.substGroups[cur.Name] {
			memberType := c.resolveDeclaredType(member)
			// The member must be substitutable for the IMMEDIATE head 'cur' ...
			// (using each head's EFFECTIVE {disallowed substitutions}: the union of
			// its element block and its declared TYPE's {prohibited substitutions},
			// plus any INTERMEDIATE type's block on the member's derivation chain).
			if substTypeDerivationBlocked(memberType, curType, cur.Block) {
				continue
			}
			// ... and also for the ORIGINAL head, whose disallowed substitutions
			// apply transitively to every member of the group.
			if substTypeDerivationBlocked(memberType, headType, head.Block) {
				continue
			}
			if _, seen := visited[member.Name]; seen {
				continue
			}
			visited[member.Name] = struct{}{}
			byName[member.Name] = append(byName[member.Name], member)
			queue = append(queue, member)
		}
	}
}

// resolveDeclaredType returns the {type definition} that an element declaration
// contributes for cos-element-consistent comparison. A substitution-group member
// declared without an explicit type has Type == nil; its effective declared type
// is that of the substitution-group head (resolved transitively), and an element
// with no resolvable type at all defaults to xs:anyType (XSD 1.1 Part 1 §3.3.2).
// Comparing the raw Type pointer would treat such a member as nil and wrongly
// flag an inconsistency against its head, so resolve through SubstitutionGroup
// here before comparing. The seen set bounds malformed cyclic substitution
// groups (rejected elsewhere) so this never loops.
func (c *compiler) resolveDeclaredType(decl *ElementDecl) *TypeDef {
	if decl == nil {
		return c.schema.types[QName{Local: typeAnyType, NS: lexicon.NamespaceXSD}]
	}
	if decl.Type != nil {
		return decl.Type
	}
	if td := inheritedTypeFromFirstSubstitutionHead(decl, func(qn QName) (*ElementDecl, bool) {
		next, ok := c.schema.elements[qn]
		return next, ok
	}); td != nil {
		return td
	}
	// No explicit type and no resolvable substitution-group head type: the
	// declaration's {type definition} defaults to xs:anyType.
	return c.schema.types[QName{Local: typeAnyType, NS: lexicon.NamespaceXSD}]
}

// elementTypesConsistent reports whether two element declarations sharing an
// expanded name have consistent {type definition}s per cos-element-consistent.
// The constraint requires the type definitions to be the same component;
// helium shares a single *TypeDef pointer per named type and copies the global
// element's type pointer onto a ref, so identical components compare equal by
// pointer. This pointer identity also covers a shared ANONYMOUS type that is
// genuinely the same component: a global element with an inline type referenced
// twice, or a named group whose inline-typed element is expanded into two group
// references, both reuse the SAME *ElementDecl/*TypeDef, so the repeated
// occurrences are the same declaration and are consistent. Two distinct named
// types with the same expanded QName (which can arise across import merges) are
// also treated as the same component. Two genuinely distinct anonymous inline
// types have distinct pointers and an absent QName, so they compare unequal and
// are correctly inconsistent.
func elementTypesConsistent(a, b *TypeDef) bool {
	if a == b {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	// Same named type (non-empty QName) is the same component even if two
	// *TypeDef values were produced for it (e.g. recovery placeholders or
	// import-merged duplicates).
	if a.Name != (QName{}) && a.Name == b.Name {
		return true
	}
	return false
}
