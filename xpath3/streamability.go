package xpath3

import "math/big"

// AST returns the root AST node of the compiled expression.
// This is used by streamability analysis in xslt3.
func (e *Expression) AST() Expr {
	return e.astExpr()
}

// computeStreamInfo performs a single AST walk to precompute all
// streamability properties stored on vmProgram.stream.
func computeStreamInfo(ast Expr) streamInfo {
	si := streamInfo{
		usedFunctions: make(map[string]bool),
	}
	switch v := derefExpr(ast).(type) {
	case ContextItemExpr:
		si.isContextItem = true
	case vmLocationPathExpr:
		// The direct compiler lowers "." to self::node(); recognize it.
		if !v.Absolute && len(v.Steps) == 1 {
			s := v.Steps[0]
			if s.Axis == AxisSelf && len(s.Predicates) == 0 {
				if tt, ok := s.NodeTest.(TypeTest); ok && tt.Kind == NodeKindNode {
					si.isContextItem = true
				}
			}
		}
	}
	// Collect axis usage, function names, downward steps, and predicate motionlessness.
	WalkExpr(ast, func(e Expr) bool {
		switch v := e.(type) {
		case LocationPath:
			for _, step := range v.Steps {
				si.axisUsed |= 1 << uint(step.Axis)
				switch step.Axis {
				case AxisChild, AxisDescendant, AxisDescendantOrSelf:
					si.hasDownwardStep = true
				}
				for _, pred := range step.Predicates {
					if predicateIsNonMotionless(pred) {
						si.hasNonMotionlessPred = true
					}
				}
			}
		case vmLocationPathExpr:
			for _, step := range v.Steps {
				si.axisUsed |= 1 << uint(step.Axis)
				switch step.Axis {
				case AxisChild, AxisDescendant, AxisDescendantOrSelf:
					si.hasDownwardStep = true
				}
				for _, pred := range step.Predicates {
					if predicateIsNonMotionless(pred) {
						si.hasNonMotionlessPred = true
					}
				}
			}
		case PathStepExpr:
			if v.DescOrSelf {
				si.hasDownwardStep = true
				si.hasDescOrSelf = true
			}
		case FilterExpr:
			for _, pred := range v.Predicates {
				if predicateIsNonMotionless(pred) {
					si.hasNonMotionlessPred = true
				}
			}
		case FunctionCall:
			if v.Prefix == "" {
				si.usedFunctions[v.Name] = true
			}
		}
		return true
	})
	si.downwardSelections = countDownwardSelectionsInExpr(ast)
	return si
}

// WalkExpr walks an XPath 3.1 AST, calling fn for each Expr node.
// If fn returns false, children of that node are not visited.
func WalkExpr(expr Expr, fn func(Expr) bool) {
	if expr == nil {
		return
	}
	// Dereference pointer types to value types so type switches work uniformly.
	expr = derefExpr(expr)
	if !fn(expr) {
		return
	}
	walkChildren(expr, fn)
}

// derefExpr converts pointer Expr types to their value equivalents.
func derefExpr(expr Expr) Expr {
	switch e := expr.(type) {
	case *LocationPath:
		if e == nil {
			return nil
		}
		return *e
	case *BinaryExpr:
		if e == nil {
			return nil
		}
		return *e
	case *FilterExpr:
		if e == nil {
			return nil
		}
		return *e
	case *PathExpr:
		if e == nil {
			return nil
		}
		return *e
	case *FunctionCall:
		if e == nil {
			return nil
		}
		return *e
	default:
		return expr
	}
}

