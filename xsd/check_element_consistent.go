package xsd

import (
	"context"
	"fmt"
	"sort"

	helium "github.com/lestrrat-go/helium"
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
// This runs after reference resolution (so every element particle's Type and
// every group reference's Particles are populated) and only when no prior
// schema error has been reported, matching the gating of checkUPA.
func (c *compiler) checkElementConsistent(ctx context.Context) {
	if c.filename == "" || c.errorCount != 0 {
		return
	}

	// Order by source line then ordinal so diagnostics are deterministic
	// regardless of Go map iteration order.
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
		c.checkModelGroupElementConsistent(ctx, ct.td, ct.src)
	}
}

// checkModelGroupElementConsistent collects every element declaration reachable
// in td's effective content model, grouped by expanded name, and reports the
// first inconsistent pair found per name.
func (c *compiler) checkModelGroupElementConsistent(ctx context.Context, td *TypeDef, src typeDefSource) {
	byName := make(map[QName][]*ElementDecl)
	collectContentModelElements(td.ContentModel, byName, make(map[*ModelGroup]struct{}))

	// Iterate names in a deterministic order so the reported error is stable.
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

	for _, qn := range names {
		decls := byName[qn]
		if len(decls) < 2 {
			continue
		}
		first := decls[0]
		for _, other := range decls[1:] {
			if elementTypesConsistent(first.Type, other.Type) {
				continue
			}
			component := componentLocalComplexType
			if !src.isLocal {
				component = td.Name.Local
			}
			msg := fmt.Sprintf("Two elements with the same name '%s' and namespace '%s', but different type definitions, appear in the content model.", qn.Local, qn.NS)
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.filename, src.line, "complexType", component, msg), helium.ErrorLevelFatal))
			c.errorCount++
			// One diagnostic per name is enough; move on to the next name.
			break
		}
	}
}

// collectContentModelElements walks a model group recursively, appending every
// element declaration term to byName keyed by its expanded name. The visited
// set bounds shared/recursive model-group structures (e.g. a group referenced
// from multiple places resolves to a shared *ModelGroup).
func collectContentModelElements(mg *ModelGroup, byName map[QName][]*ElementDecl, visited map[*ModelGroup]struct{}) {
	if mg == nil {
		return
	}
	if _, seen := visited[mg]; seen {
		return
	}
	visited[mg] = struct{}{}
	for _, p := range mg.Particles {
		switch term := p.Term.(type) {
		case *ElementDecl:
			byName[term.Name] = append(byName[term.Name], term)
		case *ModelGroup:
			collectContentModelElements(term, byName, visited)
		}
	}
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
