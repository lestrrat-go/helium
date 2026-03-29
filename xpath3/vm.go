package xpath3

import (
	"context"
	"fmt"
)

type vmOpcode uint8

const (
	vmOpLiteral vmOpcode = iota
	vmOpVariable
	vmOpRoot
	vmOpContextItem
	vmOpLocationPath
	vmOpBinary
	vmOpUnary
	vmOpConcat
	vmOpSimpleMap
	vmOpRange
	vmOpUnion
	vmOpIntersectExcept
	vmOpFilter
	vmOpPath
	vmOpPathStep
	vmOpLookup
	vmOpUnaryLookup
	vmOpFLWOR
	vmOpQuantified
	vmOpIf
	vmOpTryCatch
	vmOpInstanceOf
	vmOpCast
	vmOpCastable
	vmOpTreatAs
	vmOpFunctionCall
	vmOpDynamicFunctionCall
	vmOpNamedFunctionRef
	vmOpInlineFunction
	vmOpPlaceholder
	vmOpMapConstructor
	vmOpArrayConstructor
	vmOpSequence
)

type compiledExprRef struct {
	index int
}

func (compiledExprRef) exprNode() {}

type vmLocationStep struct {
	Axis       AxisType
	NodeTest   NodeTest
	Predicates []Expr
}

type vmPositionPredicateExpr struct {
	Position int
}

func (vmPositionPredicateExpr) exprNode() {}

type vmAttributeExistsPredicateExpr struct {
	NodeTest NodeTest
}

func (vmAttributeExistsPredicateExpr) exprNode() {}

type vmAttributeEqualsStringPredicateExpr struct {
	NodeTest NodeTest
	Value    string
	Fallback Expr
}

func (vmAttributeEqualsStringPredicateExpr) exprNode() {}

type vmLocationPathExpr struct {
	Absolute bool
	Steps    []vmLocationStep
}

func (vmLocationPathExpr) exprNode() {}

type vmPathExpr struct {
	Filter        Expr
	Path          *vmLocationPathExpr
	PreserveOrder bool // true when filter is reverse()/sort() — skip doc-order sort
}

func (vmPathExpr) exprNode() {}

type vmInstruction struct {
	op      vmOpcode
	payload any
}

// streamInfo holds precomputed static-analysis properties so that
// streamability queries are O(1) field reads instead of O(n) AST walks.
type streamInfo struct {
	axisUsed             uint16          // bitmask of AxisType values present
	hasDownwardStep      bool            // child/descendant/descendant-or-self axis or //
	hasDescOrSelf        bool            // PathStepExpr with // shorthand
	hasNonMotionlessPred bool            // predicate with downward nav or last()
	downwardSelections   int             // independent downward selection count
	usedFunctions        map[string]bool // unprefixed function names called
	isContextItem        bool            // expression is just "."
}

type vmProgram struct {
	root         int
	instructions []vmInstruction
	stream       streamInfo
}

func compileVMProgram(ast Expr) (*vmProgram, prefixValidationPlan, error) {
	return compileVMProgramWithOptions(ast, false)
}

func compileOwnedVMProgram(ast Expr) (*vmProgram, prefixValidationPlan, error) {
	return compileVMProgramWithOptions(ast, true)
}

func compileVMProgramWithOptions(ast Expr, reuseInput bool) (*vmProgram, prefixValidationPlan, error) {
	// Compute streamability info from AST before lowering.
	si := computeStreamInfo(ast)

	builder := vmBuilder{
		instructions: make([]vmInstruction, 0, 8),
		prefixPlan:   newPrefixPlanBuilder(),
		reuseInput:   reuseInput,
	}
	root, err := builder.compileExpr(ast)
	if err != nil {
		return nil, prefixValidationPlan{}, err
	}
	return &vmProgram{
		root:         root.index,
		instructions: builder.instructions,
		stream:       si,
	}, builder.prefixPlan.plan(), nil
}

type vmBuilder struct {
	instructions []vmInstruction
	prefixPlan   prefixPlanBuilder
	reuseInput   bool
}

func (b *vmBuilder) compileExpr(expr Expr) (compiledExprRef, error) {
	lowered, err := b.lowerExpr(expr)
	if err != nil {
		return compiledExprRef{}, err
	}
	appendExprLocalPrefixChecks(&b.prefixPlan, lowered)
	return b.appendInstruction(lowered), nil
}

func (b *vmBuilder) appendInstruction(lowered Expr) compiledExprRef {
	ref := compiledExprRef{index: len(b.instructions)}
	b.instructions = append(b.instructions, vmInstruction{
		op:      opcodeForExpr(lowered),
		payload: lowered,
	})
	return ref
}

func (b *vmBuilder) lowerChildExpr(expr Expr) (Expr, error) {
	lowered, err := b.lowerExpr(expr)
	if err != nil {
		return nil, err
	}
	appendExprLocalPrefixChecks(&b.prefixPlan, lowered)
	if isImmediateVMExpr(lowered) {
		return lowered, nil
	}
	return b.appendInstruction(lowered), nil
}