func walkChildren(expr Expr, fn func(Expr) bool) {
	switch e := expr.(type) {
	case LiteralExpr, VariableExpr, RootExpr, ContextItemExpr, PlaceholderExpr:
		// leaf nodes

	case LocationPath:
		for _, step := range e.Steps {
			for _, pred := range step.Predicates {
				WalkExpr(pred, fn)
			}
		}

	case vmLocationPathExpr:
		for _, step := range e.Steps {
			for _, pred := range step.Predicates {
				WalkExpr(pred, fn)
			}
		}

	case BinaryExpr:
		WalkExpr(e.Left, fn)
		WalkExpr(e.Right, fn)

	case UnaryExpr:
		WalkExpr(e.Operand, fn)

	case ConcatExpr:
		WalkExpr(e.Left, fn)
		WalkExpr(e.Right, fn)

	case SimpleMapExpr:
		WalkExpr(e.Left, fn)
		WalkExpr(e.Right, fn)

	case RangeExpr:
		WalkExpr(e.Start, fn)
		WalkExpr(e.End, fn)

	case UnionExpr:
		WalkExpr(e.Left, fn)
		WalkExpr(e.Right, fn)

	case IntersectExceptExpr:
		WalkExpr(e.Left, fn)
		WalkExpr(e.Right, fn)

	case FilterExpr:
		WalkExpr(e.Expr, fn)
		for _, pred := range e.Predicates {
			WalkExpr(pred, fn)
		}

	case PathExpr:
		WalkExpr(e.Filter, fn)
		if e.Path != nil {
			WalkExpr(*e.Path, fn)
		}

	case vmPathExpr:
		WalkExpr(e.Filter, fn)
		if e.Path != nil {
			WalkExpr(*e.Path, fn)
		}

	case PathStepExpr:
		WalkExpr(e.Left, fn)
		WalkExpr(e.Right, fn)

	case LookupExpr:
		WalkExpr(e.Expr, fn)
		WalkExpr(e.Key, fn)

	case UnaryLookupExpr:
		WalkExpr(e.Key, fn)

	case FLWORExpr:
		for _, clause := range e.Clauses {
			switch c := clause.(type) {
			case ForClause:
				WalkExpr(c.Expr, fn)
			case LetClause:
				WalkExpr(c.Expr, fn)
			}
		}
		WalkExpr(e.Return, fn)

	case QuantifiedExpr:
		for _, b := range e.Bindings {
			WalkExpr(b.Domain, fn)
		}
		WalkExpr(e.Satisfies, fn)

	case IfExpr:
		WalkExpr(e.Cond, fn)
		WalkExpr(e.Then, fn)
		WalkExpr(e.Else, fn)

	case TryCatchExpr:
		WalkExpr(e.Try, fn)
		for _, cc := range e.Catches {
			WalkExpr(cc.Expr, fn)
		}

	case InstanceOfExpr:
		WalkExpr(e.Expr, fn)

	case CastExpr:
		WalkExpr(e.Expr, fn)

	case CastableExpr:
		WalkExpr(e.Expr, fn)

	case TreatAsExpr:
		WalkExpr(e.Expr, fn)

	case FunctionCall:
		for _, arg := range e.Args {
			WalkExpr(arg, fn)
		}

	case DynamicFunctionCall:
		WalkExpr(e.Func, fn)
		for _, arg := range e.Args {
			WalkExpr(arg, fn)
		}

	case NamedFunctionRef:
		// leaf

	case InlineFunctionExpr:
		WalkExpr(e.Body, fn)

	case MapConstructorExpr:
		for _, pair := range e.Pairs {
			WalkExpr(pair.Key, fn)
			WalkExpr(pair.Value, fn)
		}

	case ArrayConstructorExpr:
		for _, item := range e.Items {
			WalkExpr(item, fn)
		}

	case SequenceExpr:
		for _, item := range e.Items {
			WalkExpr(item, fn)
		}
	}
}

// ExprUsesAxis returns true if the expression contains a step using the given axis.
func ExprUsesAxis(expr *Expression, axis AxisType) bool {
	if expr == nil {
		return false
	}
	if expr.program != nil {
		return expr.program.stream.axisUsed&(1<<uint(axis)) != 0
	}
	return false
}

// ExprUsesFunction returns true if the expression contains a call to the named function
// (with no namespace prefix, e.g. "last", "position").
func ExprUsesFunction(expr *Expression, name string) bool {
	if expr == nil {
		return false
	}
	if expr.program != nil {
		return expr.program.stream.usedFunctions[name]
	}
	return false
}

