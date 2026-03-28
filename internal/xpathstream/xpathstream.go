// Package xpathstream provides streamability analysis helpers for XPath 3.1
// expressions. These were previously exported from the xpath3 package but are
// implementation details used only by xslt3 streaming analysis.
package xpathstream

import (
	"math/big"
	"slices"

	"github.com/lestrrat-go/helium/xpath3"
)

// WalkExpr walks an XPath 3.1 AST, calling fn for each Expr node.
// If fn returns false, children of that node are not visited.
func WalkExpr(expr xpath3.Expr, fn func(xpath3.Expr) bool) {
	if expr == nil {
		return
	}
	expr = derefExpr(expr)
	if !fn(expr) {
		return
	}
	walkChildren(expr, fn)
}

func derefExpr(expr xpath3.Expr) xpath3.Expr {
	switch e := expr.(type) {
	case *xpath3.LocationPath:
		if e == nil {
			return nil
		}
		return *e
	case *xpath3.BinaryExpr:
		if e == nil {
			return nil
		}
		return *e
	case *xpath3.FilterExpr:
		if e == nil {
			return nil
		}
		return *e
	case *xpath3.PathExpr:
		if e == nil {
			return nil
		}
		return *e
	case *xpath3.FunctionCall:
		if e == nil {
			return nil
		}
		return *e
	default:
		return expr
	}
}

func walkChildren(expr xpath3.Expr, fn func(xpath3.Expr) bool) {
	switch e := expr.(type) {
	case xpath3.LiteralExpr, xpath3.VariableExpr, xpath3.RootExpr, xpath3.ContextItemExpr, xpath3.PlaceholderExpr:
		// leaf nodes

	case xpath3.LocationPath:
		for _, step := range e.Steps {
			for _, pred := range step.Predicates {
				WalkExpr(pred, fn)
			}
		}

	case xpath3.BinaryExpr:
		WalkExpr(e.Left, fn)
		WalkExpr(e.Right, fn)

	case xpath3.UnaryExpr:
		WalkExpr(e.Operand, fn)

	case xpath3.ConcatExpr:
		WalkExpr(e.Left, fn)
		WalkExpr(e.Right, fn)

	case xpath3.SimpleMapExpr:
		WalkExpr(e.Left, fn)
		WalkExpr(e.Right, fn)

	case xpath3.RangeExpr:
		WalkExpr(e.Start, fn)
		WalkExpr(e.End, fn)

	case xpath3.UnionExpr:
		WalkExpr(e.Left, fn)
		WalkExpr(e.Right, fn)

	case xpath3.IntersectExceptExpr:
		WalkExpr(e.Left, fn)
		WalkExpr(e.Right, fn)

	case xpath3.FilterExpr:
		WalkExpr(e.Expr, fn)
		for _, pred := range e.Predicates {
			WalkExpr(pred, fn)
		}

	case xpath3.PathExpr:
		WalkExpr(e.Filter, fn)
		if e.Path != nil {
			WalkExpr(*e.Path, fn)
		}

	case xpath3.PathStepExpr:
		WalkExpr(e.Left, fn)
		WalkExpr(e.Right, fn)

	case xpath3.LookupExpr:
		WalkExpr(e.Expr, fn)
		WalkExpr(e.Key, fn)

	case xpath3.UnaryLookupExpr:
		WalkExpr(e.Key, fn)

	case xpath3.FLWORExpr:
		for _, clause := range e.Clauses {
			switch c := clause.(type) {
			case xpath3.ForClause:
				WalkExpr(c.Expr, fn)
			case xpath3.LetClause:
				WalkExpr(c.Expr, fn)
			}
		}
		WalkExpr(e.Return, fn)

	case xpath3.QuantifiedExpr:
		for _, b := range e.Bindings {
			WalkExpr(b.Domain, fn)
		}
		WalkExpr(e.Satisfies, fn)

	case xpath3.IfExpr:
		WalkExpr(e.Cond, fn)
		WalkExpr(e.Then, fn)
		WalkExpr(e.Else, fn)

	case xpath3.TryCatchExpr:
		WalkExpr(e.Try, fn)
		for _, cc := range e.Catches {
			WalkExpr(cc.Expr, fn)
		}

	case xpath3.InstanceOfExpr:
		WalkExpr(e.Expr, fn)

	case xpath3.CastExpr:
		WalkExpr(e.Expr, fn)

	case xpath3.CastableExpr:
		WalkExpr(e.Expr, fn)

	case xpath3.TreatAsExpr:
		WalkExpr(e.Expr, fn)

	case xpath3.FunctionCall:
		for _, arg := range e.Args {
			WalkExpr(arg, fn)
		}

	case xpath3.DynamicFunctionCall:
		WalkExpr(e.Func, fn)
		for _, arg := range e.Args {
			WalkExpr(arg, fn)
		}

	case xpath3.NamedFunctionRef:
		// leaf

	case xpath3.InlineFunctionExpr:
		WalkExpr(e.Body, fn)

	case xpath3.MapConstructorExpr:
		for _, pair := range e.Pairs {
			WalkExpr(pair.Key, fn)
			WalkExpr(pair.Value, fn)
		}

	case xpath3.ArrayConstructorExpr:
		for _, item := range e.Items {
			WalkExpr(item, fn)
		}

	case xpath3.SequenceExpr:
		for _, item := range e.Items {
			WalkExpr(item, fn)
		}
	}
}