func (b *vmBuilder) lowerExpr(expr Expr) (Expr, error) {
	switch e := expr.(type) {
	case LiteralExpr, VariableExpr, RootExpr, ContextItemExpr, NamedFunctionRef, PlaceholderExpr:
		return e, nil
	case *LiteralExpr:
		if e == nil {
			return nil, fmt.Errorf("%w: nil *LiteralExpr", ErrUnsupportedExpr)
		}
		return *e, nil
	case *VariableExpr:
		if e == nil {
			return nil, fmt.Errorf("%w: nil *VariableExpr", ErrUnsupportedExpr)
		}
		return *e, nil
	case *RootExpr:
		if e == nil {
			return nil, fmt.Errorf("%w: nil *RootExpr", ErrUnsupportedExpr)
		}
		return *e, nil
	case *ContextItemExpr:
		if e == nil {
			return nil, fmt.Errorf("%w: nil *ContextItemExpr", ErrUnsupportedExpr)
		}
		return *e, nil
	case *NamedFunctionRef:
		if e == nil {
			return nil, fmt.Errorf("%w: nil *NamedFunctionRef", ErrUnsupportedExpr)
		}
		return *e, nil
	case *PlaceholderExpr:
		if e == nil {
			return nil, fmt.Errorf("%w: nil *PlaceholderExpr", ErrUnsupportedExpr)
		}
		return *e, nil
	case LocationPath:
		return b.lowerLocationPath(e)
	case *LocationPath:
		if e == nil {
			return nil, fmt.Errorf("%w: nil *LocationPath", ErrUnsupportedExpr)
		}
		if b.reuseInput {
			return b.lowerOwnedLocationPath(e)
		}
		return b.lowerLocationPath(*e)
	case vmLocationPathExpr:
		return b.lowerVMLocationPath(e)
	case *vmLocationPathExpr:
		if e == nil {
			return nil, fmt.Errorf("%w: nil *vmLocationPathExpr", ErrUnsupportedExpr)
		}
		return b.lowerVMLocationPath(*e)
	case BinaryExpr:
		return b.lowerBinaryExpr(e)
	case *BinaryExpr:
		if e == nil {
			return nil, fmt.Errorf("%w: nil *BinaryExpr", ErrUnsupportedExpr)
		}
		return b.lowerBinaryExpr(*e)
	case UnaryExpr:
		return b.lowerUnaryExpr(e)
	case *UnaryExpr:
		if e == nil {
			return nil, fmt.Errorf("%w: nil *UnaryExpr", ErrUnsupportedExpr)
		}
		return b.lowerUnaryExpr(*e)
	case ConcatExpr:
		return b.lowerConcatExpr(e)
	case *ConcatExpr:
		if e == nil {
			return nil, fmt.Errorf("%w: nil *ConcatExpr", ErrUnsupportedExpr)
		}
		return b.lowerConcatExpr(*e)
	case SimpleMapExpr:
		return b.lowerSimpleMapExpr(e)
	case *SimpleMapExpr:
		if e == nil {
			return nil, fmt.Errorf("%w: nil *SimpleMapExpr", ErrUnsupportedExpr)
		}
		return b.lowerSimpleMapExpr(*e)
	case RangeExpr:
		return b.lowerRangeExpr(e)
	case *RangeExpr:
		if e == nil {
			return nil, fmt.Errorf("%w: nil *RangeExpr", ErrUnsupportedExpr)
		}
		return b.lowerRangeExpr(*e)
	case UnionExpr:
		return b.lowerUnionExpr(e)
	case *UnionExpr:
		if e == nil {
			return nil, fmt.Errorf("%w: nil *UnionExpr", ErrUnsupportedExpr)
		}
		return b.lowerUnionExpr(*e)
	case IntersectExceptExpr:
		return b.lowerIntersectExceptExpr(e)
	case *IntersectExceptExpr:
		if e == nil {
			return nil, fmt.Errorf("%w: nil *IntersectExceptExpr", ErrUnsupportedExpr)
		}
		return b.lowerIntersectExceptExpr(*e)
	case FilterExpr:
		return b.lowerFilterExpr(e)
	case *FilterExpr:
		if e == nil {
			return nil, fmt.Errorf("%w: nil *FilterExpr", ErrUnsupportedExpr)
		}
		return b.lowerFilterExpr(*e)
	case PathExpr:
		return b.lowerPathExpr(e)
	case *PathExpr:
		if e == nil {
			return nil, fmt.Errorf("%w: nil *PathExpr", ErrUnsupportedExpr)
		}
		return b.lowerPathExpr(*e)
	case vmPathExpr:
		return b.lowerVMPathExpr(e)
	case *vmPathExpr:
		if e == nil {
			return nil, fmt.Errorf("%w: nil *vmPathExpr", ErrUnsupportedExpr)
		}
		return b.lowerVMPathExpr(*e)
	case PathStepExpr:
		return b.lowerPathStepExpr(e)
	case *PathStepExpr:
		if e == nil {
			return nil, fmt.Errorf("%w: nil *PathStepExpr", ErrUnsupportedExpr)
		}
		return b.lowerPathStepExpr(*e)
	case LookupExpr:
		return b.lowerLookupExpr(e)
	case *LookupExpr:
		if e == nil {
			return nil, fmt.Errorf("%w: nil *LookupExpr", ErrUnsupportedExpr)
		}
		return b.lowerLookupExpr(*e)
	case UnaryLookupExpr:
		return b.lowerUnaryLookupExpr(e)
	case *UnaryLookupExpr:
		if e == nil {
			return nil, fmt.Errorf("%w: nil *UnaryLookupExpr", ErrUnsupportedExpr)
		}
		return b.lowerUnaryLookupExpr(*e)
	case FLWORExpr:
		return b.lowerFLWORExpr(e)
	case *FLWORExpr:
		if e == nil {
			return nil, fmt.Errorf("%w: nil *FLWORExpr", ErrUnsupportedExpr)
		}
		return b.lowerFLWORExpr(*e)
	case QuantifiedExpr:
		return b.lowerQuantifiedExpr(e)
	case *QuantifiedExpr:
		if e == nil {
			return nil, fmt.Errorf("%w: nil *QuantifiedExpr", ErrUnsupportedExpr)
		}
		return b.lowerQuantifiedExpr(*e)
	case IfExpr:
		return b.lowerIfExpr(e)
	case *IfExpr:
		if e == nil {
			return nil, fmt.Errorf("%w: nil *IfExpr", ErrUnsupportedExpr)
		}
		return b.lowerIfExpr(*e)
	case TryCatchExpr:
		return b.lowerTryCatchExpr(e)
	case *TryCatchExpr:
		if e == nil {
			return nil, fmt.Errorf("%w: nil *TryCatchExpr", ErrUnsupportedExpr)
		}
		return b.lowerTryCatchExpr(*e)
	case InstanceOfExpr:
		return b.lowerInstanceOfExpr(e)
	case *InstanceOfExpr:
		if e == nil {
			return nil, fmt.Errorf("%w: nil *InstanceOfExpr", ErrUnsupportedExpr)
		}
		return b.lowerInstanceOfExpr(*e)
	case CastExpr:
		return b.lowerCastExpr(e)
	case *CastExpr:
		if e == nil {
			return nil, fmt.Errorf("%w: nil *CastExpr", ErrUnsupportedExpr)
		}
		return b.lowerCastExpr(*e)
	case CastableExpr:
		return b.lowerCastableExpr(e)
	case *CastableExpr:
		if e == nil {
			return nil, fmt.Errorf("%w: nil *CastableExpr", ErrUnsupportedExpr)
		}
		return b.lowerCastableExpr(*e)
	case TreatAsExpr:
		return b.lowerTreatAsExpr(e)
	case *TreatAsExpr:
		if e == nil {
			return nil, fmt.Errorf("%w: nil *TreatAsExpr", ErrUnsupportedExpr)
		}
		return b.lowerTreatAsExpr(*e)
	case FunctionCall:
		return b.lowerFunctionCall(e)
	case *FunctionCall:
		if e == nil {
			return nil, fmt.Errorf("%w: nil *FunctionCall", ErrUnsupportedExpr)
		}
		return b.lowerFunctionCall(*e)
	case DynamicFunctionCall:
		return b.lowerDynamicFunctionCall(e)
	case *DynamicFunctionCall:
		if e == nil {
			return nil, fmt.Errorf("%w: nil *DynamicFunctionCall", ErrUnsupportedExpr)
		}
		return b.lowerDynamicFunctionCall(*e)
	case InlineFunctionExpr:
		return b.lowerInlineFunctionExpr(e)
	case *InlineFunctionExpr:
		if e == nil {
			return nil, fmt.Errorf("%w: nil *InlineFunctionExpr", ErrUnsupportedExpr)
		}
		return b.lowerInlineFunctionExpr(*e)
	case MapConstructorExpr:
		return b.lowerMapConstructorExpr(e)
	case *MapConstructorExpr:
		if e == nil {
			return nil, fmt.Errorf("%w: nil *MapConstructorExpr", ErrUnsupportedExpr)
		}
		return b.lowerMapConstructorExpr(*e)
	case ArrayConstructorExpr:
		return b.lowerArrayConstructorExpr(e)
	case *ArrayConstructorExpr:
		if e == nil {
			return nil, fmt.Errorf("%w: nil *ArrayConstructorExpr", ErrUnsupportedExpr)
		}
		return b.lowerArrayConstructorExpr(*e)
	case SequenceExpr:
		return b.lowerSequenceExpr(e)
	case *SequenceExpr:
		if e == nil {
			return nil, fmt.Errorf("%w: nil *SequenceExpr", ErrUnsupportedExpr)
		}
		return b.lowerSequenceExpr(*e)
	default:
		return nil, fmt.Errorf("%w: %T", ErrUnsupportedExpr, expr)
	}
}

