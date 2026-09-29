package xpath1

// This file holds the compile-time rewrite that Compiler.Compile applies to a
// freshly parsed AST. Parse itself returns the unrewritten AST.
//
// The abbreviation `//` expands to a descendant-or-self::node() step, so
// `//x` evaluates as two steps: every node of the subtree, then the x children
// of each of them. Like libxml2's xmlXPathOptimizeExpression, the rewrite
// folds a bare descendant-or-self::node() step into the step that follows it:
//
//	descendant-or-self::node()/child::N              → descendant::N (N a name test)
//	descendant-or-self::node()/self::X               → descendant-or-self::X
//	descendant-or-self::node()/descendant::X         → descendant::X
//	descendant-or-self::node()/descendant-or-self::X → descendant-or-self::X
//
// Each pair selects the same node set: every descendant of the context node is
// the child of exactly one node on its descendant-or-self axis, and the
// descendant-or-self axis of the context node is closed under taking
// descendants. Every step result is sorted into document order, so the result
// order is the same as well.
//
// Predicates on the second step are allowed only when positionFree proves they
// cannot observe the context position or size: after the rewrite the step's
// candidate list is the whole subtree instead of one parent's children, so
// position() and last() would count differently. Every other part of the
// predicate's dynamic context (the context node, variables, namespaces,
// functions) is the same either way. The first step must carry no predicate.
//
// The rewritten path charges fewer operations against Evaluator.OpLimit (one
// walk of the subtree instead of two) and never builds the intermediate
// descendant-or-self::node() node-set, so an expression that exceeds the op
// limit or the node-set limit only through that intermediate succeeds after
// the rewrite. Results and errors are otherwise unchanged.

// staticType is the XPath 1.0 result type an expression is known to produce
// before evaluation.
type staticType int

const (
	staticUnknown staticType = iota
	staticBoolean
	staticNumber
	staticString
	staticNodeSet
)

// builtinResultTypes maps every builtin function name to its result type. It
// must list exactly the names in builtinFunctions.
var builtinResultTypes = map[string]staticType{
	"last":             staticNumber,
	"position":         staticNumber,
	"count":            staticNumber,
	"string-length":    staticNumber,
	"number":           staticNumber,
	"sum":              staticNumber,
	"floor":            staticNumber,
	"ceiling":          staticNumber,
	"round":            staticNumber,
	"string":           staticString,
	"concat":           staticString,
	"substring":        staticString,
	"substring-before": staticString,
	"substring-after":  staticString,
	"normalize-space":  staticString,
	"translate":        staticString,
	"local-name":       staticString,
	"namespace-uri":    staticString,
	"name":             staticString,
	"boolean":          staticBoolean,
	"not":              staticBoolean,
	"true":             staticBoolean,
	"false":            staticBoolean,
	"contains":         staticBoolean,
	"starts-with":      staticBoolean,
	"lang":             staticBoolean,
	"id":               staticNodeSet,
}

// optimizeExpr applies the descendant-or-self collapse to every location path
// reachable from e and returns the rewritten expression. It rewrites
// *LocationPath values in place, so e must be an AST no one else holds.
// Beyond maxRecursionDepth it returns the rest of the tree unchanged.
func optimizeExpr(e Expr, depth int) Expr {
	if depth > maxRecursionDepth {
		return e
	}
	depth++
	switch v := e.(type) {
	case *LocationPath:
		optimizeLocationPath(v, depth)
		return v
	case BinaryExpr:
		v.Left = optimizeExpr(v.Left, depth)
		v.Right = optimizeExpr(v.Right, depth)
		return v
	case UnaryExpr:
		v.Operand = optimizeExpr(v.Operand, depth)
		return v
	case FunctionCall:
		for i, arg := range v.Args {
			v.Args[i] = optimizeExpr(arg, depth)
		}
		return v
	case FilterExpr:
		v.Expr = optimizeExpr(v.Expr, depth)
		for i, pred := range v.Predicates {
			v.Predicates[i] = optimizeExpr(pred, depth)
		}
		return v
	case UnionExpr:
		v.Left = optimizeExpr(v.Left, depth)
		v.Right = optimizeExpr(v.Right, depth)
		return v
	case PathExpr:
		v.Filter = optimizeExpr(v.Filter, depth)
		if v.Path != nil {
			optimizeLocationPath(v.Path, depth)
		}
		return v
	}
	return e
}

// optimizeLocationPath rewrites the predicates of every step of lp, then
// collapses its descendant-or-self::node() steps.
func optimizeLocationPath(lp *LocationPath, depth int) {
	for i := range lp.Steps {
		preds := lp.Steps[i].Predicates
		for j, pred := range preds {
			preds[j] = optimizeExpr(pred, depth)
		}
	}
	lp.Steps = collapseDescendantSteps(lp.Steps, depth)
}

// collapseDescendantSteps folds each predicate-free descendant-or-self::node()
// step into the step after it when collapsedAxis allows it, compacting steps
// in place. A folded pair is re-examined against the next step, so `//x//y`,
// `//.//x`, and `.//x` (self::node()/descendant::x) all collapse fully.
func collapseDescendantSteps(steps []Step, depth int) []Step {
	out := steps[:0]
	for _, s := range steps {
		n := len(out)
		if n > 0 && isBareDescendantOrSelfNode(out[n-1]) {
			if axis, ok := collapsedAxis(s.Axis, s.NodeTest); ok && predicatesPositionFree(s.Predicates, depth) {
				out[n-1] = Step{Axis: axis, NodeTest: s.NodeTest, Predicates: s.Predicates}
				continue
			}
		}
		out = append(out, s)
	}
	return out
}