// ExprHasDownwardStep returns true if the expression contains any child::, descendant::,
// or descendant-or-self:: axis step (indicating downward navigation into the streamed document).
func ExprHasDownwardStep(expr *Expression) bool {
	if expr == nil {
		return false
	}
	if expr.program != nil {
		return expr.program.stream.hasDownwardStep
	}
	return false
}

// ExprHasNonMotionlessPredicate returns true if any step in the expression has a
// predicate that itself navigates downward (child/descendant steps, or uses
// context-dependent functions like last()).
func ExprHasNonMotionlessPredicate(expr *Expression) bool {
	if expr == nil {
		return false
	}
	if expr.program != nil {
		return expr.program.stream.hasNonMotionlessPred
	}
	return false
}

// predicateIsNonMotionless returns true if a predicate expression navigates
// downward (uses child/descendant axes), uses last(), or uses position() in
// a non-trivial way.
func predicateIsNonMotionless(pred Expr) bool {
	return predicateIsNonMotionlessWithStep(pred, nil)
}

// predicateIsNonMotionlessWithStep checks whether a predicate is non-motionless,
// optionally taking into account the step the predicate is attached to. When the
// step selects atomic-value nodes (text(), attribute, comment(), PI), "." in the
// predicate is motionless because the value is available without child navigation.
func predicateIsNonMotionlessWithStep(pred Expr, step *Step) bool {
	nonMotionless := false
	WalkExpr(pred, func(e Expr) bool {
		if nonMotionless {
			return false
		}
		switch v := e.(type) {
		case LocationPath:
			for _, step := range v.Steps {
				switch step.Axis {
				case AxisChild, AxisDescendant, AxisDescendantOrSelf:
					nonMotionless = true
					return false
				}
			}
		case vmLocationPathExpr:
			for _, step := range v.Steps {
				switch step.Axis {
				case AxisChild, AxisDescendant, AxisDescendantOrSelf:
					nonMotionless = true
					return false
				}
			}
		case PathStepExpr:
			if v.DescOrSelf {
				nonMotionless = true
				return false
			}
		case FunctionCall:
			if v.Prefix == "" && (v.Name == "last" || v.Name == "position") {
				nonMotionless = true
				return false
			}
			// Property-access functions (name, local-name, node-name, etc.)
			// only inspect node properties without consuming children.
			// current() in match patterns returns the node being matched,
			// which is motionless. Skip children so "." inside is not flagged.
			if v.Prefix == "" {
				switch v.Name {
				case "name", "local-name", "namespace-uri", "node-name",
					"self", "generate-id", "base-uri", "document-uri",
					"nilled", "has-children", "string-length", "current":
					return false
				}
			}
		case LiteralExpr:
			// A numeric literal predicate like [1] is positional (equivalent
			// to [position()=1]) and therefore non-motionless — but only when
			// it is the top-level predicate expression. When the numeric
			// literal appears inside a comparison (e.g., [f() eq 1]), it is
			// just a value operand and does not make the predicate positional.
			if e == pred {
				switch v.Value.(type) {
				case float64, *big.Int, *big.Rat:
					nonMotionless = true
					return false
				}
			}
		case InstanceOfExpr, CastableExpr:
			// Type-checking expressions (instance of, castable as) are
			// motionless — they inspect the dynamic type of the item
			// without consuming its value. Skip their children so that
			// "." inside them is not flagged.
			return false
		case ContextItemExpr:
			// "." in a predicate on a step that selects atomic-value nodes
			// (text(), attribute, comment, PI) is motionless — the value is
			// available without child navigation. On element/document steps,
			// atomizing "." requires reading descendant text, which is consuming.
			if step != nil && stepSelectsAtomicNodes(step) {
				return false
			}
			nonMotionless = true
			return false
		}
		return true
	})
	return nonMotionless
}

