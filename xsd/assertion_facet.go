package xsd

import (
	"context"
	"fmt"

	valuepkg "github.com/lestrrat-go/helium/internal/xsd/value"
	"github.com/lestrrat-go/helium/xpath3"
)

// buildValueSequence builds the XSD 1.1 $value binding for an assertion: the
// typed atomic value(s) of a (whitespace-normalized, already lexically valid)
// simple value. A list type yields a sequence of typed items; an atomic or
// union type yields a single typed atomic value. Typing uses the value's
// effective builtin primitive (via xpath3.CastFromString); a value that cannot
// be re-cast (e.g. a type xpath3 does not model) falls back to xs:untypedAtomic,
// which still atomizes and casts correctly in comparisons.
func buildValueSequence(ctx context.Context, value string, valueNS map[string]string, td *TypeDef, version Version) xpath3.Sequence {
	switch resolveVariety(td) {
	case TypeVarietyList:
		itemType := resolveItemType(td)
		var items xpath3.ItemSlice
		for _, item := range valuepkg.XSDFields(value) {
			items = append(items, atomicForType(item, itemType))
		}
		return items
	case TypeVarietyUnion:
		member := fixedUnionActiveMember(ctx, value, valueNS, resolveUnionMembers(td), version)
		return xpath3.ItemSlice{atomicForType(value, member)}
	default:
		return xpath3.ItemSlice{atomicForType(value, td)}
	}
}

// atomicForType casts a single lexical value to the typed atomic value of td's
// effective builtin primitive, falling back to xs:untypedAtomic when td is nil
// or the cast fails (the value already passed lexical validation, so a failure
// only means xpath3 does not model that exact type).
func atomicForType(value string, td *TypeDef) xpath3.AtomicValue {
	local := ""
	if td != nil {
		local = builtinBaseLocal(td)
	}
	if local != "" {
		if av, err := xpath3.CastFromString(value, "xs:"+local); err == nil {
			return av
		}
	}
	return xpath3.AtomicValue{TypeName: xpath3.TypeUntypedAtomic, Value: value}
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

	valueSeq := buildValueSequence(ctx, value, valueNS, td, vc.version)

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
				Variables(map[string]xpath3.Sequence{"value": valueSeq})
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