func (b *vmBuilder) lowerLocationPath(expr LocationPath) (Expr, error) {
	return b.lowerLocationPathSteps(expr.Absolute, expr.Steps)
}

func (b *vmBuilder) lowerOwnedLocationPath(expr *LocationPath) (Expr, error) {
	return b.lowerLocationPathSteps(expr.Absolute, expr.Steps)
}

func (b *vmBuilder) lowerVMLocationPath(expr vmLocationPathExpr) (Expr, error) {
	var steps []vmLocationStep
	if b.reuseInput {
		steps = expr.Steps
	} else {
		steps = make([]vmLocationStep, len(expr.Steps))
		copy(steps, expr.Steps)
	}
	for i := range steps {
		preds, err := b.lowerPredicateSlice(steps[i].Predicates)
		if err != nil {
			return nil, err
		}
		steps[i].Predicates = preds
	}
	return vmLocationPathExpr{Absolute: expr.Absolute, Steps: steps}, nil
}

func (b *vmBuilder) lowerLocationPathSteps(absolute bool, steps []Step) (Expr, error) {
	loweredSteps := make([]vmLocationStep, len(steps))
	for i, step := range steps {
		preds, err := b.lowerPredicateSlice(step.Predicates)
		if err != nil {
			return nil, err
		}
		loweredSteps[i] = vmLocationStep{
			Axis:       step.Axis,
			NodeTest:   step.NodeTest,
			Predicates: preds,
		}
	}
	return vmLocationPathExpr{Absolute: absolute, Steps: loweredSteps}, nil
}