// stepSelectsAtomicNodes returns true if the step selects nodes whose string
// value is available without descending into children: text(), attribute,
// comment(), and processing-instruction() nodes.
func stepSelectsAtomicNodes(step *Step) bool {
	if step.Axis == AxisAttribute {
		return true
	}
	switch v := step.NodeTest.(type) {
	case TypeTest:
		switch v.Kind {
		case NodeKindText, NodeKindComment, NodeKindProcessingInstruction:
			return true
		}
	case AttributeTest:
		return true
	case NamespaceNodeTest:
		return true
	case PITest:
		return true
	}
	return false
}

// ExprTreeHasNonMotionlessPredicate walks an entire AST tree and returns true
// if any step or filter within it contains a non-motionless predicate. This is
// used to check match patterns in streaming modes.
func ExprTreeHasNonMotionlessPredicate(expr Expr) bool {
	found := false
	WalkExpr(expr, func(e Expr) bool {
		if found {
			return false
		}
		switch v := e.(type) {
		case LocationPath:
			for i := range v.Steps {
				step := &v.Steps[i]
				for _, pred := range step.Predicates {
					if predicateIsNonMotionlessWithStep(pred, step) {
						found = true
						return false
					}
				}
			}
		case FilterExpr:
			for _, pred := range v.Predicates {
				if predicateIsNonMotionless(pred) {
					found = true
					return false
				}
			}
		}
		return true
	})
	return found
}

// CountDownwardSelections counts the number of independent downward selections
// (child/descendant/descendant-or-self axis steps or //) in the expression.
// Multiple downward selections from a streaming node means consuming the stream
// multiple times, which is not allowed.
func CountDownwardSelections(expr *Expression) int {
	if expr == nil {
		return 0
	}
	if expr.program != nil {
		return expr.program.stream.downwardSelections
	}
	return 0
}

func countDownwardSelectionsInExpr(expr Expr) int {
	expr = derefExpr(expr)
	count := 0
	switch e := expr.(type) {
	case LocationPath:
		// A single location path with multiple downward steps is one selection
		// (e.g., /a/b/c is fine), but the path itself counts as one downward selection
		// if it has any downward step.
		hasDown := false
		for _, step := range e.Steps {
			switch step.Axis {
			case AxisChild, AxisDescendant, AxisDescendantOrSelf:
				hasDown = true
			}
			for _, pred := range step.Predicates {
				count += countDownwardSelectionsInExpr(pred)
			}
		}
		if hasDown {
			count++
		}
	case vmLocationPathExpr:
		hasDown := false
		for _, step := range e.Steps {
			switch step.Axis {
			case AxisChild, AxisDescendant, AxisDescendantOrSelf:
				hasDown = true
			}
			for _, pred := range step.Predicates {
				count += countDownwardSelectionsInExpr(pred)
			}
		}
		if hasDown {
			count++
		}
	case PathStepExpr:
		// E1/E2 or E1//E2: each side can have downward selections
		count += countDownwardSelectionsInExpr(e.Left)
		count += countDownwardSelectionsInExpr(e.Right)
	case PathExpr:
		count += countDownwardSelectionsInExpr(e.Filter)
		if e.Path != nil {
			lp := *e.Path
			count += countDownwardSelectionsInExpr(lp)
		}
	case vmPathExpr:
		count += countDownwardSelectionsInExpr(e.Filter)
		if e.Path != nil {
			count += countDownwardSelectionsInExpr(*e.Path)
		}
	case BinaryExpr:
		count += countDownwardSelectionsInExpr(e.Left)
		count += countDownwardSelectionsInExpr(e.Right)
	case ConcatExpr:
		count += countDownwardSelectionsInExpr(e.Left)
		count += countDownwardSelectionsInExpr(e.Right)
	case SimpleMapExpr:
		count += countDownwardSelectionsInExpr(e.Left)
		count += countDownwardSelectionsInExpr(e.Right)
	case UnionExpr:
		count += countDownwardSelectionsInExpr(e.Left)
		count += countDownwardSelectionsInExpr(e.Right)
	case FilterExpr:
		count += countDownwardSelectionsInExpr(e.Expr)
		for _, pred := range e.Predicates {
			count += countDownwardSelectionsInExpr(pred)
		}
	case FunctionCall:
		for _, arg := range e.Args {
			count += countDownwardSelectionsInExpr(arg)
		}
	case SequenceExpr:
		for _, item := range e.Items {
			count += countDownwardSelectionsInExpr(item)
		}
	case IfExpr:
		// max of then/else + condition
		count += countDownwardSelectionsInExpr(e.Cond)
		thenCount := countDownwardSelectionsInExpr(e.Then)
		elseCount := countDownwardSelectionsInExpr(e.Else)
		if thenCount > elseCount {
			count += thenCount
		} else {
			count += elseCount
		}
	case FLWORExpr:
		for _, clause := range e.Clauses {
			switch c := clause.(type) {
			case ForClause:
				count += countDownwardSelectionsInExpr(c.Expr)
			case LetClause:
				count += countDownwardSelectionsInExpr(c.Expr)
			}
		}
		count += countDownwardSelectionsInExpr(e.Return)
	}
	return count
}

