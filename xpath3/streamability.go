package xpath3

import "math/big"

// AST returns the root AST node of the compiled expression.
// This is used by streamability analysis in xslt3 via internal/xpathstream.
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
	walkExpr(ast, func(e Expr) bool {
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

// walkExpr walks an XPath 3.1 AST, calling fn for each Expr node.
// If fn returns false, children of that node are not visited.
func walkExpr(expr Expr, fn func(Expr) bool) {
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
				walkExpr(pred, fn)
			}
		}

	case vmLocationPathExpr:
		for _, step := range e.Steps {
			for _, pred := range step.Predicates {
				walkExpr(pred, fn)
			}
		}

	case BinaryExpr:
		walkExpr(e.Left, fn)
		walkExpr(e.Right, fn)

	case UnaryExpr:
		walkExpr(e.Operand, fn)

	case ConcatExpr:
		walkExpr(e.Left, fn)
		walkExpr(e.Right, fn)

	case SimpleMapExpr:
		walkExpr(e.Left, fn)
		walkExpr(e.Right, fn)

	case RangeExpr:
		walkExpr(e.Start, fn)
		walkExpr(e.End, fn)

	case UnionExpr:
		walkExpr(e.Left, fn)
		walkExpr(e.Right, fn)

	case IntersectExceptExpr:
		walkExpr(e.Left, fn)
		walkExpr(e.Right, fn)

	case FilterExpr:
		walkExpr(e.Expr, fn)
		for _, pred := range e.Predicates {
			walkExpr(pred, fn)
		}

	case PathExpr:
		walkExpr(e.Filter, fn)
		if e.Path != nil {
			walkExpr(*e.Path, fn)
		}

	case vmPathExpr:
		walkExpr(e.Filter, fn)
		if e.Path != nil {
			walkExpr(*e.Path, fn)
		}

	case PathStepExpr:
		walkExpr(e.Left, fn)
		walkExpr(e.Right, fn)

	case LookupExpr:
		walkExpr(e.Expr, fn)
		walkExpr(e.Key, fn)

	case UnaryLookupExpr:
		walkExpr(e.Key, fn)

	case FLWORExpr:
		for _, clause := range e.Clauses {
			switch c := clause.(type) {
			case ForClause:
				walkExpr(c.Expr, fn)
			case LetClause:
				walkExpr(c.Expr, fn)
			}
		}
		walkExpr(e.Return, fn)

	case QuantifiedExpr:
		for _, b := range e.Bindings {
			walkExpr(b.Domain, fn)
		}
		walkExpr(e.Satisfies, fn)

	case IfExpr:
		walkExpr(e.Cond, fn)
		walkExpr(e.Then, fn)
		walkExpr(e.Else, fn)

	case TryCatchExpr:
		walkExpr(e.Try, fn)
		for _, cc := range e.Catches {
			walkExpr(cc.Expr, fn)
		}

	case InstanceOfExpr:
		walkExpr(e.Expr, fn)

	case CastExpr:
		walkExpr(e.Expr, fn)

	case CastableExpr:
		walkExpr(e.Expr, fn)

	case TreatAsExpr:
		walkExpr(e.Expr, fn)

	case FunctionCall:
		for _, arg := range e.Args {
			walkExpr(arg, fn)
		}

	case DynamicFunctionCall:
		walkExpr(e.Func, fn)
		for _, arg := range e.Args {
			walkExpr(arg, fn)
		}

	case NamedFunctionRef:
		// leaf

	case InlineFunctionExpr:
		walkExpr(e.Body, fn)

	case MapConstructorExpr:
		for _, pair := range e.Pairs {
			walkExpr(pair.Key, fn)
			walkExpr(pair.Value, fn)
		}

	case ArrayConstructorExpr:
		for _, item := range e.Items {
			walkExpr(item, fn)
		}

	case SequenceExpr:
		for _, item := range e.Items {
			walkExpr(item, fn)
		}
	}
}

// predicateIsNonMotionless returns true if a predicate expression navigates
// downward (uses child/descendant axes), uses last(), or uses position() in
// a non-trivial way.
func predicateIsNonMotionless(pred Expr) bool {
	return predicateIsNonMotionlessWithStep(pred, nil)
}

func predicateIsNonMotionlessWithStep(pred Expr, step *Step) bool {
	nonMotionless := false
	walkExpr(pred, func(e Expr) bool {
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
			if v.Prefix == "" {
				switch v.Name {
				case "name", "local-name", "namespace-uri", "node-name",
					"self", "generate-id", "base-uri", "document-uri",
					"nilled", "has-children", "string-length", "current":
					return false
				}
			}
		case LiteralExpr:
			if e == pred {
				switch v.Value.(type) {
				case float64, *big.Int, *big.Rat:
					nonMotionless = true
					return false
				}
			}
		case InstanceOfExpr, CastableExpr:
			return false
		case ContextItemExpr:
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

func countDownwardSelectionsInExpr(expr Expr) int {
	expr = derefExpr(expr)
	count := 0
	switch e := expr.(type) {
	case LocationPath:
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

// StreamInfo exposes precomputed streamability properties for use by
// internal packages (e.g. xslt3 streamability analysis). This struct
// is not intended for end-user consumption; it lives in an internal
// package that re-exports query helpers.
type StreamInfo struct {
	AxisUsed             uint16
	HasDownwardStep      bool
	HasDescOrSelf        bool
	HasNonMotionlessPred bool
	DownwardSelections   int
	UsedFunctions        map[string]bool
	IsContextItem        bool
}

// StreamInfo returns a snapshot of the precomputed streamability
// properties for this expression.
func (e *Expression) StreamInfo() StreamInfo {
	if e == nil || e.program == nil {
		return StreamInfo{}
	}
	s := e.program.stream
	return StreamInfo{
		AxisUsed:             s.axisUsed,
		HasDownwardStep:      s.hasDownwardStep,
		HasDescOrSelf:        s.hasDescOrSelf,
		HasNonMotionlessPred: s.hasNonMotionlessPred,
		DownwardSelections:   s.downwardSelections,
		UsedFunctions:        s.usedFunctions,
		IsContextItem:        s.isContextItem,
	}
}
