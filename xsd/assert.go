package xsd

import (
	"context"
	"fmt"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

// parseAssert reads an XSD 1.1 <xs:assert> element, capturing its @test
// expression and the in-scope namespace bindings, and pre-compiling the test as
// an XPath 3.1 expression. A missing @test or a malformed XPath is a fatal
// schema error (it returns nil), mirroring how a malformed identity-constraint
// XPath is treated — silently dropping the assertion would let an invalid schema
// validate documents as if the constraint were absent.
//
// Callers must only invoke this in XSD 1.1 mode; xs:assert is not part of the
// 1.0 grammar.
func (c *compiler) parseAssert(ctx context.Context, elem *helium.Element) *Assertion {
	if !hasAttr(elem, attrTest) {
		if c.filename != "" {
			c.schemaError(ctx, schemaParserError(c.diagSource(), elem.Line(), elem.LocalName(), elemAssert,
				"The attribute 'test' is required but missing."))
		}
		return nil
	}
	test := getAttr(elem, attrTest)
	a := &Assertion{
		Test:       test,
		Namespaces: collectNSContext(elem),
		Line:       elem.Line(),
		Source:     c.diagSource(),
	}
	compiled, err := xpath3.NewCompiler().Compile(test)
	if err != nil {
		if c.filename != "" {
			c.schemaError(ctx, schemaParserError(c.diagSource(), elem.Line(), elem.LocalName(), elemAssert,
				fmt.Sprintf("The XPath expression '%s' of the assertion is not valid: %v.", test, err)))
		}
		return nil
	}
	a.compiled = compiled
	return a
}

// checkAssertions evaluates the XSD 1.1 xs:assert constraints that apply to an
// element against its (already content-validated) instance, walking the type's
// base chain so an assertion inherited from a base type is enforced too. Each
// test is evaluated with the element as the context node; the element is invalid
// unless the test's effective boolean value is true.
//
// The in-scope namespaces captured at the xs:assert element are supplied to the
// evaluator; the standard prefixes (xs, fn, math, …) keep their default bindings
// (StrictPrefixes is intentionally NOT set) so common assertion idioms such as
// xs:integer(...) or string-length(...) work without redeclaration.
//
// Known limitation: the test is evaluated against the element as it sits in the
// full document, so an expression could navigate to ancestors/siblings, which a
// strict processor forbids (the assertion's context tree is the element and its
// descendants only). This does not affect assertions that stay within the
// element subtree.
func (vc *validationContext) checkAssertions(ctx context.Context, elem *helium.Element, td *TypeDef) error {
	var firstErr error
	for cur := td; cur != nil; cur = cur.BaseType {
		for _, a := range cur.Assertions {
			if a.compiled == nil {
				continue
			}
			ev := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Namespaces(a.Namespaces)
			res, err := ev.Evaluate(ctx, a.compiled, elem)
			if err != nil {
				vc.reportValidityError(ctx, vc.filename, elem.Line(), elemDisplayName(elem),
					fmt.Sprintf("Failed to evaluate the assertion '%s': %v.", a.Test, err))
				if firstErr == nil {
					firstErr = fmt.Errorf("assertion evaluation failed")
				}
				continue
			}
			ok, err := xpath3.EBV(res.Sequence())
			if err != nil {
				vc.reportValidityError(ctx, vc.filename, elem.Line(), elemDisplayName(elem),
					fmt.Sprintf("Failed to evaluate the assertion '%s': %v.", a.Test, err))
				if firstErr == nil {
					firstErr = fmt.Errorf("assertion evaluation failed")
				}
				continue
			}
			if !ok {
				vc.reportValidityError(ctx, vc.filename, elem.Line(), elemDisplayName(elem),
					fmt.Sprintf("The assertion '%s' is not satisfied.", a.Test))
				if firstErr == nil {
					firstErr = fmt.Errorf("assertion not satisfied")
				}
			}
		}
	}
	return firstErr
}