// ExprUsesUpwardAxis returns true if the expression uses parent::, ancestor::,
// ancestor-or-self::, preceding::, or preceding-sibling:: axes.
func ExprUsesUpwardAxis(expr *Expression) bool {
	if expr == nil {
		return false
	}
	if expr.program != nil {
		const upwardMask = 1<<uint(AxisParent) |
			1<<uint(AxisAncestor) |
			1<<uint(AxisAncestorOrSelf) |
			1<<uint(AxisPreceding) |
			1<<uint(AxisPrecedingSibling)
		return expr.program.stream.axisUsed&upwardMask != 0
	}
	return false
}

// ExprUsesPrecedingAxis returns true if the expression uses preceding:: or
// preceding-sibling:: axes, which require backward access to already-consumed
// nodes and are therefore non-streamable.
func ExprUsesPrecedingAxis(expr *Expression) bool {
	if expr == nil {
		return false
	}
	if expr.program != nil {
		const precMask = 1<<uint(AxisPreceding) | 1<<uint(AxisPrecedingSibling)
		return expr.program.stream.axisUsed&precMask != 0
	}
	return false
}

// ExprUsesFollowingSiblingAxis returns true if the expression uses
// following-sibling:: axis.
func ExprUsesFollowingSiblingAxis(expr *Expression) bool {
	if expr == nil {
		return false
	}
	if expr.program != nil {
		return expr.program.stream.axisUsed&(1<<uint(AxisFollowingSibling)) != 0
	}
	return false
}

// ExprUsesDescendantOrSelf returns true if the expression uses descendant:: or
// descendant-or-self:: axis (i.e. // shorthand or explicit descendant axis).
func ExprUsesDescendantOrSelf(expr *Expression) bool {
	if expr == nil {
		return false
	}
	if expr.program != nil {
		const descMask = 1<<uint(AxisDescendant) | 1<<uint(AxisDescendantOrSelf)
		return expr.program.stream.axisUsed&descMask != 0 ||
			expr.program.stream.hasDescOrSelf
	}
	return false
}

// ExprUsesContextItem returns true if the expression references the context
// item (.) at any level. This is used to detect consuming operations in
// streaming accumulator rule select expressions.
func ExprUsesContextItem(expr *Expression) bool {
	if expr == nil {
		return false
	}
	found := false
	WalkExpr(expr.AST(), func(e Expr) bool {
		if found {
			return false
		}
		switch e.(type) {
		case ContextItemExpr:
			found = true
			return false
		}
		// Also check for lowered "." → self::node() form.
		if lp, ok := e.(vmLocationPathExpr); ok {
			if !lp.Absolute && len(lp.Steps) == 1 {
				s := lp.Steps[0]
				if s.Axis == AxisSelf && len(s.Predicates) == 0 {
					if tt, oktt := s.NodeTest.(TypeTest); oktt && tt.Kind == NodeKindNode {
						found = true
						return false
					}
				}
			}
		}
		return true
	})
	return found
}