func (b *vmBuilder) lowerBinaryExpr(expr BinaryExpr) (Expr, error) {
	left, err := b.lowerBinaryOperand(expr.Op, expr.Left)
	if err != nil {
		return nil, err
	}
	right, err := b.lowerBinaryOperand(expr.Op, expr.Right)
	if err != nil {
		return nil, err
	}
	return BinaryExpr{Op: expr.Op, Left: left, Right: right}, nil
}

func (b *vmBuilder) lowerBinaryOperand(op TokenType, expr Expr) (Expr, error) {
	if !isGeneralComparisonToken(op) {
		return b.lowerChildExpr(expr)
	}

	switch e := expr.(type) {
	case RangeExpr:
		return b.lowerRangeExpr(e)
	case *RangeExpr:
		if e == nil {
			return nil, fmt.Errorf("%w: nil *RangeExpr", ErrUnsupportedExpr)
		}
		return b.lowerRangeExpr(*e)
	default:
		return b.lowerChildExpr(expr)
	}
}

func (b *vmBuilder) lowerUnaryExpr(expr UnaryExpr) (Expr, error) {
	operand, err := b.lowerChildExpr(expr.Operand)
	if err != nil {
		return nil, err
	}
	return UnaryExpr{Operand: operand, Negate: expr.Negate}, nil
}

func (b *vmBuilder) lowerConcatExpr(expr ConcatExpr) (Expr, error) {
	left, err := b.lowerChildExpr(expr.Left)
	if err != nil {
		return nil, err
	}
	right, err := b.lowerChildExpr(expr.Right)
	if err != nil {
		return nil, err
	}
	return ConcatExpr{Left: left, Right: right}, nil
}

func (b *vmBuilder) lowerSimpleMapExpr(expr SimpleMapExpr) (Expr, error) {
	left, err := b.lowerChildExpr(expr.Left)
	if err != nil {
		return nil, err
	}
	right, err := b.lowerChildExpr(expr.Right)
	if err != nil {
		return nil, err
	}
	return SimpleMapExpr{Left: left, Right: right}, nil
}

func (b *vmBuilder) lowerRangeExpr(expr RangeExpr) (Expr, error) {
	start, err := b.lowerChildExpr(expr.Start)
	if err != nil {
		return nil, err
	}
	end, err := b.lowerChildExpr(expr.End)
	if err != nil {
		return nil, err
	}
	return RangeExpr{Start: start, End: end}, nil
}

func (b *vmBuilder) lowerUnionExpr(expr UnionExpr) (Expr, error) {
	left, err := b.lowerChildExpr(expr.Left)
	if err != nil {
		return nil, err
	}
	right, err := b.lowerChildExpr(expr.Right)
	if err != nil {
		return nil, err
	}
	return UnionExpr{Left: left, Right: right}, nil
}

func (b *vmBuilder) lowerIntersectExceptExpr(expr IntersectExceptExpr) (Expr, error) {
	left, err := b.lowerChildExpr(expr.Left)
	if err != nil {
		return nil, err
	}
	right, err := b.lowerChildExpr(expr.Right)
	if err != nil {
		return nil, err
	}
	return IntersectExceptExpr{Op: expr.Op, Left: left, Right: right}, nil
}

func (b *vmBuilder) lowerFilterExpr(expr FilterExpr) (Expr, error) {
	base, err := b.lowerChildExpr(expr.Expr)
	if err != nil {
		return nil, err
	}
	preds, err := b.lowerChildExprSlice(expr.Predicates)
	if err != nil {
		return nil, err
	}
	return FilterExpr{Expr: base, Predicates: preds}, nil
}

func (b *vmBuilder) lowerPathExpr(expr PathExpr) (Expr, error) {
	filter, err := b.lowerChildExpr(expr.Filter)
	if err != nil {
		return nil, err
	}
	var path *vmLocationPathExpr
	if expr.Path != nil {
		lowered, err := b.lowerLocationPath(*expr.Path)
		if err != nil {
			return nil, err
		}
		lp, ok := AsExpr[vmLocationPathExpr](lowered)
		if !ok {
			return nil, fmt.Errorf("%w: expected vmLocationPathExpr", ErrUnsupportedExpr)
		}
		path = &lp
	}
	return vmPathExpr{Filter: filter, Path: path, PreserveOrder: filterPreservesOrder(expr.Filter)}, nil
}