// ExprUsesFunction returns true if the expression contains a call to the named function
// (with no namespace prefix, e.g. "last", "position").
func ExprUsesFunction(expr *xpath3.Expression, name string) bool {
	if expr == nil {
		return false
	}
	return expr.StreamInfo().UsedFunctions[name]
}

// ExprHasDownwardStep returns true if the expression contains any child::, descendant::,
// or descendant-or-self:: axis step.
func ExprHasDownwardStep(expr *xpath3.Expression) bool {
	if expr == nil {
		return false
	}
	return expr.StreamInfo().HasDownwardStep
}

// ExprUsesUpwardAxis returns true if the expression uses parent::, ancestor::,
// ancestor-or-self::, preceding::, or preceding-sibling:: axes.
func ExprUsesUpwardAxis(expr *xpath3.Expression) bool {
	if expr == nil {
		return false
	}
	const upwardMask = 1<<uint(xpath3.AxisParent) |
		1<<uint(xpath3.AxisAncestor) |
		1<<uint(xpath3.AxisAncestorOrSelf) |
		1<<uint(xpath3.AxisPreceding) |
		1<<uint(xpath3.AxisPrecedingSibling)
	return expr.StreamInfo().AxisUsed&upwardMask != 0
}

// ExprUsesPrecedingAxis returns true if the expression uses preceding:: or
// preceding-sibling:: axes.
func ExprUsesPrecedingAxis(expr *xpath3.Expression) bool {
	if expr == nil {
		return false
	}
	const precMask = 1<<uint(xpath3.AxisPreceding) | 1<<uint(xpath3.AxisPrecedingSibling)
	return expr.StreamInfo().AxisUsed&precMask != 0
}

// ExprUsesFollowingSiblingAxis returns true if the expression uses
// following-sibling:: axis.
func ExprUsesFollowingSiblingAxis(expr *xpath3.Expression) bool {
	if expr == nil {
		return false
	}
	return expr.StreamInfo().AxisUsed&(1<<uint(xpath3.AxisFollowingSibling)) != 0
}

// ExprUsesDescendantOrSelf returns true if the expression uses descendant:: or
// descendant-or-self:: axis.
func ExprUsesDescendantOrSelf(expr *xpath3.Expression) bool {
	if expr == nil {
		return false
	}
	si := expr.StreamInfo()
	const descMask = 1<<uint(xpath3.AxisDescendant) | 1<<uint(xpath3.AxisDescendantOrSelf)
	return si.AxisUsed&descMask != 0 || si.HasDescOrSelf
}

// CountDownwardSelections counts the number of independent downward selections
// in the expression.
func CountDownwardSelections(expr *xpath3.Expression) int {
	if expr == nil {
		return 0
	}
	return expr.StreamInfo().DownwardSelections
}

// ExprHasUpThenDownNavigation returns true if any location path in the
// expression first navigates upward (parent/ancestor) and then downward
// (child/descendant).
func ExprHasUpThenDownNavigation(expr *xpath3.Expression) bool {
	if expr == nil {
		return false
	}
	found := false
	WalkExpr(expr.AST(), func(e xpath3.Expr) bool {
		if found {
			return false
		}
		checkSteps := func(axes []xpath3.AxisType) {
			seenUp := false
			for _, axis := range axes {
				switch axis {
				case xpath3.AxisParent, xpath3.AxisAncestor, xpath3.AxisAncestorOrSelf:
					seenUp = true
				case xpath3.AxisChild, xpath3.AxisDescendant, xpath3.AxisDescendantOrSelf:
					if seenUp {
						found = true
						return
					}
				}
			}
		}
		switch v := e.(type) {
		case xpath3.LocationPath:
			axes := make([]xpath3.AxisType, len(v.Steps))
			for i, s := range v.Steps {
				axes[i] = s.Axis
			}
			checkSteps(axes)
		}
		return true
	})
	return found
}

