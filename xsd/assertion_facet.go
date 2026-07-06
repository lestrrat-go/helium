package xsd

import (
	"context"
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium/internal/lexicon"
	valuepkg "github.com/lestrrat-go/helium/internal/xsd/value"
	"github.com/lestrrat-go/helium/xpath3"
)

// buildValueSequence builds the XSD 1.1 $value binding for an assertion: the
// typed atomic value(s) of a (whitespace-normalized, already lexically valid)
// simple value. A list type yields a sequence of typed items; an atomic or
// union type yields a single typed atomic value. Typing uses the value's
// effective builtin primitive (via xpath3.CastFromString), resolving a union to
// its active member and a QName/NOTATION lexical to a namespace-qualified value;
// a value that cannot be re-cast (e.g. a type xpath3 does not model) falls back
// to xs:untypedAtomic, which still atomizes and casts correctly in comparisons.
func buildValueSequence(ctx context.Context, value string, valueNS map[string]string, td *TypeDef, vc *validationContext) xpath3.Sequence {
	// Resolve a union down to the value's ACTIVE member first, so list-vs-atomic
	// dispatch sees the real variety: a union whose active member is a LIST (e.g.
	// memberTypes="IntList xs:string" with value "1 2") must produce the list-item
	// sequence, not a single xs:untypedAtomic.
	effTD := td
	if td != nil && resolveVariety(td) == TypeVarietyUnion {
		var schema *Schema
		version := Version10
		allowLegacyGMonth := false
		if vc != nil {
			schema = vc.schema
			version = vc.version
			allowLegacyGMonth = vc.allowXSD10LegacyGMonthInstance
		}
		if active := fixedUnionActiveMember(ctx, value, valueNS, resolveUnionMembers(td), schema, version, allowLegacyGMonth); active != nil {
			effTD = active
		}
	}
	switch resolveVariety(effTD) {
	case TypeVarietyList:
		itemType := resolveItemType(effTD)
		var items xpath3.ItemSlice
		for _, item := range valuepkg.XSDFields(value) {
			items = append(items, typedAtomic(ctx, item, valueNS, itemType, vc))
		}
		return items
	default:
		return xpath3.ItemSlice{typedAtomic(ctx, value, valueNS, effTD, vc)}
	}
}

// typedAtomic builds the typed atomic value for a single (already whitespace-
// normalized) lexical value of type td, resolving a union type down to the
// member that actually accepts the value first so a union item is typed as its
// active member rather than falling back to xs:untypedAtomic. Active-member
// probing is SCHEMA-AWARE (vc.schema is threaded into fixedUnionActiveMember),
// so a member whose own assertion needs `castable as t:T` resolves the same way
// as the real validation path — otherwise the wrong member could be selected.
func typedAtomic(ctx context.Context, value string, valueNS map[string]string, td *TypeDef, vc *validationContext) xpath3.AtomicValue {
	if td != nil && resolveVariety(td) == TypeVarietyUnion {
		var schema *Schema
		version := Version10
		allowLegacyGMonth := false
		if vc != nil {
			schema = vc.schema
			version = vc.version
			allowLegacyGMonth = vc.allowXSD10LegacyGMonthInstance
		}
		td = fixedUnionActiveMember(ctx, value, valueNS, resolveUnionMembers(td), schema, version, allowLegacyGMonth)
	}
	return atomicForType(value, valueNS, td)
}

// atomicForType casts a single lexical value to the typed atomic value of td's
// effective builtin primitive, falling back to xs:untypedAtomic when td is nil
// or the cast fails (the value already passed lexical validation, so a failure
// only means xpath3 does not model that exact type). QName/NOTATION lexicals are
// resolved against valueNS into an xpath3.QNameValue, since CastFromString has no
// namespace context. When td is a NAMED user-defined type, the user type name is
// PRESERVED as TypeName (with the builtin cast type kept as BaseType), mirroring
// AtomizeItem / schema-aware data() atomization, so $value and data() agree on a
// user atomic/QName/NOTATION/list-item/union-leaf type's identity (a value of
// t:MyInt is typed t:MyInt, not collapsed to xs:int).
func atomicForType(value string, valueNS map[string]string, td *TypeDef) xpath3.AtomicValue {
	local := ""
	if td != nil {
		local = builtinBaseLocal(td)
	}
	av, ok := builtinAtomicForType(value, valueNS, local)
	if !ok {
		return xpath3.AtomicValue{TypeName: xpath3.TypeUntypedAtomic, Value: value}
	}
	if td != nil {
		if name := xsdTypeName(td); !xpath3.IsKnownXSDType(name) {
			av.BaseType = av.TypeName
			av.TypeName = name
		}
	}
	return av
}