func (b *vmBuilder) lowerVMPathExpr(expr vmPathExpr) (Expr, error) {
	filter, err := b.lowerChildExpr(expr.Filter)
	if err != nil {
		return nil, err
	}
	var path *vmLocationPathExpr
	if expr.Path != nil {
		lowered, err := b.lowerVMLocationPath(*expr.Path)
		if err != nil {
			return nil, err
		}
		lp, ok := AsExpr[vmLocationPathExpr](lowered)
		if !ok {
			return nil, fmt.Errorf("%w: expected vmLocationPathExpr", ErrUnsupportedExpr)
		}
		path = &lp
	}
	return vmPathExpr{Filter: filter, Path: path, PreserveOrder: expr.PreserveOrder}, nil
}

func (b *vmBuilder) lowerPathStepExpr(expr PathStepExpr) (Expr, error) {
	left, err := b.lowerChildExpr(expr.Left)
	if err != nil {
		return nil, err
	}
	right, err := b.lowerChildExpr(expr.Right)
	if err != nil {
		return nil, err
	}
	return PathStepExpr{Left: left, Right: right, DescOrSelf: expr.DescOrSelf}, nil
}

func (b *vmBuilder) lowerLookupExpr(expr LookupExpr) (Expr, error) {
	base, err := b.lowerChildExpr(expr.Expr)
	if err != nil {
		return nil, err
	}
	var key Expr
	if expr.Key != nil {
		keyRef, err := b.lowerChildExpr(expr.Key)
		if err != nil {
			return nil, err
		}
		key = keyRef
	}
	return LookupExpr{Expr: base, Key: key, All: expr.All}, nil
}

func (b *vmBuilder) lowerUnaryLookupExpr(expr UnaryLookupExpr) (Expr, error) {
	var key Expr
	if expr.Key != nil {
		keyRef, err := b.lowerChildExpr(expr.Key)
		if err != nil {
			return nil, err
		}
		key = keyRef
	}
	return UnaryLookupExpr{Key: key, All: expr.All}, nil
}

func (b *vmBuilder) lowerFLWORExpr(expr FLWORExpr) (Expr, error) {
	clauses := make([]FLWORClause, len(expr.Clauses))
	for i, clause := range expr.Clauses {
		switch c := clause.(type) {
		case ForClause:
			ref, err := b.lowerChildExpr(c.Expr)
			if err != nil {
				return nil, err
			}
			clauses[i] = ForClause{Var: c.Var, PosVar: c.PosVar, Expr: ref}
		case LetClause:
			ref, err := b.lowerChildExpr(c.Expr)
			if err != nil {
				return nil, err
			}
			clauses[i] = LetClause{Var: c.Var, Expr: ref}
		default:
			return nil, fmt.Errorf("%w: unsupported FLWOR clause %T", ErrUnsupportedExpr, clause)
		}
	}
	ret, err := b.lowerChildExpr(expr.Return)
	if err != nil {
		return nil, err
	}
	return FLWORExpr{Clauses: clauses, Return: ret}, nil
}

func (b *vmBuilder) lowerQuantifiedExpr(expr QuantifiedExpr) (Expr, error) {
	bindings := make([]QuantifiedBinding, len(expr.Bindings))
	for i, binding := range expr.Bindings {
		ref, err := b.lowerChildExpr(binding.Domain)
		if err != nil {
			return nil, err
		}
		bindings[i] = QuantifiedBinding{Var: binding.Var, Domain: ref}
	}
	satisfies, err := b.lowerChildExpr(expr.Satisfies)
	if err != nil {
		return nil, err
	}
	return QuantifiedExpr{Some: expr.Some, Bindings: bindings, Satisfies: satisfies}, nil
}

func (b *vmBuilder) lowerIfExpr(expr IfExpr) (Expr, error) {
	cond, err := b.lowerChildExpr(expr.Cond)
	if err != nil {
		return nil, err
	}
	thenExpr, err := b.lowerChildExpr(expr.Then)
	if err != nil {
		return nil, err
	}
	elseExpr, err := b.lowerChildExpr(expr.Else)
	if err != nil {
		return nil, err
	}
	return IfExpr{Cond: cond, Then: thenExpr, Else: elseExpr}, nil
}

func (b *vmBuilder) lowerTryCatchExpr(expr TryCatchExpr) (Expr, error) {
	tryExpr, err := b.lowerChildExpr(expr.Try)
	if err != nil {
		return nil, err
	}
	catches := make([]CatchClause, len(expr.Catches))
	for i, catch := range expr.Catches {
		ref, err := b.lowerChildExpr(catch.Expr)
		if err != nil {
			return nil, err
		}
		catches[i] = CatchClause{Codes: append([]string(nil), catch.Codes...), Expr: ref}
	}
	return TryCatchExpr{Try: tryExpr, Catches: catches}, nil
}

func (b *vmBuilder) lowerInstanceOfExpr(expr InstanceOfExpr) (Expr, error) {
	ref, err := b.lowerChildExpr(expr.Expr)
	if err != nil {
		return nil, err
	}
	return InstanceOfExpr{Expr: ref, Type: expr.Type}, nil
}