// isBareDescendantOrSelfNode reports whether s is descendant-or-self::node()
// with no predicate, the step the `//` abbreviation expands to.
func isBareDescendantOrSelfNode(s Step) bool {
	if s.Axis != AxisDescendantOrSelf || len(s.Predicates) != 0 {
		return false
	}
	tt, ok := s.NodeTest.(TypeTest)
	return ok && tt.Type == NodeTestNode
}

// collapsedAxis returns the axis that replaces descendant-or-self::node()
// followed by a step on axis with node test nt, and false when the pair
// cannot be folded.
//
// A child step folds only when nt is a name test. When the context node is an
// entity reference, the descendant walk (collectDescendants in
// internal/xpath) follows the entity's sibling links into the DTD and reaches
// the comments and PIs declared before the entity, while the child axis stops
// at the entity. Both pairs would then disagree for node(), comment() and
// processing-instruction() tests. A name test on these axes matches only
// elements, which a parsed DTD never holds, so it selects the same nodes
// either way. (DTD.AddChild accepts an element, and a DTD built that way
// with an element before the entity declaration would make `.//x` from the
// entity reference differ.)
// The descendant and descendant-or-self pairs make the same walk before and
// after the fold, so any node test folds there.
func collapsedAxis(axis AxisType, nt NodeTest) (AxisType, bool) {
	switch axis {
	case AxisChild:
		if _, ok := nt.(NameTest); !ok {
			return axis, false
		}
		return AxisDescendant, true
	case AxisDescendant:
		return AxisDescendant, true
	case AxisSelf, AxisDescendantOrSelf:
		return AxisDescendantOrSelf, true
	}
	return axis, false
}

func predicatesPositionFree(preds []Expr, depth int) bool {
	for _, p := range preds {
		if !positionFree(p, depth) {
			return false
		}
	}
	return true
}

// positionFree reports whether predicate p provably selects the same nodes
// whatever its context position and size are. It is a whitelist: p must have
// a static boolean, string or node-set type (a number is compared against
// position()), and every part of p must pass positionFreeTree.
func positionFree(p Expr, depth int) bool {
	switch staticResultType(p) {
	case staticBoolean, staticString, staticNodeSet:
		return positionFreeTree(p, depth)
	}
	return false
}

// positionFreeTree reports whether no part of e can read the context position
// or size. It rejects position() and last(), variable references, prefixed
// function calls, and unprefixed names that are not builtins (a custom
// function can read FunctionContext.Position). It checks nested predicates
// with positionFree, so `a[b[1]]` is rejected even though the inner
// predicate has its own context. Unknown expression kinds are rejected.
func positionFreeTree(e Expr, depth int) bool {
	if depth > maxRecursionDepth {
		return false
	}
	depth++
	switch v := e.(type) {
	case LiteralExpr, NumberExpr:
		return true
	case *LocationPath:
		return locationPathPositionFree(v, depth)
	case BinaryExpr:
		return positionFreeTree(v.Left, depth) && positionFreeTree(v.Right, depth)
	case UnaryExpr:
		return positionFreeTree(v.Operand, depth)
	case UnionExpr:
		return positionFreeTree(v.Left, depth) && positionFreeTree(v.Right, depth)
	case FunctionCall:
		return functionCallPositionFree(v, depth)
	case FilterExpr:
		return positionFreeTree(v.Expr, depth) && predicatesPositionFree(v.Predicates, depth)
	case PathExpr:
		if !positionFreeTree(v.Filter, depth) {
			return false
		}
		return v.Path == nil || locationPathPositionFree(v.Path, depth)
	}
	return false
}

func locationPathPositionFree(lp *LocationPath, depth int) bool {
	for _, s := range lp.Steps {
		if !predicatesPositionFree(s.Predicates, depth) {
			return false
		}
	}
	return true
}

// functionCallPositionFree accepts a call only when it resolves to a builtin
// other than position() and last() and all of its arguments are position
// free. evalFunctionCall resolves an unprefixed builtin name before any
// custom function, so such a name always means the builtin.
func functionCallPositionFree(fc FunctionCall, depth int) bool {
	if fc.Prefix != "" || fc.Name == "position" || fc.Name == "last" {
		return false
	}
	if _, ok := builtinResultTypes[fc.Name]; !ok {
		return false
	}
	for _, arg := range fc.Args {
		if !positionFreeTree(arg, depth) {
			return false
		}
	}
	return true
}

// staticResultType returns the result type e produces when it evaluates
// without error, or staticUnknown when it cannot be known before evaluation.
func staticResultType(e Expr) staticType {
	switch v := e.(type) {
	case *LocationPath, FilterExpr, UnionExpr, PathExpr:
		return staticNodeSet
	case LiteralExpr:
		return staticString
	case NumberExpr, UnaryExpr:
		return staticNumber
	case BinaryExpr:
		return binaryResultType(v.Op)
	case FunctionCall:
		if v.Prefix != "" {
			return staticUnknown
		}
		return builtinResultTypes[v.Name]
	}
	return staticUnknown
}

func binaryResultType(op TokenType) staticType {
	switch op {
	case TokenOr, TokenAnd, TokenEquals, TokenNotEquals, TokenLess, TokenLessEq, TokenGreater, TokenGreaterEq:
		return staticBoolean
	case TokenPlus, TokenMinus, TokenStar, TokenDiv, TokenMod:
		return staticNumber
	}
	return staticUnknown
}