// ExprUsesContextItem returns true if the expression references the context
// item (.) at any level.
func ExprUsesContextItem(expr *xpath3.Expression) bool {
	if expr == nil {
		return false
	}
	found := false
	WalkExpr(expr.AST(), func(e xpath3.Expr) bool {
		if found {
			return false
		}
		switch e.(type) {
		case xpath3.ContextItemExpr:
			found = true
			return false
		}
		return true
	})
	return found
}

// PredicateIsNonMotionless returns true if a predicate expression navigates
// downward (uses child/descendant axes), uses last(), or uses position() in
// a non-trivial way.
func PredicateIsNonMotionless(pred xpath3.Expr) bool {
	return predicateIsNonMotionlessWithStep(pred, nil)
}

// PredicateIsNonMotionlessWithStep checks whether a predicate is non-motionless,
// taking into account the step the predicate is attached to.
func PredicateIsNonMotionlessWithStep(pred xpath3.Expr, step *xpath3.Step) bool {
	return predicateIsNonMotionlessWithStep(pred, step)
}

func predicateIsNonMotionlessWithStep(pred xpath3.Expr, step *xpath3.Step) bool {
	nonMotionless := false
	WalkExpr(pred, func(e xpath3.Expr) bool {
		if nonMotionless {
			return false
		}
		switch v := e.(type) {
		case xpath3.LocationPath:
			for _, step := range v.Steps {
				switch step.Axis {
				case xpath3.AxisChild, xpath3.AxisDescendant, xpath3.AxisDescendantOrSelf:
					nonMotionless = true
					return false
				}
			}
		case xpath3.PathStepExpr:
			if v.DescOrSelf {
				nonMotionless = true
				return false
			}
		case xpath3.FunctionCall:
			if v.Prefix == "" && (v.Name == "last" || v.Name == "position") {
				nonMotionless = true
				return false
			}
			if v.Prefix == "" {
				switch v.Name {
				case "name", "local-name", "namespace-uri", "node-name",
					"self", "generate-id", "base-uri", "document-uri",
					"nilled", "has-children", "string-length", "current":
					return false
				}
			}
		case xpath3.LiteralExpr:
			if e == pred {
				switch v.Value.(type) {
				case float64, *big.Int, *big.Rat:
					nonMotionless = true
					return false
				}
			}
		case xpath3.InstanceOfExpr, xpath3.CastableExpr:
			return false
		case xpath3.ContextItemExpr:
			if step != nil && StepSelectsAtomicNodes(step) {
				return false
			}
			nonMotionless = true
			return false
		}
		return true
	})
	return nonMotionless
}

// ExprTreeHasNonMotionlessPredicate walks an entire AST tree and returns true
// if any step or filter within it contains a non-motionless predicate.
func ExprTreeHasNonMotionlessPredicate(expr xpath3.Expr) bool {
	found := false
	WalkExpr(expr, func(e xpath3.Expr) bool {
		if found {
			return false
		}
		switch v := e.(type) {
		case xpath3.LocationPath:
			for i := range v.Steps {
				step := &v.Steps[i]
				for _, pred := range step.Predicates {
					if predicateIsNonMotionlessWithStep(pred, step) {
						found = true
						return false
					}
				}
			}
		case xpath3.FilterExpr:
			if slices.ContainsFunc(v.Predicates, PredicateIsNonMotionless) {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

// StepSelectsAtomicNodes returns true if the step selects nodes whose string
// value is available without descending into children.
func StepSelectsAtomicNodes(step *xpath3.Step) bool {
	if step.Axis == xpath3.AxisAttribute {
		return true
	}
	switch v := step.NodeTest.(type) {
	case xpath3.TypeTest:
		switch v.Kind {
		case xpath3.NodeKindText, xpath3.NodeKindComment, xpath3.NodeKindProcessingInstruction:
			return true
		}
	case xpath3.AttributeTest:
		return true
	case xpath3.NamespaceNodeTest:
		return true
	case xpath3.PITest:
		return true
	}
	return false
}