func (b *vmBuilder) lowerCastExpr(expr CastExpr) (Expr, error) {
	ref, err := b.lowerChildExpr(expr.Expr)
	if err != nil {
		return nil, err
	}
	return CastExpr{Expr: ref, Type: expr.Type, AllowEmpty: expr.AllowEmpty}, nil
}

func (b *vmBuilder) lowerCastableExpr(expr CastableExpr) (Expr, error) {
	ref, err := b.lowerChildExpr(expr.Expr)
	if err != nil {
		return nil, err
	}
	return CastableExpr{Expr: ref, Type: expr.Type, AllowEmpty: expr.AllowEmpty}, nil
}

func (b *vmBuilder) lowerTreatAsExpr(expr TreatAsExpr) (Expr, error) {
	ref, err := b.lowerChildExpr(expr.Expr)
	if err != nil {
		return nil, err
	}
	return TreatAsExpr{Expr: ref, Type: expr.Type}, nil
}

func (b *vmBuilder) lowerFunctionCall(expr FunctionCall) (Expr, error) {
	args, err := b.lowerChildExprSlice(expr.Args)
	if err != nil {
		return nil, err
	}
	return FunctionCall{Prefix: expr.Prefix, Name: expr.Name, Args: args}, nil
}

func (b *vmBuilder) lowerDynamicFunctionCall(expr DynamicFunctionCall) (Expr, error) {
	fn, err := b.lowerChildExpr(expr.Func)
	if err != nil {
		return nil, err
	}
	args, err := b.lowerChildExprSlice(expr.Args)
	if err != nil {
		return nil, err
	}
	return DynamicFunctionCall{Func: fn, Args: args}, nil
}

func (b *vmBuilder) lowerInlineFunctionExpr(expr InlineFunctionExpr) (Expr, error) {
	body, err := b.lowerChildExpr(expr.Body)
	if err != nil {
		return nil, err
	}
	params := append([]FunctionParam(nil), expr.Params...)
	return InlineFunctionExpr{Params: params, ReturnType: expr.ReturnType, Body: body}, nil
}

func (b *vmBuilder) lowerMapConstructorExpr(expr MapConstructorExpr) (Expr, error) {
	pairs := make([]MapConstructorPair, len(expr.Pairs))
	for i, pair := range expr.Pairs {
		key, err := b.lowerChildExpr(pair.Key)
		if err != nil {
			return nil, err
		}
		value, err := b.lowerChildExpr(pair.Value)
		if err != nil {
			return nil, err
		}
		pairs[i] = MapConstructorPair{Key: key, Value: value}
	}
	return MapConstructorExpr{Pairs: pairs}, nil
}

func (b *vmBuilder) lowerArrayConstructorExpr(expr ArrayConstructorExpr) (Expr, error) {
	items, err := b.lowerChildExprSlice(expr.Items)
	if err != nil {
		return nil, err
	}
	return ArrayConstructorExpr{Items: items, SquareBracket: expr.SquareBracket}, nil
}

func (b *vmBuilder) lowerSequenceExpr(expr SequenceExpr) (Expr, error) {
	items, err := b.lowerChildExprSlice(expr.Items)
	if err != nil {
		return nil, err
	}
	return SequenceExpr{Items: items}, nil
}

func (b *vmBuilder) lowerChildExprSlice(items []Expr) ([]Expr, error) {
	if len(items) == 0 {
		return nil, nil
	}
	var result []Expr
	if b.reuseInput {
		result = items
	} else {
		result = make([]Expr, len(items))
	}
	for i, item := range items {
		switch item.(type) {
		case PlaceholderExpr, *PlaceholderExpr:
			result[i] = PlaceholderExpr{}
			continue
		}
		ref, err := b.lowerChildExpr(item)
		if err != nil {
			return nil, err
		}
		result[i] = ref
	}
	return result, nil
}

func (b *vmBuilder) lowerPredicateSlice(items []Expr) ([]Expr, error) {
	if len(items) == 0 {
		return nil, nil
	}
	var result []Expr
	if b.reuseInput {
		result = items
	} else {
		result = make([]Expr, len(items))
	}
	for i, item := range items {
		lowered, err := b.lowerPredicate(item)
		if err != nil {
			return nil, err
		}
		result[i] = lowered
	}
	return result, nil
}

func (b *vmBuilder) lowerPredicate(expr Expr) (Expr, error) {
	if position, ok := vmPredicatePosition(expr); ok {
		return vmPositionPredicateExpr{Position: position}, nil
	}
	if test, ok := vmPredicateAttributeExists(expr); ok {
		return vmAttributeExistsPredicateExpr{NodeTest: test}, nil
	}
	if test, value, ok := vmPredicateAttributeEqualsString(expr); ok {
		fallback, err := b.lowerChildExpr(expr)
		if err != nil {
			return nil, err
		}
		return vmAttributeEqualsStringPredicateExpr{
			NodeTest: test,
			Value:    value,
			Fallback: fallback,
		}, nil
	}
	return b.lowerChildExpr(expr)
}

