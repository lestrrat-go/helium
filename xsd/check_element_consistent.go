package xsd

import (
	"context"
	"fmt"
	"sort"

	helium "github.com/lestrrat-go/helium"
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
// substitution-group MEMBERS as implicitly-containable declarations (a head's
// particle stands in for any of its members), so a head reference colliding by
// name with a different-typed local element is rejected. The check also runs
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
		c.checkContentModelConsistent(ctx, ct.td.ContentModel, ct.src.line, c.complexTypeComponent(ct.td, ct.src))
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
	}
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
func (c *compiler) checkContentModelConsistent(ctx context.Context, mg *ModelGroup, line int, component string) {
	byName := c.collectModelGroupElements(mg)
	for _, qn := range sortedNames(byName) {
		decls := byName[qn]
		if len(decls) < 2 {
			continue
		}
		if !c.declsConsistent(decls) {
			msg := fmt.Sprintf("Two elements with the same name '%s' and namespace '%s', but different type definitions, appear in the content model.", qn.Local, qn.NS)
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, line, "complexType", component, msg), helium.ErrorLevelFatal))
			c.errorCount++
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
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(src.source, src.line, "group", component, msg), helium.ErrorLevelFatal))
			c.errorCount++
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
	}
	return true
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
// element term it also folds in the term's substitution-group MEMBERS (the
// declarations the term's particle implicitly stands in for), keyed by each
// member's own name, so a head reference colliding by name with a different-typed
// same-named element elsewhere in the model is detected. The visited set bounds
// shared/recursive model-group structures (e.g. a group referenced from multiple
// places resolves to a shared *ModelGroup).
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
			// Fold in the term's substitution-group members: a head's particle
			// implicitly contains its members, so a member declaration is
			// effectively present in this content model under its own name. An
			// abstract head is itself never present, but its members still are.
			for _, member := range c.schema.substGroups[term.Name] {
				byName[member.Name] = append(byName[member.Name], member)
			}
		case *ModelGroup:
			c.collectContentModelElements(term, byName, visited)
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
	seen := make(map[QName]struct{})
	for decl != nil {
		if decl.Type != nil {
			return decl.Type
		}
		head := decl.SubstitutionGroup
		if head == (QName{}) {
			break
		}
		if _, ok := seen[head]; ok {
			break
		}
		seen[head] = struct{}{}
		next, ok := c.schema.elements[head]
		if !ok {
			break
		}
		decl = next
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
// pointer. Two distinct named types with the same expanded QName (which can
// arise across import merges) are also treated as the same component. Distinct
// anonymous inline types are different components and therefore inconsistent.
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