// builtinAtomicForType casts value to the atomic value of the builtin primitive
// named by local (no user-type identity), returning ok=false when local is empty
// or the cast fails. QName/NOTATION lexicals are resolved against valueNS.
func builtinAtomicForType(value string, valueNS map[string]string, local string) (xpath3.AtomicValue, bool) {
	switch local {
	case lexicon.TypeQName, lexicon.TypeNotation:
		if qn, err := resolveLexicalQName(value, valueNS); err == nil {
			prefix := ""
			if p, _, found := strings.Cut(value, ":"); found {
				prefix = p
			}
			typeName := xpath3.TypeQName
			if local == lexicon.TypeNotation {
				typeName = xpath3.TypeNOTATION
			}
			return xpath3.AtomicValue{
				TypeName: typeName,
				Value:    xpath3.QNameValue{Prefix: prefix, Local: qn.Local, URI: qn.NS},
			}, true
		}
	}
	if local != "" {
		if av, err := xpath3.CastFromString(value, "xs:"+local); err == nil {
			return av, true
		}
	}
	return xpath3.AtomicValue{}, false
}

// checkSimpleTypeAssertions enforces the XSD 1.1 <xs:assertion> facets declared
// along td's restriction chain against an already whitespace-normalized,
// lexically valid simple value. $value is bound to the typed value. Per XSD 1.1,
// an assertion-facet test is evaluated with NO context item (the focus is
// absent), so an expression using ".", position() or last() raises a dynamic
// error. An assertion is satisfied only if its effective boolean value is true;
// a dynamic evaluation error (an absent focus, or a failed cast inside the test)
// makes it unsatisfied.
func checkSimpleTypeAssertions(ctx context.Context, value string, valueNS map[string]string, td *TypeDef, elemName, filename string, line int, vc *validationContext) error {
	var hasAssertion bool
	for cur := range baseChain(td) {
		if cur.Facets != nil && len(cur.Facets.Assertions) > 0 {
			hasAssertion = true
			break
		}
	}
	if !hasAssertion {
		return nil
	}

	valueSeq := buildValueSequence(ctx, value, valueNS, td, vc)
	decls := vc.assertSchemaDecls()

	var firstErr error
	for cur := range baseChain(td) {
		if cur.Facets == nil {
			continue
		}
		for _, a := range cur.Facets.Assertions {
			if a.compiled == nil {
				continue
			}
			ev := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
				Namespaces(a.Namespaces).
				Variables(map[string]xpath3.Sequence{"value": valueSeq}).
				QNameValueNoDefaultNamespace()
			if decls != nil {
				ev = ev.SchemaDeclarations(decls)
			}
			res, err := ev.Evaluate(ctx, a.compiled, nil)
			ok := false
			if err == nil {
				ok, err = xpath3.EBV(res.Sequence())
			}
			if err != nil {
				vc.reportValidityError(ctx, filename, line, elemName,
					fmt.Sprintf("Failed to evaluate the assertion '%s': %v.", a.Test, err))
				if firstErr == nil {
					firstErr = fmt.Errorf("assertion evaluation failed")
				}
				continue
			}
			if !ok {
				vc.reportValidityError(ctx, filename, line, elemName,
					fmt.Sprintf("The assertion '%s' is not satisfied.", a.Test))
				if firstErr == nil {
					firstErr = fmt.Errorf("assertion not satisfied")
				}
			}
		}
	}
	return firstErr
}