func isImmediateVMExpr(expr Expr) bool {
	switch expr.(type) {
	case LiteralExpr, VariableExpr, RootExpr, ContextItemExpr, NamedFunctionRef, PlaceholderExpr:
		return true
	default:
		return false
	}
}

func opcodeForExpr(expr Expr) vmOpcode {
	switch expr.(type) {
	case LiteralExpr:
		return vmOpLiteral
	case VariableExpr:
		return vmOpVariable
	case RootExpr:
		return vmOpRoot
	case ContextItemExpr:
		return vmOpContextItem
	case vmLocationPathExpr:
		return vmOpLocationPath
	case BinaryExpr:
		return vmOpBinary
	case UnaryExpr:
		return vmOpUnary
	case ConcatExpr:
		return vmOpConcat
	case SimpleMapExpr:
		return vmOpSimpleMap
	case RangeExpr:
		return vmOpRange
	case UnionExpr:
		return vmOpUnion
	case IntersectExceptExpr:
		return vmOpIntersectExcept
	case FilterExpr:
		return vmOpFilter
	case vmPathExpr:
		return vmOpPath
	case PathStepExpr:
		return vmOpPathStep
	case LookupExpr:
		return vmOpLookup
	case UnaryLookupExpr:
		return vmOpUnaryLookup
	case FLWORExpr:
		return vmOpFLWOR
	case QuantifiedExpr:
		return vmOpQuantified
	case IfExpr:
		return vmOpIf
	case TryCatchExpr:
		return vmOpTryCatch
	case InstanceOfExpr:
		return vmOpInstanceOf
	case CastExpr:
		return vmOpCast
	case CastableExpr:
		return vmOpCastable
	case TreatAsExpr:
		return vmOpTreatAs
	case FunctionCall:
		return vmOpFunctionCall
	case DynamicFunctionCall:
		return vmOpDynamicFunctionCall
	case NamedFunctionRef:
		return vmOpNamedFunctionRef
	case InlineFunctionExpr:
		return vmOpInlineFunction
	case PlaceholderExpr:
		return vmOpPlaceholder
	case MapConstructorExpr:
		return vmOpMapConstructor
	case ArrayConstructorExpr:
		return vmOpArrayConstructor
	case SequenceExpr:
		return vmOpSequence
	default:
		panic(fmt.Sprintf("xpath3: unknown VM opcode for %T", expr))
	}
}

func isGeneralComparisonToken(op TokenType) bool {
	switch op {
	case TokenEquals, TokenNotEquals, TokenLess, TokenLessEq, TokenGreater, TokenGreaterEq:
		return true
	default:
		return false
	}
}

type vm struct {
	program *vmProgram
}

func (p *vmProgram) execute(ctx context.Context, ec *evalContext) (Sequence, error) {
	machine := vm{program: p}
	return machine.evalExpr(ctx, ec, compiledExprRef{index: p.root})
}

func (v *vm) evalExpr(ctx context.Context, ec *evalContext, expr Expr) (Sequence, error) {
	return evalWith(v.evalExprBody, ctx, ec, expr)
}

func (v *vm) evalExprBody(ctx context.Context, ec *evalContext, expr Expr) (Sequence, error) {
	if ref, ok := expr.(compiledExprRef); ok {
		return v.evalInstruction(ctx, ec, ref)
	}
	return dispatchExpr(v.evalExpr, ctx, ec, expr)
}

// vmEvalPayload extracts a typed Expr payload from a VM instruction and
// evaluates it with fn. Returns an error if the payload type does not match.
func vmEvalPayload[T Expr](inst vmInstruction, fn func(T) (Sequence, error)) (Sequence, error) {
	e, ok := AsExpr[T](inst.payload)
	if !ok {
		return nil, fmt.Errorf("%w: bad payload for %s", ErrUnsupportedExpr, inst.op)
	}
	return fn(e)
}

func (v *vm) evalInstruction(ctx context.Context, ec *evalContext, ref compiledExprRef) (Sequence, error) {
	if ref.index < 0 || ref.index >= len(v.program.instructions) {
		return nil, fmt.Errorf("%w: invalid VM instruction %d", ErrUnsupportedExpr, ref.index)
	}
	inst := v.program.instructions[ref.index]
	switch inst.op {
	case vmOpLiteral:
		return vmEvalPayload(inst, func(e LiteralExpr) (Sequence, error) { return evalLiteral(e) })
	case vmOpVariable:
		return vmEvalPayload(inst, func(e VariableExpr) (Sequence, error) { return evalVariable(ctx, ec, e) })
	case vmOpRoot:
		return evalRootExpr(ec)
	case vmOpContextItem:
		return evalContextItemExpr(ec)
	case vmOpLocationPath:
		return vmEvalPayload(inst, func(e vmLocationPathExpr) (Sequence, error) { return evalVMLocationPath(v.evalExpr, ctx, ec, e) })
	case vmOpBinary:
		return vmEvalPayload(inst, func(e BinaryExpr) (Sequence, error) { return evalBinaryExpr(v.evalExpr, ctx, ec, e) })
	case vmOpUnary:
		return vmEvalPayload(inst, func(e UnaryExpr) (Sequence, error) { return evalUnaryExpr(v.evalExpr, ctx, ec, e) })
	case vmOpConcat:
		return vmEvalPayload(inst, func(e ConcatExpr) (Sequence, error) { return evalConcatExpr(v.evalExpr, ctx, ec, e) })
	case vmOpSimpleMap:
		return vmEvalPayload(inst, func(e SimpleMapExpr) (Sequence, error) { return evalSimpleMapExpr(v.evalExpr, ctx, ec, e) })
	case vmOpRange:
		return vmEvalPayload(inst, func(e RangeExpr) (Sequence, error) { return evalRangeExpr(v.evalExpr, ctx, ec, e) })
	case vmOpUnion:
		return vmEvalPayload(inst, func(e UnionExpr) (Sequence, error) { return evalUnionExpr(v.evalExpr, ctx, ec, e) })
	case vmOpIntersectExcept:
		return vmEvalPayload(inst, func(e IntersectExceptExpr) (Sequence, error) {
			return evalIntersectExceptExpr(v.evalExpr, ctx, ec, e)
		})
	case vmOpFilter:
		return vmEvalPayload(inst, func(e FilterExpr) (Sequence, error) { return evalFilterExpr(v.evalExpr, ctx, ec, e) })
	case vmOpPath:
		return vmEvalPayload(inst, func(e vmPathExpr) (Sequence, error) { return evalVMPathExpr(v.evalExpr, ctx, ec, e) })
	case vmOpPathStep:
		return vmEvalPayload(inst, func(e PathStepExpr) (Sequence, error) { return evalPathStepExpr(v.evalExpr, ctx, ec, e) })
	case vmOpLookup:
		return vmEvalPayload(inst, func(e LookupExpr) (Sequence, error) { return evalLookupExpr(v.evalExpr, ctx, ec, e) })
	case vmOpUnaryLookup:
		return vmEvalPayload(inst, func(e UnaryLookupExpr) (Sequence, error) { return evalUnaryLookupExpr(v.evalExpr, ctx, ec, e) })
	case vmOpFLWOR:
		return vmEvalPayload(inst, func(e FLWORExpr) (Sequence, error) { return evalFLWOR(v.evalExpr, ctx, ec, e) })
	case vmOpQuantified:
		return vmEvalPayload(inst, func(e QuantifiedExpr) (Sequence, error) { return evalQuantifiedExpr(v.evalExpr, ctx, ec, e) })
	case vmOpIf:
		return vmEvalPayload(inst, func(e IfExpr) (Sequence, error) { return evalIfExpr(v.evalExpr, ctx, ec, e) })
	case vmOpTryCatch:
		return vmEvalPayload(inst, func(e TryCatchExpr) (Sequence, error) { return evalTryCatchExpr(v.evalExpr, ctx, ec, e) })
	case vmOpInstanceOf:
		return vmEvalPayload(inst, func(e InstanceOfExpr) (Sequence, error) { return evalInstanceOfExpr(v.evalExpr, ctx, ec, e) })
	case vmOpCast:
		return vmEvalPayload(inst, func(e CastExpr) (Sequence, error) { return evalCastExpr(v.evalExpr, ctx, ec, e) })
	case vmOpCastable:
		return vmEvalPayload(inst, func(e CastableExpr) (Sequence, error) { return evalCastableExpr(v.evalExpr, ctx, ec, e) })
	case vmOpTreatAs:
		return vmEvalPayload(inst, func(e TreatAsExpr) (Sequence, error) { return evalTreatAsExpr(v.evalExpr, ctx, ec, e) })
	case vmOpFunctionCall:
		return vmEvalPayload(inst, func(e FunctionCall) (Sequence, error) { return evalFunctionCall(v.evalExpr, ctx, ec, e) })
	case vmOpDynamicFunctionCall:
		return vmEvalPayload(inst, func(e DynamicFunctionCall) (Sequence, error) {
			return evalDynamicFunctionCall(v.evalExpr, ctx, ec, e)
		})
	case vmOpNamedFunctionRef:
		return vmEvalPayload(inst, func(e NamedFunctionRef) (Sequence, error) { return evalNamedFunctionRef(ctx, ec, e) })
	case vmOpInlineFunction:
		return vmEvalPayload(inst, func(e InlineFunctionExpr) (Sequence, error) { return evalInlineFunctionExpr(v.evalExpr, ctx, ec, e) })
	case vmOpMapConstructor:
		return vmEvalPayload(inst, func(e MapConstructorExpr) (Sequence, error) { return evalMapConstructorExpr(v.evalExpr, ctx, ec, e) })
	case vmOpArrayConstructor:
		return vmEvalPayload(inst, func(e ArrayConstructorExpr) (Sequence, error) {
			return evalArrayConstructorExpr(v.evalExpr, ctx, ec, e)
		})
	case vmOpSequence:
		return vmEvalPayload(inst, func(e SequenceExpr) (Sequence, error) { return evalSequenceExpr(v.evalExpr, ctx, ec, e) })
	case vmOpPlaceholder:
		return nil, fmt.Errorf("%w: placeholder outside partial application", ErrUnsupportedExpr)
	default:
		return nil, fmt.Errorf("%w: invalid VM opcode %d", ErrUnsupportedExpr, inst.op)
	}
}
